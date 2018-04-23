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
	var tc *net.TCPConn

	if tc, err = ln.AcceptTCP(); err != nil {
		return
	}
	if err = tc.SetKeepAlive(true); err != nil {
		return
	}
	if err = tc.SetKeepAlivePeriod(ln.period); err != nil {
		return
	}

	return tc, nil
}
