package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("could not determine home directory: %v", err)
	}

	cases := []struct {
		in   string
		want string
	}{
		{"~", home},
		{"~/backups/repo", filepath.Join(home, "backups/repo")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}
	for _, c := range cases {
		got, err := expandHome(c.in)
		if err != nil {
			t.Errorf("expandHome(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("expandHome(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPullScript(t *testing.T) {
	cfg := &Config{
		Client: ClientConfig{
			Remote:     "backup-user@192.168.1.10",
			RemoteRepo: "/home/backup-user/backups/restic-repo",
			LocalPath:  "/tmp/local-repo",
		},
	}

	script, err := pullScript(cfg)
	if err != nil {
		t.Fatalf("pullScript: %v", err)
	}

	for _, want := range []string{
		"backup-user@192.168.1.10:/home/backup-user/backups/restic-repo/",
		"/tmp/local-repo",
		"rsync -az --delete",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("pullScript() does not contain %q:\n%s", want, script)
		}
	}
}

func TestPullScript_MissingFields(t *testing.T) {
	cases := []*Config{
		{Client: ClientConfig{RemoteRepo: "/repo", LocalPath: "/local"}},
		{Client: ClientConfig{Remote: "user@host", LocalPath: "/local"}},
		{Client: ClientConfig{Remote: "user@host", RemoteRepo: "/repo"}},
	}
	for _, cfg := range cases {
		if _, err := pullScript(cfg); err == nil {
			t.Errorf("pullScript(%+v) expected an error", cfg.Client)
		}
	}
}

func TestWritePullScript(t *testing.T) {
	cfg := &Config{
		Client: ClientConfig{
			Remote:     "backup-user@192.168.1.10",
			RemoteRepo: "/home/backup-user/backups/restic-repo",
			LocalPath:  "/tmp/local-repo",
		},
	}
	path := filepath.Join(t.TempDir(), "backup-pull.sh")

	if err := writePullScript(cfg, path); err != nil {
		t.Fatalf("writePullScript: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Errorf("file mode = %o, want 700", info.Mode().Perm())
	}
}
