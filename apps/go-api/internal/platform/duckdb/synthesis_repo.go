// Package duckdb â€” synthesis_repo.go : implÃ©mentation DuckDB du SynthesisRepository.
//
// Sprint 55 D1 : extrait de SquadRepo + CareerRepo pour port.SynthesisRepository.
// Combine les donnÃ©es de synthÃ¨se (matchs, heatmap) depuis SquadRepo
// et les encounters depuis CareerRepo.
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// SynthesisRepo implÃ©mente port.SynthesisRepository.
// Wraps SquadRepo (LoadSynthesisMatches, LoadSynthesisHeatmap) +
// la requÃªte encounters directement.
type SynthesisRepo struct {
	pdb      *PlayerDB
	squadRef *SquadRepo
}

// NewSynthesisRepo crÃ©e un SynthesisRepo depuis un PlayerDB.
func NewSynthesisRepo(pdb *PlayerDB) *SynthesisRepo {
	return &SynthesisRepo{pdb: pdb, squadRef: NewSquadRepo(pdb)}
}

// LoadSynthesisMatches dÃ©lÃ¨gue Ã  SquadRepo.
func (r *SynthesisRepo) LoadSynthesisMatches(ctx context.Context, xuid string) ([]legacymatch.SynthesisMatchRow, error) {
	return r.squadRef.LoadSynthesisMatches(ctx, xuid)
}

// LoadSynthesisHeatmap dÃ©lÃ¨gue Ã  SquadRepo.
func (r *SynthesisRepo) LoadSynthesisHeatmap(ctx context.Context, xuid string) ([]domain.SynthesisHeatmapRow, error) {
	return r.squadRef.LoadSynthesisHeatmap(ctx, xuid)
}

// LoadEncounters charge les encounters du joueur (Q_encounters).
// RÃ©utilise la requÃªte Q10Encounters de CareerRepo avec le xuid fourni.
func (r *SynthesisRepo) LoadEncounters(ctx context.Context, xuid string) ([]domain.EncounterRawRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := r.pdb.Shared.Query(ctx, Q10Encounters, xuid)
	if err != nil {
		return nil, fmt.Errorf("SynthesisRepo.LoadEncounters: %w", err)
	}
	defer rows.Close()

	var results []domain.EncounterRawRow
	for rows.Next() {
		var e domain.EncounterRawRow
		if err := rows.Scan(
			&e.XUID, &e.Gamertag, &e.MatchCount, &e.AsTeammate, &e.AsEnemy, &e.AvgKDA,
		); err != nil {
			return nil, fmt.Errorf("SynthesisRepo.LoadEncounters scan: %w", err)
		}
		results = append(results, e)
	}
	return results, rows.Err()
}
