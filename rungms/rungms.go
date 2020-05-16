package main

import (
	"KMSV2/helpers"
	"fmt"
	"net/http"
	"time"
)

func main() {

	ticker := time.NewTicker(10 * time.Second)

	for _ = range ticker.C {
		fmt.Println("tock")
		_, err := http.Get("http://localhost:9090/device/ip")
		if err != nil {
			fmt.Printf("%s", err)
			helpers.RunCommand("go run api\api.go")
		}
	}
}
