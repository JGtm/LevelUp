package snapshot

// snapshot_metrics.go — métriques expvar TITRÉES du cut snapshot (Phase 2 / Phase 6bis
// du PLAN_DURABILITE_SNAPSHOT_IMMUABLE). Conforme ADR 0009 (expvar stdlib, pas
// Prometheus) + ADR 0009 multi-user (clés titrées via observability.*T).
//
// Cardinalité BORNÉE : aucune clé n'est dérivée d'un match_id/xuid ; les sous-buckets
// (raison de no-op, raison d'échec) sont un enum FERMÉ. Gauge (SetIntT, overwrite) pour
// l'état courant (version, compte ready) ; cumulatif (AddIntT) pour les compteurs.

import (
	"errors"
	"time"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
)

// Raisons d'échec de cut (enum fermé). Alignées sur les sentinelles ops.ErrSnapshot*.
const (
	snapCutFailRead     = "read_failed"
	snapCutFailCopy     = "copy_failed"
	snapCutFailManifest = "manifest_flip_failed"
	snapCutFailOther    = "other"
)

// classifySnapshotCutErr réduit une erreur de cut à une raison de l'enum fermé.
func classifySnapshotCutErr(err error) string {
	switch {
	case errors.Is(err, ops.ErrSnapshotCopy):
		return snapCutFailCopy
	case errors.Is(err, ops.ErrSnapshotManifest):
		return snapCutFailManifest
	case errors.Is(err, ops.ErrSnapshotRead):
		return snapCutFailRead
	default:
		return snapCutFailOther
	}
}

// recordSnapshotCut publie les métriques d'un cut terminé (succès / no-op / échec).
//
//	snapshot_cut_total                    cumulatif — tentatives
//	snapshot_cut_duration_ms              durée agrégée (avg/max)
//	snapshot_cut_failures_total_<reason>  cumulatif — échec (enum fermé)
//	snapshot_cut_noop_total_<reason>      cumulatif — no-op (no_ready_matches / unchanged)
//	snapshot_cut_produced_total           cumulatif — versions effectivement figées
//	snapshot_version                      GAUGE — version courante (SetIntT, overwrite)
//	snapshot_ready_match_count            GAUGE — matchs ready inclus dans le dernier cut
func recordSnapshotCut(titleSlug string, res ops.SnapshotResult, err error, dur time.Duration) {
	observability.IncCounterT(titleSlug, "snapshot_cut_total")
	observability.RecordDurationMST(titleSlug, "snapshot_cut_duration_ms", dur.Milliseconds())

	if err != nil {
		// Agrégat (lu par le dashboard, sans connaître l'enum) + ventilation par raison.
		observability.IncCounterT(titleSlug, "snapshot_cut_failures_total")
		observability.IncCounterT(titleSlug, "snapshot_cut_failures_total_"+classifySnapshotCutErr(err))
		return
	}
	if !res.Produced {
		reason := res.NoopReason
		if reason == "" {
			reason = "unchanged"
		}
		observability.IncCounterT(titleSlug, "snapshot_cut_noop_total")
		observability.IncCounterT(titleSlug, "snapshot_cut_noop_total_"+reason)
		return
	}
	observability.IncCounterT(titleSlug, "snapshot_cut_produced_total")
	observability.SetIntT(titleSlug, "snapshot_version", res.Version)
	observability.SetIntT(titleSlug, "snapshot_ready_match_count", int64(res.ReadyMatchCount))
}
