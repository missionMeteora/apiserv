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

// RunTLSAndAuto allows using custom certificates and autocert together.
// It will always listen on both :80 and :443
func (s *Server) RunTLSAndAuto(certCacheDir string, certPairs []CertPair, domains []string) error {
	var (
		m = &autocert.Manager{
			Prompt: autocert.AcceptTOS,
		}
		domMap map[string]bool
	)

	if len(domains) > 0 {
		m.HostPolicy = autocert.HostWhitelist(domains...)
		domMap = make(map[string]bool, len(domains))
		for _, dom := range domains {
			domMap[dom] = true
		}
	}

	if certCacheDir == "" {
		certCacheDir = "./autocert"
	}

	if err := os.MkdirAll(certCacheDir, 0700); err != nil {
		return fmt.Errorf("couldn't create cert cache dir: %v", err)
	}

	m.Cache = autocert.DirCache(certCacheDir)

	srv := s.newHTTPServer(":https")

	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,

		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},

		NextProtos: []string{"h2", "http/1.1"}, // Enable HTTP/2

		GetCertificate: m.GetCertificate,
	}

	for _, cp := range certPairs {
		cert, err := tls.LoadX509KeyPair(cp.CertFile, cp.KeyFile)
		if err != nil {
			return fmt.Errorf("%s: %v", cp.CertFile, err)
		}
		cfg.Certificates = append(cfg.Certificates, cert)
	}

	cfg.BuildNameToCertificate()

	cfg.GetCertificate = func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		if domMap[hello.ServerName] {
			return m.GetCertificate(hello)
		}

		// fallback to default impl
		return nil, nil
	}

	srv.TLSConfig = cfg

	s.serversMux.Lock()
	s.servers = append(s.servers, srv)
	s.serversMux.Unlock()

	ch := make(chan error, 2)

	go func() {
		if err := http.ListenAndServe(":80", m.HTTPHandler(nil)); err != nil {
			s.Logf("apiserv: autocert on :80 error: %v", err)
			ch <- err
		}
	}()

	go func() {
		if err := srv.ListenAndServeTLS("", ""); err != nil {
			s.Logf("apiserv: autocert on :443 error: %v", err)
			ch <- err
		}
	}()

	return <-ch
}
