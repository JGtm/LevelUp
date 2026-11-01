# ADR 0043 — Publishing the bricks: Go modules, READMEs, versioning and the repository-split criterion

Status: accepted (2026-11-01); implementation tracked in `.ai/V8/PLAN_PUBLICATION_BRIQUES_2026-10-30.md`
(lots 0 to 3 move the code, lot 4 writes the READMEs and this ADR).
Source: decision of the maintainer of 2026-10-30 (publish the Halo API and the map generator, keep the film decoder
in the repository), confirmed on 2026-11-01 (launch, repository-split criterion and mode).
Amends: ADR 0041 (a fourth module, `libs/haloapi`; tags and the `require` order), ADR 0042 (the brick covers Halo 5,
Xbox Live achievements, UGC map assets and GameCMS files, and is a module).
Relates to: ADR 0023 and 0038 (tokens stay in LevelUp), ADR 0031 (data-source boundary), ADR 0012 (title adapters
under `games/<slug>/`).

## Vocabulary

- **Publishable brick**: a module under `libs/` that carries a README and is meant to be fetched by `go get` from
  another repository. Three: `libs/gamefiles`, `libs/himap`, `libs/haloapi`.
- **Tag**: a git tag `libs/<brick>/vX.Y.Z`, Go's form for a module in a subdirectory (ADR 0041 D4). Pushing it is the
  act of publication: the module proxy serves that version from then on.
- **Exploitation layer**: what decides who calls a service, when, with whose token, where the result is kept and
  what it means (ADR 0042). It stays in LevelUp.
- **Split**: moving a brick into a repository of its own, history kept.

## Context

ADR 0041 made the bricks modules and ADR 0042 cut the Halo API into protocol and exploitation, but neither published
anything: no README, no tag, and the Halo API brick covered only the Halo Infinite services. Four other HTTP clients to
Halo and Xbox services still sat outside it, each mixing protocol and exploitation:

- `games/halo_5/client.go`: the Halo 5 services (stats, carnage, events, profiles, appearance renders);
- `mapcatalog/ugc.go`: the Discovery UGC calls for map variants;
- `assets/fetcher_gamecms.go`: the GameCMS file and image downloads;
- `sync/xbox_client.go`: the Xbox Live achievements.

The success criteria of the plan are: (1) `go list -deps -test ./...` in `libs/haloapi` shows no `levelup/go-api`
package; (2) the four clients above no longer exist as clients, their protocol being in `libs/haloapi` and their
exploitation in LevelUp; (3) no observable change (same URLs byte for byte for sound identifiers, same retry
policies, same `netguard` surfaces, same domain values); (4) a README per brick and a tag per brick, the tags set by
the maintainer; (5) CI green and the ratchets coherent; (6) the repository-split decision stays open, with its
criterion written (D-5).

The four clients were cut the way ADR 0042 cut the first two. Where each went:

- Halo 5 -> package `halo5` (`New(doerJSON, doerBinaire, retry)`, `Endpoints`, `Auth` per call, raw DTOs); the
  adapter stays in `games/halo_5`.
- UGC maps -> a service of `infinite` (`FetchCarteUGC`, `TelechargerFichierUGC`, `Identite`); `mapcatalog` keeps the
  choice of the variant.
- GameCMS -> a service of `infinite` (`URLFichierGameCMS`, `URLImageGameCMS`, `TelechargerFichierGameCMS`); `assets`
  keeps resolution, cache and the error taxonomy.
- Xbox -> package `xboxlive` (`GetSuccesPage`, one page per call, raw DTOs); `sync` keeps the pagination loop, the
  conversion and its interface.

"No observable change" is proved, not asserted: a frozen-URL test per client (`*_url_figees_test.go` in each client's
package: URL, every header set and every header not set, for sound identifiers) was written on the old code and passes
on the new without a modified expectation; the read-only live tool (`cmd/diag_api_lecture`, one sub-command per
surface) returned the same bytes (SHA-256) before and after; the five `netguard` surface names are unchanged. The only
differences are those of D-8.

## Decision

