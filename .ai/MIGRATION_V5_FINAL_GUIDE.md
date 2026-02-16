# Guide de Migration V5 Finale - Consolidation Complète

> **Document principal** : Ce guide consolide le fichier BUGFIX_V5_2026-02-15.md avec des analyses approfondies pour chaque phase.
> 
> **Statut** : Phases 0-4 ✅ complétées | Phases 5-10 📋 en préparation

---

## 📚 Structure Documentaire

### Documents à utiliser selon votre besoin :

| Document | Quand l'utiliser | Contenu |
|----------|------------------|---------|
| **🎯 `.ai/MIGRATION_V5_FINAL_GUIDE.md`** (CE FICHIER) | **Travail quotidien principal** | Guide consolidé avec analyses approfondies, plan d'exécution, checklists |
| `BUGFIX_V5_2026-02-15.md` | Référence historique, détails techniques | Document source avec tous les bugs identifiés, corrections, inventaire exhaustif |
| `.ai/PHASES_5_10_ANALYSES.md` | Analyses détaillées des fichiers | Analyses approfondies par fichier pour phases 5-10 |
| `.ai/MIGRATION_V5_FINAL_CHECKLIST.md` | Suivi quotidien | Checklist interactive avec statuts, validation |

### Workflow recommandé :

```
Jour de travail
    ↓
1. Ouvrir: .ai/MIGRATION_V5_FINAL_GUIDE.md (CE FICHIER)
    ↓
2. Consulter la phase en cours
    ↓
3. Suivre les étapes détaillées
    ↓
4. Si besoin de détails techniques → BUGFIX_V5_2026-02-15.md
    ↓
5. Si besoin d'analyse code → .ai/PHASES_5_10_ANALYSES.md
    ↓
6. Cocher progression → .ai/MIGRATION_V5_FINAL_CHECKLIST.md
```

**Réponse à votre question** : 
- ✅ **Vous pouvez suivre principalement CE FICHIER** pour votre travail quotidien
- 📖 **Gardez le BUGFIX** comme référence technique détaillée si besoin
- 🔍 **Les analyses approfondies** sont dans `.ai/PHASES_5_10_ANALYSES.md`

---

## 📊 Vue d'ensemble de la migration

### Objectif

Finaliser la migration V5 pour éliminer toutes les lectures de tables locales et centraliser les données dans `shared_matches.duckdb`.

### Architecture Cible

```
Avant (V4) :                          Après (V5 finale) :
┌─────────────────────┐              ┌─────────────────────┐
│ Player DB (30MB)    │              │ Player DB (4MB)     │
│ ├─ match_stats      │              │ ├─ enrichment       │
│ ├─ medals_earned    │              │ ├─ citations        │
│ ├─ highlight_events │  ────────>   │ ├─ career_prog      │
│ ├─ player_match_stats              │ ├─ antagonists      │
│ ├─ xuid_aliases     │              │ └─ media_*          │
│ └─ teammates_agg    │              └─────────────────────┐
└─────────────────────┘                                     │
                                      ┌─────────────────────┘
                                      │
┌─────────────────────┐              ┌─────────────────────┐
│ Shared DB           │              │ Shared DB (central) │
│ ├─ match_registry   │              │ ├─ match_registry   │
│ ├─ match_participants (15 cols)    │ ├─ match_participants (31 cols)
│ ├─ medals_earned    │  ────────>   │ ├─ medals_earned    │
│ ├─ highlight_events │              │ ├─ highlight_events │
│ └─ xuid_aliases     │              │ ├─ xuid_aliases     │
└─────────────────────┘              │ └─ killer_victim_pairs
                                      └─────────────────────┘
```

**Changement clé** : Les 16 nouvelles colonnes dans `match_participants` (9 stats étendues + 7 MMR) permettent de tout centraliser.

---

## ✅ État d'avancement

