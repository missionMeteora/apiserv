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

// DefaultOpts are the default options used for creating new servers.
var DefaultOpts = options{
	WriteTimeout: time.Minute,
	ReadTimeout:  time.Minute,

	MaxHeaderBytes: 16 << 10, // 16kb

	Logger: log.New(os.Stderr, "APIServer: ", log.Lshortfile),
}

// New returns a new server with the specified options.
func New(opts ...OptionCallback) *Server {
	srv := &Server{
		opts: DefaultOpts,
	}

	for _, fn := range opts {
		fn(&srv.opts)
	}

	srv.r = router.New(srv.opts.RouterOptions)

	srv.r.PanicHandler = func(w http.ResponseWriter, req *http.Request, v interface{}) {
		srv.Logf("PANIC (%T): %v", v, v)
		ctx := getCtx(w, req, nil)
		defer putCtx(ctx)

		if srv.PanicHandler != nil {
			srv.PanicHandler(ctx, v)
			return
		}

		resp := NewErrorResponse(http.StatusInternalServerError, fmt.Sprintf("PANIC (%T): %v", v, v))
		resp.WriteToCtx(ctx)
	}

	srv.r.NotFoundHandler = func(w http.ResponseWriter, req *http.Request, p router.Params) {
		ctx := getCtx(w, req, p)
		defer putCtx(ctx)

		if srv.NotFoundHandler != nil {
			srv.NotFoundHandler(ctx)
			return
		}

		RespNotFound.WriteToCtx(&Context{
			Req:            req,
			ResponseWriter: w,
		})
	}

	srv.group = &group{s: srv}

	return srv
}

// Server is the main server
type Server struct {
	r    *router.Router
	opts options

	serversMux sync.Mutex
	servers    []*http.Server

	closed int32

	PanicHandler    func(ctx *Context, v interface{})
	NotFoundHandler func(ctx *Context)

	*group
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
