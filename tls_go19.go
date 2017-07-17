// +build go1.9

package apiserv

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
)

// RunTLS starts the server on the specific address, using tls
func (s *Server) RunTLS(addr string, certPairs []CertPair) error {
	cfg := tls.Config{RootCAs: x509.NewCertPool()}
	cfg.Certificates = make([]tls.Certificate, 0, len(certPairs))

	for _, cp := range certPairs {
		cert, err := tls.LoadX509KeyPair(cp.CertFile, cp.KeyFile)
		if err != nil {
			return fmt.Errorf("%s: %v", cp.CertFile, err)
		}
		cfg.Certificates = append(cfg.Certificates, cert)
	}

	cfg.BuildNameToCertificate()

	if addr == "" {
		addr = ":https"
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	srv := s.newHTTPServer(ln.Addr().String())
	srv.TLSConfig = &cfg

	s.serversMux.Lock()
	s.servers = append(s.servers, srv)
	s.serversMux.Unlock()

	if s.opts.KeepAlivePeriod == -1 {
		return srv.ServeTLS(ln, "", "")
	}

	return srv.ServeTLS(&tcpKeepAliveListener{ln.(*net.TCPListener), s.opts.KeepAlivePeriod}, "", "")
}
