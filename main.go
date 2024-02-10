package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	col := NewCollector()
	prometheus.MustRegister(col)
	http.ListenAndServe(":8188", promhttp.Handler())
}
