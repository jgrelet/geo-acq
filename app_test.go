package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jgrelet/geo-acq/config"
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

func TestNextDeviceModeCyclesThroughReadySimulateDisabled(t *testing.T) {
	cases := []struct {
		mode string
		want string
	}{
		{mode: "ready", want: "simulate"},
		{mode: "simulate", want: "disabled"},
		{mode: "disabled", want: "ready"},
		{mode: "", want: "ready"},
	}

	for _, tc := range cases {
		if got := nextDeviceMode(tc.mode); got != tc.want {
			t.Fatalf("nextDeviceMode(%q) = %q, want %q", tc.mode, got, tc.want)
		}
	}
}

func TestToggleDeviceSimulationDefersConfigWriteUntilPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	raw := `[mission]
name = "dev"

[devices.gps]
type = "nmea"
mode = "ready"
device = "udp"
sentence = "GGA"

[udp.gps]
port = "10110"
`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write test config: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load test config: %v", err)
	}

	app := NewApp()
	app.configPath = path
	app.configRaw = raw
	app.cfg = cfg
	app.deviceOrder = sortedDeviceNames(cfg)
	app.deviceStates = buildDeviceStates(cfg, app.deviceOrder)

	state, err := app.ToggleDeviceSimulation("gps")
	if err != nil {
		t.Fatalf("ToggleDeviceSimulation() error = %v", err)
	}
	if got := state.Devices[0].Mode; got != "simulate" {
		t.Fatalf("device mode after toggle = %q, want simulate", got)
	}

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config after toggle: %v", err)
	}
	if string(onDisk) != raw {
		t.Fatalf("config was written before persist:\n%s", onDisk)
	}

	if err := app.persistPendingDeviceModes(); err != nil {
		t.Fatalf("persistPendingDeviceModes() error = %v", err)
	}
	onDisk, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config after persist: %v", err)
	}
	if !strings.Contains(string(onDisk), `mode           = "simulate"`) {
		t.Fatalf("config was not updated on persist:\n%s", onDisk)
	}
}
