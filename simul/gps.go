package simul

import (
	"fmt"
	"math"
	"time"

	"github.com/jgrelet/geo-acq/util"

	nmea "github.com/jgrelet/go-nmea"
)

const earthRadiusMeters = 6371000.0

// NewGps simulates GGA and VTG sentences every interval.
func NewGps(interval time.Duration, sog, cog float64) <-chan string {
	out := make(chan string)
	ticker := time.NewTicker(time.Second * interval)

	sentenceGGA, err := nmea.Parse("$GPGGA,015540.000,4807.038,N,01131.000,E,1,17,0.6,51.6,M,0.0,M,,*59")
	if err != nil {
		panic(fmt.Sprintf("unable to decode gga sentence: %v", err))
	}
	sentenceVTG, err := nmea.Parse("$GPVTG,0.0,T,,M,0.0,N,0.0,K,A*0D")
	if err != nil {
		panic(fmt.Sprintf("unable to decode vtg sentence: %v", err))
	}

	gpgga := sentenceGGA.(*nmea.GPGGA)
	gpvtg := sentenceVTG.(*nmea.GPVTG)

	go func() {
		defer ticker.Stop()
		for range ticker.C {
			gpgga.TimeUTC = time.Now().UTC()
			latitude, longitude := computeNextPosition(
				float64(gpgga.Latitude),
				float64(gpgga.Longitude),
				distanceMeters(sog, interval),
				cog,
			)
			gpgga.Latitude = nmea.LatLong(latitude)
			gpgga.Longitude = nmea.LatLong(longitude)
			out <- gpgga.Serialize()

			gpvtg.COG = normalizeHeading(cog)
			gpvtg.SpeedKnots = sog
			gpvtg.SpeedKmh = sog * util.MileToKm
			out <- gpvtg.Serialize()
		}
	}()

	return out
}

func distanceMeters(speedKnots float64, interval time.Duration) float64 {
	hours := (time.Second * interval).Hours()
	nauticalMiles := speedKnots * hours
	return nauticalMiles * 1852.0
}

// computeNextPosition calculates next position from decimal degrees, distance in meters and heading in degrees.
func computeNextPosition(latDeg, lonDeg, distanceMeters, headingDeg float64) (newLatDeg, newLonDeg float64) {
	latRad := latDeg * math.Pi / 180.0
	lonRad := lonDeg * math.Pi / 180.0
	headingRad := normalizeHeading(headingDeg) * math.Pi / 180.0
	angularDistance := distanceMeters / earthRadiusMeters

	newLatRad := math.Asin(math.Sin(latRad)*math.Cos(angularDistance) +
		math.Cos(latRad)*math.Sin(angularDistance)*math.Cos(headingRad))
	newLonRad := lonRad + math.Atan2(
		math.Sin(headingRad)*math.Sin(angularDistance)*math.Cos(latRad),
		math.Cos(angularDistance)-math.Sin(latRad)*math.Sin(newLatRad),
	)

	return newLatRad * 180.0 / math.Pi, normalizeLongitude(newLonRad * 180.0 / math.Pi)
}

func normalizeHeading(heading float64) float64 {
	value := math.Mod(heading, 360.0)
	if value < 0 {
		value += 360.0
	}
	return value
}

func normalizeLongitude(lon float64) float64 {
	for lon > 180.0 {
		lon -= 360.0
	}
	for lon < -180.0 {
		lon += 360.0
	}
	return lon
}
