package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tomtorh96/task-scheduler/internal/pool"
	"github.com/tomtorh96/task-scheduler/internal/scheduler"
)

type submitRequest struct {
	Priority   int `json:"priority"`
	MaxRetries int `json:"max_retries"`
}

type jobView struct {
	ID         string      `json:"id"`
	Priority   int         `json:"priority"`
	MaxRetries int         `json:"max_retries"`
	Attempt    int         `json:"attempt"`
	Status     pool.Status `json:"status"`
	CreatedAt  time.Time   `json:"created_at"`
	StartedAt  time.Time   `json:"started_at,omitempty"`
	FinishedAt time.Time   `json:"finished_at,omitempty"`
}

func toJobView(j *pool.Job) jobView {
	return jobView{
		ID:         j.ID,
		Priority:   j.Priority,
		MaxRetries: j.MaxRetries,
		Attempt:    j.Attempt,
		Status:     j.Status,
		CreatedAt:  j.CreatedAt,
		StartedAt:  j.StartedAt,
		FinishedAt: j.FinishedAt,
	}
}

// decode JSON body: {priority int, max_retries int}
// call s.scheduler.Submit() with a dummy Fn for now
// respond 201 with {id: job_id}
func (s *Server) handleSubmitJob(w http.ResponseWriter, r *http.Request) {
	var req submitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	id, err := s.scheduler.Submit(func() error { return nil }, req.Priority, req.MaxRetries)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

// optional query param ?status=pending
// return all jobs as JSON array, filtered by status if provided
func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	statusFilter := pool.Status(r.URL.Query().Get("status"))
	jobs := s.scheduler.ListJobs(statusFilter)
	result := make([]jobView, 0, len(jobs))
	for _, j := range jobs {
		result = append(result, toJobView(j))
	}
	writeJSON(w, http.StatusOK, result)
}

// get {id} from URL using chi.URLParam(r, "id")
// call s.scheduler.GetJob(id)
// respond 200 with job JSON, or 404 if ErrJobNotFound
func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := s.scheduler.GetJob(id)
	if err != nil {
		if err == scheduler.ErrJobNotFound {
			http.Error(w, "job not found", http.StatusNotFound)
		} else {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}
	writeJSON(w, http.StatusOK, toJobView(job))
}

// get {id} from URL
// call s.scheduler.Cancel(id)
// respond 200 on success, 404 if ErrJobNotFound, 409 if ErrJobNotCancellable
func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := s.scheduler.Cancel(id)
	if err != nil {
		switch err {
		case scheduler.ErrJobNotFound:
			http.Error(w, "job not found", http.StatusNotFound)
		case scheduler.ErrJobNotCancellable:
			http.Error(w, "job is not cancellable", http.StatusConflict)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}

// respond 200 with stats JSON
func (s *Server) handleGetWorkers(w http.ResponseWriter, r *http.Request) {
	stats := s.scheduler.Stats()
	writeJSON(w, http.StatusOK, stats)
}

// respond 200 with {status: "ok"}
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// sets Content-Type: application/json
// sets the status code
// encodes v as JSON and writes it to w
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
