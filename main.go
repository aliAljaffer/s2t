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
	secretName string
	only       string
	output     string
	kubeconfig string
)

var rootCmd = &cobra.Command{
	Use:   "s2t",
	Short: "s2t - Kubernetes Secret to Text decoder",
	Example: `  s2t -f secret.yaml                            decode a saved manifest file
  cat secret.json | s2t                         decode piped stdin (format auto-detected)
  s2t -s db-creds -n prod                       fetch and decode a live secret via kubectl
  s2t -f secret.yaml --only username,password   only print specific keys
  s2t -f secret.yaml -o env                     print as KEY=value pairs
  s2t -s db-creds -n prod -o yaml               re-encode a live secret as a patch-ready manifest`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, file, format, namespace, secretName, only, output, kubeconfig)
	},
}

func init() {
	flags := rootCmd.Flags()
	flags.StringVarP(&kubeconfig, "kubeconfig", "", "~/.kube/config", "path to the kubeconfig file to use.")
	flags.StringVarP(&file, "file", "f", "", "path to a file containing secret data; omit to read from stdin")
	flags.StringVarP(&format, "format", "t", "any", "input format: yaml, json, or kv (ignored when --secret is used)")
	flags.StringVarP(&namespace, "namespace", "n", "", "kubernetes namespace (used with --secret)")
	flags.StringVarP(&secretName, "secret", "s", "", "name of the secret to fetch live via kubectl")
	flags.StringVarP(&only, "only", "", "", "comma-separated list of keys to print out")
	flags.StringVarP(&output, "output", "o", "", "output format: empty (plain), env, json, jsonc, or yaml (json/jsonc/yaml produce a kubectl-patch-ready stringData manifest; jsonc is compact/unindented)")

	rootCmd.RegisterFlagCompletionFunc("namespace", completeNamespaces)
	rootCmd.RegisterFlagCompletionFunc("secret", completeSecrets)
}

// completeNamespaces offers real namespace names, fetched live via kubectl,
// for shell completion of --namespace/-n.
func completeNamespaces(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return matchPrefix(kubectlResourceNames("namespace", "", kubeconfig), toComplete), cobra.ShellCompDirectiveNoFileComp
}

// completeSecrets offers real secret names within the namespace already typed
// on the command line (--namespace must come first; kubectl has no concept
// of a secret name without one). No namespace yet means no completions.
func completeSecrets(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if namespace == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return matchPrefix(kubectlResourceNames("secret", namespace, kubeconfig), toComplete), cobra.ShellCompDirectiveNoFileComp
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

func run(cmd *cobra.Command, file, format, namespace, secretName, only, output, kubeconfig string) error {
	var (
		raw []byte
		err error
	)

	switch {
	case secretName != "":
		if namespace == "" {
			return errors.New("--namespace is required when using --secret")
		}
		raw, err = fetchSecretJSON(namespace, secretName, kubeconfig)
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

	parser, err := parserFor(format)
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

	formatter, err := formatterFor(output)
	if err != nil {
		return err
	}

	decoded := decode(os.Stderr, data)
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
