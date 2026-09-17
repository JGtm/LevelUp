# Runbook — Film profiles catalogue (adding a build)

What the repository knows about a film's grammar is **data**, not code:
`data/titles/{slug}/reference/film_profiles.json`, resolved by
`title.PathResolver.FilmProfilesPath` (decision D12 of
`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`: new paths go through `PathResolver`, and a versioned
catalogue is never written at runtime).

This runbook is the procedure for **adding a build** to that catalogue — which tool to run,
which witness to add to the corpus, how a `presumee` entry becomes `prouve`, and how provenance
is written.

Related files:

- `data/titles/halo_infinite/reference/film_profiles.json` — the catalogue (24 `entries` and 9
  `registryFingerprints` as of 2026-09-16).
- `apps/go-api/internal/games/halo_infinite/filmprofile/` — the only reader. It imports nothing
  from `film/...`: a consumer without the decoder can still say what the repository knows about
  a build.
- `apps/go-api/cmd/film-profiles-build/` — the fabrication chain for the **derived** block, and
  its `gamefiles` gate.
- `apps/go-api/internal/games/halo_infinite/film/internal/profile/profile_table.go` — the lot 2.1
  table, still the source of truth for the **content** of `entries` while both coexist (lot 3.1.1
  makes the decoder read the file and removes the copy). `TestCatalogueConformeALaTableDuLot21`
  keeps them equal, line for line. It does **not** cover `registryFingerprints`: that section has
  no copy in the decoder, and is validated on its own values.
- `config/replay_corpus.toml` — the witness corpus (one `[[temoin]]` per grammar family, and
  since lot 3.2.3 at least one per registry key).

---

## 1. The key is what the film writes — there is no "profile per build"

The user settled this on 2026-09-16: **the film is self-contained**. A profile is therefore
never an external setting that would refuse a film; it is a table indexed by what the film
*carries*. The catalogue knows four key shapes:

| Key | Read from | What it decides |
|---|---|---|
| `format=<n>` (`format=20,21,24,25`) | `chunk_00+4` | the grammar of the bits |
| `build=<id>` (`build=HI_1_13_0`) | section 2 of `chunk_00`, plain text | the size of content structures |
| `majeure<=38`, `majeure=39,40`, `majeure>=41` | `chunk_00+0` | the gamertag layout in a highlight event block |
| `toutes` | nothing — invariant applied to every film | a value whose per-build variability has never been measured |

A key the reader does not understand is **refused**, not skipped: a line that selects no film is
a profile value that silently never applies.

## 2. Two natures of entry, and only one of them is fabricated

| Nature | Where | Who writes it |
|---|---|---|
| **derived** (`derived`) | fingerprint of the map-bounds catalogue | `go run ./cmd/film-profiles-build` |
| **entered** (`entries`) | what comes from the executable or from a witness film | a human, with provenance |
| **entered** (`registryFingerprints`) | the ECS registry fingerprint of a build, measured on a witness film | a human, with provenance — see §4.5 |

`registryFingerprints` is entered too, but it is a **separate section on purpose**: a
fingerprint, a block count and a named-slot count are *quantities*, and a conformity test bites
on them value against value, where `entries.value` is a sentence written for a human. The tool
copies both entered sections through untouched.

Map bounds are **not copied** into the profile. They already have their own catalogue,
`map_quant_bounds.json`, produced by `cmd/mapquant-build` from the game's `.module` files; the
profile only carries its fingerprint (schema version, map count, sha256 of the map entries), so
that a divergence — game updated, bounds regenerated — is *seen* instead of guessed. The
fingerprint deliberately ignores the bounds catalogue's `source` field: it carries the install
path of the machine that produced it, which is a fabrication trace, not game data.

## 3. Provenance is a column, not a courtesy

Every entered line carries where its value comes from. There are exactly three provenances, and
no fourth:

| `provenance` | Meaning | What `proof` must contain |
|---|---|---|
| `relue` | the executable is open, the writer is read | the Ghidra function (name + address, base `0x140000000`) |
| `mesuree` | the executable for that build is not available; the value is measured on a witness film by an oracle **internal to the film** | the witness film, the oracle, and its score |
| `presumee` | neither | what is missing — this is the register of what remains to be established |

`date` is the day that provenance was established, `YYYY-MM-DD`. An empty `proof` is refused by
validation: a value without written proof does not enter the catalogue.

## 4. Adding a build — the procedure

### 4.1 Add a witness to the corpus (before anything else)

