// +build go1.8

package apiserv

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/missionMeteora/toolkit/errors"
)

// Close immediately closes all the active underlying http servers and connections.
func (s *Server) Close() error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return errors.ErrIsClosed
	}

	var el errors.ErrorList

	s.serversMux.Lock()
	for _, srv := range s.servers {
		srv.SetKeepAlivesEnabled(false)
		el.Push(srv.Close())
	}
	s.servers = nil
	s.serversMux.Unlock()

	return el.Err()
}

// Shutdown gracefully shutsdown all the underlying http servers.
// You can optionally set a timeout.
func (s *Server) Shutdown(timeout time.Duration) error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return errors.ErrIsClosed
	}

	var (
		el  errors.ErrorList
		ctx = context.Background()
	)

	if timeout > 0 {
		var cancelFn func()
		ctx, cancelFn = context.WithDeadline(ctx, time.Now().Add(timeout))
		defer cancelFn()
	}

	s.serversMux.Lock()
	for _, srv := range s.servers {
		srv.SetKeepAlivesEnabled(false)
		el.Push(srv.Shutdown(ctx))
	}
	s.servers = nil
	s.serversMux.Unlock()

	return el.Err()
}
