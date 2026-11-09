# github.com/JGtm/LevelUp/libs/haloapi

Stateless Go clients for the Halo web services (Halo Infinite, Halo 5: Guardians, Xbox Live
achievements) and the Xbox Live authentication chain that feeds them: URLs, headers, retry
policies and raw JSON DTOs, with no storage and no decisions.

## Status

Experimental. The API is not stable before v1.0.0; minor versions may break. Published from the
LevelUp monorepo as a Go module in a subdirectory; versions are tagged `libs/haloapi/vX.Y.Z`.

## What it is

- `haloapi` (root): the `Doer` interface (a `*http.Client` satisfies it), the optional `Garde`,
  `ErrDefinitif` / `NonRetryable`, the `Retry` policy value, the `Limiter` interface and
  `ParseRetryAfter`.
- `xblauth`: Microsoft access token -> Xbox Live user token -> XSTS -> Spartan -> Clearance,
  OAuth v2 refresh and authorization code, device code, gamertag -> xuid resolution.
- `infinite`: Halo Infinite services (stats, service record, skill/CSR, career, customization,
  GameCMS, Discovery UGC, film manifests and compressed chunks, public CSR leaderboard pages).
- `halo5`: Halo 5 services (matches, carnage reports, events, profile renders).
- `xboxlive`: Xbox Live achievements, one page per call.
- `haloapitest`: test helper package (an `http.RoundTripper` request recorder). Like the rest,
  it is stable only from v1.

## What it is not

- No token store and no refresh policy: tokens, the Azure client and the title descriptor are
  passed in on every call; nothing reads the environment or a file (`xblauth/doc.go`).
- No HTTP client construction: you pass a `haloapi.Doer` (timeout, transport, redirects and any
  offline guard are yours).
- No rate-limit budget: a `Limiter` is optional and owned by the caller.
- No interpretation of statuses or values: 401, 404, 429 and a zero MMR are returned as they
  come, typed (`*infinite.HTTPError` and its `halo5` / `xboxlive` counterparts).
- No caches, no DTO -> domain conversion, no film decompression (chunks are returned compressed).

## Install

```
go get github.com/JGtm/LevelUp/libs/haloapi@v0.1.0
```

Works from any Go module; no `replace` directive is needed. (Go resolves `v0.1.0` to the
subdirectory tag `libs/haloapi/v0.1.0`.)

## Example

From `infinite/example_test.go`, compiled by `go vet`:

```go
doer := &http.Client{Timeout: 20 * time.Second}
client := infinite.New(doer, infinite.RetryProvider()) // default Halo Infinite endpoints
auth := infinite.Auth{Spartan: "<spartan token>"}

record, err := client.GetServiceRecord(context.Background(), auth, "xuid(2535400000000000)")

var httpErr *infinite.HTTPError
switch {
case errors.As(err, &httpErr):
	// The brick returns statuses typed and does not interpret them.
	fmt.Println("service answered", httpErr.StatusCode, "for", httpErr.URL)
case err != nil:
	fmt.Println("transport or decoding error:", err)
default:
	fmt.Println(record.Wins, "wins in", record.MatchesCompleted, "matches")
}
```

Retry policies differ on purpose between services (`RetryProvider`, `RetryDiscovery`,
`RetryHaloClient`, `RetryAucune`, `halo5.RetryHalo5`); the caller picks one per client.

## Requirements

Go 1.26.10 (`go.mod`). Pure Go, no cgo. Third-party dependency: `golang.org/x/sync`
(`golang.org/x/time` is used by tests only). The services need valid Xbox / Halo tokens:
`xblauth` performs the exchange chain, but you bring the Azure client and own the storage.

## License

AGPL-3.0, the repository license (`LICENSE`, `NOTICE` at the repository root).

## Contributing

Issues and pull requests on `github.com/JGtm/LevelUp`, with the path prefix `libs/haloapi/` in
the title. From the module directory: `go vet ./... && go test ./...`. The repository
convention is French-language code comments and many French identifiers (`Endpoints`, `Retry`
and `Doer` aside), so expect them in the source. Design: [ADR 0041](../../docs/adr/0041-go-modules-under-libs.md)
(modules under `libs/`) and [ADR 0042](../../docs/adr/0042-halo-api-protocol-vs-exploitation.md)
(protocol here, exploitation in LevelUp).

## Not affiliated

Not affiliated with Microsoft, 343 Industries or Halo Studios; use it within the terms of
service of the services it talks to.
