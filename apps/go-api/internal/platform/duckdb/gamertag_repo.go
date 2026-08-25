// Package duckdb --- GamertagRepo : recherche de gamertags dans xuid_aliases.
package duckdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
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

// ResolveXUIDsByGamertags fait le chemin INVERSE de ResolveGamertags : un set
// borné de gamertags → leurs xuids, via le même chokepoint v_gamertag_lookup.
// Utilisé par la présence en jeu pour traduire la liste `friend_gamertags` des
// Réglages (saisie à la main) en identifiants Xbox interrogeables.
//
// Insensible à la CASSE : la liste des Réglages est tapée par un humain, alors
// que la vue porte l'orthographe officielle du gamertag. Les clés de la map
// retournée sont les gamertags DEMANDÉS (tels que fournis), pour que l'appelant
// retrouve les siens sans re-normaliser.
//
// Sémantique : un gamertag jamais croisé (absent de la vue) est ABSENT de la
// map — l'appelant décide (ici : ami non compté). Les bots sont exclus. Si deux
// xuids partagent un gamertag (compte renommé puis repris), on retient le plus
// grand — arbitraire mais déterministe, et sans effet sur un compteur d'amis.
func (r *GamertagRepo) ResolveXUIDsByGamertags(ctx context.Context, gamertags []string) (map[string]string, error) {
	uniq := dedupNonEmpty(gamertags)
	out := make(map[string]string, len(uniq))
	if len(uniq) == 0 {
		return out, nil
	}

	lowered := make([]string, 0, len(uniq))
	for _, gt := range uniq {
		lowered = append(lowered, strings.ToLower(gt))
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	db, release, err := r.shared.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GamertagRepo.ResolveXUIDsByGamertags shared reader: %w", err)
	}
	defer release()

	q := fmt.Sprintf(`
		SELECT lower(gamertag) AS gt, MAX(xuid) AS xuid
		FROM v_gamertag_lookup
		WHERE gamertag IS NOT NULL AND gamertag != '' AND lower(gamertag) IN (%s)
		  AND %s
		GROUP BY lower(gamertag)`,
		Placeholders(len(lowered)), analysis.SQLIsNotBotCol("xuid"))
	rows, err := db.QueryContext(ctx, q, ToAnySlice(lowered)...)
	if err != nil {
		return nil, fmt.Errorf("GamertagRepo.ResolveXUIDsByGamertags: %w", err)
	}
	defer rows.Close()

	byLower := make(map[string]string, len(uniq))
	for rows.Next() {
		var gt, xuid string
		if err := rows.Scan(&gt, &xuid); err != nil {
			return nil, fmt.Errorf("GamertagRepo.ResolveXUIDsByGamertags scan: %w", err)
		}
		if xuid != "" {
			byLower[gt] = xuid
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, gt := range uniq {
		if xuid, ok := byLower[strings.ToLower(gt)]; ok {
			out[gt] = xuid
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
