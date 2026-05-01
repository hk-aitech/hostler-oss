// Unit tests for normalizePriority.
package cmd

import "testing"

func TestNormalizePriority(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"p0", "p0", false},
		{"P0", "p0", false},
		{"P1", "p1", false},
		{"P2", "p2", false},
		{"P3", "p3", false},
		{"  p1 ", "p1", false}, // whitespace trim
		{"", "", false},        // optional flag allowed
		{"p4", "", true},       // outside enum
		{"critical", "", true},
		{"0", "", true},
		{"P9", "", true},
	}
	for _, c := range cases {
		got, err := normalizePriority(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("normalizePriority(%q) err=%v, wantErr=%v", c.in, err, c.wantErr)
			continue
		}
		if got != c.want {
			t.Errorf("normalizePriority(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
