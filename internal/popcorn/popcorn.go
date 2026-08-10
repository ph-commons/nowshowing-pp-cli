// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

// Package popcorn parses popcorn.app per-cinema pages — the secondary showtimes
// source used to cross-check ClickTheCity — and computes a two-source
// confidence signal. The showtimes live in an inline `allShowtimes: {...}` JS
// blob (not standard JSON), so extraction is brace-matched. This is the most
// fragile source; ExtractAllShowtimes has a fixture test so structure drift
// surfaces as a failed test, not a silent empty result. Ported from the
// ~/src/nowshowing fetch_popcorn / popcorn_index / cross_check_badge logic.
package popcorn

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/ph-commons/nowshowing-pp-cli/internal/httpx"
	"github.com/ph-commons/nowshowing-pp-cli/internal/titles"
)

const marker = "allShowtimes: "

// Entry is popcorn.app's view of one movie: its raw title and the set of
// (normalized) showtimes it lists for the requested date.
type Entry struct {
	RawTitle  string   `json:"title"`
	Showtimes []string `json:"showtimes"`
}

// Index maps a normalized title to that movie's popcorn.app entry.
type Index map[string]Entry

// Fetch retrieves a popcorn.app cinema page and indexes its showtimes for the
// given date. A nil Index with a nil error means the page had no parseable
// allShowtimes blob (common for SM-managed cinemas) — the caller should treat
// that as "no cross-check available", not an error.
func Fetch(ctx context.Context, hc *httpx.Client, pageURL, date string) (Index, error) {
	page, err := hc.GetBytes(ctx, pageURL)
	if err != nil {
		return nil, err
	}
	raw, err := ExtractAllShowtimes(page)
	if err != nil {
		return nil, nil // no parseable blob: cross-check simply unavailable
	}
	return BuildIndex(raw, date)
}

// ExtractAllShowtimes finds the inline `allShowtimes: {...}` object in a
// popcorn.app page and returns its raw JSON bytes via brace matching. It errors
// when the marker is absent or the braces do not balance — the signal that the
// page structure changed.
func ExtractAllShowtimes(page []byte) ([]byte, error) {
	s := string(page)
	i := indexOf(s, marker)
	if i < 0 {
		return nil, fmt.Errorf("popcorn: no allShowtimes marker (page structure changed)")
	}
	start := i + len(marker)
	depth := 0
	started := false
	inStr := false
	escaped := false
	for j := start; j < len(s); j++ {
		c := s[j]
		if inStr {
			// Inside a JSON string literal braces are data, not structure.
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{':
			depth++
			started = true
		case '}':
			depth--
			if started && depth == 0 {
				return []byte(s[start : j+1]), nil
			}
		}
	}
	return nil, fmt.Errorf("popcorn: brace matching failed")
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

type pcShowtime struct {
	ShowTime string `json:"ShowTime"`
}

type pcMovie struct {
	MovieName string                  `json:"MovieName"`
	Cinemas   map[string][]pcShowtime `json:"Cinemas"`
}

// BuildIndex decodes the allShowtimes blob and indexes the movies listed for
// the given date, keyed by normalized title.
func BuildIndex(raw []byte, date string) (Index, error) {
	var byDate map[string][]pcMovie
	if err := json.Unmarshal(raw, &byDate); err != nil {
		return nil, fmt.Errorf("popcorn: decoding allShowtimes: %w", err)
	}
	idx := Index{}
	for _, m := range byDate[date] {
		key := titles.Normalize(m.MovieName)
		set := map[string]struct{}{}
		if e, ok := idx[key]; ok {
			for _, t := range e.Showtimes {
				set[t] = struct{}{}
			}
		}
		for _, showtimes := range m.Cinemas {
			for _, st := range showtimes {
				if st.ShowTime != "" {
					set[titles.NormalizeTime(st.ShowTime)] = struct{}{}
				}
			}
		}
		idx[key] = Entry{RawTitle: m.MovieName, Showtimes: sortedTimes(set)}
	}
	return idx, nil
}

func sortedTimes(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		return titles.MinutesSinceMidnight(out[i]) < titles.MinutesSinceMidnight(out[j])
	})
	return out
}

// Confidence classifies how well the two sources agree on a movie's showtimes.
type Confidence string

const (
	Verified Confidence = "verified"  // upcoming showtimes match on both sources
	Partial  Confidence = "partial"   // some upcoming showtimes agree
	Mismatch Confidence = "mismatch"  // both list the movie but upcoming times differ
	CTCOnly  Confidence = "ctc-only"  // popcorn.app has no data for this movie
)

// CrossCheck compares ClickTheCity's showtimes (full day) against popcorn.app's
// (upcoming only) for one movie. popcorn.app drops elapsed showtimes, so only
// the still-upcoming ClickTheCity subset (>= nowMinutes) is compared. Pass a
// nil pcEntry when popcorn has no data for the movie.
type CrossCheck struct {
	Status        Confidence `json:"status"`
	AgreeCount    int        `json:"agree_count"`
	UpcomingTotal int        `json:"upcoming_total"`
	ElapsedCount  int        `json:"elapsed_count,omitempty"`
}

// CheckShowtimes runs the cross-check. nowMinutes is minutes-since-midnight in
// Asia/Manila for "now"; pass a negative value to treat all showtimes as
// upcoming (e.g. when checking a future date).
func CheckShowtimes(ctcTimes []string, pcEntry *Entry, nowMinutes int) CrossCheck {
	if pcEntry == nil || len(pcEntry.Showtimes) == 0 {
		return CrossCheck{Status: CTCOnly}
	}
	pc := toSet(pcEntry.Showtimes)

	upcoming := map[string]struct{}{}
	elapsed := 0
	for _, t := range ctcTimes {
		if nowMinutes < 0 || titles.MinutesSinceMidnight(t) >= nowMinutes {
			upcoming[t] = struct{}{}
		} else {
			elapsed++
		}
	}

	res := CrossCheck{UpcomingTotal: len(upcoming), ElapsedCount: elapsed}
	if setsEqual(upcoming, pc) {
		res.Status = Verified
		res.AgreeCount = len(upcoming)
		return res
	}
	overlap := 0
	union := map[string]struct{}{}
	for t := range upcoming {
		union[t] = struct{}{}
		if _, ok := pc[t]; ok {
			overlap++
		}
	}
	for t := range pc {
		union[t] = struct{}{}
	}
	if overlap > 0 {
		res.Status = Partial
		res.AgreeCount = overlap
		res.UpcomingTotal = len(union)
		return res
	}
	res.Status = Mismatch
	return res
}

func toSet(ss []string) map[string]struct{} {
	m := make(map[string]struct{}, len(ss))
	for _, s := range ss {
		m[s] = struct{}{}
	}
	return m
}

func setsEqual(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}
