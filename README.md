# task-scheduler

A concurrent task scheduler built in Go from scratch. Features a heap-based priority queue, configurable worker pool, exponential backoff retries, HTTP API, CLI client, and Prometheus metrics — with race-free concurrency verified by Go's race detector.

---

## Features

- **Priority scheduling** — jobs with higher priority preempt lower-priority ones; ties broken by arrival time
- **Worker pool** — configurable number of goroutines, each pulling from a shared job channel
- **Exponential backoff retries** — failed jobs are re-enqueued with jitter after an exponentially increasing delay
- **Graceful shutdown** — in-flight jobs finish before the process exits; configurable drain timeout
- **HTTP API** — submit, inspect, list, and cancel jobs over REST
- **CLI client** — `taskctl` for interacting with the scheduler from the terminal
- **Prometheus metrics** — counters, gauges, and latency histograms exposed at `/metrics`
- **Race-free** — verified with `go test -race` on every CI run

---

## Quick start

```bash
# build server and CLI
make build

# run the server (defaults: 5 workers, port 8080)
./bin/server

# or configure via environment variables
APP_WORKER_COUNT=8 APP_PORT=9090 APP_BUFFER_SIZE=200 ./bin/server
```

### Submit and inspect jobs

```bash
# submit a job
./bin/taskctl submit --priority=5 --max-retries=3

# check status
./bin/taskctl status <job-id>

# list all jobs
./bin/taskctl list

# list by status
./bin/taskctl list --status=done

# cancel a pending job
./bin/taskctl cancel <job-id>

# check worker pool stats
./bin/taskctl workers
```

### HTTP API

| Method   | Endpoint    | Description                                |
| -------- | ----------- | ------------------------------------------ |
| `POST`   | `/jobs`     | Submit a job                               |
| `GET`    | `/jobs`     | List all jobs (optional `?status=` filter) |
| `GET`    | `/jobs/:id` | Get job details                            |
| `DELETE` | `/jobs/:id` | Cancel a job                               |
| `GET`    | `/workers`  | Worker pool stats                          |
| `GET`    | `/metrics`  | Prometheus metrics                         |
| `GET`    | `/health`   | Health check                               |

**Submit a job:**

```bash
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{"priority": 5, "max_retries": 3}'
```

**Response:**

```json
{ "id": "550e8400-e29b-41d4-a716-446655440000" }
```

**Get job status:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "priority": 5,
  "max_retries": 3,
  "attempt": 1,
  "status": "done",
  "created_at": "2026-06-11T15:21:56.05+03:00",
  "started_at": "2026-06-11T15:21:56.05+03:00",
  "finished_at": "2026-06-11T15:21:56.05+03:00"
}
```

---

## Architecture

```
Client
  │
  ▼
HTTP API  ──────────────────────────────────────────────────────────
  │                                                                  │
  ▼                                                                  ▼
Scheduler                                                      Prometheus
  │                                                            /metrics
  ├── Submit() ──► Priority Queue (heap)
  │                      │
  │               dispatch loop
  │                      │
  └── GetJob()           ▼
  └── Cancel()    Worker Pool
  └── ListJobs()    ├── Worker 1
                    ├── Worker 2  ──► job.Fn() ──► backoff retry
                    └── Worker N
```

**Key design decisions:**

The priority queue and worker pool are decoupled via a dispatch loop in the scheduler. Workers read from a buffered channel; the dispatch loop pulls from the heap and feeds the channel. This means priority ordering is enforced at the queue level before jobs enter the pool.

Failed jobs are re-enqueued into the priority queue (not the channel) with their original priority, so they compete fairly against new jobs on retry.

All `Job` field access is protected by a per-job `sync.RWMutex`, allowing safe concurrent reads from the HTTP API while workers write status updates.

---

## Project structure

```
cmd/
  server/         HTTP server entrypoint
  taskctl/        CLI client entrypoint

internal/
  pool/           Worker pool, job struct, stats
  queue/          Heap-based priority queue
  scheduler/      Wires pool + queue, exposes Submit/Cancel/GetJob
  api/            HTTP handlers and routes (chi router)
  metrics/        Prometheus counters, gauges, histograms

pkg/
  backoff/        Exponential backoff with jitter (reusable)

tests/
  integration/    End-to-end tests: job lifecycle, priority, retries, cancel, HTTP API
  benchmark/      Throughput, latency, and queue benchmarks
```

---

## Benchmarks

Run on an Intel Core i7-10510U (4 cores, 8 threads), Windows, Go 1.22.

```
Benchmark                         ops        ns/op     jobs/sec    allocs/op
────────────────────────────────────────────────────────────────────────────
Throughput — 1 worker           2,954,124    413 ns     2,422,216      2
Throughput — 4 workers          2,410,886    600 ns     1,665,773      2
Throughput — 8 workers          2,046,522    909 ns     1,099,632      2
Throughput — 16 workers         1,752,300    937 ns     1,067,850      2
────────────────────────────────────────────────────────────────────────────
Mixed priority load (4 producers, 8 workers)          ~1,923,955 jobs/sec
────────────────────────────────────────────────────────────────────────────
Queue enqueue                   7,457,742    148 ns         —           1
Queue dequeue                   1,888,392    673 ns         —           0
Queue round-trip               13,394,516     90 ns         —           1
────────────────────────────────────────────────────────────────────────────
Scheduling latency                            —          ~0.17 ms dispatch
Scheduler.Submit() cost           960,268  1,499 ns         —           4
```

**Notes:**

Single-worker throughput (~2.4M jobs/sec) exceeds multi-worker throughput for zero-duration jobs. This is expected — with no real work to do, the bottleneck is channel and mutex contention, not CPU. In production workloads where jobs do actual I/O or compute, more workers improve throughput.

Queue round-trip at 90ns shows the heap overhead is negligible. The `673ns` dequeue cost includes mutex acquisition and `sync.Cond` signalling.

`Scheduler.Submit()` costs ~1.5µs per call — dominated by UUID generation (via `github.com/google/uuid`) and map write under `sync.RWMutex`.

---

## Running tests

```bash
# all tests
make test

# verbose
go test -count=1 -v ./...

# integration tests only
go test -count=1 -v ./tests/integration/...

# benchmarks
go test -bench=. -benchmem -count=3 -run=^$ ./tests/benchmark/...
```

---

## CI

GitHub Actions runs on every push to `main` and `dev`:

- Build both binaries
- Run full test suite with race detector (`go test -race`)
- Smoke-run benchmarks

---

## Configuration

| Environment variable | Default                 | Description                 |
| -------------------- | ----------------------- | --------------------------- |
| `APP_PORT`           | `8080`                  | HTTP server port            |
| `APP_WORKER_COUNT`   | `5`                     | Number of worker goroutines |
| `APP_BUFFER_SIZE`    | `100`                   | Job channel buffer size     |
| `SCHEDULER_URL`      | `http://localhost:8080` | Base URL for `taskctl`      |
