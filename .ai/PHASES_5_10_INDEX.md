# Index Rapide : Analyses Phases 5-10

> **Navigation rapide** vers les sections spécifiques des analyses
> 
> **Mis à jour** : 2026-02-16

---

## 📚 Documents Disponibles

| Document | Contenu | Pages |
|----------|---------|-------|
| **PHASES_5_10_ANALYSES.md** | Phase 5 complète | ~35 |
| **PHASES_6_10_COMPLETE.md** | Phases 6-10 complètes | ~35 |
| **Total** | | **~70** |

---

## 🔍 Index Phase 5 (PHASES_5_10_ANALYSES.md)

### teammates_service.py

| Fonction | Problème | Solution | Ligne |
|----------|----------|----------|-------|
| `load_teammate_stats()` | Lit DBs individuelles filesystem | Lire shared.match_participants avec xuid | Analyse complète |
| `enrich_series_with_perfect_kills()` | Fallback local medals | Shared uniquement | Analyse complète |
| `compute_participation_profiles()` | Mapping DB path flou | Expliciter xuid→DB path | Analyse complète |
| `load_impact_data()` | Fallback highlight_events | Shared uniquement | Analyse complète |

### _match_queries.py

| Fonction | Avant | Après | Gain |
|----------|-------|-------|------|
| `_get_match_source()` | UNION ALL 100 lignes | `"shared.match_participants"` | 97 lignes |
| `load_match_mmr_batch()` | player_match_stats local | shared.match_participants | Colonnes MMR |
| `get_match_count()` | COUNT local | COUNT DISTINCT shared | Filtrage xuid |

### _roster_loader.py

- **15 fallbacks** try/except à supprimer
- Pattern répétitif (local → shared)
- ~150 lignes économisées

---

## 🔍 Index Phase 6 (PHASES_6_10_COMPLETE.md)

### duckdb_repo.py - 8 Méthodes

| Méthode | Lignes | Action | Difficulté |
|---------|--------|--------|------------|
| `load_top_medals()` | 361-415 | Supprimer fallback V4 | 🟢 |
| `load_match_medals()` | 417-445 | Supprimer fallback V4 | 🟢 |
| `count_medal_by_match()` | 447-501 | Supprimer fallback V4 | 🟢 |
| `load_first_event_times()` | 524-590 | Supprimer fallback V4 | 🟡 |
| `load_highlight_events()` | 1175-1242 | Supprimer fallback V4 | 🟡 |
| `list_other_player_xuids()` | 1244-1316 | ✅ GARDER (4 sources) | N/A |
| `get_storage_info()` | 680-709 | Ajouter shared counts | 🟢 |
| `get_match_session_info()` | 1318-1345 | Ajouter shared fallback | 🟢 |

### UI Critiques

| Fichier | Problème | Solution |
|---------|----------|----------|
| `teammates_impact.py` | Accès direct `._get_connection()` | `._get_shared_connection()` |
| `objective_analysis.py` | ✅ Bon pattern | Adapter JOIN shared |
| ⚠️ `personal_performance.py` | **NON TROUVÉ** | Retirer de scope ? |

---

## 🔍 Index Phase 7 (PHASES_6_10_COMPLETE.md)

### 🔴 3 Bugs Critiques Identifiés

| # | Fichier | Ligne | Bug | Fix |
|---|---------|-------|-----|-----|
| 1 | `filters.py` | 370 | `.empty` (Pandas) | `.is_empty()` (Polars) |
| 2 | `filters_render.py` | 303 | Type hint: tuple[int, list, DataFrame] | Ajouter 4e élément |
| 3 | `checkbox_filter.py` | 189-191, 299-301 | "Aucun" sans confirmation | Ajouter dialog confirm |

### Solutions Complètes

Toutes les solutions avec code AVANT/APRÈS sont dans PHASES_6_10_COMPLETE.md

---

## 🔍 Index Phase 8 (PHASES_6_10_COMPLETE.md)

### 11 Modules Analysés

