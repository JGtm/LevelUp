# README — Projet Unifié v5.1

> **Projet** : LevelUp v5.0 → v5.1  
> **Objectif** : Architecture Pure + Performance Optimale  
> **Statut** : 📋 PLANIFIÉ — Prêt pour exécution

---

## 🎯 Qu'est-ce que le Projet Unifié v5.1 ?

Le **Projet Unifié v5.1** fusionne deux initiatives majeures en un seul programme cohérent :

### 1. Architecture Pure (Éradication Legacy)
- ✅ Supprimer **tout** SQLite runtime
- ✅ Migrer **tout** le code métier vers Polars
- ✅ Nettoyer tables redondantes (-87% stockage)

### 2. Performance Optimale (Résolution Lenteurs UI)
- ✅ Vue matérialisée pour requêtes simples
- ✅ Cache repository pour connexions persistantes
- ✅ Index DuckDB pour jointures rapides

**Résultat** : Application moderne, rapide et maintenable.

---

## 📊 Gains Attendus

| Catégorie | Métrique | v5.0 | v5.1 | Gain |
|-----------|----------|------|------|------|
| **Architecture** ||||
|| SQLite runtime | 7 fichiers | **0** | **-100%** |
|| Pandas métier | 7 fichiers | **0** | **-100%** |
|| Taille player DB | 30 MB | **4 MB** | **-87%** |
| **Performance** ||||
|| Temps connexion | 80ms | **15ms** | **-81%** |
|| Chargement 100 matchs | 200ms | **65ms** | **-68%** |
|| Première page UI | 1500ms | **700ms** | **-53%** |
| **Qualité** ||||
|| Lignes de code | 45k | **38k** | **-16%** |
|| Couverture tests | 75% | **80%** | **+5%** |

---

## 🚀 Démarrage Rapide

### 1. Lire la Documentation (15 min)

**Essentiel** :
- [GUIDE_DEMARRAGE_RAPIDE_V5.1.md](.ai/GUIDE_DEMARRAGE_RAPIDE_V5.1.md) — Démarrer en 15 min

**Complet** :
- [PROJECT_UNIFIE_V5.1.md](.ai/PROJECT_UNIFIE_V5.1.md) — Plan maître détaillé (45 min)

**Navigation** :
- [INDEX_UNIFIED_PROJECT_V5.1.md](.ai/INDEX_UNIFIED_PROJECT_V5.1.md) — Index complet

---

### 2. Vérifier l'Environnement (5 min)

```bash
# Activer .venv
.venv/Scripts/Activate.ps1  # Windows PowerShell
# ou
source .venv/Scripts/activate  # Git Bash

# Vérifier packages
python scripts/check_env.py

# Tests baseline
pytest -q --ignore=tests/integration
```

**Critères** :
- ✅ `.venv` actif (Python 3.12.10)
- ✅ DuckDB 1.4.4 + Polars 1.38+
- ✅ Tests verts

---

### 3. Créer Branche de Travail (1 min)

```bash
git checkout -b feature/v5.1-unified-project
```

---

### 4. Exécuter Sprint 0 — Préparation (2h)

**Commandes** :
```bash
# Backups
for gt in $(jq -r '.[].gamertag' db_profiles.json); do
    python scripts/backup_player.py --gamertag "$gt"
done

# Baseline
python scripts/diagnose_performance.py --gamertag JGtm --runs 10 --output .ai/reports/baseline_v5.0.json

# Tests
pytest -q --ignore=tests/integration

# Branche secours
git checkout -b backup/pre-v5.1-project
git push origin backup/pre-v5.1-project
git checkout feature/v5.1-unified-project
```

**Validation** : Go/No-Go humain requis

---

### 5. Suivre le Planning (32h total)

| Sprint | Durée | Focus |
|--------|-------|-------|
| Sprint 0 | 2h | Préparation & sécurisation |
| Sprint 1 | 8h | 🔴 Performance (vue, cache, index) |
| Sprint 2 | 6h | 🔴 Éradication SQLite |
| Sprint 3 | 12h | 🟡 Migration Pandas → Polars |
| Sprint 4 | 4h | 🟡 Cleanup & validation |

