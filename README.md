# NowShowing CLI

**Metro Manila movie showtimes as a CLI — every tracked cinema in one now-playing board, cross-checked across two sources.**

NowShowing turns ClickTheCity's per-theater schedules into an agent-native CLI. now-playing fans out across every tracked Metro Manila and Iloilo cinema in one call, cross-checks each showtime against popcorn.app for two-source confidence, and folds in verified ticket prices and IMDb links.

> **Unofficial.** Independent, community-built tool — **not affiliated with, endorsed by, or supported by ClickTheCity, popcorn.app, IMDb, or any theater operator**. It reads publicly available showtime and ticketing pages; upstream structure can change without notice. For official showtimes and ticketing, use each theater's own website or box office. Never rely on this tool for life-safety decisions or as a substitute for official/licensed feeds.

## Install

**Source:** [github.com/ph-commons/nowshowing-pp-cli](https://github.com/ph-commons/nowshowing-pp-cli) (PH Commons)

**Trust model.** The `curl|bash` installer below is a convenience path, not a signed artifact: its trust root is the GitHub org (`ph-commons`), TLS on every fetch (installer script, release API, tarball, checksums), and a SHA-256 checksum check of the downloaded tarball against the release's published `checksums.txt` — not cryptographic signing (no cosign/minisign today; see [#11](https://github.com/ph-commons/nowshowing-pp-cli/issues/11)). On a host you don't fully trust, or where "curl a script from the internet and run it" is against policy, prefer the pinned **Go install** or **Pre-built binary (manual)** paths below instead.

### Recommended (prebuilt release)

```bash
curl -fsSL https://raw.githubusercontent.com/ph-commons/nowshowing-pp-cli/master/scripts/install.sh | bash
```

Installs `nowshowing-pp-cli` (and companion MCP binary when present in the release) into `$GOBIN` or `~/.local/bin`. Requires a matching GitHub release asset for your OS/arch.

Verifies the downloaded tarball's SHA-256 against the release's published `checksums.txt` before extracting anything; the install aborts (non-zero exit) if `checksums.txt` is missing or the hash doesn't match, rather than silently falling back to a source build.

### Go install

Requires Go 1.26.5+:

```bash
go install github.com/ph-commons/nowshowing-pp-cli/cmd/nowshowing-pp-cli@latest
go install github.com/ph-commons/nowshowing-pp-cli/cmd/nowshowing-pp-mcp@latest
```

**Pinned (recommended for untrusted hosts):** pin to a released tag instead of `@latest` so the build is reproducible and doesn't silently pick up a newer, unreviewed release:

```bash
go install github.com/ph-commons/nowshowing-pp-cli/cmd/nowshowing-pp-cli@v0.1.1
go install github.com/ph-commons/nowshowing-pp-cli/cmd/nowshowing-pp-mcp@v0.1.1
```

Check [the latest release](https://github.com/ph-commons/nowshowing-pp-cli/releases/latest) for the current tag. `go install` builds from source via the Go module proxy/checksum database (`sum.golang.org`) rather than downloading a prebuilt binary, so it sidesteps the release-tarball trust question in `scripts/install.sh` entirely.

### Pre-built binary (manual)

1. Download the archive for your OS/arch (e.g. `nowshowing-pp-cli_<version>_darwin_arm64.tar.gz`) from the [latest release](https://github.com/ph-commons/nowshowing-pp-cli/releases/latest).
2. **Verify without running the installer script:** download `checksums.txt` from the same release into the same directory as the still-archived download (`checksums.txt` has an entry per archive filename, not per extracted binary — verify before extracting):
   ```bash
   sha256sum -c checksums.txt --ignore-missing   # Linux
   shasum -a 256 -c checksums.txt --ignore-missing  # macOS
   ```
3. Extract the archive, then: on macOS: `xattr -d com.apple.quarantine <binary>`; on Unix: `chmod +x <binary>`.

### Agent skill (`pp-nowshowing`)

Install the skill from your Hermes / Claude Code skill tree (canonical: `hermes-config/skills/pp-nowshowing`), or copy `SKILL.md` from this repo.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

1. Install the CLI (curl installer or `go install` above).
2. Ensure `nowshowing-pp-cli --version` works and the bin dir is on `$PATH`.
3. Install/enable the `pp-nowshowing` skill in Hermes (`hermes skills` / skill directory), then restart the session if needed.

## Install for OpenClaw

Install the CLI binary first, then install the OpenClaw-focused skill from your skill distribution. Restart the OpenClaw session if the skill is not visible immediately.

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/ph-commons/nowshowing-pp-cli/releases/latest).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

### MCP is network-active

The MCP server is **not** offline or sandboxed. Its read tools (`theater_showtimes`, `now-playing`, `search`, `popcorn`, `movies imdb`, and the typed `theater <slug>` endpoint) make live outbound HTTP calls to ClickTheCity, popcorn.app, and IMDb — as applicable per tool — on every invocation. "Read-only" describes these tools' contract with the *remote* services (no create/update/delete/mutate), not an absence of network activity: every one of them talks to the internet each time an agent calls it. `SKILL.md`'s "MCP: network-active, and which tools write" section lists the few tools that also write, but only to this CLI's own local store, never to a third party.

Two related agent-facing risks already have merged fixes: the MCP server's optional `--transport http` mode binds loopback-only by default and has no built-in authentication ([#10](https://github.com/ph-commons/nowshowing-pp-cli/issues/10)); the `--deliver webhook:<url>` output sink is hardened against SSRF and is blocked from the MCP tool surface entirely, so it cannot be reached by an MCP caller ([#8](https://github.com/ph-commons/nowshowing-pp-cli/issues/8)). See `SKILL.md`'s "MCP Server Installation" and "Output Delivery" sections for the full detail on each.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


```bash
go install github.com/ph-commons/nowshowing-pp-cli/cmd/nowshowing-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "nowshowing": {
      "command": "nowshowing-pp-mcp"
    }
  }
}
```

</details>

## Quick Start

```bash
# confirm the ClickTheCity source is reachable
nowshowing-pp-cli doctor --dry-run

# see the tracked cinemas and their slugs
nowshowing-pp-cli theaters

# one theater's schedule for today
nowshowing-pp-cli theater sm-megamall

# the full board across every tracked cinema
nowshowing-pp-cli now-playing --agent

```

## Unique Features

These capabilities aren't available in any other tool for this API.

### Cross-theater aggregation
- **`now-playing`** — Every movie showing today across all tracked Metro Manila and Iloilo cinemas, in one call.

  _Reach for this to answer 'what's showing today near me' without querying each cinema separately._

  ```bash
  nowshowing-pp-cli now-playing --agent
  ```
- **`search`** — Find which tracked theaters are playing a given movie today, and at what times.

  _Reach for this when the user names a movie and wants to know where to watch it._

  ```bash
  nowshowing-pp-cli search "Superman" --agent
  ```
- **`theaters`** — Lists every tracked cinema with its ClickTheCity slug, display name, and city.

  _Reach for this to discover valid --slug / --theater values before querying schedules._

  ```bash
  nowshowing-pp-cli theaters --agent
  ```

### Source confidence
- **`now-playing`** — Flags each showtime as verified (two sources agree), partial, mismatched, or ClickTheCity-only by cross-checking popcorn.app.

  _Reach for this when showtime accuracy matters and a single listing site can't be trusted alone._

  ```bash
  nowshowing-pp-cli now-playing --theater sm-megamall --agent
  ```

### Enrichment
- **`movies imdb`** — Resolves a movie title to its IMDb page, flagging remake/re-release title collisions by recent year.

  _Reach for this to enrich a showtime listing with the correct IMDb link, not a decades-old namesake._

  ```bash
  nowshowing-pp-cli movies imdb --title "Moana" --agent
  ```

## Recipes

### Full board, agent-native

```bash
nowshowing-pp-cli now-playing --agent --select theater,movie,showtimes,confidence
```

Every cinema's schedule today, narrowed to the fields an agent needs.

### Where is a movie playing

```bash
nowshowing-pp-cli search "Superman" --agent
```

Inverted view: which tracked theaters and times are showing the named movie today.

### One theater, specific date

```bash
nowshowing-pp-cli theater greenbelt-3 --date 2026-07-30 --json
```

A single cinema's schedule for a chosen date as JSON.

## Usage

Run `nowshowing-pp-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.toml` and saved profiles |
| `data` | Durable local data such as `data.db` |
| `state` | Runtime state such as persisted queries, jobs, and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The ladder is:

1. Per-kind env var: `NOWSHOWING_CONFIG_DIR`, `NOWSHOWING_DATA_DIR`, `NOWSHOWING_STATE_DIR`, or `NOWSHOWING_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `NOWSHOWING_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export NOWSHOWING_HOME=/srv/nowshowing
nowshowing-pp-cli doctor
```

Under `NOWSHOWING_HOME=/srv/nowshowing`, the four dirs resolve to `/srv/nowshowing/config`, `/srv/nowshowing/data`, `/srv/nowshowing/state`, and `/srv/nowshowing/cache`.

MCP servers do not receive CLI flags from the host. Put relocation in the host `env` block:

```json
{
  "mcpServers": {
    "nowshowing": {
      "command": "nowshowing-pp-mcp",
      "env": {
        "NOWSHOWING_HOME": "/srv/nowshowing"
      }
    }
  }
}
```

Precedence matters in fleets: an ambient per-kind variable such as `NOWSHOWING_DATA_DIR` overrides an explicit `--home` for that kind. Use `NOWSHOWING_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `NOWSHOWING_HOME` does not move files back to platform defaults, and `doctor` cannot find files left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. Run `nowshowing-pp-cli doctor --fail-on warn` to check path warnings in automation.

## Commands

### theater

Per-theater movie schedules from ClickTheCity (primary source)

- **`nowshowing-pp-cli theater <slug>`** - All movies now showing at one theater on a given date, with per-screen showtimes. Join now_showing (movie metadata) to schedules (per-screen showtime lists) on movieId.


### Self-learning loop

This CLI caches per-question discovery so repeat queries skip the walk and structurally similar queries get answered via entity substitution. The loop also self-captures: every invocation is journaled locally, and failed-flag corrections plus fresh teaches surface as candidates on the next `recall` for confirm/reject judgment. Agents call `recall` before discovery and fire `teach &` after answering. See the `## Automatic learning` section in `SKILL.md` for the full protocol.

- **`nowshowing-pp-cli recall <query>`** - Look up cached resources for a query before running discovery
- **`nowshowing-pp-cli teach`** - Record a query -> resource mapping (silent on success, safe to background with `&`)
- **`nowshowing-pp-cli learnings list`** - Inspect taught rows
- **`nowshowing-pp-cli learnings forget <query>`** - Undo a teach
- **`nowshowing-pp-cli learnings candidates`** - List auto-captured candidates awaiting confirm/reject
- **`nowshowing-pp-cli learnings stats`** - Local loop metrics: recall hit rate, teach-to-reuse, playbook resolution, candidate counts
- **`nowshowing-pp-cli teach-pattern`** - Install a query/resource template up front
- **`nowshowing-pp-cli teach-lookup`** - Add an entity mapping (e.g. country code, team alias) for pattern substitution

Pass `--no-learn` or set `NOWSHOWING_NO_LEARN=true` to disable the loop for deterministic flows.

The local store's schema version stamp is one-way: once this version of `nowshowing-pp-cli` opens the database, older binaries refuse it with a version error — upgrade the binary rather than downgrading.

## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
nowshowing-pp-cli theater mock-value

# JSON for scripting and agents
nowshowing-pp-cli theater mock-value --json

# Filter to specific fields
nowshowing-pp-cli theater mock-value --json --select id,name,status

# Dry run — show the request without sending
nowshowing-pp-cli theater mock-value --dry-run

# Agent mode — JSON + compact + no prompts in one flag
nowshowing-pp-cli theater mock-value --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Read-only by default** - this CLI does not create, update, delete, publish, send, or mutate remote resources. This is a contract about not mutating remote state, not about network activity: reads still make live outbound calls to ClickTheCity, popcorn.app, and IMDb as applicable — see "MCP is network-active" above for the MCP surface specifically
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
nowshowing-pp-cli doctor
```

Verifies configuration and connectivity to the API.

## Configuration

Run `nowshowing-pp-cli doctor` to see the resolved config, data, state, and cache directories. The platform-default config path is `~/.config/nowshowing-pp-cli/config.toml`; `--home`, `NOWSHOWING_HOME`, and per-kind env vars can relocate it.

Static request headers can be configured under `headers`; per-command header overrides take precedence.

## Troubleshooting
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific
- **theater <slug> returns an error or empty result** — confirm the slug with 'nowshowing-pp-cli theaters'; ClickTheCity requires the browser User-Agent header the CLI sends by default
- **a movie shows 'ClickTheCity only' confidence** — popcorn.app has no parseable page for that cinema (common for SM-managed venues); the ClickTheCity times are still valid
