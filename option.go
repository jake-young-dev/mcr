package mcr

import (
	"net"
	"time"
)

// request option func skeleton
type Option func(cn *client)

// option to allow for custom port values
func WithPort(port int) Option {
	return func(cn *client) {
		cn.port = port
	}
}

// option to allow for custom timeout values
func WithTimeout(timeout time.Duration) Option {
	return func(cn *client) {
		cn.timeout = timeout
	}
}

// option to allow for custom request id cap
func WithCap(c int32) Option {
	return func(cn *client) {
		cn.cap = c
	}
}

// option to allow for use of custom connections
func WithConnection(c net.Conn) Option {
	return func(cn *client) {
		cn.connection = c
	}
}
