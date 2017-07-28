package apiserv

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/missionMeteora/toolkit/errors"
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

// BindResponse will bind a JSON http response from an apiserv endpoint
func BindResponse(resp *http.Response, val interface{}) (err error) {
	var r JSONResponse
	r.Data = val
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return
	}

	if !r.Success {
		var errl errors.ErrorList
		for _, v := range r.Errors {
			errl.Push(v)
		}

		if err = errl.Err(); err != nil {
			return
		}

		// No error provided, utilize the response status for messaging
		return errors.Error(resp.Status)
	}

	return
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
