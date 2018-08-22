package apiutils

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/missionMeteora/apiserv"
)

var (
	eol       = []byte("\n\n")
	mlSep     = []byte("\ndata: ")
	idBytes   = []byte("id: ")
	evtBytes  = []byte("event: ")
	dataBytes = []byte("data: ")
)

type SSEData chan []byte

type SSEHandlerFunc func(ctx *apiserv.Context) (stream SSEData, errResp apiserv.Response)

func NewSSERouter(clientChannelSize int) *SSERouter {
	if clientChannelSize < 1 {
		clientChannelSize = 1
	}

	s := &SSERouter{
		clients: make(map[SSEData]struct{}, 8),
		chSize:  clientChannelSize,

		add:  make(chan SSEData, 1),
		del:  make(chan SSEData, 1),
		evts: make(chan *bytes.Buffer),
		p: sync.Pool{
			New: func() interface{} { return bytes.NewBuffer(make([]byte, 0, 1024)) },
		},
	}

	go s.process()

	return s
}

type SSERouter struct {
	clients map[SSEData]struct{}
	chSize  int

	add  chan SSEData
	del  chan SSEData
	evts chan *bytes.Buffer

	p sync.Pool
}

func (s *SSERouter) SendAll(id, evt string, msg interface{}) error {
	buf := s.p.Get().(*bytes.Buffer)

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

	buf.Write(dataBytes)

	switch msg := msg.(type) {
	case []byte:
		if bytes.ContainsRune(msg, '\n') {
			msg = bytes.Join(bytes.Split(msg, eol[:1]), mlSep)
		}
		buf.Write(msg)

	default:
		v, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		buf.Write(v)
	}

	buf.Write(eol)
	s.evts <- buf

	return nil
}

func (s *SSERouter) Handler(ctx *apiserv.Context) (_ apiserv.Response) {
	f, ok := ctx.ResponseWriter.(http.Flusher)
	if !ok {
		return apiserv.NewJSONErrorResponse(http.StatusInternalServerError, "not a flusher")
	}

	h := ctx.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	f.Flush()

	ch := make(SSEData, s.chSize)
	doneCh := ctx.Req.Context().Done()

	s.add <- ch
	defer func() { s.del <- ch }()

	for {
		select {
		case data := <-ch:
			if _, err := ctx.Write(data); err != nil {
				return
			}
			f.Flush()
		case <-doneCh:
			return
		}
	}
}

func (s *SSERouter) process() {
	for {
		select {
		case ch := <-s.add:
			s.clients[ch] = struct{}{}

		case ch := <-s.del:
			delete(s.clients, ch)

		case evt := <-s.evts:
			b := evt.Bytes()

			for ch := range s.clients {
				if !trySend(ch, b) {
					delete(s.clients, ch)
				}
			}

			evt.Reset()
			s.p.Put(evt)
		}
	}
}

func trySend(ch SSEData, evt []byte) bool {
	select {
	case ch <- evt:
		return true
	default:
		return false
	}
}

// func SSEHandler(fn SSEHandlerFunc) apiserv.Handler {
// 	return func(ctx *apiserv.Context) apiserv.Response {
// 		ch, resp := fn(ctx)
// 		if resp != nil {
// 			return resp
// 		}

// 		flusher, ok := ctx.ResponseWriter.(http.Flusher)
// 		if !ok {
// 			return apiserv.NewJSONErrorResponse(500, "not a flusher")
// 		}

// 		for {

// 		}
// 	}
// }

func dummy(*apiserv.Context) apiserv.Response { return nil }
