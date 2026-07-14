package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