| Phase | Statut | Durée estimée | Complétée le |
|-------|--------|---------------|--------------|
| Phase 0 | ✅ | 1 jour | [Date] |
| Phase 1 | ✅ | 1 jour | [Date] |
| Phase 2 | ✅ | 1 jour | [Date] |
| Phase 3 | ✅ | 1 jour | [Date] |
| Phase 4 | ✅ | 1 jour | [Date] |
| Phase 5 | 📋 Préparation | 1 jour | - |
| Phase 6 | ⏸️ | 1 jour | - |
| Phase 7 | ⏸️ | 1 jour | - |
| Phase 8 | ⏸️ | 1 jour | - |
| Phase 9 | ⏸️ | 1 jour | - |
| Phase 10 | ⏸️ | 1 jour | - |

**Progression** : 45% (5/11 phases)

---

## 📝 Phases 0-4 : Résumé des Réalisations

### Phase 0 ✅ : Pré-requis et analyse
- Backup complet des DBs
- Baseline de tests sauvegardée
- Analyse de contexte effectuée
- Script de migration créé et testé en dry-run

### Phase 1 ✅ : ALTER TABLE shared + backfill données
- 16 colonnes ajoutées à `shared.match_participants`
- Tables `player_match_enrichment` créées pour chaque joueur
- Backfill des données locales → shared effectué
- Couverture vérifiée à 100%

### Phase 2 ✅ : Transformers, modèles, batch_insert
- `MatchParticipantRow` étendu avec 31 champs
- `extract_participants()` extrait les 16 nouvelles colonnes
- `transform_all_skill_stats()` créé pour le MMR
- Tests unitaires passent

### Phase 3 ✅ : Sync engine - stop dual write
- Sync n'écrit plus dans les tables locales
- Écriture uniquement dans shared + `player_match_enrichment`
- Schémas player DB allégés
- Sync de test validé

### Phase 4 ✅ : Scripts backfill
- Tous les scripts backfill redirigés vers shared
- Mode `--participants-enrich` créé
- Fallbacks locaux supprimés
- Scripts testés

---

## 🚀 Phase 5 : Services et repositories (partie 1)

> **Objectif** : Adapter les services critiques pour lire depuis shared au lieu des DBs individuelles.
> 
> **Durée estimée** : 1 jour (6-8h)
> 
> **Priorité** : 🔴 CRITIQUE - Services utilisés dans toutes les pages UI

### 5.1 Vue d'ensemble

Cette phase s'attaque aux services qui font le pont entre les repositories et l'UI. Ces services sont critiques car ils sont utilisés partout dans l'application.

**Fichiers impactés** :
- `src/data/services/teammates_service.py` (250 lignes)
- `src/data/repositories/_match_queries.py` (1000+ lignes)
- `src/data/repositories/_roster_loader.py` (300+ lignes)

### 5.2 Analyse détaillée : `teammates_service.py`

**État actuel** :
- Lit les stats des coéquipiers depuis leurs DBs individuelles
- Utilise le système de fichiers (`data/players/{gamertag}/stats.duckdb`)
- Charge des DBs entières en mémoire via cache

**Problèmes identifiés** :
1. ❌ **Architecture multi-DB** : Lit 20+ DBs individuelles pour afficher les coéquipiers
2. ❌ **Dépendance filesystem** : Utilise `Path` pour localiser les DBs
3. ❌ **Cache inadapté** : `load_df_optimized()` charge DB entière
4. ❌ **Données dupliquées** : `medals_earned`, `highlight_events` lus localement ET dans shared

**Fonctions à migrer** :

#### 5.2.1 `load_teammate_stats()` ⚠️ RÉÉCRITURE COMPLÈTE

**Actuellement** :
```python
def load_teammate_stats(
    teammate_gamertag: str,
    match_ids: list[str],
    reference_db_path: Path,
) -> TeammateStats:
    """Charge les stats depuis la DB du coéquipier."""
    teammate_db = base_dir / teammate_gamertag / "stats.duckdb"
    df = load_df_optimized(str(teammate_db))  # Charge DB entière
    filtered = df.filter(pl.col("match_id").is_in(match_ids))
    return TeammateStats(df=filtered, gamertag=teammate_gamertag)
```

