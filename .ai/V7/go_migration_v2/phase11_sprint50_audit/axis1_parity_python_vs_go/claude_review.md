# Axe 1 — Review Claude — Parité Python↔Go + Streamlit↔React

## Métadonnées du passage

| Champ | Valeur |
|-------|--------|
| Auteur LLM | **Claude** |
| Date du passage | 2026-04-18 |
| SHA Python (worktree `LevelUp`) | `db638c09` |
| SHA Go (worktree `LevelUp-go-migration`) | `93c3cd66` |
| SHA React (même worktree que Go) | `93c3cd66` |
| Durée de l'analyse | ~2h |
| Corpus analysé (nb fichiers) | ~55 Python pages/analysis / ~40 Go handlers+services / ~30 React features |

## Synthèse exécutive

La parité fonctionnelle entre Python/Streamlit et Go/React est **globalement atteinte** pour les parcours P0/P1 (bootstrap, career, match history, explorer, match view). Les 41 routes Go couvrent l'intégralité de la matrice OpenAPI MVP + les surfaces post-MVP (home, squad, citations, media, timeseries, sessions, settings, setup, auth, sync). Le frontend React couvre 14 features sur les ~16 pages Streamlit.

Les écarts principaux concernent : (1) la page **Win/Loss** absente du Go et React (🟠), (2) la page **Objectifs** absente (🟠), (3) l'i18n React non implémentée (🟠), (4) le TODO sur `media/reset-index` (🟡), et (5) quelques différences de richesse de contenu sur certaines pages (carrière, timeseries). Les 7 algorithmes cœur sont portés en Go avec golden values.

---

## A. Parité API (endpoints HTTP)

### A.1 Endpoints matrice OpenAPI P0/P1

| Endpoint | Côté Python | Côté Go | Parité contrat | Parité payload | Parité status codes | Écart | Classif |
|----------|-------------|---------|:--------------:|:--------------:|:-------------------:|-------|:-------:|
| `GET /bootstrap` | `routers/bootstrap.py` | `handlers/bootstrap.go` | ✅ | ✅ | ✅ | — | 🟢 |
| `GET /players` | `routers/bootstrap.py` | `handlers/bootstrap.go` | ✅ | ✅ | ✅ | — | 🟢 |
| `POST /session/context` | `routers/bootstrap.py` | `handlers/session_context.go` | ✅ | ✅ | ✅ | — | 🟢 |
| `POST /filters/resolve` | `routers/filters.py` | `handlers/filters.go` | ✅ | ✅ | ✅ | — | 🟢 |
| `GET /pages/career` | `routers/career.py` | `handlers/career.go` | ✅ | ✅ | ✅ | — | 🟢 |
| `GET /pages/career/top-matches` | `routers/career.py` | `handlers/career.go` | ✅ | ✅ | ✅ | — | 🟢 |
| `GET /pages/career/encounters` | `routers/career.py` | `handlers/career.go` | ✅ | ✅ | ✅ | — | 🟢 |
| `POST /pages/match-history/query` | `routers/match_history.py` | `handlers/match_history.go` | ✅ | ✅ | ✅ | — | 🟢 |
| `GET /pages/match-history/export` | `routers/match_history.py` | `handlers/match_history.go` | ✅ | ✅ | ✅ | — | 🟢 |
| `GET /directory/gamertags/search` | `routers/explorer.py` | `handlers/gamertag.go` | ✅ | ✅ | ✅ | — | 🟢 |
| `POST /pages/explorer/player-query` | `routers/explorer.py` | `handlers/explorer.go` | ✅ | ✅ | ✅ | — | 🟢 |
| `POST /pages/explorer/matches-query` | `routers/explorer.py` | `handlers/explorer.go` | ✅ | ✅ | ✅ | — | 🟢 |
| `GET /matches/{match_id}` | `routers/match_view.py` | `handlers/match_view.go` | ✅ | ✅ | ✅ | — | 🟢 |

**13/13 endpoints P0/P1 implémentés.** Aucun stub `501`.

### A.2 Endpoints post-MVP (hors matrice, ajoutés au fil des sprints)