**D-1 — Three publishable modules, from the monorepo, no split now.** `libs/gamefiles` (`himodule`, `ooz`, `hinavmesh`,
`mapvar`), `libs/himap` (`himap`, the map-image generator) and `libs/haloapi` (the root package `haloapi`: `Doer`,
`Garde`, `ErrDefinitif`, `Retry`, `Limiter`, `ParseRetryAfter`, `Masque`; and `xblauth`, `infinite`, `halo5`,
`xboxlive`, `haloapitest`). They are published as they sit, from this repository, in this workspace. `libs/film`
(the film decoder, ADR 0034) is a module of the workspace and **not** a publishable brick: no README, no tag, no
promise. It may change shape freely.

**D-2 — Versioning.**

- A tag per module, `libs/<brick>/vX.Y.Z`, set by the maintainer. No lot, agent or workflow creates or pushes a tag:
  a tag publishes. A tag that reached the module proxy is permanent; a mistake is corrected by a new tag and a
  `retract` directive, never by moving the tag. A consumer installs with
  `go get github.com/JGtm/LevelUp/libs/<brick>@v0.1.0`: Go itself resolves the tag `libs/<brick>/v0.1.0` of a module
  in a subdirectory (the form `@libs/<brick>/v0.1.0` would be read as a revision name).
- `v0.x` until a brick is declared stable: **no promise of API stability before `v1`** (the READMEs say so). Each
  module carries its own number; there is no shared version across the bricks.
- A module that imports another brick (`himap` imports `gamefiles`) has no `require` for it today (ADR 0041 D2:
  a `require` on an unpublished version breaks package loading). `go get` of such a module cannot resolve the
  sibling. Before the tag of the upper module, its `go.mod` therefore gains a `require` on the tag of the brick below,
  which is tagged first; the workspace keeps resolving the local copy. Each tag is checked by `go get` from a
  throwaway module outside the repository (plan item 4.2) before it is announced.
- `libs/haloapi` imports no other brick, so it is the one module that can be tagged with no ordering constraint.

**D-3 — One README per brick, one template.** English, one page: what it is; what it is not; install (`go get`);
an example that compiles, as an `example_test.go` of the module (so that `go vet` and `go test` in CI keep it true);
requirements (Go version, CGO for `ooz`, an installed game for the `gamefiles` tests); licence; how to contribute;
a statement that the project is not affiliated with or endorsed by Microsoft, 343 Industries or Halo Studios.

**D-4 — Licences.**

- `libs/gamefiles`: **GPL-3.0** (the terms of the embedded clean-room Oodle decoder `ooz`, powzix/ooz, GPLv3 without
  "or later"; see `libs/gamefiles/ooz/NOTICE.md`). Within the module only `himodule` imports `ooz` (`hinavmesh` and
  `mapvar` do not), but the module is published and licensed as a whole. A consumer that imports `ooz`, directly or
  through `himodule`, takes the GPL on its binary.
- `libs/himap` and `libs/haloapi`: **AGPL-3.0-only**, the licence of the repository (`LICENSE`, root `NOTICE`).
  `himap` reaches `ooz` through `gamefiles/himodule`, so a binary that links it is GPL as well.
- LevelUp's server must never link `ooz`; the server-side ratchet follows imports across every module under `libs/`
  (see Guard-rails).

**D-5 — The repository-split criterion.** A brick leaves for a repository of its own when **any** of these holds:

- (a) an external contributor opened a pull request or an issue on the brick;
- (b) its change cadence became independent of LevelUp (fewer than one cross-cutting pull request per month);
- (c) its CI weighs on LevelUp's.

Mode: `git subtree split` (history kept), repository `JGtm/<brick>`; the module path is kept, or changed with a
transitional `replace` in LevelUp. The exploitation layer stays in LevelUp (D-6). The decision belongs to the
maintainer, is taken last, and is executed together (user decision, 2026-11-01). Until then the question stays open
and this ADR is the only place that states when it closes.

**D-6 — What stays in LevelUp.** The bricks hold the protocol; LevelUp keeps:

- the token store and the refresh policy (`platform/auth`, ADR 0023), the pooling (ADR 0038) and the per-player token
  cache (`platform/auth/tokencache`);
- the `netguard` offline guard, as the `netguard.Doer` it passes to every client of the brick, one surface per
  exit point (`platform/netguard/doer.go`);
