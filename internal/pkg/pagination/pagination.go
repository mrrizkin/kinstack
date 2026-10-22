// Package pagination owns the page-request and page-result contract shared by
// every list endpoint.
package pagination

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

// ErrInvalidRequest reports a malformed page or per_page value.
var ErrInvalidRequest = errors.New("invalid page request")

const (
	// DefaultPerPage is used when no allowed size is available.
	DefaultPerPage = 5
	// MaxPerPage caps the page size so one request cannot stream a whole table.
	MaxPerPage = 100
	// MaxPageHint bounds a page number accepted from a redirect hint.
	MaxPageHint = 10_000
)

// allowedSizes must match the user_preferences check constraint, so adding a
// size also needs a migration.
var allowedSizes = []int{5, 10, 25, 50, 100}

// SizeOrDefault maps a size written by an older release onto the current
// default instead of breaking a list.
func SizeOrDefault(size int) int {
	for _, allowed := range allowedSizes {
		if size == allowed {
			return size
		}
	}
	return DefaultPerPage
}

// PageSizeReader keeps list slices from importing the profile slice.
type PageSizeReader interface {
	PageSize(ctx context.Context, publicID string) (int, error)
}

// ResolvePageSize falls back to the default when the preference cannot be read,
// so a list still renders.
func ResolvePageSize(ctx context.Context, reader PageSizeReader, publicID string) int {
	if reader == nil || publicID == "" {
		return DefaultPerPage
	}
	size, err := reader.PageSize(ctx, publicID)
	if err != nil {
		return DefaultPerPage
	}
	return SizeOrDefault(size)
}

// ParseRequest rejects a malformed value so a hand-written link cannot silently
// change the window; an empty per_page falls back to the account preference.
func ParseRequest(rawPage, rawPerPage string, fallbackPerPage int) (Query, error) {
	page, err := parsePositive(rawPage)
	if err != nil {
		return Query{}, err
	}
	perPage, err := parsePositive(rawPerPage)
	if err != nil {
		return Query{}, err
	}
	if perPage == 0 {
		perPage = SizeOrDefault(fallbackPerPage)
	}
	if perPage > MaxPerPage {
		return Query{}, ErrInvalidRequest
	}
	if page == 0 {
		page = 1
	}
	return Query{Page: page, PerPage: perPage}, nil
}

func parsePositive(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, ErrInvalidRequest
	}
	return value, nil
}

// PageHint reports 0 for an absent or malformed page, so a hand-written link
// cannot redirect far.
func PageHint(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	page, err := strconv.Atoi(raw)
	if err != nil || page < 1 || page > MaxPageHint {
		return 0
	}
	return page
}

// PageQuery renders a page hint as a query suffix, empty on page one.
func PageQuery(raw string) string {
	if page := PageHint(raw); page > 1 {
		return "?page=" + strconv.Itoa(page)
	}
	return ""
}

// Window is for surfaces whose rows are cheap to fetch whole (roles, sessions,
// passkeys), so the page is cut from a loaded slice.
func Window(query Query, total int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	start := query.Offset()
	if start > total {
		start = total
	}
	end := start + query.PerPage
	if end > total {
		end = total
	}
	return start, end
}

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
	page := q.Page
	if page < 1 {
		page = 1
	}
	return (page - 1) * q.Limit()
}

// Limit is capped here as well as in the parsers, so a hand-built query cannot
// ask the database for an unbounded window.
func (q Query) Limit() int {
	if q.PerPage < 1 {
		return DefaultPerPage
	}
	if q.PerPage > MaxPerPage {
		return MaxPerPage
	}
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
