package nmea

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

/*
GGA Global Positioning System Fix Data. Time, Position and fix related data
for a GPS receiver
11
       1         2       3 4        5 6 7  8   9  10 11 12 13 14   15
       |         |       | |        | | |  |   |   | |   | |   |    |
$--GGA,hhmmss.ss,llll.ll,a,yyyyy.yy,a,x,xx,x.x,x.x,M,x.x,M,x.x,xxxx*hh

1) Time (UTC)
2) Latitude
3) N or S (North or South)
4) Longitude
5) E or W (East or West)
6) GPS Quality Indicator,
0 - fix not available,
1 - GPS fix,
2 - Differential GPS fix
7) Number of satellites in view, 00 - 12
8) Horizontal Dilution of precision
9) Antenna Altitude above/below mean-sea-level (geoid)
10) Units of antenna altitude, meters
11) Geoidal separation, the difference between the WGS-84 earth
ellipsoid and mean-sea-level (geoid), "-" means mean-sea-level below ellipsoid
12) Units of geoidal separation, meters
13) Age of differential GPS data, time in seconds since last SC104
type 1 or 9 update, null field when DGPS is not used
14) Differential reference station ID, 0000-1023
15) Checksum

Example:
$GPGGA,015540.000,3150.68378,N,11711.93139,E,1,17,0.6,0051.6,M,0.0,M,,*58
*/

// NewGPGGA allocate GPGGA struct for Global Positioning System Fix Data for a GPS receiver sentence GGA.
// with Time, Position and fix related data
func NewGPGGA(m Message) *GPGGA {
	return &GPGGA{Message: m}
}

// GPGGA struct
type GPGGA struct {
	Message

	TimeUTC            time.Time // Aggregation of TimeUTC data field
	Latitude           LatLong   // In decimal format
	Longitude          LatLong   // In decimal format
	QualityIndicator   QualityIndicator
	NbOfSatellitesUsed uint64
	HDOP               float64
	Altitude           float64
	GeoIDSep           *float64
	DGPSAge            *float64
	DGPSStationID      *uint8

	// FIXME: Manage field below when I found a sample with no-empty data
	// DGPSAge        *uint64
	// DGPSiStationId *string
}

func (m *GPGGA) parse() (err error) {
	if len(m.Fields) != 14 {
		return fmt.Errorf("Incomplete GPGGA message, not enougth data fields (got: %d, wanted: %d)", len(m.Fields), 14)
	}

	// Validate fixed field
	for i, v := range map[int]string{9: "M", 11: "M"} {
		if m.Fields[i] != v {
			return fmt.Errorf("Invalid fixed field at %d (got: %s, wanted: %s)", i+1, m.Fields[i], v)
		}
	}

	if m.TimeUTC, err = time.Parse("150405.000", m.Fields[0]); err != nil {
		return m.Error(fmt.Errorf("Unable to parse time UTC from data field (got: %s)", m.Fields[0]))
	}

	if latitude := strings.TrimSpace(strings.Join(m.Fields[1:3], " ")); len(latitude) > 0 {
		if m.Latitude, err = NewLatLong(latitude); err != nil {
			return m.Error(err)
		}
	}

	if longitude := strings.TrimSpace(strings.Join(m.Fields[3:5], " ")); len(longitude) > 0 {
		if m.Longitude, err = NewLatLong(longitude); err != nil {
			return m.Error(err)
		}
	}

	if m.QualityIndicator, err = ParseQualityIndicator(m.Fields[5]); err != nil {
		return m.Error(err)
	}

	if m.NbOfSatellitesUsed, err = strconv.ParseUint(m.Fields[6], 10, 0); err != nil {
		return m.Error(err)
	}

	if hdop := m.Fields[7]; len(hdop) > 0 {
		if m.HDOP, err = strconv.ParseFloat(hdop, 64); err != nil {
			return m.Error(err)
		}
	}
	if altitude := m.Fields[8]; len(altitude) > 0 {
		if m.Altitude, err = strconv.ParseFloat(altitude, 64); err != nil {
			return m.Error(err)
		}
	}

	if geoIDSep := m.Fields[10]; len(geoIDSep) > 0 {
		id, err := strconv.ParseFloat(m.Fields[10], 64)
		if err != nil {
			return m.Error(err)
		}
		m.GeoIDSep = &id
	}

	// Age of differential GPS data, time in seconds since last SC104
	// type 1 or 9 update, null field when DGPS is not used
	if dGPSAge := m.Fields[12]; len(dGPSAge) > 0 {
		v, err := strconv.ParseFloat(dGPSAge, 64)
		if err != nil {
			return m.Error(err)
		}
		m.DGPSAge = &v
	}

	// Differential reference station ID, 0000-1023
	if dGPSStationID := m.Fields[13]; len(dGPSStationID) > 0 {
		//fmt.Printf("%T  %v\n", dGPSStationID, dGPSStationID)
		v, err := strconv.ParseUint(dGPSStationID, 0, 8)
		//fmt.Printf("%T  %v\n", v, v)
		if err != nil {
			return m.Error(err)
		}
		r := uint8(v)
		m.DGPSStationID = &r
	}

	return nil
}

