# Journal des modifications

> Version française du [CHANGELOG.md](../../CHANGELOG.md) racine.

Toutes les modifications notables de ce projet sont documentées ici.

Le format est basé sur [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/).

## [5.4.0] - 2026-03-04

### Ajouté

- **Page Explorer — recherche et navigation unifiée dans les matchs** (`src/ui/pages/explorer.py`)
  - Remplace l'ancienne page "Match" avec 6 modules (explorer, explorer_results, explorer_enrich, explorer_data, explorer_logic, match_table_html)
  - Filtres en cascade : date, escouade, type, playlist, mode, carte
  - Recherche floue par gamertag avec suggestions et résolution XUID
  - Tableau HTML OS-style avec KDA, MMR delta, performance, liens deep-link
  - Deep linking : `?page=Explorer&gamertag=XXX` ou `&match_id=XXX`
  - Badges encounter : rival, mentor, proie
  - i18n FR/EN complet + logging structuré + 40 tests unitaires

- **Historique des rencontres — section sous le scoreboard** (`src/ui/pages/match_view_encounters.py`)
  - Tableau HTML affiché sous le scoreboard sur la page Vue Match
  - Par joueur non-ami : fréquence de rencontres, répartition allié/ennemi, win rate allié/ennemi, K/D croisé, date de dernière rencontre
  - Tri : ennemis en premier, puis alliés ; dans chaque groupe par `total_encounters DESC`
  - Badges automatiques : **Dur à cuire**, **Allié+**, **Coriace**

- **Loader SQL dédié** (`src/data/repositories/_encounter_loader.py`)
  - `load_encounter_stats()` — 3 CTEs sur `shared_matches.duckdb`

- **Logique pure testable** (`src/ui/pages/match_view_encounters_logic.py`)
  - `EncounterStats` (Pydantic v2), `Badge`, `build_friends_set()`, `compute_encounter_badges()`
  - 28 tests unitaires dans `tests/test_match_view_encounters.py`

### Refactoring & Architecture

- **Split `transformers.py` (2 095L → package)** — `src/data/sync/transformers/` avec 7 sous-modules (`_helpers`, `_match`, `_skill`, `_events`, `_medals`, `_personal_scores`, `_pve`) + `__init__.py` ré-exportant tout ; aucun breaking change
- **Split `filters_render.py` (1 460L → 4 modules)** — `_filters_period.py`, `_filters_session.py`, `_filters_cascade.py` extraits
- **`_SyncProtocol`** (`src/data/sync/_protocol.py`) — contrat `Protocol` explicite pour les 8 mixins du `DuckDBSyncEngine` ; élimine 70+ `# type: ignore[attr-defined]`
- **`PageContext` + `MatchViewParams`** (`src/app/_page_context.py`) — types réels à la place de 5 champs `Any`
- **`SessionKeys` / `SK`** (`src/app/session_keys.py`) — 20+ clés `st.session_state` centralisées
- **`_sql_fragments.py`** (`src/data/query/_sql_fragments.py`) — source de vérité unique pour `WIN_RATE_EXPR` (dénominateur WIN+LOSS, NULLIF division) ; 7 duplications supprimées dans `analytics.py` et `trends.py`
- **Dettes techniques v4→v5 supprimées** : guard `_PERF_SCORE_AVAILABLE` (always-True), dead method `_ensure_performance_score_column()`, magic number `outcome == 4` → `Outcome.DID_NOT_FINISH`
- **Système de logs centralisé** (`src/utils/log_config.py`) — `setup_app_logging()` : logs fichiers uniquement (`data/logs/app.log` 5 Mo×3, `data/logs/sync.log` 10 Mo×5), aucune sortie console ; `setup_script_logging()` pour les scripts CLI ; `log_duration()` context manager avec seuil ms configurable. Câblé dans : lancement app, chargement joueur, sélection session, changements filtres, chargement DataFrame, KPIs, navigation match (boutons dernier match / carnage / match précédent), sync UI, backfill CLI, tailscale, RAG. `data/logs/` exclu du dépôt.
- **`.gitattributes`** — enforce `eol=lf` sur tout le dépôt ; résout les conflits pre-commit mixed-line-ending sur Windows
- **`pyproject.toml`** — `per-file-ignores` pour `scripts/*` et `launcher.py` (complexité tolérée dans les scripts utilitaires)
- **Enforcement qualité** : `scripts/check_code_size.py` (ratchet 247 violations connues), `tests/test_code_quality.py` (3 tests qualité structurelle), règles CLAUDE.md 13-17

### Corrections de bugs

- **Filtres auto-invalidation post-sync** — `_filters_db_key_{player}` remplace le booléen write-once ; les filtres se réinitialisent quand la DB change (sync, CLI, backfill, changement de profil)
- **Citations calculées post-sync** (`src/data/citations_backfill.py`) — module incrémental appelé après chaque sync ; plus de matchs sans citations
- **SyncLock câblé à l'UI** — `SyncLock(timeout=0)` protège contre les syncs concurrents inter-processus ; flush WAL DuckDB avant `end_sync_mode()`
- **Tailscale guard process-level** — `threading.Event` module-level remplace `st.session_state` ; une seule notification Discord par démarrage de processus
- **Fausse alerte Discord webhook** — skip du check si Doppler est actif ; chargement `.env.local` avant vérification
- **`_PERF_SCORE_AVAILABLE` manquant** (`src/data/sync/_performance.py`) — variable absente après le split en mixins ; guard `try/except ImportError` ajouté ; corrige `NameError` à l'exécution
- **NaN-check fragile** (`match_view.py`) — `x == x` (idiome NaN flottant) → `x is not None`
- **i18n** — 2 clés `PAIR_FR` tronquées restaurées, doublon `tm_session_trend` supprimé, 343 entrées redondantes nettoyées (399 → 56 entrées utiles)
- **Détection backfill per-player** (`detection.py`) — les 6 flags per-player (medals, personal_scores, performance_scores, accuracy, shots, enemy_mmr) vérifient les données réelles du joueur au lieu du bitmask global `backfill_completed` ; corrige un masquage entre joueurs lors du premier sync ; `_player_done_guard()` + 15 tests multi-joueur + 9 tests adaptés

