# backupctl

Back up your sites' and microservices' databases — MySQL and PostgreSQL —
in three commands. A server dumps the databases into a [restic](https://restic.net/)
repository, a client (your Mac or another server) pulls it over `rsync`
via SSH — no passwords anywhere, only an SSH key.

**Full documentation:** [cmd/backupctl/README.md](cmd/backupctl/README.md)
([Russian](cmd/backupctl/README.ru.md)) — or, once installed, run
`backupctl --info` (add `-lang=ru` for Russian). Site with a live demo:
[solo-it.github.io/backupctl](https://solo-it.github.io/backupctl/).

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/Solo-it/backupctl/main/backupctl-install.sh | sh
```

Detects apt or Homebrew and uses those; otherwise downloads the matching
binary from the latest release and verifies it against `checksums.txt`
before installing — see [backupctl-install.sh](backupctl-install.sh) for
exactly what it runs, nothing hidden.

<details>
<summary>Other install methods (apt / Homebrew / Go / manual)</summary>

```bash
# Debian/Ubuntu (apt repo, GPG-signed — see SECURITY.md)
curl -fsSL https://solo-it.github.io/backupctl/apt/signing-key.asc | sudo gpg --dearmor -o /usr/share/keyrings/backupctl.gpg
echo "deb [signed-by=/usr/share/keyrings/backupctl.gpg] https://solo-it.github.io/backupctl/apt stable main" | sudo tee /etc/apt/sources.list.d/backupctl.list
sudo apt update && sudo apt install backupctl

# Homebrew
brew install Solo-it/tap/backupctl
# or, for the short form afterwards (one-time setup):
#   brew tap Solo-it/tap && brew trust solo-it/tap
#   brew install backupctl

# Go
go install github.com/Solo-it/backupctl/cmd/backupctl@latest

# or grab a binary/.deb/.rpm/.apk directly from the Releases page
```

</details>

License: [MIT](LICENSE). Security / apt key rotation: [SECURITY.md](SECURITY.md).
