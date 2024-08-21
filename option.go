package mcr

import (
	"time"
)

// request option func skeleton
type Option func(cn *Client) error

// option to allow for custom timeout values
func WithTimeout(timeout time.Duration) Option {
	return func(cn *Client) error {
		cn.timeout = timeout
		return nil
	}
}
