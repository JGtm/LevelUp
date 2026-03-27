# Plan de migration production — `b5 >> 4` pour player_index

> Créé : 2026-03-19
> Auteur : agent IA (inv134/inv135/inv136)
> Statut : **IMPLÉMENTÉ — 4968 tests passent ✅**
> Branche cible : `refactor/id-resolution-cleanup` (ou nouvelle branche `refactor/b5-production`)

---

## 1. Contexte et objectif

### Situation actuelle (production)

La production utilise `fire_seq % n_players` pour résoudre le player_index dans les fire events :

```python
# weapon_parser.py — scan_fire_events_all()
events = scan_fire_events_bitstring(chunk_data, 1, ts_fn)   # marker pi=1 == 0x26
for ev in events:
    ev["player_index"] = ev["fire_seq"] % n_players         # ← APPROXIMATIF
```

**Problèmes connus** :
- Requiert `n_players` exact → impossible à calculer fiablement si un joueur rejoint/quitte
- Échoue sur a974fdeb (9 joueurs effectifs, modulo erroné)
- Déduplication par `(fire_counter, weapon_bytes)` : fire_counter boucle à 255, supprime des events légitimes sur armes automatiques (MA40 AR, CQS48, etc.)
- `POV_PLAYER_INDEX = 1` est une relique (le POV a pi=0, pas pi=1)

### Cible (après migration)

```python
# weapon_parser.py — scan_fire_events_all()
events = scan_fire_events_b5(chunk_data, ts_fn)  # b5>>4 = player_index direct, dédup byte_pos
```

**Améliorations** :
- `player_index` exact directement dans le bitstream, zéro approximation
- Déduplication robuste par proximité `byte_pos` (< 2 bytes = même event physique)
- Suppression de `n_players` et du code `fire_seq % n_players`

### Validation préalable

| Script | Matchs | Kills | gun_diff | Statut |
|--------|--------|-------|----------|--------|
| inv134 (b5>>4) | 3 | 282 | N/A | Attribution 84–95% conf |
| inv135 (sentinel) | 3 | 282 | **+0** | 24/24 lignes OK ✅ |

---

## 2. Périmètre des changements

### Fichiers modifiés

| Fichier | Type de changement |
|---------|-------------------|
| `src/analysis/_weapon_scanners.py` | Nouvelle fonction `scan_fire_events_b5`, suppression `_build_marker`, refactoring dédup |
| `src/analysis/weapon_parser.py` | `scan_fire_events_all` sans `n_players`, suppression `map_b2_to_player` + `group_events_by_pi` + `POV_PLAYER_INDEX`, `scan_fire_events` adapté |
| `src/data/services/weapon_extraction_service.py` | `_run_scan_phase` sans `n_players`, sans `map_b2_to_player`/`group_events_by_pi`, `ScanResult` épuré |
| `tests/test_weapon_parser_v2.py` | Supprimer `TestMapB2ToPlayer`, `TestGroupEventsByPi`, mettre à jour `TestConstants` |

### Fichiers NON modifiés

| Fichier | Raison |
|---------|--------|
| `src/analysis/_global_correlation.py` | `correlate_kills_global` filtre déjà `ev.get("player_index") == killer_pi` — fonctionne tel quel une fois player_index correct |
| `src/analysis/weapon_parser.py` — `correlate_kills` | Inchangé (reçoit `fire_events_by_pi` → voir note §3.3) |
| `src/analysis/packet_index.py` | Inchangé — `detect_pi_from_metadata` reste la source pour xuid→pi |
| `src/analysis/reconciliation.py` | Inchangé |
| `src/analysis/_weapon_parser_compat.py` | Inchangé (utilise toujours `scan_fire_events_bitstring` pour le pipeline v1 legacy) |

---

## 3. Détail des changements par fichier

### 3.1 `src/analysis/_weapon_scanners.py`

#### A. Ajouter la constante universelle et les offsets b5

