# Résumé Exécutif - Phases 5-10 Migration V5 Finale

**Date**: 16 février 2026  
**Statut**: Phases 0-4 ✅ Complétées | Phases 5-10 📋 Planifiées

---

## Vue d'ensemble

Les phases 5-10 complètent la migration V5 finale en:
1. Migrant tous les services et repositories vers shared DB
2. Adaptant toutes les pages UI
3. Supprimant les fallbacks locaux
4. Nettoyant brutalement les player DBs
5. Mettant à jour la documentation complète

**Impact global**: -630 LOC (simplification), 156 tests, 37 fichiers, 6 jours de travail

---

## Phase 5: Services & Repositories (partie 1) - 1 jour

### Fichiers modifiés
- `src/data/services/teammates_service.py` (~150 LOC)
- `src/data/repositories/_match_queries.py` (-140 LOC, simplification)
- `src/data/repositories/_roster_loader.py` (-650 LOC, simplification)

### Changements critiques

**teammates_service.py**:
- `load_teammate_stats()`: xuid au lieu de gamertag, lecture shared
- `enrich_series_with_perfect_kills()`: Batch loading depuis shared.medals_earned
- `compute_participation_profiles()`: ⚠️ **RESTRICTION** - Joueur principal uniquement
- `load_impact_data()`: shared.highlight_events + shared.match_participants

**_match_queries.py**:
- `_get_match_source()`: 170 → 60 lignes (suppression COALESCE)
- Toujours shared, erreur RuntimeError si absent
- Suppression fallbacks player_match_stats

**_roster_loader.py**:
- Suppression des 15 fallbacks locaux
- 930 → 280 lignes (-70%)
- Erreurs explicites si tables shared absentes

### Risques
🔴 **HAUTE**: Restriction fonctionnelle - profils radar coéquipiers désactivés

### Tests requis
42 tests (15 unitaires teammates, 12 match_queries, 15 roster_loader)

---

## Phase 6: Repositories (partie 2) + UI Critique - 1 jour

### Fichiers modifiés
- `src/data/repositories/duckdb_repo.py` (8 méthodes, ~200 LOC)
- `src/data/repositories/_materialized_views.py` (-300 LOC, **SUPPRIMÉ**)
- `src/ui/pages/teammates_impact.py` (~50 LOC)
- `src/ui/pages/objective_analysis.py` (~20 LOC)

### Changements critiques

**8 méthodes duckdb_repo.py**:
1. `load_top_medals()`: Toggle "inclure médailles équipe" (nouveau paramètre)
2. `load_match_medals()`: Shared uniquement
3. `count_medal_by_match()`: Shared uniquement
4. `load_first_event_times()`: Gestion killer_xuid/victim_xuid
5. `load_highlight_events()`: ⚠️ **COMPLEXE** - CASE normalization dual-column
6. `list_other_player_xuids()`: Self-join shared.match_participants
7. `get_storage_info()`: Inchangé (diagnostics)
8. `get_match_session_info()`: Inchangé (session_id local)

**_materialized_views.py**: Suppression complète, remplacé par @lru_cache

**teammates_impact.py**: shared.highlight_events avec CASE

**objective_analysis.py**: JOIN shared.match_registry (start_time)

### Décisions de conception
- Toggle team medals: Default player, option team
- Suppression vues matérialisées → requêtes cachées
- personal_score_awards reste locale

### Tests requis
18 tests (12 duckdb_repo, 4 teammates_impact, 2 objective_analysis)

---

## Phase 7: UI Complète + Filtres - 1 jour

### Fichiers modifiés
- `src/ui/pages/citations.py` (+150 LOC, reconstruction mappings)
- `src/ui/pages/personal_performance.py` (~30 LOC)
- `src/ui/pages/media_library.py` (~20 LOC)
- `src/app/filters.py` (~40 LOC, Polars natif)
- `src/app/filters_render.py` (~20 LOC)
- `src/ui/components/checkbox_filter.py` (~30 LOC)
- `scripts/sync.py` (-50 LOC, suppressions)

### Changements critiques

**citations.py**: 
- Fonction `rebuild_citation_mappings_v5()` depuis match_citations
- Restaure progressions (5/10, 8/25) et compteur delta +XXX
- Filtre 159 → 48 citations pertinentes

**filters.py**: 
- Polars natif (pas de conversion Pandas)
- `render_session_filters_v5()` avec pl.DataFrame

**checkbox_filter.py**: 
- Option `preserve_out_of_range=True`
- Conserve sélections hors période affichée

**sync.py**: 
- Suppression `rebuild_teammates_aggregate()` (obsolète)
- `print_stats_v5()` lit shared + player_match_enrichment

### Tests requis
21 tests (6 citations, 4 filters, 3 checkbox, 3 sync, 2+1+2 autres)

---

## Phase 8: Modules Secondaires - 1 jour

### Fichiers modifiés (10)
- `src/analysis/killer_victim.py`: TypedDict corrections
- `src/analysis/citations/engine.py`: Requêtes shared
- `src/data/media_indexer.py`: shared.match_registry
- `src/visualization/participation_radar.py`: Shared
- `launcher.py`, `src/ui/multiplayer.py`: Player discovery shared
- `src/ui/cache_loaders.py`, `cache_filters.py`, `aliases.py`: Shared
- `src/app/data_loader.py`, `src/utils/xuid.py`: Utilities

