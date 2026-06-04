package storage

import (
	"database/sql"
	"errors"
	"fmt"
)

func closeSQLiteDB(db *sql.DB) error {
	if db == nil {
		return nil
	}

	_, checkpointErr := db.Exec(`PRAGMA wal_checkpoint(TRUNCATE);`)
	closeErr := db.Close()

	if checkpointErr != nil {
		checkpointErr = fmt.Errorf("checkpoint sqlite WAL: %w", checkpointErr)
	}
	if closeErr != nil {
		closeErr = fmt.Errorf("close sqlite database: %w", closeErr)
	}
	return errors.Join(checkpointErr, closeErr)
}
