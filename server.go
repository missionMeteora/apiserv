package apiserv

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/missionMeteora/apiserv/router"
)

// DefaultOpts are the default options used for creating new servers.
var DefaultOpts = Options{
	WriteTimeout: time.Minute,
	ReadTimeout:  time.Minute,

	MaxHeaderBytes: 16 << 10, // 16kb

	KeepAlivePeriod: 3 * time.Minute, // default value in net/http

	Logger: log.New(os.Stderr, "apiserv: ", 0),
}

// New returns a new server with the specified options.
func New(opts ...Option) *Server {
	o := DefaultOpts

	for _, opt := range opts {
		opt.apply(&o)
	}

	return NewWithOpts(&o)
}

// NewWithOpts allows passing the Options struct directly
func NewWithOpts(opts *Options) *Server {
	srv := &Server{}

	if opts == nil {
		cp := DefaultOpts
		srv.opts = cp
	} else {
		srv.opts = *opts
	}

	ro := srv.opts.RouterOptions
	srv.r = router.New(ro)

	if ro == nil || !ro.NoCatchPanics {
		srv.r.PanicHandler = func(w http.ResponseWriter, req *http.Request, v interface{}) {
			srv.Logf("PANIC (%T): %v", v, v)
			if h := srv.PanicHandler; h != nil {
				ctx := getCtx(w, req, nil, srv)
				h(ctx, v)
				putCtx(ctx)
				return
			}

			resp := NewJSONErrorResponse(http.StatusInternalServerError, fmt.Sprintf("PANIC (%T): %v", v, v))
			resp.WriteToCtx(&Context{
				Req:            req,
				ResponseWriter: w,
			})
		}
	}

	srv.r.NotFoundHandler = func(w http.ResponseWriter, req *http.Request, p router.Params) {
		if h := srv.NotFoundHandler; h != nil {
			ctx := getCtx(w, req, p, srv)
			srv.NotFoundHandler(ctx)
			putCtx(ctx)
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
	*group
	r *router.Router

	PanicHandler    func(ctx *Context, v interface{})
	NotFoundHandler func(ctx *Context)

	servers    []*http.Server
	opts       Options
	serversMux sync.Mutex
	closed     int32
}

// ServeHTTP allows using the server in custom scenarios that expects an http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.r.ServeHTTP(w, req)
}

func (s *Server) newHTTPServer(addr string) *http.Server {
	opts := &s.opts
	return &http.Server{
		Addr:           addr,
		Handler:        s.r,
		ReadTimeout:    opts.ReadTimeout,
		WriteTimeout:   opts.WriteTimeout,
		MaxHeaderBytes: opts.MaxHeaderBytes,
		ErrorLog:       opts.Logger,
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

	if s.opts.KeepAlivePeriod < 1 {
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
	s.logfStack(3, f, args...)
}

func (s *Server) logfStack(n int, f string, args ...interface{}) {
	lg := s.opts.Logger
	if lg == nil {
		return
	}

	_, file, line, ok := runtime.Caller(n - 1)
	if !ok {
		file = "???"
		line = 0
	}

	// make it output the package owning the file
	parts := strings.Split(file, "/")
	if len(parts) > 2 {
		parts = parts[len(parts)-2:]
	}

	lg.Printf(strings.Join(parts, "/")+":"+strconv.Itoa(line)+": "+f, args...)
}

// AllowCORS is an alias for s.AddRoute("OPTIONS", path, AllowCORS(allowedMethods...))
func (s *Server) AllowCORS(path string, allowedMethods ...string) error {
	return s.AddRoute(http.MethodOptions, path, AllowCORS(allowedMethods, nil, nil))
}
