# Plan Consolidé Final - LevelUp v5.0

> **Date** : 2026-02-16
> **Sources** : 
> - `PLAN_UNIFIE.md` (12 sprints S0-S12, features + cleanup)
> - `MIGRATION_V5_FINAL_GUIDE.md` (11 phases 0-10, shared DB architecture)
> - `BUGFIX_V5_2026-02-15.md` (document technique détaillé)
>
> **Statut** : Plan unifié COMPLET et PRÉCIS évitant duplications et conflits
>
> **Objectif** : Consolider intelligemment les deux approches pour un plan d'exécution optimal

---

## 🎯 Vue d'ensemble de la consolidation

### Deux Plans, Un Objectif

#### Plan A : PLAN_UNIFIE.md
- **Focus** : Features UI + Cleanup code + Migration Pandas→Polars
- **Approche** : Incrémentale, feature-driven
- **Sprints** : 12 (S0-S12)
- **État** : S0-S4 ✅ livrés, S5-S8 en cours, S9-S12 à faire

#### Plan B : Migration V5 Finale  
- **Focus** : Architecture shared DB + Stop dual write + Cleanup tables
- **Approche** : Architecture-driven, migration fondamentale
- **Phases** : 11 (0-10)
- **État** : Phases 0-4 ✅ complétées, Phases 5-10 analysées en profondeur

### Relation Entre les Plans

```
PLAN_UNIFIE (Features)         MIGRATION V5 (Architecture)
S0-S4 ✅                        Phase 0-4 ✅
S5-S8 📋                        Phase 5-10 📋 (analyses complètes)
S9 (Cleanup) 📋                 
S10 (SQLite) 📋
S11 (Finalisation) 📋
S12 (Heatmap) ✅
```

**Constat** : Les deux plans sont **COMPLÉMENTAIRES** et non redondants.

---

## 🔍 Analyse Détaillée des Chevauchements

### 1. Migration Pandas→Polars

| Plan | Où | Quoi |
|------|-----|------|
| PLAN_UNIFIE Sprint 2 | ✅ Livré | `performance_score.py`, `backfill_data.py` |
| PLAN_UNIFIE Sprint 9 | 📋 À faire | ~24 fichiers restants (voir SPRINT_EXPLORATION.md) |
| Migration V5 Phase 7 | 📋 Analysé | `filters.py:370` `.empty`→`.is_empty()` bug critique |
| Migration V5 Phase 8 | 📋 Analysé | Audit 11 modules secondaires |

**Décision** : 
- ✅ Garder Sprint 9 pour migration Pandas globale
- ✅ Intégrer découvertes Phase 7 (3 bugs) dans Sprint 9
- ✅ Intégrer découvertes Phase 8 (3 modules critiques) dans Sprint 9

### 2. Suppression Code Legacy

| Plan | Où | Quoi |
|------|-----|------|
| PLAN_UNIFIE Sprint 1 | ✅ Livré | Archivage scripts (116→22) |
| PLAN_UNIFIE Sprint 9 | 📋 À faire | Suppression `src/db/loaders.py` + dépendants (33 fichiers) |
| Migration V5 Phase 8 | 📋 Analysé | 11 modules secondaires, 3 critiques |

**Décision** :
- ✅ Sprint 9 supprime `src/db/loaders.py`
- ✅ Intégrer analyses Phase 8 pour migration propre

### 3. Architecture Shared DB

| Plan | Où | Quoi |
|------|-----|------|
| PLAN_UNIFIE | ❌ Absent | Pas de migration shared DB explicite |
| Migration V5 Phases 5-10 | 📋 Analysé | Migration complète vers shared_matches.duckdb |

**DÉCOUVERTE MAJEURE** : 
- ❌ PLAN_UNIFIE ne couvre PAS la migration shared DB
- ✅ Migration V5 est **ESSENTIELLE** et **NON COUVERTE**

### 4. Tests et Validation

| Plan | Où | Quoi |
|------|-----|------|
| PLAN_UNIFIE Sprint 11 | 📋 À faire | Tests intégration, couverture >95% |
| Migration V5 Phase 9 | 📋 Analysé | Cleanup brutal tables + validation |

