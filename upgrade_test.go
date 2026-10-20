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
