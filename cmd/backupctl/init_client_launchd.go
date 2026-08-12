package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const launchdLabel = "io.github.solo-it.backupctl-pull"

// InitClientLaunchd sets up the backup pull on Mac via launchd.
func InitClientLaunchd(cfg *Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("determining home directory: %w", err)
	}
	if err := printClientSSHKey(); err != nil {
		return err
	}

	scriptPath := filepath.Join(home, ".local", "bin", "backup-pull.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(scriptPath), err)
	}
	if err := writePullScript(cfg, scriptPath); err != nil {
		return err
	}

	parts := strings.SplitN(cfg.Client.Schedule, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid schedule format %q, expected HH:MM", cfg.Client.Schedule)
	}
	hour, herr := strconv.Atoi(parts[0])
	minute, merr := strconv.Atoi(parts[1])
	if herr != nil || merr != nil {
		return fmt.Errorf("invalid schedule %q", cfg.Client.Schedule)
	}

	plistPath := filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist")
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
  </array>
  <key>StartCalendarInterval</key>
  <dict>
    <key>Hour</key>
    <integer>%d</integer>
    <key>Minute</key>
    <integer>%d</integer>
  </dict>
  <key>StandardOutPath</key>
  <string>%s/Library/Logs/%s.log</string>
  <key>StandardErrorPath</key>
  <string>%s/Library/Logs/%s.log</string>
</dict>
</plist>
`, launchdLabel, scriptPath, hour, minute, home, launchdLabel, home, launchdLabel)

	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(plistPath), err)
	}
	if err := os.WriteFile(plistPath, []byte(plist), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", plistPath, err)
	}
	fmt.Printf("launchd plist written to %s\n", plistPath)

	_ = runShell("launchctl", "unload", plistPath) // ignore the error if it isn't loaded yet
	return runShell("launchctl", "load", plistPath)
}
