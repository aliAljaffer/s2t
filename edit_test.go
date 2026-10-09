package main

import (
	"reflect"
	"testing"
)

func TestChangePatchSecret(t *testing.T) {
	old := DecodedData{"keep": "same", "password": "old", "gone": "x"}
	edited := DecodedData{"keep": "same", "password": "new", "added": "y"}

	patch, changed := changePatch(kindSecret, old, edited)
	if !changed {
		t.Fatal("changePatch() changed = false, want true")
	}

	want := map[string]any{
		"stringData": map[string]any{"password": "new", "added": "y"},
		"data":       map[string]any{"gone": nil},
	}
	if !reflect.DeepEqual(patch, want) {
		t.Errorf("changePatch() = %#v, want %#v", patch, want)
	}
}

func TestChangePatchSecretNoChanges(t *testing.T) {
	data := DecodedData{"same": "value"}
	patch, changed := changePatch(kindSecret, data, data)
	if changed {
		t.Fatalf("changePatch() changed = true, want false (patch=%v)", patch)
	}
}

func TestChangePatchSecretPureAdd(t *testing.T) {
	patch, changed := changePatch(kindSecret, DecodedData{}, DecodedData{"new": "v"})
	if !changed {
		t.Fatal("changePatch() changed = false, want true")
	}
	if _, hasData := patch["data"]; hasData {
		t.Errorf("changePatch() included empty data key: %#v", patch)
	}
	want := map[string]any{"stringData": map[string]any{"new": "v"}}
	if !reflect.DeepEqual(patch, want) {
		t.Errorf("changePatch() = %#v, want %#v", patch, want)
	}
}

func TestChangePatchConfigMap(t *testing.T) {
	old := DecodedData{"feature": "off", "gone": "x"}
	edited := DecodedData{"feature": "on"}

	patch, changed := changePatch(kindConfigMap, old, edited)
	if !changed {
		t.Fatal("changePatch() changed = false, want true")
	}

	want := map[string]any{
		"data": map[string]any{"feature": "on", "gone": nil},
	}
	if !reflect.DeepEqual(patch, want) {
		t.Errorf("changePatch() = %#v, want %#v", patch, want)
	}
	if _, hasStringData := patch["stringData"]; hasStringData {
		t.Errorf("changePatch() ConfigMap used stringData: %#v", patch)
	}
}

func TestRenderEditableEnvRejectsNewline(t *testing.T) {
	if _, err := renderEditable("env", DecodedData{"multi": "a\nb"}); err == nil {
		t.Fatal("renderEditable(env) accepted a newline value, want error")
	}
}

func TestRenderEditableEnvRoundTrip(t *testing.T) {
	original := DecodedData{"a": "1", "b": "two words", "c": "has=equals"}
	rendered, err := renderEditable("env", original)
	if err != nil {
		t.Fatalf("renderEditable(env) error = %v", err)
	}
	parsed, err := parseEnvFile(rendered)
	if err != nil {
		t.Fatalf("parseEnvFile() error = %v", err)
	}
	if !reflect.DeepEqual(parsed, original) {
		t.Errorf("round trip = %v, want %v", parsed, original)
	}
}

func TestRenderEditableJSONRoundTrip(t *testing.T) {
	original := DecodedData{"a": "1", "b": "line1\nline2"}
	rendered, err := renderEditable("json", original)
	if err != nil {
		t.Fatalf("renderEditable(json) error = %v", err)
	}
	parsed, err := parsePlaintext("json", rendered, defaultSeparator)
	if err != nil {
		t.Fatalf("parsePlaintext(json) error = %v", err)
	}
	if !reflect.DeepEqual(parsed, original) {
		t.Errorf("round trip = %v, want %v", parsed, original)
	}
}
