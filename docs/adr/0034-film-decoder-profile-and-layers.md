# ADR 0034 — Film decoder: an immutable profile per build, five layers, one gate to the bytes

**Status**: Accepted (2026-09-13). Amended on **2026-09-26** by the audit follow-up (revision model
and facts freshness: D-3 rule 2, D-6, D-7; one scan stage for the identity bridge and the facade
over the bytes: D-1, D-2). The state reached at the M2, M3 and M4 closures of
`.ai/V7.5/PLAN_DECODEUR_FILM_2026-09-13.md` (2026-09-17 and 2026-09-18), and the corrections found
on the tree during that effort, moved on 2026-09-30 to
[the history annex](0034-annex-history.md). This file keeps the decisions, their amendments and a
short current state.

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

## Current state (2026-09-30)

Measured on the tree (`feat/suite-audit-decodeur`, base `8cd560673`). Go paths are relative to
`apps/go-api/internal/`. Where a decision above reads differently from the tree, the tree wins and
the difference is written here; the history of how each decision was reached is in
[the annex](0034-annex-history.md).

- **D-1, five layers.** `film/internal/{source,profile,grammar,facts}` and `film/replay`; the
  `internal/` directory makes the compiler refuse any import from outside `film/`. Outside `film/`
  the decoder is reached through the facade `film/decfilm`, whose surface is frozen by
  `archlint/film_facade_surface_test.go` (186 symbols; 277 `replay.X` identifiers cited outside
  `film/`); reducing it was not retained (decision V25, 2026-09-18). **"`replay` decodes nothing"
  holds**: no production file of `film/replay` creates a bit reader or reads a bit; its film reads
  go through scan stages of `grammar`.
- **D-2, one gate to the bytes.** Holds, with the correction written in the 2026-09-26 (bis)
  amendment below: the facade still re-exports `Inflate`, only to hand the bytes back to a decoder
  entry point. Guard: `archlint/no_raw_film_bytes_outside_source_test.go`.
- **One scan stage.** The identity bridge is one stage, `grammar.ScanPontDIdentite`, called by both
  the cook (`film/replay`) and the collector (`sync/killcollector/positions.go`); neither calls one
  of its reads itself (`archlint/film_pont_identite_test.go`).
- **D-4, an unknown build.** Typed error, per-build expvar counter, and the film is set aside by
  both orchestrators (`sync/killcollector`, `replaybuild`). `killsource` also refuses a film
  without its map (`ErrCarteAbsente`) before any read.
- **D-5, no package-level mutable state.** `archlint/filmdec_package_vars_test.go` measures zero
  written package variables; the package decode lock is gone and may not return
  (`archlint/decode_lock_interdit_test.go`). The only remaining "one decode at a time" bound is the
  inter-process, memory-driven `filmproc.AcquireSolo`, taken by the entry points (`cmd/*`,
  `replaychild`) and never by `replaybuild` or by the HTTP layer (`archlint/no_decode_in_api_test.go`).
  **"Proven under `go test -race`" is held by the CI job `film-race`** (J12.6, 2026-09-30), which runs
  `TestDeuxFilmsEnParallele` (`film/internal/grammar/deux_films_parallele_test.go`) under `-race` on
  the `grammar` package alone (`-race` is incompatible with DuckDB, hence the narrow target).
- **D-6, revisions.** One revision per layer and per facts consumer: `source-2026-09-16.2`,
  `profile-2026-09-17.3`, `grammar-2026-09-27.3`, `killsource-2026-09-27`,
  `objectives-2026-09-27`; fingerprints ignore ordinary comments and layout (lot J3.1). The replay
  document is at `SchemaVersion` 76, the facts codec at `VersionCodecFaits` 2.
- **D-7, facts and publication.** Facts are persisted per film and the publication replays from
  them when they are fresh, decodes otherwise.
- **D-10, fallbacks.** The registry (`film/internal/facts/fallback`) holds 119 entries; every one is
  counted at its site or in data, except the one explicit out-of-production tool entry
  (`TestChaqueRepliEstCompte`, zero uncounted entries since lot J8.7-bis, 2026-09-28). The earlier
  "Reached as written" (M2 closure) did not hold: the audit of 2026-09-24 still found 81 of the
  99 entries without a wired counter.

