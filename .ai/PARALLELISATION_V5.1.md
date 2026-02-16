# Parallélisation V5.1 — Guide Exécution avec 2 IA Simultanées

> **Date de création** : 2026-02-16  
> **Version** : v5.1  
> **Objectif** : Mobiliser 2 IA en parallèle pour accélérer le projet  
> **Gain estimé** : 26-35% de réduction du temps total

---

## 🎯 Objectif de ce Document

Ce document identifie les **sprints/étapes qui peuvent être exécutés EN PARALLÈLE** par 2 IA différentes, afin de réduire le délai global du projet V5.1.

**Plan réconcilié de référence** : `.ai/RECONCILIATION_FINALE_V5.1.md`

**Question utilisateur** :
> "Est-ce que tu peux marquer les sprints qui peuvent être fait indépendamment des autres ? Comme ça je peux mobiliser deux IA dessus en même temps"

---

## 📊 Vue d'Ensemble du Plan V5.1

**10 Étapes - Temps séquentiel : ~115h (~16 jours ouvrés)**

| Étape | Nom | Durée | Type |
|-------|-----|-------|------|
| 0 | Préparation | 2h | 🔴 BLOQUANT |
| 1 | Performance UI | 2-3j | 🔴 BLOQUANT |
| 2 | Performance données | 8h | 🟡 DÉPENDANT |
| 3 | Architecture shared | 5-6j | 🟡 DÉPENDANT |
| 4 | SQLite cleanup | 4h | ⚡ PARALLÈLE |
| 5 | Scripts bannières | 2h | 🟢 INDÉPENDANT |
| 6 | Pandas→Polars | 12h | 🟡 DÉPENDANT |
| 7 | Bugs critiques | 2-3h | 🟡 DÉPENDANT |
| 8 | Cleanup tables | 6h | 🟡 DÉPENDANT |
| 9 | Tests + docs | 3h | 🟡 DÉPENDANT |
| 10 | Release | 1h | 🟡 DÉPENDANT |

---

## 🔍 Analyse de Dépendances

### Graphe Complet

```
Étape 0 (Préparation) 🔴 BLOQUANT
    ↓
    ↓ [Toutes les étapes dépendent de la préparation]
    ↓
Étape 1 (Performance UI) 🔴 BLOQUANT - PRIORITÉ CRITIQUE
    ↓
    ↓ [Vue matérialisée + cache requis pour suite]
    ↓
Étape 2 (Performance données) 🟡 DÉPENDANT
    ↓
    ↓ [Index en place]
    ↓
┌─────────────────────────────────┐
│                                 │
│  Étape 3 (Architecture) 🟡     │ ⚡ PARALLÈLE avec Ét.4+5
│  (5-6 jours)                    │
│  - teammates_service.py         │
│  - _match_queries.py            │
│  - _roster_loader.py            │
│  - duckdb_repo.py               │
│                                 │
│  Étape 4 (SQLite) ⚡            │ ⚡ PARALLÈLE avec Ét.3
│  (4h)                           │
│  - query/engine.py              │
│  - database/duckdb_engine.py   │
│  - utils/paths.py               │
│  - 4 autres fichiers           │
│                                 │
│  Étape 5 (Bannières) 🟢        │ ⚡ PARALLÈLE avec tout
│  (2h)                           │
│  - scripts/migration/*.py       │
│                                 │
└─────────────────────────────────┘
    ↓
    ↓ [Architecture + SQLite prêts]
    ↓
Étape 6 (Pandas→Polars) 🟡
    ↓
    ↓ [Code métier migré]
    ↓
Étape 7 (Bugs critiques) 🟡
    ↓
    ↓ [Bugs corrigés]
    ↓
Étape 8 (Cleanup tables) 🟡
    ↓
    ↓ [Tables supprimées]
    ↓
Étape 9 (Tests + docs) 🟡
    ↓
Étape 10 (Release) 🟡
```

### Légende

- 🔴 **BLOQUANT** : Doit être complété avant de commencer d'autres étapes
- 🟡 **DÉPENDANT** : Requiert que certaines étapes précédentes soient terminées
- ⚡ **PARALLÈLE** : Peut être exécuté simultanément avec d'autres étapes
- 🟢 **INDÉPENDANT** : Aucune dépendance technique, peut être fait à tout moment

