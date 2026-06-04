package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jgrelet/geo-acq/config"
	_ "modernc.org/sqlite"
)

// ProcessedFrame stores one decoded NMEA sentence with acquisition metadata.
type ProcessedFrame struct {
	ReceivedAt   time.Time
	DeviceName   string
	Transport    string
	SentenceType string
	DecodedJSON  string
}

// ProcessedSQLiteStore persists selected decoded sentences in queryable tables.
type ProcessedSQLiteStore struct {
	db        *sql.DB
	missionID int64
	sessionID int64
	insertGGA *sql.Stmt
	insertRMC *sql.Stmt
	insertVTG *sql.Stmt
	insertDBT *sql.Stmt
}

type ggaPayload struct {
	TimeUTC          string   `json:"time_utc"`
	Latitude         float64  `json:"latitude"`
	Longitude        float64  `json:"longitude"`
	QualityIndicator int      `json:"quality_indicator"`
	SatellitesUsed   int      `json:"satellites_used"`
	HDOP             float64  `json:"hdop"`
	AltitudeM        float64  `json:"altitude_m"`
	GeoidSeparationM *float64 `json:"geoid_separation_m"`
	DGPSAgeS         *float64 `json:"dgps_age_s"`
	DGPSStationID    *uint8   `json:"dgps_station_id"`
}

type rmcPayload struct {
	DateTimeUTC          string   `json:"datetime_utc"`
	IsValid              bool     `json:"is_valid"`
	Latitude             float64  `json:"latitude"`
	Longitude            float64  `json:"longitude"`
	SpeedKnots           float64  `json:"speed_knots"`
	CourseOverDeg        *float64 `json:"course_over_deg"`
	MagneticVariationDeg *float64 `json:"magnetic_variation_deg"`
	PositioningMode      string   `json:"positioning_mode"`
}

type vtgPayload struct {
	CourseOverDeg   float64 `json:"course_over_deg"`
	SpeedKnots      float64 `json:"speed_knots"`
	SpeedKmh        float64 `json:"speed_kmh"`
	PositioningMode string  `json:"positioning_mode"`
}

type dbtPayload struct {
	DepthFeet    float64 `json:"depth_feet"`
	DepthMeters  float64 `json:"depth_meters"`
	DepthFathoms float64 `json:"depth_fathoms"`
}

