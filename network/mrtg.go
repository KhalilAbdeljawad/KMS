package network

import (
	mymysql "KMSV2/mysql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var s sync.WaitGroup

func makeConfigFiles() {
	ssids := mymysql.GetSsids()
	i := 1
	for _, ssid := range ssids {
		if i > 4 {
			i = 1
			time.Sleep(3000)
		}
		i++

		s.Add(1)

		command := `C:\Strawberry\perl\bin\perl C:\mrtg-2.17.4\bin\cfgmaker public@` + ssid.IP + ` --global "WorkDir: C:\xampp\htdocs\mrtg\mrtg_` + strings.ReplaceAll(ssid.IP, ".", "_") + `" --output mrtgconfs\mrtg_` + strings.ReplaceAll(ssid.IP, ".", "_") + `.cfg`
		println(command)

		s.Done()

	}
}
func main() {

	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		for {
			select {
			case <-ticker.C:
				RunMrtg()

			}
		}
	}()

}

func RunMrtg() {
	files, err := ioutil.ReadDir("mrtgconfs")
	if err != nil {
		log.Fatal(err)
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dir)
	for _, f := range files {

		name := f.Name()
		if strings.Index(name, ".cfg") < 0 {
			continue
		}

		os.Mkdir(`C:\xampp\htdocs\mrtg\`+name[:strings.Index(name, ".")], 0777)

		command := `C:\Strawberry\perl\bin\perl C:\mrtg-2.17.4\bin\mrtg ` + dir + `\mrtgconfs\` + f.Name()

		exec.Command("cmd", "/C", command).Output()

	}
	fmt.Println("Mrtg finished")
}
