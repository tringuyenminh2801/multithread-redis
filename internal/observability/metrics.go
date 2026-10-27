package observability

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PoolStats is what the metrics endpoint reads from the goroutine pool.
// The values are sampled on each scrape, so the hot path only touches atomics.
type PoolStats interface {
	Size() int
	Active() int64
	Queued() int
	Handled() uint64
}

// NewHTTPServer serves /metrics (Prometheus format) and /healthz on addr.
func NewHTTPServer(addr string, pool PoolStats) *http.Server {
	reg := prometheus.NewRegistry()

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "redis_pool_workers",
			Help: "Number of worker goroutines in the pool.",
		}, func() float64 { return float64(pool.Size()) }),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "redis_connections_active",
			Help: "Connections currently being handled by a worker.",
		}, func() float64 { return float64(pool.Active()) }),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "redis_pool_queue_depth",
			Help: "Accepted connections waiting for a free worker.",
		}, func() float64 { return float64(pool.Queued()) }),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Name: "redis_connections_handled_total",
			Help: "Connections fully handled since start.",
		}, func() float64 { return float64(pool.Handled()) }),
	)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

// Healthcheck calls /healthz on the local metrics server. The container image
// has no shell or curl, so Docker's HEALTHCHECK runs `server healthcheck`.
func Healthcheck(metricsAddr string) error {
	host, port, err := net.SplitHostPort(metricsAddr)
	if err != nil {
		return fmt.Errorf("invalid metrics address %q: %w", metricsAddr, err)
	}

	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}

	client := http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get("http://" + net.JoinHostPort(host, port) + "/healthz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz returned %s", resp.Status)
	}

	return nil
}
