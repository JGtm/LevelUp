# ADR 0037 — Film intermediate representation: the production walk becomes the grammar's single reading

**Status**: Accepted (2026-10-02). Step 1 is being executed on the branch below; steps 2 and 3 are
planned, not started. Amends [ADR 0034](0034-film-decoder-profile-and-layers.md) D-1, D-2, D-6,
D-7 and D-10 (section "Amendments to ADR 0034").

**Branch**: `feat/representation-intermediaire` (worktree `LevelUp-wt-ri`, base `feat/v75`
`93cea7cdc`).

**Relates to**: [ADR 0034](0034-film-decoder-profile-and-layers.md) (five layers, one gate to the
bytes, one revision per layer, facts persisted apart from publication, named fallbacks),
[ADR 0030](0030-persist-write-aggregates.md) (close the construction at compile time, ratchet
second). Supersedes nothing.

Go paths are relative to `apps/go-api/internal/games/halo_infinite/film/`, except those that start
with `cmd/` or `internal/archlint/`, which are relative to `apps/go-api/`, and those that start with
`docs/` or `.ai/`, which are relative to the repository root. Figures were measured on `feat/v75`
`8b894a677` (2026-10-01) and come from
`.ai/ANALYSE_MISE_EN_OEUVRE_REPRESENTATION_INTERMEDIAIRE_2026-10-01.md` (sections 1 and 2) and the
two reports it cites; they are re-measured at each step closure.

---

## Context

### What the grammar produces today

The grammar knows how to read a film, but what it reads exists nowhere as a whole. Each channel of
a cook (positions, shots, equipment, movement states and so on) starts its own pass over the film,
keeps what concerns it and drops the rest; where the grammar cannot traverse yet, it searches for
record headers bit by bit.

| Fact (2026-10-01) | Value |
|---|---|
| Passes over the delta packets per cook | about 37, plus 3 under mode guards: 22 bit-by-bit scans, 3 grammatical walks, 6 reads of the view-A head, the rest in `replaybuild` and killsource |
| Grammatical walkers of delta frames | 3, incompatible: 3 views (movement states, `internal/grammar/movement_states.go`), 8 views (object deaths, `internal/grammar/object_deaths_march.go`), 8 views (`internal/facts/killsource/walk.go`) |
| `IDLowBits` | two provenances: 13 hard-coded (`grammar.DefaultFrameConfig`) in two walks, 10 to 15 calibrated in the third |
| Anchored biped walker | 10 runs per cook |
| Useful records closed to the bit, best build (HI_1_13_0) | 80.2 % (older builds 0 to 30 %) |
| Share of "view C: terminator out of frame" in the useful records not closed | 92.5 % on HI_1_13_0, 91.9 % on the corpus; closing every other cause caps HI_1_13_0 at 81.3 % |

The consequences are the ones the decoder audit of 2026-09-24 listed: twin readers that decide
differently, a grammar defect that surfaces downstream in one channel and not in another, a
closure measured by an instrument that copies the production walk's driving loop
(`internal/grammar/frame_closure.go`), and a provenance lost at serialization.

### What already exists

The germ is the production walk itself. `grammar.ScanMarcheDesTrames` runs the game's frame
processor (`FUN_142987460`) on every delta packet — configuration bit, view A (messages), view B
(entities), view C (control) — ends on the closure oracle (`vueCFermee`), and publishes movement
states and continuous fire through the hooks of the `Observation` while it reads. The closure map
(`grammar.FrameClosure`) replays the same walk by copying its chunk and packet loop. Records carry
a `HeaderBit` field that the production loop does not set.

### An independent confirmation, and its lesson

A third-party Rust port of this decoder (S. Imbleau, MIT or Apache-2.0, film version 41 only,
pinned on our `43a01721e` and the readers of `d61443ef5`) built the separation this ADR adopts — a
canonical reading with spans and typed stops, resolution apart — almost to the letter. Measured on
our ten version-41 witnesses with our closure oracle, it closes **5.5 %** of the type-0 frames
(18,587 of 335,960) against **66.9 %** for our head (224,818 of 335,960, same packets). It removed
every recovery, and with it the entity context (datum table, keyframes, anticipated table). The
direction holds; a canonical-only reading does not, on Halo Infinite. Recovery is necessary, and it
must be a separate, named and counted layer.

### Corrections to the specification

The specification (`.ai/SPEC_REPRESENTATION_INTERMEDIAIRE_FILM_2026-09-25.md`, with a draft ADR in
its section 12) was checked against the code on 2026-10-01 and corrected on eight points (C1 to C8,
section 2 of the analysis): one entity table, not three (C1); two phases, not one pass (C2); a
refused closure distinct from an opaque tail (C3); recovery that already lives inside the walk
(C4); a migration order that keeps zero difference (C5); the location of the types (C6); the origin
of the "7.9 GB" peak — a CTF film, `51101d1d`, unrolled by `NamedEventsFrom`, fixed on 2026-09-03;
current peaks are 0.08 to 1.19 GiB (C7); and step 1 as the production walk refactored and
consumed, not a second walker (C8). This ADR carries the corrected decisions; where it reads
differently from the specification, it wins.

