# Plan d'Éradication Architecture Legacy — LevelUp v5

> **Date de création** : 2026-02-16  
> **Version cible** : v5.1 (Pure DuckDB + Polars)  
> **Principe directeur** : Aucun fallback, aucune rétrocompatibilité  
> **Règle d'or** : V5 uniquement — Architecture Shared Matches DuckDB

---

## 🎯 Objectif Stratégique

Éliminer **minutieusement et définitivement** tous les vestiges des architectures obsolètes (SQLite, Pandas, modules legacy) pour atteindre un état de **pureté architecturale v5** :

- **100% DuckDB** (zéro SQLite en runtime)
- **100% Polars** (zéro Pandas dans le code métier)
- **Zéro backward compatibility** avec les architectures pré-v5
- **Zéro fallback** vers les anciens systèmes
- Code propre, testable, maintenable

---

## 📋 État des Lieux — Inventaire Complet des Vestiges Legacy

### 1. SQLite (Technologie Obsolète)

#### 1.1 Code Runtime avec Fallback SQLite (À ÉLIMINER)

| Fichier | Lignes | Type de Vestige | Impact |
|---------|--------|-----------------|--------|
| `src/data/query/engine.py` | 110-123 | Fallback runtime SQLite metadata.db | 🔴 CRITIQUE |
| `src/data/infrastructure/database/duckdb_engine.py` | 92-112 | Fallback SQLite dans _create_connection | 🔴 CRITIQUE |
| `src/utils/paths.py` | Multiples | Références `metadata.db` | 🟡 MOYEN |
| `src/ui/sync.py` | À auditer | Possibles références SQLite | 🟡 MOYEN |
| `src/ui/multiplayer.py` | À auditer | Possibles références SQLite | 🟡 MOYEN |
| `src/ai/rag.py` | À auditer | Possibles références SQLite | 🟢 FAIBLE |

**Total estimé** : ~7 fichiers avec traces SQLite runtime

#### 1.2 Scripts de Migration (À ARCHIVER)

Ces scripts sont **légitimes** pour la migration mais doivent être **clairement marqués** comme HORS SERVICE après v5.1 :

| Script | But | Action |
|--------|-----|--------|
| `scripts/migration/recover_from_sqlite.py` | Récupération données SQLite → DuckDB | ✅ Garder + bannière LEGACY |
| `scripts/migration/migrate_player_to_duckdb.py` | Migration joueur SQLite → DuckDB v4 | ✅ Garder + bannière LEGACY |
| `scripts/migration/migrate_all_to_duckdb.py` | Migration batch SQLite → DuckDB | ✅ Garder + bannière LEGACY |
| `scripts/migration/migrate_metadata_to_duckdb.py` | Migration metadata SQLite → DuckDB | ✅ Garder + bannière LEGACY |
| `scripts/migration/migrate_player_to_shared.py` | Migration v4 → v5 shared matches | ✅ Garder + bannière LEGACY |
| `scripts/_archive/migration_v5/*` | Archives migration v5 | ✅ OK (déjà archivé) |

**Total** : 5 scripts de migration + 1 dossier archivé

#### 1.3 Scripts Utilitaires SQLite (À SUPPRIMER ou PORTER)

| Script | Statut SQLite | Action Recommandée |
|--------|---------------|-------------------|
| `scripts/refetch_film_roster.py` | ❌ Utilise SQLite (`import sqlite3`) | ⚠️ DÉJÀ MARQUÉ LEGACY — à supprimer ou porter en DuckDB |
| `scripts/populate_antagonists.py` | À auditer | 🔍 Vérifier + décider |

---

### 2. Pandas (Technologie Obsolète)

#### 2.1 Code Métier avec Pandas (À MIGRER VERS POLARS)

| Fichier | Type | Criticité | Effort |
|---------|------|-----------|--------|
| `src/analysis/performance_score.py` | Logique métier | 🔴 CRITIQUE | 🔨 MOYEN |
| `src/data/services/win_loss_service.py` | Service | 🔴 CRITIQUE | 🔨 MOYEN |
| `src/ui/pages/objective_analysis.py` | Page UI | 🟡 MOYEN | 🔨 MOYEN |
| `src/ui/pages/match_view_helpers.py` | Helpers UI | 🟡 MOYEN | 🔨 FAIBLE |
| `src/ui/pages/win_loss.py` | Page UI | 🟡 MOYEN | 🔨 MOYEN |
| `src/ui/cache_filters.py` | Caching | 🟡 MOYEN | 🔨 FAIBLE |
| `src/ui/components/duckdb_analytics.py` | Composant | 🟡 MOYEN | 🔨 FAIBLE |

