package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	JobsSubmitted prometheus.Counter
	JobsCompleted prometheus.Counter
	JobsFailed    prometheus.Counter
	JobsCancelled prometheus.Counter
	QueueDepth    prometheus.Gauge
	ActiveWorkers prometheus.Gauge
	JobDuration   prometheus.Histogram
	JobQueueWait  prometheus.Histogram
)

// creates and registers all metrics with Prometheus
// called once at startup from main.go
func Init() {
	JobsSubmitted = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "scheduler_jobs_submitted_total",
		Help: "Number of jobs submitted",
	})

	JobsCompleted = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "scheduler_jobs_completed_total",
		Help: "Number of jobs completed",
	})

	JobsFailed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "scheduler_jobs_failed_total",
		Help: "Number of jobs failed",
	})

	JobsCancelled = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "scheduler_jobs_cancelled_total",
		Help: "Number of jobs cancelled",
	})

	QueueDepth = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "scheduler_queue_depth",
		Help: "Current depth of the job queue",
	})

	ActiveWorkers = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "scheduler_active_workers",
		Help: "Number of currently active workers",
	})

	JobDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "scheduler_job_duration_seconds",
		Help:    "Duration of jobs in seconds",
		Buckets: prometheus.DefBuckets,
	})

	JobQueueWait = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "scheduler_job_queue_wait_seconds",
		Help:    "Time jobs spend in the queue before execution",
		Buckets: prometheus.DefBuckets,
	})

	prometheus.MustRegister(JobsSubmitted, JobsCompleted, JobsFailed, JobsCancelled, QueueDepth, ActiveWorkers, JobDuration, JobQueueWait)
}