---

## [5.3.0] - Non publié

### Ajouté

- **LUSR (LevelUp Skill Rank) — Système de rating TrueSkill 2 per-groupe** (`src/analysis/`)
  - `skill_rating_config.py` : constantes TrueSkill 2, tiers Bronze→Onyx I-VI, score composite 5 composantes
  - `playlist_groups.py` : 6 groupes Halo Infinite isolés (ranked 1.00, arena 0.80, tactical 0.70, btb 0.60, social 0.40, fun 0.15) avec détection par `pair_name` prefix ou `playlist_name`
  - `skill_rating.py` : algorithme complet — `PlayerState` par groupe, `compute_composite_score()`, `trueskill_update()`, `compute_enemy_strength()`, inactivité par groupe, `compute_skill_ratings_batch()` séquentiel
  - `skill_rating_calibration.py` : module de calibration des poids COMPOSITE_WEIGHTS par comparaison avec `team_mmr` API (grid search aléatoire, métrique MAE ou corrélation Pearson)
  - 68 tests unitaires couvrant l'algorithme, les groupes, l'inactivité, les tiers et la calibration

- **LUSR per-groupe : état TrueSkill indépendant par contexte**
  - `existing_states: dict[str, PlayerState]` remplace `existing_state: PlayerState` — un match ranked n'affecte plus le rating arena
  - `states.setdefault(group, PlayerState())` crée un état au premier match de chaque groupe
  - Inactivité, historique précision et σ decay sont désormais par-groupe

- **Backfill LUSR/CSR** (`scripts/backfill_data.py`, `scripts/backfill/`)
  - `--lusr` / `--force-lusr` : calcul local du LUSR depuis `shared.match_participants` (séquentiel, incrémental)
  - `--csr` / `--force-csr` : récupération CSR depuis l'API Halo pour les matchs ranked
  - `compute_lusr_for_player()` dans `strategies.py` : UPSERT dans `match_skill_rank` avec `rating_delta`, tier et tier_label
  - Table `match_skill_rank` créée automatiquement par `ensure_match_skill_rank_table()` dans `migrations.py`
  - Bits backfill : `lusr = 1 << 16` (65536), `csr = 1 << 17` (131072) dans `BACKFILL_FLAGS`

- **Flags SyncScope** : `lusr`, `force_lusr`, `csr`, `force_csr` dans `src/data/sync/scope.py`

- **Modèle de données CSR** (`src/data/sync/models.py`, `src/data/sync/transformers.py`)
  - `SkillParticipantUpdate` étendu : `pre_match_csr`, `post_match_csr`, `csr_tier`, `csr_sub_tier`
  - Extraction `RankRecap.PreMatchCsr` / `PostMatchCsr` dans `transform_all_skill_stats()`

- **Visualisation LUSR** (`src/visualization/timeseries_combat.py`)
  - `plot_lusr_timeseries()` : zones de tier semi-transparentes, bande de confiance `rating ± deviation`, tendance lissée 20 matchs

- **UI — Page Carrière et Vue Match** (`src/ui/pages/`)
  - `career.py` : cartes visuelles par groupe (image rang 90px centrée, badge LUSR/CSR, delta ▲/▼) + sélecteur de groupe (`st.selectbox`) pour le graphe d'évolution — remplace le tableau en expander et les onglets
  - `match_view.py` : onglet 🏅 Rang avec badge rang, barre de progression colorée, delta vert/rouge

- **Calibration CLI**
  - `python -m src.analysis.skill_rating_calibration --player <GT> [--n-samples 300] [--metric corr]`
  - Grid search sur le simplexe des poids (distribution Dirichlet uniforme, graine reproductible)
  - Affiche les poids optimaux prêts à copier dans `skill_rating_config.py`

- **Notifications Discord post-sync/backfill** (`src/utils/discord_notifier.py`)
  - Nouveau module failsafe — aucune dépendance externe (stdlib `urllib.request` uniquement)
  - Envoi d'un Rich Embed Discord à la fin de chaque `sync.py` et `backfill_data.py`
  - Contenu de l'embed : opération, heure début/fin, durée totale, nombre de joueurs et matchs traités
  - Par joueur : matchs synchronisés (ou traités par backfill), complétude des données (médailles + events), dernier match (carte, mode, FDA, résultat, playlist)
  - Couleur de la barre : vert ✅ (tout OK), jaune ⚠️ (données incomplètes), rouge ❌ (erreur)
  - Flag `--no-discord` sur `sync.py` et `backfill_data.py` pour désactiver ponctuellement
  - `notify_operation_done()` : entrypoint public — `disabled=True` court-circuite immédiatement
  - `fetch_last_match_info(xuid)` : SQL sur `shared_matches.duckdb` (JOIN `match_registry` + `match_participants`)
  - `count_new_matches(xuid, gamertag, since)` : compte les matchs avec `first_sync_at >= since`
  - `count_matches_missing_data(xuid)` : compte les matchs avec `medals_loaded=FALSE OR events_loaded=FALSE`

- **Configuration webhook Discord sécurisée**
  - Toggle `discord_notifications_enabled: false` dans `app_settings.json` (pas de secrets dans ce fichier)
  - URL webhook lue depuis `DISCORD_WEBHOOK_URL` dans `.env.local` (gitignored) via `_load_dotenv_if_present()`
  - Fallback rétrocompatible sur la clé `discord_webhook_url` dans `app_settings.json`
  - Section documentée dans `.env.local.example`

