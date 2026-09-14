// Package app wires configuration to the proxy server loop.
package app

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"pathrelay/internal/dialer"
)

// Config holds the runtime configuration of the proxy.
type Config struct {
	Listen string // listen address, e.g. ":80"
	Target string // upstream base URL, e.g. "https://mydomain:2096"
	Socks  string // SOCKS5 proxy address; empty = unused
	HTTP   string // HTTP CONNECT proxy address; empty = unused
}

// Validate checks the configuration for obvious problems.
func (c Config) Validate() error {
	if c.Listen == "" {
		return fmt.Errorf("--listen must not be empty")
	}
	if c.Target == "" {
		return fmt.Errorf("--target must not be empty")
	}
	if c.Socks != "" && c.HTTP != "" {
		return fmt.Errorf("--socks and --http are mutually exclusive; pass only one")
	}
	return nil
}

// Run validates the config, starts the reverse proxy and blocks until the
// listener fails or is closed.
func (c Config) Run() error {
	if err := c.Validate(); err != nil {
		return err
	}

	outbound, via, err := dialer.Select(c.Socks, c.HTTP)
	if err != nil {
		return err
	}

	transport := &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}
	if outbound != nil {
		transport.Dial = func(network, addr string) (net.Conn, error) {
			return outbound.Dial(network, addr)
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		targetURL := c.Target + r.URL.RequestURI()

		req, err := http.NewRequest(r.Method, targetURL, r.Body)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		for k, vv := range r.Header {
			if k == "Host" || k == "Connection" || k == "Transfer-Encoding" {
				continue
			}
			for _, v := range vv {
				req.Header.Add(k, v)
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Upstream error: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})

	log.Printf("pathrelay listening on %s", c.Listen)
	log.Printf("  target → %s", c.Target)
	if via == "" {
		via = "direct (no outbound proxy)"
	}
	log.Printf("  via    → %s", via)

	return http.ListenAndServe(c.Listen, nil)
}
