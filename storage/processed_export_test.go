package storage

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jgrelet/geo-acq/config"
	"github.com/jgrelet/geo-acq/decoder"
)

func TestLoadProcessedForExport(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "processed.sqlite")

	store, err := OpenProcessedSQLite(dbPath, config.Mission{
		Name:         "mission-1",
		PI:           "pi",
		Organization: "org",
	}, "test.toml")
	if err != nil {
		t.Fatalf("open processed sqlite store: %v", err)
	}
	defer store.Close()

	decodedRMC, err := decoder.DecodeNMEA("$GPRMC,015540.000,A,4807.038,N,01131.000,E,0.0,0.0,040526,,,A*6C")
	if err != nil {
		t.Fatalf("decode rmc: %v", err)
	}
	if err := store.SaveProcessedFrame(ProcessedFrame{
		ReceivedAt:   time.Now().UTC(),
		DeviceName:   "gps",
		Transport:    "udp",
		SentenceType: decodedRMC.SentenceType,
		DecodedJSON:  decodedRMC.JSON,
	}); err != nil {
		t.Fatalf("save processed rmc frame: %v", err)
	}

	decodedDBT, err := decoder.DecodeNMEA("$GPDBT,108.34,f,33.02,M,18.06,F*35")
	if err != nil {
		t.Fatalf("decode dbt: %v", err)
	}
	if err := store.SaveProcessedFrame(ProcessedFrame{
		ReceivedAt:   time.Now().UTC().Add(time.Second),
		DeviceName:   "echosounder",
		Transport:    "udp",
		SentenceType: decodedDBT.SentenceType,
		DecodedJSON:  decodedDBT.JSON,
	}); err != nil {
		t.Fatalf("save processed dbt frame: %v", err)
	}

	session, samples, deviceNames, columns, err := LoadProcessedForExport(dbPath, SessionSelection{
		MissionName: "mission-1",
	})
	if err != nil {
		t.Fatalf("load processed for export: %v", err)
	}
	if session.Mission != "mission-1" {
		t.Fatalf("mission = %q, want mission-1", session.Mission)
	}
	if len(samples) != 2 {
		t.Fatalf("sample count = %d, want 2", len(samples))
	}
	if len(deviceNames) != 2 {
		t.Fatalf("device count = %d, want 2", len(deviceNames))
	}
	if len(columns) == 0 {
		t.Fatal("columns should not be empty")
	}
}
