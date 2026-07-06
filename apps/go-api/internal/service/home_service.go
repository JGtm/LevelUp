// Package service â€” home_service.go : service de la page d'accueil Mission Control.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/teammates"
)

// HomeService orchestre les donnÃ©es de la page d'accueil.
type HomeService struct {
	repo         port.HomeRepository
	cacheRepo    port.BattlePassCacheRepository
	provider     *halo.HaloProvider
	sink         port.HomePersistSink // nil â†’ pas de persistance (tests, joueurs sans auth)
	socialRepo   port.SocialRepository
	playerSlug   string
	semantic     games.TitleSemanticAdapter // nil â†’ libellÃ©s rangs construits via fallbacks (RankName)
	matchesCache *HomeMatchesCache          // nil â†’ pas de cache (tests, appels HomeCtx sans auth)
	xuid         string                     // clÃ© de cache ; vide si matchesCache est nil
	// dataAdapter (optionnel, Phase C+ multi-titres) : point d'extension pour
	// router LoadPlayerStats via la couche canonique. Ã€ ce jour, le service
	// utilise le repo direct car canonical.PlayerStats ne couvre pas encore
	// la totalitÃ© du payload home KPIs (favorite_playlist, avg_kda, etc.).
	// Le hook est en place pour permettre une bascule incrÃ©mentale.
	dataAdapter games.TitleDataAdapter
	// playerMatchesRepo (P4.1, ADR 0011) : loader canonical-aware optionnel.
	// Quand fourni avec titleSlug + gamertag, fetchMatchesAndSessions charge
	// canonical et convertit via homeMatchRowFromCanonical / homeSessionsFromCanonical.
	// SkillTierLabel et SkillRankImageURL sont laissÃ©s vides cÃ´tÃ© converter
	// (TODO P4.3 : enrichir via TitleSemanticAdapter.Ranks() et
	// TitleAssetURLAdapter.CSRRankImageURL une fois les adapters CSR cÃ¢blÃ©s
	// dans le service).
	playerMatchesRepo port.PlayerMatchesRepository
	titleSlug         string
	gamertag          string
	// careerLive (optionnel) : service live carrière découplé du post-sync
	// matchs. Quand câblé, remplace l'appel DB-only LoadSpartanIdentity par
	// un flow live throttle 5 min / 6 h + fallback DB per-field garanti.
	// Si nil, le service retombe sur l'ancien chemin (repo.LoadSpartanIdentity)
	// — utilisé par les tests et les bootstraps minimaux sans auth.
	careerLive homeSpartanIdentityProvider
	// demoMode : en démo, certaines sections sans source réelle (défis : pas d'API
	// live + cache TTL 24h) sont servies depuis des fixtures embarquées au lieu de
	// renvoyer vide. Cf. home_service_demo.go.
	demoMode bool
	// skillBadgeResolver (optionnel, multi-titres) : résout l'URL du badge CSR
	// title-aware (csr_designations h5 sinon static HINF). Injecté au boot avec le
	// slug du titre du joueur. Nil → SkillRankImageURL laissé vide côté tuiles de
	// match (dégradation gracieuse, le label suffit). Signature (tierEN capitalisé,
	// subTier 0..6 ; 0 = Onyx) → URL. Garde le package analysis title-agnostic.
	skillBadgeResolver func(tierEN string, subTier int) string
	// sessionTeammatesLoader (optionnel) : charge les coéquipiers (même équipe que
	// le joueur principal) sur une liste de matchs, pour renseigner
	// SessionSummaryItem.Teammates des sessions escouade (deep-link card → /squad).
	// Implémenté par *duckdb.SquadRepo. nil → Teammates restent vides (dégradation).
	// Cf. home_squad_session_teammates.go.
	sessionTeammatesLoader mainTeamParticipantsLoader
	// sessionFriendsResolver (optionnel) : restreint les coéquipiers de session aux
	// amis configurés (settings.friend_gamertags). nil → tous les coéquipiers alliés.
	sessionFriendsResolver teammates.FriendGamertagsResolver
}

// NewHomeService crÃ©e un HomeService avec le repository et le provider Halo.
func NewHomeService(repo port.HomeRepository) *HomeService {
	return &HomeService{
		repo:     repo,
		provider: halo.DefaultHaloProvider,
	}
}

