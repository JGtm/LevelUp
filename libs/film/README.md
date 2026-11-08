# github.com/JGtm/LevelUp/libs/film

Decoder for Halo Infinite Theater films: it reads the chunks of a film (the recording the game keeps
of a match), recovers the facts they carry (players, kill feed, weapons, objectives, positions) and
assembles them into a 2D replay document.

## Status

Experimental. The API is not stable before v1.0.0; minor versions may break. Published from the
LevelUp monorepo as a Go module in a subdirectory; versions are tagged `libs/film/vX.Y.Z`.

## What it is

One module, five layers, each one reading only the layer below (ADR 0034):

| Layer | Role |
|---|---|
| `source` | the only door to a film's bytes: loads the chunks of a directory, or of a `Source` you provide, and inflates them |
| `profile` | the immutable decoding profile of a build of the game: one table per build, each line carrying its proof |
| `grammar` | the reading of the film's bit stream: records, entities, key frames, players, kill feed, weapon use |
| `facts` | what the grammar says about a match: kill sources, objectives, scores, and the documented fallbacks with their counters |
| `replay` | the publication layer: the `ReplayDocument` (tracks, events, scores, vehicles, equipment); its schema version is `replay.SchemaVersion` |

The four lower layers live under `internal/`, so the compiler closes them. What a caller imports:

- `decfilm`, the facade: it re-exports what callers compile (`LoadDir`, `ScanDeaths`, `Decode`,
  `NamedEvents`, `StatRecords`, `LoadMapQuantCatalog`, ...) and decodes nothing itself.
- `replay`: `BuildFromFilm` and `BuildFromFacts` turn a loaded film, or facts decoded earlier, into a
  `ReplayDocument`. The document is the public output.
- the shared vocabulary: `types`, `highlightevent`, `objectiveevent`, `playerposition`, `identity`.
- the embedded name tables, offline and with no game file: `damagetag` (damage source of a death),
  `killicon` (kill-feed icon of a source), `medalname` (medal identity), `filmshell` (weapons as the film
  names them), `equipenergy` (equipment energy).
- `filmcache` (read side of an on-disk film cache), `filmobs` (counters), `memguard` (a memory ceiling
  for one decode, with the caller's callback), `revision` (layer fingerprints), `finalise`.

## What it is not

- No download: it does not fetch films. The bytes come from your own storage; a client for the Halo web
  services is a separate module (`libs/haloapi`).
- No application storage: it decides neither where a film, a decoded fact or a document is kept, nor in
  what database.
- No rendering: the replay document is data (tracks, events, bounds); drawing it is the caller's job.
- No scheduling and no priority: the module does not decide when a film is decoded, how many at once, in
  what order or at what process priority. `memguard` measures the heap and calls your callback; what to do
  at the ceiling is yours.
- No game data: the film bytes are yours. The module ships none of the game's files, only the small name
  tables listed in `NOTICE.md`.

## Install

```
go get github.com/JGtm/LevelUp/libs/film@v0.1.0
```

Works from any Go module; no `replace` directive is needed. (Go resolves `v0.1.0` to the subdirectory tag
`libs/film/v0.1.0`.) The module requires `libs/gamefiles` and `libs/himap` at their `v0.1.0` tags; `go get`
brings them in.

## Example

From `decfilm/example_test.go`, compiled and run by `go test` (its output is checked):

```go
film, err := decfilm.LoadDir("../replay/testdata/minifilm_000d5950", nil)
if err != nil {
	log.Fatal(err)
}
deaths, err := decfilm.ScanDeaths(film)
if err != nil {
	log.Fatal(err)
}
victims := map[uint64]bool{}
for _, d := range deaths {
	victims[d.XUID] = true
}
fmt.Printf("%d deaths, %d distinct victims\n", len(deaths), len(victims))
// Output: 93 deaths, 8 distinct victims
```

The directory is a minimal film that ships with the module's tests: a few chunks of a real match, in which
every player identifier is fictitious (see `NOTICE.md`). A full film is a directory of chunk files; pass the
manifest's `[]decfilm.ChunkMeta` instead of `nil` when you have it.

To go from a film to a replay document, pass the loaded film to
`replay.BuildFromFilm(ctx, matchID, titleSlug, film, replay.Options{MapQuant: &entry})`, where `entry` is the
map's line of the map-bounds catalogue read with `decfilm.LoadMapQuantCatalog(path)`. That catalogue (the
world bounds and quantisation of each map) is yours: the module does not embed it, and a map it does not
list is refused with `decfilm.ErrUnknownMapBounds`. The optional map props and structural footprints of the document
(`replay.Options.Geometry` and `Structure`) also come from your own map data (see `libs/himap`).

