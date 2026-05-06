package simul

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestComputeNextPositionEastbound(t *testing.T) {
	lat, lon := computeNextPosition(48.0, 2.0, 1852.0, 90.0)
	if math.Abs(lat-48.0) > 0.01 {
		t.Fatalf("unexpected latitude: got %f", lat)
	}
	if lon <= 2.0 {
		t.Fatalf("expected longitude to increase, got %f", lon)
	}
}

func TestDistanceMeters(t *testing.T) {
	got := distanceMeters(10, 1)
	want := 10.0 * 1852.0 / 3600.0
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("unexpected distance: got %f want %f", got, want)
	}
}

func TestNewGpsEmitsSentences(t *testing.T) {
	ch := NewGps(1, 10, 90)

	select {
	case sentence := <-ch:
		if len(sentence) == 0 || sentence[0] != '$' {
			t.Fatalf("unexpected sentence %q", sentence)
		}
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("timeout waiting for gps sentence")
	}
}

func TestNewGpsEmitsRMC(t *testing.T) {
	ch := NewGps(1, 10, 90)

	timeout := time.After(4 * time.Second)
	for {
		select {
		case sentence := <-ch:
			if strings.HasPrefix(sentence, "$GPRMC,") {
				return
			}
		case <-timeout:
			t.Fatal("timeout waiting for simulated RMC sentence")
		}
	}
}

func TestNewGpsEmitsZDA(t *testing.T) {
	ch := NewGps(1, 10, 90)

	timeout := time.After(4 * time.Second)
	for {
		select {
		case sentence := <-ch:
			if strings.HasPrefix(sentence, "$GPZDA,") {
				return
			}
		case <-timeout:
			t.Fatal("timeout waiting for simulated ZDA sentence")
		}
	}
}
