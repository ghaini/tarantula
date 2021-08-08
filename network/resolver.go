package network

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"github.com/ghaini/tarantula/constants"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
)

type Resolver struct {
	DNSServers []string
}

func NewResolver() *Resolver  {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	if _, err := os.Stat(home + "/.tarantula/resolvers.txt"); os.IsNotExist(err) {
		getResolverListFile()
	}

	home, err = os.UserHomeDir()
	if err != nil {
		return nil
	}
	resolver := &Resolver{}
	dnsServersFile, err := os.Open(home + "/.tarantula/resolvers.txt")
	if err != nil {
		return nil
	}

	scanner := bufio.NewScanner(dnsServersFile)
	for scanner.Scan() {
		resolver.DNSServers = append(resolver.DNSServers, scanner.Text())
	}
	defer dnsServersFile.Close()

	return &Resolver{}
}

func (r *Resolver) DialerWithCustomDNSResolver() func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
				}
				randomDnsServer := r.DNSServers[rand.Intn(len(r.DNSServers))]
				return d.DialContext(ctx, "udp", randomDnsServer+":53")
			},
		},
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, addr)
	}

	return dialContext
}

func (r *Resolver) DefaultTransport(dialContext func(ctx context.Context, network, addr string) (net.Conn, error)) *http.Transport {
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

func getResolverListFile() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	_ = os.MkdirAll(home+"/.tarantula", 0755)
	appsFile, err := os.Create(home + "/.tarantula/resolvers.txt")
	if err != nil {
		return err
	}

	defer appsFile.Close()
	_, resp, err := fasthttp.Get(nil, constants.DNSServerList)
	if err != nil {
		return err
	}

	r := bytes.NewReader(resp)
	_, err = io.Copy(appsFile, r)
	if err != nil {
		return err
	}

	return nil
}