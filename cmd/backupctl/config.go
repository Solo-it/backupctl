package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ResticConfig struct {
	Repo         string `yaml:"repo"`
	PasswordFile string `yaml:"password_file"`
}

type DatabaseConfig struct {
	Type string `yaml:"type"`
	// Names is ignored for types with a single instance and nothing to
	// enumerate (redis, rabbitmq) — leave it empty for those.
	Names []string `yaml:"names"`
}

// FileBackupConfig backs up a plain directory tree (a site's files, a
// user's home directory) alongside the database dumps, in the same
// restic repository. preset picks a built-in exclude list for known
// junk (caches, tmp files, archives) so backups don't balloon in size;
// exclude adds more patterns on top of it.
type FileBackupConfig struct {
	Path    string   `yaml:"path"`
	Preset  string   `yaml:"preset"`
	Exclude []string `yaml:"exclude"`
}

// OneCConfig backs up a file-mode 1C:Enterprise infobase (a directory
// containing 1Cv8.1CD) via `1cv8 DESIGNER`, in console mode. Unlike
// mysql/postgres/redis this needs a real filesystem path plus
// credentials, so it doesn't fit the databases: type+names shape.
//
// Exactly one of Password/PasswordFile must be set. Password sits in
// backup.yaml in cleartext if used — PasswordFile is the safer default
// (see the README for why, same reasoning as restic.password_file).
// Binary has no default: the exact install path varies by platform
// version (e.g. /opt/1cv8/x86_64/8.3.23.1912/1cv8) and guessing wrong
// would silently break backups, so it's required.
type OneCConfig struct {
	Name         string `yaml:"name"`
	Path         string `yaml:"path"`
	Binary       string `yaml:"binary"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	PasswordFile string `yaml:"password_file"`
	// DumpConfig also produces a separate .cf (configuration only, no
	// data) alongside the .dt full dump — useful for diffing config
	// changes over time, not needed just for disaster-recovery backups.
	DumpConfig bool `yaml:"dump_config"`
}

type ClientConfig struct {
	Remote     string `yaml:"remote"`
	RemoteRepo string `yaml:"remote_repo"`
	LocalPath  string `yaml:"local_path"`
	Schedule   string `yaml:"schedule"`
}

// RetentionConfig controls how many snapshots restic keeps after each
// backup run, via `restic forget --prune`. All-zero (the zero value,
// also what an absent retention: section unmarshals to) means "keep
// forever" — no forget command is run — matching restic's own default
// behavior when no policy is configured, so leaving this unset changes
// nothing for existing configs.
type RetentionConfig struct {
	KeepDaily   int `yaml:"keep_daily"`
	KeepWeekly  int `yaml:"keep_weekly"`
	KeepMonthly int `yaml:"keep_monthly"`
	KeepYearly  int `yaml:"keep_yearly"`
}

func (r RetentionConfig) isSet() bool {
	return r.KeepDaily > 0 || r.KeepWeekly > 0 || r.KeepMonthly > 0 || r.KeepYearly > 0
}

type Config struct {
	Restic     ResticConfig       `yaml:"restic"`
	BackupUser string             `yaml:"backup_user"`
	Databases  []DatabaseConfig   `yaml:"databases"`
	Files      []FileBackupConfig `yaml:"files"`
	OneC       []OneCConfig       `yaml:"1c"`
	Retention  RetentionConfig    `yaml:"retention"`
	Schedule   string             `yaml:"schedule"`
	Client     ClientConfig       `yaml:"client"`

	// ResticPasswordOverride is never read from backup.yaml (deliberately
	// no yaml tag): we don't want the password sitting in the file. It's
	// only ever passed once via the --init-server -restic-password flag,
	// to reuse a password from another server instead of generating a new one.
	ResticPasswordOverride string `yaml:"-"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return &cfg, nil
}
