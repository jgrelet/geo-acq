package exporter

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	FormatTSV = "tsv"
	FormatCSV = "csv"

	compactLatitudePrecision  = 7
	compactLongitudePrecision = 7
	compactSpeedPrecision     = 3
	compactDepthPrecision     = 1
)

// RawRecord represents one raw frame exported from the raw SQLite store.
type RawRecord struct {
	ReceivedAt   time.Time
	DeviceName   string
	Transport    string
	SentenceType string
	Payload      string
}

// ProcessedRecord represents one decoded record exported from the processed SQLite store.
type ProcessedRecord struct {
	ReceivedAt   time.Time
	DeviceName   string
	Transport    string
	SentenceType string
	Values       map[string]string
}

// Session describes the exported acquisition session.
type Session struct {
	ID         int64
	Mission    string
	ConfigFile string
	StartedAt  time.Time
}

// CompactProcessedRecord is a simplified processed export row for downstream GIS-style tools.
type CompactProcessedRecord struct {
	DateTimeUTC string
	Latitude    string
	Longitude   string
	SpeedKnots  string
	DepthMeters string
}

// WriteRawTSV writes raw exported records in plain TSV format.
func WriteRawTSV(w io.Writer, session Session, records []RawRecord) error {
	return WriteRawSeparated(w, session, records, "\t")
}

// WriteRawCSV writes raw exported records in plain CSV format.
func WriteRawCSV(w io.Writer, session Session, records []RawRecord) error {
	return WriteRawSeparated(w, session, records, ",")
}

// WriteRawSeparated writes raw exported records using the provided separator.
func WriteRawSeparated(w io.Writer, session Session, records []RawRecord, sep string) error {
	_ = session
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	header := []string{"received_at_utc", "device_name", "transport", "sentence_type", "payload"}
	if _, err := fmt.Fprintln(bw, "# "+strings.Join(header, sep)); err != nil {
		return err
	}

	for _, record := range records {
		fields := []string{
			sanitizeSeparated(record.ReceivedAt.UTC().Format(time.RFC3339Nano), sep),
			sanitizeSeparated(record.DeviceName, sep),
			sanitizeSeparated(record.Transport, sep),
			sanitizeSeparated(record.SentenceType, sep),
			sanitizeSeparated(record.Payload, sep),
		}
		if _, err := fmt.Fprintln(bw, strings.Join(fields, sep)); err != nil {
			return err
		}
	}

	return nil
}

// WriteProcessedTSV writes processed exported records in plain TSV format.
func WriteProcessedTSV(w io.Writer, session Session, records []ProcessedRecord) error {
	return WriteProcessedSeparated(w, session, records, "\t")
}

// WriteProcessedCSV writes processed exported records in plain CSV format.
func WriteProcessedCSV(w io.Writer, session Session, records []ProcessedRecord) error {
	return WriteProcessedSeparated(w, session, records, ",")
}

// WriteProcessedSeparated writes processed exported records using the provided separator.
func WriteProcessedSeparated(w io.Writer, session Session, records []ProcessedRecord, sep string) error {
	_ = session
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	columnSet := map[string]struct{}{}
	for _, record := range records {
		for column := range record.Values {
			columnSet[column] = struct{}{}
		}
	}
	columns := make([]string, 0, len(columnSet))
	for column := range columnSet {
		columns = append(columns, column)
	}
	sort.Strings(columns)

	header := append([]string{"received_at_utc", "device_name", "transport", "sentence_type"}, columns...)
	if _, err := fmt.Fprintln(bw, "# "+strings.Join(header, sep)); err != nil {
		return err
	}

	for _, record := range records {
		fields := []string{
			sanitizeSeparated(record.ReceivedAt.UTC().Format(time.RFC3339Nano), sep),
			sanitizeSeparated(record.DeviceName, sep),
			sanitizeSeparated(record.Transport, sep),
			sanitizeSeparated(record.SentenceType, sep),
		}
		for _, column := range columns {
			fields = append(fields, sanitizeSeparated(record.Values[column], sep))
		}
		if _, err := fmt.Fprintln(bw, strings.Join(fields, sep)); err != nil {
			return err
		}
	}

	return nil
}

// WriteCompactProcessedTSV writes one compact processed line per navigational fix.
func WriteCompactProcessedTSV(w io.Writer, session Session, records []ProcessedRecord) error {
	return WriteCompactProcessedSeparated(w, session, records, "\t")
}

// WriteCompactProcessedCSV writes one compact processed line per navigational fix.
func WriteCompactProcessedCSV(w io.Writer, session Session, records []ProcessedRecord) error {
	return WriteCompactProcessedSeparated(w, session, records, ",")
}

// WriteCompactProcessedSeparated writes compact processed rows using the provided separator.
func WriteCompactProcessedSeparated(w io.Writer, session Session, records []ProcessedRecord, sep string) error {
	_ = session
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	header := []string{"datetime_utc", "latitude", "longitude", "speed_knots", "depth_meters"}
	if _, err := fmt.Fprintln(bw, "# "+strings.Join(header, sep)); err != nil {
		return err
	}

	for _, row := range BuildCompactProcessedRecords(records) {
		fields := []string{
			sanitizeSeparated(row.DateTimeUTC, sep),
			sanitizeSeparated(row.Latitude, sep),
			sanitizeSeparated(row.Longitude, sep),
			sanitizeSeparated(row.SpeedKnots, sep),
			sanitizeSeparated(row.DepthMeters, sep),
		}
		if _, err := fmt.Fprintln(bw, strings.Join(fields, sep)); err != nil {
			return err
		}
	}

	return nil
}

// BuildCompactProcessedRecords reduces processed records to compact navigation rows.
func BuildCompactProcessedRecords(records []ProcessedRecord) []CompactProcessedRecord {
	sorted := append([]ProcessedRecord(nil), records...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].ReceivedAt.Equal(sorted[j].ReceivedAt) {
			if sorted[i].DeviceName == sorted[j].DeviceName {
				return sorted[i].SentenceType < sorted[j].SentenceType
			}
			return sorted[i].DeviceName < sorted[j].DeviceName
		}
		return sorted[i].ReceivedAt.Before(sorted[j].ReceivedAt)
	})

	latestDepth := ""
	rows := []CompactProcessedRecord{}
	for _, record := range sorted {
		switch record.SentenceType {
		case "DBT":
			if value := strings.TrimSpace(record.Values["depth_meters"]); value != "" {
				latestDepth = value
			}
		case "RMC":
			datetimeUTC := strings.TrimSpace(record.Values["datetime_utc"])
			latitude := strings.TrimSpace(record.Values["latitude"])
			longitude := strings.TrimSpace(record.Values["longitude"])
			speedKnots := strings.TrimSpace(record.Values["speed_knots"])
			if datetimeUTC == "" || latitude == "" || longitude == "" {
				continue
			}
			rows = append(rows, CompactProcessedRecord{
				DateTimeUTC: datetimeUTC,
				Latitude:    formatFixedFloat(latitude, compactLatitudePrecision),
				Longitude:   formatFixedFloat(longitude, compactLongitudePrecision),
				SpeedKnots:  formatFixedFloat(speedKnots, compactSpeedPrecision),
				DepthMeters: formatFixedFloat(latestDepth, compactDepthPrecision),
			})
		}
	}
	return rows
}

func formatFixedFloat(value string, precision int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return value
	}
	return fmt.Sprintf("%.*f", precision, parsed)
}

func sanitizeSeparated(value string, sep string) string {
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	if sep == "," {
		value = strings.ReplaceAll(value, ",", " ")
	}
	return value
}
