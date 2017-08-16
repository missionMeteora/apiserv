package apiserv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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
			extra string
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
						extra = fmt.Sprintf("\n\tHeaders: %s\n\tRequest (%d): %s", j, ln, buf.String())
					default:
						extra = fmt.Sprintf("\n\tHeaders: %s\n\tRequest (%d): <binary>", j, buf.Len())
					}
				}
			}
		}

		ctx.ExecuteHandlers()

		ctx.s.Logf("[reqID:%d] [%s] [%d] %s %s [%s]%s", id, ctx.ClientIP(), ctx.Status(), req.Method, url.Path, time.Since(start), extra)
		return nil
	}
}
