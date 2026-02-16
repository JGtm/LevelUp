# Synthèse Exécutive — Projet d'Éradication Legacy v5.1

> **Date** : 2026-02-16  
> **Projet** : LevelUp v5.0 → v5.1 (Pure Architecture)  
> **Chef de projet** : Agent IA Claude  
> **Durée estimée** : 28 heures (4 jours)

---

## 🎯 Résumé Exécutif

Le projet **Éradication Legacy v5.1** vise à transformer LevelUp en une application **pure** et **moderne**, éliminant définitivement toute trace des architectures obsolètes (SQLite, Pandas) au profit d'une stack 100% DuckDB + Polars.

### Objectifs Clés

| Objectif | État Actuel (v5.0) | Cible (v5.1) |
|----------|-------------------|--------------|
| **Imports SQLite runtime** | ~7 fichiers | **0** ✅ |
| **Imports Pandas métier** | ~7 fichiers | **0** ✅ |
| **Taille moyenne player DB** | ~30 MB | **~4 MB** ✅ |
| **Couverture tests** | 75% | **≥80%** ✅ |

### Principe Directeur

**Zero Tolerance** — Aucun fallback, aucune rétrocompatibilité.

---

## 📊 Bénéfices Attendus

### Techniques

- **-87% taille disque** : Player DBs de 30 MB → 4 MB
- **+49% performance** : Chargement matchs de 350ms → 180ms
- **-16% lignes de code** : ~45,000 → ~38,000 LOC
- **+5% couverture** : 75% → 80%+

### Qualité

- 🧹 **Code plus propre** : Moins de dépendances, moins de complexité
- 🛡️ **Meilleure sécurité** : Zéro injection SQL via fallback legacy
- 📚 **Documentation claire** : Architecture v5.1 documentée exhaustivement
- 🚀 **Maintenabilité** : Stack moderne et homogène

### Business

- ⚡ **Expérience utilisateur** : UI plus réactive (temps de chargement réduits)
- 💾 **Coûts réduits** : -87% stockage = économies infra potentielles
- 🔧 **Vélocité de développement** : Code simplifié = features plus rapides

---

## 🗂️ Inventaire des Vestiges Legacy

### SQLite (À Éradiquer)

| Type | Quantité | Action |
|------|---------|--------|
| **Code runtime avec fallback** | 4 fichiers | 🗑️ SUPPRIMER fallback |
| **Scripts migration** | 5 scripts | ✅ GARDER + bannière LEGACY |
| **Scripts utilitaires** | 2 scripts | 🔍 AUDITER + décider |

### Pandas (À Éradiquer)

| Type | Quantité | Action |
|------|---------|--------|
| **Code métier** | 7 fichiers | 🔄 MIGRER vers Polars |
| **Bridges compatibilité** | 3 fichiers | ✅ CONSERVER (frontières) |
| **Visualisations** | 2 fichiers | 🔄 REFACTORISER |

### Tables Redondantes (À Nettoyer)

| Type | Quantité | Action |
|------|---------|--------|
| **Tables obsolètes** | 4 par joueur | 🗑️ SUPPRIMER via script cleanup |
| **Views compatibilité** | 4 par joueur | 🗑️ SUPPRIMER |

### Scripts Archive (À Organiser)

| Type | Quantité | Action |
|------|---------|--------|
| **Scripts R&D** | ~60+ | 🔍 TRI + suppression/archivage |
| **Documentation manquante** | 2 README | ✍️ CRÉER |

---

## 🏗️ Plan d'Exécution (6 Phases)

### Phase 0 : Préparation (2h)

**Objectif** : Établir filet de sécurité

- [x] Backup complet de production
- [x] Validation architecture v5
- [x] Snapshot état actuel
- [x] Branche de secours

**Livrable** : Backups validés, baseline tests verte

---

### Phase 1 : Éradication SQLite Runtime (4h)

**Objectif** : Zéro SQLite en runtime

**Fichiers modifiés** :
- `src/data/query/engine.py`
- `src/data/infrastructure/database/duckdb_engine.py`
- `src/utils/paths.py`
- `src/ui/sync.py`, `src/ui/multiplayer.py`, `src/ai/rag.py`

**Actions** :
1. Supprimer fallback SQLite (remplacer `if/elif` par `if not exists: raise`)
2. Nettoyer références `metadata.db` → `metadata.duckdb`
3. Supprimer imports `sqlite3`
4. Tests de non-régression

**Livrable** : Zéro `import sqlite3` dans `src/` (hors migration)

---

### Phase 2 : Archivage Scripts Migration (2h)

**Objectif** : Marquer scripts legacy

**Actions** :
1. Ajouter bannières LEGACY (5 scripts migration)
2. Créer `scripts/migration/README.md`
3. Décision `refetch_film_roster.py` (supprimer ou garder avec bannière)

