package api

import "sync"

// fetchAllPages fetches page 1, determines the total, then fetches remaining pages
// in parallel. fetchPage returns (items, total, error) for the given page number.
func fetchAllPages[T any](fetchPage func(page int) ([]T, int, error), maxPages int) ([]T, error) {
	firstItems, total, err := fetchPage(1)
	if err != nil {
		return nil, err
	}
	if len(firstItems) == 0 || len(firstItems) >= total {
		return firstItems, nil
	}

	pageSize := len(firstItems)
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages > maxPages {
		totalPages = maxPages
	}

	type pageResult struct {
		items []T
		err   error
	}
	results := make([]pageResult, totalPages-1)
	var wg sync.WaitGroup
	for i := 2; i <= totalPages; i++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			items, _, err := fetchPage(page)
			results[page-2] = pageResult{items: items, err: err}
		}(i)
	}
	wg.Wait()

	all := make([]T, 0, total)
	all = append(all, firstItems...)
	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}
		all = append(all, r.items...)
	}

	return all, nil
}
