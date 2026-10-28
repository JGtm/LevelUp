// Package duckdb — match_view_repo_neighbors_skill.go : navigation (matchs
// précédent/suivant) + skill rank (CSR/LUSR + bulk shared scoreboard).
// Découpé de match_view_repo.go (god-file split, refactor 2026-05-27).
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// GetMatchNeighbors retourne les matchs précédent/suivant pour la navigation (Q25).
// Exécutée sur le SharedReader LIVE (ADR 0016), jamais le snapshot : voir neighborsRead.
// Un match absent du scope (`sql.ErrNoRows`) rend des voisins vides : c'est une absence constatée ; un lecteur indisponible ou une requête en échec
// remonte.
func (r *MatchViewRepo) GetMatchNeighbors(ctx context.Context, xuid, matchID string) (*domain.MatchNeighbors, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sharedDB, release, err := r.neighborsRead().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchNeighbors: shared reader: %w", err)
	}
	defer release()
	q := resolveCampaignExclusion(Q25NeighborMatches, r.pdb.TitleSlug, "mr")
	row := sharedDB.QueryRowContext(ctx, q, xuid, matchID)
	var nextID, prevID *string
	var currentIdx, total int
	scanErr := row.Scan(&nextID, &prevID, &currentIdx, &total)
	if errors.Is(scanErr, sql.ErrNoRows) {
		return &domain.MatchNeighbors{TotalMatches: 0}, nil
	}
	if scanErr != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchNeighbors: Q25: %w", scanErr)
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

	// Masquage Campagne (alias mr = match_registry dans Q25) : concaténé au
	// fragment de filtres, injecté dans /*EXTRA_WHERE*/. No-op pour Infinite.
	extraWhere := clauseRes.SQL + excludeCampaignClause(r.pdb.TitleSlug, "mr")
	query := strings.Replace(Q25NeighborMatchesTemplate, "/*EXTRA_WHERE*/", extraWhere, 1)
	args := make([]any, 0, len(clauseRes.Args)+2)
	args = append(args, xuid)
	args = append(args, clauseRes.Args...)
	args = append(args, matchID)

	sharedDB, release, err := r.neighborsRead().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchNeighborsFiltered: shared reader: %w", err)
	}
	defer release()
	row := sharedDB.QueryRowContext(ctx, query, args...)
	var nextID, prevID *string
	var currentIdx, total int
	scanErr := row.Scan(&nextID, &prevID, &currentIdx, &total)
	if errors.Is(scanErr, sql.ErrNoRows) {
		// Match hors scope filtré → voisins vides (cas normal : utilisateur
		// arrivé sur un match qui ne matche plus les filtres).
		return &domain.MatchNeighbors{TotalMatches: 0}, nil
	}
	if scanErr != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchNeighborsFiltered: Q25: %w", scanErr)
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
//
// Un match sans ligne de rang (match non classé ou sans donnée de skill : `sql.ErrNoRows`) est une
// ABSENCE constatée : nil. Toute autre défaillance (Reopen impossible, vue absente, ligne illisible,
// registre illisible) remonte : afficher un rang d'un type incertain serait pire que le dire.
func (r *MatchViewRepo) GetMatchSkillRank(ctx context.Context, matchID string) (*domain.SkillRankRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Phase A — player.match_skill_rank.
	var (
		row           domain.SkillRankRaw
		ratingTypeRaw string
	)
	rrows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, Q22aMatchSkillRankPlayer, matchID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchSkillRank: Q22a: %w", err)
	}
	defer rrows.Close()
	if err := rrows.Scan(
		&ratingTypeRaw,
		&row.RatingValue,
		&row.RatingDelta,
		&row.PlaylistGroup,
		&row.Tier,
		&row.SubTier,
		&row.ExpectedWinProb,
	); err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchSkillRank scan: %w", err)
	}

	// Phase B — shared.match_registry.
	ratingType, err := resolveMatchRatingType(ctx, r.sharedRead(), matchID, ratingTypeRaw)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchSkillRank: %w", err)
	}
	row.RatingType = ratingType
	return &row, nil
}

