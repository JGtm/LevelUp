# Brief: make the 3D scene and the asset builders consumable by LevelUp

From: JGtm (LevelUp), 2026-11-22. For: Jordan and the coding agents working in
`llampwall/installation-08`. Read that repo's `CLAUDE.md` first and check `docs/sys/lookup.json`
before searching. Facts below were read at `a66b791460` (2026-11-22); anything marked
"unverified" is a lead to confirm, not a fact.

Nothing here is decided for Jordan. It is a proposal: if a part does not suit, say which and why.

## Problem

LevelUp wants Installation 08's 3D replay inside its own replay page, rendered exactly as
Installation 08 renders it. LevelUp will not rewrite or restyle the scene: this repo is the
reference for everything 3D.

Running `installation-08.exe` next to LevelUp and driving it with commands was considered and
set aside. It would put two film decoders, two sign-ins, two film caches and two data folders
behind one match page, so the 2D and 3D views would show two different documents of the same
film (different kickoff, different corrections).

Copying the scene into LevelUp was set aside too: a copy never follows this repo's releases.

## Goal

LevelUp depends on this repo the way this repo depends on LevelUp's modules: published
versions, bumped by an automatic pull request when a tag appears here. Three pieces of work,
in this order.

1. The changes made to the three vendored LevelUp modules go upstream.
2. The 3D scene is published as a web package.
3. The asset builders are published as an importable Go module.

Same repository, same release cadence, same credits and licence terms (AGPL-3.0, NOTICE.md).
The standalone app stays what it is and consumes its own two libraries.

## Part 1: upstream the module changes (blocking, do first)

Why first: the scene reads replay-document fields that the published decoder does not produce,
and a Go module's `replace` directives do not apply to whoever imports it. While the builders
compile against `third_party/` copies, nobody else can import them.

State of the three modules:

| Module | Vendored here | Published upstream |
|---|---|---|
| `halo-infinite-film-decoder` | v1.3.6 + local port, document schema 127 | v1.4.1 (schema 125); v1.5.0 in progress on branch `feat/v1.5.0` |
| `halo-infinite-map-images` | v1.0.1 + local changes | v1.4.0 |
| `halo-infinite-files-reader` | v1.0.1 + local changes | v1.2.0 |

The list of local changes is `third_party/README.md` ("Changes from upstream"), with the rebase
inventory in `third_party/REBASE-v1.3.6.md`.

Decoder, what to know before starting:

- **Schema numbers collide.** Upstream v1.5.0 takes document schema 126 for its own additions.
  This repo already uses 126 (`Projectile.Z`) and 127 (`Projectile.FullRate`). The local steps
  get new numbers at rebase, as was done once already for 113-122.
- **Target is v1.6.0.** v1.5.0 ships as planned, without these changes. Rebase onto the v1.5.0
  tag once it exists; until then `feat/v1.5.0` shows where it is heading.
- **Check for overlap first.** v1.5.0 adds channels of its own (shots and hits, named scoreboard
  counters, kickoff from the game clock), and v1.4.1 already has a damage document
  (`replay/document_damage.go`). Some local channels may already exist upstream in another
  shape. Diff before porting; where both exist, say so in the pull request and let JGtm pick.
- Channels to bring up: full-rate tracks, shot direction from the shooter's aim, aim-only
  samples, move-stick readings and control-view stances (crouch, slide, jump), shooter by control
  place, drawn weapon per slot, damage events per slot, kill causes, player actions (reload,
  throw, punch), carried-ammo reads, sub-frame shot offset, projectile height and full-rate
  flight, `filmcache.ParseManifest` / `NewSource`, `decfilm.LoadMapQuantCatalogBytes`.
- One pull request per channel or small group, each stating what changes in the output. A field
  LevelUp does not read is welcome: the decoder is a public project of its own.

Upstream rules that reject a pull request if missed (all three repositories are public):

- No real player identifier (xuid, gamertag) in a fixture or a test. No file extracted from the
  game in the repository. A test that opens the installed game is named `*_gamefiles_test.go`
  and carries `//go:build gamefiles`.
- A schema bump comes with its chronicle entry, the regenerated shape golden and the size caps
  it moves, as this repo's own port already did.
- No new heuristic fallback without a name and a counter. The grammar wins over heuristics.
- Outside the decoder module, code goes through the `decfilm` facade; the layers under
  `internal/` stay closed.
- Follow each repository's existing naming and comment language. Do not translate upstream
  files.
- JGtm posts the tags. A change that breaks compatibility needs a `/v2`, so prefer additions.

Done when: `go.mod` here requires published tags of the three modules, with no `replace` to
`third_party/` for them, and the app builds and plays a match.

