# Axe 1 — Template parité Python↔Go + Streamlit↔React

> **À remplir à l'identique** par Claude puis par ChatGPT. Ne pas modifier la structure. Si une section ne s'applique pas, écrire explicitement « N/A — justification : ... » plutôt que de la supprimer.

## Métadonnées du passage

| Champ | Valeur |
|-------|--------|
| Auteur LLM | Claude \| ChatGPT (entourer) |
| Date du passage | `YYYY-MM-DD` |
| SHA Python (worktree `LevelUp`) | `xxxxxxx` |
| SHA Go (worktree `LevelUp-go-migration`) | `xxxxxxx` |
| SHA React (même worktree que Go) | `xxxxxxx` |
| Durée de l'analyse | `Nh` |
| Corpus analysé (nb fichiers) | `N Python / M Go / K React` |

## Synthèse exécutive (150 mots max)

> Résumé en 2-3 paragraphes : à quel niveau de parité on est, ce qui manque de critique, ce qui est modernisé volontairement.

---

## A. Parité API (endpoints HTTP)

> Comparer chaque endpoint Go (`apps/go-api/internal/api/handlers/`) avec son équivalent Python (FastAPI/Streamlit backend). Matrice des 24 endpoints référencée dans `OPENAPI_MVP_P0_P1.md`.

| Endpoint | Côté Python (fichier) | Côté Go (fichier) | Parité contrat | Parité payload | Parité status codes | Écart | Classif |
|----------|-----------------------|-------------------|:--------------:|:--------------:|:-------------------:|-------|:-------:|
| `GET /api/v1/...` | | | ✅/❌ | ✅/❌ | ✅/❌ | | 🔴🟠🟡🟢 |

## B. Parité pages UI (Streamlit → React)

> Comparer chaque page Streamlit (`src/ui/pages/*.py`) avec sa page React équivalente (`apps/web/src/features/*/*Page.tsx`).

| Page métier | Streamlit (fichier) | React (fichier) | Features couvertes | Features manquantes | Features modernisées | Classif |
|-------------|---------------------|-----------------|-------------------:|:-------------------:|:--------------------:|:-------:|
| Home | `home_mission_control.py` | `HomePage.tsx` | | | | 🔴🟠🟡🟢 |
| Career | `career.py` | `CareerPage.tsx` | | | | |
| Synthèse | `synthesis.py` | `SynthesisPage.tsx` | | | | |
| Match history | `match_history.py` | `MatchHistoryPage.tsx` | | | | |
| Match view | `match_view.py` + helpers | `MatchViewPage.tsx` | | | | |
| Last match | `last_match.py` | `LastMatchPage.tsx` | | | | |
| Explorer | `explorer.py` | `ExplorerPage.tsx` | | | | |
| Session compare | `session_compare.py` | `SessionComparePage.tsx` | | | | |
| Sessions (timeseries) | `timeseries.py` | `TimeseriesPage.tsx` | | | | |
| Squad / Teammates | `teammates.py` + `_teammates_trio.py` | `SquadPage.tsx` | | | | |
| Citations | `citations.py` | `CitationsPage.tsx` | | | | |
| Media | `media_library.py` + `media_v2.py` | `MediaPage.tsx` | | | | |
| Settings | `settings.py` | `SettingsPage.tsx` | | | | |
| Setup wizard | `setup_wizard.py` | `SetupPage.tsx` | | | | |
| Changelog | (N/A Python) | `ChangelogPage.tsx` | | | | 🟢 (feature nouvelle) |

## C. Parité algorithmes métier (7 cœurs)

| Algorithme | Python (module) | Go (package) | Golden values vertes ? | Écart observé | Classif |
|------------|-----------------|--------------|:----------------------:|---------------|:-------:|
| Performance score | `src/analysis/performance_score.py` | `internal/analysis/` | | | |
| LUSR / CSR | `src/analysis/skill_rating.py` | `internal/sync/skill_rating_loaders.go` + `internal/analysis/` | | | |
| Sessions | `src/analysis/sessions_detection.py` (ou équiv.) | `internal/analysis/` | | | |
| Citations | `src/analysis/citations_*.py` | `internal/analysis/citations*.go` | | | |
| Killer/victim | `src/analysis/killer_victim.py` (ou équiv.) | `internal/analysis/` | | | |
| Weapon parser | `src/analysis/weapon_parser.py` (ou équiv.) | `internal/analysis/weapon_parser.go` | | | |
| Spawn / comeback detection | `src/analysis/comeback_analysis.py` | `internal/analysis/` | | | |

