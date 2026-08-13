# Operational checklist (not part of the public docs)

Things only the maintainer needs to remember — not linked from README on
purpose, this is a private runbook, not user-facing documentation.

## Secrets — where they live

| Secret | Where | Purpose | Expires |
|---|---|---|---|
| `HOMEBREW_TAP_GITHUB_TOKEN` | GitHub Actions secret, repo `Solo-it/backupctl` | push the Homebrew formula to `Solo-it/homebrew-tap` on release | fine-grained PAT, **3 months** — set a reminder to renew |
| `GPG_PRIVATE_KEY` | GitHub Actions secret, repo `Solo-it/backupctl` | sign the apt repo's `Release` file | does not expire, but **only backup is in your password manager** — GitHub never returns secret values once saved |

## GPG signing key backup

- Fingerprint: see `apt-signing-key.asc` in the repo root (public half) —
  the matching private key + revocation certificate were sent to you as
  files on 2026-08-13 and should be in your password manager by now. If
  they aren't, there is no other copy anywhere.
- If the key is ever compromised: import the revocation certificate,
  publish it, generate a new key pair, replace `GPG_PRIVATE_KEY` and
  `apt-signing-key.asc`, update the README install instructions with a
  note that existing users need to re-import the new key.

## When the Homebrew token expires (~3 months from 2026-08-13)

1. GitHub → Settings → Developer settings → Fine-grained tokens → generate
   a new one, same scope as before (`Solo-it/homebrew-tap`, Contents:
   Read and write).
2. `gh secret set HOMEBREW_TAP_GITHUB_TOKEN --repo Solo-it/backupctl --body "<new token>"`
3. Until you do this, releases still work — GoReleaser just silently
   skips the Homebrew formula push and logs it.

## First release checklist

1. `git tag v1.0.0 && git push origin v1.0.0`
2. Watch the `release` workflow run (Actions tab) — two jobs:
   `goreleaser` (binaries + .deb/.rpm/.apk + GitHub Release + Homebrew
   formula) and `apt-repo` (builds and GPG-signs the apt repo, publishes
   to `gh-pages`).
3. GitHub → Settings → Pages: source should be the `gh-pages` branch,
   `/ (root)` — the workflow creates this branch on first run, but the
   Pages source setting itself may need to be pointed at it manually the
   very first time.
4. Once Pages is live, sanity-check:
   ```bash
   curl -fsSL https://solo-it.github.io/backupctl/apt/signing-key.asc | gpg --dearmor | gpg --list-packets | head
   curl -fsSL https://solo-it.github.io/backupctl/apt/dists/stable/InRelease | head
   ```
5. Try the real install flow from the README on a clean Debian/Ubuntu VM
   or container before announcing it anywhere.
