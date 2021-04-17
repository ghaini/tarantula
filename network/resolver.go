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
				return d.DialContext(ctx, "udp", randomDnsServer+":53")
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
		DialContext:         dialContext,
		MaxIdleConnsPerHost: -1,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			CipherSuites: []uint16{
				tls.TLS_RSA_WITH_RC4_128_SHA,
				tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
				tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
				tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			},
		},
		DisableKeepAlives: true,
	}
	return transport
}
