package tarantulas

type Result struct {
	Asset   string
	Domain  string
	Body    string
	Headers map[string]interface{}
}

func GetContents(domain string, subdomains []string, thread int, httpPorts, httpsPorts []int) {

}
