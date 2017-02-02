// +build httprouter

package router

import (
	"net/http"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
)

func BenchmarkHttpRouter5Params(b *testing.B) {
	req, _ := http.NewRequest("GET", "/campaignReport/:id/:cid/:start-date/:end-date/:filename", nil)
	r := buildMeteoraApiHttpRouter(b, false)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.ServeHTTP(nil, req)
		}
	})
}

func BenchmarkHttpRouterStatic(b *testing.B) {
	req, _ := http.NewRequest("GET", "/dashboard", nil)
	r := buildMeteoraApiHttpRouter(b, false)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.ServeHTTP(nil, req)
		}
	})
}

func buildMeteoraApiHttpRouter(l errorer, print bool) (r *httprouter.Router) {
	r = httprouter.New()
	for _, m := range meteoraApi[5:] { // httprouter can't handle it
		ep := m.url
		cnt := strings.Count(ep, ":")
		fn := func(_ http.ResponseWriter, req *http.Request, p httprouter.Params) {
			if ep != req.URL.EscapedPath() {
				l.Fatalf("urls don't match, expected %s, got %s", ep, req.URL.EscapedPath())
			}
			if cnt != len(p) {
				l.Fatalf("{%q: %q} expected %d params, got %d", ep, p, cnt, len(p))
			}
			if print {
				l.Logf("[%s] %s %q", req.Method, ep, p)
			}
		}
		r.Handle("GET", ep, fn)
	}
	r.NotFound = http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		panic(req.URL.String())
	})
	return
}
