package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	setNamespace  string
	setKubeconfig string
	setFile       string
	setOutput     string
	setApply      bool
	setDryRun     bool
	setSeparator  string
)

// defaultSeparator is the dotnet configuration nesting separator: a nested
// appsettings key ConnectionStrings.Default becomes ConnectionStrings__Default.
const defaultSeparator = "__"

var setCmd = &cobra.Command{
	Use:     "set <name> [key=value ...]",
	Aliases: []string{"upsert"},
	Short:   "Upsert plaintext keys into a live Secret or ConfigMap",
	Long: `set takes plaintext key/value input and produces (or applies) a merge patch
against a live Secret or ConfigMap. Values are never base64-encoded by hand:
Secrets use stringData, which the API server encodes for you.

Input comes from -f/stdin (auto-detected JSON, YAML, or env) and/or positional
key=value arguments. Nested JSON/YAML is flattened with "__" so dotnet
appsettings files map straight onto secret keys.`,
	Example: `  cat appsettings.json | s2t set db-creds -n prod --apply
  cat appsettings.json | s2t set db-creds -n prod key1=val1,key2=val2 --apply
  s2t set db-creds -n prod username=admin password=s3cret --apply
  s2t set db-creds -n prod < values.env           # prints a merge patch to stdout
  s2t set cm/app-config -n prod feature.flag=on --apply --dry-run`,
	Args:          cobra.MinimumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSet(cmd, args)
	},
}

func init() {
	flags := setCmd.Flags()
	flags.StringVarP(&setNamespace, "namespace", "n", "", "kubernetes namespace of the target resource (defaults to the kubeconfig's current context)")
	flags.StringVarP(&setKubeconfig, "kubeconfig", "", "", "path to the kubeconfig file to use; if omitted, kubectl resolves it itself from $KUBECONFIG or ~/.kube/config")
	flags.StringVarP(&setFile, "file", "f", "", "path to a file containing plaintext key/value data; omit to read from stdin")
	flags.StringVarP(&setOutput, "output", "o", "jsonc", "patch payload format: jsonc (compact), json, or yaml")
	flags.BoolVarP(&setApply, "apply", "", false, "apply the merge patch to the cluster with kubectl patch (default: print the patch only)")
	flags.BoolVarP(&setDryRun, "dry-run", "", false, "with --apply, validate server-side using kubectl --dry-run=server without writing")
	flags.StringVarP(&setSeparator, "separator", "", defaultSeparator, "separator used to flatten nested JSON/YAML keys")
}

func runSet(cmd *cobra.Command, args []string) error {
	name := args[0]
	if parsedKind, plainName, ok, err := splitKindName(name); err != nil {
		return err
	} else if ok {
		if cmd.Flags().Changed("kind") && kind != parsedKind {
			return fmt.Errorf("--kind %s conflicts with resource kind %q in %q", kind, parsedKind, name)
		}
		kind, name = parsedKind, plainName
	}

	switch setOutput {
	case "jsonc", "json", "yaml":
	default:
		return fmt.Errorf("invalid --output %q (want jsonc, json, or yaml)", setOutput)
	}

	data := DecodedData{}

	if setFile != "" {
		raw, err := os.ReadFile(setFile)
		if err != nil {
			return fmt.Errorf("reading %s: %w", setFile, err)
		}
		parsed, err := parsePlaintext(format, raw, setSeparator)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", setFile, err)
		}
		for k, v := range parsed {
			data[k] = v
		}
	} else if !isTerminal(os.Stdin) {
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		if len(bytes.TrimSpace(raw)) > 0 {
			parsed, err := parsePlaintext(format, raw, setSeparator)
			if err != nil {
				return fmt.Errorf("parsing stdin: %w", err)
			}
			for k, v := range parsed {
				data[k] = v
			}
		}
	}

	// Positional pairs are always env format, regardless of -t/-f, so a few
	// ad-hoc keys can be layered on top of (or replace) a bulk file/stdin.
	for _, arg := range args[1:] {
		pairs, err := parseEnv([]byte(arg))
		if err != nil {
			return fmt.Errorf("parsing %q: %w", arg, err)
		}
		for k, v := range pairs {
			if _, exists := data[k]; exists {
				fmt.Fprintf(os.Stderr, "warning: command-line %s overrides value from file/stdin\n", k)
			}
			data[k] = v
		}
	}

	if len(data) == 0 {
		return errors.New("no keys provided (pass key=value arguments, or pipe JSON/YAML/env via -f or stdin)")
	}
	if err := validateKeys(data); err != nil {
		return err
	}

	formatter, err := formatterFor(setOutput, kind)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := formatter.Format(&buf, data); err != nil {
		return fmt.Errorf("formatting patch: %w", err)
	}
	payload := strings.TrimRight(buf.String(), "\n")

	if !setApply {
		fmt.Fprintln(os.Stdout, payload)
		return nil
	}

	if setNamespace == "" {
		setNamespace, err = resolveNamespace(setKubeconfig)
		if err != nil {
			return fmt.Errorf("resolving current namespace: %w", err)
		}
	}

	fmt.Fprintf(os.Stderr, "applying merge patch to %s/%s in namespace %s\n", kind, name, setNamespace)
	out, err := patchResource(kind, setNamespace, name, setKubeconfig, payload, setDryRun)
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, string(out))
	return nil
}

