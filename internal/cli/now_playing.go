// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command: implemented (was a generated scaffold).
// generate --force preserves implemented bodies; untouched TODO scaffolds may refresh.

// pp:client-call

package cli

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"nowshowing-pp-cli/internal/cliutil"
	"nowshowing-pp-cli/internal/httpx"
	"nowshowing-pp-cli/internal/imdb"
	"nowshowing-pp-cli/internal/popcorn"
	"nowshowing-pp-cli/internal/registry"
	"nowshowing-pp-cli/internal/titles"
)

type cinemaRowView struct {
	Cinema    string   `json:"cinema"`
	Showtimes []string `json:"showtimes"`
	PricePHP  *float64 `json:"price_php,omitempty"`
}

type movieView struct {
	Movie      string           `json:"movie"`
	Rating     string           `json:"rating,omitempty"`
	Runtime    string           `json:"runtime,omitempty"`
	In3D       bool             `json:"in_3d,omitempty"`
	Showtimes  []string         `json:"showtimes"`
	Confidence popcorn.Confidence `json:"confidence"`
	IMDbURL    string           `json:"imdb_url,omitempty"`
	Cinemas    []cinemaRowView  `json:"cinemas"`
}

type theaterBoardView struct {
	Theater string      `json:"theater"`
	Slug    string      `json:"slug"`
	Address string      `json:"address,omitempty"`
	Movies  []movieView `json:"movies"`
}

func newNovelNowPlayingCmd(flags *rootFlags) *cobra.Command {
	var flagDate string
	var flagTheater string
	var flagNoCrossCheck bool
	var flagIMDb bool

	cmd := &cobra.Command{
		Use:         "now-playing",
		Short:       "Every movie showing today across all tracked Metro Manila and Iloilo cinemas, in one call.",
		Long:        "Fans out across every tracked theater (or one, with --theater) and returns the full board of movies and showtimes for the date. Each movie carries a two-source confidence signal cross-checked against popcorn.app (disable with --no-cross-check), verified ticket prices where available, and optional IMDb links (--imdb).",
		Example:     "  nowshowing-pp-cli now-playing --agent",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:happy-args": "--theater=sm-megamall"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would fetch showtimes across tracked theaters")
				return nil
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			date := flagDate
			if date == "" {
				date = todayManila()
			}
			nowMin := nowMinutesFor(date)

			theaters, ok := selectTheaters(flagTheater)
			if !ok {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("unknown theater slug %q; run 'theaters' for the tracked set", flagTheater))
			}
			crossCheck := !flagNoCrossCheck
			// Live dogfood runs against real sources with a flat per-command
			// timeout; curtail the fan-out so it fits.
			if cliutil.IsDogfoodEnv() {
				if len(theaters) > 2 {
					theaters = theaters[:2]
				}
				flagIMDb = false
			}

			hc := newSourceClient()
			fetches := fetchTheaters(ctx, hc, theaters, date)

			var pcIndexes map[string]popcorn.Index
			if crossCheck {
				pcIndexes = fetchPopcornAll(ctx, hc, theaters, date)
			}
			imdbCache := map[string]string{}

			boards := make([]theaterBoardView, 0, len(theaters))
			failures := make([]fetchFailure, 0)
			var rateErr *cliutil.RateLimitError

			for _, f := range fetches {
				if f.Err != nil {
					if asRateLimit(f.Err, &rateErr) {
						return classifyAPIError(f.Err, flags)
					}
					failures = append(failures, fetchFailure{Theater: f.Theater.Name(), Error: f.Err.Error()})
					continue
				}
				board := theaterBoardView{
					Theater: displayTheaterName(f),
					Slug:    f.Theater.Slug,
					Address: f.Result.Address,
				}
				pcIdx := pcIndexes[f.Theater.Slug]
				for _, m := range f.Result.Movies {
					mv := movieView{
						Movie:     m.RawTitle,
						Rating:    m.Rating,
						Runtime:   m.Runtime,
						In3D:      m.In3D,
						Showtimes: m.Showtimes,
					}
					if crossCheck {
						var pcEntry *popcorn.Entry
						if e, found := pcIdx[m.Key]; found {
							pcEntry = &e
						}
						mv.Confidence = popcorn.CheckShowtimes(m.Showtimes, pcEntry, nowMin).Status
					} else {
						mv.Confidence = popcorn.CTCOnly
					}
					if flagIMDb {
						mv.IMDbURL = resolveIMDb(ctx, hc, imdbCache, m.RawTitle, date)
					}
					for _, cr := range m.CinemaRows {
						row := cinemaRowView{Cinema: cr.Cinema, Showtimes: normalizeRowTimes(cr.Showtimes)}
						if price, okp := registry.PriceFor(f.Theater.Slug, cr.Cinema); okp {
							p := price
							row.PricePHP = &p
						}
						mv.Cinemas = append(mv.Cinemas, row)
					}
					board.Movies = append(board.Movies, mv)
				}
				boards = append(boards, board)
			}

			view := struct {
				Date          string             `json:"date"`
				Theaters      []theaterBoardView `json:"theaters"`
				FetchFailures []fetchFailure     `json:"fetch_failures"`
			}{Date: date, Theaters: boards, FetchFailures: failures}

			if len(failures) > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %d of %d theaters failed to fetch\n", len(failures), len(fetches))
			}

			if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			// Human table: flatten to one row per movie per theater.
			items := make([]map[string]any, 0)
			for _, b := range boards {
				for _, m := range b.Movies {
					items = append(items, map[string]any{
						"theater":    b.Theater,
						"movie":      m.Movie,
						"showtimes":  joinTimes(m.Showtimes),
						"confidence": string(m.Confidence),
					})
				}
			}
			if len(items) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No showtimes found for %s.\n", date)
				return nil
			}
			return printAutoTable(cmd.OutOrStdout(), items)
		},
	}
	cmd.Flags().StringVar(&flagDate, "date", "", "Schedule date, YYYY-MM-DD (default: today in Asia/Manila)")
	cmd.Flags().StringVar(&flagTheater, "theater", "", "Limit to one theater slug (default: all tracked theaters)")
	cmd.Flags().BoolVar(&flagNoCrossCheck, "no-cross-check", false, "Skip the popcorn.app cross-check (faster; every movie reports ctc-only)")
	cmd.Flags().BoolVar(&flagIMDb, "imdb", false, "Enrich each movie with an IMDb link (extra lookups)")
	return cmd
}

