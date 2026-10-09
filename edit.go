package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	editNamespace  string
	editKubeconfig string
	editOutput     string
)

var editCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit a live Secret's or ConfigMap's values in $EDITOR",
	Long: `edit fetches a live Secret or ConfigMap, shows only its data keys (no metadata,
labels, or annotations) in $EDITOR, then prints a merge patch for whatever you
changed. Added and changed keys go through stringData (the API server base64-
encodes them); deleted keys are removed with a null in data.

Nothing is written to the cluster until you confirm at the prompt.`,
	Example: `  s2t edit db-creds -n prod
  s2t edit cm/app-config -n prod -o yaml`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, args[0])
	},
}

func init() {
	flags := editCmd.Flags()
	flags.StringVarP(&editNamespace, "namespace", "n", "", "kubernetes namespace of the resource (defaults to the kubeconfig's current context)")
	flags.StringVarP(&editKubeconfig, "kubeconfig", "", "", "path to the kubeconfig file to use; if omitted, kubectl resolves it itself from $KUBECONFIG or ~/.kube/config")
	flags.StringVarP(&editOutput, "output", "o", "env", "edit format: env (KEY=value), yaml, or json")
}

func runEdit(cmd *cobra.Command, name string) error {
	switch editOutput {
	case "env", "yaml", "yml", "json":
	default:
		return fmt.Errorf("invalid -o %q for edit (want env, yaml, or json)", editOutput)
	}

	if parsedKind, plainName, ok, err := splitKindName(name); err != nil {
		return err
	} else if ok {
		if cmd.Flags().Changed("kind") && kind != parsedKind {
			return fmt.Errorf("--kind %s conflicts with resource kind %q in %q", kind, parsedKind, name)
		}
		kind, name = parsedKind, plainName
	}

	namespace := editNamespace
	if namespace == "" {
		var err error
		namespace, err = resolveNamespace(editKubeconfig)
		if err != nil {
			return fmt.Errorf("resolving current namespace: %w", err)
		}
	}

	raw, err := fetchResourceJSON(kind, namespace, name, editKubeconfig)
	if err != nil {
		return err
	}
	parser, err := parserFor("json", kind)
	if err != nil {
		return err
	}
	secretData, err := parser.Parse(raw)
	if err != nil {
		return fmt.Errorf("parsing input: %w", err)
	}
	original := decode(os.Stderr, secretData)

	for k, v := range original {
		if !utf8.ValidString(v) {
			return fmt.Errorf("key %q contains non-UTF8 binary data; s2t edit only edits text values", k)
		}
	}

	initial, err := renderEditable(editOutput, original)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp("", "s2t-edit-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(initial); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := openEditor(tmpPath); err != nil {
		return err
	}

	editedRaw, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("reading edited file: %w", err)
	}

	var edited DecodedData
	if editOutput == "env" {
		edited, err = parseEnvFile(editedRaw)
	} else {
		edited, err = parsePlaintext(editOutput, editedRaw, defaultSeparator)
	}
	if err != nil {
		return fmt.Errorf("parsing edited values: %w", err)
	}
	if err := validateKeys(edited); err != nil {
		return err
	}

	patch, changed := changePatch(kind, original, edited)
	if !changed {
		fmt.Fprintln(os.Stderr, "no changes")
		return nil
	}

	payload, err := json.MarshalIndent(patch, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(payload))

	if !confirmApply() {
		return nil
	}

	out, err := patchResource(kind, namespace, name, editKubeconfig, string(payload), false)
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, string(out))
	return nil
}

// renderEditable renders only the data map in the chosen human-editable
// format. env cannot represent newlines, so it refuses rather than silently
// truncating a value; yaml/json are the escape hatch.
func renderEditable(format string, data DecodedData) ([]byte, error) {
	switch format {
	case "env":
		var b strings.Builder
		for _, k := range sortedKeys(data) {
			if strings.ContainsAny(data[k], "\n\r") {
				return nil, fmt.Errorf("key %q contains a newline; re-run with -o yaml or -o json", k)
			}
			fmt.Fprintf(&b, "%s=%s\n", k, data[k])
		}
		return []byte(b.String()), nil
	case "yaml", "yml":
		return yaml.Marshal(map[string]string(data))
	case "json":
		return json.MarshalIndent(map[string]string(data), "", "  ")
	default:
		return nil, fmt.Errorf("invalid -o %q for edit (want env, yaml, or json)", format)
	}
}

// changePatch builds a merge patch from the difference between old and new.
// Secret additions/updates use stringData (server-side base64); deletions use
// a null in data, which JSON merge patch treats as a remove. ConfigMaps have
// no stringData, so everything goes in data as plaintext. Unchanged keys are
// omitted so the patch stays minimal.
func changePatch(kind string, old, edited DecodedData) (map[string]any, bool) {
	added := map[string]any{}
	for k, v := range edited {
		if oldV, ok := old[k]; !ok || oldV != v {
			added[k] = v
		}
	}

	var removed []string
	for k := range old {
		if _, ok := edited[k]; !ok {
			removed = append(removed, k)
		}
	}

	if len(added) == 0 && len(removed) == 0 {
		return nil, false
	}

	patch := map[string]any{}
	if kind == kindConfigMap {
		data := map[string]any{}
		for k, v := range added {
			data[k] = v
		}
		for _, k := range removed {
			data[k] = nil
		}
		patch["data"] = data
		return patch, true
	}

	if len(added) > 0 {
		patch["stringData"] = added
	}
	if len(removed) > 0 {
		data := map[string]any{}
		for _, k := range removed {
			data[k] = nil
		}
		patch["data"] = data
	}
	return patch, true
}

// confirmApply asks the user whether to apply the printed patch, defaulting to
// no on empty input or when stdin isn't interactive.
func confirmApply() bool {
	fmt.Fprint(os.Stderr, "Apply this patch? [y/N]: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

// openEditor launches $EDITOR (or $VISUAL, or vi) on path. The editor value
// may carry arguments (e.g. "code --wait"), so it's split on whitespace.
func openEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	fields := strings.Fields(editor)
	cmd := exec.Command(fields[0], append(fields[1:], path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running editor %q: %w", editor, err)
	}
	return nil
}
