package main

import (
	"encoding/hex"
	"log/slog"
	"strings"

	"github.com/anatol/smart.go"
	"github.com/prometheus/client_golang/prometheus"
)

const metric_sata = metric_head + "sata_"

var sata_metrics = make(map[string]*prometheus.Desc)

type SataDev struct {
	name string
	dev  *smart.SataDevice
}

func NewSataDev(name string, smartdev *smart.SataDevice) *SataDev {
	return &SataDev{name, smartdev}
}

func (d *SataDev) Name() string {
	return d.name
}

func getMetricName(attrName string, num uint8) (metricName string) {
	switch attrName {
	case "":
		return metric_sata + "Unknown_Attribute_" + toHex(num)
	case "Power-Off_Retract_Count":
		return metric_sata + "Power_Off_Retract_Count"
	default:
		return metric_sata + attrName
	}
}

func toHex(num uint8) string {
	return strings.ToUpper(hex.EncodeToString([]byte{num}))
}

func (d *SataDev) ListMetrics() map[string]*prometheus.Desc {
	data, err := d.dev.ReadSMARTData()
	if err != nil {
		return sata_metrics
	}
	var name string
	var ok bool
	for num, attr := range data.Attrs {
		name = getMetricName(attr.Name, num)
		if _, ok = sata_metrics[name]; ok {
			continue
		}
		sata_metrics[name] = prometheus.NewDesc(name, toHex(num), tags_dev_only, nil)
	}
	return sata_metrics
}

func (d *SataDev) GetMetrics() (out []PromValue) {
	data, err := d.dev.ReadSMARTData()
	if err != nil {
		return
	}
	var name string
	var ok bool
	template := PromValue{
		Type: prometheus.GaugeValue,
		Tags: []string{d.name},
	}
	for num, attr := range data.Attrs {
		name = getMetricName(attr.Name, num)
		if template.Desc, ok = sata_metrics[name]; !ok {
			slog.Warn("failed to find metric, didn't run ListMetrics?", "name", name)
			continue
		}
		switch num {
		case 231: // disabled attr
			continue
		case 194:
			var temp int
			temp, _, _, _, err = attr.ParseAsTemperature()
			if err != nil {
				slog.Warn("failed to parse temp", "dev", d.dev, "err", err)
				continue
			}
			template.Value = float64(temp)
		case 03: // Spin_Up_Time, don't know how to parse this. not parsed for now
			fallthrough
		default:
			template.Value = float64(attr.ValueRaw)
		}
		out = append(out, template)
	}
	return
}

func (d *SataDev) Close() error {
	return d.dev.Close()
}
