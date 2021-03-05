package tarantulas

import (
	"bufio"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ghaini/tarantulas/data"
	"github.com/ghaini/tarantulas/proxy"
	"github.com/valyala/fasthttp"
)

type tarantulas struct {
	Thread     int
	Ports      []int
	Subdomains []string
	Client     *fasthttp.Client
	Body       bool
	UserAgents []string
	Timeout    int
}

func NewTarantulas() *tarantulas {
	rand.Seed(time.Now().UTC().UnixNano())
	return &tarantulas{
		Thread:     1,
		Ports:      []int{80, 443},
		Subdomains: nil,
		Client:     &fasthttp.Client{},
		UserAgents: data.UserAgents,
		Timeout:    5,
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

func (t tarantulas) SetUserAgents(userAgents []string) tarantulas {
	t.UserAgents = userAgents
	return t
}

func (t tarantulas) SetTimeout(second int) tarantulas {
	t.Timeout = second
	return t
}

func (t tarantulas) HTTPProxy(proxyAddress string) tarantulas {
	t.Client = &fasthttp.Client{
		Dial: proxy.HTTPProxyDialer(proxyAddress),
	}
	return t
}

func (t tarantulas) SocksProxy(proxyAddress string) tarantulas {
	t.Client = &fasthttp.Client{
		Dial: proxy.SocksDialer(proxyAddress),
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
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(url)
	// set headers
	req.Header.SetUserAgent(t.UserAgents[rand.Intn(len(t.UserAgents))])
	req.Header.Set("ACCEPT", "\ttext/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	req.Header.Set("ACCEPT-ENCODING", "gzip, deflate, br")
	req.Header.Set("REFERER", "https://www.google.com/")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	resp.SkipBody = !t.Body

	err := t.Client.DoTimeout(req, resp, time.Duration(t.Timeout)*time.Second)
	if err != nil {
		url = "http://" + subdomain + ":" + strconv.Itoa(port)
		req.SetRequestURI(url)
		err = t.Client.DoTimeout(req, resp, time.Duration(t.Timeout)*time.Second)
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
