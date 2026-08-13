package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildBackupScript(t *testing.T) {
	cfg := &Config{
		Restic: ResticConfig{
			Repo:         "/home/backup-user/backups/restic-repo",
			PasswordFile: "/root/.restic-env",
		},
		Databases: []DatabaseConfig{
			{Type: "mysql", Names: []string{"app_db", "shop_db"}},
			{Type: "postgres", Names: []string{"analytics"}},
		},
	}

	script, err := buildBackupScript(cfg)
	if err != nil {
		t.Fatalf("buildBackupScript: %v", err)
	}

	for _, want := range []string{
		"mysqldump app_db",
		"mysqldump shop_db",
		"pg_dump analytics",
		"RESTIC_PASSWORD_FILE=/root/.restic-env",
		"restic -r \"$REPO\" backup",
		"export HOME=/root",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("backup script does not contain %q:\n%s", want, script)
		}
	}
}

func TestBuildBackupScript_UnknownDBType(t *testing.T) {
	cfg := &Config{
		Restic:    ResticConfig{Repo: "/repo", PasswordFile: "/pwd"},
		Databases: []DatabaseConfig{{Type: "oracle", Names: []string{"x"}}},
	}
	if _, err := buildBackupScript(cfg); err == nil {
		t.Fatal("expected an error for an unknown database type")
	}
}

func TestWriteBackupScript(t *testing.T) {
	cfg := &Config{
		Restic:    ResticConfig{Repo: "/repo", PasswordFile: "/pwd"},
		Databases: []DatabaseConfig{{Type: "mysql", Names: []string{"x"}}},
	}
	path := filepath.Join(t.TempDir(), "backup.sh")

	if err := writeBackupScript(cfg, path); err != nil {
		t.Fatalf("writeBackupScript: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Errorf("mode = %o, want 700", info.Mode().Perm())
	}
}

func TestBuildSystemdService(t *testing.T) {
	got := buildSystemdService("/root/backup.sh")
	for _, want := range []string{"[Service]", "Type=oneshot", "ExecStart=/root/backup.sh"} {
		if !strings.Contains(got, want) {
			t.Errorf("systemd service does not contain %q:\n%s", want, got)
		}
	}
}

func TestBuildSystemdTimer(t *testing.T) {
	got := buildSystemdTimer("Daily database backup timer", "02:00:00")
	for _, want := range []string{"[Timer]", "OnCalendar=*-*-* 02:00:00", "Persistent=true"} {
		if !strings.Contains(got, want) {
			t.Errorf("systemd timer does not contain %q:\n%s", want, got)
		}
	}
}

func TestWriteSystemdUnits(t *testing.T) {
	cfg := &Config{Schedule: "02:00"}
	dir := t.TempDir()
	servicePath := filepath.Join(dir, "backup.service")
	timerPath := filepath.Join(dir, "backup.timer")

	// systemctl may be missing entirely (macOS dev machines) or present but
	// non-functional (CI containers without a real systemd/polkit session,
	// where daemon-reload fails with "Interactive authentication required").
	// Either way we only care that the unit files themselves were generated
	// correctly — actually reloading systemd is out of scope for a unit test.
	_ = writeSystemdUnits(cfg, servicePath, timerPath)

	if !fileExists(servicePath) {
		t.Error("service file was not created")
	}
	if !fileExists(timerPath) {
		t.Error("timer file was not created")
	}

	timerContent, readErr := os.ReadFile(timerPath)
	if readErr != nil {
		t.Fatalf("reading %s: %v", timerPath, readErr)
	}
	if !strings.Contains(string(timerContent), "OnCalendar=*-*-* 02:00:00 UTC") {
		t.Errorf("timer does not set the time as explicit UTC (otherwise systemd would use the server local timezone):\n%s", timerContent)
	}
}

func TestWriteSystemdUnits_InvalidSchedule(t *testing.T) {
	cfg := &Config{Schedule: "bad"}
	dir := t.TempDir()
	err := writeSystemdUnits(cfg, filepath.Join(dir, "s.service"), filepath.Join(dir, "s.timer"))
	if err == nil {
		t.Fatal("expected an error for an invalid schedule")
	}
}

func TestCheckDumpToolsInstalled_UnknownDBType(t *testing.T) {
	cfg := &Config{Databases: []DatabaseConfig{{Type: "oracle", Names: []string{"x"}}}}
	if err := checkDumpToolsInstalled(cfg); err == nil {
		t.Fatal("expected an error for an unknown database type")
	}
}

func TestCheckDumpToolsInstalled_NoDatabases(t *testing.T) {
	if err := checkDumpToolsInstalled(&Config{}); err != nil {
		t.Fatalf("no error expected for an empty database list: %v", err)
	}
}

func TestCheckDumpToolsInstalled_MissingTool(t *testing.T) {
	orig := dbDumpCommand["mysql"]
	dbDumpCommand["mysql"] = "definitely-not-a-real-dump-tool-xyz"
	defer func() { dbDumpCommand["mysql"] = orig }()

	cfg := &Config{Databases: []DatabaseConfig{{Type: "mysql", Names: []string{"x"}}}}
	err := checkDumpToolsInstalled(cfg)
	if err == nil {
		t.Fatal("expected an error for a missing dump tool")
	}
	if !strings.Contains(err.Error(), "definitely-not-a-real-dump-tool-xyz") {
		t.Errorf("error does not name the missing tool: %v", err)
	}
}

func TestDetectPackageManager_PicksFirstAvailable(t *testing.T) {
	exists := func(name string) bool {
		return name == "dnf" || name == "brew"
	}
	cmd, args, ok := detectPackageManager(exists)
	if !ok {
		t.Fatal("expected a package manager to be found")
	}
	if cmd != "dnf" {
		t.Errorf("cmd = %q, want dnf (first by priority among those available)", cmd)
	}
	if len(args) == 0 || args[len(args)-1] != "restic" {
		t.Errorf("args does not contain restic: %v", args)
	}
}

func TestDetectPackageManager_NoneAvailable(t *testing.T) {
	exists := func(name string) bool { return false }
	_, _, ok := detectPackageManager(exists)
	if ok {
		t.Error("expected ok=false when no package manager is found")
	}
}

func TestBuildBackupScript_Redis(t *testing.T) {
	cfg := &Config{
		Restic:    ResticConfig{Repo: "/repo", PasswordFile: "/pwd"},
		Databases: []DatabaseConfig{{Type: "redis"}},
	}
	script, err := buildBackupScript(cfg)
	if err != nil {
		t.Fatalf("buildBackupScript: %v", err)
	}
	for _, want := range []string{
		"redis-cli config set rdbcompression no",
		"redis-cli bgsave",
		"redis-cli config set rdbcompression \"$REDIS_RDBCOMPRESSION_ORIG\"",
		"redis-cli acl list",
		"redis-cli config get '*'",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("redis backup script does not contain %q:\n%s", want, script)
		}
	}
}

