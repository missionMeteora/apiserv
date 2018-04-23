package apiserv

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	nukeCookieDate = time.Date(1991, time.August, 6, 0, 0, 0, 0, time.UTC)
)

// FromHTTPHandler returns a Handler from an http.Handler.
func FromHTTPHandler(h http.Handler) Handler {
	return FromHTTPHandlerFunc(h.ServeHTTP)
}

// FromHTTPHandlerFunc returns a Handler from an http.Handler.
func FromHTTPHandlerFunc(h http.HandlerFunc) Handler {
	return func(ctx *Context) Response {
		h(ctx, ctx.Req)
		return Break
	}
}

// StaticDirStd is a QoL wrapper for http.FileServer(http.Dir(dir)).
func StaticDirStd(prefix, dir string) Handler {
	h := http.StripPrefix(prefix, http.FileServer(http.Dir(dir)))
	return FromHTTPHandler(h)
}

// StaticDir is a shorthand for StaticDirWithLimit(dir, paramName, -1).
func StaticDir(dir, paramName string) Handler {
	return StaticDirWithLimit(dir, paramName, -1)
}

// StaticDirWithLimit returns a handler that handles serving static files.
// paramName is the path param, for example: s.GET("/s/*fp", StaticDirWithLimit("./static/", "fp", 1000)).
// if limit is > 0, it will only ever serve N files at a time.
func StaticDirWithLimit(dir, paramName string, limit int) Handler {
	var (
		sem chan struct{}
		e   struct{}
	)

	if limit > 0 {
		sem = make(chan struct{}, limit)
	}

	return func(ctx *Context) Response {
		path := ctx.Param(paramName)
		if sem != nil {
			sem <- e
			defer func() { <-sem }()
		}

		if err := ctx.File(filepath.Join(dir, path)); err != nil {
			if os.IsNotExist(err) {
				return RespNotFound
			}
			return NewJSONErrorResponse(http.StatusInternalServerError, err)
		}

		return nil
	}
}

// AllowCORS allows CORS responses.
// If allowedMethods is empty, it will respond with the requested method.
func AllowCORS(allowedMethods ...string) Handler {
	ams := strings.Join(allowedMethods, ", ")
	return func(ctx *Context) Response {
		rh, wh := ctx.Req.Header, ctx.Header()

		wh.Set("Access-Control-Allow-Origin", rh.Get("Origin"))

		if len(ams) == 0 {
			wh.Set("Access-Control-Allow-Methods", rh.Get("Access-Control-Request-Method"))
		} else {
			wh.Set("Access-Control-Allow-Methods", ams)
		}
		if reqHeaders := rh.Get("Access-Control-Request-Headers"); reqHeaders != "" {
			wh.Set("Access-Control-Allow-Headers", reqHeaders)
		}

		wh.Set("Access-Control-Max-Age", "86400") // 24 hours
		return RespOK
	}
}

// M is a QoL shortcut for map[string]interface{}
type M map[string]interface{}

type ctxValue struct {
	key   string
	value interface{}
}

// this is a cheaper version than using a map and/or context.WithValue

type ctxValues []*ctxValue

func (vs ctxValues) Set(key string, value interface{}) ctxValues {
	if v := vs.get(key); v != nil {
		v.value = value
		return vs
	}

	return append(vs, &ctxValue{key, value})
}

func (vs ctxValues) Get(key string) interface{} {
	if v := vs.get(key); v != nil {
		return v.value
	}
	return nil
}

func (vs ctxValues) get(key string) *ctxValue {
	for _, v := range vs {
		if v.key == key {
			return v
		}
	}
	return nil
}

func jsonMarshal(v interface{}) (string, error) {
	j, err := json.Marshal(v)
	return string(j), err
}

// MultiError handles returning multiple errors.
type MultiError []error

// Push adds an error to the MultiError slice if err != nil.
func (me *MultiError) Push(err error) {
	if err != nil {
		*me = append(*me, err)
	}
}

// Err returns nil if me is empty.
func (me MultiError) Err() error {
	if len(me) == 0 {
		return nil
	}

	if len(me) == 1 {
		return me[0]
	}

	return me
}

func (me MultiError) Error() string {
	errs := make([]string, 0, len(me))
	for _, err := range me {
		errs = append(errs, err.Error())
	}

	return "multiple errors returned:\n\t" + strings.Join(errs, "\n\t")
}
