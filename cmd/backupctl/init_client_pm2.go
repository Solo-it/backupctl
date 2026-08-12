package main

import (
	"fmt"
	"os"
)

// InitClientPM2 sets up the backup pull on Mac via pm2: generates a pull
// script and ecosystem.config.js with cron_restart, registers it in pm2.
func InitClientPM2(cfg *Config) error {
	if !commandExists("pm2") {
		return fmt.Errorf("pm2 not found in PATH — install it: npm install -g pm2")
	}
	if err := printClientSSHKey(); err != nil {
		return err
	}

	scriptPath := "/usr/local/bin/backup-pull.sh"
	if err := writePullScript(cfg, scriptPath); err != nil {
		return err
	}

	cron, err := toCron(cfg.Client.Schedule)
	if err != nil {
		return err
	}

	ecosystem := fmt.Sprintf(`module.exports = {
  apps: [{
    name: "backup-pull",
    script: %q,
    cron_restart: %q,
    autorestart: false,
  }],
};
`, scriptPath, cron)

	ecosystemPath := "ecosystem.backup.config.js"
	if err := os.WriteFile(ecosystemPath, []byte(ecosystem), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", ecosystemPath, err)
	}
	fmt.Printf("pm2 config written to %s\n", ecosystemPath)

	if err := runShell("pm2", "start", ecosystemPath); err != nil {
		return err
	}
	return runShell("pm2", "save")
}