- **Internationalisation FR/EN complète (i18n)** (`src/ui/i18n/`)
  - Package i18n dédié avec modules spécialisés : `common.py`, `pages.py`, `widgets.py`, `viz.py`, `cli.py`
  - Fonctions : `t(key, lang=None)` (UI Streamlit), `viz_t(key, lang)` (Plotly), `discord_t(key, **kwargs)` (Discord), `ct(key, **kwargs)` (CLI/scripts)
  - Langue stockée dans `st.session_state["lang"]` (Streamlit) ou variable d'env `LEVELUP_LANG` (scripts)
  - Sélecteur de langue 🇫🇷/🇬🇧 dans la sidebar (`_render_lang_selector()` dans `src/app/sidebar.py`)
  - Trois champs dans `AppSettings` : `lang`, `discord_lang`, `cli_lang` (défaut `"fr"`)
  - `src/ui/translations.py` bilingue : `translate_playlist_name(name, lang)` et `translate_pair_name(name, lang)` — conserve le regroupement `" on Map"` et les préfixes Halo (Arena, BTB, Ranked)
  - `src/analysis/mode_categories.py` : `normalize_pair_name_to_mode_ui(pair_name, lang)` bilingue
  - `src/utils/discord_notifier.py` entièrement bilingue : `_format_player_field`, `build_embed_payload`, outcomes (🏆/💀/⚖️/🚶), KDA (`{k}K / {d}D / {a}A` vs `{k}F / {d}D / {a}A`), labels opération, footer
  - `src/visualization/distributions_outcomes.py` bilingue : traces Wins/Losses/Ties/Unfinished, buckets temporels (match/hour/day/week/month), heatmap win rate (jours EN/FR), `plot_matches_at_top_by_week` (Others/Top Rate)
  - `src/visualization/antagonist_charts.py` bilingue : `plot_duel_history` traduit Win/Loss/Tie dans l'annotation de duel
  - `src/ui/pages/win_loss.py` : tous les appels viz passent `lang=get_lang()`

### Modifié

- **Algorithme LUSR — mise à jour Elo-style (`K_ELO = 32`)** remplace la zone draw TrueSkill
  - Cause racine de la divergence : `v_draw(t > 0)` donnait des deltas positifs même sur composite=0.5, créant un drift infini quand `state.mu > INITIAL_MU` ou quand le joueur sur-fragait ses `kills_expected`
  - Nouvelle formule mu : `delta_mu = K_ELO × (composite − 0.5) × weight_factor` → ZÉRO exact à composite=0.5, indépendant de `mu_opp`
  - Sigma conserve la réduction TrueSkill évaluée à t=0 (symétrique, `mu_opp` influence `c²` uniquement)
  - Résultat : ratings stabilisés — SpartanA (Diamant V) → Platine IV BTB / Platine VI Arena / Diamant IV Ranked, SpartanB/SpartanC → Or II-IV selon mode
- **Score composite calibré sur 1765 matchs** (SpartanA, SpartanC, SpartanB — Argent → Diamant)
  - Signal cible : `individual_mmr = team_mmr × (kills_expected / ke_avg_match)`
  - Pondération par `nb_matchs × amélioration_MAE` : SpartanA 36.7%, SpartanC 40.0%, SpartanB 13.3%
  - Nouveaux poids : kills_vs_expected=31%, deaths_vs_expected=28%, damage_efficiency=23%, accuracy_delta=13%, win_factor=5%
- **Élimination du biais damage_efficiency** : `PlayerState.damage_eff_history` per-groupe — le composant damage utilise un delta vs historique personnel (comme accuracy_delta) au lieu de la valeur brute
- **Ancrage mu_opp sur `state.mu`** : `compute_enemy_strength` utilise `player_mu=state.mu` comme base d'estimation des adversaires (matchmaking met des joueurs de niveau similaire)
- **Paramètres d'inactivité réduits** : `INACTIVITY_SIGMA_PER_DAY` 3.5→1.0, `MAX_INACTIVITY_DAYS` 30→14 — évite les swings de ±200 pts après une longue pause
- **Seed sigma CSR réduit** : `PlayerState.from_csr()` démarre à `sigma=MIN_SIGMA` (60) au lieu de `INITIAL_SIGMA × 0.6` (210) — le CSR est un ancrage fort, démarrer en état stable évite la volatilité initiale

- **Page Carrière — Courbe XP estimée pré-sync** (`src/ui/pages/career.py`)
  - Trace violette pointillée estimant l'XP pour les ~561 matchs joués avant le premier sync
  - XP moyen/match = `first_xp / n_matchs_pré_sync` — la courbe part de ~0 au match le plus ancien et rejoint le premier snapshot réel
  - Visuellement distincte de la courbe XP réelle (violet `#CE93D8` pointillé)

- **Page Carrière — Courbes de projection vers le rang Héros** (`src/ui/pages/career.py`)
  - **Projection standard** (orange, tirets) : extrapole depuis le rythme actif XP/jour en excluant les gaps d'inactivité > 14 jours
  - **Projection optimiste** (vert, tirets-points) : ajoute les défis hebdomadaires (950 XP/semaine = 4×50 + 3×100 + 3×150) et le défi quotidien (500 XP/jour), le tout avec boost XP ×2 — soit +4 450 XP/semaine en défis
  - Les deux courbes masquées par défaut — cliquer sur la légende pour les afficher
  - Ligne horizontale dorée au seuil Héros (9 319 350 XP)
  - Projection plafonnée à 10 ans pour éviter les graphes infinis
  - Légende déplacée en dessous du graphe (centrée)
  - 23 tests unitaires dans `tests/test_career_xp_projection.py`

