# Tableau de Bord — Suivi d'Avancement Projet Unifié v5.1

> **Mise à jour** : 2026-02-16  
> **Statut global** : � EN COURS — Sprint 1bis terminé  
> **Progression** : 11/30 tâches (37%)

---

## 📊 Vue d'Ensemble

### Progression Globale

```
Sprint 0 : Préparation              [    ] 0/4 (0%)
Sprint 1 : Performance              [####] 3/3 (100%) ✅ TERMINÉ
Sprint 1bis : Perf Root Causes      [####] 5/5 (100%) ✅ TERMINÉ
Sprint 2 : Éradication SQLite       [#   ] 2/6 (33%) — import sqlite3 déjà nettoyé
Sprint 3 : Migration Pandas         [#   ] 1/7 (14%) — performance_score.py déjà migré
Sprint 4 : Cleanup & Validation     [    ] 0/5 (0%)

TOTAL : 11/30 (37%)
```

### Temps Consommé vs Estimé

| Sprint | Estimé | Réel | Écart | Statut |
|--------|--------|------|-------|--------|
| Sprint 0 | 2h | - | - | ⏳ À démarrer |
| Sprint 1 | 8h | - | - | ✅ Terminé |
| Sprint 1bis | 4.5h | ~3h | -1.5h | ✅ Terminé |
| Sprint 2 | 6h | - | - | 🟡 Partiellement fait (33%) |
| Sprint 3 | 8h | - | - | 🟡 Partiellement fait (14%) |
| Sprint 4 | 4h | - | - | ⏳ À démarrer |
| **TOTAL** | **28.5h** | **~11h** | **-** | **37%** |

---

## 📅 Sprint 0 : Préparation (2h) — CRITIQUE

### Objectif
Établir filet de sécurité avant toute modification.

### Tâches

#### 0.1 Backups Production (30min)
- [ ] Backups de tous les joueurs configurés
- [ ] Backup warehouse (metadata + shared_matches)
- [ ] Test de restauration d'un backup
- [ ] Validation taille backups (>100MB)

**Livrable** : `backups/v5.1_pre_project_*/`

#### 0.2 Baseline Performance (45min)
- [ ] Créer/vérifier script diagnose_performance.py
- [ ] Exécuter diagnostic (10 runs minimum)
- [ ] Capturer métriques : connexion, load_matches, première page
- [ ] Sauvegarder résultats JSON

**Livrable** : `.ai/reports/baseline_v5.0.json`

#### 0.3 Validation Architecture v5 (30min)
- [ ] Vérifier intégrité shared_matches.duckdb
- [ ] Vérifier schéma (audit_current_data.py)
- [ ] Tests unitaires verts
- [ ] Validation 100% des matchs présents

**Critères** : Tests verts + shared_matches.duckdb valide

#### 0.4 Branche de Secours (15min)
- [ ] Créer branche `backup/pre-v5.1-project`
- [ ] Pousser sur origin
- [ ] Créer branche travail `feature/v5.1-unified-project`
- [ ] Vérifier checkout correct

**Livrables** : 2 branches créées

### Validation Sprint 0
- [ ] **Go/No-Go humain** : Validation stakeholder
- [ ] Tous les backups testés
- [ ] Baseline documentée
- [ ] Branches créées

**Date de validation** : _____________

---

## 📅 Sprint 1 : Optimisation Performance — ✅ TERMINÉ

### Objectif
Rendre v5 UI 2× plus rapide que v3.

### Tâches (toutes complétées)

#### 1.1 Vue Matérialisée `mv_player_matches` ✅
- [x] Migration `ensure_mv_player_matches_view()` dans `migrations.py`
- [x] `_get_match_source()` utilise la vue avec auto-détection + fallback legacy
- [x] Tests dans `test_performance_optimizations.py`

#### 1.2 Cache Repository Streamlit ✅
- [x] `get_cached_repository_st()` avec `@st.cache_resource(ttl=3600)` dans `cache_loaders.py`
- [x] Pages UI principales utilisent le cache (0 usage direct de `RepositoryFactory` dans pages)

#### 1.3 Index DuckDB ✅
- [x] 16+ index créés sur 9 tables (dont `idx_mp_xuid_match`, `idx_mp_match_xuid`, `idx_mr_start_time`)

