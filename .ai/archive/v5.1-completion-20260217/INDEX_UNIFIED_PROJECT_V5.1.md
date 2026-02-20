# Index — Documentation Projet Unifié v5.1

> **Dernière mise à jour** : 2026-02-16  
> **Projet** : LevelUp v5.0 → v5.1 (Architecture Pure + Performance Optimale)

---

## 📚 Documents Principaux

### 🎯 Plan de Projet

| Document | Description | Temps de lecture | Public |
|----------|-------------|------------------|--------|
| **[PROJECT_UNIFIE_V5.1.md](.ai/PROJECT_UNIFIE_V5.1.md)** | Plan maître détaillé (5 sprints, 32h) | 45 min | Tous |
| **[GUIDE_DEMARRAGE_RAPIDE_V5.1.md](.ai/GUIDE_DEMARRAGE_RAPIDE_V5.1.md)** | Guide de démarrage en 15 min | 10 min | Développeurs |
| **[SUIVI_AVANCEMENT_V5.1.md](.ai/SUIVI_AVANCEMENT_V5.1.md)** | Tableau de bord (planifié vs accompli) | 5 min | Chef de projet |

### 📖 Documentation Technique

| Document | Description |
|----------|-------------|
| [V5.1_PURE_ARCHITECTURE.md](../docs/V5.1_PURE_ARCHITECTURE.md) | Manifeste architecture pure v5.1 |
| [ARCHITECTURE_V5.md](../docs/ARCHITECTURE_V5.md) | Architecture v5 détaillée |
| [SHARED_MATCHES_SCHEMA.md](../docs/SHARED_MATCHES_SCHEMA.md) | Schéma DDL complet |

### 📋 Documents Sources (Pré-Unification)

| Document | Sujet |
|----------|-------|
| [PLAN_ERADICATION_LEGACY_V5.md](.ai/PLAN_ERADICATION_LEGACY_V5.md) | Plan initial éradication legacy (60 pages) |
| [DIAGNOSTIC_LENTEURS_V5.md](.ai/diagnostics/DIAGNOSTIC_LENTEURS_V5.md) | Analyse performance v5 |
| [PLAN_OPTIMISATION_V5.md](.ai/diagnostics/PLAN_OPTIMISATION_V5.md) | Plan initial optimisation |
| [SYNTHESE_EXECUTIVE_V5.1.md](.ai/SYNTHESE_EXECUTIVE_V5.1.md) | Résumé exécutif éradication |

---

## 🗺️ Navigation Rapide

### Pour Démarrer le Projet

```
1. Lire GUIDE_DEMARRAGE_RAPIDE_V5.1.md (10 min)
   ↓
2. Lire PROJECT_UNIFIE_V5.1.md (45 min)
   ↓
3. Exécuter Sprint 0 : Préparation (2h)
   ↓
4. Validation Go/No-Go humaine
   ↓
5. Sprint 1 : Performance (8h)
```

### Pour Suivre l'Avancement

```
1. Consulter SUIVI_AVANCEMENT_V5.1.md
   ↓
2. Marquer tâches complétées (✓)
   ↓
3. Comparer planifié vs accompli
   ↓
4. Générer rapports de sprint
```

### Pour Comprendre l'Architecture

```
1. Lire V5.1_PURE_ARCHITECTURE.md (30 min)
   ↓
2. Consulter ARCHITECTURE_V5.md (référence)
   ↓
3. Voir SHARED_MATCHES_SCHEMA.md (schéma)
```

---

## 📊 Structure du Projet Unifié

### Vue d'Ensemble

```
Projet Unifié v5.1 (32h)
│
├── Sprint 0 : Préparation (2h)
│   └── Backups, baseline, validation
│
├── Sprint 1 : Performance (8h) 🔴 PRIORITÉ
│   ├── Vue matérialisée
│   ├── Cache repository
│   └── Index DuckDB
│
├── Sprint 2 : Éradication SQLite (6h)
│   ├── Supprimer fallbacks
│   └── Marquer scripts LEGACY
│
├── Sprint 3 : Migration Pandas (12h)
│   └── 7 fichiers métier → Polars
│
└── Sprint 4 : Cleanup (4h)
    ├── Cleanup DBs
    ├── Audit archive
    └── Documentation finale
```

---

## 📅 Planning et Livrables

### Calendrier Recommandé

| Jour | Sprint | Durée | Livrables |
|------|--------|-------|-----------|
| **J0** | Préparation | 2h | Backups, baseline, branche secours |
| **J1** | Performance | 8h | Vue, cache, index + tests |
| **J2** | Éradication SQLite | 6h | Zéro SQLite runtime |
| **J3** | Migration Pandas (1/2) | 7h | 4 fichiers migrés |
| **J4** | Migration Pandas (2/2) + Cleanup | 7h | 3 fichiers + cleanup + docs |
| **J5** | Buffer / Validation | 2h | Tests finaux, démo |

