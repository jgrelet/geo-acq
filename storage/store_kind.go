package storage

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const (
	StoreKindUnknown   = "unknown"
	StoreKindRaw       = "raw"
	StoreKindProcessed = "processed"
)

// DetectStoreKind inspects the SQLite schema and reports the likely acquisition store type.
func DetectStoreKind(path string) (string, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return StoreKindUnknown, fmt.Errorf("open sqlite database: %w", err)
	}
	defer db.Close()

	hasRaw, err := tableExists(db, "raw_frames")
	if err != nil {
		return StoreKindUnknown, err
	}
	if hasRaw {
		return StoreKindRaw, nil
	}

	hasProcessed, err := tableExists(db, "rmc_records")
	if err != nil {
		return StoreKindUnknown, err
	}
	if hasProcessed {
		return StoreKindProcessed, nil
	}

	return StoreKindUnknown, nil
}

func tableExists(db *sql.DB, tableName string) (bool, error) {
	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, tableName).Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect sqlite schema for %s: %w", tableName, err)
	}
	return true, nil
}
