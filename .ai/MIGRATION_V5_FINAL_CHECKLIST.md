# Checklist Migration V5 Finale - Suivi Quotidien

> **Utilisation** : Cocher les items au fur et à mesure de la progression
> 
> **Mis à jour** : 2026-02-16

---

## ✅ Phase 0 : Pré-requis et analyse (COMPLÉTÉE)

- [x] Backup complet des DBs
- [x] Baseline de tests sauvegardée
- [x] Analyse de contexte effectuée
- [x] Script de migration créé et testé

**Date de complétion** : _________

---

## ✅ Phase 1 : ALTER TABLE shared + backfill (COMPLÉTÉE)

- [x] ALTER TABLE shared.match_participants (+16 colonnes)
- [x] player_match_enrichment créée pour chaque joueur
- [x] Backfill données locales → shared
- [x] Vérification couverture 100%
- [x] INSERT schema_version V6

**Date de complétion** : _________

---

## ✅ Phase 2 : Transformers, modèles, batch_insert (COMPLÉTÉE)

- [x] MatchParticipantRow étendu (31 champs)
- [x] extract_participants() extrait 16 colonnes
- [x] transform_all_skill_stats() créé
- [x] PARTICIPANT_COLUMNS étendu
- [x] Tests unitaires passent

**Date de complétion** : _________

---

## ✅ Phase 3 : Sync engine - stop dual write (COMPLÉTÉE)

- [x] Sync n'écrit plus dans tables locales
- [x] Écriture shared + player_match_enrichment uniquement
- [x] Schémas player DB allégés
- [x] _compute_and_update_performance_score() adapté
- [x] Sync de test validé

**Date de complétion** : _________

---

## ✅ Phase 4 : Scripts backfill (COMPLÉTÉE)

- [x] Scripts redirigés vers shared
- [x] Mode --participants-enrich créé
- [x] Fallbacks locaux supprimés
- [x] detection.py adapté
- [x] orchestrator.py adapté
- [x] Scripts testés

**Date de complétion** : _________

---

## 📋 Phase 5 : Services et repositories (partie 1)

### 5.1 teammates_service.py

- [ ] Analyser load_teammate_stats()
- [ ] Réécrire load_teammate_stats() (xuid, shared)
- [ ] Tester nouvelle signature
- [ ] Modifier enrich_series_with_perfect_kills()
- [ ] Supprimer fallbacks
- [ ] Tester enrichment
- [ ] Adapter compute_participation_profiles()
- [ ] Créer mapping xuid → DB path
- [ ] Tester profiles
- [ ] Modifier load_impact_data()
- [ ] Supprimer fallback highlight_events
- [ ] Tester impact data
- [ ] Mettre à jour src/ui/pages/teammates.py
- [ ] Tester page Coéquipiers dans l'app

### 5.2 _match_queries.py

- [ ] Analyser _get_match_source()
- [ ] Simplifier _get_match_source() (retourner shared)
- [ ] Tester simplification
- [ ] Modifier load_match_mmr_batch()
- [ ] Lire colonnes MMR depuis shared
- [ ] Tester MMR batch
- [ ] Modifier get_match_count()
- [ ] Compter dans shared avec DISTINCT
- [ ] Tester count

### 5.3 _roster_loader.py

- [ ] Identifier les 15 fallbacks (grep -n "try:")
- [ ] Supprimer fallback load_player_roster()
- [ ] Supprimer fallback load_enemy_roster()
- [ ] Supprimer fallback load_teammate_names()
- [ ] Supprimer fallback get_player_aliases()
- [ ] Supprimer fallback get_team_composition()
- [ ] Supprimer fallback load_participant_list()
- [ ] Supprimer fallback get_player_medals()
- [ ] Supprimer fallback get_player_events()
- [ ] Supprimer 7 autres fallbacks restants
- [ ] Tester chaque méthode

### 5.4 Tests Phase 5

- [ ] Créer tests/test_teammates_service_v5.py
- [ ] Test load_teammate_stats_from_shared()
- [ ] Test enrich_perfect_kills_no_fallback()
- [ ] Test compute_profiles_with_mapping()
- [ ] Test load_impact_data_shared_only()
- [ ] Créer tests/test_match_queries_v5.py
- [ ] Test _get_match_source_simplified()
- [ ] Test load_mmr_from_shared()
- [ ] Test get_match_count_shared()
- [ ] Créer tests/test_roster_loader_v5.py
- [ ] Test no_fallbacks_in_roster()
- [ ] Exécuter pytest tests/test_*_v5.py -v

### 5.5 Validation Phase 5

- [ ] Tests unitaires passent
- [ ] Page Coéquipiers fonctionne
- [ ] Page Adversaires fonctionne
- [ ] Pas d'erreurs console
- [ ] Pas d'accès DBs locales (vérifier logs)

**Date début** : _________
**Date fin** : _________
**Durée effective** : _________ heures

---

## ⏸️ Phase 6 : Repositories (partie 2) + UI critique

