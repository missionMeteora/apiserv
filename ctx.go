package apiserv

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/missionMeteora/apiserv/router"
)

// Context is the default context passed to handlers
// it is not thread safe and should never be used outside the handler
type Context struct {
	Params router.Params
	Req    *http.Request
	http.ResponseWriter

	data map[string]interface{}

	done bool
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

// WriteReader outputs the data from the passed reader with the specific http code and optional content-type.
func (ctx *Context) WriteReader(code int, contentType string, r io.Reader) (int64, error) {
	ctx.done = true
	if contentType != "" {
		ctx.SetContentType(contentType)
	}
	ctx.WriteHeader(code)
	return io.Copy(ctx, r)
}

// SetContentType sets the responses's content-type.
func (ctx *Context) SetContentType(typ string) {
	h := ctx.Header()
	h.Set("Content-Type", typ)
	h.Set("X-Content-Type-Options", "nosniff") // fixes IE xss exploit
}

// ContentType returns the request's content-type.
func (ctx *Context) ContentType() string {
	return ctx.Req.Header.Get("Content-Type")
}

// BindJSON parses the request's body as json, and closes the body.
// Note that unlike gin.Context.Bind, this does NOT verify the fields using special tags.
func (ctx *Context) BindJSON(out interface{}) error {
	var (
		body = ctx.Req.Body
		err  = json.NewDecoder(body).Decode(out)
	)
	body.Close()
	return err
}

// Break can be returned from a handler to break a handler chain.
// It doesn't write anything to the connection.
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
			if !ctx.done {
				r.Output(ctx)
			}
			break L
		}
	}
}

var ctxPool = sync.Pool{
	New: func() interface{} {
		return &Context{}
	},
}
