# backupctl

A CLI tool for setting up database backups via [restic](https://restic.net/).
The server (any machine with MySQL/PostgreSQL) dumps the databases and
stores them locally in a restic repository. A client (your Mac or another
server) periodically pulls that repository over `rsync` via SSH. The only
way in is an SSH key — the dedicated `backup-user` has no password at all.

Русская версия: `backupctl --info -lang=ru` or [README.ru.md](https://github.com/Solo-it/backupctl/blob/main/cmd/backupctl/README.ru.md).

## Install

```bash
# Debian/Ubuntu (apt repo, GPG-signed)
curl -fsSL https://solo-it.github.io/backupctl/apt/signing-key.asc | sudo gpg --dearmor -o /usr/share/keyrings/backupctl.gpg
echo "deb [signed-by=/usr/share/keyrings/backupctl.gpg] https://solo-it.github.io/backupctl/apt stable main" | sudo tee /etc/apt/sources.list.d/backupctl.list
sudo apt update && sudo apt install backupctl

# Homebrew (macOS/Linux)
brew install Solo-it/tap/backupctl

# Go
go install github.com/Solo-it/backupctl/cmd/backupctl@latest

# or grab a binary/.deb/.rpm from the Releases page
```

## Commands

| Command | Runs on | Does |
|---|---|---|
| `--gen-config` | server | generate a server `backup.yaml` |
| `--gen-config-client` | client | generate a client `backup.yaml` |
| `--init-server` | server (root) | restic, dump tools, passwordless `backup-user`, repository, scheduled dump via systemd timer |
| `--add-client-key` | server (root) | add a client's public SSH key to `backup-user`'s `authorized_keys` |
| `--init-client-pm2` | client (Mac) | SSH key + pull script + `pm2` with `cron_restart` |
| `--init-client-launchd` | client (Mac) | SSH key + pull script + `launchd` plist |
| `--init-client-systemd` | client (Linux) | SSH key + pull script + systemd timer |
| `--init-client-task-scheduler` | client (Windows) | SSH key + pull script (PowerShell) + Task Scheduler (schtasks) |
| `--version` / `-v` | — | version |
| `--help` / `-h` | — | flag reference |
| `--info` | — | this README, embedded in the binary |

Add `-lang=ru` to `--help`/`--info` for the Russian text. Every other
command, message and error string is English-only for now — see
[CONTRIBUTING](#contributing) if you'd like to help translate those too.

All commands except `--gen-config`/`--gen-config-client` read the config
from `-config backup.yaml` (override the path with the flag). Every flag
on `--gen-config`/`--gen-config-client` is optional — sane defaults, can
run with none at all.

## Step by step

### 1. On the server — generate a config and set up restic

```bash
backupctl --gen-config -out=backup.yaml \
  -restic-repo=/home/backup-user/backups/restic-repo \
  -backup-user=backup-user \
  -server-schedule=02:00
```

Open `backup.yaml` and fill in real database names under `databases:` —
the generated file only has an `example_db` placeholder with a comment
listing supported `type:` values (currently `mysql`, `postgres`). That's
the only thing you need to edit by hand.

`restic.password_file` in the config is where the restic repository
password will live. The file itself doesn't exist yet at this point.

```bash
sudo backupctl --init-server
```

(`-config` defaults to `backup.yaml` in the current directory — only pass
it if your file is named differently.)

Idempotent: running it again won't break an already-configured server.
`schedule` in `backup.yaml` is UTC, regardless of the server's own
timezone (the systemd `OnCalendar` line is generated with an explicit
`UTC` suffix) — check the actual next run time with
`systemctl list-timers restic-backup.timer`.

If `restic` isn't in PATH, `--init-server` tries to install it itself
through the first package manager it finds (`apt-get`, `dnf`, `yum`,
`apk`, `pacman`, `brew`); if none are available it stops with a clear
error and a link to the manual install instructions. It also checks
upfront that the dump tool for every `type:` in `databases:` is in PATH
(`mysqldump` for `mysql`, `pg_dump` for `postgres`).

**About the restic password — this matters if you have more than one server:**

- **First `--init-server` run on this server, nothing specified** — a
  random password is generated, written to `password_file` with `600`
  permissions (owned by root), and **printed to the console once**. That
  is the only moment you'll see it — save it to a password manager right
  away. If the server and its disk are lost, the client is left with only
  an encrypted copy of the repository (see step 2), and without this
  password there is no way to decrypt it.
- **Want the same password across several servers** (easier to manage,
  but compromising one password compromises every repository — a
  convenience/isolation trade-off) — pass it as a flag on the first run
  on each following server:
  ```bash
  sudo backupctl --init-server -restic-password="<password-from-the-first-server>"
  ```
  Only works while `password_file` doesn't exist yet — `--init-server`
  never overwrites an existing one.
- **File already exists** (however it got there) — `--init-server`
  leaves it alone and doesn't print it, just reports that a password is
  already in place.

After a successful `--init-server`, the command prints its own list of
next steps (test run, client setup, allowing the client's key) — you can
follow that output as well as this README.

### 2. On the client — generate a config, key, and scheduler

```bash
backupctl --gen-config-client -out=backup-client.yaml \
  -client-remote=backup-user@<server-ip> \
  -client-remote-repo=/home/backup-user/backups/restic-repo \
  -client-schedule=04:00

backupctl --init-client-launchd -config=backup-client.yaml   # macOS
# or --init-client-pm2 / --init-client-systemd / --init-client-task-scheduler
```

`--init-client-task-scheduler` (Windows) additionally requires `rsync` in
PATH — e.g. from WSL, cwRsync, or Git for Windows (Git Bash puts
`rsync.exe` in PATH).

The command generates an `ed25519` SSH key (if it doesn't have one yet)
and prints the public key along with the ready-to-run command for the
server.

### 3. On the server — allow the client's key

```bash
sudo backupctl --add-client-key -pubkey="ssh-ed25519 AAAA... backupctl-client"
```

Idempotent — adding the same key twice doesn't duplicate the
`authorized_keys` entry. The key is restricted with
`no-pty,no-port-forwarding,no-X11-forwarding,no-agent-forwarding` — it
can only run `rsync`, nothing else.

### 4. Test the pull by hand

```bash
~/.local/bin/backup-pull.sh        # launchd
/usr/local/bin/backup-pull.sh      # pm2 / systemd
```

```powershell
& "$env:USERPROFILE\backupctl\backup-pull.ps1"   # Task Scheduler (Windows)
```

## Added or removed a database on the server — how to update

Edit `databases:` in `backup.yaml` (add or remove `type`/`names`) and run
`--init-server` again:

```bash
sudo backupctl --init-server -config=backup.yaml
```

Nothing destructive happens — `backup-user`, the restic repository and
the password already exist, those steps just confirm everything is in
place. `/root/backup.sh` (the list of dump commands) and the systemd
unit/timer, though, are rewritten unconditionally every time — so the new
database is picked up by the very next scheduled run. There's no separate
"update" command; `--init-server` is exactly that.

## backup.yaml format

```yaml
restic:
  repo: /home/backup-user/backups/restic-repo
  password_file: /root/.restic-env    # created automatically by --init-server

backup_user: backup-user

databases:           # supported type: mysql, postgres
  - type: mysql
    names:
      - app_db
      - shop_db
  - type: postgres
    names:
      - analytics

schedule: "02:00"   # UTC, server dump+backup time

client:
  remote: backup-user@192.168.1.10
  remote_repo: /home/backup-user/backups/restic-repo
  local_path: ~/backups/mysql-restic-repo
  schedule: "04:00"   # when the client pulls the backup
```

## Tests

```bash
go test ./...                    # unit tests, pure logic
go test -tags=integration ./...  # + real ssh-keygen/restic/rsync
```

`useradd`/`chown`/`systemctl` aren't covered by real tests (need root and
mutate the system) — their behavior is only verified manually, on a test
VM.

## Versioning

Version comes from a git tag `vX.Y.Z`, built and released by
[GoReleaser](https://goreleaser.com/) on every push of such a tag:

```bash
git tag v1.0.0 && git push origin v1.0.0
```

## Contributing

Issues and PRs welcome — including translations (see `-lang=ru` above;
adding a new language means a new `README.<lang>.md`, a new `usageText`
variant, and translating the step/status messages sprinkled through
`cmd/backupctl/*.go`).

## Security

apt package signing, key rotation, and what to do if `apt update` starts
failing with a GPG error: [SECURITY.md](../../SECURITY.md).

## License

[MIT](../../LICENSE)
