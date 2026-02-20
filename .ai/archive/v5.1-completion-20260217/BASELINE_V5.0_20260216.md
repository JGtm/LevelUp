# Baseline v5.0 — Snapshot du 16 février 2026

> Document de référence généré automatiquement avant le démarrage de la v5.1.

---

## Git

- **Branche de travail** : `copilot/prepare-phases-5-6-analysis`
- **Branche de secours** : `backup-v5.0-20260216`
- **Commit HEAD** : `015ed5d` — docs: Add PARALLELISATION_V5.1 guide

---

## Tests Baseline

```
python -m pytest --ignore=tests/integration -q --tb=line
```

- **Collectés** : 2880 items
- **Passés** : 2819
- **Skipped** : 61
- **Échoués** : 0
- **Durée** : 114.26s (1min54)

**Verdict : ✅ BASELINE VERTE — 100% succès**

---

## Joueurs (4)

| Joueur | Taille DB | Tables | Matchs (player_match_stats) |
|--------|-----------|--------|----------------------------|
| Chocoboflor | 56 MB | 20 | 241 |
| JGtm | 74 MB | 17 | 516 |
| Madina97294 | 91 MB | 17 | 962 |
| XxDaemonGamerxX | 9.6 MB | 13 | 18 |

---

## Warehouse

| Base | Taille |
|------|--------|
| shared_matches.duckdb | 87 MB |
| metadata.duckdb | 1.3 MB |

### shared_matches.duckdb — Tables

| Table | Lignes |
|-------|--------|
| highlight_events | 521,991 |
| killer_victim_pairs | 193,912 |
| match_participants | 21,976 |
| match_registry | 1,289 |
| medals_earned | 7,162 |
| schema_version | 2 |
| xuid_aliases | 13,955 |

### metadata.duckdb — Tables

| Table | Lignes |
|-------|--------|
| citation_mappings | 14 |

---

## Tables par Joueur (détail)

### Chocoboflor (20 tables)

| Table | Lignes | Statut |
|-------|--------|--------|
| antagonists | 1 | ✅ Conservée |
| backfill_status | 0 | ✅ Conservée |
| career_progression | 0 | ✅ Conservée |
| highlight_events | 14,671 | ⚠️ Legacy (doublon shared) |
| match_citations | 217 | ✅ Conservée |
| match_participants | 621 | ⚠️ Legacy (doublon shared) |
| match_stats | 5 | ⚠️ Legacy |
| medals_earned | 182 | ⚠️ Legacy (doublon shared) |
| media_files | 0 | ✅ Conservée |
| media_match_associations | 0 | ✅ Conservée |
| mv_global_stats | 10 | ✅ Vue matérialisée |
| mv_map_stats | 5 | ✅ Vue matérialisée |
| mv_mode_category_stats | 0 | ✅ Vue matérialisée |
| mv_session_stats | 214 | ✅ Vue matérialisée |
| personal_score_awards | 1,836 | ✅ Conservée |
| player_match_enrichment | 241 | ✅ Conservée |
| player_match_stats | 241 | ✅ Conservée |
| sessions | 214 | ✅ Conservée |
| skill_history | 0 | ✅ Conservée |
| sync_meta | 3 | ✅ Conservée |
| v_highlight_events | — | ❌ Vue cassée (ref shared manquante) |
| v_match_participants | — | ❌ Vue cassée (ref shared manquante) |
| v_match_stats | — | ❌ Vue cassée (ref shared manquante) |
| v_medals_earned | — | ❌ Vue cassée (ref shared manquante) |
| xuid_aliases | 4,875 | ⚠️ Legacy (doublon shared) |

### JGtm (17 tables + 4 vues cassées)

| Table | Lignes |
|-------|--------|
| antagonists | 58 |
| backfill_status | 0 |
| career_progression | 0 |
| match_citations | 471 |
| media_files | 56 |
| media_match_associations | 40 |
| mv_global_stats | 0 |
| mv_map_stats | 75 |
| mv_mode_category_stats | 0 |
| mv_session_stats | 431 |
| personal_score_awards | 3,018 |
| player_match_enrichment | 518 |
| player_match_stats | 516 |
| sessions | 431 |
| skill_history | 0 |
| sync_meta | 3 |
| xuid_aliases | 5,094 |

