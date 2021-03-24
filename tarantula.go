package tarantula

import (
	"github.com/ghaini/tarantula/detector"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ghaini/tarantula/common"
	"github.com/ghaini/tarantula/constants"
	"github.com/ghaini/tarantula/data"
	"github.com/ghaini/tarantula/proxy"
	"github.com/valyala/fasthttp"
)

type tarantula struct {
	thread            int
	ports             []int
	subdomains        []string
	client            *fasthttp.Client
	withBody          bool
	withTitle         bool
	withTechnology    bool
	userAgents        []string
	timeout           int
	retry             int
	filterStatusCodes []int
}

func NewTarantula() *tarantula {
	rand.Seed(time.Now().UTC().UnixNano())
	return &tarantula{
		thread:     1,
		ports:      []int{443},
		subdomains: nil,
		client:     &fasthttp.Client{},
		userAgents: data.UserAgents,
		timeout:    5,
	}
}

func (t *tarantula) MultiThread(count int) *tarantula {
	t.thread = count
	return t
}

func (t *tarantula) SetPorts(ports []int) *tarantula {
	t.ports = ports
	return t
}

func (t *tarantula) SetUserAgents(userAgents []string) *tarantula {
	t.userAgents = userAgents
	return t
}

func (t *tarantula) SetTimeout(second int) *tarantula {
	t.timeout = second
	return t
}

func (t *tarantula) SetRetry(second int) *tarantula {
	t.timeout = second
	return t
}

func (t *tarantula) HTTPProxy(proxyAddress string) *tarantula {
	t.client = &fasthttp.Client{
		Dial: proxy.HTTPProxyDialer(proxyAddress),
	}
	return t
}

func (t *tarantula) SocksProxy(proxyAddress string) *tarantula {
	t.client = &fasthttp.Client{
		Dial: proxy.SocksDialer(proxyAddress),
	}
	return t
}

func (t *tarantula) WithBody() *tarantula {
	t.withBody = true
	return t
}

func (t *tarantula) WithTitle() *tarantula {
	t.withTitle = true
	return t
}

func (t *tarantula) WithTechnology() *tarantula {
	t.withTechnology = true
	return t
}

func (t *tarantula) FilterStatusCode(codes []int) *tarantula {
	t.filterStatusCodes = codes
	return t
}

func (t *tarantula) GetAssets(domain string, subdomains []string) []Result {
	var wg sync.WaitGroup
	result := make(chan Result, 100)
	inputs := make(chan input)
	var results []Result
	for i := 0; i < t.thread; i++ {
		wg.Add(1)
		go func(result chan<- Result, input <-chan input, domain string, work int) {
			for inp := range inputs {
				t.doRequest(domain, constants.HTTPS, inp.Subdomain, inp.Port, t.retry, result)
			}
			wg.Done()
		}(result, inputs, domain, i)
	}

	go func(subdomains []string) {
		for _, subdomain := range subdomains {
			for _, port := range t.ports {
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

func (t *tarantula) GetAssetsChan(domain string, subdomains []string) chan Result {

	var wg sync.WaitGroup
	result := make(chan Result, 100)
	inputs := make(chan input)
	for i := 0; i < t.thread; i++ {
		wg.Add(1)
		go func(result chan<- Result, input <-chan input, domain string, work int) {
			for inp := range inputs {
				t.doRequest(domain, constants.HTTPS, inp.Subdomain, inp.Port, t.retry, result)
			}
			wg.Done()
		}(result, inputs, domain, i)
	}

	go func(subdomains []string) {
		for _, subdomain := range subdomains {
			for _, port := range t.ports {
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
	return result
}

func (t *tarantula) doRequest(domain, protocol, subdomain string, port int, retry int, result chan<- Result) {
	url := protocol + "://" + subdomain + ":" + strconv.Itoa(port)
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(url)
	// set headers
	req.Header.SetUserAgent(t.userAgents[rand.Intn(len(t.userAgents))])
	req.Header.Set("ACCEPT", "\ttext/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	req.Header.Set("REFERER", "https://www.google.com/")
	req.Header.Set("Accept-Charset", "utf-8")
	req.Header.Set("access-control-allow-origin", url)
	req.Header.Set("access-control-allow-credentials", "true")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	resp.SkipBody = !t.withBody && !t.withTitle && !t.withTechnology
	err := t.client.DoTimeout(req, resp, time.Duration(t.timeout)*time.Second)
	if err != nil {
		if retry > 0 {
			t.doRequest(domain, protocol, subdomain, port, retry-1, result)
			return
		} else if protocol == constants.HTTPS {
			t.doRequest(domain, constants.HTTP, subdomain, port, t.retry, result)
			return
		} else {
			return
		}
	}

	for _, statusCode := range t.filterStatusCodes {
		if statusCode == resp.StatusCode() {
			return
		}
	}

	resp.StatusCode()
	headers := make(map[string]string)
	resp.Header.VisitAll(func(key, value []byte) {
		headers[strings.ToLower(string(key))] = strings.ToLower(string(value))
	})

	title := ""
	if t.withTitle {
		title = common.ExtractTitle(resp)
	}

	body := ""
	if t.withBody {
		body = string(resp.Body())
	}

	technologies := make(map[string]string)
	if t.withTechnology {
		d := detector.Technology{}
		matches := d.Technology(url, resp.Body(), &resp.Header)
		for _, match := range matches {
			for _, cat := range match.CatNames {
				technologies[cat] = match.AppName
			}
		}
	}

	result <- Result{
		StatusCode:   resp.StatusCode(),
		Asset:        url,
		Domain:       domain,
		Body:         body,
		Headers:      headers,
		Title:        title,
		Technologies: technologies,
	}
}
