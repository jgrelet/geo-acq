package simul

import (
	"fmt"
	"math"
	"time"

	"github.com/jgrelet/geo-acq/util"

	nmea "github.com/jgrelet/go-nmea"
)

// NewGps simulate GPS NME183 GGA sentence every second
// use a channel
func NewGps(interval time.Duration, sog, cog float64) <-chan string {
	out := make(chan string)
	tick := time.NewTicker(time.Second * interval).C
	// initialize GGA sentence
	// gga := nmea.Message{}
	// gga.Type = nmea.TypeIDs["GPGGA"]
	// msg := "000005.200,0843.74714,S,03446.48123,W,1,14,00.7,000.000,M,0.0,M,0.0,0000"
	// gga.Fields = strings.Split(msg, nmea.FieldDelimiter)
	// msg, err := nmea.Parse("$GPGGA,015540.000,0001.0,N,02300.0,W,1,17,0.6,0051.6,M,0.0,M,,*79")
	sentence, err := nmea.Parse("$GPGGA,015540.000,0001.0,N,02300.0,E,1,17,0.6,0051.6,M,0.0,M,,*5b")
	if err != nil {
		fmt.Println("Unable to decode nmea message, err:", err.Error())
	}

	// initialize GPGGA struc from sentence
	gpgga := sentence.(*nmea.GPGGA)

	// go routine
	go func() { // infinite lopp
		for {
			select {
			case <-tick:
				gpgga.TimeUTC = time.Now()
				latitude, longitude := computeNextPosition(float64(gpgga.Latitude),
					float64(gpgga.Longitude), sog, cog)
				gpgga.Latitude = nmea.LatLong(latitude)
				gpgga.Longitude = nmea.LatLong(longitude)
				out <- gpgga.Serialize()
			}
		}
	}()
	return out
}

// computeNextPosition calculate next position with speed and heading
// see: http://www.movable-type.co.uk/scripts/latlong.html
// Destination point given distance and bearing from start point
func computeNextPosition(lat, lon, speed, heading float64) (newLat, newLon float64) {
	r := 6371. * 1000. // Earth Radius in m
	distance := speed * util.KmToMile
	k := distance / r
	//fmt.Printf("Lat: %f, Lon: %f, Speed: %4.1f, Heading: %5.1f\n", lat, lon, distance, heading)
	newLat = math.Asin(math.Sin(lat)*math.Cos(k) +
		math.Cos(lat)*math.Sin(k)*math.Cos(heading))
	newLon = lon + math.Atan2(math.Sin(heading)*math.Sin(k)*math.Cos(lat),
		math.Cos(k)-math.Sin(lat)*math.Sin(newLat))
	//fmt.Printf("newLat: %f, newLon: %f, Speed (km) %4.1f, Heading: %5.1f (rad)\n", newLat, newLon, speed, heading)
	return
}
