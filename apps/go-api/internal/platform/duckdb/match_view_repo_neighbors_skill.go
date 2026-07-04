// Package duckdb — match_view_repo_neighbors_skill.go : navigation (matchs
// précédent/suivant) + skill rank (CSR/LUSR + bulk shared scoreboard).
// Découpé de match_view_repo.go (god-file split, refactor 2026-05-27).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// GetMatchNeighbors retourne les matchs précédent/suivant pour la navigation (Q25).
// Exécutée sur SharedReader (ADR 0016).
func (r *MatchViewRepo) GetMatchNeighbors(ctx context.Context, xuid, matchID string) (*domain.MatchNeighbors, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return &domain.MatchNeighbors{TotalMatches: 0}, nil
	}
	defer release()
	row := sharedDB.QueryRowContext(ctx, Q25NeighborMatches, xuid, matchID)
	var nextID, prevID *string
	var currentIdx, total int
	if err := row.Scan(&nextID, &prevID, &currentIdx, &total); err != nil {
		// Match introuvable dans le scope → voisins vides
		return &domain.MatchNeighbors{TotalMatches: 0}, nil
	}
	return &domain.MatchNeighbors{
		PreviousMatchID: prevID,
		NextMatchID:     nextID,
		CurrentIndex:    currentIdx,
		TotalMatches:    total,
	}, nil
}

// GetMatchNeighborsFiltered : variante paramétrable Phase 2b. spec=nil ou
// vide → délègue à GetMatchNeighbors (chronologie globale).
//
// Le fragment SQL est produit par analysis.BuildNeighborsWhereClause avec la
// ModeTaxonomy injectée (r.modeTax.Prefixes, nil-safe). Pour les titres sans la
// notion ModeCategory, la clause est omise (dégradation silencieuse).
func (r *MatchViewRepo) GetMatchNeighborsFiltered(
	ctx context.Context,
	xuid, matchID string,
	spec *domain.MatchFilterSpec,
) (*domain.MatchNeighbors, error) {
	if spec.IsEmpty() {
		return r.GetMatchNeighbors(ctx, xuid, matchID)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	clauseRes := analysis.BuildNeighborsWhereClause(spec, r.modeTax.Prefixes)

	if len(clauseRes.IgnoredFilters) > 0 {
		slog.WarnContext(ctx, "neighbors: filters ignored",
			"match_id", matchID, "ignored", clauseRes.IgnoredFilters)
	}

	query := strings.Replace(Q25NeighborMatchesTemplate, "/*EXTRA_WHERE*/", clauseRes.SQL, 1)
	args := make([]any, 0, len(clauseRes.Args)+2)
	args = append(args, xuid)
	args = append(args, clauseRes.Args...)
	args = append(args, matchID)

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return &domain.MatchNeighbors{TotalMatches: 0}, nil
	}
	defer release()
	row := sharedDB.QueryRowContext(ctx, query, args...)
	var nextID, prevID *string
	var currentIdx, total int
	if err := row.Scan(&nextID, &prevID, &currentIdx, &total); err != nil {
		// Match hors scope filtré → voisins vides (cas normal : utilisateur
		// arrivé sur un match qui ne matche plus les filtres).
		return &domain.MatchNeighbors{TotalMatches: 0}, nil
	}
	return &domain.MatchNeighbors{
		PreviousMatchID: prevID,
		NextMatchID:     nextID,
		CurrentIndex:    currentIdx,
		TotalMatches:    total,
	}, nil
}

// GetMatchSkillRank retourne le rang compétitif pour ce match (Q22).
//
// Cross-DB split (ADR 0016) : on lit d'abord match_skill_rank sur la conn
// Player (Q22a — rating_type_raw + tier/value/delta), puis on enrichit
// rating_type via match_registry sur SharedReader (Q22b — is_ranked +
// playlist_name + pair_name). Le calcul CASE/STRPOS qui était inline dans
// Q22 est réimplémenté en Go pour respecter la séparation des connexions.
func (r *MatchViewRepo) GetMatchSkillRank(ctx context.Context, matchID string) (*domain.SkillRankRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Phase A — player.match_skill_rank.
	var (
		row           domain.SkillRankRaw
		ratingTypeRaw string
	)
	err := r.pdb.ReadDB().QueryRow(ctx, Q22aMatchSkillRankPlayer, matchID).Scan(
		&ratingTypeRaw,
		&row.TierLabel,
		&row.RatingValue,
		&row.RatingDelta,
		&row.PlaylistGroup,
		&row.Tier,
		&row.SubTier,
		&row.ExpectedWinProb,
	)
	if err != nil {
		// Absent pour les matchs non-ranked ou sans donnée skill → nil sans erreur
		return nil, nil //nolint:nilerr
	}

	// Phase B — shared.match_registry (best-effort : si la lecture échoue
	// ou si le match n'est pas dans le registry, on retombe sur ratingTypeRaw).
	row.RatingType = resolveMatchRatingType(ctx, r.sharedRead(), matchID, ratingTypeRaw)
	return &row, nil
}

// resolveMatchRatingType reproduit le CASE/STRPOS inline de l'ancien Q22 :
//   - si match_registry présent : CSR si is_ranked OU playlist_name/pair_name
//     contient "ranked" (case-insensitive), sinon LUSR.
//   - sinon : fallback sur le rating_type_raw stocké dans match_skill_rank
//     ("CSR" si égal après TRIM/UPPER, sinon LUSR).
func resolveMatchRatingType(ctx context.Context, sr SharedReader, matchID, ratingTypeRaw string) string {
	if sr != nil {
		db, release, err := sr.Get(ctx)
		if err == nil {
			defer release()
			var (
				isRanked     bool
				playlistName string
				pairName     string
			)
			scanErr := db.QueryRowContext(ctx, Q22bMatchRegistryRankedFlag, matchID).
				Scan(&isRanked, &playlistName, &pairName)
			if scanErr == nil {
				if isRanked ||
					strings.Contains(strings.ToLower(playlistName), "ranked") ||
					strings.Contains(strings.ToLower(pairName), "ranked") {
					return ratingTypeCSR
				}
				return ratingTypeLUSR
			}
		}
	}
	if strings.EqualFold(strings.TrimSpace(ratingTypeRaw), ratingTypeCSR) {
		return ratingTypeCSR
	}
	return ratingTypeLUSR
}

// GetMatchSharedCSRs retourne le CSR de tous les participants d'un match ranked
// depuis shared.match_csrs_latest (Q30). Map xuid → SkillRankRaw.
// Dégradation gracieuse : nil sans erreur si table absente ou aucune donnée.
func (r *MatchViewRepo) GetMatchSharedCSRs(ctx context.Context, matchID string) (map[string]*domain.SkillRankRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, Q30SharedMatchCSRs, matchID)
	if err != nil {
		return nil, nil //nolint:nilerr — table absente : match non-ranked ou avant migration
	}
	defer rows.Close()

	result := make(map[string]*domain.SkillRankRaw)
	for rows.Next() {
		var (
			xuid    string
			row     domain.SkillRankRaw
			subTier sql.NullInt16
		)
		if err := rows.Scan(
			&xuid,
			&row.RatingType,
			&row.TierLabel,
			&row.RatingValue,
			&row.RatingDelta,
			&row.Tier,
			&subTier,
		); err != nil {
			return nil, fmt.Errorf("GetMatchSharedCSRs scan: %w", err)
		}
		if subTier.Valid {
			v := int(subTier.Int16)
			row.SubTier = &v
		}
		copy := row
		result[xuid] = &copy
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}
