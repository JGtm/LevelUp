# Halo Infinite Theater films — offline stats extraction: external developer handoff

> Self-contained English brief for a developer with **zero prior context**. It covers EVERYTHING that has been
> reverse-engineered from Halo Infinite Theater film files: the kill-feed, medals, objective events, live
> score-over-time per mode, per-tick biped state, and the deepest sub-problem, weapon-per-kill. Compiled
> 2026-06-23 from the project's RE docs and the Go source, then adversarially fact-checked + re-measured by a
> multi-agent verification pass (every load-bearing fact cites a source file/line or RE function address;
> contradictions and unverified figures are listed in §"Known uncertainties").
>
> **No code changes since 2026-06-14** (branch `feat/weapon-attribution-v3`, HEAD `62521aa39`). All metrics
> below are as of that date; this doc was written 2026-06-23 from that frozen state.

---

## 0. Mission and scope

**Context.** Halo's stats API gives only end-of-match totals. The **Theater film** (the replay file) is a binary
ECS replication stream that contains far more: who killed whom and when, with what, plus the full game state at
~60 fps. This effort decodes the film **100% offline** (from the film bytes alone — no Cheat Engine, no live
game, no runtime instrumentation at decode time) to recover production-grade stats.

**The data veins, by reliability tier:**

- **Tier 1 (reliable, shippable):** kill-feed (killer · victim · time), medals, **objective events** (CTF /
  Strongholds / KOTH / Oddball: actor · team · time), per-weapon **damage timeline** (family · time, aggregate),
  **melee** and **grenade** events (actor · weapon/type · time).
- **Tier 2 (partial / indicative):** **live score-over-time per mode** (CTF 2-team exact; KOTH variant B exact;
  Slayer exact; Strongholds reconstructed-but-contested), full keyframe **loadout**, **fire** events.
- **Tier 3 (identified, blocked):** **per-tick biped state** (health, shield, velocity, crouch, position, aim,
  ammo) — grammars reverse-engineered, extraction blocked.
- **The flagship sub-problem:** **weapon-per-kill** = the *damage source of the fatal blow* per kill.

**One wall links most of what is blocked** (see §5): the **biped-entity record → player-slot binding**, resolved
in RAM at replay and never serialized per-record in the film. Everything that carries the actor inline (kill-feed,
medals, melee, grenade, objective events) escapes it and ships; everything carried by the biped/weapon entity
(weapon-per-kill, loadout, per-tick state) is blocked by it.

**Hard constraints (set by the project owner):** (1) 100% offline; (2) exact and universal (all maps/modes), a
mode-specific or debugger-only method is not a solution; (3) for weapon-per-kill specifically the semantic is the
**damage source**, NOT the equipped/"held" weapon (banned ~10×), and fire/melee/grenade *events* are banned as the
firearm source. See §15 (dead-ends).

**Tech.** Go monorepo, module `levelup/go-api`. Decoder library: `apps/go-api/internal/analysis/filmdec/`.
Probes/validators: `apps/go-api/cmd/{tmp_*,diag_*}`. Static RE in Ghidra against `HaloInfinite.exe` (PE, image
base `0x140000000`, build `6.10026.19225.0`). No Python anywhere in this subject.

---

## 1. TL;DR

- **Ships today** (offline, validated): kill-feed + medals are **auto-wired in sync**; objective events (incl.
  the CTF 2-team score curve) have a **live read path but a still-manual producer** (run via `diag_weapons_v3
  -write`, no automatic backfill — see §7). All carry the actor inline / via the chunk_27 roster, so they sidestep
  the wall.
- **Built but parked** (validated recipe, not wired to a live producer): damage timeline, melee events, grenade
  events, Slayer/KOTH-B score timelines, the 2-team statborg wire parser.
- **Blocked on the binding wall:** weapon-per-kill, keyframe loadout (who holds what), per-tick biped state.
- **Weapon-per-kill specifically has two attack paths.** Path A (`0xd2` damage record + time-warp) works at
  **94-96% on Fiesta** but only **58% on Team Slayer** — capped by a cross-clock drift that is unfixable offline.
  Path B (stateful ECS frame decoder + same-clock dead-state) is the only universal route, **blocked** on a
  World-binding gap (deaths reached **0/11** against the CE oracle). The single biggest lever for Path B: the
  runtime quantization widths thought to need Cheat Engine are in fact **derivable from static `.exe` constants**
  (formula in §11.3).

---

## 2. Capability status table (the master overview)

| Capability | Source vein | Status | Blocker / note |
|---|---|---|---|
| Kill-feed (killer · victim · time) | chunk_27 type-9 footer (`ParseHighlightEvents`) | **SHIPPED** | Persisted to shared `highlight_events`/`killer_victim_pairs` in sync. 93/93 on `000d5950`, 94/94 on a 2nd match (`jgtm`), ms-precise. No method/weapon in the event. |
| Medals (player · medal · time) | chunk_27, `is_medal`/`medal_type` | **SHIPPED** | Production (`ParseHighlightEvents`); feeds `medals_earned`. `medal_type→name` via API/DB. |
| Objective events (CTF/SH/KOTH/Oddball: actor · team · time) | th=10 footer + CTF capture bursts (`tiers==6`) | **SHIPPED read path; manual producer** | `internal/analysis/objectiveevents` → repo (DELETE-then-INSERT/match, capability-gated, ART-safe) → `GET …/objective-events` → front overlay. But the WRITE path runs only via `cmd/diag_weapons_v3 -write` (no live backfill). Action sub-type / zone identity NOT in the event (`objective_id` always NULL). |
| Score-timeline — CTF (2-team) | CTF bursts + th=10/roster | **SHIPPED** | Exact, ms-precise, multi-match validated; wired into dominance (`comeback_objective.go`). |
| Score-timeline — KOTH-B / Slayer | TYPE_2 byte-aligned | **BUILT-PARKED** | KOTH variant B exact; Slayer exact (`payload[832]>>2`/`payload[842]`). Slayer not in `objectivescore` (probe-validated). |
| Score-timeline — Strongholds | TYPE_2 varint `token+24`/`+23` | **BUILT-PARKED / CONTESTED** | Reconstructed finely on ref match `7344d24f` only; a 38-match cross-validation INVALIDATED the per-team curve (`+23`=structural marker, `+24` caps ~50). Marked "do not display as score". |
| 2-team statborg wire parser | `filmdec.ParseStatborgRecord` (`FUN_140c18794`) | **BUILT-PARKED** | Validated parser; NOT wired to film framing (record dispatch open). |
| Damage timeline (family · time · cause) | type-0 `payload[0]==0xd2` | **BUILT-PARKED** | 519 records on `000d5950`, 17 families exact. No killer/victim → aggregate intensity only. |
| Melee events (actor · weapon · hit/miss · time) | type-0 `0x534`/`0x535` | **BUILT-PARKED** | Actor reliable (inline `player_index`). Shadow pkg `analysis/weaponv3`; no endpoint. |
| Grenade events (thrower · type · time) | type-0 `0x4c0c00` | **BUILT-PARKED** | 70 events, thrower reliable. Shadow pkg; no endpoint. |
| Keyframe loadout (per biped) | type-2 keyframes, WST | **BLOCKED** | Decodes 8 loadouts but record→player unresolved (the wall). |
| Per-tick biped state | type-0 deltas + type-2 keyframes | **BLOCKED** | Delta walk not bit-exact everywhere; consumers only advance (only position emits); record→player unresolved. |
| Weapon-per-kill (firearm of the kill) | Path A warp / Path B dead-state | **PARTIAL (A) / BLOCKED (B)** | A: 94-96% Fiesta, 58% Team Slayer (clock wall). B: deaths 0/11 (binding gap). |