**Total** : 32 heures réparties sur 5 jours (6h/jour en moyenne)

---

## 🎯 Métriques de Succès

### Tableau de Bord Unifié

| Catégorie | Métrique | v5.0 | v5.1 | Objectif |
|-----------|----------|------|------|----------|
| **Architecture** ||||
|| Imports SQLite runtime | 7 | TBD | **0** |
|| Imports Pandas métier | 7 | TBD | **0** |
|| Taille player DB | 30 MB | TBD | **4 MB** |
| **Performance** ||||
|| Temps connexion | 80ms | TBD | **<20ms** |
|| load_matches(100) | 200ms | TBD | **<80ms** |
|| Première page UI | 1500ms | TBD | **<800ms** |
| **Qualité** ||||
|| Lignes de code | 45k | TBD | **38k** |
|| Couverture tests | 75% | TBD | **≥80%** |

**TBD** : Sera mesuré après chaque sprint

---

## 📂 Organisation des Fichiers

### Documentation Projet (.ai/)

```
.ai/
├── PROJECT_UNIFIE_V5.1.md                 ⭐ Plan maître
├── GUIDE_DEMARRAGE_RAPIDE_V5.1.md         ⭐ Démarrage rapide
├── INDEX_UNIFIED_PROJECT_V5.1.md          ⭐ Index (ce fichier)
├── SUIVI_AVANCEMENT_V5.1.md               ⭐ Tableau de bord
│
├── PLAN_ERADICATION_LEGACY_V5.md          📋 Plan initial legacy
├── SYNTHESE_EXECUTIVE_V5.1.md             📋 Résumé exécutif
├── INDEX_ERADICATION_LEGACY_V5.1.md       📋 Index legacy
│
├── diagnostics/
│   ├── DIAGNOSTIC_LENTEURS_V5.md          🔍 Analyse perf
│   ├── PLAN_OPTIMISATION_V5.md            🔍 Plan perf
│   └── RESUME_EXECUTIF.md                 🔍 Résumé perf
│
└── reports/
    ├── baseline_v5.0.json                 📊 Baseline
    ├── sprint1_final.json                 📊 Résultats Sprint 1
    ├── sprint2_validation.md              📊 Résultats Sprint 2
    ├── sprint3_migration.md               📊 Résultats Sprint 3
    ├── sprint4_final.md                   📊 Résultats Sprint 4
    └── v5.1_benchmark_comparison.md       📊 Comparatif final
```

### Documentation Architecture (docs/)

```
docs/
├── V5.1_PURE_ARCHITECTURE.md              🏗️ Manifeste v5.1
├── ARCHITECTURE_V5.md                     🏗️ Architecture v5
├── SHARED_MATCHES_SCHEMA.md               🏗️ Schéma DDL
├── CLEANUP_V5.md                          🧹 Guide cleanup
├── MIGRATION_V4_TO_V5.md                  📦 Guide migration
└── POLARS_MIGRATION.md                    🔄 Guide Polars
```

---

## 🔗 Liens Rapides par Rôle

### Développeur

**Démarrer** :
1. [GUIDE_DEMARRAGE_RAPIDE_V5.1.md](.ai/GUIDE_DEMARRAGE_RAPIDE_V5.1.md)
2. [V5.1_PURE_ARCHITECTURE.md](../docs/V5.1_PURE_ARCHITECTURE.md)
3. [PROJECT_UNIFIE_V5.1.md](.ai/PROJECT_UNIFIE_V5.1.md)

