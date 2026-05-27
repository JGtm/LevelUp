// Package duckdb — synthesis_repo.go : implémentation DuckDB du SynthesisRepository.
//
// Sprint 55 D1 : extrait de SquadRepo + CareerRepo pour port.SynthesisRepository.
// Combine les données de synthèse (matchs, heatmap) depuis SquadRepo et
// l'enrichissement i18n canonical depuis HomeRepo (cohérence cross-page).
//
// 2026-05-27 : LoadEncounters retiré — la section "Relations de jeu" du bloc
// Synthesis a été supprimée côté FE. Les encounters restent exposés par
// CareerRepo pour la page palmares/relations.
package duckdb

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

// SynthesisRepo implémente port.SynthesisRepository.
// Wraps SquadRepo (LoadSynthesisMatches, LoadSynthesisHeatmap) + HomeRepo
// (EnrichCanonicalAssetTranslations).
type SynthesisRepo struct {
	squadRef *SquadRepo
	homeRef  *HomeRepo
}

// NewSynthesisRepo crée un SynthesisRepo depuis un PlayerDB.
func NewSynthesisRepo(pdb *PlayerDB) *SynthesisRepo {
	return &SynthesisRepo{
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
