package router

import (
	"errors"
	"sync"
)

// Options passed to the router
type Options struct {
	NoAutoCleanURL           bool // don't automatically clean URLs, not recommended
	NoDefaultNotHandler      bool // don't use the default not found handler
	NoDefaultPanicHandler    bool // don't use the default panic handler
	NoPanicOnInvalidAddRoute bool // don't panic on invalid routes, return an error instead
	NoCatchPanics            bool // don't catch panics, warning this can cause the whole app to crash rather than the handler
}

var (
	// ErrTooManyStars is returned if there are multiple *params in the path
	ErrTooManyStars = errors.New("too many stars")
	// ErrStarNotLast is returned if *param is not the last part of the path.
	ErrStarNotLast = errors.New("star param must be the last part of the path")
)

type node struct {
	h     Handler
	parts []nodePart
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
	head   routeMap
	get    routeMap
	post   routeMap
	put    routeMap
	delete routeMap
	other  map[string]routeMap

	paramsPool sync.Pool

	opts            Options
	NotFoundHandler Handler
	PanicHandler    PanicHandler
	maxParams       int
}

// New returns a new Router
func New(opts *Options) *Router {
	var r Router

	if opts != nil {
		r.opts = *opts
	}

	r.paramsPool.New = func() interface{} {
		return &paramsWrapper{make(Params, 0, r.maxParams)}
	}

	if !r.opts.NoDefaultNotHandler {
		r.NotFoundHandler = DefaultNotFoundHandler
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

// HEAD is an alias for AddRoute("HEAD", path, h)
func (r *Router) HEAD(path string, h Handler) error {
	return r.AddRoute("HEAD", path, h)
}

// GET is an alias for AddRoute("GET", path, h)
func (r *Router) GET(path string, h Handler) error {
	return r.AddRoute("GET", path, h)
}

// POST is an alias for AddRoute("POST", path, h)
func (r *Router) POST(path string, h Handler) error {
	return r.AddRoute("POST", path, h)
}

// PUT is an alias for AddRoute("PUT", path, h)
func (r *Router) PUT(path string, h Handler) error {
	return r.AddRoute("PUT", path, h)
}

// DELETE is an alias for AddRoute("DELETE", path, h)
func (r *Router) DELETE(path string, h Handler) error {
	return r.AddRoute("DELETE", path, h)
}

// Match matches a method and path to a handler.
// if METHOD == HEAD and there isn't a specific handler for it, it returns the GET handler for the path.
func (r *Router) Match(method, path string) (handler Handler, params Params) {
	h, p := r.match(method, path)
	if h == nil && method == "HEAD" {
		h, p = r.match("GET", path)
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

	revSplitPathFn(path, '/', func(p string, pidx, idx int) bool {
		if nn = m.get(path[:idx]); nn != nil {
			path, nsep = path[idx:], pidx
			return true
		}
		return false
	})

	for i := range nn {
		n := nn[i]
		if len(n.parts) == nsep || n.hasStar() {
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
	switch method {
	case "HEAD":
		if create && r.head == nil {
			r.head = routeMap{}
		}
		return r.head
	case "GET":
		if create && r.get == nil {
			r.get = routeMap{}
		}
		return r.get
	case "POST":
		if create && r.post == nil {
			r.post = routeMap{}
		}
		return r.post
	case "PUT":
		if create && r.put == nil {
			r.put = routeMap{}
		}
		return r.put
	case "DELETE":
		if create && r.delete == nil {
			r.delete = routeMap{}
		}
		return r.delete
	default:
		m, ok := r.other[method]
		if !ok && create {
			m = routeMap{}
			if r.other == nil {
				r.other = map[string]routeMap{}
			}
			r.other[method] = m
		}
		return m
	}
}

func (r *Router) getParams() *paramsWrapper {
	// this should never ever panic, if it does then there's something extremely wrong and *it should* panic
	return r.paramsPool.Get().(*paramsWrapper)
}

func (r *Router) putParams(p *paramsWrapper) {
	if p == nil || cap(p.p) != r.maxParams {
		return
	}
	p.p = p.p[:0]
	r.paramsPool.Put(p)
}
