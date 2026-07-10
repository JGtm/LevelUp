//go:build cgo

// Package migration — monitoring_schema_cgo_test.go : le schéma monitoring se
// pose idempotemment et expose ses vues _latest (driver DuckDB requis).
package migration

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestEnsureMonitoringSchema_Idempotent(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open :memory: %v", err)
	}
	defer db.Close()

	// Deux passages : réentrance.
	if err := EnsureMonitoringSchema(ctx, db); err != nil {
		t.Fatalf("ensure 1: %v", err)
	}
	if err := EnsureMonitoringSchema(ctx, db); err != nil {
		t.Fatalf("ensure 2 (idempotent): %v", err)
	}

	// Les vues _latest doivent être interrogeables.
	for _, view := range []string{"detections_latest", "detection_status_latest", "cron_runs_latest"} {
		if _, err := db.ExecContext(ctx, "SELECT * FROM "+view+" LIMIT 0"); err != nil {
			t.Errorf("vue %s absente/invalide: %v", view, err)
		}
	}

	// Insert d'un event puis d'un statut : detections_latest reflète le dernier statut.
	if _, err := db.ExecContext(ctx,
		`INSERT INTO detection_events (fingerprint, level, message, occurred_at, count_delta)
		 VALUES ('fp1', 'WARN', 'm', CURRENT_TIMESTAMP, 2)`); err != nil {
		t.Fatalf("insert event: %v", err)
	}
	var status string
	if err := db.QueryRowContext(ctx,
		`SELECT status FROM detections_latest WHERE fingerprint = 'fp1'`).Scan(&status); err != nil {
		t.Fatalf("select detection: %v", err)
	}
	if status != "open" {
		t.Errorf("status défaut = %q, attendu open", status)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO detection_status_events (fingerprint, status) VALUES ('fp1', 'acked')`); err != nil {
		t.Fatalf("insert status: %v", err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT status FROM detections_latest WHERE fingerprint = 'fp1'`).Scan(&status); err != nil {
		t.Fatalf("select detection 2: %v", err)
	}
	if status != "acked" {
		t.Errorf("status après acked = %q, attendu acked", status)
	}
}
