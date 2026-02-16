# Tableau de Bord — Suivi d'Avancement Projet Unifié v5.1

> **Mise à jour** : 2026-02-16  
> **Statut global** : 📋 PLANIFIÉ — Prêt pour exécution  
> **Progression** : 0/26 tâches (0%)

---

## 📊 Vue d'Ensemble

### Progression Globale

```
Sprint 0 : Préparation              [    ] 0/4 (0%)
Sprint 1 : Performance              [    ] 0/4 (0%)
Sprint 2 : Éradication SQLite       [    ] 0/6 (0%)
Sprint 3 : Migration Pandas         [    ] 0/8 (0%)
Sprint 4 : Cleanup & Validation     [    ] 0/4 (0%)

TOTAL : 0/26 (0%)
```

### Temps Consommé vs Estimé

| Sprint | Estimé | Réel | Écart | Statut |
|--------|--------|------|-------|--------|
| Sprint 0 | 2h | - | - | ⏳ À démarrer |
| Sprint 1 | 8h | - | - | ⏳ À démarrer |
| Sprint 2 | 6h | - | - | ⏳ À démarrer |
| Sprint 3 | 12h | - | - | ⏳ À démarrer |
| Sprint 4 | 4h | - | - | ⏳ À démarrer |
| **TOTAL** | **32h** | **0h** | **-** | **0%** |

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

## 📅 Sprint 1 : Optimisation Performance (8h) — PRIORITÉ 1

### Objectif
Rendre v5 UI 2× plus rapide que v3.

### Tâches

#### 1.1 Vue Matérialisée `mv_player_matches` (2h)
- [ ] Créer migration `migration_v5_1_create_mv_player_matches`
- [ ] Adapter `_get_match_source()` pour utiliser la vue
- [ ] Tests : vue existe + colonnes correctes
- [ ] Tests : performance (<50ms pour 100 matchs)
- [ ] Tests : calcul KDA correct

**Livrable** : Vue créée + tests verts

#### 1.2 Cache Repository Streamlit (3h)
- [ ] Créer `src/ui/data_loader.py` avec `@st.cache_resource`
- [ ] Fonction `get_cached_repository(gamertag, xuid)`
- [ ] Fonction `clear_repository_cache()`
- [ ] Migrer 24 pages UI (remplacer RepositoryFactory.create)
- [ ] Tests : cache retourne même instance
- [ ] Tests : joueurs différents = repos différents

**Livrable** : 24 pages migrées + tests verts

#### 1.3 Index DuckDB (1h)
- [ ] Créer migration `migration_v5_1_create_performance_indexes`
- [ ] Index `idx_mp_xuid_match` sur (xuid, match_id)
- [ ] Index `idx_mp_match_xuid` sur (match_id, xuid)
- [ ] Tests : index existent
- [ ] Validation : query plan utilise les index

**Livrable** : Index créés

#### 1.4 Tests & Validation Sprint 1 (2h)
- [ ] Suite de tests complète verte
- [ ] Benchmark final Sprint 1
- [ ] Comparaison avec baseline
- [ ] Validation UI manuelle (5 pages)
- [ ] Rapport de gains généré

**Livrables** :
- `.ai/reports/sprint1_final.json`
- `.ai/reports/sprint1_gains.md`

### Métriques Sprint 1

| Métrique | Avant | Après | Objectif | Statut |
|----------|-------|-------|----------|--------|
| Temps connexion | 80ms | - | <20ms | ⏳ |
| load_matches(100) | 200ms | - | <80ms | ⏳ |
| Première page UI | 1500ms | - | <800ms | ⏳ |

### Validation Sprint 1
- [ ] **Go/No-Go humain** : Validation gains performance
- [ ] Métriques atteintes (≥90% objectifs)
- [ ] Aucune régression
- [ ] UI réactive

**Date de validation** : _____________

---

## 📅 Sprint 2 : Éradication SQLite (6h) — CRITIQUE

### Objectif
Zéro SQLite en runtime.

### Tâches

#### 2.1 Supprimer Fallback `engine.py` (1h)
- [ ] Modifier `src/data/query/engine.py` (lignes 110-123)
- [ ] Remplacer `if/elif` par `if not exists: raise`
- [ ] Supprimer références à metadata.db
- [ ] Test : échec si metadata.duckdb absent
- [ ] Tests existants verts

**Livrable** : Code modifié + test

#### 2.2 Supprimer Fallback `duckdb_engine.py` (1h)
- [ ] Modifier `src/data/infrastructure/database/duckdb_engine.py`
- [ ] Même logique que 2.1
- [ ] Tests verts

**Livrable** : Code modifié

#### 2.3 Nettoyer Références `.db` (1.5h)
- [ ] Audit : `grep -r "\.db" src/`
- [ ] Nettoyer `src/utils/paths.py`
- [ ] Vérifier `db_profiles.json`
- [ ] Vérifier `app_settings.json`
- [ ] Supprimer imports `sqlite3` dans `src/ui/`, `src/ai/`
- [ ] Validation : zéro `.db` (hors `.duckdb`)

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
- [ ] Vérifier zéro `import sqlite3` runtime
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

## 📅 Sprint 3 : Migration Pandas → Polars (12h) — IMPORTANT

### Objectif
Zéro Pandas dans code métier.

### Tâches

#### 3.1 Migrer `performance_score.py` (4h)
- [ ] Audit usage Pandas
- [ ] Traduction Polars
- [ ] Tests de non-régression
- [ ] Validation : mêmes résultats
- [ ] Benchmark (optionnel)

**Livrable** : Module migré + tests

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
- [ ] Migration Polars
- [ ] Tests

#### 3.6 Migrer `cache_filters.py` (0.5h)
- [ ] Migration Polars
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

### Validation Finale Sprint 4
- [ ] **Go/No-Go humain** : Validation finale projet
- [ ] Toutes les métriques atteintes
- [ ] Tests verts (≥80% couverture)
- [ ] Documentation complète
- [ ] Release notes créées

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
| Imports SQLite runtime | 7 | - | 0 | - | ⏳ |
| Imports Pandas métier | 7 | - | 0 | - | ⏳ |
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