**Total** : 7 fichiers métier avec Pandas

#### 2.2 Couche de Compatibilité (À CONSERVER TEMPORAIREMENT)

| Fichier | But | Décision |
|---------|-----|----------|
| `src/visualization/_compat.py` | Conversions Polars↔Pandas pour Plotly/Streamlit | ✅ GARDER (nécessaire aux frontières) |
| `src/data/repositories/_arrow_bridge.py` | Bridge Arrow/Pandas | ✅ GARDER (nécessaire) |
| `src/data/integration/streamlit_bridge.py` | Bridge Streamlit | ✅ GARDER (nécessaire) |

**Justification** : Plotly et Streamlit n'acceptent que Pandas en entrée. La conversion `.to_pandas()` à la frontière est **acceptable** tant qu'elle reste confinée à ces modules bridge.

#### 2.3 Visualisations avec Pandas (À REFACTORISER)

| Fichier | Usage Pandas | Action |
|---------|--------------|--------|
| `src/visualization/participation_charts.py` | Conversion Pandas | 🔄 Utiliser `_compat.to_pandas_for_plotly()` |
| `src/visualization/distributions.py` | Conversion Pandas | 🔄 Utiliser `_compat.to_pandas_for_plotly()` |

**Total** : 2 modules de visualisation

---

### 3. Modules et Architectures Obsolètes

#### 3.1 Modules Legacy Déjà Supprimés (v4.1) ✅

Ces modules ont été éliminés lors de la migration v4.1 :

- ❌ `src/db/loaders.py` — SUPPRIMÉ
- ❌ `src/db/loaders_cached.py` — SUPPRIMÉ
- ❌ `src/data/repositories/legacy.py` — SUPPRIMÉ
- ❌ `src/data/repositories/shadow.py` — SUPPRIMÉ
- ❌ `src/data/repositories/hybrid.py` — SUPPRIMÉ

**Statut** : ✅ Aucun travail nécessaire (déjà fait)

#### 3.2 Scripts Archive (À NETTOYER/DOCUMENTER)

Le dossier `scripts/_archive/` contient **~60+ scripts** legacy de R&D, migration, ou expérimentaux.

**Actions nécessaires** :
1. Audit complet du contenu
2. Tri : supprimer / archiver / documenter
3. Créer un `README.md` d'archive explicite

---

### 4. Tables et Schémas DuckDB Obsolètes (Post-Migration v5)

#### 4.1 Tables Redondantes dans Player DBs (Déjà Nettoyables)

Depuis la migration v5 (shared matches), ces tables dans `data/players/{gamertag}/stats.duckdb` sont **redondantes** (script de cleanup existe déjà) :

| Table | Raison | Script Cleanup |
|-------|--------|----------------|
| `match_stats` | Remplacée par `_get_match_source()` + `shared.match_participants` | ✅ `cleanup_player_dbs_v5.py` |
| `match_participants` | Centralisée dans `shared.match_participants` | ✅ `cleanup_player_dbs_v5.py` |
| `highlight_events` | Centralisée dans `shared.highlight_events` | ✅ `cleanup_player_dbs_v5.py` |
| `medals_earned` | Centralisée dans `shared.medals_earned` | ✅ `cleanup_player_dbs_v5.py` |

**Statut** : ✅ Script `cleanup_player_dbs_v5.py` disponible (voir `docs/CLEANUP_V5.md`)

#### 4.2 Views de Compatibilité (À SUPPRIMER)

Views créées pendant la migration v4→v5 pour compatibilité transitoire :

| View | But | Action |
|------|-----|--------|
| `v_match_stats` | Redirection vers `shared.match_participants` | 🗑️ SUPPRIMER (option `--remove-compat-views`) |
| `v_match_participants` | Redirection vers `shared.match_participants` | 🗑️ SUPPRIMER |
| `v_highlight_events` | Redirection vers `shared.highlight_events` | 🗑️ SUPPRIMER |
| `v_medals_earned` | Redirection vers `shared.medals_earned` | 🗑️ SUPPRIMER |

