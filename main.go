package main

import (
	"context"
	"flag"
	"log/slog"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metrics string
	sys     string
	listen  string
	help    bool
)

func main() {
	flag.StringVar(&metrics, "m", "/metrics", "set metrics path")
	flag.StringVar(&sys, "s", "", "set system metrics path")
	flag.StringVar(&listen, "l", ":8188", "set listen address")
	flag.BoolVar(&help, "h", false, "show help")
	flag.Parse()
	if help {
		flag.Usage()
		return
	}

	r := prometheus.NewRegistry()
	col := NewCollector()
	defer col.Close()
	r.MustRegister(col)
	handler := promhttp.HandlerFor(r, promhttp.HandlerOpts{})
	server := NewHttpServer()
	SignalsCallback(func() { server.Shutdown(context.Background()) }, true, syscall.SIGINT, syscall.SIGTERM)
	server.Handle(metrics, handler)
	if len(sys) != 0 {
		server.Handle(sys, promhttp.Handler())
	}
	err := server.ListenAndServe(listen)
	if err != nil {
		slog.Error("http server exit with error", "err", err)
	}
}
