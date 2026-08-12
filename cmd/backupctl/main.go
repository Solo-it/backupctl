// Command backupctl sets up restic-based database backups on servers and
// on the clients that pull them.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const usageTextEN = `Usage: backupctl <command> [flags]

backupctl sets up restic-based database backups (MySQL, PostgreSQL) on
servers and on the clients that pull them.

Commands:
  --version, -v             backupctl version
  --help, -h                this help (commands and flags)
  --info                    step-by-step guide, where to start (README in the console)
  --gen-config               generate a server backup.yaml
  --gen-config-client        generate a client backup.yaml
  --init-server              set up the server: restic, backup-user (no
                              password, SSH key only), repository, scheduled
                              dump via systemd timer
  --add-client-key           add a client's public SSH key to
                              backup-user's authorized_keys on the server
  --init-client-pm2          set up the client (Mac) via pm2:
                              generates an SSH key, a pull script (rsync over
                              SSH), ecosystem.config.js with cron_restart
  --init-client-launchd      set up the client (Mac) via launchd:
                              same thing, but the scheduler is a launchd plist
  --init-client-systemd      set up the client (Linux) via systemd:
                              same thing, but the scheduler is a systemd timer
  --init-client-task-scheduler  set up the client (Windows) via Task Scheduler:
                              same thing, but the scheduler is schtasks (needs rsync in PATH)

--help / --info flags:
  -lang string       en (default) or ru

Common flags (--init-client-*, --add-client-key):
  -config string    path to backup.yaml (default ./backup.yaml)

--init-server flags:
  -config string       path to backup.yaml (default ./backup.yaml)
  -restic-password string  use this restic password instead of generating a
                        random one (e.g. to reuse a password from another
                        server — one shared password for every repository).
                        Only takes effect if password_file doesn't exist yet
                        — an existing file is never touched.

--gen-config flags (all optional, sane defaults):
  -out string                 path to write to (default ./backup.yaml)
  -force                      overwrite an existing file
  -restic-repo string         restic repository path on the server
  -restic-password-file string  path to the restic password file
  -backup-user string         system user for backups
  -server-schedule string     server backup time HH:MM UTC

--gen-config-client flags (all optional):
  -out string                 path to write to (default ./backup-client.yaml)
  -force                      overwrite an existing file
  -client-remote string       user@host of the server
  -client-remote-repo string  repo path on the server
  -client-local-path string   local path on the client
  -client-schedule string     client pull time HH:MM

--add-client-key flags:
  -pubkey string     path to a public key file, or the key itself as a string

Examples:
  backupctl --gen-config
  backupctl --init-server
  backupctl --gen-config-client -out=backup-client.yaml -client-remote=backup-user@1.2.3.4
  backupctl --init-client-systemd -config=backup-client.yaml
  backupctl --add-client-key -pubkey=~/.ssh/id_ed25519_backupctl.pub
`

const usageTextRU = `Использование: backupctl <команда> [флаги]

backupctl настраивает restic-бэкапы БД (MySQL, PostgreSQL) на серверах
и клиентах, которые их забирают.

Команды:
  --version, -v             версия backupctl
  --help, -h                эта справка (команды и флаги)
  --info                    пошаговая инструкция, с чего начать (README в консоли)
  --gen-config               сгенерировать серверный backup.yaml
  --gen-config-client        сгенерировать клиентский backup.yaml
  --init-server              настройка сервера: restic, backup-user (без пароля,
                              вход только по SSH-ключу), репозиторий, дамп БД
                              по расписанию через systemd timer
  --add-client-key           добавить публичный SSH-ключ клиента в
                              authorized_keys backup-user на сервере
  --init-client-pm2          настройка клиента (Mac) через pm2:
                              генерирует SSH-ключ, pull-скрипт (rsync по SSH),
                              ecosystem.config.js с cron_restart
  --init-client-launchd      настройка клиента (Mac) через launchd:
                              то же самое, но планировщик — launchd plist
  --init-client-systemd      настройка клиента (Linux) через systemd:
                              то же самое, но планировщик — systemd timer
  --init-client-task-scheduler  настройка клиента (Windows) через Планировщик заданий:
                              то же самое, но планировщик — schtasks (нужен rsync в PATH)

Остальной вывод программы (шаги --init-server, сообщения об ошибках) пока
только на английском — переведены только --help/--info.

Флаги --help / --info:
  -lang string       en (по умолчанию) или ru
