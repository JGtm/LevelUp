# How the weapon parser works

> Status: branch `analysis/weapon-parser-rewrite` — 2026-03-11

---

## The problem to solve

The game (via the Halo API) provides the **count** of kills per weapon type for a
match, but not the per-kill detail. To know "what weapon JGtm used to kill at t=3:24",
the only source is the match **film** — a binary file downloadable from Xbox servers.

The parser reads this binary file and maps each kill to its weapon.

**3-layer architecture — who does what**

| Layer | File | Role |
|-------|------|------|
| **Parser** | `weapon_parser.py` | Reads the film, correlates kills↔fire events, returns an in-memory list of dicts. **Never writes to DB.** |
| **Extraction service** | `weapon_extraction_service.py` | Orchestrates the parser for all players in the match, optionally applies post-processing layers such as API reconciliation and sentinel assignment, produces the final `kill_rows`. **Also never writes to DB.** |
| **Repository** | `_weapon_kills_repo.py` → `insert_weapon_kill_rows()` | Sole writer to the `weapon_kills` table (shared_matches.duckdb). Called by the service after it has finalized the data. |

**Storage note (rewrite opportunity):** beyond parsing itself, the data model may evolve
to expose weapon attribution directly in kill-pair analytics (`killer_victim_pairs`).
This is a downstream storage/design concern, not a parser responsibility.

The parser **does not produce** weapon_ids. Weapon_ids (8 bytes) are binary identifiers
set by the game and encoded as-is in the film. The parser's role is to:
1. **Read** these identifiers from the film
2. **Map them** to a weapon name via the static dictionary `WEAPON_ID_MAP` (built
   from manual film investigations)
3. **Associate them** with a specific kill via the time window

Discovering or updating weapon_ids is a separate **film investigation** task not the parser's responsibility.

**Design rule: parser output first, optional reconciliation second.**
Film parsing/correlation must remain valid on its own. API reconciliation and sentinel
assignment are **decoupled post-processing phases** layered on top of the raw film
result, not part of the parser contract. They must be switchable independently at any
time. Today they stay **enabled by default** because the current API and film signals
still leave gaps they help cover, but if the API later exposes reliable per-kill
attribution these phases should be disableable without redesigning the parser.

---

## The two sources of information in the film

A match film is split into **chunks** of approximately 19 seconds. Each chunk
contains two types of useful data:

### Section 1 — Player state (Formula A)
A periodic snapshot that says: "at this moment, player N is holding this weapon".
It is a state snapshot, not an event. It updates when a player swaps or picks up a weapon.

### Section 2 — POV fire events
Each time the **POV** (the player whose film it is) pulls the trigger, it generates
an event containing:
- The timestamp of the shot (in ms)
- The weapon used (weapon_id 8 bytes)
- The player index (player_index)
- An aim vector encoded as octahedral 3D→2D

Fire events are found in the **nibble-shifted layer** of each replication chunk
(REPLICATION_DATA). This layer is obtained as follows:

```python
nibble_shifted = bytes(
    (data[i] << 4 | data[i + 1] >> 4) & 0xFF
    for i in range(len(data) - 1)
)
```

**Fire event structure** (offsets within the nibble-shifted layer):

| Offset | Value | Description |
|--------|-------|-------------|
| [0] | `0x0D` or `0x05` | Lead byte |
| [1] | `(player_index << 5) \| 0x06` | e.g. POV (idx=1) = `0x26`, player 2 = `0x46` |
| [2] | variable | `b2_stream` — dual-stream discriminator |
| [3] | `0x40`–`0x43` | Constant (filter: `byte[3] & 0xFC == 0x40`) |
| [4] | `0`–`248` (step 8) | `fire_counter` — shot counter (0–127 then reset); can skip values (lost frames) |
| [5] | variable | `b5_correlated` — correlated with `b2_stream` |
| [6–13] | 8 bytes | `weapon_id` |
| [14] | `0`–`7` | Aim octant (octahedral encoding) |
| [15–16] | `uint16` | Magnitude within the octant |

**Alternative scan method** (acurtis166): search for the bit pattern
`0b101_0010_0110` (corresponds to `0xD26` / `0x526` depending on lead byte) in the
nibble-shifted layer, then validate that the 64 bits at offset +40 bits (`_WEAPON_OFFSET`)
match a known `weapon_id` in the `Weapon` enum.

