// +build !go1.8

package apiserv

import (
	"errors"
	"sync/atomic"
	"time"
)

const tooOld = errors.New("go < 1.8 doesn't support exiting :(")

// Close immediately closes all the active underlying http servers and connections.
func (s *Server) Close() error {
	atomic.CompareAndSwapInt32(&s.closed, 0, 1)
	return tooOld
}

// Shutdown gracefully shutsdown all the underlying http servers.
func (s *Server) Shutdown(timeout time.Duration) error {
	atomic.CompareAndSwapInt32(&s.closed, 0, 1)
	return tooOld
}
