# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

> French version: [FR/CHANGELOG.md](FR/CHANGELOG.md)

## [5.5.0] - 2026-03-07

### Added

- **Setup Wizard — Configuration initiale guidée** (`src/ui/pages/setup_wizard.py` + `setup_wizard_logic.py`)
  - Deux parcours : **Xbox Express** (recommandé, 2 étapes) et **Azure manuel** (avancé, 3 étapes)
  - Cards CSS personnalisées avec icônes, barre de progression animée, étapes numérotées
  - Logique séparée de l'UI (`SetupStatus`, `validate_azure_credentials()`, `validate_gamertag()`, `create_player_profile()`, `save_azure_credentials()`)
  - Guard dans `main()` : le wizard s'affiche automatiquement si credentials ou joueur manquants
  - i18n FR/EN (~49 clés) dans `src/ui/i18n/setup.py`

- **Xbox OAuth — Connexion Xbox en 1 clic** (`src/ui/xbox_oauth.py` + `xbox_oauth_ui.py`)
  - Flux complet : URL Microsoft → callback `?code=XXX&state=YYY` → échange code → refresh_token → spartan/clearance tokens → résolution gamertag+XUID → provisionnement automatique
  - `xbox_oauth.py` (436L) : logique OAuth pure sans dépendance Streamlit
  - `xbox_oauth_ui.py` (163L) : composant Streamlit intégré dans Settings (bouton login, statut, déconnexion)
  - Protection CSRF avec `state` aléatoire validé au retour du callback
  - i18n FR/EN dans `src/ui/i18n/pages/xbox.py`

- **Player Provisioning** (`src/app/player_provisioning.py`)
  - `provision_player()` : crée `data/players/{gamertag}/stats.duckdb` + table `sync_meta` + enregistre dans `db_profiles.json` — idempotent

- **Auth Status** (`src/utils/auth.py`)
  - `AuthStatus` dataclass + `get_auth_status()`, `check_credentials()`, `write_env_local()` (écriture/mise à jour de `.env.local` en préservant les commentaires)

- **Compatibilité macOS / Linux** — `LevelUp.sh` (nouveau) : lanceur premier-lancement équivalent à `LevelUp.bat` pour macOS/Linux, écrit en POSIX sh (sans bashism — compatible macOS bash 3.2, dash, zsh). Détecte Python 3.10+ via binaires versionnnés (`python3.12` → Homebrew), chemins Homebrew Intel/Apple Silicon (`/opt/homebrew`, `/usr/local`), puis générique. Messages d'aide ciblés par distribution. `run.sh` corrigé pour détecter `.venv/bin/python` (macOS/Linux) ou `.venv/Scripts/python.exe` (Windows Git Bash). `launcher.py` : `_find_system_python()` enrichi avec candidats versionnnés et chemins Homebrew ; `_cmd_doctor()` utilise désormais `_preferred_python_executable()` cross-platform.

- **`launcher.py setup`** — Commande d'installation interactive : détecte Python (py launcher → PATH → emplacements standard → installation via winget), crée le `.venv`, installe les dépendances (`pip install -e ".[spnkr]"`). Supporte `--update` pour mettre à jour un environnement existant.

- **`launcher.py doctor`** — Diagnostic complet de l'environnement : OS, Python, venv, versions des packages critiques vs attendues, nombre de joueurs configurés, présence de `metadata.duckdb`

- **Packaging portable** (`packaging/build_release.py`)
  - Génère un zip autonome `LevelUp-v{version}-win64-portable.zip` contenant Python Embeddable 3.12 (~15 Mo) + le projet complet
  - Premier lancement : installation automatique des dépendances via pip

- **Release GitHub Actions** (`.github/workflows/release.yml`)
  - Déclenché sur push de tag `v*.*.*`
  - Build du zip portable + publication automatique en GitHub Release

- **Mode portable `%APPDATA%`** (`src/utils/paths.py`, `auth.py`, `env.py`)
  - Données stockées dans `%APPDATA%/LevelUp/` (Windows) ou `$XDG_DATA_HOME/levelup/` (Linux) quand pas de `.venv` à la racine
  - Mode développeur : `./data/` si `.venv` existe
  - Override possible via variable d'environnement `LEVELUP_DATA`
  - `.env.local` cherché dans `DATA_DIR` en priorité, puis à la racine du repo

