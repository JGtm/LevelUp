# Guide d'Implémentation - Phases 5-10 Migration V5 Finale

**Date**: 16 février 2026  
**Auteur**: Analyse IA approfondie  
**Status**: 📋 Prêt pour implémentation

---

## Utilisation de ce guide

Ce document fournit des instructions step-by-step pour implémenter les phases 5-10. Chaque phase contient:
- ✅ Checklist de tâches
- 📝 Code snippets de référence
- ⚠️ Points de vigilance
- 🧪 Tests requis
- ✔️ Critères de validation

**Documents complémentaires**:
- `BUGFIX_V5_2026-02-15.md`: Contexte complet, phases 0-4
- `BUGFIX_V5_PHASES_5-10_ANALYSIS.md`: Analyses techniques détaillées
- `PHASES_5-10_SUMMARY.md`: Résumé exécutif

---

## Phase 5: Services & Repositories (partie 1)

**Durée estimée**: 1 jour  
**Risque**: 🔴 HAUTE  
**Priorité**: P0 (fondation pour phases suivantes)

### Checklist

#### teammates_service.py

- [ ] **5.1** Créer `load_teammate_stats_v5(teammate_xuid, match_ids, shared_db_path)`
  - [ ] Ajouter résolution gamertag depuis shared.xuid_aliases
  - [ ] Requête shared.match_participants avec JOIN match_registry
  - [ ] Retourner TeammateStats avec Polars DataFrame
  - [ ] Tests: 4 unitaires (xuid valide, invalide, match_ids vides, gamertag non trouvé)

- [ ] **5.2** Créer `enrich_series_with_perfect_kills_v5(series, shared_db_path, xuid_map)`
  - [ ] Batch query shared.medals_earned (1 requête pour tous joueurs)
  - [ ] Mapper (match_id, xuid) → perfect_count
  - [ ] Ajouter colonne perfect_kills à chaque DataFrame
  - [ ] Tests: 3 unitaires (single player, multiple players, aucun perfect kill)

- [ ] **5.3** Modifier `compute_participation_profiles()` → joueur principal uniquement
  - [ ] Renommer `compute_participation_profiles_v5(main_player_xuid, main_player_df, ...)`
  - [ ] Retirer paramètre `players_data` (liste), accepter joueur unique
  - [ ] Documenter restriction: "Profils coéquipiers désactivés en V5"
  - [ ] Tests: 2 unitaires (joueur avec awards, joueur sans awards)

- [ ] **5.4** Créer `load_impact_data_v5(shared_db_path, xuid, match_ids, friend_xuids)`
  - [ ] Requête shared.highlight_events avec CASE killer_xuid/victim_xuid
  - [ ] Requête shared.match_participants pour outcome (pas match_registry)
  - [ ] Gérer event_type avec LOWER() pour robustesse
  - [ ] Tests: 4 unitaires (kills, deaths, outcomes mixtes, match_ids vides)

- [ ] **5.5** Mettre à jour appelants
  - [ ] `src/ui/pages/teammates.py`: Ajouter résolution xuid avant appels
  - [ ] Ajouter warning UI: "Profils radar: joueur principal uniquement"
  - [ ] Tests UI: 3 tests (affichage profil, warning visible, pas de crash coéquipiers)

#### _match_queries.py

- [ ] **5.6** Simplifier `_get_match_source()`
  - [ ] Supprimer toute logique COALESCE (170 → 60 lignes)
  - [ ] Retourner subquery shared.match_registry + match_participants uniquement
  - [ ] Ajouter RuntimeError si xuid absent ou shared tables absentes
  - [ ] Tests: 5 unitaires (shared OK, xuid manquant, table registry manquante, participants manquant, colonnes valides)

- [ ] **5.7** Simplifier `load_match_mmr_batch()`
  - [ ] Supprimer fallback player_match_stats
  - [ ] Requête directe shared.match_participants (team_mmr, enemy_mmr)
  - [ ] Tests: 2 unitaires (batch OK, match_ids vides)

- [ ] **5.8** Simplifier `get_match_count()`
  - [ ] COUNT DISTINCT shared.match_participants WHERE xuid = ?
  - [ ] Tests: 2 unitaires (count > 0, xuid sans matchs)

#### _roster_loader.py

- [ ] **5.9** Simplifier `load_match_rosters()`
  - [ ] Supprimer fallbacks highlight_events (lignes 230-319)
  - [ ] Query unique shared.killer_victim_pairs
  - [ ] RuntimeError si table absente
  - [ ] Tests: 3 unitaires (rosters OK, table manquante, match_id invalide)