---

## 📅 Sprint 1bis : Causes Racines Performance (~4.5h) — ✅ TERMINÉ

### Origine
Audit post-Sprint 1 (2026-02-16) : malgré les optimisations du Sprint 1, des lenteurs persistent. 5 causes racines identifiées par analyse du code.

**Date de complétion** : 2026-02-16

### Causes racines identifiées

| RC# | Cause racine | Impact | Fichier principal |
|-----|-------------|--------|-------------------|
| RC1 | 8 fonctions `cache_loaders` créent des connexions neuves (3× ATTACH) au lieu d'utiliser `get_cached_repository_st()` | **CRITIQUE** — 50-100ms × N appels | `src/ui/cache_loaders.py` |
| RC2 | `_build_metadata_resolution()` et `_build_mmr_fallback()` font des requêtes `information_schema` non cachées à chaque appel | **IMPORTANT** — 2 requêtes/appel | `src/data/repositories/duckdb_repo.py` |
| RC3 | 3 LEFT JOIN metadata redondants quand `mv_player_matches` est utilisé (noms déjà résolus dans la vue) | **MOYEN** — jointures inutiles | `src/data/repositories/_match_queries.py` |
| RC4 | Sous-requête imbriquée (mv + match_stats local + metadata + pms) = 5 tables jointes tant que match_stats local n'est pas nettoyé | **MOYEN** — résolu par skip quand uses_mv=True |
| RC5 | `cached_load_highlight_events_for_match()` ouvre une connexion brute `duckdb.connect()` | **MINEUR** — bypass cache | `src/ui/cache_loaders.py` |

### Tâches

#### 1bis.1 RC1 — Migrer 8+ fonctions cache_loaders vers `get_cached_repository_st()` ✅
- [x] `cached_same_team_match_ids_with_friend()` → `get_cached_repository_st()`
- [x] `cached_query_matches_with_friend()` → `get_cached_repository_st()`
- [x] `cached_load_player_match_result()` → `get_cached_repository_st()`
- [x] `cached_load_match_medals_for_player()` → `get_cached_repository_st()`
- [x] `cached_load_match_rosters()` → `get_cached_repository_st()`
- [x] `cached_load_top_medals()` → `get_cached_repository_st()`
- [x] `top_medals_smart()` → `get_cached_repository_st()`
- [x] `cached_list_top_teammates()` → `get_cached_repository_st()`
- [x] `cached_get_cache_stats()` → `get_cached_repository_st()`
- [x] `cached_load_match_player_gamertags()` → `get_cached_repository_st()` + `repo.load_highlight_events()`
- [x] `cached_list_other_xuids()` → `get_cached_repository_st()` + `repo.list_other_player_xuids()`

**Livrable** : ✅ Zéro `DuckDBRepository(db_path, ...)` direct dans cache_loaders.py

#### 1bis.2 RC5 — Migrer `cached_load_highlight_events_for_match()` ✅
- [x] Remplacé `duckdb.connect(db_path)` brut par `get_cached_repository_st()` + `repo.load_highlight_events()`
- [x] Supprimé le parsing JSON manuel (déjà fait dans le repo)

**Livrable** : ✅ Zéro `duckdb.connect()` brut dans cache_loaders.py (hors `_resolve_player_xuid`)

#### 1bis.3 RC2 — Cacher `_build_metadata_resolution()` et `_build_mmr_fallback()` ✅
- [x] Cache `self._metadata_resolution_cache` ajouté
- [x] Cache `self._mmr_fallback_cache` ajouté
- [x] Caches invalidés dans `close()`

**Livrable** : ✅ 0 requête `information_schema` après le premier appel

#### 1bis.4 RC3 — Supprimer jointures metadata redondantes en mode v5.1 ✅
- [x] `_get_match_source()` retourne un 3-tuple `(source, params, uses_mv)`
- [x] Quand `uses_mv=True`, skip `_build_metadata_resolution()` et `_build_mmr_fallback()` dans les 5 méthodes de chargement
- [x] Direct column references `match_stats.map_name/playlist_name/pair_name` utilisées

**Livrable** : ✅ 3 LEFT JOIN metadata + 1 LEFT JOIN pms en moins sur le chemin critique v5.1

