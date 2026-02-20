# Diagnostic des Lenteurs v5 — Analyse Approfondie

> **Date** : 2026-02-16  
> **Contexte** : L'utilisateur rapporte que la v5 est plus lente que la v3, alors que la v5 devrait être plus rapide

---

## 🔍 Problématique

Malgré les optimisations de la v5 :
- ✅ Mutualisation des données (shared_matches.duckdb)
- ✅ Réduction des appels API (-72%)
- ✅ Parallélisation des séquences
- ✅ Calcul batch des performance scores

**→ La v5 est perçue comme PLUS LENTE que la v3**

---

## 🎯 Bottlenecks Identifiés

### 1. 🔴 **CRITIQUE : Complexité de `_get_match_source()`**

**Localisation** : `src/data/repositories/_match_queries.py:70-240`

**Problème** :
- Requête SQL de **~170 lignes** construite dynamiquement
- **20+ COALESCE** pour gérer la compatibilité v4/v5
- **10+ CASE WHEN** pour les calculs conditionnels
- **Vérifications de colonnes répétées** à chaque connexion (`_has_column`, `_has_shared_mp_column`)

**Impact** :
- Temps de parsing SQL élevé à chaque requête
- Overhead de vérification des colonnes (4-6 requêtes `information_schema` par connexion)
- Requête complexe difficile à optimiser par le query planner DuckDB

**Exemple de complexité** :
```python
kda_expr = (
    "COALESCE(ms.kda, "
    "CASE WHEN p.deaths > 0 "
    "THEN (CAST(p.kills AS FLOAT) + CAST(p.assists AS FLOAT) / 3.0) "
    "/ CAST(p.deaths AS FLOAT) "
    "ELSE CAST(p.kills AS FLOAT) + CAST(p.assists AS FLOAT) / 3.0 END)"
)
```

**Recommandation** :
1. **Court terme** : Cacher le résultat de `_get_match_source()` par session
2. **Moyen terme** : Créer une **vue matérialisée** `mv_player_matches` qui pré-calcule les COALESCE
3. **Long terme** : Enrichir `match_participants` avec toutes les colonnes lors de la sync (éliminer les COALESCE)

---

### 2. 🟡 **IMPORTANT : ATTACH Multiple à Chaque Connexion**

**Localisation** : `src/data/repositories/duckdb_repo.py:149-203`

**Problème** :
- Chaque `DuckDBRepository` fait **3 ATTACH** :
  1. Base joueur (`stats.duckdb`)
  2. Base metadata (`metadata.duckdb`) — READ_ONLY
  3. Base shared (`shared_matches.duckdb`) — READ_ONLY

**Impact** :
- Temps de connexion : **50-100ms** (mesuré dans benchmarks v4.5)
- Si connexion non réutilisée → overhead multiplicatif

**Observation** :
```python
# Connexion créée à chaque page Streamlit sans cache
repo = RepositoryFactory.create(gamertag=gt, xuid=xuid)
```

**Recommandation** :
1. **Créer un pool de connexions DuckDB** ou réutiliser la même connexion par session Streamlit
2. **Cacher l'instance `DuckDBRepository`** dans `st.session_state`
3. **Utiliser `@st.cache_resource`** pour le repository (non sérialisable → cache_resource)

---

### 3. 🟡 **IMPORTANT : Cache Streamlit Trop Granulaire**

**Localisation** : `src/ui/cache_loaders.py:748-833`

**Problème** :
- `load_df_optimized()` utilise `db_key=(mtime_ns, size)` + `cache_buster`
- **Invalidation trop fréquente** si la DB est modifiée (sync en arrière-plan)
- Pas de cache au niveau repository → chaque page re-exécute les requêtes SQL

**Impact** :
- Rechargement complet des matchs à chaque changement de page
- Pas de réutilisation des résultats entre pages
- Temps de chargement : **100-500ms** par page

**Recommandation** :
1. **Ajouter un cache au niveau repository** pour les requêtes SQL identiques
2. **Augmenter le TTL des caches** pour les données peu changeantes (metadata, profils)
3. **Implémenter un cache L2** (fichier Parquet local) pour les agrégats coûteux

---

### 4. 🟢 **MINEUR : Vérifications de Colonnes Répétées**

**Localisation** : `src/data/repositories/_match_queries.py:115-176`