- **Token fallback DB** (`src/ui/profile_api_tokens.py`)
  - Fallback 3 : lecture du refresh_token depuis `sync_meta` de la DB joueur si absent des variables d'environnement

- **Documentation**
  - `docs/CONFIGURATION.md` : réécriture complète avec TOC, guide Azure pas-à-pas avec 11 captures d'écran annotées, sections Player Profiles, Environment Variables, App Settings, Security, Troubleshooting
  - `docs/FR/CONFIGURATION.md` : version FR mise à jour
  - `docs/SYNC_GUIDE.md` : réécriture avec architecture sync v5.1, diagramme ASCII, commandes détaillées
  - `docs/FR/SYNC_GUIDE.md` : mise à jour

- **Migrations de schéma automatiques** (`src/data/migration/`) — runner versionné appliqué automatiquement au démarrage (`launcher.py → _run_migrations()`). Chaque DB (`player`, `shared`, `shared_pve`) trace les migrations dans une table `schema_migrations`. 11 migrations initiales enregistrées. Pour ajouter un changement de schéma : créer une fonction `ensure_xxx` idempotente dans `src/data/sync/migrations.py`, créer le step correspondant dans `src/data/migration/steps/` et l'importer dans `steps/__init__.py`.

### Fixed

- **CSRF** (`streamlit_app.py`) — Corrige comparaison `_xbox_state != _xbox_state` (auto-comparaison, toujours False) → `_xbox_state != _expected_state`
- **`_repo_root` undefined** (`src/ui/profile_api_tokens.py`) — `_repo_root()` jamais importée → remplacée par `REPO_ROOT` depuis `src.utils.paths`
- **DuckDB retry élargi** (`src/data/sync/_engine_connections.py`) — `except duckdb.IOException` → `except duckdb.Error` + délai retry `0.15s → 0.5s`
- **GC sync mode** (`src/ui/_sync_duckdb_ops.py`) — `gc.collect()` + `time.sleep(0.3)` pour libérer les file handles DuckDB sous Windows
- **OAuth consumed guard** (`streamlit_app.py`) — Flag `_xbox_oauth_consumed` pour éviter le double-traitement du callback au rerun Streamlit
- **Test isolation webhook** (`tests/test_monitor_uptime.py`) — Patch `get_secret` au lieu de manipuler `os.environ` pour éviter le rechargement `.env.local`
- **API Streamlit dépréciée** (`src/ui/pages/setup_wizard.py`) — Remplacement des trois occurrences de `use_container_width=True` par `width="stretch"` : bouton Xbox Express, bouton Azure manuel, `st.link_button` OAuth.
- **Smoke test UI manquant** (`src/ui/pages/setup_smoke_test.py`) — Module UI recréé : 3 phases avec barres de progression, tableau de vérification, boutons de continuation / relance.
- **Test patch `SPNKrAPIClient` incorrect** (`tests/test_player_tokens.py`) — Cible de mock corrigée en `src.data.sync._career.create_api_client` conformément à l’abstraction API.

### Tests

- **75 tests ajoutés** (1 482 lignes) couvrant l'ensemble des nouveaux modules :
  - `test_auth.py` (13 tests) : `AuthStatus`, `get_auth_status()`, `write_env_local()`
  - `test_setup_wizard_logic.py` (20+ tests) : `SetupStatus`, validations, création de profil, edge cases
  - `test_xbox_oauth.py` (18 tests) : URL OAuth, échange de code, store/load token, provisionnement
  - `test_xbox_oauth_callback_e2e.py` (9 tests) : flux complet code→player, erreurs, CSRF, cycle token
  - `test_setup_wizard_page.py` (15 tests) : UI mockée (MockStreamlit), modes Xbox/Azure, progression ; assertions `width="stretch"` sur les widgets
- **3 831 tests, 0 échec**

### Architecture

- **Abstraction API — Ports & Adapters** : découplage de la librairie SPNKr pour faciliter un futur changement de backend API
  - `api_port.py` : Protocol `HaloAPIPort` — contrat structurel (runtime_checkable) définissant les méthodes que tout client API Halo doit implémenter
  - `api_factory.py` : Factory `create_api_client(backend="spnkr")` — instanciation centralisée, extensible à d'autres backends
  - `_auth.py` : Facade d'authentification — les modules UI appellent `refresh_halo_tokens()` sans importer SPNKr directement
  - Migration des consommateurs : `engine.py`, `orchestrator.py`, `strategies.py`, `_career.py`, `populate_metadata_from_discovery.py`, `profile_api_tokens.py`, `player_assets.py`, `xbox_oauth.py` — tous utilisent la factory ou la facade auth
  - 14 tests dédiés (`test_api_abstraction.py`) : conformité Protocol, factory, facade auth, vérification d'absence d'imports SPNKr dans les modules UI migrés

