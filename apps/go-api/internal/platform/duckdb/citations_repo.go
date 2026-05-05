// Package duckdb — citations_repo.go : accès DB pour les pages Citations et Commendations.
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// CitationsRepo implémente port.CitationsRepository.
type CitationsRepo struct {
	pdb *PlayerDB
}

// NewCitationsRepo crée un CitationsRepo pour un joueur.
func NewCitationsRepo(pdb *PlayerDB) *CitationsRepo {
	return &CitationsRepo{pdb: pdb}
}

// LoadCitationMappings charge les mappings de citations depuis metadata.duckdb (Q34).
// Utilise pdb.Metadata — pas pdb.Player.
func (r *CitationsRepo) LoadCitationMappings(ctx context.Context) ([]domain.CitationMappingRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.Metadata.Query(ctx, Q34CitationMappings)
	if err != nil {
		return nil, fmt.Errorf("LoadCitationMappings: %w", err)
	}
	defer rows.Close()

	var result []domain.CitationMappingRow
	for rows.Next() {
		var row domain.CitationMappingRow
		if err := rows.Scan(
			&row.NameNorm,
			&row.NameDisplay,
			&row.MappingType,
			&row.Category,
			&row.ImagePath,
			&row.Description,
			&row.TierTargets,
		); err != nil {
			return nil, fmt.Errorf("LoadCitationMappings scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadCitationTotals charge les totaux agrégés depuis match_citations (Q35).
func (r *CitationsRepo) LoadCitationTotals(ctx context.Context) ([]domain.CitationTotalRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q35CitationTotals)
	if err != nil {
		return nil, fmt.Errorf("LoadCitationTotals: %w", err)
	}
	defer rows.Close()

	var result []domain.CitationTotalRow
	for rows.Next() {
		var row domain.CitationTotalRow
		if err := rows.Scan(
			&row.NameNorm,
			&row.Total,
		); err != nil {
			return nil, fmt.Errorf("LoadCitationTotals scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadMedalTotals charge les totaux de médailles depuis shared.medals_earned (Q36a).
func (r *CitationsRepo) LoadMedalTotals(ctx context.Context, xuid string) ([]domain.MedalEarnedRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q36aMedalTotals, xuid)
	if err != nil {
		return nil, fmt.Errorf("LoadMedalTotals: %w", err)
	}
	defer rows.Close()

	var result []domain.MedalEarnedRow
	for rows.Next() {
		var row domain.MedalEarnedRow
		if err := rows.Scan(
			&row.MedalID,
			&row.TotalCount,
		); err != nil {
			return nil, fmt.Errorf("LoadMedalTotals scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadMedalCitationMappings charge les mappings médaille→citation depuis metadata.duckdb (Q36b).
// Utilise pdb.Metadata — pas pdb.Player.
func (r *CitationsRepo) LoadMedalCitationMappings(ctx context.Context) ([]domain.MedalCitationRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.Metadata.Query(ctx, Q36bMedalCitationMappings)
	if err != nil {
		return nil, fmt.Errorf("LoadMedalCitationMappings: %w", err)
	}
	defer rows.Close()

	var result []domain.MedalCitationRow
	for rows.Next() {
		var row domain.MedalCitationRow
		if err := rows.Scan(
			&row.MedalID,
			&row.NameDisplay,
			&row.Category,
			&row.ImagePath,
		); err != nil {
			return nil, fmt.Errorf("LoadMedalCitationMappings scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadCitationMedalMappings charge les règles citation→medal_id pour le moteur de calcul (Q39).
// Utilise pdb.Metadata.
func (r *CitationsRepo) LoadCitationMedalMappings(ctx context.Context) ([]domain.CitationMedalMapping, error) {
	if r.pdb.Metadata == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.Metadata.Query(ctx, Q39CitationMedalMappings)
	if err != nil {
		return nil, fmt.Errorf("LoadCitationMedalMappings: %w", err)
	}
	defer rows.Close()

	var result []domain.CitationMedalMapping
	for rows.Next() {
		var m domain.CitationMedalMapping
		if err := rows.Scan(&m.NameNorm, &m.NameDisplay, &m.MedalID, &m.MappingType); err != nil {
			return nil, fmt.Errorf("LoadCitationMedalMappings scan: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// LoadMatchCitationsForView charge les top citations d'un match pour la vue détail (Q38).
// Utilise pdb.Player (match_citations) + pdb.Metadata (citation_mappings via LEFT JOIN
// — non disponible sans ATTACH). Si Metadata absent, retourne les lignes sans label enrichi.
func (r *CitationsRepo) LoadMatchCitationsForView(ctx context.Context, matchID string) ([]domain.CitationMatchViewRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q38MatchViewCitations, matchID)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	var result []domain.CitationMatchViewRow
	for rows.Next() {
		var row domain.CitationMatchViewRow
		if err := rows.Scan(&row.NameNorm, &row.NameDisplay, &row.Value); err != nil {
			return nil, fmt.Errorf("LoadMatchCitationsForView scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadMatchCitationsRich charge les citations d'un match avec cumul + métadonnées de paliers (Q41).
// Retourne les données nécessaires à BuildCitationSnippets (filtrage mastery, progress ring, glow).
// Dégradation silencieuse si la table match_citations est absente.
func (r *CitationsRepo) LoadMatchCitationsRich(ctx context.Context, matchID string) ([]domain.HomeMatchCitationRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q41SummaryTabCitations, matchID)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	var result []domain.HomeMatchCitationRaw
	for rows.Next() {
		var row domain.HomeMatchCitationRaw
		if err := rows.Scan(
			&row.Norm, &row.Delta, &row.Cumulative,
			&row.Display, &row.ImagePath, &row.TierTargets, &row.Description,
		); err != nil {
			return nil, fmt.Errorf("LoadMatchCitationsRich scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// WriteCitationsForMatch écrit les deltas de citations calculés dans match_citations.
// Utilise un UPSERT — si la ligne existe déjà, on n'écrase pas (idempotent).
func (r *CitationsRepo) WriteCitationsForMatch(ctx context.Context, matchID string, deltas []domain.CitationMatchDelta) error {
	if len(deltas) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rwDB, err := OpenReadWrite(r.pdb.Player.Path())
	if err != nil {
		return fmt.Errorf("WriteCitationsForMatch: open rw: %w", err)
	}
	defer rwDB.Close()

	for _, d := range deltas {
		_, err := rwDB.Exec(ctx, `
			INSERT INTO match_citations (match_id, citation_name_norm, value)
			VALUES (?, ?, ?)
			ON CONFLICT (match_id, citation_name_norm) DO NOTHING`,
			matchID, d.NameNorm, d.Value,
		)
		if err != nil {
			return fmt.Errorf("WriteCitationsForMatch insert %s: %w", d.NameNorm, err)
		}
	}
	return nil
}
