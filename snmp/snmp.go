package snmp

import (
	"KMS/snmp"
	"encoding/json"
	"fmt"
	"github.com/soniah/gosnmp"
	"log"
	"net"
	"strings"
	"time"
)

var Client = gosnmp.GoSNMP{
	Port:               161,
	Transport:          "udp",
	Community:          "public",
	Version:            gosnmp.Version1,
	Timeout:            time.Duration(3) * time.Second,
	Retries:            3,
	ExponentialTimeout: true,
	MaxOids:            gosnmp.MaxOids,
}

type PDU struct {
	Name      string `json:"name"`
	FullName  string `json:"fullName"`
	Oid       string `json:"oid"`
	ValueType string `json:"valueType"`
	Value     string `json:"value"`
}

func MapOidsWithNames(pdus []gosnmp.SnmpPDU, jsonText string) []PDU {
	Pdus := []PDU{}
	json.Unmarshal([]byte(jsonText), &Pdus)

	for _, pdu := range pdus {
		for i := 0; i < len(Pdus); i++ {
			if pdu.Name == Pdus[i].Oid {
				Pdus[i].FullName = Pdus[i].Name
				Pdus[i].Name = strings.Split(Pdus[i].Name, ".")[0]
				switch pdu.Type {
				case gosnmp.OctetString:
					b := pdu.Value.([]byte)
					Pdus[i].Value = string(net.HardwareAddr(b))
					fmt.Printf(":           :%x\n", string(net.HardwareAddr(b)))

				case gosnmp.IPAddress:
					Pdus[i].Value = fmt.Sprintf("%v", pdu.Value)

				default:
					Pdus[i].Value = fmt.Sprintf("%v", pdu.Value)
				}

			} else if strings.HasPrefix(pdu.Name, Pdus[i].Oid) {
				Pdus[i].Name = strings.Split(Pdus[i].Name, ".")[0]
				b, ok := pdu.Value.([]byte)
				if ok {
					Pdus[i].Value = string(b)
				} else {
					Pdus[i].Value = fmt.Sprintf("%v", pdu.Value)
				}

			}
		}
	}
	return Pdus

}

func Get() {

	gosnmp.Default.Target = "127.0.0.1"
	err := gosnmp.Default.Connect()
	if err != nil {
		log.Fatalf("Connect() err: %v", err)
	}
	defer gosnmp.Default.Conn.Close()

	oids := []string{".1.3.6.1.2.1.25.3.3.1"}

	result, err2 := gosnmp.Default.Get(oids) // Get() accepts up to gosnmp.MAX_OIDS
	if err2 != nil {
		log.Fatalf("Get() err: %v", err2)
	}

	for i, variable := range result.Variables {
		fmt.Printf("%d: oid: %s ", i, variable.Name)

		// the Value of each variable returned by Get() implements
		// interface{}. You could do a type switch...
		switch variable.Type {
		case gosnmp.OctetString:
			bytes := variable.Value.([]byte)
			fmt.Printf("string: %s\n", string(bytes))
		default:
			// ... or often you're just interested in numeric values.
			// ToBigInt() will return the Value as a BigInt, for plugging
			// into your calculations.
			fmt.Printf("number: %d\n", gosnmp.ToBigInt(variable.Value))
		}
	}

}

func IsConnected(ip string) bool {

	snmpClient := snmp.Client
	snmpClient.Target = ip
	snmpClient.Retries = 0
	//snmpClient.Timeout = 100
	err := snmpClient.Connect()
	if err != nil {
		return false
	}

	sysDsc, err := snmpClient.Get([]string{".1.3.6.1.2.1.1.1.0"})
	if err != nil {
		return false
	}

	if sysDsc.Variables == nil {
		return false
	}
	return true
}
