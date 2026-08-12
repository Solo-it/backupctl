package main

import (
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.yaml")
	writeFile(t, path, `
restic:
  repo: /home/backup-user/backups/restic-repo
  password_file: /root/.restic-env

backup_user: backup-user

databases:
  - type: mysql
    names:
      - app_db
      - shop_db
  - type: postgres
    names:
      - analytics

schedule: "02:00"

client:
  remote: backup-user@192.168.1.10
  remote_repo: /home/backup-user/backups/restic-repo
  local_path: ~/backups/mysql-restic-repo
  schedule: "04:00"
`)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Restic.Repo != "/home/backup-user/backups/restic-repo" {
		t.Errorf("restic.repo = %q", cfg.Restic.Repo)
	}
	if cfg.Restic.PasswordFile != "/root/.restic-env" {
		t.Errorf("restic.password_file = %q", cfg.Restic.PasswordFile)
	}
	if cfg.BackupUser != "backup-user" {
		t.Errorf("backup_user = %q", cfg.BackupUser)
	}
	if cfg.Schedule != "02:00" {
		t.Errorf("schedule = %q", cfg.Schedule)
	}
	if len(cfg.Databases) != 2 {
		t.Fatalf("len(databases) = %d, want 2", len(cfg.Databases))
	}
	if cfg.Databases[0].Type != "mysql" || len(cfg.Databases[0].Names) != 2 {
		t.Errorf("databases[0] = %+v", cfg.Databases[0])
	}
	if cfg.Databases[1].Type != "postgres" || len(cfg.Databases[1].Names) != 1 {
		t.Errorf("databases[1] = %+v", cfg.Databases[1])
	}
	if cfg.Client.Remote != "backup-user@192.168.1.10" {
		t.Errorf("client.remote = %q", cfg.Client.Remote)
	}
	if cfg.Client.Schedule != "04:00" {
		t.Errorf("client.schedule = %q", cfg.Client.Schedule)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	if _, err := LoadConfig(filepath.Join(t.TempDir(), "does-not-exist.yaml")); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.yaml")
	writeFile(t, path, "restic: [this is not a valid mapping")

	if _, err := LoadConfig(path); err == nil {
		t.Fatal("expected a parse error for invalid YAML")
	}
}
