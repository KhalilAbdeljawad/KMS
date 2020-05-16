package snmpdevice

import (
	"KMSV2/helpers"
	mymysql "KMSV2/mysql"
	"KMSV2/snmp"
	"encoding/json"
	"fmt"
	_ "fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
	"unicode"
)

type SnmpDevice struct {
	IP             string
	Communtiy      string
	SnmpVersion    string
	MIBName        string
	NormalTables   map[string]map[string]string
	SliceTables    map[string]map[string][]string
	UpdateTime     time.Duration
	IsConnected    bool
	NOfDisconnect  int
	normalTime     time.Duration
	TablesAndLines map[string]int
	canUpdate      bool
	Loaded         bool
	Accessed       bool
	Mutex          sync.RWMutex
	IsUpdating     bool
}

func NewSnmpDevice(ip string) SnmpDevice {
	device := SnmpDevice{
		IP:             ip,
		Communtiy:      "public",
		SnmpVersion:    "1",
		MIBName:        "NoMIB",
		NormalTables:   make(map[string]map[string]string),
		SliceTables:    make(map[string]map[string][]string),
		normalTime:     5 * time.Second,
		UpdateTime:     10 * time.Second,
		TablesAndLines: make(map[string]int),
		canUpdate:      false,
	}
	device.NormalTables["NoTable"] = make(map[string]string)
	device.NormalTables["System"] = make(map[string]string)

	return device
}

func (device *SnmpDevice) Update() {
	if !snmp.IsConnected(device.IP) {
		mymysql.IpLog(device.IP, "{\"status\":\"Not Connected\"}")
		return
	}
	if device.IsUpdating {
		return
	}
	device.IsUpdating = true

	defer func() {
		if a := recover(); a != nil {
			log.Println("Panic: ", a)
		}
	}()

	device.Mutex.RLock()
	device.Accessed = true

	for key := range device.NormalTables {

		if key == "System" {
			device.Mutex.RLock()
			device.NormalTables["System"] = device.SystemTable()
			device.Mutex.RUnlock()
		} else {

			device.Mutex.RLock()
			defer device.Mutex.RUnlock()
			device.NormalTables[key] = device.NormalTable(key)

			if len(device.NormalTables[key]) == 0 {
				device.NOfDisconnect++

				device.IsUpdating = false
				return
			}

		}
	}
	iplog, _ := json.Marshal(device.NormalTables)
	mymysql.IpLog(device.IP, string(iplog))
	device.Mutex.RUnlock()
	if !device.Loaded && len(device.NormalTables) > 0 {
		device.NormalTables["System"] = device.SystemTable()
		go func(device *SnmpDevice) { device.SliceTables["Interfaces"] = device.InterfacesTable() }(device)
	}

	b := device.Loaded

	device.Loaded = true

	if b {
		go func(snmpDevice *SnmpDevice) {

			for key := range snmpDevice.SliceTables {
				if key != "Interfaces" {
					device.Mutex.RLock()
					snmpDevice.SliceTables[key] = snmpDevice.SliceTable(key)
					device.Mutex.RUnlock()
				}
			}

		}(device)
	}
	device.IsUpdating = false
	//	device.Mutex.RUnlock()
	/*device.Data[0] = device.NormalTables
	device.Data[1] = device.SliceTables
		fmt.Printf("\n\n%v", device.Data[0])
	fmt.Printf("\n\n%v", device.Data[1])*/
	//}

	//ticker := time.NewTicker(device.UpdateTime)
	//fmt.Println("update time =", device.UpdateTime, "IP = ", device.IP)
	//go func() {
	//for {
	//
	//	select {
	//
	//	case   <-ticker.C:
	//
	//		if device.canUpdate == false{
	//			device.canUpdate = true
	//			//device.UpdateTime = 20
	//			//continue
	//		}
	//		fmt.Println("update time =", device.UpdateTime)
	//	//	ticker = time.NewTicker(1000 * time.Millisecond)
	//
	//		/*if(device.NOfDisconnect > 20){
	//			device.UpdateTime = 500 * time.Second;
	//		}else if(device.NOfDisconnect > 100){
	//			device.UpdateTime = 5500 * time.Second;
	//		}*/
	//		helpers.Dump(device.IP)
	//		helpers.Dump(device.UpdateTime)
	//
	//		for key := range device.NormalTables{
	//		//	fmt.Println("Getting ", key, device.UpdateTime, device.IP)
	//			device.NormalTables[key] = device.NormalTable(key)
	//			tempKey = key
	//			if len(device.NormalTables[key]) == 0{
	//				device.NOfDisconnect++
	//				break
	//			}else{
	//			//	device.NOfDisconnect = 0
	//			//	device.UpdateTime = device.normalTime
	//			}
	//			//fmt.Println("Getting ", key, len(device.NormalTables[key]))
	//
	//		}
	//		//fmt.Println(len(device.NormalTables[tempKey]))
	//		if len(device.NormalTables[tempKey]) != 0 {
	//			for key := range device.SliceTables {
	//				device.SliceTables[key] = device.SliceTable(key)
	//			}
	//			device.Data[0] = device.NormalTables
	//			device.Data[1] = device.SliceTables
	//		//	fmt.Printf("%v", device.Data)
	//		}
	//	}
	//}
	//}()

}

