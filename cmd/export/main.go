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
		databasePath = defaultExportDatabasePath(cfg, *configPath)
	}
	if databasePath == "" {
		log.Fatal("export.database must be set or a backup database must be enabled")
	}

	mode := cfg.Export.Mode
	if mode == "" {
		mode = exporter.ModeSlowestDevice
	}

	interval, err := parseExportInterval(cfg.Export.Interval)
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

	storeKind, err := storage.DetectStoreKind(databasePath)
	if err != nil {
		log.Fatal(err)
	}

	selection := storage.SessionSelection{
		MissionName: cfg.Export.Mission,
		SessionID:   cfg.Export.SessionID,
	}

	switch storeKind {
	case storage.StoreKindRaw:
		session, frames, deviceNames, err := storage.LoadFramesForExport(databasePath, selection)
		if err != nil {
			log.Fatal(err)
		}
		rows, err := exporter.BuildRows(frames, deviceNames, mode, interval)
		if err != nil {
			log.Fatal(err)
		}
		if err := exporter.WriteTSV(file, session, deviceNames, rows); err != nil {
			log.Fatal(err)
		}
	case storage.StoreKindProcessed:
		session, samples, deviceNames, columns, err := storage.LoadProcessedForExport(databasePath, selection)
		if err != nil {
			log.Fatal(err)
		}
		rows, err := exporter.BuildStructuredRows(samples, deviceNames, columns, mode, interval)
		if err != nil {
			log.Fatal(err)
		}
		if err := exporter.WriteStructuredTSV(file, session, columns, rows); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unsupported store kind %q", storeKind)
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

func defaultExportDatabasePath(cfg config.Config, configPath string) string {
	if cfg.Backup.Processed {
		processedPath := cfg.ProcessedBackupPath(configPath)
		if processedPath != "" {
			if _, err := os.Stat(processedPath); err == nil {
				return processedPath
			}
		}
	}
	if cfg.RawBackupEnabled() {
		return cfg.RawBackupPath(configPath)
	}
	return ""
}
