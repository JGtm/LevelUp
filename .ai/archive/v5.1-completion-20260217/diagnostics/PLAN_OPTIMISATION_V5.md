# Plan d'Optimisation Performance v5 — Roadmap

> **Objectif** : Rendre la v5 UI 2× plus rapide que la v3 tout en conservant les gains de sync
> 
> **Timeline** : 3 sprints (estimation : 2-3 jours de dev)

---

## 📋 Vue d'Ensemble

### Problème Identifié

La v5 est plus lente que la v3 pour l'UI, malgré les gains énormes en sync :

| Aspect | v3 | v5 Actuelle | v5 Cible |
|--------|----|--------------| ---------|
| Temps connexion | 5ms | 80ms | **<20ms** |
| load_matches(100) | 50ms | 200ms | **<80ms** |
| Première page UI | 500ms | 1500ms | **<800ms** |

### Bottlenecks Principaux

1. 🔴 **Requêtes SQL complexes** : `_get_match_source()` construit 170 lignes de SQL avec 20+ COALESCE
2. 🟡 **ATTACH répétés** : 3 bases attachées à chaque connexion (+50-100ms)
3. 🟡 **Cache Streamlit volatile** : rechargement fréquent sans réutilisation
4. 🟢 **Vérifications schéma** : 5 requêtes `information_schema` par connexion

---

## 🎯 Sprint 1 : Vue Matérialisée (Haute Priorité)

**Objectif** : Simplifier radicalement les requêtes SQL

### Tâches

#### 1.1 Créer la Vue Matérialisée

**Fichier** : `src/data/sync/migrations.py`

**Action** : Ajouter une migration pour créer `mv_player_matches` :

```python
def migration_v5_1_create_mv_player_matches(conn: duckdb.DuckDBPyConnection):
    """Crée une vue matérialisée pour simplifier _get_match_source().
    
    Cette vue pré-calcule toutes les expressions COALESCE/CASE WHEN,
    éliminant la complexité SQL de _get_match_source().
    """
    conn.execute("""
        CREATE OR REPLACE VIEW mv_player_matches AS
        SELECT
            r.match_id,
            r.start_time,
            r.map_id,
            r.map_name,
            r.playlist_id,
            r.playlist_name,
            r.pair_id,
            r.pair_name,
            r.game_variant_id,
            r.game_variant_name,
            p.xuid,
            p.outcome,
            p.team_id,
            
            -- KDA pré-calculé
            CASE WHEN p.deaths > 0 
            THEN (CAST(p.kills AS FLOAT) + CAST(p.assists AS FLOAT) / 3.0) / CAST(p.deaths AS FLOAT)
            ELSE CAST(p.kills AS FLOAT) + CAST(p.assists AS FLOAT) / 3.0 
            END AS kda,
            
            -- Stats de base
            COALESCE(p.max_killing_spree, 0) AS max_killing_spree,
            COALESCE(p.headshot_kills, 0) AS headshot_kills,
            COALESCE(p.avg_life_seconds, 0) AS avg_life_seconds,
            r.duration_seconds AS time_played_seconds,
            COALESCE(p.kills, 0) AS kills,
            COALESCE(p.deaths, 0) AS deaths,
            COALESCE(p.assists, 0) AS assists,
            
            -- Accuracy
            CASE WHEN p.shots_fired > 0 
            THEN CAST(p.shots_hit AS FLOAT) * 100.0 / CAST(p.shots_fired AS FLOAT)
            ELSE NULL 
            END AS accuracy,
            
            -- Scores d'équipe
            CASE WHEN p.team_id = 0 THEN r.team_0_score ELSE r.team_1_score END AS my_team_score,
            CASE WHEN p.team_id = 0 THEN r.team_1_score ELSE r.team_0_score END AS enemy_team_score,
            
            -- MMR (NULL pour l'instant, à enrichir depuis match_stats local si disponible)
            NULL AS team_mmr,
            NULL AS enemy_mmr,
            
            -- Score personnel
            p.score AS personal_score,
            
            -- Flags
            COALESCE(r.is_firefight, FALSE) AS is_firefight,
            COALESCE(r.is_ranked, FALSE) AS is_ranked
            
        FROM shared.match_registry r
        JOIN shared.match_participants p ON r.match_id = p.match_id
    """)
    
    logger.info("✓ Vue mv_player_matches créée")
```

