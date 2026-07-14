package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	file       string
	format     string
	namespace  string
	name       string
	only       string
	output     string
	kubeconfig string
	kind       string
	mask       bool
)

var rootCmd = &cobra.Command{
	Use:   "s2t",
	Short: "s2t - Kubernetes Secret to Text decoder",
	Example: `  s2t -f secret.yaml                              decode a saved manifest file
  cat secret.json | s2t                           decode piped stdin (format auto-detected)
  s2t --name db-creds --namespace prod            fetch and decode a live secret via kubectl
  s2t -f secret.yaml --only username,password     only print specific keys
  s2t -f secret.yaml -o env                       print as KEY=value pairs
  s2t --name db-creds --namespace prod -o yaml    re-encode a live secret as a patch-ready manifest
  s2t -f app.yaml -k configmap                    decode a ConfigMap manifest instead of a Secret
  s2t --name cm/app-config --namespace prod       fetch a ConfigMap live; kind is derived from the cm/ prefix
  s2t diff a.yaml b.yaml                          compare two secrets' decoded contents key by key`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, file, format, kind, namespace, name, only, output, kubeconfig, mask)
	},
}

func init() {
	persistent := rootCmd.PersistentFlags()
	persistent.StringVarP(&format, "format", "t", "any", "input format: yaml, json, kv, or sealed-secret (ignored when --name is used; sealed-secret requires "+sealedSecretKeyEnvVar+")")
	persistent.StringVarP(&kind, "kind", "k", kindSecret, "resource kind: secret or configmap")

	flags := rootCmd.Flags()
	flags.StringVarP(&kubeconfig, "kubeconfig", "", "~/.kube/config", "path to the kubeconfig file to use.")
	flags.StringVarP(&file, "file", "f", "", "path to a file containing secret data; omit to read from stdin")
	flags.StringVarP(&namespace, "namespace", "s", "", "kubernetes namespace (used with --name; defaults to the kubeconfig's current context if omitted)")
	flags.StringVarP(&name, "name", "n", "", "name of the resource to fetch live via kubectl; either a plain name (uses --kind) or kind/name, e.g. secret/my-secret, configmap/my-cm, or cm/my-cm")
	flags.StringVarP(&only, "only", "", "", "comma-separated list of keys to print out")
	flags.StringVarP(&output, "output", "o", "", "output format: empty (plain), env, json, jsonc, or yaml (json/jsonc/yaml produce a kubectl-patch-ready manifest; jsonc is compact/unindented)")
	flags.BoolVarP(&mask, "mask", "", false, "replace every value with a fixed-length placeholder; cannot be combined with -o json/jsonc/yaml")

	rootCmd.RegisterFlagCompletionFunc("namespace", completeNamespaces)
	rootCmd.RegisterFlagCompletionFunc("name", completeResourceNames)
	rootCmd.RegisterFlagCompletionFunc("kind", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return matchPrefix([]string{kindSecret, kindConfigMap}, toComplete), cobra.ShellCompDirectiveNoFileComp
	})

	rootCmd.AddCommand(diffCmd)
}

// completeNamespaces offers real namespace names, fetched live via kubectl,
// for shell completion of --namespace/-n.
func completeNamespaces(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return matchPrefix(kubectlResourceNames("namespace", "", kubeconfig), toComplete), cobra.ShellCompDirectiveNoFileComp
}

// completeResourceNames offers real secret or configmap names (per --kind,
// or per a "kind/" prefix already typed into --name itself, e.g. "cm/my-")
// within the namespace already typed on the command line, falling back to
// the kubeconfig's current-context namespace (matching kubectl's own
// resolution) when --namespace isn't typed yet.
func completeResourceNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ns := namespace
	if ns == "" {
		ns = kubectlCurrentNamespace(kubeconfig)
	}
	if ns == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	k, prefix, rest := kind, "", toComplete
	if alias, name, found := strings.Cut(toComplete, "/"); found {
		if resolved, ok := kindAliases[alias]; ok {
			k, prefix, rest = resolved, alias+"/", name
		}
	}

	matched := matchPrefix(kubectlResourceNames(k, ns, kubeconfig), rest)
	if prefix == "" {
		return matched, cobra.ShellCompDirectiveNoFileComp
	}
	result := make([]string, len(matched))
	for i, m := range matched {
		result[i] = prefix + m
	}
	return result, cobra.ShellCompDirectiveNoFileComp
}

func matchPrefix(names []string, prefix string) []string {
	if prefix == "" {
		return names
	}
	var matched []string
	for _, n := range names {
		if strings.HasPrefix(n, prefix) {
			matched = append(matched, n)
		}
	}
	return matched
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, file, format, kind, namespace, name, only, output, kubeconfig string, mask bool) error {
	if mask && (output == "json" || output == "jsonc" || output == "yaml") {
		return fmt.Errorf("--mask cannot be combined with -o json/jsonc/yaml: these produce a kubectl-patch-ready payload, and a masked value would silently overwrite the real secret with the literal string %q if applied", maskPlaceholder)
	}

	var (
		raw []byte
		err error
	)

	switch {
	case name != "":
		if parsedKind, plainName, ok, splitErr := splitKindName(name); splitErr != nil {
			return splitErr
		} else if ok {
			if cmd.Flags().Changed("kind") && kind != parsedKind {
				return fmt.Errorf("--kind %s conflicts with resource kind %q in --name %q", kind, parsedKind, name)
			}
			kind, name = parsedKind, plainName
		}
		if namespace == "" {
			namespace, err = resolveNamespace(kubeconfig)
			if err != nil {
				return fmt.Errorf("resolving current namespace: %w", err)
			}
		}
		raw, err = fetchResourceJSON(kind, namespace, name, kubeconfig)
		format = "json" // kubectl -o json always produces JSON, so we force the parser choice
	case file != "":
		raw, err = os.ReadFile(file)
	default:
		if isTerminal(os.Stdin) {
			cmd.Usage()
			os.Exit(1)
		}
		raw, err = io.ReadAll(os.Stdin)
	}

	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	if format == "" {
		return errors.New("--format is required (yaml, json, or kv)")
	}

	parser, err := parserFor(format, kind)
	if err != nil {
		return err
	}

	data, err := parser.Parse(raw)
	if err != nil {
		return fmt.Errorf("parsing input: %w", err)
	}

	if only != "" {
		filteredKeys := strings.Split(only, ",")
		filteredData := SecretData{}
		for _, key := range filteredKeys {
			sanitizedKey, err := sanitizeKey(key)
			if err != nil {
				continue
			}
			if val, ok := data[sanitizedKey]; ok {
				filteredData[sanitizedKey] = val
			} else {
				return fmt.Errorf("key %q not found", sanitizedKey)
			}
		}
		data = filteredData
	}

	formatter, err := formatterFor(output, kind)
	if err != nil {
		return err
	}

	decoded := decode(os.Stderr, data)
	if mask {
		decoded = maskValues(decoded)
	}
	if err := formatter.Format(os.Stdout, decoded); err != nil {
		return fmt.Errorf("formatting output: %w", err)
	}
	return nil
}

// isTerminal reports whether f is an interactive terminal rather than a pipe
// or redirected file. Used to avoid blocking forever on os.Stdin when the
// user runs the command with no input and no flags.
func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}

func sanitizeKey(key string) (string, error) {
	sanitizedKey := strings.TrimSpace(key)
	if sanitizedKey == "" {
		return "", errors.New("empty key")
	}
	return sanitizedKey, nil
}
