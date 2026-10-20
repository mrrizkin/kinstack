// Package pagination owns the shared page-request/page-result contract used by
// every list endpoint. Features keep their own filters; this package only
// normalizes page numbers and reports the visible window so the UI can render
// "menampilkan X–Y dari Z".
package pagination

import ()

const (
	// DefaultPerPage is used when a request omits or sends an invalid page size.
	DefaultPerPage = 25
	// MaxPerPage caps the page size so one request cannot stream a whole table.
	MaxPerPage = 100
)

// Query is a normalized page request.
type Query struct {
	Page    int
	PerPage int
}

// New normalizes raw page/per-page values into a safe request.
func New(page, perPage int) Query {
	if page < 1 {
		page = 1
	}
	if perPage <= 0 || perPage > MaxPerPage {
		perPage = DefaultPerPage
	}
	return Query{Page: page, PerPage: perPage}
}

// Offset is the number of rows to skip for the current page.
func (q Query) Offset() int {
	return (q.Page - 1) * q.PerPage
}

// Limit is the number of rows to fetch for the current page.
func (q Query) Limit() int {
	return q.PerPage
}

// Page is the serialized page window shared with the frontend.
type Page struct {
	Page       int  `json:"page"`
	PerPage    int  `json:"per_page"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	From       int  `json:"from"`
	To         int  `json:"to"`
	HasPrev    bool `json:"has_prev"`
	HasNext    bool `json:"has_next"`
}

// ClampPage moves a request that points past the end of the result set to the
// last page, so a stale link shows rows instead of an empty window.
func ClampPage(query Query, total int) Query {
	if total <= 0 || query.PerPage <= 0 {
		return query
	}
	pages := (total + query.PerPage - 1) / query.PerPage
	if pages < 1 {
		pages = 1
	}
	if query.Page > pages {
		query.Page = pages
	}
	return query
}

// NewPage derives the display window for one query and total row count.
func NewPage(query Query, total int) Page {
	if total < 0 {
		total = 0
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + query.PerPage - 1) / query.PerPage
	}

	from, to := 0, 0
	if total > 0 {
		from = query.Offset() + 1
		to = min(query.Offset()+query.PerPage, total)
		if from > total {
			from, to = 0, 0
		}
	}

	return Page{
		Page:       query.Page,
		PerPage:    query.PerPage,
		Total:      total,
		TotalPages: totalPages,
		From:       from,
		To:         to,
		HasPrev:    query.Page > 1,
		HasNext:    query.Page < totalPages,
	}
}
