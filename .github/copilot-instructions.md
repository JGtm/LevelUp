# Instructions pour GitHub Copilot & Assistants IA

Ce fichier définit les conventions et règles à suivre lors de modifications sur le projet LevelUp.

---

## Contexte du Projet

**LevelUp** est un dashboard Streamlit pour analyser les statistiques Halo Infinite.

- **Stack** : Python 3.10+, Streamlit, DuckDB, SPNKr (API Halo)
- **Langue UI** : Français (traductions dans `src/ui/translations.py`)
- **Architecture** : DuckDB v6 (shared matches + SQL views)

---

## Environnement de référence (Windows)

Objectif : éviter les confusions d'interpréteur (PowerShell vs Git Bash/MSYS2) et les erreurs "module introuvable".

- **Python officiel** : `.venv` à la racine du repo (Python 3.12.10)
- **Interdit** : utiliser le Python MSYS2/MinGW (`pacman ... python/pip`) pour exécuter le projet
- **Règle d'or** : toujours lancer les outils via `python -m ...` (ne pas dépendre du `PATH`)

Packages critiques vérifiés dans `.venv` :
- `pytest==9.0.2`
- `duckdb==1.4.4`
- `polars==1.38.1`
- `pyarrow==23.0.0`
- `pandas==2.3.3`
- `numpy==2.4.2`

