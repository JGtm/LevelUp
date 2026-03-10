# Plan d'optimisation — weapon kills sync & backfill

> Créé : 2026-03-10 | Branche cible : `feat/weapon-kills-perf` (à créer)
> Statut : PLANIFICATION — aucun code modifié

---

## Contexte

Le pipeline weapon kills est **correct** (v5.6, schéma per-kill, Formula A T1, acurtis pi-detection)
mais **lent** pour les backfill bulk :

- Un backfill de 50 matchs prend ~15-20 min (dominant : attente réseau séquentielle)
- Un match 8v8 génère ~16 requêtes SQL (2 par joueur) là où 2 suffiraient
- Le CPU-bound bitstring scanning bloque l'event loop asyncio

### Pipeline actuel (simplifié)

```
run_weapon_kills_backfill(match_ids)
  for match_id in match_ids:                    ← ❌ séquentiel
    process_match(match_id)
      _download_needed_chunks()                 ← ✅ async, semaphore=5
      detect_player_indices(first_chunk)        ← ❌ CPU-bound sur event loop
      build_weapon_timeline(chunks)             ← ❌ CPU-bound sur event loop
      _attribute_all_players()
        for xuid in participants:               ← ❌ séquentiel
          load_player_kills_for_match()         ← ❌ 2 requêtes/joueur
          scan_fire_events() / Formula A        ← ❌ CPU-bound sur event loop
          insert_weapon_kill_rows()             ← ❌ executemany row-by-row
      mark_weapon_backfill_done()
```

---

## Mesures de référence (à établir avant de coder)

Avant toute modification, ajouter des timings pour avoir une baseline :

```python
# scripts/backfill/_weapon_kills_logic.py
import time
t0 = time.perf_counter()
summary = await service.process_match(match_id, gamertag, xuid)
logger.info("process_match %s : %.2fs", match_id[:8], time.perf_counter() - t0)
```

Profiler sur 5 matchs représentatifs (BTB 12v12, ranked 4v4, SWAT 4v4) pour identifier
la répartition download vs CPU vs DB.

---

## Optimisation #1 — Parallélisation des matchs (ROI maximal)

### Problème
`run_weapon_kills_backfill` traite les matchs séquentiellement.
Le téléchargement des chunks représente ~70-80% du temps par match.
Ces I/O sont indépendantes entre matchs → gain direct par concurrence.

### Contrainte DuckDB
DuckDB en mode write **n'est pas safe pour des writes concurrents** depuis plusieurs coroutines
sur la même connexion. Il faut sérialiser les écritures.

### Solution : sémaphore matchs + lock écriture

**Fichier : `scripts/backfill/_weapon_kills_logic.py`**

```python
_MATCH_CONCURRENCY = 4        # 4 matchs en parallèle (tunable)
_DB_WRITE_LOCK = asyncio.Lock()  # sérialiseur d'écritures DuckDB

async def run_weapon_kills_backfill(...) -> int:
    ...
    async with SPNKrAPIClient(tokens=tokens) as api:
        sem = asyncio.Semaphore(_MATCH_CONCURRENCY)
        service = WeaponExtractionService(api, shared_conn, cache,
                                          write_lock=_DB_WRITE_LOCK)

        async def _process(mid: str) -> int:
            async with sem:
                summary = await service.process_match(mid, gamertag, xuid)
                ...
                return summary.get("rows_inserted", 0)

        results = await asyncio.gather(
            *[_process(mid) for mid in match_ids],
            return_exceptions=True,
        )
        total = sum(r for r in results if isinstance(r, int))
    ...
    return total
```

**Fichier : `src/data/services/weapon_extraction_service.py`**

Ajouter un `write_lock: asyncio.Lock | None` dans `__init__`.
Wrapper les blocs d'écriture dans `_attribute_all_players` :

```python
async with self._write_lock or _NULL_LOCK:
    WeaponKillsMixin.insert_weapon_kill_rows(self._conn, match_id, xuid_str, kill_rows)
...
async with self._write_lock or _NULL_LOCK:
    WeaponKillsMixin.mark_weapon_backfill_done(self._conn, match_id)
```

> Note : `_NULL_LOCK` = `asyncio.Lock()` partagé no-op ou contextlib.nullcontext.
> `process_match` devra devenir `async` pour les blocs d'écriture (il l'est déjà).