// OpenProcessedSQLite opens or creates the processed SQLite database.
func OpenProcessedSQLite(path string, mission config.Mission, configFile string) (*ProcessedSQLiteStore, error) {
	if path == "" {
		return nil, fmt.Errorf("processed database path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open processed sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)

	store := &ProcessedSQLiteStore{db: db}
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

	if err := store.prepareStatements(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// Close releases database resources.
func (s *ProcessedSQLiteStore) Close() error {
	if s == nil {
		return nil
	}
	for _, stmt := range []*sql.Stmt{s.insertGGA, s.insertRMC, s.insertVTG, s.insertDBT} {
		if stmt != nil {
			_ = stmt.Close()
		}
	}
	if s.db != nil {
		return closeSQLiteDB(s.db)
	}
	return nil
}

// SaveProcessedFrame persists supported decoded sentences in typed tables.
func (s *ProcessedSQLiteStore) SaveProcessedFrame(frame ProcessedFrame) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("processed sqlite store is not initialized")
	}
	if strings.TrimSpace(frame.DecodedJSON) == "" {
		return nil
	}

	switch {
	case strings.HasSuffix(frame.SentenceType, "GGA"):
		var payload ggaPayload
		if err := json.Unmarshal([]byte(frame.DecodedJSON), &payload); err != nil {
			return fmt.Errorf("decode processed GGA json: %w", err)
		}
		_, err := s.insertGGA.Exec(
			s.sessionID,
			s.missionID,
			frame.ReceivedAt.UTC().Format(time.RFC3339Nano),
			frame.DeviceName,
			frame.Transport,
			payload.TimeUTC,
			payload.Latitude,
			payload.Longitude,
			payload.QualityIndicator,
			payload.SatellitesUsed,
			payload.HDOP,
			payload.AltitudeM,
			payload.GeoidSeparationM,
			payload.DGPSAgeS,
			payload.DGPSStationID,
		)
		if err != nil {
			return fmt.Errorf("insert processed GGA frame: %w", err)
		}
	case strings.HasSuffix(frame.SentenceType, "RMC"):
		var payload rmcPayload
		if err := json.Unmarshal([]byte(frame.DecodedJSON), &payload); err != nil {
			return fmt.Errorf("decode processed RMC json: %w", err)
		}
		_, err := s.insertRMC.Exec(
			s.sessionID,
			s.missionID,
			frame.ReceivedAt.UTC().Format(time.RFC3339Nano),
			frame.DeviceName,
			frame.Transport,
			payload.DateTimeUTC,
			payload.IsValid,
			payload.Latitude,
			payload.Longitude,
			payload.SpeedKnots,
			payload.CourseOverDeg,
			payload.MagneticVariationDeg,
			payload.PositioningMode,
		)
		if err != nil {
			return fmt.Errorf("insert processed RMC frame: %w", err)
		}
	case strings.HasSuffix(frame.SentenceType, "VTG"):
		var payload vtgPayload
		if err := json.Unmarshal([]byte(frame.DecodedJSON), &payload); err != nil {
			return fmt.Errorf("decode processed VTG json: %w", err)
		}
		_, err := s.insertVTG.Exec(
			s.sessionID,
			s.missionID,
			frame.ReceivedAt.UTC().Format(time.RFC3339Nano),
			frame.DeviceName,
			frame.Transport,
			payload.CourseOverDeg,
			payload.SpeedKnots,
			payload.SpeedKmh,
			payload.PositioningMode,
		)
		if err != nil {
			return fmt.Errorf("insert processed VTG frame: %w", err)
		}
	case strings.HasSuffix(frame.SentenceType, "DBT"):
		var payload dbtPayload
		if err := json.Unmarshal([]byte(frame.DecodedJSON), &payload); err != nil {
			return fmt.Errorf("decode processed DBT json: %w", err)
		}
		_, err := s.insertDBT.Exec(
			s.sessionID,
			s.missionID,
			frame.ReceivedAt.UTC().Format(time.RFC3339Nano),
			frame.DeviceName,
			frame.Transport,
			payload.DepthFeet,
			payload.DepthMeters,
			payload.DepthFathoms,
		)
		if err != nil {
			return fmt.Errorf("insert processed DBT frame: %w", err)
		}
	}

	return nil
}

func (s *ProcessedSQLiteStore) prepareStatements() error {
	var err error
	s.insertGGA, err = s.db.Prepare(`
		INSERT INTO gga_records (
			session_id, mission_id, received_at_utc, device_name, transport,
			time_utc, latitude, longitude, quality_indicator, satellites_used,
			hdop, altitude_m, geoid_separation_m, dgps_age_s, dgps_station_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare gga_records insert: %w", err)
	}
	s.insertRMC, err = s.db.Prepare(`
		INSERT INTO rmc_records (
			session_id, mission_id, received_at_utc, device_name, transport,
			datetime_utc, is_valid, latitude, longitude, speed_knots,
			course_over_deg, magnetic_variation_deg, positioning_mode
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare rmc_records insert: %w", err)
	}
	s.insertVTG, err = s.db.Prepare(`
		INSERT INTO vtg_records (
			session_id, mission_id, received_at_utc, device_name, transport,
			course_over_deg, speed_knots, speed_kmh, positioning_mode
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare vtg_records insert: %w", err)
	}
	s.insertDBT, err = s.db.Prepare(`
		INSERT INTO dbt_records (
			session_id, mission_id, received_at_utc, device_name, transport,
			depth_feet, depth_meters, depth_fathoms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare dbt_records insert: %w", err)
	}
	return nil
}

func (s *ProcessedSQLiteStore) configurePragmas() error {
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

func (s *ProcessedSQLiteStore) initSchema() error {
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

CREATE TABLE IF NOT EXISTS gga_records (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id INTEGER NOT NULL,
	mission_id INTEGER NOT NULL,
	received_at_utc TEXT NOT NULL,
	device_name TEXT NOT NULL,
	transport TEXT NOT NULL,
	time_utc TEXT NOT NULL DEFAULT '',
	latitude REAL NOT NULL,
	longitude REAL NOT NULL,
	quality_indicator INTEGER NOT NULL,
	satellites_used INTEGER NOT NULL,
	hdop REAL NOT NULL,
	altitude_m REAL NOT NULL,
	geoid_separation_m REAL,
	dgps_age_s REAL,
	dgps_station_id INTEGER,
	FOREIGN KEY (session_id) REFERENCES acquisition_sessions(id),
	FOREIGN KEY (mission_id) REFERENCES missions(id)
);

CREATE TABLE IF NOT EXISTS rmc_records (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id INTEGER NOT NULL,
	mission_id INTEGER NOT NULL,
	received_at_utc TEXT NOT NULL,
	device_name TEXT NOT NULL,
	transport TEXT NOT NULL,
	datetime_utc TEXT NOT NULL DEFAULT '',
	is_valid INTEGER NOT NULL,
	latitude REAL NOT NULL,
	longitude REAL NOT NULL,
	speed_knots REAL NOT NULL,
	course_over_deg REAL,
	magnetic_variation_deg REAL,
	positioning_mode TEXT NOT NULL DEFAULT '',
	FOREIGN KEY (session_id) REFERENCES acquisition_sessions(id),
	FOREIGN KEY (mission_id) REFERENCES missions(id)
);

CREATE TABLE IF NOT EXISTS vtg_records (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id INTEGER NOT NULL,
	mission_id INTEGER NOT NULL,
	received_at_utc TEXT NOT NULL,
	device_name TEXT NOT NULL,
	transport TEXT NOT NULL,
	course_over_deg REAL NOT NULL,
	speed_knots REAL NOT NULL,
	speed_kmh REAL NOT NULL,
	positioning_mode TEXT NOT NULL DEFAULT '',
	FOREIGN KEY (session_id) REFERENCES acquisition_sessions(id),
	FOREIGN KEY (mission_id) REFERENCES missions(id)
);

CREATE TABLE IF NOT EXISTS dbt_records (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id INTEGER NOT NULL,
	mission_id INTEGER NOT NULL,
	received_at_utc TEXT NOT NULL,
	device_name TEXT NOT NULL,
	transport TEXT NOT NULL,
	depth_feet REAL NOT NULL,
	depth_meters REAL NOT NULL,
	depth_fathoms REAL NOT NULL,
	FOREIGN KEY (session_id) REFERENCES acquisition_sessions(id),
	FOREIGN KEY (mission_id) REFERENCES missions(id)
);

CREATE INDEX IF NOT EXISTS idx_gga_records_session_time
	ON gga_records(session_id, received_at_utc);
CREATE INDEX IF NOT EXISTS idx_rmc_records_session_time
	ON rmc_records(session_id, received_at_utc);
CREATE INDEX IF NOT EXISTS idx_vtg_records_session_time
	ON vtg_records(session_id, received_at_utc);
CREATE INDEX IF NOT EXISTS idx_dbt_records_session_time
	ON dbt_records(session_id, received_at_utc);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("initialize processed sqlite schema: %w", err)
	}
	return nil
}

func (s *ProcessedSQLiteStore) upsertMission(mission config.Mission) (int64, error) {
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
		return 0, fmt.Errorf("upsert processed mission: %w", err)
	}

	var missionID int64
	if err := s.db.QueryRow(`SELECT id FROM missions WHERE name = ?`, name).Scan(&missionID); err != nil {
		return 0, fmt.Errorf("select processed mission id: %w", err)
	}
	return missionID, nil
}

func (s *ProcessedSQLiteStore) createSession(configFile string) (int64, error) {
	result, err := s.db.Exec(`
		INSERT INTO acquisition_sessions (mission_id, config_file, started_at_utc)
		VALUES (?, ?, ?)
	`, s.missionID, configFile, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("create processed acquisition session: %w", err)
	}
	sessionID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get processed acquisition session id: %w", err)
	}
	return sessionID, nil
}
