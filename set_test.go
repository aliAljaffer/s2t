package main

import (
	"reflect"
	"testing"
)

func TestParseEnv(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    DecodedData
		wantErr bool
	}{
		{
			name:  "one pair per line",
			input: "username=admin\npassword=s3cret\n",
			want:  DecodedData{"username": "admin", "password": "s3cret"},
		},
		{
			name:  "comma separated on one line",
			input: "key1=val1,key2=val2",
			want:  DecodedData{"key1": "val1", "key2": "val2"},
		},
		{
			name:  "only first equals splits, so values keep equals signs",
			input: "conn=Server=db;User=admin",
			want:  DecodedData{"conn": "Server=db;User=admin"},
		},
		{
			name:  "blank lines and comments are skipped",
			input: "\n# comment\n\nusername=admin\n",
			want:  DecodedData{"username": "admin"},
		},
		{
			name:  "value with spaces is preserved verbatim",
			input: "greeting=hello world",
			want:  DecodedData{"greeting": "hello world"},
		},
		{
			name:    "missing equals is malformed",
			input:   "justakey",
			wantErr: true,
		},
		{
			name:    "duplicate key is rejected",
			input:   "a=1\na=2\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEnv([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseEnv() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseEnvFileNoCommaSplit pins the edit round-trip guarantee: a value that
// contains a comma must survive when it comes from the editor, unlike the
// comma-splitting parseEnv used for set's ad-hoc positional pairs.
func TestParseEnvFileNoCommaSplit(t *testing.T) {
	got, err := parseEnvFile([]byte("list=a,b,c\n"))
	if err != nil {
		t.Fatalf("parseEnvFile() error = %v", err)
	}
	want := DecodedData{"list": "a,b,c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseEnvFile() = %v, want %v", got, want)
	}
}

func TestParsePlainJSONFlatten(t *testing.T) {
	input := `{
		"ConnectionStrings": {"Default": "Server=db"},
		"Logging": {"LogLevel": {"Default": "Information"}},
		"AllowedHosts": "*",
		"port": 8080,
		"ratio": 1.5,
		"enabled": true,
		"nothing": null,
		"Servers": ["a", "b"]
	}`

	got, err := parsePlainJSON([]byte(input), defaultSeparator)
	if err != nil {
		t.Fatalf("parsePlainJSON() error = %v", err)
	}

	want := DecodedData{
		"ConnectionStrings__Default": "Server=db",
		"Logging__LogLevel__Default": "Information",
		"AllowedHosts":               "*",
		"port":                       "8080",
		"ratio":                      "1.5",
		"enabled":                    "true",
		"nothing":                    "",
		"Servers__0":                 "a",
		"Servers__1":                 "b",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parsePlainJSON() = %v, want %v", got, want)
	}
}

func TestParsePlainYAMLFlatten(t *testing.T) {
	input := "ConnectionStrings:\n  Default: Server=db\nSerilog:\n  MinimumLevel: Info\n"
	got, err := parsePlainYAML([]byte(input), defaultSeparator)
	if err != nil {
		t.Fatalf("parsePlainYAML() error = %v", err)
	}
	want := DecodedData{
		"ConnectionStrings__Default": "Server=db",
		"Serilog__MinimumLevel":      "Info",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parsePlainYAML() = %v, want %v", got, want)
	}
}

func TestParsePlaintextCustomSeparator(t *testing.T) {
	got, err := parsePlaintext("json", []byte(`{"a":{"b":"c"}}`), ":")
	if err != nil {
		t.Fatalf("parsePlaintext() error = %v", err)
	}
	if got["a:b"] != "c" {
		t.Errorf("parsePlaintext() custom separator = %v, want a:b=c", got)
	}
}

func TestParsePlaintextAutoDetect(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  DecodedData
	}{
		{
			name:  "json",
			input: `{"a":{"b":"c"}}`,
			want:  DecodedData{"a__b": "c"},
		},
		{
			name:  "yaml",
			input: "a:\n  b: c\n",
			want:  DecodedData{"a__b": "c"},
		},
		{
			name:  "env",
			input: "a=c,b=d",
			want:  DecodedData{"a": "c", "b": "d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePlaintext("any", []byte(tt.input), defaultSeparator)
			if err != nil {
				t.Fatalf("parsePlaintext(any) error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePlaintext(any) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParsePlaintextEmptyFails(t *testing.T) {
	if _, err := parsePlaintext("any", []byte("   \n"), defaultSeparator); err == nil {
		t.Fatal("parsePlaintext(any) succeeded on empty input, want error")
	}
}

func TestValidateKeys(t *testing.T) {
	valid := DecodedData{"ConnectionStrings__Default": "x", "feature.flag": "on", "api-key": "v", "A_1": "v"}
	if err := validateKeys(valid); err != nil {
		t.Errorf("validateKeys(valid) error = %v", err)
	}

	invalid := []DecodedData{
		{"bad key": "v"},
		{"bad/key": "v"},
		{"": "v"},
	}
	for _, data := range invalid {
		if err := validateKeys(data); err == nil {
			t.Errorf("validateKeys(%v) succeeded, want error", data)
		}
	}
}
