# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

> French version: [FR/CHANGELOG.md](FR/CHANGELOG.md)

## [6.0.0] - 2026-03-15

> ⚠️ **Weapon extraction still in beta** — attribution accuracy not guaranteed in all cases (estimated coverage 70–100 % depending on matches); weapon catalog in progress.

### Added

- **ID resolution layer** — three SQL views in `shared_matches.duckdb` replacing all ad-hoc 5-source gamertag cascades:
  - `v_gamertag_lookup` — FULL OUTER JOIN `xuid_aliases` + `match_participants` with deduplication and priority
  - `v_match_full` — `match_registry` enriched with i18n metadata (maps, playlists, game variants)
  - `v_killer_victim_full` — killer/victim pairs with resolved gamertags
  - `ensure_metadata_attached(conn)` helper added to `src/utils/db.py`

- **`weapon_labels` table** in `metadata.duckdb` (`src/data/migration/steps/add_weapon_labels.py`)
  - Schema: `weapon_labels(weapon_id UBIGINT PK, name_en VARCHAR, name_fr VARCHAR)`
  - DB-first resolution: `_resolve_weapon_from_db()` with `@lru_cache` + Python dict fallback
  - Automatic migration `add_weapon_labels` (`target_db="metadata"`) registered in the migration system
  - `src/ui/i18n/weapons.py` cleaned up: `get_all_weapon_ids`, `get_weapon_ids_by_faction`, `translate_weapon_name` removed

- **`src/auth/` package** — new auth layer replacing all manual Azure/env configuration:
  - `LEVELUP_CLIENT_ID` hardcoded — no Azure portal setup required for end users
  - `_msal.py`: `SerializableTokenCache` persisted in DuckDB (`sync_meta`) via MSAL
  - `provider.py`: single entry point — process cache (4 h TTL), MSAL silent refresh, `AuthRequiredError`, `start/complete_device_flow`
  - `_halo_exchange.py`: stateless `access_token → (spartan, clearance)` exchange via spnkr.auth

- **Launcher — SSO auto-detection** (`launcher.py`)
  - Gamertag automatically resolved from Microsoft login via Halo API (no manual entry required)
  - New first-launch flow: Device Code → DuckDB MSAL → sync → Streamlit (zero manual configuration)
  - Recovery menu simplified to Device Code Flow only

- **`resolve_medal_name` helper** (`src/analysis/`) — medal name resolution from `metadata.duckdb`, no hardcoded dicts

- **Last Match — previous/next navigation** — `◀ Previous` / `Next ▶` buttons to navigate between all filtered matches without reloading data

- **`populate_metadata_from_discovery.py` rewritten** for v5.1+
  - Reads `match_registry` from `shared_matches.duckdb` (replaces deprecated `match_stats`)
  - Extended DDL with i18n columns (`name_en`, `name_fr`, `mode_name`, `playlist_canonical_*`)
  - Logic extracted into `scripts/_metadata_db.py` (DDL + i18n enrichment)

- **`WeaponKillsMixin.load_grenade_melee_kills(xuid, match_ids)`** — new repo method querying
  `shared.match_participants` for grenade/melee totals. Replaces all direct `_get_connection()`
  calls in UI code (`_timeseries_weapons.py`, `match_view_weapon_kills.py`, `teammates_weapons.py`).

- **Custom medal: Avenger** — detects revenge kills (you kill the opponent who last killed you) via `killer_victim_pairs`
  - Custom ID `9 000 000 001` (beyond official Halo medal range)
  - Global backfill via correlated subquery on `killer_victim_pairs`: for each kill, checks if the victim is the killer from the player's previous death
  - CLI: `python scripts/backfill_data.py --avenger` (or `--force-avenger` for full recompute)
  - Names (`medals_fr/en.json`) and descriptions (`medals_descriptions_fr/en.json`) in static JSON files
  - `resolve_medal_description()` enhanced with JSON fallback when `metadata.duckdb` has no `medals` table
  - 18 tests (12 backfill + 6 description)

- **Top Gun label** 🔫 — badge on the Impact timeline for the first player on your team to reach 10 kills in a match
  - Constant `TOP_GUN_KILL_THRESHOLD = 10`; `_find_top_gun_event()` scans `highlight_events` chronologically
  - Integrated into the existing impact events pipeline (no UI caller changes needed)
  - Bilingual labels: "As de la gâchette" (FR) / "Top Gun" (EN)

### Changed

- **Gamertag resolver** (`src/data/sync/_gamertag_resolver.py`) — 5-source cascade replaced by a single `v_gamertag_lookup` JOIN; `load_match_player_gamertags()` reduced from 4 sequential queries to 1
- **`match_registry` consumers** migrated to `v_match_full` (asset loader, career encounters, etc.)
- **`killer_victim_repo`** and `career_encounters_data` migrate to `v_killer_victim_full`
- **i18n DuckDB migration** — `modes_fr/en.json` migrated to `metadata.duckdb`; playlist and game_variant JSON dicts removed from source code
- **`get_tokens_from_env()`** (sync) — deprecated wrapper delegating to `src.auth`; internal callers updated
- **Weapon parser — global correlation** — fire_event match rate corrected from 15 % to 95 % after fixing `b2_dispatch` routing; compact COMPLETE logs with sentinel / no_weapon distinction and `b2_dispatch` drop rate exposed
- **Backfill `--weapons --all`** — match_ids deduplicated across all players so each film is downloaded only once
- **`v_weapon_kills` view enforced app-wide** — all read queries now use `shared.v_weapon_kills`
  (exposes `effective_weapon_id = COALESCE(reconciled_as, weapon_id)`) instead of the raw
  `weapon_kills` table. Affected: `match_view_weapon_kills.py`, `citations/_data_loader.py`,
  `_roster_loader.py` (scoreboard top-weapon subquery).
- **`load_weapon_kills_for_player` replaces `load_weapon_kills_for_match`** in
  `match_view_weapon_kills.py` — SQL-level filter by `xuid` instead of Python post-filter.
- **`v_gamertag_lookup` enforced app-wide** — all remaining `LEFT JOIN xuid_aliases` patterns
  replaced; guards `_has_shared_view` / `_has_shared_table` removed (view guaranteed present
  in v6). Affected: `_encounter_loader.py`, `_career_encounters_repo.py`, `_roster_loader.py`,
  `_events_repo.py`, `_discord_queries.py`.
- **`load_match_roster()` simplified** — two redundant Python gamertag enrichment passes
  (inline `xuid_aliases` + `v_gamertag_lookup` queries with dead guards) removed; enrichment
  delegated exclusively to `resolve_gamertags_batch()`. ~45 lines removed.

### Removed

- **`highlight_events.gamertag` column** — migration `drop_highlight_events_gamertag`; gamertag resolved via `v_gamertag_lookup` instead
- **`resolve_xuid_from_input` wrapper** — dead code removed
- **`get_outcome_name_fr`** and `_refdata_outcomes` — replaced by metadata.duckdb lookup
- **14 Azure/OAuth functions** in `launcher.py` (−652 lines net): Azure wizard, `has_client_id`, `config-az`/`paste-id` recovery options, environment variable `SPNKR_AZURE_CLIENT_ID` no longer required
- **`_has_gamertag_column()`** helper in `_weapon_kills_repo.py` — dead code since `drop_highlight_events_gamertag`
- **Dead guards** `_has_shared_view("v_gamertag_lookup")` in `teammates_impact.py` and `_events_repo.py` — `else` branches returning `NULL AS gamertag` removed
- **`_append_grenade_melee()`** helper in `teammates_weapons.py` — replaced by `load_grenade_melee_kills()`

### Tests

- `tests/test_resolution_views.py` — 11 tests: view priority, alias/participant fallback, NULL filter, deduplication, EN columns always populated, FR columns NULL without metadata, idempotence, gamertag resolution for killer/victim
- `tests/test_global_correlation.py` — 19 tests: **100 % coverage** on `_global_correlation.py` (38/38 stmts, 12/12 branches)
- `_parser_logging.py` — **100 % coverage** (57/57 stmts, 10/10 branches)
- **4 719 tests total, 0 failures**

---

## [5.7.0] - 2026-03-13

### Added

- **Traductions FR des rangs Halo** (`src/ui/i18n/ranks.py`)
  - 17 rangs de carrière (Recruit→Recrue, General→Général, Hero→Héros…) + 6 tiers CSR (Silver→Argent, Gold→Or…)
  - Helper `translate_rank()` avec fallback sur le nom anglais original
  - Intégré dans le script de migration metadata (`migrate_metadata_to_duckdb.py`)

- **Launchers bilingues FR/EN** (`LevelUp.sh`, `LevelUp.bat`)
  - Détection automatique de la langue système (POSIX `LC_ALL`/`LANG`, Windows Registry `LocaleName`)
  - ~30 messages localisés dans chaque launcher (premier lancement, erreurs, winget, etc.)

### Changed

- **Pandas→Polars** : suppression de 7 appels `.to_pandas()` dans les modules UI/viz
  - `participation_charts.py` (pie, bars, stacked) : Polars natif bout en bout
  - `participation_charts_extra.py` (sunburst) : `.to_pandas()` conservé uniquement à la frontière `px.sunburst`
  - `objective_analysis.py` (3 tables assist/awards) : `st.dataframe` Polars natif
  - `duckdb_analytics.py` (KDA trend) : `st.line_chart` avec `x=`/`y=` Polars natif

- **CSS-only map thumbnails** : remplacement du script JS sandboxé (non fonctionnel) par un système hover CSS pur
  - Suppression de `_MAP_TOOLTIP_SCRIPT` dans `styles.py` (38 lignes JS)
  - Classes `.map-hover` + `.map-popup` dans `static/styles.css`
  - `match_table_html.py` et `win_loss_table_style.py` mis à jour
  - `_build_map_url_index()` amélioré : `lru_cache(maxsize=None)`, normalisation Unicode

