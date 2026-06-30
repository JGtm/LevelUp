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
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
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

// titleSlug retourne le slug du titre du joueur (mirroir exact de
// HomeRepo.titleSlug) : trim de pdb.TitleSlug, fallback titlepkg.DefaultSlug
// si vide. Source du slug pour la résolution d'URL d'assets (badges CSR/LUSR).
func (r *CareerRepo) titleSlug() string {
	if r == nil || r.pdb == nil {
		return titlepkg.DefaultSlug
	}
	trimmed := strings.TrimSpace(r.pdb.TitleSlug)
	if trimmed == "" {
		return titlepkg.DefaultSlug
	}
	return trimmed
}

// NewCareerRepo crée un CareerRepo depuis un PlayerDB.
func NewCareerRepo(pdb *PlayerDB) *CareerRepo {
	return &CareerRepo{pdb: pdb}
}

// GetLatestRank retourne le dernier état de progression de rang (per-field-merged
// via ARG_MAX, cf. Q6CareerLatestRank), enrichi des XP dérivées depuis la
// metadata career_ranks.
//
// Deux étapes complémentaires (le snapshot live le plus récent est souvent un
// partial où xp_for_next_rank/xp_total manquent — cf. career_progression_partial.go) :
//  1. Q6 ARG_MAX : rang/current_xp/identité = dernière valeur non-vide par colonne.
//  2. EnrichFromMetadata : recalcule xp_for_next_rank + xp_total depuis career_ranks
//     (source de vérité), exactement comme le flux home/live. Sans ça les jauges
//     "progression rang/Héros" et "XP prochain rang" tombent à 0.
func (r *CareerRepo) GetLatestRank(ctx context.Context) (*domain.CareerRankData, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var (
		row        CareerRankRow
		recordedAt sql.NullTime
		rankName   sql.NullString
		rankTier   sql.NullString
	)
	err := r.pdb.ReadDB().QueryRow(ctx, Q6CareerLatestRank).Scan(
		&row.Rank,
		&row.CurrentXP,
		&recordedAt,
		&rankName,
		&rankTier,
		&row.XPForNextRank,
		&row.XPTotal,
		&row.IsMaxRank,
	)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetLatestRank: %w", err)
	}
	if !recordedAt.Valid {
		// MAX(recorded_at) NULL ⟺ aucune ligne career_progression. Préserve le
		// contrat historique (sql.ErrNoRows traité gracieusement par le DataAdapter).
		return nil, fmt.Errorf("CareerRepo.GetLatestRank: %w", sql.ErrNoRows)
	}
	if rankName.Valid {
		row.RankName = rankName.String
	}
	if rankTier.Valid {
		row.RankTier = rankTier.String
	}

	// Recalcul XP depuis la metadata (parité flux home). Best-effort : un échec
	// (metadata non attachée, rang absent du catalog) laisse les valeurs ARG_MAX
	// en place plutôt que de casser la page.
	if err := NewCareerLiveRepo(r.pdb).EnrichFromMetadata(ctx, &row); err != nil {
		slog.WarnContext(ctx, "CareerRepo.GetLatestRank: enrich metadata failed", "err", err)
	}

	return careerRankRowToData(&row, recordedAt.Time), nil
}

// careerRankRowToData projette le CareerRankRow (per-field-merged + enrichi) vers
// le type de transfert domain.CareerRankData attendu par le service Carrière.
// RankLabel et RankName portent tous deux le nom de rang (parité avec l'ancienne
// Q6 qui aliasait rank_name → rank_label) ; le service écrase RankLabel via le
// RankCatalog localisé quand il est disponible.
func careerRankRowToData(row *CareerRankRow, recordedAt time.Time) *domain.CareerRankData {
	data := &domain.CareerRankData{
		RankNumber: row.Rank,
		CurrentXP:  row.CurrentXP,
		IsMaxRank:  row.IsMaxRank,
		RecordedAt: recordedAt,
	}
	if row.RankName != "" {
		name := row.RankName
		label := row.RankName
		data.RankName = &name
		data.RankLabel = &label
	}
	if row.RankTier != "" {
		tier := row.RankTier
		data.RankTier = &tier
	}
	xpNext := row.XPForNextRank
	xpTotal := row.XPTotal
	data.XPForNextRank = &xpNext
	data.XPTotal = &xpTotal
	return data
}

// GetXPHistory retourne l'historique XP complet.
func (r *CareerRepo) GetXPHistory(ctx context.Context) ([]domain.XPHistoryPoint, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q7CareerXPHistory, r.pdb.XUID)
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
