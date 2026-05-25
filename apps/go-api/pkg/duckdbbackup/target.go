// Package duckdbbackup provides scheduled, incremental DuckDB backups via restic.
//
// Usage:
//
//	sched := duckdbbackup.New(cfg, func() ([]duckdbbackup.Target, error) {
//	    return []duckdbbackup.Target{
//	        {Key: "main", Path: "/data/myapp.duckdb"},
//	    }, nil
//	})
//	go sched.Run(ctx)
//
// The package has no dependency on any project-specific internal package.
// The caller supplies a discover function that returns the list of DBs to protect.
package duckdbbackup

// Target describes a single DuckDB file to back up.
type Target struct {
	Key  string // short name used in the manifest and logs
	Path string // absolute path to the .duckdb file
}

// Result summarises a single backup cycle.
type Result struct {
	SnapshotID string   `json:"snapshot_id,omitempty"`
	Skipped    bool     `json:"skipped"`
	Exported   []string `json:"exported,omitempty"`
	DurationMs int64    `json:"duration_ms"`
}
