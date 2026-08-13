package main

import (
	"strings"
	"testing"
)

func TestShouldNotifyUpdate_NewerAvailable(t *testing.T) {
	latest, notify := shouldNotifyUpdate("1.0.1", "v1.0.2")
	if !notify {
		t.Fatal("expected a notification when a newer tag exists")
	}
	if latest != "1.0.2" {
		t.Errorf("latest = %q, want 1.0.2 (v prefix should be stripped)", latest)
	}
}

func TestShouldNotifyUpdate_AlreadyLatest(t *testing.T) {
	_, notify := shouldNotifyUpdate("1.0.1", "v1.0.1")
	if notify {
		t.Error("should not notify when already on the latest version")
	}
}

func TestShouldNotifyUpdate_AheadOfLatest(t *testing.T) {
	// A locally built binary with a version string ahead of any real
	// release should never be told to "upgrade" to something older.
	_, notify := shouldNotifyUpdate("99.0.0", "v1.0.1")
	if notify {
		t.Error("should not notify when the current version is already ahead of the latest tag")
	}
}

func TestShouldNotifyUpdate_DevBuildSkipped(t *testing.T) {
	_, notify := shouldNotifyUpdate("dev", "v1.0.1")
	if notify {
		t.Error("dev builds should never trigger an update notice")
	}
}

func TestShouldNotifyUpdate_EmptyLatestTag(t *testing.T) {
	_, notify := shouldNotifyUpdate("1.0.1", "")
	if notify {
		t.Error("an empty/unknown latest tag should not trigger a notice")
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.1", "1.0.1", 0},
		{"1.0.2", "1.0.1", 1},
		{"1.0.1", "1.0.2", -1},
		{"1.2.10", "1.2.9", 1},
		{"2.0.0", "1.9.9", 1},
		{"1.0", "1.0.0", 0},
		{"1.0.1", "1.0", 1},
	}
	for _, c := range cases {
		got := compareVersions(c.a, c.b)
		if got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestBuildNoticeBox_NoColor_PlainAndAligned(t *testing.T) {
	box := buildNoticeBox("Update available: 1.0.1 -> 1.1.0", []string{
		"brew upgrade backupctl",
		"https://github.com/Solo-it/backupctl/releases",
	}, false)
	if strings.Contains(box, "\x1b[") {
		t.Errorf("no ANSI codes expected with color=false:\n%s", box)
	}
	for _, want := range []string{"1.0.1", "1.1.0", "brew upgrade backupctl", "releases", "┌", "└", "│"} {
		if !strings.Contains(box, want) {
			t.Errorf("box missing %q:\n%s", want, box)
		}
	}
	lines := strings.Split(strings.Trim(box, "\n"), "\n")
	width := -1
	for _, l := range lines {
		n := len([]rune(l))
		if width == -1 {
			width = n
			continue
		}
		if n != width {
			t.Errorf("box lines have inconsistent width: %d vs %d, line %q", n, width, l)
		}
	}
}

func TestBuildNoticeBox_Color(t *testing.T) {
	box := buildNoticeBox("Update available: 1.0.1 -> 1.1.0", []string{
		"brew upgrade backupctl",
	}, true)
	if !strings.Contains(box, "\x1b[") {
		t.Errorf("expected ANSI codes with color=true:\n%q", box)
	}
	if !strings.Contains(box, "1.0.1") || !strings.Contains(box, "1.1.0") {
		t.Errorf("version numbers lost:\n%q", box)
	}
}
