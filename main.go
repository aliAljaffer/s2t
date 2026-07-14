package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	var (
		file       string
		format     string
		namespace  string
		secretName string
		only       string
		output     string
		help       bool
		kubeconfig string
	)

	flag.StringVar(&kubeconfig, "kubeconfig", "~/.kube/config", "path to the kubeconfig file to use.")
	flag.StringVar(&file, "file", "", "path to a file containing secret data; omit to read from stdin")
	flag.StringVar(&file, "f", "", "path to a file containing secret data; omit to read from stdin")
	flag.StringVar(&format, "format", "any", "input format: yaml, json, or kv (ignored when --secret is used)")
	flag.StringVar(&format, "t", "any", "input format: yaml, json, or kv (ignored when --secret is used)")
	flag.StringVar(&namespace, "namespace", "", "kubernetes namespace (used with --secret)")
	flag.StringVar(&namespace, "n", "", "kubernetes namespace (used with --secret)")
	flag.BoolVar(&help, "help", false, "print out the usage for the s2t tool")
	flag.BoolVar(&help, "h", false, "print out the usage for the s2t tool")
	flag.StringVar(&secretName, "secret", "", "name of the secret to fetch live via kubectl")
	flag.StringVar(&secretName, "s", "", "name of the secret to fetch live via kubectl")
	flag.StringVar(&only, "only", "", "comma-separated list of keys to print out")
	flag.StringVar(&output, "output", "", "output format: empty (plain), env, json, or yaml (json/yaml produce a kubectl-patch-ready stringData manifest)")
	flag.StringVar(&output, "o", "", "output format: empty (plain), env, json, or yaml (json/yaml produce a kubectl-patch-ready stringData manifest)")
	flag.Parse()

	var (
		raw []byte
		err error
	)

	switch {
	case help:
		usage(flag.Usage)
		os.Exit(1)
	case secretName != "":
		if namespace == "" {
			fmt.Fprintln(os.Stderr, "error: --namespace is required when using --secret")
			os.Exit(1)
		}
		raw, err = fetchSecretJSON(namespace, secretName, kubeconfig)
		format = "json" // kubectl -o json always produces JSON, so we force the parser choice
	case file != "":
		raw, err = os.ReadFile(file)
	default:
		if isTerminal(os.Stdin) {
			usage(flag.Usage)
			os.Exit(1)
		}
		raw, err = io.ReadAll(os.Stdin)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading input:", err)
		os.Exit(1)
	}

	if format == "" {
		fmt.Fprintln(os.Stderr, "error: --format is required (yaml, json, or kv)")
		os.Exit(1)
	}

	parser, err := parserFor(format)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	data, err := parser.Parse(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error parsing input:", err)
		os.Exit(1)
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
				fmt.Fprintf(os.Stderr, "error: key \"%s\" not found\n", sanitizedKey)
				os.Exit(1)
			}
		}
		data = filteredData
	}

	formatter, err := formatterFor(output)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	decoded := decode(os.Stderr, data)
	if err := formatter.Format(os.Stdout, decoded); err != nil {
		fmt.Fprintln(os.Stderr, "error formatting output:", err)
		os.Exit(1)
	}
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

func usage(flagUsage func()) (error) {
	_, err := fmt.Fprintln(os.Stdout, "s2t - Kubernetes Secret to Text decoder")
	if err != nil {
		return errors.New("error printing output: " + err.Error())
	}
	flagUsage()
	_, err = fmt.Fprint(os.Stdout, `
Examples:
  s2t -f secret.yaml                          decode a saved manifest file
  cat secret.json | s2t                       decode piped stdin (format auto-detected)
  s2t -s db-creds -n prod                     fetch and decode a live secret via kubectl
  s2t -f secret.yaml -only username,password  only print specific keys
  s2t -f secret.yaml -o env                   print as KEY=value pairs
  s2t -s db-creds -n prod -o yaml             re-encode a live secret as a patch-ready manifest
`)
	if err != nil {
		return errors.New("error printing output: " + err.Error())
	}
	return nil
}