### 6.1 duckdb_repo.py (8 méthodes)

- [ ] Modifier load_top_medals() → shared.medals_earned
- [ ] Modifier load_match_medals() → shared.medals_earned
- [ ] Modifier count_medal_by_match() → shared.medals_earned
- [ ] Modifier load_first_event_times() → shared.highlight_events
- [ ] Modifier load_highlight_events() → shared.highlight_events
- [ ] Modifier list_other_player_xuids() → shared.match_participants
- [ ] Modifier get_storage_info() (tables conservées)
- [ ] Modifier get_match_session_info() → player_match_enrichment
- [ ] Tester les 8 méthodes

### 6.2 _materialized_views.py

- [ ] Identifier les 4 requêtes FROM match_stats
- [ ] Modifier requête 1 → shared
- [ ] Modifier requête 2 → shared
- [ ] Modifier requête 3 → shared
- [ ] Modifier requête 4 → shared
- [ ] Tester vues matérialisées

### 6.3 UI critiques

- [ ] Modifier teammates_impact.py → shared.highlight_events
- [ ] Tester page Impact Coéquipiers
- [ ] Modifier objective_analysis.py → shared
- [ ] Tester page Analyse Objectifs

### 6.4 Tests Phase 6

- [ ] Mettre à jour tests/test_duckdb_repo.py
- [ ] Tester load_medals
- [ ] Tester load_events
- [ ] Tester xuids
- [ ] Tests UI passent

### 6.5 Validation Phase 6

- [ ] Tests passent
- [ ] Pages UI critiques OK
- [ ] Navigation complète app

**Date début** : _________
**Date fin** : _________

---

## ⏸️ Phase 7 : UI complète + filtres

### 7.1 Pages UI

- [ ] Modifier citations.py → shared
- [ ] Tester page Citations
- [ ] Modifier personal_performance.py → shared
- [ ] Tester page Performance
- [ ] Modifier media_library.py → shared.match_registry
- [ ] Tester page Médias

### 7.2 Filtres

- [ ] Modifier filters.py → Polars natif
- [ ] Tester render_session_filters()
- [ ] Modifier filters_render.py → type consistency
- [ ] Tester apply_filters()
- [ ] Modifier checkbox_filter.py (ne pas vider sélections)
- [ ] Tester filtres modes/maps/sessions

### 7.3 Scripts

- [ ] sync.py : supprimer rebuild_teammates_aggregate()
- [ ] sync.py : adapter print_stats()
- [ ] sync.py : adapter _resolve_player_in_db()
- [ ] Tester sync --delta

### 7.4 Validation Phase 7

- [ ] Toutes pages UI OK
- [ ] Filtres fonctionnent
- [ ] Sessions visibles

**Date début** : _________
**Date fin** : _________

---

## ⏸️ Phase 8 : Modules secondaires

### 8.1 Analyse

- [ ] killer_victim.py : TypedDict corrections
- [ ] Tester antagonistes
- [ ] citations/engine.py → shared + colonnes V5
- [ ] Tester citations

### 8.2 Media et visualisations

- [ ] media_indexer.py → shared.match_registry
- [ ] Tester indexation médias
- [ ] participation_radar.py → shared
- [ ] Tester radars

### 8.3 Utilitaires

- [ ] launcher.py → discovery shared
- [ ] Tester launcher
- [ ] multiplayer.py → list_players shared
- [ ] cache_loaders.py → shared
- [ ] cache_filters.py → shared
- [ ] aliases.py → shared.xuid_aliases
- [ ] data_loader.py → shared
- [ ] xuid.py → shared
- [ ] Tester utilitaires

### 8.4 Validation Phase 8

- [ ] Suite tests complète passe
- [ ] Toutes fonctionnalités OK

**Date début** : _________
**Date fin** : _________

---

## ⏸️ Phase 9 : Validation + cleanup brutal

### 9.1 Tests complétude

- [ ] Exécuter tests anti-régression (§16)
- [ ] test_no_from_match_stats_in_src()
- [ ] test_no_local_medals_earned_read()
- [ ] test_no_local_highlight_events_read()
- [ ] test_no_local_xuid_aliases_read()
- [ ] test_no_player_match_stats_in_src()
- [ ] test_sync_engine_does_not_write_match_stats()
- [ ] test_sync_engine_writes_player_match_enrichment()
- [ ] test_match_participants_has_extended_columns()
- [ ] test_match_participant_row_has_all_fields()
- [ ] test_extract_participants_extracts_all_stats()

### 9.2 Tests manuels

- [ ] Naviguer toutes pages app
- [ ] Tester filtres
- [ ] Tester graphiques
- [ ] Tester médias
- [ ] Tester exports

### 9.3 Sync de test

- [ ] python scripts/sync.py --player TestPlayer --delta --max-matches 50
- [ ] Vérifier données shared
- [ ] Vérifier player_match_enrichment

### 9.4 Cleanup brutal

