# Plan d'optimisation du pipeline de sync — LevelUp

> Document de référence pour les 7 axes d'optimisation identifiés après benchmarks (mars 2026).
> **Prérequis** : toujours travailler sur une branche dédiée, tester chaque phase de façon isolée.
> Les fichiers cibles sont critiques — toute régression casse la sync en production.
>
> **Périmètre** : sync PvP uniquement. La sync PvE (Firefight / `shared_pve.duckdb`) est **hors périmètre** de ce plan.
> **Tolérance aux pannes** : un match échoué (API timeout, erreur transform) n'est jamais marqué comme traité ;
> il est automatiquement re-fetché au prochain lancement de sync (pas d'ajout à `existing_ids` sans succès complet).

---

## Contexte et résultats de benchmark

| Run | Config | Engine | Wallclock | Gain |
|-----|--------|:------:|:---------:|:----:|
| Baseline | parallel=5, rps=10, batch=10 | 65.5s | 66.6s | — |
| Run 2 | parallel=10, rps=10, batch=25 | 61.7s | 63.1s | -5.8% |
| Run 3 | parallel=10, rps=15, batch=25 | 59.6s | 61.0s | **-9.0%** ✅ prod |
| Run 4 | parallel=15, rps=20, batch=30 | 59.1s | 60.4s | -9.8% |

**Plateau confirmé à Run 3** (parallel=10, rps=15, batch=25). Run 4 n'apporte rien — le
goulot d'étranglement restant est structurel, pas paramétrique.

**Conditions de test** : BDD vide (cold start), réseau domestique standard, CPU en usage normal,
~22 matchs. Résultats empiriques acceptés comme baseline — pas de variance inter-run documentée.

---

## Vue d'ensemble des 7 axes

| # | Axe | Gain estimé | Effort | Risque |
|---|-----|:-----------:|:------:|:------:|
| 1 | Post-sync parallélisé | 30-50% phase post-sync | Moyen | Moyen |
| 2 | Handle conflict shared_matches | Fiabilité + ~5% | Moyen | Moyen (*) |
| 3 | Semaphore I/O séparé du CPU | +15-20% (gros volumes) | Élevé | Moyen |
| 4 | Citations batch SQL | +25% sur 200+ matchs | Moyen | Moyen |
| 5 | Transformers run_in_executor | +10-25% CPU overlap | Moyen | Moyen |
| 6 | LUSR UPSERT vectorisé | +5-10% phase LUSR | Faible | Faible |
| 7 | batch_commit_size adaptatif | +3-5% | Très faible | Faible |

**Ordre d'implémentation recommandé** : 7 → 6 → 2 → 4 → 1 → 5 → 3
(du plus simple/sûr au plus complexe, en repoussant les axes qui changent l'architecture async)

(*) Axe 2 : risque reclassé Moyen car l'Option A (connexion directe R/O, pas d'ATTACH) est un
changement local à 2 fonctions (`citations_backfill`, `sessions_backfill`) avec rollback facile.

---

## Pré-requis transverses

Avant d'implémenter un axe, valider explicitement les points suivants :

1. **Conflits DuckDB** : distinguer indépendance fonctionnelle et indépendance de connexions/écritures. Deux traitements qui écrivent dans `player_match_enrichment` ne sont **pas** parallélisables sans stratégie de sérialisation.
2. **ATTACH post-sync** : l'objectif cible n'est pas seulement `citations_backfill`, mais **tout** le post-sync (`sessions`, `citations`, calculs secondaires) — aucun `ATTACH shared_matches` opportuniste depuis `player_conn` si une connexion directe sur `shared_matches` existe encore.
3. **Citations** : ne pas partir du principe qu'un batch SQL couvre 100% du moteur. Le moteur actuel mélange SQL, DataFrame Polars et fonctions Python custom.
4. **Verrous async** : les locks existants (`_db_lock`, `_shared_db_lock`, `_pve_db_lock`) sont des **`asyncio.Lock()`** — ils protègent les écritures entre coroutines mais **ne protègent PAS les accès depuis un `ThreadPoolExecutor`**. Tout passage en `run_in_executor()` doit ajouter des `threading.Lock()` ou une queue de persistance dédiée.
5. **Sentinelles de config** : ne pas réutiliser une valeur déjà porteuse d'une sémantique runtime (ex. `batch_commit_size=0` signifie déjà « commit final uniquement »).
6. **Thread-safety MetadataResolver** : `self._metadata_resolver` contient un cache `dict` et une connexion DuckDB R/O, tous deux **non protégés par lock**. Avant tout `run_in_executor()` utilisant le resolver (Axes 3 et 5), ajouter un `threading.RLock()` dans `MetadataResolver`. **Prérequis bloquant** pour les axes 3 et 5.

### Classification des axes

- **Optimisations sûres / locales** : 7, 6
- **Fiabilisation structurelle préalable** : 2
- **Optimisation avec phase de discovery obligatoire** : 4
- **Refactorings async / concurrence** : 1, 5, 3

---

## Axe 1 — Post-sync séquentiel → parallèle

### État actuel

Fichier : [`src/data/sync/engine.py`](../src/data/sync/engine.py), fonction `_run_post_sync_compute()` (sync, ~l. 411).

```
_run_post_sync_compute()
  [séq] 1. batch_compute_performance_scores()  → ferme shared PUIS rouvre en interne via _get_shared_connection()
  [séq] 2. backfill_sessions_for_player()      → ATTACH shared depuis player_conn (READ_ONLY)
  [séq] 3. backfill_citations_for_player()     → ATTACH shared via ensure_shared_attached() → ⚠️ conflit si handle R/W encore ouvert
  [séq] 4. _compute_dominance_post_sync()      → duckdb.connect(shared, read_only=True) isolé ✓
```

Chaque étape attend son résultat, puis la suivante démarre. La fonction est **synchrone** (pas `async`).
**Aucune de ces 4 étapes ne dépend du résultat métier des autres**, mais elles ne sont **pas**
totalement indépendantes côté stockage :

- `batch_compute_performance_scores()` écrit dans `player_match_enrichment`
- `backfill_sessions_for_player()` écrit aussi dans `player_match_enrichment`
- `_compute_dominance_post_sync()` écrit aussi dans `player_match_enrichment`
- `backfill_citations_for_player()` écrit dans `match_citations`

Le goulot n'est donc pas seulement séquentiel ; il est aussi lié au partage de connexions et à la
cohabitation d'écritures sur la DB joueur.

### Objectif

Réduire le temps post-sync **sans introduire de concurrence d'écriture non maîtrisée** sur la DB
joueur. Le parallélisme visé doit être **partiel et encadré**, pas un `gather()` brut des 4 étapes.

### Contraintes et guard-rails

- L'axe 1 dépend de l'axe 2 : tant que `sessions` et `citations` peuvent encore faire un
    `ATTACH shared_matches` sur `player_conn`, le parallélisme post-sync reste fragile.
- `batch_compute_performance_scores`, `backfill_sessions_for_player` et
    `_compute_dominance_post_sync` ne doivent pas écrire en parallèle dans
    `player_match_enrichment` sans stratégie explicite.
- `_compute_dominance_post_sync` est déjà isolée côté lecture shared (`duckdb.connect(..., read_only=True)`),
    mais pas côté écriture player DB.
- `backfill_sessions_for_player` n'écrit **pas** dans une table `sessions` séparée dans ce flow ;
    il upsert `session_id` / `session_label` dans `player_match_enrichment`.
- **Le LUSR ne doit PAS être inclus** dans ce gather — il est appelé après
  `_detach_shared_from_player_conn()` à la fin de `_sync_internal`, et dépend du cleanup
  de `_shared_connection`.

### Plan d'implémentation

**Étape 1a — Modèle de concurrence retenu : variante conservatrice**

Paralléliser uniquement les traitements qui n'écrivent
pas simultanément dans `player_match_enrichment` : `citations` en parallèle d'un bloc
sérialisé `perf → sessions → dominance`.

La variante structurée (read/compute parallèle + write sérialisé) est un over-engineering
pour un gain marginal — elle ne sera pas implémentée sauf si le profiling post-Axe 1 montre
que le goulot est dans les lectures.

**Étape 1b — Transformer `_run_post_sync_compute` en async**

```python
# Avant (sync):
def _run_post_sync_compute(self, options: SyncOptions) -> None: ...

# Après (async):
async def _run_post_sync_compute(self, options: SyncOptions) -> None: ...
```

Adapter l'appelant dans `_sync_internal` :
```python
if result.matches_inserted > 0:
    await self._run_post_sync_compute(options)
```

**Étape 1c — Factoriser chaque étape en wrappers isolés**

```python
async def _post_sync_perf_scores(self) -> int: ...
async def _post_sync_sessions(self) -> dict: ...
async def _post_sync_citations(self) -> dict: ...
async def _post_sync_dominance(self) -> dict: ...
```

Chaque wrapper :

- ouvre ses propres connexions locales si nécessaire ;
- ne dépend pas de `self._shared_connection` partagée ;
- documente explicitement sa zone d'écriture (`player_match_enrichment`, `match_citations`, etc.).

**Étape 1d — Orchestration partielle, pas gather global**

```python
async def _run_post_sync_compute(self, options: SyncOptions) -> None:
    loop = asyncio.get_event_loop()
    
    # Fermer shared_connection une seule fois avant le scatter
    if self._shared_connection is not None:
        with contextlib.suppress(Exception):
            self._shared_connection.close()
        self._shared_connection = None

    citations_task = loop.run_in_executor(None, self._post_sync_citations_sync)
    perf_result = await loop.run_in_executor(None, self._post_sync_perf_scores_sync)
    sessions_result = await loop.run_in_executor(None, self._post_sync_sessions_sync)
    dominance_result = await loop.run_in_executor(None, self._post_sync_dominance_sync)
    citations_result = await citations_task
```

Ce pattern garde un vrai recouvrement utile tout en évitant 3 écritures concurrentes dans la même
table de la DB joueur.

### Logging attendu (avant → après)

```
# Avant (séquentiel):
[INFO] Performance scores calculés en batch : 22
[INFO] Sessions recalculées post-sync : 22 mises à jour, 0 ignorées
[INFO] citations_backfill: ✅ 22/22 matchs avec citations
[INFO] Dominance flags post-sync : 22 traités (domination: 0, humiliation: 0)

# Après (partiellement parallèle):
[INFO] post_sync: démarrage (citations en parallèle, writes player DB sérialisées)
[INFO] post_sync: terminé en 1.8s (perf=22 sessions=22u/0skip citations=22/22 dominance=22x0d/0h)
```

### Tests

```python
# tests/test_post_sync_parallel.py

def test_post_sync_citations_overlap_with_player_db_sequence():
    """citations peut s'exécuter pendant la séquence perf/sessions/dominance."""
    ...

def test_post_sync_exception_non_bloquante():
    """Une exception dans une tâche ne bloque pas les autres."""
    # Mock citations_backfill pour lever RuntimeError
    # Vérifier que perf, sessions, dominance ont quand même tourné
    ...

def test_post_sync_shared_connection_closed_before_gather():
    """_shared_connection est close() avant le gather."""
    # Vérifier via mock que close() est appelé exactement une fois
    ...

async def test_post_sync_no_parallel_writes_on_player_match_enrichment():
    """Perf, sessions et dominance restent sérialisés côté player DB."""
    ...

async def test_post_sync_duration_reduced():
    """Le recouvrement partiel réduit la durée totale sans gather global."""
    ...
```

---

## Axe 2 — Handle conflict shared_matches (ATTACH/DETACH)

### État actuel

Le warning suivant apparaît systématiquement en post-sync :
```
[WARNING] shared_matches.duckdb conflit de handle, libération et retry…
          (Binder Error: Unique file handle conflict: Cannot attach
          "shared_matches_v2" - already attached by database "shared")
[WARNING] shared_matches.duckdb : retry échoué — shared non disponible
[WARNING] shared_connection ou xuid manquant pour batch performance scores
[INFO] Performance scores calculés en batch : 0
```

**Cause racine** : `_get_shared_connection()` dans [`src/data/sync/_engine_connections.py`](../src/data/sync/_engine_connections.py) ouvre une connexion R/W sur `shared_matches.duckdb`.
Ensuite, le post-sync peut faire un `ATTACH shared_matches.duckdb` depuis la connexion joueur
(`citations_backfill`, mais aussi `sessions_backfill`). DuckDB interdit ce mélange entre une
connexion directe déjà ouverte et un `ATTACH` concurrent sur le même fichier.

Le flow actuel :
```
1. _process_matches → self._shared_connection ouvert (R/W)
2. _run_post_sync_compute:
   a. batch_compute_performance_scores → ferme self._shared_connection
    b. backfill_sessions_for_player → peut ATTACH shared depuis player_conn
    c. backfill_citations_for_player → ATTACH shared depuis player_conn → CONFLIT si (a) a raté
   d. _compute_dominance_post_sync → duckdb.connect(shared, read_only=True) ✓
3. _run_lusr_post_sync → ferme shared + rouvre ✓
```

### Objectif

Garantir qu'au moment où `citations_backfill` attache `shared_matches`, aucun autre handle R/W
n'est ouvert. La stratégie propre : **ne jamais mélanger un ATTACH depuis la player_conn
et une connexion directe sur shared_matches**.

### Plan d'implémentation

**Option A (recommandée) — citations en connexion directe, sans ATTACH**

Créer un helper partagé de lecture directe `shared_matches` en R/O et l'utiliser dans :

- `backfill_citations_for_player`
- `backfill_sessions_for_player`
- tout futur calcul post-sync DB-only qui n'a pas besoin d'un `ATTACH` sur `player_conn`

Le principe est d'ouvrir `shared_matches.duckdb` via `duckdb.connect(str(shared_path), read_only=True)`
et de croiser les données via requêtes directes, Polars ou Arrow, plutôt que via `ATTACH ... FROM`.

```python
# Avant (ensure_shared_attached retourne un alias str|None, PAS un context manager) :
shared_alias = ensure_shared_attached(conn, shared_path, aliases=("shared",))
match_ids = conn.execute(f"SELECT ... FROM {shared_alias}.match_participants ...").fetchall()

# Après (connexion directe R/O, pas d'ATTACH) :
with duckdb.connect(str(shared_path), read_only=True) as shared_ro:
    match_ids = shared_ro.execute("SELECT ... FROM match_participants ...").fetchall()
```

Avantages :
- Élimine les `ATTACH` post-sync sur la player_conn → plus de conflit
- `shared_ro` est R/O → compatible avec toute autre connexion R/O ouverte en parallèle
- Cohérent avec `_compute_dominance_post_sync` qui fait déjà ça

**Option B — cleanup systématique avant citations/sessions**

S'assurer que toute connexion R/W à shared est fermée ET que la player_conn n'a aucun
ATTACH avant d'appeler `backfill_citations_for_player` ou `backfill_sessions_for_player`.

Cette option est déjà partiellement en place mais fragile (le retry échoue).

### Recommandation

Traiter l'axe 2 comme **pré-requis** de l'axe 1. Tant que `sessions` et `citations` n'utilisent pas
la même stratégie d'accès à shared, le post-sync parallèle restera instable.

### Logging attendu

```
# Avant:
[WARNING] shared_matches.duckdb conflit de handle, libération et retry…

# Après: plus aucun warning de ce type
[DEBUG] citations_backfill: shared_matches ouvert en R/O direct (pas d'ATTACH)
```

### Tests

```python
# tests/test_citations_shared_handle.py

def test_citations_no_attach_on_player_conn():
    """citations_backfill n'émet aucun ATTACH depuis la connexion joueur."""
    # Spy sur player_conn.execute
    # Vérifier qu'aucun appel ne contient "ATTACH"
    ...

def test_sessions_no_attach_on_player_conn():
    """sessions_backfill n'émet aucun ATTACH depuis la connexion joueur."""
    ...

def test_citations_readonly_shared_connection():
    """shared_matches est ouvert en read_only=True."""
    # Mock duckdb.connect, vérifier read_only=True
    ...

def test_no_handle_conflict_warning_in_full_sync(caplog):
    """Sync complète avec 22 matchs : aucun WARNING 'conflit de handle'."""
    # Test d'intégration (avec DB de test)
    # Vérifier caplog n'a aucun record WARNING contenant "conflit de handle"
    ...
```

---

## Axe 3 — Semaphore fetch/CPU séparés

### État actuel

Fichier : [`src/data/sync/_match_processing.py`](../src/data/sync/_match_processing.py) (~l. 73 + l. 114).

```python
# l. 73 :
semaphore = asyncio.Semaphore(options.parallel_matches)  # Unique pour tout

# l. 114-115 :
async def _bounded(mid: str) -> dict:
    async with semaphore:  # ← Gate fetch + transform CPU sous le même lock
        return await self._process_single_match(client, mid, options)
```

`_process_single_match` contient :
1. **Fetch API** (I/O, ~1-4s) — réseau, libérable facilement
2. **`transform_match_stats()`** (CPU, ~50-200ms) — bloque l'event loop
3. **Écritures DB** (I/O synchrone, ~20-50ms) — DuckDB

Les 3 phases sont sous le même semaphore → à 10 slots, si 8 matchs sont en phase CPU,
seulement 2 peuvent fetcher.

### Objectif

Séparer le semaphore réseau (large : 15-20) du semaphore CPU (étroit : 4-6) et exécuter
les transformations dans un `ThreadPoolExecutor` dédié.

### Statut

Cet axe est un **chantier d'architecture**, pas une simple optimisation locale. Le design final doit
préciser où s'arrêtent exactement :

- la phase fetch réseau ;
- la phase transformation CPU pure ;

> ⚠️ **Prérequis non vérifié** : le rate limiter est **délégué à SPNKr** (`HaloInfiniteClient(requests_per_second=...)`).
> Avant de passer `parallel_fetch` à 15-20, vérifier que SPNKr gère correctement cette concurrence
> (pas de token bucket interne saturé). Sinon, le gain réseau sera nul et le refactoring inutile.
- la phase écriture DB, qui doit rester sérialisée ou passer par une queue dédiée.

### Plan d'implémentation

**Étape 3a — Ajouter les paramètres dans SyncOptions**

```python
@dataclass
class SyncOptions:
    # Existants
    parallel_matches: int = 10
    requests_per_second: int = 15
    
    # Nouveaux
    parallel_fetch: int = 15         # Slots I/O réseau (>= parallel_matches)
    parallel_transform_workers: int = 4  # Threads CPU pour transform
```

**Étape 3b — Refactoriser `_process_matches`**

```python
# Deux semaphores distincts
fetch_sem = asyncio.Semaphore(options.parallel_fetch)     # Larder
cpu_sem = asyncio.Semaphore(options.parallel_matches)     # Resserré

loop = asyncio.get_event_loop()
transform_executor = ThreadPoolExecutor(
    max_workers=options.parallel_transform_workers,
    thread_name_prefix="sync_transform",
)

async def _bounded(mid: str) -> dict:
    async with fetch_sem:
        raw_data = await self._fetch_match_data_only(client, mid, options)
    # Relâche fetch_sem immédiatement, gate CPU maintenant
    async with cpu_sem:
        return await loop.run_in_executor(
            transform_executor,
            self._transform_and_insert_sync, mid, raw_data, options
        )
```

**Étape 3c — Extraire 3 phases explicites**

- `_fetch_match_data_only(client, match_id, options) -> RawMatchData` : async, I/O pur
- `_transform_match_sync(match_id, raw_data, options) -> PreparedMatchData` : sync, CPU pur
- `_write_match_data_async(prepared_data) -> dict` : async, écritures DB sous les verrous existants

> ⚠️ **Risque : thread-safety des connexions DuckDB**
> Les écritures DuckDB ne sont pas thread-safe. Il faut donc **interdire** que la phase exécutée
> dans l'executor fasse la moindre écriture DB. Le thread pool doit rester limité aux transformations
> pures ; la persistance doit revenir sur l'event loop sous `_db_lock` / `_shared_db_lock`.

Cette contrainte rend cet axe le plus **complexe et risqué**. À implémenter en dernier.

### Logging

```
[DEBUG] sync_executor: fetch_sem=15 cpu_sem=10 transform_workers=4
[DEBUG] match=abc123 fetch_done 1.2s | transform 0.08s | write 0.03s
[INFO] Batch 25 matchs: fetch_peak=12 cpu_peak=4 total=32.1s
```

### Tests

```python
# tests/test_match_processing_semaphores.py

async def test_fetch_sem_released_before_transform():
    """fetch_sem est relâché avant que transform ne démarre."""
    # Mesurer occupation du fetch_sem pendant la phase transform
    ...

async def test_transform_executor_thread_count():
    """Nombre de threads parallèles ≤ parallel_transform_workers."""
    ...

async def test_db_write_thread_safety():
    """Les writes restent sérialisées même si les transforms partent en executor."""
    ...
```

---

## Axe 4 — Citations : boucle séquentielle → batch SQL

### État actuel

Fichier : [`src/data/citations_backfill.py`](../src/data/citations_backfill.py), ~l. 72.

```python
for i, match_id in enumerate(match_ids):
    try:
        n = engine.compute_and_store_for_match(match_id, conn=conn)
        if n > 0:
            citations_computed += 1
    except Exception as exc:
        logger.debug("citations_backfill: skip %s… → %s", match_id[:8], exc)
    if i > 0 and i % 50 == 0:
        logger.info("  [%s/%s] %s matchs traités", i, len(match_ids), citations_computed)
```

Pour N matchs : N appels Python séquentiels à `CitationEngine.compute_and_store_for_match()`.
Chacun déclenche 2-3 requêtes SQL sur shared.

### Objectif

Réduire significativement le coût de calcul des citations **sans réécrire d'un coup tout le moteur**.
Le batch SQL ne doit viser que les mappings compatibles avec une vectorisation fiable.

### Analyse préliminaire requise

Avant d'implémenter, lire [`src/analysis/citations/engine.py`](../src/analysis/citations/engine.py)
pour vérifier :
- Si `compute_and_store_for_match` peut être extrait en SQL pur sans état Python
- Ou si une logique Python est nécessaire entre matchs (interdépendances)

Si les calculs sont **indépendants** entre matchs (probable) → un batch SQL est possible.

En pratique, classer les mappings en 3 familles :

1. **Batch SQL immédiat** : `medal`, `stat`, une partie des `award`
2. **Batch hybride** : lecture groupée + calcul Python vectorisé / micro-batch
3. **Fallback séquentiel requis** : règles `custom`, dépendantes de `highlight_events`, `df_match`,
   `weapon_kills`, ou de la logique composite

### Plan d'implémentation (si compatible)

**Étape 4a — Ajouter une étape de discovery formelle**

Produire d'abord une matrice du type :

| mapping_type / règle | Batch SQL | Batch hybride | Fallback séquentiel |
|----------------------|-----------|---------------|---------------------|
| medal                | ✅        | —             | —                   |
| stat                 | ✅        | —             | —                   |
| award simple         | ✅/⚠️     | ✅            | —                   |
| custom               | —         | ⚠️            | ✅                  |
| highlight_events     | —         | ⚠️            | ✅                  |
| weapon_kills         | ⚠️        | ✅            | ✅                  |

Ne coder la phase batch qu'après cette matrice.

**Étape 4b — Créer `compute_and_store_for_matches_batch(match_ids, conn)` pour le sous-ensemble batchable**

```python
def compute_and_store_for_matches_batch(
    engine: CitationEngine,
    match_ids: list[str],
    conn: duckdb.DuckDBPyConnection,
    batch_size: int = 50,
) -> dict[str, int]:
    """Version batch partielle : calcule les citations batchables pour une liste de matchs.
    
    Découpe en micro-batches de `batch_size` pour limiter la pression mémoire.
    Chaque micro-batch est une transaction atomique.
    """
    total = len(match_ids)
    computed = 0
    for i in range(0, total, batch_size):
        chunk = match_ids[i:i + batch_size]
        placeholders = ",".join("?" * len(chunk))
        # Calcul SQL batch via CTE + INSERT pour medal/stat/award simple uniquement
        conn.execute(f"""
            INSERT INTO match_citations (match_id, citation_name_norm, value)
            WITH base AS (
                SELECT mp.match_id, mp.xuid, mp.kills, mp.deaths, ...
                FROM shared_or_ro.match_participants mp
                WHERE mp.match_id IN ({placeholders})
                  AND mp.xuid = ?
            ),
            scored AS (
                SELECT b.match_id, cm.citation_name_norm,
                       -- scoring logic ici
                FROM base b JOIN citation_mappings cm ON ...
            )
            SELECT * FROM scored
            ON CONFLICT DO UPDATE SET value = EXCLUDED.value
        """, (*chunk, xuid))
        computed += len(chunk)
        conn.commit()
    return {"matches_processed": total, "citations_computed": computed}
```

**Étape 4c — Conserver la boucle comme fallback exact**

```python
try:
    batchable_ids, fallback_ids = partition_match_ids_for_batch(engine, match_ids)
    batch_result = compute_and_store_for_matches_batch(engine, batchable_ids, conn)
    fallback_result = _compute_missing_citations_sequential(engine, fallback_ids, conn)
```

### Critère de succès

Le succès n'est pas forcément « 100% des citations en batch SQL ». Le vrai critère est :

- résultats strictement identiques au moteur actuel ;
- temps post-sync réduit de façon mesurable ;
- baisse de la charge Python sur les mappings simples.

### Logging

```
# Avant:
[INFO] citations_backfill (25332748…): 22 match(s) à traiter
[INFO] citations_backfill: ✅ 22/22 matchs avec citations

# Après:
[INFO] citations_backfill (25332748…): 22 match(s) — batch partiel + fallback exact
[INFO] citations_backfill: ✅ 22/22 matchs avec citations en 0.12s (18 batch, 4 fallback)
```

### Tests

```python
# tests/test_citations_batch.py

def test_batch_same_results_as_sequential(player_db_fixture, shared_db_fixture):
    """Le batch partiel + fallback produit exactement les mêmes citations que la boucle."""
    # Calculer en séquentiel → stocker résultat ref
    # Calculer en batch → comparer chaque ligne
    assert citations_batch == citations_sequential

def test_batch_handles_empty_list():
    result = compute_and_store_for_matches_batch(engine, [], conn)
    assert result == {"matches_processed": 0, "citations_computed": 0}

def test_batch_partial_failure_doesnt_rollback_others():
    """1 match corrompu ne rollback pas les 49 autres du batch."""
    ...

def test_batch_idempotent():
    """Exécuter deux fois le batch sur les mêmes matchs → pas de doublons."""
    compute_and_store_for_matches_batch(engine, match_ids, conn)
    compute_and_store_for_matches_batch(engine, match_ids, conn)
    count = conn.execute("SELECT COUNT(*) FROM match_citations").fetchone()[0]
    assert count == len(match_ids) * avg_citations_per_match

def test_custom_rules_still_use_fallback_path():
    """Les règles custom / highlight_events ne sont pas forcées en batch SQL."""
    ...
```

---

## Axe 5 — Transformers CPU-bound → run_in_executor

### État actuel

Fichier : [`src/data/sync/_match_processing.py`](../src/data/sync/_match_processing.py), dans `_process_known_match` (~l. 227) et `_save_player_data_new_match` (~l. 449).

```python
# Appelé depuis une coroutine asyncio, sous le semaphore :
match_row = transform_match_stats(
    stats_json,
    self._xuid,
    skill_json=skill_json,
    metadata_resolver=self._metadata_resolver,
)
```

`transform_match_stats` est une fonction Python pure (dict → dataclass, définie dans
`src/data/sync/transformers/_match.py`) qui peut prendre 50-200ms selon la complexité du match.
Elle bloque l'event loop pendant ce temps. **Aucun accès DB vérifié** — CPU pur, candidate idéale
pour `run_in_executor()`.

### Objectif

Libérer l'event loop pendant les transformations pour permettre d'autres I/O réseau.

### Plan d'implémentation

**Étape 5a — Identifier et lister toutes les fonctions CPU-bound dans le path**

Dans `_process_single_match` → `_process_known_match` / `_process_new_match` :
- `transform_match_stats()` — CPU
- `extract_aliases()` — CPU
- `extract_participants()` — CPU
- `_extract_personal_data()` — CPU

**Étape 5b — Wrapper asynchrone**

```python
async def _transform_single_match_async(
    self,
    loop: asyncio.AbstractEventLoop,
    stats_json: dict,
    skill_json: dict | None,
    options: SyncOptions,
) -> tuple[MatchRow | None, ...]:
    """Exécute les transformations CPU dans un executor."""
    return await loop.run_in_executor(
        None,  # Utilise le ThreadPoolExecutor par défaut
        self._transform_single_match_sync,
        stats_json, skill_json, options
    )

def _transform_single_match_sync(self, stats_json, skill_json, options):
    """Version synchrone des transformations (thread-safe, sans I/O)."""
    match_row = transform_match_stats(stats_json, self._xuid, skill_json=skill_json,
                                      metadata_resolver=self._metadata_resolver)
    alias_rows = extract_aliases(stats_json) if options.with_aliases else []
    # ...
    return match_row, alias_rows, ...
```

### Contraintes

- ⚠️ `self._metadata_resolver` **n'est PAS thread-safe en l'état** : cache `dict` non protégé + connexion DuckDB R/O sans lock. **Prérequis bloquant** : ajouter un `threading.RLock()` dans `MetadataResolver` avant tout `run_in_executor()` (cf. pré-requis transverse §6).
- Les fonctions de transformation (`transform_match_stats`, `extract_aliases`, `extract_participants`, `_extract_personal_data`) sont **vérifiées CPU-pures** — aucun `duckdb.connect()` ni `.execute()` dans leur code.
- Le `ThreadPoolExecutor` par défaut de Python utilise `min(32, os.cpu_count() + 4)` threads.
  Sur une machine 4-core = 8 threads → suffisant.

### Dépendance

L'axe 5 doit être traité **avant** l'axe 3 si l'on veut séparer proprement `fetch`, `transform` et
`write`. L'axe 3 peut alors réutiliser un pipeline déjà scindé.

> **Note** : l'axe 5 est indépendant de l'axe 1 (post-sync parallel). L'ordre recommandé
> `7 → 6 → 2 → 4 → 1 → 5 → 3` est correct car l'axe 1 touche le post-sync tandis que
> l'axe 5 touche le match processing — pas de dépendance croisée.

> **Prérequis bloquant** : thread-safety MetadataResolver (pré-requis transverse §6).

### Tests

```python
# tests/test_transform_executor.py

async def test_transform_doesnt_block_event_loop():
    """Pendant transform, d'autres coroutines peuvent progresser."""
    progress_ticks = []
    async def background_ticker():
        for _ in range(10):
            await asyncio.sleep(0.01)
            progress_ticks.append(time.time())

    asyncio.create_task(background_ticker())
    await transform_single_match_async(loop, stats_json, None, options)
    # Vérifier que progress_ticks n'est pas vide
    assert len(progress_ticks) > 5

async def test_transform_result_identical_sync_vs_async():
    """run_in_executor donne le même résultat que l'appel direct."""
    ...

def test_transform_functions_are_thread_safe():
    """transform_match_stats peut être appelé en parallèle depuis des threads."""
    with ThreadPoolExecutor(max_workers=8) as ex:
        futs = [ex.submit(transform_match_stats, stats, xuid, ...) for _ in range(20)]
        results = [f.result() for f in futs]
    assert all(r is not None for r in results)
```

---

## Axe 6 — LUSR UPSERT unitaire → vectorisé

### État actuel

Fichier : [`src/data/sync/_skill_rating.py`](../src/data/sync/_skill_rating.py), fonction `_upsert_lusr_ratings()`.

```python
for row in df_ratings.iter_rows(named=True):
    conn.execute(_LUSR_UPSERT_SQL, [row["match_id"], ..., row["rating_value"]])
# → N appels SQL pour N matchs
```

### Objectif

Remplacer la boucle par un `executemany()` ou un `INSERT FROM VALUES` batch.

### Plan d'implémentation

```python
# Option A : executemany (léger)
rows = df_ratings.select([
        "match_id", "rating_value", "rating_deviation",
        "playlist_group"
]).rows()
conn.executemany(_LUSR_UPSERT_SQL, rows)
conn.commit()

# Option B : préparer d'abord les lignes finales en Python, puis executemany()
# pour conserver la logique de delta séquentiel par playlist_group
```

### Contraintes

- Conserver le **guard-rail ±100 pts** avant insertion (actuellement dans la boucle).
    **Attention** : ce guard-rail dépend de l'ordre séquentiel via `prev_rating[playlist_group]`.
    Il ne peut pas être basculé naïvement en SQL stateless sans reproduire cette logique.
- `executemany()` dans DuckDB 1.4+ est supporté et efficace ; c'est l'option la plus réaliste.

### Recommandation

Ne pas viser un `INSERT SELECT` full-SQL pour cet axe. Le bon compromis est :

1. conserver le calcul séquentiel du `delta` en Python ;
2. accumuler les tuples finaux ;
3. remplacer la boucle `conn.execute(...)` par un `executemany(...)` unique.

### Tests

```python
# tests/test_lusr_batch_upsert.py

def test_executemany_same_results_as_loop(player_db_fixture):
    """batch upsert = même résultat que boucle pour 50 matchs."""
    ...

def test_guardrail_applied_in_batch():
    """Les lignes avec delta > 100 pts sont exclues en batch aussi."""
    df = polars.DataFrame({"match_id": ["x"], "lusr_before": [100], "lusr_after": [300]})
    result = _upsert_lusr_ratings_batch(conn, df, ...)
    assert conn.execute("SELECT COUNT(*) FROM match_skill_rank").fetchone()[0] == 0

def test_idempotent_upsert():
    """Appeler deux fois le batch → pas de doublon ni erreur."""
    ...
```

---

## Axe 7 — batch_commit_size adaptatif

### État actuel

Fichier : [`src/data/sync/models_sync.py`](../src/data/sync/models_sync.py), ~l. 39.

```python
batch_commit_size: int = 25  # Commit tous les 25 matchs
```

Utilisé dans `_maybe_batch_commit()` ([`src/data/sync/_match_processing_helpers.py`](../src/data/sync/_match_processing_helpers.py), ~l. 303) :
```python
def _maybe_batch_commit(self, n_inserted: int, batch_size: int) -> None:
    if batch_size > 0 and n_inserted % batch_size == 0:
        conn = self._get_connection()
        conn.commit()
        logger.debug("Commit intermédiaire après %s matchs", n_inserted)
```

### Objectif

Adapter automatiquement la taille de commit selon le volume de matchs attendu.

### Changement dans models_sync.py

**Convention** : `batch_commit_size = -1` signifie « auto ». `0` conserve sa sémantique
existante (commit final uniquement). Toute valeur `> 0` est un override explicite.

### Implémentation recommandée

```python
# Dans SyncOptions (models_sync.py) :
batch_commit_size: int = -1  # -1 = auto, 0 = commit final uniquement, >0 = override explicite

@staticmethod
def compute_optimal_batch_size(max_matches: int) -> int:
    """Batch size adaptatif : plus grand batch pour gros volumes."""
    if max_matches <= 25:
        return 0  # commit final uniquement
    if max_matches <= 100:
        return 25
    if max_matches <= 500:
        return 50
    return 100
```

```python
# Dans _sync_internal (engine.py) :
if options.batch_commit_size == -1:
    options = dc_replace(
        options,
        batch_commit_size=SyncOptions.compute_optimal_batch_size(options.max_matches),
    )
```

### Tests

```python
def test_batch_size_auto_25_matches():
    assert SyncOptions.compute_optimal_batch_size(25) == 0

def test_batch_size_auto_200_matches():
    assert SyncOptions.compute_optimal_batch_size(200) == 50

def test_batch_size_explicit_not_overridden():
    options = SyncOptions(batch_commit_size=50, max_matches=200)
    # batch_commit_size explicite → ne doit pas être écrasé
    ...

def test_batch_size_zero_keeps_no_intermediate_commit_semantics():
    options = SyncOptions(batch_commit_size=0, max_matches=200)
    # 0 conserve sa sémantique historique : commit final uniquement
    ...
```

---

## Stratégie de tests globale

### Suite de tests à créer

```
tests/
  perf/
    test_post_sync_parallel.py         # Axe 1
    test_citations_shared_handle.py    # Axe 2
    test_match_processing_semaphores.py # Axe 3
    test_citations_batch.py            # Axe 4
    test_transform_executor.py         # Axe 5
    test_lusr_batch_upsert.py          # Axe 6
    test_batch_commit_adaptive.py      # Axe 7
    conftest.py                        # Fixtures shared
```

### Fixtures communes (`conftest.py`)

```python
@pytest.fixture
def player_db_fixture(tmp_path):
    """DB joueur initialisée avec 22 matchs de test."""
    db_path = tmp_path / "stats.duckdb"
    # Créer tables, insérer matchs fixtures
    return db_path

@pytest.fixture
def shared_db_fixture(tmp_path):
    """shared_matches.duckdb avec les 22 mêmes matchs."""
    shared_path = tmp_path / "warehouse" / "shared_matches.duckdb"
    # Bootstrap + insertion matchs fixtures
    return shared_path

@pytest.fixture
def sample_stats_json():
    """JSON brut d'un match Halo valide (anonymisé)."""
    return json.loads((Path(__file__).parent / "fixtures" / "match_stats.json").read_text())
```

### Benchmark de régression

Avant chaque merge, exécuter :
```bash
python -m pytest tests/perf/ -v --tb=short
python -c "
import time, subprocess
t0 = time.time()
subprocess.run(['.venv/Scripts/python.exe', 'scripts/sync.py', '--delta', '--player', 'XxDaemongamerxX'])
print(f'BENCH: {time.time()-t0:.1f}s')
" 2>&1 | grep -E 'insérés|BENCH'
```

**Seuil de régression** : durée wallclock > 65s (baseline) sur 22 matchs = régression bloquante.

---

## Ordre d'implémentation recommandé

| Phase | Axes | Branche git | Durée estimée |
|-------|------|-------------|---------------|
| Phase 1 | 7 (batch_commit adaptatif) | `perf/batch-commit-auto` | < 1 session |
| Phase 2 | 6 (LUSR vectorisé) | `perf/lusr-vectorized` | 1 session |
| Phase 3 | 2 (handle conflict) | `perf/shared-handle-fix` | 1 session |
| Phase 4 | 4 (citations batch partiel + fallback) | `perf/citations-batch` | 1-2 sessions |
| Phase 5 | 1 (post-sync partiellement parallèle) | `perf/post-sync-parallel` | 2 sessions |
| Phase 6 | 5 (transformers executor) | `perf/transform-executor` | 2 sessions |
| Phase 7 | 3 (semaphore fetch/CPU) | `perf/dual-semaphore` | 3 sessions |

> Chaque phase doit avoir ses tests verts avant de passer à la suivante.
> Les Phases 1-3 sont des optimisations **sûres** ou de fiabilisation sans changement d'architecture async.
> La Phase 4 nécessite une discovery cadrée avant implémentation complète.
> Les Phases 5-7 modifient l'architecture async / concurrence — nécessitent revue complète + benchmark.

> **Note** : Phases 5-7 ont un **prérequis bloquant** : rendre `MetadataResolver` thread-safe
> (prérequis transverse §6). Ce travail peut être fait en amont sur n'importe quelle branche.

---

## Hors périmètre explicite

| Sujet | Raison | Réévaluer si… |
|-------|--------|----------------|
| **Sync PvE / Firefight** (`shared_pve.duckdb`, `_pve_db_lock`) | Volume PvE faible, pas de plainte perf | Volume PvE augmente significativement |
| **Match registry pre-fetch** (1 query SQL par match dans `_process_single_match`) | Latence SQL locale négligeable vs latence API | Sync >200 matchs avec registry lent |
| **Retry au niveau match** | Les matchs échoués sont auto-retriés au prochain sync | Taux d'échec >5% récurrent |
| **Aggregates refresh** (`_refresh_aggregates_async`) | Non profilé comme goulot | Profiling futur le montre |

---

## Stratégie de rollback

Chaque axe est implémenté sur une **branche dédiée** (cf. tableau d'ordre). En cas de régression :

1. **Détection** : benchmark post-merge > 65s ou nouveau WARNING dans les logs
2. **Rollback** : `git revert` du merge commit de la phase concernée
3. **Pas de feature flags** : la granularité est le merge commit, pas un toggle runtime

Les axes 7, 6, 2 sont **facilement réversibles** (changements locaux, pas d'architecture).
Les axes 1, 4, 5, 3 touchent l'architecture — un revert peut nécessiter un revert de l'axe suivant
si une dépendance a été introduite. D'où l'importance de **merger et benchmarker chaque phase
indépendamment**.

---

## Checklist avant PR pour chaque phase

- [ ] Tests unitaires verts (`python -m pytest tests/perf/<phase>/ -v`)
- [ ] Suite complète stable (`python -m pytest --ignore=tests/integration`)
- [ ] Benchmark < 65s sur 22 matchs (pas de régression)
- [ ] Aucun nouveau WARNING dans les logs de sync
- [ ] `models_sync.py` : commentaires des valeurs par défaut mis à jour
- [ ] `.ai/thought_log.md` mis à jour
- [ ] Commit au format Conventional Commits : `perf(sync): <description>`
