package config

import (
	"fmt"
	"runtime"

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
	Host string
	Port string
}

// Device struct type = NMEA, Device serial or UDPs
type Device struct {
	Type   string
	Use    bool
	Device string
	Sentence string
}

// Mission describes the current acquisition campaign metadata.
type Mission struct {
	Name         string
	PI           string
	Organization string
}

// Export describes offline extraction parameters from a SQLite raw acquisition database.
type Export struct {
	Database  string `toml:"database"`
	Output    string `toml:"output"`
	Mode      string `toml:"mode"`
	Interval  string `toml:"interval"`
	Mission   string `toml:"mission"`
	SessionID int64  `toml:"session_id"`
}

// Config is the Go representation of toml file
type Config struct {
	Mission Mission
	Global  struct {
		Debug bool
		Echo  bool
		Log   string
	}
	Acq struct {
		File string
	}
	Devices map[string]Device
	Serials map[string]SerialPort
	UDP     map[string]UDP
	Export  Export
}

// Load returns a Config struct from the content of toml configFile.
func Load(configFile string) (Config, error) {
	cfg := Config{}
	if _, err := toml.DecodeFile(configFile, &cfg); err != nil {
		return Config{}, fmt.Errorf("load config %q: %w", configFile, err)
	}
	return cfg, nil
}

// DefaultFile returns the default configuration file for the current OS.
func DefaultFile() string {
	if runtime.GOOS == "windows" {
		return "windows.toml"
	}
	return "linux.toml"
}

// New preserves the historical API.
func New(configFile string) Config {
	cfg, err := Load(configFile)
	if err != nil {
		panic(fmt.Sprintf("Error func GetConfig: file= %s -> %s\n", configFile, err))
	}
	return cfg
}
