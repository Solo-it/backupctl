package main

import (
	"fmt"
	"os"
	"strings"
)

// GenConfigParams holds the parameters used to generate backup.yaml.
type GenConfigParams struct {
	Mode string // "server", "client", or "full"

	ResticRepo         string
	ResticPasswordFile string
	BackupUser         string
	Databases          []DatabaseConfig
	ServerSchedule     string

	ClientRemote     string
	ClientRemoteRepo string
	ClientLocalPath  string
	ClientSchedule   string
}

func defaultGenConfigParams() GenConfigParams {
	return GenConfigParams{
		Mode:               "server",
		ResticRepo:         "/home/backup-user/backups/restic-repo",
		ResticPasswordFile: "/root/.restic-env",
		BackupUser:         "backup-user",
		ServerSchedule:     "02:00",
		ClientRemoteRepo:   "/home/backup-user/backups/restic-repo",
		ClientLocalPath:    "~/backups/mysql-restic-repo",
		ClientSchedule:     "04:00",
	}
}

// buildConfigYAML is a pure function that renders backup.yaml as text.
// In "server" mode the client section is omitted; in "client" mode the
// server sections (restic/backup_user/databases) are omitted.
func buildConfigYAML(p GenConfigParams) (string, error) {
	switch p.Mode {
	case "server", "client", "full":
	default:
		return "", fmt.Errorf("unknown mode %q, expected server|client|full", p.Mode)
	}

	var sb strings.Builder

	if p.Mode == "server" || p.Mode == "full" {
		sb.WriteString(fmt.Sprintf("restic:\n  repo: %s\n", p.ResticRepo))
		sb.WriteString(fmt.Sprintf("  password_file: %s # created automatically by --init-server, if the file doesn't exist yet\n\n", p.ResticPasswordFile))
		sb.WriteString(fmt.Sprintf("backup_user: %s\n\n", p.BackupUser))

		sb.WriteString(fmt.Sprintf("databases: # supported type: %s\n", strings.Join(supportedDBTypes, ", ")))
		if len(p.Databases) == 0 {
			sb.WriteString("  - type: mysql\n    names:\n      - example_db\n")
		}
		for _, db := range p.Databases {
			sb.WriteString(fmt.Sprintf("  - type: %s\n    names:\n", db.Type))
			for _, name := range db.Names {
				sb.WriteString(fmt.Sprintf("      - %s\n", name))
			}
		}
		sb.WriteString("\n")

		sb.WriteString(fmt.Sprintf("schedule: %q # UTC, server backup time\n\n", p.ServerSchedule))

		sb.WriteString("# files: backs up a plain directory tree (site files, a user's home\n")
		sb.WriteString("# directory) in the same restic repository, alongside the database dumps.\n")
		sb.WriteString("# preset picks a built-in exclude list for known junk (caches, tmp,\n")
		sb.WriteString(fmt.Sprintf("# archives) — supported: %s.\n", strings.Join(supportedFilePresets, ", ")))
		sb.WriteString("# Uncomment and edit to use it:\n")
		sb.WriteString("#\n")
		sb.WriteString("# files:\n")
		sb.WriteString("#   - path: /var/www/mysite\n")
		sb.WriteString("#     preset: wordpress\n")
		sb.WriteString("#     exclude: []      # extra patterns on top of the preset, optional\n")
	}

	if p.Mode == "full" {
		sb.WriteString("\n")
	}

	if p.Mode == "client" || p.Mode == "full" {
		sb.WriteString("client:\n")
		sb.WriteString(fmt.Sprintf("  remote: %s\n", p.ClientRemote))
		sb.WriteString(fmt.Sprintf("  remote_repo: %s\n", p.ClientRemoteRepo))
		sb.WriteString(fmt.Sprintf("  local_path: %s\n", p.ClientLocalPath))
		sb.WriteString(fmt.Sprintf("  schedule: %q # when the client pulls the backup\n", p.ClientSchedule))
	}

	return sb.String(), nil
}

// GenConfig generates backup.yaml and writes it to outPath (won't
// overwrite an existing file unless force is set).
func GenConfig(p GenConfigParams, outPath string, force bool) error {
	if fileExists(outPath) && !force {
		return fmt.Errorf("%s already exists, use -force to overwrite", outPath)
	}
	yamlText, err := buildConfigYAML(p)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, []byte(yamlText), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}
	fmt.Printf("config written to %s (mode=%s)\n", outPath, p.Mode)
	printGenConfigNextSteps(p.Mode, outPath)
	return nil
}

func printGenConfigNextSteps(mode, outPath string) {
	switch mode {
	case "server", "full":
		fmt.Printf(`
Next:
  1. Open %s and fill in the real database names under databases: (right
     now there's just the example_db placeholder).
  2. If you want the SAME restic password as on another server (one
     shared password across every repository — easier to manage), run
     --init-server with:
       sudo backupctl --init-server -config=%s -restic-password="<password-from-that-server>"
     If the password doesn't matter, just:
       sudo backupctl --init-server -config=%s
     Then a random password will be generated and shown in the console once.
`, outPath, outPath, outPath)
	case "client":
		fmt.Printf(`
Next:
  backupctl --init-client-systemd -config=%s   # or pm2/launchd/task-scheduler
`, outPath)
	}
}