// WithDemoMode active les fixtures de contenu démo (défis…) servies au niveau
// lecture quand aucune source réelle n'existe en démo. Retourne le service (chaînage).
func (s *HomeService) WithDemoMode(demo bool) *HomeService {
	s.demoMode = demo
	return s
}

// WithSkillBadgeResolver injecte le résolveur d'URL de badge CSR title-aware
// (csr_designations pour les titres additionnels, sinon static HINF). Câblé au
// boot avec le slug du titre du joueur. Dégradation gracieuse si nil (badge vide
// sur les tuiles de match — le label de palier reste affiché). Retourne le service
// pour permettre le chaînage.
func (s *HomeService) WithSkillBadgeResolver(f func(tierEN string, subTier int) string) *HomeService {
	s.skillBadgeResolver = f
	return s
}

// WithHaloProvider remplace le provider Halo utilisÃ© par le service.
// Utile pour injecter un provider configurÃ© par joueur (cache local, tests).
func (s *HomeService) WithHaloProvider(provider *halo.HaloProvider) *HomeService {
	if provider != nil {
		s.provider = provider
	}
	return s
}

// WithPersistSink configure le sink de persistance fire-and-forget.
// Retourne le service pour permettre le chaÃ®nage.
func (s *HomeService) WithPersistSink(sink port.HomePersistSink) *HomeService {
	s.sink = sink
	return s
}

// WithCacheRepo configure le repository de cache BP/Challenges.
// Retourne le service pour permettre le chaÃ®nage.
func (s *HomeService) WithCacheRepo(r port.BattlePassCacheRepository) *HomeService {
	s.cacheRepo = r
	return s
}

// WithSocial configure le repository social (favoris) et le slug joueur.
// Retourne le service pour permettre le chaÃ®nage.
func (s *HomeService) WithSocial(repo port.SocialRepository, playerSlug string) *HomeService {
	s.socialRepo = repo
	s.playerSlug = playerSlug
	return s
}

// WithSemanticAdapter injecte le SemanticAdapter du titre courant pour rÃ©soudre
// les libellÃ©s des rangs de carriÃ¨re (Ranks() expose un *mappings.RankCatalog).
// Si nil, les libellÃ©s tombent sur le fallback RankName de la player DB.
func (s *HomeService) WithSemanticAdapter(semantic games.TitleSemanticAdapter) *HomeService {
	s.semantic = semantic
	return s
}

// WithDataAdapter injecte le DataAdapter multi-titres pour activer une
// future bascule LoadPlayerStats. DÃ©gradation gracieuse si nil.
func (s *HomeService) WithDataAdapter(a games.TitleDataAdapter) *HomeService {
	s.dataAdapter = a
	return s
}

// WithPlayerMatchesRepo (P4.1, ADR 0011) injecte le loader canonical-aware
// + titleSlug + gamertag. Quand les 3 sont fournis, fetchMatchesAndSessions
// charge canonical et convertit. Sinon fallback repo legacy.
func (s *HomeService) WithPlayerMatchesRepo(repo port.PlayerMatchesRepository, titleSlug, gamertag string) *HomeService {
	s.playerMatchesRepo = repo
	s.titleSlug = titleSlug
	s.gamertag = gamertag
	return s
}

// WithMatchesCache active le cache TTL process-level pour LoadHomeMatches + LoadHomeSessions.
// xuid est la clÃ© de cache ; sans lui le cache ne peut pas fonctionner.
func (s *HomeService) WithMatchesCache(cache *HomeMatchesCache, xuid string) *HomeService {
	s.matchesCache = cache
	s.xuid = xuid
	return s
}

// WithCareerLive câble le service live carrière (XP/rang + Spartan ID
// découplés du post-sync matchs). Quand fourni, GetHomePage remplace
// l'appel repo.LoadSpartanIdentity (DB-only) par un flow live throttle
// 5 min / 6 h + fallback DB per-field garanti.
// homeSpartanIdentityProvider : contrat minimal consommé par HomeService pour
// l'identité Spartan live (implémenté par *CareerLiveService). Interface
// consumer-side (K1i, ARCHI 8) — le champ n'est plus typé sur le concret, mockable
// en test (même package).
type homeSpartanIdentityProvider interface {
	GetSpartanIdentity(ctx context.Context) (*domain.HomeSpartanIdentityRow, error)
}

