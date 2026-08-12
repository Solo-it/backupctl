package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildWindowsPullScript(t *testing.T) {
	cfg := &Config{
		Client: ClientConfig{
			Remote:     "backup-user@192.168.1.10",
			RemoteRepo: "/home/backup-user/backups/restic-repo",
			LocalPath:  "C:\\backups\\mysql-restic-repo",
		},
	}

	script, err := buildWindowsPullScript(cfg)
	if err != nil {
		t.Fatalf("buildWindowsPullScript: %v", err)
	}

	for _, want := range []string{
		"backup-user@192.168.1.10:/home/backup-user/backups/restic-repo/",
		"C:\\backups\\mysql-restic-repo",
		"rsync -az --delete",
		"New-Item -ItemType Directory",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("buildWindowsPullScript() does not contain %q:\n%s", want, script)
		}
	}
}

func TestBuildWindowsPullScript_MissingFields(t *testing.T) {
	cases := []*Config{
		{Client: ClientConfig{RemoteRepo: "/repo", LocalPath: "C:\\local"}},
		{Client: ClientConfig{Remote: "user@host", LocalPath: "C:\\local"}},
		{Client: ClientConfig{Remote: "user@host", RemoteRepo: "/repo"}},
	}
	for _, cfg := range cases {
		if _, err := buildWindowsPullScript(cfg); err == nil {
			t.Errorf("buildWindowsPullScript(%+v) expected an error", cfg.Client)
		}
	}
}

func TestWriteWindowsPullScript(t *testing.T) {
	cfg := &Config{
		Client: ClientConfig{
			Remote:     "backup-user@192.168.1.10",
			RemoteRepo: "/home/backup-user/backups/restic-repo",
			LocalPath:  "C:\\backups\\mysql-restic-repo",
		},
	}
	path := filepath.Join(t.TempDir(), "sub", "backup-pull.ps1")

	if err := writeWindowsPullScript(cfg, path); err != nil {
		t.Fatalf("writeWindowsPullScript: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Errorf("file mode = %o, want 700", info.Mode().Perm())
	}
}

func TestWindowsPullScriptPath(t *testing.T) {
	path, err := windowsPullScriptPath()
	if err != nil {
		t.Fatalf("windowsPullScriptPath: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join("backupctl", "backup-pull.ps1")) {
		t.Errorf("path = %q, expected suffix backupctl/backup-pull.ps1", path)
	}
}
