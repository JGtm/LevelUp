# CLAUDE.md - Instructions pour agents IA

> Ce fichier est lu par Claude Code et autres agents IA au début de chaque session.

## Contexte Projet

**LevelUp** - Dashboard de statistiques Halo Infinite avec architecture DuckDB v6 (shared matches + SQL views).

## Workflow Agentique

**AVANT TOUTE ACTION** : Consulter les fichiers `.ai/` :
- `.ai/project_map.md` : Cartographie du projet
- `.ai/thought_log.md` : Journal des décisions
- `.ai/data_lineage.md` : Flux de données
- `.ai/SPRINT_EXPLORATION.md` : Exploration codebase

**APRÈS CHAQUE MODIFICATION SIGNIFICATIVE** : Mettre à jour ces fichiers.

**THOUGHT LOG — RÈGLE OBLIGATOIRE** :
Avant tout commit (ou à défaut avant de rendre la main à l'utilisateur), ajouter une entrée dans `.ai/thought_log.md` avec :
- La date `[YYYY-MM-DD]`
- Le titre de la tâche
- Le statut (En cours / Complété)
- La décision technique principale
- Les résultats observés
- La conclusion / prochaine étape

Ne pas sauter cette étape même pour des modifications « mineures ». L'absence d'entrée thought_log = tâche non terminée.

**Documentation architecture** : `docs/ARCHITECTURE_V6.md`
**Plans archivés** : `.ai/archive/v5.0/` (plans, audits, rapports de migration)

**Onboarding nouveau dev / nouvelle page** : `docs/FOUNDATIONS_GUIDE.md` (EN) + `docs/FR/FOUNDATIONS_GUIDE.md` — guide consolidé sur les 4 fondations transverses (canonical types + adapters + i18n manifests + ECharts wrappers).

**Décisions architecturales (ADRs)** :
- `docs/adr/0001-charts-stack-echarts.md` — pourquoi ECharts (vs Plotly/Recharts)
- `docs/adr/0002-canonical-player-match-row.md` — pourquoi `canonical.*` cross-titres
- `docs/adr/0003-i18n-manifest-and-linter.md` — pourquoi TOML manifests + lint custom
- `docs/adr/0004-narrative-engine.md` — pourquoi 8 rôles + radar 6 axes

**Skills agent** (à invoquer avant tout commit) : `.claude/skills/{arch-rules, canonical-types, color-tokens, foundations-usage, delivery-checklist, plan-review, halo-modes, db-schema, frontend-patterns, go-features}/SKILL.md`.

**READMEs catalogues** :
- `apps/go-api/internal/analysis/{temporal, breakdown, narrative}/README.md` — exports + exemples + consumers
- `apps/web/src/components/charts/README.md` — catalogue des 11 wrappers ECharts

## Architecture des Données (v5)

| Type | Stockage | Chemin |
|------|----------|--------|
| Référentiels | DuckDB | `data/warehouse/metadata.duckdb` |
| Matchs partagés | DuckDB | `data/warehouse/shared_matches_v2.duckdb` |
| Stats PvE Firefight | DuckDB | `data/warehouse/shared_pve.duckdb` |
| Enrichissements joueur | DuckDB | `data/players/{gamertag}/stats.duckdb` |
| Archives | Parquet | `data/players/{gamertag}/archive/` |
| Config | JSON | `db_profiles.json`, `app_settings.json`, `.env.local` |

## Tables DuckDB Principales

### metadata.duckdb (référentiels)

| Table | Description |
|-------|-------------|
| `career_ranks` | Paliers et noms des rangs Halo |
| `citation_mappings` | Mappings médaille→citation |
| `mode_lang_settings` | Paramètres de langue par mode |
| `mode_name_tr` | Traductions des noms de modes |
| `mode_pair_overrides` | Surcharges de paires map/mode |
| `mode_prefix_names` | Préfixes canoniques de modes |
| `weapon_labels` | Labels EN/FR par weapon_id filmshell (UBIGINT) — **v5.4** |

### shared_matches_v2.duckdb (centralisée)

| Table | Description |
|-------|-------------|
| `match_registry` | Registre central (1 ligne par match unique) |
| `match_participants` | Stats de tous les joueurs de tous les matchs (31 colonnes, incl. MMR) |
| `highlight_events` | Événements filmés de tous les matchs |
| `medals_earned` | Médailles de tous les joueurs |
| `killer_victim_pairs` | Paires killer→victim de tous les matchs |
| `xuid_aliases` | Mapping global xuid→gamertag |

### shared_pve.duckdb (stats Firefight) — v5.2

| Table | Description |
|-------|-------------|
| `pve_match_stats` | Stats par joueur par match Firefight (waves, boss, kills par type d'ennemi : Grunt/Elite/Jackal/Brute/Hunter/Skimmer/Crawler/Soldier/Knight/Warden) |

### stats.duckdb (par joueur) — v5.1 allégée

> **8 tables supprimées** lors du cleanup v5.1 : `match_stats`, `match_participants`,
> `highlight_events`, `medals_earned`, `killer_victim_pairs`, `player_match_stats`,
> `xuid_aliases`, `teammates_aggregate` — données centralisées dans shared.

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
| `match_skill_rank` | Rating LUSR ou CSR par match (PK=match_id, exclusif) — **v5.3** |
| `mv_*` | Vues matérialisées (mv_player_matches, mv_map_stats, etc.) |

## Environnement Python

**IMPORTANT : Utiliser le `.venv` à la racine du repo (Python 3.12.10 Windows natif)**

### Configuration officielle

- **Interpreter** : `.venv` à la racine du repo
- **Python** : 3.12.10
- **Commande canonique** : toujours préférer `python -m ...` (ex: `python -m pytest`)

### Packages vérifiés

- `pytest==9.0.2`
- `duckdb==1.4.4`
- `polars==1.38.1`
- `pyarrow==23.0.0`
- `pandas==2.3.3` (uniquement pour compatibilité Streamlit/Plotly, interdit dans le code métier)
- `numpy==2.4.2`

### Activation selon shell

- **PowerShell** : `./.venv/Scripts/Activate.ps1`
- **cmd.exe** : `.venv\\Scripts\\activate.bat`
- **Git Bash** : `source .venv/Scripts/activate`

### Commandes tests

```bash
# Suite stable hors intégration
python -m pytest -q --ignore=tests/integration

# Suite complète
python -m pytest

# Healthcheck environnement
python scripts/check_env.py
```

### Règles strictes

1. **Ne pas installer/mettre à jour** des packages sans motivation documentée
2. **Ne pas utiliser le Python MSYS2/MinGW** — source de conflits DLL
3. **Ne pas modifier le `PATH`** — utiliser `.venv` + `python -m pytest`

## Commandes Utiles

```bash
# Synchronisation
python scripts/sync.py --delta --gamertag MonGamertag

# Backup/Restore
python scripts/backup_player.py --gamertag MonGamertag
python scripts/restore_player.py --gamertag MonGamertag --backup ./backups/

# Backfill sessions (session_id, session_label dans match_stats)
python scripts/backfill_data.py --player MonGT --sessions
python scripts/backfill_data.py --all --sessions

# Backfill shots_fired/shots_hit (match_stats et match_participants)
python scripts/backfill_data.py --player MonGT --shots
python scripts/backfill_data.py --player MonGT --shots --force-shots
python scripts/backfill_data.py --player MonGT --participants-shots
python scripts/backfill_data.py --player MonGT --participants-shots --force-participants-shots

# Tests
.venv/Scripts/python.exe -m pytest tests/ -v
```

## Règles

1. Répondre en français
2. Utiliser Pydantic v2 pour valider les données
3. **Backfill** : Pour tout backfill ou création de nouvelles fonctions de backfill, utiliser `scripts/backfill_data.py`. Ne pas créer de scripts backfill séparés ; ajouter une option dédiée (ex. `--sessions`, `--killer-victim`) dans `backfill_data.py`.
4. **Pandas est PROSCRIT** - Utiliser **Polars** uniquement pour les DataFrames et séries (voir § Pandas interdit ci-dessous)
5. Utiliser DuckDBRepository pour l'accès aux données
6. **Documenter les décisions dans `.ai/thought_log.md` — OBLIGATOIRE avant de rendre la main** (voir § Workflow Agentique pour le format)
7. **SQLite est PROSCRIT** - Aucun fallback SQLite, tout le code doit utiliser DuckDB uniquement
8. **Streamlit** : Ne jamais utiliser `use_container_width=True` (déprécié). Utiliser `width="stretch"` à la place (`width="content"` si besoin). Pour `st.button`, `st.image`, `st.plotly_chart`, etc.
9. **Plotly** : Tout `st.plotly_chart` doit inclure `config=` (utiliser `PLOTLY_CLEAN_CONFIG` ou `PLOTLY_STATIC_CONFIG` de `src/ui/streamlit_modern.py`)
10. **Fragments** : Préférer `@fragment_if_available` (de `src/ui/streamlit_modern.py`) pour les sections interactives multi-charts
11. **Coéquipiers** : Charger les stats coéquipiers depuis `shared.match_participants` (pas les DBs individuelles)
12. **SyncScope** : Ne jamais passer 30+ kwargs individuels aux fonctions backfill/sync. Toujours construire un `SyncScope` et le passer via `scope=`. Les kwargs legacy sont marqués `LEGACY` et seront supprimés.
13. **Taille max fonctions** : 80 lignes (docstring incluse). Au-delà → extraire une sous-fonction nommée. Pas d'exception sans `# noqa: PLR0915` + commentaire justificatif. Violations existantes dans `scripts/size_baseline.txt` (dette documentée).
14. **Taille max modules** : 500 lignes. Whitelist dans `scripts/check_code_size.py` (`src/ui/i18n/`, `src/data/sync/migrations.py`). Si un module approche 500L → créer un sous-module **avant** d'atteindre la limite.
15. **Arguments max** : 5 par fonction. Au-delà → `dataclass`, `TypedDict` ou `SyncScope`. Violations existantes annotées `# noqa: PLR0913`.
19. **Typage structuré** — règle de décision :
    - `BaseModel` (Pydantic v2) → données qui traversent une frontière externe (API, JSON, CSV) et nécessitent validation/coercion
    - `@dataclass(frozen=True)` → structures internes entre modules avec types explicites et immuabilité souhaitée (contextes UI, paramètres de page, résultats d'analyse)
    - `TypedDict` → uniquement pour annoter des dicts dont la structure est imposée par une lib externe (plotly layout, kwargs d'API tierce…)
    - Dict nu → uniquement dans du code throwaway ou des fonctions locales < 10L
16. **Complexité cyclomatique** : max 12 (McCabe C901, enforced via Ruff). Violations existantes annotées `# noqa: C901`. Chaque `# noqa` restant = dette à réduire.
17. **Responsabilité unique** : le nom d'une fonction doit tenir en 1 verbe + 1 complément. `render_and_compute_X()` → 2 responsabilités → diviser en `compute_X()` + `render_X()`. Indicateurs suspects : `_and_`, `_with_`, `_then_` dans un nom de fonction. Test automatique : `tests/test_code_quality.py::test_no_srp_violation_in_function_names`.
18. **docs/FR/ — règle de synchronisation** : tout commit qui modifie un fichier dans `docs/` doit inclure la mise à jour du fichier correspondant dans `docs/FR/` si ce fichier existe. Les deux commits peuvent être séparés mais doivent être dans le même PR.
20. **Couleurs dans `apps/web/`** — Aucun hex (`#RRGGBB`) ni classe Tailwind de couleur (`text-red-*`, `bg-green-*`, etc.) dans `apps/web/src/features/` ou `apps/web/src/components/` sauf exceptions justifiées par commentaire. Toute couleur sémantique doit passer par `tokenCssVar(token)` (JSX), `resolveToken(token)` (Plotly/SVG) ou `getSeriesColors(n, tokens[])` (séries). Les palettes brutes sont centralisées dans `apps/web/src/lib/accessibility/palettes/`. Exceptions tolérées : couleurs de rareté Halo (Battlepass, `rarity.ts`), couleurs structurelles de layout SVG (fond de piste, bordure), couleurs UI génériques sans signification métier (liked/rose, warning/amber dans les badges d'état système).

## ⛔ Pandas interdit (règle critique)

- **Aucun** `import pandas` ni `import pandas as pd` dans le code applicatif (analyse, UI, sync, repositories, scripts).
- **Polars uniquement** : `import polars as pl` ; utiliser `pl.DataFrame`, `pl.Series`, `pl.LazyFrame`.
- À la frontière avec des librairies qui exigent du NumPy/Pandas (ex. certains composants Streamlit/Plotly), convertir au dernier moment avec `.to_pandas()` ou `.to_numpy()` et ne pas faire remonter du Pandas dans les modules métier.

## ⛔ SQLite interdit (règle critique)

- **Aucun** `import sqlite3` ni `sqlite3.connect()` dans le code applicatif (UI, sync, repositories, loaders).
- **Aucun** fallback sur une base `.db` (SQLite) : si une base est attendue, elle doit être `.duckdb`.
- **Aucun** usage de `sqlite_master` : utiliser `information_schema.tables` (DuckDB).
- **Seules exceptions** : les scripts de **migration** qui lisent l’ancien SQLite pour alimenter DuckDB (`recover_from_sqlite.py`, `migrate_player_to_duckdb.py`). Ils restent les seuls autorisés à ouvrir un fichier `.db`.


## Architecture Multi-Joueurs (v5.1)

Chaque joueur a sa propre DB : `data/players/{gamertag}/stats.duckdb` (enrichissements uniquement).

**Données partagées** : Toutes les stats de matchs, médailles, events, killer/victim et xuid_aliases sont dans `shared_matches_v2.duckdb`.

**Pour afficher les stats d'un coéquipier** sur des matchs communs :
1. Identifier les `match_id` communs via `shared.match_participants`
2. Charger les stats du coéquipier depuis `shared.match_participants` avec son xuid
3. Le sync écrit dans player DBs : `player_match_enrichment` + `personal_score_awards` uniquement

## Stack Technique

| Composant | Usage |
|-----------|-------|
| **DuckDB** | Moteur de requêtes OLAP |
| **Polars** | DataFrames et séries (Pandas interdit) |
| **Pydantic v2** | Validation des données |
| **Streamlit** | Interface utilisateur |
| **SPNKr** | API Halo Infinite |
| **SyncScope** | Flags sync/backfill centralisés (`src/data/sync/scope.py`) |
| **RAG / MCP** | Recherche sémantique dans la doc + serveur MCP pour Cursor (`src/ai/`) |

## Couche `src/ai/` (outillage développeur)

| Fichier | Rôle |
|---------|------|
| `rag.py` | `HaloKnowledgeBase` — indexation + recherche sémantique (ChromaDB) |
| `_rag_models.py` | Modèles Pydantic : `RAGConfig`, `Document`, `SearchResult` |
| `_rag_chunker.py` | `TextChunker` — découpage de docs en chunks |
| `_rag_github.py` | `GitHubIndexer` — indexation de repos GitHub |
| `mcp_server.py` | Serveur MCP (protocole Model Context Protocol) pour Cursor |

**Règle** : `src/ai/` est réservé à l'**outillage développeur** (RAG docs, MCP). Aucune logique métier Halo ne doit y résider. Pas d'import de `src.data` ni de `src.ui` dans ce module.

## Règle `src/analysis/` vs `src/data/services/`

| Package | Rôle | Règle |
|---------|------|-------|
| `src/analysis/` | **Algorithmes purs** — transformations stateless | Entrée : `pl.DataFrame` / listes · Sortie : résultats calculés · **0 accès DB**, 0 Streamlit |
| `src/data/services/` | **Orchestration** — combine accès repo + algos | Prend un `DuckDBRepository` + paramètres · délègue les calculs à `analysis/` · retourne dataclasses ou `pl.DataFrame` |

**Règle de décision** : si la fonction n'a pas besoin de toucher la DB → `analysis/`. Si elle doit interroger le repo ET calculer → `services/`.

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

## 🔍 Diagnostic de revue de code

Avant d'écrire ou modifier du code, l'agent IA doit vérifier que ses changements ne réintroduisent pas les anti-patterns éliminés lors du refactoring v5.

### Seuils obligatoires

| Métrique | Seuil max | Action si dépassé |
|----------|:---------:|-------------------|
| **Lignes par fichier** | **500 L** | Découper en modules (mixins, `*_logic.py`, `*_data.py`) |
| **Lignes par fonction** | **80 L** | Extraire des sous-fonctions nommées |
| **Copies d'un même pattern** | **≤ 2** | Centraliser dans un helper/constante |
| **Magic numbers/strings** | **0** | Utiliser des enums (`Outcome.WIN`) ou constantes nommées |
| **Fonctions/modules morts** | **0** | Supprimer immédiatement avec leurs tests et imports |
| **Connexions DB bare** | **0** | Toujours via context manager (`duckdb_read_only()` / `duckdb_read_write()`) |

### Anti-patterns interdits

1. **"Dead code museum"** — Conserver du code mort "au cas où" (fonctions retournant `[]`, `None`, `False`, modules entiers inutilisés)
2. **"Compatibility guard forever"** — Garder des branches `if POLARS_AVAILABLE:` ou `if SQLITE_MODE:` après migration ; toute guard de compatibilité doit avoir une **date d'expiration** en commentaire
3. **"God file"** — Fichier >500L mélangeant des responsabilités distinctes → découper en mixins ou modules séparés
4. **"Swiss-army function"** — Fonction qui fait init + logique + IO + render → extraire en fonctions à responsabilité unique
5. **"Copy-paste config"** — Même dict/string copié dans 3+ endroits → constante centralisée (ex: `PLOTLY_CLEAN_CONFIG`, `DATE_FORMAT_FR`)
6. **"Bare connect"** — `duckdb.connect()` sans `with` → fuite de ressource
7. **"Manual coercion dataclass"** — `@dataclass` + parsing JSON ad hoc 160L → Pydantic v2 `model_validate()`
8. **"Magic integer"** — `outcome == 2` sans contexte → `Outcome.WIN`  
9. **"Logique métier dans l'UI"** — Calculs purs mélangés aux appels Streamlit → séparer en `*_logic.py` testable sans Streamlit
10. **"Alias inutile"** — `_func = func` en tête de fichier sans raison → import direct
11. **"God __init__"** — `__init__.py` qui importe massivement ses propres sous-modules → `KeyError: 'src.xxx'` lors des hot-reloads Streamlit. Règle : un `__init__.py` ne doit **jamais** importer depuis ses propres sous-modules (sauf si des dizaines de callers existants utilisent déjà `from src.pkg import X` — dans ce cas les imports lazy depuis les fonctions sont tolérés, mais les imports module-level dans `streamlit_app.py` doivent pointer vers le sous-module direct). Test : `tests/test_imports.py`.

### Patterns à appliquer

| Situation | Pattern recommandé | Exemple dans le projet |
|-----------|-------------------|------------------------|
| God class >500L | **Mixins MRO** | `engine.py` → 8 mixins + `_protocol.py` (`_shared_writes`, `_performance`, `_skill_rating`, `_career`, `_aggregates`, `_match_processing`, `_engine_connections`, `_engine_schema`) |
| God function >80L | **Extract method** | `main()` 582L → `_initialize_app()`, `_load_and_filter_data()`, etc. |
| Page Streamlit avec logique | **Séparation UI/logique** | `session_compare.py` + `session_compare_logic.py` |
| Config/parsing complexe | **Pydantic v2** | `AppSettings(BaseModel)` avec `model_validate()` |
| Codes numériques | **IntEnum** | `Outcome(IntEnum): WIN=2, LOSS=3, TIE=1, DNF=4` |
| Connexions DB | **Context manager** | `duckdb_read_only(path)`, `duckdb_read_write(path)` |
| Constantes répétées | **Module dédié** | `PLOTLY_CLEAN_CONFIG`, `DATE_FORMAT_FR`, `CORE_STAT_COLUMNS` |
| Rendu chart avec error handling | **Context manager** | `safe_chart_render()` dans `src/ui/chart_utils.py` |

### Checklist de revue automatique

Avant chaque commit, l'agent doit vérifier :

- [ ] Aucun fichier créé/modifié ne dépasse **500 lignes**
- [ ] Aucune fonction ne dépasse **80 lignes**
- [ ] Pas de pattern dupliqué dans **3+ endroits** → centraliser
- [ ] Pas de **magic number** → enum ou constante
- [ ] Toute connexion DuckDB utilise un **context manager**
- [ ] Tout `st.plotly_chart` inclut `config=PLOTLY_CLEAN_CONFIG` ou `PLOTLY_STATIC_CONFIG`
- [ ] Pas de `import pandas` (sauf `.to_pandas()` à la frontière Streamlit/Plotly)
- [ ] Pas de `import sqlite3` (sauf scripts de migration)
- [ ] Logique métier testable **sans Streamlit** (pas de `st.*` dans les fonctions de calcul)
- [ ] Code mort supprimé (pas de fonctions retournant toujours `None`/`[]`/`False`)
- [ ] Guards de compatibilité retirés si la migration est terminée

---

## Stratégie de branches Git

### Règle fondamentale : 1 tâche = 1 branche, N commits

```
# ✅ Correct — phases séquentielles = commits sur une branche
git checkout -b refactor/cleanup-all
git commit -m "refactor(phase1): dead code cleanup"
git commit -m "refactor(phase2): DRY violations"
git commit -m "refactor(phase3): split god classes"
git commit -m "refactor(phase4): quality patterns"

# ❌ Interdit — phases séquentielles = branches séparées
git checkout -b refactor/phase1-dead-code-cleanup  # puis
git checkout -b refactor/phase2-dry-violations      # puis
git checkout -b refactor/phase3-god-class-splits    # puis
git checkout -b refactor/phase4-quality-patterns    # → oblige une rebase/merge manuelle
```

### Quand créer plusieurs branches ?

Uniquement si les tâches sont **indépendantes et parallélisables** (ex : deux features sans dépendance). Si les tâches sont séquentielles, tout va sur **une seule branche** avec plusieurs commits.

### Règles d'application

1. **Toujours vérifier la branche courante** avant de committer : `git branch --show-current`
2. **⛔ JAMAIS travailler sur `main`** — sans aucune exception. Si la branche courante est `main`, créer une branche de travail avant toute modification.
3. **Toute nouvelle fonction/feature/fix** → créer une nouvelle branche depuis la branche courante (`git checkout -b <type>/<nom>`), jamais travailler directement sur la branche parente.
4. **⛔ Ne jamais changer de branche** si un travail différent est déjà en cours sur la branche courante — interrompre la tâche et informer l'utilisateur pour éviter tout conflit entre agents.
5. **Si aucun nom de branche n'est spécifié** par l'utilisateur, demander ou proposer un nom avant de créer
6. **Entre sessions** : relire les commits existants (`git log --oneline -10`) pour reprendre sur la bonne branche
7. **Résumé** : une branche pour le sujet, des commits pour les étapes — pas l'inverse

---

## Modules Supprimés (v4.1)

Les anciens modules legacy ont été supprimés lors de la migration v4.1 :
- `src/db/loaders.py` — supprimé, remplacé par `DuckDBRepository`
- `src/db/loaders_cached.py` — supprimé
- `src/data/repositories/legacy.py` — supprimé
- `src/data/repositories/shadow.py` — supprimé
- `src/data/repositories/hybrid.py` — supprimé

**Tout le code doit utiliser `DuckDBRepository`** (`src/data/repositories/duckdb_repo.py`).

## Serveurs MCP Disponibles

Si les MCPs sont configurés, les utiliser :

**duckdb** :
- Exécuter SQL directement sur les données Halo
- `ATTACH 'data/warehouse/metadata.duckdb' AS meta`

**browser** (cursor-ide-browser) :
- Tester l'app Streamlit visuellement
