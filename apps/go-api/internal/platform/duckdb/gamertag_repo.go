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
//
// shared est un SharedReader (sharedprovider.Provider en prod, LegacySharedReader
// en tests / mode kill-switch). Acquiert un handle RO via Get() à chaque appel —
// le release est appelé via defer. Cette indirection permet à
// sharedprovider.Provider de coordonner avec les swaps RW du sync engine.
//
// Sprint B1 commit 11a : migration depuis *DB direct vers SharedReader pour
// éliminer le dernier handle RO non-coordonné qui pinnait le fichier shared
// pendant les swaps Provider (bug latent du sprint B1).
type GamertagRepo struct {
	shared SharedReader
}

// NewGamertagRepo cree un GamertagRepo depuis un SharedReader.
func NewGamertagRepo(shared SharedReader) *GamertagRepo {
	return &GamertagRepo{shared: shared}
}

// Search recherche les gamertags contenant le terme donne (ILIKE).
// Retourne au maximum 20 resultats tries par nombre de matchs.
func (r *GamertagRepo) Search(ctx context.Context, query string) ([]domain.GamertagSearchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	db, release, err := r.shared.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GamertagRepo.Search shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, Q11GamertagSearch, query)
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
