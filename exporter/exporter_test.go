package exporter

import (
	"testing"
	"time"
)

func TestBuildRowsSlowestDevice(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	frames := []Frame{
		{ReceivedAt: base.Add(1 * time.Second), DeviceName: "gps", Payload: "g1"},
		{ReceivedAt: base.Add(2 * time.Second), DeviceName: "gps", Payload: "g2"},
		{ReceivedAt: base.Add(2 * time.Second), DeviceName: "echosounder", Payload: "e1"},
	}

	rows, err := BuildRows(frames, []string{"echosounder", "gps"}, ModeSlowestDevice, 0)
	if err != nil {
		t.Fatalf("build rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Values["gps"] != "g2" || rows[0].Values["echosounder"] != "e1" {
		t.Fatalf("unexpected aligned values: %+v", rows[0].Values)
	}
}

func TestBuildRowsFixedInterval(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	frames := []Frame{
		{ReceivedAt: base.Add(1 * time.Second), DeviceName: "gps", Payload: "g1"},
		{ReceivedAt: base.Add(3 * time.Second), DeviceName: "echosounder", Payload: "e1"},
	}

	rows, err := BuildRows(frames, []string{"echosounder", "gps"}, ModeFixedInterval, time.Second)
	if err != nil {
		t.Fatalf("build rows: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[2].Values["gps"] != "g1" || rows[2].Values["echosounder"] != "e1" {
		t.Fatalf("unexpected final values: %+v", rows[2].Values)
	}
}