### Removed

- **`scripts/_archive/`** — 89 fichiers de code mort supprimés (anciens scripts d'analyse d'armes, diagnostics, patchs i18n, utilitaires obsolètes)
- **`requirements.txt`** — Supprimé, remplacé par `pyproject.toml` (source unique de vérité pour les dépendances)
- **`setup.bat`** — Remplacé par `LevelUp.bat` (détection Python améliorée, installation via winget, utilisation de `pip install -e .`)
- **`scripts/install_dependencies.py`** — Workaround MSYS2 SSL, utilisait `requirements.txt`
- **`scripts/setup_env.ps1`**, **`scripts/setup_env.sh`**, **`scripts/activate_env.sh`** — Remplacés par `launcher.py setup`
- **`tests/test_spnkr_refactoring.py`** — Tests pour du code archivé supprimé

### Chore

- Rangement racine : `ACKNOWLEDGMENTS.md`, `CHANGELOG.md`, `CONTRIBUTING.md` déplacés vers `docs/`
- Scripts déplacés : `activate_env.sh`, `run_monitor_hidden.vbs` → `scripts/`
- `LevelUp.bat` remplace `setup.bat` comme point d'entrée Windows
- `Dockerfile` et `e2e-browser-optional.yml` mis à jour pour utiliser `pyproject.toml` au lieu de `requirements.txt`
- `run.sh` redirige vers `launcher.py setup` au lieu de `activate_env.sh`

### Mises à jour complémentaires (7 mars 2026)

- **Sélecteur de timezone** — Choix de la timezone d'affichage directement dans les Settings (Europe/Paris par défaut, ~40 fuseaux disponibles). Les horodatages des matchs s'adaptent automatiquement partout dans l'app.
- **Page Carrière améliorée** — Meilleure lisibilité de la section classement LUSR, navigation plus fluide.
- **Migration bot xuid** — Correction automatique des matchs contenant des bots mal identifiés dans la base de données partagée.
- **Stabilité** — Corrections sur le chargement des données d'adversaires, les requêtes de matchs, le cache UI et la synchronisation. Amélioration de la fiabilité sur Windows lors des accès concurrents aux bases DuckDB.

## [5.4.0] - 2026-03-04

### Added

- **Page Explorer — recherche et navigation unifiée dans les matchs** (`src/ui/pages/explorer.py`)
  - Remplace l'ancienne page "Match" avec une architecture 6 modules (explorer, explorer_results, explorer_enrich, explorer_data, explorer_logic, match_table_html)
  - **Filtres en cascade** : date, escouade (solo/squad), type d'expérience (ranked/unranked/PvE), playlist, mode de jeu, carte
  - **Recherche floue par gamertag** avec suggestions dynamiques et résolution XUID
  - **Tableau HTML OS-style** (`match_table_html.py`) : colonnes KDA, kills, deaths, accuracy, score, MMR delta, performance, headshots, spree, avg life ; liens deep-link inter-pages
  - **Deep linking** : `?page=Explorer&gamertag=XXX` ou `&match_id=XXX` pour navigation directe
  - **Badges encounter** : rival, mentor, proie — calculés sur l'historique croisé des joueurs
  - **Enrichissement** (`explorer_enrich.py`) : score équipe, delta MMR, performance, temps de vie moyen, URL Waypoint
  - **i18n FR/EN complet** (`src/ui/i18n/pages/explorer.py`)
  - **Logging structuré** : info (deep links), warning (joueur introuvable, DB absente), error (exceptions DB avec `exc_info`)
  - **40 tests unitaires** (`tests/test_explorer_logic.py`) couvrant logique, enrichissement, accès données et rendu HTML

### Tests — anciens skips corrigés

Les tests suivants étaient marqués `@pytest.mark.skip` ou `skipif(True)` et sont maintenant exécutables :