---

## 3. Prerequisites & getting started (READ — most of this is NOT in git)

A cold-start dev is blocked without the following. None of the film data or CE artifacts are committed.

### 3.1 The code
- All of this lives on branch **`feat/weapon-attribution-v3`** (a git worktree), **NOT on `main`**. `git fetch`
  then check out that branch (or enter the worktree) before anything.
- The Path A prototype `apps/go-api/cmd/tmp_offwarp/main.go` is ~883 lines; the cited line offsets are valid at
  branch HEAD `62521aa39`.

### 3.2 Film data (gitignored — you must obtain it)
- Films are cached at the **repo root**: `data/cache/film_chunks/<shortID>/chunk_00.bin … chunk_27.bin` (some
  films go further). These are **gitignored**; ~942 are present on the original dev's machine but none are in the
  repo. The tools hard-code this repo-root path.
- To fetch a film you need its **manifest**: `data/cache/film_manifests/<shortID>.json` (blob prefix + chunk list),
  also gitignored. `cmd/fetch_film_chunks` reads the manifest and downloads the missing `REPLICATION_DATA` chunks
  from pre-signed Azure CDN URLs (no auth), zlib-decompressing them. Without a manifest a film cannot be fetched.
- The two matches with a full CE oracle are `000d5950` (Fiesta, the primary oracle) and `9b191a7f` (Team Slayer);
  a third match `jgtm` was used to cross-check the kill-feed only (94/94). Almost every number in this doc is from
  `000d5950`; re-validate on ≥1 other match before any production claim.