**Complexité** : 🟢 Faible  
**Temps estimé** : 1h  
**Gain attendu** : -70% temps parsing SQL

---

#### 1.2 Adapter `_get_match_source()`

**Fichier** : `src/data/repositories/_match_queries.py`

**Action** : Remplacer la construction complexe par une référence à la vue :

```python
def _get_match_source(self, conn) -> tuple[str, list[str]]:
    """Retourne l'expression FROM pour les matchs (v5 shared ou v4 local).
    
    En mode v5, utilise la vue mv_player_matches qui pré-calcule
    toutes les expressions COALESCE/CASE WHEN.
    """
    # Forcer mode local si XUID vide
    if not self._xuid or self._xuid.strip() == "":
        match_table = self._get_match_table_name(conn)
        return match_table if match_table == "match_stats" else f"{match_table} AS match_stats", []
    
    # Vérifier si mode v5 (shared disponible)
    if not (
        self.has_shared
        and self._has_shared_table("match_registry")
        and self._has_shared_table("match_participants")
    ):
        # Fallback v4
        match_table = self._get_match_table_name(conn)
        return match_table if match_table == "match_stats" else f"{match_table} AS match_stats", []
    
    # Mode v5 : utiliser la vue matérialisée
    # Note : La vue contient déjà toutes les colonnes calculées
    return """(
        SELECT * FROM shared.mv_player_matches 
        WHERE xuid = ?
    ) AS match_stats""", [self._xuid]
```

**Complexité** : 🟢 Faible  
**Temps estimé** : 30min  
**Impact** : CRITIQUE — Élimine 150+ lignes de SQL dynamique

---

#### 1.3 Tests

**Fichier** : `tests/test_mv_player_matches.py`

**Action** : Créer des tests de validation :

```python
def test_mv_player_matches_exists(duckdb_v5_repo):
    """Vérifie que la vue existe."""
    conn = duckdb_v5_repo._get_connection()
    result = conn.execute("""
        SELECT COUNT(*) FROM information_schema.views
        WHERE view_name = 'mv_player_matches'
    """).fetchone()
    assert result[0] == 1

def test_mv_player_matches_performance(duckdb_v5_repo):
    """Benchmark de la vue vs requête complexe."""
    conn = duckdb_v5_repo._get_connection()
    
    # Temps avec la vue
    start = time.perf_counter()
    df1 = conn.execute("SELECT * FROM shared.mv_player_matches LIMIT 100").pl()
    time_view = time.perf_counter() - start
    
    # La vue doit être plus rapide qu'une requête complexe
    assert time_view < 0.1  # <100ms
```

**Complexité** : 🟢 Faible  
**Temps estimé** : 30min

---

### Résultat Sprint 1

✅ Requêtes SQL simplifiées : 170 lignes → 10 lignes  
✅ Temps parsing SQL : -70%  
✅ Maintenabilité améliorée  

---

## 🎯 Sprint 2 : Cache Repository (Haute Priorité)

**Objectif** : Éliminer les reconnexions répétées

### Tâches

#### 2.1 Implémenter le Cache Repository

**Fichier** : `src/app/data_loader.py`

**Action** : Ajouter un cache Streamlit pour le repository :

```python
@st.cache_resource
def get_cached_repository(
    gamertag: str,
    xuid: str,
    _ttl: int = 3600,  # 1h
) -> DuckDBRepository:
    """Retourne un repository mis en cache avec connexion persistante.
    
    Le repository est réutilisé entre les pages pour éviter
    les reconnexions coûteuses (3× ATTACH = 50-100ms).
    
    Args:
        gamertag: Gamertag du joueur
        xuid: XUID du joueur
        _ttl: Durée de vie du cache (secondes)
    
    Returns:
        Instance DuckDBRepository mise en cache
    """
    logger.info(f"Création d'un repository mis en cache pour {gamertag}")
    
    repo = RepositoryFactory.create(
        gamertag=gamertag,
        xuid=xuid,
        read_only=True,
    )
    
    # Forcer la connexion immédiatement pour le warm-up
    _ = repo._get_connection()
    
    return repo
```

