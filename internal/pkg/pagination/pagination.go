// Package pagination owns the page contract shared by every list endpoint.
package pagination

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
)

var ErrInvalidRequest = errors.New("invalid page request")

const (
	// DefaultPerPage is used when no allowed size is available.
	DefaultPerPage = 5
	// PerPageMax caps the page size so one request cannot stream a whole table.
	PerPageMax = 100
	// PageHintMax bounds a page number accepted from a redirect hint.
	PageHintMax = 10_000
)

// allowedSizes must match the user_preferences check constraint, so adding a
// size also needs a migration.
var allowedSizes = []int{5, 10, 25, 50, 100}

// SizeOrDefault maps a retired size onto the default instead of failing a list.
func SizeOrDefault(size int) int {
	for _, allowed := range allowedSizes {
		if size == allowed {
			return size
		}
	}
	return DefaultPerPage
}

type PageSizeReader interface {
	PageSize(ctx context.Context, publicID string) (int, error)
}

// ResolvePageSize falls back to the default so a list still renders.
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

// ParseRequest rejects a malformed value; an empty per_page uses the account
// preference.
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
	if perPage > PerPageMax {
		return Query{}, ErrInvalidRequest
	}
	if page > PageHintMax {
		return Query{}, ErrInvalidRequest
	}
	if page == 0 {
		page = 1
	}
	return Query{Page: page, PerPage: perPage}, nil
}

// ParseLimit is for feeds, not pages: absent falls back to the caller's default,
// anything outside [1, PerPageMax] is a malformed link.
func ParseLimit(raw string, fallback int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if fallback < 1 || fallback > PerPageMax {
			return DefaultPerPage, nil
		}
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > PerPageMax {
		return 0, ErrInvalidRequest
	}
	return value, nil
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

// PageHint reports 0 for an absent or malformed page, so a link cannot redirect far.
func PageHint(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	page, err := strconv.Atoi(raw)
	if err != nil || page < 1 || page > PageHintMax {
		return 0
	}
	return page
}

func PageQuery(raw string) string {
	if page := PageHint(raw); page > 1 {
		return "?page=" + strconv.Itoa(page)
	}
	return ""
}

// Window is for rows cheap to fetch whole (roles, sessions, passkeys).
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

func New(page, perPage int) Query {
	if page < 1 {
		page = 1
	}
	// Offset multiplies the hint by the page size, so it is bounded too.
	if page > PageHintMax {
		page = PageHintMax
	}
	if perPage <= 0 || perPage > PerPageMax {
		perPage = DefaultPerPage
	}
	return Query{Page: page, PerPage: perPage}
}

func (q Query) Offset() int {
	page := q.Page
	if page < 1 {
		page = 1
	}
	limit := q.Limit()
	// Saturate rather than overflow: every consumer clamps to the row count.
	if page-1 > (math.MaxInt-1)/limit {
		return math.MaxInt / 2
	}
	return (page - 1) * limit
}

// Cap here too, so a hand-built query cannot ask for an unbounded window.
func (q Query) Limit() int {
	if q.PerPage < 1 {
		return DefaultPerPage
	}
	if q.PerPage > PerPageMax {
		return PerPageMax
	}
	return q.PerPage
}

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

// ClampPage moves a stale link to the last page instead of an empty window.
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
