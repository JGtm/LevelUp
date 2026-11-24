# Textes exacts des notices (annexe du plan v8.1.2, étape 5)

Établis le 2026-11-09 par comparaison d'empreintes avec les sources amont (téléchargements anonymes, lecture seule).
Le plan `.ai/PLAN_V8_1_2_2026-11-09.md` dit où chaque texte va.

## T1. `libs/himap/NOTICE.md` (nouveau)

```markdown
# NOTICE — origin of the data files of `libs/himap`

This module is licensed AGPL-3.0-only (see `README.md`). This page lists the files of the module that
are not source code, says where each comes from, and says what the licence of the module does NOT
cover. Nothing here is Microsoft, 343 Industries or Halo Studios software; the module is not
affiliated with them.

| File | What it is | Origin | Rights |
|---|---|---|---|
| `himap/sbsp.xml` | Layout definition of the `sbsp` tag (scenario_structure_bsp) of Halo Infinite: the order and the kind of its fields. Embedded in the binary (`go:embed`, `himap/sbsp.go`) and read at run time to locate the world bounds, the instance table and the other blocks of a map's BSP | Unmodified copy (131 448 bytes, SHA-256 `895e0d06e026e5da7170bf011ff07196174b144b307af5c9241f5b6ea1a57003`) of `Plugins/sbsp.xml` of the project Infinite Runtime Tag Viewer, https://github.com/Gamergotten/Infinite-runtime-tagviewer, checked identical to its `master` on 2026-11-09. Its README credits Lord Zedd and Exhibit for dumping the tag definitions | Not covered by the licence of this module. The upstream repository states no licence, so the file stays the property of its authors. It is kept because the module cannot read a map without it, and it is removed on request of its rights holders (open an issue on the repository) |
| `himap/testdata/carte_validee_v1.png` | Reference render of a Halo Infinite map for the render benches (read under the `gamefiles` and `research` build tags only) | Map image validated by the maintainer on 2026-07-26 (repository history); the history does not record more about how it was rendered | Game imagery: Microsoft's. Kept for the tests, not licensed for any other use |

## Prior art

Several layouts and rules of the module (the root struct of the `rtgo` tag, the mesh flags, the
instance flags override) were first read in the sources of Reclaimer by Gravemind2401
(https://github.com/Gravemind2401/Reclaimer, GPL-3.0), then measured again on game files and written in
Go. No file of Reclaimer is included; the code cites Reclaimer where it takes a layout or a rule from it.

## Removed file

`himap/coupe.png`, shipped in v0.1.0, was the output of a test of the host repository, with no use in the
module. It is not in v0.1.1.
```

## T2. `NOTICE` racine, section 4, nouveau point après celui des « Test fixtures »

```
  - Tag layout definitions copied unmodified from the Infinite Runtime Tag Viewer project
    (https://github.com/Gamergotten/Infinite-runtime-tagviewer, folder Plugins/):
    libs/himap/himap/sbsp.xml, apps/go-api/cmd/weapon-icons-build/weap.xml and
    apps/go-api/cmd/weapon-sounds/weap.xml. The upstream repository states no licence; the
    definitions remain the work of their authors (the upstream README credits Lord Zedd and Exhibit
    for dumping them). They are read at run time to find fields in game files, and are outside
    sections 1 to 3.
```

## T3. `docs/ACKNOWLEDGMENTS.md`

La phrase d'introduction « art we learned from, not code we ship » devient « art we learned from, not code we ship,
except the one data file named below ». Nouvelle entrée :

```
- **Gamergotten** ([Gamergotten](https://github.com/Gamergotten)) and the contributors of
  [Infinite Runtime Tag Viewer](https://github.com/Gamergotten/Infinite-runtime-tagviewer), with
  **Lord Zedd** and **Exhibit** who dumped the tag definitions it carries: the tag layout files
  `libs/himap/himap/sbsp.xml` and the two `weap.xml` of `apps/go-api/cmd/` are unmodified copies of
  its `Plugins/` files (see `NOTICE`).
```

## T4. `docs/FR/ACKNOWLEDGMENTS.md`

```
- **Gamergotten** ([Gamergotten](https://github.com/Gamergotten)) et les contributeurs de
  [Infinite Runtime Tag Viewer](https://github.com/Gamergotten/Infinite-runtime-tagviewer), avec
  **Lord Zedd** et **Exhibit** qui ont extrait les définitions de tags qu'il porte : les fichiers de
  disposition `libs/himap/himap/sbsp.xml` et les deux `weap.xml` de `apps/go-api/cmd/` sont des copies
  non modifiées de ses fichiers `Plugins/` (voir `NOTICE`).
```

