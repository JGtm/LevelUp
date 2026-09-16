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

- `data/titles/halo_infinite/reference/film_profiles.json` — the catalogue (23 entries as of
  2026-09-16).
- `apps/go-api/internal/games/halo_infinite/filmprofile/` — the only reader. It imports nothing
  from `film/...`: a consumer without the decoder can still say what the repository knows about
  a build.
- `apps/go-api/cmd/film-profiles-build/` — the fabrication chain for the **derived** block, and
  its `gamefiles` gate.
- `apps/go-api/internal/games/halo_infinite/film/internal/profile/profile_table.go` — the lot 2.1 table,
  still the source of truth for the **content** while both coexist (lot 3.1.1 makes the decoder
  read the file and removes the copy). `TestCatalogueConformeALaTableDuLot21` keeps them equal,
  line for line.
- `config/replay_corpus.toml` — the witness corpus (one `[[temoin]]` per grammar family).

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

### 4.2 Enter the line as `presumee`

Add the entry to `film_profiles.json` with `"provenance": "presumee"` and a `proof` that says
what is missing (for example: "widths captured at runtime on ONE session; no read at the writer,
no per-build measurement"). A presumed line is legitimate — it is the honest state of
knowledge — as long as it is *named* as such. What is not legitimate is a value with no line.

While the lot 2.1 table and the catalogue coexist, the same line must be added to
`profile/profile_table.go` in the **same commit**: `TestCatalogueConformeALaTableDuLot21` fails
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

## 5. What the gates prove

| Gate | Runs where | Proves |
|---|---|---|
| `TestCatalogueConformeALaTableDuLot21` | everywhere (CI included) | the committed catalogue **is** the lot 2.1 table, line for line — same order, keys, values, provenances, proofs and dates |
| `TestCatalogueCommisEstValide` | everywhere | schema, unique (key, field) pairs, known provenances, non-empty proofs, dated dates, and all four key shapes still present |
| `TestBlocDeriveSolidaireDesBornesCommises` | everywhere | the `derived` block describes the bounds catalogue that sits next to it in the tree |
| `TestCatalogueCommisEgaleCatalogueRegenere` | only where Halo Infinite is installed (`gamefiles` tag, CGO) | the whole chain: bounds regenerated from the game's `.module` files, then the committed profile catalogue byte-for-byte equal to what the chain produces from them |
| `TestRuntimeNEcritPasLeCatalogueVersionne` | everywhere | nothing in production ever writes this file |

The last one is the reason the catalogue has **no overlay**, unlike `map_weapon_pads.json`: the
decoder reads the profile, full stop. Anything it would write there at runtime would be a
guessed profile value — precisely what an unknown build must not produce (lot 3.1.1 puts the
film aside instead).

## 6. What the tool does not do

`cmd/film-profiles-build` writes the `derived` block and nothing else. It copies `entries`
through untouched and refuses to write a catalogue that does not validate. It never regenerates
an entered value from code: a profile value is read at the writer or measured on a witness — it
is not something a program can re-derive.