```python
# ── Section 2 — Fire Events (b5>>4, inv134) ──

_UNIVERSAL_MARKER = Bits("0b10100100110")   # 11 bits, fixe tous joueurs
_B5_BIT_OFFSET = 32                          # bits depuis event_start → b5
_WEAPON_BIT_OFFSET = 40                      # bits depuis event_start → weapon (existant)
_B5_DEDUP_PROXIMITY = 2                      # bytes — même event physique si écart ≤ 2
```

#### B. Nouvelle fonction `scan_fire_events_b5`

```python
def scan_fire_events_b5(
    chunk_data: bytes,
    estimate_ts: Callable,
) -> list[dict]:
    """Scanne les fire events via le marker universel et extrait player_index = b5 >> 4.

    Déduplication par proximité byte_pos (< _B5_DEDUP_PROXIMITY bytes = même event).
    Ne déduplique PAS par (fire_counter, weapon) : fire_counter boucle à 255.

    Returns:
        Liste de dicts triés par timestamp_ms, chaque dict contenant :
        timestamp_ms, player_index, slot, weapon_name, weapon_bytes,
        fire_seq, fire_counter, b5, byte_pos, burst_end, hit_likely.
    """
    bits = Bits(bytes=chunk_data)
    total_bits = len(bits)
    events: list[dict] = []

    for position in bits.findall(_UNIVERSAL_MARKER, bytealigned=False):
        event_start = position + 3
        weapon_start = event_start + _WEAPON_BIT_OFFSET
        b5_start = event_start + _B5_BIT_OFFSET

        if weapon_start + 64 > total_bits:
            continue

        weapon_int = bits[weapon_start : weapon_start + 64].uint
        weapon_bytes = weapon_int.to_bytes(8, byteorder="big")
        if weapon_int not in WEAPON_IDS_INT and weapon_bytes[4:] != COMMON_WEAPON_SUFFIX:
            continue

        b5_int = bits[b5_start : b5_start + 8].uint
        player_index = b5_int >> 4
        slot = b5_int & 0x03

        byte_pos = position // 8
        fire_seq = (
            bits[event_start + 8 : event_start + 16].uint
            if event_start + 16 <= total_bits else 0
        )
        fire_counter = (
            bits[event_start + 24 : event_start + 32].uint
            if event_start + 32 <= total_bits else 0
        )
        weapon_name = WEAPON_INT_TO_NAME.get(
            weapon_int,
            WEAPON_ID_MAP.get(weapon_bytes, f"INCONNU ({weapon_bytes.hex()})"),
        )
        post_start = weapon_start + 64
        if post_start + 32 <= total_bits:
            post_bytes = bits[post_start : post_start + 32].bytes
        else:
            post_bytes = b"\x00" * 4
        burst_end = bool(post_bytes[1] & 0x01) if len(post_bytes) > 1 else False
        hit_likely = bool((post_bytes[2] & 0x01) == 0) if len(post_bytes) > 2 else None

        events.append({
            "timestamp_ms": estimate_ts(byte_pos),
            "player_index": player_index,
            "slot": slot,
            "b5": b5_int,
            "weapon_name": weapon_name,
            "weapon_bytes": weapon_bytes,
            "fire_seq": fire_seq,
            "fire_counter": fire_counter,
            "byte_pos": byte_pos,
            "post_bytes": post_bytes,
            "burst_end": burst_end,
            "hit_likely": hit_likely,
        })

    # Déduplication par proximité byte_pos
    events.sort(key=lambda x: x["byte_pos"])
    deduped: list[dict] = []
    last_pos = -999
    for ev in events:
        if ev["byte_pos"] - last_pos > _B5_DEDUP_PROXIMITY:
            deduped.append(ev)
            last_pos = ev["byte_pos"]

    return sorted(deduped, key=lambda x: x["timestamp_ms"])
```

#### C. Supprimer `_build_marker`

`_build_marker(player_index)` construit un marker player-specific → **mort** une fois `scan_fire_events_b5` en place.

