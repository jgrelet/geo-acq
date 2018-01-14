package main

import (
	"fmt"

	"github.com/jgrelet/geo-acq/config"
	"github.com/jgrelet/geo-acq/devices"
)

// main entry
func main() {

	// simul.GpsChan = make(chan interface{})
	// simul.EchoSounderChan = make(chan interface{})
	devices.SerialGetInfo()
	var cfg config.Config
	cfg.GetConfig("windows.toml")

	gps := devices.New("gps", cfg)
	err := gps.Connect()
	defer gps.Disconnect()
	if err != nil {
		fmt.Println(err)
	}
	for {
		sentence, err := gps.Read()
		if err != nil {
			break
		}
		fmt.Println(sentence)
	}

}
