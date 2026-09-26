# Acknowledgements

> French version: [FR/ACKNOWLEDGMENTS.md](FR/ACKNOWLEDGMENTS.md)

LevelUp talks to the Halo services through **its own Go client**, written for this project
(`apps/go-api/`). It does not depend on any third-party Halo API library — there is no such
dependency in `go.mod`, and the sync, the film decoding and the data access are all in-house.

That client exists because other people documented the ground first. The work below is prior
art we learned from, not code we ship:

- **Andy Curtis** ([acurtis166](https://github.com/acurtis166)) for [SPNKr](https://github.com/acurtis166/SPNKr) —
  the reference that mapped out the Halo Infinite endpoints and the Theater film format. Our
  highlight-event reader began as a Go port of `spnkr/film/highlight_events.py` before the
  decoder grew its own grammar.
- **Den Delimarsky** ([dend](https://github.com/dend)) for [Grunt](https://github.com/dend/grunt)
  and [OpenSpartan](https://github.com/OpenSpartan) — community documentation of the Halo
  services, and the import path for OpenSpartan data.
- **Gravemind2401** ([Gravemind2401](https://github.com/Gravemind2401)) for [Reclaimer](https://github.com/Gravemind2401/Reclaimer) —
  its reading of the Halo map file formats is the reference behind our geometry decoder
  (`apps/go-api/internal/himap/`: SBSP sections, mesh filters, SDDT groups).

If you use this project, consider starring the upstream projects too.
