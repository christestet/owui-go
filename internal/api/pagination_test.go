package api

import "testing"

// makeFetcher returns a fetchPage func over a fixed dataset split into pages of
// pageSize, reporting the given total (which may be inaccurate on purpose).
func makeFetcher(data []int, pageSize, reportedTotal int) func(page int) ([]int, int, error) {
	return func(page int) ([]int, int, error) {
		start := (page - 1) * pageSize
		if start >= len(data) {
			return []int{}, reportedTotal, nil
		}
		end := start + pageSize
		if end > len(data) {
			end = len(data)
		}
		return append([]int(nil), data[start:end]...), reportedTotal, nil
	}
}

func TestFetchAllPages_TrustsAccurateTotal(t *testing.T) {
	data := []int{1, 2, 3, 4, 5}
	got, err := fetchAllPages(makeFetcher(data, 2, len(data)), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(data) {
		t.Fatalf("expected %d items, got %d", len(data), len(got))
	}
}

func TestFetchAllPages_ZeroTotalFallsBackToProbing(t *testing.T) {
	// total reported as 0 even though there are three full pages of items.
	data := []int{1, 2, 3, 4, 5, 6, 7}
	got, err := fetchAllPages(makeFetcher(data, 3, 0), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(data) {
		t.Fatalf("expected %d items despite total=0, got %d", len(data), len(got))
	}
}

func TestFetchAllPages_EmptyFirstPage(t *testing.T) {
	got, err := fetchAllPages(makeFetcher(nil, 3, 0), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 items, got %d", len(got))
	}
}

func TestFetchAllPages_RespectsMaxPages(t *testing.T) {
	// Exactly-full pages with total=0 would otherwise probe forever; maxPages caps it.
	data := []int{1, 2, 3, 4, 5, 6}
	got, err := fetchAllPages(makeFetcher(data, 2, 0), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("expected 4 items (2 pages capped), got %d", len(got))
	}
}