**Automatic weapons — dual-stream**: BR75, MA40 AR and similar generate **two**
entries per shot with different `b2_stream` values but the same `fire_counter`.
Semi-auto weapons (Bandit, Stalker, Commando) produce **one** entry. Deduplication
key: `(weapon_id, fire_counter)` per chunk.

**Bits after weapon_id**:
- 2nd bit = `0` for shots mid-burst, `1` for the last shot (e.g. BR75: sequence 0-0-1)
- 3rd bit ≈ hit/miss (`0`=hit, `1`=miss), not 100% reliable; a second bit further in
  the structure confirms

**Current working assumption**: Section 2 is reliably available for the POV, while
non-POV coverage is still under investigation. The historical pipeline treated fire
events as POV-only, but recent service-level findings suggest this may have been at
least partly a routing limitation rather than a hard parser limitation.

> **Rewrite prerequisite**: before locking the final architecture, start with an
> **initial exploration phase** based on [.ai/NON_POV_FIRE_EVENTS_CONCLUSIONS_2026-03-12.md].
> The goal is to measure whether non-POV fire events are present and reliable enough
> across real matches to support attribution for teammates broadly, not just for the
> film owner.

> **Current scope decision (rewrite baseline)**: opponents are **out of scope for
> now**.
> Historical results indicate very low exploitable coverage for opponents and a high
> NULL rate when trying to include them (see table below). For the rewrite baseline,
> only the POV and teammates with a resolved `player_index` are targeted.
> Opponent coverage is explicitly **gated by the initial exploration phase**:
> if that phase (including acurtis-confirmed findings) delivers strong, repeatable
> non-POV fire-event attribution quality, opponents should be brought into scope;
> otherwise, they remain excluded in the hybrid baseline.

---

Note: weapon variants (e.g. BR75 Ranked, S7 Flexfire) share the **same weapon_id**
as the base weapon.

**Confirmed weapon list** (source: acurtis166, February-March 2026):

| Weapon | weapon_id (hex) |
|--------|----------------|
| Bandit Evo | `6ACDC44D42C9679F` |
| BR75 (= BR75 Ranked) | `2B1824D542C9679F` |
| Cindershot | `230447B142C9679F` |
| CQS48 Bulldog | `B619D84A42C9679F` |
| Diminisher of Hope | `841AC5E5A730E49F` |
| Disruptor | `84BD29ED42C9679F` |
| Duelist Energy Sword | `4FF3937E8978AA7A` |
| Elite Bloodblade | `4FF3937E1EC48C7A` |
| Energy Sword | `4FF3937E42C9679F` |
| Fuel Rod SPNKr | `9D6AAED242C9679F` |
| Gravity Hammer | `841AC5E542C9679F` |
| Heatwave | `2AC9C2FF42C9679F` |
| Infected Energy Sword | `0C55765F7A9376A0` |
| M392 Bandit | `2FB21C8742C9679F` |
| M41 SPNKr | `71AB0A2C42C9679F` |
| MA40 AR | `48C19D2D42C9679F` |
| MA5K Avenger | `F5C335DFE7232C0F` |
| Mangler | `80977BA542C9679F` |
| Mk51 Sidekick | `F408190F42C9679F` |
| MLRS-2 Hydra | `767DB96D42C9679F` |
| Mutilator | `D791556542C9679F` |
| Mythic Sandwich | `B7262CA1C8FB11D0` |
| Needler | `B533957E42C9679F` |
| Plasma Pistol | `C354294642C9679F` |
| Pulse Carbine | `30484EA642C9679F` |
| Ravager | `C30D87C742C9679F` |
| Rushdown Hammer | `841AC5E5D8D07CA1` |
| S7 Sniper | `0A1992BC42C9679F` |
| Sandwich | `880FE0BC42C9679F` |
| Sentinel Beam | `A0955E9E42C9679F` |
| Shock Rifle | `9387A8B942C9679F` |
| Shock Rifle (Ranked) | `1A22FEE642C9679F` |
| Skewer | `0D20C46942C9679F` |
| Stalker Rifle | `DAF193C742C9679F` |
| Vestige Carbine | `3E07021742C9679F` |
| VK78 Commando | `FD98554C42C9679F` |

## Chunk binary structure (packet header)

