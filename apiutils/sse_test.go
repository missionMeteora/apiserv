package apiutils_test

import (
	"log"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/missionMeteora/apiserv/apiutils"

	"github.com/missionMeteora/apiserv"
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

	srv.GET("/sse", func(ctx *apiserv.Context) apiserv.Response {
		_, ss, err := apiutils.ConvertToSSE(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for ts := range time.Tick(time.Second) {
			if err := ss.SendData(ts.String()); err != nil {
				t.Log(err)
				break
			}
		}
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
		const es = new EventSource('/sse');
		// Create a callback for when a new message is received.
		es.onmessage = function(e) {
			console.log(e);
			document.body.innerHTML += e.data + '<br>';
		};
	</script>
</body>
</html>
`
