# Analyses Approfondies - Phases 5 à 10

> **Document technique** : Analyses détaillées du code pour chaque phase de la migration V5 finale.
> 
> **Audience** : Développeurs implémentant les modifications
> 
> **Mis à jour** : 2026-02-16

---

## Table des matières

- [Phase 5 : Services et repositories (partie 1)](#phase-5)
- [Phase 6 : Repositories (partie 2) + UI critique](#phase-6)
- [Phase 7 : UI complète + filtres](#phase-7)
- [Phase 8 : Modules secondaires](#phase-8)
- [Phase 9 : Validation + cleanup](#phase-9)
- [Phase 10 : Documentation](#phase-10)

---

<a name="phase-5"></a>
## Phase 5 : Services et repositories (partie 1)

### Analyse : `src/data/services/teammates_service.py`

#### Vue d'ensemble du fichier

**Taille** : ~250 lignes
**Rôle** : Service de plus haut niveau pour charger et enrichir les stats des coéquipiers
**Dépendances critiques** :
- `DuckDBRepository` (repositories)
- `load_df_optimized` (cache)
- `compute_participation_profile` (visualisations)
- `get_all_impact_events` (analyse)

#### Architecture actuelle

```
UI (teammates.py)
    ↓
TeammatesService
    ├─→ load_teammate_stats()
    │   └─→ Lit data/players/{gamertag}/stats.duckdb
    │
    ├─→ enrich_series_with_perfect_kills()
    │   └─→ DuckDBRepository.count_perfect_kills_by_match()
    │       ├─ Essaie shared.medals_earned
    │       └─ Fallback medals_earned (local)
    │
    ├─→ compute_participation_profiles()
    │   └─→ DuckDBRepository.load_personal_score_awards_as_polars()
    │       └─ Lit personal_score_awards (local)
    │
    └─→ load_impact_data()
        └─→ DuckDBRepository._get_connection()
            ├─ Essaie shared.highlight_events
            └─ Fallback highlight_events (local)
```

#### Fonction 1 : `load_teammate_stats()`

**Localisation** : Lignes 90-115

**Signature actuelle** :
```python
def load_teammate_stats(
    teammate_gamertag: str,
    match_ids: list[str],
    reference_db_path: Path,
) -> TeammateStats:
```

**Flux actuel** :
1. Construit le chemin : `data/players/{teammate_gamertag}/stats.duckdb`
2. Vérifie l'existence du fichier
3. Charge la DB complète via `load_df_optimized()`
4. Filtre par `match_ids`
5. Retourne `TeammateStats(df, gamertag, is_empty)`

**Problèmes** :

1. **Dépendance filesystem** :
   ```python
   # Ligne 93-94
   base_dir = Path("data") / "players"
   teammate_db = base_dir / teammate_gamertag / "stats.duckdb"
   ```
   - ❌ Suppose que chaque joueur a sa propre DB
   - ❌ Utilise gamertag comme clé (instable)
   - ❌ Échouera si la DB du coéquipier n'existe pas

2. **Chargement complet de DB** :
   ```python
   # Ligne 100-102
   df = load_df_optimized(str(teammate_db))
   filtered = df.filter(pl.col("match_id").is_in(match_ids))
   ```
   - ❌ Charge TOUTE la DB en mémoire (peut être 30MB+)
   - ❌ Filtre après chargement (inefficace)
   - ❌ Cache peut exploser si 20+ coéquipiers

3. **Pas de handling erreur** :
   ```python
   # Ligne 95-97
   if not teammate_db.exists():
       return TeammateStats(df=pl.DataFrame(), gamertag=teammate_gamertag, is_empty=True)
   ```
   - ⚠️ Retourne DataFrame vide silencieusement
   - ⚠️ Peut cacher des problèmes de données

**Solution proposée** :

```python
def load_teammate_stats(
    teammate_xuid: str,  # ⚠️ BREAKING: xuid au lieu de gamertag
    match_ids: list[str],
    shared_db_path: Path,  # ⚠️ BREAKING: shared au lieu de reference
) -> TeammateStats:
    """
    Charge les stats d'un coéquipier depuis shared.match_participants.
    
    Args:
        teammate_xuid: XUID du coéquipier
        match_ids: Liste des match_id à charger
        shared_db_path: Chemin vers shared_matches.duckdb
    
    Returns:
        TeammateStats avec df, gamertag, is_empty
    
    Raises:
        ValueError: Si shared_db_path n'existe pas
        RuntimeError: Si pas de données pour ce xuid
    """
    if not shared_db_path.exists():
        raise ValueError(f"Shared DB not found: {shared_db_path}")
    
    conn = duckdb.connect(str(shared_db_path), read_only=True)
    
    try:
        # Requête ciblée avec JOIN pour gamertag
        query = """
            SELECT 
                mp.*,
                xa.gamertag
            FROM shared.match_participants mp
            LEFT JOIN shared.xuid_aliases xa 
                ON mp.xuid = xa.xuid
            WHERE mp.xuid = ?
              AND mp.match_id IN (SELECT unnest(?))
            ORDER BY mp.match_id
        """
        
        result = conn.execute(query, [teammate_xuid, match_ids])
        df = pl.from_arrow(result.fetch_arrow_table())
        
        if len(df) == 0:
            # Essayer de résoudre le gamertag quand même
            gamertag_query = "SELECT gamertag FROM shared.xuid_aliases WHERE xuid = ?"
            gt_result = conn.execute(gamertag_query, [teammate_xuid]).fetchone()
            gamertag = gt_result[0] if gt_result else "Unknown"
            
            return TeammateStats(
                df=pl.DataFrame(), 
                gamertag=gamertag, 
                is_empty=True
            )
        
        gamertag = df["gamertag"][0]
        
        return TeammateStats(
            df=df,
            gamertag=gamertag,
            is_empty=False
        )
    
    finally:
        conn.close()
```

**Avantages** :
- ✅ Requête SQL ciblée (charge uniquement les matchs demandés)
- ✅ Utilise xuid (stable) au lieu de gamertag
- ✅ JOIN pour résoudre gamertag depuis shared.xuid_aliases
- ✅ Pas de dépendance filesystem
- ✅ Gestion d'erreurs explicite

**Impact sur les appelants** :

**Fichier** : `src/ui/pages/teammates.py`
**Fonction** : `_load_teammate_stats_from_own_db()`
**Ligne** : 51

**Avant** :
```python
def _load_teammate_stats_from_own_db(gamertag, match_ids, reference_db_path):
    return load_teammate_stats(gamertag, match_ids, reference_db_path)
```

**Après** :
```python
def _load_teammate_stats_from_own_db(gamertag, match_ids, shared_db_path):
    # Résoudre gamertag → xuid
    from src.utils.xuid import resolve_xuid_from_db
    
    xuid = resolve_xuid_from_db(gamertag, shared_db_path)
    if not xuid:
        st.warning(f"XUID introuvable pour {gamertag}")
        return TeammateStats(df=pl.DataFrame(), gamertag=gamertag, is_empty=True)
    
    return load_teammate_stats(xuid, match_ids, shared_db_path)
```

#### Fonction 2 : `enrich_series_with_perfect_kills()`

**Localisation** : Lignes 120-155

**Signature actuelle** :
```python
def enrich_series_with_perfect_kills(
    series: list[tuple[str, pl.DataFrame]],
    db_path: Path,
) -> list[tuple[str, pl.DataFrame]]:
```

**Flux actuel** :
1. Itère sur `series` (liste de (nom, DataFrame))
2. Pour chaque DataFrame :
   - Extrait `xuid` (supposé constant dans le DF)
   - Appelle `DuckDBRepository.count_perfect_kills_by_match()`
   - Ajoute colonne `perfect_kills` avec les comptages

**Problèmes** :

1. **Ambiguïté db_path** :
   ```python
   # Ligne 130
   repo = DuckDBRepository(db_path, xuid=None)
   ```
   - ⚠️ `db_path` peut être player DB OU shared DB
   - ⚠️ DuckDBRepository a fallback interne (local → shared)

2. **Fallback implicite** :
   - `count_perfect_kills_by_match()` essaie d'abord table locale `medals_earned`
   - Si échec, essaie `shared.medals_earned`
   - ⚠️ Comportement non déterministe

**Solution proposée** :

```python
def enrich_series_with_perfect_kills(
    series: list[tuple[str, pl.DataFrame]],
    shared_db_path: Path,  # ⚠️ Explicite : toujours shared
) -> list[tuple[str, pl.DataFrame]]:
    """
    Enrichit les séries avec le comptage de perfect kills depuis shared.
    
    Args:
        series: Liste de (nom, DataFrame) à enrichir
        shared_db_path: Chemin vers shared_matches.duckdb
    
    Returns:
        Liste enrichie avec colonne 'perfect_kills'
    """
    if not shared_db_path.exists():
        # Retourner séries inchangées si shared indisponible
        return series
    
    enriched = []
    
    for name, df in series:
        if df.is_empty():
            enriched.append((name, df))
            continue
        
        # Extraire xuid (supposé constant dans le DF)
        xuid = df["xuid"][0] if "xuid" in df.columns else None
        if not xuid:
            enriched.append((name, df))
            continue
        
        match_ids = df["match_id"].unique().to_list()
        
        # Lire depuis shared uniquement (pas de fallback)
        conn = duckdb.connect(str(shared_db_path), read_only=True)
        
        try:
            # medal_name_id = 1512363953 = Perfect Kill
            query = """
                SELECT 
                    match_id,
                    COUNT(*) as perfect_count
                FROM shared.medals_earned
                WHERE xuid = ?
                  AND match_id IN (SELECT unnest(?))
                  AND medal_name_id = 1512363953
                GROUP BY match_id
            """
            
            result = conn.execute(query, [xuid, match_ids])
            perfect_df = pl.from_arrow(result.fetch_arrow_table())
            
            # Mapper match_id → perfect_count
            perfect_map = dict(zip(
                perfect_df["match_id"].to_list(),
                perfect_df["perfect_count"].to_list()
            ))
            
        finally:
            conn.close()
        
        # Ajouter colonne perfect_kills
        df_enriched = df.with_columns([
            pl.col("match_id").map_elements(
                lambda mid: perfect_map.get(mid, 0),
                return_dtype=pl.Int32
            ).alias("perfect_kills")
        ])
        
        enriched.append((name, df_enriched))
    
    return enriched
```

**Avantages** :
- ✅ Lit TOUJOURS depuis shared (pas de fallback)
- ✅ Requête SQL optimisée (GROUP BY dans la DB)
- ✅ Gestion explicite de shared indisponible
- ✅ Pas de dépendance sur DuckDBRepository (moins de couches)

#### Fonction 3 : `compute_participation_profiles()`

**Localisation** : Lignes 160-210

**Signature actuelle** :
```python
def compute_participation_profiles(
    players_data: list[tuple[str, pl.DataFrame]],
    db_path: Path,
    shared_match_ids: list[str],
) -> dict[str, dict[str, float]]:
```

**Flux actuel** :
1. Pour chaque joueur dans `players_data`
2. Charge `personal_score_awards` depuis sa player DB
3. Calcule le profil radar (6 axes)

**Particularité** :

```python
# Ligne 175-180
repo = DuckDBRepository(db_path, xuid=xuid)
awards = repo.load_personal_score_awards_as_polars(match_ids=shared_match_ids)
```

- ℹ️ `personal_score_awards` reste dans les player DBs (table conservée)
- ℹ️ Pas de migration vers shared pour cette table

**Problème** :

- ⚠️ `db_path` suppose qu'on passe la player DB du joueur
- ⚠️ Pour charger les profils de 5 coéquipiers, il faut 5 player DBs

**Solution proposée** :

```python
def compute_participation_profiles(
    players_data: list[tuple[str, pl.DataFrame]],
    player_db_paths: dict[str, Path],  # ⚠️ Nouveau : xuid → DB path
    shared_match_ids: list[str],
) -> dict[str, dict[str, float]]:
    """
    Calcule les profils radar depuis personal_score_awards (player DBs).
    
    Args:
        players_data: Liste de (nom, DataFrame stats)
        player_db_paths: Mapping xuid → chemin player DB
        shared_match_ids: Liste des match_id communs
    
    Returns:
        Dict {nom: {axe: valeur}} pour le radar
    
    Note:
        personal_score_awards reste dans les player DBs (table conservée).
        Cette fonction nécessite donc un mapping xuid → DB path.
    """
    profiles = {}
    
    for name, df in players_data:
        if df.is_empty():
            continue
        
        xuid = df["xuid"][0] if "xuid" in df.columns else None
        if not xuid:
            continue
        
        # Trouver la player DB pour ce xuid
        player_db = player_db_paths.get(xuid)
        if not player_db or not player_db.exists():
            # Pas de DB → profil vide
            profiles[name] = {
                "Kills": 0.0,
                "Assists": 0.0,
                "Objectifs": 0.0,
                "Véhicules": 0.0,
                "Pénalités": 0.0,
                "Survie": 0.0,
            }
            continue
        
        # Charger personal_score_awards depuis player DB
        repo = DuckDBRepository(player_db, xuid=xuid)
        awards = repo.load_personal_score_awards_as_polars(
            match_ids=shared_match_ids
        )
        
        # Calcul du profil (inchangé)
        from src.visualization.participation_radar import compute_participation_profile
        profile = compute_participation_profile(awards, df)
        
        profiles[name] = profile
    
    return profiles
```

**Avantages** :
- ✅ Explicite sur le besoin de player DBs
- ✅ Mapping xuid → DB path clair
- ✅ Gestion des DBs manquantes
- ⚠️ **Pas de changement majeur** car `personal_score_awards` reste local

**Impact sur les appelants** :

Les appelants devront construire `player_db_paths` :

```python
# Dans src/ui/pages/teammates.py

def render_participation_profiles(players_data, shared_db_path):
    # Construire le mapping xuid → player DB
    player_db_paths = {}
    
    for name, df in players_data:
        if not df.is_empty():
            xuid = df["xuid"][0]
            gamertag = df["gamertag"][0]
            
            # Chemin player DB (convention)
            player_db = Path("data/players") / gamertag / "stats.duckdb"
            if player_db.exists():
                player_db_paths[xuid] = player_db
    
    # Appel avec mapping
    profiles = compute_participation_profiles(
        players_data, 
        player_db_paths, 
        shared_match_ids
    )
```

#### Fonction 4 : `load_impact_data()`

**Localisation** : Lignes 215-250

**Signature actuelle** :
```python
def load_impact_data(
    db_path: Path,
    xuid: str,
    match_ids: list[str],
    friend_xuids: list[str],
) -> dict[str, Any]:
```

**Flux actuel** :
1. Ouvre connexion à `db_path`
2. Essaie de lire `shared.highlight_events`
3. Fallback sur `highlight_events` (local)
4. Lit aussi `match_stats` pour `outcome`
5. Traite via `get_all_impact_events()`

**Problèmes** :

1. **Ambiguïté db_path** :
   ```python
   # Ligne 220
   conn = duckdb.connect(str(db_path), read_only=True)
   ```
   - ⚠️ Peut être player DB ou shared DB

2. **Fallback implicite** :
   ```python
   # Lignes 225-235 (pseudo-code)
   try:
       df = conn.execute("SELECT * FROM shared.highlight_events").fetch_arrow()
   except:
       df = conn.execute("SELECT * FROM highlight_events").fetch_arrow()
   ```

3. **Schéma différent** :
   - Shared : `killer_xuid`, `victim_xuid`
   - Local : `xuid` (seul)

**Solution proposée** :

```python
def load_impact_data(
    shared_db_path: Path,  # ⚠️ Toujours shared
    xuid: str,
    match_ids: list[str],
    friend_xuids: list[str],
) -> dict[str, Any]:
    """
    Charge les highlight_events depuis shared pour analyse d'impact.
    
    Args:
        shared_db_path: Chemin vers shared_matches.duckdb
        xuid: XUID du joueur principal
        match_ids: Liste des match_id à analyser
        friend_xuids: Liste des xuid des amis
    
    Returns:
        Dict avec first_bloods, clutch_finishers, last_casualties, scores
    """
    if not shared_db_path.exists():
        raise ValueError(f"Shared DB not found: {shared_db_path}")
    
    conn = duckdb.connect(str(shared_db_path), read_only=True)
    
    try:
        # Requête avec JOIN pour outcome
        # Schéma V5 : killer_xuid, victim_xuid
        query = """
            SELECT 
                he.match_id,
                he.time_ms,
                he.killer_xuid,
                he.victim_xuid,
                mr.outcome
            FROM shared.highlight_events he
            JOIN shared.match_registry mr 
                ON he.match_id = mr.match_id
            WHERE he.match_id IN (SELECT unnest(?))
              AND (he.killer_xuid = ? OR he.victim_xuid = ?)
            ORDER BY he.match_id, he.time_ms
        """
        
        result = conn.execute(query, [match_ids, xuid, xuid])
        df = pl.from_arrow(result.fetch_arrow_table())
        
    finally:
        conn.close()
    
    # Traiter via friends_impact (inchangé)
    from src.analysis.friends_impact import get_all_impact_events
    return get_all_impact_events(df, xuid, friend_xuids)
```

**Avantages** :
- ✅ Toujours shared (pas de fallback)
- ✅ Schéma V5 (killer_xuid/victim_xuid)
- ✅ JOIN optimisé pour outcome
- ✅ Requête ciblée par match_ids

---

### Analyse : `src/data/repositories/_match_queries.py`

#### Vue d'ensemble du fichier

**Taille** : ~1000 lignes
**Rôle** : Cœur des requêtes de matchs (load_matches, filters, etc.)
**Complexité** : ⚠️ Très élevée (nombreux fallbacks, sous-requêtes complexes)

#### Fonction critique : `_get_match_source()`

**Localisation** : Lignes 100-200 (estimé)

**Rôle** : Génère la clause FROM pour toutes les requêtes de matchs

**Actuellement** :

```python
def _get_match_source(self, filters: dict = None) -> str:
    """
    Retourne la source de données pour les requêtes de matchs.
    
    V4 : "match_stats"
    V5 : Sous-requête UNION ALL (local + shared)
    """
    if self.version == "v4":
        return "match_stats"
    
    # V5 : Fallback local → shared
    return f"""
        (
            SELECT * FROM match_stats
            WHERE xuid = '{self.xuid}'
            UNION ALL
            SELECT * FROM shared.match_participants
            WHERE xuid = '{self.xuid}'
              AND match_id NOT IN (SELECT match_id FROM match_stats)
        ) AS match_stats
    """
```

**Problèmes** :

1. ❌ **UNION ALL coûteux** : Charge 2 sources puis déduplique
2. ❌ **Sous-requête corrélée** : NOT IN (SELECT ...) peut être lent
3. ❌ **Code complexe** : ~100 lignes pour gérer les cas

**Solution proposée** :

```python
def _get_match_source(self) -> str:
    """
    Retourne la source shared uniquement (V5 finale).
    
    Returns:
        "shared.match_participants"
    """
    return "shared.match_participants"
```

**Impact** :
- ✅ Simplification drastique (100 lignes → 3 lignes)
- ✅ Performances améliorées (1 seul scan au lieu de 2)
- ✅ Code maintenable

**Conséquence** :

Toutes les requêtes qui utilisent `_get_match_source()` seront simplifiées :

```python
# AVANT
query = f"""
    SELECT *
    FROM {self._get_match_source(filters)}  -- Sous-requête UNION ALL
    WHERE ...
"""

# APRÈS
query = f"""
    SELECT *
    FROM shared.match_participants  -- Direct
    WHERE xuid = '{self.xuid}'
      AND ...
"""
```

#### Fonction 2 : `load_match_mmr_batch()`

**Localisation** : Lignes ~500-550

**Actuellement** :

```python
def load_match_mmr_batch(self, match_ids: list[str]) -> pl.DataFrame:
    """Charge MMR depuis player_match_stats (table locale)."""
    conn = self._get_connection()
    
    query = """
        SELECT 
            match_id,
            team_mmr,
            individual_mmr,
            kills_expected,
            deaths_expected
        FROM player_match_stats
        WHERE match_id IN (SELECT unnest(?))
    """
    
    result = conn.execute(query, [match_ids])
    return pl.from_arrow(result.fetch_arrow_table())
```

**Problème** :
- ❌ `player_match_stats` sera supprimée (Phase 9)
- ❌ Données déjà dans `shared.match_participants` (ajoutées Phase 1)

**Solution** :

```python
def load_match_mmr_batch(self, match_ids: list[str]) -> pl.DataFrame:
    """Charge MMR depuis shared.match_participants."""
    shared_conn = self._get_shared_connection()
    
    query = """
        SELECT 
            match_id,
            xuid,
            team_mmr,
            team_mmr_remaining,
            team_mmr_sum,
            team_skill_sigma,
            individual_mmr,
            kills_expected,
            deaths_expected
        FROM shared.match_participants
        WHERE match_id IN (SELECT unnest(?))
          AND xuid = ?
        ORDER BY match_id
    """
    
    result = shared_conn.execute(query, [match_ids, str(self.xuid)])
    return pl.from_arrow(result.fetch_arrow_table())
```

**Avantages** :
- ✅ Lit depuis shared (colonnes ajoutées Phase 1)
- ✅ Filtre par xuid du joueur
- ✅ Retourne même structure (compatibilité)

#### Fonction 3 : `get_match_count()`

**Localisation** : Lignes ~600-650

**Actuellement** :

```python
def get_match_count(self) -> int:
    """Compte les matchs dans match_stats locale."""
    conn = self._get_connection()
    result = conn.execute("SELECT COUNT(*) FROM match_stats").fetchone()
    return result[0] if result else 0
```

**Problème** :
- ❌ Compte dans table locale (sera supprimée)

**Solution** :

```python
def get_match_count(self) -> int:
    """Compte les matchs dans shared pour ce joueur."""
    shared_conn = self._get_shared_connection()
    
    query = """
        SELECT COUNT(DISTINCT match_id)
        FROM shared.match_participants
        WHERE xuid = ?
    """
    
    result = shared_conn.execute(query, [str(self.xuid)]).fetchone()
    return result[0] if result else 0
```

**Avantages** :
- ✅ Compte dans shared
- ✅ DISTINCT pour éviter doublons (si backfill partiel)
- ✅ Filtre par xuid

---

### Analyse : `src/data/repositories/_roster_loader.py`

#### Vue d'ensemble

**Taille** : ~300 lignes
**Rôle** : Charge les rosters (joueurs, coéquipiers, adversaires)
**Problème** : ~15 fonctions avec fallback local → shared

#### Pattern répété à éliminer

**Actuellement (répété ~15 fois)** :

```python
def load_something(self, match_id: str):
    """Charge depuis local avec fallback shared."""
    conn = self._get_connection()
    
    try:
        # Essayer table locale
        result = conn.execute("""
            SELECT * FROM local_table WHERE match_id = ?
        """, [match_id]).fetchall()
        
        if result:
            return result
    except:
        pass
    
    # Fallback shared
    try:
        shared_conn = self._get_shared_connection()
        result = shared_conn.execute("""
            SELECT * FROM shared.table WHERE match_id = ?
        """, [match_id]).fetchall()
        return result
    except:
        return []
```

**Problèmes** :
1. ❌ Try/except masque les erreurs
2. ❌ Double accès DB (lent)
3. ❌ Code dupliqué 15 fois

**Solution** :

```python
def load_something(self, match_id: str):
    """Charge depuis shared uniquement."""
    shared_conn = self._get_shared_connection()
    
    result = shared_conn.execute("""
        SELECT * FROM shared.table WHERE match_id = ?
    """, [match_id]).fetchall()
    
    return result
```

**Impact** :
- ✅ Simplification ~150 lignes (fallbacks supprimés)
- ✅ Performances meilleures (1 accès au lieu de 2)
- ✅ Erreurs claires (pas de masquage)

#### Liste des méthodes à simplifier

1. `load_player_roster()` - ligne 50
2. `load_enemy_roster()` - ligne 80
3. `load_teammate_names()` - ligne 110
4. `get_player_aliases()` - ligne 140
5. `get_team_composition()` - ligne 170
6. `load_participant_list()` - ligne 200
7. `get_player_medals()` - ligne 230
8. `get_player_events()` - ligne 260
9. (6 autres - voir BUGFIX §6.2)

**Stratégie** :
- Identifier avec `grep -n "try:" _roster_loader.py`
- Pour chaque `try/except` :
  1. Supprimer bloc `try` local
  2. Garder uniquement bloc shared
  3. Supprimer `except` vide
  4. Ajouter docstring clarifiée

---

## Résumé Phase 5

### Modifications par fichier

| Fichier | Fonctions modifiées | Lignes estimées | Complexité |
|---------|---------------------|-----------------|------------|
| `teammates_service.py` | 4 | ~100 | 🔴 Élevée |
| `_match_queries.py` | 3 | ~50 | 🟡 Moyenne |
| `_roster_loader.py` | 15 | ~150 | 🟢 Faible (répétitif) |

### Risques

| Risque | Impact | Mitigation |
|--------|--------|------------|
| Breaking changes signatures | 🔴 | Mettre à jour tous appelants |
| Données manquantes shared | 🔴 | Vérifier couverture 100% (Phase 1) |
| Tests cassés | 🟡 | Mettre à jour en parallèle |

### Tests critiques

```python
# tests/test_phase5.py

def test_teammates_service_uses_shared_only():
    """Vérifie qu'aucun fallback local dans teammates_service."""
    import inspect
    import src.data.services.teammates_service as ts
    
    source = inspect.getsource(ts)
    
    # Ne doit PAS contenir "data/players"
    assert "data/players" not in source
    
    # Ne doit PAS contenir try/except avec fallback
    assert "try:" not in source or "except:" not in source


def test_match_queries_simplified():
    """Vérifie que _get_match_source est simplifié."""
    from src.data.repositories._match_queries import ...
    
    repo = DuckDBRepository(...)
    source = repo._get_match_source()
    
    assert source == "shared.match_participants"
    assert "UNION" not in source


def test_roster_loader_no_fallbacks():
    """Vérifie qu'aucun fallback dans _roster_loader."""
    import inspect
    import src.data.repositories._roster_loader as rl
    
    source = inspect.getsource(rl)
    
    # Compter les try/except (doit être 0 ou très peu)
    try_count = source.count("try:")
    assert try_count < 3  # Tolérance pour vrais besoins
```

---

<a name="phase-6"></a>
## Phase 6 : Repositories (partie 2) + UI critique

> **Note** : Cette section sera complétée après implémentation Phase 5

### Fichiers concernés

- `src/data/repositories/duckdb_repo.py` (8 méthodes)
- `src/data/repositories/_materialized_views.py`
- `src/ui/pages/teammates_impact.py`
- `src/ui/pages/objective_analysis.py`

### Aperçu des modifications

Voir BUGFIX_V5_2026-02-15.md §6.3 à §6.4 pour détails.

---

<a name="phase-7"></a>
## Phase 7 : UI complète + filtres

> **Note** : Cette section sera complétée après implémentation Phase 6

---

<a name="phase-8"></a>
## Phase 8 : Modules secondaires

> **Note** : Cette section sera complétée après implémentation Phase 7

---

<a name="phase-9"></a>
## Phase 9 : Validation + cleanup brutal

> **Note** : Cette section sera complétée après implémentation Phase 8

---

<a name="phase-10"></a>
## Phase 10 : Documentation

> **Note** : Cette section sera complétée après implémentation Phase 9

---

## Annexes

### Glossaire

| Terme | Définition |
|-------|------------|
| **Fallback** | Mécanisme try/except qui essaie local puis shared |
| **Dual write** | Écriture simultanée dans local ET shared |
| **Shared DB** | Base centralisée `shared_matches.duckdb` |
| **Player DB** | Base individuelle `data/players/{gamertag}/stats.duckdb` |
| **V4** | Architecture avec tables locales complètes |
| **V5 finale** | Architecture avec shared uniquement (cible) |

### Références

- **BUGFIX source** : `BUGFIX_V5_2026-02-15.md`
- **Architecture** : `docs/ARCHITECTURE_V5.md`
- **Schéma shared** : `docs/SHARED_MATCHES_SCHEMA.md`