func (s *HomeService) WithCareerLive(svc *CareerLiveService) *HomeService {
	if svc != nil { // garde concret fiable — évite le piège interface typed-nil
		s.careerLive = svc
	}
	return s
}

// SetSessionActive implÃ©mente port.SessionNotifier.
// ConservÃ© pour compatibilitÃ© avec le watcher â€” aucun effet sur le handler HTTP
// qui appelle toujours le live directement.
func (s *HomeService) SetSessionActive(_ bool) {
}

// homePageData regroupe toutes les donnÃ©es brutes chargÃ©es en parallÃ¨le par fetchHomePageData.
//
// P4.3b (ADR 0011) : `canonicalRows` est renseignÃ© quand le path canonical est
// actif (playerMatchesRepo + titleSlug + gamertag). `matches`/`sessions`
// restent renseignÃ©s pour la rÃ©trocompatibilitÃ© (legacy fallback path).
type homePageData struct {
	canonicalRows  []canonical.PlayerMatchRow // nil = legacy path
	spartanIdent   *domain.HomeSpartanIdentityRow
	totalMatches   int
	media          []domain.HomeMediaRow
	playlistRanks  []domain.HomePlaylistRank
	favoriteIDs    map[string]bool
	favWeaponName  string
	favWeaponKills int
}

// fetchMatchesAndSessions charge les rows canonical du joueur (P4.3 finale).
// Retourne aussi un cache du `bool fromCache` pour tÃ©lÃ©mÃ©trie.
//
// P4.3 finale (ADR 0011) : path canonical exclusif. playerMatchesRepo +
// titleSlug + gamertag sont REQUIS (wirÃ©s en DI universellement). Le legacy
// fallback (LoadHomeMatches + LoadHomeSessions parallel) a Ã©tÃ© supprimÃ©.
//
// Le cache TTL stocke encore les rows canonical pour rÃ©trocompat avec les
// signatures Get/Set existantes (qui prennent matches/sessions legacy) â€” la
// suppression du cache legacy est tracker dans une follow-up dÃ©diÃ©e.
// finale, cf. commentaire ci-dessous) ; conservé pour exposition future de métrique
// + retour du cache hit quand la couche canonical-aware sera ajoutée (P4.4).
//
//nolint:unparam // fromCache aujourd'hui toujours false (cache bypassé en P4.3
func (s *HomeService) fetchMatchesAndSessions(ctx context.Context) (
	canonicalRows []canonical.PlayerMatchRow, fromCache bool, err error,
) {
	if s.matchesCache != nil && s.xuid != "" {
		if _, _, hit := s.matchesCache.Get(s.xuid); hit {
			// Cache hit : on doit reconstruire les canonical rows. Le cache n'est
			// pas encore canonical-aware ; pour P4.3 finale on bypass le cache hit
			// et recharge canonical. TODO P4.4 : adapter HomeMatchesCache Ã
			// canonical.
			slog.DebugContext(ctx, "home_cache: hit (bypass P4.3 finale)", "xuid", s.xuid)
		}
	}

	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		return nil, false, fmt.Errorf("HomeService: PlayerMatchesRepo non cÃ¢blÃ© (P4.3 finale exige le wiring DI)")
	}
	rows, e := s.playerMatchesRepo.LoadPlayerMatches(
		ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
	)
	if e != nil {
		return nil, false, e
	}
	canonicalRows = rows
	slog.DebugContext(ctx, "home: loaded canonical",
		"rows", len(canonicalRows), "title_slug", s.titleSlug)

	// Maintien du cache pour la mÃ©trique (set vide pour invalider stale).
	if s.matchesCache != nil && s.xuid != "" {
		s.matchesCache.Set(s.xuid, nil, nil)
	}
	return canonicalRows, false, nil
}