**Livrable** : Scripts legacy clairement identifiés

---

### Phase 3 : Migration Pandas → Polars (12h)

**Objectif** : Zéro Pandas en métier

**Fichiers à migrer** (par priorité) :
1. `src/analysis/performance_score.py` (4h)
2. `src/data/services/win_loss_service.py` (3h)
3. `src/ui/pages/objective_analysis.py` (2h)
4. `src/ui/pages/match_view_helpers.py` (1h)
5. `src/ui/pages/win_loss.py` (1h)
6. `src/ui/cache_filters.py` (0.5h)
7. `src/ui/components/duckdb_analytics.py` (0.5h)

**Pattern** : Remplacer `.groupby()` Pandas par `.group_by()` Polars, `.fillna()` par `.fill_null()`, etc.

**Livrable** : Zéro Pandas dans `src/analysis/`, `src/data/services/`, `src/ui/pages/` (hors frontières)

---

### Phase 4 : Cleanup DBs Player (1h)

**Objectif** : Récupérer espace disque

**Actions** :
1. Dry-run cleanup
2. Backup + cleanup avec `--remove-compat-views`
3. Validation UI + tests

**Livrable** : -87% taille player DBs

---

### Phase 5 : Audit Scripts Archive (3h)

**Objectif** : Trier `scripts/_archive/`

**Actions** :
1. Inventaire complet (~60 scripts)
2. Classification (supprimer/archiver/documenter)
3. Créer `scripts/_archive/README.md`

**Livrable** : Archive organisée et documentée

---

### Phase 6 : Validation & Documentation (4h)

**Objectif** : Clôture projet

**Actions** :
1. Suite de tests complète (≥80% couverture)
2. Audit de sécurité (zéro SQLite/Pandas métier)
3. Mise à jour docs (5 fichiers)
4. Release notes v5.1

**Livrable** : Documentation à jour, tests verts, architecture pure validée

---

## 📅 Planning

```
Jour 1 (7h)
├── Phase 0 : Préparation (2h)
├── Phase 1 : Éradication SQLite (4h)
└── Phase 2 : Archivage Scripts (1h)

Jour 2 (7h)
└── Phase 3 : Migration Pandas → Polars (7h)
    ├── performance_score.py (4h)
    └── win_loss_service.py (3h)

Jour 3 (7h)
├── Phase 3 : Migration Pandas (suite) (5h)
│   ├── objective_analysis.py (2h)
│   ├── match_view_helpers.py (1h)
│   ├── win_loss.py (1h)
│   └── cache_filters.py + duckdb_analytics.py (1h)
├── Phase 4 : Cleanup DBs (1h)
└── Phase 5 : Audit Archive (1h)

Jour 4 (7h)
├── Phase 5 : Audit Archive (suite) (2h)
├── Phase 6 : Validation & Docs (4h)
└── Buffer / Imprévus (1h)
```

**Total** : 28 heures réparties sur 4 jours (7h/jour)

---

## 🚨 Risques et Mitigations

| Risque | Impact | Probabilité | Mitigation |
|--------|--------|-------------|------------|
| **Perte de données** | 🔴 Critique | 🟡 Moyen | Backups systématiques avant chaque phase |
| **Régression fonctionnelle** | 🟡 Moyen | 🟡 Moyen | Tests exhaustifs par module migré |
| **Incompatibilité Plotly/Streamlit** | 🟡 Moyen | 🟢 Faible | Conserver bridges `_compat.py` |
| **Dépassement délai** | 🟢 Faible | 🟡 Moyen | Buffer 1 jour + priorisation |
| **Résistance équipe** | 🟢 Faible | 🟢 Faible | Documentation claire + formation |

**Plan de contingence** :
- Si perte de données : Restaurer depuis backups Phase 0
- Si régression majeure : Rollback branche `backup/pre-cleanup-v5.1`
- Si dépassement délai : Prioriser Phases 0-3 (critiques), reporter Phases 4-5 (optimisation)

---

## 📈 Indicateurs de Succès (KPIs)

### Techniques

| KPI | Cible | Mesure |
|-----|-------|--------|
| **Imports SQLite runtime** | 0 | `grep -r "import sqlite3" src/` |
| **Imports Pandas métier** | 0 | `grep -r "import pandas" src/analysis/` |
| **Taille player DB** | ≤ 5 MB | `ls -lh data/players/*/stats.duckdb` |
| **Couverture tests** | ≥ 80% | `pytest --cov=src --cov-report=term` |
| **Temps chargement 100 matchs** | < 200ms | Benchmark automatisé |

### Qualité