#### 1bis.5 RC4 — Skip jointures MMR redondantes en mode v5.1 ✅
- [x] Quand `uses_mv=True`, skip `_build_mmr_fallback()` (MMR déjà COALESCE dans la sous-requête mv_player_matches)
- [x] `team_mmr_expr` et `enemy_mmr_expr` restent `match_stats.team_mmr/enemy_mmr` (déjà enrichis dans la source)
- [x] Tests de non-régression sur l'affichage MMR (2885 tests passent)

**Livrable** : ✅ En mode v5.1, requêtes simplifiées sans jointures redondantes

### Validation Sprint 1bis ✅
- [x] Suite de tests complète verte (2885 passed, 0 failed)
- [x] 7 tests mis à jour pour le nouveau 3-tuple `_get_match_source()`
- [x] 2 tests corrigés pour PermissionError (cache_resource cleanup)
- [ ] Benchmark avant/après (connexion, load_matches, première page)
- [ ] Validation UI manuelle (5 pages)
- [ ] **Go/No-Go humain**

### Métriques cibles

| Métrique | Actuel | Objectif |
|----------|--------|----------|
| Temps connexion | 80ms | <20ms |
| load_matches(100) | 200ms | <80ms |
| Première page UI | 1500ms | <800ms |

**Date de validation** : _____________

---

## 📅 Sprint 2 : Éradication SQLite (6h) — CRITIQUE

### Objectif
Zéro SQLite en runtime.

### État de l'audit (2026-02-16)
> ✅ `import sqlite3` dans `src/` = **0 occurrence** (déjà nettoyé)
> ⚠️ `metadata.db` encore référencé dans 3 fichiers : `paths.py:62`, `duckdb_engine.py:13,92`, `engine.py:112`
> ⚠️ Méthode dépréciée `attach_sqlite` encore présente dans `duckdb_engine.py`
> ⚠️ Bannières LEGACY dans scripts/migration/ : **0** (pas encore fait)

### Tâches

#### 2.1 Supprimer Fallback `engine.py` (1h)
- [ ] Modifier `src/data/query/engine.py` (ligne 112 : référence `metadata.db`)
- [ ] Remplacer `if/elif` par `if not exists: raise`
- [ ] Supprimer références à metadata.db
- [ ] Test : échec si metadata.duckdb absent
- [ ] Tests existants verts

**Livrable** : Code modifié + test

#### 2.2 Supprimer Fallback `duckdb_engine.py` (1h)
- [ ] Modifier `src/data/infrastructure/database/duckdb_engine.py` (lignes 13, 92 : références `metadata.db`)
- [ ] Supprimer la méthode dépréciée `attach_sqlite`
- [ ] Même logique que 2.1
- [ ] Tests verts

**Livrable** : Code modifié

#### 2.3 Nettoyer Références `.db` (1.5h)
- [x] ~~Supprimer imports `sqlite3` dans `src/`~~ ✅ Déjà fait (0 occurrence)
- [ ] Nettoyer `src/utils/paths.py` (ligne 62 : référence `metadata.db`)
- [ ] Vérifier `db_profiles.json`
- [ ] Vérifier `app_settings.json`
- [ ] Validation : zéro `.db` (hors `.duckdb`) dans src/

**Livrable** : Code nettoyé

#### 2.4 Marquer Scripts Migration LEGACY (1.5h)
- [ ] Ajouter bannière LEGACY à `migrate_player_to_duckdb.py`
- [ ] Ajouter bannière LEGACY à `migrate_all_to_duckdb.py`
- [ ] Ajouter bannière LEGACY à `migrate_metadata_to_duckdb.py`
- [ ] Ajouter bannière LEGACY à `migrate_player_to_shared.py`
- [ ] Créer `scripts/migration/README.md`
- [ ] Décision `refetch_film_roster.py` (supprimer ou marquer)

**Livrable** : 4 scripts marqués + README

#### 2.5 Tests & Validation Sprint 2 (1h)
- [x] ~~Vérifier zéro `import sqlite3` runtime~~ ✅ Déjà vérifié
- [ ] Vérifier zéro `.db` dans config
- [ ] Suite de tests verte
- [ ] Validation UI (aucune régression)

**Livrables** :
- `.ai/reports/sprint2_validation.md`

