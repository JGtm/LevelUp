// boot_world_enricher.go — assemblage de l'enrichisseur du classement mondial pour le cron
// in-process : source de fetch Halo, résolveur xuid et agrégateur, câblés ensemble ici (et non dans
// `worldenrich`, qui ne dépend pas de `service`).

package main

import (
	"context"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/worldenrich"
)

// enricherOptions paramètre buildWorldEnricher.
type enricherOptions struct {
	RPS           int    // requêtes/seconde par client (défaut 5)
	TokenGamertag string // compte token ; "" → multi (tous les comptes db_profiles)
	Eager         bool   // true : résout les tokens au build (fail-fast CLI) ; false : lazy (boot serveur)
}

// buildWorldEnricher compose source de fetch + résolveur xuid + enricher (ranked-only via
// RankedPlaylistSet). En mode multi-token, le header PeopleHub est dérivé du 1er compte résolu
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
		RankedPlaylists: service.RankedPlaylistSet(),
	})
	return enr, gts, nil
}
