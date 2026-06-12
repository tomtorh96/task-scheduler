package pool

type Stats struct {
	TotalWorkers  int `json:"total_workers"`
	ActiveWorkers int `json:"active_workers"`
	IdleWorkers   int `json:"idle_workers"`
	QueueDepth    int `json:"queue_depth"`
	JobsCompleted int `json:"jobs_completed"`
	JobsFailed    int `json:"jobs_failed"`
}
