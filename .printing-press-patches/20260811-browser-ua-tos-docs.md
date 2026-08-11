# Document browser User-Agent requirement and ToS residual in README

**Date:** 2026-08-11
**Files:** modified `README.md` (Troubleshooting section only).

## Why
Security review of the fleet (issue ph-commons/nowshowing-pp-cli#16, parent
#6 "security review 20260810", Low L2) noted that `internal/httpx` and
`internal/client` both send a Chrome-like `User-Agent` because ClickTheCity
returns `{"error"}` without one. Not a code vulnerability, but a ToS/ethics
residual that wasn't documented anywhere beyond a single troubleshooting
bullet naming the symptom, not the posture.

Added a new `### Browser User-Agent & ToS posture` subsection under
`## Troubleshooting`, immediately after the existing `### API-specific`
bullets (one of which now cross-references it). The new text:

- Names both client stacks that send the browser-like UA
  (`internal/httpx/httpx.go`'s `DefaultUserAgent` and
  `internal/client/client.go`'s inline UA string) and why (ClickTheCity has
  no documented public API and errors without a browser UA).
- States the ToS/ethics residual honestly: no agreement exists with
  ClickTheCity, popcorn.app, or IMDb, and no claim is made that sending a
  browser UA is expressly permitted by their terms. Cross-references the
  existing "Unofficial" disclaimer at the top of the README rather than
  restating it.
- Documents the adaptive rate limiter (`internal/cliutil.AdaptiveLimiter`,
  wired into both client stacks) as the mitigation this project relies on,
  and instructs future maintainers not to remove it or widen parallel
  fan-out without equivalent backoff.

This is a docs-only change. No code file was touched: the UA string values
in `internal/httpx/httpx.go` and `internal/client/client.go`, and the
`AdaptiveLimiter` implementation in `internal/cliutil/ratelimit.go`, are all
unchanged — issue #16 explicitly requires no code change.

No ToS-compliance claim is made anywhere in the new text — the residual is
described as open and unresolved, not solved, per the issue's framing.

## Retro candidate
None — this is a straight docs addition following the same
`.printing-press-patches/` convention as the prior README-touching patches
in this fleet (`20260811-pinned-install-docs.md`,
`20260810-unofficial-life-safety-disclaimer.md`).
