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

func formatterFor(output, kind string) (Formatter, error) {
	switch output {
	case "":
		return plainFormatter{}, nil
	case "env":
		return envFormatter{}, nil
	case "json":
		return patchJSONFormatter{kind: kind}, nil
	case "jsonc":
		return patchJSONCompactFormatter{kind: kind}, nil
	case "yaml":
		return patchYAMLFormatter{kind: kind}, nil
	default:
		return nil, fmt.Errorf("unknown output format %q (want empty, env, json, jsonc, or yaml)", output)
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

// patchPayload wraps decoded values in the field a `kubectl patch` expects
// for the given resource kind: Secrets take stringData (the API server
// base64-encodes it server-side); ConfigMaps have no stringData equivalent -
// their data field is already plain text, so decoded values go there
// directly. Either way this needs no re-encoding and is directly usable as
// the payload for `kubectl patch <kind> NAME --type merge -p '<output>'`.
func patchPayload(kind string, data DecodedData) map[string]DecodedData {
	key := "stringData"
	if kind == kindConfigMap {
		key = "data"
	}
	return map[string]DecodedData{key: data}
}

type patchJSONFormatter struct{ kind string }

func (f patchJSONFormatter) Format(out io.Writer, data DecodedData) error {
	encoded, err := json.MarshalIndent(patchPayload(f.kind, data), "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(encoded))
	return err
}

type patchJSONCompactFormatter struct{ kind string }

func (f patchJSONCompactFormatter) Format(out io.Writer, data DecodedData) error {
	encoded, err := json.Marshal(patchPayload(f.kind, data))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(encoded))
	return err
}

type patchYAMLFormatter struct{ kind string }

func (f patchYAMLFormatter) Format(out io.Writer, data DecodedData) error {
	encoded, err := yaml.Marshal(patchPayload(f.kind, data))
	if err != nil {
		return err
	}
	_, err = out.Write(encoded)
	return err
}
