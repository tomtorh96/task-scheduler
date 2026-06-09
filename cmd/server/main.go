package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/tomtorh96/task-scheduler/internal/api"
	"github.com/tomtorh96/task-scheduler/internal/scheduler"
)

/*
1. Read config from environment variables (worker count, port, buffer size)
2. Create the Scheduler
3. Start the Scheduler
4. Create the HTTP Server
5. Start the HTTP Server in a goroutine
6. Block waiting for SIGINT or SIGTERM (Ctrl+C or kill signal)
7. On signal received — gracefully shut down the HTTP server, then the scheduler
*/
func main() {
	bufferSize, err := strconv.Atoi(os.Getenv("APP_BUFFER_SIZE"))
	if err != nil {
		bufferSize = 100
	}
	workerCount, err := strconv.Atoi(os.Getenv("APP_WORKER_COUNT"))
	if err != nil {
		workerCount = 5
	}
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	scheduler := scheduler.NewScheduler(workerCount, bufferSize)
	scheduler.Start()
	server := api.NewServer(scheduler, port)
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	server.Shutdown(ctx)
	scheduler.Shutdown(30 * time.Second)
}