## Requirements

- Go 1.26.10 (`go.mod`).
- No cgo to build or to use it: `CGO_ENABLED=0 go build ./...` and `go vet ./...` pass, no production
  package links C code. A few tests import `libs/himap`, which reaches the cgo `ooz` decoder through
  `libs/gamefiles`; running them needs `CGO_ENABLED=1` and a C++ compiler (see the `libs/gamefiles`
  README).
- Catalogues are provided by the caller: the map-bounds catalogue above, and the map data of the options.
  A film whose map has no entry is set aside, never decoded with another map's widths: `decfilm.Decode`
  returns `decfilm.ErrCarteAbsente`, and the position scans return `decfilm.ErrUnknownMapBounds`. The kill
  feed alone (`ScanDeaths`, as in the example) needs no catalogue.
- The `research/` packages and the files tagged `research` are the maintainer's instruments. They read a
  local corpus of films that is not in this repository and build only with `-tags research`. They are not
  an API.

## Layer revisions and unknown builds

Each of the five layers has a revision constant, the fingerprint of its production sources (comments and
tests do not count): `source-...`, `profile-...`, `grammar-...`, `killsource-...`, `objectives-...`. A
change to a layer's behaviour changes its revision, committed golden files pin them, and a consumer that
stores decoded facts can compare the revision it stored with the current one (`decfilm.Rev` is the
kill-feed revision) to know what to decode again. `revision` is the mechanism; `go test ./revision/` plays
it.

A film of a build, or of a chunk format, that the profile table does not list is never read as the nearest
build: the decoder returns an error that names the build (`ErrUnknownBuild`, wrapped with it), counts it
(`decfilm.UnknownBuildExpvarPairs`) and leaves the film aside. Supporting a new build means adding a
measured line to the profile table, with its proof.

Both typed errors are exported by the facade, `decfilm.ErrUnknownBuild` and `decfilm.ErrUnknownFormat`
(a chunk format the table does not list), so a caller tests them with `errors.Is`. `replay.CleDuFilm` (or
`replay.CleDeChunk0`, from the bytes of `chunk_00`) returns the key a film writes and, in its `Err` field,
the error the table opposes to it; `decfilm/erreurs_typees_test.go` plays it on a build and a format that
the table does not know.

## License

AGPL-3.0, the repository license (`LICENSE`, `NOTICE` at the repository root). The module itself imports
no GPL code. Its tests reach `libs/gamefiles/ooz` (GPL-3.0) through `libs/himap`, so a binary that links
those test dependencies is GPL-3.0-compatible only; see `libs/gamefiles/ooz/NOTICE.md`. The origin of the
embedded tables and of the test films is in [`NOTICE.md`](NOTICE.md). This is a summary, not legal advice.

## Contributing

Issues and pull requests on `github.com/JGtm/LevelUp`, with the path prefix `libs/film/` in the title.
From the module directory: `go vet ./... && go test ./...`. The repository convention is French-language
code comments and many French identifiers, so expect them in the source. Regenerating a fixture or a golden
file needs the maintainer's local film cache and is described in the `PROVENANCE.txt` next to each
fixture; a layer's revision golden is rewritten by its gate (`LEVELUP_UPDATE_<NAME>=1 go test ...`, the
exact command is in the failure message). Design: [ADR 0034](../../docs/adr/0034-film-decoder-profile-and-layers.md)
(the decoder), [ADR 0037](../../docs/adr/0037-film-intermediate-representation.md) (the intermediate
representation), [ADR 0041](../../docs/adr/0041-go-modules-under-libs.md) (modules under `libs/`).

## Not affiliated

Not affiliated with Microsoft, 343 Industries or Halo Studios; use it within the game's terms of service,
on films from your own matches.
