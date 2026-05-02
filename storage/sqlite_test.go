package storage

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/jgrelet/geo-acq/config"
	_ "modernc.org/sqlite"
)

func TestOpenSQLiteAndSaveRawFrame(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "acq.sqlite")

	store, err := OpenSQLite(dbPath, config.Mission{
		Name:         "mission-1",
		PI:           "pi",
		Organization: "org",
	}, "test.toml")
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer store.Close()

	err = store.SaveRawFrame(RawFrame{
		ReceivedAt:   time.Now(),
		DeviceName:   "gps",
		Transport:    "udp",
		Payload:      "$GPGGA,....",
		SentenceType: "GPGGA",
		DecodedJSON:  `{"sentence_type":"GPGGA"}`,
	})
	if err != nil {
		t.Fatalf("save raw frame: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db for verification: %v", err)
	}
	defer db.Close()

	var sentenceType string
	var decodedJSON string
	if err := db.QueryRow(`SELECT sentence_type, decoded_json FROM raw_frames LIMIT 1`).Scan(&sentenceType, &decodedJSON); err != nil {
		t.Fatalf("query saved frame: %v", err)
	}
	if sentenceType != "GPGGA" {
		t.Fatalf("sentence_type = %q, want GPGGA", sentenceType)
	}
	if decodedJSON == "" {
		t.Fatal("decoded_json is empty")
	}
}
