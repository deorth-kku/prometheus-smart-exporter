package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/anatol/smart.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestCollector(t *testing.T) {
	col := NewCollector()
	prometheus.MustRegister(col)
	http.ListenAndServe(":8188", promhttp.Handler())
}

func TestNvme(t *testing.T) {
	path := "/dev/nvme0n1"
	dev, err := smart.OpenNVMe(path)
	if err != nil {
		t.Error(err)
		return
	}
	d := NewNvmeDev(path, dev)
	for _, a := range d.GetMetrics() {
		fmt.Println(a)
	}
}

func TestSata(t *testing.T) {
	path := "/dev/sda"
	dev, err := smart.OpenSata(path)
	if err != nil {
		t.Error(err)
		return
	}
	d := NewSataDev(path, dev)
	d.ListMetrics()
	for _, a := range d.GetMetrics() {
		fmt.Println(a)
	}
}
