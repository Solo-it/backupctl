package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// version/commit/date are set via -ldflags at build time. GoReleaser's
// default ldflags template targets exactly these variable names, and
// {{.Version}} resolves to the tag *without* the leading "v" (e.g. "1.0.1").
//
//	-X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func printVersion() {
	fmt.Printf("backupctl %s (%s, %s)\n", version, commit, date)
	printUpdateNoticeIfAny()
}

// printUpdateNoticeIfAny checks GitHub for a newer release and prints a
// one-line nudge if there is one — same idea as pnpm/npm's update notice.
// Called from --version, --gen-config and --init-server — all three are
// always run directly by a human, never unattended. Deliberately NOT
// called from anything the systemd timer runs (backup.sh doesn't invoke
// backupctl at all, so this never comes up) — a backup tool has no
// business making surprise outbound calls on an unattended schedule.
// Best-effort: any failure (offline, GitHub down, rate limited) is
// silently ignored, this is a courtesy, not a feature you can depend on.
func printUpdateNoticeIfAny() {
	latestTag, ok := latestReleaseTag()
	if !ok {
		return
	}
	latest, notify := shouldNotifyUpdate(version, latestTag)
	if !notify {
		return
	}
	fmt.Printf("\nA newer version is available: %s (you have %s)\n", latest, version)
	fmt.Println("  brew upgrade backupctl  ·  sudo apt update && sudo apt install --only-upgrade backupctl  ·  https://github.com/Solo-it/backupctl/releases")
}

// shouldNotifyUpdate is a pure function: given the running version and the
// latest release tag from GitHub (which has a leading "v", unlike
// GoReleaser's {{.Version}}), decides whether to print an update notice.
// Uses an actual version comparison, not just "different string" — a
// locally built binary with a version string that happens to be newer
// than any release (a dev build with a fudged -X flag, say) should never
// be told to "upgrade" to something older.
func shouldNotifyUpdate(current, latestTag string) (latest string, notify bool) {
	if current == "dev" {
		return "", false
	}
	latest = strings.TrimPrefix(latestTag, "v")
	if latest == "" {
		return "", false
	}
	if compareVersions(latest, current) <= 0 {
		return "", false
	}
	return latest, true
}

// compareVersions compares two dot-separated numeric version strings
// (e.g. "1.2.10" vs "1.2.9"), returning -1/0/1 like strings.Compare.
// Missing/non-numeric components are treated as 0, so this stays a rough
// best-effort comparison rather than full semver (no pre-release/build
// metadata handling) — good enough for GitHub release tags like vX.Y.Z.
func compareVersions(a, b string) int {
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	for i := 0; i < len(ap) || i < len(bp); i++ {
		var av, bv int
		if i < len(ap) {
			av = atoiSafe(ap[i])
		}
		if i < len(bp) {
			bv = atoiSafe(bp[i])
		}
		if av != bv {
			if av < bv {
				return -1
			}
			return 1
		}
	}
	return 0
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func latestReleaseTag() (string, bool) {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/Solo-it/backupctl/releases/latest", nil)
	if err != nil {
		return "", false
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", false
	}
	return payload.TagName, true
}
