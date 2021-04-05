package network

import (
	"bufio"
	"github.com/valyala/fasthttp"
	"net"
)

func HTTPProxyDialer(proxyAddr string) fasthttp.DialFunc {
	return func(addr string) (net.Conn, error) {
		conn, err := fasthttp.Dial(proxyAddr)
		if err != nil {
			return nil, err
		}

		req := "CONNECT " + addr + " HTTP/1.1\r\n"
		// req += "Proxy-Authorization: xxx\r\n"
		req += "\r\n"

		if _, err := conn.Write([]byte(req)); err != nil {
			return nil, err
		}

		res := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(res)

		if err := res.Read(bufio.NewReader(conn)); err != nil {
			conn.Close()
			return nil, err
		}

		return conn, nil
	}
}
