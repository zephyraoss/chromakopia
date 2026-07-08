package server

import "testing"

func TestCoverageScore(t *testing.T) {
	tests := []struct {
		name     string
		hits     int
		queryLen int
		stride   int
		want     float64
	}{
		{"perfect", 100, 800, 8, 1.0},
		{"half", 50, 800, 8, 0.5},
		{"minimal", 3, 800, 8, 0.03},
		{"rounds to 2dp", 33, 800, 8, 0.33},
		{"clamped", 120, 800, 8, 1.0},
		{"ordinal cap", 256, 4096, 8, 1.0},
		{"zero query", 5, 0, 8, 0},
		{"zero stride", 5, 800, 0, 0},
		{"short query", 1, 5, 8, 1.0},
	}
	for _, tt := range tests {
		if got := coverageScore(tt.hits, tt.queryLen, tt.stride); got != tt.want {
			t.Errorf("%s: coverageScore(%d, %d, %d) = %v, want %v",
				tt.name, tt.hits, tt.queryLen, tt.stride, got, tt.want)
		}
	}
}