---

## 📋 Marquage Détaillé des Sprints

### Étape 0 : Préparation 🔴 BLOQUANT

**Statut** : DOIT être fait en premier par 1 seule IA

**Raison** : Établit baseline commune pour toutes les étapes suivantes

**Fichiers** :
- Backups production
- Tests baseline
- Snapshot métriques

**Risque de parallélisation** : ❌ IMPOSSIBLE (étape initiale unique)

---

### Étape 1 : Performance UI 🔴 BLOQUANT - PRIORITÉ

**Statut** : DOIT être fait en séquence par 1 IA

**Raison** : 
- Crée infrastructure (vue matérialisée, cache) requise pour toutes les étapes UI suivantes
- Impact utilisateur immédiat prioritaire
- Motivation équipe pour suite du projet

**Fichiers critiques** :
- Vue `mv_player_matches` (nouvelle infrastructure)
- `DuckDBRepository` (cache resource)
- `_get_match_source()` simplification

**Risque de parallélisation** : ❌ IMPOSSIBLE (infrastructure fondamentale)

---

### Étape 2 : Performance Données 🟡 DÉPENDANT

**Statut** : Dépend de Étape 1 (index en place)

**Raison** : Requiert que la vue matérialisée soit créée

**Fichiers** :
- Index DuckDB supplémentaires
- Optimisations requêtes

**Risque de parallélisation** : ⚠️ MOYEN (dépend infrastructure Étape 1)

---

### Étape 3 : Architecture Shared DB ⚡ PARALLÈLE

**Statut** : ⚡ PEUT être fait en parallèle avec Étapes 4+5

**Raison** :
- Fichiers totalement distincts des Étapes 4-5
- Domaine : Services + queries
- Aucun conflit avec SQLite cleanup ou bannières

**Fichiers concernés** :
- `src/data/services/teammates_service.py`
- `src/data/query/_match_queries.py`
- `src/data/loaders/_roster_loader.py`
- `src/data/repositories/duckdb_repo.py`

**Fichiers NON touchés par Ét.4-5** : ✅ Zéro chevauchement

**Risque de parallélisation** : 🟢 FAIBLE (domaines séparés)

---

### Étape 4 : SQLite Cleanup ⚡ PARALLÈLE

**Statut** : ⚡ PEUT être fait en parallèle avec Étape 3 ou 5

**Raison** :
- Fichiers totalement distincts de l'architecture shared
- Domaine : Éradication fallbacks SQLite
- Aucun chevauchement avec services/queries

**Fichiers concernés** :
- `src/data/query/engine.py`
- `src/data/infrastructure/database/duckdb_engine.py`
- `src/utils/paths.py`
- `src/ui/sync.py`
- `src/ui/multiplayer.py`
- `src/ai/rag.py`
- `scripts/refetch_film_roster.py`

**Fichiers NON touchés par Ét.3** : ✅ Zéro chevauchement avec teammates/queries/roster

**Risque de parallélisation** : 🟢 FAIBLE (domaines séparés)

**⚠️ Attention** : Ne pas paralléliser avec Étape 6 (Pandas) car risque conflit sur `src/data/`

---

### Étape 5 : Scripts Bannières 🟢 INDÉPENDANT

**Statut** : 🟢 TOTALEMENT INDÉPENDANT - Peut être fait n'importe quand

**Raison** :
- Ne touche que scripts de migration
- Aucun impact sur code applicatif
- Purement documentaire

**Fichiers concernés** :
- `scripts/migration/recover_from_sqlite.py`
- `scripts/migration/migrate_player_to_duckdb.py`
- `scripts/migration/migrate_all_to_duckdb.py`
- `scripts/migration/migrate_metadata_to_duckdb.py`
- `scripts/migration/migrate_player_to_shared.py`
- `scripts/migration/README.md` (nouveau)

**Fichiers NON touchés par autres étapes** : ✅ 100% isolé

**Risque de parallélisation** : 🟢 ZÉRO (totalement indépendant)

**🎯 MEILLEUR CANDIDAT** pour parallélisation avec n'importe quelle autre étape

---

### Étape 6 : Pandas→Polars 🟡 DÉPENDANT

**Statut** : Dépend de Étapes 1-5 (performance + architecture en place)