Each chunk of the film is split into **variable-length packets**, each preceded by
a **16-byte header** (acurtis166, March 2026). This structure allows indexing all
sections of the film without reading the entire data.

| Field | Type | Description |
|-------|------|-------------|
| Type | `uint16le` | Packet type |
| byte2 | `uint8` | — |
| byte3 | `uint8` | — |
| Size | `uint32le` | Packet data size (bytes) |
| Timestamp | `uint64le` | Timestamp in microseconds |

Known packet types:

| Value | Name | Note |
|-------|------|------|
| 0 | `FRAME` | Data frame |
| 1 | `START_CHUNK` | Chunk start |
| 2 | `TYPE_2` | — |
| 6 | `TYPE_6` | — |
| 7 | `END_CHUNK` | Chunk end — marks end of iteration |
| 8 | `PLAYER_METADATA` | Player metadata |
| 9 | `HIGHLIGHT_EVENTS_START` | Highlight-events section marker |
| 10 | `TYPE_10` | Interleaved with frames |
| 12 | `TYPE_12` | — |

**Rewrite recommendation (acurtis packet model):** use packet-indexed traversal as a
first-class primitive, not just raw byte scanning. In practice, this brings two major
benefits:
- **Targeted search in useful data only**: quickly focus on relevant packet zones
  (e.g. replication/frame payloads) and avoid scanning unrelated payload regions.
- **More reliable timing/correlation**: rely on packet microsecond timestamps instead
  of approximate marker interpolation, which improves kill↔event matching stability.

This packet-aware approach should be used together with chunk-level filtering so the
pipeline downloads/parses only time windows that can actually contribute to weapon
attribution.

---

## Resolving player_index

Each player in the film is identified by a `player_index` (0–31). To associate a
XUID (format `xuid(1234567890123456)`) with their film index (acurtis166 method):

```python
from bitstring import Bits

def get_player_index(bits: Bits, player_id: str) -> int:
    """Returns the player_index by searching for the player's XUID in the film."""
    if player_id.startswith("bid"):
        return -1  # bots not supported
    term = Bits(uintle=int(player_id[5:-1]), length=64)
    position = bits.find(term, bytealigned=False)
    if not position:
        raise ValueError(f"Player {player_id} not found")
    # The 5 bits preceding the first occurrence of the XUID = player_index
    return bits[position[0] - 5 : position[0]].uint
```

Application rules:
- The index is **stable** across chunks → can be cached after the 1st chunk
- Use the `join_time` from the stats payload to **skip chunks prior** to the player
  joining the match
- Skip already-resolved players (optimization: stop after first occurrence)

**Option B — `PLAYER_METADATA` packet (type 8)**:
The `pi → xuid` associations can also be extracted from the type-8 packet
(`PLAYER_METADATA`, ~25KB) present in the first chunk, via `detect_pi_from_metadata`.
This is faster than the acurtis method as it avoids the bitstring scan on the full
chunk (~700KB). The acurtis method above is used as a **fallback** if the METADATA
packet is absent or does not cover all players.

> To verify: reliability of Option B on older replays / matches with late-joining
> players (the packet may be incomplete).

---

## The two attribution paths

For each player in a match, the pipeline selects a path depending on whether they
are the POV or not.

### Rewrite phase 0 — validate the non-POV fire-event hypothesis

The rewrite should **not** immediately freeze the current two-path model as a final
truth. It should begin with an explicit exploration phase driven by the findings in
[.ai/NON_POV_FIRE_EVENTS_CONCLUSIONS_2026-03-12.md].

What must be validated:
- whether Section 2 fire events can be detected for non-POV players consistently
- whether detected non-POV events are frequent and reliable enough to replace T1
  snapshot attribution in practice
- whether a conservative merge remains necessary because non-POV fire-event coverage
  is partial or uneven across matches/players

Decision rule for the rewrite:
- if non-POV fire events are confirmed as broadly reliable for all in-scope players,
  the architecture can move toward **Path A only** (fire-event-driven attribution)
- otherwise, keep the **two-path model**: Path A where fire events are strong,
  Path B as fallback/backup where Section 2 coverage is missing or weak

This exploration is meant to happen **before** deeper parser redesign, because it
directly determines whether the rewrite should converge toward a single attribution
path or preserve the current hybrid architecture.