`

func usage(lang string) {
	fmt.Fprint(os.Stderr, usageText(lang))
}

func usageText(lang string) string {
	if lang == "ru" {
		return usageTextRU
	}
	return usageTextEN
}

// extractLang pulls -lang/--lang out of args wherever it appears (it's a
// global flag, not tied to any one subcommand's flag.FlagSet) and returns
// the remaining args unchanged otherwise. Defaults to "en".
func extractLang(args []string) (lang string, rest []string) {
	lang = "en"
	for i, a := range args {
		switch {
		case a == "-lang" || a == "--lang":
			if i+1 < len(args) {
				lang = args[i+1]
				rest = append(append([]string{}, args[:i]...), args[i+2:]...)
				return lang, rest
			}
		case strings.HasPrefix(a, "-lang="):
			lang = strings.TrimPrefix(a, "-lang=")
			rest = append(append([]string{}, args[:i]...), args[i+1:]...)
			return lang, rest
		case strings.HasPrefix(a, "--lang="):
			lang = strings.TrimPrefix(a, "--lang=")
			rest = append(append([]string{}, args[:i]...), args[i+1:]...)
			return lang, rest
		}
	}
	return lang, args
}

func main() {
	if len(os.Args) < 2 {
		usage("en")
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "--help", "-h":
		lang, _ := extractLang(os.Args[2:])
		fmt.Print(usageText(lang))
		return
	case "--info":
		lang, _ := extractLang(os.Args[2:])
		printInfo(lang)
		return
	case "--version", "-v":
		printVersion()
		return
	case "--gen-config":
		runGenConfig(os.Args[2:], "server", "backup.yaml")
		return
	case "--gen-config-client":
		runGenConfig(os.Args[2:], "client", "backup-client.yaml")
		return
	case "--add-client-key":
		runAddClientKey(os.Args[2:])
		return
	case "--init-server":
		runInitServer(os.Args[2:])
		return
	}

	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	configPath := fs.String("config", "backup.yaml", "path to backup.yaml")
	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}

	var run func(*Config) error
	switch cmd {
	case "--init-client-pm2":
		run = InitClientPM2
	case "--init-client-launchd":
		run = InitClientLaunchd
	case "--init-client-systemd":
		run = InitClientSystemd
	case "--init-client-task-scheduler":
		run = InitClientTaskScheduler
	default:
		usage("en")
		os.Exit(1)
	}

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// runGenConfig serves both --gen-config (mode="server") and
// --gen-config-client (mode="client") — the same generator, a different
// set of visible flags and a different default file name.
func runGenConfig(args []string, mode, defaultOut string) {
	fs := flag.NewFlagSet("--gen-config", flag.ExitOnError)
	out := fs.String("out", defaultOut, "path to write to")
	force := fs.Bool("force", false, "overwrite an existing file")

	p := defaultGenConfigParams()
	p.Mode = mode

	if mode == "server" {
		fs.StringVar(&p.ResticRepo, "restic-repo", p.ResticRepo, "restic repository path on the server")
		fs.StringVar(&p.ResticPasswordFile, "restic-password-file", p.ResticPasswordFile, "path to the restic password file")
		fs.StringVar(&p.BackupUser, "backup-user", p.BackupUser, "system user for backups")
		fs.StringVar(&p.ServerSchedule, "server-schedule", p.ServerSchedule, "server backup time HH:MM UTC")
	} else {
		fs.StringVar(&p.ClientRemote, "client-remote", p.ClientRemote, "user@host of the server")
		fs.StringVar(&p.ClientRemoteRepo, "client-remote-repo", p.ClientRemoteRepo, "repo path on the server")
		fs.StringVar(&p.ClientLocalPath, "client-local-path", p.ClientLocalPath, "local path on the client")
		fs.StringVar(&p.ClientSchedule, "client-schedule", p.ClientSchedule, "client pull time HH:MM")
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if err := GenConfig(p, *out, *force); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func runAddClientKey(args []string) {
	fs := flag.NewFlagSet("--add-client-key", flag.ExitOnError)
	configPath := fs.String("config", "backup.yaml", "path to backup.yaml")
	pubkey := fs.String("pubkey", "", "path to a public key file, or the key itself as a string")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	if err := AddClientKey(cfg, *pubkey); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func runInitServer(args []string) {
	fs := flag.NewFlagSet("--init-server", flag.ExitOnError)
	configPath := fs.String("config", "backup.yaml", "path to backup.yaml")
	resticPassword := fs.String("restic-password", "", "use this restic password instead of generating a random one")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	cfg.ResticPasswordOverride = *resticPassword

	if err := InitServer(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