**Après migration** :
```python
def load_teammate_stats(
    teammate_xuid: str,  # ⚠️ Changement : xuid au lieu de gamertag
    match_ids: list[str],
    shared_db_path: Path,  # ⚠️ Changement : shared au lieu de player DB
) -> TeammateStats:
    """Charge les stats depuis shared.match_participants."""
    conn = duckdb.connect(str(shared_db_path), read_only=True)
    
    # Requête ciblée au lieu de charger DB entière
    query = """
        SELECT 
            mp.*,
            xa.gamertag
        FROM shared.match_participants mp
        LEFT JOIN shared.xuid_aliases xa ON mp.xuid = xa.xuid
        WHERE mp.xuid = ?
          AND mp.match_id IN (SELECT unnest(?))
        ORDER BY mp.match_id
    """
    
    df = pl.from_arrow(
        conn.execute(query, [teammate_xuid, match_ids]).fetch_arrow_table()
    )
    conn.close()
    
    gamertag = df["gamertag"][0] if len(df) > 0 else "Unknown"
    return TeammateStats(df=df, gamertag=gamertag)
```

**Impact** :
- ✅ Ne lit plus les DBs individuelles
- ✅ Requête ciblée (pas de chargement complet)
- ✅ Utilise xuid (stable) au lieu de gamertag
- ⚠️ **BREAKING CHANGE** : Signature modifiée (`xuid` au lieu de `gamertag`)

**Appelants à mettre à jour** :
- `src/ui/pages/teammates.py::_load_teammate_stats_from_own_db()` (ligne 51)
- Passer xuid résolu via `shared.xuid_aliases` au lieu de gamertag

#### 5.2.2 `enrich_series_with_perfect_kills()` 🟡 MODIFICATION MOYENNE

**Actuellement** :
- Utilise déjà `DuckDBRepository.count_perfect_kills_by_match()`
- Mais fallback sur DB locale si shared indisponible

**Après migration** :
```python
def enrich_series_with_perfect_kills(
    series: list[tuple[str, pl.DataFrame]],
    shared_db_path: Path,  # ⚠️ Plus de fallback local
) -> list[tuple[str, pl.DataFrame]]:
    """Enrichit avec perfect_kills depuis shared.medals_earned."""
    repo = DuckDBRepository(shared_db_path, xuid=None)
    
    for name, df in series:
        if df.is_empty():
            continue
        
        xuid = df["xuid"][0]  # Supposé constant dans le DF
        
        # Lire depuis shared uniquement (pas de fallback)
        perfect_kills = repo.count_perfect_kills_by_match(
            match_ids=df["match_id"].to_list(),
            xuid=xuid
        )
        
        df = df.with_columns([
            pl.col("match_id").map_elements(
                lambda mid: perfect_kills.get(mid, 0),
                return_dtype=pl.Int32
            ).alias("perfect_kills")
        ])
        
        series[i] = (name, df)
    
    return series
```

**Impact** :
- ✅ Suppression fallback local
- ✅ Toujours lire depuis shared
- ⚠️ Échouera si shared indisponible (comportement voulu)

#### 5.2.3 `compute_participation_profiles()` 🟡 MODIFICATION MOYENNE

**Actuellement** :
- Lit `personal_score_awards` depuis DBs individuelles

**Problème** :
- `personal_score_awards` n'est **PAS** dans shared
- Reste dans les DBs player (table conservée)

**Après migration** :
```python
def compute_participation_profiles(
    players_data: list[tuple[str, pl.DataFrame]],
    player_db_paths: dict[str, Path],  # ⚠️ Carte xuid → DB path
    shared_match_ids: list[str],
) -> dict[str, dict[str, float]]:
    """Calcule les profils radar depuis player DBs (personal_score_awards)."""
    profiles = {}
    
    for name, df in players_data:
        if df.is_empty():
            continue
        
        xuid = df["xuid"][0]
        player_db = player_db_paths.get(xuid)
        
        if not player_db or not player_db.exists():
            continue
        
        # personal_score_awards reste dans player DB
        repo = DuckDBRepository(player_db, xuid=xuid)
        awards = repo.load_personal_score_awards_as_polars(
            match_ids=shared_match_ids
        )
        
        # Calcul du profil (inchangé)
        profile = compute_participation_profile(awards, df)
        profiles[name] = profile
    
    return profiles
```