- [ ] **5.10** Simplifier `load_matches_with_teammate()`, `load_same_team_match_ids()`
  - [ ] Supprimer fallbacks locaux (total -400 lignes)
  - [ ] Queries shared.match_participants uniquement
  - [ ] Tests: 4 unitaires (2 par fonction)

- [ ] **5.11** Simplifier `resolve_gamertag()`, `load_match_player_gamertags()`
  - [ ] Cascade shared.match_participants → shared.xuid_aliases (2 niveaux max)
  - [ ] Supprimer fallback highlight_events (ASCII extraction)
  - [ ] Tests: 6 unitaires (3 par fonction, success/fail/edge cases)

- [ ] **5.12** Simplifier `load_match_players_stats()`
  - [ ] Requête unique shared.match_participants
  - [ ] Tests: 2 unitaires (stats OK, match sans stats)

### Validation Phase 5

```bash
# Tests unitaires
pytest tests/test_teammates_service.py -v  # 15 tests
pytest tests/test_match_queries.py -v      # 12 tests
pytest tests/test_roster_loader.py -v      # 15 tests

# Tests d'intégration
pytest tests/integration/test_services_v5.py -v

# Vérification code
grep -r "FROM match_stats" src/data/services/ src/data/repositories/
# Expected: 0 résultats

grep -r "player_match_stats" src/data/services/ src/data/repositories/
# Expected: 0 résultats (hors commentaires)
```

✅ **Critères de succès Phase 5**:
- 42 tests passent
- 0 références match_stats/player_match_stats dans services/repositories
- Aucune erreur sur page teammates (avec warning profils coéquipiers)

---

## Phase 6: Repositories (partie 2) + UI Critique

**Durée estimée**: 1 jour  
**Risque**: 🔴 HAUTE  
**Priorité**: P0

### Checklist

#### duckdb_repo.py (8 méthodes)

- [ ] **6.1** Ajouter paramètre `include_team` à `load_top_medals()`
  - [ ] Default: False (médailles player uniquement)
  - [ ] If True: requête sans filtre xuid
  - [ ] Tests: 3 (player only, team, empty)

- [ ] **6.2** Supprimer fallbacks locaux dans 6 méthodes
  - [ ] `load_match_medals()`: shared uniquement
  - [ ] `count_medal_by_match()`: shared uniquement
  - [ ] `load_first_event_times()`: shared.highlight_events avec killer_xuid/victim_xuid
  - [ ] `load_highlight_events()`: CASE normalization (COMPLEXE)
  - [ ] `list_other_player_xuids()`: self-join shared.match_participants
  - [ ] Tests: 9 (1-2 par méthode)

- [ ] **6.3** `load_highlight_events()` - Migration critique
  - [ ] Créer query avec CASE pour normaliser xuid/gamertag
  - [ ] Gérer event_type avec LOWER()
  - [ ] Retourner tous events (pas de filtre xuid dans query)
  - [ ] Tests: 4 (kills, deaths, event_type mixtes, NULL handling)

#### _materialized_views.py

- [ ] **6.4** Supprimer fichier complet
  - [ ] Créer méthodes repository avec @lru_cache
  - [ ] `get_match_stats_summary_v5()`, `get_weapon_stats_v5()`, etc.
  - [ ] Tests: 0 (méthodes triviales, cache testé ailleurs)

#### UI Pages

- [ ] **6.5** teammates_impact.py
  - [ ] Modifier requête highlight_events avec CASE
  - [ ] Outcome depuis shared.match_participants (pas match_stats)
  - [ ] Tests: 4 (affichage heatmap, ranking, metrics, graceful degradation)

- [ ] **6.6** objective_analysis.py
  - [ ] Modifier JOIN: personal_score_awards + shared.match_registry
  - [ ] Vérifier présence shared.match_registry
  - [ ] Tests: 2 (affichage OK, shared manquant)

### Validation Phase 6

```bash
pytest tests/test_duckdb_repo.py::test_load_highlight_events -v
pytest tests/test_ui_teammates_impact.py -v
pytest tests/test_ui_objective_analysis.py -v

# Vérification suppression
ls src/data/repositories/_materialized_views.py
# Expected: Fichier non trouvé
```

✅ **Critères de succès Phase 6**:
- 18 tests passent
- _materialized_views.py supprimé
- Pages teammates_impact et objective_analysis fonctionnelles

---

## Phase 7: UI Complète + Filtres

