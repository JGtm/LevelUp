# Backlog consolidé — LevelUp

> **Dernière mise à jour** : 2026-03-13 — Backlog A→R traité sur `refactor/cleanup-all`.
> Plan v5.7 livré sur `analysis/weapon-parser-rewrite`.

---

## Historique (traité)

| Item | Description | Branche | Statut |
|------|-------------|---------|--------|
| A1 | Cherry-pick 3 commits safe (startup, citations CLI) | refactor/cleanup-all | ✅ |
| A2 | Tailscale guard process-level | refactor/cleanup-all | ✅ |
| A3 | citations_backfill + engine post-sync | refactor/cleanup-all | ✅ |
| B | Bug #0 — filtres auto-invalidation via db_key | refactor/cleanup-all | ✅ |
| C | SyncLock câblé à l'UI | refactor/cleanup-all | ✅ |
| D | Spam Tailscale/Discord (inclus A2) | refactor/cleanup-all | ✅ |
| E | Baseline taille de code | refactor/cleanup-all | ✅ |
| F | Fausse alerte Discord (inclus A1) | refactor/cleanup-all | ✅ |
| G | Fenêtre clignotante planificateur | les deux branches | ✅ |
| H | win_rate incohérent → WIN_RATE_EXPR | refactor/cleanup-all | ✅ |
| I | NaN-check fragile match_view.py | refactor/cleanup-all | ✅ |
| J | Dettes #2 #3 #4 #6 | refactor/cleanup-all | ✅ |
| J2 | Cleanup kwargs LEGACY SyncScope | refactor/cleanup-all | ✅ |
| J3 | career.py → DuckDBRepository (déjà OK) | — | ✅ |
| J4 | TODOs mineurs | refactor/cleanup-all | ✅ |
| J5 | Déduplication citations | refactor/cleanup-all | ✅ |
| K | i18n clés tronquées + doublon | refactor/cleanup-all | ✅ |
| L | PAIR_FR 399→56 entrées | refactor/cleanup-all | ✅ |
| M | CHANGELOG | refactor/cleanup-all | ✅ |
| N | PAIR_FR → modes_fr.json | refactor/cleanup-all | ✅ |
| O | Câbler t() dans l'UI | refactor/cleanup-all | ✅ |
| P | Performance UI P1→P5 | refactor/cleanup-all | ✅ |
| Q | CI/CD régression | refactor/cleanup-all | ✅ |
| R | Pandas résiduel + START_HERE archivé | refactor/cleanup-all | ✅ |

### Plan v5.7 (livré sur `analysis/weapon-parser-rewrite`)

| Chantier | Description | Statut |
|----------|-------------|--------|
| v5.7-A | Test migrations (A.4 highlight_events idempotent) | ✅ |
| v5.7-B | Pandas→Polars (7 to_pandas supprimés, 4 fichiers) | ✅ |
| v5.7-C | Dead code guard was_pandas supprimé | ✅ |
| v5.7-D | CSS-only map thumbnails (JS sandbox supprimé) | ✅ |
| v5.7-E | Launchers bilingues FR/EN (LevelUp.sh + .bat) | ✅ |
| v5.7-F | Traductions FR rangs (ranks.py + tests) | ✅ |
| v5.7-G | Version bump 5.5.1→5.7.0 + CHANGELOG | ✅ |

---

## Reste à faire

### Migration noms d'assets → IDs bruts en BDD

> Ref: `.ai/BACKLOG.md` L26-46

Dans `match_registry`, les colonnes `map_name`, `playlist_name`, `pair_name`,
`game_variant_name` stockent des noms résolus en parallèle des IDs bruts
(redondance + risque de stale data). L'UI doit résoudre les noms à la lecture
depuis `metadata.duckdb`, pas les lire depuis les colonnes `*_name`.

**Modèles de référence (déjà corrects)** : `medals_earned.medal_name_id` (UBIGINT),
`weapon_kills.weapon_id` (UBIGINT post v5.7).

**Actions** :
- [ ] Auditer les usages UI/query des colonnes `*_name` dans `match_registry`
- [ ] Créer une vue `v_match_registry` avec JOIN sur `metadata.duckdb`
- [ ] Migrer les requêtes consommatrices vers la vue
- [ ] Supprimer les colonnes `*_name` de `match_registry`

### Weapon Parser v2 (branche courante)

Travail en cours sur `analysis/weapon-parser-rewrite` — rewrite claim-and-remove du
parser. Non lié au plan v5.7. Fichiers non commités :
- `src/analysis/weapon_parser.py` (rewrite)
- `src/analysis/_kill_attribution.py` (nouveau)
- `src/analysis/_parser_logging.py` (nouveau)
- `src/analysis/_weapon_parser_compat.py` (nouveau)
- `src/analysis/_weapon_scanners.py` (nouveau)
- `src/analysis/reconciliation.py` (nouveau)
- `src/analysis/_weapon_data.py` (modifié)
- `src/data/repositories/_weapon_kills_repo.py`
- `src/data/services/weapon_extraction_service.py`
- `src/data/sync/_engine_weapon_kills.py`
- `src/data/sync/migrations.py`
- `src/data/migration/steps/__init__.py`
- `src/data/migration/steps/add_weapon_kills_reconciled_as.py`
- `src/ui/pages/explorer.py`, `explorer_logic.py`, `explorer_data.py`
- `src/ui/pages/match_view_players.py`
- `src/ui/pages/teammates_charts.py`, `teammates_impact.py`, `teammates_views.py`
- `src/ui/pages/win_loss.py`
- `src/visualization/_match_impact_events.py`, `trio.py`
- `streamlit_app.py`
- `scripts/backfill/_weapon_kills_logic.py`, `orchestrator.py`
- Tests : `test_weapon_parser_v2.py`, `test_weapon_migration.py`, `test_weapon_logging.py`, `test_weapon_reconciliation.py`, `test_weapon_service.py`

### Dette technique résiduelle

- **`_normalize_df`** dans `_performance_relative_helpers.py` : garde de compatibilité
  Pandas résiduelle (`pl.from_pandas`). Signature à resserrer vers `pl.DataFrame` pur.
  Annoter avec date d'expiration v5.8 ou supprimer.

- **`filters_render.py`** (702L) : dépasse le seuil 500L. Découpage en
  `filters_render.py` + `filters_logic.py` à planifier.

- **Fonctions longues** dans le baseline (8 violations documentées dans
  `scripts/size_baseline.txt`) : dette structurelle à réduire progressivement.