**Outil** : `cleanup_player_dbs_v5.py --remove-compat-views`

---

### 5. Configuration et Chemins Legacy

#### 5.1 Références `.db` dans Configuration

| Fichier Config | Contenu à Vérifier |
|----------------|-------------------|
| `db_profiles.json` | Chemins vers player DBs (doit être `.duckdb`) |
| `app_settings.json` | Références à des chemins de DB |
| `.env.local.example` | Variables d'environnement |

**Action** : Audit manuel + validation qu'aucun chemin `.db` (SQLite) ne subsiste.

---

## 🏗️ Plan d'Exécution — Phases d'Éradication

### Phase 0 : Préparation & Sécurisation (PRIORITÉ CRITIQUE)

**Objectif** : Établir un filet de sécurité avant toute destruction.

#### Tâches

1. **Backup complet de production**
   ```bash
   # Backup tous les joueurs
   for gt in $(jq -r '.[].gamertag' db_profiles.json); do
     python scripts/backup_player.py --gamertag "$gt"
   done
   
   # Backup warehouse
   cp -r data/warehouse/ backups/warehouse_pre_cleanup_$(date +%Y%m%d)/
   ```

2. **Validation architecture v5**
   ```bash
   # Vérifier que shared_matches.duckdb est opérationnel
   python scripts/validate_migration.py --all
   
   # Vérifier l'intégrité des données
   python scripts/audit_current_data.py
   ```

3. **Snapshot de l'état actuel**
   ```bash
   # Export des schémas
   python scripts/export_schemas.py > .ai/archive/schemas_pre_cleanup_v5.1.md
   
   # Baseline tests
   pytest --co -q > .ai/archive/test_inventory_v5.1.txt
   ```

4. **Créer une branche de secours**
   ```bash
   git checkout -b backup/pre-cleanup-v5.1
   git push origin backup/pre-cleanup-v5.1
   git checkout main
   ```

**Critères de sortie** :
- ✅ Backups validés et testables
- ✅ `shared_matches.duckdb` contient 100% des matchs
- ✅ Suite de tests passe à 100%
- ✅ Branche de secours créée

---

### Phase 1 : Éradication SQLite Runtime (CRITIQUE)

**Objectif** : Éliminer tout fallback SQLite dans le code applicatif.

#### 1.1 Supprimer Fallback `src/data/query/engine.py`

**Fichier** : `src/data/query/engine.py` lignes 110-123

**Avant** :
```python
# Attacher la base metadata (DuckDB uniquement)
# SQLite n'est plus supporté en runtime (migration v4+)
metadata_duckdb = self.warehouse_path / "metadata.duckdb"
metadata_sqlite = self.warehouse_path / "metadata.db"

if metadata_duckdb.exists():
    conn.execute(f"ATTACH '{metadata_duckdb}' AS meta (READ_ONLY)")
    self._metadata_attached = True
    logger.debug(f"DuckDB metadata attachée: {metadata_duckdb}")
elif metadata_sqlite.exists():
    # SQLite n'est plus supporté en runtime depuis la migration v4
    raise RuntimeError(
        f"Base metadata SQLite détectée ({metadata_sqlite}), mais SQLite n'est plus supporté en runtime. "
        "Veuillez migrer vers DuckDB avec les scripts de migration disponibles dans scripts/migration/."
    )
```

**Après** (v5.1 — Zero Tolerance) :
```python
# ATTACH metadata (DuckDB v5 uniquement)
metadata_duckdb = self.warehouse_path / "metadata.duckdb"

if not metadata_duckdb.exists():
    raise RuntimeError(
        f"metadata.duckdb introuvable : {metadata_duckdb}\n"
        f"L'architecture v5 requiert metadata.duckdb. SQLite n'est plus supporté.\n"
        f"Si migration nécessaire, voir scripts/migration/migrate_metadata_to_duckdb.py"
    )

conn.execute(f"ATTACH '{metadata_duckdb}' AS meta (READ_ONLY)")
self._metadata_attached = True
logger.debug(f"metadata.duckdb attachée: {metadata_duckdb}")
```

**Tests** :
```python
# Nouveau test : échec si metadata.duckdb absent
def test_query_engine_requires_metadata_duckdb(tmp_path):
    """L'engine refuse de démarrer si metadata.duckdb absent."""
    with pytest.raises(RuntimeError, match="metadata.duckdb introuvable"):
        QueryEngine(warehouse_path=tmp_path)
```

