package audit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/pagination"
	"kinstack/internal/platform/i18n"
)

const (
	maxPerPage      = pagination.MaxPerPage
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

type ServiceDependency struct {
	Repository Repository
}

type Service struct {
	repository Repository
}

func NewService(dependency ServiceDependency) *Service {
	assert.Assert(dependency.Repository != nil, "audit repository is required")
	return &Service{repository: dependency.Repository}
}

// List returns one page of audit entries for the supplied filter.
func (s *Service) List(ctx context.Context, request FilterRequest) (PageProps, error) {
	filter, err := normalizeFilter(request)
	if err != nil {
		return PageProps{}, err
	}
	page, err := s.repository.List(ctx, filter.Query)
	if err != nil {
		return PageProps{}, err
	}

	entries := make([]EntryDTO, 0, len(page.Entries))
	for _, entry := range page.Entries {
		entries = append(entries, toEntryDTO(entry))
	}
	query := filter.Query
	return PageProps{
		Entries: entries,
		Filters: Filters{
			ActorKey: query.ActorKey,
			Action:   query.Action,
			Outcome:  query.Outcome,
			Search:   query.Search,
			From:     filter.From,
			To:       filter.To,
		},
		Page: pagination.NewPage(query.Query, page.Total),
	}, nil
}

type normalizedFilter struct {
	Query Query
	From  string
	To    string
}

func normalizeFilter(request FilterRequest) (normalizedFilter, error) {
	query := Query{
		ActorKey: strings.TrimSpace(request.ActorKey),
		Action:   strings.TrimSpace(request.Action),
		Outcome:  strings.TrimSpace(request.Outcome),
		Search:   strings.TrimSpace(request.Search),
	}
	if query.Action != "" {
		if _, ok := allowedActions[query.Action]; !ok {
			return normalizedFilter{}, fmt.Errorf(
				"%w: %s",
				ErrInvalidInput,
				i18n.T("audit.unknown_action"),
			)
		}
	}
	if query.Outcome != "" {
		if _, ok := allowedOutcomes[query.Outcome]; !ok {
			return normalizedFilter{}, fmt.Errorf(
				"%w: %s",
				ErrInvalidInput,
				i18n.T("audit.unknown_outcome"),
			)
		}
	}
	if len(query.Search) > maxSearchLength {
		return normalizedFilter{}, fmt.Errorf(
			"%w: %s",
			ErrInvalidInput,
			i18n.T("audit.search_too_long", maxSearchLength),
		)
	}

	perPage := request.PerPage
	if perPage <= 0 {
		perPage = pagination.DefaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	page := request.Page
	if page < 1 {
		page = 1
	}
	query.Query = pagination.Query{Page: page, PerPage: perPage}

	from, err := parseTimeFilter(request.From, false)
	if err != nil {
		return normalizedFilter{}, err
	}
	to, err := parseTimeFilter(request.To, true)
	if err != nil {
		return normalizedFilter{}, err
	}
	query.From = from
	query.To = to
	return normalizedFilter{
		Query: query,
		From:  strings.TrimSpace(request.From),
		To:    strings.TrimSpace(request.To),
	}, nil
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
