# Remove dead sync_hint.go + failing sync_hint_test.go

**Date:** 2026-07-28
**Files:** deleted `internal/cli/sync_hint.go`, `internal/cli/sync_hint_test.go`

## Why
This CLI declares no `sync` command / syncable store resources (showtimes are
fetched live; there is no local data layer). The generator emitted
`sync_hint.go` with `const syncHintsEnabled = false` (feature compiled off) but
STILL emitted `sync_hint_test.go` whose tests assert the enabled behavior
(`hintIfStale`/`hintIfUnsynced` returning true). With the const false, those
functions always return false, so 3 generated tests fail out of the box.

`maybeEmitSyncHints`/`hintIfStale`/`hintIfUnsynced`/`emitSyncHints` have **zero
non-test callers** (no sync command exists), so both files are dead code.

## Retro candidate (machine bug)
When `syncHintsEnabled` is false (no syncable resources), the generator should
not emit `sync_hint.go` or its enabled-path tests. File upstream.
