package main

import (
	"context"
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

// fetchSecretJSON shells out to kubectl to fetch a live secret as JSON.
// kubectl's JSON output has the same "data" shape as a raw manifest file,
// so jsonParser handles both without any extra code.
func fetchSecretJSON(namespace, name, kubeconfig string) ([]byte, error) {
	kubeconfig, err := expandHome(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("resolving --kubeconfig: %w", err)
	}

	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig, "get", "secret", name, "-n", namespace, "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl get secret failed: %w", err)
	}
	return out, nil
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
