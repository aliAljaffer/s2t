package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() failed: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "bare tilde expands to home", path: "~", want: home},
		{name: "tilde slash expands relative to home", path: "~/.kube/config", want: filepath.Join(home, ".kube/config")},
		{name: "absolute path is unchanged", path: "/etc/kube/config", want: "/etc/kube/config"},
		{name: "empty path is unchanged", path: "", want: ""},
		{name: "tilde without slash is not expanded", path: "~foo/config", want: "~foo/config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandHome(tt.path)
			if err != nil {
				t.Fatalf("expandHome(%q) returned error: %v", tt.path, err)
			}
			if got != tt.want {
				t.Errorf("expandHome(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestKubeconfigArgs pins the behavior the KUBECONFIG-vs---kubeconfig fix
// depends on: an empty kubeconfig (no explicit --kubeconfig override) must
// produce no arguments at all, so the kubectl subprocess inherits KUBECONFIG
// from the environment and merges any colon-separated paths itself, rather
// than being handed a raw multi-path string via --kubeconfig, which kubectl
// treats as a single (nonexistent) file and fails on.
func TestKubeconfigArgs(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() failed: %v", err)
	}

	tests := []struct {
		name       string
		kubeconfig string
		want       []string
	}{
		{
			name:       "empty kubeconfig omits the flag entirely",
			kubeconfig: "",
			want:       nil,
		},
		{
			name:       "explicit path is passed through as --kubeconfig",
			kubeconfig: "/etc/kube/config",
			want:       []string{"--kubeconfig", "/etc/kube/config"},
		},
		{
			name:       "explicit tilde path is expanded before being passed",
			kubeconfig: "~/.kube/config",
			want:       []string{"--kubeconfig", filepath.Join(home, ".kube/config")},
		},
		{
			name:       "explicit multi-path value is passed through unsplit",
			kubeconfig: "/tmp/a:/tmp/b",
			want:       []string{"--kubeconfig", "/tmp/a:/tmp/b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := kubeconfigArgs(tt.kubeconfig)
			if err != nil {
				t.Fatalf("kubeconfigArgs(%q) returned error: %v", tt.kubeconfig, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("kubeconfigArgs(%q) = %#v, want %#v", tt.kubeconfig, got, tt.want)
			}
		})
	}
}

// TestFetchResourceJSONArgsOmitKubeconfigWhenUnset guards against a
// regression where an empty --kubeconfig accidentally re-appears as a
// literal "--kubeconfig ''" argument, which would override a KUBECONFIG the
// subprocess would otherwise inherit from the environment.
func TestFetchResourceJSONArgsOmitKubeconfigWhenUnset(t *testing.T) {
	kcArgs, err := kubeconfigArgs("")
	if err != nil {
		t.Fatalf("kubeconfigArgs(\"\") returned error: %v", err)
	}
	args := append(kcArgs, "get", "secret", "my-secret", "-n", "default", "-o", "json")

	for _, a := range args {
		if a == "--kubeconfig" {
			t.Fatalf("args unexpectedly contain --kubeconfig: %#v", args)
		}
	}

	want := []string{"get", "secret", "my-secret", "-n", "default", "-o", "json"}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("args = %#v, want %#v", args, want)
	}
}
