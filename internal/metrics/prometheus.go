package metrics

import (
	"net/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// returns the standard Prometheus HTTP handler
// used to mount /metrics route in routes.go
func Handler() http.Handler {
	return promhttp.Handler()
}