**Problème** :
```python
has_is_ranked = self._has_column(conn, "match_stats", "is_ranked")
has_is_firefight = self._has_column(conn, "match_stats", "is_firefight")
has_p_avg_life = self._has_shared_mp_column(conn, "avg_life_seconds")
has_p_max_spree = self._has_shared_mp_column(conn, "max_killing_spree")
has_p_hs_kills = self._has_shared_mp_column(conn, "headshot_kills")
```

**Impact** :
- **5 requêtes `information_schema`** à chaque construction de `_get_match_source()`
- Temps estimé : **10-20ms** d'overhead

**Recommandation** :
1. **Cacher les résultats** des vérifications de colonnes au niveau de la classe
2. **Utiliser une table `schema_version`** pour détecter les migrations
3. **Assumer un schéma stable** après migration v5 complète

---

### 5. 🟢 **MINEUR : Pas d'Index sur `match_participants(xuid, match_id)`**

**Localisation** : Base shared (`shared_matches.duckdb`)

**Problème** :
- La jointure principale utilise :
  ```sql
  JOIN shared.match_participants p
    ON r.match_id = p.match_id AND p.xuid = ?
  ```
- Pas d'index explicite sur `(xuid, match_id)` ou `(match_id, xuid)`

**Impact** :
- DuckDB peut créer des index automatiques, mais pas optimal pour tous les cas
- Scan potentiel de `match_participants` si l'index auto n'est pas utilisé

**Recommandation** :
1. **Créer un index composite** : `CREATE INDEX idx_mp_xuid_match ON match_participants(xuid, match_id)`
2. **Analyser le query plan** avec `EXPLAIN` pour confirmer l'utilisation de l'index

---

## 📊 Comparaison v3 vs v5

### Architecture

| Aspect | v3 | v5 |
|--------|----|----|
| **Stockage** | SQLite (1 DB par joueur) | DuckDB + shared_matches |
| **Requêtes** | SELECT simple sur table locale | Sous-requête complexe + 3 ATTACH |
| **Cache** | Pandas + pickle | Streamlit cache + invalidation fréquente |
| **Connexion** | 1 DB, immédiate | 3 ATTACH, 50-100ms |
| **Complexité SQL** | Faible (table dénormalisée) | Élevée (COALESCE, CASE, jointures) |

### Hypothèse : v3 Plus Rapide Pour l'UI

**Pourquoi v3 semblait plus rapide** :
1. **Requêtes SQL simples** : `SELECT * FROM match_stats` (table dénormalisée)
2. **Pas d'ATTACH** : connexion instantanée
3. **Pas de COALESCE** : colonnes directement disponibles
4. **Cache Pandas persistant** : fichiers pickle réutilisés entre sessions

**Pourquoi v5 est plus lente pour l'UI** :
1. **Requêtes complexes** : 170 lignes de SQL + 20 COALESCE
2. **3 ATTACH** par connexion : +50-100ms
3. **Vérifications de colonnes** : +10-20ms
4. **Cache Streamlit volatile** : rechargement fréquent

**MAIS v5 est plus rapide pour la sync** :
- ✅ -72% d'appels API (shared matches)
- ✅ Sync 4 joueurs : 45min → 12min (-73%)

---

## 💡 Recommandations Prioritaires

### 🔴 Priorité 1 : Simplifier `_get_match_source()`

**Action** : Créer une vue matérialisée `mv_player_matches` :

```sql
CREATE OR REPLACE VIEW mv_player_matches AS
SELECT
    r.match_id,
    r.start_time,
    r.map_id,
    r.map_name,
    -- ... toutes les colonnes pré-calculées
    CASE WHEN p.deaths > 0 
    THEN (p.kills::FLOAT + p.assists::FLOAT / 3.0) / p.deaths::FLOAT
    ELSE p.kills::FLOAT + p.assists::FLOAT / 3.0 END AS kda,
    -- ...
FROM shared.match_registry r
JOIN shared.match_participants p ON r.match_id = p.match_id;
```

**Puis dans le repository** :
```python
def _get_match_source(self, conn):
    if self.has_shared:
        return "mv_player_matches WHERE xuid = ?", [self._xuid]
    else:
        return "match_stats", []
```

**Gain estimé** : **-70% temps de parsing SQL**

---

### 🟡 Priorité 2 : Pool de Connexions + Cache Repository

**Action** : Ajouter un cache au niveau Streamlit :

