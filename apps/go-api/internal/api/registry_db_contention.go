// Package api — registry_db_contention.go : assembleur du dashboard admin
// « Contention DB (sync) ». Lit la capture du sharedprovider B-swap et calcule
// les moyennes (logique tenue hors du handler). Lecture seule.
package api

import (
	"context"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// DBContention retourne une capture des compteurs de contention du shared
// provider (B-swap). Lecture seule des métriques expvar — ne touche aucune
// écriture ni aucun handle DB.
func (r *ServiceRegistry) DBContention(_ context.Context) domain.DBContentionResponse {
	s := sharedprovider.Snapshot()
	resp := domain.DBContentionResponse{
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		State:         s.State,
		Swaps:         s.SwapsToRW,
		DrainMsTotal:  s.DrainMsTotal,
		ReadsRejected: s.ReadsTimedOut,
		ReadersInUse:  s.ReadersInUse,
		SwapFailures:  s.SwapFailures,
		AvgBlockedMs:  s.BlockedWindowAvgMs,
		MaxBlockedMs:  s.BlockedWindowMaxMs,
		// Étape 0 attribution : détention writer (agrégat + par détenteur + watchdog).
		AvgRWWindowMs: s.RWWindowAvgMs,
		MaxRWWindowMs: s.RWWindowMaxMs,
		WatchdogFired: s.WatchdogFired,
		Holders:       make([]domain.DBContentionHolder, 0, len(s.RWWindowByHolder)),
	}
	for _, h := range s.RWWindowByHolder {
		resp.Holders = append(resp.Holders, domain.DBContentionHolder{
			Label: h.Label, Count: h.Count, TotalMs: h.TotalMs,
			AvgMs: h.AvgMs, MaxMs: h.MaxMs, WatchdogFired: h.WatchdogFired,
		})
	}
	if s.SwapsToRW > 0 {
		resp.AvgAcquireMs = s.AcquireMsTotal / s.SwapsToRW
	}
	if s.SwapsToRO > 0 {
		resp.AvgReleaseMs = s.ReleaseMsTotal / s.SwapsToRO
	}
	return resp
}
