package constants

// protocols
const (
	HTTP  = "http"
	HTTPS = "https"
)

const TechnologiesFileAddress = "https://raw.githubusercontent.com/ghaini/tarantula/master/data/technologies.json"
const DNSServerList = "https://raw.githubusercontent.com/ghaini/tarantula/master/data/resolvers.txt"

var PortsProtocols = map[int]string{
	80: HTTP,
	443: HTTPS,
}
