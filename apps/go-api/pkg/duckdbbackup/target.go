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

import "time"

// Target describes a single DuckDB file to back up.
type Target struct {
	Key  string // short name used in the manifest and logs
	Path string // absolute path to the .duckdb file
}

// Result summarises a single backup cycle.
type Result struct {
	SnapshotID string        // restic snapshot ID; empty when Skipped
	Skipped    bool          // true when no DB changed since last backup
	Exported   []string      // Keys of DBs actually re-exported this cycle
	Duration   time.Duration
}
