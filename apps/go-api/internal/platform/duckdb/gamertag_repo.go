// Package duckdb --- GamertagRepo : recherche de gamertags dans xuid_aliases.
package duckdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
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

// ResolveGamertags résout les xuids d'un MATCH → gamertag par l'annuaire du match en portée base
// (lot A du plan perf « lectures par périmètre », 2026-09-26, ADR 0036 I1) : la cascade de la vue
// canonique des noms (bot, alias, participants et kill-feed de toute la base), sans l'évaluer —
// elle coûtait 2,1 s par appel. Implémente port.GamertagResolver.
//
// Sémantique : seuls les xuids que la cascade NOMME sont dans la map ; un xuid qu'elle laisse au
// libellé masqué en est ABSENT — le caller laisse l'identité sans gamertag et le rendu applique
// le masquage (front displayPlayerName, même libellé « Joueur #### » qu'analysis.MaskedXuidLabel).
// Écart nommé (DA.5) : la vue rendait ce libellé côté serveur pour un xuid connu d'une source
// mais sans nom ; il est désormais posé par le front, identique. xuids vide/dédupliqué-vide →
// map vide.
func (r *GamertagRepo) ResolveGamertags(ctx context.Context, matchID string, xuids []string) (map[string]string, error) {
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

	defer timing.FromContext(ctx).Section("resolve_gamertags_annuaire")()
	lecture := lectureANommer{xuids: uniq, porteeBase: true}
	if matchID != "" {
		lecture.matchIDs = []string{matchID}
		lecture.matchs = make([]string, len(uniq))
		for i := range lecture.matchs {
			lecture.matchs[i] = matchID
		}
	}
	noms, err := annuaireDeLecture(ctx, db, lecture)
	if err != nil {
		return nil, fmt.Errorf("GamertagRepo.ResolveGamertags: %w", err)
	}
	for _, x := range uniq {
		if noms.Nomme(x) {
			out[x] = noms.Resolve(x)
		}
	}
	return out, nil
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
