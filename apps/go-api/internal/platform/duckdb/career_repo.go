// Package duckdb — CareerRepo : données de progression de carrière.
//
// Le code est découpé en fichiers thématiques pour respecter la limite des
// 500 lignes par fichier (CLAUDE.md). Ce fichier contient le constructeur,
// les helpers de configuration CSR et les 2 lectures de base (latest rank +
// XP history). Les autres responsabilités vivent dans :
//
//   - career_repo_lusr.go        — historique LUSR (skill rating)
//   - career_repo_top_matches.go — top 10 WIN / top 10 LOSS par perf_score
//   - career_repo_highlights.go  — matchs marquants (highlights + pool + i18n)
//   - career_repo_encounters.go  — encounters globaux + rivals
//   - career_repo_csr.go         — snapshots CSR par playlist ranked
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// careerEncountersTimeout : limite hard pour Q26 (encounters scope global,
// agrège tous les matchs du joueur + JOIN killer_victim_pairs).
const careerEncountersTimeout = 30 * time.Second

// careerRivalsTimeout : Q27 (agrégat global killer_victim_pairs sur le joueur).
const careerRivalsTimeout = 20 * time.Second

// CareerRepo implémente port.CareerRepository.
type CareerRepo struct {
	pdb            *PlayerDB
	thresholdsRepo *CSRThresholdsRepo // optionnel : sans repo, default=5
	currentCSRSID  string             // saison CSR courante (vide → default)
}

// WithCSRThresholds injecte le repo de lookup season → seuil placement CSR
// (Phase 6 du plan pipeline CSR). Optionnel.
func (r *CareerRepo) WithCSRThresholds(repo *CSRThresholdsRepo, currentSeasonID string) *CareerRepo {
	r.thresholdsRepo = repo
	r.currentCSRSID = currentSeasonID
	return r
}

// csrThreshold retourne le seuil placement pour une saison. Helper interne avec
// dégradation gracieuse si thresholdsRepo n'est pas injecté.
func (r *CareerRepo) csrThreshold(seasonID string) int {
	if r.thresholdsRepo == nil {
		return CSRPlacementThresholdDefault
	}
	return r.thresholdsRepo.Get(context.Background(), seasonID)
}

// NewCareerRepo crée un CareerRepo depuis un PlayerDB.
func NewCareerRepo(pdb *PlayerDB) *CareerRepo {
	return &CareerRepo{pdb: pdb}
}

// GetLatestRank retourne la dernière entrée de progression de rang.
func (r *CareerRepo) GetLatestRank(ctx context.Context) (*domain.CareerRankData, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var row domain.CareerRankData
	err := r.pdb.ReadDB().QueryRow(ctx, Q6CareerLatestRank).Scan(
		&row.RankNumber,
		&row.CurrentXP,
		&row.RecordedAt,
		&row.RankLabel,
		&row.RankName,
		&row.RankTier,
		&row.XPForNextRank,
		&row.XPTotal,
		&row.IsMaxRank,
	)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetLatestRank: %w", err)
	}
	return &row, nil
}

// GetXPHistory retourne l'historique XP complet.
func (r *CareerRepo) GetXPHistory(ctx context.Context) ([]domain.XPHistoryPoint, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q7CareerXPHistory)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetXPHistory: %w", err)
	}
	defer rows.Close()

	var results []domain.XPHistoryPoint
	for rows.Next() {
		var p domain.XPHistoryPoint
		if err := rows.Scan(&p.RecordedAt, &p.Rank, &p.CurrentXP, &p.XPTotal); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetXPHistory scan: %w", err)
		}
		results = append(results, p)
	}
	return results, rows.Err()
}
