package notifications

import "testing"

// TestTransitionStatusIsTheOnlyWayToChangeState pins the two-state machine: read
// is terminal, so no event can walk a notification backwards.
func TestTransitionStatusIsTheOnlyWayToChangeState(t *testing.T) {
	cases := []struct {
		name    string
		from    Status
		event   Event
		want    Status
		wantErr bool
	}{
		{"mark read moves forward", StatusUnread, EventMarkRead, StatusRead, false},
		{"mark all read moves forward", StatusUnread, EventMarkAllRead, StatusRead, false},
		{"read is terminal", StatusRead, EventMarkRead, StatusRead, false},
		{"anything else is not a state", Status("deleted"), EventMarkRead, "", true},
		{"an unknown event has no transition", StatusUnread, Event("rewind"), "", true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := transitionStatus(testCase.from, testCase.event)
			if testCase.wantErr {
				if err == nil {
					t.Fatalf(
						"transition %q/%q = %q, want an error",
						testCase.from,
						testCase.event,
						got,
					)
				}
				return
			}
			if err != nil || got != testCase.want {
				t.Fatalf(
					"transition %q/%q = %q, %v; want %q",
					testCase.from,
					testCase.event,
					got,
					err,
					testCase.want,
				)
			}
		})
	}
}