func (device *SnmpDevice) SliceTable(MIBTable string) map[string][]string {
	defer func() {

		if a := recover(); a != nil {
			fmt.Println("Panic with ", a)
			log.Println("Panic: ", a)
		}
	}()
	device.Mutex.RLock()
	command := "snmpwalk -v " + device.SnmpVersion + " -c " + device.Communtiy + " " + device.IP + " " + device.MIBName + "::" + MIBTable
	output, _ := helpers.RunCommand(command)

	oids := strings.Split(output, "\n")
	name := ""
	value := ""
	data := make(map[string][]string, 40)
	lines := 10000
	i := 1
	if nlines, ok := device.TablesAndLines[MIBTable]; ok {
		lines = nlines
	}
	for _, line := range oids {
		if strings.TrimSpace(line) == "End of MIB" || strings.TrimSpace(line) == "" {
			break
		}
		if i > lines {
			break
		}
		i++

		if strings.Index(line, ".") > 0 && strings.Index(line, "::") > 0 && strings.Index(line, "=") > 0 {

			name = strings.TrimSpace(strings.Split(strings.Split(line, ".")[0], "::")[1])
			if !strings.HasSuffix(name, "ConnTime") {
				temp := strings.Split(line, " = ")
				if len(temp) > 1 {
					temp = strings.Split(temp[1], " ")
					if len(temp) > 1 {
						value = strings.TrimSpace(temp[1])
					}
				}

			} else {
				value = strings.TrimSpace(strings.Split(strings.Split(line, " = ")[1], ") ")[1])
			}

			ii := 0
			for unicode.IsUpper(rune(name[ii])) == false {
				ii++
			}
		}

		if _, ok := data[removeTableName(name, MIBTable)]; !ok {
			data[removeTableName(name, MIBTable)] = []string{value}
		} else {
			data[removeTableName(name, MIBTable)] = append(data[removeTableName(name, MIBTable)], value)
		}
	}

	device.Mutex.RUnlock()
	return data
}

func (device *SnmpDevice) NormalTable(MIBTable string) map[string]string {
	defer func() {

		if a := recover(); a != nil {
			fmt.Println("Panic with ", a)
			log.Println("Panic: ", a)
		}
	}()
	command := "snmpwalk -v " + device.SnmpVersion + " -c " + device.Communtiy + " " + device.IP + " " + device.MIBName + "::" + MIBTable

	output, _ := helpers.RunCommand(command)

	oids := strings.Split(output, "\n")
	name := ""
	value := ""
	data := make(map[string]string, 40)
	for _, line := range oids {
		if strings.TrimSpace(line) == "End of MIB" || strings.TrimSpace(line) == "" {
			break
		}

		name = strings.TrimSpace(strings.Split(strings.Split(line, ".")[0], "::")[1])
		if !strings.HasSuffix(name, "ConnTime") {
			value = strings.TrimSpace(strings.Split(strings.Split(line, " = ")[1], " ")[1])
		} else {
			value = strings.TrimSpace(strings.Split(strings.Split(line, " = ")[1], ") ")[1])
		}

		data[removeTableName(name, MIBTable)] = value
	}
	return data
}

func (device *SnmpDevice) SystemTable() map[string]string {
	defer func() {

		if a := recover(); a != nil {
			fmt.Println("Panic with ", a)
			log.Println("Panic: ", a)
		}
	}()
	MIBTable := "sys"
	command := "snmpwalk -v " + device.SnmpVersion + " -c " + device.Communtiy + " " + device.IP + " .1.3.6.1.2.1.1"
	output, _ := exec.Command("cmd", "/C", command).Output()

	oids := strings.Split(string(output), "\n")
	name := ""
	value := ""
	data := make(map[string]string, 40)
	for _, line := range oids {
		if strings.TrimSpace(line) == "End of MIB" || strings.TrimSpace(line) == "" {
			break
		}

		name = strings.TrimSpace(strings.Split(strings.Split(line, ".")[0], "::")[1])
		if !strings.HasSuffix(name, "ConnTime") {
			value = strings.TrimSpace(strings.Split(strings.Split(line, " = ")[1], " ")[1])
		} else {
			value = strings.TrimSpace(strings.Split(strings.Split(line, " = ")[1], ") ")[1])
		}

		data[removeTableName(name, MIBTable)] = value
	}
	return data
}
func (device *SnmpDevice) InterfacesTable() map[string][]string {
	defer func() {

		if a := recover(); a != nil {
			fmt.Println("Panic with ", a)
			log.Println("Panic: ", a)
		}
	}()

	MIBTable := "interfaces"
	command := "snmpwalk -v " + device.SnmpVersion + " -c " + device.Communtiy + " " + device.IP + " .1.3.6.1.2.1.2"
	output, _ := exec.Command("cmd", "/C", command).Output()

	oids := strings.Split(string(output), "\n")
	name := ""
	value := ""
	data := make(map[string][]string, 40)
	lines := 10000
	i := 1
	if nlines, ok := device.TablesAndLines[MIBTable]; ok {
		lines = nlines
	}
	for _, line := range oids {
		if strings.TrimSpace(line) == "End of MIB" || strings.TrimSpace(line) == "" {
			break
		}
		if i > lines {
			break
		}
		i++

		if strings.Index(line, ".") > 0 && strings.Index(line, "::") > 0 && strings.Index(line, "=") > 0 {

			name = strings.TrimSpace(strings.Split(strings.Split(line, ".")[0], "::")[1])
			if !strings.HasSuffix(name, "ConnTime") {
				temp := strings.Split(line, " = ")
				if len(temp) > 0 {
					temp = strings.Split(temp[1], " ")
					if len(temp) > 0 {
						value = strings.TrimSpace(temp[1])
					}
				}

			} else {
				value = strings.TrimSpace(strings.Split(strings.Split(line, " = ")[1], ") ")[1])
			}

			ii := 0
			for unicode.IsUpper(rune(name[ii])) == false {
				ii++
			}
		}

		if _, ok := data[removeTableName(name, MIBTable)]; !ok {
			data[removeTableName(name, MIBTable)] = []string{value}
			//	println("First value")
			//	helpers.Dump(data[name])
			//helpers.Dump(value)
		} else {
			data[removeTableName(name, MIBTable)] = append(data[removeTableName(name, MIBTable)], value)
		}
	}
	//device.Mutex.RUnlock()
	return data
}

