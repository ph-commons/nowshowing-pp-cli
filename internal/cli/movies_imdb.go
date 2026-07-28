// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command: implemented (was a generated scaffold).
// generate --force preserves implemented bodies; untouched TODO scaffolds may refresh.

// pp:client-call

package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"nowshowing-pp-cli/internal/imdb"
)

func newNovelMoviesImdbCmd(flags *rootFlags) *cobra.Command {
	var flagTitle string

	cmd := &cobra.Command{
		Use:         "imdb",
		Short:       "Resolves a movie title to its IMDb page, flagging remake/re-release title collisions by recent year.",
		Long:        "Resolves a movie title to its IMDb title page via IMDb's public suggestion endpoint. When a remake or re-release shares a title with an older film, the recent match wins; two or more recent exact matches are reported as ambiguous.",
		Example:     "  nowshowing-pp-cli movies imdb --title \"Moana\" --agent",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:happy-args": "--title=Superman"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would resolve IMDb match")
				return nil
			}
			if flagTitle == "" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--title is required"))
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			yearHint := time.Now().In(manilaLoc()).Year()
			res, err := imdb.Lookup(ctx, newSourceClient(), flagTitle, yearHint)
			if err != nil {
				return classifyAPIError(err, flags)
			}

			type view struct {
				Title     string `json:"title"`
				Found     bool   `json:"found"`
				IMDbID    string `json:"imdb_id,omitempty"`
				URL       string `json:"url,omitempty"`
				Ambiguous bool   `json:"ambiguous"`
			}
			v := view{Title: flagTitle}
			if res != nil {
				v.Found = true
				v.IMDbID = res.ID
				v.URL = res.URL
				v.Ambiguous = res.Ambiguous
			}

			if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), v, flags)
			}
			if !v.Found {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: no exact IMDb movie match\n", flagTitle)
				return nil
			}
			note := ""
			if v.Ambiguous {
				note = "  (ambiguous — multiple recent titles share this name)"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s: %s%s\n", flagTitle, v.URL, note)
			return nil
		},
	}
	cmd.Flags().StringVar(&flagTitle, "title", "", "Movie title to resolve to an IMDb page (required)")
	return cmd
}