### Validation Sprint 2
- [ ] **Go/No-Go humain** : Validation éradication SQLite
- [ ] Zéro SQLite runtime
- [ ] Tests verts
- [ ] Scripts migration documentés

**Date de validation** : _____________

---

## 📅 Sprint 3 : Migration Pandas → Polars (8h réduit) — IMPORTANT

### Objectif
Zéro Pandas dans code métier.

### État de l'audit (2026-02-16)
> ✅ `src/analysis/` a **0 import pandas** — `performance_score.py` déjà migré Polars
> ⚠️ `src/ui/pages/win_loss.py:8` : import pandas runtime (acceptable = frontière UI Streamlit/Plotly)
> ⚠️ `src/ui/cache_filters.py:283` : utilise `.empty` (pattern Pandas sur un objet potentiellement Polars — bug)
> ℹ️ 4 fichiers avec `import pandas` en `TYPE_CHECKING` only — OK, pas prioritaire

### Tâches

#### 3.1 ~~Migrer `performance_score.py`~~ ✅ DÉJÀ FAIT
- [x] ~~Audit usage Pandas~~ — 0 import pandas dans `src/analysis/`
- [x] ~~Traduction Polars~~ — déjà en Polars

#### 3.2 Migrer `win_loss_service.py` (3h)
- [ ] Audit + traduction Polars
- [ ] Tests de non-régression
- [ ] Validation

**Livrable** : Module migré + tests

#### 3.3 Migrer `objective_analysis.py` (2h)
- [ ] Audit + traduction Polars
- [ ] Utiliser `_compat.to_pandas_for_plotly()` en frontière
- [ ] Tests

**Livrable** : Module migré + tests

#### 3.4 Migrer `match_view_helpers.py` (1h)
- [ ] Migration Polars
- [ ] Tests

#### 3.5 Migrer `win_loss.py` (1h)
- [ ] Migration Polars (garder `.to_pandas()` uniquement à la frontière Plotly/Streamlit)
- [ ] Tests

#### 3.6 Migrer `cache_filters.py` (0.5h)
- [ ] **Bug** : corriger `.empty` (L283) — pattern Pandas, utiliser `len(df) == 0` ou `df.is_empty()` (Polars)
- [ ] Migration Polars complète
- [ ] Tests

#### 3.7 Migrer `duckdb_analytics.py` (0.5h)
- [ ] Migration Polars
- [ ] Tests

### Validation Sprint 3
- [ ] **Go/No-Go humain** : Validation migration Pandas
- [ ] Zéro `import pandas` en métier
- [ ] Bridges conservés (`_compat.py`)
- [ ] Tous les tests passent
- [ ] Aucune régression fonctionnelle

**Livrables** :
- `.ai/reports/sprint3_migration.md`

**Date de validation** : _____________

---

## 📅 Sprint 4 : Cleanup & Validation (4h) — MOYEN

### Objectif
Nettoyage final et validation complète.

### Tâches

#### 4.1 Cleanup DBs Player (1h)
- [ ] Dry-run : `cleanup_player_dbs_v5.py --all --dry-run`
- [ ] Backup + cleanup : `--all --backup --remove-compat-views`
- [ ] Validation : tables supprimées
- [ ] Validation : application fonctionne
- [ ] Tests verts

**Livrable** : Player DBs nettoyées (-85% taille)

#### 4.2 Audit Scripts Archive (1h)
- [ ] Inventaire complet `scripts/_archive/`
- [ ] Classification (supprimer/garder/documenter)
- [ ] Créer `scripts/_archive/README.md`
- [ ] Supprimer scripts R&D terminée (optionnel)

**Livrable** : Archive organisée + README

#### 4.3 Suite Tests Complète (1h)
- [ ] Tests unitaires verts
- [ ] Tests d'intégration verts
- [ ] Couverture ≥80%
- [ ] Benchmark comparatif final

**Livrables** :
- `.ai/reports/v5.1_benchmark_comparison.md`

#### 4.4 Documentation Finale (1h)
- [ ] Mettre à jour `docs/ARCHITECTURE_V5.md`
- [ ] Mettre à jour `CLAUDE.md`
- [ ] Mettre à jour `README.md`
- [ ] Mettre à jour `.ai/thought_log.md`
- [ ] Créer `.ai/RELEASE_NOTES_V5.1.md`