func (device *SnmpDevice) UpdateUnkownIP() {
	if device.IsUpdating {
		return
	}
	device.IsUpdating = true

	defer func() {

		if a := recover(); a != nil {
			fmt.Println("Panic with ", a)
			log.Println("Panic: ", a)
			//main()
		}
	}()

	device.Mutex.RLock()
	device.Accessed = true

	device.Mutex.RLock()

	device.NormalTables["System"] = device.SystemTable()

	go func(device *SnmpDevice) { device.NormalTables["NoTable"] = device.NoTable() }(device)

	device.Mutex.RUnlock()

	if !device.Loaded && len(device.NormalTables) > 0 {

		go func(device *SnmpDevice) { device.SliceTables["Interfaces"] = device.InterfacesTable() }(device)
	}

	device.Loaded = true

	device.IsUpdating = false

}

func (device *SnmpDevice) NoTable() map[string]string {
	defer func() {

		if a := recover(); a != nil {
			fmt.Println("Panic with ", a)
			log.Println("Panic: ", a)
		}
	}()
	command := "snmpwalk -v " + device.SnmpVersion + " -c " + device.Communtiy + " " + device.IP + " .1.3.6.1.4.1"
	println(command)
	output, _ := exec.Command("cmd", "/C", command).Output()

	oids := strings.Split(string(output), "\n")
	name := ""
	value := ""
	data := make(map[string]string, 40)
	for _, line := range oids {
		if strings.TrimSpace(line) == "End of MIB" || strings.TrimSpace(line) == "" {
			break
		}

		name = strings.TrimSpace(strings.Split(line, " = ")[0])
		value = strings.TrimSpace(strings.Split(strings.Split(line, " = ")[1], " ")[1])

		data[name] = value
	}
	return data
}

/*
func (device *SnmpDevice) InterfacesTable() map[string]string {

	command := "snmpwalk -v " + device.SnmpVersion + " -c " + device.Communtiy + " " + device.IP + " .1.3.6.1.2.1.2"
	output, _ := exec.Command("cmd", "/C", command).Output()

	oids := strings.Split(string(output), "\n")
	name := ""
	value := ""
	data := make(map[string]string, 40)
	for _, line := range oids {
		if strings.TrimSpace(line) == "End of MIB" || strings.TrimSpace(line) == "" {
			break;
		}

		name = strings.TrimSpace(strings.Split(strings.Split(line, ".")[0], "::")[1])
		if !strings.HasSuffix(name, "ConnTime") {
			value = strings.TrimSpace(strings.Split(strings.Split(line, " = ")[1], " ")[1])
		} else {
			value = strings.TrimSpace(strings.Split(strings.Split(line, " = ")[1], ") ")[1])
		}


		data[removeTableName(name, MIBTable)] = value
	}
	return data
}
*/

func removeTable(tableName string) string {
	return tableName[:strings.Index(tableName, "Table")]
}

func removeTableName(oidName, tableName string) string {
	if strings.Index(tableName, "Table") > 0 {
		tableName = tableName[:strings.Index(tableName, "Table")]
	}
	if strings.Index(oidName, tableName) >= 0 {
		oidName = oidName[len(tableName):]
	} else {
		ii := 0
		for unicode.IsUpper(rune(oidName[ii])) == false {
			ii++
		}
		oidName = oidName[ii:]
	}
	return oidName
}
func getTableName(tableName string) string {
	if strings.Index(tableName, "Table") > 0 {
		tableName = removeTable(tableName)
	}
	ii := 0
	for unicode.IsUpper(rune(tableName[ii])) == false {
		ii++
	}
	return tableName[ii:]
}