A build with no witness cannot be proved and cannot be regression-tested. Add one
`[[temoin]]` entry to `config/replay_corpus.toml` — the short match id (8 hex characters), its
family, and the *reason* it was chosen (what layer or defect it carries). Follow the existing
entries: `111fa685` (version 39 / build `HI_1_10_0`), `e5adf7b2` (version 40 but build
`HI_1_11_0` — the grammar where the major version lies), `60ae07c4` (version 37 / build
`HI_1_8_0`).

**The rule since lot 3.2.3: one witness per registry key.** Every key of
`registryFingerprints` — every build, plus every major version of the films whose `chunk_00`
carries no identification section — has at least one `[[temoin]]`. Family names follow the
written keys: `version_<major>_build_<build>`, or `version_<major>_sans_identification`.

**Three conditions make a film gateable** — check all three *before* adding the entry:

```bash
ls <parc>/data/cache/film_chunks/<id8>/chunk_00.bin      # 1. chunks in the cache
ls <parc>/data/cache/film_manifests/<id8>.json           # 2. chunk manifest (without it,
                                                         #    replaybuild silently scores 0)
cd apps/go-api && CGO_ENABLED=1 LEVELUP_REPO_ROOT=<parc> \
  go run ./cmd/levelup replay-facts-export --out <tmp> --title halo_infinite <id8>   # 3. facts
```

A pre-cooked artefact under `data/cache/replays/{slug}/` is **not** a condition: the gate cooks
both sides itself (`--reference base`), and as of 2026-09-16 only one of the 14 standing
witnesses had one. An id the match registry does not know makes the gate report that witness
`ABSENT` — never an error, and never a silence either (`verifierCouverture`).

### 4.2 Enter the line as `presumee`

