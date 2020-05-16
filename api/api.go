package main

import (
	"KMSV2/helpers"
	"KMSV2/network"
	"KMSV2/snmp"
	"KMSV2/snmpdevice"
	"KMSV2/ssh"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
)

var Devices map[string]snmpdevice.SnmpDevice
var locker sync.RWMutex
var Vlans []mymysql.Vlan

func showDevice(resp rest.ResponseWriter, req *rest.Request) {

	defer func() {
		if a := recover(); a != nil {
			fmt.Println("Panic with ", a)
			log.Println("Panic: ", a)
			main()
		}
	}()
	ip := req.PathParam("ip") // net.LookupIP

	fmt.Println(ip, helpers.ValidIP4(ip))
	if helpers.ValidIP4(ip) {

		locker.RLock()
		if device, ok := Devices[ip]; ok {

			if !Devices[ip].Accessed {
				if device.MIBName == "NoMIB" {
					device.UpdateUnkownIP()
				} else {
					device.Update()
				}
				Devices[ip] = device
			} else {
				if device.MIBName == "NoMIB" {
					go device.UpdateUnkownIP()
				} else {
					go device.Update()
				}
			}
			locker.RUnlock()

			resp.WriteJson(device)

		} else {
			Devices[ip] = snmpdevice.NewSnmpDevice(ip)
			device = Devices[ip]
			device.UpdateUnkownIP()
			Devices[ip] = device

			locker.RUnlock()
			resp.WriteJson(device)

		}

	} else {

	}
}
func main() {

	f, err := os.OpenFile("logfile.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error : %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

	defer func() {
		if a := recover(); a != nil {
			fmt.Println("Panic with ", a)
			log.Println("Panic: ", a)
			main()
		}
	}()

	go func() {
		mymysql.GetVlans()
	}()

	mutex := sync.Mutex{}
	Devices = make(map[string]snmpdevice.SnmpDevice, 5000)

	mymysql.Connect()

	mymysql.AllIPs()
	fmt.Printf("\nLen = %v\n", len(mymysql.Usernames_Ips))

	go func() {
		for _, device := range mymysql.Usernames_Ips {
			if device.Manfuacturer == "Ubiquiti" {
				mutex.Lock()
				Devices[device.IP] = snmpdevice.NewSnmpUbnt(device.IP)
				mutex.Unlock()
			} else if device.Manfuacturer == "Cambium" || device.Manfuacturer == "" {
				mutex.Lock()
				Devices[device.IP] = snmpdevice.NewSnmpCamb(device.IP)
				mutex.Unlock()
			}

		}
	}()

	api := rest.NewApi()
	api.Use(rest.DefaultDevStack...)
	api.Use(&rest.CorsMiddleware{
		RejectNonCorsRequests: false,
		OriginValidator: func(origin string, request *rest.Request) bool {
			return origin == "http://serverIPorDomain"
		},
		AllowedMethods: []string{"GET", "POST", "PUT"},
		AllowedHeaders: []string{
			"Accept", "Content-Type", "X-Custom-Header", "Origin"},
		AccessControlAllowCredentials: true,
		AccessControlMaxAge:           3600,
	})
	router, err := rest.MakeRouter(
		rest.Get("/device/#ip", showDevice),
		rest.Get("/search/#query", SearchIPsUsernames),
		rest.Get("/getname/#ip", getName),
		rest.Get("/getip/#name", getIp),
		rest.Get("/mrtg/#ip", getMrtg),
		rest.Get("/rxtx/#ip", getRXTX),
		rest.Get("/vlans", getVlans),
		rest.Get("/ipisconnected", ipIsConnected),
		rest.Post("/ubntconfig", getUbntConfig),
		rest.Post("/uploadscp", uploadSTringViaSSH),
		rest.Post("/configubnts", configUbnts),
		rest.Post("/cambconfig", getCambConfig),
		rest.Post("/makecambconfig", ConfigCamb),
		rest.Post("/cambsconfig", getCambsConfig),
		rest.Post("/makecambsconfig", ConfigCambs),
		rest.Get("/watchedips", getWatchedIps),
		rest.Get("/iplogs/#ip", getLog),
		rest.Get("/test", test),
	)
	go writeMrtg()
	go network.RunMrtg()
	if err != nil {
		log.Fatal(err)
	}

	go func() {

		goos := 0
		i := 1
		for _, key := range mymysql.Allips {
			mutex.Lock()
			device := Devices[key.IP]
			mutex.Unlock()

			goos++
			if goos >= 10 {
				time.Sleep(15 * time.Second)
				goos = 0
			}

			go func(device snmpdevice.SnmpDevice, delay int) {

				device.Mutex.RLock()
				device.Update()
				device.Mutex.RUnlock()
			}(device, i)

			i += 3
			if i > 3600 {
				i = 1
			}

		}
	}()

	///////////
	api.SetApp(router)

	log.Fatal(http.ListenAndServe(":9090", api.MakeHandler()))

}

func SearchIPsUsernames(resp rest.ResponseWriter, req *rest.Request) {

	query := req.PathParam("query")
	result := mymysql.Search(query)
	resp.WriteJson(result)
}

func getName(resp rest.ResponseWriter, req *rest.Request) {
	ip := req.PathParam("ip")
	result := mymysql.GetName(ip)
	resp.WriteJson(result)
}

func getIp(resp rest.ResponseWriter, req *rest.Request) {
	name := req.PathParam("name")
	result := mymysql.GetIp(name)
	resp.WriteJson(result)
}

func getMrtg(response rest.ResponseWriter, req *rest.Request) {
	ip := req.PathParam("ip")
	path := `C:\xampp\htdocs\mrtg\mrtg_` + strings.ReplaceAll(ip, ".", "_")

	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	htmls := []string{}
	for _, file := range files {
		if strings.Index(file.Name(), ".html") > 0 && file.Name() != "index.html" {

			htmls = append(htmls, file.Name())
		}
	}

	allMrtgHtml := ""
	for _, htmlFile := range htmls {
		filePath := path + "\\" + htmlFile

		htmlText, err := ioutil.ReadFile(filePath)
		if err != nil {
			panic(err)
		}

		html := string(htmlText)
		if err != nil {
			panic(err)
		}
		// show the HTML code as a string %s

		allMrtgHtml += html[:strings.Index(html, "<!-- Begin MRTG Block -->")] + "</body></html>"

	}
	allMrtgHtml = strings.ReplaceAll(allMrtgHtml, "img src=\"", "img src=\"http://localhost/mrtg/mrtg_"+strings.ReplaceAll(ip, ".", "_")+"/")
	ioutil.WriteFile("c:\\xampp\\htdocs\\mrtg\\mrtg_"+strings.ReplaceAll(ip, ".", "_")+"\\index.html", []byte(allMrtgHtml), 0777)

	response.WriteJson("http://localhost/mrtg/mrtg_" + strings.ReplaceAll(ip, ".", "_") + "/index.html")
}

func writeMrtg() {
	ticker := time.NewTicker(5 * time.Minute)
	//quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				network.RunMrtg()
			}
		}
	}()
}