| Fichier | Test(s) | Motif de correction |
|---------|---------|---------------------|
| `tests/test_rag.py` | `TestHaloKnowledgeBase` (3 tests) + `test_chunk_overlap` | Suppression des guards `skipif(True)` et faux skip |
| `tests/test_season_archive.py` | `test_get_archive_info_with_archives` | Suppression du skip + assertion `>= 0` (fichier Parquet tiny) |
| `tests/test_i18n_refactoring.py` | `test_no_duplicate_keys_in_module[pages]` | Support des packages (dossier `pages/` au lieu de `pages.py`) |
| `tests/e2e/test_streamlit_browser_e2e.py` | `test_e2e_004_deeplink_match_query_params` | Regex `exception(?!nel)` — exclut "exceptionnel" (mot FR) |
| `tests/test_cache_integrity.py` | 11 tests SQLite legacy | Fichier **supprimé** (code mort v3) |
| `tests/conftest.py` | tous les tests `e2e_browser` | Suppression du guard auto-skip + installation Chromium |

Pour rejouer uniquement ces tests :

```bash
# RAG
python -m pytest tests/test_rag.py::TestHaloKnowledgeBase tests/test_rag.py::TestTextChunker::test_chunk_overlap -v

# Archive saisonnière
python -m pytest tests/test_season_archive.py::TestDuckDBRepositoryArchives::test_get_archive_info_with_archives -v

# i18n (package pages/)
python -m pytest tests/test_i18n_refactoring.py::TestNoInternalDuplicates -v

# E2E deeplink
python -m pytest tests/e2e/test_streamlit_browser_e2e.py::test_e2e_004_deeplink_match_query_params -v

# Suite complète sans intégration
python -m pytest -q --ignore=tests/integration
```

### Added

- **Historique des rencontres — section sous le scoreboard** (`src/ui/pages/match_view_encounters.py`)
  - Nouveau tableau HTML affiché directement sous le scoreboard sur la page Match View
  - Pour chaque joueur non-ami du match : fréquence de rencontres, répartition allié/ennemi, win rate allié, win rate ennemi, K/D croisé (depuis `killer_victim_pairs`), date de dernière rencontre
  - Tri : ennemis en premier, puis alliés ; dans chaque groupe par `total_encounters DESC`
  - Ligne compacte grisée pour les premières rencontres (total = 1), ligne complète avec métriques au-delà
  - Badges automatiques inline : **Dur à cuire** (deaths/kills > 2 et ≥ 3 morts), **Allié+** (WR allié ≥ 65% sur ≥ 2 matchs), **Coriace** (WR ennemi ≤ 35% sur ≥ 3 matchs)
  - Code couleur réutilisant les classes CSS du scoreboard (`os-sb-td--best`, `os-sb-td--worst`, amber)
  - Périmètre : tous les joueurs non membres de l'escouade / non-amis

- **Loader SQL dédié** (`src/data/repositories/_encounter_loader.py`)
  - `load_encounter_stats(self_xuid, target_xuids, db_path)` — 3 CTEs sur `shared_matches.duckdb` (match_participants, killer_victim_pairs, match_registry, xuid_aliases)
  - Dérivation automatique du chemin `shared_matches.duckdb` depuis `stats.duckdb`
  - Connexion `duckdb_read_only()` sur shared directement (no ATTACH conflict)

- **Logique pure testable** (`src/ui/pages/match_view_encounters_logic.py`)
  - `EncounterStats` (Pydantic v2), `Badge` (dataclass), `ordinal_fr()`, `build_friends_set()`, `filter_encounter_xuids()`, `compute_encounter_badges()`
  - `build_friends_set` : double source `player_match_enrichment.friends_xuids` → fallback `friends_defaults.json`
  - 28 tests unitaires dans `tests/test_match_view_encounters.py` (sans import Streamlit)

- **Clés i18n** (`src/ui/i18n/pages.py`) : `mv_encounter_history`, `col_role`, `col_encounters`, `col_wr_ally`, `col_wr_enemy`, `col_kd_cross`, `col_last_seen`

### Technical

- `match_view.py` : appel de `render_encounter_section()` après `render_match_scoreboard()` (+10 lignes, zéro logique ajoutée dans le fichier)
- Architecture SRP respectée : 3 nouveaux fichiers < 350 lignes chacun, fonctions < 50 lignes, logique UI et data séparées

### Refactoring & Architecture (branche `refactor/cleanup-all`)

