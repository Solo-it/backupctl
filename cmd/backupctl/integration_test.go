//go:build integration

// Integration tests run real external commands (ssh-keygen, restic,
// rsync), but only safe ones that don't need root and don't mutate the
// system. useradd/systemctl/chown aren't covered by real tests — they can
// only be verified on a test VM.
//
// Run: go test -tags=integration ./cmd/backupctl/...
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegration_EnsureClientSSHKey_GeneratesRealKeyPair(t *testing.T) {
	if !commandExists("ssh-keygen") {
		t.Skip("ssh-keygen is not installed")
	}

	keyPath := filepath.Join(t.TempDir(), "id_ed25519_backupctl")

	pubKey, err := ensureClientSSHKey(keyPath)
	if err != nil {
		t.Fatalf("ensureClientSSHKey: %v", err)
	}
	if !strings.HasPrefix(pubKey, "ssh-ed25519 ") {
		t.Errorf("pubKey = %q, expected the ssh-ed25519 prefix", pubKey)
	}
	if !fileExists(keyPath) {
		t.Error("private key was not created")
	}

	// Verify with a real ssh-keygen -l that the file is a valid key.
	out, err := exec.Command("ssh-keygen", "-l", "-f", keyPath+".pub").CombinedOutput()
	if err != nil {
		t.Fatalf("ssh-keygen -l: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "256") { // ed25519 = 256 bits
		t.Errorf("unexpected ssh-keygen -l output: %s", out)
	}

	// Calling it again shouldn't regenerate the key.
	pubKey2, err := ensureClientSSHKey(keyPath)
	if err != nil {
		t.Fatalf("ensureClientSSHKey (again): %v", err)
	}
	if pubKey2 != pubKey {
		t.Error("calling ensureClientSSHKey again regenerated the key")
	}
}

func TestIntegration_AddClientKey_WithRealGeneratedKey(t *testing.T) {
	if !commandExists("ssh-keygen") {
		t.Skip("ssh-keygen is not installed")
	}

	keyPath := filepath.Join(t.TempDir(), "id_ed25519_backupctl")
	pubKey, err := ensureClientSSHKey(keyPath)
	if err != nil {
		t.Fatalf("ensureClientSSHKey: %v", err)
	}

	line, err := buildAuthorizedKeysLine(pubKey)
	if err != nil {
		t.Fatalf("buildAuthorizedKeysLine: %v", err)
	}
	if !strings.Contains(line, pubKey) {
		t.Errorf("authorized_keys line does not contain the generated key: %s", line)
	}
}

func TestIntegration_Restic_VersionAndInitRepo(t *testing.T) {
	if !commandExists("restic") {
		t.Skip("restic is not installed")
	}

	repo := filepath.Join(t.TempDir(), "restic-repo")
	pwdFile := filepath.Join(t.TempDir(), "restic-password")
	if err := os.WriteFile(pwdFile, []byte("test-password\n"), 0o600); err != nil {
		t.Fatalf("writing password: %v", err)
	}

	cfg := &Config{
		Restic: ResticConfig{Repo: repo, PasswordFile: pwdFile},
	}

	if err := ensureResticRepoInit(cfg); err != nil {
		t.Fatalf("ensureResticRepoInit: %v", err)
	}

	// Verify with a real restic snapshots that the repository was actually created.
	cmd := exec.Command("restic", "-r", repo, "snapshots")
	cmd.Env = append(os.Environ(), "RESTIC_PASSWORD_FILE="+pwdFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("restic snapshots after init: %v\n%s", err, out)
	}

	// Calling it again shouldn't fail (idempotency).
	if err := ensureResticRepoInit(cfg); err != nil {
		t.Fatalf("ensureResticRepoInit (again): %v", err)
	}
}

func TestIntegration_Rsync_PullScriptRunsForReal(t *testing.T) {
	if !commandExists("rsync") {
		t.Skip("rsync is not installed")
	}

	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "dst")
	if err := os.WriteFile(filepath.Join(srcDir, "snapshot.dat"), []byte("data"), 0o644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}

	// Local rsync without ssh, so it doesn't need network access — pullScript
	// itself uses ssh; here we just check that the rsync binary with the
	// same flags actually works and copies data.
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("mkdir dst: %v", err)
	}
	out, err := exec.Command("rsync", "-az", "--delete", srcDir+"/", dstDir+"/").CombinedOutput()
	if err != nil {
		t.Fatalf("rsync: %v\n%s", err, out)
	}
	if !fileExists(filepath.Join(dstDir, "snapshot.dat")) {
		t.Error("rsync did not copy the file")
	}
}
