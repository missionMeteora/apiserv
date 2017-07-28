package apiserv

import (
	"time"
)

func LogRequests() MiddlewareHandler {
	return func(ctx *Context, next func() bool) Response {
		var (
			req   = ctx.Req
			url   = req.URL
			start = time.Now()
		)
		next()

		ctx.s.Logf("[%s] [%d] %s %s [%s]", ctx.ClientIP(), ctx.Status(), req.Method, url.Path, time.Since(start))
		return nil
	}
}
