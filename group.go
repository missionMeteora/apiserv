package apiserv

import (
	"net/http"

	"github.com/missionMeteora/apiserv/router"
)

// Group represents a handler group (aka supports middleware, gin's .Use)
type Group struct {
	mw []Handler
	s  *Server
}

// Use adds more middleware to the current group.
func (g *Group) Use(mw ...Handler) {
	g.mw = append(g.mw, mw...)
}

// AddRoute adds a handler (or more) to the specific method and path
// it is NOT safe to call this once you call one of the run functions
func (g *Group) AddRoute(method, path string, handlers ...Handler) error {
	ghc := groupHandlerChain{
		hc: handlers,
		g:  g,
	}
	return g.s.r.AddRoute(method, path, ghc.Serve)
}

// GET is an alias for AddRoute("GET", path, handlers...).
func (g *Group) GET(path string, handlers ...Handler) error {
	return g.AddRoute("GET", path, handlers...)
}

// PUT is an alias for AddRoute("PUT", path, handlers...).
func (g *Group) PUT(path string, handlers ...Handler) error {
	return g.AddRoute("PUT", path, handlers...)
}

// POST is an alias for AddRoute("POST", path, handlers...).
func (g *Group) POST(path string, handlers ...Handler) error {
	return g.AddRoute("POST", path, handlers...)
}

// DELETE is an alias for AddRoute("DELETE", path, handlers...).
func (g *Group) DELETE(path string, handlers ...Handler) error {
	return g.AddRoute("DELETE", path, handlers...)
}

// Group returns a sub-handler group based on the current group's middleware
func (g *Group) Group(mw ...Handler) *Group {
	return &Group{
		mw: append(g.mw[:len(g.mw):len(g.mw)], mw...),
		s:  g.s,
	}
}

type groupHandlerChain struct {
	hc []Handler
	g  *Group
}

func (ghc *groupHandlerChain) Serve(rw http.ResponseWriter, req *http.Request, p router.Params) {
	ctx := getCtx(rw, req, p)
	defer putCtx(ctx)

	for _, h := range ghc.g.mw {
		if r := h(ctx); r != nil {
			if !ctx.done && r != Break {
				r.WriteToCtx(ctx)
			}
			return
		}
	}

	for _, h := range ghc.hc {
		if r := h(ctx); r != nil {
			if !ctx.done && r != Break {
				r.WriteToCtx(ctx)
			}
			return
		}
	}
}
