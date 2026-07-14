package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

// Formatter writes fully decoded secret data to out in one specific shape.
// Every output mode implements this, mirroring the Parser interface on the
// input side.
type Formatter interface {
	Format(out io.Writer, data DecodedData) error
}

func formatterFor(output string) (Formatter, error) {
	switch output {
	case "":
		return plainFormatter{}, nil
	case "env":
		return envFormatter{}, nil
	case "json":
		return patchJSONFormatter{}, nil
	case "yaml":
		return patchYAMLFormatter{}, nil
	default:
		return nil, fmt.Errorf("unknown output format %q (want empty, env, json, or yaml)", output)
	}
}

type plainFormatter struct{}

func (plainFormatter) Format(out io.Writer, data DecodedData) error {
	keyColor := color.New(color.FgBlue)
	valueColor := color.New(color.FgGreen)

	for _, k := range sortedKeys(data) {
		keyColor.Fprintf(out, "%s: ", k)
		valueColor.Fprintf(out, "%s\n", data[k])
	}
	return nil
}

type envFormatter struct{}

func (envFormatter) Format(out io.Writer, data DecodedData) error {
	keyColor := color.New(color.FgBlue)
	valueColor := color.New(color.FgGreen)

	for _, k := range sortedKeys(data) {
		keyColor.Fprintf(out, "%s=", k)
		valueColor.Fprintf(out, "%s\n", data[k])
	}
	return nil
}

// patchManifest wraps decoded values in Kubernetes' stringData field. The
// API server merges stringData into data itself, base64-encoding server-side,
// so this needs no re-encoding and is directly usable as the payload for
// `kubectl patch secret NAME --type merge -p '<output>'`.
type patchManifest struct {
	StringData DecodedData `json:"stringData" yaml:"stringData"`
}

type patchJSONFormatter struct{}

func (patchJSONFormatter) Format(out io.Writer, data DecodedData) error {
	encoded, err := json.MarshalIndent(patchManifest{StringData: data}, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(encoded))
	return err
}

type patchYAMLFormatter struct{}

func (patchYAMLFormatter) Format(out io.Writer, data DecodedData) error {
	encoded, err := yaml.Marshal(patchManifest{StringData: data})
	if err != nil {
		return err
	}
	_, err = out.Write(encoded)
	return err
}
