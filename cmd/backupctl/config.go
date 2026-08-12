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
	Type  string   `yaml:"type"`
	Names []string `yaml:"names"`
}

type ClientConfig struct {
	Remote     string `yaml:"remote"`
	RemoteRepo string `yaml:"remote_repo"`
	LocalPath  string `yaml:"local_path"`
	Schedule   string `yaml:"schedule"`
}

type Config struct {
	Restic     ResticConfig     `yaml:"restic"`
	BackupUser string           `yaml:"backup_user"`
	Databases  []DatabaseConfig `yaml:"databases"`
	Schedule   string           `yaml:"schedule"`
	Client     ClientConfig     `yaml:"client"`

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
