package network

import (
	"bufio"
	"bytes"
	"github.com/ghaini/tarantula/constants"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
	"math/rand"
	"net"
)

func DialerWithCustomDNSResolver() fasthttp.DialFunc {
	var dnsServers []string
	_, body, _ := fasthttp.Get(nil, constants.DNSServerList)
	r := bytes.NewReader(body)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		dnsServers = append(dnsServers, scanner.Text())
	}

	return func(addr string) (net.Conn, error) {
		randomDnsServer := dnsServers[rand.Intn(len(dnsServers))]
		var dialer = &fasthttp.TCPDialer{
			Resolver: &net.Resolver{
				PreferGo:     true,
				StrictErrors: false,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{}
					return d.DialContext(ctx, "udp", randomDnsServer + ":53")
				},
			},
		}
		return dialer.Dial(addr)
	}
}