### Corrigé

- **20 tests pré-existants corrigés** suite à la migration v5.1 (architecture shared)
  - Groupe A (assertions/fixtures) : `test_backfill_bitmask`, `test_backfill_detection`, `test_xuid_resolution_regression` (×2), `test_post_refactor_perf_contracts`, `test_data_services_contracts`, `test_media_components_sprint4` (×2), `test_media_improvements`, `test_legacy_free_global`
  - Groupe B (mocks v4→v5) : `test_lazy_loading` (×5 — `_get_match_source` v5.1), `test_data_contract_sessions` (réécriture fixture v5 shared + player_match_enrichment)
  - Groupe C (source + mocks) : `test_sessions_integration` (fallback DB production masqué par `__file__` patch), `test_duckdb_repository_schema_contract` (schéma `xuid/gamertag` dans shared fixture), `test_teammates_impact_tab` (×2 — mock `_ensure_shared_attached` + `_load_highlight_events`)

---

## [5.2.0] - 2026-02-20

### Ajouté

- **Filtres v5.2 — Persistance intent-based** (`src/ui/filter_state.py`)
  - `FilterPreferences` : dataclass sauvegardée en JSON par joueur
  - Modes persistés : `playlist_mode`, `mode_mode`, `map_mode` (`"exclude"` / `"include"`)
  - Listes d'exclusions : `excluded_playlists`, `excluded_modes`, `excluded_maps`
  - `_detect_filter_mode()` : heuristique 70/30 — si > 70% des options sont cochées, mode "exclude" ; sinon "include"
  - `reconcile_filter_prefs()` : auto-réconciliation lors de l'apparition de nouvelles options — nouvelles playlists/modes/cartes incluses par défaut sans reset des préférences existantes
  - 45 tests unitaires dans `tests/test_filter_state.py`

- **Filtres v5.2 — Sélecteur "Type d'expérience"** (`src/app/filters_render.py`)
  - Pré-filtre statique : "PVP non classé", "PVP classé", "PVE (Firefight)" activant le filtre `is_firefight`
  - Cascade suppression correcte : modes/cartes calculés depuis `dropdown_base` complet (avant filtre playlist)
  - Intégration des `FilterPreferences` dans le rendu des filtres cascades

- **Stats PvE / Firefight v5.2 — Base dédiée `shared_pve.duckdb`**
  - Nouvelle base `data/warehouse/shared_pve.duckdb` séparée de `shared_matches.duckdb` (évite les colonnes NULL sur 90%+ des matchs PvP)
  - Table `pve_match_stats` : stats par joueur par match Firefight — waves, boss kills, kills par type d'ennemi (Banished : Grunt, Elite, Jackal, Brute, Hunter, Skimmer ; Forerunner : Crawler, Soldier, Knight, Warden)
  - `ensure_pve_schema()` dans `src/data/sync/migrations.py` — création idempotente du schéma
  - `PVE_SCHEMA_DDL` : DDL complet + index `idx_pve_xuid` + `idx_pve_match_id`

- **Stats PvE — Modèles Python** (`src/data/sync/models.py`)
  - `PveMatchStatsRow` : dataclass avec 20 colonnes (waves, boss, ennemi par type, pve_bits)

- **Stats PvE — Transformer** (`src/data/sync/transformers.py`)
  - `extract_pve_stats(match_json)` : extraction pour tous les joueurs d'un match Firefight
  - `_find_pve_stats_dict(player)` : recherche récursive du bloc PvE (EliminationStats / PveStats / FirefightStats / détection par clés)
  - `_extract_enemy_kills_by_type(pve_dict)` : support double structure (champs directs `GruntKills` + sous-dict `EnemyKillsByType`)
  - `_is_firefight_match()` enrichie : 3 critères — `GameVariantCategory` (IDs 41, 42 validés sur JSON API réels), `UgcGameVariant.PublicName`, `Playlist.PublicName` (firefight/baptême/survive)

- **Stats PvE — Pipeline insertion** (`src/data/sync/batch_insert.py`)
  - `batch_insert_pve_stats(conn, rows)` : insertion batch avec `INSERT OR REPLACE`

- **Stats PvE — Bitmask** (`src/data/sync/constants.py`)
  - `PveBits(IntFlag)` : bitmask granulaire pour `pve_match_stats.pve_bits` — TOTAL_KILLS, BOSS_KILLS, GRUNT, ELITE, JACKAL, BRUTE, HUNTER, SKIMMER, SENTINEL, MARINE + combinaisons ALL_ENEMIES, FULL_PVE
  - `MatchBits.PVE_STATS = 1 << 20` : guard global dans `match_registry.backfill_completed` — posé pour tout match traité (Firefight ou non) pour éviter la re-détection infinie

- **Stats PvE — Sync Engine** (`src/data/sync/engine.py`)
  - `_pve_connection` : connexion lazy-init vers `shared_pve.duckdb`
  - `_pve_db_lock` : verrou asyncio dédié
  - `_get_pve_connection()` : lazy init + `ensure_pve_schema` au premier accès
  - `_try_insert_pve_stats(stats_json, match_id, shared_conn)` : extraction + insertion + pose du bit `MatchBits.PVE_STATS` — appelé dans `_process_new_match` et `_process_known_match`

- **Stats PvE — SyncScope** (`src/data/sync/scope.py`)
  - Champs `pve_stats: bool` et `force_pve_stats: bool` dans `SyncScope`
  - Registrés dans `_FORCE_MAP` et `_ALL_DATA_FIELDS`

