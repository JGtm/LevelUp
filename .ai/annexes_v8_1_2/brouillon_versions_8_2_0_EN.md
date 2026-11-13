# Release notes 8.2.0, EN DRAFT (2026-11-12, updated 2026-11-13)

Base: `origin/release/v8.2.0` at `31f91a79e8` (PRs #181 to #206 merged; PR #207 is open, see below).
Nothing is placed in `docs/`: the two `TestReleaseNotesDocs_*`
tests require the newest notes block to carry the number of the first versioned changelog entry, so the `v8.2`
block and the `[8.2.0]` entry go in TOGETHER at closure, with the version bump (`Cargo.toml`, `Cargo.lock`,
`tauri.conf.json`). The French counterpart is `brouillon_versions_8_2_0_FR.md`. British English throughout.

Lines marked **[TO CONFIRM]** depend on a pull request still open, **#207** (baking at 30 frames per second,
icon and killed-occupants count on vehicle destructions, decoder v1.3.6, schema 115): the PR is open and its merge
is held until the verdict on five "not established" lines of the corpus gate. Remove or finalise them before
closure; none may ship as is. #203 (vehicle destructions, decoder v1.3.4, schema 113), #205 (map backgrounds held
by the navigation mesh), #206 (dependency security alerts) and steps 4 to 8 of the architecture plan are merged:
their lines are in, unmarked. As read on 2026-11-13, without #207 baking stays at 10 frames per second (100 ms step).

## 1. Block for `docs/RELEASE_NOTES.md`

Paste at the top of the file, under `## What's new`, before the `**v8.1.2 — ...**` block. The first section is
reused as is for the updater notes and the release page.

````markdown
**v8.2 — A fuller replay, truer map backgrounds, Lead share, new match charts and compressed replays**

