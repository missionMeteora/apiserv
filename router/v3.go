package router

import (
	"errors"
	"net/http"
	"sync"
)

// Options passed to the router
type Options struct {
	NoAutoCleanURL           bool // don't automatically clean URLs, not recommended
	NoDefaultPanicHandler    bool // don't use the default panic handler
	NoPanicOnInvalidAddRoute bool // don't panic on invalid routes, return an error instead
	NoCatchPanics            bool // don't catch panics
	NoAutoHeadToGet          bool // disable automatically handling HEAD requests
}

var (
	// ErrTooManyStars is returned if there are multiple *params in the path
	ErrTooManyStars = errors.New("too many stars")
	// ErrStarNotLast is returned if *param is not the last part of the path.
	ErrStarNotLast = errors.New("star param must be the last part of the path")
)

type node struct {
	parts []nodePart
	h     Handler
}

func (n node) hasStar() bool {
	return len(n.parts) > 0 && n.parts[len(n.parts)-1].Type() == '*'
}

type routeMap map[string][]node

func (rm routeMap) get(path string) []node {
	return rm[path]
}

func (rm routeMap) append(path string, n node) {
	rm[path] = append(rm[path], n)
}

// Router is an efficient routing library
type Router struct {
	methods [10]routeMap

	pp sync.Pool

	NotFoundHandler         Handler
	MethodNotAllowedHandler Handler
	PanicHandler            PanicHandler

	maxParams int
	opts      Options
}

// New returns a new Router
func New(opts *Options) *Router {
	var r Router

	if opts != nil {
		r.opts = *opts
	}

	r.pp.New = func() interface{} {
		return &paramsWrapper{make(Params, 0, r.maxParams)}
	}

	if !r.opts.NoDefaultPanicHandler {
		r.PanicHandler = DefaultPanicHandler
	}

	return &r
}

// AddRoute adds a Handler to the specific method and route.
// Calling AddRoute after starting the http server is racy and not supported.
func (r *Router) AddRoute(method, route string, h Handler) error {
	p, rest, num, stars := splitPathToParts(route)
	if stars > 1 {
		if r.opts.NoPanicOnInvalidAddRoute {
			return ErrTooManyStars
		}
		panic(ErrTooManyStars)
	}

	if stars == 1 && rest[len(rest)-1].Type() != '*' {
		if r.opts.NoPanicOnInvalidAddRoute {
			return ErrStarNotLast
		}
		panic(ErrStarNotLast)
	}

	if n := len(p) - 1; len(p) > 1 && p[n] == '/' {
		p = p[:n]
	}

	m := r.getMap(method, true)
	m.append(p, node{h: h, parts: rest})

	if num > r.maxParams {
		r.maxParams = num
	}

	return nil
}

// GET is an alias for AddRoute("GET", path, h)
func (r *Router) GET(path string, h Handler) error {
	return r.AddRoute(http.MethodGet, path, h)
}

// POST is an alias for AddRoute("POST", path, h)
func (r *Router) POST(path string, h Handler) error {
	return r.AddRoute(http.MethodPost, path, h)
}

// PUT is an alias for AddRoute("PUT", path, h)
func (r *Router) PUT(path string, h Handler) error {
	return r.AddRoute(http.MethodPut, path, h)
}

// DELETE is an alias for AddRoute("DELETE", path, h)
func (r *Router) DELETE(path string, h Handler) error {
	return r.AddRoute(http.MethodDelete, path, h)
}

// Match matches a method and path to a handler.
// if METHOD == HEAD and there isn't a specific handler for it, it returns the GET handler for the path.
func (r *Router) Match(method, path string) (handler Handler, params Params) {
	h, p := r.match(method, path)

	if h == nil && method == http.MethodHead && !r.opts.NoAutoHeadToGet {
		h, p = r.match(http.MethodGet, path)
	}

	return h, p.Params()
}

func (r *Router) match(method, path string) (handler Handler, params *paramsWrapper) {
	m := r.getMap(method, false)

	var (
		nn   []node
		rn   node
		nsep int
	)

	if !revSplitPathFn(path, '/', func(p string, pidx, idx int) bool {
		if nn = m.get(path[:idx]); nn != nil {
			path, nsep = path[idx:], pidx
			return true
		}

		return false
	}) {
		return
	}

	for i := range nn {
		if n := nn[i]; len(n.parts) == nsep || n.hasStar() {
			rn = n
			handler = n.h
			break
		}
	}

	if len(rn.parts) == 0 {
		return
	}

	params = r.getParams()
	splitPathFn(path, '/', func(p string, pidx, idx int) bool {
		np := rn.parts[pidx]
		switch np.Type() {
		case ':':
			params.p = append(params.p, Param{np.Name(), p[1:]})
		case '*':
			params.p = append(params.p, Param{np.Name(), path[1:]})
			return true
		}
		return false
	})

	return
}

func (r *Router) getMap(method string, create bool) routeMap {
	var rm *routeMap
	switch method {
	case http.MethodGet:
		rm = &r.methods[0]
	case http.MethodHead:
		rm = &r.methods[1]
	case http.MethodPost:
		rm = &r.methods[2]
	case http.MethodPut:
		rm = &r.methods[3]
	case http.MethodPatch:
		rm = &r.methods[4]
	case http.MethodDelete:
		rm = &r.methods[5]
	case http.MethodConnect:
		rm = &r.methods[6]
	case http.MethodOptions:
		rm = &r.methods[7]
	case http.MethodTrace:
		rm = &r.methods[8]
	default:
		return nil
	}
	if create && *rm == nil {
		*rm = routeMap{}
	}

	return *rm
}

func (r *Router) getParams() *paramsWrapper {
	// this should never ever panic, if it does then there's something extremely wrong and *it should* panic
	return r.pp.Get().(*paramsWrapper)
}

func (r *Router) putParams(p *paramsWrapper) {
	if p == nil || cap(p.p) != r.maxParams {
		return
	}
	p.p = p.p[:0]
	r.pp.Put(p)
}