**Durée estimée**: 1 jour  
**Risque**: 🟡 MOYENNE  
**Priorité**: P1

### Checklist

#### citations.py

- [ ] **7.1** Créer `rebuild_citation_mappings_v5(player_db_path, xuid)`
  - [ ] Query match_citations (table locale conservée)
  - [ ] Charger halo5_commendations_fr.json
  - [ ] Calculer progressions avec niveaux
  - [ ] Cache avec @st.cache_data
  - [ ] Tests: 6 (progressions OK, JSON manquant, match_citations vide, levels edge cases)

- [ ] **7.2** Modifier affichage citations
  - [ ] Filtrer citations avec progression > 0
  - [ ] Restaurer compteur delta +XXX
  - [ ] Afficher current/target (ex: 5/10)
  - [ ] Tests UI: 3 (affichage, delta, filtrage)

#### Filtres

- [ ] **7.3** Migrer filters.py vers Polars natif
  - [ ] `render_session_filters_v5(df_polars)` sans conversion Pandas
  - [ ] Tests: 4 (filtrage OK, sessions vides, sélections multiples, Polars types)

- [ ] **7.4** Ajouter preserve_out_of_range à checkbox_filter
  - [ ] Paramètre preserve_out_of_range=True par défaut
  - [ ] Afficher warning si sélections hors options
  - [ ] Tests: 3 (preserve OK, no preserve, warning visible)

#### sync.py

- [ ] **7.5** Supprimer `rebuild_teammates_aggregate()`
  - [ ] Ligne ~650: DELETE fonction complète
  - [ ] Tests: 0 (stub obsolète)

- [ ] **7.6** Créer `print_stats_v5(player_db_path, xuid)`
  - [ ] Count shared.match_participants
  - [ ] Count player_match_enrichment
  - [ ] Tests: 3 (stats OK, shared vide, enrichment vide)

### Validation Phase 7

```bash
pytest tests/test_citations.py -v
pytest tests/test_filters.py -v
pytest tests/test_sync.py::test_print_stats_v5 -v

# Test UI manuel
streamlit run streamlit_app.py
# Naviguer: Citations page, vérifier progressions + delta
# Naviguer: Filtres, vérifier sélections préservées
```

✅ **Critères de succès Phase 7**:
- 21 tests passent
- Citations affichent progressions + delta
- Filtres préservent sélections hors période

---

## Phase 8: Modules Secondaires

**Durée estimée**: 1 jour  
**Risque**: 🟡 MOYENNE  
**Priorité**: P2

### Checklist

- [ ] **8.1** killer_victim.py: Corriger TypedDict (`.kills` → `["kills"]`)
- [ ] **8.2** citations/engine.py: FROM shared queries, colonnes V5
- [ ] **8.3** media_indexer.py: FROM shared.match_registry
- [ ] **8.4** participation_radar.py: FROM shared
- [ ] **8.5** launcher.py: Player discovery shared
- [ ] **8.6** multiplayer.py: list_duckdb_v4_players() → shared
- [ ] **8.7** cache_loaders.py, cache_filters.py, aliases.py: Shared queries
- [ ] **8.8** data_loader.py, xuid.py: Utilities shared

**Tests**: 25 (2-3 par fichier)

### Validation Phase 8

```bash
pytest tests/test_killer_victim.py -v
pytest tests/test_citations_engine.py -v
pytest tests/test_media.py -v
# ... (tous modules secondaires)

# Grep verification
grep -r "FROM match_stats" src/analysis/ src/ui/ src/app/ src/utils/
# Expected: 0 résultats
```

---

## Phase 9: Validation + Cleanup Brutal

**Durée estimée**: 1 jour (+ iterations)  
**Risque**: 🟡 MOYENNE  
**Priorité**: P0 (bloquant pour Phase 10)

### Checklist

#### Étape 1: Tests anti-régression

- [ ] **9.1** Créer `tests/test_v5_final_migration.py`
  - [ ] test_no_from_match_stats_in_src()
  - [ ] test_no_local_medals_earned_read()
  - [ ] test_no_local_highlight_events_read()
  - [ ] test_no_local_xuid_aliases_read()
  - [ ] test_no_player_match_stats_in_src()
  - [ ] test_sync_engine_writes_player_match_enrichment()
  - [ ] Tests: 10+

- [ ] **9.2** Exécuter suite complète
```bash
pytest tests/ --ignore=tests/integration --ignore=tests/performance -v
# Expected: 156+ tests PASS, 0 FAIL
```

