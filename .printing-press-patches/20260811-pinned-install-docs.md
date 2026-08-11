# Document trust model and pinned install alternatives in README

**Date:** 2026-08-11
**Files:** modified `README.md` (Install section only).

## Why
Security review of the fleet (issue ph-commons/nowshowing-pp-cli#11, parent
#6 "security review 20260810", Medium M1) found the primary advertised
install path is `curl|bash` with no cosign/minisign, acceptable early-stage
now that #7 (H1, checksum fail-closed on the prebuilt path — merged as
`82b89fbe`, PR #18) is in, but untrusted-host readers had no clearly marked
pinned alternative or explicit statement of what `curl|bash`'s trust root
actually is.

Added a "Trust model" note directly under the `## Install` heading stating
the curl|bash path's trust root (GitHub org + TLS + SHA-256 checksum, not
signing) and pointing untrusted-host readers to the Go install / manual
binary paths instead. Added a pinned `@v0.1.1` variant of the `go install`
commands (the existing example used floating `@latest`) with a note that
`go install` builds via the Go module checksum database and sidesteps the
release-tarball trust question entirely. Added a manual `sha256sum -c`
snippet to the pre-built-binary section so a reader can verify without
running the installer script at all.

This is a docs-only change. `scripts/install.sh` and its checksum
verification logic (`verify_checksum()`, added by the #7/H1 patch above)
are untouched — issue #11's acceptance criteria require the H1 installer
behavior not regress, and this patch does not touch that file.

No cosign/minisign claim is made anywhere in the new text — issue #11
explicitly scopes signing as an optional future follow-up, not part of this
fix.

## Retro candidate
None — this is a straight docs addition following the same
`.printing-press-patches/` convention as the two prior README-touching
patches in this fleet (`20260810-unofficial-life-safety-disclaimer.md`,
`20260811-install-checksum-verification.md`).
