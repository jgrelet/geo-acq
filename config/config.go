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

type Device struct {
	Type   string
	Use    bool
	Device string
}

// Config is the Go representation of toml file
type Config struct {
	Global struct {
		Author string
		Debug  bool
		Echo   bool
		Log    string
	}
	Acq struct {
		File string
	}
	Devices map[string]Device
	Serials map[string]SerialPort
}

// GetConfig give the scontent of toml configFile
func (cfg *Config) GetConfig(configFile string) {

	//  read config file
	if _, err := toml.DecodeFile(configFile, &cfg); err != nil {
		log.Fatal(fmt.Sprintf("Error func GetConfig: file= %s -> %s\n", configFile, err))
	}
}