Add the entry to `film_profiles.json` with `"provenance": "presumee"` and a `proof` that says
what is missing (for example: "widths captured at runtime on ONE session; no read at the writer,
no per-build measurement"). A presumed line is legitimate — it is the honest state of
knowledge — as long as it is *named* as such. What is not legitimate is a value with no line.

`film/internal/profile/profile_table.go` in the **same commit**: `TestCatalogueConformeALaTableDuLot21` fails
otherwise, on purpose (two truths for one profile is worse than one gap).

### 4.3 Promote it to `relue` or `mesuree`

- **`relue`** — read the writer in the executable (Ghidra, read only, shared
  `HaloInfinite.exe` instance, base `0x140000000`). Write the function name and the address in
  `proof`, and the day in `date`. This is the preferred route: where the writer is readable, the
  grammar is taken from the writer, never by measurement (decision D3).
- **`mesuree`** — when that build's executable is not available, measure on the witness with an
  oracle internal to the film, and cite the oracle and its score in `proof` (for example:
  "transposition of -2880 bits, single value across the 39 `HI_1_11_0` films in the cache").

### 4.4 Re-run the fabrication chain and the gates

```bash
cd apps/go-api
# Only if the game was updated or map bounds changed:
CGO_ENABLED=1 go run ./cmd/mapquant-build

# Rewrite the derived block (entries are copied through untouched):
go run ./cmd/film-profiles-build
go run ./cmd/film-profiles-build --check      # writes nothing, exits 1 on divergence

# Gates
go test ./internal/games/halo_infinite/filmprofile/ ./internal/archlint/ -count=1
CGO_ENABLED=1 go test -tags=gamefiles ./cmd/film-profiles-build/ -count=1
```

**From a worktree**, `cmd/film-profiles-build` locates the repository through
`title.FindRepoRoot`, which looks for `db_profiles.json` — a gitignored file that only exists in
the main checkout. Either export `LEVELUP_REPO_ROOT=<the worktree>` (never point it at another
checkout) or pass `--out` and `--bounds` explicitly.

### 4.5 Add a registry fingerprint (`registryFingerprints`)

The ECS registry is the identity of a film's *component grammar*: bit-for-bit identical from
film to film **within one build**, different across builds. Its fingerprint is an FNV-1a 64-bit
hash of the named entries (`level | name`, in chunk order) — the hashing domain changed at lot
1.2 on 2026-09-14, so **any value written before that date is unusable**; three dead domains
exist in the notes, none transposable into another.

**Measure it without decoding a film.** Only `chunk_00` is read — no bitstream, no other chunk,
no solo lock:

```go
data, _ := grammar.ReadFilmChunk(dir, 0)   // dir = a committed minifilm, or a cache film
reg, _ := grammar.ParseRegistryChunk(data) // one pass; the fingerprint is accumulated in it
fp := grammar.RegistryFingerprint(reg)     // 0x%016x
id, _ := grammar.ReadFilmIdentity(data)    // the build, plain text, section 2 (error = no section)
maj, _ := grammar.FilmMajorVersionFromHeader(data)
```

The seven **committed mini-reels** (`film/replay/testdata/minifilm_<id8>/chunk_00.bin`, one per
known build) make this measurable from the git tree alone, with no film from the cache. Measured
2026-09-16 on all seven: the mini-reel's registry equals its full film's, checked on `a521164d`
and `11de8353`.

**Write the entry.** Keys are `build=<id>`, or `majeure=<n>` for films whose `chunk_00` carries
no identification section (they write no build at all). `toutes` and `format=` are refused: a
registry is a property of the *game build*, never of the repository or of the bit format.

```json
{
  "key": "build=HI_1_11_0",
  "fingerprint": "0x8879e2b6746ba047",
  "blocks": 49,
  "namedSlots": 1031,
  "status": "presumee",
  "provenance": "mesuree",
  "witnesses": ["e5adf7b2"],
  "proof": "direct read of chunk_00 of the committed mini-reel …; same value at the corpus gate of …",
  "date": "2026-09-16"
}
```

`status` is **not** the provenance, and confusing the two is the trap of this table:

| `status` | Means |
|---|---|
| `connue` | read on at least two distinct films of that key, or on the only film the cache holds for it — the population is covered |
| `presumee` | one film of the key was read while the cache holds others: nothing contradicts the value, nothing proves it holds for the whole key yet |

The third state the decoder needs — `inconnue` — never appears in this file: it qualifies a
fingerprint *read in a film* and absent from this table, which is a runtime classification.

**Two measured facts to expect.** A fingerprint is **not unique** (`HI_1_8_0`/`HI_1_9_0`,
`HI_1_12_0`/`HI_1_13_0` and `HI_1_4_1`/`majeure=33` each share one), so validation enforces
uniqueness of the **key**, never of the fingerprint. And a `majeure=` key can also select a film
that *does* write a build; the reader
(`filmprofile.Catalogue.EmpreinteRegistrePour`) always returns the most specific key — the build.

Validation refuses: a malformed fingerprint (anything but `0x` + 16 lowercase hex digits), a key
shape that makes no sense for a registry, a `build=` absent from the catalogue's own build table
(the `entries` lines keyed `build=`), a duplicate key, an entry with no witness, an empty proof,
an undated date. Gate: `go test ./internal/games/halo_infinite/filmprofile/ -count=1`.

## 5. What the gates prove

| Gate | Runs where | Proves |
|---|---|---|
| `TestCatalogueConformeALaTableDuLot21` | everywhere (CI included) | the committed catalogue **is** the lot 2.1 table, line for line — same order, keys, values, provenances, proofs and dates |
| `TestCatalogueCommisEstValide` | everywhere | schema, unique (key, field) pairs, known provenances, non-empty proofs, dated dates, and all four key shapes still present |
| `TestValideRefuseUneEmpreinte` (+ its neighbours) | everywhere | what the registry table refuses: malformed fingerprint, key shape meaningless for a registry, build outside the catalogue's build table, duplicate key, witness-less entry |
| `TestEmpreintesCommisesCouvrentLesBuildsDuCatalogue` | everywhere | every build the catalogue knows has its registry fingerprint, and both section-less major versions are covered |
| `TestEmpreintesCommisesSontDesGrandeursComparables` | everywhere | the reference build still reads `0x36ca8c3d2a2f9b88` / 50 blocks / 1067 named slots, and the oldest grammar is not silently collapsed onto it |
| `TestBlocDeriveSolidaireDesBornesCommises` | everywhere | the `derived` block describes the bounds catalogue that sits next to it in the tree |
| `TestCatalogueCommisEgaleCatalogueRegenere` | only where Halo Infinite is installed (`gamefiles` tag, CGO) | the whole chain: bounds regenerated from the game's `.module` files, then the committed profile catalogue byte-for-byte equal to what the chain produces from them |
| `TestRuntimeNEcritPasLeCatalogueVersionne` | everywhere | nothing in production ever writes this file |

The last one is the reason the catalogue has **no overlay**, unlike `map_weapon_pads.json`: the
decoder reads the profile, full stop. Anything it would write there at runtime would be a
guessed profile value — precisely what an unknown build must not produce (lot 3.1.1 puts the
film aside instead).

## 6. What the tool does not do

`cmd/film-profiles-build` writes the `derived` block and nothing else. It copies `entries` and
`registryFingerprints` through untouched and refuses to write a catalogue that does not validate. It never regenerates
an entered value from code: a profile value is read at the writer or measured on a witness — it
is not something a program can re-derive.
