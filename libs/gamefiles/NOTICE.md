# NOTICE — origin of the test fixtures of `libs/gamefiles`

This module is licensed GPL-3.0-only (see `README.md`). Its test fixtures are not source code: three Forge
map variants (`.mvar`), two navigation meshes (`navmesh.blob`) and a set of synthetic fuzz seeds. This page says
where each group comes from, what it contains, and how it sits with the rule of ADR 0041 (no file
extracted from the game installation in a brick). Nothing here is Microsoft, 343 Industries or Halo
Studios software; the module is not affiliated with them.

The commit hashes below are those of the repository history at the time of writing (2026-11-09).
Hashes of the form `xxxx (branch)` belong to feature branches squashed into `main`; they resolve only in a clone of LevelUp that still holds those branches. On `main`, the two Cliffhanger variants appear with `1fb02cfe59` (2026-07-31, at `.ai/re_dump/mapvar/`, moved to `.ai/V7.5/dumps/mapvar/` by `5abf8c78c0` the same day), the Vagabond variant and the two navmesh blobs with `c8eeea23a2` (2026-10-20, release v8.0.0, at `.ai/V7.5/dumps/mapvar/` and `apps/go-api/internal/hinavmesh/testdata/`), and all of them move under `libs/gamefiles/` with `6c3fa49063` (2026-11-09, release v8.1.0).

## Summary

