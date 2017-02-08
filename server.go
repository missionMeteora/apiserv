package apiserv

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/missionMeteora/apiserv/router"
)

var defaultOpts = &Options{
	WriteTimeout: time.Minute,
	ReadTimeout:  time.Minute,

	MaxHeaderBytes: 16 << 10, // 16kb

	Logger: log.New(os.Stderr, "APIServer: ", log.Lshortfile),
}

// Options are options used in creating the server
type Options struct {
	ReadTimeout    time.Duration // see http.Server.ReadTimeout
	WriteTimeout   time.Duration // see http.Server.WriteTimeout
	MaxHeaderBytes int           // see http.Server.MaxHeaderBytes
	Logger         *log.Logger

	RouterOptions *router.Options // Additional options passed to the internal router.Router instance
}

// New returns a server
func New(opts *Options) *Server {
	if opts == nil {
		opts = defaultOpts
	}

	srv := &Server{
		opts: opts,
		r:    router.New(opts.RouterOptions),
	}

	srv.r.PanicHandler = func(w http.ResponseWriter, req *http.Request, v interface{}) {
		resp := NewErrorResponse(http.StatusInternalServerError, fmt.Sprintf("%T: %v", v, v))
		resp.Output(&Context{
			Req:            req,
			ResponseWriter: w,
		})
		srv.Logf("PANIC (%T): %v", v, v)
	}

	srv.r.NotFoundHandler = func(w http.ResponseWriter, req *http.Request, _ router.Params) {
		RespNotFound.Output(&Context{
			Req:            req,
			ResponseWriter: w,
		})
	}

	return srv
}

// Server is the main server
type Server struct {
	r    *router.Router
	opts *Options

	serversMux sync.Mutex
	servers    []*http.Server

	closed int32
}

// AddRoute adds a handler (or more) to the specific method and path
// it is NOT safe to call this once you call one of the run functions
func (s *Server) AddRoute(method, path string, handlers ...Handler) error {
	return s.r.AddRoute(method, path, handlerChain(handlers).Serve)
}

// GET is an alias for AddRoute("GET", path, handlers...).
func (s *Server) GET(path string, handlers ...Handler) error {
	return s.AddRoute("GET", path, handlers...)
}

// PUT is an alias for AddRoute("PUT", path, handlers...).
func (s *Server) PUT(path string, handlers ...Handler) error {
	return s.AddRoute("PUT", path, handlers...)
}

// POST is an alias for AddRoute("POST", path, handlers...).
func (s *Server) POST(path string, handlers ...Handler) error {
	return s.AddRoute("POST", path, handlers...)
}

// DELETE is an alias for AddRoute("DELETE", path, handlers...).
func (s *Server) DELETE(path string, handlers ...Handler) error {
	return s.AddRoute("DELETE", path, handlers...)
}

// ServeHTTP allows using the server in custom scenarios that expects an http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.r.ServeHTTP(w, req)
}

// Run starts the server on the specific address
func (s *Server) Run(addr string) error {
	srv := &http.Server{
		Addr:           addr,
		Handler:        s.r,
		ReadTimeout:    s.opts.ReadTimeout,
		WriteTimeout:   s.opts.WriteTimeout,
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

// Logf logs to the default server logger if set
func (s *Server) Logf(f string, args ...interface{}) {
	if s.opts.Logger != nil {
		s.opts.Logger.Printf(f, args...)
	}
}
