package apiutils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/missionMeteora/apiserv"
)

var (
	ErrBufferFull  = errors.New("buffer full")
	ErrNotAFlusher = errors.New("ctx not a flusher")

	nl        = []byte("\n")
	idBytes   = []byte("id: ")
	evtBytes  = []byte("event: ")
	dataBytes = []byte("data: ")
	pingBytes = []byte("data: ping\n\n")
)

type writeFlusher interface {
	io.Writer
	http.Flusher
}

type SSEStream struct {
	wf   writeFlusher
	done chan struct{}
	buf  *bytes.Buffer
}

func (ss *SSEStream) Ping() (err error) {
	_, err = ss.wf.Write(pingBytes)
	ss.wf.Flush()
	return
}

func (ss *SSEStream) Retry(ms int) (err error) {
	_, err = fmt.Fprintf(ss.wf, "retry: %d\n\n", ms)
	ss.wf.Flush()
	return
}

func (ss *SSEStream) SendData(data interface{}) error {
	buf := ss.buf

	switch data := data.(type) {
	case []byte:
		for _, p := range bytes.Split(data, nl) {
			buf.Write(dataBytes)
			buf.Write(p)
			buf.Write(nl)
		}
	case string:
		for _, p := range strings.Split(data, "\n") {
			buf.Write(dataBytes)
			buf.WriteString(p)
			buf.Write(nl)
		}

	default:
		v, err := json.Marshal(data)
		if err != nil {
			return err
		}

		buf.Write(dataBytes)
		buf.Write(v)
		buf.Write(nl)
	}

	buf.Write(nl)

	_, err := buf.WriteTo(ss.wf)
	ss.wf.Flush()

	return err
}

func (ss *SSEStream) SendAll(id, evt string, msg interface{}) error {
	buf := ss.buf

	if id != "" {
		buf.Write(idBytes)
		buf.WriteString(id)
		buf.WriteByte('\n')
	}

	if evt != "" {
		buf.Write(evtBytes)
		buf.WriteString(evt)
		buf.WriteByte('\n')
	}

	return ss.SendData(msg)
}

func ConvertToSSE(ctx *apiserv.Context) (lastEventID string, ss *SSEStream, err error) {
	wf, ok := ctx.ResponseWriter.(writeFlusher)
	if !ok {
		err = ErrNotAFlusher
		return
	}

	h := ctx.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	wf.Flush()

	ss = &SSEStream{
		wf:  wf,
		buf: bytes.NewBuffer(nil),
	}
	lastEventID = ctx.ReqHeader().Get("Last-Event-ID")

	return
}
