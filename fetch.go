package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// completionTimeout bounds how long a shell-completion kubectl call may run.
// Tab completion must never hang waiting on a slow or unreachable cluster.
const completionTimeout = 2 * time.Second

// fetchResourceJSON shells out to kubectl to fetch a live Secret or
// ConfigMap as JSON. kubectl's JSON output has the same data/binaryData
// shape as a raw manifest file, so jsonParser handles both without any
// extra code.
func fetchResourceJSON(kind, namespace, name, kubeconfig string) ([]byte, error) {
	kubeconfig, err := expandHome(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("resolving --kubeconfig: %w", err)
	}

	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig, "get", kind, name, "-n", namespace, "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && strings.Contains(string(exitErr.Stderr), "NotFound") {
			return nil, fmt.Errorf("%s %q not found in namespace %q", kind, name, namespace)
		}
		return nil, fmt.Errorf("kubectl get %s failed: %w", kind, err)
	}
	return out, nil
}

// currentNamespace asks kubectl for the current kubeconfig context's default
// namespace, falling back to "default" if the context doesn't set one -
// mirroring kubectl's own resolution when -n/--namespace is omitted.
func currentNamespace(ctx context.Context, kubeconfig string) (string, error) {
	kubeconfig, err := expandHome(kubeconfig)
	if err != nil {
		return "", err
	}

	out, err := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "config", "view", "--minify", "--output", "jsonpath={..namespace}").Output()
	if err != nil {
		return "", err
	}

	ns := strings.TrimSpace(string(out))
	if ns == "" {
		ns = "default"
	}
	return ns, nil
}

// resolveNamespace is the runtime entry point for currentNamespace: no
// artificial timeout (an actual fetch should behave like a normal kubectl
// invocation), and failures are surfaced as a real error.
func resolveNamespace(kubeconfig string) (string, error) {
	ns, err := currentNamespace(context.Background(), kubeconfig)
	if err != nil {
		return "", fmt.Errorf("determining current namespace: %w", err)
	}
	return ns, nil
}

// kubectlCurrentNamespace is the completion entry point for currentNamespace:
// bounded by completionTimeout, and any failure degrades to "" (no
// completions) rather than an error, consistent with kubectlResourceNames.
func kubectlCurrentNamespace(kubeconfig string) string {
	ctx, cancel := context.WithTimeout(context.Background(), completionTimeout)
	defer cancel()

	ns, err := currentNamespace(ctx, kubeconfig)
	if err != nil {
		return ""
	}
	return ns
}

// kubectlResourceNames lists the names of a resource kind (e.g. "namespace",
// "secret") via kubectl, for use in shell completion. Any failure - kubectl
// missing, no cluster reachable, timeout - returns an empty list rather than
// an error, since tab completion should degrade silently, never crash or hang
// the user's shell.
func kubectlResourceNames(kind, namespace, kubeconfig string) []string {
	kubeconfig, err := expandHome(kubeconfig)
	if err != nil {
		return nil
	}

	args := []string{"--kubeconfig", kubeconfig, "get", kind, "-o", "name"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	ctx, cancel := context.WithTimeout(context.Background(), completionTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "kubectl", args...).Output()
	if err != nil {
		return nil
	}

	var names []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		names = append(names, line[strings.LastIndex(line, "/")+1:])
	}
	return names
}

// expandHome resolves a leading "~" to the current user's home directory.
// exec.Command bypasses the shell, so "~" is never expanded automatically
// the way it would be when kubectl is run directly from a terminal.
func expandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}