⚠️ Vérifier d'abord que `_build_marker` n'est plus importé ailleurs :
```bash
grep -r "_build_marker\|scan_fire_events_bitstring" src/ tests/ --include="*.py"
```
- `scan_fire_events_bitstring` reste utilisé par `_weapon_parser_compat.py` → **ne pas supprimer**, mais le garder tel quel (pipeline compat v1)
- `_build_marker` → vérifier usages → supprimer si zéro usage hors de ce module

---

### 3.2 `src/analysis/weapon_parser.py`

#### A. Mettre à jour les imports

```python
from src.analysis._weapon_scanners import (
    COMMON_WEAPON_SUFFIX,  # noqa: F401
    FORMULA_A_PATTERN,     # noqa: F401
    FRAME_MARKER,          # noqa: F401
    estimate_ts_frames,
    find_frame_positions,  # noqa: F401
    scan_fire_events_b5,                  # ← NOUVEAU (remplace scan_fire_events_bitstring ici)
    scan_fire_events_bitstring,           # ← conserver pour _weapon_parser_compat re-export
    scan_formula_a,        # noqa: F401
    scan_formula_a_ns,     # noqa: F401
)
```

#### B. Supprimer `POV_PLAYER_INDEX = 1`

Remplacer par :
```python
# POV player a pi=0 dans le système b5>>4 (et dans PLAYER_METADATA).
# Ancienne constante POV_PLAYER_INDEX = 1 supprimée (était une relique — voir doc résolution).
```

> **Impact tests** : `test_pov_player_index_is_1` à supprimer ou réécrire → `test_pov_player_index_is_0`.

#### C. Réécrire `scan_fire_events_all`

```python
def scan_fire_events_all(
    chunk_data: bytes,
    chunk_start_ms: int,
    chunk_duration_ms: int,
    *,
    packets: list | None = None,
) -> list[dict]:
    """Scanne les fire events de tous les joueurs d'un chunk.

    Le player_index est extrait directement depuis b5 >> 4 (inv134).
    """
    if packets is not None:
        from src.analysis.packet_index import build_packet_estimator
        ts_fn = build_packet_estimator(packets, chunk_start_ms)
    else:
        ts_fn = estimate_ts_frames(chunk_data, chunk_start_ms, chunk_duration_ms)
    return scan_fire_events_b5(chunk_data, ts_fn)
```

Différences clés :
- Plus de paramètre `n_players`
- Plus de `ev["player_index"] = ev["fire_seq"] % n_players`
- Délègue directement à `scan_fire_events_b5`

#### D. Réécrire `scan_fire_events` (single player, conservé pour compat)

Cette fonction est exposée publiquement et utilisée par quelques tests. Elle peut rester mais déléguer à `scan_fire_events_b5` en filtrant sur `player_index` :

```python
def scan_fire_events(
    chunk_data: bytes,
    player_index: int,
    chunk_start_ms: int,
    chunk_duration_ms: int,
    *,
    packets: list | None = None,
) -> list[dict]:
    """Scanne les fire events d'un player_index spécifique (compat/diagnostic)."""
    all_events = scan_fire_events_all(chunk_data, chunk_start_ms, chunk_duration_ms, packets=packets)
    return [ev for ev in all_events if ev["player_index"] == player_index]
```

#### E. Supprimer `map_b2_to_player` et `group_events_by_pi`

Ces deux fonctions n'ont plus de raison d'être en production une fois b5>>4 actif :
- `map_b2_to_player` : cross-référence fire_seq → pi via NS timeline → **remplacée** par b5>>4 direct
- `group_events_by_pi` : dispatch d'events par pi depuis b2_to_pi dict → **inutile**, les events ont déjà `player_index` depuis le scan

> **Impact tests** : `TestMapB2ToPlayer` et `TestGroupEventsByPi` à supprimer (dead code).

#### F. Supprimer l'alias inutile

```python
# À supprimer :
_scan_fire_events_bitstring = scan_fire_events_bitstring
```

---

### 3.3 `src/data/services/weapon_extraction_service.py`

#### A. Mettre à jour les imports

```python
from src.analysis.weapon_parser import (
    KILL_WINDOW_MS,
    build_weapon_timelines,
    detect_player_indices,
    scan_fire_events_all,           # signature sans n_players
    # map_b2_to_player → SUPPRIMÉ
    # group_events_by_pi → SUPPRIMÉ
)
```

