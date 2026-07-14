package main

import (
	"reflect"
	"testing"
)

func TestJSONParser(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    SecretData
		wantErr bool
	}{
		{
			name:  "valid manifest",
			input: `{"data":{"username":"dXNlcg==","password":"cGFzczEyMw=="}}`,
			want:  SecretData{"username": "dXNlcg==", "password": "cGFzczEyMw=="},
		},
		{
			name:  "empty data",
			input: `{"data":{}}`,
			want:  SecretData{},
		},
		{
			name:    "not json",
			input:   `not json at all`,
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonParser{}.Parse([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestYAMLParser(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    SecretData
		wantErr bool
	}{
		{
			name:  "valid manifest",
			input: "data:\n  username: dXNlcg==\n  password: cGFzczEyMw==\n",
			want:  SecretData{"username": "dXNlcg==", "password": "cGFzczEyMw=="},
		},
		{
			name:    "malformed yaml",
			input:   "data:\n  - this is not a map\n\tbad indent:\n",
			wantErr: true,
		},
		{
			name:  "missing data key produces empty map, not an error",
			input: "kind: Secret\n",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yamlParser{}.Parse([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKVParser(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    SecretData
		wantErr bool
	}{
		{
			name:  "single line, multiple pairs",
			input: "username: dXNlcg==,password: cGFzczEyMw==",
			want:  SecretData{"username": "dXNlcg==", "password": "cGFzczEyMw=="},
		},
		{
			name:  "multiple lines",
			input: "username: dXNlcg==\npassword: cGFzczEyMw==\n",
			want:  SecretData{"username": "dXNlcg==", "password": "cGFzczEyMw=="},
		},
		{
			name:  "blank lines and extra whitespace are ignored",
			input: "\n  username:   dXNlcg==  \n\n , password: cGFzczEyMw== ,\n",
			want:  SecretData{"username": "dXNlcg==", "password": "cGFzczEyMw=="},
		},
		{
			name:  "empty input yields empty map",
			input: "",
			want:  SecretData{},
		},
		{
			name:    "missing colon is malformed",
			input:   "username-dXNlcg==",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := kvParser{}.Parse([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParserFor(t *testing.T) {
	tests := []struct {
		format  string
		want    Parser
		wantErr bool
	}{
		{format: "json", want: jsonParser{}},
		{format: "JSON", want: jsonParser{}},
		{format: "yaml", want: yamlParser{}},
		{format: "yml", want: yamlParser{}},
		{format: "kv", want: kvParser{}},
		{format: "toml", wantErr: true},
		{format: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			got, err := parserFor(tt.format)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parserFor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parserFor() = %v, want %v", got, tt.want)
			}
		})
	}
}
