package main

import (
	"bufio"
	"fmt"
	"github.com/valyala/fasthttp"
	"net"
	"regexp"
	"strings"
	"sync"
)

type Result struct {
	StatusCode int
	Asset      string
	Domain     string
	Body       string
	Headers    map[string]string
}

type tarantulas struct {
	Thread     int
	HttpPorts  []int
	HttpsPorts []int
	Subdomains []string
	Client     *fasthttp.Client
	Body       bool
}

func main() {
	t := NewTarantulas()
	fmt.Println(t.GetContents("webhook.site", []string{"https://google.com"}))
}

func NewTarantulas() *tarantulas {
	return &tarantulas{
		Thread:     1,
		HttpPorts:  []int{80},
		HttpsPorts: []int{443},
		Subdomains: nil,
		Client:     &fasthttp.Client{},
	}
}

func (t tarantulas) MultiThread(count int) tarantulas {
	t.Thread = count
	return t
}

func (t tarantulas) SetHttpPorts(ports []int) tarantulas {
	t.HttpPorts = ports
	return t
}

func (t tarantulas) SetHttpsPorts(ports []int) tarantulas {
	t.HttpsPorts = ports
	return t
}

func (t tarantulas) Proxy(proxyAddress string) tarantulas {
	t.Client = &fasthttp.Client{
		Dial: t.fasthttpHTTPProxyDialer(proxyAddress),
	}
	return t
}

func (t tarantulas) WithBody() tarantulas {
	t.Body = true
	return t
}

func (t tarantulas) GetContents(domain string, subdomains []string) []Result {
	var wg sync.WaitGroup
	 result := make(chan Result)
	var results []Result
	for i := 0; i < t.Thread; i++ {
		wg.Add(1)
		go t.doRequest(domain, subdomains[0], result, &wg)
	}
	go func() {
		wg.Wait()
		close(result)
	}()

	for r := range result {
		results = append(results, r)
	}

	return results
}

func (t tarantulas) doRequest(domain, url string, result chan<- Result, wg *sync.WaitGroup) {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(url)
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	resp.SkipBody = !t.Body
	defer fasthttp.ReleaseResponse(resp)

	defer wg.Done()

	err := t.Client.Do(req, resp)
	if err != nil {
		fmt.Printf("Client get failed: %s\n", err)
		return
	}

	headers := make(map[string]string)
	headerString := resp.Header.String()

	r, _ := regexp.Compile(`(^.+)\:(.+$)`)
	scanner := bufio.NewScanner(strings.NewReader(headerString))
	for scanner.Scan() {
		text := scanner.Text()
		if len(strings.TrimSpace(text)) == 0 {
			continue
		}

		headerMatch := r.FindStringSubmatch(text)
		if len(headerMatch) == 0 {
			continue
		}

		headers[strings.ToLower(strings.TrimSpace(headerMatch[1]))] = strings.ToLower(strings.TrimSpace(headerMatch[2]))
	}

	result <- Result{
		StatusCode: resp.StatusCode(),
		Asset:      url,
		Domain:     domain,
		Body:       string(resp.Body()),
		Headers:    headers,
	}
}

func (t tarantulas) fasthttpHTTPProxyDialer(proxyAddr string) fasthttp.DialFunc {
	return func(addr string) (net.Conn, error) {
		conn, err := fasthttp.Dial(proxyAddr)
		if err != nil {
			return nil, err
		}

		req := "CONNECT " + addr + " HTTP/1.1\r\n"
		// req += "Proxy-Authorization: xxx\r\n"
		req += "\r\n"

		if _, err := conn.Write([]byte(req)); err != nil {
			return nil, err
		}

		res := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(res)

		if err := res.Read(bufio.NewReader(conn)); err != nil {
			conn.Close()
			return nil, err
		}

		return conn, nil
	}
}
