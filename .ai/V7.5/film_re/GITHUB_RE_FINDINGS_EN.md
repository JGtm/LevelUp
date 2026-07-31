# Halo Infinite Theater film — verified findings + open questions

Sharing independent findings from a stats project that decodes Theater films offline
(read-only, downloaded from the CDN). Everything below is **verified on real films**
unless explicitly marked *(hypothesis)*. Several pieces build on dend/acurtis prior work
(noted inline) — thank you. Goal: contribute what we've nailed down, and ask the few
questions that would unblock the rest.

## Container structure (verified)

- Chunks are zlib (`0x78`). Each decompressed chunk is a stream of **16-byte packet headers**:
  `[Type u16 LE][b2 u8][b3 u8][Size u32 LE][Timestamp µs u64 LE]` + `Size` payload bytes,
  looped until `Type==7` (CHUNK_END). *(acurtis published this; we re-validated: a replication
  chunk parsed 739972/739972 bytes, exact end on CHUNK_END.)*
- Packet types seen: `0`=FRAME (~1199/chunk @ ~60 fps, ~16.7 ms apart, each carries an absolute
  µs timestamp), `1`=header/REPLICATION_DATA_START, `2`=TYPE_2 (game-state snapshot, ~20 s
  granularity), `3`=highlight events (footer), `8`=PLAYER_METADATA (~25 KB roster), `12`=BOT_METADATA.
- **chunk_00 (Type 1)** is the **ECS component registry** (~1.97 MB, ~264 named components) — the
  table of contents of all replicated state.

## player_index ↔ xuid (verified — acurtis method)

`player_index` = the **5 bits immediately before** the player's **64-bit little-endian** xuid,
found by a **bit-level** search (not byte-aligned) in a gameplay chunk. Verified on one match:
8 xuids → 8 distinct pi with no collision, and the pi we got independently cross-confirmed via a
fire-event/death-gap correlation (`pi=2 = xuid …0022`, p=0.006). This is per-match (a permutation),
**not** the DB participant order.

## Timing (verified)

Each fire event (and any byte offset) can be dated by the **µs timestamp of the FRAME packet that
contains it**, anchored to the chunk's `start_ms` at its first FRAME. The legacy frame-marker
bucketing drifts (median ~183 ms, p95 ~700 ms within a chunk); the FRAME-µs is exact. On clean
matches this lifted our fire→kill HIGH-confidence attribution from ~78% to ~93%.

## Score (verified, with a caveat) — anchor = 12-bit token `0x7B6`

The live score lives in the TYPE_2 game-state block (~bytes 810–905) but as a **bit-packed /
continuation-varint field at a per-match drifting offset**. The stable anchor is the **12-bit
token `0x7B6`** (MSB-first), found bit-level in the payload window ~[835, 912). The "7b60 / 07b6"
byte patterns various matches show are the *same* token at different bit phases.

- **Slayer** (score = kills): `byte813 = team0_score × 4` (26/26 keyframes exact), `byte823 = team1`
  (25/26). Offset *and* scaling drift per match (we saw 813×4/823, 846×1/836×1, bit6782/byte837×32)
  → scan a window, don't hardcode.
- **CTF captures**: a capture fires a **FRAME burst that re-transmits the full 6-tier objective
  ladder** (constant prefixes, only the final instance byte varies). Detector = a frame with
  `tiers==6`. **ms-exact, 0 miss / 0 false-positive over 4 matches** (counts matched the API exactly).
  Per-capture team comes from the coincident `th=10` footer event (needs the type-3 footer cached).
- **KOTH (Ranked rounds)**: `t2[token+12]` / `t2[token+16]`, score = meter/5. Validated exact (4-2).
- **Strongholds**: the leading team's accumulator decodes as a continuation varint at `token+24 bits`
  (× ~3.86), monotone, reconstructs to the leader's final. **Caveat below.**

### Verified negative — Strongholds two-team score is NOT in the keyframe

Cross-validated on **38 Strongholds + 11 KOTH** films: the TYPE_2 snapshot near the token stores
only the **leading/local team's** accumulator, not both teams symmetrically (unlike Slayer's
813/823). The adjacent candidate we first thought was team1 turned out to be a **structural
end-of-film marker** — identical values (50/32) across matches with completely different finals
(193-112 vs 78-178). A field *proportional to the winner's final* exists at several offsets
(cv ≈ 0.003 across 33 matches), but **no field proportional to the loser** appears anywhere in the
game-state region (we scanned token ±15..+90 bytes). So the per-team objective score-over-time is
**not** recoverable from the keyframe snapshot — it has to come from the named stat component (below).