#### B. Simplifier `_run_scan_phase`

Supprimer les lignes suivantes dans `_run_scan_phase` :

```python
# AVANT (à supprimer) :
n_players = len(all_participants) or 8
# ...
events = scan_fire_events_all(data, start_ms, dur_ms, packets=packets, n_players=n_players)
# ...
b2_to_pi = map_b2_to_player(all_raw_events, timeline_ns, timing, chunks_sorted)
fire_events_by_pi = group_events_by_pi(all_raw_events, b2_to_pi)
_total_raw = len(all_raw_events)
_dispatched = sum(len(v) for v in fire_events_by_pi.values())
log.record_step(
    "b2_dispatch",
    resolved_b2=len(b2_to_pi),
    total_events=_total_raw,
    dispatched_events=_dispatched,
    dropped_events=_total_raw - _dispatched,
)
```

Remplacer par :

```python
# APRÈS :
events = scan_fire_events_all(data, start_ms, dur_ms, packets=packets)
# ...
log.record_step("scan_fire_total", total_events=len(all_raw_events))
```

#### C. Épurer `ScanResult`

`ScanResult.fire_events_by_pi` n'est jamais consommé par `_correlate_all_players`
(qui utilise `scan.fire_events_global`). Le supprimer :

```python
@dataclass
class ScanResult:
    fire_events_global: list[dict]    # anciennement fire_events_by_pi + global
    timeline: dict[int, dict[int, bytes]]
    timeline_ns: dict[int, dict[int, bytes]]
    swap_pis: dict[int, set[int]]
    timing: list[tuple[int, int]]
    chunks_sorted: list[int]
    # fire_events_by_pi : SUPPRIMÉ (mort — correlate_kills_global utilise fire_events_global)
```

#### D. POV player fallback — inchangé

```python
# Ce code reste valide : le POV a pi=0 dans b5>>4 aussi
if xuid and xuid.isdigit():
    pi_map.setdefault(int(xuid), 0)
```

---

## 4. Plan de tests

### 4.1 Tests à modifier

**`tests/test_weapon_parser_v2.py`** :

| Classe | Action |
|--------|--------|
| `TestConstants::test_pov_player_index_is_1` | Supprimer (ou changer en `assert POV_PLAYER_INDEX == 0` si on garde la constante) |
| `TestMapB2ToPlayer` | **Supprimer** (dead code) |
| `TestGroupEventsByPi` | **Supprimer** (dead code) |
| `TestCorrelateKills` | Inchangés — les tests utilisent déjà `player_index` dans les fixtures `_fire()` |

### 4.2 Nouveaux tests à créer

**`tests/test_scan_fire_events_b5.py`** (nouveau fichier) :

```python
# Tests unitaires de scan_fire_events_b5 :
# 1. Événement synthétique avec b5=(pi<<4)|slot → vérifier player_index et slot
# 2. Déduplication byte_pos : deux hits à < 2 bytes → 1 event
# 3. Filtre weapon : sans suffix 42c9679f → rejeté
# 4. Tri par timestamp_ms
# 5. Zéro event sur chunk vide
```

> Ces tests ne nécessitent pas de vrais chunks binaires — construire des chunks synthétiques
> en assemblant le marker 11 bits + b5 + weapon bytes connus.

### 4.3 Tests d'intégration (post-migration)

Valider sur les 3 matchs de référence avec les chunks en cache :

```bash
# Après migration, lancer inv134 pour valider les résultats identiques :
python scripts/experimental/inv134_b5_pi_attribution.py --match a974fdeb
python scripts/experimental/inv134_b5_pi_attribution.py --match f2f81265
python scripts/experimental/inv134_b5_pi_attribution.py --match d9329229
# Attendre : conf 84–95%, gun_diff proche de 0
```

```bash
# Lancer la suite de tests complète
python -m pytest -q --ignore=tests/integration
```

---

## 5. Ordre d'implémentation recommandé

