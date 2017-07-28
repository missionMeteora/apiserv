package apiserv

import (
	"fmt"
	"log"
	"net"
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

	KeepAlivePeriod: 3 * time.Minute, // default value in net/http

	Logger: log.New(os.Stderr, "apiserv: ", log.Lshortfile),
}

// New returns a new server with the specified options.
func New(opts ...OptionCallback) *Server {
	srv := &Server{
		opts: DefaultOpts,
	}

	for _, fn := range opts {
		fn(&srv.opts)
	}
	ro := srv.opts.RouterOptions
	srv.r = router.New(ro)

	srv.r.PanicHandler = func(w http.ResponseWriter, req *http.Request, v interface{}) {
		srv.Logf("PANIC (%T): %v", v, v)
		ctx := getCtx(w, req, nil)
		defer putCtx(ctx)

		if srv.PanicHandler != nil {
			srv.PanicHandler(ctx, v)
			return
		}

		resp := NewJSONErrorResponse(http.StatusInternalServerError, fmt.Sprintf("PANIC (%T): %v", v, v))
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

func (s *Server) newHTTPServer(addr string) *http.Server {
	return &http.Server{
		Addr:           addr,
		Handler:        s.r,
		ReadTimeout:    s.opts.ReadTimeout,
		WriteTimeout:   s.opts.WriteTimeout,
		MaxHeaderBytes: s.opts.MaxHeaderBytes,
		ErrorLog:       s.opts.Logger,
	}
}

// Run starts the server on the specific address
func (s *Server) Run(addr string) error {
	if addr == "" {
		addr = ":http"
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	srv := s.newHTTPServer(ln.Addr().String())

	s.serversMux.Lock()
	s.servers = append(s.servers, srv)
	s.serversMux.Unlock()

	if s.opts.KeepAlivePeriod == -1 {
		return srv.Serve(ln)
	}

	return srv.Serve(&tcpKeepAliveListener{ln.(*net.TCPListener), s.opts.KeepAlivePeriod})
}

// CertPair is a pair of (cert, key) files to listen on TLS
type CertPair struct {
	CertFile string `json:"certFile"`
	KeyFile  string `json:"KeyFile"`
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

// Addrs returns all the listening addresses used by the underlying http.Server(s).
func (s *Server) Addrs() (out []string) {
	s.serversMux.Lock()
	out = make([]string, len(s.servers))
	for i, srv := range s.servers {
		out[i] = srv.Addr
	}
	s.serversMux.Unlock()
	return
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

// AllowCORS is an alias for s.AddRoute("OPTIONS", path, AllowCORS(allowedMethods...))
func (s *Server) AllowCORS(path string, allowedMethods ...string) error {
	return s.AddRoute("OPTIONS", path, AllowCORS(allowedMethods...))
}
