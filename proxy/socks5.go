package proxy

import (
	"github.com/valyala/fasthttp"
	p "golang.org/x/net/proxy"
	"net"
)

func SocksDialer(proxyAddr string) fasthttp.DialFunc {
	dialer, err := p.SOCKS5("tcp", proxyAddr, nil, p.Direct)

	return func(addr string) (net.Conn, error) {
		if err != nil {
			return nil, err
		}
		return dialer.Dial("tcp", addr)
	}
}