**Raison** : Requiert que l'architecture soit stabilisée

**Fichiers concernés** (7) :
1. `src/analysis/performance_score.py`
2. `src/data/services/win_loss_service.py`
3. `src/ui/pages/objective_analysis.py`
4. `src/ui/pages/match_view_helpers.py`
5. `src/ui/pages/win_loss.py`
6. `src/ui/cache_filters.py`
7. `src/ui/components/duckdb_analytics.py`

**Sous-parallélisation possible** :
- ⚡ Groupe A (critique) : fichiers 1-2 (services/analysis)
- ⚡ Groupe B (UI) : fichiers 3-7 (pages UI)

**Risque de parallélisation sous-groupes** : 🟢 FAIBLE (domaines UI vs services séparés)

---

### Étape 7 : Bugs Critiques 🟡 DÉPENDANT

**Statut** : Dépend de Étape 6 (Pandas migration complétée)

**Raison** : Corrige bugs Polars apparus pendant migration

**Fichiers concernés** :
- **Bugs Polars** : `src/ui/filters/filters.py`, `src/ui/filters/filters_render.py`, `src/ui/filters/checkbox_filter.py`
- **Metadata fallback** : `src/ui/aliases.py`, `src/utils/xuid.py`, `src/analysis/citations/engine.py`

**Sous-parallélisation possible** :
- ⚡ Sous-tâche A : Bugs Polars (3 fichiers filters)
- ⚡ Sous-tâche B : Metadata fallback (3 fichiers distincts)

**Risque de parallélisation sous-tâches** : 🟢 FAIBLE (domaines totalement séparés)

---

### Étape 8 : Cleanup Tables 🟡 DÉPENDANT

**Statut** : Dépend de Étape 7 (bugs corrigés)

**Raison** : Suppression brutale tables legacy, requiert code stable

**Fichiers concernés** :
- `scripts/cleanup_legacy_tables.py` (nouveau script)
- Bases de données players (8 tables par joueur)

**Risque de parallélisation** : ❌ IMPOSSIBLE (nécessite code finalisé)

---

### Étape 9 : Tests + Documentation 🟡 DÉPENDANT

**Statut** : Dépend de toutes les étapes précédentes

**Raison** : Validation complète du projet

**Sous-parallélisation possible** :
- ⚡ Sous-tâche A : Tests intégration
- ⚡ Sous-tâche B : Documentation (13 fichiers)

**Risque de parallélisation sous-tâches** : 🟢 FAIBLE (domaines séparés)

---

### Étape 10 : Release 🟡 DÉPENDANT

**Statut** : Dépend de Étape 9 (tests validés)

**Raison** : Livraison finale

**Risque de parallélisation** : ❌ IMPOSSIBLE (étape finale unique)

---

## 🚀 4 Stratégies de Parallélisation

### Stratégie 1 : Conservatrice (Gain 15h - 13%)

**Principe** : IA B fait uniquement les tâches totalement indépendantes pendant que IA A fait le chemin critique.

**Allocation** :

**IA A** (100h) — Chemin critique complet :
- Étape 0 : Préparation (2h)
- Étape 1 : Performance UI (2-3j)
- Étape 2 : Performance données (8h)
- Étape 3 : Architecture shared (5-6j)
- Étape 4 : SQLite cleanup (4h)
- Étape 6 : Pandas→Polars (12h)
- Étape 7 : Bugs critiques (2-3h)
- Étape 8 : Cleanup tables (6h)
- Étape 9 : Tests + docs (3h)
- Étape 10 : Release (1h)

**IA B** (15h) — Tâches support :
- Étape 5 : Scripts bannières (2h) [pendant Étape 3 IA A]
- Documentation préliminaire (13h) [pendant Étapes 1-8 IA A]

**Timeline** :
- Jours 1-14 : IA A fait tout le chemin critique
- Jour 3 : IA B fait bannières (pendant Architecture IA A)
- Jours 4-14 : IA B prépare docs (pendant travail IA A)
- Jour 15 : Merge + Release (ensemble)

**Avantages** :
- ✅ Risque minimal
- ✅ Coordination simple
- ✅ IA A autonome

**Inconvénients** :
- ❌ Gain de temps faible (13%)
- ❌ IA B sous-utilisée

**Durée totale** : ~100h (~14 jours) au lieu de 115h

