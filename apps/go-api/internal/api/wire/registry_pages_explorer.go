// Package api — registry_pages_explorer.go : factories ServiceRegistry de la page
// Explorer + resolveurs CSR/identite/banniere. Extrait de registry_pages.go (K3f god-file
// split, 2026-07-06), meme package.
package wire

import (
	"context"
	"log/slog"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	halo_games "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
	sync_pkg "levelup/go-api/internal/sync"
)

// ExplorerCtxWithAuth retourne un ExplorerService + contexte enrichi avec les
// HaloTokens du propriétaire de la page (résolus depuis le store, ADR 0023).
//
// L'encart "Profil joueur cible" est local-first : identité lue depuis la DB
// locale de la cible (si joueur suivi), sample stats calculés localement. Seule
// la carrière agrégée (servicerecord) est un fetch live → nécessite des tokens
// dans le contexte (d'où enrichWithHaloTokens, même pattern que HomeCtxWithAuth
// et Compare). Aucune privacy n'est fetchée (bruit sans valeur).
func (r *ServiceRegistry) ExplorerCtxWithAuth(ctx context.Context, slug string) (port.ExplorerService, context.Context, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, ctx, "", "", err
	}
	svc := service.NewExplorerService(duckdb.NewExplorerRepo(pdb, pdb.XUID), pdb.XUID)
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	// Encart "Profil joueur cible" :
	//   - LocalIdentity : identité depuis la DB locale de la cible (si suivie)
	//   - LiveIdentity  : career rank/emblem/backdrop live pour un xuid arbitraire
	//   - RemoteStats   : service record (stats + temps de jeu + médailles), caché
	//   - MedalDefs     : labels/descriptions médailles (top médailles)
	//   - CSR           : classements CSR saison courante (live, tout xuid)
	homeRepo := r.newHomeRepo(pdb)
	csrSeasonID := ""
	if r.cfg != nil {
		csrSeasonID = r.cfg.CSRSeasonIDForTitle(ctx, pdb.TitleSlug, nil)
	}
	var seasons []service.SeasonCatalogEntry
	if r.seasonsCatalog != nil {
		seasons = r.seasonsCatalog.Load(ctx, pdb.TitleSlug)
	}
	// Catalogue des rangs de carrière (même source que HomeService) pour
	// localiser RankTitle/NextRankTitle du DTO d'identité cible. nil-safe.
	var ranks *mappings.RankCatalog
	if sem := r.semanticFor(pdb.TitleSlug); sem != nil {
		ranks = sem.Ranks()
	}
	// CSR live (Explorer) : providers spécifiques Infinite → n'injecter que si le
	// titre expose match.skill.snapshot, sinon nil (le service dégrade : encart CSR
	// vide, pas de fuite de playlists/CSR Infinite sous un autre titre — F2).
	var csrProvider service.ExplorerTargetCSRProvider
	var seasonCSRProvider service.ExplorerSeasonCSRProvider
	if r.titleSupportsLiveCSR(pdb) {
		csrProvider = r.newExplorerCSRProvider()
		seasonCSRProvider = r.newExplorerSeasonCSRProvider()
	}
	svc = svc.WithTargetProfileProviders(service.ExplorerTargetProfileDeps{
		LocalIdentity:   r.newExplorerLocalIdentityResolver(pdb.TitleSlug),
		LiveIdentity:    r.newCareerLiveService(pdb, homeRepo),
		RemoteStats:     r.remoteStats,
		MedalDefs:       duckdb.NewMedalDefinitionsRepo(pdb),
		CSR:             csrProvider,
		CurrentSeasonID: csrSeasonID,
		Seasons:         seasons,
		SeasonSR:        r.remoteStats, // *CachedStatsProvider implémente port.SeasonStatsProvider
		SeasonCSR:       seasonCSRProvider,
		Ranks:           ranks,
		RecentMatches:   wrapRecentMatchesAuthRetry(r.recentMatches),
		LocalBannerPool: r.newExplorerLocalBannerPool(pdb.TitleSlug),
		TitleSlug:       pdb.TitleSlug,
	})
	// Fallback live gamertag→xuid (joueur jamais croisé) — nil-safe (no-op en démo).
	svc = svc.WithLiveGamertagResolver(r.liveGamertagResolver)
	enriched := r.enrichWithHaloTokens(ctx, pdb)
	return svc, enriched, pdb.XUID, pdb.Gamertag, nil
}