## T5. `libs/gamefiles/NOTICE.md`, tableau récapitulatif (colonne Origin, remplacer les cinq cellules)

```
| `mapvar/testdata/cliffhanger_map.mvar`, `mapvar/testdata/cliffhanger_ridgeline.mvar` | Two Forge map variants of the map Cliffhanger (the second one is the variant of the `ridgeline` module) | Files `map.mvar` and `ridgeline.mvar` of the UGC asset `5324364b-39a8-4f93-96a6-b80a1f18ce8a`, downloaded without token from the public Halo UGC service (verified byte for byte on 2026-11-09, see below) | see the section below |
| `mapvar/testdata/vagabond_fo08_wetland.mvar` | One Forge map variant of the map Vagabond | File `fo08_wetland.mvar` of the UGC asset `105f5d84-8de1-4908-af3a-1c4f3bf9d642`, same channel (verified on 2026-11-09) | see the section below |
| `hinavmesh/testdata/01af558d-….navmesh.blob` | Navigation mesh of the Forge asset `01af558d` (the map Isolation) | `navmesh.blob` of the UGC asset `01af558d-53ab-4f05-ba68-92d805fc6260`, same channel (verified on 2026-11-09) | see the section below |
| `hinavmesh/testdata/df7dbf08-….navmesh.blob` | Navigation mesh of the Forge asset `df7dbf08` (the map Kiken'na) | `navmesh.blob` of the UGC asset `df7dbf08-b8de-4ade-9d7f-1947128c9ae4`, same channel (verified on 2026-11-09) | see the section below |
```

## T6. `libs/gamefiles/NOTICE.md`, nouvelle section avant « Rule of ADR 0041 » (remplace les deux puces « Origin to be stated… »)

```markdown
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
```

Puce de conformité de la section « Rule of ADR 0041 », à remplacer par :
« Compliance: no file of this module was read from the installation: the five game-content fixtures are files of the
Halo UGC service, an explicit exception of D5, not a case the rule covers. »

`libs/gamefiles/README.md` (lignes 31-34) : « game content kept for the tests only » devient « public files of the Halo
UGC service, game content kept for the tests only ».

## T7. `NOTICE` racine, section 4, point « Test fixtures that carry game content… » (remplacé en entier)

```
  - Test fixtures that carry game content, kept in the tree only because tests read them: three map
    variants (libs/gamefiles/mapvar/testdata/cliffhanger_map.mvar, cliffhanger_ridgeline.mvar,
    vagabond_fo08_wetland.mvar) and two navigation meshes
    (libs/gamefiles/hinavmesh/testdata/*.navmesh.blob), all five being files of the public Halo UGC
    service (origin and hashes in libs/gamefiles/NOTICE.md), the reference map image
    libs/himap/himap/testdata/carte_validee_v1.png and the callout zones dump
    apps/go-api/internal/testfixtures/testdata/cartes/callout_zones_ridgeline_clipped.json. The
    content is Microsoft's, or that of the authors of the maps; it is kept for the tests and is not
    licensed for any other use.
```

## T8. `NOTICE` racine, section 4 : la phrase finale « Numeric account identifiers (XUID) are still present. » devient

```
    Numeric account identifiers (XUID) and gamertags in the test fixtures and test sources of this
    repository and of its dedicated repositories are invented since version 8.1.2 of LevelUp and
    version 0.1.1 of those repositories (registers: libs/film/testdata/IDENTITES.tsv and
    apps/go-api/cmd/anonymise-gamertags/testdata/IDENTITES_APPS.tsv). Earlier commits in the git
    history still carry real ones.
```

## T9. `libs/haloapi/NOTICE.md` (nouveau, écrit seulement une fois la pseudonymisation passée)

```markdown
# NOTICE — origin of the data files of `libs/haloapi`

This module is licensed AGPL-3.0-only (see `README.md`). It carries two test fixtures that are not source
code. Nothing here is Microsoft, 343 Industries or Halo Studios software; the module is not affiliated
with them.

- `infinite/testdata/leaderboard_sample.html`: a page of the Halo Waypoint leaderboard (Ranked Snipers,
  season `csrseason13-2`) captured by the maintainer on 2026-06-08. The page is Microsoft's content and is
  kept for the tests only. The gamertags and Xbox identifiers of the players it lists were replaced by
  invented ones before this version, which keeps every length and every position.
- `infinite/testdata/halo/match_history_response.json`: written by hand, with invented match identifiers.
```