#### 1.2 Supprimer Fallback `src/data/infrastructure/database/duckdb_engine.py`

**Fichier** : `src/data/infrastructure/database/duckdb_engine.py` lignes 92-112

**Action** : Même logique — remplacer `if/elif` par `if not exists(): raise`.

#### 1.3 Nettoyer Références SQLite dans `src/utils/paths.py`

**Audit** : Identifier toutes les références `metadata.db` et les remplacer par `metadata.duckdb`.

#### 1.4 Supprimer Imports `sqlite3` dans Modules Runtime

**Fichiers concernés** :
- `src/ui/sync.py`
- `src/ui/multiplayer.py`
- `src/ai/rag.py`

**Action** : Grep + suppression manuelle de tout `import sqlite3` ou usage `sqlite3.connect()`.

**Validation** :
```bash
# Aucun import sqlite3 hors scripts/migration
grep -r "import sqlite3" src/ --exclude-dir=migration
# Doit retourner 0 résultats
```

**Critères de sortie** :
- ✅ Aucun `import sqlite3` dans `src/` (hors migration)
- ✅ Aucun `metadata.db` dans le code runtime
- ✅ Tests passent (aucune régression)

---

### Phase 2 : Archivage Scripts Migration (ORGANISATION)

**Objectif** : Marquer clairement tous les scripts de migration comme LEGACY et HORS SERVICE post-v5.1.

#### 2.1 Ajouter Bannières LEGACY aux Scripts Migration

**Scripts concernés** :
- `scripts/migration/recover_from_sqlite.py` ✅ (déjà fait)
- `scripts/migration/migrate_player_to_duckdb.py`
- `scripts/migration/migrate_all_to_duckdb.py`
- `scripts/migration/migrate_metadata_to_duckdb.py`
- `scripts/migration/migrate_player_to_shared.py`

**Bannière standard** (en docstring) :
```python
"""
╔══════════════════════════════════════════════════════════════════════════════╗
║ 🚨 SCRIPT DE MIGRATION — HORS SERVICE POST-V5.1                             ║
║                                                                              ║
║ Ce script est destiné UNIQUEMENT à la migration SQLite → DuckDB (v4) ou     ║
║ DuckDB v4 → v5 (shared matches).                                            ║
║                                                                              ║
║ NE PAS UTILISER pour le flux applicatif normal (architecture v5 pure).      ║
║ Conservé pour référence historique et cas exceptionnels de recovery.        ║
╚══════════════════════════════════════════════════════════════════════════════╝
"""
```

#### 2.2 Créer `scripts/migration/README.md`

**Contenu** :
```markdown
# Scripts de Migration — Archive Historique

⚠️ **ATTENTION** : Ces scripts sont destinés uniquement à la migration depuis des architectures obsolètes (SQLite, DuckDB v4).

## Statut

| Script | Migration | Post-v5.1 |
|--------|-----------|-----------|
| `recover_from_sqlite.py` | SQLite → DuckDB v4 | ❌ OBSOLETE |
| `migrate_player_to_duckdb.py` | SQLite → DuckDB v4 | ❌ OBSOLETE |
| `migrate_metadata_to_duckdb.py` | SQLite → DuckDB v4 | ❌ OBSOLETE |
| `migrate_player_to_shared.py` | DuckDB v4 → v5 | ❌ OBSOLETE |

## Usage

Si vous devez absolument migrer depuis SQLite ou v4, consultez `docs/MIGRATION_V4_TO_V5.md`.

**Recommandation** : Pour tout nouveau déploiement, démarrer directement en v5 (shared matches).
```

#### 2.3 Décision : `scripts/refetch_film_roster.py`

**Analyse** :
- ✅ Déjà marqué LEGACY (bannière)
- ❌ Utilise SQLite (`import sqlite3`)
- 🔍 Script expérimental (extraction roster depuis film chunks binaires)

**Options** :
1. **Supprimer** — fonction très niche, peu utilisée
2. **Porter en DuckDB** — effort moyen (~4h), faible ROI
3. **Garder tel quel** — marquer EXPERIMENTAL + LEGACY

**Recommandation** : **Option 1 (SUPPRIMER)** ou **Option 3 (GARDER)**

