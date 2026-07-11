//go:build cgo

// Package ops — monitoring_store_cgo_test.go : cycle de vie des détections et
// historique crons/data-health sur la base monitoring (driver DuckDB requis).
package ops

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func newTestStore(t *testing.T) *MonitoringStore {
	t.Helper()
	ctx := context.Background()
	path := t.TempDir() + "/monitoring.duckdb"
	st, err := NewMonitoringStore(ctx, path)
	if err != nil {
		t.Fatalf("NewMonitoringStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestMonitoringStore_DetectionLifecycle(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	now := time.Now().UTC()

	sample := DetectionSample{
		Title: "", Level: "WARN", Module: "player_watcher",
		Message: "player_watcher: sync échoué", SampleDetail: "boom",
		Count: 3, FirstSeen: now.Add(-time.Hour), LastSeen: now,
	}
	if err := st.FlushDetections(ctx, []DetectionSample{sample}); err != nil {
		t.Fatalf("flush 1: %v", err)
	}

	resp, err := st.Detections(ctx, DetectionFilter{})
	if err != nil {
		t.Fatalf("detections 1: %v", err)
	}
	if len(resp.Detections) != 1 {
		t.Fatalf("attendu 1 détection, got %d", len(resp.Detections))
	}
	d := resp.Detections[0]
	if d.Count != 3 {
		t.Errorf("count = %d, attendu 3", d.Count)
	}
	if d.Status != domain.DetectionStatusOpen {
		t.Errorf("status = %q, attendu open", d.Status)
	}
	if resp.OpenCount != 1 {
		t.Errorf("open_count = %d, attendu 1", resp.OpenCount)
	}
	fp := d.Fingerprint

	// Delta : count cumulatif passe à 5 → +2.
	sample.Count = 5
	sample.LastSeen = now.Add(time.Minute)
	if err := st.FlushDetections(ctx, []DetectionSample{sample}); err != nil {
		t.Fatalf("flush 2: %v", err)
	}
	resp, _ = st.Detections(ctx, DetectionFilter{})
	if resp.Detections[0].Count != 5 {
		t.Errorf("count après delta = %d, attendu 5", resp.Detections[0].Count)
	}

	// Résolution → status resolved, hors OpenCount.
	if err := st.SetDetectionStatus(ctx, fp, domain.DetectionStatusResolved, "corrigé"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	resp, _ = st.Detections(ctx, DetectionFilter{})
	if resp.Detections[0].Status != domain.DetectionStatusResolved {
		t.Errorf("status = %q, attendu resolved", resp.Detections[0].Status)
	}
	if resp.OpenCount != 0 {
		t.Errorf("open_count après résolution = %d, attendu 0", resp.OpenCount)
	}

	// Nouvelle occurrence après resolved → ré-ouverture (DC-2).
	sample.Count = 6
	sample.LastSeen = now.Add(2 * time.Minute)
	if err := st.FlushDetections(ctx, []DetectionSample{sample}); err != nil {
		t.Fatalf("flush 3: %v", err)
	}
	resp, _ = st.Detections(ctx, DetectionFilter{})
	if resp.Detections[0].Status != domain.DetectionStatusOpen {
		t.Errorf("status après réouverture = %q, attendu open", resp.Detections[0].Status)
	}
}

func TestMonitoringStore_DetectionFilters(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	now := time.Now().UTC()
	samples := []DetectionSample{
		{Level: "WARN", Module: "a", Message: "a: x", Count: 1, LastSeen: now},
		{Level: "ERROR", Module: "b", Message: "b: y", Count: 1, LastSeen: now},
	}
	if err := st.FlushDetections(ctx, samples); err != nil {
		t.Fatalf("flush: %v", err)
	}
	resp, err := st.Detections(ctx, DetectionFilter{Level: "ERROR"})
	if err != nil {
		t.Fatalf("detections: %v", err)
	}
	if len(resp.Detections) != 1 || resp.Detections[0].Module != "b" {
		t.Fatalf("filtre level=ERROR: got %+v", resp.Detections)
	}
}

func TestMonitoringStore_InvalidStatusRejected(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	if err := st.SetDetectionStatus(ctx, "abc", "bogus", ""); err == nil {
		t.Fatal("statut invalide accepté")
	}
}

func TestMonitoringStore_CronAndDataHealth(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	if err := st.RecordCronRun(ctx, "backup", time.Now(), true, "", 120); err != nil {
		t.Fatalf("record cron: %v", err)
	}
	if err := st.RecordDataHealthRun(ctx, `{"warnings_total":2}`); err != nil {
		t.Fatalf("record data health: %v", err)
	}
	js, err := st.LatestDataHealthJSON(ctx)
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if js != `{"warnings_total":2}` {
		t.Errorf("latest data health = %q", js)
	}
}
