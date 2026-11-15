//go:build integration

// Package sync — engagement_intensity_write_test.go : l'écriture des intensités de match est
// best-effort mais jamais muette : chaque échec est journalisé et compté, le lot continue.
package sync

import (
	"context"
	"testing"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability"
)

func TestPersistMatchIntensities_EcritureEtEchecsComptes(t *testing.T) {
	shared := openBatchPathTestDB(t, migration.TargetShared)
	seedConvergenceMatch(t, shared, "int-1", "x1", true, 0)
	ctx := context.Background()
	ups := []matchIntensityUpdate{{matchID: "int-1", intensity: 0.75}, {matchID: "int-2", intensity: 0.5}}

	before := observability.LoadCounter("match_intensity_write_failed_total")
	persistMatchIntensities(ctx, shared, ups)
	if d := observability.LoadCounter("match_intensity_write_failed_total") - before; d != 0 {
		t.Fatalf("écriture réussie : compteur d'échecs +%d, want +0", d)
	}
	var got float64
	if err := shared.QueryRow(`SELECT match_intensity FROM match_registry WHERE match_id = 'int-1'`).Scan(&got); err != nil {
		t.Fatalf("lecture match_intensity: %v", err)
	}
	if got != 0.75 {
		t.Fatalf("match_intensity = %v, want 0.75", got)
	}

	before = observability.LoadCounter("match_intensity_write_failed_total")
	persistMatchIntensities(ctx, closedSharedDB(t), ups)
	if d := observability.LoadCounter("match_intensity_write_failed_total") - before; d != int64(len(ups)) {
		t.Fatalf("base fermée : compteur d'échecs +%d, want +%d (un par intensité, le lot continue)", d, len(ups))
	}
}
