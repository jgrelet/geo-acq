package simul

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	nmea "github.com/jgrelet/go-nmea"
)

const metersToFeet = 3.280839895
const metersToFathoms = 0.546806649

// NewEchoSounder simulates DBT sentences every interval.
func NewEchoSounder(interval time.Duration, startDepthMeters float64) <-chan string {
	out := make(chan string)
	ticker := time.NewTicker(interval)
	depthMeters := startDepthMeters
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	sentenceDBT, err := nmea.Parse("$GPDBT,108.34,f,33.02,M,18.06,F*35")
	if err != nil {
		panic(fmt.Sprintf("unable to decode dbt sentence: %v", err))
	}
	dbt := sentenceDBT.(*nmea.GPDBT)

	go func() {
		defer ticker.Stop()
		for range ticker.C {
			depthMeters = computeNextDepth(depthMeters, rng)
			dbt.DepthInMeters = depthMeters
			dbt.DepthInFeet = depthMeters * metersToFeet
			dbt.DepthInFathoms = depthMeters * metersToFathoms
			out <- dbt.Serialize()
		}
	}()

	return out
}

// computeNextDepth generates a bounded random walk around the previous depth.
func computeNextDepth(depth float64, rng *rand.Rand) float64 {
	next := depth + (rng.Float64()-0.5)*0.8
	if next < 0.5 {
		next = 0.5
	}
	return math.Round(next*100) / 100
}
