package main

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func recordMetrics() {
	go func() {
		for {
			opsProcessed.Inc()
			cpuPercentAvg.Inc()
			cpuPercentAvgDeux.Inc()
			time.Sleep(1 * time.Hour)
		}
	}()
}

var (
	labels       = map[string]string{"resource_name": "fake-vm-01", "resource_group": "fake-rg-01"}
	labelsDeux   = map[string]string{"resource_name": "fake-vm-02", "resource_group": "fake-rg-02"}
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "myapp_processed_ops_total",
		Help:        "The total number of processed events",
		ConstLabels: labels,
	})
	cpuPercentAvg = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "percentage_cpu_percent_average",
		Help:        "cpu percentage average",
		ConstLabels: labels,
	})
	cpuPercentAvgDeux = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "percentage_cpu_percent_average",
		Help:        "cpu percentage average",
		ConstLabels: labelsDeux,
	})
)

func main() {
	recordMetrics()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
