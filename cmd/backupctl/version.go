package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Solo-it/backupctl/internal/mdterm"
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
}

// printUpdateNoticeIfAny checks GitHub for a newer backupctl release and
// prints a boxed nudge if there is one — same idea as pnpm/npm's update
// notice. Deferred from main() for every command, so it always runs last
// and only on success (os.Exit(1) on any error path skips deferred calls
// entirely, which is exactly what we want — no update nagging after a
// failure). Every command it fires from is always run directly by a
// human, never unattended — backup.sh, which the systemd timer runs on a
// schedule, never invokes backupctl at all, so this never comes up there.
// Best-effort: any failure (offline, GitHub down, rate limited) is
// silently ignored, this is a courtesy, not a feature you can depend on.
func printUpdateNoticeIfAny() {
	latestTag, ok := latestGitHubReleaseTag("Solo-it", "backupctl")
	if !ok {
		return
	}
	latest, notify := shouldNotifyUpdate(version, latestTag)
	if !notify {
		return
	}
	fmt.Print(buildNoticeBox(
		fmt.Sprintf("Update available: %s -> %s", version, latest),
		[]string{
			"brew upgrade backupctl",
			"sudo apt update && sudo apt install --only-upgrade backupctl",
			"https://github.com/Solo-it/backupctl/releases",
		},
		mdterm.SupportsColor(),
	))
}

// buildNoticeBox is a pure function so the layout can be tested without a
// terminal. Bordered and colored like pnpm's update notice when color is
// available; plain text (still readable) otherwise.
func buildNoticeBox(headline string, body []string, color bool) string {
	lines := append([]string{headline}, body...)

	width := 0
	for _, l := range lines {
		if n := len([]rune(l)); n > width {
			width = n
		}
	}

	const (
		yellow = "\x1b[33m"
		bold   = "\x1b[1m"
		reset  = "\x1b[0m"
	)
	paint := func(s string) string {
		if !color {
			return s
		}
		return yellow + s + reset
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(paint("┌─" + strings.Repeat("─", width) + "─┐"))
	sb.WriteString("\n")
	for i, l := range lines {
		text := l
		if color && i == 0 {
			text = bold + l + reset // bold the headline
		}
		pad := width - len([]rune(l))
		sb.WriteString(paint("│") + " " + text + strings.Repeat(" ", pad) + " " + paint("│"))
		sb.WriteString("\n")
	}
	sb.WriteString(paint("└─" + strings.Repeat("─", width) + "─┘"))
	sb.WriteString("\n")
	return sb.String()
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

// latestGitHubReleaseTag fetches the tag_name of a repo's latest GitHub
// release (e.g. "v1.2.3"). Short timeout, no retries — a failure here
// should never hold up or break the command that called it.
func latestGitHubReleaseTag(owner, repo string) (string, bool) {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
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