## Decision

### IR-1 — The grammar's reading is a structure shaped by the stable levels of the format

The production walk ranges what it reads into a typed structure — packets, views (A messages, B
entities, C control), records, components — each with its exact extent in bits. An extent is
`{chunk, packet, first bit, length in bits}`; it is the unit of provenance and of closure. The
shape follows the levels of the format that do not change from one build to the next; what depends
on the build stays in the profile and the grammar (ADR 0034 D-3). A record is identified by
`types.LifeKey` (slot, generation), never by the slot alone.

### IR-2 — The walker is the production walk (correction C8)

There is no second walker. The loop that drives `grammar.ScanMarcheDesTrames` becomes
`FilmContext.Trames`, and the walk's two current consumers — movement states with continuous fire,
and the closure map — consume it from step 1 on. The copy of that loop in `frame_closure.go`
disappears. A walker that nobody consumes would be a second parser, which is what this ADR exists
to remove.

### IR-3 — Two phases and bounded preliminaries, not one pass (correction C2)

A single forward pass is impossible as the walk stands: the anticipated table reads later keyframes
(`internal/grammar/keyframe_anticipe.go`), the biped slot band reads the keyframe of the next chunk
(`bipedSlotBand`, `internal/grammar/offline_biped_band.go`), and the dated living-generation filter
refuses a record that precedes a later creation (`internal/grammar/generations_vivantes.go`). The
walker therefore has two phases, each an `iter.Seq2[*lecture.Paquet, error]` in stream order:

- `FilmContext.ImagesCles` — every keyframe, once per film, through the existing keyframe walk and
  its memo (`MarcheDImageCle`, proof included);
- `FilmContext.Trames` — every delta frame, under the current walk state.

The bounded preliminaries (anticipated table, biped slot band, living generations) run before the
delta phase, as they do today.

### IR-4 — Three states per component, three closure states per packet, never conflated (correction C3)

A component occurrence is **interpreted** (a typed value was published during the walk),
**delimited** (its extent is known, its meaning is not published) or **untraversable** (its width is
unknown: the rest of the view becomes an opaque tail). The `status` column of
`internal/grammar/testdata/ecs_table.tsv` (`porte`, `partiel`, `non_porte`) is a static capability;
the state belongs to the occurrence.

A delta packet carries one of three closure states:

- **closed** — the grammar's closure predicate holds (today `vueCFermee`: the walk ends on the
  view-C terminator with 0 to 7 zero bits left), and every record of the packet is proven with it;
- **closure refused** — the views were read to their terminators but the packet does not close: a
  width is wrong somewhere before, at an unknown position. Its records are **not proven**. This is
  today's "view C: terminator out of frame";
- **opaque tail** — the walk stopped at a known position, with a typed cause: unported component of
  a record, unported view-A message, unlocated event list, end of payload in view B, or a view-C
  stop (overflow, kind 1 or 2, the 0xbc block, the loop cap).

The closure predicate belongs to the grammar and may tighten (with invariants of the game's writer,
as the grammar campaign does); the structure carries its verdict, never a second definition. The
exit of view B is typed: terminator, rejection outside the datum table, rejection by another view,
untraversable record, end of payload.

### IR-5 — One entity table, exposed read-only (correction C1)

The walk state is ONE table: slot to full eid, archetype, view and binding provenance — the shared
decoder's datum table, of which the keyframe is the dump (`internal/grammar/frame_infer.go`, note of
lot 5.16). It is not three tables, one per view; the view is an attribute. It is exposed read-only
during an iteration (`lecture.Entites`). The provenance of each binding is explicit: a NEW record
read, a keyframe chain, the datum table read at a free position, anticipation, chain inference, or
a wildcard of unknown generation.

### IR-6 — Recovery is a separate, named and counted layer, and the recovery inside the walk is marked (correction C4)

Heuristic recovery (anchored biped headers, creation headers, world-object tracks, byte motifs, the
statborg scan) produces records marked **recovered**, named after their method and counted, never
mixed with what the walk read. Five recovery mechanisms live inside the walk today: the localisation
of view B by the slot-123 signature (`marchLocateStrict`), the leading NEW records of an event list
(`debutDeLaListe`), the datum table read at a free position (`TableDeDatums`), binding by
anticipation (`repli_liaison_par_anticipation`) and keyframe anchor election
(`repli_ancre_d_image_cle_par_election`). Each is **marked where it acts** — a packet says how the
start of its view B was found (read from the packet head, or located), a binding says where it came
from, a keyframe record says whether its anchor was chained or elected — and its behaviour does not
change. Without the mark, a closure measured on the structure would mix grammar and recovery.

