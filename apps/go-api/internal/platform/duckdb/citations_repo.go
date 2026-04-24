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
