package tarantula

import (
	"crypto/tls"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ghaini/tarantula/constants"
	"github.com/ghaini/tarantula/data"
	"github.com/ghaini/tarantula/detector"
	"github.com/ghaini/tarantula/network"
)

type tarantula struct {
	thread             int
	ports              []int
	subdomains         []string
	client             *http.Client
	withBody           bool
	withTitle          bool
	withTechnology     bool
	userAgents         []string
	timeout            int
	retry              int
	filterStatusCodes  []int
	technologyDetector *detector.Technology
}

func NewTarantula() *tarantula {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{
		Transport: tr,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse // Tell the http client to not follow redirect
		},
	}
	rand.Seed(time.Now().UTC().UnixNano())
	return &tarantula{
		thread:             1,
		ports:              []int{443},
		subdomains:         nil,
		client:             client,
		userAgents:         data.UserAgents,
		timeout:            5,
		technologyDetector: detector.NewTechnology(),
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

func (t *tarantula) SetRetry(number int) *tarantula {
	t.retry = number
	return t
}

func (t *tarantula) HTTPProxy(proxyAddress string) *tarantula {
	t.client.Transport = network.DefaultTransport(network.HTTPProxyDialer(proxyAddress))
	return t
}

func (t *tarantula) SocksProxy(proxyAddress string) *tarantula {
	t.client.Transport = network.DefaultTransport(network.SocksDialer(proxyAddress))
	return t
}

func (t *tarantula) RandomDNSServer() *tarantula {
	t.client.Transport = network.DefaultTransport(network.DialerWithCustomDNSResolver())
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
				t.doRequest(domain, constants.HTTPS, inp.Subdomain, inp.Port, t.retry, true, result)
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
				t.doRequest(domain, constants.HTTPS, inp.Subdomain, inp.Port, t.retry, true, result)
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

func (t *tarantula) doRequest(domain, protocol, subdomain string, port int, retry int, canChangeProtocol bool, result chan<- Result) {
	url := subdomain
	if protocol != "" {
		url = protocol + "://" + subdomain
	}

	if port > 0 {
		url += ":" + strconv.Itoa(port)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Close = true

	// set headers
	req.Header.Set("User-Agent", t.userAgents[rand.Intn(len(t.userAgents))])
	req.Header.Set("ACCEPT", "\ttext/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	req.Header.Set("REFERER", "https://www.google.com/")
	req.Header.Set("Accept-Charset", "utf-8")
	req.Header.Set("origin", url)

	t.client.Timeout = time.Duration(t.timeout) * time.Second
	resp, err := t.client.Do(req)
	defer t.client.CloseIdleConnections()
	if err != nil {
		if canChangeProtocol && retry > 0 {
			t.doRequest(domain, protocol, subdomain, port, retry-1, true, result)
			return
		} else if canChangeProtocol && protocol == constants.HTTPS {
			t.doRequest(domain, constants.HTTP, subdomain, port, t.retry, true, result)
			return
		} else {
			return
		}
	}

	defer resp.Body.Close()
	for _, statusCode := range t.filterStatusCodes {
		if statusCode == resp.StatusCode {
			return
		}
	}

	if resp.StatusCode >= 300 && resp.StatusCode <= 399 {
		redirectedLocation := resp.Header.Get("Location")
		redirectedLocationUrl := strings.TrimRight(redirectedLocation, "/")
		domainWithoutSlash := strings.TrimRight(domain, "/")
		if strings.HasSuffix(redirectedLocationUrl, domainWithoutSlash) {
			t.doRequest(domain, "", redirectedLocation, 0, 0, false, result)
		}
	}

	headers := make(map[string]string)
	for k, v := range resp.Header {
		headers[strings.ToLower(k)] = strings.ToLower(v[0])
	}

	title := ""
	if t.withTitle {
		title = detector.ExtractTitle(resp)
	}

	body := ""
	//if t.withBody {
	//	body = string(resp.Body())
	//}
	technologies := make(map[string]string)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err == nil {
		if t.withTechnology {
			technologies = t.getTechnologyMap(url, bodyBytes, resp.Header, resp.Cookies())
		}
	}

	result <- Result{
		StatusCode:   resp.StatusCode,
		Asset:        url,
		Domain:       domain,
		Body:         body,
		Headers:      headers,
		Title:        title,
		Technologies: technologies,
	}
}

func (t *tarantula) getTechnologyMap(url string, body []byte, headers http.Header, cookies []*http.Cookie) map[string]string {
	technologies := make(map[string]string)
	matches := t.technologyDetector.Technology(url, body, headers, cookies)
	for _, match := range matches {
		for _, cat := range match.CatNames {
			cat = strings.ToLower(cat)
			cat = strings.ReplaceAll(cat, " ", "-")
			technologies[cat] = strings.ToLower(match.AppName)
		}
	}
	return technologies
}
