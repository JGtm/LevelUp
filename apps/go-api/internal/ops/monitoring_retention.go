// Package ops — monitoring_retention.go : rétention bornée des tables monitoring
// append-only (E4 revue 2026-07). Les tables d'événements (detection_events,
// detection_status_events, cron_runs) croissent sans borne ; les vues `_latest`
// (ROW_NUMBER / agrégation) les scannent en entier à chaque poll (~30-60 s), et
// la base tient sur un VPS 2 Go. On réplique le CapAndSweep des notifications :
// au-delà d'un cap généreux mais borné, un DELETE PURGE (jamais un UPDATE)
// ramène la table sous le cap.
//
// Sûreté ART / append-only : la base monitoring est un fichier GLOBAL isolé,
// écrit par UN seul process, PK `id` BIGINT (séquence). Le DELETE passe par le
// MÊME lease KindMonitoring que le flush (writer unique). Les vues `_latest`
// restent correctes : la purge PROTÈGE le dernier événement par partition
// (fingerprint / cron_name) en plus des N plus récents globaux, donc le statut/
// run courant de chaque partition n'est jamais élagué. Ces tables ne sont pas
// dans tablesProtegees (no_art_patterns_test) : ce DELETE mono-writer y est sûr.
package ops

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// Caps de rétention par table — généreux mais bornés (justification par table).
// Choisis pour garder les scans des vues `_latest` sous quelques ms sur le VPS
// tout en conservant un historique large (jours→mois selon la fréquence).
const (
	// detection_events : ~1 ligne par fingerprint et par flush avec delta (flush
	// 60 s). Bursty pendant un incident ; l'agrégat detections_latest la scanne à
	// chaque badge/poll. 20 000 lignes ≈ un long épisode d'erreurs, ~4 Mo, scan < ~10 ms.
	detectionEventsRetentionCap = 20_000
	// detection_status_events : cycle de vie (ack/mute/resolve + ré-ouverture).
	// Basse fréquence ; 10 000 transitions couvrent des années.
	detectionStatusEventsRetentionCap = 10_000
	// cron_runs : ~1 ligne par exécution de cron (~10 crons, cadences variées).
	// Croissance la plus régulière → cap le plus large : 50 000 ≈ plusieurs mois
	// à quelques centaines de runs/jour.
	cronRunsRetentionCap = 50_000
)

// retentionSpec décrit la purge d'une table append-only : son cap et le DELETE
// qui garde les N lignes les plus récentes + le dernier événement par partition.
// Le placeholder `?` du DELETE reçoit le cap.
type retentionSpec struct {
	table     string
	cap       int
	deleteSQL string
}

// monitoringRetentionSpecs : purges appliquées à chaque sweep. Var (pas const)
// pour permettre aux tests de réduire les caps sans dupliquer les DELETE.
var monitoringRetentionSpecs = []retentionSpec{
	{
		// Agrégat par fingerprint (detections_latest) : pas de dépendance à une
		// ligne « dernière » unique → on garde simplement les N plus récentes.
		table: "detection_events",
		cap:   detectionEventsRetentionCap,
		deleteSQL: `DELETE FROM detection_events
			WHERE id NOT IN (
				SELECT id FROM detection_events ORDER BY written_at DESC, id DESC LIMIT ?
			)`,
	},
	{
		// detection_status_latest = dernier statut par fingerprint : on protège
		// MAX(id) par fingerprint pour ne jamais perdre le statut courant.
		table: "detection_status_events",
		cap:   detectionStatusEventsRetentionCap,
		deleteSQL: `DELETE FROM detection_status_events
			WHERE id NOT IN (
				SELECT id FROM detection_status_events ORDER BY written_at DESC, id DESC LIMIT ?
			)
			AND id NOT IN (
				SELECT MAX(id) FROM detection_status_events GROUP BY fingerprint
			)`,
	},
	{
		// cron_runs_latest = dernier run par cron_name : on protège MAX(id) par
		// cron_name (dernier run inséré = dernier run courant en pratique).
		table: "cron_runs",
		cap:   cronRunsRetentionCap,
		deleteSQL: `DELETE FROM cron_runs
			WHERE id NOT IN (
				SELECT id FROM cron_runs ORDER BY started_at DESC, id DESC LIMIT ?
			)
			AND id NOT IN (
				SELECT MAX(id) FROM cron_runs GROUP BY cron_name
			)`,
	},
}

// SweepRetention purge les tables monitoring au-delà de leur cap (best-effort par
// table, erreurs agrégées via errors.Join). Appelée périodiquement par la boucle
// de flush (pas un cron dédié). No-op si le store n'est pas ouvert.
func (s *MonitoringStore) SweepRetention(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}
	var errs []error
	for _, spec := range monitoringRetentionSpecs {
		if err := s.sweepTable(ctx, spec); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", spec.table, err))
		}
	}
	return errors.Join(errs...)
}

// sweepTable compte la table (lecture sans lease, comme les autres lectures du
// store) ; si elle dépasse le cap, prend le writer monitoring et applique le
// DELETE de rétention. Sous le cap : no-op (COUNT trivial sur table bornée).
func (s *MonitoringStore) sweepTable(ctx context.Context, spec retentionSpec) error {
	var n int
	if err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM "+spec.table).Scan(&n); err != nil {
		return fmt.Errorf("count: %w", err)
	}
	if n <= spec.cap {
		return nil
	}
	w, err := s.acquire()
	if err != nil {
		return err
	}
	defer w.Release()
	if _, err := w.ExecContext(ctx, spec.deleteSQL, spec.cap); err != nil {
		return fmt.Errorf("sweep: %w", err)
	}
	slog.InfoContext(ctx, "monitoring retention: table élaguée",
		"table", spec.table, "avant", n, "cap", spec.cap)
	return nil
}
