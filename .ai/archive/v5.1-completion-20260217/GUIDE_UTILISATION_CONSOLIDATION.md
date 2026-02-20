# Guide d'Utilisation — Consolidation Master V5.1

> **But** : T'aider à naviguer entre les documents et choisir le bon workflow  
> **Durée lecture** : 5 minutes  
> **Mise à jour** : 2026-02-16

---

## 🚦 Démarrage Rapide

### Tu es dans quelle situation ?

#### 📍 Situation 1 : "Je commence le projet v5.1"

**Action** :
1. ✅ Lire `CONSOLIDATION_MASTER_V5.1.md` (10 min)
2. ✅ Choisir ton workflow (voir § Workflows ci-dessous)
3. ✅ Commencer par Étape 0 (Préparation)

**Documents nécessaires** :
- `CONSOLIDATION_MASTER_V5.1.md` — Plan global
- `PROJECT_UNIFIE_V5.1.md` — Détails sprints S0-S4
- `MIGRATION_V5_FINAL_GUIDE.md` — Détails phases 5-10

---

#### 📍 Situation 2 : "Je suis en cours d'exécution"

**Action** :
1. ✅ Identifier ton étape actuelle (0-6)
2. ✅ Consulter le bon document (voir § Par Étape ci-dessous)
3. ✅ Cocher items dans `MIGRATION_V5_FINAL_CHECKLIST.md`

**Documents de suivi** :
- `MIGRATION_V5_FINAL_CHECKLIST.md` — 163 items à cocher
- `SUIVI_AVANCEMENT_V5.1.md` — État sprints S0-S4

---

#### 📍 Situation 3 : "Je cherche un détail technique spécifique"

**Action** :
1. ✅ Utiliser `PHASES_5_10_INDEX.md` (navigation rapide)
2. ✅ Chercher dans le document approprié (voir § Par Besoin)

**Index disponibles** :
- `PHASES_5_10_INDEX.md` — Index navigation phases 5-10
- `INDEX_UNIFIED_PROJECT_V5.1.md` — Index PROJECT_UNIFIE

---

## 📚 Documents par Étape

| Étape | Documents Principaux | Documents Secondaires |
|-------|---------------------|----------------------|
| **Étape 0 : Préparation** | `PROJECT_UNIFIE_V5.1.md` § Sprint 0 | `GUIDE_DEMARRAGE_RAPIDE_V5.1.md` |
| **Étape 1 : Performance** | `PROJECT_UNIFIE_V5.1.md` § Sprint 1 | - |
| **Étape 2 : Shared DB** | `PHASES_5_10_ANALYSES.md` (Phase 5)<br>`PHASES_6_10_COMPLETE.md` (Phase 6) | `MIGRATION_V5_FINAL_GUIDE.md` |
| **Étape 3 : SQLite** | `PROJECT_UNIFIE_V5.1.md` § Sprint 2 | - |
| **Étape 4 : Pandas+Bugs** | `PROJECT_UNIFIE_V5.1.md` § Sprint 3<br>`PHASES_6_10_COMPLETE.md` (Ph7-8) | - |
| **Étape 5 : Cleanup** | `PROJECT_UNIFIE_V5.1.md` § Sprint 4<br>`PHASES_6_10_COMPLETE.md` (Phase 9) | `BUGFIX_V5_2026-02-15.md` |
| **Étape 6 : Docs** | `PROJECT_UNIFIE_V5.1.md` § Sprint 4<br>`PHASES_6_10_COMPLETE.md` (Phase 10) | - |

---

## 🔍 Documents par Besoin

### Besoin : "Comprendre le plan global"

**Lire** : `CONSOLIDATION_MASTER_V5.1.md`
- Vue d'ensemble des 6 étapes
- Ordre optimal d'exécution
- Durée totale (~86h)
- Métriques de succès

---

### Besoin : "Détails techniques Sprint 0-4"

**Lire** : `PROJECT_UNIFIE_V5.1.md`
- Sprint 0 : Backups, baseline, validation
- Sprint 1 : Vue mv_player_matches, cache, index
- Sprint 2 : Fallbacks SQLite, config, scripts
- Sprint 3 : 7 fichiers Pandas → Polars
- Sprint 4 : Cleanup DBs, tests, docs

---

### Besoin : "Détails techniques Phase 5"

**Lire** : `PHASES_5_10_ANALYSES.md`
- `teammates_service.py` — 4 fonctions, breaking change xuid
- `_match_queries.py` — Simplification 100→3 lignes
- `_roster_loader.py` — 15 fallbacks à supprimer
- Code AVANT/APRÈS complet
- Tests à créer

---

### Besoin : "Détails techniques Phases 6-10"

**Lire** : `PHASES_6_10_COMPLETE.md`
- Phase 6 : 8 méthodes duckdb_repo
- Phase 7 : 3 bugs critiques Polars/type/UX
- Phase 8 : 3 modules critiques (xuid, aliases, citations)
- Phase 9 : Cleanup brutal 8 tables + validation
- Phase 10 : 13 fichiers documentation

---

### Besoin : "Trouver une fonction spécifique"

**Utiliser** : `PHASES_5_10_INDEX.md`
- Index par fonction
- Index par fichier
- Index par problème
- Liens directs vers analyses

---

### Besoin : "Suivi progression quotidien"

**Utiliser** : 
- `MIGRATION_V5_FINAL_CHECKLIST.md` — 163 items à cocher
- `SUIVI_AVANCEMENT_V5.1.md` — Métriques sprints

---

## 🎯 Workflows Recommandés

### Workflow A : Consolidé (Recommandé)

