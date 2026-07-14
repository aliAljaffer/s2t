package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"

	sscrypto "github.com/bitnami/sealed-secrets/pkg/crypto"
	"gopkg.in/yaml.v3"
)

// sealedSecretKeyEnvVar names the environment variable holding a path to the
// sealed-secrets controller's private key PEM file. A file path, not the key
// material itself, mirrors --kubeconfig: the key never touches argv or the
// env var's own value.
const sealedSecretKeyEnvVar = "S2T_SEALED_SECRETS_KEY_FILE"

// sealedSecretManifest covers only the fields needed to decrypt a
// SealedSecret's values: metadata for label derivation, and the encrypted
// data itself. Deliberately doesn't model spec.template.data (Go-template
// rendering) or the legacy single-blob spec.data field - both are advanced/
// legacy upstream features not needed to just decode and show values.
type sealedSecretManifest struct {
	Metadata struct {
		Name        string            `json:"name" yaml:"name"`
		Namespace   string            `json:"namespace" yaml:"namespace"`
		Annotations map[string]string `json:"annotations" yaml:"annotations"`
	} `json:"metadata" yaml:"metadata"`
	Spec struct {
		EncryptedData map[string]string `json:"encryptedData" yaml:"encryptedData"`
	} `json:"spec" yaml:"spec"`
}

// sealedSecretLabel reproduces sealed-secrets' own EncryptionLabel: RSA-OAEP
// decryption fails unless given the exact label used at encryption time, and
// that label is derived from the SealedSecret's scope annotations.
func sealedSecretLabel(m sealedSecretManifest) []byte {
	switch {
	case m.Metadata.Annotations["sealedsecrets.bitnami.com/cluster-wide"] == "true":
		return []byte("")
	case m.Metadata.Annotations["sealedsecrets.bitnami.com/namespace-wide"] == "true":
		return []byte(m.Metadata.Namespace)
	default:
		return []byte(m.Metadata.Namespace + "/" + m.Metadata.Name)
	}
}

// parseRSAPrivateKeyPEM reads a PEM-encoded RSA private key (PKCS1 or
// PKCS8), the two forms `kubeseal --fetch-cert`/controller key exports use.
func parseRSAPrivateKeyPEM(raw []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("no PEM data found")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}
	return rsaKey, nil
}

// loadSealedSecretKey reads the private key path from sealedSecretKeyEnvVar
// and parses it.
func loadSealedSecretKey() (*rsa.PrivateKey, error) {
	path := os.Getenv(sealedSecretKeyEnvVar)
	if path == "" {
		return nil, fmt.Errorf("%s must be set to the path of the sealed-secrets controller's private key PEM file", sealedSecretKeyEnvVar)
	}

	path, err := expandHome(path)
	if err != nil {
		return nil, fmt.Errorf("resolving %s: %w", sealedSecretKeyEnvVar, err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", sealedSecretKeyEnvVar, err)
	}

	key, err := parseRSAPrivateKeyPEM(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", sealedSecretKeyEnvVar, err)
	}
	return key, nil
}

// sealedSecretParser decrypts a SealedSecret manifest's encryptedData given
// the controller's private key. Unlike the other parsers, this one depends
// on external key material, so it's only reachable via an explicit
// --format sealed-secret - never through anyParser's auto-detect chain,
// where a missing key would otherwise look like "wrong format" instead of
// the clear, specific error it actually is.
type sealedSecretParser struct{}

func (sealedSecretParser) Parse(raw []byte) (SecretData, error) {
	privKey, err := loadSealedSecretKey()
	if err != nil {
		return nil, err
	}

	var m sealedSecretManifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, err
	}

	label := sealedSecretLabel(m)
	privKeys := map[string]*rsa.PrivateKey{"key": privKey}

	data := make(SecretData, len(m.Spec.EncryptedData))
	for key, encoded := range m.Spec.EncryptedData {
		ciphertext, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("key %q: not valid base64: %w", key, err)
		}

		plaintext, err := sscrypto.HybridDecrypt(rand.Reader, privKeys, ciphertext, label)
		if err != nil {
			return nil, fmt.Errorf("decrypting key %q: %w", key, err)
		}

		// Re-encode to base64: Parser.Parse is contractually supposed to
		// return still-base64-encoded SecretData, since decode() always
		// base64-decodes afterward. Encoding what we just decrypted looks
		// redundant, but it lets this format slot into the existing
		// pipeline (decode/formatters) with zero changes elsewhere.
		data[key] = base64.StdEncoding.EncodeToString(plaintext)
	}

	return data, nil
}