### IR-7 — Off-stream parameters are explicit inputs with their provenance

A value that the payload does not carry (`IDLowBits`, the MPP widths, the i0 layout, `gate15`, the
vehicle byte `+0x818`) is an input of the walk, with its provenance: read, calibrated, or assumed.
`IDLowBits` has two provenances today; it is the first inconsistency the structure resolves, in
step 2.

### IR-8 — Streamed, lazy, without copy; interpretation stays in the hooks; nothing persisted

A packet's payload is a sub-slice of its chunk. Records and components live in an arena reused from
one packet to the next and valid during the iteration only; a test or a tool clones what it keeps.
Target sizes: about 40 bytes per record and 12 per component, against an estimated 350 bytes
allocated per four-component delta record today. Interpretation stays where it is: the
`Observation` hooks publish values during the walk, so no published value moves. The structure is
never persisted: the facts cache already replays a cook 95 to 442 times faster than a decode
(ADR 0034 D-7).

### IR-9 — The types live in a leaf package of the grammar layer (correction C6)

The types live in `internal/grammar/lecture`: no logic, and no import but `film/types`. Because it
sits in the grammar's tree, its shape is hashed by `grammar.Rev` only (a layer is recognised by the
prefix of its root, `revision/couches.go`); in `film/types` it would be hashed byte for byte in four
perimeters. Neither `replay` nor `decfilm` imports it (ratchet in `internal/archlint`): the
publication layer never sees the structure, and the facade does not grow. Names are French, like
the rest of the package (`ImagesCles`, `Trames`, `lecture.Paquet`, `lecture.Record`,
`lecture.Composant`, `lecture.Etendue`).

### IR-10 — The migration keeps zero difference, and behaviour changes come last (correction C5)

Creations and positions cannot move to the walk at zero difference: on `bfecd02b` the anchor search
finds 162,444 biped records, the walk 97,447 (`internal/grammar/movement_states.go`). They stay in
recovery. The order:

1. **Step 1, zero difference**: the types; the delta phase on the production walk with its two
   consumers; the keyframe phase with `KeyframeClosure` as consumer; tests T1, T3, T5 and T6.
2. **Step 2, zero difference except declared fallback counts**: channels move to the structure in
   this order — movement states and continuous fire; keyframe channels; view-A head channels; one
   anchored biped recovery per film instead of ten; world-object recovery; the statborg scan down
   from `internal/facts/objectives` into the grammar. Then the **declared behaviour changes**:
   object deaths on the single walker, delta channels read by the walk where it covers better,
   killsource last.
3. **Step 3**: redundant walkers removed, one film context per cook, performance measured.

The behaviour changes wait for the grammar campaign: below about 81 % of closure, reading the delta
channels by the walk would lose records that the anchor search finds.

### IR-11 — Tests

| | Test | Form |
|---|---|---|
| T1 | Closure: for every packet, the bits consumed and the closure state agree | goldens `frame_closure.golden` and `keyframe_closure.golden`; a ratchet that refuses any decrease |
| T2 | Component: a bit vector built from the game's writer gives the width, and the value when interpreted | mandatory for every component that becomes `porte` |
| T3 | Provenance: every movement-state and continuous-fire reading cites an extent that exists in the structure | step 1; every published fact by the end of step 2 |
| T4 | Migration equivalence: each lot yields the same document | `cmd/replay-equiv` at zero difference, or a change declared at `cmd/replay-corpus-gate` |
| T5 | Robustness: no panic, allocations bounded by the remaining bytes | the fuzz harness `FuzzFilmRecordReaders` extended to the walker |
| T6 | Determinism: two walks give the same fingerprint, parallel walks are identical | `-race`, CI job `film-race` |
| T7 | External oracle (optional) | the live capture; a third-party port as a second closure counter |

### IR-12 — The 95 % threshold is an indicator, not a trigger

The specification's trigger (at least 95 % of the useful records closed on every build) is out of
reach without the dense residual: 81.3 % caps HI_1_13_0 once every other cause is closed. User
decisions: step 1 opens before the threshold, since it changes no output (2026-10-01); the
threshold becomes an indicator published at each wave of the grammar campaign, not a trigger
(2026-10-02).

## Amendments to ADR 0034

### D-1 — Five layers. **Amended: the grammar's output is the intermediate representation.**

The grammar row now reads: record and component decoders, **and the walker that ranges what they
read into the intermediate representation** (IR-1 to IR-3). The types live in
`internal/grammar/lecture`, classified in the grammar layer by
`internal/archlint/film_layers_deps_test.go`; `replay` and `decfilm` never import them. From step 2
on, the facts layer consumes the structure instead of walking the film again. "`replay` decodes
nothing" is unchanged.