// fetchHomePageData charge toutes les donnÃ©es de la page d'accueil en parallÃ¨le.
// Les erreurs non-critiques sont absorbÃ©es (dÃ©gradation silencieuse).
func (s *HomeService) fetchHomePageData(ctx context.Context, locale string) (homePageData, error) {
	var d homePageData

	// Groupe 1 : matches+sessions (cache TTL) en parallÃ¨le avec les autres appels lÃ©gers.
	var cacheHit bool
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		d.canonicalRows, cacheHit, err = s.fetchMatchesAndSessions(gctx)
		return err
	})
	g.Go(func() error {
		// Path live (CareerLiveService câblé) : flow découplé du post-sync
		// matchs avec throttle 5 min (XP) / 6 h (customisation) + fallback DB
		// per-field garanti. Fallback : repo.LoadSpartanIdentity (DB-only)
		// pour les bootstraps sans careerLive (tests, instances minimales).
		if s.careerLive != nil {
			row, err := s.careerLive.GetSpartanIdentity(gctx)
			if err != nil {
				slog.WarnContext(gctx, "home: CareerLive.GetSpartanIdentity failed", "err", err)
			}
			d.spartanIdent = row
			return nil
		}
		var err error
		d.spartanIdent, err = s.repo.LoadSpartanIdentity(gctx)
		if err != nil {
			slog.WarnContext(gctx, "home: LoadSpartanIdentity failed", "err", err)
		}
		return nil
	})
	g.Go(func() error {
		// Fallback sur len(matches) aprÃ¨s le Wait si la query Ã©choue (totalMatches reste 0).
		d.totalMatches, _ = s.repo.CountPlayerMatches(gctx)
		return nil
	})
	g.Go(func() error {
		var err error
		d.media, err = s.repo.LoadRecentMedia(gctx, 4)
		if err != nil {
			slog.WarnContext(gctx, "home: LoadRecentMedia failed", "err", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		d.playlistRanks, err = s.repo.LoadRecentPlaylistRanks(gctx, locale)
		if err != nil {
			slog.WarnContext(gctx, "home: LoadRecentPlaylistRanks failed", "err", err)
		}
		return nil
	})
	g.Go(func() error {
		if wName, wKills, err := s.repo.LoadFavoriteWeapon(gctx, locale); err == nil && wName != "" {
			d.favWeaponName = wName
			d.favWeaponKills = wKills
		}
		return nil
	})
	if s.socialRepo != nil && s.playerSlug != "" {
		slug := s.playerSlug
		g.Go(func() error {
			if ids, err := s.socialRepo.GetFavoriteMatchIDs(gctx, slug); err == nil {
				d.favoriteIDs = ids
			} else {
				slog.WarnContext(gctx, "home: GetFavoriteMatchIDs failed", "err", err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return homePageData{}, err
	}
	if d.totalMatches == 0 {
		d.totalMatches = len(d.canonicalRows)
	}
	_ = cacheHit // exploitable pour des mÃ©triques futures
	return d, nil
}

// GetHomePage retourne la page d'accueil agrÃ©gÃ©e (hero card, highlights, matchs rÃ©cents,
// mÃ©dias rÃ©cents, rÃ©sumÃ©s de sessions solo et escouade).
//
// P4.3 finale (ADR 0011) : path canonical exclusif. Toutes les analyses
// passent par les `analysis.*FromCanonical`. Le legacy fallback a Ã©tÃ© supprimÃ©.
func (s *HomeService) GetHomePage(ctx context.Context, gamertag, locale string) (*domain.HomePageResponse, error) {
	defer func(start time.Time) {
		observability.RecordDurationMS("home_get_page", time.Since(start).Milliseconds())
	}(time.Now())
	d, err := s.fetchHomePageData(ctx, locale)
	if err != nil {
		return nil, err
	}

	// Bug #2/#7 cascade : remplit Labels["fr"] des AssetReference depuis
	// metadata.asset_translations + mode_name_tr quand match_registry
	// .{...}_name_fr est NULL. Sans ça, modes/maps/playlists restent en EN.
	if err := s.repo.EnrichCanonicalAssetTranslations(ctx, d.canonicalRows); err != nil {
		slog.WarnContext(ctx, "home: EnrichCanonicalAssetTranslations failed", "err", err)
	}

	// Baseline modèle de dégâts du titre courant (PV pour tuer) — injectée dans
	// les calculs de rendement/résistance. 225 Infinite (byte-identique), per-titre sinon.
	hp := games.EffectiveHpToKill(ctxkeys.TitleSlug(ctx))
	hasRankedHistory, hasUnrankedHistory := analysis.InferHomeSkillHistoryFromCanonical(d.canonicalRows)
	hero := analysis.BuildHeroCardFromCanonical(d.canonicalRows, gamertag, d.totalMatches, locale, hp)
	highlights := analysis.BuildHighlightsFromCanonical(d.canonicalRows)
	recentMatches := analysis.BuildRecentMatchesWithFavoritesFromCanonical(d.canonicalRows, len(d.canonicalRows), d.favoriteIDs, locale, hp, s.skillBadgeResolver)
	favoriteMatches := buildFavoriteMatchListCanonical(d.canonicalRows, d.favoriteIDs, locale, hp, s.skillBadgeResolver)
	soloSession := analysis.BuildSessionSummaryFromCanonical(d.canonicalRows, false, locale, hp)
	squadSession := analysis.BuildSessionSummaryFromCanonical(d.canonicalRows, true, locale, hp)
	soloSessions := analysis.BuildSessionSummariesFromCanonical(d.canonicalRows, false, 20, locale, hp)
	squadSessions := analysis.BuildSessionSummariesFromCanonical(d.canonicalRows, true, 20, locale, hp)
	// Coéquipiers des sessions escouade (best-effort) : alimente le deep-link
	// card escouade → /squad (pré-sélection de la composition). No-op si non câblé.
	s.enrichSquadSessionsTeammates(ctx, squadSessions, d.canonicalRows)

	if d.favWeaponName != "" {
		hero.KPIs.FavoriteWeaponName = d.favWeaponName
		hero.KPIs.FavoriteWeaponKills = d.favWeaponKills
	}

	// Enrichissement mÃ©dailles + citations par liste en parallÃ¨le.
	//
	// TITLE-AGNOSTIC (P7) : citations dérivées D'ABORD, puis commendations NATIVES en
	// fallback sur le MÊME slot TopCitations quand il reste vide. Halo Infinite remplit
	// via les citations dérivées (LoadMatchCitations) ; Halo 5 — sans moteur de
	// citations (citations.engine = not_exposed) mais avec commendations natives
	// (commendations.native = supported) — voit ses commendations alimenter le slot.
	// Aucun changement frontend/OpenAPI (réutilisation de MatchCitationSnippet).
	enrichG, _ := errgroup.WithContext(ctx)
	enrichG.Go(func() error {
		enrichMatchesWithMedals(ctx, s.repo, recentMatches)
		enrichMatchesWithCitations(ctx, s.repo, recentMatches)
		enrichMatchesWithCommendations(ctx, s.repo, recentMatches)
		return nil
	})
	enrichG.Go(func() error {
		enrichMatchesWithMedals(ctx, s.repo, favoriteMatches)
		enrichMatchesWithCitations(ctx, s.repo, favoriteMatches)
		enrichMatchesWithCommendations(ctx, s.repo, favoriteMatches)
		return nil
	})
	_ = enrichG.Wait()

	var rankCatalog *mappings.RankCatalog
	if s.semantic != nil {
		rankCatalog = s.semantic.Ranks()
	}

	// Garantit slices non-nil sur les champs JSON sans omitempty : un slice nil
	// sérialise en JSON `null` et crashe le front. Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
	recentMedia := analysis.BuildRecentMedia(d.media, 4)
	if highlights == nil {
		highlights = []domain.HighlightItem{}
	}
	if recentMatches == nil {
		recentMatches = []domain.RecentMatchItem{}
	}
	if favoriteMatches == nil {
		favoriteMatches = []domain.RecentMatchItem{}
	}
	if recentMedia == nil {
		recentMedia = []domain.RecentMediaItem{}
	}

	return &domain.HomePageResponse{
		Hero:                hero,
		SpartanIdentity:     analysis.BuildSpartanIdentity(d.spartanIdent, locale, rankCatalog),
		Highlights:          highlights,
		RecentMatches:       recentMatches,
		FavoriteMatches:     favoriteMatches,
		RecentMedia:         recentMedia,
		SoloSession:         soloSession,
		SquadSession:        squadSession,
		SoloSessions:        soloSessions,
		SquadSessions:       squadSessions,
		HasRankedHistory:    hasRankedHistory,
		HasUnrankedHistory:  hasUnrankedHistory,
		RecentPlaylistRanks: d.playlistRanks,
	}, nil
}

// buildFavoriteMatchListCanonical est la variante canonical-aware de
// buildFavoriteMatchList. DÃ©lÃ¨gue Ã  la version legacy via le wrapper analysis.
