package profile

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/platform/duckdb"
)

// service.go — orchestre les queries DuckDB pour bâtir un PlayerProfile.
//
// Une instance Service ouvre la connexion stats.duckdb du joueur via le
// wrapper duckdb.DB. Toutes les queries sont read-only (le service ne
// modifie pas l'état).

// Service expose les lectures du profil de progression.
type Service struct {
	db *duckdb.DB
}

// NewService construit un Service à partir d'une connexion stats.duckdb.
func NewService(db *duckdb.DB) *Service {
	return &Service{db: db}
}

// Load construit le PlayerProfile pour (userID, titleSlug), avec une fenêtre
// LOWESS sur les `lowessWindowDays` derniers jours de matchs ratés (LUSR).
//
// now est le timestamp de référence (cutoff de la fenêtre). En prod = time.Now() ;
// en test = fixé pour déterminisme.
//
// Retourne un PlayerProfile partiellement rempli si la DB n'a pas encore
// MinMatchesForRating matchs : LUSR sera empty, Tier/NextTier empty, MuTrend
// sans slope. Aucune erreur dans ce cas (cas légitime utilisateur récent).
func (s *Service) Load(ctx context.Context, userID, titleSlug string, lowessWindowDays int, now time.Time) (*PlayerProfile, error) {
	profile := &PlayerProfile{
		UserID:    userID,
		TitleSlug: titleSlug,
		UpdatedAt: now,
	}

	// Snapshot LUSR courant + count des matchs ratés.
	lusr, err := s.loadLUSRSnapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("loadLUSRSnapshot: %w", err)
	}
	profile.LUSR = lusr

	// Si rating non fiable, on s'arrête là (Tier/MuTrend restent zero).
	if lusr.MatchesCount < MinMatchesForRating {
		slog.DebugContext(ctx, "profile: skip tier+trend, insufficient matches",
			"user_id", userID, "matches", lusr.MatchesCount, "min", MinMatchesForRating)
		return profile, nil
	}

	profile.Tier = TierFromMu(lusr.Mu)
	profile.NextTier = NextTierFromMu(lusr.Mu)

	// Série μ pour LOWESS (chronologique ascendante).
	muSeries, err := s.loadMuSeries(ctx, now.AddDate(0, 0, -lowessWindowDays), now)
	if err != nil {
		return nil, fmt.Errorf("loadMuSeries: %w", err)
	}
	if len(muSeries) >= 3 {
		profile.MuTrend = ComputeMuTrend(muSeries)
	}
	return profile, nil
}

// loadLUSRSnapshot lit le dernier rating LUSR + count total.
func (s *Service) loadLUSRSnapshot(ctx context.Context) (LUSRState, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Dernière ligne LUSR par start_time.
	var (
		lusr         LUSRState
		rating       sql.NullFloat64
		deviation    sql.NullFloat64
		lastMatchAt  sql.NullTime
	)
	err := s.db.QueryRow(ctx, `
		SELECT rating_value, rating_deviation, start_time
		FROM match_skill_rank
		WHERE rating_type = 'LUSR' AND rating_value IS NOT NULL
		ORDER BY start_time DESC NULLS LAST
		LIMIT 1
	`).Scan(&rating, &deviation, &lastMatchAt)
	if err != nil && err != sql.ErrNoRows {
		return lusr, fmt.Errorf("query latest rating: %w", err)
	}
	if rating.Valid {
		lusr.Mu = rating.Float64
	}
	if deviation.Valid {
		lusr.Sigma = deviation.Float64
	}
	if lastMatchAt.Valid {
		t := lastMatchAt.Time
		lusr.LastMatchAt = &t
	}

	// Count.
	if err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM match_skill_rank
		WHERE rating_type = 'LUSR' AND rating_value IS NOT NULL
	`).Scan(&lusr.MatchesCount); err != nil {
		return lusr, fmt.Errorf("query count: %w", err)
	}
	return lusr, nil
}

// loadMuSeries retourne la série de μ ordonnée chronologiquement (ASC) sur
// la fenêtre [since, until]. Limite implicite : on prend tout, l'orchestrateur
// choisit la fenêtre. Liste vide si pas de matchs.
func (s *Service) loadMuSeries(ctx context.Context, since, until time.Time) ([]float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := s.db.Query(ctx, `
		SELECT rating_value
		FROM match_skill_rank
		WHERE rating_type = 'LUSR' AND rating_value IS NOT NULL
		  AND start_time >= ? AND start_time <= ?
		ORDER BY start_time ASC
	`, since, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []float64
	for rows.Next() {
		var v sql.NullFloat64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		if v.Valid {
			out = append(out, v.Float64)
		}
	}
	return out, rows.Err()
}