// parsePlaintext turns plaintext input into fully decoded key/value pairs -
// the values here are what the user typed, not base64. It mirrors parserFor
// on the decode side, but for the set/edit write path.
func parsePlaintext(format string, raw []byte, separator string) (DecodedData, error) {
	switch strings.ToLower(format) {
	case "json":
		return parsePlainJSON(raw, separator)
	case "yaml", "yml":
		return parsePlainYAML(raw, separator)
	case "env", "kv":
		return parseEnv(raw)
	case "any", "":
		for _, f := range []string{"json", "yaml", "env"} {
			parsed, err := parsePlaintext(f, raw, separator)
			if err == nil && len(parsed) > 0 {
				return parsed, nil
			}
		}
		return nil, errors.New("no plaintext format detected (want JSON, YAML, or key=value)")
	default:
		return nil, fmt.Errorf("unknown plaintext format %q (want any, env, json, or yaml)", format)
	}
}

func parsePlainJSON(raw []byte, separator string) (DecodedData, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}

	decoded := DecodedData{}
	if err := flatten(m, "", decoded, separator); err != nil {
		return nil, err
	}
	if len(decoded) == 0 {
		return nil, errors.New("no keys found")
	}
	return decoded, nil
}

func parsePlainYAML(raw []byte, separator string) (DecodedData, error) {
	var m map[string]any
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, err
	}

	decoded := DecodedData{}
	if err := flatten(m, "", decoded, separator); err != nil {
		return nil, err
	}
	if len(decoded) == 0 {
		return nil, errors.New("no keys found")
	}
	return decoded, nil
}

// flatten walks a decoded JSON/YAML value and emits one flat key per leaf.
// Maps nest with separator; arrays index as Key__0, Key__1 (dotnet's own
// representation of arrays in configuration keys).
func flatten(v any, prefix string, out DecodedData, separator string) error {
	switch val := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child := k
			if prefix != "" {
				child = prefix + separator + k
			}
			if err := flatten(val[k], child, out, separator); err != nil {
				return err
			}
		}
	case map[any]any:
		converted := make(map[string]any, len(val))
		for k, item := range val {
			converted[fmt.Sprint(k)] = item
		}
		return flatten(converted, prefix, out, separator)
	case []any:
		for i, item := range val {
			if err := flatten(item, fmt.Sprintf("%s%s%d", prefix, separator, i), out, separator); err != nil {
				return err
			}
		}
	case nil:
		out[prefix] = ""
	case string:
		out[prefix] = val
	case json.Number:
		out[prefix] = val.String()
	default:
		out[prefix] = fmt.Sprint(val)
	}
	return nil
}

// parseEnv parses "KEY=VALUE" pairs, one or more per line separated by commas.
// It only splits each pair on the first "=", so values may safely contain "=".
// Values are left verbatim (not trimmed) since whitespace can be significant.
func parseEnv(raw []byte) (DecodedData, error) {
	return parseEnvPairs(raw, true)
}

// parseEnvFile is the comma-free variant used for the edit round-trip: one
// KEY=VALUE per line, so a value containing a comma survives untouched.
func parseEnvFile(raw []byte) (DecodedData, error) {
	return parseEnvPairs(raw, false)
}

func parseEnvPairs(raw []byte, splitCommas bool) (DecodedData, error) {
	out := DecodedData{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		pairs := []string{line}
		if splitCommas {
			pairs = strings.Split(line, ",")
		}

		for _, pair := range pairs {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}
			key, value, ok := strings.Cut(pair, "=")
			if !ok {
				return nil, fmt.Errorf("malformed pair %q (want key=value)", pair)
			}
			key = strings.TrimSpace(key)
			if key == "" {
				return nil, fmt.Errorf("empty key in %q", pair)
			}
			if _, dup := out[key]; dup {
				return nil, fmt.Errorf("duplicate key %q", key)
			}
			out[key] = value
		}
	}
	return out, nil
}

var validKey = regexp.MustCompile(`^[-._a-zA-Z0-9]+$`)

func validateKeys(data DecodedData) error {
	for k := range data {
		if !validKey.MatchString(k) {
			return fmt.Errorf("invalid key %q: keys may only contain alphanumerics, '-', '_', or '.'", k)
		}
	}
	return nil
}
