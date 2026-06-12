package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tomtorh96/task-scheduler/internal/api"
	"github.com/tomtorh96/task-scheduler/internal/pool"
	"github.com/tomtorh96/task-scheduler/internal/scheduler"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestServer spins up a real Server backed by a real Scheduler and returns
// an httptest.Server so tests can make real HTTP requests without binding a port.
func newTestServer(t *testing.T) (*httptest.Server, *scheduler.Scheduler) {
	t.Helper()
	sched := scheduler.NewScheduler(2, 100)
	sched.Start()
	srv := api.NewServer(sched, "0") // port "0" is ignored — httptest handles it
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(func() {
		ts.Close()
		sched.Shutdown(5 * time.Second)
	})
	return ts, sched
}

// post is a small helper that sends a POST with a JSON body and returns the response.
func post(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

// get sends a GET and returns the response.
func get(t *testing.T, url string) *http.Response {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return resp
}

// delete sends a DELETE and returns the response.
func delete(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("new DELETE request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE %s: %v", url, err)
	}
	return resp
}

// decodeJSON decodes a JSON response body into v.
func decodeJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
}

// assertStatus fails the test if the response status code does not match want.
func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Errorf("expected status %d, got %d", want, resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

func TestHealthEndpoint(t *testing.T) {
	ts, _ := newTestServer(t)

	resp := get(t, ts.URL+"/health")
	assertStatus(t, resp, http.StatusOK)

	var body map[string]string
	decodeJSON(t, resp, &body)
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %q", body["status"])
	}
}

// ---------------------------------------------------------------------------
// Submit job
// ---------------------------------------------------------------------------

func TestSubmitJob_Returns201WithID(t *testing.T) {
	ts, _ := newTestServer(t)

	resp := post(t, ts.URL+"/jobs", map[string]int{
		"priority":    5,
		"max_retries": 3,
	})
	assertStatus(t, resp, http.StatusCreated)

	var body map[string]string
	decodeJSON(t, resp, &body)
	if body["id"] == "" {
		t.Error("expected a job ID in response, got empty string")
	}
}

func TestSubmitJob_InvalidBody_Returns400(t *testing.T) {
	ts, _ := newTestServer(t)

	resp, err := http.Post(ts.URL+"/jobs", "application/json", bytes.NewBufferString("not json"))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	assertStatus(t, resp, http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// Get job
// ---------------------------------------------------------------------------

func TestGetJob_Returns200WithJobDetails(t *testing.T) {
	ts, _ := newTestServer(t)

	// submit a job first
	submitResp := post(t, ts.URL+"/jobs", map[string]int{"priority": 1, "max_retries": 1})
	var submitBody map[string]string
	decodeJSON(t, submitResp, &submitBody)
	id := submitBody["id"]

	// fetch it back
	resp := get(t, ts.URL+"/jobs/"+id)
	assertStatus(t, resp, http.StatusOK)

	var job map[string]any
	decodeJSON(t, resp, &job)
	if job["id"] != id {
		t.Errorf("expected job ID %s, got %v", id, job["id"])
	}
	if job["priority"] != float64(1) {
		t.Errorf("expected priority 1, got %v", job["priority"])
	}
}

func TestGetJob_UnknownID_Returns404(t *testing.T) {
	ts, _ := newTestServer(t)

	resp := get(t, ts.URL+"/jobs/does-not-exist")
	assertStatus(t, resp, http.StatusNotFound)
}

// ---------------------------------------------------------------------------
// List jobs
// ---------------------------------------------------------------------------

func TestListJobs_ReturnsAllJobs(t *testing.T) {
	ts, _ := newTestServer(t)

	// submit 3 jobs
	for i := 0; i < 3; i++ {
		post(t, ts.URL+"/jobs", map[string]int{"priority": i + 1, "max_retries": 1})
	}

	resp := get(t, ts.URL+"/jobs")
	assertStatus(t, resp, http.StatusOK)

	var jobs []map[string]any
	decodeJSON(t, resp, &jobs)
	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}
}

func TestListJobs_FilterByStatus(t *testing.T) {
	ts, sched := newTestServer(t)

	// submit a job and wait for it to complete
	submitResp := post(t, ts.URL+"/jobs", map[string]int{"priority": 1, "max_retries": 1})
	var submitBody map[string]string
	decodeJSON(t, submitResp, &submitBody)
	id := submitBody["id"]

	// wait for done
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		job, _ := sched.GetJob(id)
		if job.GetStatus() == pool.StatusDone {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	resp := get(t, ts.URL+"/jobs?status=done")
	assertStatus(t, resp, http.StatusOK)

	var jobs []map[string]any
	decodeJSON(t, resp, &jobs)
	if len(jobs) == 0 {
		t.Error("expected at least one done job, got none")
	}
	for _, j := range jobs {
		if j["status"] != "done" {
			t.Errorf("expected status done, got %v", j["status"])
		}
	}
}

// ---------------------------------------------------------------------------
// Cancel job
// ---------------------------------------------------------------------------

func TestCancelJob_Returns200(t *testing.T) {
	ts, _ := newTestServer(t)

	// submit without starting the scheduler so job stays pending
	sched := scheduler.NewScheduler(2, 100)
	pendingSrv := api.NewServer(sched, "0")
	pts := httptest.NewServer(pendingSrv.Router())
	defer pts.Close()

	submitResp := post(t, pts.URL+"/jobs", map[string]int{"priority": 1, "max_retries": 1})
	var submitBody map[string]string
	decodeJSON(t, submitResp, &submitBody)
	id := submitBody["id"]

	resp := delete(t, pts.URL+"/jobs/"+id)
	assertStatus(t, resp, http.StatusOK)
	_ = ts // suppress unused warning
}

func TestCancelJob_UnknownID_Returns404(t *testing.T) {
	ts, _ := newTestServer(t)

	resp := delete(t, ts.URL+"/jobs/does-not-exist")
	assertStatus(t, resp, http.StatusNotFound)
}

func TestCancelJob_AlreadyDone_Returns409(t *testing.T) {
	ts, sched := newTestServer(t)

	submitResp := post(t, ts.URL+"/jobs", map[string]int{"priority": 1, "max_retries": 1})
	var submitBody map[string]string
	decodeJSON(t, submitResp, &submitBody)
	id := submitBody["id"]

	// wait for job to complete
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		job, _ := sched.GetJob(id)
		if job.GetStatus() == pool.StatusDone {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	resp := delete(t, ts.URL+"/jobs/"+id)
	assertStatus(t, resp, http.StatusConflict)
}

// ---------------------------------------------------------------------------
// Workers
// ---------------------------------------------------------------------------

func TestWorkersEndpoint_ReturnsStats(t *testing.T) {
	ts, _ := newTestServer(t)

	resp := get(t, ts.URL+"/workers")
	assertStatus(t, resp, http.StatusOK)

	var stats map[string]any
	decodeJSON(t, resp, &stats)
	if _, ok := stats["total_workers"]; !ok {
		t.Error("expected total_workers in response")
	}
	if _, ok := stats["active_workers"]; !ok {
		t.Error("expected active_workers in response")
	}
}