**Gain** : 15h (13%)

---

### Stratégie 2 : Équilibrée ⭐ RECOMMANDÉE (Gain 30h - 26%)

**Principe** : Diviser le travail en 2 domaines bien séparés (Performance+Architecture vs Éradication+Cleanup)

**Allocation** :

**IA A** (50h) — Performance + Architecture :
- Étape 1 : Performance UI (2-3j = 20-24h)
- Étape 2 : Performance données (8h)
- Étape 3 : Architecture shared (5-6j = 40-48h)

**IA B** (40h) — Éradication Legacy + Cleanup :
- Étape 4 : SQLite cleanup (4h) [pendant Étape 3 IA A]
- Étape 5 : Scripts bannières (2h) [pendant Étape 3 IA A]
- Étape 6 : Pandas→Polars (12h) [après Étape 3 validée]
- Étape 7 : Bugs critiques (2-3h)
- Étape 8 : Cleanup tables (6h)

**Timeline parallèle** :
```
Jour 0 : Ensemble - Étape 0 (Préparation) 2h

Jours 1-3 : 
  IA A → Étape 1 (Perf UI) 20-24h
  IA B → Attente / Documentation préliminaire

Jour 4 :
  IA A → Étape 2 (Perf données) 8h
  IA B → Attente / Documentation

Jours 5-10 :
  IA A → Étape 3 (Architecture) 40-48h
  IA B → Étape 4 (SQLite) 4h + Étape 5 (Bannières) 2h
         = 6h puis attente

Jour 11 : CHECKPOINT - Merge Architecture + SQLite + Bannières

Jours 11-12 :
  IA A → Attente / Revue code
  IA B → Étape 6 (Pandas) 12h

Jour 13 :
  IA A → Attente / Revue code
  IA B → Étape 7 (Bugs) 2-3h + Étape 8 (Cleanup) 6h

Jour 14 : CHECKPOINT - Merge Final

Jour 15 : Ensemble - Étape 9 (Tests + docs) 3h + Étape 10 (Release) 1h
```

**Checkpoints de synchronisation** :
1. Après Étape 0 (Jour 0) : Validation baseline commune
2. Après Étape 1 (Jour 4) : IA A livre Performance, IA B peut commencer
3. Après Étapes 3+4+5 (Jour 11) : Merge Architecture + Éradication
4. Après Étapes 6+7+8 (Jour 14) : Merge Final
5. Étapes 9+10 (Jour 15) : Tests + Release ensemble

**Avantages** :
- ✅ Gain de temps significatif (26%)
- ✅ Domaines bien séparés
- ✅ Coordination raisonnable (5 checkpoints)
- ✅ IA A et IA B ont travail substantiel
- ✅ Risque de conflits faible

**Inconvénients** :
- ⚠️ Nécessite coordination aux checkpoints
- ⚠️ IA B doit attendre fin Étape 3 pour Pandas

**Durée totale** : ~85h (~12 jours) au lieu de 115h

**Gain** : 30h (26%)

---

### Stratégie 3 : Agressive (Gain 40h - 35%)

**Principe** : Maximiser parallélisation en acceptant plus de complexité de coordination

**Allocation** :

**IA A** (45h) — Performance + Pandas critique :
- Étape 1 : Performance UI (2-3j = 20-24h)
- Étape 2 : Performance données (8h)
- Étape 6 : Pandas→Polars CRITIQUE (fichiers 1-2 : 7h)

**IA B** (55h) — Architecture + Éradication + Cleanup :
- Étape 3 : Architecture shared (5-6j = 40-48h) [pendant Étapes 1-2 IA A]
- Étape 4 : SQLite cleanup (4h) [après Étape 3]
- Étape 5 : Scripts bannières (2h) [pendant Étape 3]
- Étape 6 : Pandas→Polars UI (fichiers 3-7 : 5h) [parallèle avec IA A]
- Étape 7 : Bugs critiques (2-3h)
- Étape 8 : Cleanup tables (6h)