// titleSupportsLiveCSR indique si le titre du joueur expose la capability
// match.skill.snapshot (Infinite = degraded → oui ; H5 = not_exposed → non).
// Les providers CSR Explorer/Compare sont spécifiques au live Infinite
// (rankedplaylists.Active() + endpoints CSR HINF) : ne les injecter que pour un
// titre déclarant cette capability évite de servir des playlists/CSR Infinite
// sous un autre titre (F2). Gate par CAPABILITY, jamais par slug (ratchet ADR 0025).
func (r *ServiceRegistry) titleSupportsLiveCSR(pdb *duckdb.PlayerDB) bool {
	a := r.dataAdapterForPDB(pdb)
	return a != nil && a.Capabilities().Has(games.CapMatchSkillSnapshot)
}

// newExplorerCSRProvider construit le provider CSR de l'encart cible : instancie
// un client Halo depuis les tokens du contexte (Spartan + clearance) et appelle
// l'endpoint skill CSR (service token, fonctionne pour tout xuid), puis mappe
// vers le type domain. nil tokens → nil (pas d'erreur).
func (r *ServiceRegistry) newExplorerCSRProvider() service.ExplorerTargetCSRProvider {
	return service.CSRProviderFunc(func(ctx context.Context, xuid, seasonID string) ([]domain.CareerPlaylistCSR, error) {
		// Filet auth (defense-in-depth) : un 401/403 sur GetPlayerCSRs (token owner
		// révoqué en cours de requête) → re-mint + retry unique. Péremption normale déjà
		// couverte par le cache token expiry-aware en amont.
		return halo.RetryWithFreshTokens(ctx, sync_pkg.IsAuthError, func(c context.Context) ([]domain.CareerPlaylistCSR, error) {
			tokens := ctxkeys.HaloTokens(c)
			if tokens == nil || tokens.SpartanToken == "" {
				return nil, nil
			}
			client := sync_pkg.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, 10)
			// 1. Playlists ranked ENGAGÉES de la saison (endpoint player-level).
			raw, err := client.GetPlayerCSRs(c, xuid, seasonID)
			if err != nil {
				return nil, err
			}
			// 2. Compléter avec les playlists ranked ACTIVES manquantes (endpoint
			//    par-playlist) — parité avec la page Carrière. Source unique
			//    partagée (H8) ; locale = requete (ctxkeys.Locale, GH-8).
			raw = sync_pkg.AugmentWithActiveRankedCSRs(c, client, xuid, seasonID, raw, ctxkeys.Locale(c))
			return mapSyncCSRsToDomain(raw), nil
		})
	})
}