- **Stats PvE — Détection backfill** (`scripts/backfill/detection.py`)
  - Double guard : `mr.is_firefight = TRUE AND (COALESCE(mr.backfill_completed, 0) & PVE_STATS) = 0`
  - `force_pve_stats` : ignore le guard, retourne tous les matchs Firefight
  - `MatchBits.PVE_STATS` ajouté à `compute_bits_needed_from_scope`

- **Stats PvE — CLI backfill** (`scripts/backfill/cli.py`)
  - Arguments `--pve-stats` et `--force-pve-stats`

- **Stats PvE — Orchestrateur backfill** (`scripts/backfill/orchestrator.py`)
  - `_backfill_pve_for_match()` : ouverture `shared_pve.duckdb`, `ensure_pve_schema`, `batch_insert_pve_stats`, pose du bit guard dans `match_registry`
  - Compteur `pve_stats_inserted` dans `_empty_result()`

- **Citations PvE** (`src/analysis/citations/engine.py`)
  - `load_match_pve_stats(match_id)` : lecture depuis `shared_pve.duckdb`
  - Fusion des stats PvE dans `match_stats` avant calcul des citations
  - `pve_stat` reconnu comme `mapping_type` (traité identiquement à `stat`)

- **81 nouveaux tests** :
  - `tests/test_filter_state.py` : 45 tests — `FilterPreferences`, `_detect_filter_mode()`, `reconcile_filter_prefs()`, save/load
  - `tests/test_pve_transformers.py` : 36 tests — `_is_firefight_match()`, `_extract_enemy_kills_by_type()`, `extract_pve_stats()`, schéma DuckDB, batch insert, `PveMatchStatsRow`, `PveBits`, `SyncScope.pve_stats`

