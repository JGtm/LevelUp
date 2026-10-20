module levelup/go-api

// Version minimale du toolchain : la CI et le build des binaires de release l'installent
// depuis cette ligne (`go-version-file: apps/go-api/go.mod`). Elle doit rester une version
// dont la bibliothèque standard n'a aucune vulnérabilité atteinte, sinon le job `govulncheck`
// de `ci.yml` rougit. L'image de build du `Dockerfile` est épinglée sur la même version : une
// image plus ancienne que cette ligne ne compile pas (GOTOOLCHAIN=local dans l'image).
go 1.26.9

require (
	github.com/HugoSmits86/nativewebp v1.3.0
	github.com/danielgtaylor/huma/v2 v2.39.1
	github.com/duckdb/duckdb-go/v2 v2.10506.0
	github.com/getkin/kin-openapi v0.151.0
	github.com/go-chi/chi/v5 v5.5.0
	github.com/go-chi/cors v1.2.2
	github.com/go-chi/httprate v0.16.1
	github.com/google/uuid v1.6.0
	github.com/pelletier/go-toml/v2 v2.4.3
	golang.org/x/crypto v0.58.0
	golang.org/x/image v0.47.0
	golang.org/x/sync v0.24.0
	golang.org/x/sys v0.49.0
	golang.org/x/time v0.16.0
	gopkg.in/yaml.v3 v3.0.1
	modernc.org/sqlite v1.60.1
)

require (
	github.com/apache/arrow-go/v18 v18.5.1 // indirect
	github.com/duckdb/duckdb-go-bindings v0.10506.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/darwin-amd64 v0.10506.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/darwin-arm64 v0.10506.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/linux-amd64 v0.10506.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/linux-arm64 v0.10506.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/windows-amd64 v0.10506.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-openapi/jsonpointer v0.22.5 // indirect
	github.com/go-openapi/swag/jsonname v0.25.5 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/google/flatbuffers v25.12.19+incompatible // indirect
	github.com/klauspost/compress v1.19.0 // indirect
	github.com/klauspost/cpuid/v2 v2.4.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/oasdiff/yaml v0.1.1 // indirect
	github.com/oasdiff/yaml3 v0.0.14 // indirect
	github.com/pierrec/lz4/v4 v4.1.25 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	golang.org/x/exp v0.0.0-20260112195511-716be5621a96 // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/telemetry v0.0.0-20260924152758-ed294f943157 // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/tools v0.51.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	modernc.org/libc v1.77.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
)
