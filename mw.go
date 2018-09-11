package apiserv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/securecookie"
)

// LogRequests is a request logger middleware.
// If logJSONRequests is true, it'll attempt to parse the incoming request's body and output it to the log.
func LogRequests(logJSONRequests bool) Handler {
	var reqID uint64
	return func(ctx *Context) Response {
		var (
			req   = ctx.Req
			url   = req.URL
			start = time.Now()
			id    = atomic.AddUint64(&reqID, 1)
			extra string
		)

		if logJSONRequests {
			switch m := req.Method; m {
			case http.MethodPost, http.MethodPut, http.MethodDelete:
				var buf bytes.Buffer
				io.Copy(&buf, req.Body)
				req.Body.Close()
				req.Body = ioutil.NopCloser(&buf)
				j, _ := json.Marshal(req.Header)
				if ln := buf.Len(); ln > 0 {
					switch buf.Bytes()[0] {
					case '[', '{', 'n': // [], {} and nullable
						extra = fmt.Sprintf("\n\tHeaders: %s\n\tRequest (%d): %s", j, ln, buf.String())
					default:
						extra = fmt.Sprintf("\n\tHeaders: %s\n\tRequest (%d): <binary>", j, buf.Len())
					}
				}
			}
		}

		ctx.NextMiddleware()
		ctx.Next()

		ctx.s.Logf("[reqID:%05d] [%s] [%d] [%s] %s %s [%s]%s",
			id, ctx.ClientIP(), ctx.Status(), req.UserAgent(), req.Method, url.Path, time.Since(start), extra)
		return nil
	}
}

const secureCookieKey = ":SC:"

// SecureCookie is a middleware to enable SecureCookies.
// For more details check `go doc securecookie.New`
func SecureCookie(hashKey, blockKey []byte) Handler {
	return func(ctx *Context) Response {
		ctx.Set(secureCookieKey, securecookie.New(hashKey, blockKey))
		return nil
	}
}

// GetSecureCookie returns the *securecookie.SecureCookie associated with the Context, or nil.
func GetSecureCookie(ctx *Context) *securecookie.SecureCookie {
	sc, ok := ctx.Get(secureCookieKey).(*securecookie.SecureCookie)
	if ok {
		return sc
	}
	return nil
}