### Removed

- Guard `was_pandas` dans `_performance_relative.py` : `compute_performance_series()` accepte désormais uniquement `pl.DataFrame`

### Tests

- `TestHighlightEventsSequenceIdempotent` ajouté dans `test_migrations.py` (couverture A.4)
- 45/45 tests migrations passants

## [5.6.0-beta] - 2026-03-10

> ⚠️ **Beta** — weapon attribution accuracy not yet guaranteed in all cases (estimated coverage 70–100 % depending on matches); weapon catalog in progress.

### Added

- **Weapon extraction from SPNKr films** (`src/analysis/weapon_parser.py`, `src/data/services/weapon_extraction_service.py`)
  - Parses `REPLICATION_DATA` chunks from match films to identify the weapon used for each POV kill (player_index=1, universal invariant)
  - kill→last fire event correlation within a 2 000 ms window; melee/grenade/vehicle kills detected via medals (`MELEE_API_ID=1`, `GRENADE_API_ID=0`)
  - POV coverage: ~87.5 % of kills
  - Hexagonal architecture: `weapon_parser.py` (pure domain, zero IO), extended `HaloAPIPort`, `WeaponExtractionService` (orchestration), enriched `WeaponKillsMixin` (upsert, backfill bit, queries)
  - Table `weapon_kills (match_id, xuid, weapon_id, kills)` in `shared_matches.duckdb` (PRIMARY KEY `match_id, xuid, weapon_id`) + index `idx_wk_match_xuid`
  - Migration `add_weapon_kills` (`target_db="shared"`) registered in the automatic migrations system
  - Local cache of downloaded chunks in `data/investigation/chunks/<match_id>/`

- **Sync integration** (`src/data/sync/_engine_weapon_kills.py`)
  - Automatic weapon extraction on new matches via `WeaponKillsEngineMixin`
  - Controlled by `SyncOptions.with_weapons`; configurable via `spnkr_refresh_backfill_weapons` in `app_settings.json` and the Settings page checkbox

- **Backfill weapon_kills** (`scripts/backfill_data.py --weapons`)
  - `--weapons [--force-weapons] [--gamertag <GT>]` via the unified backfill CLI
  - Bit `MatchBits.WEAPON_KILLS` (1 << 21) set on `match_registry.backfill_completed` after processing

- **Weapon kills section in Match View** (`src/ui/pages/match_view_weapon_kills.py`)
  - Summary tab: kills-by-weapon table for the POV player

- **Teammates Weapons tab** (`src/ui/pages/teammates_weapons.py`)
  - Per-weapon kill breakdown for all teammates on shared matches

- **MSAL Device Code Flow** (`src/utils/msal_device_flow.py`, `src/ui/xbox_oauth_ui.py`)
  - Replaces the OAuth redirect flow: user enters a short code on xbox.com/activate (no redirect URI or client secret required)
  - Pure MSAL wrapper: `initiate_device_flow()`, `acquire_token_blocking()`, `DeviceCodeResult`, `DeviceFlowError`
  - Streamlit UI component: start / polling / reset (integrated in Setup Wizard step 2 and Settings)
  - `setup_wizard_xbox.py` extracted from `setup_wizard.py` to stay within the 500-line module limit
  - `--device-code` flag added to `scripts/spnkr_get_refresh_token.py` for CLI token acquisition
  - `msal>=1.28.0` added as optional dependency
  - Azure configuration simplified: only `client_id` required (no `client_secret`, no `redirect_uri`)

- **Friends Impact matrix** (`src/visualization/friends_impact_heatmap.py`)
  - Vertical separators (Plotly shapes) between each match column for improved readability
  - Renamed from "Impact Heatmap" to "Impact Matrix" (FR i18n update)

- **Documentation** (`docs/CONFIGURATION.md`)
  - Azure guide simplified for Device Code Flow — `client_secret` and `redirect_uri` steps removed

### Fixed

- **Discord notifier** (`src/utils/discord_notifier.py`) — Lightweight embed restored when all players are idle (was accidentally suppressed in a previous commit)

### Tests

- **51 unit tests** (`tests/test_weapon_parser.py`, `tests/test_weapon_service.py`): constants, `find_frame_positions`, `build_frame_estimator`, `correlate_kills_to_weapons`, `count_kills_by_api_weapon`, `WeaponExtractionService` mocks (no kills, no film, dry-run, upsert, caching, errors), `WeaponKillsMixin` repo (upsert/conflict, bit marking, missing matches, gamertag lookup)
- **28 tests added/rewritten** for Device Code Flow (`tests/test_msal_device_flow.py`, `tests/test_auth.py`, `tests/test_xbox_oauth.py`, `tests/test_setup_wizard_logic.py`, `tests/test_setup_wizard_page.py`): `authorization_pending`, `slow_down`, DC Flow no-secret pattern, `get_spartan_tokens`, `resolve_player_identity`, `complete_device_code_flow`
- **4 041 tests total, 0 failures**

### Removed

- **Xbox OAuth redirect flow** — `build_xbox_auth_url()`, `generate_oauth_state()`, `exchange_code_for_refresh_token()`, `run_xbox_oauth_callback()`, `_handle_xbox_oauth_callback()` removed; replaced by Device Code Flow
- **`client_secret` / `redirect_uri`** — No longer required for token acquisition; `SPNKR_AZURE_CLIENT_SECRET` and `SPNKR_AZURE_REDIRECT_URI` environment variables are deprecated
- **`scripts/backfill/backfill_weapon_kills.py`** — Standalone backfill script removed (violated CLAUDE.md: all backfill must go through `scripts/backfill_data.py`)

---
## [5.5.0] - 2026-03-07

### Added

- **Session Comparison page revamped** (`src/ui/pages/session_compare.py` and related modules)
  - Outcomes distribution: W/L/T/DNF donut charts per session with win-rate in center
  - Match highlights: best/worst match per session (F/D ratio, mode name)
  - F/D + accuracy progression: K/D curve renamed F/D (FR), accuracy on secondary Y-axis (dashed), avg lifespan in hover
  - Modes breakdown: grouped horizontal bar chart of modes played per session
  - Map stats table: wins/losses per map for both sessions
  - Cumulative net score: per-match performance score coloring (green ≥70 / orange ≥45 / red <45) + LUSR or CSR overlay on secondary Y-axis (auto-detected from `match_skill_rank`)
  - Participation profile: replaced opaque stacked radar with grouped horizontal bars; thresholds scaled by number of matches

- **Setup Wizard — Guided initial configuration** (`src/ui/pages/setup_wizard.py` + `setup_wizard_logic.py`)
  - Two flows: **Xbox Express** (recommended, 2 steps) and **Azure manual** (advanced, 3 steps)
  - Custom CSS cards with icons, animated progress bar, numbered steps
  - Logic separated from UI (`SetupStatus`, `validate_azure_credentials()`, `validate_gamertag()`, `create_player_profile()`, `save_azure_credentials()`)
  - Guard in `main()`: the wizard displays automatically when credentials or player are missing
  - FR/EN i18n (~49 keys) in `src/ui/i18n/setup.py`

- **Xbox OAuth — One-click Xbox login** (`src/ui/xbox_oauth.py` + `xbox_oauth_ui.py`)
  - Full flow: Microsoft URL → callback `?code=XXX&state=YYY` → code exchange → refresh_token → spartan/clearance tokens → gamertag+XUID resolution → automatic provisioning
  - `xbox_oauth.py` (436L): pure OAuth logic without Streamlit dependency
  - `xbox_oauth_ui.py` (163L): Streamlit component integrated in Settings (login button, status, logout)
  - CSRF protection with random `state` validated on callback return
  - FR/EN i18n in `src/ui/i18n/pages/xbox.py`

- **Player Provisioning** (`src/app/player_provisioning.py`)
  - `provision_player()`: creates `data/players/{gamertag}/stats.duckdb` + `sync_meta` table + registers in `db_profiles.json` — idempotent

- **Auth Status** (`src/utils/auth.py`)
  - `AuthStatus` dataclass + `get_auth_status()`, `check_credentials()`, `write_env_local()` (writes/updates `.env.local` while preserving comments)

- **macOS / Linux compatibility** — `LevelUp.sh` (new): first-launch launcher equivalent to `LevelUp.bat` for macOS/Linux, written in POSIX sh (no bashisms — compatible with macOS bash 3.2, dash, zsh). Detects Python 3.10+ via versioned binaries (`python3.12` → Homebrew), Homebrew Intel/Apple Silicon paths (`/opt/homebrew`, `/usr/local`), then generic. Distribution-targeted help messages. `run.sh` fixed to detect `.venv/bin/python` (macOS/Linux) or `.venv/Scripts/python.exe` (Windows Git Bash). `launcher.py`: `_find_system_python()` enriched with versioned candidates and Homebrew paths; `_cmd_doctor()` now uses `_preferred_python_executable()` cross-platform.

- **`launcher.py setup`** — Interactive installation command: detects Python (py launcher → PATH → standard locations → installation via winget), creates `.venv`, installs dependencies (`pip install -e ".[spnkr]"`). Supports `--update` to update an existing environment.

- **`launcher.py doctor`** — Full environment diagnostic: OS, Python, venv, critical vs expected package versions, number of configured players, presence of `metadata.duckdb`

- **Portable packaging** (`packaging/build_release.py`)
  - Generates a self-contained zip `LevelUp-v{version}-win64-portable.zip` containing Python Embeddable 3.12 (~15 MB) + the full project
  - First launch: automatic dependency installation via pip

