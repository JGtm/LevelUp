# ADR 0034 — Film decoder: an immutable profile per build, five layers, one gate to the bytes

**Status**: Accepted (2026-09-13). To be amended at the M2 and M4 closures of
`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, when the layers exist and the publication path is built.

**Branch**: `feat/decfilm-0C` (lot 0.C of that plan)

**Relates to**: [ADR 0009](0009-expvar-monitoring-multi-user.md) (expvar counters),
[ADR 0012](0012-halo-only-adapters-extraction.md) (Halo-only code under `games/halo_infinite/`),
[ADR 0025](0025-title-agnostic-minimal-viable-window.md) (`analysis/` never imports a title),
[ADR 0030](0030-persist-write-aggregates.md) (close the construction at compile time, ratchet
second), [ADR 0008](0008-db-schema-multi-title-and-xuid-global.md) (every path through
`PathResolver`). Supersedes nothing.

---

## Context

Three September 2026 defects had one cause. Projectiles were read one bit early on Live Fire
because the region index width was written twice — the biped reader followed the map catalog, the
world-object reader a literal. The kill feed was empty on 211 films of 2025 because the film
version (first four bytes of `chunk_00`) was read by nobody, and three callers passed "unknown
version". The corpus gate stayed green on both: it had no witness on the map or on the version
concerned, and the decoder fingerprint only hashes `killsource/`.

The common cause: knowledge of the format lives in several copies — package globals, literals,
per-reader decisions — and no single object says "this is the film, and this is how it reads". When
two readers decide differently, only the one with a corpus witness gets caught.

The state this ADR closes, measured on the tree: `archlint/filmdec_package_vars_test.go` freezes
**96** package-level names in `filmdec`; `filmdec.LockProcessDecode` (`filmdec/decode_gate.go`)
serializes every decode of the process, and `archlint/decode_lock_held_test.go` requires production
paths to hold it; two bit readers exist (`filmdec.BitReader`, `killsource.evReader`) and six
packages outside the source layer read raw chunk bytes.

This ADR records the durable invariants. Lots, dates and measurements live in the plan; `.ai/`
rotates faster than it is maintained, so nothing here depends on it.

## Decision

### D-1 — Five layers, one dependency direction

The decoder is five packages under `internal/games/halo_infinite/film/`:

| Layer | Responsibility |
|---|---|
| `source` | load, decompress, cut into chunks and packets, read the header, own the canonical bit reader. Knows nothing of what a record contains. |
| `profile` | the profile table: per build, per map. Data derived from the game. No read logic. |
| `grammar` | record and component decoders: pure functions of (profile, bits) to typed values. |
| `facts` | from raw chronology to match facts: lives, identity, shots, deaths, objectives, equipment, vehicles. Each fact carries its coverage counters and its revision. |
| `replay` | publish the versioned replay document. Decodes nothing. |

Dependencies run one way only: `source` -> `profile` -> `grammar` -> `facts` -> `replay`. Never the
reverse. `analysis/` never imports a title package.

**Enforcement: the compiler first, ratchets second** (ADR 0030). The inner layers live under an
`internal/` sub-directory of `film/`, so nothing outside the decoder can import them at all.
Ratchets guard only what the compiler cannot express: sibling-layer dependencies
(`archlint/film_layers_deps_test.go`, to be added at step 2.5.0), the raw-bytes rule below, file
size (`archlint/film_file_size_test.go`, to be added at step 2.7.2). Existing ratchets in the
family: `archlint/{filmsource_leaf_test.go, no_film_reread_test.go,
no_title_package_in_analysis_test.go}` — the last one's dated allowlist
(`franchissementsToleres`) empties at step 2.5.e.

**Corrected on 2026-09-17** (measured while preparing step 2.6): that allowlist carries **five**
entries, not the single one this paragraph claimed. `analysis/sessionusage/usage_outcomes.go` is
the only *production* crossing; the four others are tests —
`analysis/objectiveevents/{assaut_footer_research_test.go, extract_test.go}` and
`analysis/filmsource/source_test.go` (films opened from the local cache) and
`analysis/weapon_index_equivalence_test.go`. Three of the five fall out mechanically with the
moves of step 2.5; the two that need a port are `sessionusage` (its equipment-usage output types
rise to `domain/` or `games/canonical/`) and `weapon_index_equivalence_test.go` (the comparison
goes down to the decoder side). The count matters because "one entry" made the emptying sound
like a rename: it is five ports, two of them real.

### D-2 — One gate to the bytes

Nobody outside `source` reads `chunk[i][j]`, and nobody outside `source` creates a bit reader.
`killsource.evReader` is absorbed by the canonical reader, with a bit-for-bit equivalence test on
the event chains. Guard: `archlint/no_raw_film_bytes_outside_source_test.go`, allowlist empty from
the day it is written (to be added at step 2.4.3).

### D-3 — The profile is immutable, resolved once, keyed by build

One `Profile` per film, resolved when the film context is built, immutable afterwards. It carries
the map bounds, axis and index widths, quantums, gamertag layout, keyframe layouts and the known
grammars; it replaces the package globals and the width literals in the readers. The key is the
**build**, read in clear text in `chunk_00` section 2, with the major version as fallback; seven
builds are present in the local cache today.

Three rules bind every entry:

1. **Every value carries its provenance, in the code**: the Ghidra function or the named witness
   film, plus a date. A value without a proof is not a profile value.
2. **The grammar comes from the game.** Where the writer is readable in the executable, the value
   is taken from the writer. Measurement over films is an *oracle* that confirms it, never the
   production path.
3. **An inference is never a profile value.** Where the profile knows, the inference runs only in
   a corpus test that compares what it sees with what the profile says. Where the profile does not
   know, we do not guess (D-4).

Data and code stay separate: what is derived from the game is a versioned catalog under
`data/titles/halo_infinite/reference/` through `PathResolver`, never written at run time
(`archlint/no_runtime_versioned_catalog_write_test.go` extends to it); what reads is typed code; a
value never lives in both. `cmd/mapquant-build` is the model for the fabrication tool.

A profile may inherit from another, but the inheritance is an explicit entry marked *presumed*, and
a test (`TestProfilPresumes`, to be added at step 2.1.1) lists the presumed entries. The corpus
turns *presumed* into *proven*, nothing else does.

### D-4 — An unknown build fails loudly

A build absent from the profile gives a typed error (`ErrUnknownBuild`, to be added at step 1.5.2),
an expvar counter per build (ADR 0009), and the film is set aside. It is **never** decoded with the
previous profile. Same for a map outside the catalog. The orchestrator (`sync/killcollector`,
`replaybuild`) is the only layer that logs and publishes counters; `grammar` returns a value plus
typed diagnostics and `facts` aggregates them into coverage. No error is swallowed.

### D-5 — No package-level mutable state in the decoder

Readers are pure functions of (profile, bytes). No install, no restore, no observation hook stored
in a package variable: hooks become an `Observer` passed as a parameter, `nil` in production, and
the width table becomes a field of that observer.

The consequence is that `LockProcessDecode` and `decode_gate.go` are **deleted**, and
`archlint/decode_lock_held_test.go` is **inverted** at step 2.3.2: today it requires the lock on
production scans, afterwards it forbids taking it at all.
`archlint/filmdec_package_vars_test.go` is ratcheted down family by family, from 96 to 0 mutable
variables (`const` stays). Two films of different builds decode in parallel with identical output,
proven under `go test -race` (`TestDeuxFilmsEnParallele`, to be added at step 2.3.3).

**Kill-switch (CLAUDE.md rule 11).** The migration keeps, for a time, a double write: the reader
takes its value from the profile and still writes the global, under an equality test
(`TestProfilEgaleGlobales`, to be added at step 2.1.3). That double write is a dated kill-switch,
and its three dates are written in the code beside it: **switch date** = the date of lot 2.1,
**target removal** = lot 2.3, **measurable criterion** = package-variable count at 0. It is not a
feature left off "for later"; it exists to make the migration provable and it dies at its target.

### D-6 — Change control: one revision per thing that can change

| Revision | Rises when | Consequence |
|---|---|---|
| `GrammarRev` (to be added at step 0.A.4) | any grammar change | fingerprint test goes red without it |
| `KillSourceDecoderRev` (`sync/killcollector/collector.go`, `killsource-2026-09-12`), later `facts.Rev` | the kill-source output can change | killsource backlog, on user signal |
| `SchemaVersion` (`film/replay/document.go`, 54) | the cooked content or the document shape changes | one chronicle entry, artifacts to re-cook |

Fingerprint tests refuse a source change without the matching revision bump. The model exists
(`sync/killcollector/decoder_rev_fingerprint_test.go`); step 0.A.4 mirrors it over `filmdec/`.

How a lot is judged depends on its nature, and the two are not interchangeable. A **structural**
step (moving code, passing the profile, splitting files) changes no byte: the proof is
`cmd/replay-equiv` at **zero difference**, and a difference stops the step instead of being
explained away. A **behaviour** lot is judged by `cmd/replay-corpus-gate` — zero loss, named gains
— and `replay-equiv` then serves to *locate* what moved, not to approve it.

Coverage per archetype never goes down: `filmdec.KeyframeClosure` and its golden
`testdata/keyframe_closure.golden` (to be added at step 0.A.3) are the ratchet. Every migrated
profile value comes with a mutation proof: falsifying it must redden a named test.

### D-7 — Facts and publication are separate

Facts are persisted per film with the revision of the layer that produced them
(`data/cache/film_facts/`, through `PathResolver`; to be added at step 4.1.1). Publication replays
from the facts without decoding again: a publication change never re-decodes, and a grammar change
re-cooks only what depends on it.

The document carries a revision **per layer** (step 4.2.1). The presence of a layer is read in its
revision, never in the absence of a field — that absence is what produced the "between two schema
versions" regressions. A field added without a layer revision reddens the shape fingerprint (D-8).

There is **one published type** (step 4.3): `domain/replaydoc` enriched, produced directly by the
replay layer; the twin conversion is deleted with its tests, and the
`X-Replay-Latest-Schema-Version` header (`api/middleware/cors.go`) carries only the producer
version. Until then the two twins are kept in step by the shape fingerprint and by
`service/replayview/parity_test.go`. One artifact sink only:
`archlint/no_second_artifact_sink_test.go`.

### D-8 — The Go / web contract

Proofs cross the boundary, and they cross it in one direction: **Go produces, vitest consumes**.
Go paths below are relative to `apps/go-api/internal/`, web paths to `apps/web/src/`.

1. **Fixtures produced by Go.** `film/replay/contract_fixtures_test.go` cooks the deterministic
   mini-film and writes `features/match-replay/test/fixtures/go/` (`manifest.json` plus a
   compressed document per schema version). Vitest runs each fixture through
   `normalizeReplayDocument` and through the pure layer logic (`goFixtures.contract.test.ts`).
2. **Compatibility matrix, pinned.** The web names the lowest version it can render:
   `MIN_RENDERABLE_SCHEMA_VERSION = 27` (`model/replaySchemaStatusLogic.ts`). The value is
   justified, not chosen: the chronicle carries exactly two bumps that *remove* a field promised to
   the client — v6 (`Inventory.a`) and v27 (`weaponChanges[].until`); from 27 on every bump is
   additive or content-only. Above the bound a fixture must render; below it the admin badge says
   `stale`, never throws. The bound is pinned by test; moving it takes a product decision, a
   chronicle entry and a Go fixture at the new bound. Known limit: it is justified by *reading* the
   chronicle, because the repository has no per-version key inventory. The inventory that would
   make it computable is the shape golden (point 4) frozen at each bump; until then the bound is a
   reviewed assertion.
3. **Strict contract at the transport boundary.** A zod schema (`lib/replay/replayDocumentSchema.ts`)
   validates the document before normalization; the failure travels as *data* to the admin badge,
   put into words in FR and EN, and no render ever falls. The root and `bounds` are strict today, so
   a renamed key comes back named. Nested strictness is deliberately deferred to the single
   published type (D-7), where the schema can be **derived** rather than written a second time by
   hand; until then the deep shape is guarded on the producer side by point 4.
4. **Shape fingerprint on the Go side.** `film/replay/document_shape_test.go` and
   `testdata/document_shape.golden` freeze the document shape in clear text next to its
   `SchemaVersion`, and refuse regeneration when the shape moves while `SchemaVersion` has not. A
   new version also requires an entry in `document_chronicle.go`, read through
   `testutil/replay_chronicle.go`.
5. **Old schemas never regenerate.** `replaybuild/artifact_schema_history_test.go` proves, per
   version the chronicle declares, that `replaybuild.Digest` classes it stale and the store refuses
   it.
6. **No hand-written schema numbers in web tests.** `testDoc.guard.test.ts` forbids a schema version
   literal outside `fixtures/go/`, in three written forms (object literal, JSX prop, positional
   call), each with its counter-test.

### D-9 — The team of a player comes from the film, and only from the film

User decision, 2026-09-13. A player's team is in the state frame: the team designator component of
the ti=9 keyframe records (component i0, 4 bits, at a derived offset; value = designator + 1).

Designator > 0 is the team, published as is. Designator 0 is FFA: **no team**, published as "no
team" and not as an unknown. A silent film gives unknown, counted as unknown.

The database is a **control counter**, never a value and never a fallback: it feeds the coverage
counters under `coverage.teams` (read from the film, agreement, contradiction, silence). A
contradiction is counted and read by a reviewer, never silently corrected. This replaces `Team: -1` hard-coded in `film/replay/build.go`
and the comment in `document.go` claiming the team is not in the film — false, corrected in the
same lot together with `film/replay/flag_assign.go`.

Nothing on screen changes: the web colours players by `team_side` from the match sheet
(`lib/replay/rosterLogic.ts`), not by the artifact. What changes is that an offline cook — no
database at all — produces a complete roster with real teams.

### D-10 — Grammar decides everywhere; a fallback is named, counted, and retired

User decisions, 2026-09-13 (plan decisions D13 and D14). D-3 applies beyond `filmdec`: in the
facts layer too, a fact the film *writes* (a named event, a creation record, a state component,
a `chunk_00` table, the footer) is read from what the film writes. A production heuristic — a
time window, a distance threshold, a majority vote, a statistical calibration, an inference over
the death thread — never decides such a fact first. The wall is the worked example: the
`EquipmentSpawnedObject` event names 216 of 216 wall panels, while the pose origin was decided
by a 200 ms window.

A heuristic that survives does so as a **fallback**, under four rules:

1. **Named.** Every fallback lives in one registry in the code (name, fact, typed trigger, date
   posted, retirement criterion), and the code that runs it names it. A fallback outside the
   registry reddens a ratchet (`archlint/no_unregistered_fallback_test.go`, to be added at
   step 1.9.0). No anonymous "else" deciding a fact in the middle of a function.
2. **Ordered.** Read first, fall back second. A fallback fires only on a typed diagnostic
   "the film is silent here", never on a disagreement with the reading and never while the
   reading is available. If it fires where the reading existed, that is a counted
   contradiction, not a fallback.
3. **Counted.** Each fact publishes `coverage.<fact>.{grammar, fallback, contradiction}`; the
   artifact says which part of itself came from a fallback.
4. **Retired.** A fallback carries a retirement criterion (CLAUDE.md rule 11: date posted,
   target removal, measurable criterion). A fallback whose count is 0 on the corpus gate at a
   milestone closure is **deleted** at the next milestone, tests included. Nothing is kept "in
   case": a stale fallback that fires wrongly corrupts a fact the reading would have got right.

Before a fact is declared "not in the film", the measured negative (report, note, figure) is
cited; otherwise the question is *open*, which is a research item, not a licence for a heuristic.
The inventory of today's heuristics is the plan's lot 0.E; conversions are its family 1.9.

#### D-10 bis — The fallback registry (lot 1.9.0, 2026-09-14)

The registry named in rule 1 above exists: package
`internal/games/halo_infinite/film/replay/fallback`. It is a **leaf** (no repository imports), so
step 5 of milestone M2 moves it to `film/facts/fallback` by a pure move.

**What an entry carries.** `Nom` (stable id, `repli_<fact>_<mechanism>`, published in artifacts
and never renamed), `Fait` (what is decided), `Mecanisme` (how, with its exact parameters),
`Condition` (typed: `film_muet`, `section_absente`, `lecture_non_portee`, `contradiction`,
`non_resolu`, `inconditionnel`), `Ordre` (`apres_lecture`, `sans_lecture`, or
`devant_la_lecture` — the last being rule 2's *violation*, recorded to be counted and removed,
never tolerated), one or more `Site{Fichier, Ancre}`, `DatePose`, `CibleRetrait`,
`CritereRetrait`, and `CompteurBranche` with `CibleComptage`.

**Why `CompteurBranche` exists.** Rule 4 deletes a fallback whose count is zero. Confusing "never
fired" with "never instrumented" would delete a live fallback, so an entry states whether its
counter is wired, and if not, which lot wires it. A zero is only a zero when the counter is
wired.

**Counting is per cooking, never per package.** `fallback.Compteur` is created by
`replay.BuildFromFilm` before the first scan (so scan-time and assembly-time fallbacks land in
the same count) and by `BuildFromPositions` when the caller supplies none. A nil counter is
valid and counts nothing, which is what makes instrumenting a site risk-free. A package-level
counter would mix two films decoded in parallel and would contradict D-5.

**How it is published.** `coverage.fallbacks[]` — a flat `{name, hits}` list, sorted by name,
carrying only the fallbacks that *fired* (schema 58). This is the shape rule 3 takes in
practice: fallbacks do not distribute over the existing layers (the default-axis-widths fallback
touches the whole decode, and most fallen-back facts have no coverage block of their own), a
flat list joins the registry by name alone, and no existing coverage block changes shape.
Per-fact `coverage.<fact>.{grammar, fallback, contradiction}` triples remain the right form for a
fact that already owns a coverage block, and each conversion lot may add one.

**The ratchet has two directions** (`archlint/no_unregistered_fallback_test.go`). Code -> registry:
every declared Go identifier in the decoder whose name carries `repli`/`Repli`/`fallback`/
`Fallback` at a camelCase word boundary must be covered by the registry. The boundary *is* the
convention: without it the scan would catch `replication`, `replique`, `replier`, `repliement` —
ordinary French words of the decoder's vocabulary. Registry -> code: every entry's site must
exist and still carry its anchor. That second direction makes rule 4 mechanical — when a
conversion lot removes a fallback, its anchor disappears, the ratchet reddens, and the entry must
leave the registry in the same commit. Anchors are literals, not line numbers: the lot 0.E audit
cited `file:line` references that had already drifted eight days later.

**Exemptions are two, dated and justified**, in `replisDeLEcrivainDuJeu`: identifiers that name a
fallback path of *the game's own writer* (grammar read from the executable, not a LevelUp
decision), plus the registry's own publication machinery. A LevelUp fallback never goes there; it
goes into the registry.

## Corrections to statements made elsewhere

Found on the tree during lot 0.B (2026-09-13), written here so the wrong sentence is not repeated:

1. **`writeArtifactBytes` does not refuse a version downgrade.** The target architecture said "the
   single write point refuses any downgrade". On the tree it refuses only *impoverishment at equal
   schema* (`wouldDowngrade`); at a different schema it deliberately stays silent. The version
   refusal lives upstream, in `validateArtifact` (`replaybuild/artifact_store.go`), on the only
   writer that can carry another version (`StoreArtifact`) — the others serialize a document from
   the current producer. The behaviour is correct; only the sentence was wrong.
2. **The chronicle has three header forms, and one number does not exist.** `document_chronicle.go`
   carries three forms, born in different months (`// v<N> (`, `// SCHEMA <N> `,
   `// CE QUE LA VERSION <N> `). Version **32 was skipped** at the renumbering of two parallel
   lots, and v1 predates the chronicle. Any guard rail deriving the list from a `1..N` range would
   assert versions that were never cooked; the shared extractor
   (`testutil/replay_chronicle.go`) recognizes the three forms and fills no gaps.
   **Corrected on 2026-09-14 (lot 1.0, review round 2).** This point, lot 0.B and
   `testutil/replay_chronicle.go` all said "32 **and 51** were skipped". Git contradicts it for
   51: `2fb53db4e` sets `SchemaVersion = 51` on 2026-09-10 and `b6b198baf` replaces it with 52
   the next day, so artifacts *were* cooked under that number. What was actually missing was its
   chronicle entry — never written, and therefore invisible to the extractor and to every guard
   rail derived from it. The entry was restored on 2026-09-14. The lesson belongs here: a gap in
   the chronicle reads exactly like a skipped number, and nothing distinguishes the two. A
   chronicle entry is written **in the commit that raises the version**, never afterwards.
