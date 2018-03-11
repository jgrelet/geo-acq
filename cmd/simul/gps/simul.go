package main

import (
	"fmt"
	"log"

	"github.com/jgrelet/geo-acq/config"
	"github.com/jgrelet/geo-acq/devices"
	"github.com/jgrelet/geo-acq/simul"
	"github.com/jgrelet/geo-acq/util"
)

// main entry
func main() {

	gps := devices.New("gps", config.New("windows.toml"))
	if err := gps.Connect(); err != nil {
		log.Fatal(err)
	}
	defer gps.Disconnect()

	// new GPS task every second, with SOG=10knt and COG=90°
	nmea := simul.NewGps(1, 10, 90)
	for {
		sentence := <-nmea
		fmt.Println("Send: " + sentence)
		gps.Write(sentence + util.CR + util.LF)
	}
}