**Impact** :
- ✅ `personal_score_awards` reste local (table conservée)
- ⚠️ Nécessite un mapping xuid → player_db_path
- ℹ️ Pas de changement majeur (table déjà locale)

#### 5.2.4 `load_impact_data()` 🟢 MODIFICATION SIMPLE

**Actuellement** :
- Fallback local → shared pour `highlight_events`

**Après migration** :
```python
def load_impact_data(
    shared_db_path: Path,  # ⚠️ Plus de player db_path
    xuid: str,
    match_ids: list[str],
    friend_xuids: list[str],
) -> dict[str, Any]:
    """Charge highlight_events depuis shared uniquement."""
    conn = duckdb.connect(str(shared_db_path), read_only=True)
    
    # Schéma V5 : killer_xuid, victim_xuid
    query = """
        SELECT 
            he.*,
            mr.outcome
        FROM shared.highlight_events he
        JOIN shared.match_registry mr ON he.match_id = mr.match_id
        WHERE he.match_id IN (SELECT unnest(?))
          AND (he.killer_xuid = ? OR he.victim_xuid = ?)
    """
    
    df = pl.from_arrow(
        conn.execute(query, [match_ids, xuid, xuid]).fetch_arrow_table()
    )
    conn.close()
    
    # Traitement via friends_impact (inchangé)
    from src.analysis.friends_impact import get_all_impact_events
    return get_all_impact_events(df, xuid, friend_xuids)
```

**Impact** :
- ✅ Suppression fallback local
- ✅ Schéma V5 (killer_xuid/victim_xuid)
- ✅ Requête optimisée avec JOIN

### 5.3 Analyse détaillée : `_match_queries.py`

**État actuel** :
- 1000+ lignes, cœur du repository
- Gère toutes les requêtes de matchs

**Problèmes identifiés** :
1. ❌ `_get_match_source()` génère des sous-requêtes complexes avec fallbacks
2. ❌ `load_match_mmr_batch()` lit depuis `player_match_stats` (table supprimée)
3. ❌ `get_match_count()` compte dans table locale

**Fonctions à migrer** :

#### 5.3.1 `_get_match_source()` ⚠️ SIMPLIFICATION MAJEURE

**Actuellement** (lignes ~100-200) :
```python
def _get_match_source(self, filters: dict) -> str:
    """Génère une sous-requête complexe avec fallbacks."""
    if self.version == "v5":
        # Fallback local → shared
        return """
            (SELECT * FROM match_stats
             UNION ALL
             SELECT * FROM shared.match_participants
             WHERE xuid = '{xuid}') AS match_stats
        """
    else:
        return "match_stats"
```

**Après migration** :
```python
def _get_match_source(self) -> str:
    """Retourne la source shared uniquement (V5)."""
    # Plus de fallback, plus de sous-requêtes
    # On lit directement depuis shared
    return "shared.match_participants"
```

**Impact** :
- ✅ Simplification drastique (200 lignes → 10 lignes)
- ✅ Plus de dual read
- ✅ Performances améliorées (pas de UNION ALL)

#### 5.3.2 `load_match_mmr_batch()` 🟡 MODIFICATION

**Actuellement** :
```python
def load_match_mmr_batch(self, match_ids: list[str]) -> pl.DataFrame:
    """Charge MMR depuis player_match_stats."""
    conn = self._get_connection()
    query = "SELECT * FROM player_match_stats WHERE match_id IN (?)"
    return pl.from_arrow(conn.execute(query, [match_ids]).fetch_arrow_table())
```

**Après migration** :
```python
def load_match_mmr_batch(self, match_ids: list[str]) -> pl.DataFrame:
    """Charge MMR depuis shared.match_participants."""
    shared_conn = self._get_shared_connection()  # Toujours shared
    
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
    """
    
    return pl.from_arrow(
        shared_conn.execute(query, [match_ids, str(self.xuid)]).fetch_arrow_table()
    )
```

