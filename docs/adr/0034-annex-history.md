# ADR 0034 — annex: history of the state reached (M2, M3, M4) and corrections

This annex holds, verbatim, the sections of [ADR 0034](0034-film-decoder-profile-and-layers.md)
that recorded the state of the tree at the closures of milestones M2, M3 and M4 of
`.ai/V7.5/PLAN_DECODEUR_FILM_2026-09-13.md`, and the corrections found on the tree during that
effort. They were moved out of the ADR on 2026-09-30 (J12.5 of
`.ai/V7.5/PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25.md`) so the ADR keeps its decisions and a short
current state. **This is history**: each section describes the tree on its date, not today. For
the current state, read the "Current state" section of the ADR.

One statement below was false when written and is corrected in the ADR's current state: the M2
section's "D-10 and D-10 bis ... Reached as written" (the audit of 2026-09-24 still found 81 of the 99
registry entries without a wired counter; every entry has been counted since 2026-09-28).

---

## State reached at M2 (2026-09-17)

Measured on the tree at the closure of milestone M2 of `.ai/V7.5/PLAN_DECODEUR_FILM_2026-09-13.md`
(base `92c83b333`). Go paths below are relative to `apps/go-api/internal/`. The plan holds the
lots, the gates and their numbers; this section holds only what the tree shows, decision by
decision, and where a decision above reads differently from the tree **the tree wins and the
difference is written here**.

The ADR carries ten decisions plus D-10 bis. The "principle 11" that several godocs cite
(`film/decfilm/decfilm.go`, `film/internal/facts/rev.go`) is principle 11 of the target
architecture note, not an eleventh decision of this ADR: its substance is the enforcement
paragraph of D-1, and it is now held literally.

### D-1 — Five layers, one dependency direction. **Reached, with three precisions.**

The four inner layers live under `games/halo_infinite/film/internal/`: `source` (6 production
files), `profile` (11), `grammar` (146, with the sub-packages `positions`, `weaponscan` and
`weaponv3`, which came down from `analysis/`), `facts` (54, with `killsource`, `objectives` and
`fallback`). Nothing outside `film/` can import them at all — the compiler, not a ratchet, and it
cannot be allowlisted.

1. **`film/replay` stays exported, and that is the written decision.** It is the publication
   layer, 239 of its symbols are cited outside the decoder (`sync/replayartifacts`,
   `sync/killcollector`, `service/*`, `api/*`, `replaybuild`, `ops`) and the replay document is
   the public contract. `film/types`, `film/revision`, `film/filmcache` and the three label
   catalogs (`damagetag`, `killicon`, `medalname`) stay exported for the same reason of nature:
   they declare shapes or name things, they decode nothing.
2. **The facade is `film/decfilm`, and it re-exports 163 symbols.** It carries, under one package
   name, the surface the outside consumers cited, which is what made the move under `internal/`
   feasible in one commit. The count is the point and it is written in its godoc: **a facade of
   163 symbols is an alias, not a boundary — the reduction is M4 material.** (Read at the M4
   closure: still an alias, 166 symbols, and the reduction is **not retained** — user decision
   V25 of 2026-09-18, a dated surface ratchet instead. See the M4 section.) Types cross as
   aliases, constants and variables as values, functions as one-line forwards whose signatures
   name the layer types, so a caller can circulate a value without being able to name it.
3. **`film/revision` sits outside `film/internal/` with no reason left.** It was there to stay
   importable by `sync/killcollector` while the facts revision lived in that package; the constant
   came down to `film/internal/facts/rev.go`, and the only importers left are the four layer gates,
   all under `film/`. The measure is written in the package header. Pure move, no fingerprint
   touched, not taken in M2.