#### Étape 2: Tests manuels UI

- [ ] **9.3** Naviguer toutes pages
  - [ ] Home / Dashboard
  - [ ] Timeseries
  - [ ] Teammates (avec warning profils)
  - [ ] Teammates Impact
  - [ ] Objective Analysis
  - [ ] Citations (progressions + delta)
  - [ ] Media Library
  - [ ] Personal Performance
  - [ ] Match View

#### Étape 3: Sync test

- [ ] **9.4** Sync de validation
```bash
python scripts/sync.py --player TestPlayer --delta --max-matches 50
```
- [ ] Vérifier shared.match_participants a nouvelles lignes
- [ ] Vérifier player_match_enrichment a nouvelles lignes
- [ ] Vérifier 0 erreurs

#### Étape 4: Cleanup brutal

- [ ] **9.5** Dry run obligatoire
```bash
python scripts/cleanup_player_dbs_v5.py --all --dry-run --verbose
```
- [ ] Vérifier couverture 100% pour TOUS les joueurs
- [ ] Vérifier liste tables à supprimer (8 tables)

- [ ] **9.6** Cleanup avec backup
```bash
python scripts/cleanup_player_dbs_v5.py --all --backup
```

#### Étape 5: Validation post-cleanup

- [ ] **9.7** Relancer app
```bash
streamlit run streamlit_app.py
```

- [ ] **9.8** Vérifier 0 erreurs "table introuvable"
  - Si erreurs → identifier fichier:ligne dans stack trace
  - Corriger code résiduel
  - Re-tester
  - Re-cleanup si nécessaire

- [ ] **9.9** Vérifier tailles DBs
```python
import json
from pathlib import Path
import duckdb

profiles = json.loads(Path('db_profiles.json').read_text())['profiles']
for gt, p in profiles.items():
    db = Path(p['db_path'])
    if db.exists():
        size_mb = db.stat().st_size / 1024 / 1024
        print(f"{gt}: {size_mb:.1f} MB")
```
- Expected: ~30% taille originale

#### Étape 6: Script vérification finale

- [ ] **9.10** Créer & exécuter `tests/test_cleanup_verification.py`
```bash
pytest tests/test_cleanup_verification.py -v
```

### Validation Phase 9

✅ **Critères de succès Phase 9**:
- [ ] 156+ tests passent (suite complète)
- [ ] 0 `grep "FROM match_stats" src/`
- [ ] App fonctionne post-cleanup (toutes pages)
- [ ] Sync fonctionne (10 matchs test OK)
- [ ] 0 tables interdites dans player DBs
- [ ] Tailles DBs réduites (~30%)

---

## Phase 10: Documentation

**Durée estimée**: 1 jour  
**Risque**: 🟢 FAIBLE  
**Priorité**: P1

### Checklist

#### P0 - Architecture

- [ ] **10.1** docs/ARCHITECTURE_V5.md
  - [ ] Section "État cible post-cleanup"
  - [ ] Schéma 31 colonnes match_participants
  - [ ] Flux sync V5 (1 sync = shared + enrichment)
  - [ ] Tables supprimées (8) et conservées (9)

- [ ] **10.2** docs/SHARED_MATCHES_SCHEMA.md
  - [ ] DDL complet match_participants (31 colonnes)
  - [ ] Index recommandés
  - [ ] Constraints

- [ ] **10.3** docs/SQL_SCHEMA.md
  - [ ] Mise à jour schéma player DBs (9 tables)
  - [ ] Suppression références match_stats, etc.

- [ ] **10.4** CLAUDE.md
  - [ ] Section "Tables DuckDB Principales"
  - [ ] Architecture Multi-Joueurs (shared, pas DBs individuelles)
  - [ ] Restrictions: profils coéquipiers

- [ ] **10.5** .github/copilot-instructions.md
  - [ ] Même màj que CLAUDE.md

#### P1 - Guides

- [ ] **10.6** docs/SYNC_GUIDE.md
  - [ ] Flux sync V5 (shared + enrichment)
  - [ ] Exemples commandes

- [ ] **10.7** docs/CLEANUP_V5.md
  - [ ] 8 tables supprimées
  - [ ] Option --skip-coverage-check
  - [ ] Procédure complète

- [ ] **10.8** docs/COMMANDS.md
  - [ ] Nouvelles options backfill (--participants-enrich)
  - [ ] Cleanup commands

- [ ] **10.9** .ai/project_map.md, .ai/data_lineage.md
  - [ ] Flux données V5
  - [ ] Cartographie fichiers modifiés