### Madina97294 (17 tables + 4 vues cassées)

| Table | Lignes |
|-------|--------|
| antagonists | 40 |
| backfill_status | 0 |
| career_progression | 0 |
| match_citations | 912 |
| media_files | 0 |
| media_match_associations | 0 |
| mv_global_stats | 0 |
| mv_map_stats | 0 |
| mv_mode_category_stats | 0 |
| mv_session_stats | 934 |
| personal_score_awards | 5,506 |
| player_match_enrichment | 971 |
| player_match_stats | 962 |
| sessions | 934 |
| skill_history | 0 |
| sync_meta | 3 |
| xuid_aliases | 13,701 |

### XxDaemonGamerxX (13 tables + 4 vues cassées)

| Table | Lignes |
|-------|--------|
| career_progression | 0 |
| match_citations | 16 |
| media_files | 0 |
| media_match_associations | 0 |
| mv_global_stats | 0 |
| mv_map_stats | 16 |
| mv_mode_category_stats | 0 |
| mv_session_stats | 0 |
| personal_score_awards | 48 |
| player_match_enrichment | 0 |
| player_match_stats | 18 |
| sync_meta | 3 |
| xuid_aliases | 0 |

---

## Métriques Legacy

### Imports SQLite dans src/

**Compteur : 0** ✅ (déjà nettoyé)

### Imports SQLite dans scripts/

- `scripts/migration/` : 5 fichiers (attendu — scripts de migration)
- `scripts/refetch_film_roster.py` : 1 fichier (à traiter en Étape 4)
- `scripts/_archive/` : 8 fichiers (archives, non prioritaire)

### Imports Pandas dans src/ (total : 5)

| Fichier | Type | À migrer ? |
|---------|------|-----------|
| `src/data/services/win_loss_service.py` | Service métier | ✅ Oui |
| `src/ui/pages/win_loss.py` | Page UI | ✅ Oui |
| `src/visualization/distributions.py` | Visualisation | 🟡 Bridge frontière |
| `src/visualization/_compat.py` | Bridge compatibilité | ❌ Non (bridge autorisé) |
| `src/data/integration/streamlit_bridge.py` | Bridge Streamlit | ❌ Non (bridge autorisé) |

**Imports Pandas métier à éliminer : 2** (vs 7 estimés dans le plan)

---

## Vues Cassées (Observation)

Les 4 joueurs ont des vues `v_*` (v_highlight_events, v_match_participants, v_match_stats, v_medals_earned) qui échouent car elles référencent des tables shared_matches qui ne sont pas ATTACHées dans le contexte de la DB joueur. Ce sont des vues créées pour lire depuis la DB partagée, mais qui nécessitent un ATTACH préalable.

**À traiter** : Étape 3 (Architecture Shared DB) ou Étape 8 (Cleanup)

---

## Backups

| Backup | Chemin | Taille | Date |
|--------|--------|--------|------|
| v5_final (précédent) | `backups/v5_final/` | — | 15 fév 2026 |
| **v5.1 baseline** | `backups/v5.1_baseline_20260216/` | **317 MB** | 16 fév 2026 |

---

## Résumé Exécutif

| Critère | Valeur | Objectif v5.1 |
|---------|--------|---------------|
| Tests verts | 2819/2819 (100%) | Maintenir 100% |
| Imports SQLite src/ | 0 | 0 ✅ |
| Imports Pandas métier | 2 | 0 |
| Tables legacy/joueur | ~4-5 | 0 |
| Vues cassées/joueur | 4 | 0 |
| Taille moyenne player DB | ~57 MB | <5 MB |
| Couverture tests | À mesurer | ≥80% |

**Le système est STABLE et prêt pour la migration v5.1.**
