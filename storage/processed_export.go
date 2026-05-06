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

// LoadProcessedForExport loads one acquisition session and its processed records from SQLite.
func LoadProcessedForExport(path string, selection SessionSelection) (exporter.Session, []exporter.Sample, []string, []string, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return exporter.Session{}, nil, nil, nil, fmt.Errorf("open processed sqlite database: %w", err)
	}
	defer db.Close()

	session, err := resolveExportSession(db, selection)
	if err != nil {
		return exporter.Session{}, nil, nil, nil, err
	}

	samples := []exporter.Sample{}
	deviceSet := map[string]struct{}{}
	columnSet := map[string]struct{}{}

	loaders := []func(*sql.DB, int64) ([]exporter.Sample, error){
		loadGGAForExport,
		loadRMCForExport,
		loadVTGForExport,
		loadDBTForExport,
	}
	for _, loader := range loaders {
		loaded, err := loader(db, session.ID)
		if err != nil {
			return exporter.Session{}, nil, nil, nil, err
		}
		for _, sample := range loaded {
			samples = append(samples, sample)
			deviceSet[sample.DeviceName] = struct{}{}
			for column := range sample.Values {
				columnSet[column] = struct{}{}
			}
		}
	}

	sort.Slice(samples, func(i, j int) bool {
		if samples[i].ReceivedAt.Equal(samples[j].ReceivedAt) {
			return samples[i].DeviceName < samples[j].DeviceName
		}
		return samples[i].ReceivedAt.Before(samples[j].ReceivedAt)
	})

	deviceNames := make([]string, 0, len(deviceSet))
	for name := range deviceSet {
		deviceNames = append(deviceNames, name)
	}
	sort.Strings(deviceNames)

	columns := make([]string, 0, len(columnSet))
	for column := range columnSet {
		columns = append(columns, column)
	}
	sort.Strings(columns)

	return session, samples, deviceNames, columns, nil
}

func loadGGAForExport(db *sql.DB, sessionID int64) ([]exporter.Sample, error) {
	rows, err := db.Query(`
		SELECT received_at_utc, device_name, latitude, longitude, quality_indicator,
		       satellites_used, hdop, altitude_m, geoid_separation_m, dgps_age_s, dgps_station_id
		FROM gga_records
		WHERE session_id = ?
		ORDER BY received_at_utc, id
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query gga records: %w", err)
	}
	defer rows.Close()

	samples := []exporter.Sample{}
	for rows.Next() {
		var receivedAtRaw, deviceName string
		var latitude, longitude, hdop, altitude float64
		var quality, satellites int
		var geoidSep, dgpsAge sql.NullFloat64
		var stationID sql.NullInt64

		if err := rows.Scan(&receivedAtRaw, &deviceName, &latitude, &longitude, &quality, &satellites, &hdop, &altitude, &geoidSep, &dgpsAge, &stationID); err != nil {
			return nil, fmt.Errorf("scan gga record: %w", err)
		}
		receivedAt, err := time.Parse(time.RFC3339Nano, receivedAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse gga timestamp: %w", err)
		}

		values := map[string]string{
			columnKey(deviceName, "gga", "latitude"):        strconv.FormatFloat(latitude, 'f', -1, 64),
			columnKey(deviceName, "gga", "longitude"):       strconv.FormatFloat(longitude, 'f', -1, 64),
			columnKey(deviceName, "gga", "quality"):         strconv.Itoa(quality),
			columnKey(deviceName, "gga", "satellites_used"): strconv.Itoa(satellites),
			columnKey(deviceName, "gga", "hdop"):            strconv.FormatFloat(hdop, 'f', -1, 64),
			columnKey(deviceName, "gga", "altitude_m"):      strconv.FormatFloat(altitude, 'f', -1, 64),
		}
		if geoidSep.Valid {
			values[columnKey(deviceName, "gga", "geoid_separation_m")] = strconv.FormatFloat(geoidSep.Float64, 'f', -1, 64)
		}
		if dgpsAge.Valid {
			values[columnKey(deviceName, "gga", "dgps_age_s")] = strconv.FormatFloat(dgpsAge.Float64, 'f', -1, 64)
		}
		if stationID.Valid {
			values[columnKey(deviceName, "gga", "dgps_station_id")] = strconv.FormatInt(stationID.Int64, 10)
		}
		samples = append(samples, exporter.Sample{ReceivedAt: receivedAt, DeviceName: deviceName, Values: values})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gga records: %w", err)
	}
	return samples, nil
}

func loadRMCForExport(db *sql.DB, sessionID int64) ([]exporter.Sample, error) {
	rows, err := db.Query(`
		SELECT received_at_utc, device_name, datetime_utc, is_valid, latitude, longitude,
		       speed_knots, course_over_deg, magnetic_variation_deg, positioning_mode
		FROM rmc_records
		WHERE session_id = ?
		ORDER BY received_at_utc, id
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query rmc records: %w", err)
	}
	defer rows.Close()

	samples := []exporter.Sample{}
	for rows.Next() {
		var receivedAtRaw, deviceName, datetimeUTC, positioningMode string
		var isValid bool
		var latitude, longitude, speedKnots float64
		var courseOver, magVar sql.NullFloat64

		if err := rows.Scan(&receivedAtRaw, &deviceName, &datetimeUTC, &isValid, &latitude, &longitude, &speedKnots, &courseOver, &magVar, &positioningMode); err != nil {
			return nil, fmt.Errorf("scan rmc record: %w", err)
		}
		receivedAt, err := time.Parse(time.RFC3339Nano, receivedAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse rmc timestamp: %w", err)
		}

		values := map[string]string{
			columnKey(deviceName, "rmc", "datetime_utc"): datetimeUTC,
			columnKey(deviceName, "rmc", "is_valid"):     strconv.FormatBool(isValid),
			columnKey(deviceName, "rmc", "latitude"):     strconv.FormatFloat(latitude, 'f', -1, 64),
			columnKey(deviceName, "rmc", "longitude"):    strconv.FormatFloat(longitude, 'f', -1, 64),
			columnKey(deviceName, "rmc", "speed_knots"):  strconv.FormatFloat(speedKnots, 'f', -1, 64),
		}
		if courseOver.Valid {
			values[columnKey(deviceName, "rmc", "course_over_deg")] = strconv.FormatFloat(courseOver.Float64, 'f', -1, 64)
		}
		if magVar.Valid {
			values[columnKey(deviceName, "rmc", "magnetic_variation_deg")] = strconv.FormatFloat(magVar.Float64, 'f', -1, 64)
		}
		if positioningMode != "" {
			values[columnKey(deviceName, "rmc", "positioning_mode")] = positioningMode
		}
		samples = append(samples, exporter.Sample{ReceivedAt: receivedAt, DeviceName: deviceName, Values: values})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rmc records: %w", err)
	}
	return samples, nil
}

