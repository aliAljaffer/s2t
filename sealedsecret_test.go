package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	sscrypto "github.com/bitnami/sealed-secrets/pkg/crypto"
	"gopkg.in/yaml.v3"
)

func generateTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating test key: %v", err)
	}
	return key
}

// writeKeyFile PEM-encodes key (PKCS1 if pkcs8 is false, PKCS8 otherwise)
// to a temp file and returns its path.
func writeKeyFile(t *testing.T, key *rsa.PrivateKey, pkcs8 bool) string {
	t.Helper()

	var block *pem.Block
	if pkcs8 {
		der, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			t.Fatalf("marshaling PKCS8 key: %v", err)
		}
		block = &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	} else {
		block = &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
	}

	path := filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("writing key file: %v", err)
	}
	return path
}

// sealManifest builds a sealedSecretManifest's raw YAML bytes, encrypting
// plaintextValues under the label implied by namespace/name/annotations -
// mirroring exactly what the sealed-secrets controller would produce.
func sealManifest(t *testing.T, pubKey *rsa.PublicKey, namespace, name string, annotations map[string]string, plaintextValues map[string]string) []byte {
	t.Helper()

	m := sealedSecretManifest{}
	m.Metadata.Name = name
	m.Metadata.Namespace = namespace
	m.Metadata.Annotations = annotations
	m.Spec.EncryptedData = map[string]string{}

	label := sealedSecretLabel(m)
	for key, value := range plaintextValues {
		ciphertext, err := sscrypto.HybridEncrypt(rand.Reader, pubKey, []byte(value), label)
		if err != nil {
			t.Fatalf("encrypting fixture key %q: %v", key, err)
		}
		m.Spec.EncryptedData[key] = base64.StdEncoding.EncodeToString(ciphertext)
	}

	raw, err := yaml.Marshal(m)
	if err != nil {
		t.Fatalf("marshaling fixture manifest: %v", err)
	}
	return raw
}

func TestSealedSecretParser(t *testing.T) {
	key := generateTestKey(t)

	tests := []struct {
		name        string
		namespace   string
		secretName  string
		annotations map[string]string
		pkcs8       bool
	}{
		{name: "strict scope (default)", namespace: "prod", secretName: "db-creds"},
		{name: "namespace-wide scope", namespace: "prod", secretName: "db-creds", annotations: map[string]string{"sealedsecrets.bitnami.com/namespace-wide": "true"}},
		{name: "cluster-wide scope", namespace: "prod", secretName: "db-creds", annotations: map[string]string{"sealedsecrets.bitnami.com/cluster-wide": "true"}},
		{name: "PKCS8 key", namespace: "prod", secretName: "db-creds", pkcs8: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := sealManifest(t, &key.PublicKey, tt.namespace, tt.secretName, tt.annotations, map[string]string{"password": "hunter2"})
			t.Setenv(sealedSecretKeyEnvVar, writeKeyFile(t, key, tt.pkcs8))

			data, err := (sealedSecretParser{}).Parse(raw)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			decoded, err := base64.StdEncoding.DecodeString(data["password"])
			if err != nil {
				t.Fatalf("Parse() returned non-base64 value: %v", err)
			}
			if string(decoded) != "hunter2" {
				t.Errorf("Parse()[password] = %q, want %q", decoded, "hunter2")
			}
		})
	}
}

func TestSealedSecretParserWrongScope(t *testing.T) {
	key := generateTestKey(t)

	// Encrypted for namespace "prod", but the manifest we hand to Parse
	// claims namespace "staging" - the label won't match, so decryption
	// must fail rather than silently succeed with the wrong secret.
	raw := sealManifest(t, &key.PublicKey, "prod", "db-creds", nil, map[string]string{"password": "hunter2"})

	var m sealedSecretManifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshaling fixture: %v", err)
	}
	m.Metadata.Namespace = "staging"
	tampered, err := yaml.Marshal(m)
	if err != nil {
		t.Fatalf("marshaling tampered fixture: %v", err)
	}

	t.Setenv(sealedSecretKeyEnvVar, writeKeyFile(t, key, false))

	if _, err := (sealedSecretParser{}).Parse(tampered); err == nil {
		t.Fatal("Parse() succeeded despite namespace/label mismatch, want error")
	}
}

func TestSealedSecretParserMissingEnvVar(t *testing.T) {
	t.Setenv(sealedSecretKeyEnvVar, "")

	_, err := (sealedSecretParser{}).Parse([]byte("metadata:\n  name: x\nspec:\n  encryptedData: {}\n"))
	if err == nil {
		t.Fatal("Parse() succeeded with no key env var set, want error")
	}
}

func TestSealedSecretParserMalformedPEM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(path, []byte("not a pem file"), 0o600); err != nil {
		t.Fatalf("writing malformed key file: %v", err)
	}
	t.Setenv(sealedSecretKeyEnvVar, path)

	_, err := (sealedSecretParser{}).Parse([]byte("metadata:\n  name: x\nspec:\n  encryptedData: {}\n"))
	if err == nil {
		t.Fatal("Parse() succeeded with malformed key PEM, want error")
	}
}

func TestSealedSecretLabel(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		secretName  string
		annotations map[string]string
		want        string
	}{
		{name: "strict (default)", namespace: "prod", secretName: "db-creds", want: "prod/db-creds"},
		{name: "namespace-wide", namespace: "prod", secretName: "db-creds", annotations: map[string]string{"sealedsecrets.bitnami.com/namespace-wide": "true"}, want: "prod"},
		{name: "cluster-wide", namespace: "prod", secretName: "db-creds", annotations: map[string]string{"sealedsecrets.bitnami.com/cluster-wide": "true"}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m sealedSecretManifest
			m.Metadata.Namespace = tt.namespace
			m.Metadata.Name = tt.secretName
			m.Metadata.Annotations = tt.annotations

			if got := string(sealedSecretLabel(m)); got != tt.want {
				t.Errorf("sealedSecretLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}
