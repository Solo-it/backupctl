package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ensureClientSSHKey creates an ed25519 SSH key at keyPath (if it doesn't
// exist yet) and returns the public key contents. The only way into the
// server is that key — backup-user has no password at all (see
// ensureBackupUser).
func ensureClientSSHKey(keyPath string) (pubKey string, err error) {
	pubKeyPath := keyPath + ".pub"

	if !fileExists(keyPath) {
		if !commandExists("ssh-keygen") {
			return "", fmt.Errorf("ssh-keygen not found in PATH")
		}
		if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
			return "", fmt.Errorf("creating %s: %w", filepath.Dir(keyPath), err)
		}
		if err := runShell("ssh-keygen", "-t", "ed25519", "-N", "", "-C", "backupctl-client", "-f", keyPath); err != nil {
			return "", fmt.Errorf("generating SSH key: %w", err)
		}
	}

	data, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", pubKeyPath, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func defaultClientSSHKeyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".ssh", "id_ed25519_backupctl"), nil
}

// printClientSSHKey generates the client's key (if needed) and prints
// instructions for adding it to the server via --add-client-key.
func printClientSSHKey() error {
	keyPath, err := defaultClientSSHKeyPath()
	if err != nil {
		return err
	}
	pubKey, err := ensureClientSSHKey(keyPath)
	if err != nil {
		return err
	}
	fmt.Printf(`
Client SSH key: %s
Public key:
%s

Add it on the server (as root):
  backupctl --add-client-key -pubkey %q -config backup.yaml

`, keyPath, pubKey, pubKey)
	return nil
}
