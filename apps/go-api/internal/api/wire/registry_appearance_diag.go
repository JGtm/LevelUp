// Package wire — registry_appearance_diag.go : runner du diagnostic apparence
// Spartan ID (volet 2 du plan .ai/PLAN_DIAG_APPARENCE_ADMIN_2026-07.md, Lot F).
//
// Assemble le service à la demande : profils db_profiles.json, lecteur de valeurs
// servies via le pool player DB, tokens du PROFIL via le store multi-user
// (ADR 0023 — source unique), fabrique de fetcher Halo bornée par le rate budget,
// et gating capability spartan_customizer via le registre de titres (JAMAIS de
// comparaison de slug).
package wire

import (
	"context"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
)

// appearanceDiagRPS aligne la cadence du client Halo du diagnostic sur le flow
// live carrière (le service borne aussi la durée totale des fetchs).
const appearanceDiagRPS = 10

// Compile-time : *duckdb.CareerLiveRepo satisfait le lecteur de valeurs servies
// du service (LoadLastCareerRank + LoadLastFetchStatus).
var _ service.CareerServedReader = (*duckdb.CareerLiveRepo)(nil)

// DiagnoseAppearance exécute le diagnostic apparence d'un joueur suivi pour le
// titre courant (ctx). Runner du handler admin GET /admin/diag/appearance/{slug}.
func (r *ServiceRegistry) DiagnoseAppearance(ctx context.Context, playerSlug string) (domain.AppearanceDiagnosisResponse, error) {
	titleSlug := ctxkeys.TitleSlug(ctx)
	return r.newAppearanceDiagService().Diagnose(ctx, playerSlug, titleSlug)
}

// newAppearanceDiagService câble le service avec les dépendances PROD.
func (r *ServiceRegistry) newAppearanceDiagService() *service.AppearanceDiagService {
	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(r.cfg.RepoRoot).WatcherTokensDir())
	return service.NewAppearanceDiagService(service.AppearanceDiagDeps{
		LoadPlayers: func(ts string) ([]domain.PlayerSummary, error) {
			return r.cfg.LoadPlayers(ts)
		},
		ResolveServedReader: func(ctx context.Context, playerSlug, ts string) (service.CareerServedReader, error) {
			pdb, err := config.ResolvePlayer(ctx, r.cfg, playerSlug, ts)
			if err != nil {
				return nil, err
			}
			return duckdb.NewCareerLiveRepo(pdb), nil
		},
		FetchTokens: func(ctx context.Context, xuid, gamertag string) (string, string, error) {
			res, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, r.provider, xuid, gamertag, auth.LegacyAuthInputs{})
			if err != nil {
				return "", "", err
			}
			if res == nil || res.Tokens == nil {
				return "", "", nil
			}
			return res.Tokens.SpartanToken, res.Tokens.ClearanceToken, nil
		},
		NewFetcher: service.NewHaloAppearanceFetcher(appearanceDiagRPS),
		HasSpartanCustomizer: func(ts string) bool {
			desc := titlePkg.DefaultRegistry().Get(ts)
			return desc != nil && desc.HasCapability(titlePkg.CapSpartanCustomizer)
		},
	})
}
