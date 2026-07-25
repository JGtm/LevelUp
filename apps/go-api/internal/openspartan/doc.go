// Package openspartan reads a vanilla OpenSpartan/grunt SQLite database and
// exposes its matches as parsed Halo API responses for downstream mapping into
// LevelUp's DuckDB v6 schema.
//
// OpenSpartan/grunt stores each Halo API response as a JSON blob in the
// MatchStats.ResponseBody column (one row per match). This package opens the
// SQLite file read-only via modernc.org/sqlite (pure Go, no CGO), detects the
// schema, locates the owner XUID via three heuristics, and streams the parsed
// matches via an iter.Seq2 iterator.
//
// This package does NOT perform the mapping to LevelUp's v6 domain types —
// that responsibility lives in apps/go-api/internal/openspartan/mapper.
// It also does NOT write to DuckDB — see apps/go-api/internal/service for the
// orchestrating import service.
//
// See docs/ARCHITECTURE_V6.md ("OpenSpartan import" section) for the feature
// overview (French counterpart: docs/FR/ARCHITECTURE_V6.md).
package openspartan
