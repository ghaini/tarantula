package main

import (
	"bufio"
	"fmt"
	"github.com/valyala/fasthttp"
	"net"
	"regexp"
	"strings"
)

type Result struct {
	StatusCode int
	Asset      string
	Domain     string
	Body       string
	Headers    map[string]string
}

func main() {
	fmt.Println(GetContents("", []string{"https://webhook.site/a1bcaec4-149c-4343-b080-723ecb78c999"}, 0, []int{}, []int{}))
}
func GetContents(domain string, subdomains []string, thread int, httpPorts, httpsPorts []int) []Result {
	doRequest(subdomains[0])
	return []Result{}
}

func doRequest(url string) Result {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI(url)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	err := fasthttp.Do(req, resp)
	if err != nil {
		fmt.Printf("Client get failed: %s\n", err)
		return Result{}
	}

	headers := make(map[string]string)
	headerString := resp.Header.String()

	r, _ := regexp.Compile("(^.+)\\:(.+$)")
	scanner := bufio.NewScanner(strings.NewReader(headerString))
	for scanner.Scan() {
		text := scanner.Text()
		headers[strings.ToLower(strings.TrimSpace(r.FindStringSubmatch(text)[1]))] = strings.ToLower(strings.TrimSpace(r.FindStringSubmatch(text)[2]))
	}


	return Result{
		StatusCode: resp.StatusCode(),
		Asset:      url,
		Domain:     "",
		Body:       string(resp.Body()),
		Headers:    headers,
	}
}

func FasthttpHTTPDialer(proxyAddr string) fasthttp.DialFunc {
	return func(addr string) (net.Conn, error) {
		fmt.Println(addr)
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

		res.SkipBody = true

		if err := res.Read(bufio.NewReader(conn)); err != nil {
			conn.Close()
			return nil, err
		}
		if res.Header.StatusCode() != 200 {
			conn.Close()
			return nil, fmt.Errorf("could not connect to proxy")
		}

		return conn, nil
	}
}