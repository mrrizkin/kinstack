// Package proptest holds the helpers shared by property tests.
package proptest

// Bound folds an arbitrary int into [0, span) so a property describes an input
// the parser accepts instead of arithmetic overflow. It is deliberately not
// random: the runner supplies the randomness, this only makes it legal.
func Bound(value, span int) int {
	if span <= 0 {
		panic("proptest: span must be positive")
	}
	bounded := value % span
	if bounded < 0 {
		bounded = -bounded
	}
	return bounded
}