**Timeline parallèle** :
```
Jour 0 : Ensemble - Étape 0 (Préparation) 2h

Jours 1-10 :
  IA A → Étape 1 (Perf UI) + Étape 2 (Perf données) = 28-32h
  IA B → Étape 3 (Architecture) = 40-48h
         + Étape 5 (Bannières) 2h

Jour 11 : CHECKPOINT - Merge Perf + Architecture

Jour 12 :
  IA A → Étape 6 (Pandas critique) fichiers 1-2 : 7h
  IA B → Étape 4 (SQLite) 4h + Étape 6 (Pandas UI) fichiers 3-7 : 5h

Jour 13 : CHECKPOINT - Merge Pandas

Jour 14 :
  IA A → Attente / Revue code
  IA B → Étape 7 (Bugs) 2-3h + Étape 8 (Cleanup) 6h

Jour 15 : CHECKPOINT - Merge Final

Jour 16 : Ensemble - Étape 9 (Tests + docs) 3h + Étape 10 (Release) 1h
```

**Avantages** :
- ✅ Gain de temps maximal (35%)
- ✅ Travail équilibré entre IA A et IA B

**Inconvénients** :
- ⚠️ Complexité coordination élevée
- ⚠️ Risque conflit moyen (Pandas parallèle)
- ⚠️ Nécessite communication active IA A ↔ IA B

**Durée totale** : ~75h (~10 jours) au lieu de 115h

**Gain** : 40h (35%)

---

### Stratégie 4 : Feature Teams (Gain 35h - 30%)

**Principe** : Diviser par domaine fonctionnel (Performance+UI vs Data+Architecture)

**Allocation** :

**IA A** (50h) — Stack Performance + UI complète :
- Étape 1 : Performance UI (2-3j = 20-24h)
- Étape 2 : Performance données (8h)
- Étape 6 : Pandas→Polars UI (fichiers 3-7 : 5h)
- Étape 7 : Bugs Polars UI (filters.*) (1h)

**IA B** (50h) — Stack Data + Architecture complète :
- Étape 3 : Architecture shared (5-6j = 40-48h)
- Étape 4 : SQLite cleanup (4h)
- Étape 5 : Scripts bannières (2h)
- Étape 6 : Pandas→Polars services (fichiers 1-2 : 7h)
- Étape 7 : Metadata fallback (1.5h)
- Étape 8 : Cleanup tables (6h)

**Timeline parallèle** :
```
Jour 0 : Ensemble - Étape 0 (Préparation) 2h

Jours 1-11 :
  IA A → Étape 1 (Perf UI) + Étape 2 (Perf données)
         = 28-32h puis attente
  IA B → Étape 3 (Architecture) + Étape 4 (SQLite) + Étape 5 (Bannières)
         = 40-48h + 4h + 2h = 46-54h

Jour 12 : CHECKPOINT - Merge Perf + Architecture + Éradication

Jours 13-14 :
  IA A → Étape 6 (Pandas UI) 5h + Étape 7 (Bugs Polars) 1h
  IA B → Étape 6 (Pandas services) 7h + Étape 7 (Metadata) 1.5h + Étape 8 (Cleanup) 6h

Jour 15 : CHECKPOINT - Merge Final

Jour 16 : Ensemble - Étape 9 (Tests + docs) 3h + Étape 10 (Release) 1h
```

**Avantages** :
- ✅ Domaines fonctionnels cohérents
- ✅ Chaque IA maîtrise son stack
- ✅ Gain de temps significatif (30%)

**Inconvénients** :
- ⚠️ IA A sous-utilisée jours 4-11
- ⚠️ Nécessite expertise domaine

**Durée totale** : ~80h (~11 jours) au lieu de 115h

**Gain** : 35h (30%)

---

## 🎯 Recommandation : Stratégie 2 (Équilibrée)

### Analyse Comparative

| Critère | Stratégie 1 | **Stratégie 2** ⭐ | Stratégie 3 | Stratégie 4 |
|---------|-------------|------------------|-------------|-------------|
| **Gain temps** | 15h (13%) | **30h (26%)** | 40h (35%) | 35h (30%) |
| **Complexité** | Faible | **Moyenne** | Élevée | Élevée |
| **Risque conflits** | Très faible | **Faible** | Moyen | Moyen |
| **Coordination** | Simple | **Raisonnable** | Complexe | Complexe |
| **Équilibre travail** | Déséquilibré | **Équilibré** | Équilibré | Moyen |
| **Autonomie IA** | IA A autonome | **Les deux autonomes** | Dépendances | Dépendances |

