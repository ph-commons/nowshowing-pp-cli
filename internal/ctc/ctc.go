// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

// Package ctc fetches and parses ClickTheCity's per-theater schedule JSON, the
// primary showtimes source. It joins now_showing (movie metadata) to schedules
// (per-screen showtime lists) on movieId, grouping by normalized title. Ported
// from the ~/src/nowshowing ctc_index logic.
package ctc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/ph-commons/nowshowing-pp-cli/internal/httpx"
	"github.com/ph-commons/nowshowing-pp-cli/internal/titles"
)

const apiBase = "https://www.clickthecity.com/api/movies/theater/"

// Movie is the now_showing metadata for one film.
type Movie struct {
	MovieID int    `json:"movieId"`
	Title   string `json:"title"`
	Rating  string `json:"mtrcb_rating"`
	Runtime string `json:"running_time"`
	In3D    bool   `json:"in3d"`
}

type schedule struct {
	MovieID     int      `json:"movieId"`
	Date        string   `json:"date"`
	TheaterName string   `json:"theaterName"`
	Showtimes   []string `json:"showtimes"`
}

type theaterInfo struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Slug    string `json:"slug"`
}

type apiResponse struct {
	Status     bool        `json:"status"`
	Theater    theaterInfo `json:"theater"`
	NowShowing []Movie     `json:"now_showing"`
	Schedules  []schedule  `json:"schedules"`
}

// CinemaRow is one screen/room's showtimes for a movie.
type CinemaRow struct {
	Cinema    string   `json:"cinema"`
	Showtimes []string `json:"showtimes"`
}

// Entry is one movie at a theater with its per-screen rows and the union of
// its (normalized) showtimes.
type Entry struct {
	Key        string      `json:"-"`
	RawTitle   string      `json:"title"`
	Rating     string      `json:"rating,omitempty"`
	Runtime    string      `json:"runtime,omitempty"`
	In3D       bool        `json:"in_3d,omitempty"`
	CinemaRows []CinemaRow `json:"cinema_rows"`
	Showtimes  []string    `json:"showtimes"`
}

// Result is a parsed theater schedule for one date.
type Result struct {
	TheaterName string  `json:"theater"`
	Address     string  `json:"address,omitempty"`
	Slug        string  `json:"slug"`
	Movies      []Entry `json:"movies"`
}

// Fetch retrieves and parses the schedule for one theater slug and date
// (YYYY-MM-DD).
func Fetch(ctx context.Context, hc *httpx.Client, slug, date string) (*Result, error) {
	u := apiBase + url.PathEscape(slug) + "?date=" + url.QueryEscape(date)
	data, err := hc.GetBytes(ctx, u)
	if err != nil {
		return nil, err
	}
	return Parse(data, date)
}

// Parse decodes a ClickTheCity theater response for the given date. It returns
// an error when status is false so a blocked/empty page is not read as "no
// movies".
func Parse(data []byte, date string) (*Result, error) {
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("ctc: decoding response: %w", err)
	}
	if !resp.Status {
		return nil, fmt.Errorf("ctc: API returned status=false (bad slug or blocked)")
	}

	movies := make(map[int]Movie, len(resp.NowShowing))
	for _, m := range resp.NowShowing {
		movies[m.MovieID] = m
	}

	byKey := map[string]*Entry{}
	var order []string
	for _, s := range resp.Schedules {
		if s.Date != date {
			continue
		}
		m := movies[s.MovieID]
		raw := m.Title
		if raw == "" {
			raw = fmt.Sprintf("Movie #%d", s.MovieID)
		}
		key := titles.Normalize(raw)
		e := byKey[key]
		if e == nil {
			e = &Entry{Key: key, RawTitle: raw, Rating: m.Rating, Runtime: m.Runtime, In3D: m.In3D}
			byKey[key] = e
			order = append(order, key)
		}
		cinema := strings.TrimSpace(strings.TrimLeft(s.TheaterName, "- "))
		e.CinemaRows = append(e.CinemaRows, CinemaRow{Cinema: cinema, Showtimes: s.Showtimes})
	}

	out := &Result{TheaterName: resp.Theater.Name, Address: resp.Theater.Address, Slug: resp.Theater.Slug}
	for _, key := range order {
		e := byKey[key]
		e.Showtimes = normalizedUnion(e.CinemaRows)
		sort.Slice(e.CinemaRows, func(i, j int) bool {
			return titles.NaturalLess(e.CinemaRows[i].Cinema, e.CinemaRows[j].Cinema)
		})
		out.Movies = append(out.Movies, *e)
	}
	sort.Slice(out.Movies, func(i, j int) bool {
		return strings.ToLower(out.Movies[i].RawTitle) < strings.ToLower(out.Movies[j].RawTitle)
	})
	return out, nil
}

func normalizedUnion(rows []CinemaRow) []string {
	set := map[string]struct{}{}
	for _, r := range rows {
		for _, t := range r.Showtimes {
			set[titles.NormalizeTime(t)] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		return titles.MinutesSinceMidnight(out[i]) < titles.MinutesSinceMidnight(out[j])
	})
	return out
}
