# Analyses Approfondies - Phases 6 à 10

> **Document technique complémentaire** : Analyses détaillées phases 6-10
> 
> **Audience** : Développeurs implémentant les modifications
> 
> **Mis à jour** : 2026-02-16

**Note** : Ce document complète `.ai/PHASES_5_10_ANALYSES.md` qui contient l'analyse complète de la Phase 5.

---

## Phase 6 : Repositories (partie 2) + UI critique

### Vue d'ensemble

Finalisation repositories (8 méthodes duckdb_repo.py) + adaptation pages UI critiques.

**Fichiers impactés** :
- `src/data/repositories/duckdb_repo.py` (8 méthodes)
- `src/ui/pages/teammates_impact.py`
- `src/ui/pages/objective_analysis.py`
- `src/data/repositories/_materialized_views.py`

### Résumé 8 Méthodes `duckdb_repo.py`

| Méthode | Action | Lignes |  
|---------|--------|--------|
| `load_top_medals()` | Supprimer fallback V4 | 361-415 |
| `load_match_medals()` | Supprimer fallback V4 | 417-445 |
| `count_medal_by_match()` | Supprimer fallback V4 | 447-501 |
| `load_first_event_times()` | Supprimer fallback V4 | 524-590 |
| `load_highlight_events()` | Supprimer fallback V4 | 1175-1242 |
| `list_other_player_xuids()` | ✅ GARDER fallbacks (4 sources) | 1244-1316 |
| `get_storage_info()` | Ajouter shared counts | 680-709 |
| `get_match_session_info()` | Ajouter shared fallback | 1318-1345 |

**Pattern répétitif** : 5 méthodes suivent le même pattern de suppression fallback (~150 lignes économisées).

### UI Critiques

#### teammates_impact.py
- ❌ Accès direct `conn._get_connection()` 
- ✅ Migrer vers `repo._get_shared_connection()`
- Tables : `highlight_events`, `match_stats` → shared

#### objective_analysis.py  
- ✅ Bon pattern (`repo.query_df()`)
- ℹ️ `personal_score_awards` reste local (conservée)
- Tables : `match_stats` → shared.match_participants JOIN match_registry

---

## Phase 7 : UI complète + filtres

### Vue d'ensemble

Finaliser pages UI + corriger problèmes filtres (Polars/Pandas mixing, type issues).

**Fichiers impactés** :
- `src/ui/pages/citations.py`
- `src/ui/pages/media_library.py`
- `src/app/filters.py`
- `src/app/filters_render.py`
- `src/ui/components/checkbox_filter.py`
- `scripts/sync.py`

### Découvertes Critiques

#### 1. **personal_performance.py** : 🚫 **Fichier NON TROUVÉ**
- Retirer de la scope Phase 7 ou créer si nécessaire

#### 2. **Filtres : Polars utilisé (pas Pandas)**
- ✅ Migration déjà faite
- ⚠️ Reste problèmes de cohérence

#### 3. **Problèmes Type & UX**

| Fichier | Ligne | Problème | Sévérité |
|---------|-------|----------|----------|
| `filters.py` | 370 | `.empty` (Pandas) au lieu `.is_empty()` (Polars) | 🔴 High |
| `filters_render.py` | 303 | Type hint retourne 3 mais code retourne 4 | 🟡 Medium |
| `checkbox_filter.py` | 189-191, 299-301 | "Aucun" vide sans confirmation | 🟡 Medium |

### Corrections Nécessaires

#### filters.py ligne 370
```python
# AVANT
if not base_s_ui.empty  # ❌ Pandas

# APRÈS  
if not base_s_ui.is_empty()  # ✅ Polars
```

#### filters_render.py ligne 303
```python
# AVANT
def _render_session_filter(...) -> tuple[int, list[str] | None, pl.DataFrame]:

# APRÈS
def _render_session_filter(...) -> tuple[int, list[str] | None, pl.DataFrame, tuple[str, ...] | None]:
```

#### checkbox_filter.py lignes 189-191
```python
# AVANT
if cols[1].button("✗ Aucun", key=f"{session_key}_none", width="stretch"):
    st.session_state[session_key] = set()  # ❌ Perte brutale
    st.rerun()

# APRÈS (avec confirmation)
if cols[1].button("✗ Aucun", key=f"{session_key}_none", width="stretch"):
    if st.session_state.get(f"{session_key}_confirm_clear"):
        st.session_state[session_key] = set()
        st.session_state[f"{session_key}_confirm_clear"] = False
        st.rerun()
    else:
        st.session_state[f"{session_key}_confirm_clear"] = True
        st.warning("⚠️ Confirmer : vider toutes les sélections ?")
```