- **GitHub Actions Release** (`.github/workflows/release.yml`)
  - Triggered on push of tag `v*.*.*`
  - Portable zip build + automatic publication as a GitHub Release

- **Portable `%APPDATA%` mode** (`src/utils/paths.py`, `auth.py`, `env.py`)
  - Data stored in `%APPDATA%/LevelUp/` (Windows) or `$XDG_DATA_HOME/levelup/` (Linux) when no `.venv` at the root
  - Developer mode: `./data/` if `.venv` exists
  - Override possible via `LEVELUP_DATA` environment variable
  - `.env.local` looked up in `DATA_DIR` first, then at the repo root

- **Token fallback DB** (`src/ui/profile_api_tokens.py`)
  - Fallback 3: reads the refresh_token from the player DB `sync_meta` if absent from environment variables

- **Documentation**
  - `docs/CONFIGURATION.md`: complete rewrite with TOC, step-by-step Azure guide with 11 annotated screenshots, sections Player Profiles, Environment Variables, App Settings, Security, Troubleshooting
  - `docs/FR/CONFIGURATION.md`: FR version updated
  - `docs/SYNC_GUIDE.md`: rewrite with v5.1 sync architecture, ASCII diagram, detailed commands
  - `docs/FR/SYNC_GUIDE.md`: updated

- **Automatic schema migrations** (`src/data/migration/`) — versioned runner applied automatically at startup (`launcher.py → _run_migrations()`). Each DB (`player`, `shared`, `shared_pve`) tracks migrations in a `schema_migrations` table. 11 initial migrations registered. To add a schema change: create an idempotent `ensure_xxx` function in `src/data/sync/migrations.py`, create the corresponding step in `src/data/migration/steps/` and import it in `steps/__init__.py`.

### Fixed

- **CSRF** (`streamlit_app.py`) — Fixes comparison `_xbox_state != _xbox_state` (self-comparison, always False) → `_xbox_state != _expected_state`
- **`_repo_root` undefined** (`src/ui/profile_api_tokens.py`) — `_repo_root()` was never imported → replaced with `REPO_ROOT` from `src.utils.paths`
- **Expanded DuckDB retry** (`src/data/sync/_engine_connections.py`) — `except duckdb.IOException` → `except duckdb.Error` + retry delay `0.15s → 0.5s`
- **GC sync mode** (`src/ui/_sync_duckdb_ops.py`) — `gc.collect()` + `time.sleep(0.3)` to release DuckDB file handles on Windows
- **OAuth consumed guard** (`streamlit_app.py`) — `_xbox_oauth_consumed` flag to prevent double-processing of the callback on Streamlit rerun
- **Test isolation webhook** (`tests/test_monitor_uptime.py`) — Patches `get_secret` instead of mutating `os.environ` to avoid reloading `.env.local`
- **Deprecated Streamlit API** (`src/ui/pages/setup_wizard.py`) — Replaces the three `use_container_width=True` occurrences with `width="stretch"`: Xbox Express button, Azure manual button, and OAuth `st.link_button`.
- **Missing UI smoke test** (`src/ui/pages/setup_smoke_test.py`) — UI module recreated: 3 phases with progress bars, verification table, and continue/retry buttons.
- **Incorrect `SPNKrAPIClient` test patch** (`tests/test_player_tokens.py`) — Mock target corrected to `src.data.sync._career.create_api_client` to match the API abstraction.

### Tests

- **75 tests added** (1,482 lines) covering all new modules:
  - `test_auth.py` (13 tests): `AuthStatus`, `get_auth_status()`, `write_env_local()`
  - `test_setup_wizard_logic.py` (20+ tests): `SetupStatus`, validation, profile creation, edge cases
  - `test_xbox_oauth.py` (18 tests): OAuth URL, code exchange, store/load token, provisioning
  - `test_xbox_oauth_callback_e2e.py` (9 tests): full code→player flow, errors, CSRF, token cycle
  - `test_setup_wizard_page.py` (15 tests): mocked UI (MockStreamlit), Xbox/Azure modes, progression; `width="stretch"` assertions on widgets
- **3,831 tests, 0 failures**

### Architecture

- **API Abstraction — Ports & Adapters**: decouples the codebase from the SPNKr library to make a future API backend switch easier
  - `api_port.py`: `HaloAPIPort` Protocol — structural contract (`runtime_checkable`) defining the methods every Halo API client must implement
  - `api_factory.py`: `create_api_client(backend="spnkr")` factory — centralized instantiation, extensible to other backends
  - `_auth.py`: authentication facade — UI modules call `refresh_halo_tokens()` without importing SPNKr directly
  - Consumer migration: `engine.py`, `orchestrator.py`, `strategies.py`, `_career.py`, `populate_metadata_from_discovery.py`, `profile_api_tokens.py`, `player_assets.py`, `xbox_oauth.py` — all now use the factory or the auth facade
  - 14 dedicated tests (`test_api_abstraction.py`): Protocol compliance, factory behavior, auth facade, and verification that migrated UI modules no longer import SPNKr

### Removed

- **`scripts/_archive/`** — 89 dead code files deleted (legacy weapon analysis scripts, diagnostics, i18n patches, obsolete utilities)
- **`requirements.txt`** — Removed, replaced by `pyproject.toml` (single source of truth for dependencies)
- **`setup.bat`** — Replaced by `LevelUp.bat` (improved Python detection, installation via winget, use of `pip install -e .`)
- **`scripts/install_dependencies.py`** — MSYS2 SSL workaround, used `requirements.txt`
- **`scripts/setup_env.ps1`**, **`scripts/setup_env.sh`**, **`scripts/activate_env.sh`** — Replaced by `launcher.py setup`
- **`tests/test_spnkr_refactoring.py`** — Tests for deleted archived code

### Chore

- Root cleanup: `ACKNOWLEDGMENTS.md`, `CHANGELOG.md`, and `CONTRIBUTING.md` moved to `docs/`
- Scripts moved: `activate_env.sh`, `run_monitor_hidden.vbs` → `scripts/`
- `LevelUp.bat` replaces `setup.bat` as the Windows entry point
- `Dockerfile` and `e2e-browser-optional.yml` updated to use `pyproject.toml` instead of `requirements.txt`
- `run.sh` now redirects to `launcher.py setup` instead of `activate_env.sh`

### Additional updates (8 March 2026)

- **XP & Hero rank multi-player comparison** — Career page now overlays XP curves and Hero projections for every player with a refresh token:
  - Real XP curve (lines + markers, distinct colour per player)
  - Pre-sync estimated XP curve (dotted, same colour) — linear interpolation over matches played before the first sync
  - "At this pace" → Hero projection (dashed) and optimistic projection (challenges + boost ×2, dash-dot)
  - All secondary curves hidden by default — click the legend to show them
  - **Variable precision** depending on available data: real XP delta between snapshots when enough syncs exist, otherwise falls back to a global average rate (total XP / days since earliest known match, or since Career Rank launch on 20 June 2023). Precision improves automatically with each new sync.

### Additional updates (7 March 2026)

- **Timezone selector** — Choose the display timezone directly in Settings (Europe/Paris by default, ~40 timezones available). Match timestamps adapt automatically throughout the app.
- **Improved Career page** — Better readability of the LUSR ranking section, smoother navigation.
- **Bot xuid migration** — Automatic correction of matches containing misidentified bots in the shared database.
- **Stability** — Fixes on adversary data loading, match queries, UI cache, and synchronisation. Improved reliability on Windows during concurrent DuckDB access.

## [5.4.0] - 2026-03-04

### Added

- **Explorer page — unified match search and navigation** (`src/ui/pages/explorer.py`)
  - Replaces the legacy "Match" page with a 6-module architecture (`explorer`, `explorer_results`, `explorer_enrich`, `explorer_data`, `explorer_logic`, `match_table_html`)
  - **Cascading filters**: date, squad (solo/squad), experience type (ranked/unranked/PvE), playlist, game mode, map
  - **Fuzzy gamertag search** with dynamic suggestions and XUID resolution
  - **OS-style HTML table** (`match_table_html.py`): KDA, kills, deaths, accuracy, score, MMR delta, performance, headshots, spree, average life; cross-page deep links
  - **Deep linking**: `?page=Explorer&gamertag=XXX` or `&match_id=XXX` for direct navigation
  - **Encounter badges**: rival, mentor, prey — computed from cross-player history
  - **Enrichment** (`explorer_enrich.py`): team score, MMR delta, performance, average lifetime, Waypoint URL
  - **Complete FR/EN i18n** (`src/ui/i18n/pages/explorer.py`)
  - **Structured logging**: info (deep links), warning (player not found, missing DB), error (DB exceptions with `exc_info`)
  - **40 unit tests** (`tests/test_explorer_logic.py`) covering logic, enrichment, data access, and HTML rendering

### Tests — previous skips fixed

The following tests were marked `@pytest.mark.skip` or `skipif(True)` and now run normally:

| File | Test(s) | Fix reason |
|------|---------|------------|
| `tests/test_rag.py` | `TestHaloKnowledgeBase` (3 tests) + `test_chunk_overlap` | Removed `skipif(True)` guards and false skip |
| `tests/test_season_archive.py` | `test_get_archive_info_with_archives` | Removed skip + `>= 0` assertion (tiny Parquet file) |
| `tests/test_i18n_refactoring.py` | `test_no_duplicate_keys_in_module[pages]` | Added package support (`pages/` folder instead of `pages.py`) |
| `tests/e2e/test_streamlit_browser_e2e.py` | `test_e2e_004_deeplink_match_query_params` | Regex `exception(?!nel)` excludes "exceptionnel" (French word) |
| `tests/test_cache_integrity.py` | 11 SQLite legacy tests | File **removed** (v3 dead code) |
| `tests/conftest.py` | all `e2e_browser` tests | Removed auto-skip guard + installed Chromium |