## Amendment of 2026-09-26 — the revision model and the freshness of facts (audit follow-up, J3)

Source: `.ai/AUDIT_DECODEUR_FILM_2026-09-24.md` (findings SRC-1, RA1-1, RA1-2, RA1-4 and
architecture weakness 1), decisions DU-2 and DU-9 of
`.ai/V7.5/PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25.md`, milestone J3. No decode revision rose:
`source.Rev`, `profile.Rev`, `grammar.Rev` keep their values and their fingerprints were re-frozen
at constant revision (tooling); `SchemaVersion` went **71 -> 72** and the facts codec
(`VersionCodecFaits`) **1 -> 2**, so every persisted facts file is refused on its prefix and
re-decoded once.

### D-6 — One revision per thing that can change. **Amended: what a fingerprint hashes, and how many revisions the facts carry.**

1. **The fingerprint hashes tokens, not bytes** (DU-2 (a)). `film/revision` feeds each source
   through `go/scanner`: ordinary comments and layout are dropped, every language token counts
   (a string literal containing `//` or `/*` stays a literal), and `//go:` directives are kept
   because they change what the compiler does. The frame is unchanged — path relative to the
   root, content length. The audit measured 46 % of the layers' lines as comments: a reworded
   comment no longer reddens a gate, and no longer invites a "regenerate at constant revision"
   that silences it.
2. **A layer's perimeter is the closure of its imports, frozen by a golden** (DU-2 (b), fixes
   SRC-1 by construction). Starting from every package of the layer's tree, the production imports
   are followed inside the module; another revised layer stops the walk and enters by its
   **value**, any other package enters by its tokens and its own imports are followed. Files a
   package embeds (`//go:embed`) enter too — the tables of `film/damagetag` are data that decide
   the killsource output. The upstream values are no longer declared: they are what the closure
   meets, and a value supplied for a layer the closure does not meet (or the reverse) is an error.
   Each layer's perimeter is listed in `testdata/<layer>_perimetre.golden`
   (`revision.TestPerimetreDeChaqueCoucheEgaleSonGolden`), so an added import reddens a golden and
   the question "does this package decide the layer's output?" is asked at review. Measured
   closures: `source` = itself + `film/types`; `profile` = itself, no upstream (it does not import
   `source`, so `source.Rev` left its fingerprint); `grammar` = its tree + `film/types`,
   `domain/highlightevent`, `domain/playerposition`, `games/weapons/filmshell`, upstream `source`,
   `profile`; `killsource` = itself + `film/types`, `film/damagetag`, `domain/highlightevent`,
   upstream `source`, `profile`, `grammar`; `objectives` = itself + `film/types`, `domain/objectiveevent`,
   upstream `source` (J3.3b moved `ObjectiveEvent` out of `internal/domain` into the leaf
   `domain/objectiveevent`, so `domain`, `domain/title` and `games/canonical` left the closure).
3. **One revision per consumer of facts** (DU-2 (c)). `facts.Rev`, which dated the whole `facts/`
   tree, is replaced by `killsource.Rev` — **same value**, `killsource-2026-09-24`, same series
   and history, so no killsource backlog is reopened — and `objectives.Rev`, born at
   `objectives-2026-09-26`. A correction to objectives no longer makes every kill row in the
   database a backlog candidate. `decfilm.Rev` now points at `killsource.Rev`; `sync/killcollector`
   writes and compares it in `decoder_rev` as before. `coverage.decoder` publishes
   `killsourceRev` and `objectivesRev` instead of `factsRev` (schema 72), and `layers` attributes
   each facts layer to its consumer: `roster`, `neutralDeaths`, `equipmentEpisodes` to killsource;
   `identity`, `objectives`, `scoreTimeline`, the flag, crown, skull and the four bomb layers to
   objectives. A layer that reads both carries the one that decodes its substance; the re-cook
   verdict stays sound because `identity` and `roster`, never guarded, declare both families on
   every assembled document (`TestChaqueConsommateurDeFaitsDateUnCalqueNonGarde`).
   `facts/fallback` is imported by neither consumer and is hashed by no layer: its changes are
   publication changes.