**Enforcement.** `archlint/film_layers_deps_test.go` carries four axes. R1 (direction), R2 (place)
and R3 (population) are **strict with no table at all** — each tolerance mechanism was deleted
with its last entry, because an empty table that is kept invites being filled. R4 ("the
publication layer does not load the film", born with lot 2.5.e-d) carries a dated allowlist of
the four `ScanFilmXxx` D2 wrappers, whose retirement criterion is the one
`archlint/no_film_reread_test.go` already writes for that family.

`analysis/` imports no title package, in production **and** in test: the `franchissementsToleres`
table of `archlint/no_title_package_in_analysis_test.go` is **empty, and it is now a ratchet**. The
correction dated 2026-09-17 in the section below counted five entries; the five fell with lots
2.4.2, 2.5.d.2, 2.5.e-a and 2.5.f, each by the port it announced or by its package moving.

### D-2 — One gate to the bytes. **Reached, allowlist never opened.**

`film/internal/source` owns the canonical reader (`Lecteur` / `LecteurSur`) and its four named
edge conventions (`BitsAt`, `BitAt`, `BitsTolerants`, `BitsTronques`), the film integers, the
packet walker, the two decompression contracts and the 64-bit pattern scan. `killsource.evReader`
**and** `filmdec.BitReader` were both absorbed — the measure corrected the item: `BitReader` was
itself not canonical, and the two came down into the source layer. Equivalence was proven call by
call on the real bit positions of the event chains, then against a golden produced by the code of
the base, before the absorption.

`archlint/no_raw_film_bytes_outside_source_test.go` went from 78 tolerated pairs to nine to zero,
and **the allowlist mechanism was deleted with its last entry** (lot 2.5.e-a). What remains is
the list of *exclusions* — "these bytes are not a film's" — each dated and kept alive by its
own test.

### D-3 — The profile is immutable, resolved once, keyed by build. **Reached, key corrected.**

`film/internal/profile` is a **leaf**: `go list -deps` returns only itself. It carries the value
types, the table (`profile_table.go`, columns `Cle / Champ / Valeur / Source / Preuve / Date`), the
map catalog and the `Profile` type with its invariants; the **detection** that produces a profile
value stays in `grammar` and returns a `profile` type, so the dependency reads `grammar -> profile`
and never the reverse. That split is decision V19 (1) of the plan and it amends this decision: the
profile is **data**, and the lot that created the layer was an extraction with a dependency
inversion, not the pure move the ADR implied.

**The key is not the build alone.** The table is keyed by the **three keys the film writes** —
format version (`chunk_00+4`), build (section 2 in clear text), major version (`chunk_00+0`) — and
the gamertag layout is keyed by a fourth. Format and build do not say the same thing: format 24
carries three builds, two of them with different customization widths. This decision said "the
build, with the major version as fallback"; the tree says three keys, and there is a second typed
sentinel to match (`profile.ErrUnknownFormat` beside `profile.ErrUnknownBuild`).

The three binding rules hold: every row carries its provenance and its date, `TestProfilPresumes`
lists **and freezes** the six presumed entries, all of them movement, and removing one demands the
table row pass to *read* or *measured*.

**Reached at lot 3.1.1 (2026-09-17, `4bf62211f`).** The versioned catalog exists
(`data/titles/halo_infinite/reference/film_profiles.json`, written by `cmd/film-profiles-build`
under the `gamefiles` tag, read by `games/halo_infinite/filmprofile`), and the decoder does not read
it — that would be a layer importing a catalog package, which the one-way rule forbids. It **copies**
it instead: `profile/registre_empreintes.go` carries the nine keys of `registryFingerprints`, and
`filmprofile.TestCatalogueConformeALaTableDesEmpreintes` keeps the two equal value for value,
proofs included. The third registry status `presumee` is therefore delivered, and the three cases
live in one place, `profile.ClasserEmpreinteRegistre`.

### D-4 — An unknown build fails loudly. **Reached at lot 3.1.1 (2026-09-17, `f6b37a662`).**

Held since lot 1.5: the typed sentinels (`profile.ErrUnknownBuild`, `profile.ErrUnknownFormat`,
both wrapped with the refused key), the per-key expvar counters, the log line, and the rule that an
unknown key is **never** decoded with another film's profile.

Now held too: **the film is set aside.** Both orchestrators read one shared verdict
(`replay.CleDuFilm`) before any read — `sync/killcollector` before `decfilm.Decode`, `replaybuild`
right after loading the film and before the death feed — and stop there: no kill-source fact, no
positions, no shots, no artefact. Each names the refused key in one `WARN`, publishes the counters
`grammar` names, and reports a *status of its own pipeline*: the outcome `ecarte-cle-inconnue` for
the collector, `filmproc.CodeSkipped` for the cooking child. No registry marker is set, on purpose:
`MBitFilmAbsent` is terminal, while a set-aside film must come back the day the table learns its
key.

One measured subtlety, written here because it is the trap: the verdict is **not**
`Profile.Err() != nil`. That error is also raised, with an empty build, for the films whose
`chunk_00` carries no identification section — and the table knows those, under `majeure=31` /
`majeure=33`. Setting them aside would have dropped five cache films, two of them corpus-gate
witnesses. The verdict bears on the key the film *writes* (`profile.CleConnue`).

### D-5 — No package-level mutable state. **Reached; the binding criterion is "zero written".**

`LockProcessDecode` and `decode_gate.go` are deleted with their 373 call sites in 276 files;
`archlint/decode_lock_held_test.go` is gone, replaced by `decode_lock_interdit_test.go`, which
forbids taking such a lock at all and forbids the file coming back. Readers take their values from
the profile, carried by the bit reader or read at the head of a scan; the dated double-write
kill-switch was removed at its target lot, as written. `TestDeuxFilmsEnParallele` decodes two films
of different builds in two goroutines under `-race` and refuses two equal fingerprints, so it cannot
pass on a silent witness.

This decision announced `filmdec_package_vars_test.go` ratcheted "from 96 to 0 mutable variables".
On the tree the frozen count is **21** and the count that binds is the other one:
`TestAucunVarDePaquetEcriteDansFilmdec` walks the AST and refuses any assignment, increment or
mutable address-of on a package variable — **zero written**. The 21 survivors are tables and
sentinels the scan proves are never assigned.

The 29 observation hooks became fields of `grammar.Observation`, and the package-level observer
itself is gone with its 28 public setters. One exception is named and bounded: a **test** harness
(`grammar/harnais_observation_test.go`) keeps a package observer and its setters for the research
instruments; it lives in a `_test.go` file, outside the ratchet's perimeter, and its header says so.

### D-6 — One revision per thing that can change. **Reached and widened: four, not two.**

| Revision | Value at the closure | Hashes |
|---|---|---|
| `source.Rev` | `source-2026-09-16.2` | the source layer, no upstream value (it is the root of the one-way) |
| `profile.Rev` | `profile-2026-09-17` | the profile layer, plus the **value** of `source.Rev` |
| `grammar.Rev` | `grammar-2026-09-15.39` | the grammar layer, plus the **values** of `profile.Rev` and `source.Rev` |
| `facts.Rev` | `killsource-2026-09-16.6` | the whole facts tree, plus the **values** of `source.Rev` and `grammar.Rev` |

Each layer hashes its own non-test sources and the **values** of the layers it depends on, never
their bytes (plan decision V15 (12)): the lowest layer that rises raises every layer above it,
up to the backlog, and the chaining is proven by mutation in both directions. `GrammarRev`
became `grammar.Rev`; **`KillSourceDecoderRev` no longer exists** — `facts.Rev` took over its
value as is, so M2 opened no backlog, and the series keeps the `killsource-` prefix because the
rows in the database carry those very strings. The row of the table above naming `SchemaVersion`
at 54 reads **61** on the tree.

The calculation, the chronicle, the regeneration door and the failure messages are shared
(`film/revision`, lot 2.6.0, posted before the third copy);
`archlint/no_ad_hoc_source_fingerprint_test.go` forbids an ad hoc fingerprint elsewhere and its
allowlist is **empty**, which turns it into a ratchet. The backlog rule — a rise of `facts.Rev`
opens the killsource backlog, on user signal — lives in the failure message of
`TestFactsRevSuitLesFaits`, in the predicate `conditionBacklog`
(`sync/killcollector/postsync.go`) and, at the present tense, in `docs/SYNC_GUIDE.md` and its French
twin.

The rule that a structural step is judged at zero difference and a behaviour lot by the corpus gate
held for every lot of M2. At the closure, the corpus gate returned **17 witnesses out of 17 at zero
loss and zero change** (schema 60 to 61) and `replay-equiv` returned, on the 20 films of the corpus,
a single divergent step out of 53 — the publication step, its difference limited to the two
telemetry blocks 2.6.3 adds and proven field by field.

### D-7 — Facts and publication are separate. **Not reached, by design: it is M4.**

`data/cache/film_facts/` does not exist, publication still re-decodes, and the two published twins
are still kept in step by the shape fingerprint and by `service/replayview/parity_test.go`. What M2
delivers is the producer side of the rule "the presence of a layer is read in its revision, never in
the absence of a field": the artifact carries `coverage.decoder.{sourceRev, profileRev, grammarRev,
factsRev, build}` plus the `registry` sub-block, at schema 61. Lot 4.3 (one published type) is
deferred past M4 by a user decision of 2026-09-17, in a clean stop.

### D-8 — The Go / web contract. **Unchanged by M2, and its numbers moved as prescribed.**

`MIN_RENDERABLE_SCHEMA_VERSION` is still 27, still pinned by test; the v61 chronicle entry was
written **in the commit that raised the version**, as correction 2 below demands; the shape golden
was regenerated by its single door; `api/openapi.yaml` and the web types were regenerated last. The
contract types of the decoder gained a golden of their own —
`film/types/testdata/shapes.golden`, 46 sections, carrying on its first data line the revisions of
the three producing layers — so a form that moves without its revision reddens to the field.

### D-9 — The team of a player comes from the film. **Reached at M1, unchanged by M2.**

The designator is read from the state frame, FFA publishes "no team", a silent film counts as
unknown, and the database stays a control counter under `coverage.teams`.

### D-10 and D-10 bis — Grammar decides; a fallback is named and counted. **Reached as written.**

The registry moved to the facts layer by a pure move, exactly as D-10 bis announced; its path is
`film/internal/facts/fallback` — one segment more than the ADR wrote, the `internal/` the layers
gained at lot 2.5.e-c. It carries **98 entries** across six files, sorted by the fact they decide,
each with its typed trigger, its order, its sites and anchors, its date posted, its target and its
retirement criterion. `coverage.fallbacks[]` publishes the ones that fired, per cooking;
`archlint/no_unregistered_fallback_test.go` still binds in both directions, with its two dated
exemptions for the game writer's own fallbacks.

### What M2 leaves partial, named here so it is not re-discovered

- ~~the registry status `presumee`~~ — **closed at lot 3.1.1** (2026-09-17): the profile table
  copies the catalog fingerprints, and the three cases are decided in one place;
- the facade's surface — 163 symbols, an alias rather than a boundary, reduction measured and
  referred to M4 (**where it was not retained**: V25, ratchet instead — M4 section);
- `film/revision` outside `film/internal/` — a pure move nobody's fingerprint sees;
- ~~a film with an unknown key is not set aside~~ — **closed at lot 3.1.1** (D-4 above,
  correction 5 below);
- the persisted facts and the single published type — M4 and past-M4 by decision.

## State reached at M3 (2026-09-17)

Measured on the tree at the closure of milestone M3 of `.ai/V7.5/PLAN_DECODEUR_FILM_2026-09-13.md`
(base `fe38d2e35`, the integration carrying lots 3.4, 3.6.a and 3.1.1 — the code of M3 is
complete). Go paths are relative to `apps/go-api/internal/`. Same rule as the M2 section above:
the plan holds the lots, the gates and their numbers; this section holds what the tree shows,
decision by decision, and **where a decision reads differently from the tree the tree wins and
the difference is written here**.

M3 neither added nor removed a decision. It *spent* the profile: what D-3 declared a table of
data, three families of measurement actually entered; what D-5 forbade at package level, M3
extended to the scans themselves. The lots and their merge shas: 3.1 data (3.1.2, 3.1.3)
`5cb7e558e`, 3.5 instruction and 3.6 preparation `4feea2ea6`, 3.2 data and witnesses
`f84a9fbf8`, 3.4 Ghidra preparation `6a360201b`, 3.3 research `8a62e7213`, 3.3 production
(3.3.1, 3.3.2, 3.3.3) `492cb0923`, 3.4 (3.4.1, 3.4.2) `accba88ac` with its CI fix `5897eb3dd`,
3.6.a `2b68aa165`, 3.1.1 `fe38d2e35`. Lot 3.7 is research only (`feat/decfilm-37r`), no
production file, so it moves nothing here.

### D-3 — The profile is immutable, resolved once, keyed by build. **Spent.**

The table grew from **23 rows to 33** (`profile/profile_table.go`, counted on the tree: **7**
read at the writer, **21** measured on a witness, **5** presumed) and it gained a second table
beside it, the nine registry fingerprints (`profile/registre_empreintes.go`). The versioned
catalog holds the same two shapes, 33 `entries` and 9 `registryFingerprints`, and
`filmprofile.TestCatalogueConformeALaTableDuLot21` plus
`TestCatalogueConformeALaTableDesEmpreintes` keep table and catalog equal value for value,
proofs included. The leaf property holds: `go list -deps` on `profile` still returns only
itself, and the decoder still does not *read* the catalog — it copies it.

1. **The law of the axis widths, read at the writer, replaced a guessed uniform** (lot 3.4.1).
   `profile/loi_largeurs.go` transcribes `FUN_140be9b88` / `FUN_140be9c78` with its two guards,
   the `2^22` bin ceiling and the `1e-4` step epsilon, both now modelled on the other side too
   (`himap/sbsp.go`: `maxBinCount`, `quantStepEpsilon`); `TestLaLoiRendLesLargeursDuCatalogue`
   demands agreement on **79 maps out of 79**. The row `Movement.LoiLargeursAxe` is *read*; the
   row `Movement.AbsoluteAxisW` — a uniform 14, guessed — **disappeared**, and the widths now
   follow the range index: the default table of the build when the index is `-1`, the map's
   per-index table otherwise. Measured consequence, not predicted: on Cliffhanger the biped's
   absolute path read 49 bits and reads **47**.
2. **The grenade preamble is a profile row per key, and it carries a VALUE, not only a width**
   (lot 3.3.1). Nine rows, one per written key — seven builds plus the two major versions with
   no identification section. The research lot had concluded "one bit" (23-bit preamble
   `0x260600` up to `HI_1_11_0`, 24-bit `0x4C0C00` from `HI_1_12_0`); the production lot
   measured that major 31 carries a **different value at equal width** (`0x20400`, 95 hits out
   of 95), so deriving the old preamble from the recent one by a shift would have worked on
   eight keys out of nine. The whitelist `GrenadeTypeIDsByRank` stays a **constant of the
   title**: the pattern is derived from the projectile `ti` resolved **by name** in the film's
   own registry, and no rank is ever guessed.
3. **The registry fingerprints are copied into the profile, never read from the catalog**
   (lot 3.1.1, already stated under D-3 above). Nine keys, five distinct fingerprints — three
   pairs of builds share theirs — which is why uniqueness is validated on the **key** and why an
   equal fingerprint authorizes no conclusion about structure sizes.
4. **What stays presumed is five rows, and they are frozen.** `TestProfilPresumes` listed six at
   M2 and lists **five** at this closure: `Movement.Traversal`, `Movement.DeltaAxisWidth`,
   `Movement.Range`, `Movement.CalibratedSkip`, `Movement.MobilityActionExtraBits`. The ratchet
   never rises; the entry that left did so by *disappearing*, not by being promoted, and the
   plan says so at the line. `Movement.Traversal` deliberately stayed presumed: it is the delta
   path's descriptor, a different quantity from the absolute width the Ghidra note actually
   read, and declaring it read on the note's word would have been a false provenance.

### D-4 — An unknown build fails loudly. **Reached at lot 3.1.1, and exercised the same day.**

D-4 above says what the two orchestrators do. What M3 adds is that the case is **no longer
hypothetical**. The park census (`TestRecensementDesClefsDuParc`, `chunk_00` only, no decode,
2.8 s) counted **1 589 films and ten written keys**, where the user's decision of 2026-09-17 had
been taken on "zero films today" (a census of 1 351 films three days earlier). One film,
`58e6f72a`, writes `build=HI_1_5_1` — a **sixth** registry fingerprint, absent from the catalog,
the only 1 034-named-slot registry of the park. It is set aside from that day: **one film out of
1 589, 0.06 %**, and no gate suffers (the film is neither a corpus-gate witness nor an
equivalence film). Two bounded consequences are written rather than hidden: the film stays a
catch-up candidate and costs one download per cycle until the key is added — which is deliberate,
because `MBitFilmAbsent` is terminal — and adding the key is the runbook procedure with measured
values on a witness, that is a lot and not a line.

### D-5 — No package-level mutable state. **Extended to the scans themselves.**

M2 removed the mutable state. M3 met the next form of the same defect: a *scan* that writes the
value a reader will use, with a criterion blind to the quantity it decides. Two cases, both
measured, and the plan holds the outputs.

- **The handle word (`Traversal.IndexW`) no longer comes from a scan's tie-break.** The lot
  3.4.1 gates found three steps moving that nothing expected (`abilityImpulses` on 7 films,
  `grappleReads.stats` on 9, `pads` on 3) and four published reads lost. The cause was not the
  map: the deciding criterion scored its candidates under a **uniform** axis width the
  production had just stopped reading, and at the triplet actually read the scores are flat —
  `272 / 272 / 272` on one witness, `61 / 61 / 61` on another — so the published value came out
  of a tie broken by an unstable `sort.Slice`. Lot 3.4.2 splits the measurement in two: an
  **axis oracle** that sweeps at a frozen handle word and **writes nothing**, and a handle-word
  **decision** scored at the read triplet, kept only if it dominates the median by a factor;
  otherwise the invariant 1 under a named, dated fallback. Both sorts became deterministic,
  `motDePoigneeRetenu` is tested exhaustively **without a film**, and a source ratchet demands a
  single write of `Traversal.IndexW` in the package.
- **`param_4` still decides, and it is now named.** The record-state parameter is still inferred
  by a scan, and the lot says so instead of implying otherwise: the fallback
  `repli_parametre_etat_record_infere` entered the registry with its target and its retirement
  criterion. The remedy was **coded and measured, then set aside**: giving `param_4` the
  handle-word discrimination test drops it to the invariant everywhere and **loses published
  data elsewhere** (`grenadeReads/n` 193 to 191 on the negative witness,
  `weaponChanges/par-kind/taken` 67 to 65 and 36 to 32, `coverage.abilities.published` 187 to
  186) to recover two matched deaths out of four. One does not replace noise with a loss; the
  candidate is written, and it is not retained.
- **The demoted inference became an oracle, and it earns its keep.** `killsource/calibrate.go`
  counts disagreements instead of deciding: on a film whose widths are not the invariant the
  oracle **confirms the map and contradicts the fallback** (axis disagreement 1 without the map,
  0 with it). The map of the match now reaches `killsource` itself — it was the only decoding
  path in the repository receiving no catalog entry — and its absence is a fallback that is
  **named, counted and warned per film**.

### D-6 — One revision per thing that can change. **Three of four raised, no schema bump.**

| Revision | Value at the M3 closure | Moved by |
|---|---|---|
| `source.Rev` | `source-2026-09-16.2` | unchanged by M3 — no lot touched the way the bytes are reached |
| `profile.Rev` | `profile-2026-09-17.3` | 3.3.1 (grenade preamble rows), 3.4.1 (the width law) |
| `grammar.Rev` | `grammar-2026-09-15.42` | 3.3.1, 3.4.1, 3.6.a (the two `ti=9` components) |
| `facts.Rev` | `killsource-2026-09-17.2` | 3.4.1 (the map reaches the death walk), 3.4.2 (the handle-word fix) |

**`SchemaVersion` stays 61 across the whole milestone.** Three lots changed the cooked content
and none of them added a field: the grenade throws travel in a block that already existed, the
death walk changes which rows are credited and not their shape, and the registry status was
already a string, so the third state `presumee` obliged no consumer to change form. That is also
why the coverage counters M3's scans now emit are **logged and not published**: publishing them
would raise the schema, and a schema rise marks the whole park for a re-cook — so they wait for
M4's single rise 61 to 62, by a pilot decision written at the lot.

The consequence of `facts.Rev` rising is the rule of this decision, and it is in force: the rows
of `match_kill_events` already written became backlog candidates. The user deferred the
deliberate drain — "it is a very long step, so we do it as late as possible" — to after M4 at
the earliest, on an explicit signal, grouped with any later rise so the base is passed over
once.
The bounded post-sync catch-up is unchanged and needs no decision.

The chronicle door held under load: at the 3.3 merge `grammar/rev_chronique.go` stood at exactly
**500 lines**, its ceiling, and the next entry would not have fit; lot 3.4 moved old ranks into
`rev_chronique_archive.go` and `fichiersDeChroniqueGrammar` now names both files (451 and 456
lines on the tree). The corpus gate learned the matching lesson at lot 3.3.3: the six telemetry
leaves — `coverage.decoder.{sourceRev, profileRev, grammarRev, factsRev}`, `build` and
`registry.fingerprint` — are shown in their own section and **counted nowhere**, because a leaf
that says which decoder cooked the artifact is not a datum of the match; and a rejection counter
declared with its denominator is a **loss** only if the ratio degrades.

### D-10 and D-10 bis — A fallback is named, counted and retired. **Three retired, four posted.**

The registry carries **99 entries across seven files** on the tree (98 across six at M2), the
seventh slice being the calibration family that lot 3.4 split out. M3 is the first milestone
where the count moved in both directions, which is what the decision asks for.

- **Retired, because the value is now read**: `repli_largeur_absolue_uniforme` (the uniform
  absolute width, replaced by the map's per-index table), `repli_calibration_paquet_exclu` and
  `repli_calibration_paquet_non_localise` (the calibration no longer decides).
- **Posted, each with its target and a measurable retirement criterion**:
  `repli_parametre_etat_record_infere` (`param_4` still decides — D-5 above),
  `repli_carte_absente_largeurs_par_defaut` (the match's map is missing; counted and warned per
  film), `repli_largeur_mot_de_poignee_inferee` (the invariant when the scan cannot
  discriminate; target = the width read at the writer),
  `repli_amorce_grenade_profil_de_reference` (a key absent from the preamble table).

The third retirement is the one to keep in mind: removing the two calibration fallbacks without
looking at what remained would have erased an inference that still decides. A fallback registry
is only worth its accuracy.

### What M3 measured, in four numbers

The plan's §5 holds the commands and the pasted outputs; these are the four the decisions above
rest on.

- **Credited deaths, six old witnesses: 106 to 737** (17-27 % of the raw couples to 64-89 %),
  measured with the same instrument on both sides, once the match's map reaches `killsource`.
- **Grenade throws on the old keys: 1 627** on the eight witnesses of the corpus where the
  artifact published zero (2 176 over the nine old-grammar films of the equivalence corpus), the
  counts agreeing to the unit between the production path and the equivalence step.
- **`ti=9` closes 1 679 / 1 679** on the six research films, from `0 / 1 679`. On the seven
  committed mini-reels it closes 1 716 / 1 717, and the remainder is a chance anchor of the
  sweeper rather than a width — which is why the honest target of a port lot is "no named
  blocker", not "100 %".
- **Corpus gate, 17 witnesses out of 17 present at every pass**, schema 61 to 61. Lot 3.6.a
  returned 0 gain / 0 loss / 0 change; lot 3.1.1 returned 0 loss / 0 gain and **6 witnesses
  changing one leaf each, the same one** (`coverage.decoder.registry.status`, `inconnue` to
  `connue`), which is the change it produces by construction. Lot 3.4 is the one that did not
  come back clean and it is written as such: **8 witnesses in loss**, and the attribution was
  made by measurement, not by reasoning — the same gate against the merge just before the
  handle-word fix returns 8 out of 8 `ok`, so the cause is 3.4.1 and the loss is vehicle deaths
  read under the map's widths.

### What M3 leaves partial, named here so it is not re-discovered

- **Two published lines lost, assumed and written**: `vehicles.tEnd/presents` and
  `vehicles/par-end/destroyed`, on two witnesses, from the map's widths reaching the `ti=40`
  dead-state walk. The cause is established (`param_4` on the films with no region, the i0
  cutting on Live Fire — where the catalog and the film disagree while **both read 41 bits**, so
  no length measurement can separate them), the remedy is measured and costlier than the defect,
  and the candidate lot is written rather than started.
- **The coverage counters of M3's scans** are logged and not published; they enter M4's single
  schema rise 61 to 62, to be re-measured there against what the lots actually added. (Read at
  the M4 closure: **one family of the three entered**, the deaths path; the other two wait for the
  next `facts.Rev` signal — M4 section, D-6.)
- **The facade's surface grew instead of shrinking**: **166** exported top-level declarations in
  `film/decfilm/decfilm.go` (163 at M2), and **245** distinct `replay.X` identifiers cited
  outside `film/` (the M2 section wrote 239, and the reproducible measure of that same set gave
  242 then). The reduction stays M4 material; what M4 adds first is a dated ratchet on both
  numbers, because nothing counts them today. (Read at the M4 closure: the ratchet was added in
  lot 4.1.1-a, the facade stayed at **166** and the companion reached **257** — and the reduction
  itself is **not retained**, V25.)
- **`film/revision` outside `film/internal/`** — unchanged, a pure move nobody's fingerprint
  sees, and `decfilm.Rev` still carries a name from the time there was one decoder revision
  rather than four. Both are M4 material, and neither is a defect of value.
- **No component port beyond `ti=9`.** The user, with the inventory in hand, decided on
  2026-09-17 that no unported device, navpoint, vehicle or other archetype component has a
  product use: the remaining cases of lot 3.6 are closed unretained, and "prepared without
  porting" is what already exists and is held by ratchet — the ECS table naming every component
  with its writer's address and its grammar where it was read, the closure golden naming each
  archetype's blocker, and the `default` of every dispatch link counting the unknown.
- **The persisted facts and the single published type** — M4 and past-M4 by decision,
  unchanged.

## State reached at M4 (2026-09-18)

Measured on the tree at the closure of milestone M4 of `.ai/V7.5/PLAN_DECODEUR_FILM_2026-09-13.md`
(base `896a9ce04`, the integration carrying lots 4.1, 4.2, 4.4 with the equivalence references
re-frozen — the code of M4 is complete, and M4 is the last milestone of the effort). Go paths are
relative to `apps/go-api/internal/`. Same rule as the two sections above: the plan holds the lots,
the gates and their numbers; this section holds what the tree shows, decision by decision, and
**where a decision reads differently from the tree the tree wins and the difference is written
here**.

M4 built the publication path and closed no decision it had not opened. The lots and their merge
shas: 4.2 (4.2.1, 4.2.2) and 4.4.2 `73a4dc580`, 4.1 (4.1.1, 4.1.2, 4.1.3) `f9ba456b2`, the
equivalence references re-frozen `ab345f537`, 4.4.1 and M4-P4 `896a9ce04`. Lot 4.3 (one published
type) is **not in M4**: the user deferred it past the effort (V16, option iii), and its two items
stay `[!]` with that reference.

**The number that frames the whole milestone: no decode revision rose.** `source.Rev`
(`source-2026-09-16.2`), `profile.Rev` (`profile-2026-09-17.3`), `grammar.Rev`
(`grammar-2026-09-15.42`) and `facts.Rev` (`killsource-2026-09-17.2`) are byte for byte the ones
M3 closed on; only `SchemaVersion` moved, 61 to 62. So M4 **opens no killsource backlog** — the
rule of D-6 is that a rise of `facts.Rev` opens it, and none happened — and every artifact of the
park is stale for one reason only: its publication.

### D-7 — Facts and publication are separate. **Reached.**

`data/cache/film_facts/{slug}/<short8>.filmfacts.bin` exists, resolved through `PathResolver`
(`domain/title/registry_film_facts.go`, the literal `film_facts` held by a ratchet). The name is
not the one the plan wrote — `<short8>.facts.json` was already taken and means the **inverse**
(what the *database* knows of the match, `replaybuild/facts_file.go`), so the distinction is
written at the head of the resolver and pinned by a test.

**The file has five length-prefixed sections** (a reader skips an unknown one): the scan inputs
(the blob plus the four channels its caller used to keep — the "not transported" mechanism is
deleted), the film identity, the fallback report **of the scan only**, the statborg, and the
kill-source. That fourth section carries a field the brief had not listed and a consumer needed:
the manifest's chunk clock, without which every Assault film replayed from facts would lose its
arming layer.

**The header is `replay.DecoderCoverage` verbatim, not a second block of revisions** — a second
one would have been the third copy — plus two distinct numbers, `VersionCodecFaits` (the
container) and `SchemaDesFaits` (the payload), and the cooking key (map module, axis widths,
detected layout) verified in **both** directions by the same function as the input blob. It is
**110 bytes**, so the "decode or re-read" decision is taken on 110 bytes and never on the megabyte
of positions: the test proves it by truncating the file to its header. Freshness is all or nothing
(codec, facts schema, the four layer revisions, cooking key); `build` and `registry` are **not**
compared, because these are facts of the film, not of the binary.

**Publication replays from the facts.** `replay.BuildFromFacts` is exported;
`replay.BuildFromFilmAvecFaits` also returns the facts to persist and `BuildFromFilm` is a call
that throws them away, so there is **one** scan stage in the source. The switch sits in
`replaybuild.BuildBytes`, after `ResolveMapEntry` (the catalog entry serves both branches and
validates the header) and before any film is loaded; what differs between the branches is one type
(`entreesDeCuisson`), while assembling, catalog collection and serialization are **common**. The
solo lock is unchanged and a ratchet holds that `replaybuild` neither takes nor releases it.
Writing the facts is atomic, non-fatal and never silent. One sink for artifact bytes is now
actually guarded: `replaybuild.TestBrancheDesFaitsTraverseLeMemePuits` demands that the callers of
`writeArtifactBytes` be exactly the three of its dated allowlist and that none of the three switch
functions call one (`archlint/no_second_artifact_sink_test.go`, which the plan cited, counts the
wiring of the *notification* sink — the measure corrected the item).

**The equality is proven, and the gain is measured rather than announced.** Criterion S8:
`cmd/replay-equiv -deux-passes` plays both branches of the **same commit** per film (forced
decode, then replay from the facts) and compares; it can read and write no reference, and
`-deux-passes -update` is refused explicitly. Verdict on ten films: **10 artifacts identical to
the byte, 0 divergent**, with one real gap everywhere — `killsource`, the deliberate loss of
`Kill.paquet`, an unexported field that `digest.Of` hashes anyway. Durations, decode against
replay-from-facts: 15.5 s / 163 ms, 46.7 s / 166 ms, 32.5 s / 326 ms, 39.1 s / 255 ms, 42.4 s /
277 ms, 47.6 s / 281 ms, 12.7 s / 122 ms, 32.4 s / 214 ms, **2 min 37 / 355 ms**, 18.0 s / 134 ms
— ratios of 95x to 442x, for facts files of 2.7 to 11.9 MB. The plan expected "seconds against
15 s"; the measure is hundreds of milliseconds against 13 s to 2 min 37.

### D-6 — One revision per thing that can change. **Reached for the layer, and `layers` is how.**

`SchemaVersion` is **62** (`film/replay/document.go`), a single rise for the whole milestone, with
its chronicle entry written in the commit that raised it. The document carries `layers` at its
root: **47** cooked root fields are attributed to a producing layer, **8** are exempt with a dated
reason each (`coverage`, which measures all layers at once; `layers` itself, which would be
circular; the catalogs no revision hashes), and **16** carry a production guard, so the table says
which layer produced a field and a closed guard means the field is absent from `layers` rather
than silently empty. Measured on a real artifact: `000d5950` declares **36 layers of the 47**, and
the 11 missing ones are exactly the guarded passes that did not run on a cooking without caller
facts — cross-checked on `zoneStates`, where `layers[zoneStates]`, `coverage.zones` and the array
itself all say the same thing.

The milestone's other new block is `coverage.deathsPaths` (`walk` and `directScan`, each with
`population` / `matched` / `published`): **six leaves**, present on **10 of the 17** corpus
witnesses — exactly those where `KillsInput.Read` is true. It is one third of what M3 had retained
for this schema rise: the grenade-scan counters and the position counters of lot 3.4.1 are **not**
published, and the plan's closure triage says why (exporting the first would move `grammar.Rev`,
hence `facts.Rev`, hence the backlog the user deferred; the second do not exist in production).

The proof that the rise added and removed nothing else was taken twice. Field by field, without
decoding: on the 8 contract fixtures, removing exactly `layers` and `coverage.deathsPaths` from
the schema-62 document and setting `schemaVersion` back to 61 gives a JSON **identical** to the
schema-61 one, 8 out of 8. And on the machine: `replay-corpus-gate` over the 17 witnesses,
**17/17 `ok`, 0 loss, 0 change**, gains exclusively under `layers.*` and
`coverage.deathsPaths.*`, verified leaf by leaf on the 34 artifacts the run kept.

### D-8 — The Go / web contract. **Unchanged in its rules, extended by one reading.**

`MIN_RENDERABLE_SCHEMA_VERSION` is still **27** and still pinned by test: no bump of this
milestone removes a field promised to the client. The web reads the new table at the same
boundary, which is **two** files rather than the one D11 of the plan named —
`lib/replay/replayNormalize.ts` fills and lets through, and `lib/replay/replayDocumentSchema.ts`
must declare the key in its `z.strictObject`, or `tsc -b` reddens and the badge would say
`invalid` in production. "Has this layer been produced?" is answered in one place,
`features/match-replay/model/calquePresent.ts`, with **three** states — produced, not produced,
unknown (an artifact older than 62 declares nothing, and claiming either answer for it would be a
lie). The admin badge gained one key per language and no fifth state: it names the **publication**
layer and only it, because that layer's revision *is* `publication-<schemaVersion>` and
`latestSchemaVersion` says the producer's — the comparison needs nothing transported. Naming the
four decode layers would require carrying their current revisions to the client; that is written
in the plan's triage as not retained, and the function's godoc says what it will never claim.

### D-1 — Five layers, one dependency direction. **Reached; the facade settled by decision.**

The facade `film/decfilm` re-exports **166** top-level exported declarations — the same number as
at the M3 closure, and the preparation note's projection that "M4 adds to the facade" is
**refuted by the measure**: the new doors of 4.1 and the two revision accessors of 4.4.1 were
needed by `replaybuild`, which imports the publication layer directly, so the facade did not grow.
What grew is the companion surface: **257** distinct `replay.X` identifiers cited outside `film/`,
against 245 at the M3 closure, every step dated in the ratchet (245 at its posting, 246, 253 for
the facts switch, 255 for the schema rise, 257 for the layer verdict).

Both numbers are counted by a test since lot 4.1.1-a — `archlint/film_facade_surface_test.go`,
with its counting method written and reproducible in one command, a per-family breakdown by
destination package, an anti-mute floor, and equality rather than inequality so that it reddens in
**both** directions. "A facade of 166 symbols is an alias, not a boundary" therefore stays true,
and it is now measured rather than asserted in three hand-written comments.

**Decision V25 (2026-09-18, the user's): leave both facades as they are, with a surface ratchet.**
The reduction is **not retained** — neither the facade's 166 nor the companion's 257. The
preparation note had already set aside the "lot 4.0" on four measured grounds (no item of M4
depends on it, zero user-visible gain, incoherent with deferring 4.3, and the real weight is the
companion, not the facade); the user's decision closes the question for the effort. What holds the
line is the dated ratchet, and nothing else: a symbol more reddens, a symbol less reddens too.

### D-4 — An unknown build fails loudly. **Unchanged by M4, and the facts path inherits it.**

No lot of M4 touched the gate: `replay.CleDuFilm` is still the single door, both orchestrators
still stop before reading a byte, and the counters are unchanged. **The facts path does not carry
the gate, and it does not need to** — and the reason is worth writing, because "the gate is on the
other branch" would look like a hole. `ecarterSiCleInconnue` runs on the decode branch only
(`replaybuild/filmfacts_cuisson.go`, right after the film is loaded), since the key is read *in
the film*. A facts file can therefore only exist for a film that passed the gate; and the day the
profile table changes — the only way a known key becomes unknown — `profile.Rev` moves too, so
every facts header goes stale (freshness is all or nothing) and every film is decoded again,
through the gate. The invariant holds by construction rather than by a second copy of the check,
which is what D-1 asks for. D-10 and D-10 bis are unchanged too: the fallback registry carries
**99 entries across seven files**, and the scan-only half of the report is what the facts file
persists (measured: **2** of the 18 firing sites are in the scan, 16 in the assembly, which is why
persisting the post-assembly report would double-count `coverage.fallbacks`).

### Lot 4.4.1 — selective re-cook. **Half delivered, half out of reach, and measured.**

Delivered: `replaybuild.Digest` carries `Layers` (a fifth parser key), `Verdict(faitsPresents)`
returns `a-jour` / `republier` / `redecoder`, `ArtifactVerdict` is the shape of the decision sites,
`UpToDate()` becomes a view of the verdict, and `wouldDowngrade` refuses to overwrite an intact
artifact with a candidate cooked under a stale layer. "A publication change does not re-decode"
**is** the `republier` verdict; it falls back to `redecoder` when the facts are missing, and there
is deliberately **no fourth state** ("I would republish if I had the facts") because two callers
would translate it differently.

Out of reach: **"a grammar change re-cooks only what depends on it."** The four revisions are
compile-time constants posed together (`coverage_decoder.go`), so the moment `grammar.Rev` moves,
every artifact carries a different `grammarRev` and "what depends on it" is the whole park — the
behaviour of today. A per-layer fingerprint, the only thing that would make the sentence true, is
forbidden outside `film/revision` by a ratchet with no exception table: that is a lot of its own,
not an item of 4.4. The limit is written in the godoc of `Verdict`, the plan's case is statused
`[~]` with it, and **the reformulation proposed in the preparation note §3.8 was never validated by
the user** — so the case stays as written rather than being rephrased into something easier to
tick.

### M4-P4 — the closing republication pass

`levelup backfill-replay --only-existing` now says what it did: among the artifacts it rebuilt it
splits `republies` (decode intact, replayed from the facts) from `redecodes` (a stale layer, or no
facts on disk), fed by the verdict above and counted **on success only**, their sum at most equal
to `construits` — a film with no prior artifact is cooked for the first time and is neither. The
domain of the pass is the artifact park, **87 artifacts**, not the 1 589 cached films. **The pass
itself is a pilot gesture**, played on signal after the merge; its result is recorded in the plan's
§5 and in the closing journal entry, not here.

### What M4 leaves partial, named here so it is not re-discovered

- **One published type** — lot 4.3, deferred past the effort by the user (V16, option iii): the
  two twins stay in step through the shape fingerprint and `service/replayview/parity_test.go`, and
  `X-Replay-Latest-Schema-Version` still carries the producer version beside a converted document.
- **The facade's surface** — 166 and 257, not reduced and not to be reduced (V25); held by a dated
  ratchet in both directions. `film/revision` still sits outside `film/internal/`, and
  `decfilm.Rev` still carries a name from the time there was one decoder revision rather than four:
  both are pure moves nobody's fingerprint sees, and neither is a defect of value.
- **Two counter families M3 had retained for this schema rise** — the grenade-scan and the
  position counters of 3.4.1 — are not published, and they now wait for the next `facts.Rev`
  signal rather than for a schema.
- **Selective re-cook by grammar layer** — out of reach by construction while the four revisions
  are constants posed together (above).
- **The regression M3 assumed** is unchanged and still written: `vehicles.tEnd/presents` and
  `vehicles/par-end/destroyed`, on two witnesses.
- **The killsource backlog** opened by M3's `facts.Rev` rise is untouched: V24 defers it to an
  explicit user signal, at the earliest after M4, grouped with any later rise.

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
   **Consumed on 2026-09-17 (M2 closure).** Lot 2.5.c brought the package down with the grammar
   layer: it is now `internal/games/halo_infinite/film/internal/grammar/weaponv3/` (7 files), it no
   longer sits under `analysis/`, and its bit reader is the canonical one of the source layer. The
   correction stays written because the wrong path still appears in the plan and in the briefs
   derived from it.
5. **A film whose key the profile does not know is not set aside.** Added 2026-09-17, measured at
   the M2 closure. **CONSUMED on 2026-09-17 by lot 3.1.1** (`f6b37a662`): both orchestrators now
   read the verdict and set the film aside — D-4 above says what they do and what they leave
   behind. The paragraph stays written because it names the defect and the measurement that found
   it; what follows is the state it described, not the state of the tree. D-4 says the film "is set aside". On the tree, the typed error exists and is
   carried to the caller (`FilmContext.ProfileErr`), the per-build expvar counter exists, the
   constructor logs one line — and no production caller reads that error: the only occurrence of
   `ProfileErr()` outside tests is its own declaration. The cook proceeds on the invariants
   profile — never on another build's, which is the half of the decision that matters most — and
   the artifact says the key did not serve, by emptying `coverage.decoder.build` while keeping
   everything actually read. So the behaviour is "loud and traced, decoded on invariants", not
   "set aside", and deciding whether a film must really be set aside is an orchestration question
   (`sync/killcollector`, `replaybuild`) that no lot of M2 opened.

