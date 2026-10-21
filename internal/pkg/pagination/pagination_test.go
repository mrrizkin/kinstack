package pagination

import "testing"

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
