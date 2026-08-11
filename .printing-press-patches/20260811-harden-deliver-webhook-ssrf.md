# Harden --deliver webhook against SSRF/exfil (issue #8)

**Date:** 2026-08-11
**Files:** modified `internal/cli/deliver.go` (deliverWebhook, new
deliverTLSConfig test-only var); added `internal/cli/deliver_ssrf.go`
(denylist/allowlist policy, deliverStderr var), `internal/cli/deliver_ssrf_test.go`,
`internal/cli/deliver_test.go`

## Why
`internal/cli/deliver.go`'s `deliverWebhook` POSTed to any user-supplied
`http://`/`https://` URL with no guard against loopback/link-local/RFC1918/
cloud-metadata targets — a classic SSRF/exfil vector (portfolio security
review 20260810, High H2, tracked as ph-commons/nowshowing-pp-cli#8). Fixed
by validating the resolved IP inside a custom `http.Transport.DialContext`
hook (closes the DNS-rebinding TOCTOU a request-build-time check would leave
open), refusing all redirects, warning on plain http, and adding an explicit
`NOWSHOWING_DELIVER_ALLOW_HOSTS` (IP/CIDR only, never hostnames — a hostname
entry would trust whatever that hostname resolves to at connect time) opt-in
for deliberate internal sinks (per the issue's own acceptance criteria).

## Retro candidate (machine bug)
The generator should not emit a webhook delivery sink without SSRF guards by
default — any printed CLI with `--deliver webhook:<url>` inherits this gap.
File upstream against CLI Printing Press.
