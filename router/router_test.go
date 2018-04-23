package router

import (
	"net/http"
	"strings"
	"testing"
)

func TestRouter(t *testing.T) {
	r := buildMeteoraAPIRouter(t, false)
	for _, m := range meteoraAPI {
		ep := m.url
		req, _ := http.NewRequest("GET", ep, nil)
		r.ServeHTTP(nil, req)
		req, _ = http.NewRequest("PATCH", ep, nil)
		r.ServeHTTP(nil, req)
		req, _ = http.NewRequest("PATCH", "../"+ep, nil)
		r.ServeHTTP(nil, req)
	}
}

func TestRouterStar(t *testing.T) {
	r := New(nil)
	fn := func(_ http.ResponseWriter, req *http.Request, p Params) {}
	_ = r.GET("/home", nil)
	_ = r.GET("/home/*path", fn)
	if h, p := r.Match("GET", "/home"); h != nil || len(p) != 0 {
		t.Fatalf("expected a 0 match, got %v %v", h, len(p))
	}
	if h, p := r.Match("GET", "/home/file"); h == nil || len(p) != 1 || p.Get("path") != "file" {
		t.Fatalf("expected a 1 match, got %v %v", h, p)
	}
	if h, p := r.Match("GET", "/home/file/file2/report.json"); h == nil || len(p) != 1 || p.Get("path") != "file/file2/report.json" {
		t.Fatalf("expected a 1 match, got %v %v", h, p)
	}

}

func BenchmarkRouter5Params(b *testing.B) {
	req, _ := http.NewRequest("GET", "/campaignReport/:id/:cid/:start-date/:end-date/:filename", nil)
	r := buildMeteoraAPIRouter(b, false)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.ServeHTTP(nil, req)
		}
	})
}

func BenchmarkRouterStatic(b *testing.B) {
	req, _ := http.NewRequest("GET", "/dashboard", nil)
	r := buildMeteoraAPIRouter(b, false)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.ServeHTTP(nil, req)
		}
	})
}

func buildMeteoraAPIRouter(l testing.TB, print bool) (r *Router) {
	r = New(nil)
	r.PanicHandler = nil
	for _, m := range meteoraAPI {
		ep := m.url
		cnt := strings.Count(ep, ":")
		fn := func(_ http.ResponseWriter, req *http.Request, p Params) {
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
		r.GET(ep, fn)
		r.AddRoute("PATCH", ep, fn)
	}
	r.NotFoundHandler = func(_ http.ResponseWriter, req *http.Request, _ Params) {
		panic(req.URL.String())
	}
	return
}
