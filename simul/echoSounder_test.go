package simul

import (
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestComputeNextDepthPositive(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	got := computeNextDepth(10.0, rng)
	if got <= 0 {
		t.Fatalf("depth must stay positive, got %f", got)
	}
}

func TestNewEchoSounderEmitsSentence(t *testing.T) {
	ch := NewEchoSounder(10*time.Millisecond, 12.0)

	select {
	case sentence := <-ch:
		if !strings.HasPrefix(sentence, "$GPDBT") {
			t.Fatalf("unexpected sentence %q", sentence)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for dbt sentence")
	}
}
