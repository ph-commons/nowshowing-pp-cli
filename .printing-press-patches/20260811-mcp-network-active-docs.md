# Document MCP as network-active; list non-read-only tools

**Date:** 2026-08-11
**Files:** modified `README.md` (Use with Claude Desktop, Agent Usage sections), `SKILL.md` (Agent Mode, new "MCP: network-active, and which tools write" section).

## Why

Security review of the fleet (issue ph-commons/nowshowing-pp-cli#14, parent
#6 "security review 20260810", Medium M4) found the README's MCP section
and the `pp-nowshowing` skill both state "read-only" without also stating
that MCP tools are network-active: `theater_showtimes`, `theater <slug>`,
`now-playing`, `search`, `popcorn`, and `movies imdb` all make live
outbound HTTP calls to ClickTheCity, popcorn.app, and/or IMDb on every
invocation. "Read-only" is true and unchanged — none of these tools
create/update/delete/mutate a remote resource — but a reader could
reasonably conflate "read-only" with "offline" or "sandboxed," which is
false.

Added a "MCP is network-active" subsection to README's `## Use with Claude
Desktop` section, extended the existing "Read-only by default" bullet in
`## Agent Usage`, extended SKILL.md's "Read-only" bullet in `## Agent
Mode`, and added a new SKILL.md section ("MCP: network-active, and which
tools write") enumerating every MCP-exposed command that does not carry
the `mcp:read-only` annotation, distinguishing `mcp:local-write` commands
(`teach`, `teach-pattern`, `teach-lookup`, `teach-playbook`, `playbook
amend`, `learnings candidates confirm`) from the two commands with no
safety annotation at all (`learnings forget`, `learnings candidates
reject`) and noting `learnings candidates purge` is `mcp:hidden` and not
MCP-exposed. All eight write only to this CLI's own local store — none
reach a third party.

Verified against source (`internal/mcp/tools.go`, `internal/cli/*.go`
`Annotations` maps, `internal/mcp/cobratree/classify.go`,
`internal/mcp/cobratree/shellout.go`'s `blockedRootFlags`) rather than
taken on trust from the issue body, which suggested "teach, deliver if
exposed" as examples — `--deliver` is in fact fully blocked from the MCP
surface (root persistent flag stripped from every tool's argument schema
before any handler runs), so the new text states that explicitly instead
of listing `--deliver` as a live non-read-only MCP surface.

Cross-linked issues #10 (MCP HTTP transport loopback-bind default,
already merged — still no built-in authentication) and #8 (`--deliver`
webhook SSRF hardening, already merged, and confirmed blocked from MCP)
as landed mitigations for the two agent-facing risks adjacent to this
one, pointing to SKILL.md's existing "MCP Server Installation" and
"Output Delivery" sections rather than re-describing them.

This is a docs-only change. No `.go` file, `internal/mcp/tools.go`
annotation, or `internal/cli/*.go` annotation was touched — every
command's actual network/local and read/write behavior already matched
its existing annotation (or correctly had none), so no annotation was
missing per issue #14's acceptance criterion 3.

## Retro candidate

None — straight docs addition following the same `.printing-press-patches/`
convention as the other README/SKILL-touching patches in this fleet
(`20260811-pinned-install-docs.md`, `20260811-mcp-http-bind-loopback.md`,
`20260811-harden-deliver-webhook-ssrf.md`).