### Justification Stratégie 2

**✅ Meilleur ratio gain/complexité** :
- Gain de temps substantiel (26%) sans complexité excessive
- 5 checkpoints de synchronisation clairs et espacés
- Domaines bien séparés (Performance+Architecture vs Éradication+Cleanup)

**✅ Risque de conflits minimal** :
- IA A travaille sur Performance + Architecture (services/queries)
- IA B travaille sur Éradication (engine, utils, scripts) + Cleanup
- Très peu de fichiers partagés

**✅ Coordination raisonnable** :
- 5 checkpoints seulement
- Durée entre checkpoints : 3-7 jours (temps confortable)
- Merge Git simple (domaines séparés)

**✅ Travail substantiel pour les deux IA** :
- IA A : 50h de travail technique profond
- IA B : 40h de travail technique varié
- Aucune IA sous-utilisée

**✅ Autonomie maximale** :
- IA A peut travailler de manière autonome jours 1-10
- IA B peut travailler de manière autonome jours 5-13
- Communication requise uniquement aux checkpoints

---

## 📋 Plan d'Exécution Détaillé (Stratégie 2)

### Phase 1 : Préparation (Ensemble)

**Jour 0 - 2h** :
- [ ] IA A + IA B : Exécuter Étape 0 (Préparation)
- [ ] Créer branches : `ia-a-performance-architecture` et `ia-b-eradication-cleanup`
- [ ] Valider baseline tests commune
- [ ] GO/NO-GO

**Checkpoint 1** : Baseline validée, branches créées

---

### Phase 2 : Performance UI (IA A seule)

**Jours 1-3 - 20-24h** :
- [ ] IA A : Étape 1 (Performance UI)
  - [ ] Sprint Perf 1 : Vue matérialisée
  - [ ] Sprint Perf 2 : Cache repository
  - [ ] Sprint Perf 3 : Index + schema cache
- [ ] IA B : Standby / Documentation préliminaire

**Livrables IA A** :
- Vue `mv_player_matches` créée
- `DuckDBRepository` avec cache
- Index DuckDB

**Validation** :
- Tests performance verts
- Métriques atteintes (connexion <20ms, load_matches <80ms, UI <800ms)

**Checkpoint 2** : Performance UI validée, IA B peut commencer Pandas

---

### Phase 3 : Performance Données (IA A) + Préparation (IA B)

**Jour 4 - 8h** :
- [ ] IA A : Étape 2 (Performance données)
- [ ] IA B : Standby / Documentation

**Livrables IA A** :
- Index supplémentaires
- Optimisations requêtes

---

### Phase 4 : Architecture (IA A) || SQLite + Bannières (IA B)

**Jours 5-10 - 40-48h** :

**IA A** :
- [ ] Étape 3 : Architecture shared (5-6j)
  - [ ] Phase 5 : teammates_service.py
  - [ ] Phase 5 : _match_queries.py
  - [ ] Phase 5 : _roster_loader.py
  - [ ] Phase 6 : duckdb_repo.py (8 méthodes)

**IA B** (en parallèle) :
- [ ] Étape 4 : SQLite cleanup (4h)
  - [ ] 7 fichiers runtime
- [ ] Étape 5 : Scripts bannières (2h)
  - [ ] 5 scripts migration

**Livrables IA A** :
- Architecture shared complète
- Lectures depuis shared.match_participants

**Livrables IA B** :
- Zéro SQLite runtime
- Scripts marqués LEGACY

**Validation** :
- Tests intégration architecture verts
- `grep -r "import sqlite3" src/` → zéro résultat

**Checkpoint 3** : Architecture + Éradication SQLite validées, merge branches

---

### Phase 5 : Pandas Migration (IA B)

**Jours 11-12 - 12h** :
- [ ] IA A : Standby / Revue code IA B
- [ ] IA B : Étape 6 (Pandas→Polars)
  - [ ] 7 fichiers métier
  - [ ] Tests migration

**Livrables IA B** :
- Zéro Pandas métier
- `grep -r "import pandas" src/ | grep -v bridge` → zéro résultat

**Validation** :
- Tests outputs identiques avant/après
- Couverture ≥80%

---

### Phase 6 : Bugs + Cleanup (IA B)