To rerun only these tests:

```bash
# RAG
python -m pytest tests/test_rag.py::TestHaloKnowledgeBase tests/test_rag.py::TestTextChunker::test_chunk_overlap -v

# Season archive
python -m pytest tests/test_season_archive.py::TestDuckDBRepositoryArchives::test_get_archive_info_with_archives -v

# i18n (package pages/)
python -m pytest tests/test_i18n_refactoring.py::TestNoInternalDuplicates -v

# E2E deeplink
python -m pytest tests/e2e/test_streamlit_browser_e2e.py::test_e2e_004_deeplink_match_query_params -v

# Full suite without integration
python -m pytest -q --ignore=tests/integration
```

### Added

- **Encounter history — section below the scoreboard** (`src/ui/pages/match_view_encounters.py`)
  - New HTML table displayed directly below the scoreboard on the Match View page
  - For each non-friend player in the match: encounter frequency, ally/enemy split, ally win rate, enemy win rate, cross K/D (from `killer_victim_pairs`), and last encounter date
  - Sorting: enemies first, then allies; within each group by `total_encounters DESC`
  - Compact grey row for first encounters (`total = 1`), full row with metrics beyond that
  - Automatic inline badges: **Hard to Kill** (deaths/kills > 2 and at least 3 deaths), **Ally+** (ally WR ≥ 65% over at least 2 matches), **Tough** (enemy WR ≤ 35% over at least 3 matches)
  - Color coding reuses scoreboard CSS classes (`os-sb-td--best`, `os-sb-td--worst`, amber)
  - Scope: all non-squad, non-friend players

- **Dedicated SQL loader** (`src/data/repositories/_encounter_loader.py`)
  - `load_encounter_stats(self_xuid, target_xuids, db_path)` — 3 CTEs on `shared_matches.duckdb` (`match_participants`, `killer_victim_pairs`, `match_registry`, `xuid_aliases`)
  - Automatically derives the `shared_matches.duckdb` path from `stats.duckdb`
  - Uses a direct `duckdb_read_only()` connection on shared (no ATTACH conflict)

- **Pure testable logic** (`src/ui/pages/match_view_encounters_logic.py`)
  - `EncounterStats` (Pydantic v2), `Badge` (dataclass), `ordinal_fr()`, `build_friends_set()`, `filter_encounter_xuids()`, `compute_encounter_badges()`
  - `build_friends_set`: dual source `player_match_enrichment.friends_xuids` → fallback `friends_defaults.json`
  - 28 unit tests in `tests/test_match_view_encounters.py` (without importing Streamlit)

- **i18n keys** (`src/ui/i18n/pages.py`): `mv_encounter_history`, `col_role`, `col_encounters`, `col_wr_ally`, `col_wr_enemy`, `col_kd_cross`, `col_last_seen`

### Technical

- `match_view.py`: calls `render_encounter_section()` after `render_match_scoreboard()` (+10 lines, zero business logic added to the file)
- SRP architecture preserved: 3 new files under 350 lines each, functions under 50 lines, UI and data logic separated

### Refactoring & Architecture (branch `refactor/cleanup-all`)

> **Massive 6-phase refactor** — 331 files modified, about 30,000 lines rewritten, 72 new submodules, 3,693 passing tests (including 79 dedicated tests added). No user-facing functional changes.

#### Phase 0-4: Infrastructure & initial splits

- **Split `transformers.py` (2,095L → package)** — `src/data/sync/transformers/` split into 7 submodules (`_helpers`, `_match`, `_skill`, `_events`, `_medals`, `_personal_scores`, `_pve`) + `__init__.py` re-exporting everything; no breaking change
- **Split `filters_render.py` (1,460L → 4 modules)** — extracted `_filters_period.py`, `_filters_session.py`, `_filters_cascade.py`; `filters_render.py` reduced to orchestration
- **Split `engine.py` (1,500L → 8 mixins)** — `_shared_writes.py`, `_performance.py`, `_skill_rating.py`, `_career.py`, `_aggregates.py`, `_tokens.py`, `_engine_connections.py`, `_engine_schema.py`
- **Split `duckdb_repo.py` (1,200L → 8 mixins)** — `_match_queries_helpers.py`, `_match_queries_polars.py`, `_archives_repo.py`, `_awards_repo.py`, `_diagnostic_repo.py`, `_events_repo.py`, `_medals_repo.py`, `_schema_introspection.py`
- **Split utility modules** — `media_indexer.py`, `api_client.py`, `batch_insert.py`, `discord_notifier.py`, `cache_loaders.py`, `radar_chart.py`, `teammates_views.py`, `sync.py`, `timeseries_combat.py`
- **`_SyncProtocol`** (`src/data/sync/_protocol.py`) — explicit `Protocol` contract for the 8 `DuckDBSyncEngine` mixins; removes 70+ `# type: ignore[attr-defined]`
- **`PageContext` + `MatchViewParams`** (`src/app/_page_context.py`) — real types instead of 5 `Any` fields in the `NamedTuple`
- **`SessionKeys` / `SK`** (`src/app/session_keys.py`) — 20+ centralized `st.session_state` keys, IDE completion, no more silent typos
- **`_sql_fragments.py`** (`src/data/query/_sql_fragments.py`) — single source of truth for `WIN_RATE_EXPR` (WIN+LOSS denominator, `NULLIF` division), `IS_WIN`, `IS_LOSS`; 7 duplicated occurrences removed from `analytics.py` and `trends.py`
- **v4→v5 technical debt removed**: `_PERF_SCORE_AVAILABLE` guard (always true), dead method `_ensure_performance_score_column()`, magic number `outcome == 4` → `Outcome.DID_NOT_FINISH`

#### Phase 5: Analysis & visualization splits

- **Split `performance_score.py` (950L → 3 modules)** — `_performance_relative.py` (relative match score), `_performance_session.py` (session score v1/v2, `ScoreComponent`); public facade unchanged
- **Split `antagonist_charts.py` (570L → 3 modules)** — `_antagonist_kv.py` (stacked bars, time series, heatmap), `_antagonist_duels.py` (duel history, nemesis summary, indicators); public facade unchanged
- **Split `rag.py` (750L → 4 modules)** — `_rag_models.py` (RAGConfig, Document, SearchResult), `_rag_github.py` (GitHubIndexer), `_rag_chunker.py` (TextChunker); public facade unchanged

#### Phase 6: UI & data splits

- **Split `refdata.py` (880L → 2 modules)** — `_refdata_personal_scores.py` (68-member `PersonalScoreNameId` enum, score/name/ID dictionaries); public facade unchanged
- **Split `_roster_loader.py` (520L → 2 mixins)** — `_gamertag_resolver.py` (`GamertagResolverMixin`, 5-source XUID→gamertag cascade); `_roster_loader.py` now inherits from the mixin
- **Split `cache_filters.py` (740L → 3 modules)** — `_cache_loading.py` (recent matches, pagination, match count), `_cache_sessions.py` (session DB computation); public facade unchanged
- **Split `filters_render.py`** — `_filters_apply.py` (`apply_filters` in 190L, empty-state diagnostic); public facade unchanged
- **Split `session_compare_charts.py` (480L → 2 modules)** — `_session_compare_history.py` (HTML history table); public facade unchanged

#### Quality & coverage

- **79 dedicated unit tests** — `test_submodules_phase5.py` (37 tests) + `test_submodules_phase6.py` (42 tests), covering the 13 submodules directly and verifying public re-exports
- **Logger added to 3 previously silent modules** — `_cache_loading.py` (6 `except` blocks → `logger.debug` with `exc_info`), `_performance_relative.py` (1 catch-all), `_rag_github.py` (1 network error); all submodule `except Exception` blocks are now traced
- **Centralized logging system** (`src/utils/log_config.py`) — `setup_app_logging()`: file-only logs (`data/logs/app.log` 5 MB×3, `data/logs/sync.log` 10 MB×5), no console output; `setup_script_logging()` for CLI scripts; `log_duration()` context manager with configurable millisecond threshold. Wired into app launch, player loading, session selection, filter changes, DataFrame loading, KPIs, match navigation (last match / carnage / previous match buttons), sync UI, backfill CLI, tailscale, and RAG. `data/logs/` is excluded from the repository.
- **`.gitattributes`** — enforces `eol=lf` across the repo; resolves pre-commit mixed line-ending conflicts on Windows (`core.autocrlf=true`)
- **`pyproject.toml`** — `per-file-ignores` for `scripts/*` and `launcher.py` (C901/PLR0912/PLR0913/PLR0915 complexity tolerated in utility scripts)
- **Quality enforcement** — `scripts/check_code_size.py` (ratchet), `tests/test_code_quality.py` (3 structural quality tests), CLAUDE.md rules 13-17 (max file/function size, max args, complexity, SRP)

### Bug fixes (backported from `main`)