**Décision** :
- ✅ Sprint 11 reste pour tests globaux
- ✅ Phase 9 (cleanup brutal) s'exécute AVANT Sprint 11

### 5. Documentation

| Plan | Où | Quoi |
|------|-----|------|
| PLAN_UNIFIE Sprint 11 | 📋 À faire | `project_map.md`, `CLAUDE.md`, release notes |
| Migration V5 Phase 10 | 📋 Analysé | 13 docs (architecture, guides, IA) |

**Décision** :
- ✅ Fusionner les deux listes de docs
- ✅ Phase 10 + Sprint 11 = Documentation complète

---

## 🗓️ Plan Unifié Final (Chronologique)

### Légende
- ✅ : Complété
- 📋 : À faire
- 🔴 : Critique
- 🟡 : Important
- 🟢 : Optionnel

### Étape 1 : Compléter PLAN_UNIFIE Sprints 5-8 (Features)

**Durée** : 10-12 jours

| Sprint | Durée | Focus | Statut |
|--------|-------|-------|--------|
| S5 | 3j | Perf Stats + Sessions détaillées | 📋 |
| S6 | 3j | Médailles personnalisées | 📋 |
| S7 | 2j | Antagonistes + Top Modes | 📋 |
| S8 | 3j | Coéquipiers comparaisons avancées | 📋 |

**Objectif** : Terminer toutes les features UI avant migration architecture

**Raison** : Les features S5-S8 utilisent l'architecture V4 existante. Migrer l'architecture AVANT compliquerait leur implémentation.

---

### Étape 2 : Migration V5 Phases 5-8 (Architecture Shared DB)

**Durée** : 4-5 jours

**🔴 CRITIQUE** : Cette étape est ABSENTE de PLAN_UNIFIE et ESSENTIELLE

#### Phase 5 : Services et repositories (partie 1) - 1 jour

**Fichiers** :
- `src/data/services/teammates_service.py` (4 fonctions)
- `src/data/repositories/_match_queries.py` (3 fonctions)
- `src/data/repositories/_roster_loader.py` (15 fallbacks)

**Actions** :
- Lire shared.match_participants au lieu DBs individuelles
- Supprimer fallbacks local→shared
- Breaking change : gamertag → xuid

**Analyses** : `.ai/PHASES_5_10_ANALYSES.md` (Phase 5 complète, 35 pages)

#### Phase 6 : Repositories (partie 2) + UI critique - 1 jour

**Fichiers** :
- `src/data/repositories/duckdb_repo.py` (8 méthodes)
- `src/ui/pages/teammates_impact.py`
- `src/ui/pages/objective_analysis.py`

**Actions** :
- 5 méthodes : Supprimer fallback V4 (~150 lignes)
- 1 méthode : Garder fallbacks multi-sources
- 2 méthodes : Ajouter shared counts/fallback

**Analyses** : `.ai/PHASES_6_10_COMPLETE.md` (Phase 6, pages 1-30)

#### Phase 7 : UI complète + filtres - 1 jour

**🔴 3 BUGS CRITIQUES IDENTIFIÉS** :
1. `filters.py:370` → `.empty` (Pandas) au lieu `.is_empty()` (Polars)
2. `filters_render.py:303` → type hint incomplet (retourne 4 au lieu de 3)
3. `checkbox_filter.py` → "Aucun" vide sans confirmation

**Fichiers** :
- `src/app/filters.py`
- `src/app/filters_render.py`
- `src/ui/components/checkbox_filter.py`
- `src/ui/pages/citations.py`
- `src/ui/pages/media_library.py`

**Analyses** : `.ai/PHASES_6_10_COMPLETE.md` (Phase 7, pages 31-50)

#### Phase 8 : Modules secondaires - 1 jour

**🔴 3 MODULES CRITIQUES** nécessitent fallback shared :
1. `src/ui/aliases.py` → xuid_aliases metadata.duckdb
2. `src/utils/xuid.py` → resolve_xuid metadata.duckdb
3. `src/analysis/citations/engine.py` → personal_score_awards shared

