package apiserv

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/missionMeteora/apiv2/router"
)

// New returns a server
func New(opts interface{}) *Server {
	srv := &Server{
		srv: &http.Server{},
		r:   router.New(nil),
	}

	srv.r.PanicHandler = func(w http.ResponseWriter, req *http.Request, v interface{}) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(http.StatusInternalServerError, fmt.Sprint(v)))
		srv.srv.ErrorLog.Printf("PANIC (%T): %+v", v, v)
	}

	srv.r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(NewErrorResponse(http.StatusNotFound, "those damn cats stole the endpoint again."))
	})

	return srv
}

// Server is the main server
type Server struct {
	srv *http.Server
	r   *router.Router

	ctxPool sync.Pool
}

// AddRoute adds a handler (or more) to the specific method and path
func (s *Server) AddRoute(method, path string, handlers ...Handler) error {
	return s.r.AddRoute(method, path, handlerChain(handlers).Serve)
}