### Path A — POV (Section 2, fire events)
**Who**: the player elected as "owner" of the film via a private, undocumented
Microsoft method.

**How**:
1. Kill timestamps (`time_ms`) are loaded **from the DB** (`highlight_events`) before
   any chunk download — they are used both to filter relevant chunks and as reference
   timestamps for correlation
2. All fire events are scanned for `player_index = 1` (the POV), accumulating a
   global list across all chunks
3. For each kill at t=T, search for the last **unclaimed** fire event in `[T−5s, T]`
4. The weapon from that fire event becomes the kill's weapon

**Reliability**: high. 5s window captures delayed-damage weapons (Cindershot,
Ravager, etc.). Short delta = high confidence, long delta = reduced confidence
(possibly a missed shot just before?).

### Path B — T1 teammates (Section 1, snapshots)
**Who**: teammates whose `player_index` can be resolved in the film.
**Out of scope**: opponents — their data is not accessible.

**Status of non-POV fire events: unresolved, must be explored first.** Historically,
Path B existed because the service routed non-POV players straight to Formula A
snapshots, and field observations suggested Section 2 was POV-centric. However, the
findings captured in [.ai/NON_POV_FIRE_EVENTS_CONCLUSIONS_2026-03-12.md] indicate the
parser can scan arbitrary `player_index` values and that non-POV Section 2 attempts
should be evaluated explicitly during the rewrite.

**Practical consequence:** Path B remains the safe fallback model unless and until
that exploration proves Section 2 coverage is strong enough to replace it.

**How (Section 1 only)**:

Section 1 contains **Formula A** events (`scan_formula_a`): each time a player swaps
or picks up a weapon, the film records a snapshot `[20 00 02 pb ... wid:8B]`
where `pi = pb >> 5`. These events are scanned to reconstruct a chunk-granularity
timeline.

`build_weapon_timeline` produces for each chunk:
- `timeline[chunk_idx][pi]` = **last** weapon seen for this `pi` (state at end of chunk)
- `swap_pis[chunk_idx]` = set of `pi` values that had **> 1 distinct weapon** in the
  chunk (intra-chunk swap detection)

T1 attribution for a kill at `t_ms`:
1. Find the chunk covering `t_ms` via `find_chunk_at_time`
2. Read `timeline[chunk][pi]` → weapon_id
3. If `pi in swap_pis[chunk]`: `confidence = "medium"` (swap detected within ~19s
   window, impossible to tell if before or after the kill)
4. Otherwise: `confidence = "high"` (known weapon) or `"low"` (unknown hex)

**Edge case**: if `timeline[chunk]` is empty for this `pi`, fall back to
`timeline[chunk - 1]` (the previous chunk).

**Reliability**:
- `confidence=high`: player did not change weapon during the chunk — solid attribution
- `confidence=medium`: a Formula A swap was detected in the same chunk (~19s).
  The stored weapon is the **last** one held in that chunk, not necessarily the one
  used for the kill. This is the main source of T1 inaccuracy.
- `confidence=low`: held weapon identified (hex present) but not in `WEAPON_ID_MAP`

**Possible improvement**: Formula A events have a byte position within the chunk.
Combined with `build_frame_estimator` (timestamp interpolation by byte position),
it would be possible to estimate at which millisecond the swap occurred and compare
to `kill_t` → upgrade `medium` to `high` if the swap is after the kill, or confirm
the pre-swap weapon if the swap is before.

---

## How weapons are identified (hex → name mapping)

Each weapon in the film is represented by **8 bytes** (a binary identifier). The
vast majority of "standard" weapons share the same suffix (`42c9679f` for the last
4 bytes). Special weapons (Energy Sword variants, Gravity Hammer variants) have
different suffixes.

`WEAPON_ID_MAP` in `_weapon_data.py` maps these 8 bytes to a weapon name. Only
weapons **confirmed by direct film investigation** are in this dictionary. An unknown
hex remains unknown — the kill is stored with `confidence=low` and its raw numeric ID.

**Key principle: every hex is recoverable, known or not.**
Since all fire events share the same format and the weapon_id is always at bytes
[6–13], the parser can extract and store this ID even if it is not in `WEAPON_ID_MAP`.
An unknown weapon_id today can be resolved later (film investigation, new dictionary
entry) without re-parsing the film — the raw ID is already in the database.

