# backupctl

A CLI tool for setting up database backups via [restic](https://restic.net/):
a server dumps MySQL/PostgreSQL into a restic repository, and a client
(your Mac or another server) pulls it over `rsync` via SSH — no
passwords anywhere, only an SSH key.

**Full documentation:** [cmd/backupctl/README.md](cmd/backupctl/README.md)
([Russian](cmd/backupctl/README.ru.md)) — or, once installed, run
`backupctl --info` (add `-lang=ru` for Russian).

```bash
# Debian/Ubuntu (apt repo, GPG-signed — see SECURITY.md)
curl -fsSL https://solo-it.github.io/backupctl/apt/signing-key.asc | sudo gpg --dearmor -o /usr/share/keyrings/backupctl.gpg
echo "deb [signed-by=/usr/share/keyrings/backupctl.gpg] https://solo-it.github.io/backupctl/apt stable main" | sudo tee /etc/apt/sources.list.d/backupctl.list
sudo apt update && sudo apt install backupctl

# Homebrew
brew install Solo-it/tap/backupctl

# Go
go install github.com/Solo-it/backupctl/cmd/backupctl@latest
```

License: [MIT](LICENSE). Security / apt key rotation: [SECURITY.md](SECURITY.md).