**Impact** :
- ✅ Lit depuis shared (colonnes MMR ajoutées en Phase 1)
- ✅ Filtre par xuid du joueur principal
- ℹ️ Retourne mêmes colonnes (compatibilité)

#### 5.3.3 `get_match_count()` 🟢 MODIFICATION SIMPLE

**Actuellement** :
```python
def get_match_count(self) -> int:
    """Compte les matchs dans match_stats locale."""
    conn = self._get_connection()
    result = conn.execute("SELECT COUNT(*) FROM match_stats").fetchone()
    return result[0]
```

**Après migration** :
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
    return result[0]
```

**Impact** :
- ✅ Compte dans shared
- ✅ DISTINCT pour éviter doublons
- ✅ Filtre par xuid

### 5.4 Analyse détaillée : `_roster_loader.py`

**État actuel** :
- ~300 lignes
- ~15 fallbacks `local → shared`

**Problèmes identifiés** :
1. ❌ Tous les fallbacks doivent être supprimés
2. ❌ Lit tables locales en premier, shared en second

**Stratégie de migration** :

#### 5.4.1 Pattern actuel à éliminer

```python
# ❌ AVANT - Pattern répété 15 fois
def load_data(self):
    conn = self._get_connection()
    try:
        result = conn.execute("SELECT * FROM local_table").fetchall()
    except:
        # Fallback shared
        shared_conn = self._get_shared_connection()
        result = shared_conn.execute("SELECT * FROM shared.table").fetchall()
    return result
```

#### 5.4.2 Pattern après migration

```python
# ✅ APRÈS - Toujours shared
def load_data(self):
    shared_conn = self._get_shared_connection()
    result = shared_conn.execute("SELECT * FROM shared.table").fetchall()
    return result
```

#### 5.4.3 Méthodes à modifier

Liste des ~15 méthodes avec fallback à supprimer :
1. `load_player_roster()` - ligne 50
2. `load_enemy_roster()` - ligne 80
3. `load_teammate_names()` - ligne 110
4. `get_player_aliases()` - ligne 140
5. `get_team_composition()` - ligne 170
6. `load_participant_list()` - ligne 200
7. (etc. - voir BUGFIX §6.2 pour liste complète)

**Impact global** :
- ✅ Simplification ~150 lignes (fallbacks supprimés)
- ✅ Code plus maintenable
- ✅ Performances meilleures (1 seul accès DB au lieu de 2)

### 5.5 Plan d'implémentation Phase 5

#### Étape 1 : `teammates_service.py` (2-3h)

1. ✅ Modifier `load_teammate_stats()` - signature + implémentation
2. ✅ Modifier `enrich_series_with_perfect_kills()` - supprimer fallback
3. ✅ Modifier `compute_participation_profiles()` - clarifier player DB usage
4. ✅ Modifier `load_impact_data()` - shared uniquement
5. ✅ Mettre à jour les appelants dans `src/ui/pages/teammates.py`

#### Étape 2 : `_match_queries.py` (2-3h)

1. ✅ Simplifier `_get_match_source()` - retourner shared directement
2. ✅ Modifier `load_match_mmr_batch()` - lire colonnes MMR de shared
3. ✅ Modifier `get_match_count()` - compter dans shared
4. ✅ Tester avec `pytest tests/test_repositories.py -v`

#### Étape 3 : `_roster_loader.py` (1-2h)

1. ✅ Identifier les 15 fallbacks (grep "try.*except")
2. ✅ Supprimer chaque fallback, garder uniquement shared
3. ✅ Tester avec `pytest tests/test_roster.py -v`

#### Étape 4 : Validation (1h)

1. ✅ Tests unitaires : `pytest tests/test_teammates_service.py -v`
2. ✅ Tests repositories : `pytest tests/test_repositories.py -v`
3. ✅ Test manuel : Ouvrir page "Analyse coéquipiers" dans l'app

### 5.6 Tests à créer/mettre à jour

```python
# tests/test_teammates_service_v5.py

