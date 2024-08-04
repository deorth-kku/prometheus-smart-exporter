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
	metrics   string
	sys       string
	listen    string
	skip_devs arrayFlags
	help      bool
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	flag.StringVar(&metrics, "m", "/metrics", "set metrics path")
	flag.StringVar(&sys, "s", "", "set system metrics path")
	flag.StringVar(&listen, "l", ":8188", "set listen address")
	flag.Var(&skip_devs, "skip", "set skipped devs")
	flag.BoolVar(&help, "h", false, "show help")
	flag.Parse()
	if help {
		flag.Usage()
		return
	}

	r := prometheus.NewRegistry()
	col := NewCollector(skip_devs...)
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
