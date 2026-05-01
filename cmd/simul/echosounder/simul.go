package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/jgrelet/geo-acq/config"
	"github.com/jgrelet/geo-acq/devices"
	"github.com/jgrelet/geo-acq/simul"
	"github.com/jgrelet/geo-acq/util"
)

// main entry
func main() {
	configPath := flag.String("config", config.DefaultFile(), "configuration TOML file")
	interval := flag.Duration("interval", 600*time.Millisecond, "sounder emission interval")
	depth := flag.Float64("depth", 12.0, "initial depth in meters")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	sounder := devices.New("echosounder", cfg)
	if err := sounder.Connect(); err != nil {
		log.Fatal(err)
	}
	defer sounder.Disconnect()

	dbt := simul.NewEchoSounder(*interval, *depth)
	for {
		sentence := <-dbt
		fmt.Println("Send: " + sentence)
		if err := sounder.Write(sentence + util.CR + util.LF); err != nil {
			log.Fatal(err)
		}
	}
}
