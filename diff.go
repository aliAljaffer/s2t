package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var showValues bool

var diffCmd = &cobra.Command{
	Use:   "diff <fileA> <fileB>",
	Short: "Compare two secrets' decoded contents key by key",
	Example: `  s2t diff staging.yaml prod.yaml                  show which keys were added, removed, or changed
  s2t diff --show-values old.yaml new.yaml         also print the actual values`,
	Args:          cobra.ExactArgs(2),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDiff(args[0], args[1], format, kind, showValues)
	},
}

func init() {
	diffCmd.Flags().BoolVarP(&showValues, "show-values", "", false, "print the actual old/new values, not just which keys changed")
}

func runDiff(pathA, pathB, format, kind string, showValues bool) error {
	a, err := decodeFile(pathA, format, kind)
	if err != nil {
		return fmt.Errorf("reading %s: %w", pathA, err)
	}
	b, err := decodeFile(pathB, format, kind)
	if err != nil {
		return fmt.Errorf("reading %s: %w", pathB, err)
	}

	added := color.New(color.FgGreen)
	removed := color.New(color.FgRed)
	changed := color.New(color.FgYellow)

	for _, key := range sortedUnionKeys(a, b) {
		valA, inA := a[key]
		valB, inB := b[key]

		switch {
		case inA && !inB:
			printDiffLine(removed, "-", key, valA, showValues)
		case !inA && inB:
			printDiffLine(added, "+", key, valB, showValues)
		case valA != valB:
			if showValues {
				changed.Fprintf(os.Stdout, "~ %s: %s -> %s\n", key, valA, valB)
			} else {
				changed.Fprintf(os.Stdout, "~ %s\n", key)
			}
		}
		// valA == valB: unchanged, silently omitted (matches standard diff).
	}

	return nil
}

func printDiffLine(c *color.Color, prefix, key, value string, showValues bool) {
	if showValues {
		c.Fprintf(os.Stdout, "%s %s: %s\n", prefix, key, value)
		return
	}
	c.Fprintf(os.Stdout, "%s %s\n", prefix, key)
}

// decodeFile runs a manifest file through the same parserFor/decode pipeline
// as the root command, so diff sees exactly what `s2t -f <path>` would show.
func decodeFile(path, format, kind string) (DecodedData, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parser, err := parserFor(format, kind)
	if err != nil {
		return nil, err
	}

	data, err := parser.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing input: %w", err)
	}

	return decode(os.Stderr, data), nil
}

// sortedUnionKeys returns the sorted union of a's and b's keys, so diff
// output is stable and reproducible like sortedKeys.
func sortedUnionKeys(a, b DecodedData) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}

	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
