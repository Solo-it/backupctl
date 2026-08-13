package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	backupScriptPath   = "/root/backup.sh"
	systemdServicePath = "/etc/systemd/system/restic-backup.service"
	systemdTimerPath   = "/etc/systemd/system/restic-backup.timer"
)

// dbDumpCommand maps each supported database type to the tool it needs in
// PATH (checked by checkDumpToolsInstalled before anything is touched).
// supportedDBTypes keeps the display order for --gen-config and error
// messages.
var dbDumpCommand = map[string]string{
	"mysql":    "mysqldump",
	"postgres": "pg_dump",
	"redis":    "redis-cli",
	"rabbitmq": "rabbitmqctl",
}

var supportedDBTypes = []string{"mysql", "postgres", "redis", "rabbitmq"}

const backupExcludesPath = "/root/backup-excludes.txt"

// InitServer sets up the server: restic, database dump tools, backup-user,
// the repository, the dump script, and the systemd timer. Idempotent —
// running it again shouldn't break an already-configured server.
func InitServer(cfg *Config) error {
	steps := []func(*Config) error{
		checkResticInstalled,
		checkDumpToolsInstalled,
		ensureBackupUser,
		ensureRepoDirs,
		setRepoPermissions,
		ensureResticPassword,
		ensureResticRepoInit,
		func(cfg *Config) error { return writeBackupScript(cfg, backupScriptPath) },
		func(cfg *Config) error { return writeSystemdUnits(cfg, systemdServicePath, systemdTimerPath) },
		enableSystemdTimer,
	}
	for _, step := range steps {
		if err := step(cfg); err != nil {
			return err
		}
	}
	fmt.Println("--init-server: done")
	printInitServerNextSteps()
	return nil
}

func printInitServerNextSteps() {
	fmt.Print(`
What's next:
  1. Test a one-off run without waiting for the schedule:
       sudo systemctl start restic-backup.service
       journalctl -u restic-backup.service -n 50 --no-pager
  2. On the client that will pull the backups:
       backupctl --gen-config-client -out=backup-client.yaml -client-remote=` + "`whoami`" + `@<this-server-ip> ...
       backupctl --init-client-systemd -config=backup-client.yaml   # or pm2/launchd/task-scheduler
  3. On this server, allow the client's SSH key (the client will print the
     ready-to-run command in step 2):
       sudo backupctl --add-client-key -pubkey="ssh-ed25519 ..."

Added a new database to databases: in backup.yaml? Just run --init-server
again — it's idempotent and will rewrite backup.sh and the timer, nothing
gets broken.

Details: backupctl --info
`)
}

func checkResticInstalled(cfg *Config) error {
	if commandExists("restic") {
		fmt.Println("[1/10] restic is already installed")
		return nil
	}

	fmt.Println("[1/10] restic not found — trying to install it via a package manager")
	if err := installRestic(); err != nil {
		return fmt.Errorf("restic isn't installed and automatic install failed: %w — install it manually: https://restic.readthedocs.io/en/stable/020_installation.html", err)
	}
	if !commandExists("restic") {
		return fmt.Errorf("restic still not found in PATH after installing — check it manually")
	}
	fmt.Println("[1/10] restic installed")
	return nil
}

// packageManagers lists known package managers and the command to install
// restic through each of them, in order of preference.
var packageManagers = []struct {
	cmd  string
	args []string
}{
	{"apt-get", []string{"install", "-y", "restic"}},
	{"dnf", []string{"install", "-y", "restic"}},
	{"yum", []string{"install", "-y", "restic"}},
	{"apk", []string{"add", "restic"}},
	{"pacman", []string{"-Sy", "--noconfirm", "restic"}},
	{"brew", []string{"install", "restic"}},
}

// detectPackageManager is a pure function for tests: exists is injected
// instead of commandExists so the test doesn't depend on the machine's
// real PATH.
func detectPackageManager(exists func(string) bool) (cmd string, args []string, ok bool) {
	for _, m := range packageManagers {
		if exists(m.cmd) {
			return m.cmd, m.args, true
		}
	}
	return "", nil, false
}

func installRestic() error {
	cmd, args, ok := detectPackageManager(commandExists)
	if !ok {
		return fmt.Errorf("no known package manager found (apt-get/dnf/yum/apk/pacman/brew)")
	}
	if cmd == "apt-get" {
		if err := runShell("apt-get", "update"); err != nil {
			return err
		}
	}
	fmt.Printf("installing restic via %s\n", cmd)
	return runShell(cmd, args...)
}