---

## Phase 8 : Modules secondaires

### Vue d'ensemble

11 modules à auditer pour accès shared, TypedDict, fallbacks.

**Fichiers impactés** :
- `src/analysis/killer_victim.py` - TypedDict
- `src/analysis/citations/engine.py` - SQL shared
- `src/data/media_indexer.py`
- `src/visualization/participation_radar.py`
- `launcher.py`
- `src/ui/multiplayer.py`
- `src/ui/cache_loaders.py`
- `src/ui/cache_filters.py`
- `src/ui/aliases.py` 🔴
- `src/app/data_loader.py`
- `src/utils/xuid.py` 🔴

### Résumé Rapide

| Module | TypedDict OK | Shared Access | Action |
|--------|--------------|---------------|--------|
| killer_victim.py | ✅ dict `.get()` | N/A | ✅ Aucune |
| citations/engine.py | ✅ | Hybride | ⚠️ Ajouter fallback awards |
| media_indexer.py | N/A | Local seul | ✅ Aucune |
| participation_radar.py | N/A | Indépendant | ✅ Aucune |
| launcher.py | N/A | File system | ✅ Aucune |
| multiplayer.py | ✅ | Fallbacks | ⚠️ Ajouter shared xuid_aliases |
| cache_loaders.py | ✅ | Repository | ✅ Note ajout |
| cache_filters.py | N/A | Délégation | ✅ Aucune |
| **aliases.py** | N/A | **Local seul** | 🔴 **CRITIQUE** |
| data_loader.py | N/A | Délégation | ✅ Aucune |
| **xuid.py** | N/A | **Local seul** | 🔴 **CRITIQUE** |

### 🔴 Actions Critiques

#### 1. `src/ui/aliases.py` (lignes 56-78)

**Problème** : Lit xuid_aliases seulement depuis DB locale

**Solution** : Ajouter fallback shared metadata.duckdb

```python
def _load_aliases_from_duckdb_cached(db_path: str, mtime: float | None) -> dict[str, str]:
    con = duckdb.connect(db_path, read_only=True)
    
    # Try local first
    local_aliases = {}
    try:
        result = con.execute(
            "SELECT xuid, gamertag FROM xuid_aliases WHERE gamertag IS NOT NULL"
        ).fetchall()
        local_aliases = {str(row[0]).strip(): str(row[1]).strip() for row in result}
    except:
        pass
    
    # Try shared metadata as fallback
    shared_aliases = {}
    try:
        metadata_path = Path(db_path).parent.parent / "warehouse" / "metadata.duckdb"
        if metadata_path.exists():
            con.execute(f"ATTACH '{metadata_path}' AS meta (READ_ONLY)")
            result = con.execute(
                "SELECT xuid, gamertag FROM meta.xuid_aliases WHERE gamertag IS NOT NULL"
            ).fetchall()
            shared_aliases = {str(row[0]).strip(): str(row[1]).strip() for row in result}
    except:
        pass
    
    # Merge: local precedence
    result = shared_aliases.copy()
    result.update(local_aliases)
    con.close()
    return result
```

#### 2. `src/utils/xuid.py` (lignes 158-172)

**Problème** : resolve_xuid_from_db() lit seulement local

**Solution** : Ajouter fallback shared metadata.duckdb

```python
def resolve_xuid_from_db(...) -> str | None:
    # ... existing logic ...
    
    if db_path.endswith(".duckdb"):
        try:
            conn = duckdb.connect(db_path, read_only=True)
            
            # Try local first
            result = conn.execute(
                "SELECT xuid FROM xuid_aliases WHERE LOWER(gamertag) = LOWER(?)",
                [p],
            ).fetchone()
            if result and result[0]:
                conn.close()
                return str(result[0])
            
            # Try shared metadata fallback
            metadata_path = Path(db_path).parent.parent / "warehouse" / "metadata.duckdb"
            if metadata_path.exists():
                conn.execute(f"ATTACH '{metadata_path}' AS meta (READ_ONLY)")
                result = conn.execute(
                    "SELECT xuid FROM meta.xuid_aliases WHERE LOWER(gamertag) = LOWER(?)",
                    [p],
                ).fetchone()
                if result and result[0]:
                    conn.close()
                    return str(result[0])
            
            conn.close()
        except:
            pass
```

#### 3. `src/analysis/citations/engine.py` (lignes 447-459)