| Endpoint | Go (handler) | Implémenté ? | Classif |
|----------|-------------|:------------:|:-------:|
| `GET /health` | `health.go` | ✅ | 🟢 |
| `GET /changelog` | `changelog.go` | ✅ | 🟢 — feature nouvelle |
| `POST /auth/device-flow/start` | `auth.go` | ✅ | 🟢 |
| `GET /auth/device-flow/{attempt_id}` | `auth.go` | ✅ | 🟢 |
| `GET /settings` | `settings.go` | ✅ | 🟢 |
| `PATCH /settings` | `settings.go` | ✅ | 🟢 |
| `POST /settings/media/reset-index` | `settings.go` | ⚠️ TODO | 🟡 — handler existe mais logique = TODO Sprint 19 |
| `POST /setup/players` | `setup.go` | ✅ | 🟢 |
| `POST /setup/smoke-test` | `setup.go` | ✅ | 🟢 |
| `GET /jobs/{job_id}` | `jobs.go` | ✅ | 🟢 |
| `POST /sync/initial` | `sync_handler.go` | ✅ | 🟢 |
| `GET /pages/home` | `home.go` | ✅ | 🟢 |
| `GET /battlepass` | `home.go` | ✅ | 🟢 |
| `GET /challenges` | `home.go` | ✅ | 🟢 |
| `GET /pages/sessions` | `sessions.go` | ✅ | 🟢 |
| `POST /pages/stats/query` | `stats.go` | ✅ | 🟢 |
| `GET /pages/squad` | `squad.go` | ✅ | 🟢 |
| `POST /pages/synthesis` | `squad.go` | ✅ | 🟢 |
| `POST /pages/citations` | `citations.go` | ✅ | 🟢 |
| `POST /pages/commendations` | `citations.go` | ✅ | 🟢 |
| `POST /pages/media` | `media.go` | ✅ | 🟢 |
| `POST /media/upload` | `media.go` | ✅ | 🟢 — feature nouvelle |
| `POST /pages/teammates` | `teammates.go` | ✅ | 🟢 |
| `POST /pages/timeseries` | `timeseries.go` | ✅ | 🟢 |
| `POST /pages/session-compare` | `session_compare.go` | ✅ | 🟢 |
| `POST /pages/last-match/resolve` | `last_match.go` | ✅ | 🟢 |
| `PATCH /matches/{match_id}/exclusion` | `match_exclusion.go` | ✅ | 🟢 — feature nouvelle |
| `GET /match-exclusions` | `match_exclusion.go` | ✅ | 🟢 — feature nouvelle |

**Total : 41 routes Go, 0 stub `501`, 1 TODO mineur (media reset-index).**

### A.3 Endpoints Python sans équivalent Go

| Fonctionnalité Python | Écart | Classif |
|-----------------------|-------|:-------:|
| Page **Win/Loss** (`win_loss.py`) — outcomes, stacked bars, heatmap, streaks, bullet charts | Aucun endpoint Go dédié ni page React | 🟠 |
| Page **Objectifs** (`objective_analysis.py`) — scores objectif/frag, scatter, gauge, trend, awards | Aucun endpoint Go dédié ni page React | 🟠 |

---

## B. Parité pages UI (Streamlit → React)

| Page métier | Streamlit | React | Features couvertes | Features manquantes | Modernisées | Classif |
|-------------|-----------|-------|-------------------:|:-------------------:|:--------------------:|:-------:|
| Home | `home_mission_control.py` + 6 modules | `HomePage.tsx` | KPIs, battle pass, défis, hero card | Médias récents, activité résumé session embed | — | 🟡 |
| Career | `career.py` + 8 modules | `CareerPage.tsx` + 4 composants | Rang, gauges, XP history, LUSR, top matchs, encounters | Projection Héros multi-joueurs, encounters full | Composants découpés | 🟡 |
| Synthesis | `synthesis.py` | `SynthesisPage.tsx` | KPIs, heatmap, breakdown | Duel Solo vs Escouade | — | 🟡 |
| Match history | `match_history.py` + `match_table_html.py` | `MatchHistoryPage.tsx` + `MatchHistoryTable.tsx` | Tableau paginé, filtres, CSV export | Win rate historique par carte inline | TanStack Table | 🟢 |
| Match view | `match_view.py` + 10 modules | `MatchViewPage.tsx` + 4 composants | Header, onglets, scoreboard, stats, rank | Timeline weapon kills, citations détaillées match | React composants dédiés | 🟡 |
| Last match | `last_match.py` | `LastMatchPage.tsx` | Navigation prev/next, résolution match | Détection changement session | — | 🟢 |
| Explorer | `explorer.py` + 4 modules | `ExplorerPage.tsx` + `GamertagSearchInput` | Recherche floue, filtres, résultats | Badges encounter, deep linking | — | 🟡 |
| Session compare | `session_compare.py` + 5 modules | `SessionComparePage.tsx` | Sélection 2 sessions, métriques, radar | Auto-matching, historique complet sessions | — | 🟡 |
| Timeseries | `timeseries.py` + 5 modules | `TimeseriesPage.tsx` | Charts principaux | 5 onglets complets (Résumé/Cartes/Distributions/Progression/Avancé), ~20 charts | — | 🟠 |
| Squad / Teammates | `teammates.py` + 10+ modules | `SquadPage.tsx` | KPIs, analyse basique | Multi-sélection 3 max, radars synergie, heatmaps carte, trios, impact | — | 🟡 |
| Citations | `citations.py` | `CitationsPage.tsx` | KPIs, grille médailles, distribution | Deltas filtrés, commendations full | — | 🟡 |
| Media | `media_library.py` + `media_v2.py` + 6 modules | `MediaPage.tsx` | Grille, filtres | V2 likes, association temporelle, grille groupée | — | 🟡 |
| Settings | `settings.py` | `SettingsPage.tsx` | Config principale | Backfill toggles granulaires, Discord webhook | — | 🟡 |
| Setup wizard | `setup_wizard.py` + 3 modules | `SetupPage.tsx` | Xbox OAuth + smoke test | Azure manuel, CSS custom avec progression | — | 🟢 — modernisé |
| Win/Loss | `win_loss.py` | — | — | **Page absente** | — | 🟠 |
| Objectifs | `objective_analysis.py` | — | — | **Page absente** | — | 🟠 |
| Changelog | — | `ChangelogPage.tsx` | Parse CHANGELOG.md | — | — | 🟢 — feature nouvelle |

