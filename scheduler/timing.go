package scheduler

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

// timingEnvVar enables a per-phase time breakdown on stderr. The numbers are
// summed over all goroutines, so they equal wall-clock time for one thread.
// They are used for the Amdahl's law calculations in the benchmark.
const timingEnvVar = "PROJ1_TIMING"

// phaseTimer accumulates the time spent in one phase of the program.
type phaseTimer struct {
	nanos int64
}

func (t *phaseTimer) add(start time.Time) {
	atomic.AddInt64(&t.nanos, int64(time.Since(start)))
}

func (t *phaseTimer) seconds() float64 {
	return time.Duration(atomic.LoadInt64(&t.nanos)).Seconds()
}

var (
	loadTime    phaseTimer // Reading and decoding PNG files
	effectsTime phaseTimer // Applying effects (the part parslices parallelizes)
	saveTime    phaseTimer // Encoding and writing PNG files
)

// reportTimings prints the phase breakdown if timing output was requested.
func reportTimings() {
	if os.Getenv(timingEnvVar) == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "load=%.4f effects=%.4f save=%.4f\n",
		loadTime.seconds(), effectsTime.seconds(), saveTime.seconds())
}
