# Add unofficial / life-safety disclaimer to README

**Date:** 2026-08-10
**Files:** `README.md`

## Why
Security review (#6, "Trust & docs") flagged the README as missing the
standing unofficial-tool banner naming the upstream operators it reads from
(ClickTheCity, popcorn.app, IMDb) and a non-official-feed / non-life-safety
warning. Filed as child issue #15 (Low, L1), adapting the hub template
already shipped on `pse-edge-pp-cli`'s README.

Added a blockquote banner directly below the product blurb, above `## Install`,
naming the three upstream sources plus "any theater operator" and pointing
readers to each theater's own site/box office for official showtimes.

## Retro candidate (machine-level)
The hub-wide unofficial/life-safety disclaimer is expected on every printed
CLI per the hub charter (see #6). Whether this should become a Printing
Press template default (emitted for every generated CLI) rather than a
per-CLI hand-patch is worth raising upstream, but this CLI (`ph-commons/nowshowing-pp-cli`)
was detached from `mvanhorn/printing-press-library` at the 2026-08-10 org
transfer, so there is no longer an upstream template path to route this
through for this repo specifically.
