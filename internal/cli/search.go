// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command: implemented (was a generated scaffold).
// generate --force preserves implemented bodies; untouched TODO scaffolds may refresh.

// pp:client-call

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ph-commons/nowshowing-pp-cli/internal/cliutil"
)

func newNovelSearchCmd(flags *rootFlags) *cobra.Command {
	var flagDate string
	var flagTheater string

	cmd := &cobra.Command{
		Use:         "search <movie>",
		Short:       "Find which tracked theaters are playing a given movie today, and at what times.",
		Long:        "Searches every tracked theater (or one, with --theater) for a movie whose title contains the query, and reports where and when it is showing on the given date.",
		Example:     "  nowshowing-pp-cli search \"Superman\" --agent",
		// A search with no matching movie is a valid empty result (exit 0), not
		// an error: a garbage title is indistinguishable from a real movie that
		// simply is not showing today. Skip dogfood's error-path probe.
		Annotations: map[string]string{"mcp:read-only": "true", "pp:happy-args": "Superman", "pp:no-error-path-probe": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would search theaters for a movie")
				return nil
			}
			if len(args) < 1 || args[0] == "" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a movie title argument is required"))
			}
			query := args[0]

			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			date := flagDate
			if date == "" {
				date = todayManila()
			}
			theaters, ok := selectTheaters(flagTheater)
			if !ok {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("unknown theater slug %q; run 'theaters' for the tracked set", flagTheater))
			}
			if cliutil.IsDogfoodEnv() && len(theaters) > 2 {
				theaters = theaters[:2]
			}

			fetches := fetchTheaters(ctx, newSourceClient(), theaters, date)

			type hit struct {
				Theater   string   `json:"theater"`
				Slug      string   `json:"slug"`
				Movie     string   `json:"movie"`
				Rating    string   `json:"rating,omitempty"`
				Runtime   string   `json:"runtime,omitempty"`
				Showtimes []string `json:"showtimes"`
			}
			hits := make([]hit, 0)
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
				for _, m := range f.Result.Movies {
					if !titleMatches(m.RawTitle, query) {
						continue
					}
					hits = append(hits, hit{
						Theater:   displayTheaterName(f),
						Slug:      f.Theater.Slug,
						Movie:     m.RawTitle,
						Rating:    m.Rating,
						Runtime:   m.Runtime,
						Showtimes: m.Showtimes,
					})
				}
			}

			view := struct {
				Query         string         `json:"query"`
				Date          string         `json:"date"`
				Hits          []hit          `json:"hits"`
				FetchFailures []fetchFailure `json:"fetch_failures"`
			}{Query: query, Date: date, Hits: hits, FetchFailures: failures}

			if len(failures) > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %d of %d theaters failed to fetch\n", len(failures), len(fetches))
			}

			if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			if len(hits) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No tracked theater is showing %q on %s.\n", query, date)
				return nil
			}
			items := make([]map[string]any, 0, len(hits))
			for _, h := range hits {
				items = append(items, map[string]any{"theater": h.Theater, "movie": h.Movie, "showtimes": joinTimes(h.Showtimes)})
			}
			return printAutoTable(cmd.OutOrStdout(), items)
		},
	}
	cmd.Flags().StringVar(&flagDate, "date", "", "Schedule date, YYYY-MM-DD (default: today in Asia/Manila)")
	cmd.Flags().StringVar(&flagTheater, "theater", "", "Limit the search to one theater slug (default: all tracked theaters)")
	return cmd
}
