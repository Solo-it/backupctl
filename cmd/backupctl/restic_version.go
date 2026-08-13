package main

import (
	"fmt"
	"os/exec"
	"regexp"

	"github.com/Solo-it/backupctl/internal/mdterm"
)

var resticVersionRe = regexp.MustCompile(`restic (\d+\.\d+\.\d+)`)

// installedResticVersion parses the version out of `restic version`
// (e.g. "restic 0.19.1 compiled with go1.23.4 on linux/amd64").
func installedResticVersion() (string, bool) {
	out, err := exec.Command("restic", "version").Output()
	if err != nil {
		return "", false
	}
	m := resticVersionRe.FindStringSubmatch(string(out))
	if m == nil {
		return "", false
	}
	return m[1], true
}

// printResticUpdateNoticeIfAny checks GitHub for a newer restic release
// and prints a boxed nudge if there is one. Only called from
// --init-server (a human-run command, never off a schedule) — same
// reasoning as printUpdateNoticeIfAny. restic ships its own upgrade
// mechanism (`restic self-update`), so there's no need to reimplement
// downloading/replacing the binary ourselves — just point at it.
// Best-effort: any failure is silently ignored.
func printResticUpdateNoticeIfAny() {
	current, ok := installedResticVersion()
	if !ok {
		return
	}
	latestTag, ok := latestGitHubReleaseTag("restic", "restic")
	if !ok {
		return
	}
	latest, notify := shouldNotifyUpdate(current, latestTag)
	if !notify {
		return
	}
	fmt.Print(buildNoticeBox(
		fmt.Sprintf("restic update available: %s -> %s", current, latest),
		[]string{
			"sudo restic self-update",
			"https://github.com/restic/restic/releases",
		},
		mdterm.SupportsColor(),
	))
}
