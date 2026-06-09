package pool

type Stats struct {
	TotalWorkers  int
	ActiveWorkers int
	IdleWorkers   int
	QueueDepth    int
	JobsCompleted int
	JobsFailed    int
}
