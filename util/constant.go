package util

import (
	"fmt"
	"math"

	nmea "github.com/jgrelet/go-nmea"
)

// P is usefull shortcut macros
var P = fmt.Println

// F is usefull shortcut macros
var F = fmt.Printf

var badLatLong = nmea.LatLong(1e+36)

const (
	// Allowed devices names
	gps     = "gps"
	sounder = "sounder"
	radar   = "radar"
)

const (
	radToDeg  = 180 / math.Pi
	degToRad  = math.Pi / 180
	radToGrad = 200 / math.Pi
	gradToDeg = math.Pi / 200
)

const (
	// KmToMile convert km to mile
	KmToMile = 1. / 1.852
	// MileToKm convert mile to km
	MileToKm = 1. * 1.852
)

const (
	// CR is carriage return (0x0D)
	CR = string("\r")
	// LF is line feed (0x0A)
	LF = string("\n")
)