**The replay**
- **A feed you can filter** — the feed title opens a menu: All, Kills, Objectives, Weapons and power-ups, Vehicles, Medals, Assists. Objective actions appear as banners, weapons picked up from the ground join the feed, and a filter with no line in the match says so instead of staying empty. Your choice is remembered
- **Click a kill** — playback pauses at that moment, and the killer and the victim are highlighted on the map with their names and the weapon
- **The zone of the kill** — each kill in the feed shows the map zone the killer stood in, when the map names one
- **A "View" button** — in the corner of the map, four layer presets (Full, Combat, Movement, Objective) and a panel gathering effects, heat, dot colour and sound. The Settings button leaves the playback bar
- **A player card in a panel** — a click on a card replaces the feed with their end-of-match totals (accuracy, damage, score breakdown, objective, weapons), with Follow, Solo and Hide. Esc closes the panel
- **The film read more faithfully** — positions are read at the right widths on maps with several regions, and object types and the Spartan's abilities are recognised more rigorously: a regenerated replay gains in precision
- **Fixed turrets, bots and places of death** — a fixed turret is recognised as such, stray tracks are no longer shown, the bots of a film without a kill feed keep their names, and kills that had no location get one
- **Game icons for grenades and abilities** — the replay thumbnails come from the game's own interface sheets, in line art for grenades
- **Destroyed vehicles** — the feed and its "Vehicles" filter list every vehicle destruction ("Ghost destroyed", or "destroyed: Ghost" when the destroyer is known), with the player aboard in brackets, or their number when there are several. Players who died with their vehicle are read from the film; a player ejected then killed, or killed just before the vehicle, is not counted. Destructions appear in a regenerated replay
- **Postures and ground weapons read more accurately** — a sprint or a climb no longer stays open after the player's death, and a ground weapon is more often tied to the player who picks it up
- **Map backgrounds held by the navigation mesh** — the twenty native maps are cropped to the area that is really playable, the reference floor comes from the game's mesh, and a new "Wall plan" layer draws the walls as vectors between the background and the zones (switch in the "View" panel, on by default, available for 111 backgrounds out of 114). Live Fire has no mesh: its background stays as before
- **A map you can turn** — an "Orientation" setting in the "View" panel turns the map by quarter turns, clockwise, and remembers it for each map. Tactics and "Terrain occupation" read the same orientation and the same wall plan
- **[TO CONFIRM, #207 open] Replays at 30 frames per second** — movement is smoother, position, aim and shield included; a replay baked at 10 frames per second is recognised as "to redo" ("stale cadence"), and nothing is re-baked on its own
- **[TO CONFIRM, #207 open] Vehicle destructions, completed** — the "destroyed" feed line carries the icon of the weapon that destroyed the vehicle, and the brackets say "(N killed with the vehicle)" when players died with it

**The match view and Lead share**
- **Two charts in the Timeline tab** — "Score gap" and "Numerical advantage" (players alive, bots included), drawn as areas around zero in the team colours. With more than two teams the comparison is against the best opposing team; without a replay, the card says so
- **Lead share** — the share of the match time spent alone in the lead of the score, with the number of lead changes. On the match view, in Sessions, Time series (by match, by month, by map) and Squad, for Halo Infinite. A dash replaces the figure when it cannot be worked out, for example in a mode where the score moves too rarely (flag, hill, bomb)

**Compressed replays**
- **Five times less space** — replays are stored compressed, with identical content: on a local collection, 3.6 GB become 0.75 GB. Older replays stay readable with nothing to do
- **Compression from the Windows app** — the replay update launched from the app writes them compressed, and also offers to compress the ones that can no longer be rebuilt

**Scores and statistics**
- **LUSR scores are recalculated at the first start** — a match the calculation could not resolve finally gets a score. As each score depends on the previous ones, most of those in your history move, usually by less than five points
- **Net FDA in Solo versus Squad and in behaviour alerts** — FDA is read as kills plus a third of assists minus deaths, per match, rather than as a quotient; the formula in Help is corrected in every language, and the losing-streak alert is judged on a drop in FDA points
- **Behaviour alerts in your language** — eleven sentences written in French by the server now go through the language catalogue
- **Win probability removed** — an abandoned statistic, removed from the Synergies and Progression tables and from the match detail

**The interface and the Windows app**
- **A more compact sidebar** — the foot becomes a row of icons, folded it keeps a "More" menu, and a tooltip opens on focus only when the focus comes from the keyboard
- **A sliding references pane** — maps, weapons and medals open from the side, with a pinned mode
- **A "Browse" button** in the Windows app settings to choose a folder
- **Cleaner updates** — the old interface files are removed on update and on uninstall
- **Link previews in your language** — a link copied from the app carries your language, a bare link shows in English
- **One sort rule in the tables** — the administration tables, the crossed players of the Career and the Leaderboard sort by the same rule: first click in the column's own direction, then toggling between ascending and descending, missing values always last
- **Two charts that follow the accessibility palette** — the cumulative frag gap (Honours, Explorer) and the sequence tape change colour as soon as the palette changes, without reloading the page
````

## 2. Block for `docs/CHANGELOG.md`

Replace `## [Unreleased]` by an empty `## [Unreleased]`, then place this below it, the date being the
publication date. The "Game icons" entry takes over the one already under `[Unreleased]`. The file is read at
runtime by the changelog page: leave nothing **[TO CONFIRM]** in it at merge.

````markdown
## [8.2.0] - YYYY-MM-DD

> The replay, the map backgrounds and Lead share on the player side; on the code side, LevelUp consumes the four published bricks from their repositories, `libs/` leaves the repository and the HTTP, storage, multi-title and web layers are separated by guard rails at ceiling 0.

### Added
- **Lead share (`film.lead_share`)** — persisted per-match derived value: share of the played time spent alone in the lead of the score, and number of lead changes. Migration `shared_match_lead_share_v1` (append-only table `match_lead_share`, view `match_lead_share_latest`), `persist.LeadSharePersister`, pure computation `internal/analysis/leadshare` with a case set shared by the Go and web tests, command `levelup backfill-lead-share` (`--dry-run` read-only, no migration). Fields served on the match view, Sessions, Time series and Squad; absent, never zero, when the match is not measured (few-points mode, score not covered, kick-off not measured).
- **Replay page** — filterable feed (All, Kills, Objectives, Weapons and power-ups, Vehicles, Medals, Assists; filter remembered in `replayPreferences`), click on a kill (pause and highlight of the killer and the victim), weapons picked up from the ground in the feed, "View" button and four layer presets (`viewPresetLogic`), player card in a panel with Follow, Solo and Hide, killer's zone in the feed (port of `NommerZone`, no new field and no re-bake).
- **Match view charts** — "Score gap" (`_scoreDelta.ts`) and "Numerical advantage" (`lib/replay/aliveBalanceLogic.ts`) in the Timeline tab, shared ECharts option `_divergentStepOption.ts`.
- **Compressed replays** — artefacts are stored as `{short8}.json.gz` (gzip level 6) behind a single gate, `internal/replayartifact` (form detected by the magic bytes, mixed collection read, the compressed one wins); `levelup replay compress [--dry-run]` (in-place, atomic, identical read-back before deletion, never automatic); on the desktop, the regeneration queue compresses at its end the raw files it could not rebuild and the update offer proposes compression alone (`raw`). Fallback without a hard link for exFAT and FAT32. Decompressed content unchanged.
- **Replay thresholds in milliseconds** — every threshold expressed in frames becomes a duration, converted with the document's step (`frameIntervalMs` on the web side, `framesOfMS` in the decoder, `HistoricalFrameIntervalMS` = 100); at 100 ms no value changes.
- **Film reading (decoder)** — positions read at the widths of the region the stream designates (per-region-index table in `map_quant_bounds.json`, `profile.MapQuantEntry.Regions`), entity type bound to its full identifier, Spartan ability (tag 3). Decoder v1.3.4 at the end of the release: `SchemaVersion` 113, revisions `grammar-2026-11-12.4`, `killsource-2026-11-10`, `profile-2026-11-12.2`, `UsageSummaryRev` `us7` unchanged.
- **Vehicle destructions** — the replay document carries `vehicles[].destruction` (human destroyer by xuid or bot by roster entry, players aboard, occupants killed with the vehicle read from the film: the biped's dead-state written in the destruction packet or after, and a parent reading that ties it to the vehicle; ejected then killed, killed just before, and end-of-round clean-up deaths are not counted) and the coverage block `coverage.vehicles.destructions` (`killedNoIdentity`, `killedCleanup`); the page reads these fields (`vehicleDestructionsOf`, "destroyed" line of the feed and of its "Vehicles" filter, "aboard" bracket in the language's plural). `destruction.causeKey` is in the contract but stays empty: filling it at request time is a separate batch. On 266 films with vehicles (1,687 named occupants): 1,554 killed read, agreement on 1,529 with the former proximity rule (1,589).
- **Map backgrounds held by the navigation mesh** (ADR 0048, amendment to ADR 0043) — native maps are baked with their mesh (read in the `any` module by `halo-infinite-files-reader` and `halo-infinite-map-images` v1.1.0): the rich background stays the base, the mesh serves as floor reference and crop mask (1.5 m by default, per-map settings in `map_fond_reglages.json`, levers `rogneAuNavmesh` and `sansNavmesh`); 20 native backgrounds re-baked, Live Fire without a mesh (named degradation). Wall plan (`MapBackground.plan`, boundary edges in world metres) in the sidecar for 111 backgrounds out of 114, native and Forge (`mapfond-build -plan-seul` adds it without re-baking); package `internal/mapfondsidecar`, the only reader and writer of the sidecar on LevelUp's side.
- **Web: wall plan and orientation** — static "Wall plan" layer (switch `replay-show-wall-plan`, on by default) and orientation by clockwise quarter turns (`replay-orientation.<map_id>`, per map) in the replay's single projection; Tactics and "Terrain occupation" read the same preferences (`useMapViewPreferences`).
- **Replay: fixed turret, bots, place of death** — fixed turret chassis recognised, stray tracks dropped, bots named on a film without a kill feed (roster read in `BOT_METADATA`), place of death read per body (`IsolationDecoderRev` raised).
- **Desktop** — `pick_folder` IPC command ("Browse" button, a single permission, limited to `127.0.0.1`) and capability `can_pick_folder` true for the `desktop` profile only; ADR 0045 amended. The old interface files are removed on update (`hooks.nsh`) and on uninstall (`uninstall_cleanup_engine.go`).
- **Web** — compact sidebar foot and "More" menu, sliding references pane with pinned mode, tooltips opened on keyboard focus only (`lib/dom/focusVisible.ts`), `--scrim` token in place of absolute blacks and whites (guard rail extended to `bg-black` and `text-white`).
- **Game icons for grenades and abilities** — thumbnails extracted from the game's interface atlases, at the same paths; the resource pack changes (a single download for the desktop application); the unused `static/abilities-assets` folder is removed.

### Changed
- **Published bricks** — LevelUp consumes `halo-infinite-files-reader`, `halo-infinite-map-images`, `halo-api` and `halo-infinite-film-decoder` from their dedicated repositories (at the end of the release: `halo-api` v1.0.1, `halo-infinite-files-reader` v1.1.0, `halo-infinite-map-images` v1.1.0, `halo-infinite-film-decoder` v1.3.4; #207 would raise the latter to v1.3.6); `libs/`, `go.work` and `go.work.sum` are removed, module paths renamed (`github.com/JGtm/...`). `halo-api` carries `retract [v0.1.0, v1.0.0]` (the ranking capture in its `v1.0.0` was not pseudonymised). Single resolver `internal/briques` (`go list -m -f {{.Dir}}`), script `scripts/briques-workspace.sh` to develop a brick in a clone (`levelup-briques.work`, never a `go.work`), `replay-equiv -update` accepted on a clone outside the repository and refused on the module cache. Docker, CI, Makefile, lefthook, Dependabot and baselines rewritten; the bricks' tests run by module path from `apps/go-api`, their lint lives in their own repository. File-size and function-length ratchets reduced to LevelUp's own tree.
- **Environment switches removed** — `LEVELUP_USE_SHARED_PROVIDER`, `LEVELUP_POSTSYNC_BURST`, `LEVELUP_SYNC_V2_DRYRUN`, `LEVELUP_PERSIST_NO_FETCH_CACHE`, `LEVELUP_LUSR_V2_MODE_COUPLING`, `LEVELUP_LUSR_V2_SQUAD_OFFSET`, `LEVELUP_WATCHER_BROADCAST`, `PRESTIGE_ENABLED` and `LEVELUP_LOG_JSON`; the image sets `LEVELUP_LOG_FORMAT=json`. Six variables are read once by `internal/config`; `os.Getenv` and `os.LookupEnv` are forbidden outside `internal/config` and `cmd` (`archlint` guard rail). Two compatibility branches remain, dated 2026-12-31. Automatic repair of rating gaps is removed.
- **Net FDA** — Solo versus Squad reads the match's native FDA (fallback on the canonical calculation), the losing-streak alert is judged on a drop in points (`TiltFDADropPoints` 2.5, high severity 4.5), the Help formula and `Assistances/2` in both `fields.toml` are corrected.
- **LUSR** — solver fallback without the kill and death counts when it does not converge (`LUSRRev` = `lusr-2026-11-09`): full replay at the first start (ADR 0047), non-converging matches receive a score.
- **Behaviour alerts localised** — eleven sentences move to the `internal/i18n/messages/patterns.toml` catalogue.
- **Writes** — awards and the `psa_checked_at` marker written after the alias, match intensities not written are logged and counted, unreadable rows are logged, the `duckdb.OpenReadWrite` surface frozen by a dated list of 43 files, exception lists dated from git.
- **Architecture, steps 4 to 8 (architecture plan, closed)** — HTTP: handlers hold port interfaces and no longer import the `service` package or a sync engine, `api/wire` carries neither SQL nor `database/sql`, static files and the web page live in `api/handlers`, the classification of a read failure goes through `platform/dbfault`, session purge is a background task declared in the four profiles. Storage: all SQL and all `database/sql` leave `service/`, `analysis/`, `progression/`, `scheduler/`, `prestige/` and `port/` for repositories behind ports (`importstore`, `cronstore`, `profilerepo`, `worldrepo`, `sqlfrag`, `playerresolve`); the "not found" error is a domain error; four guard rails at ceiling 0. Multi-title: `service/` imports no title package any more (16 imports brought to 0), a title with no replay gets `ErrCapabilityNotSupported`. Web: feature boundaries checked with no per-file exception, `echarts` imported from a single folder, network calls through the common client (`api.getText`, `api.getStatic`), eleven hand-sorted tables moved to TanStack Table (`useColumnSort`), chart statistics in `lib/charts/stats`, single read of the bootstrap, the 2D Replay page becomes a feature page, hand-written API types replaced by the generated contract (the `GET /settings` contract is completed: 56 fields instead of 30). Closure: identical witnesses before and after on 90 routes (12 of them for Halo 5). Dated remainders, not held at zero: `os.Getenv` outside `internal/config` (13 in a dated list), `fetch` outside the client (4), API types sharing a name with a contract schema (11).
- **Link previews** — language of the link (`?lang=`), then the crawler's, then the fallback language; the "Copy link" button carries the sharer's language.
- **Licences** — origin of the bricks' data written down, `coupe.png` removed, `sbsp.xml` credited to Infinite Runtime Tag Viewer, `halo-api` ranking capture pseudonymised, real identities replaced in the `apps/` fixtures (registry `IDENTITES_APPS.tsv`, guard rail extended).
- **Replay baking kept at 100 ms** — the decoder's construction default moved to 33 ms in v1.3.0; `replaybuild.pasDuRejeu` sets the step the page reads, so replays stay baked at 10 frames per second in this release. `replay-equiv` plays at the decoder's reference step (33 ms). The move to 30 frames per second is PR #207, open.
- **[TO CONFIRM, #207 open]** Baking at 30 frames per second (`internal/replaybuild/cadence.go`, 33 ms step), the step enters the freshness criterion ("stale cadence" server side and admin badge, nothing is re-baked on its own), decoder v1.3.6 (`SchemaVersion` 115: match facts independent of the grid step in v1.3.5, bit-by-bit passes behind the frame walk and vehicle positions and occupant aim changed in v1.3.6), route `GET /assets/{title_id}/kill-sources` for the icons, icon and killed count on the "destroyed" line. The PR waits for the verdict on five "not established" lines of the corpus gate and for the acceptance of two FALSE results attributed to the grid.

### Security
- **Dependencies** — `undici` 8.12.4 (via `jsdom`), `vitest` and `@vitest/mocker` 4.1.12 (`npm update`, ranges unchanged) and `minimatch` 11 (three targeted overrides: `eslint`, `@eslint/config-array`, `@typescript-eslint/typescript-estree`); `npm audit` at 0 vulnerabilities. These are development dependencies: nothing of them ships in the application. The `glib` alert (Rust, 0.18.5, pulled by `gtk` for Tauri's Linux targets) is not compiled on Windows (ADR 0045) and is only fixed by moving Tauri to gtk-rs 0.20: close it as not applicable or follow it with the next Tauri upgrade.

### Removed
- **Win probability** — capability, fields of five OpenAPI schemas and of `SkillSnapshot`, `platform/duckdb` readers, Synergies and Progression columns; the calculation and the stored column do not change.
- **Commands with no user** and the callerless code they held: 96 commands removed, 17 restored after comparing unreachable code (the only callers of 63 functions), four more removed with their code (world leaderboard, Halo 5 read trial); `docs/COMMANDS.md` documents the maintenance commands kept.
- Coach pattern alerts that were never fed (`PatternReport`).

### Fixed
- **Replay corpus gate** — `replay-build` recompiles without cgo, the log level is read in `logging`.
- **Decoder (v1.3.1)** — record creation for builds HI_1_11_0 to HI_1_13_0 (the game skips the mask under a gate at 0 and writes no gate for ten archetypes with an empty creation mask: 35 packets lost since v1.0.1 are read again) and view A loads (genres 12, 14, 106) limited to the builds where they are proven. On HI_1_9_0 and HI_1_10_0 their real layout remains unestablished.
- **Corpus gate** — two bounded requalifications (counters moved to the frame walk, the new destructions block); the truth-bench false positive on the smoke-bubble fallback of one film is accepted as investigated.
- **Comparison** — the privacy banner and the "partial data" warning, which the server never emitted, are removed from the page.
- **Decoder** — validation of a map with an invalid region, index bits of a local module with more than two regions, a region index with no row reads zero bits as the game does.
````
