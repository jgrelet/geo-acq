package storage

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/jgrelet/geo-acq/exporter"
	_ "modernc.org/sqlite"
)

// LoadProcessedRecordsForExport loads one acquisition session and its processed records from SQLite.
func LoadProcessedRecordsForExport(path string, selection SessionSelection) (exporter.Session, []exporter.ProcessedRecord, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return exporter.Session{}, nil, fmt.Errorf("open processed sqlite database: %w", err)
	}
	defer db.Close()

	session, err := resolveExportSession(db, selection)
	if err != nil {
		return exporter.Session{}, nil, err
	}

	loaders := []func(*sql.DB, int64) ([]exporter.ProcessedRecord, error){
		loadGGARecordsForExport,
		loadRMCRecordsForExport,
		loadVTGRecordsForExport,
		loadDBTRecordsForExport,
	}

	records := []exporter.ProcessedRecord{}
	for _, loader := range loaders {
		loaded, err := loader(db, session.ID)
		if err != nil {
			return exporter.Session{}, nil, err
		}
		records = append(records, loaded...)
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].ReceivedAt.Equal(records[j].ReceivedAt) {
			if records[i].DeviceName == records[j].DeviceName {
				return records[i].SentenceType < records[j].SentenceType
			}
			return records[i].DeviceName < records[j].DeviceName
		}
		return records[i].ReceivedAt.Before(records[j].ReceivedAt)
	})

	return session, records, nil
}