**11 modules** analysés :
- ✅ 7 modules déjà sûrs
- 🔴 3 modules critiques ci-dessus
- 🟡 1 module à améliorer

**Analyses** : `.ai/PHASES_6_10_COMPLETE.md` (Phase 8, pages 51-70)

---

### Étape 3 : Sprint 9 Fusionné (Cleanup + Pandas Migration)

**Durée** : 5-6 jours

**Contenu consolidé** :

#### Sprint 9A : Suppression src/db/ (PLAN_UNIFIE)
- Supprimer `src/db/loaders.py`
- Migrer 33 dépendants vers DuckDBRepository
- Supprimer fallbacks legacy

#### Sprint 9B : Migration Pandas→Polars (PLAN_UNIFIE + Phase 7-8 insights)
- ~24 fichiers restants (voir SPRINT_EXPLORATION.md)
- **+ Intégrer fixes Phase 7** :
  - `filters.py:370` → `.is_empty()`
  - `filters_render.py:303` → type hint complet
  - `checkbox_filter.py` → confirmation dialog
- **+ Intégrer Phase 8 critiques** :
  - `aliases.py` → metadata fallback
  - `xuid.py` → metadata fallback
  - `citations/engine.py` → awards fallback

#### Sprint 9C : Validation intermédiaire
- Tests Pandas : `grep -r "import pandas" src/`
- Tests SQLite : `grep -r "import sqlite3" src/`
- Suite tests complète

**Analyses** : 
- PLAN_UNIFIE.md Sprint 9
- PHASES_6_10_COMPLETE.md Phases 7-8

---

### Étape 4 : Sprint 10 Fusionné (SQLite + Cleanup DB)

**Durée** : 2-3 jours

**Contenu consolidé** :

#### Sprint 10A : Suppression SQLite (PLAN_UNIFIE)
- 5 fichiers avec sqlite3 (voir SPRINT_EXPLORATION.md)
- Migration scripts uniquement (pas src/)

#### Sprint 10B : Pas de changement spécifique ici
- Poursuivre validation

**Analyses** : PLAN_UNIFIE.md Sprint 10

---

### Étape 5 : Phase 9 Migration V5 (Cleanup Brutal Tables)

**Durée** : 1 jour

**🔴 ÉTAPE CRITIQUE** : Suppression brutale tables locales

**Actions** :
1. Tests de complétude (§16 BUGFIX)
2. Tests manuels app complète
3. Sync de test 50 matchs
4. **Cleanup brutal** : `python scripts/cleanup_player_dbs_v5.py --all --backup`
5. Validation post-cleanup
6. Identifier code résiduel si erreurs

**8 tables supprimées** :
- match_stats
- medals_earned
- highlight_events
- match_participants
- player_match_stats
- xuid_aliases
- killer_victim_pairs
- teammates_aggregate

**9 tables conservées** :
- player_match_enrichment
- personal_score_awards
- antagonists
- match_citations
- career_progression
- media_files
- media_match_associations
- sync_meta
- sessions

**Script validation fourni** dans PHASES_6_10_COMPLETE.md

**Analyses** : `.ai/PHASES_6_10_COMPLETE.md` (Phase 9)

---

### Étape 6 : Sprint 11 + Phase 10 Fusionné (Finalisation + Documentation)

**Durée** : 3 jours

**Contenu consolidé** :

#### Tests et Validation (Sprint 11)
- Tests intégration
- Tests charge (1000+, 5000+ matchs)
- Couverture >95%
- Combler trous critiques

#### Documentation Complète (Sprint 11 + Phase 10)

**13+4 fichiers** à mettre à jour :

**Architecture (P0)** :
1. `docs/ARCHITECTURE_V5.md` - Schéma final, 31 colonnes, stop dual write
2. `docs/SHARED_MATCHES_SCHEMA.md` - DDL complet
3. `docs/SQL_SCHEMA.md` - Tables supprimées/conservées
4. `docs/DATA_ARCHITECTURE.md` - Flux sync→shared