### Impact attendu
- Backfill 50 matchs : ~15 min → ~4-5 min (×3-4)
- Limiter à 4 matchs concurrents pour ne pas saturer l'API SPNKr (rate limiting)

### Réglages suggérés
| Constante | Valeur initiale | Notes |
|-----------|----------------|-------|
| `_MATCH_CONCURRENCY` | 4 | Augmenter si pas de rate limit API |
| `_CHUNK_TIMEOUT_S` | 30s (existant) | OK |
| `_MAX_CONCURRENT_CHUNKS` | 5 (existant) | OK |

---

## Optimisation #2 — Batch SQL (2 requêtes par match au lieu de 2N)

### Problème
Dans `_attribute_all_players`, `load_player_kills_for_match` est appelé
**une fois par joueur** avec deux requêtes SQL chacun (kills + médailles).
Pour un match 8v8 : 16 requêtes → 2 suffisent.

### Solution : nouvelle méthode `load_all_kills_for_match`

**Fichier : `src/data/repositories/_weapon_kills_repo.py`**

Ajouter une méthode statique :

```python
@staticmethod
def load_all_kills_for_match(
    conn: duckdb.DuckDBPyConnection,
    match_id: str,
) -> dict[str, list[dict]]:
    """Charge kills + médailles de TOUS les joueurs en 2 requêtes.

    Returns:
        {xuid: [{"time_ms", "gamertag", "xuid", "medals_nearby",
                 "is_melee", "is_grenade"}, ...]}
    """
    from src.analysis.weapon_parser import GRENADE_MEDALS, MELEE_MEDALS

    kill_rows = conn.execute(
        "SELECT he.time_ms, he.gamertag, he.xuid "
        "FROM highlight_events he "
        "WHERE he.match_id = ? AND he.event_type = 'kill' "
        "ORDER BY he.xuid, he.time_ms",
        (match_id,),
    ).fetchall()

    medal_rows = conn.execute(
        "SELECT he.xuid, he.time_ms, "
        "json_extract_string(he.raw_json, '$.medal_name') AS medal_name "
        "FROM highlight_events he "
        "WHERE he.match_id = ? AND he.event_type = 'medal' "
        "AND json_extract_string(he.raw_json, '$.is_medal') = 'true' "
        "ORDER BY he.xuid, he.time_ms",
        (match_id,),
    ).fetchall()

    # Grouper médailles par xuid
    medals_by_xuid: dict[str, list[tuple[int, str]]] = {}
    for xuid, t, name in medal_rows:
        if name:
            medals_by_xuid.setdefault(xuid, []).append((t, name))

    # Grouper kills par xuid et enrichir avec médailles
    result: dict[str, list[dict]] = {}
    for time_ms, gt, xuid in kill_rows:
        nearby = [
            name for (mt, name) in medals_by_xuid.get(xuid, [])
            if abs(mt - time_ms) <= 500
        ]
        result.setdefault(xuid, []).append({
            "time_ms": time_ms,
            "gamertag": gt,
            "xuid": xuid,
            "medals_nearby": nearby,
            "is_melee": any(m in MELEE_MEDALS for m in nearby),
            "is_grenade": any(m in GRENADE_MEDALS for m in nearby),
        })
    return result
```

**Fichier : `src/data/services/weapon_extraction_service.py`**

Dans `process_match`, pré-charger avant la boucle :

```python
# Avant _attribute_all_players
all_kills_by_xuid = WeaponKillsMixin.load_all_kills_for_match(self._conn, match_id)
```

Puis passer `all_kills_by_xuid` à `_attribute_all_players` et remplacer l'appel
`load_player_kills_for_match(...)` par `all_kills_by_xuid.get(xuid_str, [])`.

La méthode existante `load_player_kills_for_match` reste pour usage ponctuel UI.

### Impact attendu
- Réduction de 8-10x sur les requêtes DB par match
- Particulièrement visible avec _MATCH_CONCURRENCY=4 (moins de contention DuckDB)

---

## Optimisation #3 — Offload CPU vers threads (asyncio.to_thread)

### Problème
Trois opérations CPU-bound bloquent l'event loop :

1. `detect_player_indices(first_chunk_data, ...)` — `bits.find()` per XUID
2. `build_weapon_timeline(chunks)` — `scan_formula_a()` sur tous les chunks
3. `_scan_fire_events_bitstring(chunk_data, ...)` — scanning exhaustif bitstring POV

