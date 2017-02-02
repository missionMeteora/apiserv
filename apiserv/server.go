package apiserv

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/missionMeteora/apiv2/router"
)

// Options are options used in creating the server
type Options struct {
	ReadTimeout    time.Duration // see http.Server.ReadTimeout
	WriteTimeout   time.Duration // see http.Server.WriteTimeout
	MaxHeaderBytes int           // see http.Server.MaxHeaderBytes
	Logger         *log.Logger

	RouterOptions *router.Options // Additional options passed to the internal router.Router instance
}

// New returns a server
func New(opts interface{}) *Server {
	srv := &Server{
		//srv: &http.Server{},
		r: router.New(nil),
	}

	srv.r.PanicHandler = func(w http.ResponseWriter, req *http.Request, v interface{}) {
		NewErrorResponse(http.StatusInternalServerError, fmt.Sprint(v)).Output(w)
	}

	srv.r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(NewErrorResponse(http.StatusNotFound, "those damn cats stole the endpoint again."))
	})

	return srv
}

// Server is the main server
type Server struct {
	r    *router.Router
	opts Options

	ctxPool sync.Pool

	serversMux sync.Mutex
	servers    []*http.Server

	closed int32
}

// AddRoute adds a handler (or more) to the specific method and path
// it is NOT safe to call this once you call one of the run functions
func (s *Server) AddRoute(method, path string, handlers ...Handler) error {
	return s.r.AddRoute(method, path, handlerChain(handlers).Serve)
}

// Run starts the server on the specific address
func (s *Server) Run(addr string) error {
	srv := &http.Server{
		Addr:           addr,
		Handler:        s.r,
		ReadTimeout:    s.opts.ReadTimeout,
		WriteTimeout:   s.opts.WriteTimeout, // otherwise it'll time out on slow connections like mine
		MaxHeaderBytes: s.opts.MaxHeaderBytes,
		ErrorLog:       s.opts.Logger,
	}
	s.serversMux.Lock()
	s.servers = append(s.servers, srv)
	s.serversMux.Unlock()
	return srv.ListenAndServe()
}

// CertPair is a pair of (cert, key) files to listen on TLS
type CertPair struct {
	CertFile string `json:"certFile"`
	KeyFile  string `json:"KeyFile"`
}

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

	srv := &http.Server{
		Addr:           addr,
		Handler:        s.r,
		ReadTimeout:    s.opts.ReadTimeout,
		WriteTimeout:   s.opts.WriteTimeout, // otherwise it'll time out on slow connections like mine
		MaxHeaderBytes: s.opts.MaxHeaderBytes,
		ErrorLog:       s.opts.Logger,
		TLSConfig:      &cfg,
	}

	s.serversMux.Lock()
	s.servers = append(s.servers, srv)
	s.serversMux.Unlock()

	return srv.ListenAndServeTLS("", "")
}

// SetKeepAlivesEnabled controls whether HTTP keep-alives are enabled.
// By default, keep-alives are always enabled.
func (s *Server) SetKeepAlivesEnabled(v bool) {
	s.serversMux.Lock()
	for _, srv := range s.servers {
		srv.SetKeepAlivesEnabled(v)
	}
	s.serversMux.Unlock()
}

// Closed returns true if the server is already shutdown/closed
func (s *Server) Closed() bool {
	return atomic.LoadInt32(&s.closed) == 1
}