func getRXTX(resp rest.ResponseWriter, req *rest.Request) {
	ip := req.PathParam("ip")
	log.Println("RXTX")
	snmpClient := snmp.Client
	snmpClient.Target = ip
	err := snmpClient.Connect()
	if err != nil {
		log.Fatalf("Connect() err: %v", err)
	}
	rx, err := snmpClient.Get([]string{".1.3.6.1.2.1.2.2.1.10.2", ".1.3.6.1.2.1.2.2.1.16.2"})
	if err != nil {
		log.Println(err)
		jso := `{"data":{"rec":0,"snd":0}}`
		resp.WriteString(jso)
		return
	}
	log.Println("Connected")

	rec := fmt.Sprintf("%v", rx.Variables[0].Value)
	snd := fmt.Sprintf("%v", rx.Variables[1].Value)
	log.Printf("%#v\n\n", rx)
	println(rx)

	jso := `{"data":{"rec":` + rec + `,"snd":` + snd + `}}`
	resp.WriteString(jso)

}

func ipIsConnected(resp rest.ResponseWriter, req *rest.Request) {
	ip := req.PathParam("ip")
	resp.WriteJson(snmp.IsConnected(ip))
}

func getVlans(resp rest.ResponseWriter, req *rest.Request) {
	resp.WriteJson(mymysql.Vlans)
}

type ConfigReq struct {
	IP, Username, Password, Text string
}

func getUbntConfig(resp rest.ResponseWriter, req *rest.Request) {

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Println("Errrror")
		panic(err)
	}
	var ucr ConfigReq
	err = json.Unmarshal(body, &ucr)
	if err != nil {
		panic(err)
	}

	resp.WriteJson(ssh.GetUbntConfig(ucr.IP, ucr.Username, ucr.Password))
}

