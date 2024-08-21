package mcr

import "net"

// request option func skeleton
type Option func(cn *Client) error

// option to allow for custom tcp client for testing
func WithClient(cli net.Conn) Option {
	return func(cn *Client) error {
		cn.connection = cli
		return nil
	}
}
