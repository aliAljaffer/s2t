package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SecretData maps a secret key to its still-base64-encoded value.
type SecretData map[string]string

// Parser turns raw input bytes into SecretData. Every input format implements this.
type Parser interface {
	Parse(raw []byte) (SecretData, error)
}

func parserFor(format string) (Parser, error) {
	switch strings.ToLower(format) {
	case "yaml", "yml":
		return yamlParser{}, nil
	case "json":
		return jsonParser{}, nil
	case "kv":
		return kvParser{}, nil
	case "any":
		return anyParser{}, nil
	default:
		return nil, fmt.Errorf("unknown format %q (want yaml, json, or kv)", format)
	}
}

// k8sManifest mirrors the one field we care about in a real Kubernetes Secret
// object: "data", a map of key to base64-encoded value. Both a raw manifest
// file and `kubectl -o json/yaml` output share this shape.
type k8sManifest struct {
	Data map[string]string `json:"data" yaml:"data"`
}

type jsonParser struct{}

func (jsonParser) Parse(raw []byte) (SecretData, error) {
	var m k8sManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return SecretData(m.Data), nil
}

type yamlParser struct{}

func (yamlParser) Parse(raw []byte) (SecretData, error) {
	var m k8sManifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return SecretData(m.Data), nil
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

type anyParser struct{}

func (anyParser) Parse(raw []byte) (SecretData, error) {
	parsers := []string{"json", "yaml", "kv"}
	for _, p := range parsers {
		parser, err := parserFor(p)
		if err != nil {
			continue
		}
		secretData, err := parser.Parse(raw)
		if err != nil {
			continue
		}
		// fmt.Fprintf(os.Stderr, "%s format detected.\n", strings.ToUpper(p))
		return secretData, nil
	}
	return nil, errors.New("no format detected")
}
