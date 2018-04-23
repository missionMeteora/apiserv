package router

import (
	"net/http"
	"testing"
)

func TestBugGithub3(t *testing.T) {
	r := New(nil)
	_ = r.GET("/api/files/:bkt/:type/:filename", func(w http.ResponseWriter, req *http.Request, p Params) {
		if p.Get("bkt") != "Personal" || p.Get("type") != "data" || p.Get("filename") != "hi.txt" {
			t.Fatalf(`expected "Personal/data/hi.txt", got "%s/%s/%s"`, p[0].Value, p[1].Value, p[2].Value)
		}
	})
	h, p := r.Match("GET", "/api/files/Personal/data/hi.txt")
	if h == nil {
		t.Fatal("couldn't find the handler")
	}
	h(nil, nil, p)
}
