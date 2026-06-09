package api

func (s *Server) registerRoutes() {
	s.router.Post("/jobs", s.handleSubmitJob)
	s.router.Get("/jobs", s.handleListJobs)
	s.router.Get("/jobs/{id}", s.handleGetJob)
	s.router.Delete("/jobs/{id}", s.handleCancelJob)
	s.router.Get("/workers", s.handleGetWorkers)
	s.router.Get("/health", s.handleHealth)
}

// registers all routes on the chi router:
// POST   /jobs           → s.handleSubmitJob
// GET    /jobs           → s.handleListJobs
// GET    /jobs/{id}      → s.handleGetJob
// DELETE /jobs/{id}      → s.handleCancelJob
// GET    /workers        → s.handleGetWorkers
// GET    /health         → s.handleHealth
