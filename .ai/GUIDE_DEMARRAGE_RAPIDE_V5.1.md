# Guide de Démarrage Rapide — Projet Unifié v5.1

> **Objectif** : Démarrer le projet en 15 minutes  
> **Public** : Développeurs et chefs de projet

---

## 🚀 En 3 Minutes

### Qu'est-ce que le Projet Unifié v5.1 ?

**2 initiatives fusionnées** :
1. ✅ Éradication architecture legacy (SQLite + Pandas)
2. ✅ Optimisation performance UI (-53% temps de chargement)

**Résultat** : Architecture pure + Performance optimale

---

## 📋 Prérequis (5 min)

### Vérifier l'Environnement

```bash
# 1. Activer l'environnement virtuel
.venv/Scripts/Activate.ps1  # Windows PowerShell
# ou
source .venv/Scripts/activate  # Git Bash

# 2. Vérifier packages critiques
python scripts/check_env.py

# 3. Vérifier tests baseline
pytest -q --ignore=tests/integration
```

**Critères de validation** :
- ✅ `.venv` actif (Python 3.12.10)
- ✅ DuckDB 1.4.4 installé
- ✅ Polars 1.38+ installé
- ✅ Tests verts

---

## 📖 Documentation à Lire (10 min)

### Essentiel

1. **[PROJECT_UNIFIE_V5.1.md](.ai/PROJECT_UNIFIE_V5.1.md)** — Plan maître (15 min)
   - Vue d'ensemble des 5 sprints
   - Métriques de succès
   - Tâches détaillées

### Optionnel (Selon Rôle)

**Développeur** :
- [V5.1_PURE_ARCHITECTURE.md](../docs/V5.1_PURE_ARCHITECTURE.md) — Guidelines techniques

**Chef de projet** :
- [SYNTHESE_EXECUTIVE_V5.1.md](.ai/SYNTHESE_EXECUTIVE_V5.1.md) — Résumé exécutif
- [INDEX_UNIFIED_PROJECT_V5.1.md](.ai/INDEX_UNIFIED_PROJECT_V5.1.md) — Navigation

---

## 🎯 Démarrage du Projet

### Étape 1 : Créer la Branche de Travail (1 min)

```bash
# Branche de travail
git checkout -b feature/v5.1-unified-project
```

### Étape 2 : Exécuter Sprint 0 — Préparation (2h)

**Voir** : [PROJECT_UNIFIE_V5.1.md — Sprint 0](.ai/PROJECT_UNIFIE_V5.1.md#sprint-0)

**Actions** :
1. Backups production
2. Baseline performance
3. Validation architecture v5
4. Branche de secours

**Commandes** :
```bash
# Backups
for gt in $(jq -r '.[].gamertag' db_profiles.json); do
    python scripts/backup_player.py --gamertag "$gt"
done

# Baseline
python scripts/diagnose_performance.py --gamertag JGtm --runs 10 --output .ai/reports/baseline_v5.0.json

# Validation
pytest -q --ignore=tests/integration

# Branche secours
git checkout -b backup/pre-v5.1-project
git push origin backup/pre-v5.1-project
git checkout feature/v5.1-unified-project
```

**Critères de validation** :
- ✅ Backups dans `backups/v5.1_pre_project_*/`
- ✅ Baseline : `.ai/reports/baseline_v5.0.json`
- ✅ Tests verts
- ✅ Branche secours poussée

**Go/No-Go** : Valider avec stakeholder avant Sprint 1

---

### Étape 3 : Sprint 1 — Performance (8h)

**Voir** : [PROJECT_UNIFIE_V5.1.md — Sprint 1](.ai/PROJECT_UNIFIE_V5.1.md#sprint-1)

**Objectif** : UI 2× plus rapide

**Tâches** :
1. Vue `mv_player_matches` (2h)
2. Cache repository (3h)
3. Index DuckDB (1h)
4. Tests & validation (2h)

**Commandes de validation** :
```bash
# Appliquer migrations
python scripts/apply_migrations.py

# Tests
pytest tests/integration/test_mv_player_matches.py -v
pytest tests/ui/test_data_loader_cache.py -v

# Benchmark
python scripts/diagnose_performance.py --gamertag JGtm --runs 10 --output .ai/reports/sprint1_final.json
```

**Métriques attendues** :
- ✅ Temps connexion : <20ms (vs 80ms)
- ✅ Première page : <800ms (vs 1500ms)

---

### Étape 4 : Sprints 2-4 (Selon Planning)

Suivre le plan détaillé dans [PROJECT_UNIFIE_V5.1.md](.ai/PROJECT_UNIFIE_V5.1.md) :

- **Sprint 2** : Éradication SQLite (6h)
- **Sprint 3** : Migration Pandas → Polars (12h)
- **Sprint 4** : Cleanup & Validation (4h)

---

## 📊 Suivi d'Avancement

### Tableau de Bord Simplifié

```
Sprint 0 : Préparation              [    ] 0/4
Sprint 1 : Performance              [    ] 0/4
Sprint 2 : Éradication SQLite       [    ] 0/6
Sprint 3 : Migration Pandas         [    ] 0/8
Sprint 4 : Cleanup & Validation     [    ] 0/4

Progression globale : 0/26 (0%)
```

**Mise à jour** : Marquer `[✓]` pour chaque tâche complétée

---

## 🔧 Commandes Utiles

### Diagnostic

```bash
# Performance actuelle
python scripts/diagnose_performance.py --gamertag JGtm --runs 10

# Vérifier imports SQLite
grep -r "import sqlite3" src/ --exclude-dir=migration

# Vérifier imports Pandas
grep -r "import pandas" src/analysis/ src/data/services/
```

### Tests

```bash
# Suite rapide (hors intégration)
pytest -q --ignore=tests/integration

# Suite complète
pytest

# Couverture
pytest --cov=src --cov-report=term --cov-fail-under=80
```

### Backups

```bash
# Backup un joueur
python scripts/backup_player.py --gamertag MonGamertag

# Restore
python scripts/restore_player.py --gamertag MonGamertag --backup ./backups/xxx
```

---

## 🚨 En Cas de Problème

### Rollback

```bash
# Retour à l'état pré-projet
git checkout backup/pre-v5.1-project

# Restaurer données
python scripts/restore_player.py --gamertag MonGamertag --backup ./backups/v5.1_pre_project_*/
```

### Support

1. Consulter [thought_log.md](.ai/thought_log.md) pour historique décisions
2. Lire [V5.1_PURE_ARCHITECTURE.md](../docs/V5.1_PURE_ARCHITECTURE.md) pour guidelines
3. Vérifier [PROJECT_UNIFIE_V5.1.md](.ai/PROJECT_UNIFIE_V5.1.md) pour détails tâches

---

## ✅ Checklist Avant de Commencer

- [ ] Environnement vérifié (`.venv` + packages)
- [ ] Tests baseline verts
- [ ] Documentation lue (PROJECT_UNIFIE_V5.1.md)
- [ ] Branche créée (`feature/v5.1-unified-project`)
- [ ] Validation stakeholder obtenue
- [ ] Sprint 0 complété

---

## 🎯 Prochaines Étapes

1. ✅ **Maintenant** : Exécuter Sprint 0 (Préparation)
2. ⏳ **Après validation** : Sprint 1 (Performance)
3. ⏳ **Puis** : Sprints 2-4 selon planning

---

**Durée totale estimée** : 32 heures (4-5 jours)

**Bonne chance !** 🚀
