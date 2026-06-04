package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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
	Type     string
	Use      bool
	Mode     string
	Device   string
	Sentence string
}

func (d Device) NormalizedMode() string {
	mode := d.Mode
	if mode == "" {
		if d.Use {
			mode = "ready"
		} else {
			mode = "disabled"
		}
	}

	switch mode {
	case "ready", "simulate", "disabled":
		return mode
	default:
		return "disabled"
	}
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

// Backup describes acquisition persistence policy.
type Backup struct {
	Raw           bool   `toml:"raw"`
	Processed     bool   `toml:"processed"`
	RawPath       string `toml:"raw_path"`
	ProcessedPath string `toml:"processed_path"`
}

// Config is the Go representation of toml file
type Config struct {
	Mission Mission
	Global  struct {
		Debug bool
		Echo  bool
		Log   string
	}
	Backup  Backup `toml:"backup"`
	Devices map[string]Device
	Serials map[string]SerialPort
	UDP     map[string]UDP
	Export  Export
	Acq     struct {
		File string
	} `toml:"acq"`
}

// Load returns a Config struct from the content of toml configFile.
func Load(configFile string) (Config, error) {
	cfg := Config{}
	if _, err := toml.DecodeFile(configFile, &cfg); err != nil {
		return Config{}, fmt.Errorf("load config %q: %w", configFile, err)
	}
	cfg.normalize()
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

func (c *Config) normalize() {
	if !c.Backup.Raw && !c.Backup.Processed && strings.TrimSpace(c.Acq.File) != "" {
		c.Backup.Raw = true
	}
}

// RawBackupEnabled reports whether raw acquisition persistence is enabled.
func (c Config) RawBackupEnabled() bool {
	return c.Backup.Raw || strings.TrimSpace(c.Acq.File) != ""
}

// RawBackupPath returns the raw SQLite path for the current mission.
func (c Config) RawBackupPath(configFile string) string {
	if strings.TrimSpace(c.Acq.File) != "" {
		return c.Acq.File
	}
	if !c.RawBackupEnabled() {
		return ""
	}
	return backupPath(configFile, c.Backup.RawPath, backupBaseName(c.Mission.Name)+"-raw.sqlite")
}

// ProcessedBackupPath returns the processed SQLite path for the current mission.
func (c Config) ProcessedBackupPath(configFile string) string {
	if !c.Backup.Processed {
		return ""
	}
	return backupPath(configFile, c.Backup.ProcessedPath, backupBaseName(c.Mission.Name)+"-data.sqlite")
}

func backupPath(configFile string, configuredPath string, defaultName string) string {
	configuredPath = strings.TrimSpace(configuredPath)
	if configuredPath != "" {
		path := configuredPath
		if !filepath.IsAbs(path) {
			path = filepath.Join(configDir(configFile), path)
		}
		path = filepath.Clean(path)
		if backupPathIsDir(configuredPath, path) {
			return filepath.Join(path, defaultName)
		}
		return path
	}
	return filepath.Join(configDir(configFile), defaultName)
}

func backupPathIsDir(configuredPath string, resolvedPath string) bool {
	if strings.HasSuffix(configuredPath, "/") || strings.HasSuffix(configuredPath, "\\") {
		return true
	}
	if filepath.Ext(configuredPath) == "" {
		return true
	}
	info, err := os.Stat(resolvedPath)
	return err == nil && info.IsDir()
}

func configDir(configFile string) string {
	dir := filepath.Dir(configFile)
	if dir == "" {
		return "."
	}
	return dir
}

func backupBaseName(missionName string) string {
	name := strings.TrimSpace(missionName)
	if name == "" {
		return "geo-acq"
	}

	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	)
	name = replacer.Replace(name)
	name = strings.Join(strings.Fields(name), "-")
	name = strings.Trim(name, ".- ")
	if name == "" {
		return "geo-acq"
	}
	return name
}