**Résumé** : 14/16 pages Python migrées en React. 2 pages absentes (Win/Loss, Objectifs). Les pages migrées ont un niveau de richesse généralement inférieur au Streamlit (~70-80% des features), ce qui est attendu pour un MVP mais constitue une dette à combler.

### B.2 i18n React

| Aspect | Python/Streamlit | React | Classif |
|--------|-----------------|-------|:-------:|
| Langues | 14 langues (i18n complet via `src/ui/i18n/`) | Labels FR en dur, locale stockée mais pas de framework i18n | 🟠 |

---

## C. Parité algorithmes métier (7 cœurs)

| Algorithme | Python (module) | Go (package) | Golden values vertes ? | Écart observé | Classif |
|------------|-----------------|--------------|:----------------------:|---------------|:-------:|
| Performance score | `src/analysis/_performance_relative.py` + `_performance_session.py` | `internal/analysis/` + `internal/sync/performance.go` | ✅ 12 golden values | — | 🟢 |
| LUSR / CSR | `src/analysis/skill_rating.py` + `_calibration` + `_config` | `internal/sync/skill_rating.go` + `skill_rating_loaders.go` | ✅ golden values | — | 🟢 |
| Sessions | `src/analysis/sessions.py` (2 modes) | `internal/analysis/sessions*.go` (11 tests) | ✅ golden values | — | 🟢 |
| Citations | `src/analysis/citations/engine.py` + `custom_rules.py` | `internal/analysis/citations*.go` (9 tests) | ✅ golden values | — | 🟢 |
| Killer/victim | `src/analysis/killer_victim.py` | `internal/analysis/killer_victim.go` | ✅ | — | 🟢 |
| Weapon parser | `src/analysis/weapon_parser.py` | `internal/analysis/weapon_scanner.go` | ✅ | — | 🟢 |
| Spawn / comeback detection | `src/analysis/comeback_analysis.py` + `spawn_detection.py` | `internal/analysis/` | ✅ | — | 🟢 |

**7/7 algorithmes portés avec golden values.** Aucun écart de parité algorithmique.

---

## D. Parité sync / backfill

| Flux | Python | Go | Écart | Classif |
|------|--------|----|-------|:-------:|
| Sync delta | `src/data/sync/engine.py` (10 mixins, ~6 119L) | `internal/sync/engine.go` (~5 307L, 19 fichiers) | Parité fonctionnelle | 🟢 |
| Backfill (SyncScope) | `scripts/backfill_data.py` + `scripts/backfill/` | `internal/sync/backfill.go` + `backfill_cli.go` | Flags portés via `backfill_cli.go` (156L) | 🟢 |
| Write lease | `src/data/sync/` (sémantique ~5s timeout) | `internal/sync/` | Sémantique reproduite | 🟢 |
| Bitmask `backfill_completed` | 18 bits Python | Bits Go alignés (8 tests flags) | — | 🟢 |
| PvE Firefight | `src/data/sync/` | `internal/sync/pve.go` | — | 🟢 |

---

## E. Parité CLI / scripts opérationnels