// checkDumpToolsInstalled checks that for every database type in the
// config, the corresponding dump tool (mysqldump/pg_dump) is in PATH —
// without this check --init-server would succeed and the very first
// scheduled run would fail with a confusing error.
func checkDumpToolsInstalled(cfg *Config) error {
	seen := map[string]bool{}
	var missing []string
	for _, db := range cfg.Databases {
		cmdName, ok := dbDumpCommand[db.Type]
		if !ok {
			return fmt.Errorf("unknown database type %q, supported: %s", db.Type, strings.Join(supportedDBTypes, ", "))
		}
		if seen[cmdName] {
			continue
		}
		seen[cmdName] = true
		if !commandExists(cmdName) {
			missing = append(missing, cmdName)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("not found in PATH: %s — install before running", strings.Join(missing, ", "))
	}
	fmt.Println("[2/10] database dump tools are installed")
	return nil
}

// ensureBackupUser creates the backup-user system user with no password
// and explicitly locks password login (usermod -L) — the only way in is
// an SSH key, see --add-client-key.
func ensureBackupUser(cfg *Config) error {
	if cfg.BackupUser == "" {
		return fmt.Errorf("backup_user is not set in the config")
	}
	if userExists(cfg.BackupUser) {
		fmt.Printf("[3/10] user %s already exists\n", cfg.BackupUser)
		return nil
	}
	fmt.Printf("[3/10] creating user %s (no password, SSH key only)\n", cfg.BackupUser)
	if err := runShell("useradd", "--system", "--create-home", "--shell", "/usr/sbin/nologin", cfg.BackupUser); err != nil {
		return err
	}
	return runShell("usermod", "-L", cfg.BackupUser)
}

func ensureRepoDirs(cfg *Config) error {
	if cfg.Restic.Repo == "" {
		return fmt.Errorf("restic.repo is not set in the config")
	}
	dumpsDir := filepath.Join(filepath.Dir(cfg.Restic.Repo), "dumps")
	for _, dir := range []string{cfg.Restic.Repo, dumpsDir} {
		if fileExists(dir) {
			continue
		}
		fmt.Printf("[4/10] creating directory %s\n", dir)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}
	return nil
}

func setRepoPermissions(cfg *Config) error {
	root := filepath.Dir(cfg.Restic.Repo)
	fmt.Printf("[5/10] setting root:%s 750+setgid permissions on %s\n", cfg.BackupUser, root)
	if err := runShell("chown", "-R", "root:"+cfg.BackupUser, root); err != nil {
		return err
	}
	return runShell("chmod", "-R", "2750", root)
}

func ensureResticPassword(cfg *Config) error {
	if cfg.Restic.PasswordFile == "" {
		return fmt.Errorf("restic.password_file is not set in the config")
	}
	if fileExists(cfg.Restic.PasswordFile) {
		fmt.Printf("[6/10] restic password already exists (%s)\n", cfg.Restic.PasswordFile)
		return nil
	}

	if cfg.ResticPasswordOverride != "" {
		fmt.Printf("[6/10] writing the supplied restic password to %s\n", cfg.Restic.PasswordFile)
		if err := os.WriteFile(cfg.Restic.PasswordFile, []byte(cfg.ResticPasswordOverride+"\n"), 0o600); err != nil {
			return fmt.Errorf("writing password: %w", err)
		}
		return nil
	}

	fmt.Printf("[6/10] generating a restic password into %s\n", cfg.Restic.PasswordFile)
	pwd, err := generatePassword(32)
	if err != nil {
		return err
	}
	if err := os.WriteFile(cfg.Restic.PasswordFile, []byte(pwd+"\n"), 0o600); err != nil {
		return fmt.Errorf("writing password: %w", err)
	}
	printResticPasswordWarning(cfg.Restic.PasswordFile, pwd)
	return nil
}

// printResticPasswordWarning prints the just-generated password to the
// console once. The password only lives on this server, in
// password_file — if the server and its disk are lost, the client is
// left with only an encrypted copy of the repository, which can't be
// decrypted without this password. Save it somewhere else (a password
// manager, a safe) — backupctl will never show or copy it anywhere again.
func printResticPasswordWarning(passwordFile, pwd string) {
	fmt.Printf(`
⚠ RESTIC PASSWORD (save it separately — this is the only chance to decrypt
  the backups if the server is lost, backupctl will never show it again):

  %s

It has also been written to %s (owned by root, mode 600).

`, pwd, passwordFile)
}

func generatePassword(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating password: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func ensureResticRepoInit(cfg *Config) error {
	cmd := exec.Command("restic", "-r", cfg.Restic.Repo, "snapshots")
	cmd.Env = append(os.Environ(), "RESTIC_PASSWORD_FILE="+cfg.Restic.PasswordFile)
	out, err := cmd.CombinedOutput()
	if err == nil {
		fmt.Println("[7/10] restic repository is already initialized")
		return nil
	}
	if !strings.Contains(string(out), "unable to open config file") && !strings.Contains(string(out), "Is there a repository") {
		return fmt.Errorf("checking the restic repository: %s: %w", strings.TrimSpace(string(out)), err)
	}
	fmt.Printf("[7/10] initializing restic repository %s\n", cfg.Restic.Repo)
	initCmd := exec.Command("restic", "-r", cfg.Restic.Repo, "init")
	initCmd.Env = append(os.Environ(), "RESTIC_PASSWORD_FILE="+cfg.Restic.PasswordFile)
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("restic init: %w", err)
	}
	return nil
}

// buildBackupScript is a pure function that renders the contents of backup.sh.
func buildBackupScript(cfg *Config) (string, error) {
	dumpsDir := filepath.Join(filepath.Dir(cfg.Restic.Repo), "dumps")
	var sb strings.Builder
	sb.WriteString("#!/bin/sh\nset -e\n\n")
	// HOME is needed by restic under systemd (the service runs as root
	// with no User=), otherwise it can't find its cache directory and
	// warns about it in every log line.
	sb.WriteString(fmt.Sprintf("export HOME=/root\nDUMPS=%s\nexport RESTIC_PASSWORD_FILE=%s\nREPO=%s\n\n", dumpsDir, cfg.Restic.PasswordFile, cfg.Restic.Repo))

	for _, db := range cfg.Databases {
		if err := writeDatabaseDump(&sb, db); err != nil {
			return "", err
		}
	}

	var excludePatterns []string
	for _, f := range cfg.Files {
		patterns, err := buildExcludeList(f)
		if err != nil {
			return "", err
		}
		excludePatterns = append(excludePatterns, patterns...)
	}
	if len(excludePatterns) > 0 {
		sb.WriteString(fmt.Sprintf("\ncat > %s <<'BACKUPCTL_EXCLUDES'\n", backupExcludesPath))
		for _, p := range excludePatterns {
			sb.WriteString(p)
			sb.WriteString("\n")
		}
		sb.WriteString("BACKUPCTL_EXCLUDES\n")
	}

	sb.WriteString("\nrestic -r \"$REPO\" backup \"$DUMPS\"")
	for _, f := range cfg.Files {
		sb.WriteString(fmt.Sprintf(" %q", f.Path))
	}
	if len(excludePatterns) > 0 {
		sb.WriteString(" --exclude-file=" + backupExcludesPath)
	}
	for _, tag := range buildBackupTags(cfg) {
		sb.WriteString(" --tag " + tag)
	}
	sb.WriteString("\n")
	sb.WriteString("rm -f \"$DUMPS\"/*\n")
	return sb.String(), nil
}

// writeDatabaseDump appends the dump command(s) for one databases: entry.
// mysql/postgres are a simple "tool dbname > file.sql" per name; redis and
// rabbitmq have a single instance each (Names is ignored for them) and
// need more than one command, so they get their own blocks.
func writeDatabaseDump(sb *strings.Builder, db DatabaseConfig) error {
	switch db.Type {
	case "mysql":
		for _, name := range db.Names {
			sb.WriteString(fmt.Sprintf("mysqldump %s > \"$DUMPS/%s.sql\"\n", name, name))
		}
	case "postgres":
		for _, name := range db.Names {
			sb.WriteString(fmt.Sprintf("pg_dump %s > \"$DUMPS/%s.sql\"\n", name, name))
		}
	case "redis":
		// RDB files are LZF-compressed by default (rdbcompression yes),
		// which wrecks restic's content-defined deduplication — a tiny
		// data change can shift the whole compressed byte stream. Turn
		// compression off just for this dump (restic compresses the
		// repository itself anyway, after deduplication, which is the
		// right order) and restore whatever the setting actually was
		// before, not a hardcoded "yes".
		sb.WriteString("REDIS_DIR=$(redis-cli config get dir | tail -1)\n")
		sb.WriteString("REDIS_DBFILENAME=$(redis-cli config get dbfilename | tail -1)\n")
		sb.WriteString("REDIS_RDBCOMPRESSION_ORIG=$(redis-cli config get rdbcompression | tail -1)\n")
		sb.WriteString("redis-cli config set rdbcompression no\n")
		sb.WriteString("redis-cli bgsave\n")
		sb.WriteString("while [ \"$(redis-cli info persistence | grep -c 'rdb_bgsave_in_progress:1')\" = \"1\" ]; do sleep 1; done\n")
		sb.WriteString("redis-cli config set rdbcompression \"$REDIS_RDBCOMPRESSION_ORIG\"\n")
		sb.WriteString("cp \"$REDIS_DIR/$REDIS_DBFILENAME\" \"$DUMPS/redis.rdb\"\n")
		// ACLs aren't part of the RDB — save them, plus the full effective
		// config, so a restore isn't just data with no access control.
		sb.WriteString("redis-cli acl list > \"$DUMPS/redis-acl.txt\"\n")
		sb.WriteString("redis-cli config get '*' > \"$DUMPS/redis-config.txt\"\n")
	case "rabbitmq":
		// export_definitions already covers users+permissions (the ACL
		// equivalent), vhosts, policies, exchanges, queues and bindings —
		// nothing else needs a separate dump.
		sb.WriteString("rabbitmqctl export_definitions \"$DUMPS/rabbitmq-definitions.json\"\n")
	default:
		return fmt.Errorf("unknown database type %q, supported: %s", db.Type, strings.Join(supportedDBTypes, ", "))
	}
	return nil
}

// buildBackupTags returns a deduplicated, sorted list of restic tags for
// this run — one per configured database type, plus "files" and the
// preset name (if any) for each files: entry — so snapshots can be
// filtered later with `restic snapshots --tag redis`, etc.
func buildBackupTags(cfg *Config) []string {
	set := map[string]bool{}
	for _, db := range cfg.Databases {
		set[db.Type] = true
	}
	for _, f := range cfg.Files {
		set["files"] = true
		if f.Preset != "" && f.Preset != "none" {
			set[f.Preset] = true
		}
	}
	tags := make([]string, 0, len(set))
	for t := range set {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

func writeBackupScript(cfg *Config, path string) error {
	script, err := buildBackupScript(cfg)
	if err != nil {
		return err
	}
	fmt.Printf("[8/10] writing %s\n", path)
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// buildSystemdService/buildSystemdTimer are pure functions that render
// the unit file contents.
func buildSystemdService(execStart string) string {
	return fmt.Sprintf(`[Unit]
Description=Database backup via restic

[Service]
Type=oneshot
ExecStart=%s
`, execStart)
}

func buildSystemdTimer(description string, calendar string) string {
	return fmt.Sprintf(`[Unit]
Description=%s

[Timer]
OnCalendar=*-*-* %s
Persistent=true

[Install]
WantedBy=timers.target
`, description, calendar)
}

func writeSystemdUnits(cfg *Config, servicePath, timerPath string) error {
	calendar, err := toSystemdCalendar(cfg.Schedule)
	if err != nil {
		return err
	}
	// schedule in backup.yaml is documented as UTC (see the comment in
	// the generated config) — without the "UTC" suffix systemd
	// interprets OnCalendar as the server's local time, which silently
	// shifts the backup by the server's timezone offset from what the
	// user expected.
	service := buildSystemdService(backupScriptPath)
	timer := buildSystemdTimer("Daily database backup timer", calendar+" UTC")

	fmt.Println("[9/10] writing the systemd unit + timer")
	if err := os.WriteFile(servicePath, []byte(service), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", servicePath, err)
	}
	if err := os.WriteFile(timerPath, []byte(timer), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", timerPath, err)
	}
	return runShell("systemctl", "daemon-reload")
}

func enableSystemdTimer(cfg *Config) error {
	fmt.Println("[10/10] starting restic-backup.timer")
	if err := runShell("systemctl", "enable", "restic-backup.timer"); err != nil {
		return err
	}
	return runShell("systemctl", "start", "restic-backup.timer")
}