Quand 4 matchs tournent en parallèle (opt #1), ces opérations se bloquent mutuellement.

### Solution : `asyncio.to_thread` sur les fonctions pures

**Fichier : `src/data/services/weapon_extraction_service.py`**

```python
# detect_player_indices → to_thread
import asyncio, functools

xuid_int_to_pi = await asyncio.to_thread(
    self._resolve_player_indices, chunks, t1_participants, xuid
)

# build_weapon_timeline → to_thread
timeline, swap_pis, timing = await asyncio.to_thread(
    build_weapon_timeline, chunks
)

# scan_fire_events dans _scan_player_chunks → to_thread
fire_events = await asyncio.to_thread(
    self._scan_player_chunks, chunks, player_index
)
```

> Les fonctions de domaine pur (`weapon_parser.py`) ne sont pas modifiées.
> `asyncio.to_thread` utilise le ThreadPoolExecutor par défaut (GIL libéré par bitstring
> car c'est du C — à vérifier ; sinon gain limité mais event loop reste libre).

### Note GIL
`bitstring` est du Python pur → le GIL n'est PAS libéré → `to_thread` libère l'event loop
mais n'offre pas de vrai parallélisme CPU. Pour un vrai gain CPU multi-cœur, il faudrait
`ProcessPoolExecutor` avec sérialisation des chunks (pickle overhead). À évaluer après
avoir mesuré la proportion CPU vs I/O.

Si la mesure montre >40% CPU : envisager `ProcessPoolExecutor` pour `_scan_player_chunks`
uniquement (les chunks sont des `bytes` — sérialisables efficacement).

### Impact attendu
- Event loop fluide pendant les scans CPU → les downloads des autres matchs ne sont plus bloqués
- Gain réel CPU uniquement si bitstring libère le GIL (à mesurer)

---

## Optimisation #4 — Bulk insert via Polars → DuckDB

### Problème
`insert_weapon_kill_rows` fait un `executemany` row-by-row.
Pour un match dense (50+ kills × 8 joueurs = 400 lignes), c'est lent.

### Solution : insert via `conn.register` + `INSERT INTO SELECT`

**Fichier : `src/data/repositories/_weapon_kills_repo.py`**

```python
@staticmethod
def insert_weapon_kill_rows_bulk(
    conn: duckdb.DuckDBPyConnection,
    match_id: str,
    xuid: str,
    kill_rows: list[dict],
) -> int:
    if not kill_rows:
        return 0
    conn.execute(
        "DELETE FROM weapon_kills WHERE match_id = ? AND xuid = ?",
        (match_id, xuid),
    )
    df = pl.DataFrame({
        "match_id":      [match_id] * len(kill_rows),
        "xuid":          [xuid] * len(kill_rows),
        "time_ms":       [r["time_ms"] for r in kill_rows],
        "weapon_name":   [r["weapon_name"] for r in kill_rows],
        "delta_ms":      [r.get("delta_ms") for r in kill_rows],
        "confidence":    [r.get("confidence", "none") for r in kill_rows],
        "swap_detected": [bool(r.get("swap_detected", False)) for r in kill_rows],
        "delayed_damage":[bool(r.get("delayed_damage", False)) for r in kill_rows],
    })
    conn.register("_wk_tmp", df.to_arrow())
    conn.execute(
        "INSERT INTO weapon_kills SELECT * FROM _wk_tmp"
    )
    conn.unregister("_wk_tmp")
    return len(kill_rows)
```

Remplacer les appels à `insert_weapon_kill_rows` par `insert_weapon_kill_rows_bulk`.
Garder l'ancienne méthode avec `# deprecated` pour compatibilité transitoire.

### Impact attendu
- Modéré sur petits matchs (<20 kills/joueur)
- Significatif sur BTB 12v12 avec 600-800 lignes par match

---

## Ordre d'implémentation recommandé

```
Phase 1 — Mesures baseline
  [ ] Ajouter timings perf_counter dans _weapon_kills_logic.py
  [ ] Profiler 5 matchs représentatifs (BTB, ranked, SWAT)
  [ ] Documenter répartition download / CPU / DB

Phase 2 — Gains immédiats (sans risque)
  [ ] #2 : Implémenter load_all_kills_for_match dans _weapon_kills_repo.py
  [ ] #2 : Brancher dans weapon_extraction_service.py (_attribute_all_players)
  [ ] Tester : python -m pytest tests/test_weapon_parser.py -v
  [ ] Mesurer le gain DB (avant/après)

Phase 3 — Parallélisation matchs (gros gain)
  [ ] #1 : Ajouter write_lock: asyncio.Lock dans WeaponExtractionService.__init__
  [ ] #1 : Rendre _attribute_all_players async (déjà le cas pour process_match)
  [ ] #1 : Wrapper écritures insert+mark avec async with self._write_lock
  [ ] #1 : Modifier run_weapon_kills_backfill → asyncio.gather + sémaphore
  [ ] Tester sur 10 matchs, vérifier intégrité (pas de double insert, bit correctement posé)
  [ ] Mesurer le gain end-to-end

Phase 4 — CPU offload (si mesure Phase 1 montre >40% CPU)
  [ ] #3 : asyncio.to_thread sur detect_player_indices
  [ ] #3 : asyncio.to_thread sur build_weapon_timeline
  [ ] #3 : asyncio.to_thread sur _scan_player_chunks (POV fire events)
  [ ] Si GIL = bloquant → évaluer ProcessPoolExecutor sur _scan_player_chunks uniquement

Phase 5 — Bulk insert (si Phase 1 montre DB write significatif)
  [ ] #4 : insert_weapon_kill_rows_bulk dans _weapon_kills_repo.py
  [ ] Brancher dans _attribute_all_players
  [ ] Tester sur match dense BTB 12v12
```

---

## Fichiers impactés

| Fichier | Modifications |
|---------|--------------|
| `scripts/backfill/_weapon_kills_logic.py` | Phase 3 : asyncio.gather + sémaphore + lock |
| `src/data/services/weapon_extraction_service.py` | Phase 2+3+4 : batch kills, write_lock, to_thread |
| `src/data/repositories/_weapon_kills_repo.py` | Phase 2+5 : load_all_kills_for_match, bulk insert |
| `scripts/backfill/strategies.py` | Aucun changement (delègue à _weapon_kills_logic) |
| `scripts/backfill/orchestrator.py` | Aucun changement (appel inchangé) |
| `src/analysis/weapon_parser.py` | Aucun changement (fonctions pures) |
| `tests/test_weapon_parser.py` | Adapter si signature change |

---

## Risques et mitigations

| Risque | Probabilité | Mitigation |
|--------|-------------|-----------|
| Race condition DuckDB writes | Haute sans lock | `asyncio.Lock` sérialiseur obligatoire |
| Rate limit API SPNKr | Moyenne | `_MATCH_CONCURRENCY=4` conservateur, augmenter progressivement |
| Double-insert si gather + exception | Faible | `DELETE WHERE match_id+xuid` avant insert (déjà en place) + bit idempotent |
| Régression attribution (confiance) | Faible | Tests existants `test_weapon_parser.py` couvrent le domaine pur |
| Chunk cache corrompue (writes concurrents) | Faible | Un chunk par fichier séparé → pas de conflit |
| GIL annule to_thread | Probable (bitstring Python pur) | to_thread libère event loop même sans gain CPU |

---

## Métriques de succès

- [ ] Backfill 50 matchs < 5 min (vs ~15-20 min baseline)
- [ ] Aucune régression sur `python -m pytest tests/test_weapon_parser.py -v`
- [ ] Intégrité DB : pas de double-lignes dans `weapon_kills` (vérifiable avec `SELECT match_id, xuid, time_ms, COUNT(*) FROM weapon_kills GROUP BY 1,2,3 HAVING COUNT(*)>1`)
- [ ] Bit `WEAPON_KILLS` correctement posé pour tous les matchs traités

---

## Références

- `src/data/services/weapon_extraction_service.py` — orchestrateur principal
- `src/data/repositories/_weapon_kills_repo.py` — requêtes DB
- `src/analysis/weapon_parser.py` — domaine pur (ne pas toucher)
- `scripts/backfill/_weapon_kills_logic.py` — entrée backfill bulk
- `scripts/backfill/strategies.py:1501` — stub `backfill_weapon_kills`
- `scripts/backfill/orchestrator.py:957` — appel dans la boucle match
- FINDINGS weapon extraction : `scripts/experimental/FINDINGS_weapon_extraction_EN_full.md`
