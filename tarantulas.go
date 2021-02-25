package tarantulas

import (
	"fmt"
	"github.com/valyala/fasthttp"
)

type Result struct {
	Asset   string
	Domain  string
	Body    string
	Headers map[string]interface{}
}

func GetContents(domain string, subdomains []string, thread int, httpPorts, httpsPorts []int) {
	isListening(subdomains[0])
}

func isListening(url string) bool {

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req) // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	req.SetRequestURI(url)

	_ = fasthttp.Do(req, resp)

	bodyBytes := resp.Body()
	fmt.Println(string(bodyBytes))

	return true

}
