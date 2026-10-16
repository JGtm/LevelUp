//go:build cgo

// Package ops — pagination_ceiling_cgo_test.go : plafond des lectures paginées du
// store de monitoring face à une limite démesurée. Ces tests sont cités par
// paginationStringParams (internal/api/pagination_guard_test.go).
package ops

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestMonitoringStore_DetectionsLimitCeiling(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	now := time.Now().UTC()
	samples := make([]DetectionSample, 0, 520)
	for i := 0; i < 520; i++ {
		samples = append(samples, DetectionSample{
			Level: "WARN", Module: "m", Message: fmt.Sprintf("m: détection %04d", i), Count: 1, LastSeen: now,
		})
	}
	if err := st.FlushDetections(ctx, samples); err != nil {
		t.Fatalf("flush : %v", err)
	}
	resp, err := st.Detections(ctx, DetectionFilter{Limit: 1_000_000_000})
	if err != nil {
		t.Fatalf("Detections : %v", err)
	}
	if len(resp.Detections) == 0 || len(resp.Detections) > 500 {
		t.Errorf("détections servies = %d, attendu dans [1, 500]", len(resp.Detections))
	}
}

func TestMonitoringStore_BuildQueueLimitCeiling(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	for i := 0; i < 210; i++ {
		enqueueTestJob(t, st, fmt.Sprintf("%08d-1111-2222-3333-444444444444", i))
	}
	resp, err := st.BuildQueueReport(ctx, 1_000_000_000)
	if err != nil {
		t.Fatalf("BuildQueueReport : %v", err)
	}
	if len(resp.Jobs) == 0 || len(resp.Jobs) > 200 {
		t.Errorf("jobs servis = %d, attendu dans [1, 200]", len(resp.Jobs))
	}
}
