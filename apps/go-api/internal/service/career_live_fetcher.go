// Package service — career_live_fetcher.go : helpers d'accès live HTTP avec
// cache mémoire process-level (singleflight) pour l'API Halo Economy.
//
// Extrait de career_live_service.go (refactor V2 dette technique 2026-05-26)
// pour respecter la limite de 500 lignes par fichier (arch-rules).
//
// Responsabilités :
//   - fetchProgressCached : récupère CareerProgress depuis cache ou live API
//   - fetchCustomizationCached : pendant pour la customisation Spartan
//   - makeFetcher : construit un fetcher depuis le contexte (factory injection)
//   - CareerFetcherFactoryFromTokens : factory production basée sur les tokens ctx
//
// Règles de bord :
//   - Cache hit → return immédiatement (pas d'appel HTTP)
//   - Cache miss + factory nil → return nil (tokens absents, dégradation silent)
//   - Live nil sans erreur → log "API silent skip", ne pas cacher (évite poison)
//   - Live OK → put cache + return
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/ratebudget"
	syncpkg "levelup/go-api/internal/sync"
)

// fetchProgressCached retourne la progression depuis le cache si frais, sinon
// fait l'appel live (avec singleflight). Erreurs live → log warn + nil.
func (s *CareerLiveService) fetchProgressCached(ctx context.Context, xuid string) *domain.CareerRankSnapshot {
	// Titre du contexte : dimension de la clé de cache (isolation cross-titre V72-29).
	slug := ctxkeys.TitleSlug(ctx)
	if s.cache != nil {
		if cached, hit := s.cache.GetProgress(xuid, slug); hit {
			careerLiveProgressCache.Add(1)
			slog.DebugContext(ctx, careerLiveLogModule+": progress cache hit", "xuid", xuid)
			return cached
		}
	}

	fetcher := s.makeFetcher(ctx)
	if fetcher == nil {
		return nil
	}

	// Note : le filet 401 n'est PAS câblé ici — GetCareerProgress passe par
	// doPlayerGatedGet qui avale 401/403 (retourne nil pour dégrader sans poison cache).
	// La péremption normale du token owner est déjà couverte en amont par le cache token
	// expiry-aware (enrichWithHaloTokens / ResolveFreshPlayerTokens).
	fetch := func() (*domain.CareerRankSnapshot, error) {
		return fetcher.GetCareerProgress(ctx, xuid)
	}
	var (
		data *domain.CareerRankSnapshot
		err  error
	)
	if s.cache != nil {
		data, err = s.cache.DoProgress(xuid, slug, fetch)
	} else {
		data, err = fetch()
	}
	if err != nil {
		careerLiveProgressFail.Add(1)
		slog.WarnContext(ctx, careerLiveLogModule+": progress fetch failed",
			"xuid", xuid, "err", err)
		return nil
	}
	if data == nil {
		// L'API a répondu sans erreur mais sans données exploitables (401/403
		// silencieux ou payload non parseable). Ne pas mettre nil en cache :
		// un nil caché retourne hit=true et supprime les tentatives suivantes.
		slog.WarnContext(ctx, careerLiveLogModule+": progress fetch returned nil (API silent skip)",
			"xuid", xuid)
		return nil
	}
	careerLiveProgressLive.Add(1)
	if s.cache != nil {
		s.cache.PutProgress(xuid, slug, data)
	}
	return data
}

// fetchCustomizationCached : pendant pour la customisation (TTL 6 h).
func (s *CareerLiveService) fetchCustomizationCached(ctx context.Context, xuid string) *domain.SpartanCustomizationData {
	// Titre du contexte : dimension de la clé de cache (isolation cross-titre V72-29).
	slug := ctxkeys.TitleSlug(ctx)
	if s.cache != nil {
		if cached, hit := s.cache.GetCustomization(xuid, slug); hit {
			careerLiveCustomCache.Add(1)
			slog.DebugContext(ctx, careerLiveLogModule+": customization cache hit", "xuid", xuid)
			return cached
		}
	}

	fetcher := s.makeFetcher(ctx)
	if fetcher == nil {
		return nil
	}

	// Idem fetchProgressCached : pas de filet 401 (doPlayerGatedGet avale l'auth, et
	// le 403 sur /appearance est un gating tiers NORMAL géré par le fallback vue publique).
	fetch := func() (*domain.SpartanCustomizationData, error) {
		return fetcher.GetSpartanCustomization(ctx, xuid)
	}
	var (
		data *domain.SpartanCustomizationData
		err  error
	)
	if s.cache != nil {
		data, err = s.cache.DoCustomization(xuid, slug, fetch)
	} else {
		data, err = fetch()
	}
	if err != nil {
		careerLiveCustomFail.Add(1)
		slog.WarnContext(ctx, careerLiveLogModule+": customization fetch failed",
			"xuid", xuid, "err", err)
		return nil
	}
	if data == nil {
		slog.WarnContext(ctx, careerLiveLogModule+": customization fetch returned nil (API silent skip)",
			"xuid", xuid)
		return nil
	}
	careerLiveCustomLive.Add(1)
	if s.cache != nil {
		s.cache.PutCustomization(xuid, slug, data)
	}
	return data
}

// makeFetcher construit un fetcher depuis le contexte. Retourne nil si la
// factory n'est pas câblée ou si elle elle-même retourne nil (tokens absents).
func (s *CareerLiveService) makeFetcher(ctx context.Context) CareerFetcher {
	if s.fetcherFactory == nil {
		return nil
	}
	return s.fetcherFactory(ctx)
}

// CareerFetcherFactoryFromTokens retourne une factory qui instancie un
// HaloAPIClient depuis les tokens du contexte. requestsPerSecond contrôle le
// rate limiting du client (defaults à 10 si <= 0).
//
// Le client est jetable : un nouvel objet par requête. Le coût d'allocation
// est négligeable comparé au HTTP call lui-même, et permet de bénéficier
// systématiquement des tokens à jour (refresh rotation handled in middleware).
func CareerFetcherFactoryFromTokens(requestsPerSecond int) CareerFetcherFactory {
	return func(ctx context.Context) CareerFetcher {
		tokens := ctxkeys.HaloTokens(ctx)
		if tokens == nil || tokens.SpartanToken == "" {
			return nil
		}
		client := syncpkg.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, requestsPerSecond)
		// Sujet 2 T1 : budget PAR COMPTE partagé — le fetch career live dépense
		// le quota du même compte Xbox que le pool de sync ; sans limiteur
		// partagé, le pool ne voyait jamais cette pression (429 « surprises »).
		//
		// Finding ID3 (revue 2026-07) : le débit s'impute au PORTEUR RÉEL des
		// tokens (tokensOwnerXUID = compte connecté), PAS au sujet HaloXUID — qui,
		// après forcePageIdentityXUID, est le xuid de la PAGE. Clé sur le sujet, le
		// quota du compte connecté était dépensé hors comptage et le bucket de la
		// page faussement throttlé (retour possible des 429 que Sujet 2 éliminait).
		// Porteur absent du ctx → limiteur local (comportement historique).
		if owner := ctxkeys.TokensOwnerXUID(ctx); owner != "" {
			client = client.WithLimiter(ratebudget.ForXUID(owner, float64(requestsPerSecond)))
		}
		return client
	}
}

// Compile-time check : sync.HaloAPIClient implémente bien CareerFetcher.
var _ CareerFetcher = (*syncpkg.HaloAPIClient)(nil)
