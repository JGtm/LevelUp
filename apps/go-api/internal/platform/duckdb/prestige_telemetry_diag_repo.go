// Package duckdb — prestige_telemetry_diag_repo.go : agrégation diagnostic de la
// table append-only prestige_telemetry (player DB) par origine du défi.
//
// ADR 0020. Lecture pure (GROUP BY), pas de write. Alimente le handler
// /api/v1/_diag/prestige/telemetry. Les lignes sans source (historique
// pré-plumbing, ou callers qui ne renseignent pas Source) sont agrégées sous
// "unknown" via COALESCE(NULLIF(source,”),'unknown').
package duckdb

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
)

// PrestigeTelemetryDiagRepo agrège la télémétrie Prestige d'un joueur.
type PrestigeTelemetryDiagRepo struct {
	pdb *PlayerDB
}

// NewPrestigeTelemetryDiagRepo construit le repo.
func NewPrestigeTelemetryDiagRepo(pdb *PlayerDB) *PrestigeTelemetryDiagRepo {
	return &PrestigeTelemetryDiagRepo{pdb: pdb}
}

// prestigeTelemetryDiagQuery ventile les événements par origine. rejected matche
// tous les event_type "rejected:*" (la raison est suffixée). Les transitions
// (completed/expired/abandoned) et created sont des event_type exacts.
const prestigeTelemetryDiagQuery = `
	SELECT
		COALESCE(NULLIF(source, ''), 'unknown')                          AS src,
		SUM(CASE WHEN event_type = 'created'    THEN 1 ELSE 0 END)        AS created,
		SUM(CASE WHEN event_type LIKE 'rejected%' THEN 1 ELSE 0 END)      AS rejected,
		SUM(CASE WHEN event_type = 'completed'  THEN 1 ELSE 0 END)        AS completed,
		SUM(CASE WHEN event_type = 'expired'    THEN 1 ELSE 0 END)        AS expired,
		SUM(CASE WHEN event_type = 'abandoned'  THEN 1 ELSE 0 END)        AS abandoned,
		COUNT(*)                                                          AS total
	FROM prestige_telemetry
	GROUP BY src
	ORDER BY src`

// GetPrestigeTelemetryDiag retourne l'agrégat par origine. Satisfait
// handlers.PrestigeTelemetryDiagProvider par signature. Best-effort : si la table
// est absente (DB legacy pré-migration), retourne un diag vide sans erreur.
func (r *PrestigeTelemetryDiagRepo) GetPrestigeTelemetryDiag(ctx context.Context, slug string) (*domain.PrestigeTelemetryDiag, error) {
	out := &domain.PrestigeTelemetryDiag{
		PlayerSlug: slug,
		BySource:   []domain.PrestigeTelemetrySourceStats{},
	}
	if r == nil || r.pdb == nil || r.pdb.Player == nil {
		return nil, fmt.Errorf("PrestigeTelemetryDiagRepo: PlayerDB nil")
	}

	rows, err := r.pdb.ReadDB().QueryRecovered(ctx, prestigeTelemetryDiagQuery)
	if err != nil {
		// Table absente (legacy pré-migration) / reopen concurrent → diag vide
		// (best-effort : le diag ne doit pas 500 sur une DB sans télémétrie).
		// Loggé avant dégradation (CLAUDE.md : jamais d'erreur avalée en silence).
		slog.WarnContext(ctx, "prestige telemetry diag: query failed, returning empty",
			"player", slug, "err", err)
		return out, nil
	}
	defer rows.Close()

	for rows.Next() {
		var s domain.PrestigeTelemetrySourceStats
		var total int
		if err := rows.Scan(&s.Source, &s.Created, &s.Rejected, &s.Completed,
			&s.Expired, &s.Abandoned, &total); err != nil {
			return nil, fmt.Errorf("scan prestige telemetry diag: %w", err)
		}
		s.AcceptanceRate = ratio(s.Created, s.Created+s.Rejected)
		s.CompletionRate = ratio(s.Completed, s.Created)
		s.AbandonRate = ratio(s.Abandoned, s.Created)
		out.BySource = append(out.BySource, s)
		// total inclut aussi committed/palier_recomputed (non ventilés en compteurs).
		out.TotalEvents += total
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate prestige telemetry diag: %w", err)
	}
	return out, nil
}

// ratio retourne num/den, ou -1 (sentinel JSON) si le dénominateur est nul —
// distingue "taux 0" (aucun succès) de "non calculable" (aucun défi).
func ratio(num, den int) float64 {
	if den <= 0 {
		return -1
	}
	return float64(num) / float64(den)
}
