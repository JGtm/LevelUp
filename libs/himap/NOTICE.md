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