// Serialize return a valid sentence GGA as string
func (m GPGGA) Serialize() string { // Implement NMEA interface

	hdr := TypeIDs["GPGGA"]
	fields := make([]string, 0)
	////////
	//fmt.Printf("Lat: %s Lon: %s\n", m.Latitude.ToDM(), m.Longitude.ToDM())
	fields = append(fields, m.TimeUTC.Format("150405.000"),
		strings.Trim(m.Latitude.ToDM(), "0"), m.Latitude.CardinalPoint(true).String(),
		strings.Trim(m.Longitude.ToDM(), "0"), m.Longitude.CardinalPoint(false).String(),
		strconv.Itoa(int(m.QualityIndicator)),
		fmt.Sprintf("%d", int(m.NbOfSatellitesUsed)),
	)
	/////////
	//fmt.Println(fields)
	if m.HDOP > 0 {
		fields = append(fields, fmt.Sprintf("%03.1f", m.HDOP))
	} else {
		fields = append(fields, "")
	}

	if m.Altitude > 0 {
		fields = append(fields, PrependXZero(m.Altitude, "%03.1f", 4))

	} else {
		fields = append(fields, "")
	}

	fields = append(fields, "M")

	if m.GeoIDSep != nil {
		fields = append(fields, fmt.Sprintf("%03.1f", *m.GeoIDSep))
	} else {
		fields = append(fields, "")
	}

	fields = append(fields, "M")

	if m.DGPSAge != nil {
		fields = append(fields, fmt.Sprintf("%.1f", *m.DGPSAge))
	} else {
		fields = append(fields, "")
	}

	if m.DGPSStationID != nil {
		fields = append(fields, fmt.Sprintf("%04d", *m.DGPSStationID))
	} else {
		fields = append(fields, "")
	}
	/*
		fields = append(fields,
			"M",
			"", // DGPSAge always empty ?
			"", // DGPSiStationId always empty ?
		)
	*/
	msg := Message{Type: hdr, Fields: fields}
	msg.Checksum = msg.ComputeChecksum()

	return msg.Serialize()
}

const (
	// InvalidIndicator const as 0
	InvalidIndicator = iota
	// GNSSS const as 1
	GNSSS
	// DGPS const as 2
	DGPS
)

// QualityIndicator type as int
type QualityIndicator int

// String return QualityIndicator as human string
func (s QualityIndicator) String() string {
	switch s {
	case InvalidIndicator:
		return "invalid"
	case GNSSS:
		return "GNSS fix"
	case DGPS:
		return "DGPS fix"
	default:
		return "unknow"

	}
}

// ParseQualityIndicator check QualityIndicator validity, return an error
// "unknow value" if not
func ParseQualityIndicator(raw string) (qi QualityIndicator, err error) {
	i, err := strconv.ParseInt(raw, 10, 0)
	if err != nil {
		return
	}

	qi = QualityIndicator(i)
	switch qi {
	case InvalidIndicator, GNSSS, DGPS:
	default:
		err = fmt.Errorf("unknow value")
	}
	return
}