**Suivre** : `CONSOLIDATION_MASTER_V5.1.md`

```
Étape 0 (2h)
  ↓
Étape 1 (8h) — Performance
  ↓
Étape 2 (5-6j) — Shared DB ⭐ ESSENTIEL
  ↓
Étape 3 (6h) — SQLite
  ↓
Étape 4 (14-15h) — Pandas + Bugs critiques
  ↓
Étape 5 (5h) — Cleanup brutal
  ↓
Étape 6 (2h) — Documentation
```

**Durée totale** : ~86h (~12 jours)

**Avantages** :
- ✅ Ordre optimal
- ✅ Bugs corrigés tôt
- ✅ Pas de refactoring inutile

**Inconvénients** :
- ⚠️ Jongler entre 2 documents de référence

---

### Workflow B : Séquentiel Simple

**Suivre** : `PROJECT_UNIFIE_V5.1.md` (S0-S4), puis `MIGRATION_V5_FINAL_GUIDE.md` (Ph5-10)

```
Sprints S0-S4 (32h)
  ↓
Phases 5-10 (~5j)
```

**Durée totale** : ~72h (~10 jours)

**Avantages** :
- ✅ Plus simple (1 doc à la fois)
- ✅ Peut s'arrêter après S4

**Inconvénients** :
- ❌ Ordre sous-optimal
- ❌ Risque refactoring

---

## 📖 Structure Documentation Complète

```
.ai/
├── 🚀 DÉMARRAGE
│   ├── START_HERE.md                          [Phases 5-10]
│   ├── GUIDE_DEMARRAGE_RAPIDE_V5.1.md         [PROJECT_UNIFIE]
│   └── GUIDE_UTILISATION_CONSOLIDATION.md     [Ce fichier]
│
├── 📋 PLANS PRINCIPAUX
│   ├── CONSOLIDATION_MASTER_V5.1.md           [⭐ PLAN UNIFIÉ]
│   ├── PROJECT_UNIFIE_V5.1.md                 [Sprints S0-S4]
│   └── MIGRATION_V5_FINAL_GUIDE.md            [Phases 0-10]
│
├── 🔍 ANALYSES DÉTAILLÉES
│   ├── PHASES_5_10_ANALYSES.md                [Phase 5]
│   ├── PHASES_6_10_COMPLETE.md                [Phases 6-10]
│   └── BUGFIX_V5_2026-02-15.md                [Référence technique]
│
├── 📑 INDEX & NAVIGATION
│   ├── PHASES_5_10_INDEX.md                   [Index phases 5-10]
│   ├── INDEX_UNIFIED_PROJECT_V5.1.md          [Index PROJECT]
│   ├── README_MIGRATION_V5.md                 [Index migration]
│   └── README_UNIFIED_PROJECT_V5.1.md         [Index PROJECT]
│
└── ✅ SUIVI PROGRESSION
    ├── MIGRATION_V5_FINAL_CHECKLIST.md        [163 items]
    └── SUIVI_AVANCEMENT_V5.1.md               [Métriques sprints]
```

---

## ❓ FAQ

### Q1 : "Quel document lire en premier ?"

**R** : `CONSOLIDATION_MASTER_V5.1.md` — Vue d'ensemble unifiée (15 min)

---

### Q2 : "Je veux juste faire les sprints S0-S4, sans phases 5-10 ?"

**R** : Possible, mais **NON RECOMMANDÉ**.  
Phases 5-6 (Architecture shared DB) sont **ESSENTIELLES** et absentes de S0-S4.  
Sans elles, l'architecture reste hybride (dual write).

---

### Q3 : "Phases 5-10 sont-elles incluses dans PROJECT_UNIFIE ?"

**R** : **NON**. PROJECT_UNIFIE (S0-S4) ne couvre PAS :
- ❌ Migration vers `shared.match_participants` (Phase 5-6)
- ❌ Bugs critiques Polars (Phase 7)
- ❌ Fallbacks shared metadata (Phase 8)
- ❌ Cleanup brutal tables legacy (Phase 9)

C'est pourquoi la consolidation est nécessaire.

---

### Q4 : "Puis-je faire Phases 5-10 AVANT Sprints S0-S4 ?"

**R** : **NON**. Ordre recommandé :
1. Sprint 0 (Préparation) — backups, baseline
2. Sprint 1 (Performance) — vue, cache, index
3. **Ensuite** Phases 5-10 peuvent commencer

Sprint 1 crée l'infrastructure performance nécessaire aux phases 5-10.

---

### Q5 : "Combien de temps total ?"

**R** :
- **Workflow A (Consolidé)** : ~86h (~12 jours ouvrés)
- **Workflow B (Séquentiel)** : ~72h (~10 jours ouvrés) mais risque refactoring

---

### Q6 : "Je suis perdu, quoi faire ?"

**R** : Protocole de dépannage :
1. ✅ Relire `CONSOLIDATION_MASTER_V5.1.md` § Plan Consolidé Final
2. ✅ Identifier ton étape actuelle (0-6)
3. ✅ Consulter le document approprié (voir § Documents par Étape)
4. ✅ Cocher items dans checklist

---

## 🎁 Conclusion

**Pour 99% des cas** : Suivre **Workflow A (Consolidé)**
- Lire `CONSOLIDATION_MASTER_V5.1.md`
- Suivre les 6 étapes dans l'ordre
- Consulter détails techniques selon besoin
- Cocher items dans `MIGRATION_V5_FINAL_CHECKLIST.md`

**Durée** : ~12 jours  
**Résultat** : Application v5.1 moderne, performante, maintenable ✅

---

*Guide généré le 2026-02-16 — Consolidation Master V5.1*
