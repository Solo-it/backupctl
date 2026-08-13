package main

import "testing"

func TestResticVersionRe(t *testing.T) {
	cases := []struct {
		output string
		want   string
		ok     bool
	}{
		{"restic 0.19.1 compiled with go1.23.4 on linux/amd64\n", "0.19.1", true},
		{"restic 0.16.0\n", "0.16.0", true},
		{"not restic output at all", "", false},
	}
	for _, c := range cases {
		m := resticVersionRe.FindStringSubmatch(c.output)
		if !c.ok {
			if m != nil {
				t.Errorf("expected no match for %q, got %v", c.output, m)
			}
			continue
		}
		if m == nil {
			t.Fatalf("expected a match for %q, got none", c.output)
		}
		if m[1] != c.want {
			t.Errorf("got %q, want %q", m[1], c.want)
		}
	}
}