// newExplorerSeasonCSRProvider construit le provider de PIC CSR PAR SAISON PASSÉE
// (badge au-dessus des barres "matchs par saison").
//
// IMPORTANT : le endpoint player-level /hi/players/.../csrs?Season= renvoie 404
// (vérifié empiriquement, y compris pour la saison courante). Le CSR par saison
// — y compris PASSÉE — n'est servi de façon fiable que par GetPlaylistCsr
// (/hi/playlist/{id}/csrs?players=...&season=, HTTP 200 + SeasonMax = pic de la
// saison). On interroge donc chaque playlist ranked active et on retient le plus
// haut tier. (nil, nil) sans tokens / si aucune donnée CSR pour la saison.
func (r *ServiceRegistry) newExplorerSeasonCSRProvider() service.ExplorerSeasonCSRProvider {
	return service.SeasonCSRPeakFunc(func(ctx context.Context, xuid, csrSeasonID string, engagedPlaylistIDs []string) (*service.SeasonCSRPeak, error) {
		tokens := ctxkeys.HaloTokens(ctx)
		if tokens == nil || tokens.SpartanToken == "" || len(engagedPlaylistIDs) == 0 {
			return nil, nil
		}
		// Optim : ne requêter que les playlists ranked actives RÉELLEMENT engagées
		// par le joueur (intersection avec Subqueries.PlaylistAssetIds). Un joueur
		// social → engaged ∩ ranked = ∅ → 0 appel CSR.
		engaged := make(map[string]struct{}, len(engagedPlaylistIDs))
		for _, id := range engagedPlaylistIDs {
			engaged[strings.ToLower(strings.TrimSpace(id))] = struct{}{}
		}
		client := sync_pkg.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, 10)

		var best *sync_pkg.CSRRankSnapshot
		for _, pl := range rankedplaylists.Active() {
			if _, ok := engaged[strings.ToLower(strings.TrimSpace(pl.AssetID))]; !ok {
				continue // playlist ranked jamais jouée par ce joueur → skip
			}
			res, err := client.GetPlaylistCsr(ctx, pl.AssetID, xuid, csrSeasonID)
			if err != nil || res == nil {
				continue // playlist absente cette saison-là / erreur → on ignore
			}
			s := res.Season // SeasonMax = pic de rang de la saison demandée
			if strings.TrimSpace(s.Tier) == "" {
				continue
			}
			if best == nil || s.Value > best.Value {
				snap := s
				best = &snap
			}
		}
		if best == nil {
			return nil, nil
		}
		peak := &service.SeasonCSRPeak{Tier: best.Tier, SubTier: best.SubTier}
		if url := csrBadgeURL(halo_games.NewAssetURLAdapter(), best.Tier, best.SubTier); url != "" {
			peak.BadgeURL = &url
		}
		return peak, nil
	})
}

// wrapRecentMatchesAuthRetry décore un RecentMatchesProvider avec le filet auth
// (defense-in-depth) : un 401/403 du token owner en cours de requête → re-mint + retry
// unique. nil → nil (pas de provider). Le re-mint cible le xuid OWNER (ctx), pas la cible.
func wrapRecentMatchesAuthRetry(inner port.RecentMatchesProvider) port.RecentMatchesProvider {
	if inner == nil {
		return nil
	}
	return authRetryRecentMatches{inner: inner}
}

type authRetryRecentMatches struct{ inner port.RecentMatchesProvider }

func (a authRetryRecentMatches) FetchRecentMatches(ctx context.Context, xuid string, limit int) ([]domain.ExplorerTargetRecentMatch, error) {
	return halo.RetryWithFreshTokens(ctx, sync_pkg.IsAuthError, func(c context.Context) ([]domain.ExplorerTargetRecentMatch, error) {
		return a.inner.FetchRecentMatches(c, xuid, limit)
	})
}

// mapSyncCSRsToDomain projette les CSR du client sync vers le type domain
// (CareerPlaylistCSR), avec résolution du badge image via l'AssetURLAdapter.
func mapSyncCSRsToDomain(in []sync_pkg.PlayerPlaylistCSR) []domain.CareerPlaylistCSR {
	adapter := halo_games.NewAssetURLAdapter()
	out := make([]domain.CareerPlaylistCSR, 0, len(in))
	for i := range in {
		c := &in[i]
		out = append(out, domain.CareerPlaylistCSR{
			PlaylistID:   c.PlaylistID,
			PlaylistName: c.PlaylistName,
			Queue:        c.Queue,
			Input:        c.Input,
			Current:      mapSyncCSRSnapshot(adapter, c.Current),
			Season:       mapSyncCSRSnapshot(adapter, c.Season),
			AllTime:      mapSyncCSRSnapshot(adapter, c.AllTime),
		})
	}
	return out
}

func mapSyncCSRSnapshot(adapter *halo_games.AssetURLAdapter, s sync_pkg.CSRRankSnapshot) domain.CareerCSRRank {
	out := domain.CareerCSRRank{
		Value:                       s.Value,
		Tier:                        s.Tier,
		SubTier:                     s.SubTier,
		MeasurementMatchesRemaining: s.MeasurementMatchesRemaining,
	}
	if url := csrBadgeURL(adapter, s.Tier, s.SubTier); url != "" {
		out.BadgeImageURL = &url
	}
	return out
}

