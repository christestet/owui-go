package api

import "sync"

// fetchAllPages fetches page 1 to learn the total, then fetches the remaining
// pages in parallel. fetchPage returns (items, total, error) for a page number.
// At most maxPages are fetched.
//
// The server reports total as part of the list contract and it is trusted when
// positive. Some endpoints, however, return total <= 0 while still returning a
// full page of items; trusting that value would silently drop later pages, so
// in that case we fall back to paging sequentially until a short/empty page
// confirms the end.
func fetchAllPages[T any](fetchPage func(page int) ([]T, int, error), maxPages int) ([]T, error) {
	firstItems, total, err := fetchPage(1)
	if err != nil {
		return nil, err
	}

	pageSize := len(firstItems)
	// An empty first page means there is nothing more to fetch.
	if pageSize == 0 {
		return firstItems, nil
	}

	// total is unusable (some endpoints don't populate it): probe forward until
	// a page comes back shorter than the first, which marks the end.
	if total <= 0 {
		all := make([]T, 0, pageSize*2)
		all = append(all, firstItems...)
		for page := 2; page <= maxPages; page++ {
			items, _, err := fetchPage(page)
			if err != nil {
				return nil, err
			}
			all = append(all, items...)
			if len(items) < pageSize {
				break
			}
		}
		return all, nil
	}

	// Page 1 already holds everything the server reported.
	if len(firstItems) >= total {
		return firstItems, nil
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages > maxPages {
		totalPages = maxPages
	}

	rest, err := fetchPagesParallel(fetchPage, 2, totalPages)
	if err != nil {
		return nil, err
	}

	capacity := len(firstItems)
	for _, items := range rest {
		capacity += len(items)
	}
	all := make([]T, 0, capacity)
	all = append(all, firstItems...)
	for _, items := range rest {
		all = append(all, items...)
	}
	return all, nil
}

// fetchPagesParallel fetches pages [from, to] concurrently and returns their
// item slices in page order.
func fetchPagesParallel[T any](fetchPage func(page int) ([]T, int, error), from, to int) ([][]T, error) {
	type pageResult struct {
		items []T
		err   error
	}
	results := make([]pageResult, to-from+1)
	var wg sync.WaitGroup
	for page := from; page <= to; page++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			items, _, err := fetchPage(page)
			results[page-from] = pageResult{items: items, err: err}
		}(page)
	}
	wg.Wait()

	pages := make([][]T, 0, len(results))
	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}
		pages = append(pages, r.items)
	}
	return pages, nil
}
