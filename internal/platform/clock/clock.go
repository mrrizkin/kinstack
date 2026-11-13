// Package clock supplies the wall clock that feature code never reads directly.
package clock

import "time"

// System is the production wall clock, in UTC. This is the only place in the
// repository that reads it; slices take `now func() time.Time` instead, which is
// what makes them replayable under simulation.
func System() time.Time { return time.Now().UTC() } //kinstack:allow-nondeterminism the one real clock
