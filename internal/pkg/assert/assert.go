// Package assert provides small programmer-error guards used during wiring and orchestration.
package assert

import "fmt"

func Assert(condition bool, message string) {
	if !condition {
		panic(message)
	}
}

func Assertf(condition bool, format string, args ...any) {
	if !condition {
		panic(fmt.Sprintf(format, args...))
	}
}

func NoError(err error) {
	if err != nil {
		panic(err)
	}
}

func OK[T any](value T, err error) T {
	NoError(err)
	return value
}
