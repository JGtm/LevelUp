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
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
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

	// 5) contrôle croisé référence des playlists classées <-> CSR par match (alerte A3).
	cov.ReferenceControl = r.controlAgainstReference(ctx, xuid)
	if cov.ReferenceControl.Alert {
		slog.WarnContext(ctx, "csr-coverage: la référence des playlists classées et le CSR par match divergent",
			"player_slug", playerSlug, "xuid", xuid,
			"reference_matches_without_csr", cov.ReferenceControl.ReferenceMatchesWithoutCSR,
			"csr_matches_off_reference", cov.ReferenceControl.CSRMatchesOffReference)
	}

	return cov, nil
}

// playlistCSRCount : pour une playlist, les matchs du joueur et ceux qui portent un CSR.
type playlistCSRCount struct {
	playlistID string
	matches    int
	withCSR    int
}

// controlCSRAgainstReference applique la référence rankedplaylists à des comptes par playlist :
// matchs de playlist classée sans CSR, et matchs à CSR sur une playlist hors référence. Pure.
func controlCSRAgainstReference(rows []playlistCSRCount) domain.CSRReferenceControl {
	var c domain.CSRReferenceControl
	for _, row := range rows {
		if rankedplaylists.IsRanked(row.playlistID) {
			c.ReferenceMatchesWithoutCSR += row.matches - row.withCSR
		} else {
			c.CSRMatchesOffReference += row.withCSR
		}
	}
	c.Alert = c.ReferenceMatchesWithoutCSR > 0 || c.CSRMatchesOffReference > 0
	return c
}

// controlAgainstReference lit, par playlist, les matchs du joueur et ceux qui portent un CSR dans
// shared (match_csrs_latest), puis applique la référence. Diagnostic : une lecture qui échoue
// rend un contrôle vide (journalisé), jamais une erreur.
func (r *CSRCoverageRepo) controlAgainstReference(ctx context.Context, xuid string) domain.CSRReferenceControl {
	if r.pdb == nil || r.pdb.SharedReader == nil {
		return domain.CSRReferenceControl{}
	}
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "csr-coverage: contrôle de référence, base partagée indisponible", "err", err)
		return domain.CSRReferenceControl{}
	}
	defer release()
	rs, err := sharedDB.QueryContext(ctx, `
		SELECT r.playlist_id, COUNT(DISTINCT r.match_id), COUNT(DISTINCT c.match_id)
		FROM match_registry r
		JOIN match_participants mp ON mp.match_id = r.match_id AND mp.xuid = ?
		LEFT JOIN match_csrs_latest c
		       ON c.match_id = r.match_id AND c.xuid = mp.xuid AND c.rating_type = 'CSR'
		WHERE r.playlist_id IS NOT NULL AND r.playlist_id <> ''
		GROUP BY r.playlist_id`, xuid)
	if err != nil {
		slog.WarnContext(ctx, "csr-coverage: contrôle de référence, lecture échouée", "err", err)
		return domain.CSRReferenceControl{}
	}
	defer rs.Close()
	var rows []playlistCSRCount
	for rs.Next() {
		var row playlistCSRCount
		if err := rs.Scan(&row.playlistID, &row.matches, &row.withCSR); err != nil {
			slog.WarnContext(ctx, "csr-coverage: contrôle de référence, scan échoué", "err", err)
			return domain.CSRReferenceControl{}
		}
		rows = append(rows, row)
	}
	if err := rs.Err(); err != nil {
		slog.WarnContext(ctx, "csr-coverage: contrôle de référence, itération échouée", "err", err)
		return domain.CSRReferenceControl{}
	}
	return controlCSRAgainstReference(rows)
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