**Jour 13 - 8-9h** :
- [ ] IA A : Standby / Revue code
- [ ] IA B : Étape 7 (Bugs critiques)
  - [ ] 3 bugs Polars
  - [ ] 3 fallbacks metadata
- [ ] IA B : Étape 8 (Cleanup tables)
  - [ ] Script cleanup
  - [ ] 8 tables par joueur supprimées

**Livrables IA B** :
- Bugs corrigés
- Tables legacy supprimées
- Taille DB -87%

**Validation** :
- UI fonctionne sans erreur
- Taille DBs ~4MB

**Checkpoint 4** : Migration complète validée, merge final

---

### Phase 7 : Tests + Documentation (Ensemble)

**Jour 14 - 3h** :
- [ ] IA A + IA B : Étape 9 (Tests + docs)
  - [ ] Tests intégration complète
  - [ ] 13 documents à jour
  - [ ] Validation couverture ≥80%

**Livrables** :
- Tests 100% verts
- Documentation synchronisée
- Métriques v5.1 validées

---

### Phase 8 : Release (Ensemble)

**Jour 15 - 1h** :
- [ ] IA A + IA B : Étape 10 (Release)
  - [ ] Tag Git v5.1.0-final
  - [ ] Release notes
  - [ ] Merge vers main

**Checkpoint 5** : Projet terminé ! 🎉

---

## 🔗 Points de Synchronisation

### Checkpoint 1 : Après Étape 0 (Jour 0)

**Validation** :
- [ ] Baseline tests verte (≥75% couverture)
- [ ] Backups validés (2+ joueurs)
- [ ] Branches créées (`ia-a-*`, `ia-b-*`)
- [ ] Snapshot métriques baseline

**Communication** :
- IA A annonce démarrage Étape 1
- IA B confirme réception, standby

**Décision** : GO/NO-GO projet

---

### Checkpoint 2 : Après Étape 1 (Jour 3-4)

**Validation** :
- [ ] Vue `mv_player_matches` créée et testée
- [ ] Cache repository fonctionne
- [ ] Métriques performance atteintes :
  - [ ] Connexion <20ms
  - [ ] load_matches(100) <80ms
  - [ ] Première page <800ms

**Communication** :
- IA A annonce fin Étape 1, push branche
- IA B pull branche IA A, valide performance
- IA B commence Étape 4-5 (parallèle Étape 3)

**Décision** : GO/NO-GO Phase 4 (Architecture || Éradication)

---

### Checkpoint 3 : Après Étapes 3+4+5 (Jour 10-11)

**Validation** :
- [ ] Architecture shared complète (IA A)
  - [ ] 4 fichiers migrés (teammates, queries, roster, repo)
  - [ ] Tests intégration verts
- [ ] SQLite éradiqué (IA B)
  - [ ] `grep -r "import sqlite3" src/` → zéro
  - [ ] 7 fichiers nettoyés
- [ ] Scripts bannières (IA B)
  - [ ] 5 scripts marqués LEGACY

**Communication** :
- IA A annonce fin Architecture, push branche
- IA B annonce fin SQLite + Bannières, push branche
- Merge branches (`git merge ia-a-* ia-b-*`)
- Résolution conflits (si nécessaire)

**Décision** : GO/NO-GO Phase 5 (Pandas migration)

---

### Checkpoint 4 : Après Étapes 6+7+8 (Jour 13-14)

**Validation** :
- [ ] Pandas→Polars complet (IA B)
  - [ ] 7 fichiers migrés
  - [ ] `grep -r "import pandas" src/ | grep -v bridge` → zéro
- [ ] Bugs corrigés (IA B)
  - [ ] 3 bugs Polars
  - [ ] 3 fallbacks metadata
- [ ] Cleanup tables (IA B)
  - [ ] 8 tables supprimées/joueur
  - [ ] Taille DB ~4MB (-87%)

**Communication** :
- IA B annonce fin migration complète, push branche
- IA A pull branche IA B, valide
- Merge final (`git merge ia-b-*`)

**Décision** : GO/NO-GO Phase 7 (Tests + Release)

---

### Checkpoint 5 : Après Étapes 9+10 (Jour 15)

**Validation** :
- [ ] Tests 100% verts
- [ ] Couverture ≥80%
- [ ] Toutes métriques v5.1 atteintes
- [ ] Documentation complète (13 docs)
- [ ] Tag Git v5.1.0-final créé
- [ ] Release notes publiées

