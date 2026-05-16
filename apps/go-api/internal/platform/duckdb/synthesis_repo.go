// Package duckdb — synthesis_repo.go : implémentation DuckDB du SynthesisRepository.
//
// Sprint 55 D1 : extrait de SquadRepo + CareerRepo pour port.SynthesisRepository.
// Combine les données de synthèse (matchs, heatmap) depuis SquadRepo,
// les encounters depuis CareerRepo, et l'enrichissement i18n canonical
// depuis HomeRepo (cohérence cross-page).
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

// SynthesisRepo implémente port.SynthesisRepository.
// Wraps SquadRepo (LoadSynthesisMatches, LoadSynthesisHeatmap) + HomeRepo
// (EnrichCanonicalAssetTranslations) + la requête encounters directement.
type SynthesisRepo struct {
	pdb      *PlayerDB
	squadRef *SquadRepo
	homeRef  *HomeRepo
}

// NewSynthesisRepo crée un SynthesisRepo depuis un PlayerDB.
func NewSynthesisRepo(pdb *PlayerDB) *SynthesisRepo {
	return &SynthesisRepo{
		pdb:      pdb,
		squadRef: NewSquadRepo(pdb),
		homeRef:  NewHomeRepo(pdb),
	}
}

// LoadSynthesisMatches délègue à SquadRepo.
func (r *SynthesisRepo) LoadSynthesisMatches(ctx context.Context, xuid string) ([]legacymatch.SynthesisMatchRow, error) {
	return r.squadRef.LoadSynthesisMatches(ctx, xuid)
}

// LoadSynthesisHeatmap délègue à SquadRepo.
func (r *SynthesisRepo) LoadSynthesisHeatmap(ctx context.Context, xuid string) ([]domain.SynthesisHeatmapRow, error) {
	return r.squadRef.LoadSynthesisHeatmap(ctx, xuid)
}

// EnrichCanonicalAssetTranslations délègue à HomeRepo qui possède déjà
// l'enrichissement complet Map/Playlist/GameVariant/PairMode via
// asset_translations + mode_name_tr. Même appel = même cohérence FR que la home.
func (r *SynthesisRepo) EnrichCanonicalAssetTranslations(ctx context.Context, rows []canonical.PlayerMatchRow) error {
	return r.homeRef.EnrichCanonicalAssetTranslations(ctx, rows)
}

// LoadEncounters charge les encounters du joueur (Q_encounters).
// Réutilise la requête Q10Encounters de CareerRepo avec le xuid fourni.
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