#### P2 - Référence

- [ ] **10.10** docs/BACKUP_RESTORE.md
  - [ ] Backups plus petits post-cleanup

- [ ] **10.11** .ai/thought_log.md
  - [ ] Décisions: stop dual write, cleanup brutal
  - [ ] Justifications restrictions

- [ ] **10.12** CHANGELOG.md
  - [ ] Section V5 Finale
  - [ ] Breaking changes: profils coéquipiers désactivés
  - [ ] Migration path

### Validation Phase 10

```bash
# Vérifier cohérence
diff docs/ARCHITECTURE_V5.md CLAUDE.md
# Expected: Schémas match_participants identiques

# Vérifier complétude
grep -l "match_participants.*31 colonnes" docs/*.md CLAUDE.md
# Expected: 3+ fichiers

# Vérifier restrictions documentées
grep -l "profils coéquipiers désactivés" CHANGELOG.md docs/*.md
# Expected: 2+ fichiers
```

✅ **Critères de succès Phase 10**:
- [ ] 13 documents mis à jour
- [ ] Schéma 31 colonnes documenté (≥3 fichiers)
- [ ] Restrictions UX documentées
- [ ] CHANGELOG complet

---

## Procédure de rollback (si nécessaire)

**Quand utiliser**: Problème critique post-Phase 9, app inutilisable

### Étapes

1. **Arrêter l'application**
```bash
pkill -f streamlit
# ou CTRL+C si lancé en foreground
```

2. **Restaurer backups Phase 0**
```bash
python scripts/restore_player.py --all --backup backups/v5_final/
```

3. **Revenir au code pré-Phase 5**
```bash
git log --oneline | head -20  # Trouver commit pré-Phase 5
git revert <commit-hash>
# ou
git reset --hard <commit-pré-phase-5>
git push --force origin <branch>
```

4. **Relancer app**
```bash
streamlit run streamlit_app.py
```

5. **Vérifier fonctionnement**
- Naviguer toutes pages
- Vérifier données affichées
- Sync test

**⚠️ Fenêtre de rollback**: 1 mois après Phase 9 (conserver backups)

---

## Ressources et références

### Documents projet
- [BUGFIX_V5_2026-02-15.md](BUGFIX_V5_2026-02-15.md): Contexte complet, phases 0-4
- [BUGFIX_V5_PHASES_5-10_ANALYSIS.md](BUGFIX_V5_PHASES_5-10_ANALYSIS.md): Analyses techniques
- [PHASES_5-10_SUMMARY.md](PHASES_5-10_SUMMARY.md): Résumé exécutif

### Tests
- `tests/test_v5_final_migration.py`: Tests anti-régression
- `tests/test_cleanup_verification.py`: Vérification post-cleanup
- `tests/integration/test_services_v5.py`: Tests d'intégration

### Scripts
- `scripts/cleanup_player_dbs_v5.py`: Cleanup brutal
- `scripts/backup_player.py`: Backups
- `scripts/restore_player.py`: Restauration

---

## Checklist globale de progression

### Phase 5 (1 jour)
- [ ] teammates_service.py (5 fonctions)
- [ ] _match_queries.py (3 fonctions)
- [ ] _roster_loader.py (7 fonctions)
- [ ] Tests: 42

### Phase 6 (1 jour)
- [ ] duckdb_repo.py (8 méthodes)
- [ ] _materialized_views.py (suppression)
- [ ] teammates_impact.py, objective_analysis.py
- [ ] Tests: 18

### Phase 7 (1 jour)
- [ ] citations.py (rebuild mappings)
- [ ] filters.py (Polars natif)
- [ ] checkbox_filter.py (preserve)
- [ ] sync.py (suppressions)
- [ ] Tests: 21

### Phase 8 (1 jour)
- [ ] 10 modules secondaires
- [ ] Tests: 25

### Phase 9 (1-2 jours)
- [ ] Tests anti-régression
- [ ] Tests manuels UI
- [ ] Sync test
- [ ] Cleanup brutal
- [ ] Validation post-cleanup
- [ ] Tests: 50

### Phase 10 (1 jour)
- [ ] 13 documents
- [ ] Cohérence vérifiée

---

**Total estimation**: 6-8 jours ouvrés  
**Risque global**: 🟡 MOYEN-ÉLEVÉ (gérable avec tests)  
**Go/No-Go**: ✅ **GO**

---

**Dernière mise à jour**: 16 février 2026  
**Version**: 1.0  
**Statut**: Prêt pour implémentation
