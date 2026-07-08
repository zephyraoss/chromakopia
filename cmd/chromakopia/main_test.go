package main

import "testing"

func TestParseMode(t *testing.T) {
	for input, want := range map[string]modeSet{
		"identify": {identify: true},
		"metadata": {metadata: true},
		"both":     {identify: true, metadata: true},
	} {
		got, err := parseMode(input)
		if err != nil || got != want {
			t.Errorf("parseMode(%q) = (%+v, %v), want %+v", input, got, err, want)
		}
	}
	for _, input := range []string{"", "Identify", "catalog"} {
		if _, err := parseMode(input); err == nil {
			t.Errorf("parseMode(%q) succeeded, want error", input)
		}
	}
}

func TestValidateFlags(t *testing.T) {
	identify := modeSet{identify: true}
	metadata := modeSet{metadata: true}
	both := modeSet{identify: true, metadata: true}

	cases := []struct {
		name                            string
		mode                            modeSet
		dataset, metadataDB, metadataURL string
		wantErr                         bool
	}{
		{"identify needs dataset", identify, "", "", "", true},
		{"identify with dataset only", identify, "/d/ckaf", "", "", false},
		{"identify with local metadata", identify, "/d/ckaf", "/d/meta.db", "", false},
		{"identify with remote metadata", identify, "/d/ckaf", "", "http://node-b:3000", false},
		{"metadata needs db", metadata, "", "", "", true},
		{"metadata with db", metadata, "", "/d/meta.db", "", false},
		{"metadata rejects url", metadata, "", "/d/meta.db", "http://node-b:3000", true},
		{"metadata ignores dataset", metadata, "/d/ckaf", "/d/meta.db", "", false},
		{"both needs dataset", both, "", "/d/meta.db", "", true},
		{"both needs db", both, "/d/ckaf", "", "", true},
		{"both complete", both, "/d/ckaf", "/d/meta.db", "", false},
		{"both rejects url", both, "/d/ckaf", "/d/meta.db", "http://node-b:3000", true},
	}
	for _, tc := range cases {
		err := validateFlags(tc.mode, tc.dataset, tc.metadataDB, tc.metadataURL)
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: err = %v, wantErr = %t", tc.name, err, tc.wantErr)
		}
	}
}
