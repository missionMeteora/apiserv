package apiserv

import (
	"io"
	"net/http"

	"github.com/missionMeteora/apiv2/router"
)

// Context is the default context passed to handlers
// it is not thread safe and should never be used outside the handler
type Context struct {
	Params router.Params
	Req    *http.Request
	http.ResponseWriter

	data map[string]interface{}
}

// Get returns a context value
func (ctx *Context) Get(key string) interface{} {
	return ctx.data[key]
}

// Set sets a context value, useful in passing data to other handlers down the chain
func (ctx *Context) Set(key string, val interface{}) {
	if ctx.data == nil {
		ctx.data = make(map[string]interface{})
	}
	ctx.data[key] = val
}

// DataFromReader outputs the data from the passed reader with the specific http code and optional content-type.
func (ctx *Context) DataFromReader(code int, contentType string, r io.Reader) (int64, error) {
	if contentType != "" {
		ctx.Header().Set("content-type", contentType)
	}
	ctx.WriteHeader(code)
	return io.Copy(ctx, r)
}

// Break can be returned from a handler to break a handler chain
var Break = &Response{Code: -1}

// Handler is the default server Handler
// In a handler chain, returning a non-nil breaks the chain.
type Handler func(ctx *Context) *Response

type handlerChain []Handler

func (hh handlerChain) Serve(rw http.ResponseWriter, req *http.Request, p router.Params) {
	ctx := &Context{
		Params:         p,
		Req:            req,
		ResponseWriter: rw,
	}
L:
	for _, h := range hh {
		switch r := h(ctx); r {
		case nil: // do nothing on nil
		case Break: // break means break the chain
			break L
		default:
			r.Output(rw)
			break L
		}
	}

}
