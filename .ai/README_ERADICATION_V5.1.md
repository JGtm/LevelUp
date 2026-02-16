# 🎯 Projet Éradication Architecture Legacy v5.1

> **Date** : 2026-02-16  
> **Statut** : 📋 PLANIFIÉ (prêt pour exécution)  
> **Objectif** : Éradication complète SQLite + Pandas → Architecture pure DuckDB + Polars

---

## 🚀 Démarrage Rapide

### Pour les Stakeholders (5 min)

Lire **[SYNTHESE_EXECUTIVE_V5.1.md](.ai/SYNTHESE_EXECUTIVE_V5.1.md)** — Résumé exécutif :
- Objectifs et bénéfices
- Planning 4 jours
- Risques et KPIs

### Pour l'Équipe Technique (30 min)

1. **Comprendre la vision** : [V5.1_PURE_ARCHITECTURE.md](docs/V5.1_PURE_ARCHITECTURE.md)
2. **Plan d'exécution** : [PLAN_ERADICATION_LEGACY_V5.md](.ai/PLAN_ERADICATION_LEGACY_V5.md)
3. **Navigation** : [INDEX_ERADICATION_LEGACY_V5.1.md](.ai/INDEX_ERADICATION_LEGACY_V5.1.md)

---

## 📊 En Chiffres

| Métrique | État Actuel (v5.0) | Cible (v5.1) | Gain |
|----------|-------------------|--------------|------|
| **Imports SQLite runtime** | 7 fichiers | **0** | -100% |
| **Imports Pandas métier** | 7 fichiers | **0** | -100% |
| **Taille player DB** | ~30 MB | **~4 MB** | **-87%** |
| **Temps chargement 100 matchs** | 350ms | **<200ms** | **-49%** |
| **Lignes de code** | ~45,000 | **~38,000** | **-16%** |
| **Couverture tests** | 75% | **≥80%** | +5% |

---

## 📚 Documents Produits

### 1. Plan Détaillé (60+ pages)

**[.ai/PLAN_ERADICATION_LEGACY_V5.md](.ai/PLAN_ERADICATION_LEGACY_V5.md)**

- Inventaire complet des vestiges legacy
- 6 phases d'exécution détaillées (28h total)
- Risques, mitigations, décisions clés
- Métriques de succès

### 2. Manifeste Architecture Pure

**[docs/V5.1_PURE_ARCHITECTURE.md](docs/V5.1_PURE_ARCHITECTURE.md)**

- Vision et principes v5.1
- Stack technique pure (100% DuckDB + Polars)
- Guidelines développement
- Anti-patterns à éviter
- Checklist de conformité

### 3. Synthèse Exécutive (10 pages)

**[.ai/SYNTHESE_EXECUTIVE_V5.1.md](.ai/SYNTHESE_EXECUTIVE_V5.1.md)**

- Résumé projet pour stakeholders
- Planning 4 jours
- KPIs et livrables
- Processus de validation

### 4. Index de Navigation

**[.ai/INDEX_ERADICATION_LEGACY_V5.1.md](.ai/INDEX_ERADICATION_LEGACY_V5.1.md)**

- Guide d'accès aux documents
- Checklist d'avancement
- Inventaire vestiges legacy
- Métriques en temps réel

---

## 🗂️ Inventaire Legacy

### SQLite (À Éradiquer)

**Runtime (7 fichiers)** :
- `src/data/query/engine.py` — Fallback metadata.db
- `src/data/infrastructure/database/duckdb_engine.py` — Fallback connection
- `src/utils/paths.py` — Références `.db`
- `src/ui/sync.py`, `src/ui/multiplayer.py`, `src/ai/rag.py` — Imports `sqlite3`

**Scripts migration (5)** :
- `scripts/migration/recover_from_sqlite.py` ✅ (déjà marqué LEGACY)
- `scripts/migration/migrate_player_to_duckdb.py`
- `scripts/migration/migrate_all_to_duckdb.py`
- `scripts/migration/migrate_metadata_to_duckdb.py`
- `scripts/migration/migrate_player_to_shared.py`

### Pandas (À Migrer vers Polars)

**Code métier (7 fichiers)** :
1. `src/analysis/performance_score.py` (4h)
2. `src/data/services/win_loss_service.py` (3h)
3. `src/ui/pages/objective_analysis.py` (2h)
4. `src/ui/pages/match_view_helpers.py` (1h)
5. `src/ui/pages/win_loss.py` (1h)
6. `src/ui/cache_filters.py` (0.5h)
7. `src/ui/components/duckdb_analytics.py` (0.5h)

**Total effort** : 12h

### Tables Redondantes (À Nettoyer)

**Par player DB** :
- `match_stats`, `match_participants`, `highlight_events`, `medals_earned`
- Views `v_*` (compatibilité)

**Script existant** : `cleanup_player_dbs_v5.py --all --remove-compat-views`

### Scripts Archive (~60)

**À trier** : `scripts/_archive/`
- R&D terminée → SUPPRIMER (~30)
- Migration/Recovery → GARDER + DOC (~10)
- Benchmarks → GARDER (~5)
- Utilitaires → ÉVALUER (~15)

---

## 🏗️ Phases d'Exécution (28h)