**Problème** : personal_score_awards local seul

**Solution** : Ajouter fallback shared

```python
# Try shared.personal_score_awards first
try:
    if self._conn_has_shared(conn):
        rows = conn.execute(
            "SELECT award_name, SUM(award_count) FROM shared.personal_score_awards "
            "WHERE match_id = ? AND xuid = ? GROUP BY award_name",
            [match_id, self._xuid],
        ).fetchall()
        if rows:
            return rows
except:
    pass

# Fallback local
rows = conn.execute(
    "SELECT award_name, SUM(award_count) FROM personal_score_awards "
    "WHERE match_id = ? GROUP BY award_name",
    [match_id],
).fetchall()
```

---

## Phase 9 : Validation + cleanup brutal

### Vue d'ensemble

Validation complète + suppression brutale tables locales.

**Objectif** : Forcer détection code résiduel via erreurs "table introuvable".

### Étapes

1. **Tests de complétude** (§16 BUGFIX)
   - test_no_from_match_stats_in_src()
   - test_no_local_medals_earned_read()
   - test_no_local_highlight_events_read()
   - test_no_player_match_stats_in_src()
   - test_sync_engine_writes_player_match_enrichment()

2. **Tests manuels**
   - Navigation complète app
   - Tester toutes pages
   - Vérifier graphiques
   - Tester filtres/sessions

3. **Sync de test**
   ```bash
   python scripts/sync.py --player TestPlayer --delta --max-matches 50
   ```

4. **Cleanup brutal**
   ```bash
   python scripts/backup_player.py --all
   python scripts/cleanup_player_dbs_v5.py --all --dry-run --verbose
   python scripts/cleanup_player_dbs_v5.py --all --backup
   ```

5. **Validation post-cleanup**
   - Relancer app
   - Si erreur "table introuvable" → identifier code résiduel
   - Corriger immédiatement
   - Re-cleanup

### Tables Supprimées (8)

| Table | Raison |
|-------|--------|
| `match_stats` | → shared.match_participants + match_registry |
| `match_participants` | → shared.match_participants |
| `highlight_events` | → shared.highlight_events |
| `medals_earned` | → shared.medals_earned |
| `killer_victim_pairs` | → shared.killer_victim_pairs |
| `player_match_stats` | → colonnes MMR dans shared.match_participants |
| `xuid_aliases` | → shared.xuid_aliases |
| `teammates_aggregate` | → obsolète |

### Tables Conservées (9 + mv_*)

| Table | Raison |
|-------|--------|
| `player_match_enrichment` | Performance_score, session_id, is_with_friends |
| `personal_score_awards` | Awards objectifs (PersonalScores API) |
| `antagonists` | Rivalités killer/victim agrégées |
| `match_citations` | Citations calculées par match |
| `career_progression` | Historique rangs |
| `media_files` | Fichiers médias indexés |
| `media_match_associations` | Associations médias↔matchs |
| `sync_meta` | Métadonnées sync |
| `sessions` | Sessions groupées |

---

## Phase 10 : Documentation

### Vue d'ensemble

Mise à jour COMPLÈTE documentation pour refléter V5 finale.

**13 fichiers** à mettre à jour.

### Architecture (P0)

1. **docs/ARCHITECTURE_V5.md**
   - Schéma état cible (§17)
   - 31 colonnes match_participants
   - Stop dual write
   - 7 points critiques (§18.4)

2. **docs/SHARED_MATCHES_SCHEMA.md**
   - 16 colonnes étendues DDL
   - Schéma complet tables

3. **docs/SQL_SCHEMA.md**
   - Tables supprimées player DB
   - Tables conservées

4. **docs/DATA_ARCHITECTURE.md**
   - Flux sync → shared uniquement
   - Plus de dual write

### Guides (P1-P2)

5. **docs/SYNC_GUIDE.md**
   - Sync écrit shared + player_match_enrichment
   - Plus de match_stats locale

6. **docs/CLEANUP_V5.md**
   - 8 tables supprimées (liste complète)
   - --skip-coverage-check option

7. **docs/CLEANUP_V5_QUICKSTART.md**
   - Séquence : backup → dry-run → cleanup

8. **docs/COMMANDS.md**
   - --participants-enrich
   - Autres nouvelles options

9. **docs/BACKUP_RESTORE.md**
   - Backups plus petits (~4MB vs 30MB)

### Docs Internes IA (P0-P1)

10. **CLAUDE.md**
    - Tables player DB réduites
    - Colonnes match_participants 31
    - Coéquipiers depuis shared

