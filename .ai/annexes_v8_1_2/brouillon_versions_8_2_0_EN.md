# Release notes 8.2.0, EN DRAFT (2026-11-12)

Base: `origin/release/v8.2.0` at `20b8f386a6`. Nothing is placed in `docs/`: the two `TestReleaseNotesDocs_*`
tests require the newest notes block to carry the number of the first versioned changelog entry, so the `v8.2`
block and the `[8.2.0]` entry go in TOGETHER at closure, with the version bump (`Cargo.toml`, `Cargo.lock`,
`tauri.conf.json`). The French counterpart is `brouillon_versions_8_2_0_FR.md`. British English throughout.

Lines marked **[TO CONFIRM]** depend on a pull request still open (#203, the 30 frames-per-second cadence):
remove or finalise them before closure; none may ship as is.

## 1. Block for `docs/RELEASE_NOTES.md`

Paste at the top of the file, under `## What's new`, before the `**v8.1.2 — ...**` block. The first section is
reused as is for the updater notes and the release page.

````markdown
**v8.2 — A fuller replay, Lead share, new match charts and compressed replays**

**The replay**
- **A feed you can filter** — the feed title opens a menu: All, Kills, Objectives, Weapons and power-ups, Vehicles, Medals, Assists. Objective actions appear as banners, weapons picked up from the ground join the feed, and a filter with no line in the match says so instead of staying empty. Your choice is remembered
- **Click a kill** — playback pauses at that moment, and the killer and the victim are highlighted on the map with their names and the weapon
- **The zone of the kill** — each kill in the feed shows the map zone the killer stood in, when the map names one
- **A "View" button** — in the corner of the map, four layer presets (Full, Combat, Movement, Objective) and a panel gathering effects, heat, dot colour and sound. The Settings button leaves the playback bar
- **A player card in a panel** — a click on a card replaces the feed with their end-of-match totals (accuracy, damage, score breakdown, objective, weapons), with Follow, Solo and Hide. Esc closes the panel
- **The film read more faithfully** — positions are read at the right widths on maps with several regions, and object types and the Spartan's abilities are recognised more rigorously: a regenerated replay gains in precision
- **Fixed turrets, bots and places of death** — a fixed turret is recognised as such, stray tracks are no longer shown, the bots of a film without a kill feed keep their names, and kills that had no location get one
- **Game icons for grenades and abilities** — the replay thumbnails come from the game's own interface sheets, in line art for grenades
- **[TO CONFIRM, #203] Destroyed vehicles** — the feed and its "Vehicles" filter list vehicle destructions
- **[TO CONFIRM, cadence] Replays at 30 frames per second** — movement is smoother, position, aim and shield included

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
````

## 2. Block for `docs/CHANGELOG.md`

Replace `## [Unreleased]` by an empty `## [Unreleased]`, then place this below it, the date being the
publication date. The "Game icons" entry takes over the one already under `[Unreleased]`. The file is read at
runtime by the changelog page: leave nothing **[TO CONFIRM]** in it at merge.

````markdown
## [8.2.0] - YYYY-MM-DD

> The replay and Lead share on the player side; on the code side, LevelUp consumes the four bricks published as v1.0.1 and `libs/` leaves the repository.

### Added
- **Lead share (`film.lead_share`)** — persisted per-match derived value: share of the played time spent alone in the lead of the score, and number of lead changes. Migration `shared_match_lead_share_v1` (append-only table `match_lead_share`, view `match_lead_share_latest`), `persist.LeadSharePersister`, pure computation `internal/analysis/leadshare` with a case set shared by the Go and web tests, command `levelup backfill-lead-share` (`--dry-run` read-only, no migration). Fields served on the match view, Sessions, Time series and Squad; absent, never zero, when the match is not measured (few-points mode, score not covered, kick-off not measured).
- **Replay page** — filterable feed (All, Kills, Objectives, Weapons and power-ups, Vehicles, Medals, Assists; filter remembered in `replayPreferences`), click on a kill (pause and highlight of the killer and the victim), weapons picked up from the ground in the feed, "View" button and four layer presets (`viewPresetLogic`), player card in a panel with Follow, Solo and Hide, killer's zone in the feed (port of `NommerZone`, no new field and no re-bake).
- **Match view charts** — "Score gap" (`_scoreDelta.ts`) and "Numerical advantage" (`lib/replay/aliveBalanceLogic.ts`) in the Timeline tab, shared ECharts option `_divergentStepOption.ts`.
- **Compressed replays** — artefacts are stored as `{short8}.json.gz` (gzip level 6) behind a single gate, `internal/replayartifact` (form detected by the magic bytes, mixed collection read, the compressed one wins); `levelup replay compress [--dry-run]` (in-place, atomic, identical read-back before deletion, never automatic); on the desktop, the regeneration queue compresses at its end the raw files it could not rebuild and the update offer proposes compression alone (`raw`). Fallback without a hard link for exFAT and FAT32. Decompressed content unchanged.
- **Replay thresholds in milliseconds** — every threshold expressed in frames becomes a duration, converted with the document's step (`frameIntervalMs` on the web side, `framesOfMS` in the decoder, `HistoricalFrameIntervalMS` = 100); at 100 ms no value changes.
- **Film reading (decoder)** — positions read at the widths of the region the stream designates (per-region-index table in `map_quant_bounds.json`, `profile.MapQuantEntry.Regions`), entity type bound to its full identifier, Spartan ability (tag 3). Revisions: `grammar-2026-11-10.3`, `killsource-2026-11-10`, `profile-2026-11-10`, `SchemaVersion` 108.
- **Replay: fixed turret, bots, place of death** — fixed turret chassis recognised, stray tracks dropped, bots named on a film without a kill feed (roster read in `BOT_METADATA`), place of death read per body (`IsolationDecoderRev` raised).
- **Desktop** — `pick_folder` IPC command ("Browse" button, a single permission, limited to `127.0.0.1`) and capability `can_pick_folder` true for the `desktop` profile only; ADR 0045 amended. The old interface files are removed on update (`hooks.nsh`) and on uninstall (`uninstall_cleanup_engine.go`).
- **Web** — compact sidebar foot and "More" menu, sliding references pane with pinned mode, tooltips opened on keyboard focus only (`lib/dom/focusVisible.ts`), `--scrim` token in place of absolute blacks and whites (guard rail extended to `bg-black` and `text-white`).
- **Game icons for grenades and abilities** — thumbnails extracted from the game's interface atlases, at the same paths; the resource pack changes (a single download for the desktop application); the unused `static/abilities-assets` folder is removed.

### Changed
- **Published bricks** — LevelUp consumes `halo-infinite-files-reader`, `halo-infinite-map-images`, `halo-api` and `halo-infinite-film-decoder` at v1.0.1 from their dedicated repositories; `libs/`, `go.work` and `go.work.sum` are removed, module paths renamed (`github.com/JGtm/...`). `halo-api` carries `retract [v0.1.0, v1.0.0]` (the ranking capture in its `v1.0.0` was not pseudonymised). Single resolver `internal/briques` (`go list -m -f {{.Dir}}`), script `scripts/briques-workspace.sh` to develop a brick in a clone (`levelup-briques.work`, never a `go.work`), `replay-equiv -update` accepted on a clone outside the repository and refused on the module cache. Docker, CI, Makefile, lefthook, Dependabot and baselines rewritten; the bricks' tests run by module path from `apps/go-api`, their lint lives in their own repository. File-size and function-length ratchets reduced to LevelUp's own tree.
- **Environment switches removed** — `LEVELUP_USE_SHARED_PROVIDER`, `LEVELUP_POSTSYNC_BURST`, `LEVELUP_SYNC_V2_DRYRUN`, `LEVELUP_PERSIST_NO_FETCH_CACHE`, `LEVELUP_LUSR_V2_MODE_COUPLING`, `LEVELUP_LUSR_V2_SQUAD_OFFSET`, `LEVELUP_WATCHER_BROADCAST`, `PRESTIGE_ENABLED` and `LEVELUP_LOG_JSON`; the image sets `LEVELUP_LOG_FORMAT=json`. Six variables are read once by `internal/config`; `os.Getenv` and `os.LookupEnv` are forbidden outside `internal/config` and `cmd` (`archlint` guard rail). Two compatibility branches remain, dated 2026-12-31. Automatic repair of rating gaps is removed.
- **Net FDA** — Solo versus Squad reads the match's native FDA (fallback on the canonical calculation), the losing-streak alert is judged on a drop in points (`TiltFDADropPoints` 2.5, high severity 4.5), the Help formula and `Assistances/2` in both `fields.toml` are corrected.
- **LUSR** — solver fallback without the kill and death counts when it does not converge (`LUSRRev` = `lusr-2026-11-09`): full replay at the first start (ADR 0047), non-converging matches receive a score.
- **Behaviour alerts localised** — eleven sentences move to the `internal/i18n/messages/patterns.toml` catalogue.
- **Writes** — awards and the `psa_checked_at` marker written after the alias, match intensities not written are logged and counted, unreadable rows are logged, the `duckdb.OpenReadWrite` surface frozen by a dated list of 43 files, exception lists dated from git.
- **Architecture, step 4 (4.1 to 4.3)** — `worldenrich` no longer imports `service`, sync jobs go through a port and a service, progression, campaign and profile go through ports, the databases' memory limit goes through a port; guard rail on storage in handlers. **[TO CONFIRM]**: 4.4 to 4.8 (SQL of `api/wire` to `platform/duckdb`, moved handlers, `port/` interfaces, import guard rails) are still open.
- **Link previews** — language of the link (`?lang=`), then the crawler's, then the fallback language; the "Copy link" button carries the sharer's language.
- **Licences** — origin of the bricks' data written down, `coupe.png` removed, `sbsp.xml` credited to Infinite Runtime Tag Viewer, `halo-api` ranking capture pseudonymised, real identities replaced in the `apps/` fixtures (registry `IDENTITES_APPS.tsv`, guard rail extended).
- **[TO CONFIRM, #203]** Decoder v1.2.3: vehicle destructions (`vehicles[].destruction`), view A read through the genre table, `SchemaVersion` 109.
- **[TO CONFIRM, cadence]** Replays at 30 frames per second, `SchemaVersion` 110, usage summary revision `us8`.

### Removed
- **Win probability** — capability, fields of five OpenAPI schemas and of `SkillSnapshot`, `platform/duckdb` readers, Synergies and Progression columns; the calculation and the stored column do not change.
- **Commands with no user** and the callerless code they held: 96 commands removed, 17 restored after comparing unreachable code (the only callers of 63 functions), four more removed with their code (world leaderboard, Halo 5 read trial); `docs/COMMANDS.md` documents the maintenance commands kept.
- Coach pattern alerts that were never fed (`PatternReport`).

### Fixed
- **Replay corpus gate** — `replay-build` recompiles without cgo, the log level is read in `logging`.
- **Decoder** — validation of a map with an invalid region, index bits of a local module with more than two regions, a region index with no row reads zero bits as the game does.
````
