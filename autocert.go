package apiserv

import (
	"fmt"
	"os"

	"golang.org/x/crypto/acme/autocert"
)

// RunAutoCert enables automatic support for LetsEncrypt, using the optional passed domains list.
// certCacheDir is where the certificates will be cached, defaults to "./autocert".
// Note that it must always run on ":https" so the addr param is omitted.
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

	s.serversMux.Lock()
	s.servers = append(s.servers, srv)
	s.serversMux.Unlock()

	ln := m.Listener()

	if s.opts.KeepAlivePeriod == -1 {
		return srv.Serve(ln)
	}

	return srv.Serve(ln)
}
