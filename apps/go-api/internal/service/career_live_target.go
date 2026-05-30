// Package service — career_live_target.go : résolution SYNCHRONE de l'identité
// Spartan d'une cible arbitraire (cas Explorer), sans persistance player DB.
//
// Pourquoi un chemin distinct de GetSpartanIdentityFor (career_live_service.go) :
//
//	GetSpartanIdentityFor sert le hot-path Home : stale-while-revalidate, zéro
//	HTTP synchrone, et le refresh background (qui PERSISTE en career_progression)
//	est réservé au user connecté. Pour un xuid TIERS, ce chemin ne lit que
//	cache + DB locale et ne déclenche aucun fetch live → l'appearance d'un
//	adversaire non suivi n'est jamais récupérée.
//
//	FetchLiveIdentity est le pendant délibéré : un deep-fetch SYNCHRONE et borné
//	d'une cible unique (l'Explorer l'appelle dans son contexte budgété, au même
//	titre que le service record / CSR). Il réutilise les helpers cachés
//	(fetchProgressCached / fetchCustomizationCached) qui n'alimentent QUE le
//	cache mémoire — jamais persistPartial — donc aucun risque de polluer la
//	career_progression du user connecté avec les données d'un autre joueur.
//
// Dégradation : pour un joueur tiers, /careerranks et /customization/appearance
// sont player-gated (403). La progression (career rank) est donc absente, mais
// la customisation retombe sur la vue publique côté HaloAPIClient
// (GetSpartanCustomization → /customization?view=public) qui expose
// emblem/backdrop/service-tag pour n'importe quel xuid. On rend alors l'identité
// visuelle sans le rang carrière. Si aucune donnée live n'est disponible, on
// retombe sur la DB locale (cible suivie). Retourne nil si rien n'est résolu.
package service

import (
	"context"
	"log/slog"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// FetchLiveIdentity résout l'identité Spartan d'un xuid arbitraire par un fetch
// live synchrone (career progress + customization), met le résultat en cache
// mémoire, et NE PERSISTE PAS en player DB. Voir l'en-tête de fichier pour la
// distinction avec GetSpartanIdentityFor (hot-path Home).
func (s *CareerLiveService) FetchLiveIdentity(ctx context.Context, xuid string) (*domain.HomeSpartanIdentityRow, error) {
	// Filet DB systématique (cible déjà connue localement). includePeaks=false :
	// les skill peaks lus sur la player DB sont ceux du propriétaire de la page,
	// pas du target — même règle que GetSpartanIdentityFor pour un xuid tiers.
	dbFallback := s.serveDBFallback(ctx, xuid, false)
	if strings.TrimSpace(xuid) == "" {
		return dbFallback, nil
	}

	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil || tokens.SpartanToken == "" {
		// Sans auth, pas de fetch live possible : on sert ce que la DB sait.
		return dbFallback, nil
	}

	dbLast, err := s.repo.LoadLastCareerRank(ctx, xuid)
	if err != nil {
		slog.WarnContext(ctx, careerLiveLogModule+": FetchLiveIdentity LoadLastCareerRank failed",
			"xuid", xuid, "err", err)
		dbLast = nil
	}

	// Fetch SYNCHRONE, cache-aware (singleflight). fetchProgressCached /
	// fetchCustomizationCached écrivent uniquement le cache mémoire — aucune
	// écriture player DB, donc aucun persistPartial pour un xuid tiers.
	progress := s.fetchProgressCached(ctx, xuid)
	custom := s.fetchCustomizationCached(ctx, xuid)

	merged := mergeCareerRow(progress, custom, dbLast)
	if merged != nil {
		if metaErr := s.repo.EnrichFromMetadata(ctx, merged); metaErr != nil {
			slog.WarnContext(ctx, careerLiveLogModule+": FetchLiveIdentity EnrichFromMetadata failed",
				"xuid", xuid, "err", metaErr)
		}
	}

	identity := s.builder.BuildSpartanIdentityFromCareerRow(ctx, merged, false)
	identity = overlayIdentityFromFallback(identity, dbFallback)
	if identity == nil {
		careerLiveIdentityMissing.Add(1)
		careerLiveEmptyResult.Add(1)
		slog.InfoContext(ctx, careerLiveLogModule+": FetchLiveIdentity sans résultat",
			"xuid", xuid, "had_live_progress", progress != nil, "had_live_custom", custom != nil)
		return nil, nil
	}
	careerLiveIdentityServed.Add(1)
	slog.InfoContext(ctx, careerLiveLogModule+": FetchLiveIdentity résolu",
		"xuid", xuid,
		"has_rank", identity.RankNumber > 0,
		"has_emblem", identity.EmblemImageURL != nil,
		"has_backdrop", identity.BackdropImageURL != nil,
		"has_service_tag", identity.SpartanID != nil)
	return identity, nil
}
