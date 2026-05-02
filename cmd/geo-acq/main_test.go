package main

import (
	"strings"
	"testing"
	"time"
)

func TestFormatTerminalFrame(t *testing.T) {
	msg := deviceMessage{
		receivedAt:   time.Date(2026, 5, 2, 10, 11, 12, 0, time.UTC),
		deviceName:   "gps",
		transport:    "serial",
		port:         "COM3",
		payload:      "$GPGGA,123",
		sentenceType: "GPGGA",
	}

	got := formatTerminalFrame(msg)

	for _, want := range []string{
		"2026-05-02T10:11:12Z",
		"gps",
		"serial",
		"COM3",
		"GPGGA",
		"$GPGGA,123",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatted frame %q does not contain %q", got, want)
		}
	}
}

func TestFormatTerminalFrameFallsBackToRaw(t *testing.T) {
	msg := deviceMessage{
		receivedAt: time.Date(2026, 5, 2, 10, 11, 12, 0, time.UTC),
		deviceName: "gps",
		transport:  "serial",
		port:       "COM3",
		payload:    "???",
	}

	got := formatTerminalFrame(msg)
	if !strings.Contains(got, "RAW") {
		t.Fatalf("formatted frame %q does not contain RAW fallback", got)
	}
}
