// +build !fasthttp

package router

import (
	"fmt"
	"net/http"
	"path"
)

// Handler is what handler looks like, duh?
// *note* `p` is NOT safe to be used outside the handler, call p.Copy() if you need to use it.
type Handler func(w http.ResponseWriter, req *http.Request, p Params)

// PanicHandler is a special handler that gets called if a panic happens
type PanicHandler func(w http.ResponseWriter, req *http.Request, v interface{})

// DefaultPanicHandler is the default panic handler
func DefaultPanicHandler(w http.ResponseWriter, req *http.Request, v interface{}) {
	http.Error(w, fmt.Sprintf("panic (%T): %v", v, v), http.StatusInternalServerError)
}

// DefaultNotFoundHandler is the default panic handler
func DefaultNotFoundHandler(w http.ResponseWriter, req *http.Request, _ Params) {
	http.Error(w, "404 page not found", http.StatusNotFound)
}

// ServerHTTP implements http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// if !r.opts.NoCatchPanics {
	// 	defer func() {
	// 		if v := recover(); v != nil && r.PanicHandler != nil {
	// 			r.PanicHandler(w, req, v)
	// 		}
	// 	}()
	// }

	u := req.URL.EscapedPath()

	if !r.opts.NoAutoCleanURL {
		u = path.Clean(u)
	}

	if h, p := r.match(req.Method, u); h != nil {
		h(w, req, p.Params())
		r.putParams(p)
	} else if r.NotFoundHandler != nil {
		r.NotFoundHandler(w, req, nil)
	}
}
