package tarantulas

import (
	"bufio"
	"github.com/valyala/fasthttp"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Result struct {
	StatusCode int
	Asset      string
	Domain     string
	Body       string
	Headers    map[string]string
}

type input struct {
	Subdomain string
	Port      int
}

type tarantulas struct {
	Thread     int
	Ports      []int
	Subdomains []string
	Client     *fasthttp.Client
	Body       bool
}
func NewTarantulas() *tarantulas {
	return &tarantulas{
		Thread:     1,
		Ports:      []int{80, 443},
		Subdomains: nil,
		Client: &fasthttp.Client{},
	}
}

func (t tarantulas) MultiThread(count int) tarantulas {
	t.Thread = count
	return t
}

func (t tarantulas) SetPorts(ports []int) tarantulas {
	t.Ports = ports
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
	inputs := make(chan input)
	var results []Result
	for i := 0; i < t.Thread; i++ {
		wg.Add(1)
		go func(result chan<- Result, input <-chan input, domain string, work int) {
			for inp := range inputs {
				t.doRequest(domain, inp.Subdomain, inp.Port, result)
			}
			wg.Done()
		}(result, inputs, domain, i)
	}

	go func(subdomains []string) {
		for _, subdomain := range subdomains {
			for _, port := range t.Ports {
				inputs <- input{
					Subdomain: subdomain,
					Port:      port,
				}
			}
		}
		close(inputs)
	}(subdomains)

	go func() {
		wg.Wait()
		close(result)
	}()

	for r := range result {
		results = append(results, r)
	}

	return results
}

func (t tarantulas) doRequest(domain, subdomain string, port int, result chan<- Result) {
	url := "https://" + subdomain + ":" + strconv.Itoa(port)
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(url)
	req.Header.SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36")
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	resp.SkipBody = !t.Body
	defer fasthttp.ReleaseResponse(resp)

	err := t.Client.DoTimeout(req, resp, 5 * time.Second)
	if err != nil {
		url = "http://" + subdomain + ":" + strconv.Itoa(port)
		req.SetRequestURI(url)
		err = t.Client.DoTimeout(req, resp, 5 * time.Second)
		if err != nil {
			return
		}
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