**Suivi** : [SUIVI_AVANCEMENT_V5.1.md](.ai/SUIVI_AVANCEMENT_V5.1.md)

---

## 📂 Structure du Projet

### Documents Clés

```
.ai/
├── PROJECT_UNIFIE_V5.1.md              ⭐ Plan maître (45 min)
├── GUIDE_DEMARRAGE_RAPIDE_V5.1.md      ⭐ Démarrage rapide (15 min)
├── INDEX_UNIFIED_PROJECT_V5.1.md       ⭐ Index navigation (5 min)
├── SUIVI_AVANCEMENT_V5.1.md            ⭐ Tableau de bord
│
├── PLAN_ERADICATION_LEGACY_V5.md       📋 Plan initial legacy
├── diagnostics/
│   ├── DIAGNOSTIC_LENTEURS_V5.md       🔍 Analyse performance
│   └── PLAN_OPTIMISATION_V5.md         🔍 Plan optimisation
│
└── reports/
    ├── baseline_v5.0.json              📊 Baseline
    └── sprint*_*.{json,md}             📊 Résultats sprints
```

---

## 🎯 5 Sprints en Détail

### Sprint 0 : Préparation (2h)
- Backups production
- Baseline performance
- Validation architecture
- Branches de secours

### Sprint 1 : Performance (8h) 🔴
- Vue `mv_player_matches` : -70% parsing SQL
- Cache repository : -80% temps connexion
- Index DuckDB : -30% temps jointure

### Sprint 2 : Éradication SQLite (6h) 🔴
- Supprimer fallbacks runtime
- Nettoyer références `.db`
- Marquer scripts migration LEGACY

### Sprint 3 : Migration Pandas (12h) 🟡
- 7 fichiers métier → Polars
- Tests de non-régression
- Bridges compatibilité conservés

### Sprint 4 : Cleanup (4h) 🟡
- Cleanup DBs player (-85% taille)
- Audit scripts archive
- Documentation finale

---

## ✅ Critères de Validation

### Par Sprint

Chaque sprint requiert :
- ✅ Tous les tests verts
- ✅ Métriques atteintes (≥90% objectifs)
- ✅ Aucune régression
- ✅ Go/No-Go humain

### Finale

Projet réussi si :
- ✅ Zéro SQLite runtime
- ✅ Zéro Pandas métier
- ✅ Performance UI 2× meilleure que v3
- ✅ Couverture ≥80%
- ✅ Documentation complète

---

## 📊 Suivi d'Avancement

### Tableau de Bord Simplifié

```
Sprint 0 : Préparation              [    ] 0/4 (0%)
Sprint 1 : Performance              [    ] 0/4 (0%)
Sprint 2 : Éradication SQLite       [    ] 0/6 (0%)
Sprint 3 : Migration Pandas         [    ] 0/8 (0%)
Sprint 4 : Cleanup & Validation     [    ] 0/4 (0%)

Progression globale : 0/26 (0%)
```

**Voir** : [SUIVI_AVANCEMENT_V5.1.md](.ai/SUIVI_AVANCEMENT_V5.1.md) pour détails

---

## 🔧 Commandes Utiles

### Tests

```bash
# Suite rapide
pytest -q --ignore=tests/integration

# Suite complète
pytest

# Couverture
pytest --cov=src --cov-report=term --cov-fail-under=80
```

### Diagnostic

```bash
# Performance
python scripts/diagnose_performance.py --gamertag JGtm --runs 10

# Vérifier SQLite
grep -r "import sqlite3" src/ --exclude-dir=migration

# Vérifier Pandas
grep -r "import pandas" src/analysis/ src/data/services/
```

### Backups

```bash
# Backup
python scripts/backup_player.py --gamertag MonGamertag

# Restore
python scripts/restore_player.py --gamertag MonGamertag --backup ./backups/xxx
```