| Module | Statut | Action |
|--------|--------|--------|
| `killer_victim.py` | ✅ Safe | Aucune |
| `citations/engine.py` | ⚠️ | Ajouter fallback awards |
| `media_indexer.py` | ✅ Safe | Aucune |
| `participation_radar.py` | ✅ Safe | Aucune |
| `launcher.py` | ✅ Safe | Aucune |
| `multiplayer.py` | ⚠️ | Ajouter shared xuid_aliases |
| `cache_loaders.py` | ✅ Safe | Note |
| `cache_filters.py` | ✅ Safe | Aucune |
| **`aliases.py`** | 🔴 **CRITIQUE** | Fallback metadata |
| `data_loader.py` | ✅ Safe | Aucune |
| **`xuid.py`** | 🔴 **CRITIQUE** | Fallback metadata |

### 🔴 3 Modules Critiques

| Module | Fonction | Ligne | Problème | Solution |
|--------|----------|-------|----------|----------|
| `aliases.py` | `_load_aliases_from_duckdb_cached()` | 56-78 | Local seul | ATTACH metadata.duckdb |
| `xuid.py` | `resolve_xuid_from_db()` | 158-172 | Local seul | ATTACH metadata.duckdb |
| `citations/engine.py` | Awards loading | 447-459 | Local seul | Try shared.personal_score_awards |

**Code complet** dans PHASES_6_10_COMPLETE.md

---

## 🔍 Index Phase 9 (PHASES_6_10_COMPLETE.md)

### Tables

| Catégorie | Count | Liste |
|-----------|-------|-------|
| **À supprimer** | 8 | match_stats, medals_earned, highlight_events, match_participants, player_match_stats, xuid_aliases, killer_victim_pairs, teammates_aggregate |
| **Conservées** | 9 | player_match_enrichment, personal_score_awards, antagonists, match_citations, career_progression, media_files, media_match_associations, sync_meta, sessions |

### Script Validation

Script Python complet fourni dans le document pour vérifier absence tables interdites.

---

## 🔍 Index Phase 10 (PHASES_6_10_COMPLETE.md)

### 13 Fichiers Documentation

#### Architecture (P0) - 4 fichiers
1. `docs/ARCHITECTURE_V5.md`
2. `docs/SHARED_MATCHES_SCHEMA.md`
3. `docs/SQL_SCHEMA.md`
4. `docs/DATA_ARCHITECTURE.md`

#### Guides (P1-P2) - 5 fichiers
5. `docs/SYNC_GUIDE.md`
6. `docs/CLEANUP_V5.md`
7. `docs/CLEANUP_V5_QUICKSTART.md`
8. `docs/COMMANDS.md`
9. `docs/BACKUP_RESTORE.md`

#### Docs IA (P0-P1) - 4 fichiers
10. `CLAUDE.md`
11. `.github/copilot-instructions.md`
12. `.ai/project_map.md`
13. `.ai/data_lineage.md`

### 7 Points Critiques

1. match_stats n'existe plus dans player DBs
2. player_match_stats n'existe plus
3. xuid_aliases dans shared uniquement
4. player_match_enrichment SEULE table match
5. Coéquipiers depuis shared
6. Cleanup brutal intentionnel
7. Sync écrit enrichment + awards uniquement

---

## 🎯 Navigation Rapide

### Je cherche...

**...une fonction spécifique** :
- Ctrl+F dans PHASES_5_10_ANALYSES.md (Phase 5)
- Ctrl+F dans PHASES_6_10_COMPLETE.md (Phases 6-10)

**...un fichier spécifique** :
- Voir index ci-dessus
- Chercher nom fichier dans documents

**...un problème identifié** :
- Phase 7 : 3 bugs → PHASES_6_10_COMPLETE.md
- Phase 8 : 3 critiques → PHASES_6_10_COMPLETE.md

**...du code AVANT/APRÈS** :
- Toutes les phases ont exemples complets
- Format : Code actuel → Code après migration

**...les tests à créer** :
- Fin de chaque phase
- Section "Tests" dans documents

---

## 📊 Statistiques Globales

**Documentation totale** : ~70 pages
**Fichiers analysés** : 38
**Problèmes identifiés** : 6 critiques
**Solutions fournies** : 100%
**Code exemples** : Complets pour chaque modification
**Durée estimée** : 6 jours (phases 5-10)

---

**Bon travail avec la migration !** 🚀
