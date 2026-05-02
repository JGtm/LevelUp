# Axe 1 · SCOPE — Parité Python↔Go + Streamlit↔React

## Objectif de l'axe

Vérifier que **toute fonctionnalité présente dans la version Python/Streamlit de référence** est soit présente à l'identique dans la version Go/React, soit explicitement marquée comme modernisée (🟢) avec motivation.

## Baseline (SHAs à figer au démarrage de l'audit)

| Worktree | Chemin | Branche | SHA à remplir |
|----------|--------|---------|---------------|
| Python (référence) | `c:\Users\Guillaume\Downloads\Scripts\LevelUp` | `v7/cockpit` | `db638c09` |
| Go (cible backend) | `c:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration` | `recovery/reapply-wip-s49-closure-2026-04-18` | `93c3cd66` |
| React (cible frontend) | même worktree que Go, dossier `apps/web/` | idem | `93c3cd66` |

> **Règle** : dès que les SHAs sont figés, on ne les change plus pendant toute la durée de l'axe. Si la base évolue, on rouvre un nouvel axe (Sprint 51+).

## Périmètre inclus

### Backend API (Python → Go)

- L'**intégralité des endpoints** de la matrice `OPENAPI_MVP_P0_P1.md` (source de vérité — le Sprint 50 compte, ne présume pas) + tout endpoint livré Sprint 49 qui ne figurerait pas encore dans la matrice
- Les endpoints historiquement listés dans `notYetImplemented` : le Sprint 49 est censé les avoir résolus → traitement en **vérification post-S49** (statut attendu : tous implémentés ; toute absence = 🔴 bloquant)
- Les handlers Go : tout `apps/go-api/internal/api/handlers/*.go` (hors `_test.go`)
- Les middlewares : `apps/go-api/internal/api/middleware/*.go`
- Le routage : `apps/go-api/cmd/` (point d'entrée, wire-up)

### Frontend (Streamlit → React)

- Toutes les pages Python de `src/ui/pages/*.py` (hors `_*.py` internes et `*_logic.py` / `*_data.py` qui sont des helpers)
- Pages canoniques à comparer (colonne gauche) :
  - `home_mission_control.py` → `HomePage.tsx`
  - `career.py` → `CareerPage.tsx`
  - `synthesis.py` → `SynthesisPage.tsx`
  - `match_history.py` → `MatchHistoryPage.tsx`
  - `match_view.py` (+ tous les `match_view_*`) → `MatchViewPage.tsx` + `MatchScoreboard.tsx` + `PlayerDetailPanel.tsx` + `MatchStatCards.tsx`
  - `last_match.py` → `LastMatchPage.tsx`
  - `explorer.py` (+ `explorer_logic.py`, `explorer_results.py`) → `ExplorerPage.tsx`
  - `session_compare.py` (+ logique & viz) → `SessionComparePage.tsx`
  - `timeseries.py` (+ helpers) → `TimeseriesPage.tsx`
  - `teammates.py` (+ `_teammates_trio.py` + 10+ modules `teammates_*`) → `SquadPage.tsx`
  - `citations.py` → `CitationsPage.tsx`
  - `media_library.py` + `media_v2.py` + modules associés → `MediaPage.tsx`
  - `settings.py` → `SettingsPage.tsx`
  - `setup_wizard.py` (+ logique) → `SetupPage.tsx`

### Algorithmes métier (code commun backend)

Les 7 algorithmes listés dans `SPRINT_ROADMAP.md §Rappels transverses` :
1. Performance score
2. LUSR (rolling MMR maison)
3. CSR
4. Sessions (détection + labellisation)
5. Citations (mapping médailles → citations)
6. Killer/victim (paires nemesis)
7. Weapon parser (filmshell → weapon_id)

+ comeback/spawn detection (v6.2)

### Sync, backfill, CLI

- `scripts/sync.py` + `src/data/sync/` (Python) ↔ `internal/sync/` (Go)
- `scripts/backfill_data.py` + `scripts/backfill/` (Python) ↔ `internal/sync/backfill*.go` (Go)
- `scripts/backup_player.py` / `restore_player.py` ↔ `internal/ops/` (Go)
- `scripts/check_env.py` ↔ `internal/validation/gate.go` (Go)

### Données

Schémas DuckDB identiques côté Python et Go :
- `shared_matches_v2.duckdb` (match_registry, match_participants, medals_earned, killer_victim_pairs, xuid_aliases, weapon_kills, highlight_events)
- `shared_pve.duckdb` (pve_match_stats)
- Player `stats.duckdb` (player_match_enrichment, match_skill_rank, sessions, media_files, media_match_associations, personal_score_awards, match_citations, career_progression, sync_meta)
- `metadata.duckdb` (career_ranks, citation_mappings, mode_lang_settings, mode_name_tr, mode_pair_overrides, mode_prefix_names, weapon_labels)
- Vues : `v_gamertag_lookup`, `v_match_full`, `v_killer_victim_full`, `v_weapon_kills`

### i18n

- 14 langues côté Python (`src/ui/i18n/`)
- Résolution runtime via DuckDB
- Header `Accept-Language` côté backend

## Périmètre EXCLU

- L'outillage développeur (`src/ai/` Python — RAG ChromaDB, serveur MCP Cursor) : outillage interne dev, non exposé aux utilisateurs finaux, **pas prévu** pour port Go → hors audit parité (cf. `CLAUDE.md` §`src/ai/`)
- Les scripts de migration historiques (`recover_from_sqlite.py`, `migrate_player_to_duckdb.py`) — one-shot déjà consommés
- Les tests eux-mêmes (traités en axe 3)
- La qualité du code interne des modules (traitée en axe 2)

## Critères mesurables

| Critère | Seuil acceptable |
|---------|------------------|
| Endpoints avec écart **bloquant** | **0** |
| Endpoints avec écart **majeur** non ticketé | **0** |
| Pages React avec feature critique manquante par rapport Streamlit | **0** |
| Algorithmes sans golden values vertes | **0** |
| Tables DuckDB avec schéma divergent | **0** |
| Modernisations 🟢 sans motivation écrite | **0** |
| Endpoints `notYetImplemented` restants (héritage pré-S49) | **0** |

## Entrées pour le LLM

Pour chaque passage LLM (Claude ou ChatGPT), fournir :
1. Le `SCOPE.md` (ce fichier)
2. La `CHECKLIST.md`
3. Le template vide `templates/axis1_parity_template.md`
4. Accès lecture au code des deux worktrees (chemins absolus listés en haut de ce fichier)
5. Les 4 documents de référence Go :
   - `PLAN_MIGRATION_PYTHON_TO_GO_V2.md`
   - `AUDIT_PARITE_GO_VS_PYTHON.md`
   - `OPENAPI_MVP_P0_P1.md`
   - `HALO_CANONICAL_MODEL.md`

## Sortie attendue

`claude_review.md` et `chatgpt_review.md` remplis à 100% selon le template, chaque écart classé, chaque ligne du tableau endpoints/pages renseignée.
