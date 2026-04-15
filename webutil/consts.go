package webutil

import "time"

const (
	// Dial and keep-alive settings.
	dialTimeout = 10 * time.Second
	keepAlive   = 30 * time.Second

	// Connection pool settings.
	// Single-domain crawler — per-host limit equals total limit.
	maxIdleConns        = 100
	maxIdleConnsPerHost = 100
	idleConnTimeout     = 90 * time.Second

	// TLS and response timeouts.
	tlsHandshakeTimeout   = 10 * time.Second
	responseHeaderTimeout = 10 * time.Second
)
