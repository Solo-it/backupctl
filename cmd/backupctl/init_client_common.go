package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// pullScript generates a shell script that pulls the restic repository
// from the server to the local machine over rsync (backup-user only
// knows the restic-repo path — it never has the restic password itself).
func pullScript(cfg *Config) (string, error) {
	if cfg.Client.Remote == "" || cfg.Client.RemoteRepo == "" || cfg.Client.LocalPath == "" {
		return "", fmt.Errorf("client.remote / client.remote_repo / client.local_path are not set in the config")
	}
	localPath, err := expandHome(cfg.Client.LocalPath)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`#!/bin/sh
set -e
mkdir -p %q
rsync -az --delete %s:%s/ %q/
`, localPath, cfg.Client.Remote, cfg.Client.RemoteRepo, localPath)
	return script, nil
}

func expandHome(path string) (string, error) {
	if path == "~" || len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determining home directory: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func writePullScript(cfg *Config, path string) error {
	script, err := pullScript(cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	fmt.Printf("backup pull script written to %s\n", path)
	return nil
}
