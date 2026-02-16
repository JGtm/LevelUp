# Processus Qualité V5.1 — Framework Validation et Assurance Qualité

> **Date de création** : 2026-02-16  
> **Version** : v5.1  
> **Complément à** : RECONCILIATION_FINALE_V5.1.md  
> **Objectif** : Définir processus qualité rigoureux pour garantir succès du projet

---

## 🎯 Objectif

Ce document définit **TOUS** les processus qualité, validations, checklists et procédures pour chaque étape du projet V5.1, ainsi que les procédures de fin de projet.

---

## 📋 Table des Matières

1. [Framework Général](#framework-général)
2. [Processus Avant Chaque Sprint](#processus-avant-chaque-sprint)
3. [Definition of Done par Étape](#definition-of-done-par-étape)
4. [Processus Revue de Code](#processus-revue-de-code)
5. [Framework Tests et Validation](#framework-tests-et-validation)
6. [Processus Après Chaque Sprint](#processus-après-chaque-sprint)
7. [Procédures Fin de Projet](#procédures-fin-de-projet)

---

## 📐 Framework Général

### Principes Qualité V5.1

1. **Pas de livraison sans tests** : Chaque modification doit avoir des tests
2. **Revue de code systématique** : Aucun commit direct sur main
3. **Documentation synchronisée** : Docs mises à jour AVANT merge
4. **Métriques mesurables** : Chaque étape a des critères de succès chiffrés
5. **Baseline préservée** : Toujours possibilité de rollback

### Rôles et Responsabilités

| Rôle | Responsabilité |
|------|----------------|
| **Développeur** | Implémentation + tests unitaires + doc technique |
| **Reviewer** | Revue code + validation tests + cohérence architecture |
| **QA** | Tests intégration + validation métriques + non-régression |
| **Tech Lead** | Validation architecture + décisions techniques + merge final |

**Note** : Pour projet solo, même personne endosse tous les rôles avec checklist rigoureuse.

---

## 🚀 Processus AVANT Chaque Sprint

### Checklist Pré-Sprint (OBLIGATOIRE)

#### ✅ Phase Préparation

- [ ] **Objectifs clairs** : Lire section du plan concernée (RECONCILIATION_FINALE_V5.1.md)
- [ ] **Pré-requis validés** : Étapes précédentes 100% complètes
- [ ] **Baseline sauvegardée** : Backup complet état actuel
- [ ] **Branch créée** : `git checkout -b sprint-X-nom-descriptif`
- [ ] **Environnement testé** : `python scripts/check_env.py` → ✅
- [ ] **Tests verts** : `python -m pytest` → ✅ (≥75% couverture)

#### ✅ Phase Analyse

- [ ] **Fichiers identifiés** : Liste exhaustive fichiers à modifier
- [ ] **Dépendances comprises** : Graphe dépendances fichiers/modules
- [ ] **Risques listés** : Identification points de rupture potentiels
- [ ] **Temps estimé** : Estimation réaliste durée (buffer +30%)

#### ✅ Phase Planification

- [ ] **Checklist tâches** : Décomposition en sous-tâches (<2h chacune)
- [ ] **Ordre défini** : Séquencement logique des tâches
- [ ] **Points de contrôle** : Jalons intermédiaires avec validation
- [ ] **Critères succès** : Definition of Done rédigée (voir section dédiée)

#### ✅ Validation GO/NO-GO

**Critères GO** : TOUTES les cases ci-dessus cochées ✅  
**Si NO-GO** : Compléter préparation avant de commencer

**Commande checkpoint** :
```bash
# Sauvegarder état pré-sprint
git tag "pre-sprint-X-$(date +%Y%m%d)"
python scripts/backup_state.py --label "pre-sprint-X"
```

---

## 📝 Definition of Done par Étape

### Étape 0 : Préparation

#### Critères de Succès

- [ ] **Backups créés** : ≥2 joueurs de test avec data complète
- [ ] **Baseline établie** :
  - [ ] Temps connexion mesuré (baseline_connexion.txt)
  - [ ] load_matches(100) mesuré (baseline_load.txt)
  - [ ] Première page UI mesurée (baseline_ui.txt)
  - [ ] Taille DBs mesurée (baseline_size.txt)
- [ ] **Tests verts** : `pytest` → 100% pass, couverture ≥75%
- [ ] **Environnement validé** : `check_env.py` → ✅
- [ ] **Documentation à jour** : README.md reflète état actuel

#### Tests Obligatoires

```bash
python -m pytest tests/ -v
python scripts/check_env.py
python scripts/measure_baseline.py --output baseline_v5.0.json
```

#### Revue de Code

- [ ] README.md vérifié
- [ ] Scripts baseline fonctionnels
- [ ] Backups récupérables (test restore)

---

### Étape 1 : Performance UI (3 sprints)

#### Sprint Perf 1 : Vue Matérialisée

**Definition of Done** :

- [ ] **Implémentation** :
  - [ ] Vue `mv_player_matches` créée dans DuckDB
  - [ ] `_get_match_source()` simplifié (170→10 lignes)
  - [ ] Migration toutes requêtes vers vue
- [ ] **Tests** :
  - [ ] Tests unitaires vue matérialisée
  - [ ] Tests équivalence résultats ancienne/nouvelle requête
  - [ ] Tests performance (parsing SQL < 10ms)
  - [ ] Couverture tests ≥80%
- [ ] **Validation** :
  - [ ] Benchmark avant/après (gain ≥60% parsing)
  - [ ] Aucune régression fonctionnelle
  - [ ] UI fonctionne identique
- [ ] **Documentation** :
  - [ ] Docstring vue matérialisée
  - [ ] README migration mise à jour
  - [ ] CHANGELOG.md entrée ajoutée

**Tests Obligatoires** :
```bash
python -m pytest tests/test_mv_player_matches.py -v
python scripts/benchmark_sql_parsing.py --before --after
python -m pytest tests/integration/ -v
```

**Revue de Code Checklist** :
- [ ] Vue DuckDB syntaxe correcte
- [ ] Index appropriés définis
- [ ] Pas de régression performance
- [ ] Code simplifié (moins de COALESCE)
- [ ] Tests couvrent edge cases

---

#### Sprint Perf 2 : Cache Repository

**Definition of Done** :

- [ ] **Implémentation** :
  - [ ] `@st.cache_resource` sur connexion DB
  - [ ] Repository persistant entre pages
  - [ ] Gestion erreurs connexion
- [ ] **Tests** :
  - [ ] Tests cache hit/miss
  - [ ] Tests invalidation cache
  - [ ] Tests reconnexion après erreur
  - [ ] Couverture ≥80%
- [ ] **Validation** :
  - [ ] Temps connexion < 20ms (objectif atteint)
  - [ ] Cache fonctionne entre pages
  - [ ] Pas de memory leak
- [ ] **Documentation** :
  - [ ] Architecture cache documentée
  - [ ] Guide invalidation cache
  - [ ] CHANGELOG.md mise à jour

**Tests Obligatoires** :
```bash
python -m pytest tests/test_cache_repository.py -v
python scripts/measure_connexion_time.py --iterations 100
python scripts/test_cache_behavior.py
```

**Revue de Code Checklist** :
- [ ] `@st.cache_resource` utilisé correctement
- [ ] Gestion erreurs robuste
- [ ] Pas de global state dangereux
- [ ] Tests couvrent invalidation

---

#### Sprint Perf 3 : Index + Schema Cache

**Definition of Done** :

- [ ] **Implémentation** :
  - [ ] Index DuckDB `(xuid, match_id)` créés
  - [ ] Cache vérifications schéma
  - [ ] Optimisations query planner
- [ ] **Tests** :
  - [ ] Tests performance jointures
  - [ ] Tests schema cache
  - [ ] Tests avec/sans index
  - [ ] Couverture ≥80%
- [ ] **Validation** :
  - [ ] Jointures -30% temps (mesuré)
  - [ ] load_matches(100) < 80ms (objectif atteint)
  - [ ] Première page < 800ms (objectif atteint)
- [ ] **Documentation** :
  - [ ] Index documentés
  - [ ] Stratégie cache schéma expliquée
  - [ ] CHANGELOG.md mise à jour

**Tests Obligatoires** :
```bash
python -m pytest tests/test_duckdb_indexes.py -v
python scripts/benchmark_jointures.py --before --after
python scripts/measure_page_load.py --page home
```

**Revue de Code Checklist** :
- [ ] Index appropriés (pas de sur-indexation)
- [ ] Cache schéma thread-safe
- [ ] Validation schéma efficace
- [ ] Métriques performance atteintes

---

### Étape 2 : Performance Données

**Definition of Done** :

- [ ] **Implémentation** :
  - [ ] Index DuckDB supplémentaires
  - [ ] Optimisations query planner
  - [ ] Statistiques tables à jour
- [ ] **Tests** :
  - [ ] Tests performance requêtes
  - [ ] Tests avec données volumineuses
  - [ ] Couverture ≥80%
- [ ] **Validation** :
  - [ ] Query planner utilise index
  - [ ] Scans séquentiels minimisés
  - [ ] Pas de régression mémoire
- [ ] **Documentation** :
  - [ ] Stratégie indexation documentée
  - [ ] CHANGELOG.md mise à jour

**Tests Obligatoires** :
```bash
python -m pytest tests/test_query_performance.py -v
python scripts/analyze_query_plans.py
```

**Revue de Code Checklist** :
- [ ] Index couvrent queries fréquentes
- [ ] Statistiques DuckDB à jour
- [ ] Query plans optimaux

---

### Étape 3 : Architecture Shared DB (Phases 5-6)

#### Phase 5 : Services & Queries

**Definition of Done** :

- [ ] **Implémentation** :
  - [ ] `teammates_service.py` : 4 fonctions migrées vers shared
  - [ ] `_match_queries.py` : `_get_match_source()` simplifié (100→3 lignes)
  - [ ] `_roster_loader.py` : 15 fallbacks supprimés
  - [ ] Lecture depuis `shared.match_participants`
- [ ] **Tests** :
  - [ ] Tests services avec shared DB
  - [ ] Tests équivalence résultats
  - [ ] Tests intégration multi-joueurs
  - [ ] Couverture ≥80%
- [ ] **Validation** :
  - [ ] Aucun accès DBs individuelles
  - [ ] Performance équivalente ou meilleure
  - [ ] Tests multi-joueurs passent
- [ ] **Documentation** :
  - [ ] Architecture shared DB documentée
  - [ ] Migration guide créé
  - [ ] CHANGELOG.md mise à jour

**Tests Obligatoires** :
```bash
python -m pytest tests/test_teammates_service_shared.py -v
python -m pytest tests/test_match_queries_shared.py -v
python -m pytest tests/test_roster_loader_shared.py -v
python scripts/validate_no_individual_db_access.py
```

**Revue de Code Checklist** :
- [ ] Aucun fallback DB individuelle
- [ ] Signature fonctions cohérente (xuid)
- [ ] Gestion erreurs robuste
- [ ] Tests couvrent edge cases

---

#### Phase 6 : duckdb_repo

**Definition of Done** :

- [ ] **Implémentation** :
  - [ ] 8 méthodes migrées vers shared
  - [ ] Fallbacks V4 supprimés
  - [ ] `get_storage_info()` avec shared counts
- [ ] **Tests** :
  - [ ] Tests 8 méthodes
  - [ ] Tests absence fallbacks
  - [ ] Tests storage info
  - [ ] Couverture ≥80%
- [ ] **Validation** :
  - [ ] Aucun fallback V4 runtime
  - [ ] Performance équivalente
  - [ ] UI fonctionne correctement
- [ ] **Documentation** :
  - [ ] API repository documentée
  - [ ] CHANGELOG.md mise à jour

**Tests Obligatoires** :
```bash
python -m pytest tests/test_duckdb_repo_shared.py -v
python scripts/audit_v4_fallbacks.py → zéro résultat
```

**Revue de Code Checklist** :
- [ ] Toutes méthodes utilisent shared
- [ ] Pas de code mort (anciens fallbacks)
- [ ] Documentation API claire

---

### Étape 4 : Éradication SQLite Runtime

**Definition of Done** :

- [ ] **Implémentation** :
  - [ ] 7 fichiers runtime nettoyés :
    - [ ] `src/data/query/engine.py`
    - [ ] `src/data/infrastructure/database/duckdb_engine.py`
    - [ ] `src/utils/paths.py`
    - [ ] `src/ui/sync.py`
    - [ ] `src/ui/multiplayer.py`
    - [ ] `src/ai/rag.py`
    - [ ] `scripts/refetch_film_roster.py`
  - [ ] Fallbacks SQLite supprimés
  - [ ] Références `metadata.db` supprimées
- [ ] **Tests** :
  - [ ] Tests fonctionnels sans SQLite
  - [ ] Tests de non-régression
  - [ ] Couverture ≥80%
- [ ] **Validation** :
  - [ ] `grep -r "import sqlite3" src/ | grep -v "^#"` → zéro résultat
  - [ ] Aucune régression UI/fonctionnelle
  - [ ] Pas de fichier `.db` créé runtime
- [ ] **Documentation** :
  - [ ] Guide migration SQLite→DuckDB
  - [ ] CHANGELOG.md mise à jour

**Tests Obligatoires** :
```bash
python -m pytest tests/test_no_sqlite.py -v
python scripts/audit_sqlite_imports.py → zéro résultat
python scripts/test_no_db_files_created.py
```

**Revue de Code Checklist** :
- [ ] Aucun `import sqlite3` dans src/
- [ ] Aucune référence metadata.db
- [ ] Tests passent sans SQLite installé

---

### Étape 5 : Scripts Migration Bannières

**Definition of Done** :

- [ ] **Implémentation** :
  - [ ] 5 scripts migration + bannière LEGACY :
    - [ ] `scripts/migrate_player_to_duckdb.py`
    - [ ] `scripts/recover_from_sqlite.py`
    - [ ] `scripts/convert_sqlite_to_duckdb.py`
    - [ ] `scripts/migrate_legacy_db.py`
    - [ ] `scripts/rebuild_from_sqlite.py`
  - [ ] README scripts migration créé
- [ ] **Tests** :
  - [ ] Scripts exécutent avec bannière
  - [ ] Bannière claire et visible
- [ ] **Validation** :
  - [ ] Chaque script affiche bannière LEGACY
  - [ ] README explique contexte
- [ ] **Documentation** :
  - [ ] `scripts/migration/README.md` créé
  - [ ] CHANGELOG.md mise à jour

**Bannière Standard** :
```python
print("=" * 80)
print("⚠️  SCRIPT LEGACY - MIGRATION v3→v5")
print("⚠️  Ce script est conservé pour référence historique uniquement")
print("⚠️  NE PAS UTILISER en production v5.1")
print("=" * 80)
```

**Validation** :
```bash
python scripts/migrate_player_to_duckdb.py --help | grep "LEGACY"
```

**Revue de Code Checklist** :
- [ ] Bannière visible et claire
- [ ] README explique pourquoi conservé
- [ ] Scripts ne sont pas appelés par code actif

---

### Étape 6 : Migration Pandas→Polars

**Definition of Done** :

- [ ] **Implémentation** :
  - [ ] 7 fichiers métier migrés :
    - [ ] `src/analysis/performance_score.py`
    - [ ] `src/data/services/win_loss_service.py`
    - [ ] `src/ui/pages/objective_analysis.py`
    - [ ] `src/ui/pages/match_view_helpers.py`
    - [ ] `src/ui/pages/win_loss.py`
    - [ ] `src/ui/cache_filters.py`
    - [ ] `src/ui/components/duckdb_analytics.py`
  - [ ] Patterns migration appliqués :
    - [ ] `.groupby()` → `.group_by()`
    - [ ] `.fillna()` → `.fill_null()`
    - [ ] `.isna()` → `.is_null()`
    - [ ] `.rename()` → `.rename()`
    - [ ] `.merge()` → `.join()`
- [ ] **Tests** :
  - [ ] Tests migration (outputs identiques)
  - [ ] Tests performance Polars vs Pandas
  - [ ] Couverture ≥80%
- [ ] **Validation** :
  - [ ] `grep -r "import pandas" src/ | grep -v "_compat|_bridge|streamlit_bridge" | grep -v "^#"` → zéro résultat
  - [ ] Performance équivalente ou meilleure
  - [ ] UI fonctionne identique
- [ ] **Documentation** :
  - [ ] Guide migration Pandas→Polars
  - [ ] Patterns migration documentés
  - [ ] CHANGELOG.md mise à jour

**Tests Obligatoires** :
```bash
python -m pytest tests/test_pandas_to_polars_migration.py -v
python scripts/audit_pandas_imports.py → zéro résultat métier
python scripts/benchmark_polars_vs_pandas.py
```

**Revue de Code Checklist** :
- [ ] Aucun `import pandas` dans métier
- [ ] Patterns Polars corrects
- [ ] Tests vérifient équivalence résultats
- [ ] Performance au moins équivalente

---

### Étape 7 : Bugs Critiques (Ph7-8)

**Definition of Done** :

- [ ] **Bugs Polars corrigés** :
  - [ ] `filters.py:370` : `.empty` → `.is_empty()`
  - [ ] `filters_render.py:303` : Type hint complet (4 valeurs)
  - [ ] `checkbox_filter.py` : Confirmation "Aucun" button
- [ ] **Fallbacks metadata ajoutés** :
  - [ ] `src/ui/aliases.py` : fallback xuid_aliases
  - [ ] `src/utils/xuid.py` : fallback resolve_xuid
  - [ ] `src/analysis/citations/engine.py` : fallback personal_score_awards
- [ ] **Tests** :
  - [ ] Tests bugs corrigés
  - [ ] Tests fallbacks metadata
  - [ ] Couverture ≥80%
- [ ] **Validation** :
  - [ ] Aucun `.empty` (Pandas) dans code Polars
  - [ ] Type hints corrects
  - [ ] Fallbacks metadata fonctionnels
- [ ] **Documentation** :
  - [ ] Bugs documentés (pourquoi/comment)
  - [ ] CHANGELOG.md mise à jour

**Tests Obligatoires** :
```bash
python -m pytest tests/test_polars_bugs_fixed.py -v
python -m pytest tests/test_metadata_fallbacks.py -v
python scripts/audit_polars_empty.py → zéro résultat
```

**Revue de Code Checklist** :
- [ ] `.is_empty()` utilisé (Polars)
- [ ] Type hints complets
- [ ] Fallbacks testés
- [ ] UX confirmation appropriée

---

### Étape 8 : Cleanup Tables Legacy

**Definition of Done** :

- [ ] **Implémentation** :
  - [ ] 8 tables supprimées par joueur :
    - [ ] `match_stats`
    - [ ] `medals_earned`
    - [ ] `highlight_events`
    - [ ] `player_stats`
    - [ ] `mv_match_stats_with_context`
    - [ ] `mv_recent_matches`
    - [ ] `mv_team_stats`
    - [ ] `mv_opponent_stats`
  - [ ] Script cleanup automatique
  - [ ] Backup obligatoire avant cleanup
- [ ] **Tests** :
  - [ ] Tests dry-run cleanup
  - [ ] Tests intégration post-cleanup
  - [ ] Tests UI post-cleanup
  - [ ] Couverture ≥80%
- [ ] **Validation** :
  - [ ] Tables interdites absentes
  - [ ] Aucune erreur UI après cleanup
  - [ ] Taille DBs réduite (~4MB objectif)
  - [ ] Fonctionnalités intactes
- [ ] **Documentation** :
  - [ ] Script cleanup documenté
  - [ ] Procédure rollback documentée
  - [ ] CHANGELOG.md mise à jour

**Tests Obligatoires** :
```bash
python scripts/cleanup_legacy_tables.py --dry-run --player TestPlayer
python -m pytest tests/test_cleanup_tables.py -v
python scripts/validate_no_legacy_tables.py → zéro table legacy
python scripts/measure_db_size.py → ~4MB par joueur
```

**Revue de Code Checklist** :
- [ ] Backup obligatoire avant cleanup
- [ ] Dry-run fonctionne
- [ ] Rollback testé
- [ ] Script idempotent

---

### Étape 9 : Tests + Documentation

**Definition of Done** :

- [ ] **Tests** :
  - [ ] Suite tests intégration complète
  - [ ] Couverture ≥80% atteinte
  - [ ] Tous tests verts
  - [ ] Tests performance validés
- [ ] **Documentation** :
  - [ ] 13 docs architecture mis à jour :
    - [ ] `docs/ARCHITECTURE_V5.md`
    - [ ] `docs/DATABASE_SCHEMA.md`
    - [ ] `docs/DATA_FLOW.md`
    - [ ] `docs/API.md`
    - [ ] `docs/PERFORMANCE.md`
    - [ ] `.ai/project_map.md`
    - [ ] `.ai/data_lineage.md`
    - [ ] `.ai/thought_log.md`
    - [ ] `CLAUDE.md`
    - [ ] `COPILOT_INSTRUCTIONS.md`
    - [ ] `README.md`
    - [ ] `CHANGELOG.md`
    - [ ] `docs/USER_GUIDE.md`
  - [ ] Docs cohérentes entre elles
  - [ ] Exemples à jour
- [ ] **Validation** :
  - [ ] Grep cohérence (pas de v4 dans docs)
  - [ ] Liens docs fonctionnels
  - [ ] Screenshots UI à jour
- [ ] **Documentation** :
  - [ ] Meta : ce sont les docs finales

**Tests Obligatoires** :
```bash
python -m pytest tests/ -v --cov=src --cov-report=term-missing
python scripts/validate_documentation.py
python scripts/check_doc_consistency.py
```

**Revue de Code Checklist** :
- [ ] Couverture ≥80%
- [ ] Docs cohérentes
- [ ] Pas de références legacy
- [ ] Screenshots à jour

---

### Étape 10 : Release v5.1

**Definition of Done** :

- [ ] **Release** :
  - [ ] Tag Git `v5.1.0`
  - [ ] Release notes complètes
  - [ ] Métriques avant/après documentées
  - [ ] CHANGELOG.md finalisé
- [ ] **Validation** :
  - [ ] Toutes métriques v5.1 atteintes
  - [ ] Zéro SQLite runtime
  - [ ] Zéro Pandas métier
  - [ ] Couverture ≥80%
- [ ] **Communication** :
  - [ ] Release notes publiées
  - [ ] Annonce faite
- [ ] **Archivage** :
  - [ ] Voir section "Procédures Fin de Projet"

**Commandes Release** :
```bash
git tag -a v5.1.0 -m "Release v5.1.0 - Pure Architecture DuckDB + Polars"
git push origin v5.1.0
python scripts/generate_release_notes.py --version v5.1.0
```

**Revue Finale** :
- [ ] Toutes métriques atteintes
- [ ] Documentation complète
- [ ] Tests verts
- [ ] Release notes claires

---

## 🔍 Processus Revue de Code

### Workflow Revue

1. **Développeur** :
   - Crée branch feature
   - Implémente + tests + docs
   - Auto-revue checklist
   - Crée PR avec template

2. **Reviewer** :
   - Lit description PR
   - Vérifie checklist complète
   - Revue code (voir checklist)
   - Tests localement si nécessaire
   - Demande changements ou approuve

3. **Développeur** :
   - Adresse commentaires
   - Update PR

4. **Reviewer** :
   - Re-revue
   - Approuve

5. **Tech Lead** :
   - Validation finale
   - Merge

### Checklist Revue de Code Universelle

#### ✅ Architecture & Design

- [ ] Cohérent avec architecture v5.1 (DuckDB + Polars)
- [ ] Pas de régression architecture
- [ ] Abstraction appropriée
- [ ] Pas de code dupliqué

#### ✅ Code Quality

- [ ] Lisible et maintenable
- [ ] Nommage clair et cohérent
- [ ] Complexité raisonnable (pas de fonctions >50 lignes)
- [ ] Commentaires utiles (pourquoi, pas quoi)
- [ ] Type hints complets (Python)

#### ✅ Sécurité

- [ ] Pas d'injection SQL
- [ ] Validation inputs
- [ ] Pas de secrets hardcodés
- [ ] Gestion erreurs robuste

#### ✅ Performance

- [ ] Pas de régression performance
- [ ] Algorithmes efficaces
- [ ] Pas de boucles inutiles
- [ ] Cache utilisé appropriément

#### ✅ Tests

- [ ] Tests unitaires présents
- [ ] Tests couvrent edge cases
- [ ] Tests passent localement
- [ ] Couverture satisfaisante (≥80%)

#### ✅ Documentation

- [ ] Docstrings complets
- [ ] README mis à jour si nécessaire
- [ ] CHANGELOG.md mis à jour
- [ ] Commentaires code appropriés

#### ✅ Commits & PR

- [ ] Commits atomiques et logiques
- [ ] Messages commits clairs (Conventional Commits)
- [ ] PR description complète
- [ ] Pas de fichiers inutiles (build, tmp)

### Template Pull Request

```markdown
## Description

[Description claire du changement]

## Motivation

[Pourquoi ce changement est nécessaire]

## Type de changement

- [ ] Bug fix
- [ ] Nouvelle fonctionnalité
- [ ] Breaking change
- [ ] Documentation
- [ ] Performance
- [ ] Refactoring

## Checklist

- [ ] Tests ajoutés/mis à jour
- [ ] Documentation mise à jour
- [ ] CHANGELOG.md mis à jour
- [ ] Auto-revue effectuée
- [ ] Tests passent localement
- [ ] Pas de régression

## Métriques (si applicable)

- Avant : [valeur]
- Après : [valeur]
- Gain : [%]

## Screenshots (si UI)

[Screenshots avant/après]

## Liens

- Issue : #XXX
- Documentation : [lien]
```

---

## 🧪 Framework Tests et Validation

### Pyramide Tests

```
        /\
       /  \      E2E Tests (5%)
      /    \     - Tests bout en bout
     /------\    - UI complète
    /        \   
   /  Intég.  \  Tests Intégration (15%)
  /            \ - Multi-modules
 /--------------\- DB + Services
/   Unitaires   \Tests Unitaires (80%)
------------------Base de la pyramide
```

### Types de Tests

#### 1. Tests Unitaires (80%)

**Objectif** : Tester chaque fonction/méthode isolément

**Exemple** :
```python
def test_calculate_kd_ratio():
    assert calculate_kd_ratio(10, 5) == 2.0
    assert calculate_kd_ratio(10, 0) == 10.0
    assert calculate_kd_ratio(0, 10) == 0.0
```

**Checklist** :
- [ ] Teste cas nominal
- [ ] Teste edge cases
- [ ] Teste erreurs
- [ ] Mock dépendances externes
- [ ] Rapide (<100ms par test)

#### 2. Tests Intégration (15%)

**Objectif** : Tester interaction entre modules

**Exemple** :
```python
def test_load_matches_from_shared_db():
    repo = DuckDBRepository(shared_db_path, xuid)
    matches = repo.load_matches(limit=10)
    assert len(matches) == 10
    assert all(m.xuid == xuid for m in matches)
```

**Checklist** :
- [ ] Teste flow complet
- [ ] DB réelle (ou test DB)
- [ ] Données test cohérentes
- [ ] Cleanup après test

#### 3. Tests E2E (5%)

**Objectif** : Tester application complète

**Exemple** :
```python
def test_ui_displays_player_stats():
    # Simule navigation Streamlit
    # Vérifie affichage stats
    pass
```

**Checklist** :
- [ ] Teste parcours utilisateur
- [ ] Environnement test isolé
- [ ] Données test réalistes

### Outils Tests

| Outil | Usage |
|-------|-------|
| **pytest** | Framework tests principal |
| **pytest-cov** | Couverture tests |
| **pytest-mock** | Mocking |
| **pytest-benchmark** | Performance tests |
| **hypothesis** | Property-based testing |

### Commandes Tests Standard

```bash
# Tous les tests
python -m pytest

# Avec couverture
python -m pytest --cov=src --cov-report=html

# Tests spécifiques
python -m pytest tests/test_specific.py -v

# Tests rapides (skip slow)
python -m pytest -m "not slow"

# Tests parallèles
python -m pytest -n auto
```

### Critères Validation Tests

| Critère | Objectif | Commande Validation |
|---------|----------|---------------------|
| **Couverture** | ≥80% | `pytest --cov --cov-fail-under=80` |
| **Tous verts** | 100% pass | `pytest` → exit 0 |
| **Pas de skip** | 0 tests skip | `pytest --strict-markers` |
| **Performance** | <5min suite | `time pytest` |

---

## ✅ Processus APRÈS Chaque Sprint

### Checklist Post-Sprint (OBLIGATOIRE)

#### ✅ Validation Technique

- [ ] **Tous tests verts** : `pytest` → 100% pass
- [ ] **Couverture ≥80%** : `pytest --cov --cov-fail-under=80`
- [ ] **Pas de régression** : Tests non-régression passent
- [ ] **Métriques atteintes** : Validation critères succès sprint

#### ✅ Revue Code

- [ ] **Auto-revue** : Checklist revue code complète
- [ ] **Peer review** : Si équipe, reviewer approuve
- [ ] **Pas de TODOs critiques** : Tous TODOs traités ou ticketés

#### ✅ Documentation

- [ ] **CHANGELOG.md** : Entrée sprint ajoutée
- [ ] **Docs techniques** : Mises à jour synchronisées
- [ ] **README** : Reflète nouvel état si nécessaire

#### ✅ Qualité Code

- [ ] **Linting** : `ruff check src/` → zéro erreur
- [ ] **Formatting** : `black --check src/` → zéro changement
- [ ] **Type checking** : `mypy src/` → zéro erreur
- [ ] **Pas de warnings** : Logs propres

#### ✅ Commit & Tag

- [ ] **Commit atomique** : Message Conventional Commits
- [ ] **Tag sprint** : `git tag sprint-X-done-$(date +%Y%m%d)`
- [ ] **Push** : `git push && git push --tags`

#### ✅ Validation GO/NO-GO Étape Suivante

**Critères GO** : TOUTES les cases ci-dessus cochées ✅  
**Si NO-GO** : Corriger avant de passer à l'étape suivante

**Commande checkpoint** :
```bash
# Valider état post-sprint
git tag "post-sprint-X-$(date +%Y%m%d)"
python scripts/validate_sprint_completion.py --sprint X
```

### Retrospective Sprint

**À faire après chaque sprint** :

1. **Ce qui a bien fonctionné** :
   - [Liste points positifs]

2. **Ce qui peut être amélioré** :
   - [Liste points amélioration]

3. **Actions pour prochain sprint** :
   - [Actions concrètes]

4. **Métriques réelles vs estimées** :
   - Durée estimée : Xh
   - Durée réelle : Yh
   - Écart : Z%

**Documenter dans** : `.ai/retrospectives/sprint-X.md`

---

## 🏁 Procédures Fin de Projet

### Phase 1 : Validation Finale (1h)

#### ✅ Validation Complétude

- [ ] **Toutes étapes complètes** : Checklist réconciliation 100%
- [ ] **Tous tests verts** : Suite complète passe
- [ ] **Couverture ≥80%** : Objectif atteint
- [ ] **Métriques v5.1** : Toutes atteintes (voir tableau)

**Commandes Validation** :
```bash
python scripts/validate_all_metrics.py → Tous ✅
python -m pytest --cov --cov-fail-under=80 → Pass
python scripts/audit_sqlite_imports.py → Zéro
python scripts/audit_pandas_imports.py → Zéro métier
python scripts/validate_no_legacy_tables.py → Zéro
```

#### ✅ Validation Métriques

| Métrique | Objectif | Validation | Statut |
|----------|----------|------------|--------|
| Connexion DB | <20ms | `measure_connexion.py` | [ ] |
| load_matches(100) | <80ms | `measure_load.py` | [ ] |
| Première page | <800ms | `measure_ui.py` | [ ] |
| SQLite runtime | 0 | `audit_sqlite.py` | [ ] |
| Pandas métier | 0 | `audit_pandas.py` | [ ] |
| Tables obsolètes | 0 | `validate_tables.py` | [ ] |
| Taille DB | ~4MB | `measure_size.py` | [ ] |
| Couverture tests | ≥80% | `pytest --cov` | [ ] |

---

### Phase 2 : Archivage Documentation (2h)

#### ✅ Archivage Plans et Analyses

**Créer** : `.ai/archive/v5.1-completion-$(date +%Y%m%d)/`

**Fichiers à archiver** :
- [ ] **Plans sources** :
  - [ ] `RECONCILIATION_FINALE_V5.1.md` → archive
  - [ ] `CONSOLIDATION_FINALE_V5.1.md` → archive
  - [ ] `PROJECT_UNIFIE_V5.1.md` → archive
  - [ ] `PLAN_ERADICATION_LEGACY_V5.md` → archive
  - [ ] `diagnostics/PLAN_OPTIMISATION_V5.md` → archive
  - [ ] `MIGRATION_V5_FINAL_GUIDE.md` → archive

- [ ] **Analyses** :
  - [ ] `PHASES_5_10_ANALYSES.md` → archive
  - [ ] `PHASES_6_10_COMPLETE.md` → archive
  - [ ] `BUGFIX_V5_2026-02-15.md` → archive
  - [ ] `diagnostics/DIAGNOSTIC_LENTEURS_V5.md` → archive

- [ ] **Suivi** :
  - [ ] `SUIVI_AVANCEMENT_V5.1.md` (final) → archive
  - [ ] `MIGRATION_V5_FINAL_CHECKLIST.md` (complétée) → archive
  - [ ] Retrospectives sprints → archive

**Commande Archivage** :
```bash
python scripts/archive_project_docs.py --version v5.1 --date $(date +%Y%m%d)
```

#### ✅ Création README Archive

**Créer** : `.ai/archive/v5.1-completion-YYYYMMDD/README.md`

**Contenu** :
```markdown
# Archive Projet V5.1 — Completion YYYY-MM-DD

## Contexte

Projet v5.1 : Architecture Pure DuckDB + Polars
Durée : [X jours]
Date completion : YYYY-MM-DD

## Métriques Finales

[Tableau métriques atteintes]

## Documents Archivés

[Liste fichiers avec description]

## Leçons Apprises

[Retrospective globale projet]

## Références

- Tag release : v5.1.0
- Branch finale : main
- Commit final : [SHA]
```

---

### Phase 3 : Mise à Jour Documentation Utilisateur (2h)

#### ✅ Documentation Utilisateur Finale

**Fichiers à mettre à jour** :

- [ ] **README.md** (racine) :
  - [ ] Section "Getting Started" à jour
  - [ ] Section "Architecture" reflète v5.1
  - [ ] Captures d'écran UI mises à jour
  - [ ] Métriques performance v5.1
  - [ ] Suppression références v3/v4

- [ ] **docs/USER_GUIDE.md** :
  - [ ] Guide installation v5.1
  - [ ] Guide utilisation UI
  - [ ] FAQ mise à jour
  - [ ] Troubleshooting v5.1
  - [ ] Exemples à jour

- [ ] **docs/ARCHITECTURE_V5.md** :
  - [ ] Diagrammes architecture finaux
  - [ ] Schéma DB shared finalisé
  - [ ] Flow données v5.1
  - [ ] Décisions architecture documentées

- [ ] **CHANGELOG.md** :
  - [ ] Section v5.1.0 complète
  - [ ] Toutes modifications listées
  - [ ] Breaking changes documentés
  - [ ] Métriques avant/après

**Validation** :
```bash
python scripts/validate_user_docs.py
python scripts/check_screenshots_uptodate.py
python scripts/verify_no_legacy_references.py
```

---

### Phase 4 : Mise à Jour Documentation IA (2h)

#### ✅ Documentation IA Finale

**Fichiers à mettre à jour** :

- [ ] **CLAUDE.md** :
  - [ ] Section "Architecture v5.1" finale
  - [ ] Règles strictes (Pandas interdit, SQLite interdit)
  - [ ] Exemples code v5.1
  - [ ] Conventions finales
  - [ ] Suppression références legacy

- [ ] **COPILOT_INSTRUCTIONS.md** :
  - [ ] Guidelines v5.1
  - [ ] Patterns à suivre
  - [ ] Anti-patterns à éviter
  - [ ] Exemples migrations

- [ ] **.ai/project_map.md** :
  - [ ] Structure projet finale
  - [ ] Modules principaux
  - [ ] Dépendances
  - [ ] Points d'entrée

- [ ] **.ai/data_lineage.md** :
  - [ ] Flow données v5.1 final
  - [ ] Tables shared DB
  - [ ] Transformations
  - [ ] Sources/destinations

- [ ] **.ai/thought_log.md** :
  - [ ] Décisions architecture v5.1
  - [ ] Rationale choix techniques
  - [ ] Leçons apprises
  - [ ] Post-mortem projet

**Validation** :
```bash
grep -r "v3\|v4\|legacy\|SQLite\|Pandas" .ai/*.md CLAUDE.md COPILOT_INSTRUCTIONS.md
# → Vérifier que seules références historiques/archivées
```

---

### Phase 5 : Cleanup et Organisation (1h)

#### ✅ Cleanup Fichiers Temporaires

- [ ] **Supprimer fichiers build** :
  ```bash
  rm -rf .pytest_cache/
  rm -rf .mypy_cache/
  rm -rf .ruff_cache/
  rm -rf htmlcov/
  rm -rf dist/
  rm -rf build/
  ```

- [ ] **Supprimer fichiers temp** :
  ```bash
  find . -name "*.pyc" -delete
  find . -name "__pycache__" -type d -delete
  find . -name "*.tmp" -delete
  ```

- [ ] **Vérifier .gitignore** :
  - [ ] Tous patterns appropriés
  - [ ] Pas de fichiers sensibles commitables

#### ✅ Organisation Finale

- [ ] **Structure .ai/** :
  ```
  .ai/
  ├── archive/
  │   └── v5.1-completion-YYYYMMDD/  [Plans archivés]
  ├── diagnostics/                   [Conservés référence]
  ├── project_map.md                 [À jour v5.1]
  ├── data_lineage.md                [À jour v5.1]
  ├── thought_log.md                 [Complété]
  └── README.md                      [Index à jour]
  ```

- [ ] **Index .ai/** :
  - [ ] README.md liste tous documents
  - [ ] Pointe vers archives
  - [ ] Guide navigation

---

### Phase 6 : Release et Communication (1h)

#### ✅ Release Git

- [ ] **Tag final** :
  ```bash
  git tag -a v5.1.0 -m "Release v5.1.0 - Pure Architecture DuckDB + Polars"
  git push origin v5.1.0
  ```

- [ ] **Release notes** :
  - [ ] Créer fichier `RELEASE_NOTES_v5.1.0.md`
  - [ ] Lister toutes améliorations
  - [ ] Métriques avant/après
  - [ ] Breaking changes
  - [ ] Migration guide si nécessaire

- [ ] **GitHub Release** :
  - [ ] Créer release sur GitHub
  - [ ] Attacher release notes
  - [ ] Marquer comme stable

#### ✅ Communication

- [ ] **Annonce release** :
  - [ ] Email/message équipe
  - [ ] Documentation publiée
  - [ ] Migration guide disponible

- [ ] **Handoff** :
  - [ ] Guide maintenance créé
  - [ ] Points de contact documentés
  - [ ] Procédures support définies

---

### Phase 7 : Validation Finale et Clôture (30min)

#### ✅ Checklist Finale Projet

- [ ] **Code** :
  - [ ] Tous commits pushés
  - [ ] Tous tags créés
  - [ ] Branch main stable
  - [ ] Pas de WIP

- [ ] **Tests** :
  - [ ] Suite complète verte
  - [ ] Couverture ≥80%
  - [ ] Pas de tests skip
  - [ ] CI/CD passe

- [ ] **Documentation** :
  - [ ] Utilisateur à jour
  - [ ] IA à jour
  - [ ] Archives organisées
  - [ ] Cohérence vérifiée

- [ ] **Métriques** :
  - [ ] Toutes atteintes
  - [ ] Documentées
  - [ ] Baseline conservée

- [ ] **Cleanup** :
  - [ ] Fichiers temp supprimés
  - [ ] Structure organisée
  - [ ] .gitignore approprié

#### ✅ Sign-off Final

**Validation finale** :
```bash
python scripts/final_project_validation.py --version v5.1
```

**Résultat attendu** :
```
✅ Code : Stable
✅ Tests : 100% pass, 82% couverture
✅ Documentation : À jour
✅ Métriques : Toutes atteintes
✅ Archive : Complète
✅ Release : v5.1.0 créée

🎉 PROJET V5.1 COMPLÉTÉ AVEC SUCCÈS
```

---

## 📊 Checklist Master Projet V5.1

### Vue d'Ensemble

- [ ] Étape 0 : Préparation
- [ ] Étape 1 : Performance UI (3 sprints)
- [ ] Étape 2 : Performance Données
- [ ] Étape 3 : Architecture Shared DB (Ph5-6)
- [ ] Étape 4 : Éradication SQLite Runtime
- [ ] Étape 5 : Scripts Migration Bannières
- [ ] Étape 6 : Migration Pandas→Polars
- [ ] Étape 7 : Bugs Critiques
- [ ] Étape 8 : Cleanup Tables Legacy
- [ ] Étape 9 : Tests + Documentation
- [ ] Étape 10 : Release v5.1
- [ ] **Fin de Projet** : Archivage + Docs + Clôture

### Métriques Globales

- [ ] Connexion DB < 20ms
- [ ] load_matches(100) < 80ms
- [ ] Première page < 800ms
- [ ] SQLite runtime = 0
- [ ] Pandas métier = 0
- [ ] Tables obsolètes = 0
- [ ] Taille DB ~4MB
- [ ] Couverture tests ≥80%

### Validation Finale

- [ ] Tous tests verts
- [ ] Documentation complète
- [ ] Archives organisées
- [ ] Release créée
- [ ] Communication faite

---

## 📎 Annexes

### Annexe A : Scripts Utilitaires

**Scripts à créer pour validation** :

```bash
# Validation environnement
scripts/check_env.py

# Mesure baseline
scripts/measure_baseline.py

# Validation métriques
scripts/validate_all_metrics.py

# Audits
scripts/audit_sqlite_imports.py
scripts/audit_pandas_imports.py
scripts/validate_no_legacy_tables.py

# Archivage
scripts/archive_project_docs.py

# Validation finale
scripts/final_project_validation.py
```

### Annexe B : Templates

**Template Retrospective** : `.ai/retrospectives/TEMPLATE.md`

**Template Release Notes** : `RELEASE_NOTES_TEMPLATE.md`

**Template Archive README** : `.ai/archive/TEMPLATE_README.md`

---

## 🎯 Conclusion

Ce document définit **TOUS** les processus qualité pour garantir le succès du projet V5.1 :

✅ **Checklists avant/après chaque sprint**  
✅ **Definition of Done claire pour chaque étape**  
✅ **Processus revue de code rigoureux**  
✅ **Framework tests exhaustif**  
✅ **Procédures archivage et documentation fin de projet**

**Utilisation** : Référencer ce document à chaque étape du projet et cocher systématiquement toutes les cases de validation.

**Garantie** : Respect de ce processus = Projet V5.1 réussi avec qualité maximale 🎯✨
