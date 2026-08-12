package main

import "testing"

func TestToCron(t *testing.T) {
	cases := []struct {
		schedule string
		want     string
	}{
		{"02:00", "0 2 * * *"},
		{"04:30", "30 4 * * *"},
		{"23:59", "59 23 * * *"},
		{"0:0", "0 0 * * *"},
	}
	for _, c := range cases {
		got, err := toCron(c.schedule)
		if err != nil {
			t.Errorf("toCron(%q) error: %v", c.schedule, err)
			continue
		}
		if got != c.want {
			t.Errorf("toCron(%q) = %q, want %q", c.schedule, got, c.want)
		}
	}
}

func TestToCron_Invalid(t *testing.T) {
	cases := []string{"", "02", "25:00", "10:60", "aa:bb", "02:00:00"}
	for _, c := range cases {
		if _, err := toCron(c); err == nil {
			t.Errorf("toCron(%q) expected an error", c)
		}
	}
}

func TestToSystemdCalendar(t *testing.T) {
	got, err := toSystemdCalendar("04:30")
	if err != nil {
		t.Fatalf("toSystemdCalendar: %v", err)
	}
	if got != "04:30:00" {
		t.Errorf("toSystemdCalendar(04:30) = %q, want 04:30:00", got)
	}
}

func TestToSystemdCalendar_Invalid(t *testing.T) {
	if _, err := toSystemdCalendar("bad"); err == nil {
		t.Fatal("expected an error for an invalid schedule")
	}
}
