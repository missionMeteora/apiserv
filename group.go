package apiserv

import (
	"net/http"
	"strings"

	"github.com/missionMeteora/apiserv/router"
)

// Handler is the default server Handler
// In a handler chain, returning a non-nil breaks the chain.
type Handler func(ctx *Context) Response

// Group represents a handler group.
type Group interface {
	// Use adds more middleware to the current group.
	// returning non-nil from a middleware returns early and doesn't execute the handlers.
	Use(mw ...Handler)

	// Group returns a sub-group starting at the specified path with this group's middlewares + any other ones.
	Group(path string, mw ...Handler) Group

	// AddRoute adds a handler (or more) to the specific method and path
	// it is NOT safe to call this once you call one of the run functions
	AddRoute(method, path string, handlers ...Handler) error

	// GET is an alias for AddRoute("GET", path, handlers...).
	GET(path string, handlers ...Handler) error
	// PUT is an alias for AddRoute("PUT", path, handlers...).
	PUT(path string, handlers ...Handler) error
	// POST is an alias for AddRoute("POST", path, handlers...).
	POST(path string, handlers ...Handler) error
	// DELETE is an alias for AddRoute("DELETE", path, handlers...).
	DELETE(path string, handlers ...Handler) error

	// Static is a QoL wrapper to serving a directory.
	// If allowListing is true, it will fallback to using http.FileServer.
	Static(path, localPath string, allowListing bool) error

	// StaticFile is a QoL wrapper to serving a static file.
	StaticFile(path, localPath string) error
}

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
	return g.AddRoute(http.MethodGet, path, handlers...)
}

// PUT is an alias for AddRoute("PUT", path, handlers...).
func (g *group) PUT(path string, handlers ...Handler) error {
	return g.AddRoute(http.MethodPut, path, handlers...)
}

// POST is an alias for AddRoute("POST", path, handlers...).
func (g *group) POST(path string, handlers ...Handler) error {
	return g.AddRoute(http.MethodPost, path, handlers...)
}

// DELETE is an alias for AddRoute("DELETE", path, handlers...).
func (g *group) DELETE(path string, handlers ...Handler) error {
	return g.AddRoute(http.MethodDelete, path, handlers...)
}

func (g *group) staticStd(path, localPath string) error {
	if strings.HasSuffix(path, "/") { // make sure the path doesn't end in / or StaticDirStd will break
		path = strings.TrimSuffix(path, "/")
	}
	h := StaticDirStd(path, localPath)
	if err := g.AddRoute(http.MethodGet, joinPath(path, "*fp"), h); err != nil {
		return err
	}
	return g.AddRoute(http.MethodGet, path, h)
}

func (g *group) Static(path, localPath string, allowListing bool) error {
	if allowListing {
		return g.staticStd(path, localPath)
	}
	return g.AddRoute(http.MethodGet, joinPath(path, "*fp"), StaticDir(localPath, "fp"))
}

func (g *group) StaticFile(path, localPath string) error {
	return g.AddRoute(http.MethodGet, path, func(ctx *Context) Response {
		ctx.File(localPath)
		return Break
	})
}

// group returns a sub-handler group based on the current group's middleware
func (g *group) Group(path string, mw ...Handler) Group {
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
	var (
		ctx = getCtx(rw, req, p, ghc.g.s)

		mwIdx, hIdx int
	)
	defer putCtx(ctx)

	ctx.next = func() (r Response) {
		for hIdx < len(ghc.hc) {
			h := ghc.hc[hIdx]
			hIdx++
			if r = h(ctx); r != nil {
				if !ctx.done && r != Break {
					r.WriteToCtx(ctx)
				}
				break
			}
		}
		ctx.next = nil
		return
	}

	ctx.nextMW = func() (r Response) {
		for mwIdx < len(ghc.g.mw) {
			h := ghc.g.mw[mwIdx]
			mwIdx++
			if r = h(ctx); r != nil {
				if !ctx.done && r != Break {
					r.WriteToCtx(ctx)
				}
				break
			}
		}
		ctx.nextMW = nil

		return
	}

	ctx.Next()
}
