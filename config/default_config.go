package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultUserFile returns the per-user configuration file path.
func DefaultUserFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil || home == "" {
			if err != nil {
				return "", fmt.Errorf("locate user config directory: %w", err)
			}
			return "", fmt.Errorf("locate user config directory")
		}
		dir = filepath.Join(home, ".geo-acq")
	} else {
		dir = filepath.Join(dir, "geo-acq")
	}
	return filepath.Join(dir, DefaultFile()), nil
}

// DefaultContent returns a minimal editable TOML configuration for the current OS.
func DefaultContent() string {
	if DefaultFile() == "windows.toml" {
		return windowsDefaultContent
	}
	return linuxDefaultContent
}

const windowsDefaultContent = `[mission]
name           = "dev"
pi             = "J.Grelet"
organization   = "IRD"

[global]
debug          = false
echo           = true
log            = "geo-acq.log"

[backup]
raw            = true
processed      = true
raw_path       = ""
processed_path = ""

[devices]

	[devices.gps]
	type           = "nmea"
	mode           = "ready"
	device         = "serial"
	sentence       = "GGA,RMC"

	[devices.echosounder]
	type           = "nmea"
	mode           = "disabled"
	device         = "serial"
	sentence       = "DBT"

[serials]

	[serials.gps]
	port           = "COM3"
	baud           = 4800
	parity         = "none"
	databit        = 8
	stopbit        = 1

	[serials.echosounder]
	port           = "COM16"
	baud           = 4800
	parity         = "none"
	databit        = 8
	stopbit        = 1

[udp]

	[udp.gps]
	host           = ""
	port           = "10183"

	[udp.echosounder]
	host           = ""
	port           = "10184"
`

const linuxDefaultContent = `[mission]
name           = "test"
pi             = "jgrelet"
organization   = "IRD"

[global]
debug          = false
echo           = true
log            = "geo-acq.log"

[backup]
raw            = true
processed      = false
raw_path       = ""
processed_path = ""

[devices]

	[devices.gps]
	type           = "nmea"
	mode           = "ready"
	device         = "serial"
	sentence       = "GGA,RMC"

	[devices.echosounder]
	type           = "nmea"
	mode           = "disabled"
	device         = "serial"
	sentence       = "DBT"

[serials]

	[serials.gps]
	port           = "/dev/ttyUSB0"
	baud           = 4800
	parity         = "none"
	databit        = 8
	stopbit        = 1

	[serials.echosounder]
	port           = "/dev/ttyUSB1"
	baud           = 4800
	parity         = "none"
	databit        = 8
	stopbit        = 1

[udp]

	[udp.gps]
	host           = ""
	port           = "10183"

	[udp.echosounder]
	host           = ""
	port           = "10184"
`
