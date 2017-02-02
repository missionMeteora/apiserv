package router

import (
	"errors"
	"net/http"
	"path"
	"sync"
)

var (
	// ErrTooManyStars is returned if there are multiple *params in the path
	ErrTooManyStars = errors.New("too many stars")
	// ErrStarNotLast is returned if *param is not the last part of the path.
	ErrStarNotLast = errors.New("star param must be the last part of the path")
)

type Handler func(h http.ResponseWriter, req *http.Request, p Params)

type node struct {
	h         Handler
	parts     []nodePart
	numParams int
	hasStar   bool
}

type routeMap map[string][]node

// Router is an efficient routing library
type Router struct {
	get    routeMap
	post   routeMap
	put    routeMap
	delete routeMap
	other  map[string]routeMap

	paramsPool sync.Pool

	opts            Options
	NotFoundHandler http.Handler
	PanicHandler    func(http.ResponseWriter, *http.Request, interface{})
	maxParams       int
}

// New returns a new Router
func New(opts *Options) *Router {
	if opts == nil {
		opts = &DefaultOptions
	} else if opts.MaxParamsPoolSize < 1 {
		opts.MaxParamsPoolSize = DefaultOptions.MaxParamsPoolSize
	}

	r := &Router{
		opts: *opts,
	}

	r.paramsPool.New = func() interface{} {
		return make(Params, 0, r.maxParams)
	}

	if !opts.NoDefaultNotHandler {
		r.NotFoundHandler = http.NotFoundHandler()
	}

	if !opts.NoDefaultPanicHandler {
		r.PanicHandler = PanicHandler
	}
	return r
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
	if stars == 1 && rest[len(rest)-1].Type != '*' {
		if r.opts.NoPanicOnInvalidAddRoute {
			return ErrStarNotLast
		}
		panic(ErrStarNotLast)
	}
	m := r.getMap(method, true)
	m[p] = append(m[p], node{h: h, parts: rest, numParams: num, hasStar: stars == 1})
	if num > r.maxParams {
		r.maxParams = num
	}
	return nil
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

// TODO: fix * matching
// Match matches a method and path to a handler
func (r *Router) Match(method, path string) (handler Handler, params Params) {
	m := r.getMap(method, false)
	if m == nil {
		return
	}
	ln := len(path) - 1
	for i, slashes := ln, 0; i > -1; i-- {
		if path[i] != '/' && i < ln {
			continue
		}
		p := path[:i+1]
	O:
		for rm, mi := m[p], 0; mi < len(rm); mi++ {
			n := &rm[mi]
			if len(n.parts) != slashes && !n.hasStar {
				continue
			}
			p := path[i+1:]
			for ln, x, y, last := len(p)-1, 0, 0, 0; x <= ln; x++ {
				c, isEnd := p[x], x == ln
				if c != '/' && !isEnd {
					continue
				}
				if isEnd {
					x++
				}
				np, v := &n.parts[y], p[last+1:x]
				if np.Type == 0 && np.Name != v {
					continue O
				}
				if np.Type == '*' {
					break
				}
				y++
				last = x
			}
			if n.numParams > 0 {
				params = r.getParams()
				for ln, x, y, last := len(p)-1, 0, 0, 0; x <= ln; x++ {
					c, isEnd := p[x], x == ln
					if c != '/' && !isEnd {
						continue
					}
					if isEnd {
						x++
					}
					np := &n.parts[y]
					if np.Type == ':' {
						params = append(params, Param{np.Name, p[last:x]})
					} else if np.Type == '*' {
						params = append(params, Param{np.Name, p[last:]})
						break
					}
					y++
					last = x + 1
				}
			}
			return n.h, params
		}
		slashes++
	}
	return
}

// ServerHTTP implements http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.opts.CatchPanics {
		defer func() {
			if v := recover(); v != nil && r.PanicHandler != nil {
				r.PanicHandler(w, req, v)
			}
		}()
	}

	u := req.URL.EscapedPath()

	if !r.opts.NoAutoCleanURL {
		u = path.Clean(u)
	}

	if h, p := r.Match(req.Method, u); h != nil {
		h(w, req, p)
		r.putParams(p)
		return
	}

	if r.NotFoundHandler != nil {
		r.NotFoundHandler.ServeHTTP(w, req)
	}

}

func (r *Router) getMap(method string, create bool) routeMap {
	switch method {
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

func (r *Router) getParams() Params {
	// this should never ever panic, if it does then there's something extremely wrong and *it should* panic
	return r.paramsPool.Get().(Params)
}

func (r *Router) putParams(p Params) {
	if cap(p) != r.maxParams {
		return
	}
	r.paramsPool.Put(p[:0])
}
