package nmea

import (
	"fmt"
	"strconv"
)

/*
GSA GPS DOP and active satellites
       1 2 3                        14 15 16  17  18
       | | |                         | |   |   |   |
$--GSA,a,a,x,x,x,x,x,x,x,x,x,x,x,x,x,x,x.x,x.x,x.x*hh

1) Selection mode
2) Mode
3) ID of 1st satellite used for fix
4) ID of 2nd satellite used for fix
...
14) ID of 12th satellite used for fix
15) PDOP in meters
16) HDOP in meters
17) VDOP in meters
18) Checksum

 Example:
 $GPGSA,A,3,14,06,16,31,23,,,,,,,,1.66,1.42,0.84*0F
*/

// NewGPGSA allocate struct GPGSA for GSA sentence GPS DOP and active satellites
func NewGPGSA(m Message) *GPGSA {
	return &GPGSA{Message: m}
}

// GPGSA struct
type GPGSA struct {
	Message

	Mode                   Mode
	FixStatus              FixStatus
	SatelliteUsedOnChannel [13]int // Note: index 0 not used (channel 1..12)
	PDOP, HDOP, VDOP       float64
}

func (m *GPGSA) parse() (err error) {
	if len(m.Fields) != 17 {
		return m.Error(fmt.Errorf("Incomplete GPGSA message, not enougth data fields (got: %d, wanted: %d)", len(m.Fields), 17))
	}

	if m.Mode, err = ParseMode(m.Fields[0]); err != nil {
		return m.Error(err)
	}

	if m.FixStatus, err = ParseFixStatus(m.Fields[1]); err != nil {
		return m.Error(err)
	}

	for k, v := range m.Fields[2:14] {
		m.SatelliteUsedOnChannel[k+1], _ = strconv.Atoi(v)
	}

	// data could be empty
	pdop, hdop, vdop := m.Fields[14], m.Fields[15], m.Fields[16]

	if len(pdop) > 0 {
		if m.PDOP, err = strconv.ParseFloat(pdop, 64); err != nil {
			return m.Error(err)
		}
	}

	if len(hdop) > 0 {
		if m.HDOP, err = strconv.ParseFloat(hdop, 64); err != nil {
			return m.Error(err)
		}
	}

	if len(vdop) > 0 {
		if m.VDOP, err = strconv.ParseFloat(vdop, 64); err != nil {
			return m.Error(err)
		}
	}

	return nil
}

const (
	_ = iota // pass value 0
	// FixStatusNoFix constante as 1
	FixStatusNoFix
	// FixStatus2D constante as 2
	FixStatus2D
	// FixStatus3D constante as 3
	FixStatus3D
)

// FixStatus type as int
type FixStatus int

// String return FixStatus as human string
func (s FixStatus) String() string {
	switch s {
	case FixStatusNoFix:
		return "No fix"
	case FixStatus2D:
		return "2D fix"
	case FixStatus3D:
		return "3D fix"
	default:
		return "unknow"
	}
}

// ParseFixStatus check FixStatus validity, return an error
// "unknow value (got: %d)" if not
func ParseFixStatus(raw string) (fs FixStatus, err error) {
	i, err := strconv.ParseInt(raw, 10, 0)
	if err != nil {
		return
	}

	fs = FixStatus(i)
	switch fs {
	case FixStatusNoFix, FixStatus2D, FixStatus3D:
	default:
		err = fmt.Errorf("unknow value (got: %d)", i)
	}
	return
}

const (
	// ModeManual constante as "M"
	ModeManual Mode = "M"
	// ModeAuto constante as "A"
	ModeAuto Mode = "A"
)

// Mode type as string
type Mode string

// String return Mode type as string
func (m Mode) String() string {
	return string(m)
}

// ParseMode check Mode validity, return an error
// "unknow value (got: %d)" if not
func ParseMode(raw string) (m Mode, err error) {
	m = Mode(raw)
	switch m {
	case ModeManual, ModeAuto:
	default:
		err = fmt.Errorf("unknow value (got: %s)", m)
	}
	return
}
