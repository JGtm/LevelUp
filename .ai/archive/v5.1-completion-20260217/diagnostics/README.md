# Diagnostics Performance v5

Ce répertoire contient les analyses et diagnostics de performance pour LevelUp v5.

> **Ancien contenu archivé** : `.ai/archive/plans_treated_2026-02/diagnostics/`

## 📁 Fichiers

### Analyses de Performance v5 (2026-02-16)

- **`DIAGNOSTIC_LENTEURS_V5.md`** : Analyse approfondie des bottlenecks de performance
  - Identification des 5 bottlenecks principaux
  - Comparaison architecture v3 vs v5
  - Recommandations prioritaires

- **`PLAN_OPTIMISATION_V5.md`** : Roadmap d'optimisation en 3 sprints
  - Sprint 1 : Vue matérialisée (gain -70% parsing SQL)
  - Sprint 2 : Cache repository (gain -80% connexion)
  - Sprint 3 : Index + Schema cache (optimisations micro)
  - Objectif : v5 UI 2× plus rapide que v3

### Rapports de Benchmark

Les rapports JSON générés par `scripts/diagnose_performance.py` :

- `perf_{gamertag}_{timestamp}.json` : Résultats détaillés des benchmarks

## 🔍 Problème Identifié

**Symptôme** : La v5 est perçue comme plus lente que la v3 pour l'UI

**Causes** :
1. 🔴 Requêtes SQL complexes (170 lignes, 20+ COALESCE)
2. 🟡 ATTACH multiple à chaque connexion (+50-100ms)
3. 🟡 Cache Streamlit volatile (rechargements fréquents)
4. 🟢 Vérifications de schéma répétées (+10-20ms)

## 💡 Solution

### Architecture Hybride

**Conserver** :
- ✅ Shared matches (gain stockage + sync)
- ✅ Parallélisation API
- ✅ Calcul batch performance scores

**Optimiser** :
- ✅ Vue matérialisée pour requêtes simples
- ✅ Connexion persistante (cache repository)
- ✅ Index DuckDB optimisés

### Gains Attendus

| Métrique | v5 Actuelle | v5 Optimisée | Gain |
|----------|-------------|--------------|------|
| Temps connexion | 80ms | 15ms | **-81%** |
| load_matches(100) | 200ms | 60ms | **-70%** |
| Première page UI | 1500ms | 600ms | **-60%** |

## 🧪 Utilisation

### Lancer un Diagnostic

```bash
# Diagnostic complet avec 10 runs
python scripts/diagnose_performance.py --gamertag JGtm --runs 10

# Diagnostic rapide (3 runs)
python scripts/diagnose_performance.py --gamertag JGtm --runs 3

# Spécifier le fichier de sortie
python scripts/diagnose_performance.py --gamertag JGtm --output custom_report.json
```

### Interpréter les Résultats

Le script affiche :
1. **Statistiques DB** : Nombre de matchs, taille des bases
2. **Benchmarks** : Temps d'exécution moyens ± écart-type
3. **Bottlenecks** : Points critiques identifiés avec sévérité
4. **Recommandations** : Actions prioritaires

Codes de sortie :
- `0` : OK, pas de bottleneck critique
- `2` : Bottlenecks critiques détectés
- `1` : Erreur d'exécution

### Comparer avec v4.5

Les benchmarks v4.5 sont dans `.ai/reports/` :
- `benchmark_v4_5_post_s19.json` : Post Sprint 19
- `benchmark_v4_5_post_migration.json` : Post migration v5

Le script fait automatiquement la comparaison et signale les régressions.

## 🚀 Implémentation des Optimisations

Voir `PLAN_OPTIMISATION_V5.md` pour le plan détaillé.

### Checklist

**Sprint 1 : Vue Matérialisée** ⏳
- [ ] Créer migration `migration_v5_1_create_mv_player_matches()`
- [ ] Adapter `_get_match_source()` pour utiliser la vue
- [ ] Tests de validation
- [ ] Benchmark post-Sprint 1

**Sprint 2 : Cache Repository** ⏳
- [ ] Implémenter `get_cached_repository()` avec `@st.cache_resource`
- [ ] Adapter les 24 pages UI
- [ ] Invalidation après sync
- [ ] Benchmark post-Sprint 2

**Sprint 3 : Index + Schema Cache** ⏳
- [ ] Créer index `idx_mp_xuid_match`, `idx_mr_start_time`
- [ ] Cacher vérifications de schéma dans DuckDBRepository
- [ ] Benchmark post-Sprint 3

**Validation Finale** ⏳
- [ ] Tous les benchmarks cibles atteints
- [ ] Pas de régression fonctionnelle
- [ ] Documentation mise à jour

## 📊 Métriques de Succès

**Objectifs Minimums** :
- ✅ Temps connexion < 20ms
- ✅ load_matches(100) < 80ms
- ✅ Première page < 800ms

**Objectif Stretch** :
- 🎯 v5 UI 2× plus rapide que v3

## 📝 Notes

### Hypothèse : Pourquoi v3 Était Plus Rapide

La v3 utilisait :
- SQLite avec tables dénormalisées → requêtes simples
- Pas d'ATTACH → connexion instantanée
- Cache Pandas persistant (pickle) → réutilisation entre sessions

La v5 introduit :
- Requêtes complexes (COALESCE, jointures) → parsing lent
- 3 ATTACH par connexion → overhead +50-100ms
- Cache Streamlit volatile → rechargement fréquent

**MAIS** la v5 est beaucoup plus rapide pour la sync (-72% appels API).

### Trade-off Résolu

Avec les optimisations v5.1 :
- ✅ Gain sync v5 préservé (shared matches)
- ✅ Performance UI restaurée (vue + cache)
- ✅ Best of both worlds

## 🔗 Ressources

- Architecture v5 : `docs/ARCHITECTURE_V5.md`
- Optimisations sync : `docs/SYNC_OPTIMIZATIONS_V5.md`
- Script diagnostic : `scripts/diagnose_performance.py`