11. **.github/copilot-instructions.md**
    - Mêmes mises à jour que CLAUDE.md

12. **.ai/project_map.md**
    - Flux sync actualisés
    - Tables par DB

13. **.ai/data_lineage.md**
    - API → shared uniquement
    - Plus de dual write

### Vérifications

```bash
# Rechercher références obsolètes
grep -r "match_stats" docs/ --exclude-dir=.git
grep -r "player_match_stats" docs/
grep -r "dual write" docs/

# Vérifier cohérence 31 colonnes
grep -r "match_participants" docs/ | grep -c "31"
```

### 7 Points Critiques à Documenter (§18.4)

1. **match_stats n'existe plus** dans player DBs
2. **player_match_stats n'existe plus** - MMR dans shared
3. **xuid_aliases est dans shared uniquement**
4. **player_match_enrichment est la SEULE table** match dans player DB
5. **Coéquipiers depuis shared**, pas leurs DBs individuelles
6. **Cleanup brutal intentionnel** : erreurs explicites
7. **Sync écrit dans player DBs** : enrichment + awards uniquement

### Livrables

- 13 fichiers mis à jour
- Cohérence vérifiée (grep)
- CHANGELOG.md à jour
- Tag Git : `v5.1.0-final`

---

## Récapitulatif Global

| Phase | Fichiers | Actions Principales | Durée |
|-------|----------|---------------------|-------|
| 6 | 4 | 8 méthodes repos + 2 UI critiques | 1j |
| 7 | 7 | 3 corrections type/UX + 4 UI | 1j |
| 8 | 11 | 3 ajouts fallback shared critiques | 1j |
| 9 | Tests + cleanup | Validation + suppression brutale | 1j |
| 10 | 13 docs | Mise à jour complète documentation | 1j |

**Total** : 5 jours (1 semaine)

**Complexité globale** : 🟡 Moyenne

**Risques** :
- 🔴 Phase 8 : 3 modules critiques (aliases, xuid, citations)
- 🟡 Phase 9 : Cleanup peut révéler code résiduel
- 🟢 Phases 6-7-10 : Modifications standards

---

## Tests Phase 6-10

```python
# tests/test_phases_6_10.py

def test_phase6_duckdb_repo_no_v4_fallbacks():
    """Vérifie que 5 méthodes n'ont plus de fallback V4."""
    from src.data.repositories import DuckDBRepository
    import inspect
    
    repo = DuckDBRepository(...)
    methods = [
        "load_top_medals",
        "load_match_medals", 
        "count_medal_by_match",
        "load_first_event_times",
        "load_highlight_events"
    ]
    
    for method_name in methods:
        method = getattr(repo, method_name)
        source = inspect.getsource(method)
        
        # Ne doit PAS avoir try/except avec "medals_earned" ou "highlight_events" local
        assert "FROM medals_earned" not in source or "FROM shared.medals_earned" in source
        assert "FROM highlight_events" not in source or "FROM shared.highlight_events" in source


def test_phase7_filters_use_polars():
    """Vérifie que filters.py utilise .is_empty() Polars."""
    import src.app.filters as filters
    import inspect
    
    source = inspect.getsource(filters)
    
    # Ne doit PAS utiliser .empty (Pandas)
    assert ".empty" not in source or "# Pandas" in source
    # DOIT utiliser .is_empty() (Polars)
    assert ".is_empty()" in source


def test_phase8_aliases_has_shared_fallback():
    """Vérifie que aliases.py a fallback shared."""
    import src.ui.aliases as aliases
    import inspect
    
    source = inspect.getsource(aliases)
    
    # DOIT avoir ATTACH metadata ou shared
    assert "ATTACH" in source or "metadata.duckdb" in source


def test_phase9_no_forbidden_tables_in_player_db():
    """Vérifie qu'aucune table interdite après cleanup."""
    import duckdb, json
    from pathlib import Path
    
    profiles = json.loads(Path('db_profiles.json').read_text())['profiles']
    forbidden = {'match_stats', 'player_match_stats', 'medals_earned', 'highlight_events'}
    
    for gt, p in profiles.items():
        db = Path(p['db_path'])
        if not db.exists():
            continue
        
        conn = duckdb.connect(str(db), read_only=True)
        tables = [r[0] for r in conn.execute(
            'SELECT table_name FROM information_schema.tables'
        ).fetchall()]
        
        found = set(tables) & forbidden
        conn.close()
        
        assert not found, f"{gt}: tables interdites {found}"
```

---

**Fin des analyses Phase 6-10**
