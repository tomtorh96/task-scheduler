package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

const defaultURL = "http://localhost:8080"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "submit":
		submitCmd()
	case "status":
		statusCmd()
	case "list":
		listCmd()
	case "cancel":
		cancelCmd()
	case "workers":
		workersCmd()
	default:
		printUsage()
		os.Exit(1)
	}
}

// returns SCHEDULER_URL env var if set, otherwise defaultURL
func getBaseURL() string {
	if url := os.Getenv("SCHEDULER_URL"); url != "" {
		return url
	}
	return defaultURL
}

// prints a help message listing all available commands and their flags
func printUsage() {
	println("Usage: taskctl <command> [flags]")
	println("Commands:")
	println("  submit   --priority <int> --max-retries <int>   Submit a new job")
	println("  status    <job_id>                             Get status of a job")
	println("  list      --status <status>                    List jobs, optionally filtered by status")
	println("  cancel    <job_id>                             Cancel a job")
	println("  workers                                      Get worker stats")
}

// pretty prints raw JSON bytes to stdout with indentation
// use json.MarshalIndent or json.Indent
func printJSON(data []byte) {
	var buf bytes.Buffer
	err := json.Indent(&buf, data, "", "  ")
	if err != nil {
		// data wasn't valid JSON, just print it raw
		fmt.Println(string(data))
		return
	}
	fmt.Println(buf.String())
}

// parses flags: --priority (default 0), --max-retries (default 3)
// POST {priority, max_retries} to /jobs
// prints the returned job ID
func submitCmd() {
	// create a new flag set for this subcommand
	flags := flag.NewFlagSet("submit", flag.ExitOnError)
	priority := flags.Int("priority", 0, "job priority")
	maxRetries := flags.Int("max-retries", 3, "max retry attempts")
	flags.Parse(os.Args[2:]) // parse everything after "submit"
	// use *priority and *maxRetries
	body, _ := json.Marshal(map[string]int{
		"priority":    *priority,
		"max_retries": *maxRetries,
	})
	resp, err := http.Post(getBaseURL()+"/jobs", "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	printJSON(respBody)

}

// expects os.Args[2] to be a job ID
// GET /jobs/{id}
// prints the job as formatted JSON
// exits with error if no ID provided
func statusCmd() {
	if len(os.Args) < 3 {
		println("Error: job ID required")
		os.Exit(1)
	}
	id := os.Args[2]
	url := getBaseURL() + "/jobs/" + id
	resp, err := http.Get(url)
	if err != nil {
		println("Error fetching job status:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		println("Error: job not found")
		os.Exit(1)
	} else if resp.StatusCode != http.StatusOK {
		println("Error fetching job status:", string(respBody))
		os.Exit(1)
	}
	printJSON(respBody)
}

// parses optional flag: --status (default empty = all)
// GET /jobs?status=<value>
// prints the job array as formatted JSON
func listCmd() {
	flags := flag.NewFlagSet("list", flag.ExitOnError)
	status := flags.String("status", "", "filter jobs by status")
	flags.Parse(os.Args[2:])
	url := getBaseURL() + "/jobs"
	if *status != "" {
		url += "?status=" + *status
	}
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	printJSON(respBody)
}

// expects os.Args[2] to be a job ID
// DELETE /jobs/{id}
// prints "job cancelled" on success
// prints error message on 404 or 409
// exits with error if no ID provided
func cancelCmd() {
	if len(os.Args) < 3 {
		println("Error: job ID required")
		os.Exit(1)
	}
	id := os.Args[2]
	req, err := http.NewRequest(http.MethodDelete, getBaseURL()+"/jobs/"+id, nil)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Println("job cancelled")
	case http.StatusNotFound:
		fmt.Println("error: job not found")
	case http.StatusConflict:
		fmt.Println("error: job is not cancellable (already running, done, or failed)")
	default:
		fmt.Println("error: unexpected status", resp.StatusCode)
	}
}

// GET /workers
// prints stats as formatted JSON

func workersCmd() {
	resp, err := http.Get(getBaseURL() + "/workers")
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	printJSON(respBody)
}
