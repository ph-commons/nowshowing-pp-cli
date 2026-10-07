# Automate release cutting with release-please + GoReleaser

**Date:** 2026-09-26
**Files:** modified `.github/workflows/release.yml`, `.github/dependabot.yml`,
`CHANGELOG.md`, `.goreleaser.yaml`; added `release-please-config.json`,
`.release-please-manifest.json`.

## Why

Dependabot bumps auto-merged to `master` but nothing cut a release: the last tag
was `v0.1.2` (2026-08-12) while six dependency bumps landed after it. This makes
every releasable commit produce a patch release.

release-please maintains a release PR from Conventional Commits. A GitHub App
token lets that PR trigger `pull_request` CI, so branch protection and
auto-merge land it without a human. Merging the release PR creates the tag and
GitHub Release; the same workflow run then runs GoReleaser to attach prebuilt
binaries. One workflow gated on `release_created`, so the default `GITHUB_TOKEN`
never has to trigger a second workflow. `.goreleaser.yaml` sets
`release.mode: keep-existing` so GoReleaser does not overwrite release-please's
notes.

## Operator setup (not code)

Needs a GitHub App installed on the repo with repository permissions Contents:
read/write, Issues: read/write, Pull requests: read/write, plus repo variable
`RELEASE_APP_CLIENT_ID` and secret `RELEASE_APP_PRIVATE_KEY`. Without them the
Release workflow fails at the token step and no release is cut.

## Retro candidate

Printing Press could ship release-please automation by default instead of a
tag-only `release.yml`. Raise upstream.
