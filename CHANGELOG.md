# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

> French version: [docs/FR/CHANGELOG.md](docs/FR/CHANGELOG.md)

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