- [ ] Backup avant cleanup
- [ ] python scripts/cleanup_player_dbs_v5.py --all --dry-run --verbose
- [ ] Vérifier rapport couverture
- [ ] python scripts/cleanup_player_dbs_v5.py --all --backup
- [ ] Vérifier tables supprimées (script Python)

### 9.5 Validation post-cleanup

- [ ] Relancer app
- [ ] Naviguer toutes pages
- [ ] Si erreurs "table introuvable" → identifier code résiduel
- [ ] Corriger code résiduel
- [ ] Re-cleanup si nécessaire

### 9.6 Rapport cleanup

- [ ] Tailles DBs avant/après
- [ ] Tables supprimées confirmées
- [ ] Tables conservées confirmées
- [ ] Aucune erreur app

**Date début** : _________
**Date fin** : _________

---

## ⏸️ Phase 10 : Documentation

### 10.1 Architecture (P0)

- [ ] ARCHITECTURE_V5.md : schéma état cible
- [ ] ARCHITECTURE_V5.md : 31 colonnes match_participants
- [ ] ARCHITECTURE_V5.md : stop dual write
- [ ] ARCHITECTURE_V5.md : 7 points critiques (§18.4)
- [ ] SHARED_MATCHES_SCHEMA.md : 16 colonnes étendues
- [ ] SHARED_MATCHES_SCHEMA.md : DDL complet
- [ ] SQL_SCHEMA.md : tables supprimées player DB
- [ ] DATA_ARCHITECTURE.md : flux sync → shared uniquement

### 10.2 Guides (P1-P2)

- [ ] SYNC_GUIDE.md : sync écrit shared + enrichment
- [ ] CLEANUP_V5.md : 8 tables supprimées
- [ ] CLEANUP_V5.md : --skip-coverage-check
- [ ] CLEANUP_V5_QUICKSTART.md : séquence backup → dry-run → cleanup
- [ ] COMMANDS.md : --participants-enrich
- [ ] BACKUP_RESTORE.md : backups plus petits

### 10.3 Docs IA (P0-P1)

- [ ] CLAUDE.md : tables player DB réduites
- [ ] CLAUDE.md : colonnes match_participants étendues
- [ ] CLAUDE.md : coéquipiers depuis shared
- [ ] .github/copilot-instructions.md : mêmes mises à jour
- [ ] .ai/project_map.md : flux sync, tables par DB
- [ ] .ai/data_lineage.md : API → shared uniquement
- [ ] .ai/thought_log.md : décision V5 finale

### 10.4 Vérification cohérence

- [ ] Toutes docs mentionnent 31 colonnes match_participants
- [ ] Aucune doc ne mentionne match_stats dans player DB
- [ ] Flux de données cohérents entre docs
- [ ] Exemples SQL à jour

### 10.5 Validation finale

- [ ] Relire ARCHITECTURE_V5.md complet
- [ ] Vérifier tous liens internes
- [ ] Pas de références obsolètes (V4, SQLite)
- [ ] Mise à jour CHANGELOG.md

**Date début** : _________
**Date fin** : _________

---

## 📊 Métriques de Progression

| Phase | Items totaux | Items complétés | % Progression |
|-------|--------------|-----------------|---------------|
| Phase 0 | 4 | 4 | 100% ✅ |
| Phase 1 | 5 | 5 | 100% ✅ |
| Phase 2 | 5 | 5 | 100% ✅ |
| Phase 3 | 5 | 5 | 100% ✅ |
| Phase 4 | 6 | 6 | 100% ✅ |
| Phase 5 | 40 | 0 | 0% |
| Phase 6 | 20 | 0 | 0% |
| Phase 7 | 15 | 0 | 0% |
| Phase 8 | 18 | 0 | 0% |
| Phase 9 | 20 | 0 | 0% |
| Phase 10 | 25 | 0 | 0% |
| **TOTAL** | **163** | **30** | **18%** |

---

## 🎯 Prochaines Actions

**Aujourd'hui** :
1. [ ] Lire MIGRATION_V5_FINAL_GUIDE.md Phase 5
2. [ ] Lire PHASES_5_10_ANALYSES.md Phase 5 détaillée
3. [ ] Commencer teammates_service.py

**Cette semaine** :
- [ ] Compléter Phase 5
- [ ] Commencer Phase 6

**Bloqueurs actuels** :
- Aucun

**Questions en suspens** :
- Aucune

---

## 📝 Notes de Session

### Session du _________ :

**Travail effectué** :


**Problèmes rencontrés** :


**Décisions prises** :


**À faire demain** :


---

## ✅ Critères de Succès Final

- [ ] Toutes les 11 phases complétées
- [ ] Suite de tests complète passe
- [ ] Aucune table interdite dans player DBs
- [ ] App fonctionne après cleanup brutal
- [ ] Documentation à jour et cohérente
- [ ] Tailles player DBs ~4MB (vs 30MB avant)
- [ ] Temps chargement page <800ms
- [ ] Aucun accès SQLite runtime
- [ ] Aucun Pandas dans code métier

---

**Bon courage pour la suite !** 🚀
