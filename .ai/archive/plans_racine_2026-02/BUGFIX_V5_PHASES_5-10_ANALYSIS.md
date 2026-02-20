# Analyse Approfondie - Phases 5-10 de la Migration V5 Finale

**Date**: 16 février 2026  
**Contexte**: Préparation des phases 5-10 suite à la complétion des phases 0-4  
**Objectif**: Analyses complémentaires et recherche de contexte approfondie pour les phases restantes

---

## Table des matières

1. [Phase 5 - Analyse approfondie: Services et Repositories (partie 1)](#phase-5)
2. [Phase 6 - Analyse approfondie: Repositories (partie 2) + UI critique](#phase-6)
3. [Phase 7 - Analyse approfondie: UI complète + Filtres](#phase-7)
4. [Phase 8 - Analyse approfondie: Modules secondaires](#phase-8)
5. [Phase 9 - Plan de validation + Cleanup brutal](#phase-9)
6. [Phase 10 - Plan de documentation](#phase-10)
7. [Synthèse et recommandations](#synthese)

---

<a name="phase-5"></a>
## Phase 5 - Analyse Approfondie: Services et Repositories (partie 1)

### Objectif
Adapter les services critiques (teammates, match queries, roster) pour lire depuis shared au lieu des DBs individuelles.

### Fichiers concernés
- `src/data/services/teammates_service.py` (§5)
- `src/data/repositories/_match_queries.py` (§6.1)
- `src/data/repositories/_roster_loader.py` (§6.2)

---

### 5.1 Analyse détaillée: `teammates_service.py`

#### 5.1.1 Vue d'ensemble du module

**Rôle**: Service de haut niveau pour charger et enrichir les statistiques des coéquipiers.

**Fonctions principales**:

| Fonction | Objectif | Complexité migration |
|----------|----------|---------------------|
| `load_teammate_stats()` | Charge stats d'un coéquipier depuis sa DB | 🔴 **HAUTE** |
| `enrich_series_with_perfect_kills()` | Ajoute colonne perfect_kills via médailles | 🟡 **MOYENNE** |
| `compute_participation_profiles()` | Calcule profils radar 6-axes | 🟡 **MOYENNE** |
| `load_impact_data()` | Charge événements d'impact (first blood, etc.) | 🟡 **MOYENNE** |

#### 5.1.2 Analyse fonction par fonction

##### **`load_teammate_stats(teammate_gamertag, match_ids, reference_db_path)`**

**Implémentation actuelle**:
```python
def load_teammate_stats(
    teammate_gamertag: str, 
    match_ids: list[str], 
    reference_db_path: Path
) -> TeammateStats:
    base_dir = reference_db_path.parent.parent  # data/players/
    teammate_db = base_dir / teammate_gamertag / "stats.duckdb"
    
    if not teammate_db.exists():
        return TeammateStats(df=pl.DataFrame(), gamertag=teammate_gamertag, is_empty=True)
    
    df = load_df_optimized(str(teammate_db), cache_key=f"stats_{teammate_gamertag}")
    df_filtered = df.filter(pl.col("match_id").is_in(match_ids))
    
    return TeammateStats(df=df_filtered, gamertag=teammate_gamertag, is_empty=df_filtered.is_empty())
```

**Problèmes identifiés**:

1. **Architecture multi-DB incompatible avec shared**
   - Principe actuel: 1 DB par joueur (`data/players/{gamertag}/stats.duckdb`)
   - Cible V5: 1 shared DB unique (`data/warehouse/shared_matches.duckdb`)
   - Impact: Ne peut pas ouvrir 20+ DBs individuelles

2. **Dépendance au filesystem**
   - Résolution de chemin via `base_dir / teammate_gamertag`
   - Shared DB nécessite requête SQL avec filtre xuid/gamertag

3. **Cache optimisé pour chargement complet**
   - `load_df_optimized()` charge TOUTE la DB du coéquipier
   - Shared DB: requête filtrée plus efficace (match_ids + xuid)

4. **Type de retour inchangé**
   - `TeammateStats` dataclass conservé
   - Migration transparente pour les appelants

**Plan de migration**:

```python
def load_teammate_stats_v5(
    teammate_xuid: str,          # ⚠️ Changement: xuid au lieu de gamertag
    match_ids: list[str],
    shared_db_path: Path
) -> TeammateStats:
    """
    Charge les stats d'un coéquipier depuis shared.match_participants.
    
    Args:
        teammate_xuid: XUID du coéquipier (stable, non gamertag)
        match_ids: Liste des match_ids à filtrer
        shared_db_path: Chemin vers shared_matches.duckdb
    
    Returns:
        TeammateStats avec DataFrame Polars filtré
    """
    import duckdb
    
    conn = duckdb.connect(str(shared_db_path), read_only=True)
    
    # Résoudre le gamertag depuis shared.xuid_aliases
    gamertag_query = """
        SELECT gamertag 
        FROM shared.xuid_aliases 
        WHERE xuid = ? 
        LIMIT 1
    """
    result = conn.execute(gamertag_query, [teammate_xuid]).fetchone()
    gamertag = result[0] if result else teammate_xuid
    
    # Charger les stats depuis shared.match_participants
    placeholders = ",".join("?" * len(match_ids))
    stats_query = f"""
        SELECT 
            mp.match_id,
            mp.xuid,
            mp.rank,
            mp.outcome,
            mp.kills,
            mp.deaths,
            mp.assists,
            mp.kda,
            mp.damage_dealt,
            mp.damage_taken,
            mp.headshot_kills,
            mp.max_killing_spree,
            mp.accuracy,
            mp.grenade_kills,
            mp.melee_kills,
            mp.power_weapon_kills,
            mp.personal_score,
            mp.time_played_seconds,
            mp.team_mmr,
            mp.enemy_mmr,
            mr.start_time,
            mr.map_name,
            mr.playlist_name
        FROM shared.match_participants mp
        JOIN shared.match_registry mr ON mp.match_id = mr.match_id
        WHERE mp.xuid = ?
          AND mp.match_id IN ({placeholders})
        ORDER BY mr.start_time ASC
    """
    
    params = [teammate_xuid] + match_ids
    df = conn.execute(stats_query).pl()  # DuckDB → Polars zero-copy
    conn.close()
    
    is_empty = df.is_empty()
    return TeammateStats(df=df, gamertag=gamertag, is_empty=is_empty)
```

**Changements requis dans les appelants**:

Fichier `src/ui/pages/teammates.py` (ligne 51):

```python
# AVANT (V4)
teammate_stats = load_teammate_stats(
    teammate_gamertag=teammate_name,
    match_ids=common_matches,
    reference_db_path=Path(db_profiles[selected_player]["db_path"])
)

# APRÈS (V5)
from src.utils.xuid import resolve_xuid_from_gamertag

teammate_xuid = resolve_xuid_from_gamertag(teammate_name, shared_db_path)
if not teammate_xuid:
    st.warning(f"Impossible de résoudre le XUID pour {teammate_name}")
    return

teammate_stats = load_teammate_stats_v5(
    teammate_xuid=teammate_xuid,
    match_ids=common_matches,
    shared_db_path=SHARED_DB_PATH
)
```

**Points de vigilance**:

- ⚠️ **Résolution gamertag → xuid**: Ajouter étape de résolution avant appel
- ⚠️ **Gestion erreurs**: Handle cas où xuid inexistant dans shared.xuid_aliases
- ⚠️ **Performance**: Requête SQL filtrée vs cache complet - benchmark requis
- ⚠️ **Colonnes**: Vérifier que toutes les 31 colonnes sont utilisées par les appelants

---

##### **`enrich_series_with_perfect_kills(series, db_path)`**

**Implémentation actuelle**:
```python
def enrich_series_with_perfect_kills(
    series: list[tuple[str, pl.DataFrame]], 
    db_path: str
) -> list[tuple[str, pl.DataFrame]]:
    """
    Ajoute une colonne 'perfect_kills' à chaque DataFrame dans series.
    
    series: [(player_name, df_matches), ...]
    db_path: Chemin vers la DB (joueur principal à idx=0, autres = leurs DBs)
    """
    enriched = []
    
    for idx, (player_name, df) in enumerate(series):
        match_ids = df["match_id"].to_list()
        
        if idx == 0:
            # Joueur principal: utiliser sa DB
            repo = DuckDBRepository(db_path, xuid=None)
        else:
            # Coéquipier: chercher sa DB
            teammate_db = Path(db_path).parent.parent / player_name / "stats.duckdb"
            if not teammate_db.exists():
                # Pas de DB → 0 perfect kills partout
                df = df.with_columns(pl.lit(0).alias("perfect_kills"))
                enriched.append((player_name, df))
                continue
            repo = DuckDBRepository(str(teammate_db), xuid=None)
        
        # Charger les perfect kills (medal_name_id = 1512363953)
        perfect_counts = repo.count_perfect_kills_by_match(match_ids)
        
        # Mapper match_id → count
        df = df.with_columns(
            pl.col("match_id").map_dict(perfect_counts, default=0).alias("perfect_kills")
        )
        
        enriched.append((player_name, df))
    
    return enriched
```

**Problèmes identifiés**:

1. **Multi-DB pour coéquipiers**
   - Ouvre autant de DBs que de coéquipiers (N DBs)
   - Shared DB: 1 seule requête avec IN clause sur xuids

2. **Dépendance `DuckDBRepository`**
   - Utilise `count_perfect_kills_by_match()` qui lit medals_earned
   - Déjà compatible shared (méthode a fallback shared → local)

3. **Gestion des DB manquantes**
   - Fallback: 0 perfect kills si DB inexistante
   - Shared: toujours disponible

**Plan de migration**:

```python
def enrich_series_with_perfect_kills_v5(
    series: list[tuple[str, pl.DataFrame]],
    shared_db_path: str,
    xuid_map: dict[str, str]  # {player_name: xuid}
) -> list[tuple[str, pl.DataFrame]]:
    """
    Ajoute colonne 'perfect_kills' en lisant shared.medals_earned.
    
    Args:
        series: [(player_name, df_matches), ...]
        shared_db_path: Chemin shared_matches.duckdb
        xuid_map: Mapping player_name → xuid pour filtrage
    """
    import duckdb
    
    conn = duckdb.connect(shared_db_path, read_only=True)
    
    # Collecter tous les match_ids + xuids nécessaires
    all_match_ids = set()
    xuids_needed = set()
    for player_name, df in series:
        all_match_ids.update(df["match_id"].to_list())
        xuids_needed.add(xuid_map.get(player_name))
    
    # Requête batch pour TOUS les joueurs
    match_placeholders = ",".join("?" * len(all_match_ids))
    xuid_placeholders = ",".join("?" * len(xuids_needed))
    
    query = f"""
        SELECT 
            me.match_id,
            me.xuid,
            SUM(me.count) as perfect_count
        FROM shared.medals_earned me
        WHERE me.medal_name_id = 1512363953
          AND me.match_id IN ({match_placeholders})
          AND me.xuid IN ({xuid_placeholders})
        GROUP BY me.match_id, me.xuid
    """
    
    params = list(all_match_ids) + list(xuids_needed)
    result_df = conn.execute(query).pl()
    conn.close()
    
    # Créer un dict {(match_id, xuid): perfect_count}
    perfect_map = {
        (row["match_id"], row["xuid"]): row["perfect_count"]
        for row in result_df.iter_rows(named=True)
    }
    
    # Enrichir chaque série
    enriched = []
    for player_name, df in series:
        xuid = xuid_map.get(player_name)
        
        # Mapper avec (match_id, xuid) composite key
        def get_perfect_count(match_id):
            return perfect_map.get((match_id, xuid), 0)
        
        df = df.with_columns(
            pl.col("match_id").apply(get_perfect_count).alias("perfect_kills")
        )
        
        enriched.append((player_name, df))
    
    return enriched
```

**Avantages de la migration**:

- ✅ **1 requête au lieu de N**: Batch loading pour tous les joueurs
- ✅ **Plus de dépendance filesystem**: Tout passe par xuid
- ✅ **Toujours disponible**: shared.medals_earned contient les médailles de TOUS les joueurs
- ✅ **Performance**: Requête SQL optimisée avec index sur (match_id, xuid, medal_name_id)

**Points de vigilance**:

- ⚠️ **xuid_map requis**: Les appelants doivent fournir le mapping name → xuid
- ⚠️ **Gestion NULL**: Vérifier que perfect_count=NULL devient 0
- ⚠️ **Index**: Créer index sur shared.medals_earned si nécessaire

---

##### **`compute_participation_profiles(players_data, db_path, shared_match_ids)`**

**Implémentation actuelle**:
```python
def compute_participation_profiles(
    players_data: list[tuple[str, pl.DataFrame]],
    db_path: str,
    shared_match_ids: set[str]
) -> list[dict]:
    """
    Calcule les profils de participation radar (6 axes) pour chaque joueur.
    
    Axes:
    - Kills normalisés
    - Assists normalisés
    - Objectives (personal_score_awards)
    - Vehicle destruction
    - Pénalités (suicides, betrayals)
    - Consistency (écart-type des perfs)
    
    Returns: [{"name": str, "profile": dict}, ...]
    """
    from src.visualization.participation_radar import (
        compute_participation_profile,
        get_radar_thresholds
    )
    
    profiles = []
    metadata_db = Path(db_path).parent.parent.parent / "warehouse" / "metadata.duckdb"
    thresholds = get_radar_thresholds(str(metadata_db))
    
    for player_name, df in players_data:
        # Charger personal_score_awards depuis la DB du joueur
        player_db = Path(db_path).parent.parent / player_name / "stats.duckdb"
        
        if not player_db.exists():
            # Pas de DB → profil vide
            profiles.append({
                "name": player_name,
                "profile": {ax: 0.0 for ax in ["kills", "assists", "objectives", "vehicle", "penalties", "consistency"]}
            })
            continue
        
        repo = DuckDBRepository(str(player_db), xuid=None)
        awards_df = repo.load_personal_score_awards_as_polars(list(shared_match_ids))
        
        # Combiner avec stats match (deaths, time_played depuis df)
        combined_df = df.join(awards_df, on="match_id", how="left")
        
        # Calculer le profil
        profile = compute_participation_profile(combined_df, thresholds)
        
        profiles.append({"name": player_name, "profile": profile})
    
    return profiles
```

**Problèmes identifiés**:

1. **`personal_score_awards` est dans player DBs**
   - Table locale, PAS ENCORE dans shared
   - Migration V5 ne prévoit PAS de migrer cette table vers shared

2. **Multi-DB pour coéquipiers**
   - Ouvre N DBs pour charger awards de chaque joueur
   - Inefficace si awards migré vers shared

3. **Plan V5 actuel**:
   - ✅ `personal_score_awards` reste dans player DBs
   - ⚠️ Contradiction: comment lire awards d'un coéquipier si sa DB est nettoyée?

**🔴 BLOCAGE CRITIQUE DÉTECTÉ**:

Le plan V5 Phase 5 dit "supprimer les tables locales", MAIS `personal_score_awards` doit rester locale (selon §14.2).

**Résolution**:

1. **Option A (conservateur)**: Garder `personal_score_awards` locale
   - PRO: Pas de migration complexe
   - CON: Impossible de calculer profils coéquipiers après cleanup

2. **Option B (migration complète)**: Migrer `personal_score_awards` vers shared
   - PRO: Toutes les données dans shared
   - CON: Table très volumineuse (1 ligne/award/match)

**Recommandation**: **Option A avec restriction fonctionnelle**

- Garder `personal_score_awards` locale
- `compute_participation_profiles()` ne fonctionne QUE pour le joueur principal
- Désactiver profils radar pour coéquipiers dans l'UI

**Plan de migration**:

```python
def compute_participation_profiles_v5(
    main_player_xuid: str,
    main_player_df: pl.DataFrame,
    main_player_db: str,
    shared_match_ids: set[str]
) -> dict:
    """
    Calcule le profil radar du joueur principal UNIQUEMENT.
    
    ⚠️ V5: Ne supporte plus les profils de coéquipiers (awards locaux uniquement).
    
    Args:
        main_player_xuid: XUID du joueur principal
        main_player_df: DataFrame des matchs
        main_player_db: Chemin vers stats.duckdb du joueur
        shared_match_ids: Matchs à analyser
    
    Returns:
        {"name": str, "profile": dict}
    """
    from src.visualization.participation_radar import (
        compute_participation_profile,
        get_radar_thresholds
    )
    
    # Charger personal_score_awards DEPUIS LA DB LOCALE (table conservée)
    repo = DuckDBRepository(main_player_db, xuid=main_player_xuid)
    awards_df = repo.load_personal_score_awards_as_polars(list(shared_match_ids))
    
    # Combiner avec stats depuis shared (via main_player_df)
    combined_df = main_player_df.join(awards_df, on="match_id", how="left")
    
    # Thresholds depuis metadata
    metadata_db = Path(main_player_db).parent.parent / "warehouse" / "metadata.duckdb"
    thresholds = get_radar_thresholds(str(metadata_db))
    
    # Calculer profil
    profile = compute_participation_profile(combined_df, thresholds)
    
    return {"name": "Joueur principal", "profile": profile}
```

**Changements UI requis**:

Fichier `src/ui/pages/teammates.py` (section participation):

```python
# AVANT (V4)
profiles = compute_participation_profiles(
    players_data=[(main_player, main_df)] + teammate_series,
    db_path=db_path,
    shared_match_ids=common_match_ids
)

# Afficher radar pour tous les joueurs
for profile_data in profiles:
    st.plotly_chart(create_radar(profile_data))

# APRÈS (V5)
# Profil uniquement pour le joueur principal
main_profile = compute_participation_profiles_v5(
    main_player_xuid=xuid,
    main_player_df=main_df,
    main_player_db=db_path,
    shared_match_ids=common_match_ids
)

st.plotly_chart(create_radar(main_profile))

# Avertissement pour coéquipiers
st.info("ℹ️ Profils de participation disponibles uniquement pour le joueur principal en V5.")
```

**Points de vigilance**:

- ⚠️ **Restriction fonctionnelle**: Documenter que profils coéquipiers désactivés
- ⚠️ **Migration future possible**: Garder porte ouverte pour migrer awards vers shared
- ⚠️ **UX dégradée**: Utilisateurs perdent comparaison radar multi-joueurs

---

##### **`load_impact_data(db_path, xuid, match_ids, friend_xuids)`**

**Implémentation actuelle**:
```python
def load_impact_data(
    db_path: str,
    xuid: str,
    match_ids: list[str],
    friend_xuids: set[str]
) -> dict:
    """
    Charge highlight_events + match outcomes pour analyse d'impact.
    
    Détecte:
    - First bloods (premier kill du match)
    - Clutch finishers (dernier kill d'un match gagné)
    - Last casualties (dernier mort d'un match perdu)
    
    Returns:
        {
            "first_bloods": {match_id: [gamertags]},
            "clutch_finishers": {match_id: [gamertags]},
            "last_casualties": {match_id: [gamertags]},
            "scores": {xuid: impact_score}
        }
    """
    from src.analysis.friends_impact import get_all_impact_events
    
    conn = DuckDBRepository(db_path, xuid)._get_connection()
    
    # Détecter shared vs local
    has_shared_events = _has_shared_table(conn, "highlight_events")
    has_shared_stats = _has_shared_table(conn, "match_stats")  # ⚠️ Erreur: shared.match_registry
    
    if has_shared_events:
        # V5: Lire depuis shared
        events_query = """
            SELECT 
                he.match_id,
                he.event_type,
                he.time_ms,
                CASE 
                    WHEN LOWER(he.event_type) = 'kill' THEN he.killer_xuid
                    ELSE he.victim_xuid
                END as xuid,
                CASE
                    WHEN LOWER(he.event_type) = 'kill' THEN he.killer_gamertag
                    ELSE he.victim_gamertag
                END as gamertag
            FROM shared.highlight_events he
            WHERE he.match_id IN ({placeholders})
              AND (he.killer_xuid IN ({xuid_placeholders}) 
                   OR he.victim_xuid IN ({xuid_placeholders}))
            ORDER BY he.match_id, he.time_ms ASC
        """
        params = match_ids + list(friend_xuids) * 2
    else:
        # V4: Lire depuis local
        events_query = """
            SELECT match_id, event_type, time_ms, xuid, gamertag
            FROM highlight_events
            WHERE match_id IN ({placeholders})
            ORDER BY match_id, time_ms ASC
        """
        params = match_ids
    
    events = conn.execute(events_query, params).fetchall()
    
    # Charger outcomes
    if has_shared_stats:
        outcomes_query = """
            SELECT mr.match_id, mp.outcome
            FROM shared.match_registry mr
            JOIN shared.match_participants mp ON mr.match_id = mp.match_id
            WHERE mr.match_id IN ({placeholders})
              AND mp.xuid = ?
        """
        params_out = match_ids + [xuid]
    else:
        outcomes_query = """
            SELECT match_id, outcome
            FROM match_stats
            WHERE match_id IN ({placeholders})
        """
        params_out = match_ids
    
    outcomes = conn.execute(outcomes_query, params_out).fetchall()
    outcomes_map = {row[0]: row[1] for row in outcomes}
    
    # Analyser les événements
    impact_data = get_all_impact_events(events, outcomes_map, friend_xuids)
    
    return impact_data
```

**Problèmes identifiés**:

1. **Requête shared.match_stats inexistante**
   - Ligne 307: `_has_shared_table(conn, "match_stats")`
   - ⚠️ ERREUR: shared.match_stats n'existe PAS, c'est shared.match_registry

2. **Outcome dans shared**
   - outcome est dans `shared.match_participants.outcome` (1 par joueur)
   - PAS dans `shared.match_registry`

3. **Schéma highlight_events V5**
   - Correctement géré: killer_xuid/victim_xuid
   - CASE pour normaliser en xuid unique

**Plan de migration**:

```python
def load_impact_data_v5(
    shared_db_path: str,
    xuid: str,
    match_ids: list[str],
    friend_xuids: set[str]
) -> dict:
    """
    V5: Charge depuis shared.highlight_events + shared.match_participants.
    
    ⚠️ Toujours lire shared (plus de fallback local).
    """
    from src.analysis.friends_impact import get_all_impact_events
    import duckdb
    
    conn = duckdb.connect(shared_db_path, read_only=True)
    
    # Charger highlight events (TOUS les joueurs du friend_xuids set)
    match_placeholders = ",".join("?" * len(match_ids))
    xuid_placeholders = ",".join("?" * len(friend_xuids))
    
    events_query = f"""
        SELECT 
            he.match_id,
            he.event_type,
            he.time_ms,
            CASE 
                WHEN LOWER(he.event_type) = 'kill' THEN he.killer_xuid
                WHEN LOWER(he.event_type) = 'death' THEN he.victim_xuid
                ELSE NULL
            END as xuid,
            CASE
                WHEN LOWER(he.event_type) = 'kill' THEN he.killer_gamertag
                WHEN LOWER(he.event_type) = 'death' THEN he.victim_gamertag
                ELSE ''
            END as gamertag
        FROM shared.highlight_events he
        WHERE he.match_id IN ({match_placeholders})
          AND (he.killer_xuid IN ({xuid_placeholders}) 
               OR he.victim_xuid IN ({xuid_placeholders}))
        ORDER BY he.match_id, he.time_ms ASC
    """
    
    events_params = match_ids + list(friend_xuids) * 2
    events = conn.execute(events_query, events_params).fetchall()
    
    # Charger outcomes (depuis match_participants, pas match_registry)
    outcomes_query = f"""
        SELECT 
            mp.match_id,
            mp.outcome
        FROM shared.match_participants mp
        WHERE mp.match_id IN ({match_placeholders})
          AND mp.xuid = ?
    """
    
    outcomes_params = match_ids + [xuid]
    outcomes = conn.execute(outcomes_query, outcomes_params).fetchall()
    outcomes_map = {row[0]: row[1] for row in outcomes}
    
    conn.close()
    
    # Analyser
    impact_data = get_all_impact_events(events, outcomes_map, friend_xuids)
    
    return impact_data
```

**Points de vigilance**:

- ✅ **shared.match_participants.outcome** correct
- ⚠️ **CASE WHEN**: Gérer event_type avec LOWER() pour robustesse
- ⚠️ **NULL xuids**: Filter NULL après CASE si nécessaire
- ⚠️ **Performance**: Index sur (match_id, killer_xuid, victim_xuid)

---

#### 5.1.3 Résumé des modifications `teammates_service.py`

| Fonction | Changement signature | Migration DB | Complexité | Restriction fonctionnelle |
|----------|---------------------|--------------|------------|--------------------------|
| `load_teammate_stats()` | xuid au lieu de gamertag | Local → Shared | 🔴 HAUTE | Non |
| `enrich_series_with_perfect_kills()` | Ajouter xuid_map | Local → Shared | 🟡 MOYENNE | Non |
| `compute_participation_profiles()` | Supporter joueur principal uniquement | Hybride (awards local) | 🟡 MOYENNE | ✅ **OUI** (coéquipiers désactivés) |
| `load_impact_data()` | shared_db_path au lieu de db_path | Local → Shared | 🟡 MOYENNE | Non |

**Total LOC modifiées**: ~150 lignes  
**Tests requis**: 12 tests unitaires + 3 tests d'intégration

---

### 5.2 Analyse détaillée: `_match_queries.py`

#### 5.2.1 Fonction `_get_match_source()`

**Rôle central**: Router toutes les requêtes de chargement de matchs vers la bonne source (shared vs local).

**Complexité actuelle**: ~170 lignes de logique conditionnelle avec 8 vérifications de colonnes.

**Plan de simplification**:

```python
def _get_match_source_v5_final(self) -> tuple[str, list]:
    """
    V5 Final: TOUJOURS utiliser shared.match_registry + shared.match_participants.
    
    ⚠️ Suppression des fallbacks locaux.
    
    Returns:
        (source_sql, params)
        
    Raises:
        RuntimeError: Si shared schema absent ou xuid non défini
    """
    if not self._xuid:
        raise RuntimeError(
            "XUID requis pour charger les matchs en V5. "
            "Utilisez DuckDBRepository(db_path, xuid=...)."
        )
    
    # Vérifier présence shared schema
    conn = self._get_connection()
    if not self._has_shared_table("match_registry"):
        raise RuntimeError(
            "Table shared.match_registry introuvable. "
            "Exécutez la migration V5 avant de continuer."
        )
    
    if not self._has_shared_table("match_participants"):
        raise RuntimeError(
            "Table shared.match_participants introuvable. "
            "Exécutez la migration V5 avant de continuer."
        )
    
    # Construire la source unique (pas de COALESCE)
    source_sql = """
    (
        SELECT
            mr.match_id,
            mr.start_time,
            mr.end_time,
            mr.duration_seconds,
            mr.map_name,
            mr.map_asset_id,
            mr.playlist_name,
            mr.playlist_asset_id,
            mr.game_variant_name,
            mr.is_ranked,
            mr.is_firefight,
            mp.xuid,
            mp.rank,
            mp.outcome,
            mp.kills,
            mp.deaths,
            mp.assists,
            mp.kda,
            mp.damage_dealt,
            mp.damage_taken,
            mp.headshot_kills,
            mp.max_killing_spree,
            mp.accuracy,
            mp.grenade_kills,
            mp.melee_kills,
            mp.power_weapon_kills,
            mp.personal_score,
            mp.time_played_seconds,
            mp.team_mmr,
            mp.enemy_mmr,
            mp.kills_expected,
            mp.deaths_expected
        FROM shared.match_registry mr
        JOIN shared.match_participants mp 
            ON mr.match_id = mp.match_id
        WHERE mp.xuid = ?
    ) AS match_stats
    """
    
    return (source_sql, [self._xuid])
```

**Avantages**:

- ✅ **170 → 60 lignes**: Suppression de toute la logique COALESCE
- ✅ **0 vérifications de colonnes**: Schéma V5 stable et documenté
- ✅ **Erreurs explicites**: RuntimeError si migration non faite
- ✅ **Performance**: 1 JOIN au lieu de 3 LEFT JOINs + COALESCE

**Migration des appelants**:

Toutes les méthodes `load_matches*()` utilisent `_get_match_source()` sans changement:

```python
def load_matches(self, playlist=None, map_name=None, ...):
    source_sql, source_params = self._get_match_source()  # ✅ Inchangé
    
    # Construire WHERE
    where_clauses = []
    params = list(source_params)
    
    if playlist:
        where_clauses.append("playlist_name = ?")
        params.append(playlist)
    
    # ... reste identique
```

**Points de vigilance**:

- ⚠️ **Gestion erreurs**: Les RuntimeError doivent être catchées par l'UI
- ⚠️ **Documentation**: Documenter que V5 nécessite migration préalable
- ⚠️ **Tests**: Ajouter tests d'échec si shared absent

---

#### 5.2.2 Fonction `load_match_mmr_batch()`

**Problème actuel**: Fallback à `player_match_stats` (table locale supprimée en V5).

**Plan de migration**:

```python
def load_match_mmr_batch_v5(
    self, 
    match_ids: list[str]
) -> dict[str, tuple[float | None, float | None]]:
    """
    V5: Charge MMR depuis shared.match_participants directement.
    
    Plus de fallback à player_match_stats (table supprimée).
    
    Returns:
        {match_id: (team_mmr, enemy_mmr), ...}
    """
    if not match_ids:
        return {}
    
    conn = self._get_connection()
    placeholders = ",".join("?" * len(match_ids))
    
    # Lire depuis shared.match_participants (colonnes team_mmr, enemy_mmr)
    query = f"""
        SELECT 
            mp.match_id,
            mp.team_mmr,
            mp.enemy_mmr
        FROM shared.match_participants mp
        WHERE mp.match_id IN ({placeholders})
          AND mp.xuid = ?
    """
    
    params = match_ids + [self._xuid]
    rows = conn.execute(query, params).fetchall()
    
    return {
        row[0]: (row[1], row[2]) 
        for row in rows
    }
```

**Simplification**:

- ✅ **Suppression du fallback**: player_match_stats n'existe plus
- ✅ **1 requête**: Pas de LEFT JOIN conditionnel
- ✅ **Colonnes garanties**: team_mmr et enemy_mmr dans shared.match_participants (migration Phase 1)

---

#### 5.2.3 Fonction `get_match_count()`

**Problème**: Utilise COUNT(*) sur match_stats locale.

**Plan de migration**:

```python
def get_match_count_v5(self) -> int:
    """
    V5: Compte matchs depuis shared.match_participants.
    
    Returns:
        Nombre de matchs pour ce xuid
    """
    conn = self._get_connection()
    
    query = """
        SELECT COUNT(DISTINCT mp.match_id)
        FROM shared.match_participants mp
        WHERE mp.xuid = ?
    """
    
    result = conn.execute(query, [self._xuid]).fetchone()
    return result[0] if result else 0
```

**Points de vigilance**:

- ⚠️ **DISTINCT**: Nécessaire si match_participants peut avoir duplicatas (ne devrait pas)
- ⚠️ **Performance**: Index sur (xuid, match_id)

---

#### 5.2.4 Résumé des modifications `_match_queries.py`

| Fonction | LOC avant | LOC après | Réduction | Complexité |
|----------|-----------|-----------|-----------|------------|
| `_get_match_source()` | 170 | 60 | -65% | 🔴 HAUTE |
| `load_match_mmr_batch()` | 45 | 20 | -56% | 🟡 MOYENNE |
| `get_match_count()` | 15 | 10 | -33% | 🟢 FAIBLE |
| **Total** | **230** | **90** | **-61%** | - |

**Tests requis**: 8 tests unitaires + 4 tests d'intégration

---

### 5.3 Analyse détaillée: `_roster_loader.py`

#### 5.3.1 Fallbacks identifiés (15 au total)

Voir analyse détaillée section précédente (explore agent).

#### 5.3.2 Plan de migration

**Principe**: Supprimer TOUS les fallbacks locaux, forcer lecture shared.

**Exemple**: `load_match_rosters()`

```python
def load_match_rosters_v5(
    self,
    match_id: str
) -> dict[str, list[str]]:
    """
    V5: Charge rosters depuis shared.killer_victim_pairs.
    
    Plus de fallback highlight_events.
    
    Returns:
        {
            "team_0": [xuid, xuid, ...],
            "team_1": [xuid, xuid, ...]
        }
    """
    conn = self._get_connection()
    
    # Vérifier présence shared.killer_victim_pairs
    if not self._has_shared_table("killer_victim_pairs"):
        raise RuntimeError(
            "Table shared.killer_victim_pairs introuvable. "
            "Exécutez le backfill --killer-victim avant de continuer."
        )
    
    query = """
        SELECT 
            kvp.xuid,
            kvp.team_id
        FROM shared.killer_victim_pairs kvp
        WHERE kvp.match_id = ?
        GROUP BY kvp.xuid, kvp.team_id
    """
    
    rows = conn.execute(query, [match_id]).fetchall()
    
    rosters = {"team_0": [], "team_1": []}
    for xuid, team_id in rows:
        if team_id == 0:
            rosters["team_0"].append(xuid)
        elif team_id == 1:
            rosters["team_1"].append(xuid)
    
    return rosters
```

**Simplification**:

- ✅ **Suppression 300 lignes**: Fallback highlight_events + 50/50 split
- ✅ **Erreurs explicites**: RuntimeError si table absente
- ✅ **Fiabilité**: killer_victim_pairs autoritaire sur team_id

---

#### 5.3.3 Résumé des modifications `_roster_loader.py`

| Fonction | Fallbacks supprimés | LOC avant | LOC après | Réduction |
|----------|---------------------|-----------|-----------|-----------|
| `load_match_rosters()` | 3 | 350 | 50 | -86% |
| `load_matches_with_teammate()` | 2 | 80 | 30 | -63% |
| `load_same_team_match_ids()` | 3 | 100 | 40 | -60% |
| `has_match_participants()` | 2 | 30 | 10 | -67% |
| `resolve_gamertag()` | 5 | 120 | 50 | -58% |
| `load_match_player_gamertags()` | 4 | 150 | 60 | -60% |
| `load_match_players_stats()` | 1 | 100 | 40 | -60% |
| **Total** | **15** | **930** | **280** | **-70%** |

**Tests requis**: 15 tests unitaires

---

## Résumé Phase 5

### Changements globaux

| Fichier | LOC modifiées | Tests requis | Risque | Restriction fonctionnelle |
|---------|---------------|--------------|--------|---------------------------|
| `teammates_service.py` | ~150 | 15 | 🔴 HAUTE | ✅ Profils coéquipiers désactivés |
| `_match_queries.py` | -140 (simplification) | 12 | 🟡 MOYENNE | Non |
| `_roster_loader.py` | -650 (simplification) | 15 | 🟡 MOYENNE | Non |
| **Total** | **-640** | **42** | - | - |

### Bénéfices

- ✅ **70% réduction de code** dans roster_loader
- ✅ **Suppression de 15 fallbacks** sources de bugs
- ✅ **Architecture simplifiée**: shared autoritaire
- ✅ **Performance**: Moins de requêtes conditionnelles

### Risques

- ⚠️ **Restriction UX**: Profils radar coéquipiers désactivés (personal_score_awards locale)
- ⚠️ **Dépendance migration**: Erreurs explicites si shared absent
- ⚠️ **Tests critiques**: 42 tests à écrire/adapter

### Recommandations

1. **Migration `personal_score_awards` vers shared** (optionnelle, Phase 11)
   - Restaure fonctionnalité profils coéquipiers
   - Volumineuse: ~50k lignes/joueur

2. **Indexes shared DB** (critique)
   - `shared.match_participants (xuid, match_id)`
   - `shared.medals_earned (match_id, xuid, medal_name_id)`
   - `shared.highlight_events (match_id, killer_xuid, victim_xuid)`

3. **Documentation restrictions**
   - Documenter dans UI que profils coéquipiers désactivés
   - CHANGELOG: "Profils radar: joueur principal uniquement (V5)"

---
