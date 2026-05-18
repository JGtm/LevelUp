// Package mapper transforms parsed OpenSpartan Halo API responses into the
// row shapes that LevelUp's DuckDB v6 schema expects.
//
// Inputs come from the sibling internal/openspartan package
// (openspartan.ParsedMatch, openspartan.MatchStats). Outputs are plain Go
// structs the import service will hand to the DuckDB Appender API in PR 3.
//
// The mapper is intentionally pure: no SQL, no filesystem, no network.
// Anything that requires I/O (reading XuidAliases or HighlightEvents rows,
// looking up playlist/map names in metadata.duckdb) lives elsewhere.
//
// Coverage caveats for PR 2:
//   - Names for playlist/map/pair/game_variant are left empty — they live in
//     the metadata.duckdb referential, which the post-import recompute
//     (PR 3.5) fills in.
//   - is_ranked and is_firefight default to false. Inferring them requires
//     metadata that's not in the Halo API payload itself.
//   - kills_expected / deaths_expected / *_stddev are left nil. They come
//     from PlayerMatchStats.Result.StatPerformances, kept as RawMessage by
//     PR 1 to be promoted later.
//   - enemy_mmr is left nil — it's a derived value from the opposing team's
//     MMR, computed during recompute.
//
// All mapping functions are deterministic. Given the same parsed input plus
// the same MapOptions, they always produce identical row sets.
package mapper
