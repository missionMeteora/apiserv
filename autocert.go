package apiserv

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/crypto/acme/autocert"
)

// RunAutoCert enables automatic support for LetsEncrypt, using the optional passed domains list.
// certCacheDir is where the certificates will be cached, defaults to "./autocert".
// Note that it must always run on *BOTH* ":80" and ":443" so the addr param is omitted.
func (s *Server) RunAutoCert(certCacheDir string, domains ...string) error {
	m := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
	}

	if len(domains) > 0 {
		m.HostPolicy = autocert.HostWhitelist(domains...)
	}

	if certCacheDir == "" {
		certCacheDir = "./autocert"
	}

	if err := os.MkdirAll(certCacheDir, 0700); err != nil {
		return fmt.Errorf("couldn't create cert cache dir: %v", err)
	}

	m.Cache = autocert.DirCache(certCacheDir)

	srv := s.newHTTPServer(":https")

	srv.TLSConfig = &tls.Config{
		GetCertificate: m.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1"}, // Enable HTTP/2
	}

	s.serversMux.Lock()
	s.servers = append(s.servers, srv)
	s.serversMux.Unlock()

	go func() {
		if err := http.ListenAndServe(":80", m.HTTPHandler(nil)); err != nil {
			s.Logf("apiserv: autocert on :80 error: %v", err)
		}
	}()

	return srv.ListenAndServeTLS("", "")
}
