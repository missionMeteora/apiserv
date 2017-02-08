package apiserv

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

var testData = []struct {
	path string
	*Response
}{
	{"/ping", NewResponse("pong")},
	{"/ping/world", NewResponse("pong:world")},
	{"/random", RespNotFound},
	{"/panic", NewErrorResponse(http.StatusInternalServerError, "PANIC (string): well... poo")},
}

func TestServer(t *testing.T) {
	var (
		srv = New(nil)
		ts  = httptest.NewServer(srv)
	)

	srv.GET("/ping", func(ctx *Context) *Response {
		return NewResponse("pong")
	})
	srv.GET("/panic", func(ctx *Context) *Response {
		panic("well... poo")
	})

	srv.GET("/ping/:id", func(ctx *Context) *Response {
		return NewResponse("pong:" + ctx.Params.Get("id"))
	})

	defer ts.Close()

	for _, td := range testData {
		var (
			res, err = http.Get(ts.URL + td.path)
			resp     Response
		)
		if err != nil {
			t.Fatal(td.path, err)
		}
		err = json.NewDecoder(res.Body).Decode(&resp)
		res.Body.Close()
		if err != nil {
			t.Fatal(td.path, err)
		}

		if resp.Code != td.Code || resp.Data != td.Data {
			t.Fatalf("expected (%s) %+v, got %+v", td.path, td.Response, resp)
		}

		if len(resp.Errors) > 0 {
			if len(resp.Errors) != len(td.Errors) {
				t.Fatalf("expected (%s) %+v, got %+v", td.path, td.Response, resp)
			}

			for i := range resp.Errors {
				if re, te := resp.Errors[i], td.Errors[i]; re != te {
					t.Fatalf("expected %+v, got %+v", te, re)
				}
			}
		}
	}
}