---

## 🚨 En Cas de Problème

### Rollback Complet

```bash
# Retour branche secours
git checkout backup/pre-v5.1-project

# Restaurer données
python scripts/restore_player.py --gamertag MonGamertag --backup ./backups/v5.1_pre_project_*/
```

### Support

1. [PROJECT_UNIFIE_V5.1.md](.ai/PROJECT_UNIFIE_V5.1.md) — Détails tâches
2. [thought_log.md](.ai/thought_log.md) — Historique décisions
3. [V5.1_PURE_ARCHITECTURE.md](../docs/V5.1_PURE_ARCHITECTURE.md) — Guidelines

---

## 📚 Ressources

### Documentation Projet

- Plan maître : [PROJECT_UNIFIE_V5.1.md](.ai/PROJECT_UNIFIE_V5.1.md)
- Démarrage : [GUIDE_DEMARRAGE_RAPIDE_V5.1.md](.ai/GUIDE_DEMARRAGE_RAPIDE_V5.1.md)
- Index : [INDEX_UNIFIED_PROJECT_V5.1.md](.ai/INDEX_UNIFIED_PROJECT_V5.1.md)
- Suivi : [SUIVI_AVANCEMENT_V5.1.md](.ai/SUIVI_AVANCEMENT_V5.1.md)

### Documentation Architecture

- Manifeste v5.1 : [V5.1_PURE_ARCHITECTURE.md](../docs/V5.1_PURE_ARCHITECTURE.md)
- Architecture v5 : [ARCHITECTURE_V5.md](../docs/ARCHITECTURE_V5.md)
- Schéma DDL : [SHARED_MATCHES_SCHEMA.md](../docs/SHARED_MATCHES_SCHEMA.md)

### Plans Sources (Pré-Unification)

- Éradication legacy : [PLAN_ERADICATION_LEGACY_V5.md](.ai/PLAN_ERADICATION_LEGACY_V5.md)
- Diagnostic perf : [DIAGNOSTIC_LENTEURS_V5.md](.ai/diagnostics/DIAGNOSTIC_LENTEURS_V5.md)
- Plan optimisation : [PLAN_OPTIMISATION_V5.md](.ai/diagnostics/PLAN_OPTIMISATION_V5.md)

---

## 🎓 Leçons Apprises (Anticipées)

### Approche

✅ **Incrémentale** : Sprints courts et validés  
✅ **Performance d'abord** : Résoudre lenteurs UI en priorité  
✅ **Tests systématiques** : Zéro régression  
✅ **Benchmarks** : Mesurer avant/après

### Synergies Identifiées

**Performance + Éradication** :
- Vue matérialisée remplace COALESCE complexes
- Cache repository élimine fallbacks répétés
- Migration Polars améliore performance + élimine Pandas

**Win-Win** : Gains cumulatifs au lieu de trade-offs

---

## 🏁 Conclusion

Le **Projet Unifié v5.1** transforme LevelUp :

- 🎯 **Architecture moderne** : 100% DuckDB + Polars
- ⚡ **Performance optimale** : -53% temps chargement UI
- 💾 **Efficacité** : -87% stockage
- 🧪 **Qualité** : ≥80% couverture tests

**Durée** : 32 heures (4-5 jours)

**ROI** : Énorme — Résout lenteurs + élimine dette technique

---

## ✅ Checklist Avant de Démarrer

- [ ] Documentation lue (PROJECT_UNIFIE_V5.1.md)
- [ ] Environnement vérifié (`.venv` + packages)
- [ ] Tests baseline verts
- [ ] Validation stakeholder obtenue
- [ ] Branche créée
- [ ] Sprint 0 prêt à lancer

---

**Prêt à démarrer ?** 🚀

👉 **Prochaine étape** : [GUIDE_DEMARRAGE_RAPIDE_V5.1.md](.ai/GUIDE_DEMARRAGE_RAPIDE_V5.1.md)
