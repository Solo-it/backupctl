package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	return buf.String()
}

func TestEnsureResticPassword_GeneratesAndPrintsWarning(t *testing.T) {
	pwdFile := filepath.Join(t.TempDir(), "restic-password")
	cfg := &Config{Restic: ResticConfig{PasswordFile: pwdFile}}

	var err error
	output := captureStdout(t, func() {
		err = ensureResticPassword(cfg)
	})
	if err != nil {
		t.Fatalf("ensureResticPassword: %v", err)
	}

	data, readErr := os.ReadFile(pwdFile)
	if readErr != nil {
		t.Fatalf("reading %s: %v", pwdFile, readErr)
	}
	pwd := strings.TrimSpace(string(data))
	if pwd == "" {
		t.Fatal("password was not written to the file")
	}

	if !strings.Contains(output, pwd) {
		t.Errorf("generated password was not printed to the console:\n%s", output)
	}
	if !strings.Contains(output, "save it") && !strings.Contains(output, "RESTIC PASSWORD") {
		t.Errorf("output does not contain the password-saving warning:\n%s", output)
	}

	info, statErr := os.Stat(pwdFile)
	if statErr != nil {
		t.Fatalf("stat %s: %v", pwdFile, statErr)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("password file mode = %o, want 600", info.Mode().Perm())
	}
}

func TestEnsureResticPassword_ExistingFileNotOverwrittenOrReprinted(t *testing.T) {
	pwdFile := filepath.Join(t.TempDir(), "restic-password")
	writeFile(t, pwdFile, "existing-password\n")
	cfg := &Config{Restic: ResticConfig{PasswordFile: pwdFile}}

	var err error
	output := captureStdout(t, func() {
		err = ensureResticPassword(cfg)
	})
	if err != nil {
		t.Fatalf("ensureResticPassword: %v", err)
	}

	data, readErr := os.ReadFile(pwdFile)
	if readErr != nil {
		t.Fatalf("reading %s: %v", pwdFile, readErr)
	}
	if strings.TrimSpace(string(data)) != "existing-password" {
		t.Errorf("existing password was overwritten: %q", string(data))
	}
	if strings.Contains(output, "existing-password") {
		t.Errorf("existing password should not be printed to the console:\n%s", output)
	}
}

func TestEnsureResticPassword_UsesOverrideInsteadOfGenerating(t *testing.T) {
	pwdFile := filepath.Join(t.TempDir(), "restic-password")
	cfg := &Config{
		Restic:                 ResticConfig{PasswordFile: pwdFile},
		ResticPasswordOverride: "reused-password-from-another-server",
	}

	if err := ensureResticPassword(cfg); err != nil {
		t.Fatalf("ensureResticPassword: %v", err)
	}

	data, readErr := os.ReadFile(pwdFile)
	if readErr != nil {
		t.Fatalf("reading %s: %v", pwdFile, readErr)
	}
	if strings.TrimSpace(string(data)) != "reused-password-from-another-server" {
		t.Errorf("wrong password was written: %q", string(data))
	}

	info, statErr := os.Stat(pwdFile)
	if statErr != nil {
		t.Fatalf("stat %s: %v", pwdFile, statErr)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("password file mode = %o, want 600", info.Mode().Perm())
	}
}

func TestEnsureResticPassword_OverrideIgnoredIfFileAlreadyExists(t *testing.T) {
	pwdFile := filepath.Join(t.TempDir(), "restic-password")
	writeFile(t, pwdFile, "existing-password\n")
	cfg := &Config{
		Restic:                 ResticConfig{PasswordFile: pwdFile},
		ResticPasswordOverride: "should-not-be-used",
	}

	if err := ensureResticPassword(cfg); err != nil {
		t.Fatalf("ensureResticPassword: %v", err)
	}

	data, readErr := os.ReadFile(pwdFile)
	if readErr != nil {
		t.Fatalf("reading %s: %v", pwdFile, readErr)
	}
	if strings.TrimSpace(string(data)) != "existing-password" {
		t.Errorf("existing password was overwritten by the override: %q", string(data))
	}
}