```python
# src/app/data_loader.py

@st.cache_resource
def get_cached_repository(gamertag: str, xuid: str) -> DuckDBRepository:
    """Retourne un repository mis en cache (connexion persistante)."""
    return RepositoryFactory.create(gamertag=gamertag, xuid=xuid)
```

**Utilisation** :
```python
# Dans les pages
repo = get_cached_repository(gt, xuid)
matches = repo.load_matches(limit=100)
```

**Gain estimé** : **-80% temps de connexion**

---

### 🟡 Priorité 3 : Index DuckDB

**Action** :
```python
# Dans migrations.py
def create_performance_indexes(conn: duckdb.DuckDBPyConnection):
    """Crée les index pour optimiser les requêtes v5."""
    conn.execute("""
        CREATE INDEX IF NOT EXISTS idx_mp_xuid_match 
        ON match_participants(xuid, match_id)
    """)
    conn.execute("""
        CREATE INDEX IF NOT EXISTS idx_mp_match_xuid 
        ON match_participants(match_id, xuid)
    """)
```

**Gain estimé** : **-30% temps de jointure**

---

### 🟢 Priorité 4 : Cacher les Vérifications de Schéma

**Action** :
```python
class DuckDBRepository:
    def __init__(self, ...):
        # ...
        self._schema_cache = {}  # Cache des vérifications de colonnes
    
    def _has_column(self, conn, table_name, column_name):
        cache_key = f"{table_name}.{column_name}"
        if cache_key not in self._schema_cache:
            # Vérification réelle
            self._schema_cache[cache_key] = self._check_column(conn, table_name, column_name)
        return self._schema_cache[cache_key]
```

**Gain estimé** : **-10ms par requête**

---

## 🧪 Plan de Validation

### 1. Benchmark Actuel (Baseline)

```bash
python scripts/diagnose_performance.py --gamertag JGtm --runs 10
```

Métriques à capturer :
- Temps de connexion (ATTACH)
- Temps `_get_match_source()`
- Temps `load_matches(limit=100)`
- Temps total première page UI

---

### 2. Implémentation des Optimisations

**Sprint 1 : Vue matérialisée**
- [ ] Créer `mv_player_matches` dans migration
- [ ] Adapter `_get_match_source()` pour utiliser la vue
- [ ] Benchmark : mesurer l'amélioration

**Sprint 2 : Cache repository**
- [ ] Implémenter `@st.cache_resource` pour le repository
- [ ] Benchmark : mesurer l'amélioration

**Sprint 3 : Index + Schema cache**
- [ ] Créer les index DuckDB
- [ ] Cacher les vérifications de schéma
- [ ] Benchmark : mesurer l'amélioration

---

### 3. Comparaison Finale

**Objectif** : Atteindre ou dépasser les performances v3 pour l'UI

| Métrique | v3 (référence) | v5 actuelle | v5 optimisée | Objectif |
|----------|----------------|-------------|--------------|----------|
| Temps connexion | 5ms | 80ms | 10ms | <20ms |
| Temps load_matches(100) | 50ms | 200ms | 60ms | <80ms |
| Temps première page | 500ms | 1500ms | 600ms | <800ms |

---

## 📝 Conclusion

### Causes Racines

1. **Requêtes SQL trop complexes** (COALESCE, CASE, vérifications)
2. **Pas de réutilisation de connexion** entre pages
3. **Cache Streamlit trop granulaire** (invalidation fréquente)

### Solution : Architecture Hybride

**Conserver les avantages de la v5** :
- ✅ Shared matches (gain stockage + sync)
- ✅ Parallélisation API
- ✅ Calcul batch performance scores

**Restaurer les avantages de la v3** :
- ✅ Requêtes SQL simples (via vue matérialisée)
- ✅ Connexion persistante (cache repository)
- ✅ Schéma stable (plus de vérifications)

### Gains Attendus

**Après optimisations** :
- 🎯 Temps de chargement UI : **-60%**
- 🎯 Temps de connexion : **-80%**
- 🎯 Temps requêtes SQL : **-70%**

**Objectif global** : v5 UI **2× plus rapide que v3** tout en conservant les gains de la sync

---

## 🔗 Ressources

- Script de diagnostic : `scripts/diagnose_performance.py`
- Benchmarks v4.5 : `.ai/reports/benchmark_v4_5_post_s19.json`
- Architecture v5 : `docs/ARCHITECTURE_V5.md`
- Optimisations sync : `docs/SYNC_OPTIMIZATIONS_V5.md`
