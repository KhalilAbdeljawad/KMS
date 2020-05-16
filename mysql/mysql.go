package mymysql

import (
	"KMSV2/helpers"
	"KMSV2/snmp"
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type IP struct {
	DeviceName   string `db:"name"`
	IP           string `db:"ip"`
	Manfuacturer string `db:"manufacturer"`
}

type Vlan struct {
	Address     string         `db:"vlan_addres"`
	Chanbw      sql.NullInt64  `db:"vlan_chanbw"`
	Coordinates sql.NullString `db:"vlan_coordinates"`
	DataID      int            `db:"vlan_data_Id"`
	GatewayIP   string         `db:"vlan_gateway_ip"`
	ID          string         `db:"vlan_id"`
	Location    string         `db:"vlan_location"`
	MgmtID      int            `db:"vlan_mgmt_Id"`
	SiteName    string         `db:"vlan_site_name"`
	WifiEqID    string         `db:"vlan_wifi_eq_id"`
	Ssids       []VlanSsid     `json:"ssids,omitempty" db:"ssids,omitempty"`
	Created_at  string         `db:"created_at"`
	Flag        int            `db:"flag"`
	UserId      int            `db:"user_id"`
}

type VlanSsid struct {
	Description  string `db:"description"`
	DeviceName   string `db:"device_name"`
	IP           string `db:"ip"`
	Manufacturer string `db:"manufacturer"`
	Model        string `db:"model"`
	Password     string `db:"password"`
	SsidActive   int    `db:"ssid_active"`
	SsidID       int    `db:"ssid_id"`
	SsidName     string `db:"ssid_name"`
	SsidVlanID   int    `db:"ssid_vlan_id"`
	Username     string `db:"username"`
	IsConnected  bool
	Created_at   string `db:"created_at"`
}

var Vlans = []Vlan{}
var Allips []IP
var IPs_Usernames map[string]IP
var Usernames_Ips map[string]IP
var DBConnect *sqlx.DB

func Connect() {
	if DBConnect != nil {
		return
	}

	db, err := sqlx.Connect("mysql", "user:password@(ip:3306)/db")

	if err != nil {
		log.Printf("Error in DB connect.. %v\n", err.Error())
	}
	DBConnect = db
}

func AllIPs() {
	// this Pings the database trying to connect, panics on error
	// use sqlx.Open() for sql.Open() semantics

	ips := []IP{}
	DBConnect.Select(&ips, "SELECT wifi_ip_ip AS ip, wifi_ip_device_name AS name, TRIM(SUBSTRING(`wifi_eq_manufacturer`, 1, INSTR(`wifi_eq_manufacturer`, ' ')-1)) AS manufacturer FROM  `wifi_ip_all`,`wifi_equipment` WHERE wifi_eq_id = wifi_ip_eq AND wifi_ip_ip !='' AND wifi_ip_device_name !='' AND wifi_ip_ip is not null  ORDER BY wifi_ip_ip ASC")

	ssidips := []IP{}
	DBConnect.Select(&ssidips, "SELECT ip, ssid_name as `name`, manufacturer FROM `vlan_ssid` where ip != ''")

	Allips = append(Allips, ssidips...)
	Allips = append(Allips, ips...)

	SelectToMap(Allips)
}

func SelectToMap(result []IP) {
	IPs_Usernames = make(map[string]IP)
	Usernames_Ips = make(map[string]IP)
	for _, v := range result {
		if strings.TrimSpace(v.IP) == "" {
			continue
		}
		Usernames_Ips[v.DeviceName] = v
		IPs_Usernames[v.IP] = v
	}

}

func SearchIPS(query string) []IP {
	ips := []IP{}

	DBConnect.Select(&ips, "SELECT wifi_ip_device_name, wifi_ip_ip FROM wifi_ip_all WHERE wifi_ip_eq in(1,3,4,5,6,7,8,9,10,11,12,13,14,16) AND (wifi_ip_ip like '%"+query+"%' or wifi_ip_device_name like  '%"+query+"%') AND wifi_ip_ip != '' AND wifi_ip_ip is not null  ORDER BY wifi_ip_ip ASC")
	return ips
}

func Search(query string) []IP {
	ips := []IP{}
	for _, value := range IPs_Usernames {
		if value.DeviceName == query || helpers.CaseInsenstiveContains(value.DeviceName, query) {
			ips = append(ips, value)
		}
		if value.IP == query || helpers.CaseInsenstiveContains(value.IP, query) {
			ips = append(ips, value)
		}
	}
	return ips
}

func GetName(ip string) string {
	return IPs_Usernames[ip].DeviceName
}

func GetIp(name string) string {
	return Usernames_Ips[name].IP
}

func GetSsids() []IP {
	ssidips := []IP{}
	Connect()
	DBConnect.Select(&ssidips, "SELECT ip, ssid_name as `name`, manufacturer FROM `vlan_ssid` where ip != ''")
	return ssidips
}

func GetVlans() []Vlan {
	defer func() {
		if a := recover(); a != nil {
			fmt.Println("Panic with ", a)
		}
	}()
	Connect()
	err := DBConnect.Select(&Vlans, "SELECT * FROM `vlan`;")

	fmt.Println("Vlan error:", err)
	for i, vlan := range Vlans {

		ssids := []VlanSsid{}
		err := DBConnect.Select(&ssids, "SELECT * FROM `vlan_ssid` where ip != '' AND ssid_vlan_id = "+vlan.ID)
		if err != nil {
			log.Println(err)
		}
		//log.Println(ssids)
		for j := range ssids {
			ssids[j].Username = ""
			ssids[j].Password = ""
			ssids[j].IsConnected = snmp.IsConnected(ssids[j].IP)
		}
		Vlans[i].Ssids = ssids

	}
	return Vlans

}

type WatchIp struct {
	Id         int    `db:"id"`
	Name       string `db:"name"`
	Ip         string `db:"ip"`
	LastStatus string `db:"last_status"`
}

func GetWatchedIps() []WatchIp {
	var watchedIps []WatchIp
	Connect()
	DBConnect.Select(&watchedIps, "SELECT * FROM `watchedips`;")
	fmt.Println(watchedIps)
	return watchedIps
}

func IpLog(ip, log string) {

	if strings.TrimSpace(ip) == "" {
		return
	}
	defer func() {
		if a := recover(); a != nil {
			fmt.Println("Panic with ", a, "ip = ", ip, "log data = ", log)
			fmt.Println("Panic: ", a)
			//main()
		}
	}()
	Connect()
	stmt, err := DBConnect.Prepare("insert into ipslog (ip, logdata, creation_date, creation_time) values(?,?, CURDATE(), CURTIME())")
	if err != nil {
		fmt.Println(err)
	}
	stmt.Exec(ip, log)
	/*if err != nil {
		fmt.Println(err)
	}
	log.Printf(res)
	*/
}

type IpLogStruct struct {
	Id           int    `db:"id"`
	IP           string `db:"ip,omitempty"`
	Logs         string `db:"logdata,omitempty"`
	CreationDate string `db:"creation_date,omitempty"`
	CreationTime string `db:"creation_time,omitempty"`
}

func GetIpLog(ip string) []IpLogStruct {

	iplogs := []IpLogStruct{}
	Connect()

	DBConnect.Select(&iplogs, "SELECT * FROM `ipslog` where ip = ? ORDER BY id desc limit 3", ip)

	return iplogs
}
