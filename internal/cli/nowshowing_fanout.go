// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored concurrent theater fetch shared by now-playing and search.
// Separate file (no generated header) so regen preserves it.

package cli

import (
	"context"
	"sync"

	"github.com/ph-commons/nowshowing-pp-cli/internal/ctc"
	"github.com/ph-commons/nowshowing-pp-cli/internal/httpx"
	"github.com/ph-commons/nowshowing-pp-cli/internal/registry"
)

// theaterFetch is the outcome of fetching one theater's ClickTheCity schedule.
type theaterFetch struct {
	Theater registry.Theater
	Result  *ctc.Result
	Err     error
}

// fetchTheaters retrieves the ClickTheCity schedule for each theater
// concurrently, sharing one rate-limited client so outbound requests stay
// paced. Results are returned in the input order; each entry carries either a
// Result or an Err (never both), so callers can account for partial failures.
func fetchTheaters(ctx context.Context, hc *httpx.Client, theaters []registry.Theater, date string) []theaterFetch {
	out := make([]theaterFetch, len(theaters))
	var wg sync.WaitGroup
	for i, t := range theaters {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i] = theaterFetch{Theater: t}
			res, err := ctc.Fetch(ctx, hc, t.Slug, date)
			if err != nil {
				out[i].Err = err
				return
			}
			out[i].Result = res
		}()
	}
	wg.Wait()
	return out
}

// selectTheaters resolves the --theater filter. An empty filter returns the
// full registry; a non-empty filter returns just that theater (or an empty
// slice with ok=false when the slug is unknown).
func selectTheaters(slug string) ([]registry.Theater, bool) {
	if slug == "" {
		return registry.Theaters, true
	}
	if t, ok := registry.BySlug(slug); ok {
		return []registry.Theater{t}, true
	}
	return nil, false
}
