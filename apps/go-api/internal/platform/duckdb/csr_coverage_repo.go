// Package duckdb — csr_coverage_repo.go : diagnostic coverage CSR par joueur.
//
// Phase 9 du plan pipeline CSR. Utilisé par le handler /_diag/csr-coverage/{slug}
// pour répondre rapidement à la question "le pipeline CSR a-t-il bien capturé
// les données de ce joueur, ou faut-il lancer un backfill ?".
package duckdb

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/domain"
)

// CSRCoverageRepo expose la méthode GetCoverage(ctx, xuid).
type CSRCoverageRepo struct {
	pdb *PlayerDB
}

// NewCSRCoverageRepo construit un repo coverage à partir du PlayerDB.
func NewCSRCoverageRepo(pdb *PlayerDB) *CSRCoverageRepo {
	return &CSRCoverageRepo{pdb: pdb}
}

// GetCoverage construit le diagnostic complet pour le joueur identifié par xuid.
// Toute erreur intermédiaire est tolérée silencieusement (champs à 0) — l'idée
// est de produire UN payload exploitable même si une partie des tables manque.
func (r *CSRCoverageRepo) GetCoverage(ctx context.Context, playerSlug, xuid string) (*domain.CSRCoverage, error) {
	if r == nil || r.pdb == nil {
		return nil, fmt.Errorf("CSRCoverageRepo: pdb nil")
	}
	cov := &domain.CSRCoverage{
		PlayerSlug: playerSlug,
		XUID:       xuid,
	}

	// 1) player_csr_snapshots — 3 compteurs en une query.
	// Lecture via vue _latest (Phase 2.G) : la table physique est
	// append-only, on veut le compte fonctionnel par (playlist_id, season_id).
	if rows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, `
		SELECT
			COUNT(*),
			SUM(CASE WHEN COALESCE(alltime_value,0) > 0 THEN 1 ELSE 0 END),
			SUM(CASE WHEN COALESCE(current_measurement_remaining,0) > 0 THEN 1 ELSE 0 END)
		FROM player_csr_snapshots_latest
	`); err == nil {
		var total, alltime, placement sql.NullInt64
		if err := rows.Scan(&total, &alltime, &placement); err == nil {
			cov.Snapshots.Total = int(total.Int64)
			cov.Snapshots.WithAlltimeValue = int(alltime.Int64)
			cov.Snapshots.WithPlacementRem = int(placement.Int64)
		}
		rows.Close()
		// (table absente → lecture err → champs restent à 0, comportement attendu)
	}

	// 2) match_skill_rank rating_type='CSR' — 3 compteurs
	if rows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, `
		SELECT
			COUNT(*),
			SUM(CASE WHEN tier IS NOT NULL AND tier <> '' AND tier <> 'Placement' THEN 1 ELSE 0 END),
			SUM(CASE WHEN tier = 'Placement' THEN 1 ELSE 0 END)
		FROM match_skill_rank_latest
		WHERE rating_type = 'CSR'
	`); err == nil {
		var total, matured, placement sql.NullInt64
		if err := rows.Scan(&total, &matured, &placement); err == nil {
			cov.MatchSkillRankCSR.Total = int(total.Int64)
			cov.MatchSkillRankCSR.Matured = int(matured.Int64)
			cov.MatchSkillRankCSR.Placement = int(placement.Int64)
		}
		rows.Close()
	}

	// 3) match_registry (shared) — count ranked matches pour ce xuid
	rankedInRegistry := r.countRankedMatchesInRegistry(ctx, xuid)
	cov.MatchSkillRankCSR.RankedMatchesInRegistry = rankedInRegistry
	if rankedInRegistry > cov.MatchSkillRankCSR.Total {
		cov.MatchSkillRankCSR.CoverageGap = rankedInRegistry - cov.MatchSkillRankCSR.Total
	}

	// 4) needs_backfill : true si gap > 0 OU snapshots vides ET registry a du ranked
	cov.NeedsBackfill = cov.MatchSkillRankCSR.CoverageGap > 0 ||
		(cov.Snapshots.Total == 0 && rankedInRegistry > 0)

	return cov, nil
}

// countRankedMatchesInRegistry interroge shared.match_registry via SharedReader.
// Fallback heuristique playlist_name LIKE '%ranked%' pour les rows où is_ranked
// n'a pas (encore) été backfilled (Phase 1 du plan).
func (r *CSRCoverageRepo) countRankedMatchesInRegistry(ctx context.Context, xuid string) int {
	if r.pdb == nil || r.pdb.SharedReader == nil {
		return 0
	}
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return 0
	}
	defer release()
	var count int
	err = sharedDB.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT r.match_id)
		FROM match_registry r
		JOIN match_participants mp ON mp.match_id = r.match_id
		WHERE mp.xuid = ?
		  AND (
		      COALESCE(r.is_ranked, FALSE) = TRUE
		      OR LOWER(COALESCE(r.playlist_name, '')) LIKE '%ranked%'
		      OR LOWER(COALESCE(r.pair_name, '')) LIKE 'ranked:%'
		  )
	`, xuid).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}