- **Post-sync filter auto-invalidation** (`src/app/filters_render.py`) — `_filters_db_key_{player}` replaces the write-once `_filters_loaded_*` boolean; filters now reset automatically when the DB changes (sync, CLI, backfill, profile change)
- **Post-sync citation computation** (`src/data/citations_backfill.py`) — incremental module called by `DuckDBSyncEngine` after each sync; newly inserted matches get their citations immediately
- **SyncLock wired into the UI** (`src/ui/sync.py`) — `SyncLock(timeout=0)` protects against concurrent inter-process syncs; `SyncAlreadyRunning` is surfaced cleanly to the user, and DuckDB WAL is flushed before `end_sync_mode()`
- **Process-level Tailscale guard** (`src/utils/tailscale.py`) — module-level `threading.Event` replaces per-session `st.session_state`; `ensure_funnel_started_once()` guarantees only one startup and one Discord notification per Python process
- **False Discord webhook alert** (`src/utils/startup_check.py`) — skips the check when Doppler is active; loads `.env.local` before validation
- **Missing `_PERF_SCORE_AVAILABLE`** (`src/data/sync/_performance.py`) — module-level variable missing after the `engine.py` split into mixins; added `try/except ImportError` guard with `_PERF_SCORE_AVAILABLE = True/False`; fixes `F821 Undefined name` and runtime `NameError`
- **Fragile NaN check** (`src/ui/pages/match_view.py`) — replaces floating-point NaN idiom `x == x` with `x is not None`
- **i18n** (`src/ui/translations.py`, `src/ui/i18n/widgets.py`) — restored 2 truncated `PAIR_FR` keys, removed duplicate `tm_session_trend`, cleaned 343 redundant entries (399 → 56 useful entries)
- **Per-player backfill detection** (`scripts/backfill/detection.py`) — the 6 per-player flags (`medals`, `personal_scores`, `performance_scores`, `accuracy`, `shots`, `enemy_mmr`) now check the current player's real data instead of the global `backfill_completed` bitmask; fixes a bug where the first synced player masked matches for other players; new `_player_done_guard()` function; 15 new multi-player tests + 9 adapted tests

---

## [5.3.0] - 2026-02-28

### Added

- **LUSR (LevelUp Skill Rank) — TrueSkill 2 per-group rating system** (`src/analysis/`)
  - `skill_rating_config.py`: TrueSkill 2 constants, Bronze→Onyx I-VI tiers, 5-component composite score
  - `playlist_groups.py`: 6 isolated Halo Infinite groups (ranked 1.00, arena 0.80, tactical 0.70, btb 0.60, social 0.40, fun 0.15) with detection by `pair_name` prefix or `playlist_name`
  - `skill_rating.py`: full algorithm — `PlayerState` per group, `compute_composite_score()`, `trueskill_update()`, `compute_enemy_strength()`, per-group inactivity, sequential `compute_skill_ratings_batch()`
  - `skill_rating_calibration.py`: COMPOSITE_WEIGHTS calibration module by comparison with `team_mmr` API (random grid search, MAE or Pearson correlation metric)
  - 68 unit tests covering the algorithm, groups, inactivity, tiers, and calibration

- **Per-group LUSR: independent TrueSkill state per context**
  - `existing_states: dict[str, PlayerState]` replaces `existing_state: PlayerState` — a ranked match no longer affects the arena rating
  - `states.setdefault(group, PlayerState())` creates a state on the first match of each group
  - Inactivity, accuracy history, and σ decay are now per-group

- **LUSR/CSR Backfill** (`scripts/backfill_data.py`, `scripts/backfill/`)
  - `--lusr` / `--force-lusr`: local LUSR computation from `shared.match_participants` (sequential, incremental)
  - `--csr` / `--force-csr`: CSR retrieval from the Halo API for ranked matches
  - `compute_lusr_for_player()` in `strategies.py`: UPSERT into `match_skill_rank` with `rating_delta`, tier, and tier_label
  - `match_skill_rank` table auto-created by `ensure_match_skill_rank_table()` in `migrations.py`
  - Backfill bits: `lusr = 1 << 16` (65536), `csr = 1 << 17` (131072) in `BACKFILL_FLAGS`

- **SyncScope flags**: `lusr`, `force_lusr`, `csr`, `force_csr` in `src/data/sync/scope.py`

- **CSR data model** (`src/data/sync/models.py`, `src/data/sync/transformers.py`)
  - `SkillParticipantUpdate` extended: `pre_match_csr`, `post_match_csr`, `csr_tier`, `csr_sub_tier`
  - `RankRecap.PreMatchCsr` / `PostMatchCsr` extraction in `transform_all_skill_stats()`

- **LUSR visualization** (`src/visualization/timeseries_combat.py`)
  - `plot_lusr_timeseries()`: semi-transparent tier zones, `rating ± deviation` confidence band, 20-match smoothed trend

- **UI — Career page and Match View** (`src/ui/pages/`)
  - `career.py`: visual cards per group (90px centered rank image, LUSR/CSR badge, ▲/▼ delta) + group selector (`st.selectbox`) for the progression graph — replaces the expander table and tabs
  - `match_view.py`: ��� Rank tab with rank badge, colored progress bar, green/red delta

- **Calibration CLI**
  - `python -m src.analysis.skill_rating_calibration --player <GT> [--n-samples 300] [--metric corr]`
  - Grid search over the weight simplex (uniform Dirichlet distribution, reproducible seed)
  - Displays optimal weights ready to copy into `skill_rating_config.py`

- **Post-sync/backfill Discord notifications** (`src/utils/discord_notifier.py`)
  - New failsafe module — no external dependencies (stdlib `urllib.request` only)
  - Sends a Rich Embed to Discord at the end of each `sync.py` and `backfill_data.py` run
  - Embed content: operation, start/end time, total duration, number of players and matches processed
  - Per player: synced matches (or backfill-processed), data completeness (medals + events), last match (map, mode, KDA, result, playlist)
  - Bar color: green ✅ (all OK), yellow ⚠️ (incomplete data), red ❌ (error)
  - `--no-discord` flag on `sync.py` and `backfill_data.py` to disable on demand
  - `notify_operation_done()`: public entrypoint — `disabled=True` short-circuits immediately
  - `fetch_last_match_info(xuid)`: SQL on `shared_matches.duckdb` (JOIN `match_registry` + `match_participants`)
  - `count_new_matches(xuid, gamertag, since)`: counts matches with `first_sync_at >= since`
  - `count_matches_missing_data(xuid)`: counts matches with `medals_loaded=FALSE OR events_loaded=FALSE`

- **Secure Discord webhook configuration**
  - `discord_notifications_enabled: false` toggle in `app_settings.json` (no secrets in this file)
  - Webhook URL read from `DISCORD_WEBHOOK_URL` in `.env.local` (gitignored) via `_load_dotenv_if_present()`
  - Backwards-compatible fallback on the `discord_webhook_url` key in `app_settings.json`
  - Documented section in `.env.local.example`

- **Full FR/EN internationalization (i18n)** (`src/ui/i18n/`)
  - Dedicated i18n package with specialized modules: `common.py`, `pages.py`, `widgets.py`, `viz.py`, `cli.py`
  - Functions: `t(key, lang=None)` (Streamlit UI), `viz_t(key, lang)` (Plotly), `discord_t(key, **kwargs)` (Discord), `ct(key, **kwargs)` (CLI/scripts)
  - Language stored in `st.session_state["lang"]` (Streamlit) or `LEVELUP_LANG` env variable (scripts)
  - ������/������ language selector in the sidebar (`_render_lang_selector()` in `src/app/sidebar.py`)
  - Three fields in `AppSettings`: `lang`, `discord_lang`, `cli_lang` (default `"fr"`)
  - `src/ui/translations.py` bilingual: `translate_playlist_name(name, lang)` and `translate_pair_name(name, lang)` — preserves `" on Map"` grouping and Halo prefixes (Arena, BTB, Ranked)
  - `src/analysis/mode_categories.py`: bilingual `normalize_pair_name_to_mode_ui(pair_name, lang)`
  - `src/utils/discord_notifier.py` fully bilingual: `_format_player_field`, `build_embed_payload`, outcomes (���/���/⚖️/���), KDA (`{k}K / {d}D / {a}A` vs `{k}F / {d}D / {a}A`), operation labels, footer
  - `src/visualization/distributions_outcomes.py` bilingual: Wins/Losses/Ties/Unfinished traces, time buckets (match/hour/day/week/month), win rate heatmap (EN/FR days), `plot_matches_at_top_by_week` (Others/Top Rate)
  - `src/visualization/antagonist_charts.py` bilingual: `plot_duel_history` translates Win/Loss/Tie in duel annotation
  - `src/ui/pages/win_loss.py`: all viz calls pass `lang=get_lang()`

### Changed

- **Sessions filter — Solo / Squad redesign** (`src/app/filters_render.py`, `src/ui/cache_filters.py`, `src/ui/filter_state.py`, `src/ui/pages/teammates.py`)
  - Sidebar Sessions section split into two mutually exclusive subsections: **"En solo"** (sessions where no selected friend was present) and **"Mon escouade"** (sessions where at least one friend was present)
  - Replaces the "Dernière session en trio" single button with two full subsections, each with Last/Previous buttons and a selectbox
  - Classification is dynamic: friend XUIDs come from the Teammates multiselect (max 3), matched against `teammates_signature` via vectorized Polars `str.contains` — O(k×N) in C/SIMD, no Python loops on match rows
  - Friend selection persisted in `FilterPreferences` (`friends_selected_labels`, `picked_solo_session_label`, `picked_squad_session_label`)
  - Shadow keys added for Streamlit 1.54+ widget-state persistence across navigation
  - `teammates_signature` propagated through all return paths of `cached_compute_sessions_db`
  - 7 new i18n keys in `widgets.py` (`filter_solo_title`, `filter_squad_title`, `filter_last_carnage`, `filter_prev_carnage`, `filter_solo_session_label`, `filter_squad_session_label`, `filter_squad_no_friends`)
  - `apply_filters` now uses `filter_state.base_s_ui` directly (single source of truth) instead of re-calling `cached_compute_sessions_db`, with an empty-intersection guard