### D-2 — One gate to the bytes. **Amended: the facts read the structure.**

"Nobody outside `source` reads a bit" stays. The target adds: nobody outside the grammar walks the
packets of a film; the facts read the structure. Two facts consumers still walk packets themselves,
named here so they are not rediscovered: the statborg scan of `internal/facts/objectives` (it moves
into the grammar at step 2) and the killsource walk `internal/facts/killsource/walk.go` (it folds
into the single walker at the end of step 2, then disappears at step 3).

### D-6 — One revision per thing that can change. **Amended: the shape of the structure is the grammar's.**

The structure is an output of the grammar, and `internal/grammar/lecture` sits in the grammar's
tree, so a change of its shape raises `grammar.Rev` and nothing else. A lot proven at zero
difference (`cmd/replay-equiv`, the closure goldens) regenerates the grammar's fingerprint and
perimeter at constant revision, as the structural lots of the audit follow-up did. `grammar.Rev` rises only for
a behaviour lot; each rise raises `killsource.Rev` and reopens the killsource backlog, on user
signal.

### D-7 — Facts and publication are separate. **Amended: the structure is not persisted.**

The persisted unit stays the facts. The structure lives during a walk and dies with it. Persisting
it would be decided on a measurement (the specification's step 4); none is planned.

### D-10 — Grammar decides; a fallback is named, counted and retired. **Amended: recovery is a layer, and the recovery inside the walk is marked.**

The four rules hold for every recovery method: named in the registry, ordered after the reading,
counted, retired. In addition, a record produced by recovery is marked "recovered" in the
structure, and the five mechanisms that live inside the walk are marked where they act (IR-6). A
published count that depends today on the number of passes (`repli_bande_bipede_comblee`, summed
per scan) changes when the passes are mutualised: that is a declared change, not a regression.

## Non-goals

- **Not a second parser.** The grammar — the profile per build and the readers ported from the
  game's writer — stays the only authority on what is where; it fills the structure.
- **Not a rewrite of the readers.** The component readers are the walker's bricks, unchanged.
- **Not a change of the published document.** The structure is internal to `film/`.
- **Not an interpretation of everything.** Only the components the product uses get an
  interpreter; traversing the others is enough.
- **Not Halo 5**, whose film is another format, and **not an encoder**.
- **No persistence of the structure** (IR-8).

## Consequences

Gains: one reading per film instead of about forty; closure becomes a per-packet, per-build
invariant measured on the structure itself; a new build breaks in a localized, measured way ("from
build X on, component Y of archetype Z no longer closes") instead of channels finding less without
saying so; every published fact can cite the extent it comes from; one `IDLowBits`.

Costs and risks, each with its parry:

- **A staged migration.** Every lot is proven at zero difference or declares its change; a
  difference stops a structural lot instead of being explained away (ADR 0034 D-6).
- **A parallel grammar campaign that changes outputs.** After each merge of a campaign wave into
  `feat/v75`, the equivalence references move: this effort merges `feat/v75`, re-freezes its
  references and replays its zero-difference proof against them. File ownership is agreed per step
  (`.ai/PLAN_REPRESENTATION_INTERMEDIAIRE_ETAPE1_2026-10-02.md`, section 1.3).
- **Memory.** The film stays resident (the forward reads of IR-3 forbid releasing a chunk after its
  walk); the gain of IR-8 is on records, not on the film. Duration and memory peak of a cook are
  measured before and after each step on three witnesses and one BTB; a regression beyond 10 % stops
  the step.
- **A structure that grows a second definition of closure.** Parried by IR-4: the predicate is the
  grammar's, the structure only carries its verdict.

## References

- `.ai/SPEC_REPRESENTATION_INTERMEDIAIRE_FILM_2026-09-25.md` — the specification and its draft ADR.
- `.ai/ANALYSE_MISE_EN_OEUVRE_REPRESENTATION_INTERMEDIAIRE_2026-10-01.md` — the corrections C1 to C8,
  the form, the lots, the order.
- `.ai/V7.5/film_re/RAPPORT_IR_CARTOGRAPHIE_GO_2026-10-01.md` — the inventory of passes, the germs,
  the Go design, the detailed migration.
- `.ai/V7.5/film_re/RAPPORT_PORT_RUST_2026-10-01.md` — the third-party port and the measurement on
  our witnesses.
- `.ai/PLAN_REPRESENTATION_INTERMEDIAIRE_ETAPE1_2026-10-02.md` — the execution plan of step 1.
- Every package, test and ratchet named above, cited inline where it applies. Chronicle, not
  source: the code is.
