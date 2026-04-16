// Package webutil provides shared HTTP client infrastructure for the Anansi
// web crawler. Centralises transport configuration so all packages use
// consistent connection pooling, timeouts, and TLS settings.
package webutil

import (
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	sharedTransport *http.Transport
	transportOnce   sync.Once
)

// Transport returns the singleton http.Transport configured for long-lived
// crawling. Safe for concurrent use - initialized once on first call.
// All http.Clients should wrap this shared transport.
//
// Configuration rationale:
//   - MaxIdleConns: total idle connections across all hosts.
//   - MaxIdleConnsPerHost: single-domain crawler, so this equals MaxIdleConns.
//   - IdleConnTimeout: how long to keep idle connections before closing.
//   - TLSHandshakeTimeout: max wait for TLS negotiation.
//   - ResponseHeaderTimeout: max wait for response headers after request sent.
//   - DialContext timeout: max wait for TCP connection.
func Transport() *http.Transport {
	transportOnce.Do(func() {
		sharedTransport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   dialTimeout,
				KeepAlive: keepAlive,
			}).DialContext,
			MaxIdleConns:          maxIdleConns,
			MaxIdleConnsPerHost:   maxIdleConnsPerHost,
			IdleConnTimeout:       idleConnTimeout,
			TLSHandshakeTimeout:   tlsHandshakeTimeout,
			ResponseHeaderTimeout: responseHeaderTimeout,
			ForceAttemptHTTP2:     true,
		}
	})

	return sharedTransport
}

// NewClient creates an http.Client with the given timeout backed by the
// singleton transport. Each goroutine/worker should have its own Client
// wrapping the shared Transport.
//
// TODO: Consider making redirect policy configurable at the Client level instead of Crawler level.
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: Transport(),
	}
}
