package storage

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jgrelet/geo-acq/config"
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
		ReceivedAt: time.Now(),
		DeviceName: "gps",
		Transport:  "udp",
		Payload:    "$GPGGA,....",
	})
	if err != nil {
		t.Fatalf("save raw frame: %v", err)
	}
}