Si conservé :
```python
# Ajouter en docstring
"""
╔══════════════════════════════════════════════════════════════════════════════╗
║ 🧪 SCRIPT EXPERIMENTAL + LEGACY                                             ║
║ Utilise SQLite (obsolète en v5) pour des opérations de recovery binaire.    ║
║ NE PAS UTILISER dans le flux standard. Conservation pour R&D uniquement.    ║
╚══════════════════════════════════════════════════════════════════════════════╝
"""
```

**Critères de sortie** :
- ✅ Tous les scripts migration ont bannière LEGACY
- ✅ `scripts/migration/README.md` créé
- ✅ Décision documentée pour `refetch_film_roster.py`

---

### Phase 3 : Migration Pandas → Polars (TRANSFORMATION)

**Objectif** : Éliminer Pandas du code métier (hors bridges de compatibilité).

#### 3.1 Audit Détaillé Usage Pandas

**Script d'audit** :
```bash
# Créer rapport détaillé
python scripts/audit_pandas_usage.py > .ai/reports/pandas_audit_v5.1.md
```

**Contenu du rapport** :
- Nombre de `import pandas` par fichier
- Fonctions utilisant `.to_pandas()` ou `pd.DataFrame()`
- Dépendances entre modules
- Estimation effort de migration

#### 3.2 Migration par Priorité

**Ordre de migration** (du plus critique au moins) :

1. **`src/analysis/performance_score.py`** — Logique métier core
   - Remplacer `pd.Series` par `pl.Series`
   - Remplacer `.fillna()` par `.fill_null()`
   - Remplacer `.rolling()` par `.rolling()` Polars
   
2. **`src/data/services/win_loss_service.py`** — Service critique
   - Remplacer agrégations Pandas par Polars
   - Utiliser `.group_by()` Polars au lieu de `.groupby()`
   
3. **`src/ui/pages/objective_analysis.py`** — Page UI
   - Travailler en Polars jusqu'à la frontière Plotly
   - Utiliser `_compat.to_pandas_for_plotly()` uniquement au moment du render
   
4. **`src/ui/pages/match_view_helpers.py`** — Helpers
5. **`src/ui/pages/win_loss.py`** — Page UI
6. **`src/ui/cache_filters.py`** — Caching
7. **`src/ui/components/duckdb_analytics.py`** — Composant

#### 3.3 Pattern de Migration Standard

**Avant (Pandas)** :
```python
import pandas as pd

def compute_stats(df: pd.DataFrame) -> pd.DataFrame:
    return df.groupby("player").agg({
        "kills": "sum",
        "deaths": "sum"
    }).fillna(0)
```

**Après (Polars)** :
```python
import polars as pl

def compute_stats(df: pl.DataFrame) -> pl.DataFrame:
    return df.group_by("player").agg([
        pl.col("kills").sum(),
        pl.col("deaths").sum()
    ]).fill_null(0)
```

#### 3.4 Tests de Non-Régression

Pour chaque module migré :
```python
def test_compute_stats_polars_equivalence():
    """Vérifie que la version Polars produit les mêmes résultats que Pandas (référence)."""
    # Données de test
    data = pl.DataFrame({...})
    
    # Résultat Polars
    result = compute_stats(data)
    
    # Résultat attendu (baseline)
    expected = pl.DataFrame({...})
    
    assert_frame_equal(result, expected)
```

**Critères de sortie** :
- ✅ Zéro `import pandas` dans `src/analysis/`
- ✅ Zéro `import pandas` dans `src/data/services/`
- ✅ Zéro `import pandas` dans `src/ui/pages/` (hors frontière Plotly)
- ✅ Modules `_compat.py`, `_arrow_bridge.py` conservés (frontières)
- ✅ Tous les tests passent

---

### Phase 4 : Nettoyage DBs Player v5 (OPTIMISATION)

**Objectif** : Supprimer les tables redondantes post-migration v5.

#### 4.1 Vérification Pré-Cleanup

```bash
# Dry-run pour visualiser ce qui serait supprimé
python scripts/cleanup_player_dbs_v5.py --all --dry-run --verbose
```

#### 4.2 Exécution Cleanup avec Backup

```bash
# Backup + cleanup de tous les joueurs
python scripts/cleanup_player_dbs_v5.py --all --backup --remove-compat-views
```

