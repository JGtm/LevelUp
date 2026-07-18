//go:build cgo

// Package ops — monitoring_retention_cgo_test.go : la rétention CapAndSweep
// (E4) ramène les tables monitoring sous leur cap tout en PRÉSERVANT les vues
// `_latest` (statut courant par fingerprint, dernier run par cron). Driver
// DuckDB requis.
package ops

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func countMonitoringRows(t *testing.T, st *MonitoringStore, table string) int {
	t.Helper()
	var n int
	if err := st.db.QueryRow(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func statusOf(t *testing.T, st *MonitoringStore, fingerprint string) string {
	t.Helper()
	var s string
	if err := st.db.QueryRow(context.Background(),
		`SELECT status FROM detection_status_latest WHERE fingerprint = ?`, fingerprint).Scan(&s); err != nil {
		t.Fatalf("status_latest %s: %v", fingerprint, err)
	}
	return s
}

func TestMonitoringStore_SweepRetention_BoundsAndPreservesLatest(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	// Caps réduits (5) pour un test rapide, sans dupliquer les DELETE de prod.
	orig := monitoringRetentionSpecs
	t.Cleanup(func() { monitoringRetentionSpecs = orig })
	small := make([]retentionSpec, len(orig))
	copy(small, orig)
	for i := range small {
		small[i].cap = 5
	}
	monitoringRetentionSpecs = small

	// ── detection_events : 10 lignes pour un fingerprint (Count cumulatif) ──
	now := time.Now().UTC()
	for i := 1; i <= 10; i++ {
		sample := DetectionSample{
			Title: "", Level: "WARN", Module: "evt", Message: "evt: boucle",
			Count: int64(i), FirstSeen: now, LastSeen: now.Add(time.Duration(i) * time.Second),
		}
		if err := st.FlushDetections(ctx, []DetectionSample{sample}); err != nil {
			t.Fatalf("flush %d: %v", i, err)
		}
	}

	// ── detection_status_events : fp1 (→ resolved) et fp2 (→ muted), 8 chacun ──
	for i := 0; i < 7; i++ {
		if err := st.SetDetectionStatus(ctx, "fp1", domain.DetectionStatusAcked, ""); err != nil {
			t.Fatalf("status fp1 %d: %v", i, err)
		}
	}
	if err := st.SetDetectionStatus(ctx, "fp1", domain.DetectionStatusResolved, "corrigé"); err != nil {
		t.Fatalf("status fp1 resolved: %v", err)
	}
	for i := 0; i < 7; i++ {
		if err := st.SetDetectionStatus(ctx, "fp2", domain.DetectionStatusAcked, ""); err != nil {
			t.Fatalf("status fp2 %d: %v", i, err)
		}
	}
	if err := st.SetDetectionStatus(ctx, "fp2", domain.DetectionStatusMuted, "silence"); err != nil {
		t.Fatalf("status fp2 muted: %v", err)
	}

	// ── cron_runs : cronA (dernier ok=false) et cronB (dernier ok=true), 8 chacun.
	// cronB entièrement plus récent (started_at) que cronA → tri déterministe.
	base := now.Add(-2 * time.Hour)
	for i := 0; i < 8; i++ {
		if err := st.RecordCronRun(ctx, "cronA", base.Add(time.Duration(i)*time.Minute), i != 7, boolErr(i == 7), 10); err != nil {
			t.Fatalf("cronA %d: %v", i, err)
		}
	}
	for i := 0; i < 8; i++ {
		if err := st.RecordCronRun(ctx, "cronB", base.Add(time.Duration(100+i)*time.Minute), true, "", 10); err != nil {
			t.Fatalf("cronB %d: %v", i, err)
		}
	}

	// Sanity : chaque table dépasse le cap avant sweep.
	if got := countMonitoringRows(t, st, "detection_events"); got != 10 {
		t.Fatalf("detection_events pré-sweep = %d, attendu 10", got)
	}
	if got := countMonitoringRows(t, st, "detection_status_events"); got != 16 {
		t.Fatalf("detection_status_events pré-sweep = %d, attendu 16", got)
	}
	if got := countMonitoringRows(t, st, "cron_runs"); got != 16 {
		t.Fatalf("cron_runs pré-sweep = %d, attendu 16", got)
	}

	if err := st.SweepRetention(ctx); err != nil {
		t.Fatalf("SweepRetention: %v", err)
	}

	// ── Bornes ── detection_events : exactement le cap (pas de protection partition).
	if got := countMonitoringRows(t, st, "detection_events"); got != 5 {
		t.Errorf("detection_events post-sweep = %d, attendu 5 (cap)", got)
	}
	// status/cron : cap + au plus 1 protégé par partition (2 partitions).
	if got := countMonitoringRows(t, st, "detection_status_events"); got > 7 {
		t.Errorf("detection_status_events post-sweep = %d, attendu ≤ 7 (cap 5 + 2 protégés)", got)
	}
	if got := countMonitoringRows(t, st, "cron_runs"); got > 7 {
		t.Errorf("cron_runs post-sweep = %d, attendu ≤ 7 (cap 5 + 2 protégés)", got)
	}

	// ── Vues `_latest` correctes après purge ──
	if s := statusOf(t, st, "fp1"); s != domain.DetectionStatusResolved {
		t.Errorf("detection_status_latest[fp1] = %q après sweep, attendu resolved", s)
	}
	if s := statusOf(t, st, "fp2"); s != domain.DetectionStatusMuted {
		t.Errorf("detection_status_latest[fp2] = %q après sweep, attendu muted", s)
	}
	runs, err := st.LatestCronRuns(ctx)
	if err != nil {
		t.Fatalf("LatestCronRuns: %v", err)
	}
	byName := map[string]PersistedCronRun{}
	for _, r := range runs {
		byName[r.Name] = r
	}
	if a, ok := byName["cronA"]; !ok || a.OK {
		t.Errorf("cron_runs_latest[cronA] = %+v (ok=%v), attendu dernier run ok=false conservé", a, ok)
	}
	if b, ok := byName["cronB"]; !ok || !b.OK {
		t.Errorf("cron_runs_latest[cronB] = %+v (ok=%v), attendu dernier run ok=true conservé", b, ok)
	}

	// Idempotence : un 2e sweep ne casse rien et ne re-supprime pas sous le cap.
	before := countMonitoringRows(t, st, "detection_events")
	if err := st.SweepRetention(ctx); err != nil {
		t.Fatalf("SweepRetention (2e): %v", err)
	}
	if after := countMonitoringRows(t, st, "detection_events"); after != before {
		t.Errorf("2e sweep non idempotent : %d → %d", before, after)
	}
}

func boolErr(fail bool) string {
	if fail {
		return "boom"
	}
	return ""
}
