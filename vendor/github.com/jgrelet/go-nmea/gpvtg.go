package nmea

import (
	"fmt"
	"strconv"
)

/*
VTG Track Made Good and Ground Speed
       1   2 3   4 5   6 7   8 9
       |   | |   | |   | |   | |
$--VTG,x.x,T,x.x,M,x.x,N,x.x,K*hh
1) Track Degrees
2) T = True
3) Track Degrees
4) M = Magnetic
5) Speed Knots
6) N = Knots
7) Speed Kilometers Per Hour
8) K = Kilometres Per Hour
9) Checksum

Examples:
$GPVTG,0.0,T,,M,0.0,N,0.1,K,A*0C
*/

// NewGPVTG allocate vessel track sentence VTG
func NewGPVTG(m Message) *GPVTG {
	return &GPVTG{Message: m}
}

// GPVTG struct
type GPVTG struct {
	Message

	COG             float64 // Course over ground (true) in degree
	SpeedKnots      float64 // Speed over ground in knots
	SpeedKmh        float64 // Speed over ground in km/h
	PositioningMode PositioningMode
}

func (m *GPVTG) parse() (err error) {
	if len(m.Fields) != 9 {
		return m.Error(fmt.Errorf("Incomplete GPVTG message, not enougth data fields (got: %d, wanted: %d)", len(m.Fields), 9))
	}

	// Validate fixed field
	for i, v := range map[int]string{1: "T", 3: "M", 5: "N", 7: "K"} {
		if m.Fields[i] != v {
			return m.Error(fmt.Errorf("Invalid fixed field at %d (got: %s, wanted: %s)", i+1, m.Fields[i], v))
		}
	}

	if m.COG, err = strconv.ParseFloat(m.Fields[0], 64); err != nil {
		return m.Error(fmt.Errorf("Unable to parse true course over ground from data field (got: %s)", m.Fields[0]))
	}

	if m.SpeedKnots, err = strconv.ParseFloat(m.Fields[4], 64); err != nil {
		return m.Error(fmt.Errorf("Unable to parse speed from data field (got: %s)", m.Fields[4]))
	}

	if m.SpeedKmh, err = strconv.ParseFloat(m.Fields[6], 64); err != nil {
		return m.Error(fmt.Errorf("Unable to parse speed from data field (got: %s)", m.Fields[6]))
	}

	if m.PositioningMode, err = ParsePositioningMode(m.Fields[8]); err != nil {
		return m.Error(fmt.Errorf("Unable to parse GPS positioning mode from data field (got: %s)", m.Fields[8]))
	}

	return nil
}

// Serialize return a valid sentence VTG as string
func (m GPVTG) Serialize() string { // Implement NMEA interface

	hdr := TypeIDs["GPVTG"]
	fields := make([]string, 0)
	fields = append(fields, fmt.Sprintf("%03.1f", m.COG), "T",
		"", "M",
		fmt.Sprintf("%03.1f", m.SpeedKnots), "N",
		fmt.Sprintf("%03.1f", m.SpeedKmh), "K",
		string(m.PositioningMode))
	msg := Message{Type: hdr, Fields: fields}
	msg.Checksum = msg.ComputeChecksum()

	return msg.Serialize()
}
