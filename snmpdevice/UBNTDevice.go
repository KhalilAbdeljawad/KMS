package snmpdevice

import (
	"time"
)

func NewSnmpUbnt(ip string) SnmpDevice {
	device := SnmpDevice{
		IP:             ip,
		Communtiy:      "public",
		SnmpVersion:    "1",
		MIBName:        "UBNT-AirMAX-MIB",
		NormalTables:   make(map[string]map[string]string),
		SliceTables:    make(map[string]map[string][]string),
		normalTime:     5 * time.Second,
		UpdateTime:     10 * time.Second,
		TablesAndLines: make(map[string]int),
		canUpdate:      false,
	}

	device.NormalTables["ubntRadioTable"] = make(map[string]string)
	device.NormalTables["ubntAirMaxTable"] = make(map[string]string)
	device.NormalTables["ubntWlStatTable"] = make(map[string]string)
	device.SliceTables["ubntStaTable"] = make(map[string][]string)

	return device
}
