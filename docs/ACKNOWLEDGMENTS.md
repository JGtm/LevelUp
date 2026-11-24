# Acknowledgements

> French version: [FR/ACKNOWLEDGMENTS.md](FR/ACKNOWLEDGMENTS.md)

LevelUp talks to the Halo services through **its own Go client**, written for this project
(`apps/go-api/`). It does not depend on any third-party Halo API library — there is no such
dependency in `go.mod`, and the sync, the film decoding and the data access are all in-house.

That client exists because other people documented the ground first. The work below is prior
art we learned from, not code we ship, except the data files named below:

- **Andy Curtis** ([acurtis166](https://github.com/acurtis166)) for [SPNKr](https://github.com/acurtis166/SPNKr) —
  the reference that mapped out the Halo Infinite endpoints and the Theater film format. Our
  highlight-event reader began as a Go port of `spnkr/film/highlight_events.py` before the
  decoder grew its own grammar.
- **Den Delimarsky** ([dend](https://github.com/dend)) for [Grunt](https://github.com/dend/grunt)
  and [OpenSpartan](https://github.com/OpenSpartan) — community documentation of the Halo
  services, and the import path for OpenSpartan data.
- **Den Delimarsky** ([dend](https://github.com/dend)) for [OpenSpartan Cartographer](https://github.com/OpenSpartan/cartographer) and [films-ui](https://github.com/OpenSpartan/films-ui) —
  the principles of the Relief background (height-tinted terrain, vector lines, cut by storey); MIT licence, no code copied.
- **Den Delimarsky** ([dend](https://github.com/dend)) for libfilmshell (OpenSpartan, MIT licence) — the tables of the
  score responses, of the medals and of the damage effects, and the payload widths of some film messages, read by the
  film decoder (`halo-infinite-film-decoder`, v1.6.1); the credit headers and licence texts sit next to the tables.
- **llampwall** ([llampwall](https://github.com/llampwall)) for the 3D replay reader built on the film decoder
  (`llampwall/installation-08`) and the readings it brought: gestures, crouch and jump commands, the shooter by control
  place, aims without a position, the weapon in hand, the ammunition carried, positions at the film's full rate,
  projectiles that stick, equipment pads. They were written again under the decoder's rules (v1.6.1); no line of the
  fork is copied.
- **nuzzles** (the Rust parser `halo_api`) — the method of reading the chunks of a film on several threads, behind the
  decoder's thread option (v1.6.1, replay generation two to three times faster); no code taken.
- **Gravemind2401** ([Gravemind2401](https://github.com/Gravemind2401)) for [Reclaimer](https://github.com/Gravemind2401/Reclaimer) —
  its reading of the Halo map file formats is the reference behind our geometry decoder
  (`halo-infinite-map-images/himap/`: SBSP sections, mesh filters, SDDT groups).
- **Gamergotten** ([Gamergotten](https://github.com/Gamergotten)) and the contributors of
  [Infinite Runtime Tag Viewer](https://github.com/Gamergotten/Infinite-runtime-tagviewer), with
  **Lord Zedd** and **Exhibit** who dumped the tag definitions it carries: the tag layout files
  `halo-infinite-map-images/himap/sbsp.xml` and the two `weap.xml` of `apps/go-api/cmd/` are unmodified copies of
  its `Plugins/` files (see `NOTICE`).

If you use this project, consider starring the upstream projects too.
