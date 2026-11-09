# NOTICE — origin of the test fixtures of `libs/gamefiles`

This module is licensed AGPL-3.0 (see `README.md`). Its test fixtures are not source code: three Forge
map variants (`.mvar`), two navigation meshes (`navmesh.blob`) and a set of synthetic fuzz seeds. This page says
where each group comes from, what it contains, and how it sits with the rule of ADR 0041 (no file
extracted from the game installation in a brick). Nothing here is Microsoft, 343 Industries or Halo
Studios software; the module is not affiliated with them.

The commit hashes below are those of the repository history at the time of writing (2026-11-09).

## Summary

| File(s) | Content | Origin as recorded in the history | Status against ADR 0041 |
|---|---|---|---|
| `mapvar/testdata/cliffhanger_map.mvar`, `mapvar/testdata/cliffhanger_ridgeline.mvar` | Two Forge map variants of the map Cliffhanger (the second one is the variant of the `ridgeline` module) | origin to be stated by the maintainer | see the section below |
| `mapvar/testdata/vagabond_fo08_wetland.mvar` | One Forge map variant of the map Vagabond | downloaded from the Halo UGC service (`mapobj-build --save-mvar`) | see the section below |
| `hinavmesh/testdata/01af558d-53ab-4f05-ba68-92d805fc6260.navmesh.blob` | Navigation mesh of the Forge asset `01af558d` (the map Isolation) | origin of these bytes to be stated by the maintainer; the documented channel is the anonymous UGC download | see the section below |
| `hinavmesh/testdata/df7dbf08-b8de-4ade-9d7f-1947128c9ae4.navmesh.blob` | Navigation mesh of the Forge asset `df7dbf08` (the map Kiken'na) | origin of these bytes to be stated by the maintainer; the documented channel is the anonymous UGC download | see the section below |
| `mapvar/testdata/fuzz/FuzzMapvarParse/seed_00` to `seed_05` | Synthetic corpus seeds for the fuzz test of the `.mvar` reader | written by the test generator (see below) | no game content |

## Forge map variants (`.mvar`)

A `.mvar` is the object list of a Forge map variant (object positions, type identifiers, team indexes,
objectives), in a Bond CompactBinary v2 container. It is read by `mapvar.Parse`. The variants are the
files of the User Generated Content (UGC) assets of the Halo service; they are not read from the game's
`.module` archives, which the history records as a path that does not give the object list (commit
`59bb7b2eb0`, 2026-09-01).

- `cliffhanger_map.mvar` (93 795 bytes) and `cliffhanger_ridgeline.mvar` (84 794 bytes): the map Cliffhanger
  (the registry of the maps names the module `ridgeline` for it). Introduced by `2044b71397` (2026-07-31,
  "feat(B): porter l additif du rejeu 2D sur la branche saine"), first at `.ai/re_dump/mapvar/`; moved to
  `libs/gamefiles/mapvar/testdata/` in the modules move `a5bae50b44` (2026-10-28). Origin to be stated by the
  maintainer: the commit does not say where these bytes come from.
- `vagabond_fo08_wetland.mvar` (17 280 bytes): one of the two variants of the map Vagabond (the other is
  `map.mvar`, not versioned). Introduced by `fb33bbddc2` (2026-08-08, "fix(v7.5-lot5): mapobj-build publiait
  le rack du canevas, pas la carte jouee"), whose message says the variants are saved by map identifier from the
  UGC service with `--save-mvar`. Moved to the module path by `c8eeea23a2` (2026-10-20, release v8.0.0).

Contents: the object lists and objectives of the map variants. No player identifier, no film and no
installation file is part of these bytes as far as the history shows. The fixtures are read by
`mapvar_test.go`, `landgrab_test.go`, `cb2_fuzz_test.go`, `noms_lieux_hunt_research_test.go` and by
the research files of the package (behind the `research` build tag).

The `.gitignore` of the repository ignores `libs/gamefiles/mapvar/testdata/*.mvar` and keeps these three files
by name (lines 367 to 370). It describes the rest of that folder as a local cache of re-downloadable variants.

## Navigation meshes (`navmesh.blob`)

A `navmesh.blob` is the navigation mesh that a Forge asset publishes next to its variant. The two files are
the blobs of the Forge assets `01af558d` (Isolation, asset registry entry `forge`) and `df7dbf08` (Kiken'na,
`forge`). They are read by `hinavmesh.Decode`, through the test helper `decodeTemoin` and the
`TestOracle*` tests.

- Introduced by `682ae21951` (2026-08-27, "feat(v7.5-cartes): Isolation est lisible — le fond vient du
  maillage de navigation"; the same message is on `3fecc17ad7`). Moved to the module path by `c8eeea23a2`
  (2026-10-20) and by the modules move `a5bae50b44`.
- The documented channel for a `navmesh.blob` is the anonymous download from the public page of the Forge
  asset (`cmd/mapnav-fetch`, whose header says the navmesh is not in the game installation). The commit that
  added these two files does not record the download, and the bytes are not tied to a download by a hash in
  the history. Origin of these two versioned files to be stated by the maintainer.

## Synthetic fuzz seeds

`mapvar/testdata/fuzz/FuzzMapvarParse/seed_00` to `seed_05` are written by the test generator
(`TestFuzzMapvarSeedsRegenerate`, flag `-update-graines-mvar`, `cb2_fuzz_test.go`). They are a minimal valid
root and the five forms of the count fields of the format. They are not taken from a game file. The commit
that first introduced them under this path was not found by the history search; the content is what the
generator writes.

## Rule of ADR 0041 (no game-extracted file in a brick)

ADR 0041, decision D5, says that the fixtures above are reverse-engineering artefacts already versioned in
the repository, that they stay (one copy each, owned by this brick), and that nothing read from the
installation may be added. The root `NOTICE` (section "Test fixtures") says that the content of the map
variants and of the film fixtures belongs to Microsoft and is kept only because tests read it.

Reading of the history for this page:

- The `.mvar` files and the navmesh blobs were not read from the installation, according to the commits
  and notes that describe their channel (`59bb7b2eb0`, `fb33bbddc2`, the navmesh notes of 2026-08-27). For the
  two Cliffhanger files and the two navmesh blobs, the commits do not state the channel.
- Their content is game content (map objects and navigation geometry of Halo Infinite maps). D5 names
  them as the exception to the rule, so the rule holds only by that exception.
- The AGPL-3.0 licence of this module does not give rights to that content. The root `NOTICE` says it is
  "not licensed for any other use".

Compliance: no file of `libs/gamefiles` is recorded as read from the installation. The game-content fixtures
are an explicit exception of D5, not a case the rule covers.