// resolveMatchRatingType reproduit le CASE/STRPOS inline de l'ancien Q22 :
//   - si match_registry présent : CSR si is_ranked OU playlist_name/pair_name
//     contient "ranked" (case-insensitive), sinon LUSR.
//   - sinon : fallback sur le rating_type_raw stocké dans match_skill_rank
//     ("CSR" si égal après TRIM/UPPER, sinon LUSR).
//
// « Sinon » ne vise que l'ABSENCE du match dans le registre (`sql.ErrNoRows`) ou l'absence de
// lecteur configuré. Un lecteur indisponible ou une requête en échec remonte en erreur.
func resolveMatchRatingType(ctx context.Context, sr SharedReader, matchID, ratingTypeRaw string) (string, error) {
	if sr != nil {
		db, release, err := sr.Get(ctx)
		if err != nil {
			return "", fmt.Errorf("registre du match: shared reader: %w", err)
		}
		defer release()
		var (
			isRanked     bool
			playlistName string
			pairName     string
		)
		scanErr := db.QueryRowContext(ctx, Q22bMatchRegistryRankedFlag, matchID).
			Scan(&isRanked, &playlistName, &pairName)
		switch {
		case scanErr == nil:
			if isRanked ||
				strings.Contains(strings.ToLower(playlistName), "ranked") ||
				strings.Contains(strings.ToLower(pairName), "ranked") {
				return ratingTypeCSR, nil
			}
			return ratingTypeLUSR, nil
		case !errors.Is(scanErr, sql.ErrNoRows):
			return "", fmt.Errorf("registre du match: Q22b: %w", scanErr)
		}
	}
	if strings.EqualFold(strings.TrimSpace(ratingTypeRaw), ratingTypeCSR) {
		return ratingTypeCSR, nil
	}
	return ratingTypeLUSR, nil
}

// GetMatchSharedCSRs retourne le CSR de tous les participants d'un match ranked
// depuis shared.match_csrs_latest (Q30). Map xuid → SkillRankRaw.
//
// Nil sans erreur = absence constatée : la vue `match_csrs_latest` n'existe pas dans ce lecteur
// (sonde du catalogue, `relationExists`) ou le match n'a aucune ligne (match non classé). Tout
// autre échec (lecteur, sonde, requête, lecture) remonte.
func (r *MatchViewRepo) GetMatchSharedCSRs(ctx context.Context, matchID string) (map[string]*domain.SkillRankRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchSharedCSRs: shared reader: %w", err)
	}
	defer release()

	present, err := relationExists(ctx, sharedDB, "match_csrs_latest")
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchSharedCSRs: %w", err)
	}
	if !present {
		slog.DebugContext(ctx, "match_view: match_csrs_latest absente de ce lecteur, aucun CSR partagé",
			"match_id", matchID)
		return nil, nil
	}

	rows, err := sharedDB.QueryContext(ctx, Q30SharedMatchCSRs, matchID)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchSharedCSRs: Q30: %w", err)
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

// neighborsRead : le lecteur partagé de Q25 — TOUJOURS le live, jamais l'override snapshot.
//
// Depuis que la liste de tuiles de l'Accueil ne transporte plus de liste d'identifiants (D-9), Q25
// est la source de la navigation prev/next ET du compteur « X/Y » : elle doit lire la MÊME source
// que cette liste (le chargeur canonique, lecture live). Un snapshot immuable est coupé à une
// heure donnée : les matchs joués depuis en sont absents (voisins vides pour eux) et le total d'un
// match plus ancien est faux. Une bascule sur absence ne suffirait pas : le total des matchs
// présents dans un snapshot ancien resterait faux.
func (r *MatchViewRepo) neighborsRead() SharedReader {
	return r.pdb.SharedReadDB()
}
