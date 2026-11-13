package pagination

import (
	"testing"
	"testing/quick"

	"kinstack/internal/pkg/proptest"
)

func normalized(page, perPage int) Query {
	return New(proptest.Bound(page, 1<<20), proptest.Bound(perPage, 1<<13))
}

func TestPropertyWindowStaysInBounds(t *testing.T) {
	property := func(page, perPage, total int) bool {
		query := normalized(page, perPage)
		total = proptest.Bound(total, 1<<20)

		start, end := Window(query, total)
		return start >= 0 && end >= start && end <= total && end-start <= query.Limit()
	}
	if err := quickCheck(property); err != nil {
		t.Fatal(err)
	}
}

func TestPropertyWindowsTileTheResultSet(t *testing.T) {
	property := func(page, perPage, total int) bool {
		query := normalized(page, perPage)
		total = proptest.Bound(total, 1<<16)

		_, firstEnd := Window(query, total)
		next := query
		next.Page = query.Page + 1
		secondStart, _ := Window(next, total)
		if firstEnd >= total {
			return secondStart == total
		}
		return firstEnd == secondStart
	}
	if err := quickCheck(property); err != nil {
		t.Fatal(err)
	}
}

func TestPropertyClampPageLandsOnARealPage(t *testing.T) {
	property := func(page, perPage, total int) bool {
		query := normalized(page, perPage)
		total = proptest.Bound(total, 1<<20)

		clamped := ClampPage(query, total)
		if clamped.Page < 1 {
			return false
		}
		if total == 0 {
			return true
		}
		pages := (total + clamped.PerPage - 1) / clamped.PerPage
		return clamped.Page <= pages
	}
	if err := quickCheck(property); err != nil {
		t.Fatal(err)
	}
}

func TestPropertyPageIsSelfConsistent(t *testing.T) {
	property := func(page, perPage, total int) bool {
		query := normalized(page, perPage)
		total = proptest.Bound(total, 1<<16)

		result := NewPage(query, total)
		if result.Total != total || result.PerPage != query.PerPage || result.Page != query.Page {
			return false
		}
		if result.HasPrev != (query.Page > 1) {
			return false
		}
		if total == 0 {
			return result.From == 0 && result.To == 0 && !result.HasNext
		}
		if result.From > result.To && result.To != 0 {
			return false
		}
		hasNextPage := result.To < total && result.To > 0
		return result.HasNext == hasNextPage
	}
	if err := quickCheck(property); err != nil {
		t.Fatal(err)
	}
}

func quickCheck(property any) error {
	return quick.Check(property, &quick.Config{MaxCount: 2000})
}