### 3.3 Cheat-Engine ground-truth artifacts (the validation oracles)
These prove the answer and where in the bitstream it lives; they are NOT part of the offline decode path. They
are produced by CE `.lua` hooks on a live replay and **cannot be regenerated offline** — for a new match you must
run the CE capture (game running + CE attached via the `cheatengine` MCP).
**Tracking:** the `tools/ce/*.bin` captures and `*.lua` scripts ARE committed in git (only `tools/ce/*.csv` is
gitignored). By contrast the film chunks/manifests AND the `world_dump*.txt` / `calib.txt` seeds live under
`data/cache/` and ARE gitignored (present only for the validated matches).
- `tools/ce/000d5950_deadstate.bin` (tracked) — resolved dead-state oracle (consumed by `tmp_dscap`, `tmp_cematch`).
- `tools/ce/dmgcapture_run2.bin` + `killcapture.bin` (tracked) — dual-hook damage+kill ground truth, `000d5950`,
  97/98 (consumed by `tmp_dualcap`; also by `tmp_offwarp`'s validation block).
- `tools/ce/9b191a7f_dmg.bin` + `9b191a7f_kill.bin` (tracked) — second (Team Slayer) capture.
- **World seed for Path B** (under `data/cache/film_chunks/000d5950/`, gitignored): `world_dump.txt` (250
  entities, the default) and `world_dump_full.txt` (actual body ~2228 slots; its header line claims "1979 slots"
  but is stale). Both are CE-captured slot→archetype snapshots, present only for `000d5950`, selected via `WORLD_DUMP`.
- `calib.txt` (beside the chunks, gitignored): per-film, CE/empirically-derived `component-name:modal_width
  prevalence`, feeding the **research-only** `calibratedWidth` path in `traverse.go` (NOT a production decode
  path; the §11.3 static-width formula is what should ultimately replace it).
- The CE capture scripts are `tools/ce/filmdec_*.lua` (tracked; see §13.1/§13.2 for the exact hooks/RVAs/record strides),
  driven through the `cheatengine` MCP attached to `HaloInfinite.exe`.

### 3.4 RE environment
- All `FUN_xxxxxxxx` / RVA citations target `HaloInfinite.exe` build **`6.10026.19225.0`**, image base
  `0x140000000`, reversed in Ghidra. **RVAs are build-specific** — on a different game build, re-resolve them by
  string/xref, do not trust the literal addresses. The exe is not in the repo (you supply your own copy).

### 3.5 Build / run
- **Filmdec / offline tools** (`tmp_offwarp`, `tmp_cleanframe`, `tmp_cematch`, …): `CGO_ENABLED=0 go run ./cmd/…`
  from `apps/go-api/`. No native dependency; cross-platform.
- **DuckDB tools** (`merge_weapon_kills`, `diag_weapons_v3`, `diag_film`): need CGO + a C toolchain. On the
  original Windows machine: `export PATH=/c/msys64/ucrt64/bin:$PATH` then `CGO_ENABLED=1 go run …`. CGO-only tools
  carry a `//go:build cgo` tag.
- `gofmt -w` on changed files (pre-commit enforced). `go test -race` is incompatible with the DuckDB driver
  (`checkptr` misalignment); CI adds `-gcflags=all=-d=checkptr=0` (irrelevant for the `CGO_ENABLED=0` filmdec tools).
- The offline kill-feed/catalogue helpers are self-contained: `analysis.ParseHighlightEvents` and
  `analysis.WeaponIDToName` are in `apps/go-api/internal/analysis/` (`WeaponIDToName` declared+populated in
  `weapon_data.go`); they need no DuckDB/metadata at runtime.

---

## 4. Film file format and on-the-wire record grammar

Reverse-engineered statically from `HaloInfinite.exe` and re-implemented in `apps/go-api/internal/analysis/filmdec/`.

### 4.1 Container: chunks, zlib, packets

A film is a sequence of **chunks**, cached one file per chunk under `data/cache/film_chunks/<matchID>/`. Each is
**zlib-compressed** (magic `0x78`); inflating yields the payload (`ParseRegistryChunk`, `registry.go:68-82`).

| Chunk | Role |
|-------|------|
| `chunk_00` | ECS archetype **registry** (inflates ~1.97 MB; §4.3) |
| `chunk_01` | **keyframe** (full snapshot; 8 NEW bipeds/loadouts) |
| `chunk_02`–`chunk_25` | **gameplay frames** (per-tick replication, §4.4) |
| `chunk_26` | **re-sync** |
| `chunk_27` | **kill-feed / highlight events** (the type-9 footer; see note below) |

> **Chunk-index vs packet-type note (reconciliation):** the kill-feed is the single ~714 KB **type-9** footer
> packet. On `000d5950` it lands in `chunk_27`, but **the chunk index varies per film** — which is exactly why the
> production code sweeps chunks ~18..41 and keeps the one yielding the most kill events, rather than hard-coding 27.
> ("type-3" appears in older docs for the same footer stream; treat type-9 as current.)

Inside an inflated chunk, data is framed as **packets** with a 16-byte header then payload:
```
+0x00 u16 LE Type    +0x02 b2   +0x03 b3   +0x04 u32 LE Size   +0x08 u64 LE ts   (+0x10 payload)
```
**Critical: `ts` at `+0x08` is a buffer/write clock, NOT in-game time.** Correlating it to the kill-feed
(game-time) clock is the "warp" that caps weapon attribution at 58% on Team Slayer (§10). Match-clock convention
used by the probes: `t0Us = 4537898226`, `tms = (ts - t0Us)/1000`.

Packet types seen (`000d5950`): 0 (FRAME, ×30418), 1 (registry), 2 (keyframe), 6, 8 (roster), 9 (the 714 KB
footer ×1), 10 (×30418, spectator camera — not a damage carrier), 12, 41.

### 4.2 Bit ordering

All fields read from an **MSB-first big-endian bitstream** (`filmdec.BitReader`, `bitreader.go`): bit `i` = bit
`7-(i mod 8)` of byte `i/8`; past-end reads as zero. A **signed var-width codec** `ReadSignedVarWidth` (mirrors
`FUN_140c18a1c`): 2-bit selector `sel` → width `w = 8<<sel` (8/16/32/64), sign-extended only for w∈{8,16}.

### 4.3 ECS archetype registry (chunk_00)

An array of fixed-size **archetype blocks**; block `#N` holds the ORDERED component list of archetype `#N` (the
order the engine iterator walks, the bit index the presence-mask gates). Layout (`registry.go:21-25, 84-122`):
slot = 260 bytes `[u32 kind][u32 flags][name ASCII@+8]`; block = 64 slots = `0x4100` bytes; the list is the leading
run of non-empty names. **118 archetypes; biped/player = `#35`** (block `@0x08e300`); on it,
`object-position-dynamic-precision` = i0, `weapon-state-type-info` ×4 (held weapon) = i43..i46, **dead-state = i11**.

### 4.4 FRAME record grammar (type-0 packet)

`DecodeFrameRecords` (`frame_records.go`, ports `FUN_1406cd128`/`FUN_141f86b58`). Loop reads records until a
type-0 (end) record. Header traps:

1. Optional `R(32)` prefix iff `hasExtraFields` (`FUN_14076cea8`) — assumed false offline.
2. **Type is a prefix code:** `R(1); if 1→DELTA; else R(2)∈{0=end,1=new,2=del,3=delta}`.
3. **ID is table-driven:** `low=R(idLowBits)+idBase` (idLowBits default 13=bitLen(0x1FFF)); `tag=R(2)` (bits 30-31);
   `id=(tag<<30)|(low&0x3fffffff)`; `slot=id&0x3fffffff`.

Per-type body:
- **NEW (1):** `R(6)` typeIndex + default-state (in stream) + `R(1)` gate + mask + components; binds the slot.
- **DEL (2):** `R(32)` then unbind.
- **DELTA (3):** **no typeIndex/default-state/gate** — mask + present components only; archetype from the World.
  **If the slot is unbound, decode cannot proceed (length unknown) → desync** (the binding gap, §13.5).

Mask (`consumeMask`, `FUN_1406d7610`): `R(1)`; if 0 → `R(3)` count + count×`R(6)` index (sparse); if 1 → `R(64)`
dense. Component loop (`traverseComponentLoop`): mask bit set → decode via `consumeByName`, clear → 0 bits;
un-ported present component → stop cleanly (`DesyncAt=i`), naming the next component to port.

### 4.5 World state (stateful), and the multi-sub-frame discovery

The decoder is **stateful**: `World` (`world.go`) maps `slot → {TypeIndex, FullID, HeldWeapon}`. A DELTA needs the
archetype from a prior NEW/keyframe; a delta on a never-bound slot is undecodable. **Discovery 2026-06-14:** a
single type-0 packet holds **multiple concatenated sub-frames** (ticks). A death's dead-state was ~bit 6761 of an
8112-bit payload — ~6 sub-frames deep. So reaching it needs every preceding sub-frame clean; at ~63% clean,
`0.63^6 ≈ 6%`, explaining deaths 0/11. Found via `cmd/tmp_deathchain` (loops sub-frames + the World).

---

## 5. The ONE shared wall: biped-entity record → player slot

There is exactly one unsolved problem at the root of most of what is blocked:

> **Given a decoded biped-entity record (keyframe or delta), we cannot map it to its player slot / `player_index`.**
> The record carries a runtime **world entity handle** (resolved in RAM), not a `player_index`; the `obje` LocalID
> is absent at the keyframe; emission order ≠ slot order (tested vs melee ground truth, record→slot purity ~29-50%).

The negative roster exploration (`cmd/tmp_roster`, 2026-06-07) confirmed no packet escapes it: **type-8**
(~27×25 KB) is a stats/scoreboard structure (only 2/8 xuids byte-aligned); **type-9** (714 KB footer) carries all 8
xuids (order `[3,0,2,4,5,7,6,1]`) but repeated + 0 weapon family = a profile/social metadata blob, no entity
binding; **unit-actor-control (i19)** carries RAM handles, not a clean 0-7 index.

**Blocks:** weapon-per-kill, fire events (Tier 2.5), keyframe loadout (Tier 2.6), per-tick biped state (Tier 3).
**Escapes** (actor inline or via the solved roster): kill-feed, medals, mode/objective events, melee, grenade.

**The roster that IS solved:** each chunk_27 event carries `gamertag` + xuid inline; the pair `(duo=b36, team=b37)`
gives 8 distinct combinations = a per-match bijection slot→player. (Team itself is re-derived from
`match_participants`; the film team bit is unreliable.) This roster is why the inline-actor pipelines attribute
per-player without a RAM dump.

---

## 6. Tier-1 reliable veins: damage timeline, melee, grenade, medals

All pure Go, offline, scalable. Validated on `000d5950`; re-validate on ≥1 other match before prod. Shared
conventions per §4.

### 6.1 Per-weapon damage timeline (family + time, NO actor)
Type-0 packets with `payload[0]==0xd2`. On `000d5950`: 30418 type-0 packets, exactly **519 begin with `0xd2`** =
519 strict damage records (0 outside). Deser mirrors `FUN_14080c1f8` (dispatch event-type 11, vtable `143d0ac08`).
Struct field layout: skip the 36-bit header; **`+0x0c` = R(5) ATTACKER index** (player = R5>>1); **`+0x14` =
`variant_name` R(32 BE) = high-32 weapon FAMILY** (read by `FUN_14080dec4`), with the contiguous low-32
`0x42c9679f` reconstituting the id64; `cause/slot` at `+0x08` (`consumeId2`). 17 families exact, 0 aberrant
(Needler 76, MA40 67, Disruptor 59, …). **No killer/victim** in the record (one record = one damage tick, often
non-lethal) → **aggregate match-wide weapon intensity only.** Probes `cmd/tmp_dmgscan`, `tmp_dmgdecode`.
(Historical note: a 2026-06-07 hypothesis fused the R5 attacker and the R32 family into a single "+0x0c
global-id" treated as the weapon — that was a BUG, corrected 2026-06-12. The family is `variant_name` at
**+0x14**; the R5 attacker is at +0x0c; and `tmp_offwarp` reads them as **separate** fields, not a fused
global-id — see the bit-stream layout in §9.1, which is the authoritative working decode.)

### 6.2 Melee events (actor inline → attributable)
Type-0, 11-bit marker `0x534` (HIT) / `0x535` (MISS); `anchor = bp+3`. type @anchor+76 (`0x47` Gravity/Rushdown
Hammer, `0x60` Energy Sword, `0x42` other); weapon high-32 @anchor+86 (type-dependent: `0x42`→+88, `0x60`→+101/+103);
**`player_index` (5 bits) @anchor+23** — the actor, covering all 8 players. **Mandatory filter:** the 11-bit marker
is noisy (~36k raw hits), so require `type∈{0x47,0x60,0x42}` AND weapon∈catalogue. The event is the swing, not the
kill (join to kill-feed by `player_index==killer` + small |dt|). Validated: IKE(pi4) hammer→JGtm 4/4.

### 6.3 Grenade events (thrower inline)
Type-0, 24-bit marker `0x4c0c00` → grenade_id 32b → skip 47b → **`player_index` 5b** (thrower). Ids: Frag
`0xB0171062`, Plasma `0xC0E34C44`, Shock `0x3B2567D4`, Spike `0x9212E428`. **70 events, all `player_index` 0-7
valid.** Probe `cmd/tmp_meleegrenade`. The most solid action-event recipe.

### 6.4 Medals & mode events (chunk_27, same stream as kill-feed)
Decoded by the production `analysis.ParseHighlightEvents` (`highlight_event_parser.go`). 60-byte block:
`Gamertag[0:32]` (UTF-16LE; `[12:44]` for film versions 39-40), `TypeHint[47]`, `TimeMS[48:52]` (uint32 BE),
`IsMedal[55]`, `MedalType[59]`. Type hints: kill=50, death=20, mode=10. Medal when `IsMedal` set and
`TypeHint∈medalSortingWeights`. Yields medal-per-player+time. **`MedalType` on KILL events == 0** — the method is
NOT in the kill event; medals are separate events (partial weapon proxy, e.g. Sniper medal ⇒ S7 Sniper).

### 6.5 Derived deliverable: the "enriched kill-feed"
Because melee/grenade events carry the actor inline, join them to the kill-feed without solving the wall:
> kill-feed killer (slot, T) ⋈ melee/grenade event of the same `player_index`, within a short |dt| → "killer melees
> victim (weapon W)" / "killer grenades victim (type G)".
Reliable for melee/grenade kills; does NOT extend to firearm kills (no clean per-shot actor; §15).

---

## 7. Objective events pipeline (CTF / Strongholds / KOTH / Oddball) — shipped read path

The one decoder beyond the kill-feed carried to a wired API + React match view: timestamped objective
interactions (`actor xuid · team · time`). Extraction is mode-driven (`classifyObjectiveMode` on
`match_registry.game_variant_name` → `flag`/`zone`/`hill`/`skull`; Slayer → no-op).

**Two veins:** (a) **`th=10` footer events** (chunk_27, same stream as kill-feed; per-event block where byte
`ebs+47*8==10`; `b59==2` is the documented "mode event" marker; `confidence=approx` ~5-20 s); (b) **CTF capture
bursts** (TYPE_2 FRAME re-transmitting the 6-tier objective ladder — `ladderTiers` constant tokens
`a4 00 00 00`…`34 80 00 00 15`; qualifies only when all 6 co-occur, `captureMinTiers=6`; `confidence=exact`,
ms-precise). For CTF, the capturing team = the th=10 event of **max t** within ±2000 ms of the burst, mapped to a
team **via the roster** (the film team bit is unreliable).

**Validated:** CTF `53ce4390` (34 th=10 events, 3 captures = DB final 1-2); Strongholds `7344d24f` (71 events);
CTF burst counts == API capture totals exactly (`53ce4390=3`, `0f9550e5=5`; the committed test exercises those
two). **Hard limit:** the action sub-type is NOT in the event (grab/return/capture indistinguishable; zone A/B/C
absent) → `objective_id` always NULL.

**Production status:** the **read path is fully wired and live** — repo (tables `match_objective_events` +
`match_objective_event_players`, DELETE-then-INSERT/match, capability-gated → 503 if absent, ART-safe), service,
handler `GET …/objective-events` (`server.go`), front overlay `_objectiveCaptures.ts`, CTF dominance
(`comeback_objective.go`). **But the WRITE path is NOT automatic:** the only caller of `objectiveevents.Extract`
is the manual CLI `cmd/diag_weapons_v3` (dry-run by default, `-write` to persist). Live sync only calls
`ObjectiveTypeOf` (classification). So: **shipped read path + manually-driven producer.**

---

## 8. Live score over time, per game mode

The API already gives the **final** score + per-player personal_score (in `match_participants` /
`personal_score_awards`) but **without timestamps**. The film's unique contribution is the **curve** (when each
team's score moved). Granularity = one point per TYPE_2 chunk (~18-20 s). Objective modes anchor on a 12-bit token
`0x7B6` (MSB-first) in the byte window `[835,912)` of the TYPE_2 payload (`objectivescore/decode.go`:
`scoreToken=0x7B6`, `tokenWinLo=835`, `tokenWinHi=912`).

- **CTF — 2-team EXACT, generalised.** Does NOT use the `0x7B6` block; uses the objective-events pipeline (capture
  bursts + th=10/roster). The only objective curve validated multi-match; wired into dominance.
- **KOTH — variant B EXACT, variant A approx.** `objectivescore/koth.go`, byte-aligned on `ab=tokenBit/8`. Variant
  B (meters/Ranked, low finals): `team0=byte(ab+12)/5`, `team1=byte(ab+16)/5` (validated 4-2 on `0a247154`).
  Variant A (points): approximate, off-by-one possible. Auto-selected by final > 20.
- **Slayer — 2-team EXACT, byte-aligned.** `score_team0=payload[832]>>2` (26/26), `score_team1=payload[842]`
  (25/26). Not handled by `objectivescore` (probe-validated fact). In Slayer score=kills (derivable from deaths).
- **Strongholds — reconstructed on the ref match, cross-match generalisation INVALIDATED.** `strongholds.go` reads
  per-chunk varints `team0=varAtBit(token+24)`, `team1=varAtBit(token+23)`, rescaled per match by `calibrateByFinal`
  to the DB final. On `7344d24f` (193-112) the local team reconstructs ±1 on 4 anchors. **But** the header of
  `objectivescore/decode.go` documents a 38-SH + 11-KOTH cross-validation that found `+23` decodes structural
  end-markers (50/32 identical across very different finals) and `+24` caps ~50 and tracks the recorder team — it
  is marked "do not display as score", PARKED. (The recap's `×3.86` is the engine display scale, explained by the
  RE, not a code constant; the package uses `calibrateByFinal`.)
- **Oddball — excluded** (single global accumulator, not per-team).

**Native getter (RE confirmation, `RE_EXE_GHIDRA_FINDINGS.md`):** the HavokScript binding
`Team_GetCurrentRoundStatValue` @`0x142C6B118` → `FUN_142b7974c` → `FUN_1406ada4c`, and it is `FUN_1406ada4c`
that computes `value_raw = *(int32*)(world + statSlot*0x88 + teamIdx*0x1DF0 + 0x38 + round*4)`,
`displayed = raw*scale`. **Both
team statlines exist in RAM** (`teamIdx*0x1DF0`) → a live CE read of both scores is feasible; the Theater replay
rehydrates them. Registry component: `statborg-current-round-value-stat-component`.

**The validated 2-team wire parser:** `filmdec.ParseStatborgRecord` (`FUN_140c18794`, `statborg.go`) ports the
consecutive-teams grammar `header5(A),header5(B),valA,valB,flags,[val2…]` bit-exact (values via
`ReadSignedVarWidth`). **This is the route to all teams on all matches** once the per-frame framing (locating the
statborg record by component type-id) is solved — **not done**. NB: it is guarded by a single unit test vector
(`TestParseStatborgRecord_V8`), not "8 vectors" as an older recap claimed.

**Production status:** `objectivescore` (`DecodeScoreTimeline` in `dispatch.go`) + table
`match_objective_score_timeline` exist but are **PARKED** (called only by `diag_weapons_v3`/test; table
unpopulated/unserved). Only the **CTF curve is actually wired** (via objective-events + dominance).

---

## 9. Weapon-per-kill, Path A (working): the `0xd2` + time-warp pipeline

The pipeline that works fully offline today and is the productionisation candidate. Reference:
`apps/go-api/cmd/tmp_offwarp/main.go` (~883 lines, every offset below confirmed there).

### 9.1 The damage record (`payload[0]==0xd2`)
Deser `FUN_14080c1f8` (RVA `0x80C1F8`). Bit layout as decoded (`tmp_offwarp/main.go:103-112`):
```
bit  0..35 : 36-bit header (skipped)
bit 36..40 : R5 ATTACKER     -> attacker slot = R5 >> 1   (0..7)
bit 41..   : slot/cause field, VARIABLE 1 or 3 bits (read 1 bit; if 1 advance 1 else advance 3)
next 32    : variant_name BE  -> WEAPON FAMILY (high-32 key into analysis.WeaponIDToName)
next 32    : suffix BE        -> must equal 0x42c9679f (validity gate)
```
Why the attacker is reliable: **R5 is read at bit 36, BEFORE the variable slot field** (the historical bug read
the slot first, desyncing R5 to noise `{1,3,17,19}`; reading R5 first gives clean even `{0,2,…,14}` = `slot*2`).
The 8 players occupy slots 0..7 (handles `0xEC500000+idx*0x10002`). The low-32 variant is generally not
recoverable offline — the **family** is what's attributable. `0xd2` is the **only** damage record with a clean
attacker; the sibling markers (`0xe9`/`0x89`/…) have no clean 5-bit player field (dead end).

### 9.2 Roster slot→xuid (type-8)
From the longest `typ==8` payload: bit-scan **little-endian** (`u64LE`, lines 44-57 — a naive bit order returns 0
matches) for each kill-feed xuid; sort by first bit-position; that rank = slot order. Validated 8/8 on `000d5950`.
Kill-feed via `analysis.ParseHighlightEvents` over the footer; deaths paired to kills ±400 ms for the victim slot.
(Identity `slot==idx` is only proven on `000d5950`; `tmp_offwarp` also has a brute-force 8! Hungarian remap, so
identity is not guaranteed per match — player-index reconciliation is an open implementation decision.)

### 9.3 Linear time-warp `TimeMS = a*ts + b`
Init from extents, then ICP refine (3 nearest + 3 last-before iters, least-squares, window `|Δ|<8000ms`, slope
accepted only if positive). Measured slope `a ≈ 9.78e-4..9.84e-4` (so the `ts` clock is ~1020 units/ms; an older
"946 units/ms" figure is imprecise). **`R²≈0.93-0.99` is misleading** — it is computed over the self-selected
last-before anchors, so a high R² does not mean the right record was chosen (Team Slayer: R²=0.9967 yet 58%, §10).

### 9.4 Attribution (default `ATTRIB=lastbefore`)
> weapon of a kill = family of the **last `0xd2` with `attacker slot==killer slot` and `warp(ts)<=kill.TimeMS`**.
"Nearest" drops to ~74-78% (picks follow-up shots after the kill). Alternatives for experiments: `ATTRIB=nearest`,
`ATTRIB=majw WIN=<ms>` (majority in window).

### 9.5 Numbers (re-measured 2026-06-23)
| Match | Mode | Coverage | Per-kill exact (vs live) | Error |
|---|---|---|---|---|
| `000d5950` | Fiesta | 89/93 = **96%** | 84/89 = **94%** (warp R²=0.9309; earlier run 85/89=96%) | — |
| `9b191a7f` | Team Slayer | 84/87 = 97% | 48/83 = **58%** (aggregate per-player ~79%) | BR75↔MA40 (×18) |
BTB (e.g. `00ba2e1c` ≈ 46% firearm) — the rest have **no `0xd2` at all** (splatter/collision/fall/grenade), largely
irreducible offline; the 46% is a **coverage estimate, not live-validated** (no BTB CE oracle). Label such kills
non-firearm rather than mis-attribute.

### 9.6 Sanity tool `cmd/tmp_d2hist`
Compares the offline `0xd2` weapon histogram vs the live dual-hook histogram + filter-rejection reasons. Proved the
Team Slayer bias is attribution/warp, NOT a decode undercount (BR75 1515 offline / 1769 live = 86%; MA40
1007/1161 = 87%; live total 3194).

### 9.7 Productionisation target
Port into a pure `internal/analysis/killweapon/` (`decode_damage.go`/`roster.go`/`warp.go`/`attribute.go`, façade
`AttributeMatchKillWeapons(chunks)`), wire into `internal/sync/backfill_weapons.go` by **replacing** the old
fire-events scanner + `CorrelateKillsGlobal` (steps 2-4+6) while keeping download/kills-from-DB/persistence into the
`weapon_kills` table (NOT append-only: DELETE-then-INSERT/match; ART safety via shared lease + `MaxOpenConns(1)`,
so don't add write concurrency). Split into <80-line functions; gate behind a capability — **a new
`CapabilityKey` must be CREATED** (the name `CapabilityFilmWeaponAttribution` is only a *proposed* example in the
plan; it does not exist in the codebase) and wired into the capability mapping set, never `slug=="halo_infinite"`;
load chunks via the film path resolver (the prototype hardcodes the path).

---

## 10. Why Path A caps at 58% on Team Slayer (definitive diagnosis, 2026-06-14)

The owner's intuition (Team Slayer has fewer weapons → 58% must be a bug) prompted a full investigation:
1. **Not a decode bug.** `tmp_d2hist`: offline `0xd2` matches live weapon-for-weapon (~86% of each weapon; MA40 not
   under-represented). Filter discards only 115/2874 (`noFam`, `noSfx=0`).
2. **It's the warp.** Offline over-picks BR75 (51 vs 41 live), under-picks MA40 (14 vs 25). Strategy sweep:
   `lastbefore` 58%, `nearest` 51%, `majw(±W)` 43-50% — and `majw`'s denominator collapses: **51/83 kills have NO
   killer-`0xd2` within ±1000 ms of the warped kill** (a killer's MA40 burst warps to +2153 ms, *after* its own
   kill — impossible → warp locally wrong by ~2 s).
3. **The packet-ts is a write clock, not game time.** Even the live-tsc bridge gives 58%, so `packet-ts→real-time`
   itself drifts.
4. **Un-fixable offline.** A piecewise/local warp is circular (the only cross-clock anchor is the kills).
5. **Why Fiesta survives:** varied weapons → unambiguous even with drift; Team Slayer's two rifles → any drift
   confuses BR75/MA40. "Fewer weapons" is *harder*.

**Conclusion:** the warp is the wrong tool for universality. The fix is same-clock matching — the dead-state
(victim, in the same packet-ts clock as the weapon) — i.e. Path B. The warp's failure *validates* Path B.

---

## 11. Weapon-per-kill, Path B: the stateful ECS frame decoder (the universal path, blocked)

### 11.1 Why stateful
A DELTA record carries no typeIndex/default-state; you must hold a `World` of `slot→archetype` from the keyframe +
NEW/DEL. The held weapon (`weapon-state-type-info`, biped i43-46, `FUN_1407f06bc`) carries the family at the
keyframe for all 8 players but **is NOT re-emitted in motion deltas** (only ammo/overheat/chamber state) — current
weapon = keyframe loadout propagated forward, updated at rare swap/pickup records. Hence the World `HeldWeapon`
cache is mandatory. (Source contradiction, resolved: one pass said "weapon not in deltas", a later pass found 247
weapon literals in the type-0 stream — reconciled: motion deltas don't re-emit, swap/pickup records do.)

### 11.2 The binding wall: keyframe default-state deser
To build the World offline you must decode every NEW record's **default-state** (between `R(6) typeIndex` and the
mask). It is a deserializer, not a fixed skip: `CALL [RAX+0x60](…bitreader…)` at `141f868c2` = `vtable[0x60]`; for
the biped (typeIndex 35) → `FUN_140F44C38`, width variable/self-delimited (155/380/382/224/157/44/6 bits across
records). `consumeBipedDefaultState` (`default_state.go`) ports 120/380 bits bit-exact; the remaining ~260 use
per-axis widths populated at map-load (`DAT_1445cc9e0`/`DAT_144632be0`), read 0 statically. Each of the **26**
distinct keyframe archetypes (typeIndex 2,4,5,6,9,13,14,15,17,18,19,21,22,25,26,27,29,34,35,37,38,41,42,43,45,47)
has its own `vtable[0x60]` deser with its own widths.

### 11.3 The key lever: the static quantization-width formula (closes the runtime-width gap)
The widths are NOT an irreducible runtime mystery — they are computable from static `.exe` constants:
```
step(L)   = 2^(16-L) / 120                                  (const 1/120f = 0x3c088889 @143cd9758)
width(L)  = min(26, bitLen(ceil(40000 / (2*step(L)))))      (default range +/-20000 = DAT_143b8c6b8)
```
**Validation L=0:** step=546.13; 40000/(2·546.13)=36.6; ceil=37; bitLen(37)=6 → **6/6/6**, matching the CE
measurement. So the default-precision path is offline-pure exact in principle. Remaining work: apply it to the
high-prevalence components (i6 region ~84%, i7 damage-sections ~90%, i9 obje ~94%).

### 11.4 State of the ported components
- **Biped #35 spine to the weapon is bit-exact** (calibration `start=194126, defaultBits=380, recordStateParam=2`;
  the Hydra record traverses to `endBit=195892`, WST i45 = `0x767db96d`; all 8 keyframe loadouts recovered).
- Biped tail i47..i63 ported; object/unit i0..i17 + dead-state ported. This session's **`obje` i9 fix
  (`FUN_1407d4c94`, TLV form, commit `8fd113cc8`) doubled the clean-frame gradient ~30.8%→~63.1%** (full seed).
- **Batch8** (`components_batch8.go`): contains **11** RE'd deserializer functions (the workflow RE'd 16
  components; 3 already existed elsewhere, 1 was a duplicate in batch3, 1 discarded). They are **deliberately NOT
  wired** — wiring them into the persistent World-seed dropped the gradient ~63%→~25%: subtly-wrong runtime widths
  complete records at a wrong bit, producing false-clean frames whose garbage bindings cascade into unbound slots.
  **Lesson: the component-grind via a static World seed does NOT converge; the binding must come from a clean
  keyframe replay.**

