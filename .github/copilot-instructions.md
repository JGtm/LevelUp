# Instructions pour GitHub Copilot & Assistants IA

Ce fichier définit les conventions et règles à suivre lors de modifications sur le projet LevelUp.

---

## Contexte du Projet

**LevelUp** est un dashboard Streamlit pour analyser les statistiques Halo Infinite.

- **Stack** : Python 3.10+, Streamlit, DuckDB, SPNKr (API Halo)
- **Langue UI** : Français (traductions dans `src/ui/translations.py`)
- **Architecture** : DuckDB v5 (shared matches)

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

## Architecture des Données (v5)

| Données | Stockage | Chemin |
|---------|----------|--------|
| Référentiels | DuckDB | `data/warehouse/metadata.duckdb` |
| Matchs partagés | DuckDB | `data/warehouse/shared_matches.duckdb` |
| Enrichissements joueur | DuckDB | `data/players/{gamertag}/stats.duckdb` |
| Archives | Parquet | `data/players/{gamertag}/archive/` |
| Config | JSON | `db_profiles.json` |

### Tables Principales

#### shared_matches.duckdb (centralisée)

| Table | Description |
|-------|-------------|
| `match_registry` | Registre central (1 ligne par match unique) |
| `match_participants` | Stats de tous les joueurs de tous les matchs |
| `highlight_events` | Événements filmés de tous les matchs |
| `medals_earned` | Médailles de tous les joueurs |
| `xuid_aliases` | Mapping global xuid→gamertag |

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
| `sync_meta` | Métadonnées sync |
| `mv_*` | Vues matérialisées |

### Règles Streamlit v5.1

- Tout `st.plotly_chart` doit inclure `config=` (PLOTLY_CLEAN_CONFIG ou PLOTLY_STATIC_CONFIG)
- Préférer `@fragment_if_available` pour les sections interactives multi-charts
- Coéquipiers chargés depuis `shared.match_participants` (pas les DBs individuelles)
- `width="stretch"` au lieu de `use_container_width=True` (déprécié)

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
2. Ne jamais travailler sur `main` sans instruction explicite
3. Si aucun nom de branche n'est spécifié, proposer un nom avant de créer
4. Entre sessions : relire `git log --oneline -10` pour reprendre sur la bonne branche

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
