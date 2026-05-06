package storage

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/jgrelet/geo-acq/config"
	"github.com/jgrelet/geo-acq/decoder"
	_ "modernc.org/sqlite"
)

func TestOpenProcessedSQLiteAndSaveProcessedFrame(t *testing.T) {
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

	decoded, err := decoder.DecodeNMEA("$GPRMC,015540.000,A,4807.038,N,01131.000,E,0.0,0.0,040526,,,A*6C")
	if err != nil {
		t.Fatalf("decode rmc: %v", err)
	}

	err = store.SaveProcessedFrame(ProcessedFrame{
		ReceivedAt:   time.Now(),
		DeviceName:   "gps",
		Transport:    "udp",
		SentenceType: decoded.SentenceType,
		DecodedJSON:  decoded.JSON,
	})
	if err != nil {
		t.Fatalf("save processed frame: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db for verification: %v", err)
	}
	defer db.Close()

	var sentenceCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM rmc_records`).Scan(&sentenceCount); err != nil {
		t.Fatalf("query processed frame count: %v", err)
	}
	if sentenceCount != 1 {
		t.Fatalf("rmc record count = %d, want 1", sentenceCount)
	}

	var datetimeUTC string
	var isValid bool
	if err := db.QueryRow(`SELECT datetime_utc, is_valid FROM rmc_records LIMIT 1`).Scan(&datetimeUTC, &isValid); err != nil {
		t.Fatalf("query processed frame: %v", err)
	}
	if datetimeUTC == "" {
		t.Fatal("datetime_utc is empty")
	}
	if !isValid {
		t.Fatal("is_valid = false, want true")
	}
}

func TestProcessedStoreIgnoresUnsupportedSentence(t *testing.T) {
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

	err = store.SaveProcessedFrame(ProcessedFrame{
		ReceivedAt:   time.Now(),
		DeviceName:   "gps",
		Transport:    "udp",
		SentenceType: "GPZDA",
		DecodedJSON:  `{"sentence_type":"GPZDA","datetime_utc":"2002-07-04T20:15:30Z"}`,
	})
	if err != nil {
		t.Fatalf("save unsupported processed frame: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db for verification: %v", err)
	}
	defer db.Close()

	for _, table := range []string{"gga_records", "rmc_records", "vtg_records", "dbt_records"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("query %s count: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("%s count = %d, want 0", table, count)
		}
	}
}