**Update (two-team ground truth + static EXE confirmation).** We read both teams' live scores off the
in-game Theater HUD for one Strongholds film (winner 54/120/120/167/193, loser 17/19/87/107/112) and
re-ran an exhaustive joint search (whole TYPE_2 payload, the 343 KB REPLICATION_DATA_START packet,
per-frame and TYPE_10 packets, fixed-width + varint, monotone + 5-anchor fit): only **one structural
`0x7B6` token** exists per snapshot and only the recording player's team decodes — the opposing team is
**absent** from every decodable field. Separately, static RE of `HaloInfinite.exe` (unpacked; we located
the native getter behind the `Team_GetCurrentRoundStatValue` script binding and decompiled it) shows the
score is `displayed = raw_stat × scale` with a per-stat scale table — which **explains the `× ~3.86`**
above (the keyframe stores the *raw* stat, which caps low). Both teams' statlines exist in memory
(`world + statSlot*0x88 + teamIdx*0x1DF0 + 0x38 + round*4`, int32), so it's a relevance-based
replication issue: the recorder's keyframe only carries its own team's statline.

### The replication value encoding — decompiled (this is the part nobody published)

We decompiled the actual replication **deserializer** in `HaloInfinite.exe` (static analysis of the EXE
file). The integer encoding used by the per-frame/full-state replication is:

```
read_int(bitreader):                 // the engine's value reader
    sel = read 2 bits                // width selector
    w   = 8 << sel                   //  -> 8 / 16 / 32 / 64 bits  (1/2/4/8 bytes)
    v   = read w bits   (MSB-first)
    if w < 32 and top_bit(v): v = sign_extend(v)   // SIGNED
    return (int32) v
```

So every replicated integer is a **signed, 2-bit-length-prefixed variable-width int, MSB-first**. A 1-bit
field is just `read 1 bit` (MSB-first). The deserializer reads a **two-team stat record** in the order
`[5-bit header A][5-bit header B][value A][value B][1-bit flag A][1-bit flag B][conditional values]` — i.e.
**both teams are serialized consecutively** (so the opposing team's score is right after the local team's
in the bitstream, just with no separate anchor — which is exactly why our anchor-based scans only ever
found the local team). The model is **delta + dirty-bits** (a per-component changed-fields mask). The
on-disk **memory** layout the deserialized values land in: 48 statlines × `0x1DF0`, with `+0x18`=flags,
`+0x38`=per-round int32 values (stat stride `0x88`), then a per-stat float `scale` applied on read.

**Caveat / what's left:** two encodings coexist — the **keyframe (TYPE_2)** uses a *continuation* varint
(that's the local team we decoded at `token+24`, ×3.86), while the **per-frame deltas** use the 2-bit-selector
encoding above; the opposing team's absolute value flows through the FRAME deltas. To finish a pure-offline
decoder you need (1) a bit-exact reimplementation of the engine's BitReader state machine (8-byte refill +
byte-swap + bit cursor), and (2) the FRAME-delta **component framing** (where a given component's record
starts — dispatch by component handle/type-id). We have the value encoding and the two-team record shape;
the framing is the remaining piece.

## Weapon of kill (verified)