## Part 2: the scene as a web package

`web/src/features/match-replay/ui/Replay3DView.tsx` already takes what a host page owns as
props (`doc`, `frameRef`, `viewpoint`, `colorOfXuid`, `locale`, `width`, `height`, `cameraMode`,
`playWindow`, `meshPath`, `variantPath`, `weaponsPath`, `armor`, `onLoadState`,
`onMinimapClick`). So this is mostly packaging, not a redesign.

What LevelUp's page will own and pass in: the replay document, the clock (one frame reference,
shared with the 2D view), the followed player, the solo / hide state per player, the locale, the
team colors, the size of the slot the view is mounted in. LevelUp keeps its own transport bar,
kill feed and player cards. The 3D view and its first-person HUD are this repo's.

Requirements:

- One public entry point that exports the view, its prop types and the camera modes. Everything
  else stays private.
- No assumption about the host: no router, no sign-in, no fetch to a fixed `/api/...` or
  `/static/...` path. Every URL arrives as a prop or through one resolver the host supplies.
  Unverified: whether code under `scene3d/`, `sound/` or the HUD builds URLs itself (weapon
  icons, HUD icons, sounds, callouts, first-person rigs). List each one found.
- No global stylesheet required by the host. Colors and insets come from CSS custom properties
  with defaults, documented in one place (`--bc-chrome-top`, `--bc-chrome-bottom` and the
  `--replay-hud-*` tokens exist already).
- `react`, `react-dom` and `three` as peer dependencies, exact `three` version stated. LevelUp
  runs React 19, TypeScript 6, Vite 8.
- Solo / hide: a player the host hides is hidden in every layer that belongs to that player
  (body, outline, name tag, shots, HUD marks). A player still in the match is always drawn,
  even when the film has no fresh sample for them. Unverified: how the view takes this today.
- The corner minimap and the X-ray team outlines stay this repo's. The host sets two things: whether
  the minimap is drawn, and its orientation (LevelUp locks it to the orientation of its own 2D view,
  it never turns with the camera). `onMinimapClick` exists already. Unverified: whether the minimap's
  background image can be supplied by the host; LevelUp would like to pass its own top-down map.
- **Frame-exact rendering for export.** LevelUp has a video export and a screenshot for the 2D
  view and they must work on the 3D view. The package exposes a way to render a given frame at a
  given size into a canvas the host can read, without the wall clock. This repo's own export
  already renders every frame in full, so the capability exists: make it part of the public
  entry point.
- Semver. A change to the entry point's props or to the fields it reads from the document is a
  breaking change and says so in the changelog.

Done when: this repo's own `MatchViewScreen` mounts the view through the public entry point
only, and a bare test page (no app shell, no router) can mount it with a document and asset
URLs and reach `ready`.

## Part 3: the asset builders as a Go module

LevelUp's server builds and serves the assets itself, in its own data folders, with its own
disk rules (on-demand build per map, size cap, free-space guard). It needs the builders as
functions, not the server.

- Move what builds assets out of `internal/` into importable packages: map mesh and variant
  layer, weapons, Spartan pieces and armor, first-person rigs, coatings, HUD icons, callouts,
  ambience, announcer and sounds. Unverified: which of `internal/` is builder and which is app;
  `cmd/infinite-theater-3d/build*.go` holds logic that may belong with the builders.
- Inputs are explicit: the game's `deploy` root, an output directory, a context for
  cancellation, a progress callback. No environment variable read, no fixed data folder, no
  global state. LevelUp strips `LEVELUP_*` variables from its server's environment.
- Memory is bounded and stated: LevelUp runs one heavy build at a time on a user's machine.
- A function that says whether a given map can be built from a given install, without building
  it. LevelUp uses it to mark matches that have a 3D replay.
- Each build writes a format version next to its output, so the host knows when to rebuild
  after an update here.
- cgo stays (`ooz`, Basis Universal, meshoptimizer). State the toolchain in the module README.
- Nothing under `cmd/`, the window, sign-in, match fetch or the HTTP server is part of this
  module.

Done when: `go list -deps` of the builder packages shows nothing from `cmd/` and no vendored
copy of the three LevelUp modules, and a ten-line program outside this repo builds one map.

## Out of scope

- Any change to how the scene looks or sounds.
- A command or remote-control API for the exe.

## Open questions for Jordan

1. Is the split acceptable as described, or is there a shape that costs you less?
2. Publishing: npm registry, GitHub Packages, or a git tag LevelUp installs from?
3. Do you want to open the upstream pull requests yourself, or should JGtm's side port from
   `third_party/` with your review?