**Communication** :
- IA A + IA B : Annonce fin projet
- Retrospective (10min)

**Décision** : Projet TERMINÉ ! 🎉

---

## ❓ FAQ Parallélisation

### Q1 : Que faire si IA A et IA B modifient le même fichier ?

**R** : Les domaines ont été choisis pour minimiser ce risque. Si conflit :
1. Identifier le conflit lors du merge
2. Prioriser les modifications de l'IA responsable du domaine principal
3. L'autre IA adapte son code

**Exemple** : Si conflit sur `duckdb_repo.py` :
- IA A (Architecture) a priorité
- IA B adapte son code SQLite cleanup

### Q2 : Comment IA A et IA B communiquent ?

**R** : Via les 5 checkpoints de synchronisation :
- Messages texte simples : "Étape X terminée, branche poussée"
- Pull requests avec description claire
- Revue de code mutuelle aux checkpoints

### Q3 : Que fait IA B pendant que IA A fait Performance UI (jours 1-4) ?

**R** : Optionnel :
- Documentation préliminaire
- Lecture/compréhension code à modifier
- Préparation environnement
- Standby (économie ressources)

### Q4 : Peut-on changer de stratégie en cours de route ?

**R** : Oui, aux checkpoints. Exemple :
- Si Étape 1 prend plus de temps : retarder Phase 4
- Si IA B termine tôt : peut aider sur Pandas migration

### Q5 : Quelle stratégie si une seule IA disponible à temps plein ?

**R** : Suivre l'ordre séquentiel du plan réconcilié :
- `.ai/RECONCILIATION_FINALE_V5.1.md`
- Durée : 115h (~16 jours)
- Pas de parallélisation

### Q6 : Comment gérer les dépendances implicites ?

**R** : Les checkpoints forcent la validation :
- Checkpoint 2 : IA B ne peut pas commencer Pandas sans Perf UI validée
- Checkpoint 3 : Merge Architecture + SQLite vérifie compatibilité
- Tests intégration à chaque checkpoint

---

## 📚 Références

**Documents à lire** :
1. `.ai/RECONCILIATION_FINALE_V5.1.md` — Plan maître réconcilié
2. `.ai/PROCESSUS_QUALITE_V5.1.md` — Framework qualité (checklists, DoD, revue)
3. `.ai/INDEX_FINAL_V5.1.md` — Navigation documentation

**Pour chaque étape** :
- Étape 1 : `.ai/diagnostics/PLAN_OPTIMISATION_V5.md`
- Étapes 3 : `.ai/PHASES_5_10_ANALYSES.md`, `.ai/PHASES_6_10_COMPLETE.md`
- Étape 4 : `.ai/PLAN_ERADICATION_LEGACY_V5.md` § Phase 1
- Étape 5 : `.ai/PLAN_ERADICATION_LEGACY_V5.md` § Phase 2
- Étape 6 : `.ai/PLAN_ERADICATION_LEGACY_V5.md` § Phase 3
- Étapes 7-8 : `.ai/PHASES_6_10_COMPLETE.md` § Phases 7-9

---

## ✅ Résumé Exécutif

**Question** : "Est-ce que tu peux marquer les sprints qui peuvent être fait indépendamment des autres ?"

**Réponse** :

**Sprints INDÉPENDANTS (marqués dans le plan)** :
- ⚡ Étape 5 : Scripts bannières (🟢 INDÉPENDANT - peut être fait n'importe quand)
- ⚡ Étape 4 : SQLite cleanup (peut être fait en parallèle avec Étape 3)
- ⚡ Sous-tâches Étape 7 : Bugs Polars vs Metadata fallback (fichiers distincts)

**Stratégie recommandée** : **Stratégie 2 (Équilibrée)**
- IA A : Performance + Architecture (50h)
- IA B : Éradication + Cleanup (40h)
- Gain temps : 30h (26%)
- Durée : 115h → 85h (~16j → 12j)

**Checkpoints** : 5 points de synchronisation clairs

**Avantage** : Réduction délai projet de 26% avec coordination raisonnable

---

**Prochaine action** : Choisir une stratégie et démarrer Étape 0 (Préparation) ! 🚀