- **Scoreboard "Dernier match"** (`src/ui/pages/match_view_players.py`, `src/data/repositories/_roster_loader.py`)
  - `load_match_scoreboard(match_id)` : requête DuckDB joignant `match_participants` + `xuid_aliases` + sous-requête `medals_earned` (Perfect Kill, ID 1512363953). 20 champs par joueur, trié par `(team_id, rank)`.
  - `render_match_scoreboard()` : tableau HTML par équipe avec 18 colonnes — Gamertag, Rang, Score, Frags, Morts, Assist., FDA, Folie meurtrière, Tirs à la tête, Frags parfaits, Tirs, Tirs au but, Précision, Corps à corps, Armes lourdes, Dégâts infligés, Dégâts subis, Durée de vie moy.
  - Gestion N équipes + joueurs sans `team_id` (NULL → groupe séparé en fin)
  - En-têtes couleur Okabe-Ito : bleu `#0072B2` pour l'équipe du joueur, vermillon `#D55E00` pour les adversaires
  - Ligne du joueur surlignée (cyan `#00e5ff`)
  - Résolution gamertags via `load_match_gamertags_fn` (même pipeline que l'ancien roster)
  - CSS `.os-scoreboard` / `.os-sb-*` avec wrapping colonnes (`max-width: 80px`, `word-break`)
  - Remplace la section "Joueurs" (roster) supprimée

- **Tokens per-player pour endpoints player-gated** (`src/data/sync/api_client.py`, `src/ui/profile_api_tokens.py`)
  - `SPNKR_OAUTH_REFRESH_TOKEN_<GT_NORMALISÉ>` dans `.env.local` pour chaque joueur (ex: `_SPARTANC`, `_MON_GT_2`)
  - Normalisation : `re.sub(r"[^A-Za-z0-9]", "_", gt.strip()).upper()`
  - `get_tokens_for_player(gamertag)` : async, retourne `Tokens | None` — skip + warning si absent (pas de fallback global sur endpoint restreint)
  - `get_player_token_env_key(gamertag)` : retourne la clé env normalisée
  - `profile_api_tokens.get_tokens()` enrichi : param `gamertag` optionnel — priorité token joueur > token global (fallback naturel pour endpoints publics)
  - `profile_api.py`, `get_profile_appearance()` : param `gamertag` propagé jusqu'au fetch SPNKr
  - `load_profile_api()` : dérive le gamertag depuis la DB et le passe à `get_profile_appearance()` — corrige l'adornment/career rank pour les joueurs non-propriétaire du token global

- **Sync Career Rank player-gated** (`src/data/sync/engine.py`)
  - `sync_career_rank()` utilise `get_tokens_for_player()` — skip silencieux + warning si absent
  - Persiste `spartan_id` dans `career_progression` (colonne ajoutée via migration `add_spartan_id_to_career_progression()`)
  - `CareerRankRow.spartan_id` dans `src/data/sync/models.py`

- **Spartan ID dans le hero banner** (`src/ui/styles.py`, `src/app/main_helpers.py`)
  - `get_hero_html()` : nouveau paramètre `spartan_id` — affiché dans la section career-rank sous le label de rang (`.career-rank__spartan-id`)
  - `render_profile_hero()` : charge `spartan_id` depuis `career_progression` (DB, source de vérité) et le passe au hero HTML
  - CSS `.career-rank__spartan-id` : style compact, semi-transparent, lettres espacées

- **32 nouveaux tests** (`tests/test_player_tokens.py`)

### Modifié

- **Accessibilité daltonisme — Migration palette Okabe-Ito** (`src/visualization/`)
  - 7 fichiers de visualisation mis à jour : `antagonist_charts.py`, `performance.py`, `objective_charts.py`, `participation_charts.py`, `team_dominance_timeline.py`, `match_impact_timeline.py`, `friends_impact_heatmap.py`
  - Remplacement des couples rouge/vert néon saturés (incompatibles deuteranopie et protanopie) par la palette **Okabe-Ito** (Wong 2011), référence internationale pour les graphiques accessibles
  - Correspondances principales : vert néon `#00ff00` → vert bleuté `#009E73` · rouge `#ff4444` → vermillon `#D55E00` · magenta `#ff66ff` → rose mauve `#CC79A7` · couleurs équipe `#3DFFB5`/`#FF4D6D` → bleu `#0072B2`/vermillon `#D55E00`
  - Chaque palette documentée avec les anciens hex et la justification dans un bloc de commentaires

- **`_is_firefight_match()`** — Fusion des deux définitions dupliquées en une seule fonction unifiée couvrant les 3 critères (GameVariantCategory + UgcGameVariant.PublicName + Playlist.PublicName)

### Déprécié

- **`display_name_from_xuid()` et `get_xuid_aliases()`** (`src/ui/aliases.py`) — Marquées `.. deprecated::`. Utiliser `load_match_gamertags_fn` pour le contexte match. Conservées pour scripts/migration/export.

### Supprimé

- **Section "Joueurs" (roster)** de la page Dernier match — Remplacée par le scoreboard. `render_roster_section` n'est plus appelée depuis `match_view.py`.

### Corrigé

- **Duplication `_is_firefight_match()`** — Deux définitions coexistaient dans `transformers.py`. La deuxième écrasait silencieusement la première, rendant inopérante la détection via `UgcGameVariant`. Fusion en une seule définition complète.

---

## [5.1.0] - 2026-02-17

### Ajouté

- **Module `src/data/sync/scope.py`** — Dataclass **SyncScope** centralisant les flags
  - Remplace les 30+ kwargs booléens copiés dans 6 fichiers (cli → backfill_data → orchestrator → detection → API)
  - `SyncScope.from_cli_args(args)` : construction depuis argparse
  - `SyncScope.make_all()` : factory pour `--all-data`
  - `resolve()` : implications automatiques (`all_data` → champs, `force_X` → X)
  - Propriétés : `has_any_option()`, `needs_api`, `needs_local_only`, `requested_types`
  - Registres : `_ALL_DATA_FIELDS`, `_FORCE_MAP`, `_REQUESTED_TYPE_MAP`
  - 98 tests unitaires dans `tests/test_sync_scope.py`
  - **Ajouter un nouveau type** : 1 champ dans SyncScope + 1 arg CLI + implémentation métier
- **Module `src/ui/streamlit_modern.py`** — Wrappers compatibilité Streamlit moderne
  - `fragment_if_available` : décorateur graceful-degradation pour `@st.fragment`
  - `PLOTLY_CLEAN_CONFIG` : config Plotly sans barre d'outils
  - `plotly_chart` : wrapper avec config propre par défaut
  - `HAS_FRAGMENT`, `HAS_NAVIGATION` : détection de version
- **Module `src/ui/vectorize_helpers.py`** — Remplacement vectorisé de `map_elements()`
  - `build_mapping()` : pré-calcul dict mapping sur valeurs distinctes
  - `vectorized_apply()` : apply vectorisé via `replace_strict()`
  - `safe_int_format()`, `format_score_pair()` : expressions Polars réutilisables
- **Helpers `get_shared_matches_path()`** — Fonctions centralisées dans `src/utils/paths.py`
  - `get_shared_matches_path()` : chemin absolu vers `shared_matches.duckdb`
  - `get_shared_matches_path_from_player()` : déduction depuis path joueur
- **Script `cleanup_legacy_tables.py`** — Suppression tables obsolètes
  - 9 tables supprimées : `match_stats`, `medals_earned`, `highlight_events`, `player_stats`, `xuid_aliases`, + 4 vues `mv_*`
  - Options : `--dry-run`, `--backup`, `--all`
  - Backups automatiques dans `backups/pre_cleanup/`
- **Vue matérialisée `mv_player_matches`** — Optimisation performance v5.1
  - Pré-calcul jointures match_participants + match_registry + metadata
  - Réduction parsing SQL de 170→10 lignes par requête
  - Gain performance : -70% parsing SQL
- **Cache Repository Streamlit** — `get_cached_repository_st()` avec `@st.cache_resource(ttl=3600)`
  - Connexion DB persistante entre pages UI
  - Gain : 80ms→<20ms connexion
- **Index DuckDB Performance** — 16+ index créés sur 9 tables
  - Index composites `(xuid, match_id)`, `(match_id, xuid)`
  - Index triés sur `start_time`
- **Cache schema metadata** — `_has_column()` et `_has_shared_mp_column()` cachés
  - Évite requêtes `information_schema` répétées
- **Scripts migration bannières LEGACY** — 5 scripts marqués + README.md
  - Bannière claire "HORS SERVICE POST-V5.1"
  - Documentation dans `scripts/migration/README.md`

### Modifié

- **`backfill_data.py` refactoré** — `main()` utilise `SyncScope.from_cli_args()` (−90 lignes)
  - Plus besoin de copier 30+ `args.X` deux fois pour `--all` et `--player`
- **`orchestrator.py` refactoré** — `backfill_player_data`, `backfill_all_players`, `_backfill_with_api` acceptent `scope=SyncScope`
  - Anciens kwargs conservés (marqués `LEGACY`) pour rétro-compatibilité
  - `requested_types` construit via `scope.requested_types` au lieu de 16 `if/append`
- **`detection.py` refactoré** — `find_matches_missing_data` accepte `scope=SyncScope`
  - Anciens kwargs conservés (marqués `LEGACY`) pour rétro-compatibilité
- **Bump Streamlit ≥1.37.0** — Requis pour `@st.fragment` et futures migrations `st.navigation`
- **Plotly `config={"displayModeBar": False}`** — Appliqué sur 69 `st.plotly_chart` (15 fichiers)
  - Suppression barre d'outils Plotly pour une UI plus propre
- **`@fragment_if_available`** — Décorateur appliqué sur 5 pages multi-charts
  - timeseries, session_compare, win_loss, objective_analysis, career
  - Réduit le re-render au fragment seul lors d'interactions filtre
- **`match_history.py` modernisé** — Remplacement HTML custom par `st.dataframe` + `column_config`
  - Suppression dead code : `_format_score_label`, `_fmt`, `_fmt_mmr_int`
  - Virtualisation native Streamlit pour tableaux larges
- **`st.navigation` lazy loading** — 11 page closures dans `streamlit_app.py`
  - `build_navigation()` + `render_page_selector_nav()` dans `page_router.py`
  - Fallback legacy `dispatch_page()` pour Streamlit < 1.36
  - Seules les pages visitées sont importées → -60% mémoire initiale
- **Centralisation `duckdb_read_only()`** — Context manager dans `src/utils/db.py`
  - 7 fichiers migrés (career, cache_loaders, cache_filters, media_library, multiplayer, data_loader)
  - `duckdb.connect` directs : 14 → 4 (restants : sync engine, écriture légitime)
- **Réduction `st.rerun()`** — 32 → 14 dans `src/`
  - `checkbox_filter.py` : 16 reruns → 0 via callbacks `on_click`/`on_change`
  - Trio button filters : `on_click=_apply_trio_filter`
- **Sécurisation `unsafe_allow_html`** — html.escape() sur données dynamiques
  - `kpi.py` et `performance.py` : XSS protection
  - `sidebar.py` brand : HTML → `st.header()` + `st.divider()`
- **Tests non-régression modernisation** — 30 tests dans `test_8ter_modernisation.py`
  - Couverture : staticPlot, fragments, st.navigation, duckdb_read_only, st.rerun, html.escape
- **Éradication complète `map_elements()`** — 28 occurrences remplacées dans 15 fichiers
  - Remplacement par `build_mapping()` + `replace_strict()` ou expressions Polars natives
  - Fichiers : filters.py, filters_render.py, win_loss.py, last_match.py, stats.py,
    match_view_charts.py, media_library.py, teammates_helpers.py, session_compare.py,
    session_compare_charts.py, duckdb_analytics.py, match_view.py, citations.py,
    teammates_service.py, media_indexer.py
- **Migration `xuid_aliases` → `shared_matches.duckdb`** — Source unique centralisée
  - 9 fichiers migrés pour lire depuis `shared.xuid_aliases` (13 955 rows)
  - Suppression fallbacks locaux `stats.duckdb`
  - Fichiers : `aliases.py`, `xuid.py`, `multiplayer.py`, `cache_loaders.py`, `engine.py`, `_roster_loader.py`, `sessions_backfill.py`, `sync.py`, `resolve_missing_gamertags.py`
- **`_get_match_source()`** retourne maintenant un 3-tuple `(source_sql, params, uses_mv)`
  - Permet skip jointures redondantes en mode v5.1
- **8+ fonctions cache_loaders** migrées vers `get_cached_repository_st()`
  - Suppression connexions neuves redondantes
- **Jointures metadata/MMR** skippées en mode v5.1 quand `uses_mv=True`
  - RC3/RC4 : -3 LEFT JOIN sur chemin critique

### Corrigé

- **Onglet Citations affichait 159 citations au lieu de 45** — Filtrage par `citation_mappings.enabled` réactivé
  - Le JSON `halo5_commendations_fr.json` contient 159 citations (armes, Spartan Companies, etc.)
  - Le filtrage avait été supprimé, affichant toutes les citations y compris celles sans mapping
  - Correction : les items JSON sont maintenant filtrés par les noms normalisés des citations activées via `CitationEngine.load_mappings()`
  - Fichier : `src/ui/commendations.py`

### Supprimé

- **Tables legacy player DBs** — 9 tables par joueur, données centralisées
  - `match_stats`, `medals_earned`, `highlight_events`, `player_stats`, `xuid_aliases`
  - Vues obsolètes : `mv_match_stats_with_context`, `mv_recent_matches`, `mv_team_stats`, `mv_opponent_stats`
  - 38 528 rows libérées sur 4 joueurs
- **Références SQLite runtime** — 0 `import sqlite3` dans `src/`
- **Références `metadata.db`** — Tout migré vers `metadata.duckdb`
- **Méthode dépréciée `attach_sqlite`** — Supprimée de duckdb_engine.py

### Performance

| Métrique | v5.0 | v5.1 | Gain |
|----------|------|------|------|
| Connexion DB | 80ms | <20ms | **-75%** |
| load_matches(100) | 200ms | <80ms | **-60%** |
| Première page UI | 1500ms | <800ms | **-47%** |
| Parsing SQL/requête | 170 lignes | 10 lignes | **-94%** |

---

## [5.0.0] - 2026-02-15

### Ajouté

- **Architecture shared_matches.duckdb** — Base de données partagée centralisant les matchs de tous les joueurs
  - 6 tables : `match_registry`, `match_participants`, `highlight_events`, `medals_earned`, `xuid_aliases`, séquence `highlight_events_id_seq`
  - 14 index optimisés (match_id, xuid, start_time, composites)
  - Schéma DDL complet : `scripts/migration/schema_v5.sql`
  - Documentation : `docs/SHARED_MATCHES_SCHEMA.md`
- **Migration v4 → v5** — Scripts de migration incrémentale par joueur
  - `scripts/migration/create_shared_matches_db.py` : création de la DB partagée
  - `scripts/migration/migrate_player_to_shared.py` : migration par joueur
  - Résultat : 1289 matchs migrés, 285 partagés (22.1%), 0 orphelins
- **Détection matchs partagés dans Sync Engine** — Sync allégée pour matchs déjà connus
  - `_process_known_match()` : enrichissement personnel uniquement (économie 1-2 appels API/match)
  - `_process_new_match()` : sync complète vers shared (registry + participants + events + medals)
  - `extract_all_medals()` : extraction des médailles de TOUS les joueurs du match
  - `extract_match_registry_data()` : extraction données communes du match
- **ATTACH multi-DB dans DuckDBRepository** — Lecture transparente depuis `shared_matches.duckdb`
  - `shared_db_path` auto-détecté ou configurable
  - Queries natives `shared.match_participants`, `shared.match_registry`, `shared.medals_earned`
  - Propagation dans la factory repository
- **Sous-requête `_get_match_source()`** — Abstraction permettant à toutes les pages UI de lire depuis shared sans modification
- **Optimisations API Sync v5**
  - Parallélisation appels API skill + events (`asyncio.gather`)
  - Batching des insertions DB (commit tous les 10 matchs)
  - Performance scores calculés en batch post-sync
  - Rate limit optimisé (10 req/s, parallel_matches=5)
- **Citations DuckDB-first** — Nouveau système de citations stockées par match
  - `CitationEngine` : moteur de calcul et agrégation SQL
  - Table `citation_mappings` dans `metadata.duckdb` : 14 règles (8 existantes + 6 réintégrées)
  - Table `match_citations` dans chaque `stats.duckdb` joueur
  - Backfill CLI : `--citations` / `--force-citations` dans `scripts/backfill_data.py`
  - 6 citations objectives réintégrées : Défenseur du drapeau, Je te tiens !, Sus au porteur du drapeau, Partie prenante, À la charge, Annexion forcée
  - Colonne `enabled` dans `citation_mappings` pour désactivation sans suppression
  - Support V5 (shared_matches) dans `CitationEngine` avec fallback V4
  - Documentation : `docs/CITATIONS.md`
- **Framework de test MockStreamlit** — Fixture `MockStreamlit` dans `conftest.py` pour tester les pages UI en mode headless
- **+946 tests** ajoutés (S1→S7ter) — total 2768 passed, 0 failed, 38 skipped
- **Script de nettoyage post-migration** — `scripts/cleanup_player_dbs_v5.py`
  - Supprime les tables redondantes des player DBs après migration v5 (match_stats, match_participants, highlight_events, medals_earned)
  - Mode --dry-run pour simulation sans modification
  - Backup optionnel avant nettoyage
  - Validation automatique de l'existence de shared_matches.duckdb
  - VACUUM automatique pour récupération d'espace disque (-85% de taille en moyenne)
  - Documentation : `docs/CLEANUP_V5.md`
- **Documentation** : `docs/SHARED_MATCHES_SCHEMA.md`, `docs/SYNC_OPTIMIZATIONS_V5.md`, `docs/TESTING_V5.md`, `docs/ARCHITECTURE_V5.md`, `docs/MIGRATION_V4_TO_V5.md`, `docs/CLEANUP_V5.md`

### Modifié

- **`DuckDBSyncEngine`** refactoré pour écrire dans `shared_matches.duckdb` (matchs, participants, events, médailles)
- **`DuckDBRepository`** refactoré avec ATTACH `shared_matches.duckdb` en read-only
  - `load_match_participants()` → lecture depuis `shared.match_participants`
  - `load_highlight_events()` → lecture depuis `shared.highlight_events`
  - `load_medals_for_match()` → lecture depuis `shared.medals_earned`
  - `load_matches()` → JOIN `shared.match_participants` + `shared.match_registry` + `player_match_enrichment`
- **Toutes les pages UI** utilisent `_get_match_source()` au lieu de `match_stats` directement
- **`render_h5g_commendations_section()`** utilise `CitationEngine` (agrégation SQL, ~90% plus rapide)
- **`render_citations_page()`** simplifié — ne pré-agrège plus les médailles/stats pour les citations
- **Filtrage des citations** piloté par `citation_mappings.enabled` (plus besoin du JSON d'exclusion)
- **Version `pyproject.toml`** bumpée de 3.0.0 à 5.0.0
- **Statut projet** : Development Status 4-Beta → 5-Production/Stable

### Supprimé

- **VIEWs de compatibilité v4** supprimées (`scripts/migration/remove_compat_views.py`)
- **Données dupliquées** dans les player DBs : `match_participants`, `highlight_events`, `medals_earned` centralisés dans shared
- **Shim `src/db/migrations.py`** — déprécié, supprimé en faveur de `src.data.sync.migrations`
- `CUSTOM_CITATION_RULES` dict (ancien `commendations.py`)
- `_compute_custom_citation_value()` (itérations lentes, remplacé par SQL)
- `load_h5g_commendations_tracking_rules()` (remplacé par `citation_mappings` DuckDB)
- Constantes `DEFAULT_H5G_TRACKING_ASSUMED_PATH` / `DEFAULT_H5G_TRACKING_UNMATCHED_PATH`
- Dépendance aux fichiers JSON de tracking commendations
- Logique d'exclusion JSON dans `render_h5g_commendations_section()`

### Corrigé

- **Tests flaky Windows** : `tmp_dir` → `tmp_path` pour éviter DuckDB `WinError 32` (file locking)
- **Tests lazy_loading** : mode v4 forcé pour compatibilité

### Performance

| Métrique | v4 | v5 | Gain |
|----------|----|----|------|
| Stockage (4 joueurs) | 800 MB | 250 MB | **-69%** |
| DB size par joueur | 200 MB | 30 MB | **-85%** |
| Appels API (sync 4 joueurs) | 12 000 | 3 300 | **-72%** |
| Temps sync (100 matchs) | 45 min | 12 min | **-73%** |
| Temps/match (partagé) | 16s | 0.5s | **-97%** |
| Temps/match (nouveau) | 16s | 2-3s | **-81%** |
