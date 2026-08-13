# Security

## apt repository signing

Releases published to the apt repository (`solo-it.github.io/backupctl/apt`)
are signed with a dedicated GPG key — see [Install](cmd/backupctl/README.md#install)
for the import command. That key only signs this repository's package
index; it isn't used for anything else and isn't tied to any personal key.

The current public key is [`apt-signing-key.asc`](apt-signing-key.asc) in
this repo's root.

## If `apt update` starts failing with a GPG error

This means the signing key was rotated (see below for why that can
happen). Re-run the import step from the README with the current
`apt-signing-key.asc` — that always fixes it. Announced rotations are
also noted in the GitHub Releases changelog.

## Key rotation

The signing key may be rotated if:

- it's ever suspected to be compromised,
- or simply as routine hygiene.

Rotation is: generate a new key pair, replace the `GPG_PRIVATE_KEY`
GitHub Actions secret, replace `apt-signing-key.asc` in this repo. The
next release is then signed with the new key. Old signatures made with
the previous key remain valid for packages already downloaded, but new
`apt update` runs need the new public key imported — there's no way
around that with unsigned distribution being the alternative, so this is
the accepted trade-off for verifiable packages.

If you maintain this project and need to rotate the key: generate the
new key pair **on your own machine**, upload only the private half to
the `GPG_PRIVATE_KEY` secret yourself (`gh secret set` locally — don't
paste a private key into a chat or issue), and hand over just the public
half (safe to share anywhere) for the `apt-signing-key.asc` commit. Keep
a revocation certificate for the key in your password manager separately
from the key itself — it lets you declare the key invalid even if you
later lose the private key outright, which the private key alone
wouldn't let you do.
