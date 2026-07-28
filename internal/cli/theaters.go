// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command: implemented (was a generated scaffold).
// generate --force preserves implemented bodies; untouched TODO scaffolds may refresh.

// pp:novel-static-reference

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"nowshowing-pp-cli/internal/registry"
)

func newNovelTheatersCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "theaters",
		Short:       "Lists every tracked cinema with its ClickTheCity slug, display name, and city.",
		Long:        "Lists the curated set of tracked Metro Manila and Iloilo cinemas. Use the slug values here with 'theater <slug>', 'now-playing --theater <slug>', and 'search --theater <slug>'.",
		Example:     "  nowshowing-pp-cli theaters --agent",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			type row struct {
				Slug       string `json:"slug"`
				Name       string `json:"name"`
				PopcornURL string `json:"popcorn_url"`
				FallbackNm string `json:"fallback_name,omitempty"`
			}
			rows := make([]row, 0, len(registry.Theaters))
			for _, t := range registry.Theaters {
				rows = append(rows, row{Slug: t.Slug, Name: t.Name(), PopcornURL: t.PopcornURL, FallbackNm: t.FallbackName})
			}

			if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), rows, flags)
			}
			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				items := make([]map[string]any, 0, len(rows))
				for _, r := range rows {
					items = append(items, map[string]any{"slug": r.Slug, "name": r.Name})
				}
				return printAutoTable(cmd.OutOrStdout(), items)
			}
			b, _ := json.Marshal(rows)
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		},
	}
	return cmd
}
