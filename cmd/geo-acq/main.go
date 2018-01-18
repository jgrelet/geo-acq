package main

import (
	"fmt"
	"log"

	"github.com/jgrelet/geo-acq/config"
	"github.com/jgrelet/geo-acq/devices"
)

// main entry
func main() {

	// simul.GpsChan = make(chan interface{})
	// simul.EchoSounderChan = make(chan interface{})

	gps := devices.New("gps", config.New("windows.toml"))
	if err := gps.Connect(); err != nil {
		log.Fatal(err)
	}
	defer gps.Disconnect()

	// main loop
	for {
		select {
		case msg := <-gps.Data:
			fmt.Println(msg)
		}
	}
}
