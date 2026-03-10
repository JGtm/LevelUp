# Weapon Extraction — Research Summary (inv #1–125) — CLOSED

> Branch: `experimental/film-weapon-extraction` | Last updated: 2026-03-09
> Main corpus: 11 matches, 170+ chunks | Reference: `04f7d9d5` (11/11 kills, 100%)

---

## 1. Film Binary Architecture

Each match film = ~28 chunks × 2 sections:

```
Section 1 — State snapshot (~85% of bytes)
  Formula A: [20 00 02 pb][N bytes][slot_byte][wid:8B]   → teammate weapon state
    pi = pb >> 5  (warning: aliases in large matches, inv #110)
    slot_byte: 0x00=primary, 0x81=secondary, 0x80=Spike, 0x01=Dynamo/grenade, 0x27=power
  Formula B: [pi_byte][44/34 0C][flags][wid:8B]          → POV/teammate state (independent)
  Formula C: [20 00 03 pb]...  — rare variant, 1/11 matches only (00162144)

Section 2 — Frame events (~15% of bytes, nibble-shifted)
  Fire events: (byte[0] & 0x07)==0x05, (byte[3] & 0xFC)==0x40, wid in [T-5000ms, T]
  Melee events: byte[0]==0x40, wid at variable offset (13–18B)
  → POV player ONLY (pi=1). Server does not replicate opponent fire events. Architecturally final.
```

