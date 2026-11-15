// boot_world_enricher.go — assemblage de l'enrichisseur du classement mondial pour le cron
// in-process : source de fetch Halo, résolveur xuid et agrégateur, câblés ensemble ici (et non dans
// `worldenrich`, qui ne dépend pas de `service`).

package main

import (
	"context"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/worldenrich"
)

// rankedPlaylistSet rend l'ensemble des asset IDs des playlists CLASSÉES actives de Halo Infinite : le
// classement mondial est CSR (classé), alors que l'historique matchmaking mêle classé et social. Le
// câblage fournit cet ensemble à l'enricher, qui n'importe aucun paquet de titre.
func rankedPlaylistSet() map[string]bool {
	pls := rankedplaylists.Active()
	out := make(map[string]bool, len(pls))
	for _, pl := range pls {
		out[pl.AssetID] = true
	}
	return out
}

// enricherOptions paramètre buildWorldEnricher.
type enricherOptions struct {
	RPS           int    // requêtes/seconde par client (défaut 5)
	TokenGamertag string // compte token ; "" → multi (tous les comptes db_profiles)
	Eager         bool   // true : résout les tokens au build (fail-fast CLI) ; false : lazy (boot serveur)
}

// buildWorldEnricher compose source de fetch + résolveur xuid + enricher (ranked-only via
// rankedPlaylistSet). En mode multi-token, le header PeopleHub est dérivé du 1er compte résolu
// (sauf TokenGamertag explicite). Retourne aussi la liste des gamertags de tokens actifs (pour le
// log de boot).
func buildWorldEnricher(ctx context.Context, cfg *config.AppConfig, opts enricherOptions) (*service.WorldStatsEnricher, []string, error) {
	rps := opts.RPS
	if rps <= 0 {
		rps = 5
	}
	var (
		src worldenrich.MatchSource
		gts []string
		err error
	)
	if strings.TrimSpace(opts.TokenGamertag) == "" {
		src, gts, err = worldenrich.BuildMultiHaloSource(ctx, cfg, rps, opts.Eager)
	} else {
		src, err = worldenrich.BuildHaloSource(ctx, cfg, opts.TokenGamertag, rps, opts.Eager)
		gts = []string{opts.TokenGamertag}
	}
	if err != nil {
		return nil, nil, err
	}
	resolverGT := strings.TrimSpace(opts.TokenGamertag)
	if resolverGT == "" && len(gts) > 0 {
		resolverGT = gts[0]
	}
	resolver, err := worldenrich.BuildResolver(cfg, resolverGT)
	if err != nil {
		return nil, nil, err
	}
	enr := service.NewWorldStatsEnricher(src, resolver, service.WorldStatsAggregatorConfig{
		RankedPlaylists: rankedPlaylistSet(),
	})
	return enr, gts, nil
}
