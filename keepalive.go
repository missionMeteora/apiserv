package apiserv

import (
	"net"
	"time"
)

// tcpKeepAliveListener copied from net/http to allow a custom keepalive period, useful for testing
type tcpKeepAliveListener struct {
	*net.TCPListener
	period time.Duration
}

func (ln *tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(ln.period)
	return tc, nil
}