// csrBadgeURL résout l'URL du badge CSR (/static/ranks/...) via l'AssetURLAdapter
// halo_infinite. "" si le tier est vide (non classé) ou hors plage.
func csrBadgeURL(adapter *halo_games.AssetURLAdapter, tier string, subTier int) string {
	if strings.TrimSpace(tier) == "" {
		return ""
	}
	if strings.EqualFold(tier, "Onyx") {
		return adapter.CSRRankImageURLOnyx()
	}
	if subTier < 1 || subTier > 6 {
		return ""
	}
	return adapter.CSRRankImageURL(tier, subTier)
}

// newExplorerLocalIdentityResolver construit le résolveur d'identité local de
// l'encart Explorer : si le gamertag cible correspond à un joueur suivi
// (db_profiles), on ouvre SA player DB et on lit son identité Spartan
// (rang/emblem/skill peaks). Sinon nil (un adversaire n'a pas d'identité
// publiée — aucun fetch live). resolveByGT est pool-cached.
func (r *ServiceRegistry) newExplorerLocalIdentityResolver(titleSlug string) service.ExplorerLocalIdentityResolver {
	return service.LocalIdentityResolverFunc(func(ctx context.Context, targetGamertag string) *domain.HomeSpartanIdentityRow {
		if r.resolveByGT == nil || targetGamertag == "" {
			return nil
		}
		tpdb, err := r.resolveByGT(ctx, titleSlug, targetGamertag)
		if err != nil || tpdb == nil {
			return nil // cible non suivie localement → pas d'identité
		}
		row, lerr := r.newHomeRepo(tpdb).LoadSpartanIdentity(ctx)
		if lerr != nil {
			slog.WarnContext(ctx, "explorer_local_identity_failed",
				"gamertag", targetGamertag, "err", lerr)
			return nil
		}
		return row
	})
}

// newExplorerLocalBannerPool construit le résolveur PARESSEUX du pool de
// bannières locales (Phase 3.6) : la liste dédupliquée et à ordre stable des
// banner_image_url des joueurs suivis (db_profiles). Appelé uniquement quand une
// cible non-locale n'a ni bannière ni backdrop → on lui attribue une nameplate
// de repli déterministe par xuid. resolveByGT est pool-cached ;
// LoadSpartanIdentity lit la player DB (même chemin que l'identité locale).
func (r *ServiceRegistry) newExplorerLocalBannerPool(titleSlug string) func(ctx context.Context) []string {
	return func(ctx context.Context) []string {
		if r.cfg == nil || r.resolveByGT == nil {
			return nil
		}
		players, err := r.cfg.LoadPlayers(titleSlug)
		if err != nil {
			slog.WarnContext(ctx, "explorer_banner_pool_load_players_failed", "err", err)
			return nil
		}
		seen := make(map[string]struct{}, len(players))
		out := make([]string, 0, len(players))
		for _, p := range players {
			if p.Gamertag == "" {
				continue
			}
			tpdb, gerr := r.resolveByGT(ctx, titleSlug, p.Gamertag)
			if gerr != nil || tpdb == nil {
				continue
			}
			row, lerr := r.newHomeRepo(tpdb).LoadSpartanIdentity(ctx)
			if lerr != nil || row == nil || row.BannerImageURL == nil || *row.BannerImageURL == "" {
				continue
			}
			if _, ok := seen[*row.BannerImageURL]; ok {
				continue
			}
			seen[*row.BannerImageURL] = struct{}{}
			out = append(out, *row.BannerImageURL)
		}
		slog.DebugContext(ctx, "explorer_banner_pool_built", "title_slug", titleSlug, "size", len(out))
		return out
	}
}

// skillBadgeResolverFor construit le résolveur d'URL de badge CSR title-aware
// pour un slug de titre donné. C'est le pont entre le package analysis (pur,
// title-agnostic) et la résolution title-aware de duckdb.TitleSkillBadgeURL
// (csr_designations pour Halo 5, sinon static HINF). subTier : 0 pour Onyx, 1..6 sinon.
func skillBadgeResolverFor(slug string) func(tierEN string, subTier int) string {
	return func(tierEN string, subTier int) string {
		return duckdb.TitleSkillBadgeURL(slug, tierEN, subTier)
	}
}
