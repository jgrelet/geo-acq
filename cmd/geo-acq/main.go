package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jgrelet/geo-acq/config"
	"github.com/jgrelet/geo-acq/devices"
)

// main entry
func main() {
	configPath := flag.String("config", config.DefaultFile(), "configuration TOML file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	gps := devices.New("gps", cfg)
	if err := gps.Connect(); err != nil {
		log.Fatal(err)
	}
	defer gps.Disconnect()

	for {
		select {
		case msg, ok := <-gps.Data:
			if ok {
				fmt.Println(msg)
			} else {
				fmt.Println("Exit ...")
				os.Exit(0)
			}
		}
	}
}
