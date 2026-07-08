package fpcalc

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
)

type Result struct {
	Duration int
	Values   []uint32
}

type Runner struct {
	Path string
}

func (r *Runner) Run(ctx context.Context, path string) (*Result, error) {
	bin := r.Path
	if bin == "" {
		bin = "fpcalc"
	}
	cmd := exec.CommandContext(ctx, bin, "-raw", path)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := stdout.String()
	if err != nil || strings.Contains(strings.ToLower(out), "error:") {
		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = strings.TrimSpace(stderr.String())
		}
		if msg == "" {
			if err != nil {
				msg = err.Error()
			} else {
				msg = "unknown error"
			}
		}
		return nil, fmt.Errorf("fpcalc failed: %s", msg)
	}

	return ParseOutput(out)
}

func ParseOutput(out string) (*Result, error) {
	res := &Result{}
	var haveFingerprint bool
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		key, value, found := strings.Cut(strings.TrimRight(line, "\r"), "=")
		if !found {
			continue
		}
		switch key {
		case "DURATION":
			d, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid DURATION %q: %w", value, err)
			}
			res.Duration = int(math.Round(d))
		case "FINGERPRINT":
			values, err := ParseRawFingerprint(value)
			if err != nil {
				return nil, err
			}
			res.Values = values
			haveFingerprint = true
		}
	}
	if !haveFingerprint || len(res.Values) == 0 {
		return nil, errors.New("no fingerprint generated")
	}
	return res, nil
}

func ParseRawFingerprint(s string) ([]uint32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("empty fingerprint")
	}
	parts := strings.Split(s, ",")
	values := make([]uint32, 0, len(parts))
	for i, p := range parts {
		v, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("fingerprint value %d: %w", i, err)
		}
		if v < math.MinInt32 || v > math.MaxUint32 {
			return nil, fmt.Errorf("fingerprint value %d (%d) outside 32-bit range", i, v)
		}
		values = append(values, uint32(v))
	}
	return values, nil
}
