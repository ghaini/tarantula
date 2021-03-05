package tarantulas

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
