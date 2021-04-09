package network

import (
	"golang.org/x/net/context"
	p "golang.org/x/net/proxy"
	"net"
)

func SocksDialer(proxyAddr string) func(ctx context.Context, network, addr string) (net.Conn, error) {
	d, _ := p.SOCKS5("tcp", proxyAddr, nil, p.Direct)
	contextDialer := d.(p.ContextDialer)
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return contextDialer.DialContext(context.Background(), "tcp", addr)
	}
}
