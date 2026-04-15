// Package duckdb --- GamertagRepo : recherche de gamertags dans xuid_aliases.
package duckdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// GamertagRepo implemente port.GamertagRepository.
type GamertagRepo struct {
	shared *DB
}

// NewGamertagRepo cree un GamertagRepo depuis la DB partagee.
func NewGamertagRepo(sharedDB *DB) *GamertagRepo {
	return &GamertagRepo{shared: sharedDB}
}

// Search recherche les gamertags contenant le terme donne (ILIKE).
// Retourne au maximum 20 resultats tries par nombre de matchs.
func (r *GamertagRepo) Search(ctx context.Context, query string) ([]domain.GamertagSearchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.shared.Query(ctx, Q11GamertagSearch, query)
	if err != nil {
		return nil, fmt.Errorf("GamertagRepo.Search(%q): %w", query, err)
	}
	defer rows.Close()

	var results []domain.GamertagSearchResult
	for rows.Next() {
		var gamertag, xuid string
		var matchCount int
		if err := rows.Scan(&gamertag, &xuid, &matchCount); err != nil {
			return nil, fmt.Errorf("GamertagRepo.Search scan: %w", err)
		}
		results = append(results, domain.GamertagSearchResult{
			Gamertag:   gamertag,
			XUID:       xuid,
			Score:      float64(matchCount),
			ExactMatch: strings.EqualFold(gamertag, query),
		})
	}
	return results, rows.Err()
}