def test_load_teammate_stats_from_shared():
    """Vérifie que load_teammate_stats lit depuis shared."""
    teammate_xuid = "xuid(12345)"
    match_ids = ["match1", "match2"]
    shared_db = Path("data/warehouse/shared_matches.duckdb")
    
    stats = load_teammate_stats(teammate_xuid, match_ids, shared_db)
    
    assert not stats.is_empty
    assert stats.df["xuid"].unique()[0] == teammate_xuid
    assert set(stats.df["match_id"].unique()) == set(match_ids)


def test_enrich_perfect_kills_no_fallback():
    """Vérifie qu'enrich ne fallback pas sur local."""
    series = [("Player1", sample_df)]
    shared_db = Path("data/warehouse/shared_matches.duckdb")
    
    # Supprimer la player DB pour forcer shared
    # Ne doit PAS échouer
    result = enrich_series_with_perfect_kills(series, shared_db)
    
    assert "perfect_kills" in result[0][1].columns


def test_match_queries_use_shared_only():
    """Vérifie que _get_match_source retourne shared."""
    repo = DuckDBRepository(shared_db_path, xuid="xuid(123)")
    
    source = repo._get_match_source()
    
    assert source == "shared.match_participants"
    assert "UNION" not in source
    assert "match_stats" not in source
