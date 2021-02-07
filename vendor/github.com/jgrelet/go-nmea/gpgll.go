package nmea

import (
	"fmt"
	"strings"
	"time"
)

/*
GLL Geographic Position – Latitude/Longitude
       1       2 3        4 5         6 7
       |       | |        | |         | |
$--GLL,llll.ll,a,yyyyy.yy,a,hhmmss.ss,A*hh

1) Latitude
2) N or S (North or South)
3) Longitude
4) E or W (East or West)
5) Time (UTC)
6) Status A - Data Valid, V - Data Invalid
7) Checksum

Example:
$GPGLL,3110.2908,N,12123.2348,E,041139.000,A,A*59
*/

// NewGPGLL allocate struct GPGLL for GLL sentence GLL (Geographic Position)
// with Latitude/Longitude
func NewGPGLL(m Message) *GPGLL {
	return &GPGLL{Message: m}
}

// GPGLL struct
type GPGLL struct {
	Message

	TimeUTC             time.Time // Aggregation of TimeUTC data field
	Latitude, Longitude LatLong   // In decimal format
	IsValid             DataValid
	PositioningMode     PositioningMode
}

func (m *GPGLL) parse() (err error) {
	if len(m.Fields) != 7 {
		return m.Error(fmt.Errorf("Incomplete GPGLL message, not enougth data fields (got: %d, wanted: %d)", len(m.Fields), 7))
	}

	if latitude := strings.TrimSpace(strings.Join(m.Fields[0:2], " ")); len(latitude) > 0 {
		if m.Latitude, err = NewLatLong(latitude); err != nil {
			return m.Error(err)
		}
	}

	if longitude := strings.TrimSpace(strings.Join(m.Fields[2:4], " ")); len(longitude) > 0 {
		if m.Longitude, err = NewLatLong(longitude); err != nil {
			return m.Error(err)
		}
	}

	if m.TimeUTC, err = time.Parse("150405.000", m.Fields[4]); err != nil {
		return m.Error(fmt.Errorf("Unable to parse time UTC from data field (got: %s)", m.Fields[4]))
	}

	m.IsValid = (m.Fields[5] == "A")

	if m.PositioningMode, err = ParsePositioningMode(m.Fields[6]); err != nil {
		return m.Error(fmt.Errorf("Unable to parse GPS positioning mode from data field (got: %s)", m.Fields[6]))
	}

	return nil
}

// Serialize return a valid sentence GLL as string
func (m GPGLL) Serialize() string { // Implement NMEA interface

	hdr := TypeIDs["GPGLL"]
	fields := make([]string, 0)
	fields = append(fields,
		m.TimeUTC.Format("150405.000"),
		strings.Trim(m.Latitude.ToDM(), "0"), m.Latitude.CardinalPoint(true).String(),
		strings.Trim(m.Longitude.ToDM(), "0"), m.Longitude.CardinalPoint(false).String(),
		string(m.IsValid.Serialize()))
	msg := Message{Type: hdr, Fields: fields}
	msg.Checksum = msg.ComputeChecksum()

	return msg.Serialize()
}
