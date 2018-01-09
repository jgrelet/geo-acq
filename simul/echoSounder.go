package simul

import (
	"math/rand"
	"strconv"
	"strings"
	"time"

	nmea "github.com/jgrelet/go-nmea"
)

const payload = "108.34,f,33.02,M,18.06,F"

// EchoSounderChan
var (
	EchoSounderChan chan interface{}
)

// EchoSounder simulate echoSounder
func EchoSounder() {
	tickChan := time.NewTicker(time.Millisecond * 600).C

	// initialize DBT sentence
	dbt := nmea.Message{}
	dbt.Type = nmea.TypeIDs["GPDBT"]
	dbt.Fields = strings.Split(payload, nmea.FieldDelimiter)
	depth, _ := strconv.ParseFloat(dbt.Fields[0], 64)

	for {
		select {
		case <-tickChan:
			dbt.Fields[0] = time.Now().Format("150405.000")

			depthInMeters, depthInFeet, depthInFathoms := computeNextDepth(depth)
			dbt.Fields[0] = strconv.FormatFloat(depthInFeet, 'f', 2, 64)
			dbt.Fields[2] = strconv.FormatFloat(depthInMeters, 'f', 2, 64)
			dbt.Fields[4] = strconv.FormatFloat(depthInFathoms, 'f', 2, 64)
			dbt.Checksum = dbt.ComputeChecksum()
			// s := gga.Serialize()
			// fmt.Printf("%v\n", s)
			EchoSounderChan <- dbt.Serialize()
		}
	}
}

// from depth, compute a new randon depth in meters, feet and fathoms
func computeNextDepth(depth float64) (float64, float64, float64) {
	return (rand.Float64() * 5) + depth, depth / 0.3048, depth * 0.5468
}
