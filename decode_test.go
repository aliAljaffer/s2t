package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name        string
		data        SecretData
		want        DecodedData
		wantWarnHas string // substring expected in warnOut, "" if none expected
	}{
		{
			name: "decodes every value",
			data: SecretData{"username": "dXNlcg==", "password": "cGFzczEyMw=="},
			want: DecodedData{"username": "user", "password": "pass123"},
		},
		{
			name: "empty data decodes to empty map",
			data: SecretData{},
			want: DecodedData{},
		},
		{
			name:        "invalid base64 passes through raw with a warning",
			data:        SecretData{"broken": "not-valid-base64!!"},
			want:        DecodedData{"broken": "not-valid-base64!!"},
			wantWarnHas: "broken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var warnOut bytes.Buffer
			got := decode(&warnOut, tt.data)

			if len(got) != len(tt.want) {
				t.Fatalf("decode() = %v, want %v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("decode()[%q] = %q, want %q", k, got[k], v)
				}
			}

			switch {
			case tt.wantWarnHas == "" && warnOut.Len() != 0:
				t.Errorf("expected no warning, got %q", warnOut.String())
			case tt.wantWarnHas != "" && !strings.Contains(warnOut.String(), tt.wantWarnHas):
				t.Errorf("warnOut = %q, want it to contain %q", warnOut.String(), tt.wantWarnHas)
			}
		})
	}
}