- the request budgets and rate limiters;
- the conversion of raw DTOs into `domain` and canonical values, and the adapters that make them
  (`games/halo_5`, `games/halo_infinite/haloadapter`, `mapcatalog`, `assets`, `sync`, `sync/filmfetch`);
- the endpoint singleton `games.DefaultEndpointResolver`, from which the adapters build the brick's `Endpoints`.

**D-7 — `haloapitest` is a public test-helper package.** `libs/haloapi/haloapitest` (the `Enregistreur`, an
`http.RoundTripper` that records method, absolute URL and sorted headers) is importable by users of the module, like
`net/http/httptest`, and falls under the same stability regime as the rest of the module (D-2). It uses the standard
library only, builds no HTTP client, and no production code imports it.

**D-8 — Differences of form accepted at the cuts.** Three, observable only in the offline demo mode or on an error
path, and documented in the `doc.go` of the packages concerned:

- the guard is consulted before the limiter and before the URL is built, once at the start of each call (the Halo 5
  and Xbox clients used to consult it after the limiter or after building the URL): a refused exit consumes no limiter
  token, and an invalid identifier in demo mode returns the guard's refusal rather than the construction error;
- the `Retry-After` header is read as seconds **and** as an HTTP date for Halo 5 (it used to be seconds only), then
  capped by the policy as before;
- a guard refusal is wrapped `haloapi.ErrDefinitif` (no retry, no wait), with `errors.Is(err, netguard.ErrOffline)`
  preserved and the message unchanged; the error is no longer equal to the sentinel by `==`.

## Consequences

- **CI per module.** `libs/haloapi` has: a `go-build` step on both operating systems with `CGO_ENABLED=0` (vet, tests,
  and `go list -deps -test` refusing `levelup/go-api`); a fifth `golangci-lint` pass; `govulncheck`; the lot `haloapi`
  under `module libs/haloapi` in `.github/ci/go-shards.txt`; its own test baseline,
  `.ai/baselines/tests_libs_haloapi.jsonl`; a `gomod` block in `.github/dependabot.yml`; its `go.sum` in the cache
  paths of the workflows.
- **Lint.** The configuration is shared (`apps/go-api/.golangci.yml`). One dated rule is specific to the brick: the
  `Id` field-name remark is silenced on the three Halo 5 DTO files only, because the fields carry the service's JSON
  names.
- **What a consumer gets.** Typed clients that build no HTTP client: the consumer passes a `haloapi.Doer`, the
  `Endpoints`, the tokens per call, the language and a `Retry` value, and may pass a `Limiter`. A `Doer` that also
  implements `haloapi.Garde` is consulted before any wait. Raw DTOs, raw bytes and typed errors come back; nothing is
  interpreted or cached. The film bytes come back compressed.
- **Cost of publishing.** A tagged exported name is a name to keep or to retire with a new minor version. Under `v0`
  that cost is accepted; the first `v1` is a separate decision.
- **Deferred, noted, not decided here** (numbers refer to the discoveries of the plan):
  - the Halo 5 render redirect (302 to `image.halocdn.com`) re-sends the Spartan token to the CDN; this is the
    behaviour from before the cut, frozen by the frozen-URL test, and is to be reviewed before the brick is promoted
    beyond `v0` (PUB-2.5-e);
  - the brick's baseline is checked in CI only; `make libs-test` runs its tests without confronting the baseline
    (PUB-3-a);
  - whether `haloapitest` is public (PUB-3-e) is settled by D-7;
  - **open point, to settle before the tag of `himap`**: `libs/himap/go.mod` carries no `require` of `libs/gamefiles`,
    which `himap` imports (ADR 0041: `go.work` without `require`). Outside the workspace, `go get` of `himap` does
    not resolve `gamefiles`. The mode is the maintainer's to choose; the one D-2 describes is a `require` on the
    tagged version of `gamefiles`, added after its tag and before the tag of `himap`. Not done by this ADR.
- **Not covered.** No ratchet requires a README or a tag; the examples are kept true by compiling, the tags by the
  maintainer.

## Guard-rails

In `apps/go-api/internal/archlint/` unless a path says otherwise.