**Dual-wid architecture (inv #117 — confirmed):**
- Formula A events = **cosmetic/skin wid** (display layer)
- Kill/fire events = **base weapon wid** (gameplay layer)
- The two systems are binary-segregated: zero co-occurrence across all 1390 Formula A events and all kill events.

---

## 2. What Works — Production Algorithms

### POV Kill Attribution (weapon-swap-aware, inv #1–54)

```
1. Collect fire events (Section 2, nibble-shifted, deduplicate by (wid, fire_counter))
2. For kill at time T: wid = last fire event with wid in [T-5000ms, T]
3. Confidence:
   Zone A  delta < swap_ms(W)             → HIGH
   Zone B  swap_ms ≤ delta ≤ travel_max   → MEDIUM (HIGH if no swap weapon found after T)
   Zone C  delta > travel_max             → LOW, delayed_damage=True (area/grenade)
4. API reconciliation: compare attributed class counts vs grenade_kills/melee_kills/
   power_weapon_kills/headshot_kills; promote/demote at margins.
```

### T1 Teammate Kill Attribution (snapshot-based, inv #35–39)

```
1. Build weapon timeline from Section 1:
   Formula A: weapon_A[chunk][pb>>5] = last wid seen
   Formula B: weapon_B[chunk][pi_byte & 0x07] = last wid seen
2. At kill time T by xuid X:
   pi = pi_from_xuid_map[X]
   wid = weapon_A[chunk(T)].get(pi) or weapon_A[chunk(T)-1].get(pi)
3. Confidence: HIGH if wid in WEAPON_ID_MAP and no swap; MEDIUM if swap in window;
   LOW if wid unknown (skin variant); NONE if pi not in map (T0 opponents — always NONE)
4. API reconciliation (same as POV step 4)
```

**Coverage:** 96.8% T1 kills attributed in reference match (inv #37).

### `weapon_kills` table schema

```sql
CREATE TABLE weapon_kills (
    match_id       TEXT,
    xuid           BIGINT,
    time_ms        INTEGER,
    weapon_name    TEXT,       -- None if unknown skin
    delta_ms       INTEGER,
    confidence     TEXT,       -- 'high' / 'medium' / 'low' / 'none'
    swap_detected  BOOLEAN,
    delayed_damage BOOLEAN
);
```

### Weapon class timing

| Class | swap_ms | travel_max_ms |
|---|---|---|
| Sidearm (Sidekick, Plasma Pistol) | 400–450 | ≤300 |
| Standard (MA40, BR75, Bulldog, Mangler, etc.) | 650 | ≤500 |
| Slow projectile (Heatwave, Needler, Hydra) | 650 | ≤2000 |
| Area/delayed (Ravager, Disruptor chain) | 650 | ≤5000 |
| Heavy draw (S7 Sniper, Stalker) | 900 | ≤300 |
| Heavy slow projectile (Skewer) | 900 | ≤3000 |
| Heavy area (Cindershot) | 900 | ≤5000 |
| Launcher (M41/Fuel Rod SPNKr) | 1100 | ≤2000 |
| Melee/destroy (Gravity Hammer, Energy Sword, Mutilator) | 1100 | ≤1400 |
| Grenade (Frag 1650, Plasma 1350, Spike 1550, Dynamo 1500) | 950 | ≤1650 |

---

## 3. What Is Impossible — Final

| Target | Verdict | Root cause |
|---|---|---|
| T0 opponent fire events | **Impossible** | Server asymmetric replication: opponent weapon states not transmitted to client (inv #27, #39, #44–53) |
| T0 opponent weapon at kill time | **Impossible** | Follows from above; no indirect method found (inv #50, #118–119) |
| Base weapon behind Formula-A skin wid | **Impossible from binary** | Zero co-occurrence of skin wid + base wid at any scale (inv #119–123). Not in any game module file (inv #125) |
| Formula-C generalization | **Not viable** | Appears in 1/11 matches; unknown reason; wids produced are skin variants too (inv #77, #90) |

---

## 4. Known Weapon IDs

### Group A — Confirmed

| Wid (16 hex chars) | Name |
|---|---|
| `48c19d2d42c9679f` | MA40 AR |
| `2b1824d542c9679f` | BR75 |
| `fd98554c42c9679f` | VK78 Commando |
| `6acdc44d42c9679f` | Bandit Evo |
| `2fb21c8742c9679f` | M392 Bandit |
| `b619d84a42c9679f` | CQS48 Bulldog |
| `f408190f42c9679f` | Mk51 Sidekick |
| `c354294642c9679f` | Plasma Pistol |
| `30484ea642c9679f` | Pulse Carbine |
| `80977ba542c9679f` | Mangler |
| `9387a8b942c9679f` | Shock Rifle |
| `84bd29ed42c9679f` | Disruptor |
| `c30d87c742c9679f` | Ravager |
| `b533957e42c9679f` | Needler |
| `2ac9c2ff42c9679f` | Heatwave |
| `daf193c742c9679f` | Stalker Rifle |
| `0a1992bc42c9679f` | S7 Sniper |
| `230447b142c9679f` | Cindershot |
| `0d20c46942c9679f` | Skewer |
| `71ab0a2c42c9679f` | M41 SPNKr |
| `9d6aaed242c9679f` | Fuel Rod SPNKr |
| `767db96d42c9679f` | MLRS-2 Hydra |
| `c24e549e42c9679f` | Sentinel Beam |
| `8afc085542c9679f` | Gravity Hammer |
| `1488d0bb42c9679f` | Energy Sword |
| `d791556542c9679f` | Mutilator |
| `3e07021742c9679f` | Vestige Carbine |
| `f5c335dfe7232c0b` | MA5K Avenger *(different group suffix)* |
| `b6dbead842c9679f` | M9 Frag Grenade |
| `c1e1bab042c9679f` | Plasma Grenade |
| `6683257c42c9679f` | Spike Grenade |
| `3ad55da442c9679f` | Dynamo Grenade (hand/loadout) |
| `18e1fea042c9679f` | Dynamo Grenade (projectile — Super Fiesta only) |

### Formula-A-exclusive wids (virtual slot identifiers — not physical weapon tags)

These wids appear exclusively in Formula A state events (never in kill/fire events). They are **not** `weap` group entries in any game module file (inv #125). They encode the player's **inventory slot**, not the specific weapon.

| Wid | Slot | Structural family (inv #106–107) |
|---|---|---|
| `6d32c7dc42c9679f` | Primary (0x00) | WEAPON_VARIANT — primary baseline, holdtime ~41% |
| `f55c4bd242c9679f` | Primary (0x00) | WEAPON_VARIANT — primary variant |
| `0131ea1042c9679f` | Secondary (0x81) | WEAPON_VARIANT — secondary slot |
| `d48d9b8442c9679f` | Grenade (0x01, len=16) | GRENADE_VARIANT — secondary grenade |
| `67fed82c42c9679f` | Spike Grenade (0x80) | SPIKE_GRENADE_FAMILY — grenade skin |
| `0af3952e42c9679f` | Pair wid | Formula-A only, adjacent loadout slot reference |

### Group B — Skin variants (unconfirmed base weapon)

Classified by slot_byte + pre_wid_len. Base weapon irrecoverable from binary.

| Slot family | Top unknown wids |
|---|---|
| SPIKE_GRENADE (0x80, len=16) | `91eb16de`, `60f1d512`, `87fab1d4` |
| DYNAMO (0x01, len=20) | `6a672afc`, `b5e3278e` |
| GRENADE_VARIANT (0x01, len=16) | `92f99df4` |
| GRENADE_VARIANT (0x01, len=15) | `f9514800`, `edff0e96` |
| WEAPON_VARIANT primary (0x00, len=16) | `82a3f54a` |
| WEAPON_VARIANT secondary (0x81, len=16) | `d0b802c4`, `5ded6cf2`, `510f248a`, `6c587a12` |

---

## 5. Known Issues (production edge cases)

1. **Formula A pi aliasing** — `pb >> 5` is a 3-bit field; in large matches or matches with many players, multiple real players share the same pi value (13 collision groups in 7/11 matches, inv #110). Attribution to a specific player via pi alone is unreliable except in the clean control match (`73284037`).
2. **Silent grenade misattribution** — fire event after a throw may attribute the kill to the grenade. Mitigation: `delayed_damage` flag + API reconciliation Step 4.
3. **Ravager overcharge** — zone persists 4–5s; treat as grenade class if delta > 3000ms.
4. **Nibble-shift duplicates** — same fire event at nibble=0 and nibble=1; deduplicate by `(wid, fire_counter)`.
5. **Post-match shots** — shots after match end are in film but not in API ShotsHit/ShotsFired; expected discrepancy.
6. **Empty chunks** — `e44bfaaa` chunks 21/24/25 have 0 events (encoding variant or unknown IDs).

---

## 6. mohd Module Format (inv #125 — reference)

For future tag ID lookups in game module files:

| Field | Value |
|---|---|
| Magic | `mohd` |
| Version (v53) | `35 00 00 00` at offset 4 |
| Item count | offset 0x10 (LE u32) |
| Entry size | **88 bytes** |
| Group tag within entry | offset +0 (LE-reversed 4CC; "weap" stored as `70 61 65 77`) |
| Tag ID prefix (BE4) | offsets **+20** and **+60** (same value, two copies) |
| Header size (`globals-rtx-new`) | 0x10C |

Byte-swap for lookup: `le_prefix_bytes[::-1].hex().upper()` → matches IDDump / module entry.

*Research closed. Investigations #1–125 on branch `experimental/film-weapon-extraction`.*
