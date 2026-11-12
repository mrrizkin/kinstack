package audit

import (
	"fmt"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"kinstack/internal/pkg/proptest"
)

// date generates a real calendar date so the properties describe inputs the
// filter accepts rather than valid-looking strings.
func date(year, month, day int) string {
	return fmt.Sprintf(
		"%04d-%02d-%02d",
		1900+proptest.Bound(year, 200),
		1+proptest.Bound(month, 12),
		1+proptest.Bound(day, 28),
	)
}

func TestPropertyEndOfDayIsTheNextMidnight(t *testing.T) {
	property := func(year, month, day int) bool {
		value := date(year, month, day)
		start, err := parseTimeFilter(value, false)
		if err != nil || start == nil {
			return false
		}
		end, err := parseTimeFilter(value, true)
		if err != nil || end == nil {
			return false
		}
		if start.Location() != time.UTC || end.Location() != time.UTC {
			return false
		}
		return start.Hour() == 0 && start.Minute() == 0 && start.Second() == 0 &&
			end.Equal(start.AddDate(0, 0, 1))
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 1000}); err != nil {
		t.Fatal(err)
	}
}

func TestPropertyRFC3339IgnoresEndOfDay(t *testing.T) {
	property := func(offset int) bool {
		instant := time.Date(2024, time.March, 7, 13, 45, 0, 0, time.UTC).
			Add(time.Duration(proptest.Bound(offset, 1<<20)) * time.Second)
		value := instant.Format(time.RFC3339)

		start, err := parseTimeFilter(value, false)
		if err != nil || start == nil || !start.Equal(instant) {
			return false
		}
		end, err := parseTimeFilter(value, true)
		if err != nil || end == nil {
			return false
		}
		return end.Equal(instant)
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 1000}); err != nil {
		t.Fatal(err)
	}
}

// TestPropertyAcceptedTimesAreUTC states the contract every caller relies on:
// anything the filter accepts is an absolute UTC instant, never a local time
// whose meaning depends on the machine.
func TestPropertyAcceptedTimesAreUTC(t *testing.T) {
	property := func(raw string) bool {
		if strings.TrimSpace(raw) == "" {
			// An empty filter is the absence of a bound, not a zero instant.
			instant, err := parseTimeFilter(raw, false)
			return err == nil && instant == nil
		}
		instant, err := parseTimeFilter(raw, proptest.Bound(len(raw), 2) == 1)
		if err != nil {
			return true
		}
		return instant != nil && instant.Location() == time.UTC && !instant.IsZero()
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 2000}); err != nil {
		t.Fatal(err)
	}
}