**Livrables** : 5 fichiers docs à jour + release notes

#### 4.5 Release 5.1 & Changelog (30min)
- [ ] Ajouter section `## [5.1.0] - YYYY-MM-DD` dans `CHANGELOG.md` (en haut, après le header, AVANT la section 5.0.0 existante — ne pas supprimer l'existant)
  - [ ] Section `### Added` : vue matérialisée `mv_player_matches`, cache repository Streamlit, index DuckDB performance
  - [ ] Section `### Changed` : 24 pages UI migrées vers cache, modules Pandas → Polars (performance_score, win_loss_service, objective_analysis, match_view_helpers, win_loss, cache_filters, duckdb_analytics)
  - [ ] Section `### Removed` : fallbacks SQLite runtime (engine.py, duckdb_engine.py), imports `sqlite3` dans src/, références `.db` dans configs
  - [ ] Section `### Performance` : métriques avant/après (connexion, load_matches, première page UI)
- [ ] Bumper version dans `pyproject.toml` (5.0.0 → 5.1.0)
- [ ] Créer tag git `v5.1.0`
- [ ] Créer release GitHub via `gh release create v5.1.0` avec les release notes

**Livrables** : `CHANGELOG.md` mis à jour + tag `v5.1.0` + release GitHub

### Validation Finale Sprint 4
- [ ] **Go/No-Go humain** : Validation finale projet
- [ ] Toutes les métriques atteintes
- [ ] Tests verts (≥80% couverture)
- [ ] Documentation complète
- [ ] Release notes créées
- [ ] CHANGELOG.md à jour avec section 5.1.0
- [ ] Tag git v5.1.0 créé
- [ ] Release GitHub publiée

**Livrables** :
- `.ai/reports/sprint4_final.md`
- `.ai/RELEASE_NOTES_V5.1.md`

**Date de validation** : _____________

---

## 📊 Métriques Finales (À Remplir Après Projet)

### Résultats vs Objectifs

| Métrique | v5.0 | v5.1 | Objectif | Écart | Statut |
|----------|------|------|----------|-------|--------|
| **Architecture** ||||||
| Imports SQLite runtime | 7 | 0 | 0 | 0 | ✅ |
| Refs `metadata.db` | ? | 3 | 0 | -3 | ⏳ |
| Imports Pandas métier | 7 | ~5 | 0 | -5 | ⏳ |
| Taille player DB | 30 MB | - | 4 MB | - | ⏳ |
| **Performance** ||||||
| Temps connexion | 80ms | - | <20ms | - | ⏳ |
| load_matches(100) | 200ms | - | <80ms | - | ⏳ |
| Première page UI | 1500ms | - | <800ms | - | ⏳ |
| **Qualité** ||||||
| Lignes de code | 45k | - | 38k | - | ⏳ |
| Couverture tests | 75% | - | ≥80% | - | ⏳ |

---

## 📝 Journal de Bord

### Format d'Entrée

```
Date : YYYY-MM-DD HH:MM
Sprint : X
Tâche : X.X
Durée : XXh
Statut : ✅ Complétée / ⚠️ Problème / 🔄 En cours

Résumé :
- ...

Problèmes rencontrés :
- ...

Décisions prises :
- ...

Next steps :
- ...
```

---

### Entrées

_Ajouter les entrées au fur et à mesure du projet..._

---

## 🎯 Actions Requises

### Sprint Actuel : Sprint 0

**Prochaines tâches** :
1. Exécuter backups production
2. Capturer baseline performance
3. Valider architecture v5
4. Créer branches

**Blocages** : Aucun

**Validation requise** : Go/No-Go humain après Sprint 0

---

## 📞 Contact

### Pour Questions

- Consulter [PROJECT_UNIFIE_V5.1.md](.ai/PROJECT_UNIFIE_V5.1.md) pour détails tâches
- Voir [thought_log.md](.ai/thought_log.md) pour historique décisions

### Pour Rapporter Problème

1. Noter dans Journal de Bord
2. Décider : bloquer ou contourner
3. Documenter décision

---

**Dernière mise à jour** : 2026-02-16 — Création tableau de bord ✅

**Prochain sprint** : Sprint 0 (Préparation)
