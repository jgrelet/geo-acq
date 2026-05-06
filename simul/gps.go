package simul

import (
	"fmt"
	"math"
	"time"

	"github.com/jgrelet/geo-acq/util"

	nmea "github.com/jgrelet/go-nmea"
)

const earthRadiusMeters = 6371000.0

// NewGps simulates GGA, RMC, ZDA and VTG sentences every interval.
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
	sentenceRMC, err := nmea.Parse("$GPRMC,015540.000,A,4807.038,N,01131.000,E,0.0,0.0,040526,,,A*6C")
	if err != nil {
		panic(fmt.Sprintf("unable to decode rmc sentence: %v", err))
	}

	gpgga := sentenceGGA.(*nmea.GPGGA)
	gpvtg := sentenceVTG.(*nmea.GPVTG)
	gprmc := sentenceRMC.(*nmea.GPRMC)

	go func() {
		defer ticker.Stop()
		for range ticker.C {
			nowUTC := time.Now().UTC()
			gpgga.TimeUTC = nowUTC
			latitude, longitude := computeNextPosition(
				float64(gpgga.Latitude),
				float64(gpgga.Longitude),
				distanceMeters(sog, interval),
				cog,
			)
			gpgga.Latitude = nmea.LatLong(latitude)
			gpgga.Longitude = nmea.LatLong(longitude)
			out <- gpgga.Serialize()

			gprmc.DateTimeUTC = nowUTC
			gprmc.IsValid = true
			gprmc.Latitude = gpgga.Latitude
			gprmc.Longitude = gpgga.Longitude
			gprmc.Speed = sog
			gprmc.COG = normalizeHeading(cog)
			out <- gprmc.Serialize()

			out <- buildZDASentence(nowUTC)

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

func buildZDASentence(now time.Time) string {
	payload := fmt.Sprintf(
		"GPZDA,%02d%02d%02d.000,%02d,%02d,%04d,00,00",
		now.UTC().Hour(),
		now.UTC().Minute(),
		now.UTC().Second(),
		now.UTC().Day(),
		int(now.UTC().Month()),
		now.UTC().Year(),
	)
	return "$" + payload + "*" + checksum(payload)
}

func checksum(payload string) string {
	sum := byte(0)
	for i := 0; i < len(payload); i++ {
		sum ^= payload[i]
	}
	return fmt.Sprintf("%02X", sum)
}
