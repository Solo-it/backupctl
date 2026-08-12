package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildConfigYAML_Full(t *testing.T) {
	p := defaultGenConfigParams()
	p.Mode = "full"
	p.ClientRemote = "backup-user@192.168.1.10"

	yamlText, err := buildConfigYAML(p)
	if err != nil {
		t.Fatalf("buildConfigYAML: %v", err)
	}

	for _, want := range []string{"restic:", "backup_user:", "databases:", "schedule:", "client:", "remote: backup-user@192.168.1.10"} {
		if !strings.Contains(yamlText, want) {
			t.Errorf("full config doesn't contain %q:\n%s", want, yamlText)
		}
	}

	cfg, err := parseConfigYAML(t, yamlText)
	if err != nil {
		t.Fatalf("generated yaml doesn't parse: %v", err)
	}
	if cfg.BackupUser != p.BackupUser {
		t.Errorf("backup_user = %q, want %q", cfg.BackupUser, p.BackupUser)
	}
}

func TestBuildConfigYAML_ServerOnly(t *testing.T) {
	p := defaultGenConfigParams()
	p.Mode = "server"

	yamlText, err := buildConfigYAML(p)
	if err != nil {
		t.Fatalf("buildConfigYAML: %v", err)
	}
	if strings.Contains(yamlText, "client:") {
		t.Errorf("server config shouldn't contain a client: section:\n%s", yamlText)
	}
	if !strings.Contains(yamlText, "restic:") {
		t.Errorf("server config should contain a restic: section:\n%s", yamlText)
	}
}

func TestBuildConfigYAML_ClientOnly(t *testing.T) {
	p := defaultGenConfigParams()
	p.Mode = "client"

	yamlText, err := buildConfigYAML(p)
	if err != nil {
		t.Fatalf("buildConfigYAML: %v", err)
	}
	if strings.Contains(yamlText, "restic:") || strings.Contains(yamlText, "backup_user:") {
		t.Errorf("client config shouldn't contain server sections:\n%s", yamlText)
	}
	if !strings.Contains(yamlText, "client:") {
		t.Errorf("client config should contain a client: section:\n%s", yamlText)
	}
}

func TestBuildConfigYAML_InvalidMode(t *testing.T) {
	p := defaultGenConfigParams()
	p.Mode = "bogus"
	if _, err := buildConfigYAML(p); err == nil {
		t.Fatal("expected an error for an unknown mode")
	}
}

func TestGenConfig_DoesNotOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backup.yaml")
	writeFile(t, path, "existing: true")

	err := GenConfig(defaultGenConfigParams(), path, false)
	if err == nil {
		t.Fatal("expected an overwrite error without -force")
	}

	data, _ := os.ReadFile(path)
	if string(data) != "existing: true" {
		t.Error("the file was overwritten despite the missing -force")
	}
}

func TestGenConfig_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backup.yaml")
	writeFile(t, path, "existing: true")

	if err := GenConfig(defaultGenConfigParams(), path, true); err != nil {
		t.Fatalf("GenConfig with -force: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if strings.Contains(string(data), "existing: true") {
		t.Error("the file wasn't overwritten")
	}
}

// parseConfigYAML is a helper that parses yaml through the same LoadConfig.
func parseConfigYAML(t *testing.T, yamlText string) (*Config, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "backup.yaml")
	writeFile(t, path, yamlText)
	return LoadConfig(path)
}