func loadGGARecordsForExport(db *sql.DB, sessionID int64) ([]exporter.ProcessedRecord, error) {
	rows, err := db.Query(`
		SELECT received_at_utc, device_name, transport, latitude, longitude, quality_indicator,
		       satellites_used, hdop, altitude_m, geoid_separation_m, dgps_age_s, dgps_station_id
		FROM gga_records
		WHERE session_id = ?
		ORDER BY received_at_utc, id
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query gga records: %w", err)
	}
	defer rows.Close()

	records := []exporter.ProcessedRecord{}
	for rows.Next() {
		var receivedAtRaw, deviceName, transport string
		var latitude, longitude, hdop, altitude float64
		var quality, satellites int
		var geoidSep, dgpsAge sql.NullFloat64
		var stationID sql.NullInt64

		if err := rows.Scan(&receivedAtRaw, &deviceName, &transport, &latitude, &longitude, &quality, &satellites, &hdop, &altitude, &geoidSep, &dgpsAge, &stationID); err != nil {
			return nil, fmt.Errorf("scan gga record: %w", err)
		}
		receivedAt, err := time.Parse(time.RFC3339Nano, receivedAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse gga timestamp: %w", err)
		}

		values := map[string]string{
			"latitude":        strconv.FormatFloat(latitude, 'f', -1, 64),
			"longitude":       strconv.FormatFloat(longitude, 'f', -1, 64),
			"quality":         strconv.Itoa(quality),
			"satellites_used": strconv.Itoa(satellites),
			"hdop":            strconv.FormatFloat(hdop, 'f', -1, 64),
			"altitude_m":      strconv.FormatFloat(altitude, 'f', -1, 64),
		}
		if geoidSep.Valid {
			values["geoid_separation_m"] = strconv.FormatFloat(geoidSep.Float64, 'f', -1, 64)
		}
		if dgpsAge.Valid {
			values["dgps_age_s"] = strconv.FormatFloat(dgpsAge.Float64, 'f', -1, 64)
		}
		if stationID.Valid {
			values["dgps_station_id"] = strconv.FormatInt(stationID.Int64, 10)
		}
		records = append(records, exporter.ProcessedRecord{
			ReceivedAt:   receivedAt,
			DeviceName:   deviceName,
			Transport:    transport,
			SentenceType: "GGA",
			Values:       values,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gga records: %w", err)
	}
	return records, nil
}

func loadRMCRecordsForExport(db *sql.DB, sessionID int64) ([]exporter.ProcessedRecord, error) {
	rows, err := db.Query(`
		SELECT received_at_utc, device_name, transport, datetime_utc, is_valid, latitude, longitude,
		       speed_knots, course_over_deg, magnetic_variation_deg, positioning_mode
		FROM rmc_records
		WHERE session_id = ?
		ORDER BY received_at_utc, id
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query rmc records: %w", err)
	}
	defer rows.Close()

	records := []exporter.ProcessedRecord{}
	for rows.Next() {
		var receivedAtRaw, deviceName, transport, datetimeUTC, positioningMode string
		var isValid bool
		var latitude, longitude, speedKnots float64
		var courseOver, magVar sql.NullFloat64

		if err := rows.Scan(&receivedAtRaw, &deviceName, &transport, &datetimeUTC, &isValid, &latitude, &longitude, &speedKnots, &courseOver, &magVar, &positioningMode); err != nil {
			return nil, fmt.Errorf("scan rmc record: %w", err)
		}
		receivedAt, err := time.Parse(time.RFC3339Nano, receivedAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse rmc timestamp: %w", err)
		}

		values := map[string]string{
			"datetime_utc": datetimeUTC,
			"is_valid":     strconv.FormatBool(isValid),
			"latitude":     strconv.FormatFloat(latitude, 'f', -1, 64),
			"longitude":    strconv.FormatFloat(longitude, 'f', -1, 64),
			"speed_knots":  strconv.FormatFloat(speedKnots, 'f', -1, 64),
		}
		if courseOver.Valid {
			values["course_over_deg"] = strconv.FormatFloat(courseOver.Float64, 'f', -1, 64)
		}
		if magVar.Valid {
			values["magnetic_variation_deg"] = strconv.FormatFloat(magVar.Float64, 'f', -1, 64)
		}
		if positioningMode != "" {
			values["positioning_mode"] = positioningMode
		}
		records = append(records, exporter.ProcessedRecord{
			ReceivedAt:   receivedAt,
			DeviceName:   deviceName,
			Transport:    transport,
			SentenceType: "RMC",
			Values:       values,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rmc records: %w", err)
	}
	return records, nil
}

func loadVTGRecordsForExport(db *sql.DB, sessionID int64) ([]exporter.ProcessedRecord, error) {
	rows, err := db.Query(`
		SELECT received_at_utc, device_name, transport, course_over_deg, speed_knots, speed_kmh, positioning_mode
		FROM vtg_records
		WHERE session_id = ?
		ORDER BY received_at_utc, id
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query vtg records: %w", err)
	}
	defer rows.Close()

	records := []exporter.ProcessedRecord{}
	for rows.Next() {
		var receivedAtRaw, deviceName, transport, positioningMode string
		var courseOver, speedKnots, speedKmh float64

		if err := rows.Scan(&receivedAtRaw, &deviceName, &transport, &courseOver, &speedKnots, &speedKmh, &positioningMode); err != nil {
			return nil, fmt.Errorf("scan vtg record: %w", err)
		}
		receivedAt, err := time.Parse(time.RFC3339Nano, receivedAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse vtg timestamp: %w", err)
		}

		values := map[string]string{
			"course_over_deg": strconv.FormatFloat(courseOver, 'f', -1, 64),
			"speed_knots":     strconv.FormatFloat(speedKnots, 'f', -1, 64),
			"speed_kmh":       strconv.FormatFloat(speedKmh, 'f', -1, 64),
		}
		if positioningMode != "" {
			values["positioning_mode"] = positioningMode
		}
		records = append(records, exporter.ProcessedRecord{
			ReceivedAt:   receivedAt,
			DeviceName:   deviceName,
			Transport:    transport,
			SentenceType: "VTG",
			Values:       values,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vtg records: %w", err)
	}
	return records, nil
}

func loadDBTRecordsForExport(db *sql.DB, sessionID int64) ([]exporter.ProcessedRecord, error) {
	rows, err := db.Query(`
		SELECT received_at_utc, device_name, transport, depth_feet, depth_meters, depth_fathoms
		FROM dbt_records
		WHERE session_id = ?
		ORDER BY received_at_utc, id
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query dbt records: %w", err)
	}
	defer rows.Close()

	records := []exporter.ProcessedRecord{}
	for rows.Next() {
		var receivedAtRaw, deviceName, transport string
		var depthFeet, depthMeters, depthFathoms float64

		if err := rows.Scan(&receivedAtRaw, &deviceName, &transport, &depthFeet, &depthMeters, &depthFathoms); err != nil {
			return nil, fmt.Errorf("scan dbt record: %w", err)
		}
		receivedAt, err := time.Parse(time.RFC3339Nano, receivedAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse dbt timestamp: %w", err)
		}

		values := map[string]string{
			"depth_feet":    strconv.FormatFloat(depthFeet, 'f', -1, 64),
			"depth_meters":  strconv.FormatFloat(depthMeters, 'f', -1, 64),
			"depth_fathoms": strconv.FormatFloat(depthFathoms, 'f', -1, 64),
		}
		records = append(records, exporter.ProcessedRecord{
			ReceivedAt:   receivedAt,
			DeviceName:   deviceName,
			Transport:    transport,
			SentenceType: "DBT",
			Values:       values,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dbt records: %w", err)
	}
	return records, nil
}
