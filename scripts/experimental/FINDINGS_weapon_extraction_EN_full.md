# Weapon Extraction — Research Summary (Investigations #1–125)

> Branch: `experimental/film-weapon-extraction`
> Last updated: 2026-03-09
> Main dataset: `d9329229` — Team Slayer 4v4, 8 players, ~545s, 97 kills, 28 chunks
> Reference match: `04f7d9d5` — Slayer Arena, 11/11 kills (100%)

---

## Contents

1. [Key Architecture](#1-key-architecture)
2. [What Is Confirmed Working](#2-what-is-confirmed-working)
3. [What Is Architecturally Impossible](#3-what-is-architecturally-impossible)
4. [Current Frontier — Formula-C (match `00162144`)](#4-current-frontier--formula-c)
5. [Dead Ends](#5-dead-ends)
6. [Production Algorithms](#6-production-algorithms)
7. [Known Weapon IDs](#7-known-weapon-ids)
8. [Match Corpus](#8-match-corpus)
9. [Known Issues](#9-known-issues)
10. [Investigation Log](#10-investigation-log)

---

## 1. Key Architecture

### Film Binary: Chunk Structure

Each match film is split into ~28 chunks. Every chunk has two sections:

```
┌──────────────────────────────────────────────────────────────┐
│  SECTION 1 — State Snapshot  (~80–88% of chunk bytes)        │
│                                                              │
│  Byte-aligned component updates                              │
│                                                              │
│  Formula A: [20 00 02 pb][N bytes][wid:8B]  pi = pb >> 5    │
│    data[-1]=slot_byte (inv slot: 0x80=Spike, 0x01=Dynamo,   │
│    0x00/0x81=weapons, 0x27=power weapon) — inv #106–107     │
│    → teammate weapon snapshots (T1 only, inv #35–39)        │
│                                                              │
│  Formula B: [pi_byte][44/34 0C][flags][wid:8B]              │
│    pi = pi_byte & 0x07  (POV = pi=3 in this scheme)         │
│    → POV weapon state (inv #39–40)                          │
│                                                              │
│  Formula C: [20 00 03 pb ... wid:8B]  (variant branch)      │
│    → present only on match 00162144 (inv #77–100)           │
├──────────────────────────────────────────────────────────────┤
│  SECTION 2 — Frame Stream  (last ~12–20% of chunk)          │
│                                                              │
│  Frame markers: A0 7B 42  (~565–691 per chunk, ~30fps)      │
│  Per-frame fire events — POV only (pi=1)                    │
│  Nibble-shifted (4-bit offset), not byte-aligned            │
└──────────────────────────────────────────────────────────────┘
```

### Fire Event Structure (Section 2, nibble-shifted)

```
Bitstring scan: 11-bit marker search at all bit positions (bytealigned=False)
Marker: 0b101 0010 0110  (LSBs=101, then 0x26 = pi=1 POV)

[0]    (variable:5 | 0b101:3)   lead byte — only 3 LSBs constant
                                correct filter: (byte[0] & 0x07) == 0x05
[1]    (pi << 5) | 0x06        e.g. 0x26 = pi=1 (POV)
[2]    b2_stream                dual-stream discriminator
[3]    0x40–0x43                filter: (byte3 & 0xFC) == 0x40
[4]    fire_counter             tick-indexed, increments by 8, wraps at 256
[5]    b5                       correlated with b2
[6–13] weapon_id                8 bytes, suffix 42c9679f (most weapons)
[14]   post-wid byte            bit 1 = burst indicator, bit 2 = hit/miss (~95%)
```

> Scanning method: `bitstring` 11-bit search at all bit positions (inv #54).
> Yields +6.1% events vs legacy double-pass (5247 vs 4947 on 3 matches, 79 chunks).

### Player Index: Two Conflicting Schemes

| Stream | Scheme | POV value | Formula |
|---|---|---|---|
| Fire events (Section 2) | `pi = pb >> 5` | pi = 1 | upper 3 bits of payload byte |
| Snapshot events (Section 1, Formula A/B) | `pi = pb & 0x07` | pi = 3 | lower 3 bits |

**0% concordance** — these are independent numbering systems.

### Player Index Detection (non-POV, inv #26)

```python
from bitstring import Bits

def get_player_index(bits: Bits, xuid: int) -> int | None:
    term = Bits(uintle=xuid, length=64)
    position = bits.find(term, bytealigned=False)
    if not position:
        return None
    bit_pos = position[0]
    return bits[bit_pos - 5 : bit_pos].uint if bit_pos >= 5 else None
```

> Each XUID appears twice per chunk. 6/8 XUIDs fall on non-byte boundaries
> → byte-aligned scan only finds 2/8. The bitstring method recovers all 8 (inv #26).

---

## 2. What Is Confirmed Working

### 2a. POV Fire Events (Section 2)

**Status: production-ready, 87.5% kill coverage.**

- Player index = 1 always, confirmed across 16+ matches
- Window: `[T - 5000ms, T]` — last fire event before kill timestamp
- Deduplication: `(weapon_id, fire_counter)` per chunk
- BR75 / MA40 record each fire twice with same `fire_counter`, different `b2` → deduplicate

**Fire counts (d9329229, 28 chunks, bitstring scan):**

| Weapon | Fires (dedup) | Note |
|---|---|---|
| BR75 | 1240 | ~413 bursts |
| MA40 AR | 592 | full-auto frame-dropping |
| Fuel Rod SPNKr | 43 | |
| VK78 Commando | 39 | |
| M392 Bandit | 38 | |
| Stalker Rifle | 25 | |

### 2b. POV Melee Events (Section 2, inv #43)

```
[0]    (variable:5 | 0b011:3)   lead byte — 3 LSBs = 011
                                correct filter: (byte[0] & 0x07) == 0x03
[1]    0x40                     fixed (NOT pi-encoded)
[N]    0x0d 0x26                embedded fire event reference (at offset 7, 8, or 12)
[N+2]  counter                  melee counter (increases across match)
[N+5]  animation_type           0x01–0x73, 0xc3
[N+6:N+14] weapon_id            weapon HELD during melee (not a "fist" ID)
```

Melee event counts: d9329229=17, 00162144=11, 63d6f727=19 events.
All POV-only (same architectural limit as fire events).

> Note: weapon_id = weapon held, not melee weapon class. Use `MELEE_MEDALS`
> (Pummel, Assassination, Back Smack) for independent melee kill confirmation.

### 2c. T1 Teammate Snapshot Attribution — Formula A (inv #35–39)

**Status: 96.8% T1 non-POV kills attributed (30/31 on d9329229).**

Formula A transmits weapon state for the POV's team only (T1: pi=4, 5, 6).
State persists: if no swap detected between chunks, last seen weapon is definitively current.

**Coverage on d9329229:**

| Player | pi | Events / chunks | Coverage | Confidence |
|---|---|---|---|---|
| POV Nilton410 | 1 | via fire events | ✅ 87.5% | HIGH |
| Madina97294 | 4 | 114 / 28 | ✅ 100% (9/9 kills) | LOW (unknown skin variants) |
| Chocoboflor | 5 | 60 / 28 | ✅ 85.7% (6/7 kills) | MEDIUM–LOW |
| JGtm | 6 | 105 / 28 | ✅ 100% (15/15 kills, 10 HIGH) | HIGH |
| T0 opponents | 0,2,3,7 | 0–1 | ❌ near-zero | NONE |

**pi→xuid resolution:**

| pi | xuid | Method |
|---|---|---|
| 6 | JGtm | CONFIRMED — grenade+power both match API (score 7/7) |
| 4 | Madina97294 | PROBABILISTIC — shots_fired ratio (sf=636 → 114 events) |
| 5 | Chocoboflor | PROBABILISTIC — shots_fired ratio (sf=328 → 60 events) |

**Confidence model:**

| Zone | Condition | Confidence |
|---|---|---|
| A — No-swap | delta < swap_time(W) | **HIGH** — swap was physically impossible |
| B — Ambiguous | swap_time ≤ delta ≤ travel_max | Check next weapon; if found → HIGH. Else MEDIUM. |
| C — Outside physics | delta > travel_max | **LOW** — set `delayed_damage=True` if area/grenade |

**State persistence rule:**

> Formula A emits an update only when the weapon **changes**. If no swap is detected across
> a chunk and adjacent chunks, the last seen weapon is definitively current (HIGH confidence).
> The ~19s window only produces ambiguity when multiple weapons appear for the same pi
> within one chunk.

---

## 3. What Is Architecturally Impossible

### T0 opponent weapon extraction

**Exhaustively proven absent across inv #23–53.**

```
Root cause (inv #39): the Halo Infinite server applies asymmetric state replication.

  T1 teammates  →  full weapon component updates  →  visible in Formula A / B
  T0 opponents  →  positional / biped data only   →  no weapon component state

This is a server-side design decision (anti-cheat / bandwidth), not a parsing gap.
```

**Evidence summary:**

| Data source | T0 weapon data? | Key evidence |
|---|---|---|
| Fire events (Section 2) | ❌ NONE | 0/7 non-POV in FFA (#23); 0/7 in 3 Team Slayer matches (#27); all 8 bit-shifts, raw+nibble × all 8 event types × 7 T0 pi = 0 (#44 series); 3-match cross-validation: 0/~10K non-POV fire-like patterns (#52) |
| Snapshot Formula A (Section 1) | ❌ NONE | 271/362 hits T1, only 2/362 T0 (noise 0.55%) (#39) |
| Halo API | ❌ NONE | No `weapon_damage_source_id` in kill events (#24); no per-player film API (#34) |
| State blocks `440c`/`340c` | ❌ NONE | All 8 players parsed; non-POV blocks = position floats, 0 weapon IDs (#30) |
| `cb 18 4a` records | ❌ NONE | entity_ref = slot ID, not weapon type; 0/286 cb blocks (#24) |
| Weapon medals | ❌ NEGLIGIBLE | 0/90 kills matched in ±500ms window (#24) |

### Non-fire structures discovered in the process (inv #44 series)

These are **not** weapon events — catalogued to avoid re-investigating:

| Pattern | Layer | Player | Classification |
|---|---|---|---|
| `0d 34/35 XX 30 YY 68 [b0 60 b2 b3 b4 b5] [wid 8B]` | nibble S2 | pi=3 (JGtm) | Per-chunk loadout snapshot, 1/chunk, b3 increments 0x10→0x15 |
| `85 46 08 41 cd 95 [8B variable]` | nibble S2 | pi=2 | Position/state update — b4 constant WITHIN chunk, varies between chunks. NOT a fire counter. |
| `[20 10 00 00 00 00 c0 00] ... [MA40 wid]` | raw S2 | all | MA40 AR inventory/state snapshot, ~35/chunk |
| `c3 57 07 92` | nibble S2 | pi=0,2 | Section 1 BR75 snapshot leaking into nibble view |
| `08 01 01 04/08` | nibble S2 | N/A | Frame sub-headers or padding — not player-specific |

---

## 4. Current Frontier — Formula-C

> Only observed on match `00162144`. Still under investigation.

### Formula-C vs Formula-A: structural comparison

```
Formula A (standard, 5 train matches):   20 00 02 [pb] ... [wid:8B]
Formula C (00162144 only):               20 00 03 [pb] ... [wid:8B]
                                                 ↑
                                         same slot, same delta (-19), different prefix byte

In both: pb == pre16[0]  (inv #80/#81 — zero mismatches across all matches)
In Formula A: pb >> 5 = player_index (pi)
In Formula C: pb bits do NOT map directly to pi — pb encodes multiple dimensions
```

### Active weapon families in `00162144` (Formula-C branch)

| wid | Nickname | Occurrences | Best pi inference | Confidence |
|---|---|---|---|---|
| `edff0e9642c9679f` | `edff` | 8 strict hits, many raw | pi=6 mostly, pi=5 in ck16–18 | MEDIUM (neighborhood rule validated on train, inv #70) |
| `831d801242c9679f` | `831d` | 1 strict hit | pi=5 | MEDIUM (anchor rule validated on train) |
| `f951480042c9679f` | `f951` | 4 strict hits | pi=5 in ck17, pi=6 in ck20, ck19 ambiguous | LOW (family out-of-manifold, inv #76) |
| `b1eb...42c9679f` | `b1eb` | 13 occurrences | pi=6 (phase marker role) | SUPPORT SIGNAL only |

### Formula-C inference pipeline

```
                    ┌──────────────────────────────────┐
                    │  Strict 2283 hits (inv57 parser)  │
                    └──────────────┬───────────────────┘
                                   │
              ┌────────────────────▼────────────────────┐
              │  Direct projection: strict → raw neighbour │
              │  (same/adjacent chunk, delta stable)      │
              │  → inherits raw neighbour's pi context    │
              │  (inv #67 — edff 8/8, 831d 1/1)          │
              └────────────────────┬────────────────────┘
                                   │ no raw neighbour (f951)
                                   │
              ┌────────────────────▼────────────────────┐
              │  Phase fallback (inv #68 / inv #69)       │
              │  Contiguous segment structure:            │
              │  ck07–13: pi6 → ck16–18: pi5 → ck20–21: pi6│
              └────────────────────┬────────────────────┘
                                   │
              ┌────────────────────▼────────────────────┐
              │  b1eb phase support check (inv #93/#94)  │
              │  b1eb = coarse subsystem activity proxy  │
              │  12/13 strict hits supported by b1eb     │
              └────────────────────┬────────────────────┘
                                   │
              ┌────────────────────▼────────────────────┐
              │  Synthetic timeline output (inv #100)    │
              │  Labels: pi5 / pi6 / pi5_or_pi6 /        │
              │          visible_flank / bootstrap_only  │
              └──────────────────────────────────────────┘
```

### Synthetic timeline — match `00162144`

| Chunks | edff state | pi label | b1eb phase | Basis |
|---|---|---|---|---|
| ck03 | 67/58/5b | visible_flank | none | pre-strict flank |
| ck04–13 | 5b / 58 | **pi=6** | active / lock | direct raw projection |
| ck14 | — | pi5_or_pi6 | adjacent | boundary chunk |
| ck15 | 831d:67 | **pi=5** | silent_bridge | raw recross ck14 |
| ck16–18 | 65 | **pi=5** | active / lock | direct raw projection |
| ck19 | f951:5e | ambiguous | adjacent | out-of-manifold family, no raw |
| ck20–21 | 65 | **pi=6** | late_lock | direct raw projection |
| ck23 | 65 | visible_flank | none | post-strict flank |

### Formula-C: what is still unresolved

1. **f951 at ck19** — the only unresolved internal ambiguity. Family is out-of-manifold
   (inv #76), ck19 is frame-normal (inv #99), sits at a pi5→pi6 boundary transition.
2. **ck03 / ck23** — visible Formula-C states with no strict hit and no b1eb support.
   These are external flanks of the strict timeline, not internal holes.
3. **27 unknown weapon_ids** — behavioral evidence points to skin/coating variants of
   BR75, MA40, Stalker, Hydra, Scattershot. Definitive resolution needs asset lookup.
4. **Formula-C rarity** — confirmed absent on 6/7 corpus matches (inv #90). Specific
   to `00162144`; not yet understood why this match uses the `20 00 03` branch.

---

## 5. Dead Ends

These were investigated, produced no usable signal, and are closed.

| Approach | Investigated | Why it failed |
|---|---|---|
| **Detonation events with wid** (#4) | Searched for distinct event type around kill times | No such event type exists in the binary |
| **Spawn inventory / slot=2 grenade** (#5, #7, #8) | Group C/D IDs, Fiesta slot scan (10 matches) | These are state blocks, never fire events; slot=2 = noise without `42c9679f` suffix |
| **Non-POV fire events at all 8 bit-shifts** (#23, #27, #44–53) | Exhaustive: raw+nibble, S1+S2, 3 matches, 78 chunks, 64MB, all 8 event types × 7 T0 pi | 0/~10K non-POV fire-like patterns; root cause is server-side asymmetric replication |
| **Kill events / text strings in Section 2** (#53) | Full binary scan for XUIDs, ASCII, 8-fold periodicity | Section 2 is 100% binary, no text, no XUIDs, no periodicity structures |
| **Microsoft Bond format** (#33, #58) | 6 hypotheses including compact U8+U8+U8+STOP, tag scan, density near fire events | Bond = HTTP API only. 0/3189 wid hits preceded by Bond tag. Custom packed format confirmed. |
| **Per-player film API** (#34) | 3 endpoint patterns × 8 players | All 404 — one film per match, no per-player API |
| **`cb 18 4a` / `07 10` records** (#21–22, #24) | entity_ref structure scan, 286 cb blocks | entity_ref = slot ID, not weapon type; 0/286 blocks contain weapon_id |
| **State blocks `440c`/`340c` for non-POV** (#30) | All 8 players, full block parse | Non-POV blocks = position floats; weapon_id appears only in POV's Formula B |
| **Melee events — fire event structure applied** (#42) | Used fire event byte offsets on melee lead byte | Wrong structure: melee has b1=0x40, wid at variable offset 13–18 (not 6). See #43 for correct decoder. |
| **T0 loadout markers** (#44d) | `0d 3X` pattern extended to all players | EXCLUSIVE to pi=3 only (7 entries, 0 for pi=0/2/7); post-loadout = high entropy, no fire events |
| **Frame-break explanation for ck19** (#99) | Compared frame markers, S2 size, avg frame size ck17–20 | ck19 is frame-normal — not a structural anomaly |
| **Exact pre16/post16 family transfer for f951** (#72) | Learned 11 exact families on train, applied to 00162144 | Target family `5e8...` / `5eca...` is out-of-manifold — structurally non-transferable |
| **pb low bits as pi for Formula-C** (#79) | Recrossed pb low 3 bits vs pi5/pi6 neighborhood per occurrence | Same low-bit buckets appear under both contexts — pb is not a hidden player index |
| **Halo Waypoint/API for skin→base weapon mapping** (#116–117) | Querying API for cosmetic wid→base weapon ID | API does not expose this mapping. Skin wids are internal asset IDs not retrievable via public endpoints. Dead end. |
| **B0..B7 pre-wid bytes as weapon-type discriminant** (#116) | Formula A pre-wid structure, first 8 bytes before fixed sequence | Low purity (<10%) for all prefixes; bytes are dynamic stream data (timer/counter), not weapon-type-specific. Dead end. |
| **Kill-event chunk overlap to identify base weapon** (#118) | Jaccard similarity between Formula A pi chunks and kill-event chunks | Max Jaccard=0.200 in 4v4 match. Multiple players share base weapons; signal too noisy to attribute. Dead end. |
| **Base weapon wid adjacent to skin wid in initial state** (#119) | Scanned ±256B around skin wid in chunk 0/1 for known weapon wids | 0 known wids found within 256B of any skin wid in initial state block. Dead end. |
| **Dual-wid in same Formula A event (post-wid bytes)** (#120) | Searched 32B after display wid for any known wid or suffix (42c9679f / e7232c0b) | 0 known wids and 0 known suffixes in post-wid bytes across all 1390 Formula A events. Post-wid bytes are dynamic (purity 6-16%). Dead end. |
| **Full-corpus adjacency scan for known wids** (#121) | Searched ±512B around ALL occurrences of exclusive wids (not just initial-state) | 0/97 occurrences of 6d32c7dc and 0 for all other exclusive wids have any known wid in window. Segregation is total and global. Dead end. |
| **Unknown suffix proximity (base wid not in known list)** (#122) | Searched ±512B for ANY 42c9679f suffix near exclusive wids, regardless of whether prefix is known | Small number of hits, all at large distances (≥191B). Fixed-offset pair: 6d32c7dc → 0af3952e at +191B (inv123 confirms this is another Formula-A-exclusive wid, not a base wid). Dead end for base-weapon identification. |

---

## 6. Production Algorithms

### 6a. POV Kill Attribution (weapon-swap-aware, v2)

```
Step 1 — Collect fire events
  Scan Section 2 (nibble-shifted) for:
    (byte[0] & 0x07) == 0x05
    (byte[3] & 0xFC) == 0x40
    weapon_id in [T - 5000ms, T]
  Deduplicate by (weapon_id, fire_counter) per chunk.

Step 2 — Confidence zone
  delta = T_kill - T_last_fire
  Zone A  delta < swap_ms(W)          → HIGH (swap was physically impossible)
  Zone B  swap_ms ≤ delta ≤ travel_max → check for W2 fire after T+swap_ms;
                                         if W2 found → use W2, HIGH; else MEDIUM
  Zone C  delta > travel_max           → LOW, delayed_damage=True if area/grenade

Step 3 — Store result
  weapon_kills(match_id, xuid, time_ms, weapon_name, delta_ms,
               confidence, swap_detected, delayed_damage)

Step 4 — API Aggregate Reconciliation
  Compare attributed weapon-class counts vs API aggregates:
    grenade_kills, melee_kills, power_weapon_kills, headshot_kills
  4a: if HIGH count > API count → demote least-certain (largest delta) to MEDIUM
  4b: if HIGH count == API count → promote remaining MEDIUM/LOW non-grenade kills
  4c: if HIGH count < API count → promote delayed/no-fire candidates up to deficit
  4d: cross-class exclusion — if class fully accounted → remove from other candidates
```

### 6b. T1 Non-POV Kill Attribution (snapshot-based)

```
Input: chunks, kill events (time_ms, killer_xuid), pi→xuid map (partial)

Step 1 — Build weapon timeline
  For each chunk i, scan Section 1:
    Formula A: [20 00 02 pb ... wid]  → weapon_A[i][pb >> 5] = last wid seen
    Formula B: [pi_byte][44/34 0C][wid] → weapon_B[i][pi_byte & 0x07] = last wid

Step 2 — Attribute kill at time T by xuid X
  pi = pi_from_xuid_map[X]
  ck = find_chunk(T)   # from frame-marker timing (~30fps)
  wid = weapon_A[ck].get(pi) or weapon_A[ck-1].get(pi)
  if wid is None: return UNKNOWN

Step 3 — Confidence
  HIGH   : wid in WEAPON_ID_MAP AND no swap in current + adjacent chunks
  MEDIUM : multiple weapons for pi in kill's chunk (swap within ~19s window)
  LOW    : wid not in WEAPON_ID_MAP (unknown skin variant)
  NONE   : no weapon state for pi (T0 players)

Step 4 — API Aggregate Reconciliation (same as POV Step 4)
```

### Weapon class timing table

| Class | swap_ms | travel_max_ms | Weapons |
|---|---|---|---|
| instant / sidearm | 400–450 | ≤300 | Sidekick (400), Plasma Pistol (450) |
| instant / standard | 650 | ≤500 | MA40, BR75, Bandit, Bulldog, Commando, Shock Rifle, Mangler, Pulse Carbine |
| slow_projectile standard | 650 | ≤2000 | Heatwave, Needler, Hydra |
| area_delayed standard | 650 | ≤5000 | Ravager overcharge, Disruptor chain |
| instant / heavy draw | 900 | ≤300 | S7 Sniper, Stalker Rifle |
| slow_projectile / heavy | 900 | ≤3000 | Skewer |
| area_delayed / heavy | 900 | ≤5000 | Cindershot (bounce) |
| slow_projectile / destroy | 1100 | ≤2000 | M41 SPNKr, Fuel Rod SPNKr |
| melee / destroy | 1100 | ≤1400 | Gravity Hammer, Energy Sword, Mutilator |
| grenade | 950 | ≤1650 | M9 Frag (1650), Plasma (1350), Spike (1550), Dynamo (1500) |

### `weapon_kills` table schema

```sql
CREATE TABLE weapon_kills (
    match_id       TEXT,
    xuid           BIGINT,
    time_ms        INTEGER,
    weapon_name    TEXT,
    delta_ms       INTEGER,
    confidence     TEXT,           -- 'high' / 'medium' / 'low' / 'none'
    swap_detected  BOOLEAN,
    delayed_damage BOOLEAN
);
```

---

## 7. Known Weapon IDs

### Group A — Infantry weapons (confirmed)

| Hex (16 chars) | Weapon |
|---|---|
| `6acdc44d42c9679f` | Bandit Evo |
| `2b1824d542c9679f` | BR75 |
| `230447b142c9679f` | Cindershot |
| `b619d84a42c9679f` | CQS48 Bulldog |
| `84bd29ed42c9679f` | Disruptor |
| `9d6aaed242c9679f` | Fuel Rod SPNKr |
| `2ac9c2ff42c9679f` | Heatwave |
| `71ab0a2c42c9679f` | M41 SPNKr |
| `2fb21c8742c9679f` | M392 Bandit |
| `48c19d2d42c9679f` | MA40 AR |
| `f5c335dfe7232c0b` | MA5K Avenger *(different suffix)* |
| `80977ba542c9679f` | Mangler |
| `767db96d42c9679f` | MLRS-2 Hydra |
| `f408190f42c9679f` | Mk51 Sidekick |
| `d791556542c9679f` | Mutilator |
| `b533957e42c9679f` | Needler |
| `c354294642c9679f` | Plasma Pistol |
| `30484ea642c9679f` | Pulse Carbine |
| `c30d87c742c9679f` | Ravager |
| `0a1992bc42c9679f` | S7 Sniper |
| `9387a8b942c9679f` | Shock Rifle |
| `0d20c46942c9679f` | Skewer |
| `daf193c742c9679f` | Stalker Rifle |
| `fd98554c42c9679f` | VK78 Commando |
| `3e07021742c9679f` | Vestige Carbine |
| `c24e549e42c9679f` | Sentinel Beam |
| `8afc085542c9679f` | Gravity Hammer |
| `1488d0bb42c9679f` | Energy Sword |
| `b6dbead842c9679f` | M9 Frag Grenade |
| `c1e1bab042c9679f` | Plasma Grenade |
| `6683257c42c9679f` | Spike Grenade variant *(confirmed inv #37 — ×42 events for pi=6 JGtm)* |
| `3ad55da442c9679f` | Dynamo Grenade *(Formula A, loadout/hand — ×20 for pi=6, variant candidate)* |
| `18e1fea042c9679f` | Dynamo Grenade *(raw world object / projectile — confirmed inv #89: present Super Fiesta ×5 matches, absent d9329229/00162144)* |

### Group B — Unconfirmed variants (PvE / post-update)

`7e53b3c6`, `04e7f00b`, `3d344885`, `f5ef3bdb`, `fcc6aa76`, `7deb133f`, `cb30ec5e`,
`2b1d61e4`, `7a11aeef`, `c2a6d5e0`, `1f6ae655` (all suffix `42c9679f`)

### Still unknown (26 IDs)

Scattershot, Beam Rifle, Rushdown Hammer, Duelist Energy Sword not yet found.
Top by frequency: `87fab1d4` (×31), `60f1d512` (×23), `0131ea10` (×22), `1d60d24c` (×11).

**`6d32c7dc` = LOADOUT WEAPON (Formula-A-exclusive wid)** — inv107: confirmed primary WEAPON
(slot=0x00, not a grenade). inv109/115: identified as **PRIMARY_BASELINE** in the clean
control match with holdtime=41%, fragmentation=0.55. NOT a Scattershot pickup.
inv117: Formula-A-exclusive (0 kill-event occurrences). The corresponding base weapon wid is
unknown and **not recoverable from the film binary** (inv119-123 exhausted all binary
approaches). Relationship to a specific base weapon (weapon variant, cosmetic skin, or other)
is **unconfirmed**. Former "Scattershot candidate" hypothesis **refuted**.

**Structural family classification (inv105–107, 8-match corpus):**

The byte at offset -1 from wid in Formula A events (the `slot_byte`) encodes the player's
inventory slot. Combined with the pre-wid data block length (`pre_wid_len`), this classifies
all unknown wids into families:

| slot_byte | pre_wid_len | Known anchor | Family | Top unknowns |
|---|---|---|---|---|
| `0x80` | 16 | **Spike Grenade** | SPIKE_GRENADE_FAMILY | `67fed82c` (×39), `91eb16de` (×38), `60f1d512` (×23), `87fab1d4` (×31 mixed) |
| `0x01` | 20 | **Dynamo (hand)** | DYNAMO_FAMILY | `6a672afc` (×8), `b5e3278e` (×1) |
| `0x01` | 16 | — | GRENADE_VARIANT (secondary slot) | `d48d9b84` (×107), `92f99df4` (×61), `0131ea10` (×44) |
| `0x01` | 15 | — | GRENADE_VARIANT (alt structure) | `f9514800` (×47), `edff0e96` (×33) |
| `0x00` | 16 | — | WEAPON_VARIANT (primary slot) | `6d32c7dc` (×93), `f55c4bd2` (×44), `82a3f54a` (×30) |
| `0x81` | 16 | — | WEAPON_VARIANT (secondary slot) | `d0b802c4` (×40), `5ded6cf2` (×23) |
| `0x27` | 18 | **Skewer** | POWER_WEAPON | — (1 known event only) |

**Key confirmed findings (HIGH confidence):**
- `67fed82c`, `91eb16de`, `60f1d512` = **Spike Grenade skins** (same slot+len as confirmed Spike Grenade)
- `6d32c7dc` = **primary weapon** (slot=0x00, not grenade) — Scattershot candidate confirmed as weapon
- `d48d9b84`, `92f99df4` = **secondary grenade type** (slot=0x01, len=16, ≠ Dynamo structure)
- `f9514800`, `edff0e96` = **secondary grenade** (slot=0x01, len=15 — distinct structure)
- `d0b802c4` = **secondary weapon** (slot=0x81)

**Structural note (inv103)**: The XUID bitstream pi (inv26 method) is DIFFERENT from Formula A pi
(pb >> 5). In d9329229: bitstream Madina=pi=1 but Formula A Madina=pi=4. The offset is not
constant across matches. Cross-match pi→XUID resolution via bitstring method is unreliable
without per-match Formula A calibration.

---

## 8. Match Corpus

| Match ID | Mode | Notes |
|---|---|---|
| `04f7d9d5` | Slayer Arena | Reference — **11/11 kills (100%)** |
| `d9329229` | Team Slayer 4v4 | Main dataset — 28 chunks, 97 kills |
| `0beaa1ca` | FFA Arena | 8 players, ~391s, 90 kills — POV-only confirmed |
| `00162144` | Team Slayer | Formula-C target match — 23 chunks |
| `63d6f727` | Team Slayer | 27 chunks — train set for Formula-A inference |
| `000d5950` | — | Positive validation: `edff`/`831d` neighborhood rule confirmed |
| `ebfb64f2` | — | Negative `f951` case: Formula A pi=5, neighborhood predicts pi=6 |
| `1bd7303b` | — | Low-signal recent control |
| `f3bc46ab` | Super Fiesta | 22 chunks — Dynamo Grenade double-frag inv #89 |
| `73284037` | — | Formula-C negative control (0 hits, inv #90) |
| `e44bfaaa` | Super Fiesta | 3/10 (30%) — empty chunks + rare weapons |

---

## 9. Known Issues

1. **Empty chunks** (`e44bfaaa` chunks 21/24/25): 0 events — encoding variant or unknown IDs.
2. **Silent grenade misattribution**: fire event after grenade throw = false attribution.
   Mitigation: `delayed_damage` flag + Step 4 reconciliation.
3. **Ravager overcharge**: zone persists 4–5s — treat as grenade class if delta > 3000ms.
4. **Nibble-shift artefacts**: same fire event at nibble=0 and nibble=1 → deduplicate
   across shifts by `(weapon_id, fire_counter)`.
5. **API aggregate edge cases**: grenade+melee combo kills may be double-counted in both
   `grenade_kills` and `melee_kills`. Needs validation across 10+ matches.
6. **Post-match shots**: shots fired after match end ARE in film data but NOT in API
   ShotsHit/ShotsFired → expected discrepancy when verifying against API stats.
7. **Burst/hit-miss bit alignment**: acurtis's bit 1 (burst) and bit 2 (hit/miss) positions
   on byte[14] may not align to our nibble-shifted layer. Only 12% match rate on BR 0-0-1
   pattern. Needs further mapping.

---

## 10. Investigation Log

Condensed log — key findings only. Full investigation scripts in `scripts/experimental/`.

### Phase 1: POV fire events and player index (#1–22)

| # | Finding |
|---|---|
| 1 | Fire events (`0x0D`, nibble-shifted) → 11/11 kills on Arena reference match |
| 2–3 | Weapon ID map built (41 confirmed IDs, 217 unknown across 10 matches) |
| 6 | player_index=1 = POV universally confirmed (slot=1 rule) |
| 16 | chunk_00 registry → 7588 blocks, `weapon-state-type-info` identified |
| 26 | acurtis bitstring method → all 8 XUIDs recovered (6/8 non-byte-aligned) |

### Phase 2: POV-only confirmed, non-POV exhaustive search (#23–53)

| # | Finding |
|---|---|
| 23 | 0 fire events for all 7 non-POV players (FFA, 8 players) |
| 24 | 87.5% POV↔kills match; API has no per-kill weapon source |
| 27 | 0 fire events for ALL non-POV: 3 matches, 78 chunks, 64 MB |
| 31 | 1794 dedup fire events, ALL pi=1 — weapon-ID-first scan confirmed |
| 33 | Bond = HTTP API only — film binary is bit-packed custom format |
| 39 | T0 architecturally absent: server does not transmit opponent weapon states |
| 41 | Lead byte: only 3 LSBs constant → `(byte[0] & 0x07) == 0x05` |
| 43 | Melee events: correct structure (b1=0x40, wid at var offset, weapon HELD) |
| 44–50 | Exhaustive T0 hunt: 8 bit-shifts × raw+nibble × S1+S2 × Bond × frame-level → 0 T0 events |
| 50 | Ultimate proof: 0/194 T0 fire-like patterns have valid weapon_id vs 153/153 POV |
| 53 | Section 2 = 100% binary: zero XUIDs, kill events, text strings, 8-fold periodicity |
| 54 | Bitstring 11-bit scan (bytealigned=False) → +6.1% fire event yield |

### Phase 3: T1 snapshot attribution (#35–39)

| # | Finding |
|---|---|
| 35 | Formula A (`20 00 02 [pb]...wid`) found for T1 non-POV: pi=4 (114 events), pi=5 (60), pi=6 (102) |
| 36 | 100% chunk coverage for 81 non-POV kills; Formula A/B concordance = 0% (independent streams) |
| 37 | pi=6=JGtm CONFIRMED (API score 7/7). 96.8% T1 kills attributed (30/31) |
| 38 | 27 unknown wids: behavioral evidence → skin variants. `edff0e96` = JGtm post-grenade primary. |
| 39 | Suffix-anchored scan: 271/362 T1, 2/362 T0 (noise). T0 conclusion final. |

### Phase 4: FILM_HEADER, component registry, 2285/2283 parsers (#55–74)

| # | Finding |
|---|---|
| 55 | FILM_HEADER confirmed stable across 3 matches. Weapon-related component indices: `weapon-state-type-info` (2283–2286), `biped-desired-weapon-set` (2282) |
| 56 | Component-context scan: `2285` (nibble-shifted) is strongest next parser target — 5 nearby BR75 IDs on d9329229 |
| 57 | Strict parser for 2285/2283 families: 5 HIGH `2285`→BR75 hits (d9329229), 8 `2283` hits on 00162144 |
| 58 | Bond decoding is wrong mental model for film chunks — custom packed format, nibble-shifted alignment |
| 59 | `filmshell` confirms same acurtis nibble-shift model. Formula-A bridge: stable at delta=-19 bytes |
| 60–69 | Family transfer pipeline: edff/831d → neighborhood rule validated; f951 → out-of-manifold |
| 70 | Anchor-neighborhood rule: 0 mismatches for edff/831d on train; f951 breaks rule (wrong pi predicted) |
| 74 | Method reliability matrix: edff/831d = reusable neighborhood rule; f951 = separate problem |

### Phase 5: Formula-C model, b1eb, synthetic timeline (#75–100)

| # | Finding |
|---|---|
| 75 | 3 new recent matches added. `000d5950` = positive edff/831d validation; `ebfb64f2` = negative f951 |
| 76 | f951 has strong intra-manifold family model, but target families are out-of-manifold |
| 77 | `00162144` uses `20 00 03` at same structural slot where train uses `20 00 02` (Formula-C) |
| 80 | pb == pre16[0] in both Formula-A and Formula-C — zero mismatches across all matches |
| 82 | Formula-C state-trajectory map: edff cycles 58/5b/59/65; b1eb cycles 5a/6c/6f |
| 89 | Dynamo Grenade projectile wid confirmed: `18e1fea042c9679f` (present Super Fiesta ×5, absent Arena) |
| 90 | Formula-C confirmed rare/context-specific — 0 hits on 2 recent matches |
| 91–92 | b1eb = phase marker for Formula-C subsystem. Field model: `field89` progression 0x0894→0x189a |
| 93–94 | b1eb as activity proxy: 12/13 strict hits supported. Activity windows: active/lock/bridge/flank |
| 95 | Synthetic Formula-C timeline fuses states, windows, and weak pi labels per chunk |
| 96–97 | Residual: ck03/ck23 = external flanks; ck19 (f951) = last unresolved internal ambiguity |
| 98–99 | f951 ck19: out-of-manifold boundary ambiguity, not frame anomaly (ck19 is frame-normal) |
| 100 | Formula-C → coarse non-POV reconstruction layer. Conservative labels with explicit basis. |

---

### Phase 6: Unknown weapon ID resolution (#101–103)

| # | Finding |
|---|---|
| 101 | Cross-corpus co-occurrence scan (99 unknown prefixes, 8 matches, 170 chunks). Co-occurrence with Spike Grenade detects player identity, NOT weapon family — unknowns co-occurring with Spike Grenade belong to the same player who also carries a grenade, not to the grenade class. |
| 102 | Temporal transition analysis (prev_known → unknown → next_known). Result: JGtm's unknowns (6d32c7dc, edff0e96 etc.) appear as "Spike Grenade sandwich" (post-grenade primary weapon return). Madina/Choco unknowns have START→END transition (exclusive loadout skins with no transition to known weapons). Both patterns confirm player identity but not weapon class. |
| 103 | Pickup vs loadout classification using XUID bitstream pi (inv26) × Formula A pi. Key structural finding: the XUID bitstream pi system and Formula A pi system are DIFFERENT numbering spaces with no fixed cross-match offset. Results therefore partially unreliable. Best confirmed finding: `6d32c7dc` = Scattershot candidate (20 occurrences at JGtm's Formula A pi=6 in d9329229, matching inv39 "20/28 chunks"). Cross-corpus: appears at different pi/XUIDs = pickup weapon behavior. |

### Phase 7: Unknown weapon ID structural analysis (#105–109)

| # | Finding |
|---|---|
| 105 | Holdtime + chunk-continuity analysis. All significant unknowns appear in scattered patterns (fragmentation 0.40–0.55) across every match. No power-weapon single-block pattern found in entire corpus (filter: ≤2 blocks, ≥5 chunks → 0 results). Hypothesis: Formula A is event-driven, not continuous; holdtime does not distinguish power weapon pickups from loadout skins in this corpus. |
| 106 | Formula A byte structure analysis. Discovery: the `N bytes` data block before each wid has a fixed structure. The LAST BYTE (position -1 from wid start) consistently encodes the player's **inventory slot** (`slot_byte`). Confirmed anchors: Spike Grenade=`0x80`, Dynamo(hand)=`0x01`, Skewer=`0x27`. The 7 bytes at -8..-2 are `18 00 00 00 03 44 0c` for all len=16 events (irrespective of weapon type). |
| 107 | Slot-byte + pre_wid_len structural family classification (102 prefixes, 8-match corpus). **Key results**: (1) 14 unknowns in SPIKE_GRENADE_FAMILY (slot=0x80, len=16) — confirmed grenade skins. (2) `6d32c7dc` = WEAPON_VARIANT slot=0x00 — confirmed primary weapon, NOT a grenade. (3) `d48d9b84`, `92f99df4` = GRENADE_VARIANT slot=0x01, len=16 — secondary grenade type (distinct from Dynamo structure). (4) `f9514800`, `edff0e96` = GRENADE_VARIANT slot=0x01, len=15 — alternate grenade structure. (5) `d0b802c4` = WEAPON_VARIANT slot=0x81 — secondary weapon. Full classification table in outputs/inv107_slot_family_classification.json. |
| 108 | Slot exclusivity analysis (inv108). Confirms: **0 mixed groups** across entire corpus — no (match, pi, slot) group ever has both a known and an unknown weapon in the same slot. Conclusion: Formula A events report SKIN wids exclusively when a skin is equipped (the base wid is fully replaced). 18 unknown prefixes are "LOADOUT candidates" (appear exclusive, n>=2 groups): `6d32c7dc`, `b1eb695e`, `f55c4bd2`, `0131ea10`, `b0171062` are the most frequent. |
| 109 | Multi-unknown slot timeline analysis (inv109). For (match, pi, slot) groups with >=2 unknown prefixes, temporal distribution distinguishes loadout vs pickup. **Key result**: `6d32c7dc` is LOADOUT in 5 groups (holdtime 44-54%, fragmentation 0.40-0.55) — **Scattershot hypothesis refuted**. Real pickups confirmed: `b1eb695e`, `831d8012`, `559385fe`, `7c0e042e` appear as single-chunk appearances alongside a dominant loadout. Anomaly: match `63d6f727` pi=5 has 13 different slot=0x00 unknowns — likely multiple players sharing pi=5 (3-bit pi field saturates in large matches or Formula A fires for all 8 players, not just T1 teammates). |

### Phase 8: Formula-A pi collision + skin/base dual-wid architecture (#110–119)

| # | Finding |
|---|---|
| 110 | Formula A pi collision audit. Criterion: ≥2 distinct unknown prefixes in same (chunk, pi, slot). Result: **13 collision groups in 7/8 matches** — pi = pb >> 5 is an alias bucket (multiple real players share same pi value), not a unique player ID. This invalidates per-pi player attribution in most matches. |
| 111 | Collision-aware filter for inv109 timelines. Re-classifies multi-unknown groups as SAFE vs ALIAS_UNSAFE based on inv110 results. |
| 112 | Collision-aware filter for inv108 slot exclusivity. Separates SAFE_EXCLUSIVE from ALIAS_UNSAFE_EXCLUSIVE groups. |
| 113 | Collision distribution by match. **Key finding: `73284037` is the ONLY collision-free match** in the corpus. All other 7 matches have ≥1 collision-positive group. 73284037 is the unique clean control match for reliable analysis. |
| 114 | Control match `73284037` profile. Safe exclusive groups: (1) pi=4, slot=0x00, `6d32c7dc` — 11 chunks, PRIMARY_BASELINE; (2) pi=5, slot=0x81, `510f248a` — 1 chunk, SECONDARY_PUNCTUAL; (3) pi=6, slot=0x81, `6c587a12` — 1 chunk, SECONDARY_PUNCTUAL. |
| 115 | Temporal role audit in 73284037. Confirms `6d32c7dc` = PRIMARY_BASELINE (holdtime=40.7%, frag=0.545); `510f248a`/`6c587a12` = SECONDARY_PUNCTUAL (1 chunk each). |
| 116 | Pre-wid B0..B7 byte analysis. Structure `[B0..B7][18 00 00 00 03 44 0c][slot_byte][wid]`. B0..B7 has <10% purity even for the same prefix (23 unique values for `6d32c7dc`). These bytes are dynamic stream data (counter/timer), NOT a weapon-type discriminant. **Dead end.** |
| 117 | **DEFINITIVE: Dual-wid architecture confirmed.** Full-wid binary scan across all contexts: (a) Unknown wids appear EXCLUSIVELY in Formula A contexts (0 fire/kill events). (b) Known wids appear EXCLUSIVELY outside Formula A (fire/kill events, 0 Formula A hits). The "OTHER" hits for unknowns are Formula A events with initial-state encoding (different pre_wid structure in chunk 0). Architecture: Formula A = cosmetic/skin display wid; kill events = base weapon wid. Two independent systems. |
| 118 | Kill-event chunk overlap for `73284037` pi=4. Jaccard max=0.200 (Cindershot, 3 shared chunks). No known weapon dominates the pi=4 active chunks. MA40 AR, Skewer, Sidekick excluded (jaccard=0). Analysis underpowered in 4v4: multiple players share base weapons. **Dead end** for individual skin attribution via chunk overlap. |
| 119 | Initial-state wid proximity scan. Searched ±256B around all skin wid occurrences in chunk 0/1 for known base weapon wids. Result: 0 known wids found in any proximity window. Initial state encodes skin wid ONLY, not the base weapon alongside it. **Dead end.** |

### Phase 9: Formula-C operationalization (#104)

*[renommé — était Phase 8]*

| # | Finding |
|---|---|
| 104 | Formula-C reconstruction export layer. `inv104_formula_c_reconstruction_export.py` serializes the inv100 non-POV reconstruction into reusable JSON, CSV, and JSONL outputs under `scripts/experimental/outputs/`. Current bundle on `00162144`: 21 chunk assignments, 16 segments, with explicit label/confidence counts. The rerun path was also hardened after cleanup: `inv69` now falls back to its stable CSV cache when `inv67`/`inv68` are absent, and `inv94_formula_c_activity_windows.py` was restored so the Formula-C chain reruns cleanly in a fresh process. |

### Phase 9: Formula-A identity limits (#110)

| # | Finding |
|---|---|
| 110 | Intra-chunk Formula A pi collision audit (`inv110_formula_a_pi_collision_audit.py`). Test: same chunk + same match + same Formula A pi + same weapon slot + at least 2 distinct unknown prefixes. Result: **117 collision chunks across 13 groups**. This is impossible for a single player inventory slot, so Formula A pi is an **alias bucket**, not a guaranteed one-player identity, in those groups. Strongest cases: `63d6f727 pi=5 slot=0x00` (20 collision chunks, 13 distinct prefixes), `000d5950 pi=6 slot=0x81` (18 collision chunks), `00162144 pi=5/6/7 slot=0x00` (13–14 collision chunks each). Practical impact: slot/length structural classification from inv106–107 remains valid per event, but multi-chunk `(match, pi, slot)` timelines from inv108–109 must not be read as single-player attribution when this collision signal is present. Output: `outputs/inv110_formula_a_pi_collisions.json`. |

### Phase 10: Collision-aware timeline filtering (#111)

| # | Finding |
|---|---|
| 111 | Collision-aware re-read of inv109 (`inv111_collision_aware_timeline_filter.py`). After excluding all inv109 groups that are collision-positive in inv110, only **2 safe groups remain out of 15**: `000d5950 pi=4 slot=0x81` (`5ded6cf2`, `9ed65a38`) and `f3bc46ab pi=7 slot=0x81` (`94c3a67a`, `0131ea10`). All other inv109 timeline groups are **alias-unsafe**. Consequence: timeline-based roles from inv109 are only trustworthy inside those 2 safe groups. In particular, `6d32c7dc` had `LOADOUT` in 5 inv109 groups before filtering, but appears in **0 safe groups** after filtering, so inv109 no longer supports a collision-safe loadout claim for `6d32c7dc`. Output: `outputs/inv111_collision_aware_timelines.json`. |

### Phase 11: Collision-aware exclusivity filtering (#112)

| # | Finding |
|---|---|
| 112 | Collision-aware re-read of inv108 (`inv112_collision_aware_exclusivity_filter.py`). After filtering original inv108 exclusivity groups through inv110, the corpus keeps **23 SAFE_EXCLUSIVE groups** and **13 ALIAS_UNSAFE_EXCLUSIVE groups**. This preserves **19 unknown prefixes** with at least one collision-safe exclusivity signal and marks **40 prefixes** as unsafe-only. For `6d32c7dc`, the result is now precise: it has **1 SAFE_EXCLUSIVE group** (`73284037 pi=4 slot=0x00`) and **5 ALIAS_UNSAFE_EXCLUSIVE groups**. Chunk audit on the safe group shows `6d32c7dc` alone on `ck03,05,07,08,09,10,14,18,19,21,25`, with no same-chunk peer in that slot. Practical consequence: inv112 restores a **minimal collision-safe exclusivity signal** for `6d32c7dc` as an exclusive primary-slot wid, but still does **not** restore the stronger inv109-style mono-player timeline/loadout claim. Output: `outputs/inv112_collision_aware_exclusivity.json`. |

### Phase 12: Collision distribution by match (#113)

| # | Finding |
|---|---|
| 113 | Match-level collision distribution audit (`inv113_formula_a_collision_distribution.py`). Crossed inv108 unknown-slot groups with inv110 collision groups to profile where Formula A pi aliasing is concentrated. Result on the 8-match corpus: **`73284037` is the only match with 0 collision groups**, making it the cleanest current control for player-attribution work. Highest collision-group rates are `d9329229` (**4/5 groups = 0.80**), `00162144` (**3/5 = 0.60**), then `ebfb64f2` (**1/2 = 0.50**) and `000d5950` (**2/5 = 0.40**). `63d6f727` has only **1/4** collision-positive groups but that single group is the most severe one in the corpus (`20` collision chunks, `7` concurrent prefixes max). Practical consequence: later attribution work should treat `73284037` as the main clean control, while `d9329229` and `00162144` should be assumed collision-prone by default. Output: `outputs/inv113_formula_a_collision_distribution.json`. |

### Phase 13: Clean control profile for `73284037` (#114)

| # | Finding |
|---|---|
| 114 | Control-match profile (`inv114_control_match_73284037_profile.py`). Materialized the exact safe-exclusive Formula A groups inside the only no-collision match. `73284037` contains **3 safe-exclusive groups**: `pi=4 slot=0x00 -> 6d32c7dc`, `pi=5 slot=0x81 -> 510f248a`, `pi=6 slot=0x81 -> 6c587a12`. Structural classes from inv107: all 3 are `WEAPON_VARIANT`; `6d32c7dc` is the only recurrent one, appearing on **11 chunks** (`ck03,05,07,08,09,10,14,18,19,21,25`) as the sole wid in that primary slot. In the contextual chunk view, `6d32c7dc` sometimes coexists only with `e54f1b22` on `pi5/slot=0x80`, which inv107 classifies as `SPIKE_GRENADE_FAMILY`; this supports a clean reading of `6d32c7dc` as a primary-slot weapon skin in the control match, without restoring any broader mono-player claim outside that match. Output: `outputs/inv114_control_match_73284037_profile.json`. |

### Phase 14: Local temporal roles in the clean control match (#115)

| # | Finding |
|---|---|
| 115 | Temporal-role audit on `73284037` (`inv115_control_match_temporal_roles.py`). Compared the 3 safe groups from inv114 using chunk count, contiguous blocks, fragmentation, and holdtime within the 27-chunk match. Result: `6d32c7dc` (`slot=0x00`) is the **only sustained primary baseline**, with **11 chunks**, **7 blocks**, `holdtime=41%`, `fragmentation=0.55`. By contrast, `510f248a` and `6c587a12` (`slot=0x81`) are both **secondary punctual appearances**, each seen on exactly **1 chunk** (`holdtime=4%`). Practical consequence: in the only clean control match, the local pattern cleanly separates `6d32c7dc` from the other safe unknowns as the sole recurring primary-slot baseline, which is consistent with a primary-weapon skin and inconsistent with a one-chunk pickup-style appearance. Output: `outputs/inv115_control_match_temporal_roles.json`. |

### Phase 15: Binary base-wid resolution — exhaustive (#120–123)

| # | Finding |
|---|---|
| 120 | Post-wid bytes scan in Formula A events (`inv120_formula_a_post_wid_scan.py`). For every confirmed Formula A event (1390 total), extracted 32 bytes after the display wid and searched for any known wid or known suffix (42c9679f / e7232c0b). Result: **0 known wids and 0 known suffixes** in any post-wid window. Post-wid bytes are dynamic (purity 6–16%, similar to B0..B7). The Formula A event structure does not embed a base wid after the display wid. **Dead end.** |
| 121 | Full-corpus adjacency scan (`inv121_full_adjacency_scan.py`). Extended inv119 to ALL occurrences of each exclusive wid (not just initial-state), scanning ±512B for known wids. Result: **0 occurrences** with any known wid in window — for all 5 exclusive wids (97+49+49+110+40 occurrences total). The segregation between Formula-A-exclusive wids and known wids is total throughout the entire binary. **Dead end.** |
| 122 | Suffix proximity scan for unknown base wids (`inv122_suffix_proximity_scan.py`). Instead of searching for KNOWN wids only, scanned ±512B for ANY occurrence of suffix 42c9679f (base wid might not be in our known list). For `6d32c7dc`: 6/97 occurrences have a suffix nearby, all at **exactly +191B** (fixed offset). Single unknown prefix found: `0af3952e42c9679f`. |
| 123 | Fixed-offset pair analysis for `0af3952e` (`inv123_fixed_offset_pair_analysis.py`). `0af3952e` appears **6 times, all in FORMULA_A context** (pb=`a9`, same pi=5). It is NEVER found in kill/fire events. The fixed +191B offset reflects two consecutive Formula A events for the same player (same pi=5 in matches `1bd7303b` and `ebfb64f2`), representing two different inventory slots of that player's loadout. `0af3952e` is itself a Formula-A-exclusive wid — NOT a base weapon wid. **Conclusion: the film binary does not contain any recoverable mapping from Formula-A-exclusive wids to base weapon wids. All binary approaches exhausted.** External data required (game asset files, cosmetic API). |

### Phase 16: External tag dump lookup — InfiniteIDDump v6.10020.17500.0 (#124)

| # | Finding |
|---|---|
| 124 | Tag path lookup via InfiniteIDDump (Krevil, launch build `6.10020.17500.0`). Byte-swap convention confirmed: LE wid prefix bytes reversed → 4-byte uppercase BE → lookup in `globals-rx-new.txt`. E.g. `48c19d2d` LE → `2D9DC148` → `assault_rifle_mp.weapon`. **18/33 known wids resolved** to weapon tag paths in `globals-rx-new.txt`. See table below. `common-rtx-new.txt` yielded 0 additional matches. All **5 unknown wids** (`6d32c7dc`, `f55c4bd2`, `0131ea10`, `d48d9b84`, `0af3952e`) are absent from both launch dump files — confirmed post-Season-3 additions. Grenades (`b6dbead8`, `c1e1bab0`, `6683257c`) are `.equipment` tags in the game's asset system (not `.weapon`); their launch IDs differ from the wids seen in kill events. Some known wids (Gravity Hammer `8afc0855`, Energy Sword `1488d0bb`, Skewer `0d20c469`, Sentinel Beam `c24e549e`) are present in the launch dump under **different IDs** — their tag IDs changed between the launch build and the current live game. The MA5K Avenger, Bandit Evo, M392 Bandit, Mutilator, and Vestige are absent entirely (post-launch additions). **Conclusion: a post-Season-3 tag dump is required to identify the 5 unknown wids.** |

**Known-wid → tag path mapping (from launch dump, `globals-rx-new.txt`):**

| LE prefix | Wid label | Internal tag path |
|---|---|---|
| `2d9dc148` | MA40 AR | `objects\weapons\rifle\assault_rifle\assault_rifle_mp.weapon` |
| `a57b9780` | Mangler | `objects\weapons\proto\proto_spike_revolver\proto_spike_revolver_mp.weapon` |
| `ed29bd84` | Disruptor | `objects\weapons\proto\proto_arc_zapper\proto_arc_zapper_mp.weapon` |
| `b9a88793` | Shock Rifle | `objects\weapons\proto\proto_volt_action\proto_volt_action_mp.weapon` |
| `7e9533b5` | Needler | `objects\weapons\pistol\needler\needler_mp.weapon` |
| `4ad819b6` | Bulldog | `objects\weapons\proto\proto_combat_shotgun\proto_combat_shotgun_mp.weapon` |
| `c7870dc3` | Ravager | `objects\weapons\rifle\provoker\provoker_mp.weapon` |
| `462954c3` | Plasma Pistol | `objects\weapons\pistol\plasma_pistol\plasma_pistol_mp.weapon` |
| `c793f1da` | Stalker Rifle | `objects\weapons\rifle\stalker_rifle\stalker_rifle_mp.weapon` |
| `0f1908f4` | Mk51 Sidekick | `objects\weapons\pistol\sidearm_pistol\sidearm_pistol_mp.weapon` |
| `4c5598fd` | VK78 Commando | `objects\weapons\rifle\commando_rifle\commando_rifle_mp.weapon` |
| `bc92190a` | S7 Sniper | `objects\weapons\rifle\sniper_rifle\sniper_rifle_mp.weapon` |
| `b1470423` | Cindershot | `objects\weapons\rifle\cindershot\cindershot_mp.weapon` |
| `ffc2c92a` | Heatwave | `objects\weapons\rifle\heatwave\heatwave_mp.weapon` |
| `d524182b` | BR75 | `objects\weapons\rifle\br\br_mp.weapon` |
| `a64e4830` | Pulse Carbine | `objects\weapons\rifle\pulse_carbine\pulse_carbine_mp.weapon` |
| `2c0aab71` | M41 SPNKr | `objects\weapons\support_high\spnker_rocket_launcher_olympus\spnker_rocket_launcher_olympus_mp.weapon` |
| `6db97d76` | MLRS-2 Hydra | `objects\weapons\rifle\mlrs\mlrs_mp.weapon` |

**Not found in launch dump (tag IDs changed or post-launch content):**

| LE prefix | Wid label | Reason absent |
|---|---|---|
| `8afc0855` | Gravity Hammer | Launch dump has `E5C51A84` (`gravity_hammer_mp.weapon`) — ID changed post-launch |
| `1488d0bb` | Energy Sword | Launch dump has `7E93F34F` (`energy_sword_mp.weapon`) — ID changed post-launch |
| `0d20c469` | Skewer | Launch dump has `E4BA1B61` (`proto_skewer_mp.weapon`) — ID changed post-launch |
| `c24e549e` | Sentinel Beam | Launch dump has `9E5E95A0` (`proto_hardlight_sentinel_beam_mp.weapon`) — ID changed post-launch |
| `b6dbead8` | M9 Frag | Grenade `.equipment` tag; wid references a weapon object absent from `globals` module |
| `c1e1bab0` | Plasma Grenade | Same — `.equipment` tag; not in `globals` module |
| `6683257c` | Spike Grenade | Same — `.equipment` tag; not in `globals` module |
| `3ad55da4` | Dynamo (hand) | Not in `globals` module; likely in a separate equipment module |
| `18e1fea0` | Dynamo (projectile) | Same |
| `f5c335df` | MA5K Avenger | Post-Season-1 addition; not in launch dump |
| `9d6aaed2` | Fuel Rod SPNKr | Not found in searched modules |
| `6acdc44d` | Bandit Evo | Post-Season-3 addition; absent |
| `2fb21c87` | M392 Bandit | Post-Season-3 addition; absent |
| `d7915565` | Mutilator | Post-launch addition; absent |
| `3e070217` | Vestige | Post-launch addition; absent |
| `6d32c7dc` | UNK_PRIMARY_0x00 | Post-Season-3; absent from all launch dump files |
| `f55c4bd2` | UNK_PRIMARY_0x00_2 | Post-Season-3; absent |
| `0131ea10` | UNK_SECONDARY_0x81 | Post-Season-3; absent |
| `d48d9b84` | UNK_GRENADE_0x01 | Post-Season-3; absent |
| `0af3952e` | UNK_PAIR_WID | Post-Season-3; absent |

---

### Conclusion on Formula-A-exclusive wid resolution

After inv117–123, the evidence is definitive:

1. **Total binary segregation**: Formula-A-exclusive wids and known base wids occupy completely separate byte regions throughout the binary — zero co-occurrences at any scale (±512B, full event, full chunk, all chunks).
2. **No inline mapping**: Formula A events do not embed a base wid before or after the display wid.
3. **No proximity mapping**: Initial-state blocks do not store (display_wid, base_wid) pairs in adjacent memory.
4. **Unknown adjacency resolves to another exclusive wid**: The only structural neighbor found (inv122–123) is `0af3952e`, itself Formula-A-exclusive.

The base weapon for `6d32c7dc` (and other exclusive wids) is **not determinable from the film binary alone**. The mapping must be resolved via external data (game asset files, Halo API cosmetic endpoints) or via corpus expansion (finding matches where the same player appears without the variant, confirming their base weapon from kill events).

**Next step (external data):** Obtain a post-Season-3 tag dump (e.g. using `Infinite-runtime-tagviewer` with the game running, or a community-contributed newer IDDump) to identify the 5 unknown wids. With a current dump, the byte-swap lookup procedure is confirmed to work (18 known wids successfully mapped via inv124).

---

### Phase 17: mohd module entry table parsing — definitive identity of unknown wids (#125)

| # | Finding |
|---|---|
| 125 | Direct parsing of mohd v53 module entry tables (`inv125_mohd_module_entry_scan.py`). Empirically determined layout: **entry size = 88 bytes**; group tag (LE-reversed 4CC) at offset 0; tag ID prefix (BE4 = reversed LE local prefix) at entry offsets **20 and 60** (same value, two copies per entry); header size varies by module file. Verified layout against all 5 sample known wids in `globals-rtx-new.module` (665MB, 78,174 items, H=0x10C): each found exactly once as a `weap` group entry ✓. Scanned all globals-family modules: `globals`, `common` (131MB), `multiplayer`, `multiplayer_r1/r2/r3`, `levels`. **Result: all 6 Formula-A-exclusive wids produce zero hits in every module file.** Known wids are found exclusively in `globals-rtx-new.module`; multiplayer/levels modules contain no `weap` group entries at all (weapon tags not duplicated there). |

**Definitive conclusion — unknown wid identity:**

The 6 Formula-A-exclusive wids (`6d32c7dc`, `f55c4bd2`, `0131ea10`, `d48d9b84`, `67fed82c`, `0af3952e`) do **not** exist as physical weapon tag entries anywhere in the installed game's module files. They are **virtual/abstract identifiers** — most likely weapon slot references (primary slot, secondary slot, grenade slot) defined as runtime constants in the game engine (`HaloInfinite.exe`, which is permission-restricted). Their `42c9679f` suffix matches the "weap" group hash used for all concrete weapon wids, confirming they are semantically weapon-typed, but they have no corresponding game asset.

**Practical implication:** For Formula-A playlist matches, the film binary records **slot identifiers** rather than specific weapon tag IDs. The name of the specific weapon being held by a non-POV player at the time of a kill **cannot be determined** from the film data alone. Only the weapon category (primary/secondary/grenade) is recoverable. This is a fundamental constraint of how the Cinema system encodes loadout data for non-POV players in this playlist format.

**Remaining avenue (outside the binary):** Cross-reference the Formula-A player's weapon slot with Halo API personal score data or armory data from other match types to probabilistically infer the base weapon. This requires external API data, not film binary analysis.

*Investigations #1–125 on branch `experimental/film-weapon-extraction`.*
*Technical note on Formula-C model: `FORMULA_C_TECHNICAL_MODEL.md` (inv #92).*