- `briques_imports_test.go` (`TestBriquesImports`): no file of a brick, tests included, imports `levelup/go-api` or a
  `libs/` module outside its row; the `haloapi` row permits no other brick and bounds third parties per sub-tree
  (`errgroup` and `rate` under `infinite` only). `briques_garde_test.go`
  (`TestChaqueDossierGoDeLibsEstDansLaTable`): a Go directory under `libs/` that the table does not name fails.
- `haloapi_fermeture_domaine_test.go` (`TestFermetureDeDomainEtAnalysisSansHaloapi`): the dependency closure of
  `domain`, `analysis` and `games/canonical` does not contain the brick.
- `haloapi_sans_client_http_test.go` (`TestHaloapiNePossedeAucunClientHTTP`): the brick names no `http.Client`,
  `http.Transport`, `http.DefaultClient`, `http.DefaultTransport`, `http.Get/Post/Head/PostForm` and calls no
  `.RoundTrip(` in production code; every byte leaves through the `Doer` it is given.
- `haloapi_doer_garde_test.go` (`TestBriqueInfiniteRecoitLeDoerGarde`): every `Doer` that go-api's production hands to
  `infinite.New`, `infinite.NewClassement`, `halo5.New` or `xboxlive.New` is a literal `netguard.Doer{...}`.
- `haloapi_sans_etat_test.go` (`TestXblauthSansVariableDePaquet`, `TestHaloapiSansEnvNiFichier`): no package variable
  in `xblauth` (sentinel errors and interface assertions excepted); the brick reads neither the environment nor a file.
- `haloapi_parametres_test.go` (`TestHaloapiCinqParametresAuPlus`): no function of the brick takes more than five
  parameters.
- `haloapi_masque_unique_test.go` (`TestHaloapiMasqueUnique`): the masking of a secret has one definition,
  `haloapi.Masque`.
- `enregistreur_http_unique_test.go` (`TestEnregistreurHTTPUnique`): a test file that declares a `RoundTrip` must be
  under `libs/haloapi/haloapitest/` or in the dated allowlist, which only goes down.
- `no_halowaypoint_literal_test.go` (`TestNoNewHalowaypointLiteral`): the Halo host name appears only in the boundary
  files of its allowlist, among them the address files of `infinite` and `halo5`.
- `no_raw_film_bytes_outside_source_test.go` (`TestAucuneLectureDOctetsBrutsHorsDeLaSource`): `libs/haloapi` is among
  its roots, so the brick cannot inflate or read a film.
- `dockerfile_go_version_test.go` (`TestDockerfileGoImageMatchesGoMod`, `TestGoVersionIdentiqueDansLEspaceDeTravail`,
  `TestModulesLibsSurveillesParDependabot`): one `go` line across the workspace and the image, and a Dependabot block
  for every module under `libs/`.
- `ci_go_shards_test.go` (`TestCIGoShardsDefinitionAndMatrixAgree`, `TestCIGoShardsPartitionIsExact`): the lots of
  `go-shards.txt` and the CI matrix agree, and every package of each module is in exactly one lot.
- `cmd/server/licence_gplv3_test.go` (`TestServeurNeLinkePasDeCodeGPL`, `TestLeParcoursFranchitLeModuleGamefiles`,
  `TestLaChaineDeHimapAtteintOoz`): `cmd/server` does not link `libs/gamefiles/ooz`, the walk crosses the module
  boundary, and it sees the `himap -> himodule -> ooz` chain.
- `platform/netguard/netguard_coverage_test.go` (`TestOutboundCallsAreNetguarded`): a file of go-api's own code that
  sends a request must name the guard; the brick is outside its walk, and the guard reaches it through the `Doer` that
  `haloapi_doer_garde_test.go` requires.

## References

- Plan: `.ai/V8/PLAN_PUBLICATION_BRIQUES_2026-10-30.md`; survey of the four clients:
  `.ai/V8/RELEVE_PUBLICATION_BRIQUES_LOT0_2026-11-01.md`.
- ADR 0041 (modules under `libs/`), ADR 0042 (protocol vs exploitation), ADR 0034 (film decoder, not published),
  ADR 0023 and 0038 (tokens).
- Code: `libs/gamefiles/`, `libs/himap/`, `libs/haloapi/` (`doc.go` of each package), `go.work`,
  `.github/ci/go-shards.txt`, `.github/dependabot.yml`.
