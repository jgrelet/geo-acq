package decoder

import (
	"encoding/json"
	"testing"
)

func TestDecodeNMEACommonSentences(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		sentenceType string
	}{
		{
			name:         "GGA",
			raw:          "$GPGGA,015540.000,4807.038,N,01131.000,E,1,17,0.6,51.6,M,0.0,M,,*59",
			sentenceType: "GPGGA",
		},
		{
			name:         "VTG",
			raw:          "$GPVTG,0.0,T,,M,0.0,N,0.0,K,A*0D",
			sentenceType: "GPVTG",
		},
		{
			name:         "DBT",
			raw:          "$GPDBT,108.34,f,33.02,M,18.06,F*35",
			sentenceType: "GPDBT",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decoded, err := DecodeNMEA(tc.raw)
			if err != nil {
				t.Fatalf("DecodeNMEA() error = %v", err)
			}
			if decoded.SentenceType != tc.sentenceType {
				t.Fatalf("sentence type = %q, want %q", decoded.SentenceType, tc.sentenceType)
			}

			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(decoded.JSON), &payload); err != nil {
				t.Fatalf("unmarshal decoded json: %v", err)
			}
			if payload["sentence_type"] != tc.sentenceType {
				t.Fatalf("json sentence type = %v, want %q", payload["sentence_type"], tc.sentenceType)
			}
		})
	}
}