| Script Python | Équivalent Go | Couvert ? | Écart | Classif |
|---------------|---------------|:---------:|-------|:-------:|
| `scripts/sync.py` | `POST /sync/initial` (async job) | ✅ | API-driven vs CLI — modernisation | 🟢 |
| `scripts/backup_player.py` | `internal/ops/archive.go` + `backup.go` | ✅ | — | 🟢 |
| `scripts/restore_player.py` | `internal/ops/restore.go` | ✅ | — | 🟢 |
| `scripts/backfill_data.py` | `internal/sync/backfill_cli.go` | ✅ | — | 🟢 |
| `scripts/check_env.py` | `internal/validation/gate.go` | ✅ | — | 🟢 |
| `scripts/healthcheck_db.py` | `internal/ops/healthcheck.go` | ✅ | — | 🟢 |
| `scripts/diagnose_player_db.py` | `internal/ops/diagnose.go` | ✅ | — | 🟢 |
| `scripts/post_sync_compute.py` | Intégré dans sync pipeline | ✅ | — | 🟢 |
| `scripts/populate_*.py` (6 scripts) | `internal/ops/seed.go` | ✅ | Consolidé en 1 fichier | 🟢 |
| `scripts/archive_season.py` | `internal/ops/archive.go` | ✅ | — | 🟢 |
| `scripts/index_media.py` | `POST /media/upload` | ✅ | API-driven | 🟢 |
| `scripts/monitor_uptime.py` | — | ❌ | Script de monitoring externe non porté | 🟡 |
| `scripts/generate_thumbnails.py` | — | ❌ | Script utilitaire non porté | 🟡 |
| `scripts/prepare_demo_data.py` | — | ❌ | Script utilitaire non porté | 🟡 |

---

## F. Parité données (schémas DuckDB)

| Table | Colonnes identiques ? | Types identiques ? | Index/vues identiques ? | Écart | Classif |
|-------|:---------------------:|:------------------:|:-----------------------:|-------|:-------:|
| `match_registry` | ✅ | ✅ | ✅ | — | 🟢 |
| `match_participants` | ✅ | ✅ | ✅ | — | 🟢 |
| `medals_earned` | ✅ | ✅ | ✅ | — | 🟢 |
| `killer_victim_pairs` | ✅ | ✅ | ✅ | — | 🟢 |
| `xuid_aliases` | ✅ | ✅ | ✅ | — | 🟢 |
| `weapon_kills` | ✅ | ✅ | ✅ | — | 🟢 |
| `highlight_events` | ✅ | ✅ | ✅ | — | 🟢 |
| `player_match_enrichment` | ✅ | ✅ | ✅ | — | 🟢 |
| `match_skill_rank` | ✅ | ✅ | ✅ | — | 🟢 |
| `sessions` | ✅ | ✅ | ✅ | — | 🟢 |
| `pve_match_stats` | ✅ | ✅ | ✅ | — | 🟢 |
| Vues `v_gamertag_lookup`, `v_match_full`, `v_killer_victim_full`, `v_weapon_kills` | ✅ | ✅ | ✅ | Go crée les mêmes vues via migrations | 🟢 |

---

## Tableau récapitulatif des écarts

| # | Zone | Description | Classif |
|--:|------|-------------|:-------:|
| 1 | Pages UI | Page **Win/Loss** absente de Go et React | 🟠 |
| 2 | Pages UI | Page **Objectifs** absente de Go et React | 🟠 |
| 3 | Pages UI | **Timeseries** React très simplifié vs les 5 onglets / ~20 charts Python | 🟠 |
| 4 | i18n React | Labels FR en dur, 14 langues Python non portées | 🟠 |
| 5 | Endpoint | `POST /settings/media/reset-index` — handler existe mais logique = TODO | 🟡 |
| 6 | Pages UI | Home — médias récents et activité session embed manquants | 🟡 |
| 7 | Pages UI | Career — projection Héros multi-joueurs manquante | 🟡 |
| 8 | Pages UI | Squad — radars synergie, heatmaps, trios, impact détaillé manquants | 🟡 |
| 9 | Pages UI | Session compare — auto-matching, historique sessions complet manquants | 🟡 |
| 10 | Pages UI | Explorer — badges encounter, deep linking manquants | 🟡 |
| 11 | Pages UI | Match view — timeline weapon kills, citations détaillées manquantes | 🟡 |
| 12 | Scripts | `monitor_uptime.py`, `generate_thumbnails.py`, `prepare_demo_data.py` non portés | 🟡 |
| 13 | API | Routes `match_exclusion` — features nouvelles Go, pas en Python | 🟢 |
| 14 | API | `POST /media/upload` — feature nouvelle Go | 🟢 |
| 15 | UI | `ChangelogPage.tsx` — feature nouvelle React | 🟢 |
| 16 | Architecture | Setup wizard simplifié (Xbox OAuth seul, pas Azure manuel) | 🟢 — modernisation voulue |

---

**Fin de la review Claude — Axe 1.**