func uploadSTringViaSSH(resp rest.ResponseWriter, req *rest.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Println("Errrror")
		panic(err)
	}

	var ucr ConfigReq
	err = json.Unmarshal(body, &ucr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("\n\n%v\n\n", ucr.Text)
	client, err := ssh.DialWithPasswd(ucr.IP+":22", ucr.Username, ucr.Password)
	if client.UploadText(ucr.Text, "system.cfg", "/tmp/") {
		resp.WriteJson(`{"result":"Conifguration done"}`)
	} else {
		resp.WriteString(`{"error":"File didn't uploaded to device"}`)
	}

}

func configUbnts(resp rest.ResponseWriter, req *rest.Request) {

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Println("Errrror")
		panic(err)
	}

	var ucr ConfigReq
	err = json.Unmarshal(body, &ucr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("\n\n%v\n\n", ucr)
	ips := strings.Split(ucr.IP, ",")

	for key := range ips {
		ips[key] = strings.TrimSpace(ips[key])
		client, er := ssh.DialWithPasswd(ips[key]+":22", ucr.Username, ucr.Password)
		if er != nil {
			fmt.Printf("\n%v\n", er)
		} else {
			client.ConfigByLine(ucr.Text, "system.cfg", "/tmp/")
		}
	}
	resp.WriteJson(ips)

}

func getCambConfig(resp rest.ResponseWriter, req *rest.Request) {

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic(err)
	}

	var ucr ConfigReq
	err = json.Unmarshal(body, &ucr)
	if err != nil {
		panic(err)
	}

	resp.WriteJson(ssh.GetCambConfig(ucr.IP, ucr.Username, ucr.Password))
}

func getCambsConfig(resp rest.ResponseWriter, req *rest.Request) {

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic(err)
	}
	var ucr ConfigReq
	err = json.Unmarshal(body, &ucr)
	if err != nil {
		panic(err)
	}

	resp.WriteJson(ssh.GetCambConfig(ucr.IP, ucr.Username, ucr.Password))
}

func ConfigCamb(resp rest.ResponseWriter, req *rest.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Println("Errrror")
		resp.WriteJson(err)
	}

	var ucr ConfigReq
	err = json.Unmarshal(body, &ucr)
	if err != nil {
		resp.WriteJson(err)
	}
	client, err := ssh.DialWithPasswd(ucr.IP+":22", ucr.Username, ucr.Password)
	configArr := strings.Split(ucr.Text, "\n")

	for _, line := range configArr {

		if strings.Contains(line, "Timeout") {
			fmt.Println(line)
			err = client.Cmd(`config set wirelessRadiusTimeout "15"`).Run()
			fmt.Println(err)
			client.Cmd("config save").Run()
			client.Cmd("config apply").Run()
		}

	}
	err = client.Cmd("config save").Run()
	fmt.Println(err)
	err = client.Cmd("config apply").Run()
	fmt.Println(err)
	time.Sleep(0 * time.Second)
	fmt.Println(err)
	println("Config should be done")
	resp.WriteString(`{"result":"Config Done"}`)
}

func ConfigCambs(resp rest.ResponseWriter, req *rest.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Println("Errrror")
		panic(err)
	}

	var ucr ConfigReq
	err = json.Unmarshal(body, &ucr)
	if err != nil {
		panic(err)
	}
	ips := strings.Split(ucr.IP, ",")

	for key := range ips {
		ips[key] = strings.TrimSpace(ips[key])
		client, er := ssh.DialWithPasswd(ips[key]+":22", ucr.Username, ucr.Password)
		if er != nil {
			fmt.Printf("\n%v\n", er)
		} else {
			client.Cmd(ucr.Text)
			client.Cmd("config save").Run()
			client.Cmd("config apply").Run()
			client.Cmd("reboot").Run()
		}
	}

	resp.WriteString(`{"data":"Config Done"}`)
}

func getWatchedIps(resp rest.ResponseWriter, req *rest.Request) {
	resp.WriteJson(mymysql.GetWatchedIps())
}

func getLog(resp rest.ResponseWriter, req *rest.Request) {
	ip := req.PathParam("ip")
	resp.WriteJson(mymysql.GetIpLog(ip))
}

func test(resp rest.ResponseWriter, req *rest.Request) {
	resp.WriteJson([]string{"1", "3"})
}