3. **`internal/domain/replaydoc` has no tests of its own.** The served document package is a leaf
   with no `_test.go` file. Its shape is guarded from `film/replay` (the shape fingerprint, D-8.4)
   and its field-by-field parity from `service/replayview/parity_test.go`. That is enough while
   both guards exist, and it is why neither may move without moving its subject.
4. **`weaponv3` does not live under `games/halo_infinite/`.** Added 2026-09-17, measured while
   preparing step 2.6. The plan (item 2.4.2) and the briefs derived from it name
   `internal/games/halo_infinite/weaponv3/`; that directory does not exist. The package is
   `internal/analysis/weaponv3/` (`bits_word.go`, `canon.go`, `pi_resolver.go`, `timing.go`).
   The location is not a detail: it sits under `analysis/`, where D-1 forbids importing a title
   package, so the gate of D-2 cannot simply be imported from there — which is why its own bit
   reader (`pi_resolver.go`) and its divergent copy of `wordBitsAt` are a perimeter question of
   step 2.4, not a licence to leave them outside the single gate to the bytes.

## Non-goals

- **Not a rewrite.** Every grammar acquired by reverse engineering stays as it is; only its
  parameters and its location move.
- **Not a rendering change.** The web is touched at the normalization boundary only; no card, no
  colour, no chart changes, and the BTB compact density stays `mode_category === 'BTB'`.