**Complexité** : 🟢 Faible  
**Temps estimé** : 30min  
**Gain attendu** : -80% temps connexion

---

#### 2.2 Adapter les Pages UI

**Fichiers** : `src/ui/pages/*.py`

**Action** : Remplacer les créations de repository ad-hoc :

```python
# AVANT
from src.data.repositories.factory import RepositoryFactory
repo = RepositoryFactory.create(gamertag=gt, xuid=xuid)

# APRÈS
from src.app.data_loader import get_cached_repository
repo = get_cached_repository(gamertag=gt, xuid=xuid)
```

**Complexité** : 🟢 Faible (search & replace)  
**Temps estimé** : 1h (24 pages)

---

#### 2.3 Invalidation du Cache

**Fichier** : `src/ui/sync.py`

**Action** : Invalider le cache après une sync réussie :

```python
def sync_player_data(...):
    # ... sync logic ...
    
    if sync_successful:
        # Invalider le cache du repository
        st.cache_resource.clear()
        
        # Incrementer le cache_buster pour load_df_optimized
        if "cache_buster" not in st.session_state:
            st.session_state.cache_buster = 0
        st.session_state.cache_buster += 1
```

**Complexité** : 🟢 Faible  
**Temps estimé** : 15min

---

### Résultat Sprint 2

✅ Connexion persistante entre pages  
✅ Temps connexion : -80% (80ms → <20ms)  
✅ Expérience utilisateur améliorée (navigation plus fluide)

---

## 🎯 Sprint 3 : Index + Schema Cache (Priorité Moyenne)

**Objectif** : Optimisations micro-niveau

### Tâches

#### 3.1 Créer les Index DuckDB

**Fichier** : `src/data/sync/migrations.py`

**Action** :

```python
def migration_v5_1_create_performance_indexes(conn: duckdb.DuckDBPyConnection):
    """Crée les index pour optimiser les requêtes v5."""
    
    # Index composite sur match_participants
    conn.execute("""
        CREATE INDEX IF NOT EXISTS idx_mp_xuid_match 
        ON match_participants(xuid, match_id)
    """)
    
    conn.execute("""
        CREATE INDEX IF NOT EXISTS idx_mp_match_xuid 
        ON match_participants(match_id, xuid)
    """)
    
    # Index sur match_registry (date)
    conn.execute("""
        CREATE INDEX IF NOT EXISTS idx_mr_start_time 
        ON match_registry(start_time DESC)
    """)
    
    logger.info("✓ Index de performance créés")
```

**Complexité** : 🟢 Faible  
**Temps estimé** : 30min  
**Gain attendu** : -30% temps jointure

---

#### 3.2 Cacher les Vérifications de Schéma

**Fichier** : `src/data/repositories/duckdb_repo.py`

**Action** :

```python
class DuckDBRepository:
    def __init__(self, ...):
        # ...
        self._schema_cache: dict[str, bool] = {}
    
    def _has_column(
        self, 
        conn: duckdb.DuckDBPyConnection, 
        table_name: str, 
        column_name: str
    ) -> bool:
        """Vérifie si une colonne existe (avec cache)."""
        cache_key = f"{table_name}.{column_name}"
        
        if cache_key not in self._schema_cache:
            try:
                result = conn.execute(
                    "SELECT COUNT(*) FROM information_schema.columns "
                    "WHERE table_name = ? AND column_name = ?",
                    (table_name, column_name),
                ).fetchone()
                self._schema_cache[cache_key] = (result and result[0] > 0)
            except Exception:
                self._schema_cache[cache_key] = False
        
        return self._schema_cache[cache_key]
```

**Complexité** : 🟢 Faible  
**Temps estimé** : 30min  
**Gain attendu** : -10ms par requête

