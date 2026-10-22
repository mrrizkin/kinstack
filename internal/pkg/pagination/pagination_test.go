package pagination

import (
	"errors"
	"testing"
)

func TestNewNormalizesBounds(t *testing.T) {
	cases := []struct {
		name    string
		page    int
		perPage int
		want    Query
	}{
		{
			name:    "default when zero",
			page:    0,
			perPage: 0,
			want:    Query{Page: 1, PerPage: DefaultPerPage},
		},
		{name: "negative page", page: -3, perPage: 10, want: Query{Page: 1, PerPage: 10}},
		{
			name:    "oversized per page",
			page:    2,
			perPage: MaxPerPage + 1,
			want:    Query{Page: 2, PerPage: DefaultPerPage},
		},
		{name: "valid", page: 4, perPage: 50, want: Query{Page: 4, PerPage: 50}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := New(test.page, test.perPage); got != test.want {
				t.Fatalf("New(%d, %d) = %#v, want %#v", test.page, test.perPage, got, test.want)
			}
		})
	}
}

func TestSizeOrDefault(t *testing.T) {
	for _, size := range allowedSizes {
		if got := SizeOrDefault(size); got != size {
			t.Fatalf("SizeOrDefault(%d) = %d, want %d", size, got, size)
		}
	}
	for _, size := range []int{0, -5, 1, 7, MaxPerPage + 1} {
		if got := SizeOrDefault(size); got != DefaultPerPage {
			t.Fatalf("SizeOrDefault(%d) = %d, want %d", size, got, DefaultPerPage)
		}
	}
}

func TestQueryOffsetAndLimit(t *testing.T) {
	query := New(3, 25)
	if query.Offset() != 50 || query.Limit() != 25 {
		t.Fatalf("unexpected offset/limit: %d/%d", query.Offset(), query.Limit())
	}
}

func TestNewPageWindows(t *testing.T) {
	cases := []struct {
		name  string
		query Query
		total int
		want  Page
	}{
		{
			name:  "empty",
			query: New(1, 10),
			total: 0,
			want: Page{
				Page:       1,
				PerPage:    10,
				Total:      0,
				TotalPages: 0,
				From:       0,
				To:         0,
				HasPrev:    false,
				HasNext:    false,
			},
		},
		{
			name:  "first page",
			query: New(1, 10),
			total: 25,
			want: Page{
				Page:       1,
				PerPage:    10,
				Total:      25,
				TotalPages: 3,
				From:       1,
				To:         10,
				HasPrev:    false,
				HasNext:    true,
			},
		},
		{
			name:  "last page partial",
			query: New(3, 10),
			total: 25,
			want: Page{
				Page:       3,
				PerPage:    10,
				Total:      25,
				TotalPages: 3,
				From:       21,
				To:         25,
				HasPrev:    true,
				HasNext:    false,
			},
		},
		{
			name:  "beyond last page",
			query: New(9, 10),
			total: 25,
			want: Page{
				Page:       9,
				PerPage:    10,
				Total:      25,
				TotalPages: 3,
				From:       0,
				To:         0,
				HasPrev:    true,
				HasNext:    false,
			},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := NewPage(test.query, test.total); got != test.want {
				t.Fatalf("NewPage(%#v, %d) = %#v, want %#v", test.query, test.total, got, test.want)
			}
		})
	}
}

func TestClampPageMovesPastEndRequestsToLastPage(t *testing.T) {
	query := Query{Page: 99, PerPage: 2}
	if got := ClampPage(query, 5); got.Page != 3 {
		t.Fatalf("clamped page = %d, want 3", got.Page)
	}
	if got := ClampPage(query, 0); got.Page != 99 {
		t.Fatalf("empty result must keep the request, got %d", got.Page)
	}
	if got := ClampPage(Query{Page: 1, PerPage: 2}, 5); got.Page != 1 {
		t.Fatalf("first page must stay, got %d", got.Page)
	}
}

func TestParseRequestFallsBackAndRejects(t *testing.T) {
	cases := []struct {
		name     string
		page     string
		perPage  string
		fallback int
		want     Query
		wantErr  bool
	}{
		{name: "preference fallback", fallback: 25, want: New(1, 25)},
		{name: "explicit values", page: "3", perPage: "10", fallback: 25, want: New(3, 10)},
		{name: "empty preference uses default", want: New(1, DefaultPerPage)},
		{name: "page zero", page: "0", fallback: 5, wantErr: true},
		{name: "negative per page", perPage: "-1", fallback: 5, wantErr: true},
		{name: "not a number", page: "two", fallback: 5, wantErr: true},
		{name: "above the hard cap", perPage: "1000", fallback: 5, wantErr: true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseRequest(test.page, test.perPage, test.fallback)
			if test.wantErr {
				if !errors.Is(err, ErrInvalidRequest) {
					t.Fatalf("expected ErrInvalidRequest, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != test.want {
				t.Fatalf("ParseRequest = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestWindowClampsToAvailableRows(t *testing.T) {
	cases := []struct {
		name      string
		query     Query
		total     int
		wantStart int
		wantEnd   int
	}{
		{name: "empty list", query: New(1, 10), total: 0, wantStart: 0, wantEnd: 0},
		{name: "first page", query: New(1, 10), total: 25, wantStart: 0, wantEnd: 10},
		{name: "last partial page", query: New(3, 10), total: 25, wantStart: 20, wantEnd: 25},
		{name: "past the end", query: New(9, 10), total: 25, wantStart: 25, wantEnd: 25},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			start, end := Window(test.query, test.total)
			if start != test.wantStart || end != test.wantEnd {
				t.Fatalf(
					"Window = (%d, %d), want (%d, %d)",
					start,
					end,
					test.wantStart,
					test.wantEnd,
				)
			}
		})
	}
}

func TestLimitCapsHandBuiltQueries(t *testing.T) {
	if got := (Query{Page: 1, PerPage: 1000}).Limit(); got != MaxPerPage {
		t.Fatalf("limit = %d, want %d", got, MaxPerPage)
	}
	if got := (Query{Page: 1, PerPage: 0}).Limit(); got != DefaultPerPage {
		t.Fatalf("limit = %d, want %d", got, DefaultPerPage)
	}
	if got := (Query{Page: 2, PerPage: 1000}).Offset(); got != MaxPerPage {
		t.Fatalf("offset = %d, want %d", got, MaxPerPage)
	}
}

func TestPageHintAndQueryIgnoreLowAndMalformedValues(t *testing.T) {
	cases := []struct {
		raw       string
		wantPage  int
		wantQuery string
	}{
		{raw: "", wantPage: 0, wantQuery: ""},
		{raw: "1", wantPage: 1, wantQuery: ""},
		{raw: "3", wantPage: 3, wantQuery: "?page=3"},
		{raw: "0", wantPage: 0, wantQuery: ""},
		{raw: "-2", wantPage: 0, wantQuery: ""},
		{raw: "many", wantPage: 0, wantQuery: ""},
		{raw: "100000", wantPage: 0, wantQuery: ""},
	}
	for _, test := range cases {
		if got := PageHint(test.raw); got != test.wantPage {
			t.Fatalf("PageHint(%q) = %d, want %d", test.raw, got, test.wantPage)
		}
		if got := PageQuery(test.raw); got != test.wantQuery {
			t.Fatalf("PageQuery(%q) = %q, want %q", test.raw, got, test.wantQuery)
		}
	}
}