- **Not a research dump.** Open reverse-engineering questions (the semantics of the per-type table
  values, the unidentified footer bytes, the 48-bit token, the displayed label of a team
  designator) stay out, each with its resume condition in `.ai/V7.5/REGISTRE_REPORTS.md`.
- **Not a port of the neighbouring efforts** (vehicles, assault, duels): they have their own plans.
- **No layer and no ratchet without a consumer** at the step that introduces it.

## Consequences

Gains: two films decode in parallel, and a grammar can be tested against a synthetic profile with
no film; a new build is a data entry rather than a code change, and an unrecognized one is loud
instead of wrong; structural work is provable at zero difference, so the effort can stop cleanly
after any step; a publication change re-cooks in seconds instead of re-decoding the fleet.

Costs and risks, each with its parry. The layer split needs a window with no other decoder branch
in flight, and its ratchets are posted before the first `git mv`. The profile can become a
catch-all: one entry per family, each with its dated proof, reviewed family by family. Passing the
profile can cost time in hot loops: a benchmark baseline is compared at each closure, and the
profile is read at the head of a scan, never inside the bit loop. A source-level ratchet stays a
backstop for review, not a semantic proof.

## References

- `.ai/PLAN_DECODEUR_FILM_2026-09-13.md` — the lots, gates and dates this ADR abstracts from.
- `.ai/ARCHITECTURE_CIBLE_DECODEUR_FILM_2026-09-12.md`, `.ai/HANDOFF_DECODEUR_FILM_2026-09-13.md`
  — the dated survey and the proofs behind D-3, D-4 and D-9. Chronicle, not source: the code is.
- Every package, test and ratchet named above, cited inline where it applies.
