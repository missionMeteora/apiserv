// +build go1.8

package apiserv

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

// Close immediately closes all the active underlying http servers and connections.
func (s *Server) Close() error {
	var me MultiError
	s.serversMux.Lock()
	for _, srv := range s.servers {
		srv.SetKeepAlivesEnabled(false)
		if err := srv.Close(); err != nil {
			err = fmt.Errorf("%s (%T): %s", srv.Addr, err, err)
			me.Push(err)
		}
	}

	s.servers = nil
	s.serversMux.Unlock()

	return me.Err()
}

// Shutdown gracefully shutsdown all the underlying http servers.
// You can optionally set a timeout.
func (s *Server) Shutdown(timeout time.Duration) error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return http.ErrServerClosed
	}

	var (
		me  MultiError
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
		me.Push(srv.Shutdown(ctx))
	}
	s.servers = nil
	s.serversMux.Unlock()

	return me.Err()
}
