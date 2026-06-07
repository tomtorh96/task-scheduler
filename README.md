# task-scheduler

A concurrent task scheduler built in Go. Features a priority job queue, configurable worker pool, exponential backoff retries, HTTP API, and CLI client.

## Quick start

```bash
# run the server (default: 4 workers, port 8080)
make run

# submit a job
./bin/taskctl submit --priority=5

# check status
./bin/taskctl status <job-id>

# list all jobs
./bin/taskctl list --status=pending
```

## Build

```bash
make build        # builds server + CLI into bin/
make test         # runs all tests with race detector
make bench        # runs benchmark suite
make lint         # runs golangci-lint
```

## Project structure

```
cmd/server          http server entrypoint
cmd/taskctl         cli client entrypoint
internal/pool       worker pool
internal/queue      priority job queue
internal/scheduler  wires pool + queue, exposes Submit/Cancel
internal/api        http handlers and routes
internal/metrics    prometheus metrics
pkg/backoff         exponential backoff with jitter
tests/              integration tests and benchmarks
```

## Benchmarks

> fill in after running: make bench

## Architecture

Client -> HTTP API -> Scheduler -> Priority Queue -> Worker Pool -> Job execution
