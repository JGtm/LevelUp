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

// ResolveGamertags résout un set BORNÉ de xuid → gamertag via le chokepoint
// canonique v_gamertag_lookup (mêmes garanties que Q12 scoreboard : bots résolus
// en nom officiel, cascade xuid_aliases/match_participants/killer_victim_pairs,
// jamais de xuid brut). Implémente port.GamertagResolver.
//
// Sémantique : un xuid SANS gamertag résolu (orphelin hors sources) est ABSENT de
// la map retournée — le caller laisse l'identité sans gamertag et le rendu applique
// le masquage (front displayPlayerName). xuids vide/dédupliqué-vide → map vide.
func (r *GamertagRepo) ResolveGamertags(ctx context.Context, xuids []string) (map[string]string, error) {
	uniq := dedupNonEmpty(xuids)
	out := make(map[string]string, len(uniq))
	if len(uniq) == 0 {
		return out, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	db, release, err := r.shared.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GamertagRepo.ResolveGamertags shared reader: %w", err)
	}
	defer release()

	q := fmt.Sprintf(
		"SELECT xuid, gamertag FROM v_gamertag_lookup WHERE gamertag IS NOT NULL AND xuid IN (%s)",
		Placeholders(len(uniq)))
	rows, err := db.QueryContext(ctx, q, ToAnySlice(uniq)...)
	if err != nil {
		return nil, fmt.Errorf("GamertagRepo.ResolveGamertags: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var xuid, gamertag string
		if err := rows.Scan(&xuid, &gamertag); err != nil {
			return nil, fmt.Errorf("GamertagRepo.ResolveGamertags scan: %w", err)
		}
		if gamertag != "" {
			out[xuid] = gamertag
		}
	}
	return out, rows.Err()
}

// dedupNonEmpty retourne les valeurs uniques non-vides en préservant l'ordre.
func dedupNonEmpty(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
