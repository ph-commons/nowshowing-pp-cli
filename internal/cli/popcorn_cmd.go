// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored porcelain: raw popcorn.app cross-check source. Registered via
// the registerNovelCommand hook so the generated root stays regenerable.

// pp:client-call

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"nowshowing-pp-cli/internal/popcorn"
	"nowshowing-pp-cli/internal/registry"
)

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		root.AddCommand(newPopcornCmd(flags))
	})
}

func newPopcornCmd(flags *rootFlags) *cobra.Command {
	var flagURL string
	var flagTheater string
	var flagDate string

	cmd := &cobra.Command{
		Use:   "popcorn",
		Short: "Raw popcorn.app showtimes for one cinema — the secondary cross-check source.",
		Long: "Fetches and parses one popcorn.app cinema page (the secondary source now-playing cross-checks against) and prints its movies and showtimes for the date. " +
			"Pass a page URL with --url, or a tracked theater slug with --theater to use its known popcorn.app URL. popcorn.app drops showtimes that have already started, and some SM-managed cinemas have no parseable page.",
		Example:     "  nowshowing-pp-cli popcorn --theater greenbelt-3 --agent",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:happy-args": "--theater=greenbelt-3"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would fetch a popcorn.app cinema page")
				return nil
			}
			url := flagURL
			if url == "" && flagTheater != "" {
				t, ok := registry.BySlug(flagTheater)
				if !ok {
					_ = cmd.Usage()
					return usageErr(fmt.Errorf("unknown theater slug %q; run 'theaters' for the tracked set", flagTheater))
				}
				url = t.PopcornURL
			}
			if url == "" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("provide --url or --theater"))
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			date := flagDate
			if date == "" {
				date = todayManila()
			}
			idx, err := popcorn.Fetch(ctx, newSourceClient(), url, date)
			if err != nil {
				return classifyAPIError(err, flags)
			}

			type row struct {
				Movie     string   `json:"movie"`
				Showtimes []string `json:"showtimes"`
			}
			rows := make([]row, 0, len(idx))
			for _, e := range idx {
				rows = append(rows, row{Movie: e.RawTitle, Showtimes: e.Showtimes})
			}

			view := struct {
				URL       string `json:"url"`
				Date      string `json:"date"`
				Parseable bool   `json:"parseable"`
				Movies    []row  `json:"movies"`
			}{URL: url, Date: date, Parseable: idx != nil, Movies: rows}

			if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			if idx == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "No parseable popcorn.app showtimes for %s (page has no allShowtimes blob).\n", url)
				return nil
			}
			if len(rows) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "popcorn.app lists no movies for %s on %s.\n", url, date)
				return nil
			}
			items := make([]map[string]any, 0, len(rows))
			for _, r := range rows {
				items = append(items, map[string]any{"movie": r.Movie, "showtimes": joinTimes(r.Showtimes)})
			}
			return printAutoTable(cmd.OutOrStdout(), items)
		},
	}
	cmd.Flags().StringVar(&flagURL, "url", "", "popcorn.app cinema page URL")
	cmd.Flags().StringVar(&flagTheater, "theater", "", "Tracked theater slug (uses its known popcorn.app URL)")
	cmd.Flags().StringVar(&flagDate, "date", "", "Schedule date, YYYY-MM-DD (default: today in Asia/Manila)")
	return cmd
}