```
Étape 1 — _weapon_scanners.py
    1a. Ajouter _UNIVERSAL_MARKER, _B5_BIT_OFFSET, _B5_DEDUP_PROXIMITY (constantes)
    1b. Implémenter scan_fire_events_b5()
    [ scan_fire_events_bitstring reste intact pour _weapon_parser_compat ]

Étape 2 — weapon_parser.py
    2a. Mettre à jour les imports (ajouter scan_fire_events_b5)
    2b. Supprimer POV_PLAYER_INDEX = 1
    2c. Réécrire scan_fire_events_all() sans n_players
    2d. Réécrire scan_fire_events() pour filtrer depuis scan_fire_events_all
    2e. Supprimer map_b2_to_player() + group_events_by_pi()
    2f. Supprimer l'alias _scan_fire_events_bitstring

Étape 3 — weapon_extraction_service.py
    3a. Mettre à jour les imports (retirer map_b2_to_player/group_events_by_pi)
    3b. Épurer _run_scan_phase (supprimer n_players + b2_dispatch)
    3c. Épurer ScanResult (supprimer fire_events_by_pi)

Étape 4 — tests
    4a. Supprimer TestMapB2ToPlayer + TestGroupEventsByPi dans test_weapon_parser_v2.py
    4b. Mettre à jour TestConstants
    4c. Créer tests/test_scan_fire_events_b5.py

Étape 5 — validation
    5a. python -m pytest -q --ignore=tests/integration
    5b. Lancer inv134 sur les 3 matchs de référence
    5c. Lancer le backfill weapon_kills sur un match récent (dry_run=True)
```

---

## 6. Risques et mitigations

| Risque | Probabilité | Mitigation |
|--------|------------|------------|
| b5 incorrect sur un match avec >16 joueurs | Faible (max 12 en Big Team Battle) | b5>>4 donne 0–15 → couvre tous les cas |
| POV player reste pi=0 dans b5>>4 | Confirmé (inv134) | Fallback `pi_map.setdefault(xuid, 0)` inchangé |
| `scan_fire_events_bitstring` cassé pour compat | Nul (non modifié) | `_weapon_parser_compat.py` continue d'importer l'ancienne fonction |
| Performance dégradée (nouveau scan) | Faible | `bitstring.findall` est identique, juste un offset extrait en plus |
| Tests cassés sur map_b2_to_player/group_events_by_pi | Certain | Supprimer les tests correspondants (dead code) |
| `correlate_kills_global` rate des events | Nul | Filtre `ev.get("player_index") == killer_pi` inchangé, player_index est maintenant correct |

---

## 7. Ce qui NE change PAS

- `correlate_kills_global` dans `_global_correlation.py` — aucun changement
- `detect_pi_from_metadata` / `extract_metadata_payload` — inchangés
- `_fallback_formula_a` — inchangé (fallback Formula A NS reste actif)
- `reconcile_api_aggregates` — inchangé
- Le champ `player_index` dans `KillAttribution` — inchangé
- Le sentinel grenade/melee (`is_melee`/`is_grenade` flag) — inchangé
- `scan_formula_a` / `scan_formula_a_ns` / `build_weapon_timelines` — inchangés
- La logique `xuid → pi` via `detect_pi_from_metadata` — inchangée
- La logique `pi_map.setdefault(xuid, 0)` pour le joueur POV — inchangée

---

## 8. Rollback

Si un problème bloquant est détecté après migration :

1. Revenir au commit précédent : `git revert <commit>`
2. OU réintroduire temporairement `n_players` dans `scan_fire_events_all` avec un flag `use_b5: bool = True`

Le flag est une garde de compatibilité avec **date d'expiration** : `# compat_guard: supprimer après 2026-04-30`

---

## 9. Commentaires dans `_global_correlation.py` à mettre à jour

Le docstring contient :
> "fire_seq % n_players, posé lors du scan"

Mettre à jour après migration :
> "b5>>4, extrait directement du bitstream lors du scan (inv134)"

---

*Document vivant — mettre à jour le statut à chaque étape complétée.*