```
Phase 0 : Préparation (2h)
├── Backups production
├── Validation architecture v5
├── Snapshot état actuel
└── Branche de secours

Phase 1 : Éradication SQLite Runtime (4h)
├── Supprimer fallback engine.py
├── Supprimer fallback duckdb_engine.py
├── Nettoyer références .db
└── Supprimer imports sqlite3

Phase 2 : Archivage Scripts Migration (2h)
├── Bannières LEGACY (5 scripts)
├── scripts/migration/README.md
└── Décision refetch_film_roster.py

Phase 3 : Migration Pandas → Polars (12h)
├── performance_score.py (4h)
├── win_loss_service.py (3h)
├── objective_analysis.py (2h)
├── match_view_helpers.py (1h)
├── win_loss.py (1h)
└── cache_filters.py + duckdb_analytics.py (1h)

Phase 4 : Cleanup DBs Player (1h)
├── Dry-run
├── Backup + cleanup
└── Validation

Phase 5 : Audit Scripts Archive (3h)
├── Inventaire
├── Classification
└── scripts/_archive/README.md

Phase 6 : Validation & Documentation (4h)
├── Tests complets (≥80% couverture)
├── Audit sécurité
├── Mise à jour docs (5 fichiers)
└── Release notes v5.1
```

---

## ✅ Principes Directeurs

### Zero Tolerance

- ❌ Aucun fallback SQLite
- ❌ Aucune rétrocompatibilité
- ❌ Aucune dette technique

### Stack Pure

- ✅ 100% DuckDB (runtime)
- ✅ 100% Polars (code métier)
- ✅ Bridges Pandas (frontières uniquement)

### Approche

- 🔐 Sécurisée (backups systématiques)
- 📈 Incrémentale (6 phases)
- 🧪 Testée (≥80% couverture)
- 📚 Documentée (3 documents de référence)

---

## 🎯 Bénéfices Attendus

### Techniques

- **Performance** : -49% temps chargement
- **Stockage** : -87% taille DBs
- **Code** : -16% lignes de code
- **Tests** : +5% couverture

### Qualité

- 🧹 Code plus propre et maintenable
- 🛡️ Meilleure sécurité (zéro injection SQL via legacy)
- 📚 Documentation exhaustive
- 🚀 Stack moderne et homogène

---

## 📅 Planning

**Durée** : 28 heures (4 jours × 7h)

```
Jour 1 : Phase 0-2 (Préparation + SQLite + Scripts)
Jour 2 : Phase 3 (Migration Pandas partie 1)
Jour 3 : Phase 3-5 (Migration Pandas partie 2 + Cleanup + Archive)
Jour 4 : Phase 5-6 (Archive suite + Validation finale)
```

---

## 🚨 Risques Clés

| Risque | Impact | Mitigation |
|--------|--------|------------|
| **Perte de données** | 🔴 Critique | Backups systématiques avant chaque phase |
| **Régression fonctionnelle** | 🟡 Moyen | Tests exhaustifs par module migré |
| **Dépassement délai** | 🟢 Faible | Buffer 1 jour + priorisation |

---

## 🔗 Liens Rapides

### Documentation Projet v5.1

- 📋 [Plan détaillé](.ai/PLAN_ERADICATION_LEGACY_V5.md)
- 🎯 [Synthèse exécutive](.ai/SYNTHESE_EXECUTIVE_V5.1.md)
- 📖 [Manifeste architecture](docs/V5.1_PURE_ARCHITECTURE.md)
- 🗺️ [Index navigation](.ai/INDEX_ERADICATION_LEGACY_V5.1.md)

### Documentation Architecture v5

- [ARCHITECTURE_V5.md](docs/ARCHITECTURE_V5.md)
- [SHARED_MATCHES_SCHEMA.md](docs/SHARED_MATCHES_SCHEMA.md)
- [CLEANUP_V5.md](docs/CLEANUP_V5.md)
- [MIGRATION_V4_TO_V5.md](docs/MIGRATION_V4_TO_V5.md)

### Ressources Externes

- [DuckDB Documentation](https://duckdb.org/docs/)
- [Polars User Guide](https://docs.pola.rs/)
- [Streamlit API Reference](https://docs.streamlit.io/)

---

## 📞 Contact

### Questions Techniques

- Consulter [thought_log.md](.ai/thought_log.md)
- Lire [V5.1_PURE_ARCHITECTURE.md](docs/V5.1_PURE_ARCHITECTURE.md)

### Contribuer

1. Lire guidelines architecture pure
2. Suivre workflow de développement
3. Respecter checklist de conformité

---

## 🏁 Prêt à Démarrer ?

### Validation Humaine Requise

Avant d'exécuter, valider :
- ✅ Objectifs alignés avec vision projet
- ✅ Planning compatible avec ressources
- ✅ Risques acceptables

### Lancement Phase 0

```bash
# Créer branche de travail
git checkout -b feature/eradicate-legacy-v5.1

# Démarrer Phase 0 : Préparation
# Voir détails dans PLAN_ERADICATION_LEGACY_V5.md
```

---

**Projet planifié et documenté — Prêt pour exécution** ✅

**Prochaine étape** : Validation humaine + lancement Phase 0
