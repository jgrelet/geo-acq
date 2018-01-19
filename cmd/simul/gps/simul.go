package main

import (
	"fmt"
	"log"

	"github.com/jgrelet/geo-acq/config"
	"github.com/jgrelet/geo-acq/devices"
	"github.com/jgrelet/geo-acq/simul"
)

// main entry
func main() {

	gps := devices.New("gps", config.New("windows.toml"))
	if err := gps.Connect(); err != nil {
		log.Fatal(err)
	}
	defer gps.Disconnect()
	nmea := simul.NewGps(2, 10, 90)
	for {
		fmt.Println(<-nmea)
	}
}
