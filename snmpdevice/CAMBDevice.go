package snmpdevice

import (
	_ "fmt"
	"time"
)

func NewSnmpCamb(ip string) SnmpDevice {
	device := SnmpDevice{
		IP:             ip,
		Communtiy:      "public",
		SnmpVersion:    "1",
		MIBName:        " CAMBIUM-PMP80211-MIB",
		NormalTables:   make(map[string]map[string]string),
		SliceTables:    make(map[string]map[string][]string),
		normalTime:     5 * time.Second,
		UpdateTime:     10 * time.Second,
		TablesAndLines: make(map[string]int),
		canUpdate:      false,
	}
	device.NormalTables["cambiumGeneralStatus"] = make(map[string]string)
	device.NormalTables["cambiumRFStatus"] = make(map[string]string)
	device.SliceTables["cambiumTDDStatsPerSTATable"] = make(map[string][]string)

	device.TablesAndLines["cambiumRFStatus"] = 12

	return device
}