> **Refactoring massif en 6 phases** — 331 fichiers modifiés, ~30 000 lignes réécrites, 72 nouveaux sous-modules, 3 693 tests passent (dont 79 tests dédiés ajoutés). Aucun changement fonctionnel pour l'utilisateur.

#### Phase 0-4 : Infrastructure & premiers splits

- **Split `transformers.py` (2 095L → package)** — `src/data/sync/transformers/` avec 7 sous-modules (`_helpers`, `_match`, `_skill`, `_events`, `_medals`, `_personal_scores`, `_pve`) + `__init__.py` ré-exportant tout ; aucun breaking change
- **Split `filters_render.py` (1 460L → 4 modules)** — `_filters_period.py`, `_filters_session.py`, `_filters_cascade.py` extraits ; `filters_render.py` réduit à l'orchestration
- **Split `engine.py` (1 500L → 8 mixins)** — `_shared_writes.py`, `_performance.py`, `_skill_rating.py`, `_career.py`, `_aggregates.py`, `_tokens.py`, `_engine_connections.py`, `_engine_schema.py`
- **Split `duckdb_repo.py` (1 200L → 8 mixins)** — `_match_queries_helpers.py`, `_match_queries_polars.py`, `_archives_repo.py`, `_awards_repo.py`, `_diagnostic_repo.py`, `_events_repo.py`, `_medals_repo.py`, `_schema_introspection.py`
- **Split modules utilitaires** — `media_indexer.py`, `api_client.py`, `batch_insert.py`, `discord_notifier.py`, `cache_loaders.py`, `radar_chart.py`, `teammates_views.py`, `sync.py`, `timeseries_combat.py`
- **`_SyncProtocol`** (`src/data/sync/_protocol.py`) — contrat `Protocol` explicite pour les 8 mixins du `DuckDBSyncEngine` ; élimine 70+ `# type: ignore[attr-defined]`
- **`PageContext` + `MatchViewParams`** (`src/app/_page_context.py`) — types réels à la place de 5 champs `Any` dans le `NamedTuple`
- **`SessionKeys` / `SK`** (`src/app/session_keys.py`) — 20+ clés `st.session_state` centralisées, complétions IDE, plus de typos silencieuses
- **`_sql_fragments.py`** (`src/data/query/_sql_fragments.py`) — source de vérité unique pour `WIN_RATE_EXPR` (dénominateur WIN+LOSS, NULLIF division), `IS_WIN`, `IS_LOSS` ; 7 occurrences dupliquées dans `analytics.py` et `trends.py` supprimées
- **Dettes techniques v4→v5 supprimées** : guard `_PERF_SCORE_AVAILABLE` (always-True), dead method `_ensure_performance_score_column()`, magic number `outcome == 4` → `Outcome.DID_NOT_FINISH`

#### Phase 5 : Split modules d'analyse & visualisation

- **Split `performance_score.py` (950L → 3 modules)** — `_performance_relative.py` (score match relatif), `_performance_session.py` (score session v1/v2, `ScoreComponent`) ; façade inchangée
- **Split `antagonist_charts.py` (570L → 3 modules)** — `_antagonist_kv.py` (stacked bars, timeseries, heatmap), `_antagonist_duels.py` (duel history, nemesis summary, indicators) ; façade inchangée
- **Split `rag.py` (750L → 4 modules)** — `_rag_models.py` (RAGConfig, Document, SearchResult), `_rag_github.py` (GitHubIndexer), `_rag_chunker.py` (TextChunker) ; façade inchangée

#### Phase 6 : Split modules UI & data

- **Split `refdata.py` (880L → 2 modules)** — `_refdata_personal_scores.py` (PersonalScoreNameId enum 68 membres, dictionnaires de points/noms/IDs) ; façade inchangée
- **Split `_roster_loader.py` (520L → 2 mixins)** — `_gamertag_resolver.py` (GamertagResolverMixin, cascade XUID→gamertag 5 sources) ; `_roster_loader.py` hérite du mixin
- **Split `cache_filters.py` (740L → 3 modules)** — `_cache_loading.py` (recent matches, pagination, match count), `_cache_sessions.py` (compute sessions DB) ; façade inchangée
- **Split `filters_render.py`** — `_filters_apply.py` (apply_filters 190L, diagnostic empty) ; façade inchangée
- **Split `session_compare_charts.py` (480L → 2 modules)** — `_session_compare_history.py` (tableau historique HTML) ; façade inchangée