### 11.5 Source map (`internal/analysis/filmdec/*.go`)
`bitreader.go` (MSB-first BE + var-width); `quantize.go` (dequant); `registry.go` (chunk_00 schema); `world.go`
(slot map + rollback); `frame_records.go` (FRAME loop); `traverse.go` (`consumeByName` switch + calibration knobs);
`default_state.go` (biped `vtable[0x60]`, 120/380); `components_movement.go` (i0/i1/i3); `components_object.go`
(i0-i17 + dead-state); `components_object_state.go` (i6,i7,i8,i10,i12-i17); `unit_weaponstate.go` (i18-i46);
`components_biped_ability.go` (i47-i63); `components_team_mapping.go` / `components_world.go` /
`components_batch3/4/5/7.go` (non-biped/world desers); `components_batch8.go` (11 unwired); `statborg.go`
(2-team score parser); `entity.go`/`entity_quant.go` (full-state entity decoders); `position_capture.go` /
`probe_export.go` (observation hooks).

---

## 12. Dead-state & kill-event components (the kill structures)

**Neither carries the weapon model.** The dead-state gives killer/victim/category; the kill-event gives
killer/victim/assistant. The weapon comes from the same-clock `0xd2` join.

Player index packed as `0xE1500000 + idx*0x10002` (film-serialized; `0xEC500000` same stride is the *live*
runtime base — don't conflate). The entity id (`0x4000xxxx`) changes per respawn; the participant index is stable.

### 12.1 Dead-state (object-dead-state, biped form)
Deser `FUN_140c1dd44` (via `FUN_140c1dce0`); biped slot **i11**; Go `consumeObjectDeadStateBiped`
(`components_object.go`). Bit layout:
```
mort = R(1)                          comp+0x70  death flag
present=R(1); if 1 -> R(32)          comp+0x00  anim handle
R(8)
R(1); if 0 -> R(5)                   comp+0x04  VICTIM participant index
R(1); if 0 -> R(5)                   comp+0x08  KILLER participant index
R(4)                                 comp+0x0c  category (4-bit bitstream field)
R(3)                                 comp+0x0e
+0x10 block: presentA=R(1); if 1 -> [gidPresent=R(1); if 1 R(32) raw global-id] + R(3) + (R(1); if 0 R(6))
--- data-dependent tail (R(4)/R(5)/position/RAM-gated velocity) ---
R(1)                                 comp+0xc4  (typeIndex==0x23 only)
```
`+0x04`/`+0x08` = victim/killer (CE-confirmed against real-match narration). **`+0x0c` is a 4-bit bitstream field**;
the category *values* `0x40000` (melee) / `0x10001` (thrown) quoted in some notes are the **resolved RAM struct
values** measured by CE, not the 4-bit field — don't expect a 4-bit field to hold them.

**Why it does NOT carry the weapon:** (1) the `+0x10` resolved handle is **constant across kills** — CE measured
`116963283` over 25 deaths (= `0x06f8b7d3`, the same value reported in another run; they are one constant, not
two); (2) a Gravity Hammer kill vs an antigrav-variant hammer kill produced an identical struct; BR75 vs sniper
identical except the modifier bit. One untested loophole: the *raw film global-id* (read by `FUN_14080d6f0` before
handle resolution) might vary per weapon where the resolved handle is constant — never validated, assessed unlikely.

### 12.2 Kill-event component
Deser `FUN_14104bd08`; strings `victim-/killer-/assistant-participant-handle`. Layout: victim `R(1);if0 R(5)`,
killer `R(1);if0 R(5)`, `R(32)` scalar, `R(1)` flag, **assistant** `R(1);if0 R(5)`, `R(32)` scalar, flag-conditional
tail (sequence/timestamp). The **only source of the assistant per kill**; no weapon. **Locating it in the stream is
open** — the 5-bit grammar matches too many positions across ~29257 frames; all scan/time/xuid-cluster probes
failed. Readable only by a bit-exact stateful decoder.

### 12.3 Where the weapon actually lives (and why not offline through this path)
The kill-feed event consumed at replay by `FUN_1406730c4` assembles a 64-bit weapon id: `high =
*(killEvent+0x1f30)` (family), resolved to a definition, `low = *(def+0x478)`. CE-validated (BR75 `3964796932`,
etc.). The **family is the high-32** = the `'obje'.variant_name R(32)` on the weapon entity (already decoded as
`rec.VariantName`). **Offline wall:** linking the kill-event weapon *handle* to the weapon *entity* needs
`FUN_140495abc`→entity, which depends on live tables (`DAT_144eae7b8`, `DAT_14494a908`) NULL in a replay.

---

## 13. CE ground-truth oracle and the binding gap (Path B blocker)

CE is used ONLY to prove the answer and where in the bitstream it lives. Reference match `000d5950`.

### 13.1 Dual-hook weapon oracle (`tools/ce/filmdec_dualcap_capture.lua`)
Two pure-read hooks sharing one RDTSC clock: **`FUN_1407e00ac`** (damage, RVA `0x7E00AC`) → attacker `[R8+0x0c]`,
victim `*[RDX]`, weapon family `[R8+0x10]`; **`FUN_1406730c4`** (kill, RVA `0x6730C4`) → victim `[RDX+0x04]`, killer
`[RDX+0x08]`. idx = `(handle-base)/0x10002`, base `0xEC500000` (dmg) / `0xE1500000` (kill). **97/98 exact** on
`000d5950` (`cmd/tmp_dualcap`). Not scalable (manual replay per match).

### 13.2 Dead-state capture (`tools/ce/filmdec_deadstate_capture.lua`)
Hooks `FUN_140c1dd44` (RVA `0xC1DD44`). Record stride **`0x60`** (an earlier `0x40` mis-strided the dump; fixed
2026-06-13). Carries resolved victim/killer/source-GID, bitreader counters, and a **16-byte fingerprint of the
bitreader buffer head** at `[0x40:0x50]`.

### 13.3 Validator `cmd/tmp_cematch`: 11/11 paired, 0/11 reached
Reads `000d5950_deadstate.bin`, replays chunks 1..26 against a persistent `World` (seeded from `world_dump.txt`),
finds the packet containing each fingerprint at byte `p`, computes `expected = p*8 + b2c`, looks for a biped
dead-state near it. **11/11 deaths pair to an offline frame** (fingerprint matches → the CE buffer IS the offline
payload) but **0/11 are reached** (real deaths at `b2c` ~6000-9500 bits, never reached cleanly). Target 0/11→11/11.
(Note: `tmp_cematch`/`tmp_deathchain` replay only chunks 1..26 — whether a dead-state could live in chunk_27 was
not checked.)

### 13.4 `cmd/tmp_cleanframe`: the gradient (progress signal, NOT a correctness oracle)
% of frames decoding cleanly to `recEnd`. **Seed-dependent:** ~**56.6%** (16572/29257) with the default light
`world_dump.txt`, ~**63.1%** (18465/29257) with `WORLD_DUMP=world_dump_full.txt`. A clean frame can be
"wrong-but-self-compensated", so validate correctness with `tmp_cematch`, not this.

### 13.5 The blocker: world-object slots never bound
Death frames do NOT desync on player bipeds (512-519) — those decode. They desync **earlier** on a delta whose
slot is a **world object never bound in the World**: slot 1038 is in the full seed but its run-time `recNew` is in
a frame that itself desyncs; slots 1494/1569/1583 are absent from both seeds. To reach `b2c` offline you must
decode `recNew`/`recDel`/delta for the world-object archetypes bit-exactly so the binding stays correct
frame-to-frame. (Even the full seed is a snapshot; entities spawn/despawn, so a static seed can't stay correct
without decoding the intervening NEW/DEL.)

---

## 14. Per-tick biped state (Tier 3 — identified, blocked)

The decoder has RE'd and ported the bit grammars of essentially the **entire biped spine** (i0..i63), so we know
which component carries which stat and which bits it consumes. **Not in production**, blocked by the same wall plus
the walk being incomplete.

Component → stat (verified): **position** `object-position-dynamic-precision` (i0, `FUN_1406cfe44`, the **only one
that emits a value** via `emitPos`); **velocity** i1 (`mag19+scale10`) / i3 angular; **health** `object-body-vitality`
(i4, `R(8)`+3 flags); **shield** `object-shield-vitality` (i5, `R(8)`+regen+`R(16)`+4 flags — **inferred by template**,
re-validate the tail); **max vitalities** i13; **crouch/sprint/slide** biped-mobility-action i54 / biped-slide i62 /
spartan-ability-energy i56; **aim** object-forward-and-up i2 + unit-desired-aiming-vector; **ammo** weapon-state-ammo/
rounds/overheated; **grenades** i47; **abilities** i48; **death** i11 (the one heavy component that captures fields).

**Three blockers:** (1) the delta walk is not bit-exact everywhere — runtime/data-dependent widths, the hard floor
being i63 `biped-action` whose 2nd loop count = popcount of a 73-bit RAM bitmask with 0 stream bits; (2) most
consumers only ADVANCE the reader (only position emits) — wiring each to emit is feasible but subject to (1); (3)
record→player unresolved (the shared wall). Until solved, only keyframe snapshots (~18 s) + position are reachable.

---

## 15. Dead-ends ledger — do NOT retry

**Two hard prohibitions (owner):** (1) "held weapon at tick" is banned (~10×) — the semantic is the damage source.
(2) fire/melee/grenade *events* are banned as the firearm source (v1/v2 unreliable; melee/grenade are fine for the
*actor*, not the firearm-per-kill).

| # | Approach | Verdict | Why |
|---|----------|---------|-----|
| 1 | Held-weapon at tick | REJECTED (owner) | Must be damage source; banned |
| 2 | `0xd2` → victim by bit-scan | INSUFFICIENT | victim-keyed, no clean field; needs live RAM |
| 3 | Dead-state `+0x10` = the weapon | ABANDONED | constant across kills; doesn't discriminate model |
| 4 | Locate kill-event by scan/pattern | ALL NEGATIVE | 5-bit grammar too permissive (29257 frames) |
| 5 | Position-linking (damage→victim by position) | REFUTED | spatial association fails |
| 6 | Sibling markers `0xe9`/`0x89`/… | BECAME the warp | evolved into Path A (not a dead-end) |
| 7 | Tighten the clock warp to beat 58% | CEILING | two clocks, ~1-2 s residual; packet-ts is a write clock (§10) |
| 8 | Fire/melee/grenade events for the weapon | REJECTED (owner) | firearm-per-kill only ~9-13/93 reliable AND banned |
| 9 | Stateless dead-state decode | GARBAGE | ~2.2% valid; delta needs prior state |
| 10 | Global sweeps of `recordStateParam`/`defaultReplRange` | FLAT | per-record/stateful, not global |
| 11 | Quantization widths require CE | REFUTED | static formula §11.3; L=0→6/6/6 |
| 12 | Reach priority "i23→i0/i1/i5" | FALSE LEAD | dead-state is i11; i23 after; i0/i1/i5 absent ~44% |
| 13 | Live CE debugger for the weapon | WORKS, LIVE-ONLY | resolver crashes offline; oracle only |
| 14 | `tmp_cleanframe` gradient as alignment oracle | INVALID | clean ≠ bit-exact; use `tmp_cematch` |
| 15 | The ~413 `tmp_dsraw` dead-states are real | GARBAGE (CE-proven) | 11/11 paired, 0/11 reached |
| 16 | `descriptor+0x28` is the frame-deser | WRONG | that's the KEYFRAME deser; frame uses a wrapper (i9 frame form `FUN_1407d4c94`) |
| 17 | i5 shield = R(8)+3×R(1) | CORRECTED | true deser `FUN_140d50cbc` = 29-55 bits |
| 18 | Reach blocker = component widths i0/i5/i6/i7/i9 | DOWNSTREAM | the death frame desyncs earlier on an unbound slot (binding gap) |

Detail (firearm events): grenade marker `0x4c0c00`, melee `0x534`/`0x535` — both carry the actor and are usable for
the *actor*; but there is **no per-shot firearm event** with a clean player_index (candidate field covers only
`{2,3,6,7}`, never player 0/1; unified hit-event refuted at kill-feed validation 1/93). The deep reason all firearm
paths die in the same place: **biped→player is resolved in RAM at replay** (the `obje` LocalID is absent at
keyframes; i63 `count2` is a RAM popcount with 0 stream bits).

---

## 16. Tooling inventory

From `apps/go-api/`. `CGO_ENABLED=0` for filmdec tools.

| Tool (`cmd/`) | Role | Example |
|---|---|---|
| `tmp_offwarp` | Path A pipeline (96% Fiesta / 58% TS) | `CGO_ENABLED=0 go run ./cmd/tmp_offwarp 000d5950` |
| `tmp_cleanframe` | Clean-frame gradient (seed-dependent; progress signal) | `[WORLD_DUMP=…full] CGO_ENABLED=0 go run ./cmd/tmp_cleanframe` |
| `tmp_cematch` | Ground-truth validator (0/11, target 11/11; `VERBOSE=1`) | `WORLD_DUMP=…full … go run ./cmd/tmp_cematch` |
| `tmp_deathchain` | Sub-frame loop; first desync on the death path | `WORLD_DUMP=…full … go run ./cmd/tmp_deathchain` |
| `tmp_d2hist` | Offline `0xd2` vs live histogram | `go run ./cmd/tmp_d2hist 9b191a7f` |
| `tmp_dscap` | Decode the CE dead-state capture | `go run ./cmd/tmp_dscap` |
| `tmp_dualcap` | Decode the dual-hook capture (97/98) | `go run ./cmd/tmp_dualcap` |
| `tmp_killweaponoffline` | `0xd2` attacker + family vs kill-feed | `go run ./cmd/tmp_killweaponoffline` |
| `tmp_dmgscan`/`tmp_meleegrenade` | damage-timeline / melee+grenade recipes | `go run ./cmd/tmp_dmgscan` |
| `diag_weapons_v3` | Objective-events + score-timeline producer (`-write`) | `CGO_ENABLED=1 go run ./cmd/diag_weapons_v3 -match … -dry-run` |
| `fetch_film_chunks` | Download chunks from a manifest | `go run ./cmd/fetch_film_chunks/ -cache ../../data/cache -dry-run` |

The `cmd/tmp_*` are throwaway/untracked specs for production code, not production code.

---

## 17. Recommended attack plan

Ranked by leverage. The owner's bar is **universal ≥95%**, which only Path B meets for weapon-per-kill.

**A. Ship the already-built Tier-1 producers (light, high value, no wall).** Auto-wire the objective-events
producer (today it only runs via the manual `diag_weapons_v3 -write` — add it to the sync/backfill so the
`match_objective_events` table is populated automatically). Promote the melee/grenade scanners
(`analysis/weaponv3`) into `internal/analysis` + an endpoint (the "enriched kill-feed", §6.5). These carry the
actor inline, so they are unblocked.

**B. Productionise Path A weapon-per-kill (light-moderate, partial).** Port `tmp_offwarp` (§9.7), labelling per-mode
confidence: reliable on varied-loadout modes (Fiesta/Super Fiesta/Husky Raid), "unavailable/low-confidence" on
Arena/Ranked (BR↔MA40) and non-`0xd2` deaths. Visible value now while C proceeds.

**C. The real universal solution — finish the stateful ECS decoder (heavy).** (1) sub-frame-aware stateful walk
(`tmp_deathchain` already loops sub-frames); (2) build the World from a clean keyframe replay, not the static CE
seed — port `vtable[0x60]` per archetype using the **static width formula §11.3** (the widths are NOT a wall);
(3) close the binding gap → `tmp_cematch` 0/11→11/11; (4) same-clock dead-state ⋈ `0xd2` → exact universal weapon.
Do NOT: the component-grind via the static seed (batch8 proved it regresses), dead-state `+0x10`, scan-locating the
kill-event. This same decoder also unlocks Tier 3 per-tick stats and the keyframe loadout once record→player is
solved.

**D. Score-timeline beyond CTF (moderate).** Wire `filmdec.ParseStatborgRecord` to the film framing (locate the
statborg record by component type-id) → all teams/modes; OR ship the validated Slayer/KOTH-B byte-aligned decoders.
Re-validate Strongholds cross-match before exposing it (currently invalidated, §8).

**E. Exact per-match oracle (not scalable).** The dual-hook CE (`tmp_dualcap`, 97/98) for individual high-value
matches; also the validation oracle for C.

---

## 18. Source map (where to read more)

**Master broad recap:** `.ai/V7.5/film_re/RECAP_STATS_EXPLOITABLES.md` (the 3-tier overview — read first for the full scope).
**Weapon master index:** `.ai/README_KILLWEAPON_INDEX.md` (impasses, RE facts, tools — weapon-scoped).

Key docs (`.ai/`): `../killweapon/FIREARM_PER_KILL_OFFLINE_SOLVED.md` (Path A; its §3-§5 "-20.3 s offset" is stale — only the
banner + `tmp_offwarp` are current), `HANDOFF_FRAME_DECODER_L3.md` (Path B `vtable[0x60]` wall),
`../killweapon/KILLFEED_DEATH_RECAP_FIELDS_RE.md` (dead-state/kill-event grammars + negative scans), `../killweapon/KILLFEED_STATE.md`
(packet carto), `RESEARCH_THEATER_RE.md` §M (objective/score RE), `RE_EXE_GHIDRA_FINDINGS.md` (native getters),
`../killweapon/FIRE_MELEE_GRENADE_EVENTS.md` (melee/grenade/fire), `PLAN_FILM_ECS_DECODER.md`, `../killweapon/PLAN_SAMECLOCK_ATTRIBUTION.md`
(width formula), `../killweapon/PLAN_WEAPON_PER_KILL_PRODUCTION.md`, `../../thought_log.md` (chronology; the 2026-06-14 entries =
the latest weapon findings).

Code: `apps/go-api/internal/analysis/{filmdec, objectiveevents, objectivescore}/`, `comeback_objective.go`,
`highlight_event_parser.go`, `weapon_data.go`; `cmd/{tmp_offwarp, tmp_cematch, tmp_deathchain, tmp_d2hist,
tmp_dualcap, tmp_killweaponoffline, diag_weapons_v3, fetch_film_chunks}`. CE: `tools/ce/*.lua` + `*.bin`.

Recent commits (branch `feat/weapon-attribution-v3`, HEAD `62521aa39`): `8fd113cc8` (obje i9, gradient ~31→63%),
`9054510cd` (object-low-frequency), `62521aa39` (warp non-universal diagnosis + multi-frame).

---

## 19. Known uncertainties / contradictions (resolve empirically)

1. **Clean-frame gradient is seed-dependent:** ~56.6% (default light `world_dump.txt`) vs ~63.1% (`world_dump_full.txt`).
2. **Fiesta number 94% vs 96%:** two metrics on the same match (tsc-bridge 84/89 vs an earlier offline 85/89);
   quote "94-96%". The 9b191a7f 58% and the d2hist/cematch results were re-measured 2026-06-23 and match.
3. **`world_dump_full.txt`** body has ~2228 slots; its header line ("1979 slots") is stale.
4. **Strongholds score curve** is contested — finely reconstructed on `7344d24f` only; a 38-match cross-validation
   invalidated the per-team mapping. Do not expose it as a score.
5. **`recordStateParam=2`** is the calibration value but not proven at the binary (only its static provenance is).
6. **Runtime World datum-store stride** is `0xa0` in `world.go`'s comment vs `0xC8` per a later RE verdict —
   UNRESOLVED, but **irrelevant to offline decode** (the Go World is a slot-keyed map, not a stride-indexed access).
   Separate fact: the on-disk chunk_00 registry slot = 260 bytes / block = 0x4100 (verified).
7. **Dead-state `+0x10`** raw film global-id was never captured for per-weapon variance (only the resolved RAM
   handle, constant); loophole open but assessed unlikely.
8. **`CapabilityFilmWeaponAttribution`** is a proposed name; it does not exist in the codebase yet.
9. **Superseded RE hypotheses (do not reintroduce):** sibling markers (`0xe9`/`0x89`/…) as a usable damage
   source; the **"+0x0c global-id = family" fusion** (the family is `variant_name` at **+0x14**; +0x0c is the
   R5 attacker — this was the bug, corrected 2026-06-12); the "-20.3 s constant offset" cross-correlation
   (replaced by the linear warp); "946 units/ms" (the measured clock is ~1020 units/ms, a≈9.8e-4). Current
   facts: `0xd2`-only, **family at +0x14 / attacker R5 at +0x0c**, linear warp.
10. **All cross-match counts** (4 CTF, 38 SH, 11 KOTH, 519 dmg records, 70 grenades, 97/98 dual-hook) come from code
    comments / RE docs; the live numbers re-run for this doc are the ones in §9.5 and §13.3-13.4. Re-validate any
    figure against a fresh tool run before quoting it in a report.