Healthcheck (à lancer avant de diagnostiquer un souci d'environnement) :
- `python scripts/check_env.py`

---

## Architecture des Données (v6)

| Données | Stockage | Chemin |
|---------|----------|--------|
| Référentiels | DuckDB | `data/warehouse/metadata.duckdb` |
| Matchs partagés | DuckDB | `data/warehouse/shared_matches_v2.duckdb` |
| Enrichissements joueur | DuckDB | `data/players/{gamertag}/stats.duckdb` |
| Archives | Parquet | `data/players/{gamertag}/archive/` |
| Config | JSON | `db_profiles.json` |
| Auth / token cache | DuckDB (`sync_meta`) + MSAL | `src/auth/` |

### Tables Principales

#### metadata.duckdb (référentiels)

| Table | Description |
|-------|-------------|
| `career_ranks` | Paliers et noms des rangs Halo |
| `citation_mappings` | Mappings médaille→citation |
| `mode_name_tr` / `mode_*` | Traductions et paramètres des modes de jeu |
| `weapon_labels` | Labels EN/FR par weapon_id filmshell (UBIGINT) |

#### shared_matches_v2.duckdb (centralisée)

| Table | Description |
|-------|-------------|
| `match_registry` | Registre central (1 ligne par match unique) |
| `match_participants` | Stats de tous les joueurs de tous les matchs |
| `highlight_events` | Événements filmés (`gamertag` **supprimé** en v6 — résolu via `v_gamertag_lookup`) |
| `medals_earned` | Médailles de tous les joueurs |
| `xuid_aliases` | Mapping global xuid→gamertag |
| `weapon_kills` | Kills par arme par joueur par match (filmshell) |

**Vues SQL garanties présentes en v6** (ne jamais recréer de guards `_has_shared_view`) :

| Vue | Description |
|-----|-------------|
| `v_gamertag_lookup` | FULL OUTER JOIN `xuid_aliases` + `match_participants` + `match_kill_events_latest` — résolution gamertag unifiée |
| `v_match_full` | `match_registry` enrichi avec métadonnées i18n (maps, playlists, game variants) |
| `v_weapon_kills` | `weapon_kills` avec `effective_weapon_id = COALESCE(reconciled_as, weapon_id)` |

> `v_killer_victim_full` a été **supprimée le 2026-08-02** : elle n'est plus garantie et ne doit
> plus être écrite dans une requête. Les paires tueur → victime se lisent dans
> `killer_victim_pairs` (historique) ou dans `match_kill_events` via sa vue
> `match_kill_events_latest` (ADR 0026 — jamais la table brute).

#### stats.duckdb (par joueur) — v5.1 allégée

> 8 tables supprimées : match_stats, match_participants, highlight_events,
> medals_earned, killer_victim_pairs, player_match_stats, xuid_aliases, teammates_aggregate

| Table | Description |
|-------|-------------|
| `player_match_enrichment` | performance_score, session_id, is_with_friends — **SEULE table match** |
| `personal_score_awards` | Awards objectifs (PersonalScores API) |
| `match_citations` | Citations calculées par match |
| `career_progression` | Historique rangs |
| `media_files` | Fichiers médias indexés |
| `media_match_associations` | Associations médias↔matchs |
| `sessions` | Sessions groupées |
| `sync_meta` | Métadonnées sync + cache MSAL sérialisé (token Microsoft) |
| `mv_*` | Vues matérialisées |

### Règles Streamlit v6

- Tout `st.plotly_chart` doit inclure `config=` (PLOTLY_CLEAN_CONFIG ou PLOTLY_STATIC_CONFIG)
- Préférer `@fragment_if_available` pour les sections interactives multi-charts
- Coéquipiers chargés depuis `shared.match_participants` (pas les DBs individuelles)
- `width="stretch"` au lieu de `use_container_width=True` (déprécié)
- Gamertag résolu **exclusivement** via `v_gamertag_lookup` — pas de `LEFT JOIN xuid_aliases` ad hoc
- Armes lues via `v_weapon_kills` — pas la table `weapon_kills` directement
- **Pas de guards** `_has_shared_view` / `_has_shared_table` — les vues V6 sont garanties présentes

---

## Authentification (`src/auth/`)

- **Entry point** : `src/auth/provider.py` — process cache (4 h TTL), MSAL silent refresh, `AuthRequiredError`, `start/complete_device_flow`
- **`LEVELUP_CLIENT_ID`** hardcodé dans `src/auth/_msal.py` — les utilisateurs finaux n'ont **plus** besoin de configurer Azure
- **`SPNKR_AZURE_CLIENT_ID`** (env var) reste supporté comme **fallback backend** — ne pas supprimer les fonctions qui l'exploitent ni celles gérant le refresh token
- Le cache MSAL est persisté en DuckDB dans `sync_meta` via `SerializableTokenCache`
- L'échange `access_token → (spartan_token, clearance)` est géré par `src/auth/_halo_exchange.py` (sans état)
- **Interdit** : recréer un wizard de configuration Azure dans l'UI ou des popups de saisie de `client_id`

---

## Couche `src/ai/` (outillage développeur)

| Fichier | Rôle |
|---------|------|
| `rag.py` | `HaloKnowledgeBase` — indexation + recherche sémantique (ChromaDB) |
| `_rag_models.py` | Modèles Pydantic : `RAGConfig`, `Document`, `SearchResult` |
| `_rag_chunker.py` | `TextChunker` — découpage de docs en chunks |
| `_rag_github.py` | `GitHubIndexer` — indexation de repos GitHub |
| `mcp_server.py` | Serveur MCP (protocole Model Context Protocol) pour Cursor |

**Règle** : `src/ai/` est réservé à l'**outillage développeur** (RAG docs, MCP). Aucune logique métier Halo ne doit y résider. Pas d'import de `src.data` ni de `src.ui` dans ce module.

---

## Workflow d'Interaction IA

### Avant toute modification

1. **Analyser la demande** : Reformuler pour confirmer la compréhension
2. **Explorer le contexte** : Lire les fichiers concernés
3. **Proposer un plan** : Lister les étapes avant d'implémenter
4. **Valider** : Attendre le "go" avant les modifications majeures

### Structure d'une réponse idéale

```markdown
## Compréhension de la demande
[Reformulation en 1-2 phrases]

## Analyse de l'existant
- Fichiers impactés : ...
- Dépendances : ...

## Plan d'implémentation
1. [ ] Étape 1
2. [ ] Étape 2

## Points de vigilance
- ...

Tu veux que je procède ?
```

---

## Conventions de Code

### Python

- **Type hints** obligatoires sur fonctions publiques
- **Docstrings** en français
- **Formatage** : Black + isort + ruff

```python
# Bon
def compute_kd_ratio(kills: int, deaths: int) -> float:
    """Calcule le ratio kills/deaths."""
    if deaths == 0:
        return float(kills)
    return kills / deaths
```

### Accès aux Données

**TOUJOURS** utiliser `DuckDBRepository` :

```python
from src.data.repositories import DuckDBRepository

repo = DuckDBRepository(db_path, xuid)
matches = repo.load_matches(limit=100)
```

**INTERDIT** : Utiliser `src/db/loaders.py` (déprécié)

### SQL / DuckDB

```python
# Bon - Paramètres
cursor.execute("SELECT * FROM match_stats WHERE match_id = ?", (match_id,))

# Mauvais - Injection SQL
cursor.execute(f"SELECT * FROM match_stats WHERE match_id = '{match_id}'")
```

---

## SyncScope (`src/data/sync/scope.py`)

Dataclass centralisant **tous les flags de données** partagés entre sync et backfill.

### Usage recommandé

```python
from src.data.sync.scope import SyncScope

# Construction depuis CLI
scope = SyncScope.from_cli_args(args)

# Tout activer
scope = SyncScope.make_all(max_matches=100)

# Sélection fine
scope = SyncScope(medals=True, force_medals=True)
scope.resolve()

# Passer aux fonctions
await backfill_player_data(gamertag, scope=scope)
```

### Pour ajouter un nouveau type de données

1. Ajouter le champ dans `SyncScope` + registres (`_ALL_DATA_FIELDS`, `_FORCE_MAP`, `_REQUESTED_TYPE_MAP`)
2. Ajouter l'argument CLI dans `scripts/backfill/cli.py`
3. Implémenter la logique métier dans l'orchestrateur / engine

### Legacy

Les fonctions `backfill_player_data`, `backfill_all_players`, `_backfill_with_api` et
`find_matches_missing_data` conservent les 30+ kwargs individuels marqués `LEGACY` dans le code.
**Nouveau code : toujours passer `scope=SyncScope(...)`.**

---

## Migrations de Schéma DuckDB

Quand tu ajoutes ou modifies une colonne, une table ou un index dans une DB DuckDB (`player`, `shared`, `shared_pve`, `metadata`) :

**Étapes obligatoires :**

1. **Créer/modifier `ensure_xxx`** dans `src/data/sync/migrations.py` — doit être **idempotente** (utiliser `_add_column_if_missing()` ou `IF NOT EXISTS`)

2. **Créer un fichier step** dans `src/data/migration/steps/add_{nom}.py` :
   ```python
   from src.data.migration.registry import Migration, register
   from src.data.sync.migrations import ensure_xxx

   register(Migration(
       name="add_{nom}",
       target_db="player",  # ou "shared" ou "shared_pve"
       description="Description courte",
       apply_schema=ensure_xxx,
       # apply_backfill=ma_fonction,  # si backfill API nécessaire
       # requires_api=True,
   ))
   ```

3. **Ajouter l'import** dans `src/data/migration/steps/__init__.py` :
   ```python
   from src.data.migration.steps import add_{nom}  # noqa: F401
   ```

**Règles :**

- Les migrations sont appliquées **automatiquement** au démarrage du backend Go via `apps/go-api/cmd/server/main.go` puis `migration.RunForDB(...)`
- Chaque DB trace les migrations dans une table `schema_migrations` (colonnes : `name`, `applied_at`, `schema_done`, `backfill_done`)
- Une migration déjà appliquée ne tourne **jamais** deux fois (idempotence garantie)
- `target_db` détermine quelle DB reçoit la migration : `"player"` (stats.duckdb de chaque joueur), `"shared"` (shared_matches_v2.duckdb), `"shared_pve"` (shared_pve.duckdb), `"metadata"` (metadata.duckdb)

---

## Synchronisation

### Mode Delta (incrémental)

```bash
python scripts/sync.py --delta --gamertag MonGamertag
```

### Mode Full (complet)

```bash
python scripts/sync.py --full --gamertag MonGamertag --max-matches 500
```

---

## Tests

```bash
# Tous les tests (recommandé)
python -m pytest

# Avec couverture
python -m pytest --cov=src

# Tests spécifiques
python -m pytest tests/test_duckdb_repository.py -v

# Suite stable hors intégration (Windows)
python -m pytest --ignore=tests/integration
```

---

## Stratégie de branches Git

### Règle : 1 tâche = 1 branche, N commits

- Phases séquentielles d'un même sujet → **commits** sur une branche unique
- Plusieurs branches uniquement si les tâches sont **indépendantes et parallélisables**
- Anti-pattern à éviter : créer `feature/phase1`, `feature/phase2`, `feature/phase3`… pour un travail linéaire

### Règles opérationnelles

1. Vérifier la branche courante avant de committer : `git branch --show-current`
2. **⛔ JAMAIS travailler sur `main`** — sans aucune exception. Si la branche courante est `main`, créer une branche de travail avant toute modification.
3. **Toute nouvelle fonction/feature/fix** → créer une nouvelle branche depuis la branche courante (`git checkout -b <type>/<nom>`), jamais travailler directement sur la branche parente.
4. **⛔ Ne jamais changer de branche** si un travail différent est déjà en cours sur la branche courante — interrompre la tâche et informer l'utilisateur pour éviter tout conflit entre agents.
5. Si aucun nom de branche n'est spécifié, proposer un nom avant de créer
6. Entre sessions : relire `git log --oneline -10` pour reprendre sur la bonne branche

---

## Commits

### Format Conventional Commits

```
<type>(<scope>): <description>
```

### Types autorisés

| Type | Description |
|------|-------------|
| `feat` | Nouvelle fonctionnalité |
| `fix` | Correction de bug |
| `docs` | Documentation |
| `refactor` | Refactoring |
| `test` | Tests |
| `chore` | Maintenance |

### Exemples

```
feat(ui): ajouter graphe radar des stats par minute
fix(sync): corriger détection des modes Firefight
docs: mettre à jour README avec branding LevelUp
```

---

## Diagnostic de revue de code

Avant chaque commit, vérifier que le code ne réintroduit pas d'anti-patterns connus.

### Seuils

| Métrique | Max | Conséquence |
|----------|:---:|-------------|
| Lignes par fichier | **500** | Découper en modules (mixins, `*_logic.py`) |
| Lignes par fonction | **80** | Extraire des sous-fonctions |
| Copies d'un pattern | **≤ 2** | Centraliser (helper/constante) |
| Magic numbers | **0** | Enum (`Outcome.WIN`) ou constante |
| Code mort | **0** | Supprimer avec tests et imports associés |
| Connexions DB bare | **0** | Context manager obligatoire |

### Anti-patterns interdits

1. **Dead code museum** — code mort conservé "au cas où"
2. **Compatibility guard forever** — `if POLARS_AVAILABLE:` après migration terminée
3. **God file** — fichier >500L avec responsabilités distinctes
4. **Swiss-army function** — fonction qui fait tout (init + logique + IO + render)
5. **Copy-paste config** — même valeur dans 3+ endroits au lieu d'une constante
6. **Bare connect** — `duckdb.connect()` sans context manager
7. **Manual coercion** — `@dataclass` + parsing ad hoc → préférer Pydantic v2
8. **Magic integer** — `outcome == 2` → `Outcome.WIN`
9. **Logique dans l'UI** — calculs purs dans des fichiers Streamlit → séparer en `*_logic.py`

### Patterns recommandés

- **God class** → mixins MRO (`engine.py` → 8 mixins + `_protocol.py`)
- **God function** → extract method (`main()` → sous-fonctions nommées)
- **Page UI complexe** → `page.py` + `page_logic.py` + `page_data.py`
- **Config/parsing** → Pydantic v2 `BaseModel` + `model_validate()`
- **Codes numériques** → `IntEnum`
- **Connexions DB** → `duckdb_read_only()` / `duckdb_read_write()`
- **Constantes** → modules dédiés (`PLOTLY_CLEAN_CONFIG`, `DATE_FORMAT_FR`)

---

## À Éviter

1. **Ne pas** utiliser les loaders legacy (`src/db/loaders.py`)
2. **Ne pas** modifier les tables DB sans migration
3. **Ne pas** hardcoder des chemins Windows
4. **Ne pas** créer de dépendances sans les ajouter à `pyproject.toml`
5. **Ne pas** committer des tokens ou secrets

---

## Checklist avant PR

- [ ] Tests passent (`pytest`)
- [ ] Pas d'erreurs de type
- [ ] Traductions FR à jour si nouvelle UI
- [ ] Documentation mise à jour si nouvelle feature
- [ ] Commit message au format Conventional Commits

---

## Ressources

- [DuckDB Documentation](https://duckdb.org/docs/)
- [Streamlit Docs](https://docs.streamlit.io/)
- [SPNKr Documentation](https://github.com/acurtis166/SPNKr)
