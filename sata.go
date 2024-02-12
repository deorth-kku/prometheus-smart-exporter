package main

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/bits"
	"strconv"
	"strings"

	"github.com/anatol/smart.go"
	"github.com/dustin/go-humanize"
	"github.com/prometheus/client_golang/prometheus"
)

const metric_sata = metric_head + "sata_"

var sata_metrics = make(map[string]*prometheus.Desc)

type SataDev struct {
	name      string
	dev       *smart.SataDevice
	info_sent bool
}

func NewSataDev(name string, smartdev *smart.SataDevice) *SataDev {
	return &SataDev{name, smartdev, false}
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

const sata_info_metric = metric_sata + "Info"

var tags_sata_info = []string{
	tag_dev,
	"Device_Model",
	"Serial_Number",
	"LU_WWN_Device_Id",
	"Firmware_Version",
	"User_Capacity",
	"Sector_Sizes",
	"Sectors",
	"Rotation_Rate",
	// I do not know how smartctl read "Form Factor"
	"SATA_Version",
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
	sata_metrics[sata_info_metric] = prometheus.NewDesc(sata_info_metric, "", tags_sata_info, nil)
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
		case 03:
			attr.ValueRaw, _ = ParseSpinUpTime(attr.ValueRaw)
			fallthrough
		default:
			template.Value = float64(attr.ValueRaw)
		}
		out = append(out, template)
	}

	template.Desc, ok = sata_metrics[sata_info_metric]
	if ok && !d.info_sent {
		d.info_sent = true
		id, err := d.dev.Identify()
		if err == nil {
			sectors, capacity, logicalSectorSize, physicalSectorSize, _ := id.Capacity()
			template.Tags = []string{
				d.name,
				id.ModelNumber(),
				id.SerialNumber(),
				strconv.FormatUint(id.WWN(), 16),
				id.FirmwareRevision(),
				fmt.Sprintf("%s bytes [%s]", humanize.Comma(int64(capacity)), humanize.Bytes(capacity)),
				fmt.Sprintf("%d bytes logical, %d bytes physical", logicalSectorSize, physicalSectorSize),
				humanize.Comma(int64(sectors)),
				fmt.Sprintf("%d rpm", id.RotationRate),
				ParseSATAVersion(id),
			}
			template.Type = prometheus.GaugeValue
			template.Value = 0
			out = append(out, template)
		}
	}
	return
}

func (d *SataDev) Close() error {
	return d.dev.Close()
}

func ParseSpinUpTime(raw uint64) (current uint64, average uint64) {
	current = raw & 0xFFF
	average = (raw & 0xFFF0000) >> 16
	return
}

const (
	sata1_speed = "1.5 Gb/s"
	sata2_speed = "3.0 Gb/s"
	sata3_speed = "6.0 Gb/s"
)

var current_speeds = map[uint16]string{
	2: sata1_speed,
	4: sata2_speed,
	6: sata3_speed,
}

var sata_versions = map[int]string{
	0: "ATA8-AST",
	1: "SATA 1.0a",
	2: "SATA II Ext",
	3: "SATA 2.5",
	4: "SATA 2.6",
	5: "SATA 3.0",
	6: "SATA 3.1",
	7: "SATA 3.2",
	8: "SATA 3.3",
	9: "SATA 3.4",
}

func getSupportedSpeed(SATACap uint16) string {
	switch {
	case SATACap&0x0008 != 0:
		return sata3_speed
	case SATACap&0x0004 != 0:
		return sata2_speed
	case SATACap&0x0002 != 0:
		return sata1_speed
	default:
		return "unknown"
	}
}

func ParseSATAVersion(d *smart.AtaIdentifyDevice) (sata_version string) {
	return fmt.Sprintf("%s, %s (current %s)", sata_versions[Log2b(uint(d.TransportMajor&0xfff))], getSupportedSpeed(d.SATACap), current_speeds[d.SATACapAddl&0xe])
}

func Log2b(x uint) int {
	if x == 0 {
		return 0
	}

	return bits.Len(x) - 1
}
