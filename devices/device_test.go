package devices

import (
	"bufio"
	"io"
	"strings"
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

func TestNormalizeSerialLine(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "keeps valid sentence",
			in:   "$GPGGA,123*00\r\n",
			want: "$GPGGA,123*00",
		},
		{
			name: "resyncs on dollar sign",
			in:   "garbage$GPRMC,123*00\r\n",
			want: "$GPRMC,123*00",
		},
		{
			name: "keeps non nmea fragment as is",
			in:   "16,79.57,040526,,,D*47\r\n",
			want: "16,79.57,040526,,,D*47",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeSerialLine(tc.in); got != tc.want {
				t.Fatalf("normalizeSerialLine(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestDeviceReadDelimitedByCRLF(t *testing.T) {
	dev := &Device{
		reader: bufio.NewReader(strings.NewReader("$GPGGA,123*00\r\n$GPRMC,456*00\r\n")),
	}

	first, err := dev.Read()
	if err != nil {
		t.Fatalf("first Read() error = %v", err)
	}
	if first != "$GPGGA,123*00" {
		t.Fatalf("first Read() = %q", first)
	}

	second, err := dev.Read()
	if err != nil {
		t.Fatalf("second Read() error = %v", err)
	}
	if second != "$GPRMC,456*00" {
		t.Fatalf("second Read() = %q", second)
	}
}

func TestDeviceReadDelimitedByCR(t *testing.T) {
	dev := &Device{
		reader: bufio.NewReader(strings.NewReader("$GPGGA,123*00\r$GPRMC,456*00\r")),
	}

	first, err := dev.Read()
	if err != nil {
		t.Fatalf("first Read() error = %v", err)
	}
	if first != "$GPGGA,123*00" {
		t.Fatalf("first Read() = %q", first)
	}

	second, err := dev.Read()
	if err != nil {
		t.Fatalf("second Read() error = %v", err)
	}
	if second != "$GPRMC,456*00" {
		t.Fatalf("second Read() = %q", second)
	}
}

func TestDeviceReadEOFWithPartialSentence(t *testing.T) {
	dev := &Device{
		reader: bufio.NewReader(strings.NewReader("$GPGGA,123*00")),
	}

	got, err := dev.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got != "$GPGGA,123*00" {
		t.Fatalf("Read() = %q", got)
	}

	_, err = dev.Read()
	if err == nil || err == io.EOF {
		return
	}
	t.Fatalf("expected EOF or nil after draining reader, got %v", err)
}

func TestExtractNMEASentences(t *testing.T) {
	data := []byte("junk$GPGGA,123*00\r\n$GPRMC,456*00\r\n")

	got, rest := extractNMEASentences(data)

	if len(rest) != 0 {
		t.Fatalf("expected empty rest, got %q", string(rest))
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 sentences, got %d", len(got))
	}
	if got[0] != "$GPGGA,123*00" {
		t.Fatalf("first sentence = %q", got[0])
	}
	if got[1] != "$GPRMC,456*00" {
		t.Fatalf("second sentence = %q", got[1])
	}
}

func TestExtractNMEASentencesKeepsIncompleteTail(t *testing.T) {
	data := []byte("junk$GPGGA,123*00\r\n$GPRMC,456")

	got, rest := extractNMEASentences(data)

	if len(got) != 1 {
		t.Fatalf("expected 1 sentence, got %d", len(got))
	}
	if got[0] != "$GPGGA,123*00" {
		t.Fatalf("first sentence = %q", got[0])
	}
	if string(rest) != "$GPRMC,456" {
		t.Fatalf("rest = %q", string(rest))
	}
}