- **LUSR algorithm — Elo-style update (`K_ELO = 32`)** replaces TrueSkill draw zone
  - Root cause of divergence: `v_draw(t > 0)` gave positive deltas even at composite=0.5, creating infinite drift when `state.mu > INITIAL_MU` or when the player over-fragged their `kills_expected`
  - New mu formula: `delta_mu = K_ELO × (composite − 0.5) × weight_factor` → exact ZERO at composite=0.5, independent of `mu_opp`
  - Sigma retains TrueSkill reduction evaluated at t=0 (symmetric, `mu_opp` only influences `c²`)
  - Result: stabilized ratings — SpartanA (Diamond V) → Platinum IV BTB / Platinum VI Arena / Diamond IV Ranked, SpartanB/SpartanC → Gold II-IV depending on mode
- **Composite score calibrated on 1765 matches** (SpartanA, SpartanC, SpartanB — Silver → Diamond)
  - Target signal: `individual_mmr = team_mmr × (kills_expected / ke_avg_match)`
  - Weighting by `nb_matches × MAE_improvement`: SpartanA 36.7%, SpartanC 40.0%, SpartanB 13.3%
  - New weights: kills_vs_expected=31%, deaths_vs_expected=28%, damage_efficiency=23%, accuracy_delta=13%, win_factor=5%
- **damage_efficiency bias elimination**: `PlayerState.damage_eff_history` per-group — the damage component now uses a delta vs personal history (like accuracy_delta) instead of the raw value
- **mu_opp anchored on `state.mu`**: `compute_enemy_strength` uses `player_mu=state.mu` as the base estimate for opponents (matchmaking pairs players of similar level)
- **Reduced inactivity params**: `INACTIVITY_SIGMA_PER_DAY` 3.5→1.0, `MAX_INACTIVITY_DAYS` 30→14 — avoids ±200 pt swings after a long break
- **Reduced CSR seed sigma**: `PlayerState.from_csr()` starts at `sigma=MIN_SIGMA` (60) instead of `INITIAL_SIGMA × 0.6` (210) — CSR is a strong anchor; starting in a stable state avoids initial volatility

- **Career page — Estimated pre-sync XP curve** (`src/ui/pages/career.py`)
  - Dotted purple trace retroactively estimating XP for the ~561 matches played before the first sync
  - Average XP/match = `first_xp / n_pre_sync_matches` — curve starts near 0 at the oldest match and connects seamlessly to the first real snapshot
  - Visually distinct from the real XP trace (purple `#CE93D8` dotted line)

- **Career page — Projection curves to Hero rank** (`src/ui/pages/career.py`)
  - **Standard projection** (orange dashed): extrapolates from the current active XP/day rate, excluding inactivity gaps > 14 days
  - **Optimistic projection** (green dash-dot): adds weekly challenge XP (950 XP/week = 4×50 + 3×100 + 3×150) plus daily challenge XP (500 XP/day), all with ×2 XP boost — total +4 450 XP/week from challenges
  - Both curves hidden by default — click the legend to reveal them
  - Gold horizontal line at the Hero threshold (9,319,350 XP)
  - Projection capped at 10 years to avoid infinite charts
  - Legend moved to the bottom of the chart (centered)
  - 23 unit tests in `tests/test_career_xp_projection.py`

### Fixed

- **20 pre-existing tests fixed** following the v5.1 migration (shared architecture)
  - Group A (assertions/fixtures): `test_backfill_bitmask`, `test_backfill_detection`, `test_xuid_resolution_regression` (×2), `test_post_refactor_perf_contracts`, `test_data_services_contracts`, `test_media_components_sprint4` (×2), `test_media_improvements`, `test_legacy_free_global`
  - Group B (v4→v5 mocks): `test_lazy_loading` (×5 — `_get_match_source` v5.1), `test_data_contract_sessions` (v5 shared + player_match_enrichment fixture rewrite)
  - Group C (source + mocks): `test_sessions_integration` (production DB fallback hidden by `__file__` patch), `test_duckdb_repository_schema_contract` (`xuid/gamertag` schema in shared fixture), `test_teammates_impact_tab` (×2 — mock `_ensure_shared_attached` + `_load_highlight_events`)

---

## [5.2.0] - 2026-02-20

### Added

- **v5.2 Filters — Intent-based persistence** (`src/ui/filter_state.py`)
  - `FilterPreferences`: dataclass saved as JSON per player
  - Persisted modes: `playlist_mode`, `mode_mode`, `map_mode` (`"exclude"` / `"include"`)
  - Exclusion lists: `excluded_playlists`, `excluded_modes`, `excluded_maps`
  - `_detect_filter_mode()`: 70/30 heuristic — if > 70% of options are checked, use "exclude" mode; otherwise "include"
  - `reconcile_filter_prefs()`: auto-reconciliation when new options appear — new playlists/modes/maps included by default without resetting existing preferences
  - 45 unit tests in `tests/test_filter_state.py`

- **v5.2 Filters — "Experience Type" selector** (`src/app/filters_render.py`)
  - Static pre-filter: "Unranked PVP", "Ranked PVP", "PVE (Firefight)" enabling the `is_firefight` filter
  - Correct cascade deletion: modes/maps computed from full `dropdown_base` (before playlist filter)
  - `FilterPreferences` integrated into cascade filter rendering

- **PvE / Firefight v5.2 stats — Dedicated `shared_pve.duckdb` database**
  - New `data/warehouse/shared_pve.duckdb` database separate from `shared_matches.duckdb` (avoids NULL columns on 90%+ of PvP matches)
  - `pve_match_stats` table: per-player per-match Firefight stats — waves, boss kills, kills by enemy type (Banished: Grunt, Elite, Jackal, Brute, Hunter, Skimmer; Forerunner: Crawler, Soldier, Knight, Warden)
  - `ensure_pve_schema()` in `src/data/sync/migrations.py` — idempotent schema creation
  - `PVE_SCHEMA_DDL`: full DDL + `idx_pve_xuid` + `idx_pve_match_id` indexes

- **PvE stats — Python models** (`src/data/sync/models.py`)
  - `PveMatchStatsRow`: dataclass with 20 columns (waves, boss, enemy by type, pve_bits)

- **PvE stats — Transformer** (`src/data/sync/transformers.py`)
  - `extract_pve_stats(match_json)`: extraction for all players of a Firefight match
  - `_find_pve_stats_dict(player)`: recursive search for the PvE block (EliminationStats / PveStats / FirefightStats / key detection)
  - `_extract_enemy_kills_by_type(pve_dict)`: dual-structure support (direct `GruntKills` fields + `EnemyKillsByType` sub-dict)
  - `_is_firefight_match()` enhanced: 3 criteria — `GameVariantCategory` (IDs 41, 42 validated on real API JSON), `UgcGameVariant.PublicName`, `Playlist.PublicName` (firefight/baptême/survive)

- **PvE stats — Insert pipeline** (`src/data/sync/batch_insert.py`)
  - `batch_insert_pve_stats(conn, rows)`: batch insert with `INSERT OR REPLACE`

- **PvE stats — Bitmask** (`src/data/sync/constants.py`)
  - `PveBits(IntFlag)`: granular bitmask for `pve_match_stats.pve_bits` — TOTAL_KILLS, BOSS_KILLS, GRUNT, ELITE, JACKAL, BRUTE, HUNTER, SKIMMER, SENTINEL, MARINE + ALL_ENEMIES, FULL_PVE combinations
  - `MatchBits.PVE_STATS = 1 << 20`: global guard in `match_registry.backfill_completed` — set for every processed match (Firefight or not) to avoid infinite re-detection

- **PvE stats — Sync Engine** (`src/data/sync/engine.py`)
  - `_pve_connection`: lazy-init connection to `shared_pve.duckdb`
  - `_pve_db_lock`: dedicated asyncio lock
  - `_get_pve_connection()`: lazy init + `ensure_pve_schema` on first access
  - `_try_insert_pve_stats(stats_json, match_id, shared_conn)`: extraction + insert + set `MatchBits.PVE_STATS` bit — called in `_process_new_match` and `_process_known_match`

- **PvE stats — SyncScope** (`src/data/sync/scope.py`)
  - `pve_stats: bool` and `force_pve_stats: bool` fields in `SyncScope`
  - Registered in `_FORCE_MAP` and `_ALL_DATA_FIELDS`

- **PvE stats — Backfill detection** (`scripts/backfill/detection.py`)
  - Double guard: `mr.is_firefight = TRUE AND (COALESCE(mr.backfill_completed, 0) & PVE_STATS) = 0`
  - `force_pve_stats`: ignores the guard, returns all Firefight matches
  - `MatchBits.PVE_STATS` added to `compute_bits_needed_from_scope`

- **PvE stats — Backfill CLI** (`scripts/backfill/cli.py`)
  - `--pve-stats` and `--force-pve-stats` arguments

- **PvE stats — Backfill orchestrator** (`scripts/backfill/orchestrator.py`)
  - `_backfill_pve_for_match()`: opens `shared_pve.duckdb`, `ensure_pve_schema`, `batch_insert_pve_stats`, sets guard bit in `match_registry`
  - `pve_stats_inserted` counter in `_empty_result()`

- **PvE citations** (`src/analysis/citations/engine.py`)
  - `load_match_pve_stats(match_id)`: reads from `shared_pve.duckdb`
  - PvE stats merged into `match_stats` before citation computation
  - `pve_stat` recognized as `mapping_type` (handled identically to `stat`)

- **81 new tests**:
  - `tests/test_filter_state.py`: 45 tests — `FilterPreferences`, `_detect_filter_mode()`, `reconcile_filter_prefs()`, save/load
  - `tests/test_pve_transformers.py`: 36 tests — `_is_firefight_match()`, `_extract_enemy_kills_by_type()`, `extract_pve_stats()`, DuckDB schema, batch insert, `PveMatchStatsRow`, `PveBits`, `SyncScope.pve_stats`