| File(s) | Content | Origin as recorded in the history | Status against ADR 0041 |
|---|---|---|---|
| `mapvar/testdata/cliffhanger_map.mvar`, `mapvar/testdata/cliffhanger_ridgeline.mvar` | Two Forge map variants of the map Cliffhanger (the second one is the variant of the `ridgeline` module) | Files `map.mvar` and `ridgeline.mvar` of the UGC asset `5324364b-39a8-4f93-96a6-b80a1f18ce8a`, downloaded without token from the public Halo UGC service (verified byte for byte on 2026-11-09, see below) | see the section below |
| `mapvar/testdata/vagabond_fo08_wetland.mvar` | One Forge map variant of the map Vagabond | File `fo08_wetland.mvar` of the UGC asset `105f5d84-8de1-4908-af3a-1c4f3bf9d642`, same channel (verified on 2026-11-09) | see the section below |
| `hinavmesh/testdata/01af558d-53ab-4f05-ba68-92d805fc6260.navmesh.blob` | Navigation mesh of the Forge asset `01af558d` (the map Isolation) | `navmesh.blob` of the UGC asset `01af558d-53ab-4f05-ba68-92d805fc6260`, same channel (verified on 2026-11-09) | see the section below |
| `hinavmesh/testdata/df7dbf08-b8de-4ade-9d7f-1947128c9ae4.navmesh.blob` | Navigation mesh of the Forge asset `df7dbf08` (the map Kiken'na) | `navmesh.blob` of the UGC asset `df7dbf08-b8de-4ade-9d7f-1947128c9ae4`, same channel (verified on 2026-11-09) | see the section below |
| `mapvar/testdata/fuzz/FuzzMapvarParse/seed_00` to `seed_05` | Synthetic corpus seeds for the fuzz test of the `.mvar` reader | written by the test generator (see below) | no game content |

## Forge map variants (`.mvar`)

A `.mvar` is the object list of a Forge map variant (object positions, type identifiers, team indexes,
objectives), in a Bond CompactBinary v2 container. It is read by `mapvar.Parse`. The variants are the
files of the User Generated Content (UGC) assets of the Halo service; they are not read from the game's
`.module` archives, which the history records as a path that does not give the object list (commit
`59bb7b2eb0 (branch)`, 2026-09-01).

- `cliffhanger_map.mvar` (93 795 bytes) and `cliffhanger_ridgeline.mvar` (84 794 bytes): the map Cliffhanger
  (the registry of the maps names the module `ridgeline` for it). Introduced by `2044b71397 (branch)` (2026-07-31,
  "feat(B): porter l additif du rejeu 2D sur la branche saine"; on `main`, `1fb02cfe59`), first at `.ai/re_dump/mapvar/`;
  moved under `libs/gamefiles/` on `main` by `6c3fa49063` (2026-11-09; on the release branch, `a5bae50b44 (branch)`,
  2026-10-28, into `libs/gamefiles/mapvar/testdata/`).
- `vagabond_fo08_wetland.mvar` (17 280 bytes): one of the two variants of the map Vagabond (the other is
  `map.mvar`, not versioned). Introduced by `fb33bbddc2 (branch)` (2026-08-08, "fix(v7.5-lot5): mapobj-build publiait
  le rack du canevas, pas la carte jouee"), whose message says the variants are saved by map identifier from the
  UGC service with `--save-mvar`. Introduced on `main` by `c8eeea23a2` (2026-10-20, release v8.0.0, under
  `.ai/V7.5/dumps/mapvar/`) and moved under `libs/gamefiles/` by `6c3fa49063` (2026-11-09).

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

- Introduced by `682ae21951 (branch)` (2026-08-27, "feat(v7.5-cartes): Isolation est lisible — le fond vient du
  maillage de navigation"; the same message is on `3fecc17ad7 (branch)`). Introduced on `main` by `c8eeea23a2`
  (2026-10-20, release v8.0.0, under `apps/go-api/internal/hinavmesh/testdata/`) and moved under `libs/gamefiles/`
  by `6c3fa49063` (2026-11-09).
- The documented channel for a `navmesh.blob` is the anonymous download from the public page of the Forge
  asset (`cmd/mapnav-fetch`, whose header says the navmesh is not in the game installation). The commit that
  added these two files does not record the download, and the bytes are not tied to a download by a hash in
  the history.

## Synthetic fuzz seeds

`mapvar/testdata/fuzz/FuzzMapvarParse/seed_00` to `seed_05` are written by the test generator
(`TestFuzzMapvarSeedsRegenerate`, flag `-update-graines-mvar`, `cb2_fuzz_test.go`). They are a minimal valid
root and the five forms of the count fields of the format. They are not taken from a game file. The commit
that first introduced them under this path was not found by the history search; the content is what the
generator writes.

## Origin of the five game-content fixtures, verified

All five files are files that the Halo UGC service publishes for a map asset. On 2026-11-09 each was
downloaded again without any token (the public page of the asset gives the storage prefix, the storage
serves the file) and compared with the versioned file: the five are identical byte for byte.

| Versioned file | UGC asset | Published file | SHA-256 |
|---|---|---|---|
| `mapvar/testdata/cliffhanger_map.mvar` | `5324364b-39a8-4f93-96a6-b80a1f18ce8a` | `map.mvar` | `e81eff51154a28d521b53211557d65d5dbdd90fd3333aa999ecd4fc9fc4f9c07` |
| `mapvar/testdata/cliffhanger_ridgeline.mvar` | `5324364b-39a8-4f93-96a6-b80a1f18ce8a` | `ridgeline.mvar` | `39da68769a36dc9481e1ea4668278124a95546057b205e289336073506ea42e4` |
| `mapvar/testdata/vagabond_fo08_wetland.mvar` | `105f5d84-8de1-4908-af3a-1c4f3bf9d642` | `fo08_wetland.mvar` | `d9ad81a9233ab3136b1ef7c33be65114b04ce3f47a3ca2312c74e43e795c4275` |
| `hinavmesh/testdata/01af558d-53ab-4f05-ba68-92d805fc6260.navmesh.blob` | `01af558d-53ab-4f05-ba68-92d805fc6260` | `navmesh.blob` | `b015084c6b22ead9f28cad8cf1d1b2abee5771e908001f4c6534669910bc5708` |
| `hinavmesh/testdata/df7dbf08-b8de-4ade-9d7f-1947128c9ae4.navmesh.blob` | `df7dbf08-b8de-4ade-9d7f-1947128c9ae4` | `navmesh.blob` | `3b447e3e41a4d9043ee4527f600fb0c9f5f711fb98693579d632d3069a0a198e` |

The history of the repository does not record these downloads (the commits that added the Cliffhanger files
and the navmesh blobs name no source); the hashes above are the evidence. None of the five was read from
the game installation. The maps are Microsoft's content; the Forge maps of the service may also have
community authors, whom the history does not name. Nothing here is licensed for other use, and a rights
holder who wants a file removed can open an issue on the repository.

## Rule of ADR 0041 (no game-extracted file in a brick)

ADR 0041, decision D5, says that the fixtures above are reverse-engineering artefacts already versioned in
the repository, that they stay (one copy each, owned by this brick), and that nothing read from the
installation may be added. The root `NOTICE` (section "Test fixtures") says that the content of the map
variants and of the film fixtures belongs to Microsoft and is kept only because tests read it.

Reading of the history for this page:

- The `.mvar` files and the navmesh blobs were not read from the installation, according to the commits
  and notes that describe their channel (`59bb7b2eb0 (branch)`, `fb33bbddc2 (branch)`, the navmesh notes of 2026-08-27). For the
  two Cliffhanger files and the two navmesh blobs, the commits do not state the channel.
- Their content is game content (map objects and navigation geometry of Halo Infinite maps). D5 names
  them as the exception to the rule, so the rule holds only by that exception.
- The GPL-3.0-only licence of this module does not give rights to that content. The root `NOTICE` says it is
  "not licensed for any other use".

Compliance: no file of this module was read from the installation: the five game-content fixtures are files of the
Halo UGC service, an explicit exception of D5, not a case the rule covers.