### D-7 — Facts and publication are separate. **Amended: the facts carry what the caller commanded, and the whole catalog entry.**

1. **The caller's guards are written in the facts and compared** (DU-2 (d), RA1-1, P1). Three scan
   channels are read only when the caller commands them — the flag markers and return gauge (CTF),
   the zone states (a zone catalog), the arming ring (bomb family with a manifest clock) — and the
   player-index table is read against the caller's roster. `replay.GardesDe(Options)` derives these
   guards in one place, the scan uses the same predicates to decide what it reads, and the facts
   header stores them (flag, zones, bomb, and a fingerprint of the roster's xuid set).
   `FilmFactsEntete.Utilisable(entry, guards)` refuses facts baked without a guard the current
   cooking asks for, or under another roster (`ErrFilmFactsGardes`); a superset serves, and
   `BuildFromFacts` first removes from the inputs what the current cooking does not ask for, so
   the replayed document is the one a decode under those guards would give. Freshness is judged
   in two halves: `Frais(entry)` (codec, schema, layer revisions, cooking key) at the switch, before
   any film is opened; the guards once the options exist, in `replaybuild.documentDeLaCuisson`,
   which loads the film when they do not cover. **No facts are written for a refused artifact**:
   `BuildBytes` stores the facts of a decoding cook after serialization, and only if the artifact
   sink would store the artifact (`refusParLePuits`: the same validation and anti-downgrade guard
   as `writeArtifactBytes`) — an impoverished cook no longer leaves poor facts behind for the next
   repair to replay.
2. **The cooking key is the fingerprint of the whole catalog entry** (DU-2 (e), RA1-4). The header
   carries `EmpreinteDeCle(entry)` — module, bounds by their exact bits, axis widths, region and
   effective region-index width — and the key check compares it (`ErrFilmFactsCarte`). A catalog
   entry corrected on its bounds, region or region-index width used to leave "fresh" facts whose
   quanta re-dequantize wrongly. A field added to `MapQuantEntry` reddens
   `TestCleDeCuisson_ChaqueChampDeLEntreeGouverne` until it enters the fingerprint.
3. **A nil inventory stays nil** (RA1-2). The inputs blob reads every list back as an empty slice,
   but `Options.Inventory == nil` means "unreadable" and guards the `inventory` layer and its
   coverage. Section 1 now carries a presence witness, and a film with an unreadable inventory
   replays from its facts to the same layer set and coverage as from its film.
4. **Every codec change raises its version, with a refusal test** (the rule the RA1 family asked
   for, and finding 8.1 of the plan). The header changes above are one rise, codec 2, refused on
   the prefix for codec 1 (`TestFaitsDuCodec1SontRefusesSurLePrefixe`). The header grew by the
   second consumer revision, 32 bytes of catalog fingerprint and 35 bytes of guards; the decision
   "decode or re-read" is still taken on the header alone.

### D-3, rule 2 — The grammar comes from the game. **Amended by DU-9: presumed widths by measurement.**

Rule 2 stays the default: where the writer is readable in the executable, the value is taken from
the writer. One case is admitted as a production path by measurement, and only this one: **a
fixed-size component that the decoder only skips** (its value is used by nothing). For it, a
width found by closure — the only width with which the packets close to the bit — is accepted on
three conditions: it is verified on the whole corpus of every build; it is recorded as
**presumed**, with its provenance (closure measurement, films, date) in `ecs_table.tsv` and in the
code; and it is listed by a frozen test on the model of `empreintesPresumeesGelees`, so that no
presumed width enters or leaves unseen. The writer (Ghidra) remains mandatory for components
whose value is used, for variable-size components (content-dependent fields, gates, counts), and
for any presumed width that stops closing on a new build. The rule is safe because a wrong width
shifts everything after it: packets stop closing at once, and the error is seen and counted, never
silent. It complements, and does not relax, the user's rule of 2026-09-21 on player states (Ghidra
for used values).

## Amendment of 2026-09-26 (bis) — one scan stage for the identity bridge (audit follow-up, J4)

