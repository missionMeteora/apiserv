package apiserv

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/missionMeteora/apiserv/router"
	"github.com/missionMeteora/toolkit/errors"
)

const (
	// ErrDir is Returned from ctx.File when the path is a directory not a file.
	ErrDir = errors.Error("file is a directory")

	// ErrInvalidURL gets returned on invalid redirect urls.
	ErrInvalidURL = errors.Error("invalid redirect error")
)

// Context is the default context passed to handlers
// it is not thread safe and should never be used outside the handler
type Context struct {
	Params router.Params
	Req    *http.Request
	http.ResponseWriter

	data ctxValues

	status             int
	hijackServeContent bool
	done               bool

	s      *Server
	next   func() Response
	nextMW func() Response
}

// Param is a shorthand for ctx.Params.Get(name).
func (ctx *Context) Param(key string) string {
	return ctx.Params.Get(key)
}

// Query is a shorthand for ctx.Req.URL.Query().Get(key).
func (ctx *Context) Query(key string) string {
	return ctx.Req.URL.Query().Get(key)
}

// Get returns a context value
func (ctx *Context) Get(key string) interface{} {
	return ctx.data.Get(key)
}

// Set sets a context value, useful in passing data to other handlers down the chain
func (ctx *Context) Set(key string, val interface{}) {
	ctx.data = ctx.data.Set(key, val)
}

// WriteReader outputs the data from the passed reader with optional content-type.
func (ctx *Context) WriteReader(contentType string, r io.Reader) (int64, error) {
	if contentType != "" {
		ctx.SetContentType(contentType)
	}
	return io.Copy(ctx, r)
}

// File serves a file using http.ServeContent.
// See http.ServeContent.
func (ctx *Context) File(fp string) error {
	f, err := os.Open(fp)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}

	if fi.IsDir() {
		return ErrDir
	}
	ctx.hijackServeContent = true
	http.ServeContent(ctx, ctx.Req, fp, fi.ModTime(), f)
	return nil
}

// Path is a shorthand for ctx.Req.URL.EscapedPath().
func (ctx *Context) Path() string {
	return ctx.Req.URL.EscapedPath()
}

// SetContentType sets the responses's content-type.
func (ctx *Context) SetContentType(typ string) {
	if typ == "" {
		return
	}
	h := ctx.Header()
	h.Set("Content-Type", typ)
	h.Set("X-Content-Type-Options", "nosniff") // fixes IE xss exploit
}

// ContentType returns the request's content-type.
func (ctx *Context) ContentType() string {
	return ctx.Req.Header.Get("Content-Type")
}

// Read is a QoL shorthand for ctx.Req.Body.Read.
// Context implements io.Reader
func (ctx *Context) Read(p []byte) (int, error) {
	return ctx.Req.Body.Read(p)
}

// CloseBody closes the request body.
func (ctx *Context) CloseBody() error {
	return ctx.Req.Body.Close()
}

// BindJSON parses the request's body as json, and closes the body.
// Note that unlike gin.Context.Bind, this does NOT verify the fields using special tags.
func (ctx *Context) BindJSON(out interface{}) error {
	err := json.NewDecoder(ctx).Decode(out)
	ctx.CloseBody()
	return err
}

// Printf is a QoL function to handle outputing plain strings with optional fmt.Printf-style formating.
// calling this function marks the Context as done, meaning any returned responses won't be written out.
func (ctx *Context) Printf(code int, contentType, s string, args ...interface{}) (int, error) {
	ctx.done = true

	if contentType == "" {
		contentType = MimePlain
	}

	ctx.SetContentType(contentType)

	if code > 0 {
		ctx.WriteHeader(code)
	}

	return fmt.Fprintf(ctx, s, args...)
}

// JSON outputs a json object, it is highly recommended to return *Response rather than use this directly.
// calling this function marks the Context as done, meaning any returned responses won't be written out.
func (ctx *Context) JSON(code int, indent bool, v interface{}) error {
	ctx.done = true
	ctx.SetContentType(MimeJSON)

	enc := json.NewEncoder(ctx)

	if indent {
		enc.SetIndent("", "\t")
	}

	if code > 0 {
		ctx.WriteHeader(code)
	}

	return enc.Encode(v)
}

// JSONP outputs a jsonP object, it is highly recommended to return *Response rather than use this directly.
// calling this function marks the Context as done, meaning any returned responses won't be written out.
func (ctx *Context) JSONP(code int, callbackKey string, v interface{}) (err error) {
	ctx.done = true
	ctx.SetContentType(MimeJavascript)

	if code > 0 {
		ctx.WriteHeader(code)
	}

	var b []byte
	if b, err = json.Marshal(v); err != nil {
		return
	}

	_, err = fmt.Fprintf(ctx, "%s(%s)", callbackKey, string(b))
	return
}

// ClientIP returns the current client ip, accounting for X-Real-Ip and X-forwarded-For headers as well.
func (ctx *Context) ClientIP() string {
	h := ctx.Req.Header

	// handle proxies
	if ip := h.Get("X-Real-Ip"); ip != "" {
		return strings.TrimSpace(ip)
	}

	if ip := h.Get("X-Forwarded-For"); ip != "" {
		if index := strings.IndexByte(ip, ','); index >= 0 {
			if ip = strings.TrimSpace(ip[:index]); len(ip) > 0 {
				return ip
			}
		}
		if ip = strings.TrimSpace(ip); ip != "" {
			return ip
		}
	}

	if ip, _, err := net.SplitHostPort(strings.TrimSpace(ctx.Req.RemoteAddr)); err == nil {
		return ip
	}

	return ""
}