**Exception — sentinels: `weapon_id = 0`, `1`, `2` do not come from the film.**
These three values are **artificial IDs** assigned by medal-based detection logic,
not weapon_ids read from the film:

| Value | Constant | Meaning |
|-------|----------|---------|
| `0` | `GRENADE_WEAPON_ID` | Kill attributed to a grenade (via medals) |
| `1` | `MELEE_WEAPON_ID` | Kill attributed to a melee strike (via medals) |
| `2` | `VEHICLE_WEAPON_ID` | Kill attributed to a vehicle |

A real film weapon_id converted to `uint64` is always a very large number
(e.g. `0x6ACDC44D42C9679F` ≈ 7.7 × 10¹⁸). Values 0, 1 and 2 can never appear
naturally in the film — they never collide with real IDs. If `weapon_id = 0`
appears in the database, it means medal detection classified the kill as a grenade,
**not** that a hex `000...0` was read from the film.

**Important design constraint:** sentinels are an **optional compatibility layer**,
not a core parser output. The system must be able to turn them off entirely and keep
serving raw film attribution only. They are currently kept **active by default**
because they remain necessary to compensate for present-day data gaps, but they should
be easy to disable or remove if the API eventually makes them obsolete.

### Design note: should multiple weapon_ids per kill be stored?

**Intuition**: sentinels are fallbacks. If detection improves in the future, rows
with `weapon_id = 0/1` in the database become useless — we have overwritten
potentially valid data with an approximation.

**Initial counter-argument:** apply a confidence hierarchy at write time — sentinel
only allowed on `NULL` or `confidence=low`, never on `medium` or `high`. This fixes
Step 4b without changing the schema.

