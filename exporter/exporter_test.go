package exporter

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestWriteRawTSV(t *testing.T) {
	var buf bytes.Buffer
	session := Session{
		ID:         1,
		Mission:    "test",
		ConfigFile: "test.toml",
		StartedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	records := []RawRecord{
		{
			ReceivedAt:   time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC),
			DeviceName:   "gps",
			Transport:    "udp",
			SentenceType: "GPGGA",
			Payload:      "$GPGGA,...",
		},
	}

	if err := WriteRawTSV(&buf, session, records); err != nil {
		t.Fatalf("write raw tsv: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "# received_at_utc\tdevice_name\ttransport\tsentence_type\tpayload") {
		t.Fatalf("missing raw comment header in output: %s", output)
	}
	if strings.Contains(output, "\nreceived_at_utc\tdevice_name\ttransport\tsentence_type\tpayload\n") {
		t.Fatalf("unexpected duplicated raw header in output: %s", output)
	}
	if !strings.Contains(output, "gps\tudp\tGPGGA\t$GPGGA,...") {
		t.Fatalf("missing raw record in output: %s", output)
	}
}

func TestWriteRawCSV(t *testing.T) {
	var buf bytes.Buffer
	session := Session{}
	records := []RawRecord{
		{
			ReceivedAt:   time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC),
			DeviceName:   "gps",
			Transport:    "udp",
			SentenceType: "GPGGA",
			Payload:      "$GPGGA,...",
		},
	}

	if err := WriteRawCSV(&buf, session, records); err != nil {
		t.Fatalf("write raw csv: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "# received_at_utc,device_name,transport,sentence_type,payload") {
		t.Fatalf("missing raw csv comment header in output: %s", output)
	}
	if strings.Contains(output, "\nreceived_at_utc,device_name,transport,sentence_type,payload\n") {
		t.Fatalf("unexpected duplicated raw csv header in output: %s", output)
	}
	if !strings.Contains(output, "gps,udp,GPGGA,") {
		t.Fatalf("missing raw csv record in output: %s", output)
	}
}

func TestWriteProcessedTSV(t *testing.T) {
	var buf bytes.Buffer
	session := Session{
		ID:         1,
		Mission:    "test",
		ConfigFile: "test.toml",
		StartedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	records := []ProcessedRecord{
		{
			ReceivedAt:   time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC),
			DeviceName:   "gps",
			Transport:    "udp",
			SentenceType: "RMC",
			Values: map[string]string{
				"latitude":    "48.1173",
				"longitude":   "11.5167",
				"speed_knots": "0",
			},
		},
	}

	if err := WriteProcessedTSV(&buf, session, records); err != nil {
		t.Fatalf("write processed tsv: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "# received_at_utc\tdevice_name\ttransport\tsentence_type\tlatitude\tlongitude\tspeed_knots") {
		t.Fatalf("missing processed comment header in output: %s", output)
	}
	if strings.Contains(output, "\nreceived_at_utc\tdevice_name\ttransport\tsentence_type\tlatitude\tlongitude\tspeed_knots\n") {
		t.Fatalf("unexpected duplicated processed header in output: %s", output)
	}
	if !strings.Contains(output, "gps\tudp\tRMC\t48.1173\t11.5167\t0") {
		t.Fatalf("missing processed record in output: %s", output)
	}
}

func TestBuildCompactProcessedRecords(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	records := []ProcessedRecord{
		{
			ReceivedAt:   base.Add(1 * time.Second),
			DeviceName:   "echosounder",
			Transport:    "udp",
			SentenceType: "DBT",
			Values: map[string]string{
				"depth_meters": "12.5",
			},
		},
		{
			ReceivedAt:   base.Add(2 * time.Second),
			DeviceName:   "gps",
			Transport:    "udp",
			SentenceType: "RMC",
			Values: map[string]string{
				"datetime_utc": "2026-01-01T00:00:02Z",
				"latitude":     "48.1173",
				"longitude":    "11.5167",
				"speed_knots":  "3.2",
			},
		},
	}

	rows := BuildCompactProcessedRecords(records)
	if len(rows) != 1 {
		t.Fatalf("compact row count = %d, want 1", len(rows))
	}
	if rows[0].Latitude != "48.1173000" {
		t.Fatalf("latitude = %q, want 48.1173000", rows[0].Latitude)
	}
	if rows[0].Longitude != "11.5167000" {
		t.Fatalf("longitude = %q, want 11.5167000", rows[0].Longitude)
	}
	if rows[0].SpeedKnots != "3.200" {
		t.Fatalf("speed_knots = %q, want 3.200", rows[0].SpeedKnots)
	}
	if rows[0].DepthMeters != "12.5" {
		t.Fatalf("depth_meters = %q, want 12.5", rows[0].DepthMeters)
	}
	if rows[0].DateTimeUTC != "2026-01-01T00:00:02Z" {
		t.Fatalf("datetime_utc = %q, want 2026-01-01T00:00:02Z", rows[0].DateTimeUTC)
	}
}

func TestWriteCompactProcessedTSV(t *testing.T) {
	var buf bytes.Buffer
	session := Session{
		ID:         1,
		Mission:    "test",
		ConfigFile: "test.toml",
		StartedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	records := []ProcessedRecord{
		{
			ReceivedAt:   time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC),
			DeviceName:   "gps",
			Transport:    "udp",
			SentenceType: "RMC",
			Values: map[string]string{
				"datetime_utc": "2026-01-01T00:00:01Z",
				"latitude":     "48.1173",
				"longitude":    "11.5167",
				"speed_knots":  "0",
			},
		},
	}

	if err := WriteCompactProcessedTSV(&buf, session, records); err != nil {
		t.Fatalf("write compact processed tsv: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "# datetime_utc\tlatitude\tlongitude\tspeed_knots\tdepth_meters") {
		t.Fatalf("missing compact comment header in output: %s", output)
	}
	if strings.Contains(output, "\ndatetime_utc\tlatitude\tlongitude\tspeed_knots\tdepth_meters\n") {
		t.Fatalf("unexpected duplicated compact header in output: %s", output)
	}
	if !strings.Contains(output, "2026-01-01T00:00:01Z\t48.1173000\t11.5167000\t0.000\t") {
		t.Fatalf("missing compact record in output: %s", output)
	}
}
