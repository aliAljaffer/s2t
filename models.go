package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SecretData maps a secret key to its still-base64-encoded value.
type SecretData map[string]string

// Parser turns raw input bytes into SecretData. Every input format implements
// this. Parse must always return still-base64-encoded values, even when the
// underlying source (e.g. a ConfigMap's plain-text data, or a decrypted
// SealedSecret value) isn't naturally base64 - callers re-encode as needed so
// that decode()'s always-base64-decode step works uniformly regardless of
// source.
type Parser interface {
	Parse(raw []byte) (SecretData, error)
}

// kindSecret and kindConfigMap are the two --kind values. They select how
// jsonParser/yamlParser interpret a manifest's data/binaryData fields -
// kvParser and sealedSecretParser ignore kind entirely, since neither format
// has a defined ConfigMap shape.
const (
	kindSecret    = "secret"
	kindConfigMap = "configmap"
)

// kindAliases maps every accepted --name kind prefix (e.g. "cm/my-cm") to its
// canonical kind constant.
var kindAliases = map[string]string{
	kindSecret:    kindSecret,
	kindConfigMap: kindConfigMap,
	"cm":          kindConfigMap,
}

// splitKindName parses a --name value that may carry a "kind/name" prefix,
// e.g. "secret/my-secret", "configmap/my-cm", or "cm/my-cm" (kubectl's
// TYPE/NAME convention). It returns ok=false when raw has no such prefix, so
// callers know to keep whatever --kind is already in effect.
func splitKindName(raw string) (kind, name string, ok bool, err error) {
	prefix, rest, found := strings.Cut(raw, "/")
	if !found {
		return "", raw, false, nil
	}
	if rest == "" {
		return "", "", false, fmt.Errorf("missing name after %q in --name %q", prefix+"/", raw)
	}
	kind, known := kindAliases[prefix]
	if !known {
		return "", "", false, fmt.Errorf("unknown resource kind %q in --name %q (want secret, configmap, or cm)", prefix, raw)
	}
	return kind, rest, true, nil
}

func parserFor(format, kind string) (Parser, error) {
	switch strings.ToLower(format) {
	case "yaml", "yml":
		return yamlParser{kind: kind}, nil
	case "json":
		return jsonParser{kind: kind}, nil
	case "kv":
		if kind == kindConfigMap {
			return nil, fmt.Errorf("--kind configmap is only supported with --format json, yaml, or any")
		}
		return kvParser{}, nil
	case "sealed-secret":
		if kind == kindConfigMap {
			return nil, fmt.Errorf("--kind configmap is only supported with --format json, yaml, or any")
		}
		return sealedSecretParser{}, nil
	case "any":
		return anyParser{kind: kind}, nil
	default:
		return nil, fmt.Errorf("unknown format %q (want yaml, json, kv, or sealed-secret)", format)
	}
}

// k8sManifest mirrors the two fields we care about in a real Kubernetes
// Secret or ConfigMap object. Secrets only ever populate Data (always
// base64). ConfigMaps populate Data with plain text and, optionally,
// BinaryData with base64 (for actual binary content).
type k8sManifest struct {
	Data       map[string]string `json:"data" yaml:"data"`
	BinaryData map[string]string `json:"binaryData" yaml:"binaryData"`
}

// normalizeForKind folds a manifest's data/binaryData into a single
// still-base64-encoded SecretData, per the Parser contract.
func normalizeForKind(kind string, m k8sManifest) (SecretData, error) {
	result := make(SecretData, len(m.Data)+len(m.BinaryData))

	if kind == kindConfigMap {
		for key, value := range m.Data {
			result[key] = base64.StdEncoding.EncodeToString([]byte(value))
		}
		for key, value := range m.BinaryData {
			if _, exists := result[key]; exists {
				return nil, fmt.Errorf("key %q present in both data and binaryData", key)
			}
			result[key] = value
		}
		return result, nil
	}

	// kindSecret (default): Data is already base64, unchanged from before
	// --kind existed.
	for key, value := range m.Data {
		result[key] = value
	}
	return result, nil
}

type jsonParser struct{ kind string }

func (p jsonParser) Parse(raw []byte) (SecretData, error) {
	var m k8sManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return normalizeForKind(p.kind, m)
}

type yamlParser struct{ kind string }

func (p yamlParser) Parse(raw []byte) (SecretData, error) {
	var m k8sManifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return normalizeForKind(p.kind, m)
}

// kvParser handles the custom "key1: value1,key2: value2" format, one record per line.
type kvParser struct{}

func (kvParser) Parse(raw []byte) (SecretData, error) {
	data := SecretData{}

	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		for _, pair := range strings.Split(line, ",") {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}

			parts := strings.SplitN(pair, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("malformed key:value pair %q", pair)
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			data[key] = value
		}
	}

	return data, nil
}

type anyParser struct{ kind string }

func (p anyParser) Parse(raw []byte) (SecretData, error) {
	formats := []string{"json", "yaml", "kv"}
	if p.kind == kindConfigMap {
		formats = []string{"json", "yaml"} // kv has no data/binaryData split
	}

	for _, f := range formats {
		parser, err := parserFor(f, p.kind)
		if err != nil {
			continue
		}
		secretData, err := parser.Parse(raw)
		if err != nil {
			continue
		}
		return secretData, nil
	}
	return nil, errors.New("no format detected")
}
