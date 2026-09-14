// Package dialer builds outbound dialers for SOCKS5 and HTTP CONNECT proxies.
package dialer

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// httpConnectDialer dials TCP connections through an HTTP proxy using CONNECT.
type httpConnectDialer struct {
	proxyAddr string
}

func (d *httpConnectDialer) Dial(network, addr string) (net.Conn, error) {
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return nil, fmt.Errorf("network %q not supported by HTTP CONNECT dialer", network)
	}

	conn, err := net.Dial("tcp", d.proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("connecting to HTTP proxy %s: %w", d.proxyAddr, err)
	}

	// Bound the handshake so a stuck proxy can't hang the dial forever.
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: make(http.Header),
	}
	if err := req.Write(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("sending CONNECT %s to %s: %w", addr, d.proxyAddr, err)
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, req)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("reading CONNECT response from %s: %w", d.proxyAddr, err)
	}
	if resp.StatusCode != http.StatusOK {
		conn.Close()
		return nil, fmt.Errorf("proxy %s rejected CONNECT %s: %s", d.proxyAddr, addr, resp.Status)
	}

	conn.SetDeadline(time.Time{})
	return &bufferedConn{Conn: conn, r: br}, nil
}

// bufferedConn replays any tunnel bytes buffered while parsing the CONNECT
// response, so nothing read past the header is lost.
type bufferedConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) { return c.r.Read(p) }

// Select builds the outbound dialer from proxy addresses. Exactly one of
// socks/httpProxy should be non-empty; both empty means direct connection and
// returns a nil dialer. Both set is an error.
func Select(socks, httpProxy string) (proxy.Dialer, string, error) {
	switch {
	case socks != "" && httpProxy != "":
		return nil, "", fmt.Errorf("--socks and --http are mutually exclusive; pass only one")
	case httpProxy != "":
		return &httpConnectDialer{proxyAddr: httpProxy}, "http proxy " + httpProxy, nil
	case socks != "":
		d, err := proxy.SOCKS5("tcp", socks, nil, proxy.Direct)
		if err != nil {
			return nil, "", fmt.Errorf("SOCKS5 dialer error: %w", err)
		}
		return d, "socks5 proxy " + socks, nil
	default:
		return nil, "", nil
	}
}
