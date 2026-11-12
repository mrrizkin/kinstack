// Package clock supplies the wall clock that feature code never reads directly.
package clock

import "time"

// System is the production wall clock, in UTC. Code that needs the time takes a
// `now func() time.Time` and receives this at the composition root; passing it
// as a value is what makes a slice replayable under simulation.
//
// This is the only place in the repository that reads the wall clock. The
// determinism gate exempts it by name.
func System() time.Time { return time.Now().UTC() } //kinstack:allow-nondeterminism the one real clock