**Tables supprimées** :
- `match_stats`
- `match_participants`
- `highlight_events`
- `medals_earned`
- `v_*` (views de compatibilité)

**Tables conservées** :
- `player_match_enrichment`
- `teammates_aggregate`
- `antagonists`
- `match_citations`
- `career_progression`

#### 4.3 Validation Post-Cleanup

```bash
# Lancer l'app et vérifier fonctionnement
python launcher.py run

# Suite de tests complète
pytest

# Sync delta pour validation
python scripts/sync.py --delta --gamertag MonGamertag
```

**Critères de sortie** :
- ✅ Tables redondantes supprimées (gain ~85% espace)
- ✅ Views `v_*` supprimées
- ✅ Application fonctionne sans erreur
- ✅ Tests passent

---

### Phase 5 : Audit et Nettoyage Scripts Archive (ORGANISATION)

**Objectif** : Trier le dossier `scripts/_archive/` (~60+ scripts).

#### 5.1 Inventaire Complet

```bash
# Lister tous les scripts archive
ls -1 scripts/_archive/*.py > .ai/archive/scripts_inventory.txt
```

#### 5.2 Classification

| Catégorie | Action | Exemples |
|-----------|--------|----------|
| **R&D terminée** | 🗑️ SUPPRIMER | `analyze_binary_patterns.py`, `analyze_chunks_bitshifted.py` |
| **Utilitaires migration** | 📦 GARDER + DOC | `migrate_*.py`, `recover_*.py` |
| **Benchmarks historiques** | 📦 GARDER | `benchmark_polars.py`, `benchmark_v4_vs_v5.py` |
| **Scripts expérimentaux** | 🔍 ÉVALUER | `analyze_weapon_ids_*.py` |

#### 5.3 Créer `scripts/_archive/README.md`

```markdown
# Archive Scripts — Historique R&D et Migration

Ce dossier contient des scripts utilisés pour :
- Recherche & Développement (binary analysis, weapon IDs, etc.)
- Migration de données (v3 → v4 → v5)
- Benchmarking de performance

## Statut

⚠️ La plupart de ces scripts sont **obsolètes** et conservés pour référence historique uniquement.

### Scripts de Migration (Conservés)

- `migrate_*.py` — Scripts de migration entre versions
- `recover_*.py` — Scripts de recovery depuis SQLite

### Scripts R&D (Historiques)

- `analyze_*.py` — Analyses binaires, weapon IDs, patterns
- `benchmark_*.py` — Benchmarks Polars vs Pandas, v4 vs v5

## Usage

⚠️ NE PAS UTILISER ces scripts sans validation préalable. Ils peuvent dépendre d'architectures obsolètes.
```

**Critères de sortie** :
- ✅ Scripts R&D terminée supprimés
- ✅ Scripts migration documentés
- ✅ `README.md` créé

---

### Phase 6 : Validation Finale et Documentation (CLÔTURE)

**Objectif** : Valider l'état final et documenter la nouvelle architecture pure v5.

#### 6.1 Suite de Tests Complète

```bash
# Tests hors intégration (rapide)
pytest -q --ignore=tests/integration

# Tests d'intégration
pytest tests/integration/

# Tests slow (performance)
pytest -m "slow"

# Couverture
pytest --cov=src --cov-report=html --cov-report=term
```

**Seuil de couverture minimum** : 80% (actuel : ~75%)

#### 6.2 Audit de Sécurité

```bash
# Aucune référence SQLite runtime
grep -r "import sqlite3" src/ --exclude-dir=migration
# → 0 résultats

# Aucune référence .db dans config
grep -r "\.db" db_profiles.json app_settings.json
# → 0 résultats

# Vérifier Pandas dans métier
grep -r "import pandas" src/analysis/ src/data/services/
# → 0 résultats
```

#### 6.3 Mise à Jour Documentation

**Fichiers à mettre à jour** :

1. **`docs/ARCHITECTURE_V5.md`**
   - Ajouter section "État Post-Cleanup v5.1"
   - Documenter l'élimination complète de SQLite/Pandas

2. **`CLAUDE.md`**
   - Renforcer règles anti-SQLite/Pandas
   - Supprimer mentions de fallback

3. **`README.md`**
   - Mettre à jour stack technique (100% DuckDB + Polars)
   - Supprimer références migration SQLite

