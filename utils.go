package apiserv

import (
	"os"
	"path/filepath"
)

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

	return func(ctx *Context) *Response {
		path := ctx.Param(paramName)
		if sem != nil {
			sem <- e
			defer func() { <-sem }()
		}

		err := ctx.File(filepath.Join(dir, path))
		ctx.done = true // ctx.File only sets this on success, we need to set it here regardless.
		if err != nil {
			if os.IsNotExist(err) {
				return RespNotFound
			}
			return NewErrorResponse(500, err)
		}
		return nil
	}
}
