# Allowlist NOWSHOWING_BASE_URL / config base_url host

**Date:** 2026-08-11
**Files:** `internal/config/config.go` (added allowlist + `Load()` signature
change), `internal/cli/root.go` (new `--allow-custom-base-url` persistent
flag, `newClient()` call-site update), `internal/cli/doctor.go` (call-site
update), `internal/mcp/tools.go` (call-site update), new
`internal/config/config_test.go`.

## Why

Issue #13 (M3, parent #6 security review): `internal/config/config.go`
accepted `NOWSHOWING_BASE_URL` (and the TOML config file's `base_url`)
without any host check. A hostile environment on a shared agent host could
retarget the generated HTTP client's browser-UA traffic at an arbitrary or
internal host.

`Load()` now validates the fully-resolved `BaseURL` (default -> config file
-> env override) against a small exact-match host allowlist
(`clickthecity.com`, `www.clickthecity.com`) and fails closed — returns a
non-nil error and a nil `*Config` — for any other host, unless one of two
explicit break-glass mechanisms is used: the new `--allow-custom-base-url`
CLI flag, or the `NOWSHOWING_ALLOW_CUSTOM_BASE_URL=1` env var (the latter is
the only option for callers with no flag surface, e.g. the MCP server in
`internal/mcp/tools.go`). See the doc comments on `allowedBaseURLHosts` and
`allowCustomBaseURLEnvVar` in `config.go` for the full threat-model
reasoning, including the documented residual limitation (an attacker with
full env-write control can set both the BaseURL and the break-glass var at
once — accepted for a Medium-severity issue; the fix raises the bar rather
than claiming unbypassable isolation).

`doctor` already printed `report["base_url"]` before this change
(`internal/cli/doctor.go`); no new code was needed there beyond threading
the new `Load()` parameter through, since a denied host now surfaces as a
clear `config: error: ...` line instead of a silent fallback to the default.

## Retro candidate (possible upstream/verify-pipeline impact)

`config.go`'s pre-existing comment on the `NOWSHOWING_BASE_URL` override
says it is "used by printing-press verify to point at mock/test servers."
That verify pipeline lives in `mvanhorn/cli-printing-press` (external to
this repo) and was not modified as part of this fix. If that pipeline's
mock-server flow depends on setting *only* `NOWSHOWING_BASE_URL` (pointing
at a `127.0.0.1:<port>` httptest server, which is never in
`allowedBaseURLHosts`), it will now hit the new fail-closed error unless it
is updated to also set `NOWSHOWING_ALLOW_CUSTOM_BASE_URL=1` in its mock-mode
subprocess env (mirroring the existing `PRINTING_PRESS_VERIFY` /
`PRINTING_PRESS_VERIFY_LIVE_HTTP` pairing pattern already used for the
mutating-verb short-circuit). This repo's own `go test ./...` does not
exercise that external pipeline, so this is flagged here rather than
verified — file upstream against `mvanhorn/cli-printing-press` if a verify
run against this or another printed CLI starts failing on base_url
rejection.

Separately: `internal/httpx`'s `allowedRedirectHosts` (issue #9 / PR #20,
open/unmerged as of this patch) is a parallel, larger host allowlist for a
different subsystem (redirect-hop gating on the hand-written
`ctc`/`popcorn`/`imdb` source clients, which never route through
`internal/config`/`internal/client`). Consolidating both into one shared
allowlist package is a reasonable follow-up once #20 merges; this patch
does not depend on it.