Source: `.ai/V7.5/PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25.md`, decision DU-3 (option S, first half S1),
finding RA1-3 and architecture weaknesses 3 and 5. No decode revision rose: the moves are pure and
the fingerprints of `source`, `grammar`, `killsource` and `objectives` were re-frozen at constant
revision. `IsolationDecoderRev` (the killsource collector's isolation facts) rose to
`isolement-2026-09-26-pont-unique-exemptions`.

### D-1 — Five layers. **"`replay` decodes nothing" is now true of the identity-bridge reads.**

Five reading files lived in `film/replay` — the death feed (`deaths_source.go`), the player-index
table read in the replication chunks (`player_index.go`), the film clock origin (`origin.go`), the
film's own player table (`film_player_table.go`) and the keyframe inventory with its three rule
files (`inventory_*.go`, which carried a private bit reader). They went down to `grammar` (lot J4.2,
`dd0d4dfa3`); their result types went to `film/types` (`Death`, `PlayerIndexTable`,
`KeyframeInventory`, `SlotAmmo`, `KeyframeInventoryStats`), except `FilmPlayerTable`, which carries
a method and stays in `grammar` with its read. What those files did that is not a read stayed in
`replay`: the roster given to the index, the injectivity it demands, the origin of frame 0 and its
witness, the log and the expvar counter of an unknown build (D-4: the grammar names its counters,
the orchestrator wires them). The finalisation predicate (`ChunkTypeTempsForts`, `EstTempsForts`,
`ErrFilmNonFinalise`, `Finalise`) left `film/filmcache` for the leaf `film/finalise`: importing the
cache from `grammar` would have pulled `observability`, `ctxkeys` and the whole of
`internal/domain` into the grammar's revision perimeter.

What remains, named so it is not re-discovered: `replay/cle_du_film.go` and
`replay/mpp_format_inconnu.go` still fetch the `chunk_00` bytes (`grammar.FilmRegistryChunk`) and
hand them to `grammar.ReadFilmIdentity`. They read no bit themselves, but they are the last places
where the publication layer holds film bytes.

### "One scan stage". **Reached for the identity bridge (lot J4.3).**

The replay cook and the killsource collector (`sync/killcollector`, `buildPositionRows`) both build
the identity registry from the same six reads: translocator teleports, biped positions **with the
teleport exemptions** (decision D2 of the equipment plan), biped creations, the death feed, the
player-index table and the film clock origin. The collector used to copy the sequence, and the copy
had drifted: its positions were read without the exemptions. Both now call
`grammar.ScanPontDIdentite` (re-exported by the facade), which returns each read with its error and
decides nothing. The **policies** that legitimately differ stay with each caller: the roster given
to the index (`OptionsDuPont.RosterDesMorts` — death feed plus caller roster for the cook, the match
sheet for the collector), direction capture (in the base scan options the caller passes), the
injectivity of the index table, and which errors are fatal. Guard:
`archlint/film_pont_identite_test.go` (neither caller calls one of the six reads itself, the
collector calls no `Scan*` of the publication layer). The collector's only output change is the
exemptions: on a film without translocator events its reads are byte-identical to the old sequence
(`killcollector.TestPontDuCollecteur_SeuleLExemptionChange`).

### D-2 — One gate to the bytes. **Corrected: the facade still re-exports `Inflate`, and says why.**

The M2 section claimed the gate was held while the facade `film/decfilm` re-exported the canonical
bit reader (`LecteurSur`), the packet walker (`Paquets`) and the tolerant decompressor (`Inflate`).
Measured on 2026-09-26, outside `film/`:

- `LecteurSur` and `Paquets` had one consumer, the throwaway research tool
  `cmd/rdata_weapon_scan`. It moved to `film/research/cmd_rdata_weapon_scan` (build tag
  `research`, split into four files under the 500-line threshold, pure move) and reads the inner
  layers directly, like its neighbours. **The facade no longer re-exports the bit reader or the
  packet walker**, nor the six symbols only that tool used (`DecodeFrameRecords`, `FrameConfig`,
  `NewWorld`, `ProfilDeBalayageParDefaut`, `Registry`, `World`).
- `Inflate` keeps production consumers outside the decoder, and it **stays**:
  `sync/haloclient` and `cmd/levelup` (`backfill-medailles-feed`) decompress the cached `chunk_00`
  to hand it to `FilmMajorVersionFromHeader` / `HighlightProfileFromHeader`; `cmd/diag_weapons_v3`
  (an operational CLI that writes positions) decompresses chunks for
  `DecodeKeyframePositions`; three tests build fixtures with it. None of them reads a bit: they
  decompress with the source layer's single decompression contract and give the bytes back to a
  decoder parser.

So D-2 reads, from now on: **nobody outside `source` reads a bit of a film or walks its packets;
decompressing a chunk outside the decoder is allowed only through the re-exported
`Inflate`/`Decompresser`, and only to hand the bytes back to a decoder entry point.**

**Lot J4.6 (S2), same day: the seven hand-written bit readers are gone, and the gate is guarded by
shape, not by name.** `readBitsAt`, `PeekBits`, `kfReadBits`, `kfReadBitsLoop`, `kfBitAt`,
`invBitAt` and `invBits` were replaced by named edge conventions of the source layer
(`source.BitsStricts` — panics on both sides; `source.BitsBourres` — zero past the end, panics
before the start; `source.BitsTolerants` and `source.BitAt` — zero on both sides). Each old reader
is kept as a reference copy in `source/bits_conventions_test.go` and opposed to its convention,
value and panic alike, around every byte, word and buffer edge. The one documented difference is
out of reach: `kfBitAt` panicked on a negative position where `BitAt` returns 0, and its six
callers read at positions that are non-negative by construction. The DU-3 condition held: the
dedicated benchmark (`grammar.BenchmarkBalayageBitABit`, the production scans that called the
seven readers, on the contiguous killsource reel) measured a median paired difference of -0.9 %
(dispersion 1.3 points) over 17 alternated A/B pairs at high priority. No decode revision rose;
the `source` and `grammar` fingerprints were re-frozen at constant revision.

The raw-bytes ratchet (`archlint/no_raw_film_bytes_outside_source_test.go`) gained a sixth,
structural pattern (`archlint/no_raw_film_bytes_extraction_test.go`): any function in the watched
roots that addresses the byte of a bit position (`x[p>>3]`, `x[p/8]`, directly or through a local
index) or shifts a byte of a `[]byte` by a variable, non-multiple-of-eight amount is red, whatever
its name; the seven names are also listed as an anti-resurrection ratchet. **Correction to the
premise of S2:** the plan counted seven readers outside `source`; the structural pattern found nine
more functions in six files that nobody had listed — `grammar/frame_vue_controle.go`
(`vueCFermee`), `grammar/weaponscan/scanner.go` (`matchMarkerAt`, `readBitsUint64`,
`readBitsUint8`), `facts/killsource/botmeta.go` (`byteAtBit`), `research/cmd_rdata_weapon_scan`
(`bitsAt`), `cmd/diag_film` (`countMarkerBits`) and `sync/killcollector/shots.go`
(`chercherDansChunk`, `lireIndiceAvant`). The last two break D-2 in production code outside the
decoder, and `sync` cannot import the source layer. Porting them was not lot J4.6's scope: they are
recorded as dated exceptions, one line per function, each with its reason and its removal criterion
(the function no longer extracts a bit itself); an exception that stops matching turns the ratchet
red. D-2 therefore holds for the seven, and is **not yet true** for those nine functions.

The facade surface ratchet (`archlint/film_facade_surface_test.go`, decision V25) records every
step, dated: 166 → 170 (J4.2, the four bridge reads re-exported) → 173 (J4.3, the stage and its
two types) → 165 (J4.5, eight symbols without consumers removed). The companion surface
(`replay.X` cited outside `film/`) went 281 → 276 at J4.2.

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

- `.ai/V7.5/PLAN_DECODEUR_FILM_2026-09-13.md` — the lots, gates and dates this ADR abstracts from.
- `.ai/V7.5/ARCHITECTURE_CIBLE_DECODEUR_FILM_2026-09-12.md`, `.ai/V7.5/HANDOFF_DECODEUR_FILM_2026-09-13.md`
  — the dated survey and the proofs behind D-3, D-4 and D-9. Chronicle, not source: the code is.
- Every package, test and ratchet named above, cited inline where it applies.
