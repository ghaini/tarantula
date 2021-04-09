package network

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"github.com/ghaini/tarantula/constants"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
	"math/rand"
	"net"
	"net/http"
)

func DialerWithCustomDNSResolver() func(ctx context.Context, network, addr string) (net.Conn, error) {
	var dnsServers []string
	_, body, _ := fasthttp.Get(nil, constants.DNSServerList)
	r := bytes.NewReader(body)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		dnsServers = append(dnsServers, scanner.Text())
	}

	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
				}
				randomDnsServer := dnsServers[rand.Intn(len(dnsServers))]

				return d.DialContext(ctx, "udp", randomDnsServer + ":53")
			},
		},
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, addr)
	}

	return dialContext
}

func DefaultTransport(dialContext func(ctx context.Context, network, addr string) (net.Conn, error)) *http.Transport {
	transport := &http.Transport{
		DialContext: dialContext,
		MaxIdleConnsPerHost: -1,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DisableKeepAlives: true,
	}
	return transport
}
