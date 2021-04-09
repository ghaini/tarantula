package network

import (
	"bufio"
	"context"
	"net"
	"net/http"
)

func HTTPProxyDialer(proxyAddr string)  func(ctx context.Context, network, addr string) (net.Conn, error)  {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		c, err := net.Dial("tcp", proxyAddr)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("CONNECT", addr, nil)
		if err != nil {
			c.Close()
			return nil, err
		}
		req.Close = false
		err = req.Write(c)
		if err != nil {
			c.Close()
			return nil, err
		}

		resp, err := http.ReadResponse(bufio.NewReader(c), req)
		if err != nil {
			c.Close()
			return nil, err
		}
		resp.Body.Close()
		return c, nil
	}
}