- **"Last match" scoreboard** (`src/ui/pages/match_view_players.py`, `src/data/repositories/_roster_loader.py`)
  - `load_match_scoreboard(match_id)`: DuckDB query joining `match_participants` + `xuid_aliases` + `medals_earned` sub-query (Perfect Kill, ID 1512363953). 20 fields per player, sorted by `(team_id, rank)`.
  - `render_match_scoreboard()`: per-team HTML table with 18 columns — Gamertag, Rank, Score, Kills, Deaths, Assists, KDA, Killing Spree, Headshots, Perfect Kills, Shots, Shots Hit, Accuracy, Melee, Power Weapons, Damage Dealt, Damage Taken, Avg Lifetime
  - Handles N teams + players without `team_id` (NULL → separate group at the end)
  - Okabe-Ito color headers: blue `#0072B2` for the player's team, vermillion `#D55E00` for opponents
  - Player row highlighted (cyan `#00e5ff`)
  - Gamertag resolution via `load_match_gamertags_fn` (same pipeline as the former roster)
  - CSS `.os-scoreboard` / `.os-sb-*` with column wrapping (`max-width: 80px`, `word-break`)
  - Replaces the removed "Players" (roster) section

- **Per-player tokens for player-gated endpoints** (`src/data/sync/api_client.py`, `src/ui/profile_api_tokens.py`)
  - `SPNKR_OAUTH_REFRESH_TOKEN_<NORMALIZED_GT>` in `.env.local` per player (e.g.: `_SPARTANC`, `_MON_GT_2`)
  - Normalization: `re.sub(r"[^A-Za-z0-9]", "_", gt.strip()).upper()`
  - `get_tokens_for_player(gamertag)`: async, returns `Tokens | None` — skip + warning if absent (no global fallback on restricted endpoint)
  - `get_player_token_env_key(gamertag)`: returns the normalized env key
  - `profile_api_tokens.get_tokens()` extended: optional `gamertag` param — priority player token > global token (natural fallback for public endpoints)
  - `profile_api.py`, `get_profile_appearance()`: `gamertag` param propagated to SPNKr fetch
  - `load_profile_api()`: derives the gamertag from the DB and passes it to `get_profile_appearance()` — fixes adornment/career rank for players who do not own the global token

- **Player-gated Career Rank sync** (`src/data/sync/engine.py`)
  - `sync_career_rank()` uses `get_tokens_for_player()` — silent skip + warning if absent
  - Persists `spartan_id` in `career_progression` (column added via `add_spartan_id_to_career_progression()` migration)
  - `CareerRankRow.spartan_id` in `src/data/sync/models.py`

- **Spartan ID in the hero banner** (`src/ui/styles.py`, `src/app/main_helpers.py`)
  - `get_hero_html()`: new `spartan_id` parameter — displayed in the career-rank section under the rank label (`.career-rank__spartan-id`)
  - `render_profile_hero()`: loads `spartan_id` from `career_progression` (DB, source of truth) and passes it to the hero HTML
  - CSS `.career-rank__spartan-id`: compact, semi-transparent, letter-spaced style

- **32 new tests** (`tests/test_player_tokens.py`)

### Changed

- **Colorblind accessibility — Okabe-Ito palette migration** (`src/visualization/`)
  - 7 visualization files updated: `antagonist_charts.py`, `performance.py`, `objective_charts.py`, `participation_charts.py`, `team_dominance_timeline.py`, `match_impact_timeline.py`, `friends_impact_heatmap.py`
  - Replaced saturated neon red/green pairs (incompatible with deuteranopia and protanopia) with the **Okabe-Ito** palette (Wong 2011), the international reference for accessible charts
  - Main mappings: neon green `#00ff00` → blue-green `#009E73` · red `#ff4444` → vermillion `#D55E00` · magenta `#ff66ff` → mauve pink `#CC79A7` · team colors `#3DFFB5`/`#FF4D6D` → blue `#0072B2`/vermillion `#D55E00`
  - Each palette documented with previous hex values and justification in a comment block

- **`_is_firefight_match()`** — Merging of the two duplicated definitions into a single unified function covering all 3 criteria (GameVariantCategory + UgcGameVariant.PublicName + Playlist.PublicName)

### Deprecated

- **`display_name_from_xuid()` and `get_xuid_aliases()`** (`src/ui/aliases.py`) — Marked `.. deprecated::`. Use `load_match_gamertags_fn` for match context. Kept for scripts/migration/export.

### Removed

- **"Players" (roster) section** from the Last Match page — Replaced by the scoreboard. `render_roster_section` is no longer called from `match_view.py`.

### Fixed

- **`_is_firefight_match()` duplication** — Two definitions coexisted in `transformers.py`. The second silently overrode the first, making detection via `UgcGameVariant` inoperative. Merged into a single complete definition.

---

## [5.1.0] - 2026-02-17

### Added

- **`src/data/sync/scope.py` module** — **SyncScope** dataclass centralizing flags
  - Replaces 30+ boolean kwargs copied across 6 files (cli → backfill_data → orchestrator → detection → API)
  - `SyncScope.from_cli_args(args)`: construction from argparse
  - `SyncScope.make_all()`: factory for `--all-data`
  - `resolve()`: automatic implications (`all_data` → fields, `force_X` → X)
  - Properties: `has_any_option()`, `needs_api`, `needs_local_only`, `requested_types`
  - Registries: `_ALL_DATA_FIELDS`, `_FORCE_MAP`, `_REQUESTED_TYPE_MAP`
  - 98 unit tests in `tests/test_sync_scope.py`
  - **Add a new type**: 1 field in SyncScope + 1 CLI arg + business logic implementation
- **`src/ui/streamlit_modern.py` module** — Modern Streamlit compatibility wrappers
  - `fragment_if_available`: graceful-degradation decorator for `@st.fragment`
  - `PLOTLY_CLEAN_CONFIG`: Plotly config without toolbar
  - `plotly_chart`: wrapper with clean config by default
  - `HAS_FRAGMENT`, `HAS_NAVIGATION`: version detection
- **`src/ui/vectorize_helpers.py` module** — Vectorized replacement for `map_elements()`
  - `build_mapping()`: pre-computed dict mapping on distinct values
  - `vectorized_apply()`: vectorized apply via `replace_strict()`
  - `safe_int_format()`, `format_score_pair()`: reusable Polars expressions
- **`get_shared_matches_path()` helpers** — Centralized functions in `src/utils/paths.py`
  - `get_shared_matches_path()`: absolute path to `shared_matches.duckdb`
  - `get_shared_matches_path_from_player()`: deduction from player path
- **`cleanup_legacy_tables.py` script** — Obsolete table removal
  - 9 tables removed: `match_stats`, `medals_earned`, `highlight_events`, `player_stats`, `xuid_aliases`, + 4 `mv_*` views
  - Options: `--dry-run`, `--backup`, `--all`
  - Automatic backups in `backups/pre_cleanup/`
- **`mv_player_matches` materialized view** — v5.1 performance optimization
  - Pre-computed joins on match_participants + match_registry + metadata
  - SQL parsing reduced from 170→10 lines per query
  - Performance gain: -70% SQL parsing
- **Streamlit Repository Cache** — `get_cached_repository_st()` with `@st.cache_resource(ttl=3600)`
  - Persistent DB connection between UI pages
  - Gain: 80ms→<20ms connection
- **DuckDB Performance Indexes** — 16+ indexes created on 9 tables
  - Composite indexes `(xuid, match_id)`, `(match_id, xuid)`
  - Sorted indexes on `start_time`
- **Metadata schema cache** — `_has_column()` and `_has_shared_mp_column()` cached
  - Avoids repeated `information_schema` queries
- **LEGACY banner migration scripts** — 5 scripts flagged + README.md
  - Clear "OUT OF SERVICE POST-V5.1" banner
  - Documentation in `scripts/migration/README.md`

### Changed

- **`backfill_data.py` refactored** — `main()` uses `SyncScope.from_cli_args()` (−90 lines)
  - No longer need to copy 30+ `args.X` twice for `--all` and `--player`
- **`orchestrator.py` refactored** — `backfill_player_data`, `backfill_all_players`, `_backfill_with_api` accept `scope=SyncScope`
  - Old kwargs preserved (marked `LEGACY`) for backward compatibility
  - `requested_types` built via `scope.requested_types` instead of 16 `if/append`
- **`detection.py` refactored** — `find_matches_missing_data` accepts `scope=SyncScope`
  - Old kwargs preserved (marked `LEGACY`) for backward compatibility
- **Bumped Streamlit ≥1.37.0** — Required for `@st.fragment` and future `st.navigation` migration
- **Plotly `config={"displayModeBar": False}`** — Applied to 69 `st.plotly_chart` calls (15 files)
  - Removes Plotly toolbar for a cleaner UI
- **`@fragment_if_available`** — Decorator applied to 5 multi-chart pages
  - timeseries, session_compare, win_loss, objective_analysis, career
  - Reduces re-renders to the fragment only on filter interactions
- **`match_history.py` modernized** — Replaced custom HTML with `st.dataframe` + `column_config`
  - Dead code removed: `_format_score_label`, `_fmt`, `_fmt_mmr_int`
  - Native Streamlit virtualization for wide tables
- **`st.navigation` lazy loading** — 11 page closures in `streamlit_app.py`
  - `build_navigation()` + `render_page_selector_nav()` in `page_router.py`
  - Legacy fallback `dispatch_page()` for Streamlit < 1.36
  - Only visited pages are imported → -60% initial memory