func loadVTGForExport(db *sql.DB, sessionID int64) ([]exporter.Sample, error) {
	rows, err := db.Query(`
		SELECT received_at_utc, device_name, course_over_deg, speed_knots, speed_kmh, positioning_mode
		FROM vtg_records
		WHERE session_id = ?
		ORDER BY received_at_utc, id
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query vtg records: %w", err)
	}
	defer rows.Close()

	samples := []exporter.Sample{}
	for rows.Next() {
		var receivedAtRaw, deviceName, positioningMode string
		var courseOver, speedKnots, speedKmh float64

		if err := rows.Scan(&receivedAtRaw, &deviceName, &courseOver, &speedKnots, &speedKmh, &positioningMode); err != nil {
			return nil, fmt.Errorf("scan vtg record: %w", err)
		}
		receivedAt, err := time.Parse(time.RFC3339Nano, receivedAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse vtg timestamp: %w", err)
		}

		values := map[string]string{
			columnKey(deviceName, "vtg", "course_over_deg"): strconv.FormatFloat(courseOver, 'f', -1, 64),
			columnKey(deviceName, "vtg", "speed_knots"):     strconv.FormatFloat(speedKnots, 'f', -1, 64),
			columnKey(deviceName, "vtg", "speed_kmh"):       strconv.FormatFloat(speedKmh, 'f', -1, 64),
		}
		if positioningMode != "" {
			values[columnKey(deviceName, "vtg", "positioning_mode")] = positioningMode
		}
		samples = append(samples, exporter.Sample{ReceivedAt: receivedAt, DeviceName: deviceName, Values: values})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vtg records: %w", err)
	}
	return samples, nil
}

func loadDBTForExport(db *sql.DB, sessionID int64) ([]exporter.Sample, error) {
	rows, err := db.Query(`
		SELECT received_at_utc, device_name, depth_feet, depth_meters, depth_fathoms
		FROM dbt_records
		WHERE session_id = ?
		ORDER BY received_at_utc, id
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query dbt records: %w", err)
	}
	defer rows.Close()

	samples := []exporter.Sample{}
	for rows.Next() {
		var receivedAtRaw, deviceName string
		var depthFeet, depthMeters, depthFathoms float64

		if err := rows.Scan(&receivedAtRaw, &deviceName, &depthFeet, &depthMeters, &depthFathoms); err != nil {
			return nil, fmt.Errorf("scan dbt record: %w", err)
		}
		receivedAt, err := time.Parse(time.RFC3339Nano, receivedAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse dbt timestamp: %w", err)
		}

		values := map[string]string{
			columnKey(deviceName, "dbt", "depth_feet"):    strconv.FormatFloat(depthFeet, 'f', -1, 64),
			columnKey(deviceName, "dbt", "depth_meters"):  strconv.FormatFloat(depthMeters, 'f', -1, 64),
			columnKey(deviceName, "dbt", "depth_fathoms"): strconv.FormatFloat(depthFathoms, 'f', -1, 64),
		}
		samples = append(samples, exporter.Sample{ReceivedAt: receivedAt, DeviceName: deviceName, Values: values})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dbt records: %w", err)
	}
	return samples, nil
}

func columnKey(deviceName string, sentence string, field string) string {
	return deviceName + "." + sentence + "." + field
}
