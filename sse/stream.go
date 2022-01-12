package sse

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/missionMeteora/apiserv"
	"github.com/missionMeteora/apiserv/internal"
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

func NewStream(ctx *apiserv.Context, bufSize int) (lastEventID string, ss *Stream, err error) {
	wf, ok := ctx.ResponseWriter.(writeFlusher)
	if !ok {
		err = ErrNotAFlusher
		return
	}

	h := ctx.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")

	ss = &Stream{
		wch:  make(chan []byte, bufSize),
		done: ctx.Req.Context().Done(),
	}
	lastEventID = LastEventID(ctx)

	go processStream(ss, wf)

	return
}

type Stream struct {
	wch  chan []byte
	done <-chan struct{}
}

func (ss *Stream) send(msg []byte) error {
	select {
	case <-ss.done:
		return os.ErrClosed
	case ss.wch <- msg:
		return nil
	default:
		return ErrBufferFull
	}
}

func (ss *Stream) Ping() error {
	return ss.send(pingBytes)
}

func (ss *Stream) Retry(ms int) (err error) {
	return ss.send([]byte(fmt.Sprintf("retry: %d\n\n", ms)))
}

func (ss *Stream) SendData(data interface{}) error {
	b, err := makeData("", "", data)
	if err != nil {
		return err
	}

	return ss.send(b)
}

func (ss *Stream) SendAll(id, evt string, msg interface{}) error {
	b, err := makeData(id, evt, msg)
	if err != nil {
		return err
	}

	return ss.send(b)
}

func processStream(ss *Stream, wf writeFlusher) {
	wf.Flush()

	for {
		select {
		case m := <-ss.wch:
			if _, err := wf.Write(m); err != nil {
				return
			}
			wf.Flush()
		case <-ss.done:
			return
		}
	}
}

func makeData(id, evt string, data interface{}) ([]byte, error) {
	var buf bytes.Buffer

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

	switch data := data.(type) {
	case nil:
		buf.WriteString("data: \n")

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
		v, err := internal.Marshal(data)
		if err != nil {
			return nil, err
		}

		buf.Write(dataBytes)
		buf.Write(v)
		buf.Write(nl)
	}

	buf.Write(nl)

	return buf.Bytes(), nil
}
