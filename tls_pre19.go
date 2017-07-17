// +build !go1.9

package apiserv

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"
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

	srv := s.newHTTPServer(addr)
	srv.TLSConfig = &cfg

	s.serversMux.Lock()
	s.servers = append(s.servers, srv)
	s.serversMux.Unlock()

	if s.opts.KeepAlivePeriod != 3*time.Minute {
		s.Logf("KeepAlivePeriod is not supported on go < 1.9")
	}

	return srv.ListenAndServeTLS("", "")
}
