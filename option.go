package mcr

import (
	"net"
	"time"
)

type Option func(cn *Client)

// apply a custom port number to the connection
func WithPort(port int) Option {
	return func(cn *Client) {
		cn.port = port
	}
}

// set a specific timeout for the underlying connection, this timeout applies to the
// resolution of addresses and establishing the initial connection
func WithTimeout(timeout time.Duration) Option {
	return func(cn *Client) {
		cn.timeout = timeout
	}
}

// set custom request ID cap, the request ID is the identifier used per-packet
// a max value is set to prevent any overflow issues
func WithCap(c int32) Option {
	return func(cn *Client) {
		cn.cap = c
	}
}

// enables use of a custom connection instead of the default one
func WithConnection(c net.Conn) Option {
	return func(cn *Client) {
		cn.connection = c
	}
}

// updates the current request id
func WithID(i int32) Option {
	return func(cn *Client) {
		cn.requestID = i
	}
}
