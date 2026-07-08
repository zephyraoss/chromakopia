package fpcalc

import (
	"strings"
	"testing"
)

func TestParseOutput(t *testing.T) {
	out := "DURATION=216.42\nFINGERPRINT=1,2,-3,4294967295\nFILE=/tmp/x.mp3\n"
	res, err := ParseOutput(out)
	if err != nil {
		t.Fatalf("ParseOutput: %v", err)
	}
	if res.Duration != 216 {
		t.Errorf("Duration = %d, want 216", res.Duration)
	}
	want := []uint32{1, 2, 0xFFFFFFFD, 0xFFFFFFFF}
	if len(res.Values) != len(want) {
		t.Fatalf("Values = %v, want %v", res.Values, want)
	}
	for i, v := range want {
		if res.Values[i] != v {
			t.Errorf("Values[%d] = %d, want %d", i, res.Values[i], v)
		}
	}
}

func TestParseOutputCRLFAndUnknownKeys(t *testing.T) {
	out := "SOMETHING=else\r\nDURATION=10\r\nFINGERPRINT=5,6\r\n"
	res, err := ParseOutput(out)
	if err != nil {
		t.Fatalf("ParseOutput: %v", err)
	}
	if res.Duration != 10 || len(res.Values) != 2 || res.Values[0] != 5 || res.Values[1] != 6 {
		t.Errorf("got %+v", res)
	}
}

func TestParseOutputValueWithEquals(t *testing.T) {
	res, err := ParseOutput("DURATION=1\nFINGERPRINT=7\nFILE=/tmp/a=b.mp3\n")
	if err != nil {
		t.Fatalf("ParseOutput: %v", err)
	}
	if len(res.Values) != 1 || res.Values[0] != 7 {
		t.Errorf("got %+v", res)
	}
}

func TestParseOutputErrors(t *testing.T) {
	cases := map[string]string{
		"no fingerprint":  "DURATION=12\n",
		"empty output":    "",
		"bad duration":    "DURATION=abc\nFINGERPRINT=1,2\n",
		"bad value":       "DURATION=1\nFINGERPRINT=1,zap\n",
		"value too large": "DURATION=1\nFINGERPRINT=4294967296\n",
		"value too small": "DURATION=1\nFINGERPRINT=-2147483649\n",
	}
	for name, out := range cases {
		if _, err := ParseOutput(out); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestParseRawFingerprint(t *testing.T) {
	values, err := ParseRawFingerprint(" 10, -1 ,0 ")
	if err != nil {
		t.Fatalf("ParseRawFingerprint: %v", err)
	}
	want := []uint32{10, 0xFFFFFFFF, 0}
	for i, v := range want {
		if values[i] != v {
			t.Errorf("values[%d] = %d, want %d", i, values[i], v)
		}
	}

	if _, err := ParseRawFingerprint(""); err == nil {
		t.Error("empty input: expected error")
	}
	if _, err := ParseRawFingerprint("1,,2"); err == nil {
		t.Error("empty element: expected error")
	}
}

func TestRunnerReportsFpcalcErrors(t *testing.T) {
	r := &Runner{Path: "/nonexistent/fpcalc"}
	_, err := r.Run(t.Context(), "/tmp/nope.mp3")
	if err == nil || !strings.Contains(err.Error(), "fpcalc failed") {
		t.Errorf("expected fpcalc failure, got %v", err)
	}
}