4. **`.ai/thought_log.md`**
   - Ajouter entrée "Éradication Legacy v5.1 Complétée"

5. **Créer `docs/V5.1_PURE_ARCHITECTURE.md`**
   - Document manifeste de l'architecture pure v5.1
   - Règles strictes (zero tolerance SQLite/Pandas)

#### 6.4 Release Notes v5.1

**Créer** : `.ai/RELEASE_NOTES_V5.1.md`

**Contenu** :
```markdown
# Release Notes v5.1 — Pure Architecture

## 🎯 Changements Majeurs

### ❌ Supprimé (Breaking Changes)

- **SQLite** : Tout support SQLite runtime supprimé
  - Fallback `metadata.db` : SUPPRIMÉ
  - Scripts : conservés en `scripts/migration/` (legacy)
  
- **Pandas** : Éliminé du code métier
  - `src/analysis/` : 100% Polars
  - `src/data/services/` : 100% Polars
  - Bridges de compatibilité : conservés (`_compat.py`)

- **Tables redondantes** : Nettoyage automatique
  - `match_stats`, `match_participants`, etc. : SUPPRIMÉES des player DBs
  - Views `v_*` : SUPPRIMÉES

### ✅ Améliorations

- Performance : -85% taille player DBs
- Maintenabilité : code plus simple, moins de dépendances
- Sécurité : aucune injection SQL via fallback SQLite

## 🔧 Migration

Si migration depuis v5.0 → v5.1 :
1. Backup : `python scripts/backup_player.py --all`
2. Cleanup : `python scripts/cleanup_player_dbs_v5.py --all --backup`
3. Tests : `pytest`

## 📚 Documentation

- [V5.1_PURE_ARCHITECTURE.md](docs/V5.1_PURE_ARCHITECTURE.md)
- [ARCHITECTURE_V5.md](docs/ARCHITECTURE_V5.md)
```

**Critères de sortie** :
- ✅ Tests passent à 100%
- ✅ Couverture ≥ 80%
- ✅ Documentation mise à jour
- ✅ Release notes créées
- ✅ Zéro fallback SQLite
- ✅ Zéro Pandas métier

---

## 🚨 Risques et Mitigations

| Risque | Probabilité | Impact | Mitigation |
|--------|-------------|--------|------------|
| **Perte de données pendant cleanup** | 🟡 Moyen | 🔴 Critique | Backups systématiques avant toute opération |
| **Régression fonctionnelle migration Pandas→Polars** | 🟡 Moyen | 🟡 Moyen | Tests de non-régression exhaustifs par module |
| **Incompatibilité Plotly/Streamlit sans Pandas** | 🟢 Faible | 🟡 Moyen | Conserver bridges `_compat.py` |
| **Impossibilité de migrer depuis SQLite post-v5.1** | 🟢 Faible | 🟢 Faible | Scripts migration conservés en archive |
| **Break backward compatibility installations v4** | 🔴 Certain | 🟢 Faible | ASSUMÉ — pas de rétrocompatibilité par design |

---

## 📊 Métriques de Succès

### Indicateurs Techniques

| Métrique | État Actuel (v5.0) | Cible (v5.1) |
|----------|-------------------|--------------|
| **Imports SQLite runtime** | ~7 fichiers | **0** |
| **Imports Pandas métier** | ~7 fichiers | **0** |
| **Taille moyenne player DB** | ~30 MB | **~4 MB** (-87%) |
| **Scripts migration actifs** | ~10 | **0** (archivés) |
| **Couverture tests** | 75% | **≥80%** |
| **Temps suite tests** | ~72s | **<60s** |

### Indicateurs Qualité

- ✅ Zéro fallback vers architecture obsolète
- ✅ Zéro dépendance SQLite en runtime
- ✅ Zéro Pandas dans logique métier
- ✅ Documentation complète et à jour
- ✅ Code maintenable (moins de complexité)

---

## 📅 Planning Estimé

| Phase | Durée Estimée | Complexité |
|-------|---------------|------------|
| **Phase 0** : Préparation | 2h | 🟢 Faible |
| **Phase 1** : SQLite Runtime | 4h | 🟡 Moyenne |
| **Phase 2** : Archivage Scripts | 2h | 🟢 Faible |
| **Phase 3** : Pandas → Polars | 12h | 🔴 Élevée |
| **Phase 4** : Cleanup DBs | 1h | 🟢 Faible |
| **Phase 5** : Audit Archive | 3h | 🟡 Moyenne |
| **Phase 6** : Validation & Docs | 4h | 🟡 Moyenne |
| **TOTAL** | **28h** | 🔴 Élevée |

