package network

import (
	"fmt"
	"github.com/sparrc/go-ping"
)

func mainn() {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		panic(err)
	}
	pinger.Count = 3
	pinger.SetPrivileged(true)
	pinger.Run()                 // blocks until finished
	stats := pinger.Statistics() // get send/receive/rtt stats
	fmt.Printf("%v\n", stats)
}
