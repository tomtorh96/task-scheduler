package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tomtorh96/task-scheduler/internal/scheduler"
)

type Server struct {
	scheduler  *scheduler.Scheduler
	router     *chi.Mux
	httpServer *http.Server
}

// creates a new Server, wires the scheduler in, calls registerRoutes()
// sets up the http.Server with the port and router
func NewServer(s *scheduler.Scheduler, port string) *Server {
	srv := &Server{
		scheduler: s,
		router:    chi.NewRouter(),
		httpServer: &http.Server{
			Addr:    ":" + port,
			Handler: nil, // set this to srv.router after registerRoutes()
		},
	}
	srv.httpServer.Handler = srv.router
	srv.registerRoutes()
	return srv
}

// calls httpServer.ListenAndServe() — blocks until the server stops
func (s *Server) Start() error {

	return s.httpServer.ListenAndServe()
}

// gracefully shuts down the HTTP server using httpServer.Shutdown(ctx)
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
