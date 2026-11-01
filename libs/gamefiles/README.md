# github.com/JGtm/LevelUp/libs/gamefiles

Readers for the files of a local Halo Infinite installation: the `.module` archives, the Kraken
block decoder they need, the Forge map variants (`.mvar`) and the navigation meshes.

## Status

Experimental. The API is not stable before v1.0.0; minor versions may break. Published from the
LevelUp monorepo as a Go module in a subdirectory; versions are tagged `libs/gamefiles/vX.Y.Z`.

## What it is

- `himodule`: opens a `.module` archive (header `mohd`, version 53) memory-mapped, lists its
  entries by tag group (`Files`), returns a tag's decompressed bytes (`Extract`) and its resource
  blob (`ResourceBlob`); follows the `_hd1` companion file.
- `ooz`: `Decompress`, the Oodle-compatible decoder for Kraken, Mermaid, Selkie and Leviathan
  blocks, vendored from the clean-room `ooz` project (C++ through cgo).
- `mapvar`: `Parse` decodes a Forge map variant (`.mvar`) into placed objects, positions, team
  indexes, label hashes and shapes; `DecodeRoot` reads the underlying Bond CompactBinary v2
  container; `NewVolume` gives a `Contains` test for an object's shape.
- `hinavmesh`: `Decode` turns the `navmesh.blob` that ships with a Forge map into a polygon mesh
  in world coordinates. Pure Go.

## What it is not

- No map catalogue: nothing here knows which module, identifier or name belongs to which map.
- No installation discovery: you pass file paths to `himodule.Open`. (The sibling `libs/himap`
  has a `DeployRoot` helper if you want one.)
- Not a generic tag decoder: `himodule` hands you raw tag bytes; only the `.mvar` container and
  the navmesh are interpreted here, as far as they were measured.
- No game data: the module ships code and small test fixtures, not the game's assets. You
  supply the installation.

## Install

```
go get github.com/JGtm/LevelUp/libs/gamefiles@v0.1.0
```

Works from any Go module; no `replace` directive is needed. (Go resolves `v0.1.0` to the
subdirectory tag `libs/gamefiles/v0.1.0`.)

## Example

From `himodule/example_test.go`, compiled by `go vet` (it does not run without the game):

```go
path := filepath.Join("deploy", "pc", "levels", "multi", "ridgeline", "ridgeline-rtx-new.module")

m, err := himodule.Open(path)
if err != nil {
	log.Fatal(err)
}
defer m.Close()

for _, f := range m.Files("sbsp") { // "" lists every entry
	tag, err := m.Extract(f)
	if err != nil {
		log.Printf("tag %08x: %v", f.GlobalID, err)
		continue
	}
	fmt.Printf("%s %08x: %d bytes (%d compressed)\n", f.Group, f.GlobalID, len(tag), f.CompSize)
}
```

## Requirements

Go 1.26.10 (`go.mod`). `himodule` and `ooz` need cgo (`CGO_ENABLED=1`) and a C++ compiler
(developed with mingw-w64 g++; the build flags include `-msse4.1`). `mapvar` and `hinavmesh`
are pure Go. To use the module you need a local Halo Infinite installation; to build it you do
not.

## License

The vendored `ooz` decoder is GPLv3 (Powzix, see [`ooz/NOTICE.md`](ooz/NOTICE.md)), so this
module is a GPL chain: any binary that links `himodule` or `ooz` must be distributed under
GPL-3.0-compatible terms, and `ooz/NOTICE.md` recommends keeping it to offline tooling. The
upstream notice says "GPLv3", not "or later", and this README claims no more. The other code
(`mapvar`, `hinavmesh`, which do not import `ooz`) is AGPL-3.0, the repository license
(`LICENSE`, `NOTICE` at the repository root). This is a summary, not legal advice.

## Contributing

Issues and pull requests on `github.com/JGtm/LevelUp`, with the path prefix `libs/gamefiles/`
in the title. From the module directory: `go vet ./... && go test ./...` (exploratory probes
sit behind the `research` build tag and are not run by default). The repository convention is French-language code
comments and many French identifiers, so expect them in the source. Design:
[ADR 0041](../../docs/adr/0041-go-modules-under-libs.md) (modules under `libs/`).

## Not affiliated

Not affiliated with Microsoft, 343 Industries or Halo Studios; use it within the game's terms
of service, on files from your own installation.