## D. Parité sync / backfill

| Flux | Python | Go | Écart | Classif |
|------|--------|----|-------|:-------:|
| Sync delta | `scripts/sync.py` + `src/data/sync/engine.py` | `internal/sync/engine.go` | | |
| Backfill (tous flags SyncScope) | `scripts/backfill_data.py` + `scripts/backfill/` | `internal/sync/backfill.go` + `backfill_cli.go` | | |
| Write lease | `src/data/sync/lease.py` (ou équiv.) | `internal/sync/lease.go` | | |
| Bitmask `backfill_completed` (18 bits) | `src/data/sync/migrations.py` | `internal/sync/` + `internal/migration/` | | |

## E. Parité CLI / scripts opérationnels

| Script Python | Équivalent Go/CLI | Couvert ? | Écart | Classif |
|---------------|-------------------|:---------:|-------|:-------:|
| `scripts/sync.py` | | | | |
| `scripts/backup_player.py` | `internal/ops/` | | | |
| `scripts/restore_player.py` | `internal/ops/restore.go` | | | |
| `scripts/backfill_data.py` | `internal/sync/backfill_cli.go` | | | |
| `scripts/check_env.py` | | | | |
| (autres — compléter) | | | | |

## F. Parité données (schémas DuckDB)

> Vérifier que les tables `shared_matches_v2`, `shared_pve`, player `stats.duckdb`, `metadata.duckdb` ont le même schéma (colonnes, types, contraintes) côté Go et Python.

| Table | Colonnes identiques ? | Types identiques ? | Index/vues identiques ? | Écart | Classif |
|-------|:---------------------:|:------------------:|:-----------------------:|-------|:-------:|
| `match_registry` | | | | | |
| `match_participants` | | | | | |
| `medals_earned` | | | | | |
| `killer_victim_pairs` | | | | | |
| `xuid_aliases` | | | | | |
| `weapon_kills` | | | | | |
| `player_match_enrichment` | | | | | |
| `match_skill_rank` | | | | | |
| `sessions` | | | | | |
| `pve_match_stats` | | | | | |
| `weapon_labels` | | | | | |
| `career_ranks` | | | | | |
| Vues (`v_gamertag_lookup`, `v_match_full`, `v_killer_victim_full`, `v_weapon_kills`) | | | | | |

## G. Parité i18n

| Aspect | Python | Go/React | Écart | Classif |
|--------|--------|----------|-------|:-------:|
| Nombre de langues | 14 (via `src/ui/i18n/`) | | | |
| Source de traduction runtime | DuckDB (`mode_name_tr`, etc.) | | | |
| Résolution `Accept-Language` | | | | |

## H. Parité observabilité & erreurs

| Aspect | Python | Go/React | Écart | Classif |
|--------|--------|----------|-------|:-------:|
| Notifier Discord | `src/utils/discord_notifier.py` | | | |
| Logs structurés | | | | |
| Taxonomie erreurs provider | | `HALO_PROVIDER_ERROR_TAXONOMY.md` | | |

## I. Modernisations volontaires (🟢)

> Lister les écarts Python → Go / Streamlit → React qui sont intentionnels (amélioration, simplification, suppression). Chaque ligne doit avoir une motivation.

| Modernisation | Motivation | Remplacement | Impact utilisateur |
|---------------|------------|--------------|--------------------|
| | | | |

## J. Récap classifications

| Niveau | Nombre d'items | Liste des IDs (ou courtes descriptions) |
|--------|:--------------:|-----------------------------------------|
| 🔴 Bloquant | | |
| 🟠 Majeur | | |
| 🟡 Mineur | | |
| 🟢 Toléré | | |

## K. Observations libres

> Remarques qui ne rentrent dans aucune case ci-dessus. Max 300 mots.

---

**Fin du template axe 1.**