**Développer** :
- Guidelines : [V5.1_PURE_ARCHITECTURE.md § Conventions](../docs/V5.1_PURE_ARCHITECTURE.md#conventions)
- Anti-patterns : [V5.1_PURE_ARCHITECTURE.md § Anti-Patterns](../docs/V5.1_PURE_ARCHITECTURE.md#anti-patterns)
- Tests : [TESTING_V5.md](../docs/TESTING_V5.md)

---

### Chef de Projet

**Démarrer** :
1. [SUIVI_AVANCEMENT_V5.1.md](.ai/SUIVI_AVANCEMENT_V5.1.md)
2. [PROJECT_UNIFIE_V5.1.md § Métriques](.ai/PROJECT_UNIFIE_V5.1.md#metriques)
3. [SYNTHESE_EXECUTIVE_V5.1.md](.ai/SYNTHESE_EXECUTIVE_V5.1.md)

**Suivre** :
- Tableau de bord : [SUIVI_AVANCEMENT_V5.1.md](.ai/SUIVI_AVANCEMENT_V5.1.md)
- KPIs : [PROJECT_UNIFIE_V5.1.md § Métriques](.ai/PROJECT_UNIFIE_V5.1.md#metriques)
- Risques : [PROJECT_UNIFIE_V5.1.md § Risques](.ai/PROJECT_UNIFIE_V5.1.md#risques)

---

### Architecte / Reviewer

**Comprendre** :
1. [V5.1_PURE_ARCHITECTURE.md](../docs/V5.1_PURE_ARCHITECTURE.md)
2. [ARCHITECTURE_V5.md](../docs/ARCHITECTURE_V5.md)
3. [DIAGNOSTIC_LENTEURS_V5.md](.ai/diagnostics/DIAGNOSTIC_LENTEURS_V5.md)

**Valider** :
- Checklist conformité : [V5.1_PURE_ARCHITECTURE.md § Checklist](../docs/V5.1_PURE_ARCHITECTURE.md#checklist)
- Tests : [PROJECT_UNIFIE_V5.1.md § Tests](.ai/PROJECT_UNIFIE_V5.1.md#tests)
- Benchmarks : `.ai/reports/`

---

## 📊 Rapports et Analyses

### Rapports de Baseline

| Fichier | Description |
|---------|-------------|
| `.ai/reports/baseline_v5.0.json` | Métriques v5.0 (avant projet) |
| `.ai/archive/v5.0/v5-baseline-audit.md` | Audit complet v5.0 |

### Rapports par Sprint

| Sprint | Fichier | Contenu |
|--------|---------|---------|
| **Sprint 1** | `.ai/reports/sprint1_gains.md` | Gains performance |
| **Sprint 2** | `.ai/reports/sprint2_validation.md` | Validation éradication SQLite |
| **Sprint 3** | `.ai/reports/sprint3_migration.md` | Migration Pandas → Polars |
| **Sprint 4** | `.ai/reports/sprint4_final.md` | Validation finale |

### Rapport Final

| Fichier | Description |
|---------|-------------|
| `.ai/reports/v5.1_benchmark_comparison.md` | Comparatif v3 / v5.0 / v5.1 |
| `.ai/RELEASE_NOTES_V5.1.md` | Release notes complètes |

---

## 🧪 Tests et Validation

### Commandes de Test

```bash
# Tests rapides (unitaires hors intégration)
pytest -q --ignore=tests/integration

# Tests complets
pytest

# Couverture
pytest --cov=src --cov-report=html --cov-report=term --cov-fail-under=80

# Tests par module
pytest tests/integration/test_mv_player_matches.py -v
pytest tests/ui/test_data_loader_cache.py -v
```

### Scripts de Diagnostic

```bash
# Performance actuelle
python scripts/diagnose_performance.py --gamertag JGtm --runs 10

# Vérifier imports SQLite
grep -r "import sqlite3" src/ --exclude-dir=migration

# Vérifier imports Pandas
grep -r "import pandas" src/analysis/ src/data/services/
```

---

## 🎯 Ressources Externes

### Technologies

- [DuckDB Documentation](https://duckdb.org/docs/)
- [Polars User Guide](https://docs.pola.rs/)
- [Streamlit API Reference](https://docs.streamlit.io/)
- [Pydantic v2 Documentation](https://docs.pydantic.dev/)

### Halo Infinite

- [SPNKr GitHub](https://github.com/acurtis166/SPNKr) — Client API Python

---

## ✅ Checklist de Démarrage

Avant de commencer le projet :

- [ ] Environnement vérifié (`.venv` + packages)
- [ ] Documentation lue (PROJECT_UNIFIE_V5.1.md)
- [ ] Tests baseline verts
- [ ] Validation stakeholder obtenue
- [ ] Branche créée (`feature/v5.1-unified-project`)
- [ ] Sprint 0 complété (backups, baseline, secours)

---

## 📞 Contact et Support

### Questions Techniques

1. Consulter [thought_log.md](.ai/thought_log.md) pour historique décisions
2. Lire [V5.1_PURE_ARCHITECTURE.md](../docs/V5.1_PURE_ARCHITECTURE.md) pour guidelines
3. Vérifier [PROJECT_UNIFIE_V5.1.md](.ai/PROJECT_UNIFIE_V5.1.md) pour détails tâches

### Contribuer

1. Lire guidelines architecture pure
2. Suivre workflow de développement
3. Respecter checklist de conformité

---

**Dernière mise à jour** : 2026-02-16 — Création index projet unifié ✅

**Prochaine étape** : Démarrer Sprint 0 (Préparation)
