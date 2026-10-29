# ADR 0041 — Go modules under `libs/`, linked to the application by `go.work`

Status: accepted (2026-10-28); implementation tracked in
`.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D1_2026-10-28.md` (lot 3 creates the first module, `libs/gamefiles`; lot 4 the second, `libs/himap`)
and `.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D2_2026-10-28.md` (lot 3 creates the third, `libs/film`, 2026-10-29).
Source: `.ai/AUDIT_DECOUPLAGE_BRIQUES_2026-10-27.md` (decision 1 and the addendum of 2026-10-28).
Relates to: ADR 0012 (title-specific adapters live under `games/<slug>/`), ADR 0034 (film decoder,
its façade and layers), ADR 0011 (canonical / semantic / asset-URL boundary).

## Vocabulary

- **Brick**: a body of code that is autonomous by nature (it reads a file format, decodes a stream,
  speaks a protocol) and that LevelUp uses but does not define. Three are identified: the game-file
  readers and map images (D1), the film decoder (D2), the Halo API protocol (D3).
- **Workspace**: the set of Go modules listed in the `go.work` at the repository root.
- **Autonomy criterion**: `go list -deps ./...` run in a brick's module shows no package of the
  `levelup/go-api` module.
- **Game-extracted file**: any byte sequence read out of the installed game (module archives, memory
  captures, texture or navigation dumps).

## Context

Every import added between a brick and the rest of LevelUp was local and reasonable, and nothing
refused it; the audit found the bricks tied to the application by a handful of packages (`i18n`,
`testfixtures`, `testutil`, `film/replay`) and by sixteen test files of the map-image package. A
package under `apps/go-api/internal/` can only be imported from `apps/go-api`: that is the Go rule
for `internal/`, and it also holds for tests. A brick that must be importable from another
repository, or simply lifted out one day, therefore has to be a module of its own, and the
repository has to learn to hold several modules: the Docker image, the CI jobs, the Makefile, the
linter and the `archlint` ratchets all assumed one module rooted at `apps/go-api`.

## Decision

**D1 — Form.** One Go module per brick, under `libs/<brick>/`, in the same repository. Its packages
sit at the first level of the module (`libs/gamefiles/himodule`, never `libs/gamefiles/internal/...`:
`apps/go-api` could not import them). The module path is
`github.com/JGtm/LevelUp/libs/<brick>`, resolvable by `go get` once a tag exists; the application
keeps `levelup/go-api`. The first module is `libs/gamefiles` (`himodule`, `ooz`, `hinavmesh`,
`mapvar`); the second is `libs/himap` (package `himap`, the map-image brick: `github.com/JGtm/LevelUp/libs/himap`);
the third is `libs/film` (the film decoder, `github.com/JGtm/LevelUp/libs/film`: the facade `decfilm`, the
publication layer `replay`, its catalogues and output types at the first level, and the closed layers of
ADR 0034 D-1 under `libs/film/internal/` — the `internal/` rule now closes them to LevelUp at the module
boundary, which is what D-1 asks).

