package devices

import (
	"testing"

	"github.com/jgrelet/geo-acq/config"
)

func TestNewUDPConnDialMode(t *testing.T) {
	conn, err := newUDPConn(config.UDP{Host: "127.0.0.1", Port: "10183"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn.mode != "dial" {
		t.Fatalf("expected dial mode, got %q", conn.mode)
	}
}

func TestNewUDPConnListenMode(t *testing.T) {
	conn, err := newUDPConn(config.UDP{Port: "10183"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn.mode != "listen" {
		t.Fatalf("expected listen mode, got %q", conn.mode)
	}
}

func TestNewUDPConnRequiresPort(t *testing.T) {
	if _, err := newUDPConn(config.UDP{}); err == nil {
		t.Fatal("expected error when port is missing")
	}
}
