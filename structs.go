package tarantula

type Result struct {
	StatusCode   int
	Asset        string
	Domain       string
	Body         string
	Headers      map[string]string
	Technologies map[string]string
	Title        string
}

type input struct {
	Subdomain string
	Port      int
}

type Technology struct {
	Name       string
	Categories []string
	Website    string
}