- **Centralized `duckdb_read_only()`** — Context manager in `src/utils/db.py`
  - 7 files migrated (career, cache_loaders, cache_filters, media_library, multiplayer, data_loader)
  - Direct `duckdb.connect` calls: 14 → 4 (remaining: sync engine, legitimate writes)
- **Reduced `st.rerun()`** — 32 → 14 in `src/`
  - `checkbox_filter.py`: 16 reruns → 0 via `on_click`/`on_change` callbacks
  - Trio button filters: `on_click=_apply_trio_filter`
- **`unsafe_allow_html` hardening** — `html.escape()` on dynamic data
  - `kpi.py` and `performance.py`: XSS protection
  - `sidebar.py` brand: HTML → `st.header()` + `st.divider()`
- **Modernization regression tests** — 30 tests in `test_8ter_modernisation.py`
  - Coverage: staticPlot, fragments, st.navigation, duckdb_read_only, st.rerun, html.escape
- **Complete `map_elements()` eradication** — 28 occurrences replaced in 15 files
  - Replaced with `build_mapping()` + `replace_strict()` or native Polars expressions
  - Files: filters.py, filters_render.py, win_loss.py, last_match.py, stats.py,
    match_view_charts.py, media_library.py, teammates_helpers.py, session_compare.py,
    session_compare_charts.py, duckdb_analytics.py, match_view.py, citations.py,
    teammates_service.py, media_indexer.py
- **`xuid_aliases` migration → `shared_matches.duckdb`** — Single centralized source
  - 9 files migrated to read from `shared.xuid_aliases` (13,955 rows)
  - Local `stats.duckdb` fallbacks removed
  - Files: `aliases.py`, `xuid.py`, `multiplayer.py`, `cache_loaders.py`, `engine.py`, `_roster_loader.py`, `sessions_backfill.py`, `sync.py`, `resolve_missing_gamertags.py`
- **`_get_match_source()`** now returns a 3-tuple `(source_sql, params, uses_mv)`
  - Enables skipping redundant joins in v5.1 mode
- **8+ cache_loaders functions** migrated to `get_cached_repository_st()`
  - Redundant new connections removed
- **metadata/MMR joins** skipped in v5.1 mode when `uses_mv=True`
  - RC3/RC4: -3 LEFT JOINs on the critical path

### Fixed

- **Citations tab showed 159 citations instead of 45** — Filtering by `citation_mappings.enabled` re-enabled
  - The `halo5_commendations_fr.json` JSON contains 159 citations (weapons, Spartan Companies, etc.)
  - Filtering had been removed, displaying all citations including those without mapping
  - Fix: JSON items are now filtered by normalized names of enabled citations via `CitationEngine.load_mappings()`
  - File: `src/ui/commendations.py`

### Removed

- **Legacy player DB tables** — 9 tables per player, data centralized
  - `match_stats`, `medals_earned`, `highlight_events`, `player_stats`, `xuid_aliases`
  - Obsolete views: `mv_match_stats_with_context`, `mv_recent_matches`, `mv_team_stats`, `mv_opponent_stats`
  - 38,528 rows freed across 4 players
- **SQLite runtime references** — 0 `import sqlite3` in `src/`
- **`metadata.db` references** — Everything migrated to `metadata.duckdb`
- **Deprecated `attach_sqlite` method** — Removed from duckdb_engine.py

### Performance

| Metric | v5.0 | v5.1 | Gain |
|--------|------|------|------|
| DB connection | 80ms | <20ms | **-75%** |
| load_matches(100) | 200ms | <80ms | **-60%** |
| First UI page | 1500ms | <800ms | **-47%** |
| SQL parsing/query | 170 lines | 10 lines | **-94%** |

---

## [5.0.0] - 2026-02-15

### Added

- **shared_matches.duckdb architecture** — Shared database centralizing matches for all players
  - 6 tables: `match_registry`, `match_participants`, `highlight_events`, `medals_earned`, `xuid_aliases`, `highlight_events_id_seq` sequence
  - 14 optimized indexes (match_id, xuid, start_time, composites)
  - Full DDL schema: `scripts/migration/schema_v5.sql`
  - Documentation: `docs/SHARED_MATCHES_SCHEMA.md`
- **v4 → v5 Migration** — Incremental per-player migration scripts
  - `scripts/migration/create_shared_matches_db.py`: shared DB creation
  - `scripts/migration/migrate_player_to_shared.py`: per-player migration
  - Result: 1289 matches migrated, 285 shared (22.1%), 0 orphans
- **Shared match detection in Sync Engine** — Lightweight sync for already-known matches
  - `_process_known_match()`: personal enrichment only (saves 1-2 API calls/match)
  - `_process_new_match()`: full sync to shared (registry + participants + events + medals)
  - `extract_all_medals()`: medal extraction for ALL players in the match
  - `extract_match_registry_data()`: common match data extraction
- **Multi-DB ATTACH in DuckDBRepository** — Transparent reads from `shared_matches.duckdb`
  - `shared_db_path` auto-detected or configurable
  - Native queries on `shared.match_participants`, `shared.match_registry`, `shared.medals_earned`
  - Propagation in the repository factory
- **`_get_match_source()` sub-query** — Abstraction allowing all UI pages to read from shared without modification
- **v5 Sync API optimizations**
  - Parallelized skill + events API calls (`asyncio.gather`)
  - DB insert batching (commit every 10 matches)
  - Performance scores computed in batch post-sync
  - Optimized rate limit (10 req/s, parallel_matches=5)
- **DuckDB-first citations** — New per-match stored citations system
  - `CitationEngine`: computation and SQL aggregation engine
  - `citation_mappings` table in `metadata.duckdb`: 14 rules (8 existing + 6 reintegrated)
  - `match_citations` table in each player's `stats.duckdb`
  - Backfill CLI: `--citations` / `--force-citations` in `scripts/backfill_data.py`
  - 6 reintegrated objective citations: Flag Defender, Got Your Back!, Flag Stalker, Stake a Claim, Charge!, Forced Annexation
  - `enabled` column in `citation_mappings` for disabling without deletion
  - V5 (shared_matches) support in `CitationEngine` with V4 fallback
  - Documentation: `docs/CITATIONS.md`
- **MockStreamlit test framework** — `MockStreamlit` fixture in `conftest.py` for headless UI page testing
- **+946 tests** added (S1→S7ter) — total 2768 passed, 0 failed, 38 skipped
- **Post-migration cleanup script** — `scripts/cleanup_player_dbs_v5.py`
  - Removes redundant tables from player DBs after v5 migration (match_stats, match_participants, highlight_events, medals_earned)
  - `--dry-run` mode for simulation without modification
  - Optional backup before cleanup
  - Automatic `shared_matches.duckdb` existence validation
  - Automatic VACUUM for disk space recovery (-85% average size reduction)
  - Documentation: `docs/CLEANUP_V5.md`
- **Documentation**: `docs/SHARED_MATCHES_SCHEMA.md`, `docs/SYNC_OPTIMIZATIONS_V5.md`, `docs/TESTING_V5.md`, `docs/ARCHITECTURE_V5.md`, `docs/MIGRATION_V4_TO_V5.md`, `docs/CLEANUP_V5.md`

### Changed

- **`DuckDBSyncEngine`** refactored to write to `shared_matches.duckdb` (matches, participants, events, medals)
- **`DuckDBRepository`** refactored with ATTACH `shared_matches.duckdb` in read-only
  - `load_match_participants()` → reads from `shared.match_participants`
  - `load_highlight_events()` → reads from `shared.highlight_events`
  - `load_medals_for_match()` → reads from `shared.medals_earned`
  - `load_matches()` → JOIN `shared.match_participants` + `shared.match_registry` + `player_match_enrichment`
- **All UI pages** use `_get_match_source()` instead of `match_stats` directly
- **`render_h5g_commendations_section()`** uses `CitationEngine` (SQL aggregation, ~90% faster)
- **`render_citations_page()`** simplified — no longer pre-aggregates medals/stats for citations
- **Citation filtering** driven by `citation_mappings.enabled` (no longer needs the exclusion JSON)
- **`pyproject.toml` version** bumped from 3.0.0 to 5.0.0
- **Project status**: Development Status 4-Beta → 5-Production/Stable

### Removed

- **v4 compatibility VIEWs** removed (`scripts/migration/remove_compat_views.py`)
- **Duplicated data** in player DBs: `match_participants`, `highlight_events`, `medals_earned` centralized in shared
- **`src/db/migrations.py` shim** — deprecated, removed in favor of `src.data.sync.migrations`
- `CUSTOM_CITATION_RULES` dict (old `commendations.py`)
- `_compute_custom_citation_value()` (slow iterations, replaced by SQL)
- `load_h5g_commendations_tracking_rules()` (replaced by `citation_mappings` DuckDB)
- `DEFAULT_H5G_TRACKING_ASSUMED_PATH` / `DEFAULT_H5G_TRACKING_UNMATCHED_PATH` constants
- Dependency on commendation tracking JSON files
- JSON exclusion logic in `render_h5g_commendations_section()`

### Fixed

- **Flaky Windows tests**: `tmp_dir` → `tmp_path` to avoid DuckDB `WinError 32` (file locking)
- **`lazy_loading` tests**: v4 mode forced for compatibility

### Performance

| Metric | v4 | v5 | Gain |
|--------|----|----|------|
| Storage (4 players) | 800 MB | 250 MB | **-69%** |
| DB size per player | 200 MB | 30 MB | **-85%** |
| API calls (sync 4 players) | 12,000 | 3,300 | **-72%** |
| Sync time (100 matches) | 45 min | 12 min | **-73%** |
| Time/match (shared) | 16s | 0.5s | **-97%** |
| Time/match (new) | 16s | 2-3s | **-81%** |
