package mcr

import (
	"net"
	"time"
)

// request option func skeleton
type Option func(cn *Client)

// option to allow for custom port values
func WithPort(port int) Option {
	return func(cn *Client) {
		cn.port = port
	}
}

// option to allow for custom timeout values
func WithTimeout(timeout time.Duration) Option {
	return func(cn *Client) {
		cn.timeout = timeout
	}
}

// option to allow for custom request id cap
func WithCap(c int32) Option {
	return func(cn *Client) {
		cn.cap = c
	}
}

// option to allow for use of custom connections
func WithConnection(c net.Conn) Option {
	return func(cn *Client) {
		cn.connection = c
	}
}
