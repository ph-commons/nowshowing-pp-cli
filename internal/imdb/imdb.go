// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

// Package imdb resolves a movie title to its IMDb title id via IMDb's public
// suggestion endpoint. A remake or re-release exact-title-collides with a
// decades-old original, so genuine ambiguity is 2+ recent exact-title matches;
// a single recent match wins even when older namesakes exist. Ported from the
// ~/src/nowshowing imdb_lookup logic.
package imdb

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"nowshowing-pp-cli/internal/httpx"
	"nowshowing-pp-cli/internal/titles"
)

const suggestionBase = "https://v3.sg.media-imdb.com/suggestion/"

var (
	stripPunct = regexp.MustCompile(`[^\w\s]`)
	wsRun      = regexp.MustCompile(`\s+`)
)

// Result is a resolved IMDb match.
type Result struct {
	ID        string `json:"imdb_id"`
	URL       string `json:"url"`
	Ambiguous bool   `json:"ambiguous"`
}

type suggestion struct {
	ID    string      `json:"id"`
	Label string      `json:"l"`
	QID   string      `json:"qid"`
	Year  json.Number `json:"y"`
}

type suggestResponse struct {
	D []suggestion `json:"d"`
}

// queryFor builds the IMDb suggestion path segment: lowercase, punctuation
// stripped, spaces to underscores.
func queryFor(title string) string {
	q := strings.TrimSpace(stripPunct.ReplaceAllString(strings.ToLower(title), ""))
	q = wsRun.ReplaceAllString(q, "_")
	return q
}

// Lookup resolves title to an IMDb match, or nil when no exact-title movie is
// found. yearHint is the current year (e.g. from the schedule date) used to
// distinguish a current release from an older namesake.
func Lookup(ctx context.Context, hc *httpx.Client, title string, yearHint int) (*Result, error) {
	q := queryFor(title)
	if q == "" {
		return nil, nil
	}
	u := suggestionBase + string(q[0]) + "/" + q + ".json"
	data, err := hc.GetBytes(ctx, u)
	if err != nil {
		return nil, err
	}
	return parseSuggestions(data, title, yearHint)
}

// parseSuggestions filters the suggestion payload to exact-title movie matches
// and applies the recent-year disambiguation rule.
func parseSuggestions(data []byte, title string, yearHint int) (*Result, error) {
	var resp suggestResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("imdb: decoding suggestions: %w", err)
	}
	target := titles.Normalize(title)
	var exact []suggestion
	for _, s := range resp.D {
		if s.QID == "movie" && titles.Normalize(s.Label) == target {
			exact = append(exact, s)
		}
	}
	if len(exact) == 0 {
		return nil, nil
	}
	var recent []suggestion
	for _, s := range exact {
		if y, err := s.Year.Int64(); err == nil && int(y) >= yearHint-1 {
			recent = append(recent, s)
		}
	}
	if len(recent) == 1 {
		return &Result{ID: recent[0].ID, URL: titleURL(recent[0].ID), Ambiguous: false}, nil
	}
	// Fall back to the first exact match. It is only genuinely ambiguous when
	// more than one exact match competes; a lone (older) namesake is not.
	return &Result{ID: exact[0].ID, URL: titleURL(exact[0].ID), Ambiguous: len(exact) > 1}, nil
}

func titleURL(id string) string {
	return "https://www.imdb.com/title/" + id + "/"
}
