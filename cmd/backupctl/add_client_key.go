package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// authorizedKeysOptions restricts what the client's key can do: only
// forwarding is disabled, a full shell isn't needed — the client only
// runs rsync over ssh.
const authorizedKeysOptions = "no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty"

// buildAuthorizedKeysLine is a pure function for tests.
func buildAuthorizedKeysLine(pubKey string) (string, error) {
	pubKey = strings.TrimSpace(pubKey)
	if pubKey == "" {
		return "", fmt.Errorf("empty public key")
	}
	if !strings.HasPrefix(pubKey, "ssh-") && !strings.HasPrefix(pubKey, "ecdsa-") {
		return "", fmt.Errorf("this doesn't look like a public SSH key: %q", pubKey)
	}
	return fmt.Sprintf("%s %s", authorizedKeysOptions, pubKey), nil
}

// AddClientKey adds the client's public key to backup-user's
// authorized_keys on the server (idempotent — doesn't duplicate it).
func AddClientKey(cfg *Config, pubKeyArg string) error {
	if cfg.BackupUser == "" {
		return fmt.Errorf("backup_user is not set in the config")
	}
	if !userExists(cfg.BackupUser) {
		return fmt.Errorf("user %s doesn't exist — run --init-server first", cfg.BackupUser)
	}

	pubKey, err := resolvePubKeyArg(pubKeyArg)
	if err != nil {
		return err
	}
	line, err := buildAuthorizedKeysLine(pubKey)
	if err != nil {
		return err
	}

	home, err := backupUserHome(cfg.BackupUser)
	if err != nil {
		return err
	}
	sshDir := filepath.Join(home, ".ssh")
	authKeysPath := filepath.Join(sshDir, "authorized_keys")

	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return fmt.Errorf("creating %s: %w", sshDir, err)
	}

	existing := ""
	if data, err := os.ReadFile(authKeysPath); err == nil {
		existing = string(data)
	}
	if strings.Contains(existing, pubKey) {
		fmt.Println("key is already in authorized_keys")
		return nil
	}

	f, err := os.OpenFile(authKeysPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("opening %s: %w", authKeysPath, err)
	}
	defer f.Close()
	if _, err := f.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("writing to %s: %w", authKeysPath, err)
	}

	if err := runShell("chown", "-R", cfg.BackupUser+":"+cfg.BackupUser, sshDir); err != nil {
		return err
	}
	fmt.Printf("key added to %s\n", authKeysPath)
	return nil
}

// resolvePubKeyArg accepts either a path to a public key file, or the key
// itself as a string (starting with ssh-/ecdsa-).
func resolvePubKeyArg(arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", fmt.Errorf("-pubkey is not set")
	}
	if strings.HasPrefix(arg, "ssh-") || strings.HasPrefix(arg, "ecdsa-") {
		return arg, nil
	}
	data, err := os.ReadFile(arg)
	if err != nil {
		return "", fmt.Errorf("reading key file %s: %w", arg, err)
	}
	return string(data), nil
}

func backupUserHome(username string) (string, error) {
	out, err := exec.Command("getent", "passwd", username).Output()
	if err != nil {
		return "", fmt.Errorf("getent passwd %s: %w", username, err)
	}
	fields := strings.Split(strings.TrimSpace(string(out)), ":")
	if len(fields) < 6 {
		return "", fmt.Errorf("unexpected getent passwd output for %s", username)
	}
	return fields[5], nil
}
