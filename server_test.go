package apiserv

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

var testData = []struct {
	path string
	*JSONResponse
}{
	{"/ping", NewJSONResponse("pong")},
	{"/ping/world", NewJSONResponse("pong:world")},
	{"/random", RespNotFound.(*JSONResponse)},
	{"/panic", NewJSONErrorResponse(http.StatusInternalServerError, "PANIC (string): well... poo")},
	{"/mw/sub", NewJSONResponse("data:test")},
}

func TestServer(t *testing.T) {
	var (
		srv = New(SetErrLogger(nil)) // don't need the spam with panics for the /panic handler
		ts  = httptest.NewServer(srv)
	)

	srv.GET("/ping", func(ctx *Context) Response {
		return NewJSONResponse("pong")
	})
	srv.GET("/panic", func(ctx *Context) Response {
		panic("well... poo")
	})

	srv.GET("/ping/:id", func(ctx *Context) Response {
		return NewJSONResponse("pong:" + ctx.Params.Get("id"))
	})

	srv.StaticStd("/s-std/", "./")

	srv.Static("/s/", "./")

	srv.StaticFile("/README.md", "./router/README.md")

	srv.Group("/mw", func(ctx *Context) Response {
		ctx.Set("data", "test")
		return nil
	}).GET("/sub", func(ctx *Context) Response {
		v, _ := ctx.Get("data").(string)
		return NewJSONResponse("data:" + v)
	})

	defer ts.Close()

	for _, td := range testData {
		t.Run("Path:"+td.path, func(t *testing.T) {
			var (
				res, err = http.Get(ts.URL + td.path)
				resp     JSONResponse
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
				t.Fatalf("expected (%s) %+v, got %+v", td.path, td.JSONResponse, resp)
			}

			if len(resp.Errors) > 0 {
				if len(resp.Errors) != len(td.Errors) {
					t.Fatalf("expected (%s) %+v, got %+v", td.path, td.JSONResponse, resp)
				}

				for i := range resp.Errors {
					if re, te := resp.Errors[i], td.Errors[i]; *re != *te {
						t.Fatalf("expected %+v, got %+v", te, re)
					}
				}
			}

		})
	}

	t.Run("Static", func(t *testing.T) {
		readme, _ := ioutil.ReadFile("./router/README.md")
		res, err := http.Get(ts.URL + "/s/router/README.md")

		if err != nil {
			t.Fatal(err)
		}

		b, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(readme, b) {
			t.Fatal("files not equal")
		}

		res, err = http.Get(ts.URL + "/s-std/router/README.md")

		if err != nil {
			t.Fatal(err)
		}

		b, err = ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(readme, b) {
			t.Fatal("files not equal")
		}

		res, err = http.Get(ts.URL + "/README.md")

		if err != nil {
			t.Fatal(err)
		}

		b, err = ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(readme, b) {
			t.Fatal("files not equal", string(b))
		}
	})

	t.Run("ReadResp", func(t *testing.T) {
		res, err := http.Get(ts.URL + "/ping")

		if err != nil {
			t.Fatal(err)
		}

		var s string
		r, err := ReadJSONResponse(res.Body, &s)
		if err != nil {
			t.Fatal(err)
		}
		if s != "pong" {
			t.Fatalf("expected pong, got %+v", r)
		}
	})
}
