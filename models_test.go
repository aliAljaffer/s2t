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
			got, err := jsonParser{kind: kindSecret}.Parse([]byte(tt.input))
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
			want:  SecretData{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yamlParser{kind: kindSecret}.Parse([]byte(tt.input))
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
		kind    string
		want    Parser
		wantErr bool
	}{
		{format: "json", kind: kindSecret, want: jsonParser{kind: kindSecret}},
		{format: "JSON", kind: kindSecret, want: jsonParser{kind: kindSecret}},
		{format: "yaml", kind: kindSecret, want: yamlParser{kind: kindSecret}},
		{format: "yml", kind: kindSecret, want: yamlParser{kind: kindSecret}},
		{format: "kv", kind: kindSecret, want: kvParser{}},
		{format: "toml", kind: kindSecret, wantErr: true},
		{format: "", kind: kindSecret, wantErr: true},
		{format: "kv", kind: kindConfigMap, wantErr: true},
		{format: "sealed-secret", kind: kindConfigMap, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.format+"/"+tt.kind, func(t *testing.T) {
			got, err := parserFor(tt.format, tt.kind)
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

func TestSplitKindName(t *testing.T) {
	tests := []struct {
		raw      string
		wantKind string
		wantName string
		wantOK   bool
		wantErr  bool
	}{
		{raw: "my-secret", wantOK: false, wantName: "my-secret"},
		{raw: "secret/my-secret", wantOK: true, wantKind: kindSecret, wantName: "my-secret"},
		{raw: "configmap/my-cm", wantOK: true, wantKind: kindConfigMap, wantName: "my-cm"},
		{raw: "cm/my-cm", wantOK: true, wantKind: kindConfigMap, wantName: "my-cm"},
		{raw: "cm/my-cm/extra", wantOK: true, wantKind: kindConfigMap, wantName: "my-cm/extra"},
		{raw: "pod/my-pod", wantErr: true},
		{raw: "cm/", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			kind, name, ok, err := splitKindName(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("splitKindName(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if ok != tt.wantOK || kind != tt.wantKind || name != tt.wantName {
				t.Errorf("splitKindName(%q) = (%q, %q, %v), want (%q, %q, %v)", tt.raw, kind, name, ok, tt.wantKind, tt.wantName, tt.wantOK)
			}
		})
	}
}

func TestNormalizeForKindConfigMap(t *testing.T) {
	m := k8sManifest{
		Data:       map[string]string{"config.yaml": "key: value"},
		BinaryData: map[string]string{"logo.png": "aGVsbG8="},
	}

	got, err := normalizeForKind(kindConfigMap, m)
	if err != nil {
		t.Fatalf("normalizeForKind() error = %v", err)
	}

	// config.yaml (plain text) must come back re-encoded as base64, since
	// Parser.Parse always returns still-base64-encoded SecretData.
	wantConfig := "a2V5OiB2YWx1ZQ=="
	if got["config.yaml"] != wantConfig {
		t.Errorf("normalizeForKind()[config.yaml] = %q, want %q", got["config.yaml"], wantConfig)
	}
	// logo.png (already base64 in binaryData) passes through unchanged.
	if got["logo.png"] != "aGVsbG8=" {
		t.Errorf("normalizeForKind()[logo.png] = %q, want %q", got["logo.png"], "aGVsbG8=")
	}
}

func TestNormalizeForKindSecretUnchanged(t *testing.T) {
	m := k8sManifest{Data: map[string]string{"password": "cGFzczEyMw=="}}

	got, err := normalizeForKind(kindSecret, m)
	if err != nil {
		t.Fatalf("normalizeForKind() error = %v", err)
	}
	if got["password"] != "cGFzczEyMw==" {
		t.Errorf("normalizeForKind()[password] = %q, want unchanged %q", got["password"], "cGFzczEyMw==")
	}
}

func TestNormalizeForKindCollision(t *testing.T) {
	m := k8sManifest{
		Data:       map[string]string{"dup": "plain"},
		BinaryData: map[string]string{"dup": "cGxhaW4="},
	}

	if _, err := normalizeForKind(kindConfigMap, m); err == nil {
		t.Fatal("normalizeForKind() succeeded despite key present in both data and binaryData, want error")
	}
}

func TestAnyParserConfigMapSkipsKV(t *testing.T) {
	// A kv-format input has no json/yaml structure, so with kind=configmap
	// (which only tries json/yaml) it must fail to detect a format, unlike
	// kind=secret (which would fall through to kv and succeed).
	input := []byte("username: dXNlcg==,password: cGFzczEyMw==")

	if _, err := (anyParser{kind: kindSecret}).Parse(input); err != nil {
		t.Fatalf("anyParser{kind: secret}.Parse() error = %v, want success via kv fallback", err)
	}

	if _, err := (anyParser{kind: kindConfigMap}).Parse(input); err == nil {
		t.Fatal("anyParser{kind: configmap}.Parse() succeeded on kv-shaped input, want error (kv should be skipped)")
	}
}
