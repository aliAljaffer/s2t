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

func TestIsNewer(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"0.2.3", "0.2.1", true},
		{"0.2.1", "0.2.1", false},
		{"0.2.0", "0.2.1", false},
		{"0.10.0", "0.9.9", true},
		{"1.0", "0.9.9", true},
		{"0.2.1", "dev", false},
	}
	for _, tt := range tests {
		if got := isNewer(tt.a, tt.b); got != tt.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestAssetName(t *testing.T) {
	if got := assetName("darwin", "arm64"); got != "s2t-darwin-arm64" {
		t.Errorf("got %q", got)
	}
	if got := assetName("windows", "amd64"); got != "s2t-windows-amd64.exe" {
		t.Errorf("got %q", got)
	}
}