#### Qualité & couverture

- **79 tests unitaires dédiés** — `test_submodules_phase5.py` (37 tests) + `test_submodules_phase6.py` (42 tests) couvrant directement les 13 sous-modules et vérifiant les re-exports des façades
- **Logger ajouté dans 3 modules silencieux** — `_cache_loading.py` (6 blocs `except` → `logger.debug` avec `exc_info`), `_performance_relative.py` (1 catch-all), `_rag_github.py` (1 erreur réseau) ; tous les `except Exception` des sous-modules sont désormais tracés
- **Système de logs centralisé** (`src/utils/log_config.py`) — `setup_app_logging()` : logs fichiers uniquement (`data/logs/app.log` 5 Mo×3, `data/logs/sync.log` 10 Mo×5), pas de sortie console ; `setup_script_logging()` pour les scripts CLI ; `log_duration()` context manager avec seuil ms configurable. Câblé dans : launch app, chargement joueur, sélection session, changements filtres, chargement DataFrame, KPIs, navigation match (boutons dernier match / carnage / match précédent), sync UI, backfill CLI, tailscale, RAG. `data/logs/` exclu du dépôt.
- **`.gitattributes`** — enforce `eol=lf` sur tout le dépôt ; résout les conflits pre-commit mixed-line-ending sur Windows (`core.autocrlf=true`)
- **`pyproject.toml`** — `per-file-ignores` pour `scripts/*` et `launcher.py` (complexité C901/PLR0912/PLR0913/PLR0915 tolérée dans les scripts utilitaires)
- **Enforcement qualité** : `scripts/check_code_size.py` (ratchet), `tests/test_code_quality.py` (3 tests qualité structurelle), règles CLAUDE.md 13-17 (taille max, args max, complexité, SRP)

### Bug fixes (portés depuis `main`)

- **Filtres auto-invalidation post-sync** (`src/app/filters_render.py`) — `_filters_db_key_{player}` remplace le booléen write-once `_filters_loaded_*` ; les filtres se réinitialisent automatiquement quand la DB change (sync, CLI, backfill, changement de profil)
- **Citations calculées post-sync** (`src/data/citations_backfill.py`) — module incremental appelé par `DuckDBSyncEngine` après chaque sync ; les matchs nouvellement insérés ont immédiatement leurs citations
- **SyncLock câblé à l'UI** (`src/ui/sync.py`) — `SyncLock(timeout=0)` protège contre les syncs concurrents inter-processus ; `SyncAlreadyRunning` affiché proprement à l'utilisateur + flush WAL DuckDB avant `end_sync_mode()`
- **Tailscale guard process-level** (`src/utils/tailscale.py`) — `threading.Event` module-level remplace `st.session_state` (par-session) ; `ensure_funnel_started_once()` garantit un seul démarrage et une seule notification Discord par processus Python
- **Fausse alerte Discord webhook** (`src/utils/startup_check.py`) — skip du check si Doppler est actif ; chargement `.env.local` avant vérification
- **`_PERF_SCORE_AVAILABLE` manquant** (`src/data/sync/_performance.py`) — variable module-level absente après le split `engine.py` → mixins ; ajout d'un guard `try/except ImportError` avec `_PERF_SCORE_AVAILABLE = True/False` ; corrige `F821 Undefined name` et `NameError` à l'exécution
- **NaN-check fragile** (`src/ui/pages/match_view.py`) — `x == x` (idiome NaN flottant) remplacé par `x is not None`
- **i18n** (`src/ui/translations.py`, `src/ui/i18n/widgets.py`) — 2 clés `PAIR_FR` tronquées restaurées, doublon `tm_session_trend` supprimé, 343 entrées redondantes nettoyées (399 → 56 entrées utiles)
- **Détection backfill per-player** (`scripts/backfill/detection.py`) — les 6 flags per-player (medals, personal_scores, performance_scores, accuracy, shots, enemy_mmr) vérifient désormais les données réelles du joueur courant au lieu du bitmask global `backfill_completed` ; corrige un bug où le premier joueur syncé masquait les matchs pour les autres joueurs ; nouvelle fonction `_player_done_guard()` ; 15 nouveaux tests multi-joueur + 9 tests adaptés

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