**D2 — The workspace.** The `go.work` at the root lists `./apps/go-api` and every `./libs/<brick>`;
`go.work.sum` is versioned. There is **no `replace` directive** and **no `require` line for a
workspace module** in `apps/go-api/go.mod`: a `require` on an unpublished version
(`v0.0.0-00010101000000-000000000000`) makes `go mod download` and package loading fail ("unknown
revision") even in workspace mode, while without it the workspace resolves the import. The price is
that `go mod tidy` can no longer run inside `apps/go-api` (it resolves the sibling module on the
network and fails): dependency bumps of a brick are made in its own `go.mod`, and `go.mod` of the
application only changes for its own dependencies. The same holds for a brick that imports another
brick (`libs/himap` imports `libs/gamefiles`; `libs/film` imports both): its `go.mod` carries no `require`
for the sibling, `go mod tidy` fails there too, and its requirements (`golang.org/x/sys`, reached through
`libs/gamefiles/himodule`, recorded `// indirect` at the version of `apps/go-api`; for `libs/film`,
`github.com/pelletier/go-toml/v2` and `golang.org/x/sys`, its own imports) are kept by hand;
`make tidy` in `apps/go-api` names such a module and skips it. The `go` line is **identical** in `go.work`,
`apps/go-api/go.mod`, every `libs/*/go.mod` and the `Dockerfile` image
(`archlint/dockerfile_go_version_test.go`).

**D3 — Autonomy and import rule.** A brick imports nothing from `levelup/go-api` (the compiler
enforces it for `internal/` packages, tests included) and only the bricks of lower rank listed in
its table row. The table is `archlint/briques_imports_test.go`: brick → packages (directories
relative to the repository root) + permitted imports of other bricks; any import of the
`levelup/go-api` module or of another `libs/` module outside the row is a failure, tests included,
with an allowlist that must stay empty or dated and whose dead entries fail. Rank today: `gamefiles`
imports nothing from the repository; `himap` imports `gamefiles` only; `film` imports `gamefiles` and
`himap` (its production imports only `gamefiles/mapvar`; `himap` is reached by tests), and no brick
imports `film`. The `film` row is a whole tree (`libs/film`, recursive): the module has several dozen
packages.

**D4 — Import from outside.** A module in a subdirectory is tagged `libs/<brick>/vX.Y.Z`; that is
Go's rule for the module path above. Until the first tag the workspace is the only way to build
against a brick; no tag is created by the lots that move code.

**D5 — No game-extracted file in a brick.** A brick contains source, the vendored `ooz` C++ sources
with their `NOTICE.md`, and test fixtures. The fixtures that exist (three `.mvar` map variants in
`libs/gamefiles/mapvar/testdata`, two navigation blobs in `libs/gamefiles/hinavmesh/testdata`,
the fuzz seeds) are reverse-engineering artefacts that were already versioned in the repository
(root `NOTICE`, § test fixtures): they stay, one copy each, owned by the brick that reads them, and
LevelUp reads them by path from the repository root (`testfixtures.MapvarFixturesDir`,
`NavmeshFixturesDir`). Nothing read from the installation (module archives, textures, memory
captures) may be added; the `.mvar` cache beside the fixtures is ignored by git (`.gitignore`).
`ooz` is GPLv3 (powzix/ooz): `cmd/server` must never link it, and the licence ratchet
(`cmd/server/licence_gplv3_test.go`) follows imports across the module boundary into `libs/gamefiles`
(`TestLeParcoursFranchitLeModuleGamefiles` fails if it stops doing so). The walk follows every module
under `libs/`, because the GPL chain of the map-image brick crosses two of them
(`himap` -> `gamefiles/himodule` -> `gamefiles/ooz`): `TestLaChaineDeHimapAtteintOoz` proves the walk sees it
the day a server package imports `himap`.

**D6 — Research instruments.** The reverse-engineering probes of a brick that read `.ai/re_dump` or
data of the host repository are kept as `*_research_test.go` files behind the `research` tag
(`gamefiles && research` when they open the game): out of the default build and of the `gamefiles`
suite, compiled in CI by `go vet`, run on demand. A brick's default tests read only its `testdata/`
and the installed game; a test that needs LevelUp data (objective catalogue, replay caches,
configuration) is a test of LevelUp and lives in `apps/go-api` (`internal/gamefilescorpus/`).

**D7 — Infrastructure.** `./...` never crosses a module boundary, so every job that must cover a
brick says so. The `Dockerfile` copies `go.work`, `go.work.sum` and the `go.mod`/`go.sum` of each
module before `go mod download`, then `apps/go-api/` and `libs/`, and builds from
`/build/apps/go-api`. `ci.yml` runs vet, tests, lint and `govulncheck` in `libs/<brick>` in addition
to `apps/go-api`; its path filter treats `libs/**`, `go.work` and `go.work.sum` as Go changes and
not as web changes. The coverage shards of `go-coverage` belong to a module (`module` directive of
`.github/ci/go-shards.txt`, partition proved module by module by `scripts/ci_go_shard.sh --verifier`) and run
from it. The test baseline is one file per module whose tests run in the shards
(`.ai/baselines/tests_pre_migration.jsonl` for `apps/go-api`, `tests_libs_film.jsonl` for `libs/film`;
`go list ./...` of the module is the reference set; a package that leaves a module leaves its baseline in a
dated `RETRAIT` entry of `scripts/check_test_baseline.sh` and, when it keeps its tests, enters the baseline
of its new module with its import path rewritten). The coverage ratchet stays on `apps/go-api`. `archlint` ratchets with a general rule
(`gamefiles_tag`, `research_tag`, `pas_de_lecture_sous_ai`, `todo_expiry`, `no_mojibake`,
`no_accidental_goos_suffix`, `doc_chemins_ai`, `briques_imports`, the Go version check) are rooted
at the workspace through `workspaceGoSources`; ratchets about the SQL or `go-api` layers stay on
`apiModuleRoot`. Ratchets about the film decoder name its paths from one constant (`racineBriqueFilm`,
`libs/film`) and read them from the workspace root; a table that mixes `go-api` and brick paths speaks
two frames that cannot overlap (`surDisque` / `cheminDeGardeRail`: under `libs/` relative to the
repository root, otherwise relative to `apps/go-api`), and a ratchet that walked all of `go-api` walks
the brick too (`marcherAPIEtBriqueFilm`).

**D8 — A brick that carries content fingerprints (the film decoder).** ADR 0034 dates the decoder's
layers by revisions whose goldens freeze a fingerprint of their sources. A token of an import path
would change with the module path, so `libs/film/revision` hashes in a **historical frame** (plan D2,
decision D2-F): a string literal that starts with the current import path of the brick is hashed under
`levelup/go-api/internal/games/halo_infinite/film`, and the directories of the layers and of the
packages they import are named by their historical location in the hash and in the perimeter goldens.
The move into `libs/film` therefore changed no fingerprint and no revision (no persisted fact went
stale). The order of import specs is part of the token stream and is NOT in the frame: under the new
path, `gofmt` / `goimports` would reorder an import block where a brick import sat among standard
library imports, so the hashed sources concerned keep their historical order in separate groups (a
comment above their `import` says so; the layer gate turns red if a tool regroups them).

**Adding a brick** costs: the `go.mod` (same `go` line), a `use` line in `go.work`, two `COPY` lines
in the `Dockerfile`, the steps of `ci.yml` (vet/test, lint, `govulncheck`), a `gomod` block in
`.github/dependabot.yml` (checked by `TestModulesLibsSurveillesParDependabot`), a row in the
`briques_imports_test.go` table and the module in `racinesEspaceDeTravail`
(`archlint/go_source_walk_test.go`). `make libs-test`, `make go-api-lint` and `make gate-push` loop on
`libs/*/go.mod`: no line to add there; a brick that carries `*_gamefiles_test.go` files adds its line to
`go-api-test-gamefiles` (`TestCibleMakefileGamefilesCouvreLeCorpus`). A brick whose tests run in the coverage
shards (`libs/film`) also has a `module` block in `.github/ci/go-shards.txt` (its lots in the `go-coverage`
matrix) and its own baseline file under `.ai/baselines/`.

## Consequences

- `cd apps/go-api && go test ./...` no longer exercises the bricks: `make libs-test`, the `go-build`
  job and `make gate-push` do. A contributor who changes a brick runs both.
- `go mod tidy` is unavailable in `apps/go-api`; Dependabot's `gomod` block for that directory may
  fail to resolve the workspace modules (observed risk, not yet measured on GitHub): the answer, if
  it does, is a `replace` directive or an ignore, a new decision.
- The vendored `ooz` stays GPLv3 inside a module that is meant to be importable: the module carries
  that licence note (`libs/gamefiles/ooz/NOTICE.md`) and a consumer that imports `ooz` takes it on.
- The monorepo test baseline loses the entries of the packages that move (here 48 pairs of the
  `mapvar` package at lot 3, none for `himap` at lot 4: no test of that package was in the baseline); their
  tests keep running in the module's own CI step. For the film decoder (plan D2, lot 3) the 1 050
  lines (877 pairs) of its packages moved to `.ai/baselines/tests_libs_film.jsonl`: no test left the presence
  check.
- `TestAucuneAutreCopieDuCheminDuJeu` (one copy of the game's install path in the whole workspace, the source
  being `libs/himap/himap/deploy_root.go`) moved from the brick to `archlint`: a guard run from inside a
  brick would only see `libs/`.

## References

- Plan: `.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D1_2026-10-28.md`; audit: `.ai/AUDIT_DECOUPLAGE_BRIQUES_2026-10-27.md`.
- Ratchets: `apps/go-api/internal/archlint/briques_imports_test.go`, `dockerfile_go_version_test.go`,
  `go_source_walk_test.go`, `gamefiles_tag_test.go`; `apps/go-api/cmd/server/licence_gplv3_test.go`.
- Plans: `.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D2_2026-10-28.md` (film decoder).
- Modules: `libs/film/`, `libs/gamefiles/`, `libs/himap/`; workspace: `go.work`; commands: `docs/COMMANDS.md` § "Go modules under `libs/`".