```

### 5.7 Checklist Phase 5

- [ ] **teammates_service.py**
  - [ ] `load_teammate_stats()` - signature + implémentation shared
  - [ ] `enrich_series_with_perfect_kills()` - supprimer fallback
  - [ ] `compute_participation_profiles()` - documenter player DB
  - [ ] `load_impact_data()` - shared uniquement
  - [ ] Mettre à jour appelants UI

- [ ] **_match_queries.py**
  - [ ] `_get_match_source()` - simplification
  - [ ] `load_match_mmr_batch()` - colonnes MMR shared
  - [ ] `get_match_count()` - COUNT shared

- [ ] **_roster_loader.py**
  - [ ] Identifier 15 fallbacks
  - [ ] Supprimer tous les fallbacks
  - [ ] Tester chaque méthode

- [ ] **Tests**
  - [ ] Créer `test_teammates_service_v5.py`
  - [ ] Mettre à jour `test_repositories.py`
  - [ ] Créer `test_roster_v5.py`

- [ ] **Validation**
  - [ ] Tests unitaires passent
  - [ ] Page "Coéquipiers" fonctionne
  - [ ] Pas d'erreurs console

### 5.8 Risques et mitigations

| Risque | Impact | Probabilité | Mitigation |
|--------|--------|-------------|------------|
| Breaking change signatures | 🔴 Élevé | Haute | Mettre à jour tous les appelants immédiatement |
| Données manquantes dans shared | 🔴 Élevé | Moyenne | Vérifier couverture 100% avant (Phase 1) |
| Performance dégradée | 🟡 Moyen | Faible | Requêtes ciblées au lieu de UNION ALL |
| Tests cassés | 🟡 Moyen | Haute | Mettre à jour tests en même temps que code |

---

## 🚀 Phase 6 : Repositories (partie 2) + UI critique

> **Objectif** : Finir les repositories et adapter les pages UI critiques.
> 
> **Durée estimée** : 1 jour (6-8h)
> 
> **Priorité** : 🔴 CRITIQUE - Pages utilisées quotidiennement

### 6.1 Vue d'ensemble

Cette phase finalise les repositories (partie 2) et adapte les pages UI les plus critiques.

**Fichiers impactés** :
- `src/data/repositories/duckdb_repo.py` (8 méthodes)
- `src/data/repositories/_materialized_views.py` (4 requêtes)
- `src/ui/pages/teammates_impact.py`
- `src/ui/pages/objective_analysis.py`

### 6.2 Détails disponibles

Voir `.ai/PHASES_5_10_ANALYSES.md` section "Phase 6" pour analyses détaillées.

### 6.3 Checklist Phase 6

- [ ] **duckdb_repo.py** (8 méthodes)
  - [ ] `load_top_medals()` - shared.medals_earned
  - [ ] `load_match_medals()` - shared.medals_earned
  - [ ] `count_medal_by_match()` - shared.medals_earned
  - [ ] `load_first_event_times()` - shared.highlight_events
  - [ ] `load_highlight_events()` - shared.highlight_events
  - [ ] `list_other_player_xuids()` - shared.match_participants
  - [ ] `get_storage_info()` - tables conservées
  - [ ] `get_match_session_info()` - player_match_enrichment

- [ ] **_materialized_views.py**
  - [ ] 4 requêtes FROM match_stats → shared

- [ ] **UI critiques**
  - [ ] teammates_impact.py - shared.highlight_events
  - [ ] objective_analysis.py - shared

---

## 🚀 Phase 7 : UI complète + filtres

### 7.1 Checklist Phase 7

- [ ] **Pages UI**
  - [ ] citations.py - requêtes shared
  - [ ] personal_performance.py - shared
  - [ ] media_library.py - shared.match_registry

- [ ] **Filtres**
  - [ ] filters.py - Polars natif
  - [ ] filters_render.py - type consistency
  - [ ] checkbox_filter.py - ne pas vider sélections

- [ ] **Scripts**
  - [ ] sync.py - supprimer teammates_aggregate

---

## 🚀 Phase 8 : Modules secondaires

### 8.1 Checklist Phase 8

- [ ] **Analyse**
  - [ ] killer_victim.py - TypedDict corrections
  - [ ] citations/engine.py - shared + colonnes V5

- [ ] **Media et viz**
  - [ ] media_indexer.py - shared.match_registry
  - [ ] participation_radar.py - shared

- [ ] **Utilitaires**
  - [ ] launcher.py - discovery shared
  - [ ] multiplayer.py - list_players shared
  - [ ] cache_loaders/filters - shared
  - [ ] aliases.py - shared.xuid_aliases
  - [ ] data_loader.py - shared
  - [ ] xuid.py - shared

---

## 🚀 Phase 9 : Validation + cleanup brutal

### 9.1 Checklist Phase 9

- [ ] **Tests complétude**
  - [ ] Tous tests anti-régression passent
  - [ ] Navigation complète app
  - [ ] Sync de test

- [ ] **Cleanup brutal**
  - [ ] Backup avant cleanup
  - [ ] `cleanup_player_dbs_v5.py --all`
  - [ ] Vérifier app fonctionne après
  - [ ] Identifier/corriger code résiduel

---

## 🚀 Phase 10 : Documentation

### 10.1 Checklist Phase 10

- [ ] **Architecture (P0)**
  - [ ] ARCHITECTURE_V5.md
  - [ ] SHARED_MATCHES_SCHEMA.md
  - [ ] SQL_SCHEMA.md
  - [ ] DATA_ARCHITECTURE.md

- [ ] **Guides (P1-P2)**
  - [ ] SYNC_GUIDE.md
  - [ ] CLEANUP_V5.md
  - [ ] CLEANUP_V5_QUICKSTART.md
  - [ ] COMMANDS.md
  - [ ] BACKUP_RESTORE.md

- [ ] **Docs IA (P0-P1)**
  - [ ] CLAUDE.md
  - [ ] .github/copilot-instructions.md
  - [ ] .ai/project_map.md
  - [ ] .ai/data_lineage.md
  - [ ] .ai/thought_log.md

---

## 📋 Ressources

- **Document source** : `BUGFIX_V5_2026-02-15.md` (détails techniques)
- **Analyses code** : `.ai/PHASES_5_10_ANALYSES.md` (analyses approfondies)
- **Checklist suivi** : `.ai/MIGRATION_V5_FINAL_CHECKLIST.md` (progression)
- **Architecture** : `docs/ARCHITECTURE_V5.md` (schémas)

---

## 🎯 Prochaine étape

**Commencer Phase 5** :
1. Ouvrir `.ai/PHASES_5_10_ANALYSES.md` section "Phase 5"
2. Lire l'analyse détaillée de `teammates_service.py`
3. Suivre le plan d'implémentation étape par étape
4. Cocher les items dans `.ai/MIGRATION_V5_FINAL_CHECKLIST.md`

**Bon courage !** 🚀