// fetchPopcornAll fetches each theater's popcorn.app index concurrently,
// sharing the rate-limited client. A theater with no parseable popcorn page
// maps to a nil index (cross-check unavailable), never an error.
func fetchPopcornAll(ctx context.Context, hc *httpx.Client, theaters []registry.Theater, date string) map[string]popcorn.Index {
	out := make(map[string]popcorn.Index, len(theaters))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, t := range theaters {
		wg.Add(1)
		go func() {
			defer wg.Done()
			idx, _ := popcorn.Fetch(ctx, hc, t.PopcornURL, date)
			mu.Lock()
			out[t.Slug] = idx
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}

func resolveIMDb(ctx context.Context, hc *httpx.Client, cache map[string]string, title, date string) string {
	if u, ok := cache[title]; ok {
		return u
	}
	res, err := imdb.Lookup(ctx, hc, title, yearOf(date))
	url := ""
	if err == nil && res != nil {
		url = res.URL
	}
	cache[title] = url
	return url
}

// yearOf parses the leading year from a YYYY-MM-DD date, falling back to the
// current year in Asia/Manila.
func yearOf(date string) int {
	if len(date) >= 4 {
		if y, err := strconv.Atoi(date[:4]); err == nil {
			return y
		}
	}
	return time.Now().In(manilaLoc()).Year()
}

func normalizeRowTimes(ts []string) []string {
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		out = append(out, titles.NormalizeTime(t))
	}
	return out
}