- ✅ Documentation à jour (5 fichiers modifiés)
- ✅ Zéro fallback architecture legacy
- ✅ Tests passent à 100%
- ✅ Zéro warning linter

---

## 📦 Livrables

### Documents Produits

| Document | Statut |
|----------|--------|
| `.ai/PLAN_ERADICATION_LEGACY_V5.md` | ✅ CRÉÉ |
| `docs/V5.1_PURE_ARCHITECTURE.md` | ✅ CRÉÉ |
| `.ai/SYNTHESE_EXECUTIVE_V5.1.md` | ✅ CRÉÉ |
| `.ai/RELEASE_NOTES_V5.1.md` | ⏳ À CRÉER (Phase 6) |
| `scripts/migration/README.md` | ⏳ À CRÉER (Phase 2) |
| `scripts/_archive/README.md` | ⏳ À CRÉER (Phase 5) |
| `.ai/reports/pandas_audit_v5.1.md` | ⏳ À CRÉER (Phase 3) |

### Code Modifié

| Type | Quantité Estimée |
|------|-----------------|
| Fichiers supprimés | ~20 (scripts R&D) |
| Fichiers modifiés | ~15 (migration Pandas + SQLite) |
| Tests ajoutés | ~30 (non-régression) |
| Lignes nettes supprimées | ~-7,000 |

### Données Nettoyées

| Type | Gain |
|------|------|
| Espace disque player DBs | -85% (~100 MB récupérés pour 4 joueurs) |
| Tables obsolètes | -16 tables (4 par joueur × 4 joueurs) |
| Views compatibilité | -16 views |

---

## 👥 Rôles et Responsabilités

| Rôle | Responsabilité |
|------|---------------|
| **Agent IA (Exécution)** | Implémentation technique, tests, documentation |
| **Humain (Validation)** | Revue code, validation tests, décision go/no-go |
| **CI/CD** | Validation automatique (linter, tests, couverture) |

---

## 🔄 Processus de Validation

### Par Phase

1. **Code review** : Agent auto-revue + humain spot-check
2. **Tests** : Suite complète verte (zéro régression)
3. **Documentation** : Mise à jour synchronisée
4. **Go/No-Go** : Validation humaine avant phase suivante

### Finale (Phase 6)

1. **Suite tests complète** : `pytest` (100% passant)
2. **Audit sécurité** : Grep SQLite/Pandas (0 résultat métier)
3. **Couverture** : ≥ 80% sur modules critiques
4. **Benchmark** : Validation performance (200ms chargement 100 matchs)
5. **Documentation** : 5 fichiers à jour + release notes
6. **Validation humaine** : Revue finale + approbation

---

## 📞 Communication

### Reporting Intermédiaire

**Fréquence** : Fin de chaque phase

**Format** :
```markdown
## Phase X : [NOM] — COMPLÉTÉE ✅

### Résumé
- Durée : Xh (estimé : Yh)
- Fichiers modifiés : N
- Tests : XX passed, 0 failed

### Livrables
- [x] Livrable 1
- [x] Livrable 2

### Next Steps
Phase suivante : [NOM] (Xh estimées)
```

### Reporting Final

**Format** : Release notes complètes + démo

**Contenu** :
1. Récapitulatif objectifs vs résultats
2. Métriques de succès validées
3. Known issues (si applicable)
4. Recommandations post-déploiement

---

## 🎓 Leçons Apprises (À Capturer)

### Pendant le Projet

- Documenter décisions architecturales dans `.ai/thought_log.md`
- Capturer patterns de migration Pandas→Polars réutilisables
- Noter les pièges techniques rencontrés

### En Fin de Projet

- Créer `.ai/archive/v5.1/RETRO.md` (rétrospective)
- Mettre à jour `.ai/project_map.md` avec nouvelles structures
- Archiver audits et rapports dans `.ai/archive/v5.1/`

---

## 🏁 Conclusion

Le projet **Éradication Legacy v5.1** transformera LevelUp en une application de référence :

- 🎯 **Architecture pure** : 100% DuckDB + Polars
- ⚡ **Performance optimale** : -49% temps de chargement
- 💾 **Efficacité disque** : -87% stockage
- 🧹 **Code propre** : -16% lignes de code
- 📚 **Documentation exhaustive** : 3 nouveaux documents de référence

**Approche** : Méthodique, incrémentale, sécurisée (backups à chaque phase).

**Durée** : 28 heures réparties sur 4 jours.

**Résultat attendu** : Une application moderne, performante et maintenable, alignée sur les meilleures pratiques 2026.

---

**Prêt à démarrer ?** 🚀

👉 Commencer par **Phase 0 (Préparation)** dans [PLAN_ERADICATION_LEGACY_V5.md](PLAN_ERADICATION_LEGACY_V5.md)
