package apiserv

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"sync/atomic"
	"time"
)

func LogRequests(logJSONRequests bool) Handler {
	var reqID uint64
	return func(ctx *Context) Response {
		var (
			req   = ctx.Req
			url   = req.URL
			start = time.Now()
			id    = atomic.AddUint64(&reqID, 1)
		)

		if logJSONRequests {
			switch m := req.Method; m {
			case "POST", "PUT", "DELETE":
				var buf bytes.Buffer
				io.Copy(&buf, req.Body)
				req.Body.Close()
				req.Body = ioutil.NopCloser(&buf)
				j, _ := json.Marshal(req.Header)
				if ln := buf.Len(); ln > 0 {
					switch buf.Bytes()[0] {
					case '[', '{', 'n': // [], {} and nullable
						log.Printf("[reqID:%5d] %s: %s\n\tHeaders: %s\n\tRequest (%d): %s", id, m, ctx.Path(), j, ln, buf.String())
					default:
						log.Printf("[reqID:%5d] %s: %s\n\t\n\tHeaders: %s\n\tRequest (%d): <binary>", id, m, ctx.Path(), j, ln)
					}
				}
			}
		}

		ctx.ExecuteHandlers()

		ctx.s.Logf("[%s] [%d] %s %s [%s]", ctx.ClientIP(), ctx.Status(), req.Method, url.Path, time.Since(start))
		return nil
	}
}
