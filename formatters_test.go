package main

import (
	"bytes"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPlainFormatter(t *testing.T) {
	var out bytes.Buffer
	data := DecodedData{"username": "user", "password": "pass123"}

	if err := (plainFormatter{}).Format(&out, data); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	want := "password: pass123\nusername: user\n"
	if out.String() != want {
		t.Errorf("Format() = %q, want %q", out.String(), want)
	}
}

func TestEnvFormatter(t *testing.T) {
	var out bytes.Buffer
	data := DecodedData{"username": "user", "password": "pass123"}

	if err := (envFormatter{}).Format(&out, data); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	want := "password=pass123\nusername=user\n"
	if out.String() != want {
		t.Errorf("Format() = %q, want %q", out.String(), want)
	}
}

func TestPatchJSONFormatter(t *testing.T) {
	var out bytes.Buffer
	data := DecodedData{"username": "user", "password": "pass123"}

	if err := (patchJSONFormatter{kind: kindSecret}).Format(&out, data); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	want := "{\n  \"stringData\": {\n    \"password\": \"pass123\",\n    \"username\": \"user\"\n  }\n}\n"
	if out.String() != want {
		t.Errorf("Format() = %q, want %q", out.String(), want)
	}
}

func TestPatchJSONFormatterConfigMap(t *testing.T) {
	var out bytes.Buffer
	data := DecodedData{"config.yaml": "key: value"}

	if err := (patchJSONFormatter{kind: kindConfigMap}).Format(&out, data); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	want := "{\n  \"data\": {\n    \"config.yaml\": \"key: value\"\n  }\n}\n"
	if out.String() != want {
		t.Errorf("Format() = %q, want %q", out.String(), want)
	}
}

func TestPatchJSONCompactFormatter(t *testing.T) {
	var out bytes.Buffer
	data := DecodedData{"username": "user", "password": "pass123"}

	if err := (patchJSONCompactFormatter{kind: kindSecret}).Format(&out, data); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	want := "{\"stringData\":{\"password\":\"pass123\",\"username\":\"user\"}}\n"
	if out.String() != want {
		t.Errorf("Format() = %q, want %q", out.String(), want)
	}
}

func TestPatchYAMLFormatter(t *testing.T) {
	var out bytes.Buffer
	data := DecodedData{"username": "user", "password": "pass123"}

	if err := (patchYAMLFormatter{kind: kindSecret}).Format(&out, data); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Assert by round-tripping through yaml rather than comparing raw text,
	// since go-yaml's exact formatting is an implementation detail.
	var got map[string]DecodedData
	if err := yaml.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output isn't valid yaml: %v, got:\n%s", err, out.String())
	}
	want := map[string]DecodedData{"stringData": data}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Format() round-trip = %+v, want %+v", got, want)
	}
}

func TestPatchYAMLFormatterConfigMap(t *testing.T) {
	var out bytes.Buffer
	data := DecodedData{"config.yaml": "key: value"}

	if err := (patchYAMLFormatter{kind: kindConfigMap}).Format(&out, data); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var got map[string]DecodedData
	if err := yaml.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output isn't valid yaml: %v, got:\n%s", err, out.String())
	}
	want := map[string]DecodedData{"data": data}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Format() round-trip = %+v, want %+v", got, want)
	}
}

func TestFormatterFor(t *testing.T) {
	tests := []struct {
		output  string
		want    Formatter
		wantErr bool
	}{
		{output: "", want: plainFormatter{}},
		{output: "env", want: envFormatter{}},
		{output: "json", want: patchJSONFormatter{kind: kindSecret}},
		{output: "yaml", want: patchYAMLFormatter{kind: kindSecret}},
		{output: "toml", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.output, func(t *testing.T) {
			got, err := formatterFor(tt.output, kindSecret)
			if (err != nil) != tt.wantErr {
				t.Fatalf("formatterFor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("formatterFor() = %v, want %v", got, tt.want)
			}
		})
	}
}
