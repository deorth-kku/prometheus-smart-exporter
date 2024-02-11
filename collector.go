package main

import (
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/anatol/smart.go"
	"github.com/prometheus/client_golang/prometheus"
)

type collector struct {
	devs          []PromDev
	metrics_names []string
}

func NewCollector() *collector {
	c := collector{}
	dir, _ := os.ReadDir("/sys/block/")
	for _, disk := range dir {
		var dev smart.Device
		var err error
		for _, prefix := range []string{"loop", "zram", "zd", "sr"} {
			if strings.HasPrefix(disk.Name(), prefix) {
				goto SkipDev
			}
		}
		dev, err = smart.Open("/dev/" + disk.Name())
		if err != nil {
			// some devices (like dmcrypt) do not support SMART interface
			slog.Warn("failed to open smart", "dev", disk.Name(), "err", err)
			continue
		}
		switch sm := dev.(type) {
		case *smart.SataDevice:
			c.devs = append(c.devs, NewSataDev(disk.Name(), sm))
		case *smart.ScsiDevice:
		case *smart.NVMeDevice:
			c.devs = append(c.devs, NewNvmeDev(disk.Name(), sm))
		}
	SkipDev:
	}
	return &c
}

func (c *collector) Close() {
	for _, dev := range c.devs {
		err := dev.Close()
		if err != nil {
			slog.Error("failed to close dev", "dev", dev.Name(), "err", err)
		}
	}
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	for _, dev := range c.devs {
		for name, desc := range dev.ListMetrics() {
			if slices.Contains(c.metrics_names, name) {
				continue
			}
			c.metrics_names = append(c.metrics_names, name)
			ch <- desc
		}
	}
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	var err error
	var metric prometheus.Metric
	for _, dev := range c.devs {
		for _, m := range dev.GetMetrics() {
			metric, err = prometheus.NewConstMetric(m.Desc, m.Type, m.Value, m.Tags...)
			if err != nil {
				slog.Warn("failed to get metric", "args", m, "err", err)
				continue
			}
			ch <- metric
		}
	}
}
