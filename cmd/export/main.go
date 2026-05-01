package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jgrelet/geo-acq/config"
	"github.com/jgrelet/geo-acq/exporter"
	"github.com/jgrelet/geo-acq/storage"
)

func main() {
	configPath := flag.String("config", "examples/export-slowest.toml", "export configuration TOML file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	databasePath := cfg.Export.Database
	if databasePath == "" {
		databasePath = cfg.Acq.File
	}
	if databasePath == "" {
		log.Fatal("export.database or acq.file must be set")
	}

	mode := cfg.Export.Mode
	if mode == "" {
		mode = exporter.ModeSlowestDevice
	}

	interval, err := parseExportInterval(cfg.Export.Interval)
	if err != nil {
		log.Fatal(err)
	}

	session, frames, deviceNames, err := storage.LoadFramesForExport(databasePath, storage.SessionSelection{
		MissionName: cfg.Export.Mission,
		SessionID:   cfg.Export.SessionID,
	})
	if err != nil {
		log.Fatal(err)
	}

	rows, err := exporter.BuildRows(frames, deviceNames, mode, interval)
	if err != nil {
		log.Fatal(err)
	}

	outputPath := cfg.Export.Output
	if outputPath == "" {
		outputPath = defaultExportPath(databasePath, mode)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil && filepath.Dir(outputPath) != "." {
		log.Fatal(err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	if err := exporter.WriteTSV(file, session, deviceNames, rows); err != nil {
		log.Fatal(err)
	}
}

func parseExportInterval(value string) (time.Duration, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	return time.ParseDuration(value)
}

func defaultExportPath(databasePath string, mode string) string {
	ext := filepath.Ext(databasePath)
	base := strings.TrimSuffix(databasePath, ext)
	if mode == "" {
		mode = exporter.ModeSlowestDevice
	}
	return base + "-" + mode + ".tsv"
}
