// Package duckdbbackup provides on-demand, incremental DuckDB backups via restic.
//
// Usage:
//
//	sched := duckdbbackup.New(cfg, func() ([]duckdbbackup.Target, error) {
//	    return []duckdbbackup.Target{
//	        {Key: "main", Path: "/data/myapp.duckdb"},
//	    }, nil
//	})
//	result, err := sched.RunOnce(ctx)
//
// Periodic scheduling is intentionally left to the caller's environment
// (e.g. systemd timers) rather than an in-app loop.
//
// The package has no dependency on any project-specific internal package.
// The caller supplies a discover function that returns the list of DBs to protect.
package duckdbbackup

import (
	"context"
	"database/sql"
)

// OpenDBFunc retourne une connexion réutilisable au fichier DuckDB ciblé. La
// fonction release (jamais nil quand err == nil) doit être appelée par
// l'appelant pour libérer l'emprunt — typiquement un no-op quand la connexion
// est partagée avec le serveur applicatif.
//
// Sémantique : la *sql.DB retournée NE doit PAS être fermée par l'exporter —
// elle est possédée par le caller (typiquement le pool process-wide du
// serveur). Pour les opérations longues, l'exporter passe son ctx.
type OpenDBFunc func(ctx context.Context) (*sql.DB, func(), error)

// Target describes a single DuckDB file to back up.
//
// Si OpenDB est non-nil, l'exporter REUTILISE la connexion fournie au lieu
// d'ouvrir un nouveau handle. C'est requis pour les fichiers déjà détenus en
// lecture-écriture par le serveur applicatif (metadata, shared_social) : DuckDB
// refuse une seconde ouverture avec `?access_mode=read_only` tant qu'un autre
// handle in-process tient le fichier en RW ("different configuration"). Si nil,
// l'exporter ouvre une connexion RO autonome — adapté aux DBs fermées ou déjà
// en RO (ex. shared_matches_v2 via sharedprovider, player stats.duckdb).
type Target struct {
	Key    string // short name used in the manifest and logs
	Path   string // absolute path to the .duckdb file
	OpenDB OpenDBFunc
}

// Result summarises a single backup cycle.
type Result struct {
	SnapshotID string   `json:"snapshot_id,omitempty"`
	Skipped    bool     `json:"skipped"`
	Exported   []string `json:"exported,omitempty"`
	DurationMs int64    `json:"duration_ms"`
}