**Guides (P1-P2)** :
5. `docs/SYNC_GUIDE.md` - Sync écrit shared + enrichment
6. `docs/CLEANUP_V5.md` - 8 tables supprimées
7. `docs/CLEANUP_V5_QUICKSTART.md` - Séquence complète
8. `docs/COMMANDS.md` - Nouvelles options
9. `docs/BACKUP_RESTORE.md` - Backups optimisés

**Docs IA (P0-P1)** :
10. `CLAUDE.md` - Tables réduites, colonnes étendues
11. `.github/copilot-instructions.md` - Idem
12. `.ai/project_map.md` - Flux sync, tables par DB
13. `.ai/data_lineage.md` - API → shared uniquement

**Docs Sprint 11** :
14. `.ai/RELEASE_NOTES_2026_Q1.md`
15. `.ai/thought_log.md` - Synthèse finale
16. Plans `.ai/features/` - Statut final
17. Lint CI (ruff rule) bloquer `import pandas`
18. Tag git `v5.0-final`

**Analyses** : 
- PLAN_UNIFIE.md Sprint 11
- PHASES_6_10_COMPLETE.md Phase 10

---

### Étape 7 : Sprint 12 (Heatmap Impact) - Optionnel

**Durée** : 2-3 jours

**Statut** : ✅ Déjà livré selon PLAN_UNIFIE

Si non livré :
- Heatmap d'impact coéquipiers
- Tableau taquinerie MVP/Boulet
- Intégration onglet Coéquipiers

**Analyses** : PLAN_UNIFIE.md Sprint 12

---

## 📊 Calendrier Consolidé Final

| Étape | Contenu | Durée | Jours Cumulés |
|-------|---------|-------|---------------|
| ✅ S0-S4 | Features UI initiales | - | 0 (déjà fait) |
| 📋 S5-S8 | Features UI avancées | 10-12j | 12 |
| 🔴 Phase 5-8 | Migration Shared DB | 4-5j | **17** |
| 🔴 S9 | Cleanup + Pandas Migration | 5-6j | **23** |
| 📋 S10 | SQLite Migration | 2-3j | 26 |
| 🔴 Phase 9 | Cleanup Brutal Tables | 1j | **27** |
| 📋 S11+Ph10 | Tests + Documentation | 3j | **30** |
| 🟢 S12 | Heatmap (optionnel) | 2-3j | 33 |

**Total estimé** : **30 jours** (6 semaines) pour migration complète

**Étapes critiques** (🔴) : **11 jours**

---

## 🎯 Dépendances Critiques

### Ordre Obligatoire

```
S5-S8 (Features)
    ↓
Phase 5-8 (Shared DB Architecture) 🔴 ESSENTIEL
    ↓
S9 (Cleanup + Pandas)
    ↓
S10 (SQLite)
    ↓
Phase 9 (Cleanup Brutal) 🔴 VALIDATION FINALE
    ↓
S11 + Phase 10 (Tests + Docs)
    ↓
S12 (Heatmap - optionnel)
```

**⚠️ NE PAS** :
- Faire Phase 5-8 AVANT S5-S8 (les features cassent)
- Faire Phase 9 AVANT S9 (code Pandas/SQLite reste)
- Faire S11 AVANT Phase 9 (tests échoueront post-cleanup)

---

## 🔍 Points de Vigilance

### 1. Migration Shared DB (Phases 5-8)

**Risques** :
- Breaking changes signatures (gamertag→xuid)
- Données manquantes si shared incomplet
- Tests cassés par changements architecture

**Mitigations** :
- Vérifier couverture 100% shared (fait Phase 1)
- Mettre à jour tous appelants immédiatement
- Tests unitaires en parallèle

### 2. Cleanup Brutal (Phase 9)

**Risques** :
- App plante si code résiduel lit tables locales
- Perte de données si backup incomplet

**Mitigations** :
- **BACKUP OBLIGATOIRE** avant cleanup
- Script validation fourni (check tables interdites)
- Dry-run d'abord

### 3. Coordination Pandas + Shared DB

**Risque** :
- Migrer Pandas sans shared → code cassé
- Migrer shared sans Pandas → imports legacy restent

