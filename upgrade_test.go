package main

import "testing"

func TestIsHomebrewPath(t *testing.T) {
	tests := map[string]bool{
		"/opt/homebrew/Cellar/s2t/0.2.1/bin/s2t":          true,
		"/usr/local/Cellar/s2t/0.2.1/bin/s2t":             true,
		"/home/linuxbrew/.linuxbrew/Cellar/s2t/1/bin/s2t": true,
		"/Users/h/.local/bin/s2t":                         false,
		"/usr/local/bin/s2t":                              false,
	}
	for p, want := range tests {
		if got := isHomebrewPath(p); got != want {
			t.Errorf("isHomebrewPath(%q) = %v, want %v", p, got, want)
		}
	}
}

func TestIsGoInstallPath(t *testing.T) {
	t.Setenv("GOBIN", "/tmp/gobin")
	if !isGoInstallPath("/tmp/gobin/s2t") {
		t.Error("binary in GOBIN should match")
	}
	if isGoInstallPath("/usr/local/bin/s2t") {
		t.Error("binary elsewhere should not match")
	}
}
