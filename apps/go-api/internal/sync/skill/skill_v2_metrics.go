package skill

// skill_v2_metrics.go — métriques expvar + sentinelle dual-row pour la
// Stratégie C (cf. ADR 0024 + .ai/LUSR_V2_HANDOFF.md).
//
// Exporte via /debug/vars (stdlib expvar) — conforme à ADR 0009 (pas de
// Prometheus). Les compteurs sont incrémentés par writeCanonicalLUSRRow
// et par RunDualRowSentinel.

import (
	"context"
	"database/sql"
	"expvar"
	"fmt"
	"log/slog"
)

// Compteurs expvar publiés sous le namespace `levelup.lusr_v2.*`.
//
// canonicalWritesTotal : batchs (match) écrits avec succès en canonical mode
// (= 1 row LUSR + 1 row LUSR_V2 inséré atomiquement).
// canonicalWriteErrors : échecs d'écriture canonical (transaction rollback).
// dualRowInconsistencies : matchs détectés par RunDualRowSentinel avec un seul
// des deux rating_types présents dans match_skill_rank_latest.
// sentinelScansTotal : nombre d'exécutions de RunDualRowSentinel.
// predictionsTotal : probabilités de victoire pré-match calculées (Sprint 1.A).
// Compte les TENTATIVES de compute : peut dépasser 1/match sur le chemin
// write-held/retry (cf. canonicalWriteHeldWatermark), où un match est recompute
// à chaque cycle tant que son écriture canonical échoue.
var (
	canonicalWritesTotal   = expvar.NewInt("levelup.lusr_v2.canonical_writes_total")
	canonicalWriteErrors   = expvar.NewInt("levelup.lusr_v2.canonical_write_errors_total")
	dualRowInconsistencies = expvar.NewInt("levelup.lusr_v2.dual_row_inconsistencies_total")
	sentinelScansTotal     = expvar.NewInt("levelup.lusr_v2.sentinel_scans_total")
	predictionsTotal       = expvar.NewInt("levelup.lusr_v2.predictions_total")
	// canonicalWriteHeldWatermark : écritures canonical échouées dont le watermark
	// (player_skill_state_v2, shared) a été VOLONTAIREMENT tenu → le match repassera
	// au prochain cycle (fix désync 2026-06-07, cf. .ai/thought_log.md). Croissance
	// soutenue = player DB durablement en échec → investiguer (sinon poison-pill du
	// groupe : le watermark ne peut plus avancer tant que ce match n'est pas écrit).
	canonicalWriteHeldWatermark = expvar.NewInt("levelup.lusr_v2.canonical_write_held_watermark_total")
	// canonicalOwnerMissing : matchs où le owner est absent des rosters 2-équipes
	// alors qu'il est participant (mismatch team_id) → aucune ligne LUSR écrite.
	// Doit rester 0 ; >0 = anomalie data (avant le fix 2026-06-07 ce cas avançait
	// le watermark en SILENCE → gap permanent invisible).
	canonicalOwnerMissing = expvar.NewInt("levelup.lusr_v2.canonical_owner_missing_total")
)

// SentinelReport est le résultat d'un scan dual-row.
type SentinelReport struct {
	// Total matchs présents dans match_skill_rank_latest (LUSR + LUSR_V2 conf.).
	MatchesScanned int
	// Matchs avec LUSR seul (sans LUSR_V2). Cas typique : ancien batch v1 jamais
	// re-traité par v2 — légitime tant que la migration n'a pas re-process tout.
	OnlyLUSR int
	// Matchs avec LUSR_V2 seul (sans LUSR). NE DOIT JAMAIS arriver — bug.
	OnlyLUSRV2 int
	// Matchs avec les deux types (le cas nominal post-bascule).
	BothPresent int
	// Échantillon de match_ids avec inconsistance (max 10) pour debug.
	SampleInconsistent []string
}

// RunDualRowSentinel scanne la table match_skill_rank_latest du player DB
// fourni et vérifie l'invariant dual-row de la Stratégie C : tout match
// doit avoir soit (LUSR seul, héritage v1) soit (LUSR + LUSR_V2, écrit par v2).
// Le cas (LUSR_V2 seul) signale un bug — incrémente dualRowInconsistencies.
//
// Idempotent et read-only ; safe à exécuter périodiquement (cron / endpoint
// debug). Coût : 1 query agrégée par player DB.
//
// Retourne nil + SentinelReport vide si la table n'existe pas (player DB
// pas encore migrée vers append-only).
func RunDualRowSentinel(ctx context.Context, playerDB *sql.DB) (*SentinelReport, error) {
	if playerDB == nil {
		return nil, fmt.Errorf("RunDualRowSentinel: playerDB nil")
	}
	sentinelScansTotal.Add(1)

	if _, err := playerDB.ExecContext(ctx,
		`SELECT 1 FROM match_skill_rank LIMIT 0`); err != nil {
		// Table absente — pas une erreur, juste pas migré.
		slog.DebugContext(ctx, "sentinel: match_skill_rank absent", "err", err)
		return &SentinelReport{}, nil
	}

	// Lecture sur la table RAW (pas la vue _latest) : l'invariant dual-row
	// concerne la coexistence des rows LUSR + LUSR_V2 d'un même match, or
	// match_skill_rank_latest collapse à 1 row/match (priorité CSR>LUSR>LUSR_V2)
	// et ne pourrait jamais montrer les deux. BOOL_OR sur le raw les voit.
	rows, err := playerDB.QueryContext(ctx, `
		WITH per_match AS (
		  SELECT match_id,
		         BOOL_OR(rating_type = 'LUSR')    AS has_lusr,
		         BOOL_OR(rating_type = 'LUSR_V2') AS has_lusr_v2
		  FROM match_skill_rank
		  WHERE rating_type IN ('LUSR', 'LUSR_V2')
		  GROUP BY match_id
		)
		SELECT match_id, has_lusr, has_lusr_v2 FROM per_match`)
	if err != nil {
		return nil, fmt.Errorf("sentinel query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	report := &SentinelReport{}
	for rows.Next() {
		var matchID string
		var hasLUSR, hasLUSRV2 bool
		if err := rows.Scan(&matchID, &hasLUSR, &hasLUSRV2); err != nil {
			return nil, fmt.Errorf("sentinel scan: %w", err)
		}
		report.MatchesScanned++
		switch {
		case hasLUSR && hasLUSRV2:
			report.BothPresent++
		case hasLUSR && !hasLUSRV2:
			report.OnlyLUSR++
		case !hasLUSR && hasLUSRV2:
			report.OnlyLUSRV2++
			dualRowInconsistencies.Add(1)
			if len(report.SampleInconsistent) < 10 {
				report.SampleInconsistent = append(report.SampleInconsistent, matchID)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sentinel rows: %w", err)
	}
	if report.OnlyLUSRV2 > 0 {
		slog.WarnContext(ctx, "sentinel dual-row: LUSR_V2 sans LUSR",
			"count", report.OnlyLUSRV2,
			"sample", report.SampleInconsistent,
		)
	}
	slog.InfoContext(ctx, "sentinel dual-row terminé",
		"matches_scanned", report.MatchesScanned,
		"both_present", report.BothPresent,
		"only_lusr", report.OnlyLUSR,
		"only_lusr_v2", report.OnlyLUSRV2,
	)
	return report, nil
}
