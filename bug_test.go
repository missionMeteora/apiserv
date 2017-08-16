package apiserv

import (
	"net/http"
	"testing"
)

// Using .Next in a middleware didn't execute other middlewars
func TestIssue12(t *testing.T) {
	s := newServerAndWait(t, "localhost:0")
	defer s.Shutdown(0)

	s.Use(LogRequests(true))
	g := s.Group("", func(ctx *Context) Response {
		ctx.Set("mw", true)
		return nil
	})
	g.GET("/ping", func(ctx *Context) Response {
		if v, ok := ctx.Get("mw").(bool); !ok || !v {
			return RespNotFound
		}
		return RespOK
	})

	resp, err := http.Get("http://" + s.Addrs()[0] + "/ping")
	if err != nil {
		t.Error(err)
		return
	}
	if resp.StatusCode != 200 {
		t.Error("couldn't get the ctx value")
	}
}
