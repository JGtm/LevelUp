# ADR 0041 — Go modules under `libs/`, linked to the application by `go.work`

Status: accepted (2026-10-28); implementation tracked in
`.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D1_2026-10-28.md` (lot 3 creates the first module, `libs/gamefiles`).
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
`mapvar`); `libs/himap` follows in lot 4.

**D2 — The workspace.** The `go.work` at the root lists `./apps/go-api` and every `./libs/<brick>`;
`go.work.sum` is versioned. There is **no `replace` directive** and **no `require` line for a
workspace module** in `apps/go-api/go.mod`: a `require` on an unpublished version
(`v0.0.0-00010101000000-000000000000`) makes `go mod download` and package loading fail ("unknown
revision") even in workspace mode, while without it the workspace resolves the import. The price is
that `go mod tidy` can no longer run inside `apps/go-api` (it resolves the sibling module on the
network and fails): dependency bumps of a brick are made in its own `go.mod`, and `go.mod` of the
application only changes for its own dependencies. The `go` line is **identical** in `go.work`,
`apps/go-api/go.mod`, every `libs/*/go.mod` and the `Dockerfile` image
(`archlint/dockerfile_go_version_test.go`).

**D3 — Autonomy and import rule.** A brick imports nothing from `levelup/go-api` (the compiler
enforces it for `internal/` packages, tests included) and only the bricks of lower rank listed in
its table row. The table is `archlint/briques_imports_test.go`: brick → packages (directories
relative to the repository root) + permitted imports of other bricks; any import of the
`levelup/go-api` module or of another `libs/` module outside the row is a failure, tests included,
with an allowlist that must stay empty or dated and whose dead entries fail. Rank today: `gamefiles`
imports nothing from the repository; `himap` (until lot 4 inside `apps/go-api`) imports `gamefiles`
only.

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
(`TestLeParcoursFranchitLeModuleGamefiles` fails if it stops doing so).

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
not as web changes. The test-baseline and coverage ratchets stay on the `apps/go-api` module
(`go list ./...` is the reference set; a package that leaves the module leaves the baseline in a
dated `RETRAIT` entry of `scripts/check_test_baseline.sh`). `archlint` ratchets with a general rule
(`gamefiles_tag`, `research_tag`, `pas_de_lecture_sous_ai`, `todo_expiry`, `no_mojibake`,
`no_accidental_goos_suffix`, `doc_chemins_ai`, `briques_imports`, the Go version check) are rooted
at the workspace through `workspaceGoSources`; ratchets about the film, SQL or `go-api` layers stay
on `apiModuleRoot`.

**Adding a brick** costs: the `go.mod` (same `go` line), a `use` line in `go.work`, two `COPY` lines
in the `Dockerfile`, the steps of `ci.yml` (vet/test, lint, `govulncheck`), a `gomod` block in
`.github/dependabot.yml` (checked by `TestModulesLibsSurveillesParDependabot`), a row in the
`briques_imports_test.go` table, the module in `racinesEspaceDeTravail` (`archlint/go_source_walk_test.go`)
and in `make libs-test`.

## Consequences

- `cd apps/go-api && go test ./...` no longer exercises the bricks: `make libs-test`, the `go-build`
  job and `make gate-push` do. A contributor who changes a brick runs both.
- `go mod tidy` is unavailable in `apps/go-api`; Dependabot's `gomod` block for that directory may
  fail to resolve the workspace modules (observed risk, not yet measured on GitHub): the answer, if
  it does, is a `replace` directive or an ignore, a new decision.
- The vendored `ooz` stays GPLv3 inside a module that is meant to be importable: the module carries
  that licence note (`libs/gamefiles/ooz/NOTICE.md`) and a consumer that imports `ooz` takes it on.
- The monorepo test baseline loses the entries of the packages that move (here 48 pairs of the
  `mapvar` package); their tests keep running in the module's own CI step.

## References

- Plan: `.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D1_2026-10-28.md`; audit: `.ai/AUDIT_DECOUPLAGE_BRIQUES_2026-10-27.md`.
- Ratchets: `apps/go-api/internal/archlint/briques_imports_test.go`, `dockerfile_go_version_test.go`,
  `go_source_walk_test.go`, `gamefiles_tag_test.go`; `apps/go-api/cmd/server/licence_gplv3_test.go`.
- Module: `libs/gamefiles/`; workspace: `go.work`; commands: `docs/COMMANDS.md` § "Go modules under `libs/`".
