package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"sort"
)

// DecodedData maps a secret key to its already base64-decoded plaintext value.
type DecodedData map[string]string

// decode base64-decodes every value in data. Any value that isn't valid
// base64 is passed through as-is, with a warning written to warnOut.
func decode(warnOut io.Writer, data SecretData) DecodedData {
	decoded := make(DecodedData, len(data))

	for k, v := range data {
		raw, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			fmt.Fprintf(warnOut, "warning: %s is not valid base64, printing raw value\n", k)
			decoded[k] = v
			continue
		}
		decoded[k] = string(raw)
	}

	return decoded
}

// sortedKeys returns data's keys in sorted order. Map iteration order in Go
// is intentionally randomized, so formatters that print line-by-line sort
// first to get stable, reproducible output.
func sortedKeys(data DecodedData) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// maskPlaceholder replaces every value with a fixed-length placeholder,
// deliberately not preserving the real value's length - a length-preserving
// mask would still leak how long the real secret is.
const maskPlaceholder = "********"

// maskValues replaces every value in data with maskPlaceholder, for sharing
// output (screen-share, chat, CI logs) without leaking the actual secret.
func maskValues(data DecodedData) DecodedData {
	masked := make(DecodedData, len(data))
	for k := range data {
		masked[k] = maskPlaceholder
	}
	return masked
}
