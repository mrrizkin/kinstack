package audit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/platform/i18n"
)

const (
	defaultPerPage  = 25
	maxPerPage      = 100
	maxSearchLength = 200
)

var allowedActions = map[string]struct{}{
	"http.read":      {},
	"http.mutate":    {},
	"background.job": {},
}

var allowedOutcomes = map[string]struct{}{
	"success": {},
	"failure": {},
}

// ServiceDependency contains the persistence boundary used by the service.
type ServiceDependency struct {
	Repository Repository
}

// Service owns the audit viewer business rules.
type Service struct {
	repository Repository
}

// NewService constructs an audit service around its repository.
func NewService(dependency ServiceDependency) *Service {
	assert.Assert(dependency.Repository != nil, "audit repository is required")
	return &Service{repository: dependency.Repository}
}

// List returns one page of audit entries for the supplied filter.
func (s *Service) List(ctx context.Context, request FilterRequest) (PageProps, error) {
	query, err := normalizeFilter(request)
	if err != nil {
		return PageProps{}, err
	}
	page, err := s.repository.List(ctx, query)
	if err != nil {
		return PageProps{}, err
	}

	entries := make([]EntryDTO, 0, len(page.Entries))
	for _, entry := range page.Entries {
		entries = append(entries, toEntryDTO(entry))
	}
	pages := 0
	if query.Limit > 0 {
		pages = (page.Total + query.Limit - 1) / query.Limit
	}
	return PageProps{
		Entries: entries,
		Filters: Filters{
			ActorKey: query.ActorKey,
			Action:   query.Action,
			Outcome:  query.Outcome,
			Search:   query.Search,
			From:     formatFilterTime(query.From),
			To:       formatFilterTime(query.To),
		},
		Total:   page.Total,
		Page:    query.Offset/query.Limit + 1,
		PerPage: query.Limit,
		Pages:   pages,
	}, nil
}

func normalizeFilter(request FilterRequest) (Query, error) {
	query := Query{
		ActorKey: strings.TrimSpace(request.ActorKey),
		Action:   strings.TrimSpace(request.Action),
		Outcome:  strings.TrimSpace(request.Outcome),
		Search:   strings.TrimSpace(request.Search),
	}
	if query.Action != "" {
		if _, ok := allowedActions[query.Action]; !ok {
			return Query{}, fmt.Errorf("%w: %s", ErrInvalidInput, i18n.T("audit.unknown_action"))
		}
	}
	if query.Outcome != "" {
		if _, ok := allowedOutcomes[query.Outcome]; !ok {
			return Query{}, fmt.Errorf("%w: %s", ErrInvalidInput, i18n.T("audit.unknown_outcome"))
		}
	}
	if len(query.Search) > maxSearchLength {
		return Query{}, fmt.Errorf("%w: %s", ErrInvalidInput, i18n.T("audit.search_too_long", maxSearchLength))
	}

	perPage := request.PerPage
	switch {
	case perPage <= 0:
		perPage = defaultPerPage
	case perPage > maxPerPage:
		perPage = maxPerPage
	}
	page := request.Page
	if page < 1 {
		page = 1
	}
	query.Limit = perPage
	query.Offset = (page - 1) * perPage

	from, err := parseTimeFilter(request.From, false)
	if err != nil {
		return Query{}, err
	}
	to, err := parseTimeFilter(request.To, true)
	if err != nil {
		return Query{}, err
	}
	query.From = from
	query.To = to
	return query, nil
}

func parseTimeFilter(value string, endOfDay bool) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		parsed, err := time.Parse(layout, value)
		if err != nil {
			continue
		}
		if layout == "2006-01-02" && endOfDay {
			parsed = parsed.AddDate(0, 0, 1)
		}
		utc := parsed.UTC()
		return &utc, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrInvalidInput, i18n.T("audit.invalid_date"))
}

func formatFilterTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func toEntryDTO(entry Entry) EntryDTO {
	metadata := entry.Metadata
	if len(metadata) == 0 {
		metadata = nil
	}
	return EntryDTO{
		ID:         entry.ID,
		OccurredAt: entry.OccurredAt.UTC().Format(time.RFC3339),
		RequestID:  entry.RequestID,
		ActorKey:   entry.ActorKey,
		Action:     entry.Action,
		Resource:   entry.Resource,
		Method:     entry.Method,
		Status:     entry.Status,
		Outcome:    entry.Outcome,
		Metadata:   metadata,
	}
}
