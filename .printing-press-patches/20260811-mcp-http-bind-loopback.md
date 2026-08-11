# Bind the MCP HTTP transport to loopback by default; require --insecure-bind for anything else

**Date:** 2026-08-11
**Files:** modified `cmd/nowshowing-pp-mcp/main.go` (changed
`defaultHTTPAddr`, added `--insecure-bind` flag, added `resolveBindAddr` and
`startHTTPServer`, refactored `main()`'s flag-parsing/dispatch into a
testable `run()`), added `cmd/nowshowing-pp-mcp/main_test.go`, modified
`SKILL.md` (MCP Server Installation section).

## Why

Security review of the fleet (issue ph-commons/nowshowing-pp-cli#10, parent
#6 "security review 20260810", High H4) found `cmd/nowshowing-pp-mcp/main.go`
set `defaultHTTPAddr = ":7777"`, which binds all interfaces when the server
runs `--transport http`. The HTTP transport has no authentication, so the
all-interfaces default silently exposed the MCP tool surface to the LAN (or
further, if hosted with an open security group) with no signal to the
operator.

Fix: default to `127.0.0.1:7777`. A new `resolveBindAddr` helper validates
that `--addr` resolves exclusively to loopback addresses (literal IP check
via `net.IP.IsLoopback()`, or — for a hostname — every resolved address must
be loopback) and refuses to start the HTTP transport on anything else unless
`--insecure-bind` is passed. For a hostname that resolves to loopback, the
validated address is pinned into the address actually bound (rather than
re-handing the original hostname string to the HTTP server, which would
re-resolve it independently and could diverge from what was validated).
`stdio` (the default transport) is unaffected — its dispatch logic was
relocated into a new `run()` function alongside the HTTP logic to make the
whole thing testable, but the `server.ServeStdio(s)` call and its error
handling are unchanged.

## Residual risk (documented, not fixed by this patch)

Binding loopback-only closes the network-exposure gap (H4's actual finding)
but does **not** add authentication. Any other local process or user account
on the same machine can still reach a loopback-bound HTTP MCP server with no
credential check. The issue frames a bearer token for HTTP mode as a later,
optional follow-up ("Optional: bearer token for HTTP mode (later)") — this
patch does not claim to close that gap, and `SKILL.md`'s new note says so
explicitly ("has no built-in authentication either way").

## Retro candidate (machine bug / generator gap)

Same class as the `.printing-press-patches/20260811-install-checksum-verification.md`
entry from issue #7: `cli-printing-press`-generated MCP server mains default
the HTTP transport to an all-interfaces bind with no loopback gate. Worth
raising upstream in `mvanhorn/cli-printing-press` as a generator-template
default, rather than a one-off per printed CLI — no local generator checkout
was available in this run to make that change directly.