// NextMiddleware is a middleware-only func to execute all the other middlewares in the group and return before the handlers.
// will panic if called from a handler.
func (ctx *Context) NextMiddleware() Response {
	if ctx.nextMW != nil {
		return ctx.nextMW()
	}
	return nil
}

// NextHandler is a func to execute all the handlers in the group up until one returns a Response.
func (ctx *Context) NextHandler() Response {
	if ctx.next != nil {
		return ctx.next()
	}
	return nil
}

// Next is a QoL function that calls NextMiddleware() then NextHandler() if NextMiddleware() didn't return a response.
func (ctx *Context) Next() Response {
	if r := ctx.NextMiddleware(); r != nil {
		return r
	}
	return ctx.NextHandler()
}

// WriteHeader and Write are to implement ResponseWriter and allows ghetto hijacking of http.ServeContent errors,
// without them we'd end up with plain text errors, we wouldn't want that, would we?
// WriteHeader implements http.ResponseWriter
func (ctx *Context) WriteHeader(s int) {
	if ctx.status = s; ctx.hijackServeContent && ctx.status >= 300 {
		return
	}
	ctx.ResponseWriter.WriteHeader(s)

}

// Write implements http.ResponseWriter
func (ctx *Context) Write(p []byte) (int, error) {
	if ctx.hijackServeContent && ctx.status >= 300 {
		ctx.hijackServeContent = false
		NewJSONErrorResponse(ctx.status, p).WriteToCtx(ctx)
		return len(p), nil
	}
	ctx.done = true
	return ctx.ResponseWriter.Write(p)
}

// Status returns last value written using WriteHeader.
func (ctx *Context) Status() int {
	if ctx.status == 0 {
		return 200
	}
	return ctx.status
}

// Done returns wither the context is marked as done or not.
func (ctx *Context) Done() bool { return ctx.done }

// SetCookie sets an http-only cookie using the passed name, value and domain.
// Returns an error if there was a problem encoding the value.
// if forceSecure is true, it will set the Secure flag to true, otherwise it sets it based on the connection.
// if duration == -1, it sets expires to 10 years in the past, if 0 it gets ignored (aka session-only cookie),
// if duration > 0, the expiration date gets set to now() + duration.
// Note that for more complex options, you can use http.SetCookie(ctx, &http.Cookie{...}).
func (ctx *Context) SetCookie(name string, value interface{}, domain string, forceHTTPS bool, duration time.Duration) (err error) {
	var encValue string
	if sc := GetSecureCookie(ctx); sc != nil {
		if encValue, err = sc.Encode(name, value); err != nil {
			return
		}
	} else if s, ok := value.(string); ok {
		encValue = s
	} else if encValue, err = jsonMarshal(value); err != nil {
		return
	}

	cookie := &http.Cookie{
		Path:     "/",
		Name:     name,
		Value:    encValue,
		Domain:   domain,
		HttpOnly: true,
		Secure:   forceHTTPS || ctx.Req.TLS != nil,
	}

	switch duration {
	case 0: // session only
	case -1:
		cookie.Expires = nukeCookieDate
	default:
		cookie.Expires = time.Now().UTC().Add(duration)

	}

	http.SetCookie(ctx, cookie)
	return
}

// RemoveCookie deletes the given cookie and sets its expires date in the past.
func (ctx *Context) RemoveCookie(name string) {
	http.SetCookie(ctx, &http.Cookie{
		Path:     "/",
		Name:     name,
		Value:    "::deleted::",
		HttpOnly: true,
		Expires:  nukeCookieDate,
	})
}

// GetCookie returns the given cookie's value.
func (ctx *Context) GetCookie(name string) (out string, ok bool) {
	c, err := ctx.Req.Cookie(name)
	if err != nil {
		return
	}
	if sc := GetSecureCookie(ctx); sc != nil {
		ok = sc.Decode(name, c.Value, &out) == nil
		return
	}
	return c.Value, true
}

// GetCookieValue unmarshals a cookie, only needed if you stored an object for the cookie not a string.
func (ctx *Context) GetCookieValue(name string, valDst interface{}) error {
	c, err := ctx.Req.Cookie(name)
	if err != nil {
		return err
	}

	if sc := GetSecureCookie(ctx); sc != nil {
		return sc.Decode(name, c.Value, valDst)
	}

	return json.Unmarshal([]byte(c.Value), valDst)
}

// StdContext returns the context.Context associated with this Context's Request.
func (ctx *Context) StdContext() context.Context {
	return ctx.Req.Context()
}

// SetStdContext sets the current Context Request's context.Context.
func (ctx *Context) SetStdContext(c context.Context) {
	ctx.Req = ctx.Req.WithContext(c)
}

var ctxPool = sync.Pool{
	New: func() interface{} { return &Context{} },
}

func getCtx(rw http.ResponseWriter, req *http.Request, p router.Params, s *Server) *Context {
	ctx, ok := ctxPool.Get().(*Context)
	if !ok {
		ctx = &Context{}
	}

	ctx.ResponseWriter, ctx.Req = rw, req
	ctx.Params, ctx.s = p, s

	return ctx
}

func putCtx(ctx *Context) {
	*ctx = Context{}
	ctxPool.Put(ctx)
}
