package main

import (
	"strings"
	"testing"
)

func TestUpdateDeviceModeInTOMLReplacesModeInDeviceSection(t *testing.T) {
	raw := `[mission]
name = "dev"

[devices.gps]
type = "nmea"
mode = "ready"

[devices.echosounder]
type           = "nmea"
mode           = "disabled"
device         = "serial"
`

	got, err := updateDeviceModeInTOML(raw, "echosounder", "simulate")
	if err != nil {
		t.Fatalf("updateDeviceModeInTOML() error = %v", err)
	}

	if !strings.Contains(got, `[devices.echosounder]`+"\n"+`type           = "nmea"`+"\n"+`mode           = "simulate"`) {
		t.Fatalf("updated TOML did not replace echosounder mode:\n%s", got)
	}
	if !strings.Contains(got, `[devices.gps]`+"\n"+`type = "nmea"`+"\n"+`mode = "ready"`) {
		t.Fatalf("updated TOML changed the gps section:\n%s", got)
	}
}

func TestUpdateDeviceModeInTOMLInsertsMissingMode(t *testing.T) {
	raw := `[devices.echosounder]
type = "nmea"
device = "serial"

[udp]
`

	got, err := updateDeviceModeInTOML(raw, "echosounder", "simulate")
	if err != nil {
		t.Fatalf("updateDeviceModeInTOML() error = %v", err)
	}

	if !strings.Contains(got, `[devices.echosounder]`+"\n"+`type = "nmea"`+"\n"+`device = "serial"`+"\n"+`	mode           = "simulate"`+"\n\n"+`[udp]`) {
		t.Fatalf("updated TOML did not insert mode before next section:\n%s", got)
	}
}
