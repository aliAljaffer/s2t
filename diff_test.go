package main

import (
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func b64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func writeManifestFile(t *testing.T, dir, name string, data map[string]string) string {
	t.Helper()

	var b strings.Builder
	b.WriteString("data:\n")
	for k, v := range data {
		b.WriteString("  " + k + ": " + v + "\n")
	}

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("writing fixture %s: %v", path, err)
	}
	return path
}

func TestSortedUnionKeys(t *testing.T) {
	a := DecodedData{"shared": "1", "only-a": "1"}
	b := DecodedData{"shared": "2", "only-b": "1"}

	got := sortedUnionKeys(a, b)
	want := []string{"only-a", "only-b", "shared"}

	if len(got) != len(want) {
		t.Fatalf("sortedUnionKeys() = %v, want %v", got, want)
	}
	for i, k := range want {
		if got[i] != k {
			t.Errorf("sortedUnionKeys()[%d] = %q, want %q", i, got[i], k)
		}
	}
}

func TestRunDiff(t *testing.T) {
	dir := t.TempDir()

	pathA := writeManifestFile(t, dir, "a.yaml", map[string]string{
		"unchanged": b64("same"),
		"removed":   b64("gone"),
		"changed":   b64("old-value"),
	})
	pathB := writeManifestFile(t, dir, "b.yaml", map[string]string{
		"unchanged": b64("same"),
		"added":     b64("new"),
		"changed":   b64("new-value"),
	})

	// runDiff writes to os.Stdout directly today; redirect it for the
	// duration of this test to capture output.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	errCh := make(chan error, 1)
	go func() {
		err := runDiff(pathA, pathB, "yaml", kindSecret, true, "", "", "", "")
		w.Close()
		errCh <- err
	}()

	outBytes, _ := io.ReadAll(r)
	os.Stdout = origStdout

	if err := <-errCh; err != nil {
		t.Fatalf("runDiff() error = %v", err)
	}

	out := string(outBytes)
	for _, want := range []string{"- removed: gone", "+ added: new", "~ changed: old-value -> new-value"} {
		if !strings.Contains(out, want) {
			t.Errorf("runDiff() output missing %q, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "unchanged") {
		t.Errorf("runDiff() output should omit unchanged keys, got:\n%s", out)
	}
}

func TestIsFile(t *testing.T) {
	dir := t.TempDir()
	filePath := writeManifestFile(t, dir, "a.yaml", map[string]string{"k": b64("v")})

	if !isFile(filePath) {
		t.Errorf("isFile(%q) = false, want true", filePath)
	}
	if isFile(dir) {
		t.Errorf("isFile(%q) = true for a directory, want false", dir)
	}
	if isFile(filepath.Join(dir, "does-not-exist")) {
		t.Errorf("isFile() = true for a nonexistent path, want false")
	}
	if isFile("db-creds") {
		t.Errorf("isFile(%q) = true for a bare resource name, want false", "db-creds")
	}
}

// TestDecodeSourceDispatchesToLiveFetch pins that a source which isn't a
// real file is treated as a live resource name, not a file-read error - the
// error should come from the (mocked-out-by-absence) kubectl call, not from
// os.ReadFile.
func TestDecodeSourceDispatchesToLiveFetch(t *testing.T) {
	_, err := decodeSource("does-not-exist-on-disk", "any", kindSecret, "default", "")
	if err == nil {
		t.Fatal("decodeSource() error = nil, want an error (no live cluster in test env)")
	}
	if strings.Contains(err.Error(), "no such file") {
		t.Errorf("decodeSource() = %v, want a live-fetch error, not a file-read error", err)
	}
}