### Changements
- killer_victim.py: `.kills` → `["kills"]` (TypedDict)
- Tous les modules: FROM match_stats → shared queries
- Performance: Moins de fallbacks, requêtes directes

### Tests requis
25 tests (répartis sur 10 fichiers)

---

## Phase 9: Validation + Cleanup Brutal - 1 jour

### Stratégie

**Étape 1**: Tests anti-régression
```bash
pytest tests/test_v5_final_migration.py -v
```

**Étape 2**: Tests manuels UI (toutes pages)

**Étape 3**: Sync test
```bash
python scripts/sync.py --player TestPlayer --delta --max-matches 50
```

**Étape 4**: Cleanup brutal
```bash
# Dry run OBLIGATOIRE
python scripts/cleanup_player_dbs_v5.py --all --dry-run --verbose

# Vérifier couverture 100%

# Cleanup avec backup
python scripts/cleanup_player_dbs_v5.py --all --backup
```

**Étape 5**: Validation post-cleanup
- Relancer app
- Vérifier 0 erreurs "table introuvable"
- Si erreurs → identifier code résiduel

### Checklist validation
- [ ] 156 tests passent
- [ ] 0 `grep "FROM match_stats" src/`
- [ ] App fonctionne après cleanup
- [ ] Sync fonctionne
- [ ] Aucune table interdite dans player DBs
- [ ] Tailles DBs réduites (~30% original)

### Tests requis
50 tests (anti-régression + validation)

---

## Phase 10: Documentation - 1 jour

### Documents à mettre à jour (13)

**P0 - Architecture** (5):
1. `docs/ARCHITECTURE_V5.md`: Schéma état cible, 31 colonnes
2. `docs/SHARED_MATCHES_SCHEMA.md`: DDL complet
3. `docs/SQL_SCHEMA.md`: Tables supprimées
4. `CLAUDE.md`: Tables player DB, colonnes shared
5. `.github/copilot-instructions.md`: Même màj

**P1 - Guides** (5):
6. `docs/SYNC_GUIDE.md`: Flux sync V5
7. `docs/CLEANUP_V5.md`: 8 tables supprimées
8. `docs/COMMANDS.md`: Nouvelles options
9. `.ai/project_map.md`: Cartographie
10. `.ai/data_lineage.md`: Flux données

**P2 - Référence** (3):
11. `docs/BACKUP_RESTORE.md`: Backups plus petits
12. `.ai/thought_log.md`: Décisions
13. `CHANGELOG.md`: Restrictions

### Contenu clé à documenter
- 31 colonnes match_participants (15 base + 9 stats + 7 MMR)
- 8 tables supprimées de player DBs
- Restriction: profils radar joueur principal uniquement
- Flux: sync → shared + player_match_enrichment

---

## Synthèse Globale

### Métriques

| Métrique | Valeur |
|----------|--------|
| Fichiers modifiés | 37 |
| LOC nettes | -630 (simplification) |
| Tests à créer/adapter | 156 |
| Jours de travail | 6 (ou 12 conservateur) |
| Risque global | 🟡 MOYEN-ÉLEVÉ |

### Décisions critiques

1. **personal_score_awards**: Garder locale
   - Profils coéquipiers désactivés (restriction UX)
   - Migration possible Phase 11 future

2. **Médailles**: Toggle UI player/team
   - Default: player uniquement
   - Option: inclure équipe

3. **Citations**: Rebuild depuis match_citations
   - Cache @st.cache_data
   - Restaure progressions + delta

### Bénéfices

✅ Architecture simplifiée (shared autoritaire)  
✅ 70% réduction code roster_loader  
✅ Suppression 15 fallbacks sources de bugs  
✅ Performance: moins de requêtes conditionnelles  
✅ Détection forcée code résiduel (cleanup brutal)

### Risques majeurs

⚠️ **Risque #1**: Performance shared DB (self-joins)  
**Mitigation**: Index (xuid, match_id), ANALYZE après sync

⚠️ **Risque #2**: Cleanup prématuré → perte données  
**Mitigation**: Couverture 100% obligatoire, backup --backup

⚠️ **Risque #3**: Code résiduel post-cleanup  
**Mitigation**: Tests anti-régression, grep, cleanup brutal intentionnel

### Ordre d'implémentation

1. ✅ Phase 5 D'ABORD (services = fondation)
2. ✅ Phase 6 ENSUITE (repos = couche data)
3. ✅ Phase 7 + 8 PARALLÈLE (UI indépendantes)
4. ✅ Phase 9 SÉQUENTIELLEMENT (validation bloque cleanup)
5. ✅ Phase 10 EN CONTINU (doc au fur et à mesure)

### Plan de rollback

Si problème critique post-Phase 9:
1. Arrêter l'application
2. Restaurer backups Phase 0
3. Revenir au code pré-Phase 5
4. Relancer app

**Fenêtre rollback**: 1 mois après Phase 9

---

## Prochaines étapes

1. ✅ Valider ce plan avec l'équipe
2. ✅ Créer branche `feature/v5-final-phases-5-10`
3. ✅ Démarrer Phase 5 (services teammates)
4. ✅ Tests continus après chaque fichier
5. ✅ Documentation inline (docstrings)

---

**Conclusion**: Migration V5 finale prête pour implémentation. Risque gérable avec tests robustes. Bénéfices architecture substantiels.

✅ **GO/NO-GO**: **GO** (avec précautions cleanup brutal)
