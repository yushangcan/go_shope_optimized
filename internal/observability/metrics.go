package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	Requests        = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "seckill_http_requests_total", Help: "HTTP requests handled by the optimized service."}, []string{"route", "method", "status"})
	RequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "seckill_http_request_duration_seconds", Help: "HTTP request duration."}, []string{"route", "method"})
	Admissions      = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "seckill_admissions_total", Help: "Redis Lua admission decisions."}, []string{"result"})
	WorkerEvents    = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "seckill_worker_events_total", Help: "Worker processing outcomes."}, []string{"result"})
)

func init() {
	prometheus.MustRegister(Requests, RequestDuration, Admissions, WorkerEvents)
}

func Handler() http.Handler { return promhttp.Handler() }

func Observe(route, method string, status int, started time.Time) {
	Requests.WithLabelValues(route, method, strconv.Itoa(status)).Inc()
	RequestDuration.WithLabelValues(route, method).Observe(time.Since(started).Seconds())
}