**Répartition recommandée** : 4 jours × 7h (avec revues intermédiaires)

---

## 🎯 Décisions Architecturales Clés

### 1. Zéro Fallback, Zéro Backward Compatibility

**Principe** : L'architecture v5.1 **refuse** de démarrer si metadata.duckdb est absent ou si SQLite est détecté.

**Justification** :
- Forcer la migration complète
- Éviter la dette technique de compatibilité
- Simplifier le code et les tests

### 2. Conservation des Bridges Pandas (Frontières)

**Modules conservés** :
- `src/visualization/_compat.py`
- `src/data/repositories/_arrow_bridge.py`
- `src/data/integration/streamlit_bridge.py`

**Justification** : Plotly et Streamlit imposent Pandas. La conversion `.to_pandas()` **à la frontière** est acceptable.

### 3. Scripts Migration en Archive Documentée

**Décision** : Ne PAS supprimer les scripts de migration, mais les marquer clairement LEGACY avec bannières.

**Justification** :
- Référence historique
- Possibilité de recovery exceptionnel
- Transparence de l'évolution du projet

### 4. Cleanup Tables Player Optionnel

**Décision** : Le cleanup des tables redondantes est **recommandé** mais pas obligatoire (gain d'espace).

**Justification** : L'application fonctionne avec ou sans cleanup (tables ignorées en v5).

---

## 📝 Livrables

### Documents

- [x] **`.ai/PLAN_ERADICATION_LEGACY_V5.md`** — Ce document (plan détaillé)
- [ ] **`docs/V5.1_PURE_ARCHITECTURE.md`** — Manifeste architecture pure
- [ ] **`.ai/RELEASE_NOTES_V5.1.md`** — Release notes v5.1
- [ ] **`scripts/migration/README.md`** — Documentation archive scripts migration
- [ ] **`scripts/_archive/README.md`** — Documentation archive scripts R&D
- [ ] **`.ai/reports/pandas_audit_v5.1.md`** — Rapport audit Pandas

### Code

- [ ] Migration de 7 modules Pandas → Polars
- [ ] Suppression fallback SQLite (2 fichiers)
- [ ] Ajout bannières LEGACY (5 scripts)
- [ ] Nettoyage références `.db` (config)
- [ ] Tests de non-régression (nouveaux)

### Validation

- [ ] Suite de tests verte (100%)
- [ ] Couverture ≥ 80%
- [ ] Documentation mise à jour
- [ ] Backups validés

---

## 🔗 Références

### Documentation Existante

- [ARCHITECTURE_V5.md](../docs/ARCHITECTURE_V5.md) — Architecture actuelle
- [CLEANUP_V5.md](../docs/CLEANUP_V5.md) — Guide cleanup tables
- [MIGRATION_V4_TO_V5.md](../docs/MIGRATION_V4_TO_V5.md) — Migration v4→v5
- [POLARS_MIGRATION.md](../docs/POLARS_MIGRATION.md) — Guide migration Polars

### Audits et Rapports

- `.ai/archive/v5.0/v5-baseline-audit.md` — Audit baseline v5
- `.ai/archive/v5.0/v5-migration-report.md` — Rapport migration v5
- `.ai/thought_log.md` — Journal de décisions

---

## 🏁 Conclusion

Ce plan d'éradication vise à atteindre un **état de pureté architecturale v5.1** :

- **100% DuckDB** — Zéro SQLite en runtime
- **100% Polars** — Zéro Pandas dans le code métier
- **Zéro legacy** — Pas de fallback, pas de rétrocompatibilité

**Bénéfices attendus** :
- 🚀 Performance améliorée
- 🧹 Code plus propre et maintenable
- 🛡️ Moins de surface d'attaque (injection SQL)
- 📉 Réduction de la dette technique
- 📚 Documentation claire et à jour

**Approche** : Méthodique, incrémentale, sécurisée (backups systématiques).

---

**Prêt à exécuter** ? 🚀

Commencer par **Phase 0 (Préparation)** pour établir le filet de sécurité.