func TestBuildBackupScript_Rabbitmq(t *testing.T) {
	cfg := &Config{
		Restic:    ResticConfig{Repo: "/repo", PasswordFile: "/pwd"},
		Databases: []DatabaseConfig{{Type: "rabbitmq"}},
	}
	script, err := buildBackupScript(cfg)
	if err != nil {
		t.Fatalf("buildBackupScript: %v", err)
	}
	if !strings.Contains(script, "rabbitmqctl export_definitions") {
		t.Errorf("rabbitmq backup script missing export_definitions:\n%s", script)
	}
}

func TestBuildBackupScript_FilesAddsPathsAndExcludeFile(t *testing.T) {
	cfg := &Config{
		Restic: ResticConfig{Repo: "/repo", PasswordFile: "/pwd"},
		Files: []FileBackupConfig{
			{Path: "/var/www/site", Preset: "wordpress"},
		},
	}
	script, err := buildBackupScript(cfg)
	if err != nil {
		t.Fatalf("buildBackupScript: %v", err)
	}
	for _, want := range []string{
		"\"/var/www/site\"",
		"--exclude-file=" + backupExcludesPath,
		"wp-content/cache/**",
		"cat > " + backupExcludesPath,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("backup script does not contain %q:\n%s", want, script)
		}
	}
}

func TestBuildBackupScript_NoFilesNoExcludeFile(t *testing.T) {
	cfg := &Config{
		Restic:    ResticConfig{Repo: "/repo", PasswordFile: "/pwd"},
		Databases: []DatabaseConfig{{Type: "mysql", Names: []string{"x"}}},
	}
	script, err := buildBackupScript(cfg)
	if err != nil {
		t.Fatalf("buildBackupScript: %v", err)
	}
	if strings.Contains(script, "--exclude-file") {
		t.Errorf("no files: configured, should not reference an exclude file:\n%s", script)
	}
}

func TestBuildBackupScript_TagsAppended(t *testing.T) {
	cfg := &Config{
		Restic: ResticConfig{Repo: "/repo", PasswordFile: "/pwd"},
		Databases: []DatabaseConfig{
			{Type: "mysql", Names: []string{"x"}},
			{Type: "redis"},
		},
		Files: []FileBackupConfig{
			{Path: "/var/www/site", Preset: "wordpress"},
		},
	}
	script, err := buildBackupScript(cfg)
	if err != nil {
		t.Fatalf("buildBackupScript: %v", err)
	}
	for _, want := range []string{"--tag files", "--tag mysql", "--tag redis", "--tag wordpress"} {
		if !strings.Contains(script, want) {
			t.Errorf("backup script does not contain %q:\n%s", want, script)
		}
	}
}

func TestBuildBackupTags_DeduplicatedAndSorted(t *testing.T) {
	cfg := &Config{
		Databases: []DatabaseConfig{
			{Type: "mysql", Names: []string{"a"}},
			{Type: "mysql", Names: []string{"b"}},
			{Type: "postgres", Names: []string{"c"}},
		},
		Files: []FileBackupConfig{
			{Path: "/a", Preset: "wordpress"},
			{Path: "/b", Preset: "wordpress"},
			{Path: "/c", Preset: "none"},
		},
	}
	got := buildBackupTags(cfg)
	want := []string{"files", "mysql", "postgres", "wordpress"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
			break
		}
	}
}
