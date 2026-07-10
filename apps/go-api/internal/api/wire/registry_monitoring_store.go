// Package api — registry_monitoring_store.go : runners branchés sur le store
// monitoring persistant (ops.MonitoringStore, base globale monitoring.duckdb).
//
// Le store survit au restart (l'overview mémoire, lui, repart de zéro). Ces
// runners exposent : les détections avec cycle de vie (flush de l'ErrorCollector
// puis lecture de la vue detections_latest), le changement de statut, et la
// boucle de flush périodique câblée au boot. Tous best-effort si le store est nil
// (échec d'ouverture au boot / tests) — jamais de panic.
package wire

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
)

// detectionFlushInterval : période de flush du delta ErrorCollector → store.
// Court assez pour limiter la perte au restart, sans I/O inutile (le flush
// n'écrit que s'il y a un delta).
const detectionFlushInterval = 60 * time.Second

// WithMonitoringStore attache le store monitoring persistant. Nil possible :
// les détections dégradent en liste vide et le PATCH répond 503.
func (r *ServiceRegistry) WithMonitoringStore(s *ops.MonitoringStore) *ServiceRegistry {
	r.monitoringStore = s
	return r
}

// flushDetections mappe les buckets ErrorCollector courants (mémoire) vers le
// store (delta append-only). Best-effort : une erreur est loguée, pas propagée.
func (r *ServiceRegistry) flushDetections(ctx context.Context) {
	if r.monitoringStore == nil {
		return
	}
	buckets := observability.ErrorBuckets()
	if len(buckets) == 0 {
		return
	}
	samples := make([]ops.DetectionSample, 0, len(buckets))
	for _, b := range buckets {
		samples = append(samples, ops.DetectionSample{
			Title:        b.Title,
			Level:        b.Level,
			Module:       b.Module,
			Message:      b.Message,
			SampleDetail: b.LastDetail,
			Count:        b.Count,
			FirstSeen:    b.FirstSeen,
			LastSeen:     b.LastSeen,
		})
	}
	if err := r.monitoringStore.FlushDetections(ctx, samples); err != nil {
		monitoringLog.WarnContext(ctx, "admin_monitoring: flush détections échoué", "err", err)
		return
	}
	// Gauge des détections open (source des badges nav — l'overview la lit sans I/O).
	if n, err := r.monitoringStore.OpenDetectionCount(ctx); err == nil {
		observability.SetInt("monitoring_detections_open", int64(n))
	}
}

// RunDetectionFlushLoop boucle un flush périodique jusqu'à ctx.Done() (câblé au
// boot sur schedulerCtx/schedulerWG, donc drainé avant duckdb.CloseAll). Un flush
// final s'exécute à l'arrêt (contexte frais : le handle est encore ouvert car ce
// goroutine est attendu par schedulerWG AVANT CloseAll).
func (r *ServiceRegistry) RunDetectionFlushLoop(ctx context.Context) {
	if r.monitoringStore == nil {
		return
	}
	t := time.NewTicker(detectionFlushInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			r.flushDetections(context.Background())
			return
		case <-t.C:
			r.flushDetections(ctx)
		}
	}
}

// DetectionsReport flush les buckets courants puis lit la vue detections_latest
// filtrée. Le flush garantit que l'admin voit l'état à jour sans attendre le
// tick périodique. Store nil → réponse vide propre.
func (r *ServiceRegistry) DetectionsReport(ctx context.Context, status, level, module, title string, limit int) (domain.AdminDetectionsResponse, error) {
	if r.monitoringStore == nil {
		return domain.AdminDetectionsResponse{
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Detections:  []domain.MonitoringDetection{},
		}, nil
	}
	r.flushDetections(ctx)
	return r.monitoringStore.Detections(ctx, ops.DetectionFilter{
		Status: status, Level: level, Module: module, Title: title, Limit: limit,
	})
}

// SetDetectionStatus statue une détection (append cycle de vie).
func (r *ServiceRegistry) SetDetectionStatus(ctx context.Context, fingerprint, status, note string) error {
	if r.monitoringStore == nil {
		return fmt.Errorf("store monitoring non câblé")
	}
	return r.monitoringStore.SetDetectionStatus(ctx, fingerprint, status, note)
}
