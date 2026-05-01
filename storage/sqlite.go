package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jgrelet/geo-acq/config"
	"github.com/jgrelet/geo-acq/exporter"
	_ "modernc.org/sqlite"
)

// RawFrame stores one received NMEA sentence with its acquisition metadata.
type RawFrame struct {
	ReceivedAt time.Time
	DeviceName string
	Transport  string
	Payload    string
}

// SQLiteStore persists raw acquisition frames in append-only form.
type SQLiteStore struct {
	db        *sql.DB
	missionID int64
	sessionID int64
	insertRaw *sql.Stmt
}

// SessionSelection selects which acquisition session to export.
type SessionSelection struct {
	MissionName string
	SessionID   int64
}

// OpenSQLite opens or creates the configured SQLite database and registers the mission/session.
func OpenSQLite(path string, mission config.Mission, configFile string) (*SQLiteStore, error) {
	if path == "" {
		return nil, fmt.Errorf("acquisition database path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)

	store := &SQLiteStore{db: db}
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.configurePragmas(); err != nil {
		db.Close()
		return nil, err
	}

	missionID, err := store.upsertMission(mission)
	if err != nil {
		db.Close()
		return nil, err
	}
	store.missionID = missionID

	sessionID, err := store.createSession(configFile)
	if err != nil {
		db.Close()
		return nil, err
	}
	store.sessionID = sessionID

	stmt, err := db.Prepare(`
		INSERT INTO raw_frames (
			session_id,
			mission_id,
			received_at_utc,
			device_name,
			transport,
			payload
		) VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("prepare raw_frames insert: %w", err)
	}
	store.insertRaw = stmt

	return store, nil
}

// Close releases database resources.
func (s *SQLiteStore) Close() error {
	if s == nil {
		return nil
	}
	if s.insertRaw != nil {
		_ = s.insertRaw.Close()
	}
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SaveRawFrame persists one raw frame in the append-only table.
func (s *SQLiteStore) SaveRawFrame(frame RawFrame) error {
	if s == nil || s.insertRaw == nil {
		return fmt.Errorf("sqlite store is not initialized")
	}
	_, err := s.insertRaw.Exec(
		s.sessionID,
		s.missionID,
		frame.ReceivedAt.UTC().Format(time.RFC3339Nano),
		frame.DeviceName,
		frame.Transport,
		frame.Payload,
	)
	if err != nil {
		return fmt.Errorf("insert raw frame: %w", err)
	}
	return nil
}

func (s *SQLiteStore) configurePragmas() error {
	pragmas := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA synchronous=NORMAL;`,
		`PRAGMA foreign_keys=ON;`,
	}
	for _, pragma := range pragmas {
		if _, err := s.db.Exec(pragma); err != nil {
			return fmt.Errorf("apply sqlite pragma %q: %w", pragma, err)
		}
	}
	return nil
}

func (s *SQLiteStore) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS missions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	pi TEXT NOT NULL,
	organization TEXT NOT NULL,
	created_at_utc TEXT NOT NULL,
	updated_at_utc TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS acquisition_sessions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	mission_id INTEGER NOT NULL,
	config_file TEXT NOT NULL,
	started_at_utc TEXT NOT NULL,
	FOREIGN KEY (mission_id) REFERENCES missions(id)
);

CREATE TABLE IF NOT EXISTS raw_frames (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id INTEGER NOT NULL,
	mission_id INTEGER NOT NULL,
	received_at_utc TEXT NOT NULL,
	device_name TEXT NOT NULL,
	transport TEXT NOT NULL,
	payload TEXT NOT NULL,
	FOREIGN KEY (session_id) REFERENCES acquisition_sessions(id),
	FOREIGN KEY (mission_id) REFERENCES missions(id)
);

CREATE INDEX IF NOT EXISTS idx_raw_frames_session_time
	ON raw_frames(session_id, received_at_utc);

CREATE INDEX IF NOT EXISTS idx_raw_frames_device_time
	ON raw_frames(device_name, received_at_utc);
`

	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("initialize sqlite schema: %w", err)
	}
	return nil
}

func (s *SQLiteStore) upsertMission(mission config.Mission) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	name := mission.Name
	if name == "" {
		name = "default-mission"
	}
	pi := mission.PI
	if pi == "" {
		pi = "unknown"
	}
	org := mission.Organization
	if org == "" {
		org = "unknown"
	}

	if _, err := s.db.Exec(`
		INSERT INTO missions (name, pi, organization, created_at_utc, updated_at_utc)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			pi = excluded.pi,
			organization = excluded.organization,
			updated_at_utc = excluded.updated_at_utc
	`, name, pi, org, now, now); err != nil {
		return 0, fmt.Errorf("upsert mission: %w", err)
	}

	var missionID int64
	if err := s.db.QueryRow(`SELECT id FROM missions WHERE name = ?`, name).Scan(&missionID); err != nil {
		return 0, fmt.Errorf("select mission id: %w", err)
	}
	return missionID, nil
}

func (s *SQLiteStore) createSession(configFile string) (int64, error) {
	result, err := s.db.Exec(`
		INSERT INTO acquisition_sessions (mission_id, config_file, started_at_utc)
		VALUES (?, ?, ?)
	`, s.missionID, configFile, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("create acquisition session: %w", err)
	}
	sessionID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get acquisition session id: %w", err)
	}
	return sessionID, nil
}

// LoadFramesForExport loads one acquisition session and its raw frames from SQLite.
func LoadFramesForExport(path string, selection SessionSelection) (exporter.Session, []exporter.Frame, []string, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return exporter.Session{}, nil, nil, fmt.Errorf("open sqlite database: %w", err)
	}
	defer db.Close()

	session, err := resolveExportSession(db, selection)
	if err != nil {
		return exporter.Session{}, nil, nil, err
	}

	rows, err := db.Query(`
		SELECT rf.received_at_utc, rf.device_name, rf.payload
		FROM raw_frames rf
		WHERE rf.session_id = ?
		ORDER BY rf.received_at_utc, rf.id
	`, session.ID)
	if err != nil {
		return exporter.Session{}, nil, nil, fmt.Errorf("query raw frames: %w", err)
	}
	defer rows.Close()

	frames := []exporter.Frame{}
	deviceSet := map[string]struct{}{}
	for rows.Next() {
		var receivedAtRaw string
		var deviceName string
		var payload string
		if err := rows.Scan(&receivedAtRaw, &deviceName, &payload); err != nil {
			return exporter.Session{}, nil, nil, fmt.Errorf("scan raw frame: %w", err)
		}
		receivedAt, err := time.Parse(time.RFC3339Nano, receivedAtRaw)
		if err != nil {
			return exporter.Session{}, nil, nil, fmt.Errorf("parse frame timestamp: %w", err)
		}
		frames = append(frames, exporter.Frame{
			ReceivedAt: receivedAt,
			DeviceName: deviceName,
			Payload:    payload,
		})
		deviceSet[deviceName] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return exporter.Session{}, nil, nil, fmt.Errorf("iterate raw frames: %w", err)
	}

	deviceNames := make([]string, 0, len(deviceSet))
	for name := range deviceSet {
		deviceNames = append(deviceNames, name)
	}
	sort.Strings(deviceNames)

	return session, frames, deviceNames, nil
}

func resolveExportSession(db *sql.DB, selection SessionSelection) (exporter.Session, error) {
	rowQuery := `
		SELECT s.id, m.name, s.config_file, s.started_at_utc
		FROM acquisition_sessions s
		JOIN missions m ON m.id = s.mission_id
	`
	args := []interface{}{}

	switch {
	case selection.SessionID > 0:
		rowQuery += ` WHERE s.id = ?`
		args = append(args, selection.SessionID)
	case selection.MissionName != "":
		rowQuery += ` WHERE m.name = ? ORDER BY s.started_at_utc DESC LIMIT 1`
		args = append(args, selection.MissionName)
	default:
		rowQuery += ` ORDER BY s.started_at_utc DESC LIMIT 1`
	}

	var session exporter.Session
	var startedAtRaw string
	if err := db.QueryRow(rowQuery, args...).Scan(&session.ID, &session.Mission, &session.ConfigFile, &startedAtRaw); err != nil {
		if err == sql.ErrNoRows {
			return exporter.Session{}, fmt.Errorf("no acquisition session found")
		}
		return exporter.Session{}, fmt.Errorf("select acquisition session: %w", err)
	}

	startedAt, err := time.Parse(time.RFC3339Nano, startedAtRaw)
	if err != nil {
		return exporter.Session{}, fmt.Errorf("parse session timestamp: %w", err)
	}
	session.StartedAt = startedAt
	return session, nil
}