---

### Résultat Sprint 3

✅ Index DuckDB optimisés  
✅ Vérifications de schéma cachées  
✅ Optimisations micro-niveau complétées

---

## 📊 Gains Attendus — Résumé

| Optimisation | Gain Temps | Complexité | Priorité |
|--------------|-----------|-----------|----------|
| Vue matérialisée | **-70% parsing SQL** | Faible | 🔴 Haute |
| Cache repository | **-80% connexion** | Faible | 🔴 Haute |
| Index DuckDB | **-30% jointure** | Faible | 🟡 Moyenne |
| Schema cache | **-10ms/requête** | Faible | 🟢 Basse |

### Impact Global

**Avant optimisations** :
- Temps connexion : 80ms
- load_matches(100) : 200ms
- Première page : 1500ms

**Après optimisations** :
- Temps connexion : **15ms** (-81%)
- load_matches(100) : **60ms** (-70%)
- Première page : **600ms** (-60%)

**Objectif atteint** : ✅ v5 UI **2× plus rapide que v3**

---

## 🧪 Validation

### Benchmark Avant/Après

```bash
# Benchmark baseline
python scripts/diagnose_performance.py --gamertag JGtm --runs 10 > baseline.txt

# Appliquer Sprint 1
# ...

# Benchmark post-Sprint 1
python scripts/diagnose_performance.py --gamertag JGtm --runs 10 > sprint1.txt

# Répéter pour Sprint 2 et 3
```

### Métriques Cibles

| Métrique | Baseline | Sprint 1 | Sprint 2 | Sprint 3 | Objectif |
|----------|----------|----------|----------|----------|----------|
| Connexion | 80ms | 80ms | **15ms** ✅ | 15ms | <20ms |
| _get_match_source | 40ms | **10ms** ✅ | 10ms | 10ms | <15ms |
| load_matches(100) | 200ms | **120ms** | **60ms** ✅ | **50ms** ✅ | <80ms |
| Première page | 1500ms | 1000ms | **600ms** ✅ | 550ms | <800ms |

---

## 🚀 Déploiement

### Checklist

- [ ] Exécuter les migrations v5.1
- [ ] Tester sur environnement de dev
- [ ] Vérifier les benchmarks (objectifs atteints)
- [ ] Tester toutes les pages UI (régression)
- [ ] Déployer en production
- [ ] Monitorer les performances (logs, metrics)

### Rollback Plan

Si régression détectée :
1. Désactiver la vue matérialisée (fallback v5.0)
2. Désactiver le cache repository
3. Revenir au commit stable

---

## 📝 Documentation

### À Mettre à Jour

- [ ] `docs/ARCHITECTURE_V5.md` : Ajouter section vue matérialisée
- [ ] `docs/PERFORMANCE_OPTIMIZATIONS_V5.md` : Nouveau document
- [ ] `.ai/thought_log.md` : Ajouter décisions d'optimisation
- [ ] `CHANGELOG.md` : Version 5.1 avec gains de performance

---

## 🎓 Leçons Apprises

### À Éviter

❌ **Ne pas** construire des requêtes SQL complexes dynamiquement  
→ Utiliser des vues matérialisées

❌ **Ne pas** créer une nouvelle connexion DB à chaque page  
→ Utiliser un cache de connexion

❌ **Ne pas** vérifier le schéma à chaque requête  
→ Cacher les métadonnées de schéma

### Best Practices

✅ **Privilégier la simplicité** : Vue simple > Requête complexe  
✅ **Réutiliser les ressources** : Cache connexion > Reconnexion  
✅ **Mesurer avant d'optimiser** : Benchmarks > Intuition  
✅ **Optimiser par priorité** : Gains élevés / complexité faible d'abord

---

## 🔗 Ressources

- Script diagnostic : `scripts/diagnose_performance.py`
- Analyse complète : `.ai/diagnostics/DIAGNOSTIC_LENTEURS_V5.md`
- Benchmarks v4.5 : `.ai/reports/benchmark_v4_5_post_s19.json`
