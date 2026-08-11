# Verify release tarball checksum in scripts/install.sh before extract

**Date:** 2026-08-11
**Files:** modified `scripts/install.sh` (added `verify_checksum()`, wired into
the prebuilt-download branch), modified `README.md` (Install section note).

## Why
Security review of the fleet (issue ph-commons/nowshowing-pp-cli#7, parent #6
"security review 20260810", High H1) found `scripts/install.sh` extracted the
downloaded release tarball with `tar -xzf` without verifying it against
goreleaser's published `checksums.txt`. An attacker who could MITM the
download or compromise the release CDN could ship a tampered binary that the
installer would happily extract and `chmod +x` into `$GOBIN`/`~/.local/bin`,
running with the invoking user's shell privileges on next invocation.

Fix: download `checksums.txt` from the same release tag, verify the tarball's
SHA-256 against it, and fail closed (non-zero exit, no extraction) on any
problem — checksums.txt unreachable, no matching entry, or hash mismatch. A
checksum failure does not fall through to the existing `go install`
source-build fallback (that fallback is for network/availability problems,
not a security signal).

## Residual risk (documented, not fixed by this patch)
`checksums.txt` is fetched from the same GitHub release / CDN as the tarball
itself. An attacker capable of MITM'ing or compromising the release artifacts
enough to swap the tarball could in principle also swap the accompanying
checksums.txt to match. This patch closes the "corrupted download / partial
mirror / accidental tampering without also faking the checksum file" gap and
the "silently ships whatever tar produces" gap, but does not provide
cryptographic provenance (e.g. GPG-signed checksums, sigstore/cosign). A full
fix for the MITM-on-both-files case is out of scope for H1 and would need a
signed-checksums or sigstore-attestation follow-up.

## Retro candidate (machine bug / generator gap)
`cli-printing-press`-generated `install.sh` scripts across the fleet do not
verify release checksums by default (confirmed absent in sibling
`pse-edge-pp-cli`'s `scripts/install.sh` as well). This looks like a
generator-template gap worth raising upstream in `mvanhorn/cli-printing-press`
rather than a one-off — no local generator checkout was available in this
run to make that change directly.
