// +build fasthttp

package router

import (
	"fmt"
	"net/http"
	"path"

	"github.com/valyala/fasthttp"
)

// Handler is what handler looks like, duh?
// *note* `p` is NOT safe to be used outside the handler, call p.Copy() if you need to use it.
type Handler func(ctx *fasthttp.RequestCtx, p Params)

// PanicHandler is a special handler that gets called if a panic happens
type PanicHandler func(ctx *fasthttp.RequestCtx, v interface{})

// DefaultPanicHandler is the default panic handler
func DefaultPanicHandler(ctx *fasthttp.RequestCtx, v interface{}) {
	ctx.SetStatusCode(http.StatusInternalServerError)
	fmt.Fprintf(ctx, "panic (%T): %v\n", v, v)
}

// DefaultNotFoundHandler is the default panic handler
func DefaultNotFoundHandler(ctx *fasthttp.RequestCtx, _ Params) {
	ctx.SetStatusCode(http.StatusNotFound)
	fmt.Fprintf(ctx, "404 page not found\n")
}

// ServerFastHTTP implements fasthttp handler
func (r *Router) ServeFastHTTP(ctx *fasthttp.RequestCtx) {
	if !r.opts.NoCatchPanics {
		defer func() {
			if v := recover(); v != nil && r.PanicHandler != nil {
				r.PanicHandler(ctx, v)
			}
		}()
	}

	u := string(ctx.Path())

	if !r.opts.NoAutoCleanURL {
		u = path.Clean(u)
	}

	if h, p := r.Match(string(ctx.Method()), u); h != nil {
		h(ctx, p)
		r.putParams(p)
	} else if r.NotFoundHandler != nil {
		r.NotFoundHandler(ctx, nil)
	}
}
