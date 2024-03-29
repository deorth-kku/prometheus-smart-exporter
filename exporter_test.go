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

func TestHttp(t *testing.T) {
	server := NewHttpServer()
	server.ListenAndServe("/tmp/123.sock,0666")
}

func TestParseSpinUpTime(t *testing.T) {
	// See https://github.com/netdata/netdata/issues/5919#issuecomment-487087591
	cur, avg := ParseSpinUpTime(38684000679)
	if cur == 423 && avg == 447 {

	} else {
		t.Errorf("incorrect value %d %d", cur, avg)
	}
}
