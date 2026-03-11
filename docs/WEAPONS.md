# Weapon Kill Attribution — Technical Reference

This document describes the weapon kill attribution system used by LevelUp to
identify which weapon was responsible for each kill in a Halo Infinite match,
based on filmshell chunk data retrieved via the SPNKr API.

---

## Table of Contents

1. [Overview](#overview)
2. [Data Sources](#data-sources)
3. [Attribution Algorithm](#attribution-algorithm)
   - [Section 2 — POV Fire Events (§6a)](#section-2--pov-fire-events-6a)
   - [Formula A — T1 Teammates (§6b)](#formula-a--t1-teammates-6b)
4. [Confidence Levels](#confidence-levels)
5. [Special Weapon Names](#special-weapon-names)
6. [Unknown WIDs (`?hex` format)](#unknown-wids-hex-format)
7. [Database Schema — `weapon_kills`](#database-schema--weapon_kills)
8. [WEAPON_ID_MAP Reference](#weapon_id_map-reference)
9. [Weapon Timing Parameters](#weapon-timing-parameters)
10. [Resolving an Unknown WID](#resolving-an-unknown-wid)
11. [Adding a New Weapon ID](#adding-a-new-weapon-id)

---

## Overview

LevelUp parses binary filmshell chunks (type `REPLICATION_DATA`) to extract
per-player kill-to-weapon attribution. Results are stored in the
`weapon_kills` table inside `shared_matches.duckdb`.

Two independent code paths are used, depending on the player's role in the
film:

| Path | Scope | Source | Confidence |
|------|-------|--------|------------|
| **Section 2** (fire events) | POV player only | `scan_fire_events()` | high/medium |
| **Formula A** (snapshot) | All other players (T1) | `build_weapon_timeline()` | high/medium/low |

---

## Data Sources

| Source | File |
|--------|------|
| Algorithm & parser | `src/analysis/weapon_parser.py` |
| Weapon ID map & timings | `src/analysis/_weapon_data.py` |
| Orchestration service | `src/data/services/weapon_extraction_service.py` |
| DB writes | `src/data/repositories/_weapon_kills_repo.py` |
| Backfill CLI | `scripts/backfill_data.py --weapons` |

Chunk files are cached locally at:

```
data/investigation/chunks/<match_id[:8]>/chunk_NN.bin
```

---

## Attribution Algorithm

### Section 2 — POV Fire Events (§6a)

**Scope:** The single player who recorded the film (POV). This player is always
at `player_index = 1` in Section 2 of the filmshell — regardless of their
acurtis player index.

**Steps:**

1. Download all `REPLICATION_DATA` chunks that cover any kill window
   (`kill_time_ms − KILL_WINDOW_MS` to `kill_time_ms`).
2. Call `scan_fire_events(chunk, pi=1, ...)` on every chunk. This returns a
   list of `{weapon_name, timestamp_ms, swap_detected, delayed_damage}` events
   for weapons whose IDs are in `WEAPON_IDS_INT ∪ COMMON_WEAPON_SUFFIX`.
3. Call `correlate_kills_to_weapons(kills, fire_events)` — each kill is matched
   to the closest preceding fire event within the timing window for that weapon
   class (`swap_ms`, `travel_max_ms` from `WEAPON_TIMING`).
4. **API reconciliation** (`_reconcile_api_aggregates`): Compare the
   attributed HIGH kills against the `match_participants.kills` value from
   the Halo API.
   - If HIGH kills > API kills → demote the least confident (highest `delta_ms`)
     excess kills to MEDIUM.
   - If HIGH kills < API kills → promote MEDIUM kills (lowest `delta_ms` first)
     to HIGH until the count matches.

> **Note:** Section 2 only returns kills for weapons already in
> `WEAPON_ID_MAP`. Unknown WIDs never appear via this path.

---

### Formula A — T1 Teammates (§6b)

**Scope:** All players except the POV. Their player indices are resolved via
the *acurtis method* (FINDINGS inv #26) — XUIDs are matched to `player_index`
values found in the first chunk.

**Steps:**

1. Call `build_weapon_timeline(chunks)` → builds a dict:
   `timeline[chunk_index][player_index] = wid (bytes)`.
   Each entry records the weapon held by each player at the *snapshot moment*
   of that chunk (~19 s/chunk).
2. For each kill at time `T`:
   - Find the chunk covering `T` with `find_chunk_at_time(...)`.
   - Look up `timeline[chunk][player_index]`.
   - Fall back to `timeline[chunk - 1][player_index]` if no update in the
     current chunk.
3. Map `wid` → weapon name via `WEAPON_ID_MAP`.
   - **Known WID:** `confidence = "high"` (demoted to `"medium"` if a weapon
     swap was detected within the same chunk — `swap_pis`).
   - **Unknown WID:** stored verbatim as `?{wid.hex()}` (16 hex chars),
     `confidence = "low"`.
   - **No snapshot found:** stored as `"UNKNOWN"`, `confidence = "none"`.

> **Important:** Because Formula A uses raw WID bytes from the chunk without
> filtering, it **can** produce unknown WID entries. Section 2 cannot.

---

## Confidence Levels

| Value | Meaning |
|-------|---------|
| `high` | Weapon matched with precise timing or confirmed snapshot (API-reconciled) |
| `medium` | Weapon matched but swap or timing ambiguity detected |
| `low` | Unknown WID — raw bytes stored as `?hex`, weapon name unresolved |
| `none` | Kill could not be attributed (fallback values: `NON TROUVE`, `UNKNOWN`) |

---

## Special Weapon Names

| Value | Origin | Meaning |
|-------|--------|---------|
| `MELEE` | Medal detection | Kill attributed to a melee strike |
| `GRENADE` | Medal detection | Kill attributed to a grenade (type unknown) |
| `NON TROUVE` | Section 2 path | Fire event found but WID not in `WEAPON_ID_MAP` |
| `UNKNOWN` | Formula A path | No weapon snapshot found at kill time (T0 case) |
| `?{16 hex chars}` | Formula A path | Raw 8-byte WID not yet in `WEAPON_ID_MAP` |

> The `T0 case` occurs when a player is not the POV *and* their player index
> could not be resolved via the acurtis method (e.g. the player was not in the
> first chunk).

---

## Unknown WIDs (`?hex` format)

When Formula A encounters a WID not present in `WEAPON_ID_MAP`, it stores the
full 8-byte weapon ID as a hexadecimal string prefixed with `?`:

```
?91eb16de42c9679f   ← 16 hex chars = 8 bytes
```

Properties:
- The `?` prefix is intentional — it distinguishes unresolved WIDs from real
  weapon names and makes them easy to filter or search.
- The full 8 bytes are preserved so the WID can be identified later without
  re-processing the chunks.
- `confidence` is set to `"low"` for all `?hex` entries.

To list the most common unknown WIDs in your database:

```sql
SELECT weapon_name, COUNT(*) AS kills, COUNT(DISTINCT match_id) AS matches
FROM weapon_kills
WHERE weapon_name LIKE '?%'
GROUP BY weapon_name
ORDER BY kills DESC;
```

---

## Database Schema — `weapon_kills`

Table lives in `data/warehouse/shared_matches.duckdb`.

| Column | Type | Description |
|--------|------|-------------|
| `match_id` | VARCHAR | Halo match UUID |
| `xuid` | VARCHAR | Player XUID (killer) |
| `victim_xuid` | VARCHAR | Victim XUID |
| `time_ms` | INTEGER | Kill timestamp (ms from match start) |
| `weapon_name` | VARCHAR | Weapon name, `?hex`, `MELEE`, `GRENADE`, `NON TROUVE`, or `UNKNOWN` |
| `confidence` | VARCHAR | `high`, `medium`, `low`, or `none` |
| `delta_ms` | INTEGER | Time delta between last fire event and kill (Section 2 only) |
| `swap_detected` | BOOLEAN | Weapon swap occurred near the kill |
| `delayed_damage` | BOOLEAN | Projectile travel time may have inflated delta |

Primary key: `(match_id, xuid, victim_xuid, time_ms)`.

---

## WEAPON_ID_MAP Reference

Defined in `src/analysis/_weapon_data.py`. Each entry maps an 8-byte WID
(sourced from the filmshell binary format designed by Andy Curtis / acurtis)
to a human-readable weapon name.

**WID structure:**
- Bytes 1–4: weapon-specific identifier (unique per weapon type/variant).
- Bytes 5–8: shared suffix `42c9679f` for most standard weapons. Weapons with
  a different suffix belong to special families (e.g. Energy Sword variants,
  Mythic Sandwich).

**Organisation of entries:**

| Group | Description |
|-------|-------------|
| Standard weapons | `MA40 AR`, `BR75`, `Mk51 Sidekick`, … — unique suffix per weapon |
| Energy Sword family | Same bytes 1–4 (`4ff3937e`), different suffix per skin |
| Gravity Hammer family | Same bytes 1–4 (`841ac5e5`), different suffix per variant |
| Grenades | Identified via Formula A inventory snapshot (not fire events) |
| Variants / skins | Same gameplay weapon, cosmetic variant — stored separately |

---

## Weapon Timing Parameters

Defined in `WEAPON_TIMING` in `src/analysis/_weapon_data.py`.

Each weapon class has two parameters used by `correlate_kills_to_weapons()`:

| Parameter | Meaning |
|-----------|---------|
| `swap_ms` | Minimum physical delay to switch to this weapon before a kill. Fire events earlier than `kill_time − swap_ms` are excluded. |
| `travel_max_ms` | Maximum projectile/blast travel time. Fire events later than `kill_time − travel_max_ms` are excluded. |

Effective matching window: `[kill_time − swap_ms, kill_time − travel_max_ms]` (reversed: lower bound is `swap_ms` before kill, upper bound is `travel_max_ms` before kill).

| Class | `swap_ms` | `travel_max_ms` | Notes |
|-------|-----------|-----------------|-------|
| Sidearms (Sidekick, etc.) | 400 | 300 | Fast swap, hitscan |
| Assault rifles, BRs | 650 | 500 | Standard |
| Beam weapons, projectiles | 650 | 2000–5000 | Long travel |
| Snipers, Stalker Rifle | 900 | 300 | Slow equip, hitscan |
| Heavy (SPNKr, Hammer, Sword) | 1100 | 1400–2000 | Slow equip, splash |
| Grenades | 950 | 1350–1650 | Cook + travel |

---

## Resolving an Unknown WID

When a `?hex` WID is positively identified as a specific weapon (via asset
dump, community research, or chunk inspection), the resolution process has
**two mandatory steps**:

### Step 1 — Add to WEAPON_ID_MAP

Edit `src/analysis/_weapon_data.py` and add a new entry to `WEAPON_ID_MAP`:

```python
# Example: resolving ?91eb16de42c9679f as "Some Weapon"
bytes.fromhex("91eb16de42c9679f"): "Some Weapon",  # pragma: allowlist secret
```

Place the entry in the appropriate group (standard, grenade, variant, etc.).
If the weapon has timing characteristics different from the default group,
also add an entry to `WEAPON_TIMING`.

### Step 2 — Migrate existing rows

Create a migration step in `src/data/migration/steps/` to update rows already
in the database:

```python
# src/data/migration/steps/add_some_weapon_wid.py
from src.data.migration.registry import Migration, register

def apply_schema(conn):
    conn.execute("""
        UPDATE weapon_kills
        SET weapon_name = 'Some Weapon', confidence = 'high'
        WHERE weapon_name = '?91eb16de42c9679f'
    """)

register(Migration(
    name="add_some_weapon_wid",
    target_db="shared",
    description="Resolve WID 91eb16de → Some Weapon",
    apply_schema=apply_schema,
))
```

Register it in `src/data/migration/steps/__init__.py`:

```python
from src.data.migration.steps import add_some_weapon_wid  # noqa: F401
```

### Step 3 — New matches

No additional action needed. Since `WEAPON_ID_MAP` is now updated, all future
`process_match` runs will automatically use the new weapon name for this WID.

---

## Adding a New Weapon ID

Same as [Resolving an Unknown WID](#resolving-an-unknown-wid) except that the
WID has never appeared in the database yet (new weapon added in a Halo
season update):

1. Add the WID to `WEAPON_ID_MAP` in `_weapon_data.py`.
2. Add timing parameters to `WEAPON_TIMING` if the weapon class is new.
3. No migration step needed since no existing rows reference this WID.

> **Caution:** Do not add WIDs speculatively based on structural similarity
> (e.g. same byte family). Only add WIDs that are positively identified via
> reliable asset sources or direct chunk inspection. Wrong entries will cause
> weapon names to be misattributed across all past and future matches.
