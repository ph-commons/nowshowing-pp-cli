// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored shared helpers for the nowshowing novel commands. Separate file
// (no generated header) so regen preserves it.

package cli

import (
	"errors"
	"strings"
	"time"

	"nowshowing-pp-cli/internal/cliutil"
	"nowshowing-pp-cli/internal/httpx"
	"nowshowing-pp-cli/internal/titles"
)

// fetchFailure records one theater that failed to fetch, so partial results are
// honest about the gap instead of silently dropping the theater.
type fetchFailure struct {
	Theater string `json:"theater"`
	Error   string `json:"error"`
}

// manilaLoc returns Asia/Manila, falling back to a fixed +08:00 zone if the
// tzdata is unavailable (the Philippines has observed no DST since 1978).
func manilaLoc() *time.Location {
	if loc, err := time.LoadLocation("Asia/Manila"); err == nil {
		return loc
	}
	return time.FixedZone("PHT", 8*60*60)
}

// todayManila returns today's date in Asia/Manila as YYYY-MM-DD.
func todayManila() string {
	return time.Now().In(manilaLoc()).Format("2006-01-02")
}

// nowMinutesFor returns minutes-since-midnight in Asia/Manila for cross-check
// elapsed-time filtering when date is today; for any other date it returns -1,
// meaning "treat every showtime as upcoming".
func nowMinutesFor(date string) int {
	now := time.Now().In(manilaLoc())
	if date != now.Format("2006-01-02") {
		return -1
	}
	return now.Hour()*60 + now.Minute()
}

// newSourceClient builds the shared rate-limited GET client used by the source
// packages (ctc, popcorn, imdb).
func newSourceClient() *httpx.Client {
	return httpx.New()
}

// asRateLimit reports whether err is (or wraps) a *cliutil.RateLimitError,
// binding target when so. A throttle must surface as a hard error, never as an
// empty result.
func asRateLimit(err error, target **cliutil.RateLimitError) bool {
	return errors.As(err, target)
}

// displayTheaterName prefers ClickTheCity's own theater name from the response,
// falling back to the registry display name when the fetch failed or was empty.
func displayTheaterName(f theaterFetch) string {
	if f.Result != nil && f.Result.TheaterName != "" {
		return f.Result.TheaterName
	}
	return f.Theater.Name()
}

// joinTimes renders a showtime slice as a comma-separated string for tables.
func joinTimes(ts []string) string {
	return strings.Join(ts, ", ")
}

// titleMatches reports whether a movie's normalized title contains the
// normalized query, so "moana" matches "Disney: Moana 2".
func titleMatches(movieTitle, query string) bool {
	q := titles.Normalize(query)
	if q == "" {
		return false
	}
	return strings.Contains(titles.Normalize(movieTitle), q)
}
