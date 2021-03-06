package config

import (
	"fmt"
	"log"

	"github.com/BurntSushi/toml"
)

// SerialPort structure for a serial port
type SerialPort struct {
	Port    string
	Baud    int
	Databit int
	Stopbit int
	Parity  string
}

// UDP struct for ethernet port
type UDP struct {
	Port string
}

// Device struct type = NMEA, Device serial or UDPs
type Device struct {
	Type   string
	Use    bool
	Device string
}

// Config is the Go representation of toml file
type Config struct {
	Global struct {
		Mission string
		Author  string
		Debug   bool
		Echo    bool
		Log     string
	}
	Acq struct {
		File string
	}
	Devices map[string]Device
	Serials map[string]SerialPort
	UDP     map[string]UDP
}

// New  return a Config struct from the content of toml configFile
func New(configFile string) Config {

	cfg := Config{}
	//  read config file
	if _, err := toml.DecodeFile(configFile, &cfg); err != nil {
		log.Fatal(fmt.Sprintf("Error func GetConfig: file= %s -> %s\n", configFile, err))
	}
	//fmt.Printf("%v\n", cfg)
	return cfg
}
