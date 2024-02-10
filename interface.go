package main

import "github.com/prometheus/client_golang/prometheus"

type PromDev interface {
	Name() string
	ListMetrics() map[string]*prometheus.Desc
	GetMetrics() []PromValue
	Close() error
}

type PromValue struct {
	Desc  *prometheus.Desc
	Type  prometheus.ValueType
	Value float64
	Tags  []string
}