**Mitigation** :
- Phase 5-8 AVANT S9 garantit shared en place
- S9 migre Pandas en connaissant nouvelle architecture

---

## 📋 Checklist Pré-Démarrage

Avant de commencer toute étape :

### Pour S5-S8 (Features)
- [ ] Lire PLAN_UNIFIE.md sprint concerné
- [ ] Consulter SPRINT_EXPLORATION.md si sprint ≥6
- [ ] `pytest tests/ -v` pour baseline

### Pour Phase 5-8 (Shared DB)
- [ ] Lire MIGRATION_V5_FINAL_GUIDE.md phase concernée
- [ ] Lire PHASES_5_10_ANALYSES.md (Phase 5) ou PHASES_6_10_COMPLETE.md (Phases 6-10)
- [ ] Vérifier phases 0-4 complétées
- [ ] Backup DBs

### Pour S9 (Cleanup + Pandas)
- [ ] S5-S8 complétés
- [ ] Phase 5-8 complétées
- [ ] Lire PLAN_UNIFIE.md Sprint 9
- [ ] Lire PHASES_6_10_COMPLETE.md Phases 7-8 (bugs identifiés)
- [ ] SPRINT_EXPLORATION.md audit Pandas (35 fichiers)

### Pour Phase 9 (Cleanup Brutal)
- [ ] S9 complété (Pandas migration)
- [ ] S10 complété (SQLite migration)
- [ ] **BACKUP COMPLET** de toutes les DBs
- [ ] Lire PHASES_6_10_COMPLETE.md Phase 9
- [ ] Script validation Python prêt

### Pour S11 + Phase 10 (Finalisation)
- [ ] Toutes étapes précédentes complétées
- [ ] Lire PLAN_UNIFIE.md Sprint 11
- [ ] Lire PHASES_6_10_COMPLETE.md Phase 10
- [ ] Liste 17 docs à mettre à jour

---

## 🎓 Ressources par Étape

| Étape | Documents Principaux | Documents Secondaires |
|-------|---------------------|----------------------|
| S5-S8 | PLAN_UNIFIE.md | SPRINT_EXPLORATION.md |
| Phase 5 | MIGRATION_V5_FINAL_GUIDE.md | PHASES_5_10_ANALYSES.md |
| Phase 6-8 | MIGRATION_V5_FINAL_GUIDE.md | PHASES_6_10_COMPLETE.md |
| S9 | PLAN_UNIFIE.md Sprint 9 | PHASES_6_10_COMPLETE.md (bugs Ph7-8) |
| S10 | PLAN_UNIFIE.md Sprint 10 | - |
| Phase 9 | PHASES_6_10_COMPLETE.md | BUGFIX_V5_2026-02-15.md |
| S11+Ph10 | PLAN_UNIFIE.md + PHASES_6_10_COMPLETE.md | - |

---

## ✅ Métriques de Succès Finales

### Techniques
- ✅ Zéro `import pandas` dans `src/` (hors frontière)
- ✅ Zéro `import sqlite3` dans `src/` (hors migration)
- ✅ Zéro tables interdites dans player DBs
- ✅ Couverture tests ≥ 95%
- ✅ Temps connexion < 20ms
- ✅ load_matches(100) < 80ms
- ✅ Taille player DB ~4MB (vs 30MB)

### Fonctionnelles
- ✅ App fonctionne après cleanup brutal
- ✅ Toutes features S0-S12 opérationnelles
- ✅ Shared DB contient 100% données matchs
- ✅ Documentation complète et cohérente

---

## 🚀 Prochaine Action Immédiate

**Si S5-S8 non complétés** :
→ Continuer PLAN_UNIFIE.md Sprint 5

**Si S5-S8 complétés** :
→ Démarrer Phase 5 Migration V5 avec PHASES_5_10_ANALYSES.md

**Commande pour vérifier état** :
```bash
# Voir quels sprints livrés
git log --oneline | grep -E "Sprint|Phase" | head -20

# Voir structure DBs
python scripts/check_db_structure.py
```

---

**Ce plan consolidé évite toute duplication et garantit un ordre d'exécution optimal.** 🎯
