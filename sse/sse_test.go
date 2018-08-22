package sse_test

import (
	"log"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/missionMeteora/apiserv"
	"github.com/missionMeteora/apiserv/sse"
)

func TestSSE(t *testing.T) {
	if testing.Short() {
		t.Skip("not supported in short mode")
	}

	srv := apiserv.New()
	if testing.Verbose() {
		srv.Use(apiserv.LogRequests(true))
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	done := make(chan struct{}, 1)
	sr := sse.NewRouter()

	srv.GET("/sse/:id", func(ctx *apiserv.Context) apiserv.Response {
		log.Println("new connection", ctx.Req.RemoteAddr)
		return sr.Handle(ctx.Param("id"), 10, ctx)
	})

	srv.GET("/send/:id", func(ctx *apiserv.Context) apiserv.Response {
		log.Println("new connection", ctx.Req.RemoteAddr)
		sr.SendAll(ctx.Param("id"), time.Now().String(), "", ctx.Query("m"))
		return nil
	})

	srv.GET("/close", func(ctx *apiserv.Context) apiserv.Response {
		close(done)
		return nil
	})

	srv.GET("/", func(ctx *apiserv.Context) apiserv.Response {
		ctx.Write([]byte(page))
		return nil
	})

	log.Printf("listening on: %s", ts.URL)

	<-done
}

const page = `
<!DOCTYPE html>
<html>
<head>
	<title>Test</title>
</head>
<body>
	<script type="text/javascript">
		const es = new EventSource('/sse/user1');
		// Create a callback for when a new message is received.
		es.onmessage = function(e) {
			console.log(e);
			document.body.innerHTML += e.data + '<br>';
		};
	</script>
	<a href="/close">close</a>
</body>
</html>
`
