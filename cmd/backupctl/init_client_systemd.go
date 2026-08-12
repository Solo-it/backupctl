package main

import (
	"fmt"
	"os"
)

const (
	clientScriptPath         = "/usr/local/bin/backup-pull.sh"
	clientSystemdServicePath = "/etc/systemd/system/backup-pull.service"
	clientSystemdTimerPath   = "/etc/systemd/system/backup-pull.timer"
)

// InitClientSystemd sets up the backup pull on a Linux client via systemd.
func InitClientSystemd(cfg *Config) error {
	if err := printClientSSHKey(); err != nil {
		return err
	}

	if err := writePullScript(cfg, clientScriptPath); err != nil {
		return err
	}

	calendar, err := toSystemdCalendar(cfg.Client.Schedule)
	if err != nil {
		return err
	}

	service := buildSystemdService(clientScriptPath)
	timer := buildSystemdTimer("Daily backup pull timer", calendar)

	if err := os.WriteFile(clientSystemdServicePath, []byte(service), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", clientSystemdServicePath, err)
	}
	if err := os.WriteFile(clientSystemdTimerPath, []byte(timer), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", clientSystemdTimerPath, err)
	}
	if err := runShell("systemctl", "daemon-reload"); err != nil {
		return err
	}
	if err := runShell("systemctl", "enable", "backup-pull.timer"); err != nil {
		return err
	}
	return runShell("systemctl", "start", "backup-pull.timer")
}
