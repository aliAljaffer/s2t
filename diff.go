package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	showValues     bool
	diffNamespace  string
	namespaceA     string
	namespaceB     string
	diffKubeconfig string
)

var diffCmd = &cobra.Command{
	Use:   "diff <fileA|nameA> <fileB|nameB>",
	Short: "Compare two secrets' decoded contents key by key",
	Example: `  s2t diff staging.yaml prod.yaml                        compare two files
  s2t diff --show-values old.yaml new.yaml               also print the actual values
  s2t diff db-creds db-creds -n staging -B prod          compare a live secret across two namespaces
  s2t diff cm/app-config cm/app-config -n stg -B prod    compare a live ConfigMap across two namespaces`,
	Args:          cobra.ExactArgs(2),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDiff(args[0], args[1], format, kind, showValues, diffNamespace, namespaceA, namespaceB, diffKubeconfig)
	},
}

func init() {
	diffCmd.Flags().BoolVarP(&showValues, "show-values", "", false, "print the actual old/new values, not just which keys changed")
	diffCmd.Flags().StringVarP(&diffNamespace, "namespace", "n", "", "namespace for either argument that's a live resource rather than a file; overridden per-side by -A/-B; defaults to the kubeconfig's current context")
	diffCmd.Flags().StringVarP(&namespaceA, "namespace-a", "A", "", "namespace for the first argument, if it's a live resource (overrides --namespace)")
	diffCmd.Flags().StringVarP(&namespaceB, "namespace-b", "B", "", "namespace for the second argument, if it's a live resource (overrides --namespace)")
	diffCmd.Flags().StringVarP(&diffKubeconfig, "kubeconfig", "", "", "path to the kubeconfig file to use for live resources; if omitted, kubectl resolves it itself from $KUBECONFIG or ~/.kube/config")
}

func runDiff(srcA, srcB, format, kind string, showValues bool, namespace, namespaceA, namespaceB, kubeconfig string) error {
	a, err := decodeSource(srcA, format, kind, firstNonEmpty(namespaceA, namespace), kubeconfig)
	if err != nil {
		return fmt.Errorf("reading %s: %w", srcA, err)
	}
	b, err := decodeSource(srcB, format, kind, firstNonEmpty(namespaceB, namespace), kubeconfig)
	if err != nil {
		return fmt.Errorf("reading %s: %w", srcB, err)
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

// decodeSource resolves one diff argument, dispatching to decodeFile if src
// names a real file on disk, or to decodeLive (a live kubectl fetch)
// otherwise - the same file-vs-name distinction the root command makes via
// -f vs a positional name, but inferred here since diff's positional
// arguments serve double duty.
func decodeSource(src, format, kind, namespace, kubeconfig string) (DecodedData, error) {
	if isFile(src) {
		return decodeFile(src, format, kind)
	}
	return decodeLive(src, kind, namespace, kubeconfig)
}

// isFile reports whether path names an existing regular file (not a
// directory), matching real files rather than tripping over a live resource
// name that happens to collide with a directory.
func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// decodeLive fetches a live Secret or ConfigMap via kubectl and runs it
// through the same JSON-parse/decode pipeline as the root command's live
// fetch path, so diff sees exactly what `s2t <name> -n <namespace>` would
// show. name may carry a "kind/name" prefix (e.g. "cm/app-config"), which
// overrides kind for this source only, matching the root command's own
// kind/name positional syntax.
func decodeLive(name, kind, namespace, kubeconfig string) (DecodedData, error) {
	if parsedKind, plainName, ok, err := splitKindName(name); err != nil {
		return nil, err
	} else if ok {
		kind, name = parsedKind, plainName
	}

	if namespace == "" {
		var err error
		namespace, err = resolveNamespace(kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("resolving current namespace: %w", err)
		}
	}

	raw, err := fetchResourceJSON(kind, namespace, name, kubeconfig)
	if err != nil {
		return nil, err
	}

	// kubectl -o json always produces JSON, so the parser choice is forced,
	// same as the root command's live fetch path.
	parser, err := parserFor("json", kind)
	if err != nil {
		return nil, err
	}

	data, err := parser.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing input: %w", err)
	}

	return decode(os.Stderr, data), nil
}

// firstNonEmpty returns a, or b if a is empty - used to let a per-side
// namespace override (-A/-B) win over the shared --namespace.
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
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