**Limit of this counter-argument:** even with the hierarchy, we still overwrite
the film hex of a `confidence=low` kill. If that hex is identified later ("oh, that
was a Needler"), the database row says "melee" and the film data is lost — the film
would need to be re-parsed.

**Mask approach (adopted proposal)**:
Add a `reconciled_as UBIGINT` column (NULL by default) separate from `weapon_id`.

| Column | Content | Mutable? |
|--------|---------|----------|
| `weapon_id` | Raw hex read from the film. Never overwritten. | ❌ no |
| `reconciled_as` | Sentinel assigned by API reconciliation (`0/1/2`). | ✅ yes, reversible |

Effective attribution for queries = `COALESCE(reconciled_as, weapon_id)`.

**Advantages:**
- Zero film data loss: the hex is always recoverable
- Reversible: `UPDATE SET reconciled_as = NULL WHERE ...` if detection improves
- Traceable: distinguishes "observed in film" vs "elected by API"
- Compatible with future hex resolution without re-parsing

**Cost:** all queries must use `COALESCE(reconciled_as, weapon_id)`. This risk of
omission is mitigated by creating a view:
```sql
CREATE VIEW v_weapon_kills AS
SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id
FROM weapon_kills;
```
Consumers (UI, aggregates) read `v_weapon_kills.effective_weapon_id`,
never `weapon_id` alone.

**Eligibility rule for `reconciled_as`:**

| `confidence` (Python string value) | `reconciled_as` may be set? |
|-------------------------------------|-----------------------------|
| `"high"` | ❌ never |
| `"medium"` | ❌ never |
| `"low"` | ✅ yes — unknown hex, API is more reliable |
| `"none"` | ✅ yes — no film data (no fire event / no snapshot) |

> **Note**: the value `"none"` (Python string) corresponds to what was referred to
> as `NULL` in conceptual discussions. In DuckDB, the `confidence` column stores
> the string `'none'` — never a SQL `NULL`. Do not confuse the two.

---

## Sentinels (melee, grenade, vehicle)

Sentinel assignment must be implemented as a **separate, optional phase** after raw
film attribution has been computed. It should never be entangled with low-level film
parsing. Current default: **enabled**, because sentinel inference is still needed in
today's data pipeline.

### Current detection: medals

The current method uses **medals** obtained within 500ms around the kill to classify
sentinels:

| Medal present | → Attribution |
|---------------|---------------|
| Pummel, Back Smack, Ninja, Assassination, Pancake… | `weapon_id = 1` (melee) |
| Sticky Fingers, Grenadier, Boom!, Stick… | `weapon_id = 0` (grenade) |
| — (no specific medal) | Normal path (fire event / snapshot) |

### Melee events in the film (acurtis166, March 2026)

> **POV only.** Melee events share the same nibble-shifted layer and the same
> encoding structure as fire events. Since fire events are confirmed POV-only,
> melee events are very likely POV-only as well — not exploitable for T1 teammates
> or opponents.

Melee strikes have their own event type in the film, identified by the marker
`0xd340`. They share the **same weapon_ids** as fire events and add an
**animation type** field (`5` or `d`) that distinguishes the two melee animations
possible per weapon:

| Weapon | Animation `5` | Animation `d` |
|--------|---------------|---------------|
| BR75 | Weapon toe bottom→top | Weapon toe right→left |
| Energy Sword (all variants) | Diagonal slash top-left | Stab right→left |
| Gravity Hammer / Rushdown / Diminisher / Mythic Sandwich | Smash A | Smash B |
| Mk51 Sidekick | Left hand | Weapon grip |
| Mangler | Bayonet left→right | Bayonet top-right→bottom-left |
| MA40 AR / Cindershot / Bulldog / Heatwave / Bandit Evo / Hydra / Commando / Stalker / Pulse Carbine / Sniper / Sentinel Beam / Vestige | Right elbow | Weapon toe |
| Needler | Weapon base | Needle stab |
| Skewer / Ravager | Bayonet slash | Bayonet stab |

The "fire" button of melee weapons (Hammer, Sword) uses the same weapon_ids with
the `0xd340` marker but may have a slightly different structure (not fully
characterized).

**Opportunity for the new parser**: melee film events would allow attributing melee
kills directly without relying on medals.

---

## API reconciliation

The Halo API provides, per match and **per player** (table `match_participants`), the
columns `grenade_kills` and `melee_kills`. These end-of-match aggregates are available
for **all** players — POV and teammates alike.

**Design rule:** API reconciliation is a **decoupled, optional post-processing phase**.
The parser/service must be able to run without it and keep the unreconciled film
result. In practice, reconciliation is **enabled today** because it is still useful to
compensate for missing or ambiguous film data, but it must remain easy to switch off
at any time if the API evolves and makes this layer unnecessary.

The current parser only uses them for the POV (T1 reconciliation is absent). For the
new parser, this signal can also be used for teammates: if the parser attributes 3
grenade kills to a teammate but the API counts 1, there is an overestimate.

Current mechanism (POV only):

- **Too many weapon kills** → demote the least certain from `high` to `medium`
- **Too few melee/grenade kills** → reclassify the most uncertain kills as
  melee/grenade to make up the deficit (Step 4b)
- **Too few weapon kills** → promote `medium` to `high`

**Friction point**: Step 4b draws from the least certain kills, and T1 kills with
unknown hex (`confidence=low`) are the first candidates. Result: real kills with an
unknown weapon can be reclassified as grenade/melee to "balance the books" with
the API.

**Opportunity for the new parser**: extend grenade/melee validation via
`match_participants.grenade_kills` / `melee_kills` to teammates as well, not just
the POV.

**Interaction with the non-POV exploration phase:** if Section 2 proves reliable for
non-POV players, API reconciliation should be evaluated on top of those fire-event
results first. If Section 2 remains partial, reconciliation must continue to support
the hybrid model and work across both Path A and Path B outputs.

---

## Why there are NULL weapon_ids in the database

A NULL kill means the parser found **no information** about the weapon:
- POV path: no fire event in the 5s before the kill (vehicle kill, edge case)
- T1 path: the film encoded no snapshot for this player in this chunk

The table below shows the reality in the database (85,247 kills total):

| State | Kills | % |
|-------|------:|--:|
| NULL — no info | 54,313 | 63.7% |
| Unknown hex (`conf=low`) | 15,377 | 18.0% |
| Identified weapon (`conf=high`) | 10,631 | 12.5% |
| Melee / Grenade sentinel | 4,447 | 5.2% |

NULLs come almost exclusively from the T1 path for opponents — which is why they
are excluded from the new parser's scope: a player's film encodes neither fire events
nor weapon snapshots for opposing players in any exploitable way.

---

## Data model opportunity (outside parser scope)

During the rewrite, it is worth explicitly challenging where weapon attribution should
live for analytics consumers:

### Candidate A — keep `weapon_kills` canonical, enrich `killer_victim_pairs`

Current approach remains: parser/service produce kill-level weapon attribution in
`weapon_kills` (canonical source), then a downstream step enriches
`killer_victim_pairs` with weapon fields (or a dedicated view joins both).

Pros:
- clear separation of concerns (parsing vs analytics projection)
- low risk for parser rewrite (no coupling to pair-generation internals)
- preserves full weapon metadata (`confidence`, source, reconciliation details)

Cons:
- requires a stable join key strategy and synchronization discipline
- risk of drift if enrichment pipeline is not strictly managed

### Candidate B — move/merge weapon attribution into `killer_victim_pairs`

Alternative: make `killer_victim_pairs` the primary storage for weapon-attributed kill
events, potentially reducing joins for downstream queries.

Pros:
- simpler consumption for rivalry/duel analytics
- fewer joins in some dashboards

Cons:
- tighter coupling between parser outcomes and pair-building logic
- potential loss/flattening of parser-specific metadata unless schema is expanded
- higher migration risk during rewrite

### Recommended baseline for rewrite

Start with **Candidate A** (canonical `weapon_kills`, enrichment/projection into
`killer_victim_pairs`) and evaluate Candidate B only if query complexity/performance
remains a proven issue after the parser redesign.

If `killer_victim_pairs` is enriched, retain at least:
- effective weapon id (`COALESCE(reconciled_as, weapon_id)` semantics)
- raw `weapon_id` (film value)
- `confidence`
- attribution source/path (`section2_fire_event`, `formula_a_snapshot`, `reconciled`)

This keeps analytics-friendly storage without losing parser-level traceability.

---

## POV confidence zones — per-weapon thresholds

`_get_confidence(weapon_id, delta_ms)` applies a 3-zone logic based on
`WEAPON_TIMING_BY_ID[weapon_id] = (swap_ms, travel_max_ms)`:

| Zone | Condition | `confidence` | Meaning |
|------|-----------|:------------:|---------|
| A | `delta_ms < swap_ms` | `high` | Weapon swap physically impossible in this window |
| B | `swap_ms ≤ delta_ms ≤ travel_max` | `medium` | Ambiguous window — weapon may have changed |
| C | `delta_ms > travel_max` | `low` | Delayed damage — fire event weapon is suspect |

**Per-weapon values (from `WEAPON_TIMING` / `_weapon_data.py`)**:

| Weapon class | `swap_ms` | `travel_max_ms` |
|--------------|----------:|----------------:|
| Sidekick | 400 | 300 |
| Plasma Pistol | 450 | 300 |
| AR, BR75, Bandit, Bulldog, Commando, Shock Rifle, Mangler… | 650 | 500 |
| Heatwave, Needler, Hydra | 650 | 2000 |
| Ravager, Disruptor, Cindershot | 650 | **5000** |
| Sniper, Stalker, Mutilator | 900 | 300 |
| Skewer | 900 | 3000 |
| SPNKr, Fuel Rod, Gravity Hammer, Energy Sword (all variants) | 1100 | 1400 |
| M41 SPNKr, Fuel Rod SPNKr | 1100 | 2000 |
| *Default (unknown weapon)* | 650 | 2000 |

**Consequence for the rewrite**: thresholds are per-weapon, so confidence can only
be computed after resolving the `weapon_id`. For `confidence=low` kills: the rule is
`delta_ms > travel_max_ms` for that weapon — the shot was probably correct but the
projectile traveled a very long time.

### Zone B — W2 check mechanism (`_check_zone_b_swap`)

When a kill falls in Zone B (`confidence=medium`), an additional check searches
whether the POV **swapped weapon immediately after** the kill:

```python
post_swap = [
    ev for ev in fire_events_all
    if kill_t < ev["timestamp_ms"] <= kill_t + swap_ms
    and ev["weapon_bytes"] != best["weapon_bytes"]
]
```

Logic: if within `[kill_t, kill_t + swap_ms]` a fire event with **a different weapon**
exists, the player swapped *after* the kill — proof that W1 (the candidate fire event's
weapon) was indeed the kill weapon. W2 (the post-swap weapon) is then retained as the
kill's weapon and confidence is raised to `high`.

> **Warning**: the logic retains weapon **W2** (the weapon after the swap), not W1.
> The implicit assumption is that the player swapped to W2 *because of* the kill
> (e.g. emptied their clip, immediately switched). This assumption is not always
> correct — if the player had already started swapping before the kill, W1 would be
> the right answer. To be re-evaluated in the rewrite.

---

## Optimizations identified for the rewrite

### 1. Fire event deduplication → one kill per event (claim-and-remove)

**Current bug**: `_match_kill_to_fire_event` does `max(candidates, by=timestamp)`
without marking fire events as used. Two consecutive kills within a 5s window can
both "claim" the same fire event — particularly for AOE weapons (Hammer, Cindershot)
that can produce two near-simultaneous kills.

**Solution using `highlight_events`**: kill timestamps are known to the ms via
`highlight_events.time_ms`. Sort kills by ascending timestamp, then for each kill:
1. Find the last **unclaimed** fire event in `[kill_t - 5000, kill_t]`
2. Mark it as claimed (remove from the pool) once attributed

This mechanism applies to **both POV and T1 teammates** via the same reference
timestamp — `highlight_events` is available for all players in a match.

For T1, the benefit is different: instead of "which chunk covers this kill" (~19s
granularity), we know the exact `kill_t` → the active snapshot is "the last state
recorded before `kill_t`", more precise than chunk granularity.

### 2. Cross-chunk — two-phase architecture

The key is strict separation between **scan phase** and **correlation phase**.

Scan phase accumulates all fire events across all chunks into a single flat list
sorted by absolute `timestamp_ms`:

```python
all_events: list[dict] = []
for _idx, (chunk_data, start_ms, dur_ms) in sorted(chunks.items()):
    packets = index_chunk(chunk_data)
    all_events.extend(
        scan_fire_events(chunk_data, player_index, start_ms, dur_ms, packets=packets)
    )
all_events.sort(key=lambda e: e["timestamp_ms"])
```

The correlation phase receives this global list and, for each kill at `kill_t`, filters:
```python
candidates = [ev for ev in fire_events_all
              if (kill_t - KILL_WINDOW_MS) <= ev["timestamp_ms"] <= kill_t]
```

A kill at `t = 19,050 ms` (start of chunk N+1) will find without issue a fire event
at `t = 18,800 ms` (end of chunk N), as both are in the same flat list.

> **Design rule**: maintain the two-phase architecture (global scan → global
> correlation). Do not process chunks one-by-one inside the correlation step.

### 3. Skip chunks without kills

Compute a `needed: set[int]` by filtering the film metadata chunks against
`kill_times_ms`:

```python
needed: set[int] = set()
for kill_t in kill_times_ms:
    window_start = kill_t - KILL_WINDOW_MS
    for ch in chunks_meta:
        if ch.chunk_type.value != 2:
            continue
        ch_start = ch.chunk_start_time_offset_milliseconds
        ch_end = ch_start + ch.duration_milliseconds
        if ch_end >= window_start and ch_start <= kill_t:
            needed.add(ch.index)
```

Only chunks whose time window `[ch_start, ch_end]` overlaps `[kill_t - 5000ms, kill_t]`
are downloaded. Chunks with no kill in their window are skipped — both download and
parsing.

> **Design rule**: `kill_times_ms` come from `highlight_events`, loaded before
> chunk downloads.

### 4. Packet-aware filtering inside kept chunks

After chunk-level filtering, packet indexing (`<HBBIQ`) should be used to reduce work
inside each kept chunk. Instead of scanning the whole byte payload blindly, the parser
can prioritize packet ranges that are known to carry actionable data and ignore
irrelevant packet sections.

Expected benefits:
- lower parsing cost on large chunks
- fewer false positives from random byte patterns
- cleaner and more reproducible event correlation when combined with packet timestamps

---

## Summary of current limitations

| Limitation | Impact |
|------------|--------|
| Section 2 (fire events) POV-only | Teammates have lower coverage; opponents are excluded |
| Film does not encode opponents | → 63% NULLs currently, justifies their exclusion from scope |
| ~279 distinct unknown hexes (T1 low confidence) | → 18% of "identified" kills have no name |
| Step 4b reclassifies unknown hexes as grenade/melee | → false mixing in grenade/weapon stats for the POV |
| `WEAPON_ID_MAP` at 36 confirmed weapons | → any hex outside the map remains unknown |
