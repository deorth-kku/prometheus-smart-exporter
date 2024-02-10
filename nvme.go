package main

import (
	"strconv"

	"github.com/anatol/smart.go"
	"github.com/prometheus/client_golang/prometheus"
)

type NvmeDev struct {
	name     string
	smartdev *smart.NVMeDevice
}

const (
	metric_head                = "smart_exporter_"
	metric_nvme                = metric_head + "nvme_"
	nvmeCritWarning            = metric_nvme + "CritWarning"
	nvmeTemperature            = metric_nvme + "Temperature"
	nvmeAvailSpare             = metric_nvme + "AvailSpare"
	nvmeSpareThresh            = metric_nvme + "SpareThresh"
	nvmePercentUsed            = metric_nvme + "PercentUsed"
	nvmeEnduranceCritWarning   = metric_nvme + "EnduranceCritWarning"
	nvmeDataUnitsRead          = metric_nvme + "DataUnitsRead"
	nvmeDataUnitsWritten       = metric_nvme + "DataUnitsWritten"
	nvmeHostReads              = metric_nvme + "HostReads"
	nvmeHostWrites             = metric_nvme + "HostWrites"
	nvmeCtrlBusyTime           = metric_nvme + "CtrlBusyTime"
	nvmePowerCycles            = metric_nvme + "PowerCycles"
	nvmePowerOnHours           = metric_nvme + "PowerOnHours"
	nvmeUnsafeShutdowns        = metric_nvme + "UnsafeShutdowns"
	nvmeMediaErrors            = metric_nvme + "MediaErrors"
	nvmeNumErrLogEntries       = metric_nvme + "NumErrLogEntries"
	nvmeWarningTempTime        = metric_nvme + "WarningTempTime"
	nvmeCritCompTime           = metric_nvme + "CritCompTime"
	nvmeTempSensor             = metric_nvme + "TempSensor"
	nvmeThermalTransitionCount = metric_nvme + "ThermalTransitionCount"
	nvmeThermalManagementTime  = metric_nvme + "ThermalManagementTime"
	tag_dev                    = "dev"
)

var (
	tags_dev_only  = []string{tag_dev}
	tags_dev_index = []string{tag_dev, "index"}
	nvme_metrics   = list_nvme_metrics()
)

func list_nvme_metrics() (out map[string]*prometheus.Desc) {
	out = make(map[string]*prometheus.Desc)
	normal_metrics := []string{
		nvmeCritWarning,
		nvmeTemperature,
		nvmeAvailSpare,
		nvmeSpareThresh,
		nvmePercentUsed,
		nvmeEnduranceCritWarning,
		nvmeDataUnitsRead,
		nvmeDataUnitsWritten,
		nvmeHostReads,
		nvmeHostWrites,
		nvmeCtrlBusyTime,
		nvmePowerCycles,
		nvmePowerOnHours,
		nvmeUnsafeShutdowns,
		nvmeMediaErrors,
		nvmeNumErrLogEntries,
		nvmeWarningTempTime,
		nvmeCritCompTime,
	}
	for _, metric_name := range normal_metrics {
		out[metric_name] = prometheus.NewDesc(metric_name, "", tags_dev_only, nil)
	}

	metrics_with_index := []string{
		nvmeTempSensor,
		nvmeThermalTransitionCount,
		nvmeThermalManagementTime,
	}
	for _, metric_name := range metrics_with_index {
		out[metric_name] = prometheus.NewDesc(metric_name, "", tags_dev_index, nil)
	}
	return
}

func NewNvmeDev(name string, smartdev *smart.NVMeDevice) *NvmeDev {
	return &NvmeDev{name, smartdev}
}

func (d *NvmeDev) Name() string {
	return d.name
}

func (d *NvmeDev) Close() error {
	return d.smartdev.Close()
}

func (*NvmeDev) ListMetrics() map[string]*prometheus.Desc {
	return nvme_metrics
}

func (d *NvmeDev) GetMetrics() (out []PromValue) {
	info, err := d.smartdev.ReadSMART()
	if err != nil {
		return
	}
	template := PromValue{
		Type: prometheus.GaugeValue,
		Tags: []string{d.name},
	}
	out = make([]PromValue, 30)

	uint8_metrics := map[string]uint8{
		nvmeCritWarning:          info.CritWarning,
		nvmeAvailSpare:           info.AvailSpare,
		nvmeSpareThresh:          info.SpareThresh,
		nvmePercentUsed:          info.PercentUsed,
		nvmeEnduranceCritWarning: info.EnduranceCritWarning,
	}
	uint128_metrics := map[string]smart.Uint128{
		nvmeDataUnitsRead:    info.DataUnitsRead,
		nvmeDataUnitsWritten: info.DataUnitsWritten,
		nvmeHostReads:        info.HostReads,
		nvmeHostWrites:       info.HostWrites,
		nvmeCtrlBusyTime:     info.CtrlBusyTime,
		nvmePowerCycles:      info.PowerCycles,
		nvmePowerOnHours:     info.PowerOnHours,
		nvmeUnsafeShutdowns:  info.UnsafeShutdowns,
		nvmeMediaErrors:      info.MediaErrors,
		nvmeNumErrLogEntries: info.NumErrLogEntries,
	}

	uint32_metrics := map[string]uint32{
		nvmeWarningTempTime: info.WarningTempTime,
		nvmeCritCompTime:    info.CritCompTime,
	}

	uint32_pair_metrics := map[string][2]uint32{
		nvmeThermalTransitionCount: info.ThermalTransitionCount,
		nvmeThermalManagementTime:  info.ThermalManagementTime,
	}

	i := 0

	for name, value := range uint8_metrics {
		template.Desc = nvme_metrics[name]
		template.Value = float64(value)
		out[i] = template
		i++
	}

	template.Desc = nvme_metrics[nvmeTemperature]
	template.Value = float64(info.Temperature)
	out[i] = template
	i++

	template.Desc = nvme_metrics[nvmeTempSensor]
	for index, v := range info.TempSensor {
		template.Tags = []string{d.name, strconv.Itoa(index)}
		template.Value = float64(v)
		out[i] = template
		i++
	}
	template.Tags = []string{d.name}

	template.Type = prometheus.CounterValue

	for name, value := range uint128_metrics {
		template.Desc = nvme_metrics[name]
		template.Value = Uint128toFloat64(value)
		out[i] = template
		i++
	}

	for name, value := range uint32_metrics {
		template.Desc = nvme_metrics[name]
		template.Value = float64(value)
		out[i] = template
		i++
	}

	for name, value := range uint32_pair_metrics {
		template.Desc = nvme_metrics[name]
		for index, v := range value {
			template.Value = float64(v)
			template.Tags = []string{d.name, strconv.Itoa(index)}
			out[i] = template
			i++
		}
	}

	return
}
