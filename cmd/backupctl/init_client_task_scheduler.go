package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const taskSchedulerTaskName = "BackupctlPull"

// InitClientTaskScheduler sets up the backup pull on a Windows client via
// Task Scheduler (schtasks). Requires rsync in PATH — e.g. from WSL,
// cwRsync, or Git for Windows (Git Bash puts rsync.exe in PATH).
func InitClientTaskScheduler(cfg *Config) error {
	if !commandExists("schtasks") {
		return fmt.Errorf("schtasks not found — this command only works on Windows")
	}
	if !commandExists("rsync") {
		return fmt.Errorf("rsync not found in PATH — install it (WSL, cwRsync, or Git for Windows)")
	}
	if err := printClientSSHKey(); err != nil {
		return err
	}
	if _, err := toCron(cfg.Client.Schedule); err != nil {
		return err
	}

	scriptPath, err := windowsPullScriptPath()
	if err != nil {
		return err
	}
	if err := writeWindowsPullScript(cfg, scriptPath); err != nil {
		return err
	}

	taskCommand := fmt.Sprintf("powershell.exe -NoProfile -ExecutionPolicy Bypass -File \"%s\"", scriptPath)
	return runShell("schtasks", "/Create", "/F",
		"/SC", "DAILY",
		"/ST", cfg.Client.Schedule,
		"/TN", taskSchedulerTaskName,
		"/TR", taskCommand,
	)
}

func windowsPullScriptPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, "backupctl", "backup-pull.ps1"), nil
}

// buildWindowsPullScript is a pure function that renders a PowerShell
// script pulling the restic repository from the server over rsync via SSH.
func buildWindowsPullScript(cfg *Config) (string, error) {
	if cfg.Client.Remote == "" || cfg.Client.RemoteRepo == "" || cfg.Client.LocalPath == "" {
		return "", fmt.Errorf("client.remote / client.remote_repo / client.local_path are not set in the config")
	}
	localPath, err := expandHome(cfg.Client.LocalPath)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`$ErrorActionPreference = "Stop"
New-Item -ItemType Directory -Force -Path "%s" | Out-Null
rsync -az --delete %s:%s/ "%s"/
`, localPath, cfg.Client.Remote, cfg.Client.RemoteRepo, localPath), nil
}

func writeWindowsPullScript(cfg *Config, path string) error {
	script, err := buildWindowsPullScript(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	fmt.Printf("backup pull script written to %s\n", path)
	return nil
}
