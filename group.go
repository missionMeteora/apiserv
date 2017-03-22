package apiserv

import (
	"net/http"
	"strings"

	"github.com/missionMeteora/apiserv/router"
)

// group represents a handler group (aka supports middleware, gin's .Use)
type group struct {
	mw   []Handler
	path string
	s    *Server
}

// Use adds more middleware to the current group.
func (g *group) Use(mw ...Handler) {
	g.mw = append(g.mw, mw...)
}

// AddRoute adds a handler (or more) to the specific method and path
// it is NOT safe to call this once you call one of the run functions
func (g *group) AddRoute(method, path string, handlers ...Handler) error {
	ghc := groupHandlerChain{
		hc: handlers,
		g:  g,
	}
	return g.s.r.AddRoute(method, joinPath(g.path, path), ghc.Serve)
}

// GET is an alias for AddRoute("GET", path, handlers...).
func (g *group) GET(path string, handlers ...Handler) error {
	return g.AddRoute("GET", path, handlers...)
}

// PUT is an alias for AddRoute("PUT", path, handlers...).
func (g *group) PUT(path string, handlers ...Handler) error {
	return g.AddRoute("PUT", path, handlers...)
}

// POST is an alias for AddRoute("POST", path, handlers...).
func (g *group) POST(path string, handlers ...Handler) error {
	return g.AddRoute("POST", path, handlers...)
}

// DELETE is an alias for AddRoute("DELETE", path, handlers...).
func (g *group) DELETE(path string, handlers ...Handler) error {
	return g.AddRoute("DELETE", path, handlers...)
}

// group returns a sub-handler group based on the current group's middleware
func (g *group) Group(path string, mw ...Handler) *group {
	return &group{
		mw:   append(g.mw[:len(g.mw):len(g.mw)], mw...),
		path: joinPath(g.path, path),
		s:    g.s,
	}
}

func joinPath(p1, p2 string) string {
	if p2 == "" {
		return p1
	}

	if p1 != "" && p1[0] != '/' {
		p1 = "/" + p1
	}

	if p2 != "" && p2[0] != '/' {
		p2 = "/" + p2
	}
	return strings.Replace(p1+p2, "//", "/", -1)
}

type groupHandlerChain struct {
	hc []Handler
	g  *group
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