- Weapon-id is **64-bit**: the **high-32 is the identity**; the **low-32 is a near-universal tag
  `0x42c9679f`** *(acurtis noted the suffix is repetitive; confirmed — it's a reliable "this is a
  real weapon" signal)*. Canonicalize by high-32: ~174 raw ids fold to ~37 real weapons; ids whose
  low-32 ≠ `0x42c9679f` and whose high-32 isn't a known weapon are FormulaA noise.
- **Melee (acurtis §K-bis spec, validated)**: anchor byte `0x34/0x35` (3-bit prefix `101`),
  type byte at +76 ∈ {`0x42`, `0x47` hammer, `0x60` sword}, weapon-id at type-specific bit offsets,
  `pi = byte@+20 & 0x1f`. The weapon-id-at-offset validation is the noise filter (≈98% of raw hits
  are noise without it). Verified 56 weapon-validated swings on one match; Gravity Hammer (a pickup)
  appeared across multiple pi as expected.
- **Grenade (§C)**: byte-aligned marker `0x4c0c00`, weapon32 just after, allowlist {Frag `0xB0171062`,
  Plasma `0xC0E34C44`, Shock `0x3B2567D4`, Spike `0x9212E428`}. ~0 false positives among valid hits.
  No per-thrower pi in this record, though.

## Player positions (verified — §N / FilmShell)

- **Keyframe (TYPE_2) full-state records**: each player record is delimited by a **comb** = the bit
  pattern `(8 one-bits then 16 zero-bits) × 4` (96 bits); position = a **float32 little-endian** triplet at `combStart − 273 bits`.
  Validated on one match (76 full-state positions, bounds x[-6,35] y[-24,25] z[-2.8,2]); generalizes
  across maps (each map its own coordinate space). Delta records omit the absolute position.
- **Per-frame**: float32 **big-endian** at bit offsets, frame header anchor `0xA07B4200` + a TICK byte.
  Delta-compressed (only changed players re-emitted). Death→freeze→respawn-teleport proven with
  coordinates (body frozen bit-identical between the death burst and the respawn burst, ~66-unit jump).
- **Open**: dense per-player attributed tracks for all 8 simultaneously — blocked by the delta
  compression (the player index isn't at a fixed offset; continuity breaks at the death gap). We can
  isolate one player via an event anchor, not all eight generically.

## The wall: ECS component values need the off-film `.module` schema

Everything above worked because it had an **incidental anchor** (a token, a marker, a comb, the xuid
adjacency, the `42c9679f` tag) that lets us read a value *without* the schema. The remaining
high-value targets have no such anchor — their values live in the **handle-indexed ECS replication**,
and the binary layout is defined by the engine's `.module` tag files (off-film). chunk_00 names the
components but doesn't expose a usable handle→offset bridge. Specifically:

- **Two-team objective score**: `statborg-current-round-value-stat-component` (and `…finalized-rounds…`).
- **Zone control / identity** (which Stronghold, owner over time): `selectable-zone-data-component`,
  `managed-objective-sub-objective-entities-component`. Zone id is not co-located with any constant
  position and not at a fixed offset; the `b36` slot is the player slot, not a zone.
- **Aim vector** (`unit-desired-aiming-vector-component`), **vitality/ammo**, dense positions — same wall.

Note: **HavokScript bytecode (`\x1bLua`) was NOT present** in any of our cached chunks (type 1/2/3),
so we couldn't use the named-stat schema directly. *(It may live in a part we don't cache, or the
earlier extraction came from a different source.)*

### Registry (chunk type-1) layout — fully probed (verified)

We fetched and decompressed the **type-1 HEADER chunk** (`/filmChunk0`, ~1.97 MB; note the public
stats tooling usually keeps only ChunkType==2 and **discards** this one). Findings:

- It is a **pure name table**: **264 unique component names**, ~991 slots, **stride exactly 260 bytes** =
  `[kind u32 LE][ASCII name null-padded to 256]`. After each name: **pure zeros** to the next slot.
- `kind` ∈ {0,1,2,3,4} (1:×754, 0:×128, 2:×83, 3:×19, 4:×6) — a small struct-category enum, **not** a GUID.
  (Shape echoes the CachedTag struct-type enum 0=Main/1=TagBlock/2=Resource/3=Custom, but it sits at slot
  offset 0x00, whereas the CachedTag `type` is at 0x10 after a 16-byte GUID — so it's not a Tag Struct Def.)
- **No Type GUID, no handle, no string_id, no field layout** anywhere in/adjacent to the slots. The whole
  registry is 75% zeros / 10% ASCII / 14% sparse binary, with exactly **one** dense ≥64-byte binary run in
  the whole 1.97 MB (a low-entropy `a0/30/70` bitmask — not a GUID table). Header = two u32: `41`, then `27`
  (CachedTag version is also 27 — possibly coincidence). `statborg-current-round-value-stat-component`
  appears as **28 consecutive identical slots** (= 28 stat instances), with no stat-id/value inline.

### Verified negative — replication does NOT reference components by `Murmur3_32(name)`

We tested whether the type-2/FRAME replication keys components by `string_id = Murmur3_32(name)` (the
Slipspace string-hash; our impl is bit-exact: `__default__` → `0x9B555AD2`). We searched all 264 component
hashes — **byte-aligned (LE+BE) AND bit-level (MSB-first + LSB-first)** — across the registry and every
type-2 chunk, on two matches including a **confirmed Strongholds film (zone active, 71 capture events)**:
**0 byte-aligned hits; bit-level hits below the random-chance floor** (3–7 vs ~20 expected, scattered, never a
target). So the replication does **not** reference components by name-hash. The most likely mechanism is a
**registry-ordinal index** (small int 0..263, bit-packed) — which is not a searchable signature, leaving the
value layout schema-bound (off-film). For reference, the target hashes are
`statborg-current-round-value-stat-component = 0xF709F8BC`, `selectable-zone-data-component = 0x3080F77C`.

## Hypotheses / leads (unverified)

- **BTB fire-event pi**: the 4-bit pi field (`b5 >> 4`) appears to **overflow one bit upward** (the
  bit before b5) on >16-player lobbies — reading 5 bits as `[bit before b5][b5>>4]` recovered killers
  with pi>15 on one BTB film (NULL 65%→52%, agreement unchanged). Needs more validation.
- The per-team objective score is almost certainly the `statborg-current-round-value-stat-component`
  value; if its layout (list of (stat-id, value)?) and the stat-id enum were known, it'd be direct.

## Questions for the experts

1. **Replication format**: is the TYPE_2 snapshot / FRAME delta serialization **Microsoft Bond**, or a
   custom format? Is there a film-internal handle→component table + per-component layout we can
   reconstruct, or are the `.module` tag files required?
2. **`.module` / runtime-tagviewer**: are the component layouts (e.g. `statborg-current-round-value-stat-component`,
   `selectable-zone-data-component`) available anywhere?
3. **chunk_00 registry**: how are component handles/IDs assigned and referenced from the replication?
   We ruled out `Murmur3_32(name)` (byte + bit-packed, see above) — is it the **registry ordinal index**
   (slot 0..263), and is that index what the per-entity replication packets carry?
4. **statborg**: is a stat component a list of (stat-id, value) pairs, and is the stat-id enum
   (FlagCaptures, PersonalScore, objective score…) stable/known?
5. **HavokScript**: where does the bytecode live in the film (not in our cached type-1/2/3 chunks)?
