package apiserv

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
			return NewJSONErrorResponse(500, err)
		}

		return nil
	}
}

// AllowCORS allows CORS responses.
// If allowedMethods is empty, it will respond with the requested method.
func AllowCORS(allowedMethods ...string) Handler {
	return func(ctx *Context) Response {
		rh, wh := ctx.Req.Header, ctx.Header()
		wh.Set("Access-Control-Allow-Origin", rh.Get("Origin"))
		if len(allowedMethods) == 0 {
			wh.Set("Access-Control-Allow-Methods", rh.Get("Access-Control-Request-Method"))
		} else {
			wh.Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
		}
		wh.Set("Access-Control-Allow-Headers", rh.Get("Access-Control-Request-Headers"))
		wh.Set("Access-Control-Max-Age", "86400") // 24 hours
		return nil
	}
}
