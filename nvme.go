package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	"github.com/anatol/smart.go"
	"github.com/dustin/go-humanize"
	"github.com/prometheus/client_golang/prometheus"
)

type NvmeDev struct {
	name    string
	dev     *smart.NVMeDevice
	info    []string
	ns_info [][]string
}

const (
	metric_head                = "smart_"
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
	nvmeInfo                   = metric_nvme + "Info"
	nvmeNamespaceInfo          = metric_nvme + "NamespaceInfo"
	tag_dev                    = "dev"
)

var (
	tags_dev_only  = []string{tag_dev}
	tags_dev_index = []string{tag_dev, "index"}
	tags_nvme_info = []string{
		tag_dev,
		"Model_Number",
		"Serial_Number",
		"Firmware_Version",
		"PCI_Vendor_Subsystem_ID",
		"IEEE_OUI_Identifier",
		"Total_NVM_Capacity",
		"Unallocated_NVM_Capacity",
		"Controller_ID",
		"NVMe_Version",
		"Number_of_Namespaces",
	}
	tags_nvme_namespace_info = []string{
		tag_dev,
		"namespace",
		"Size_Capacity",
		"Formatted_LBA_Size",
		"IEEE_EUI_64",
	}
	nvme_metrics = list_nvme_metrics()
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

	out[nvmeInfo] = prometheus.NewDesc(nvmeInfo, "", tags_nvme_info, nil)
	out[nvmeNamespaceInfo] = prometheus.NewDesc(nvmeNamespaceInfo, "", tags_nvme_namespace_info, nil)
	return
}

func NewNvmeDev(name string, smartdev *smart.NVMeDevice) (d *NvmeDev) {
	d = &NvmeDev{name, smartdev, nil, nil}
	id, nss, err := d.dev.Identify()
	if err == nil {
		d.info = []string{
			name,
			id.ModelNumber(),                        // Model_Number
			id.SerialNumber(),                       // Serial_Number
			id.FirmwareRev(),                        // Firmware_Version
			makeUint16ID(id.VendorID),               // PCI_Vendor_Subsystem_ID
			"0x" + hex.EncodeToString(id.IEEE[:]),   // IEEE_OUI_Identifier
			bigCapString(bigFromInt128(id.Tnvmcap)), // Total_NVM_Capacity
			bigCapString(bigFromInt128(id.Unvmcap)), // Unallocated_NVM_Capacity
			makeUint16ID(id.Cntlid),                 // Controller_ID
			makeNvmeVer(id.Ver),                     // NVMe_Version
			strconv.Itoa(len(nss)),                  // Number_of_Namespaces
		}
		for i, ns := range nss {
			d.ns_info = append(d.ns_info, []string{
				name,
				strconv.Itoa(i),
				bigCapString(new(big.Int).Mul(new(big.Int).SetUint64(ns.Nsze), new(big.Int).SetUint64(ns.LbaSize()))), // Size_Capacity
				strconv.FormatUint(ns.LbaSize(), 10),                                      // Formatted_LBA_Size
				hex.EncodeToString(ns.Eui64[:4]) + " " + hex.EncodeToString(ns.Eui64[4:]), // IEEE_EUI_64
			})
		}
	} else {
		d.info = make([]string, len(tags_nvme_info))
		d.info[0] = name
	}
	return
}

func (d *NvmeDev) Name() string {
	return d.name
}

func (d *NvmeDev) Close() error {
	return d.dev.Close()
}

func (*NvmeDev) ListMetrics() map[string]*prometheus.Desc {
	return nvme_metrics
}

func (d *NvmeDev) GetMetrics() (out []PromValue) {
	info, err := d.dev.ReadSMART()
	if err != nil {
		return
	}
	template := PromValue{
		Type: prometheus.GaugeValue,
		Tags: []string{d.name},
	}
	out = make([]PromValue, 31+len(d.ns_info))

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

	template.Desc = nvme_metrics[nvmeInfo]
	template.Value = 0
	template.Type = prometheus.GaugeValue
	template.Tags = d.info
	out[i] = template
	i++

	template.Desc = nvme_metrics[nvmeNamespaceInfo]
	for _, ns := range d.ns_info {
		template.Tags = ns
		out[i] = template
		i++
	}
	return
}

func makeUint16ID(in uint16) string {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, in)
	return "0x" + hex.EncodeToString(b)
}

func bigCapString(cap *big.Int) string {
	cap2 := new(big.Int)
	cap2 = cap2.Add(cap, cap2)
	return fmt.Sprintf("%s bytes [%s]", humanize.BigComma(cap), humanize.BigBytes(cap2))
}

func makeNvmeVer(ver uint32) string {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, ver)
	if b[3] != 0 {
		return "?.?"
	}
	return fmt.Sprintf("%d.%d", b[1], b[2])
}
