// Package service — ExplorerService : matchs communs, recherche croisée.
//
// Port Go de apps/api/app/routers/explorer.py.
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/port"
)

// explorerTargetLiveBudget plafonne la durée du SEUL fetch live de l'encart
// "Profil joueur cible" : les stats carrière remote (servicerecord). Au-delà,
// le contexte est annulé : la carrière reste nil et la réponse part avec ce qui
// est disponible (identité locale + sample stats locales). Même philosophie que
// CareerLiveBudget côté home — la page ne doit jamais bloquer 10s+ sur un fetch
// Halo lent. L'identité (local-only) et le sample (calcul local) ne sont PAS
// bornés par ce budget.
//
// Relevé à 8s (depuis 3s) car l'encart inclut désormais le breakdown par saison
// (parité SpartanRecord) : ~2-3 appels live par saison jouée (service record
// classé/non-classé + pic CSR), bornés en concurrence + cachés. Reste < 10s ;
// résultat partiel si dépassement (best-effort). Lookup délibéré d'un joueur.
const explorerTargetLiveBudget = 8 * time.Second

// ExplorerLocalIdentityResolver résout l'identité Spartan d'une cible STRICTEMENT
// depuis les données locales : si la cible est un joueur suivi (db_profiles), on
// lit son identité depuis SA propre player DB ; sinon nil. Aucun fetch live Halo
// (l'identité d'un adversaire n'est de toute façon jamais publiée).
//
// Interface locale (et non port.go) car le contrat est spécifique au flow
// Explorer. Implémentée en production par une closure câblée dans le registry
// (resolveByGT + HomeRepo.LoadSpartanIdentity).
type ExplorerLocalIdentityResolver interface {
	LocalSpartanIdentity(ctx context.Context, targetGamertag string) *domain.HomeSpartanIdentityRow
}

// LocalIdentityResolverFunc adapte une closure en ExplorerLocalIdentityResolver
// (le registry câble la résolution resolveByGT + HomeRepo.LoadSpartanIdentity).
type LocalIdentityResolverFunc func(ctx context.Context, targetGamertag string) *domain.HomeSpartanIdentityRow

// LocalSpartanIdentity implémente ExplorerLocalIdentityResolver.
func (f LocalIdentityResolverFunc) LocalSpartanIdentity(ctx context.Context, targetGamertag string) *domain.HomeSpartanIdentityRow {
	return f(ctx, targetGamertag)
}

// ExplorerTargetIdentityProvider résout l'identité Spartan d'un xuid arbitraire
// (career rank + emblem + backdrop + service-tag) par un fetch live SYNCHRONE
// sans persistance, implémenté par *CareerLiveService (FetchLiveIdentity). Sert
// l'identité même pour une cible non suivie (appearance via la vue publique).
type ExplorerTargetIdentityProvider interface {
	FetchLiveIdentity(ctx context.Context, xuid string) (*domain.HomeSpartanIdentityRow, error)
}

// ExplorerTargetCSRProvider fetch les CSR par playlist (saison) d'un xuid
// arbitraire. Implémenté par une closure câblée dans le registry (sync client
// GetPlayerCSRs + mapping → domain.CareerPlaylistCSR). Contrat : retourne
// (nil, nil) si les tokens auth sont absents/insuffisants (dégradation
// gracieuse, pas d'erreur).
type ExplorerTargetCSRProvider interface {
	SeasonCSRs(ctx context.Context, xuid, seasonID string) ([]domain.CareerPlaylistCSR, error)
}

// CSRProviderFunc adapte une closure en ExplorerTargetCSRProvider.
type CSRProviderFunc func(ctx context.Context, xuid, seasonID string) ([]domain.CareerPlaylistCSR, error)

// SeasonCSRs implémente ExplorerTargetCSRProvider.
func (f CSRProviderFunc) SeasonCSRs(ctx context.Context, xuid, seasonID string) ([]domain.CareerPlaylistCSR, error) {
	return f(ctx, xuid, seasonID)
}

// SeasonCSRPeak : pic de rang CSR d'un joueur sur UNE saison passée (plus haut
// tier parmi les playlists ranked engagées) + URL du badge dérivée (tier/subTier).
type SeasonCSRPeak struct {
	Tier     string
	SubTier  int
	BadgeURL *string
}

// ExplorerSeasonCSRProvider retourne le pic de rang CSR d'un xuid sur une saison
// PASSÉE donnée (csrSeasonID, format "CsrSeasonN-1"), en n'interrogeant QUE les
// playlists ranked que le joueur a réellement engagées (engagedPlaylistIDs =
// Subqueries.PlaylistAssetIds, intersecté côté impl avec les playlists ranked).
// Évite ~4 appels futiles/saison pour un joueur peu/non classé. (nil, nil) si
// aucune donnée CSR / engagedPlaylistIDs vide (dégradation gracieuse).
type ExplorerSeasonCSRProvider interface {
	SeasonPeakCSR(ctx context.Context, xuid, csrSeasonID string, engagedPlaylistIDs []string) (*SeasonCSRPeak, error)
}

// SeasonCSRPeakFunc adapte une closure en ExplorerSeasonCSRProvider.
type SeasonCSRPeakFunc func(ctx context.Context, xuid, csrSeasonID string, engagedPlaylistIDs []string) (*SeasonCSRPeak, error)

// SeasonPeakCSR implémente ExplorerSeasonCSRProvider.
func (f SeasonCSRPeakFunc) SeasonPeakCSR(ctx context.Context, xuid, csrSeasonID string, engagedPlaylistIDs []string) (*SeasonCSRPeak, error) {
	return f(ctx, xuid, csrSeasonID, engagedPlaylistIDs)
}

// ExplorerService orchestre les requêtes de l'Explorer.
type ExplorerService struct {
	repo port.ExplorerRepository
	xuid string
	// dataAdapter (optionnel, Phase 2 plan finition multi-titres) :
	// quand fourni, le service mesure la capability match.history pour
	// loguer une éventuelle dégradation. La bascule fonctionnelle reste
	// future car canonical.MatchSummary ne couvre pas encore le filtrage
	// Explorer (joueur commun + plage temporelle).
	dataAdapter games.TitleDataAdapter

	// Dépendances optionnelles pour l'encart "Profil joueur cible" (cf.
	// TargetProfileDeps). Une dépendance nil laisse la sous-section à nil ;
	// sample_stats (local) reste toujours produit. Aucune privacy n'est fetchée.
	deps ExplorerTargetProfileDeps

	// liveResolver (optionnel) : résout gamertag→xuid via l'API Halo/Xbox quand la
	// résolution locale (v_gamertag_lookup) échoue pour un joueur JAMAIS croisé. nil
	// (démo/offline) → comportement d'origine (erreur dure sur joueur inconnu).
	liveResolver GamertagXUIDResolver

	// weaponKillsRepo (frag v2) : loader weapon_kills agrégé alimentant la « Répartition
	// des frags » v2 (sunburst classe→rôle) + « Outils de destruction » de l'encart cible.
	// Optionnel — nil → FragDistribution nil (le front retombe sur le donut kill-type
	// legacy). Le repo concret DuckDB fournit AUSSI les mécaniques natives H5 via
	// type-assertion (explorerKillMechanicsLoader), capability OPTIONNELLE façon lobbySizeProvider.
	weaponKillsRepo port.WeaponKillsRepository
}

// ExplorerRelationsProvider fournit les agrégats relationnels joueur↔cible
// RÉUTILISÉS par l'encart « matchs joués ensemble » : le taux de victoire
// historique perso (repère des donuts) et la timeline de duels (écart de frags
// cumulé). Satisfait tel quel par duckdb.CareerRepo (port.RelationsRepository) —
// on n'expose ici que les 2 méthodes nécessaires pour ne pas sur-coupler l'Explorer.
type ExplorerRelationsProvider interface {
	GetCoreEngagement(ctx context.Context, coreXUIDs []string, scope []string, limit int) (domain.CoreEngagement, error)
	GetRivalTimeline(ctx context.Context, rivalXUID string, scope []string, limit int) ([]domain.RelationDuelRawRow, error)
}

// ExplorerTargetProfileDeps regroupe les dépendances de l'encart "Profil joueur
// cible". Tout est optionnel (nil → sous-section masquée).
type ExplorerTargetProfileDeps struct {
	// LocalIdentity : identité Spartan lue depuis la DB locale de la cible (si
	// joueur suivi). Source prioritaire.
	LocalIdentity ExplorerLocalIdentityResolver
	// LiveIdentity : identité Spartan live (career rank/emblem/backdrop) pour un
	// xuid arbitraire — secours quand la cible n'est pas locale.
	LiveIdentity ExplorerTargetIdentityProvider
	// RemoteStats : service record agrégé (stats + temps de jeu + médailles).
	RemoteStats port.ServiceRecordProvider
	// MedalDefs : métadonnées médailles (label/description) pour le top médailles.
	MedalDefs port.MedalDefinitionsRepository
	// CSR : classements CSR saison courante de la cible (live).
	CSR ExplorerTargetCSRProvider
	// CurrentSeasonID : saison CSR courante (pour le fetch CSR).
	CurrentSeasonID string
	// Seasons : calendrier des saisons (plages temporelles) pour le bucketing
	// "matchs par saison" — résolu côté registry depuis SeasonsCatalog.
	Seasons []SeasonCatalogEntry
	// SeasonSR : service record FILTRÉ par saison (matchs classés/non-classés par
	// saison) pour le breakdown live. nil → fallback bucketing local.
	SeasonSR port.SeasonStatsProvider
	// SeasonCSR : pic de rang CSR par saison passée (badge au-dessus des barres).
	// nil → pas de badge.
	SeasonCSR ExplorerSeasonCSRProvider
	// Ranks : catalogue des rangs de carrière du titre (issu du SemanticAdapter)
	// pour localiser RankTitle/NextRankTitle dans le DTO d'identité. nil → titres
	// résolus via le fallback RankName (player DB) puis "Rang N" (cf.
	// analysis.BuildSpartanIdentity, nil-safe).
	Ranks *mappings.RankCatalog
	// RecentMatches : fetch live read-only des N derniers matchs d'une cible NON
	// suivie (cache TTL 20 min, aucune persistance). Utilisé en repli quand la
	// lecture locale des matchs récents est vide (cible non-locale) pour peupler
	// les graphes profil de combat. nil → pas de repli live (graphes vides pour
	// les non-locaux, comportement historique).
	RecentMatches port.RecentMatchesProvider
	// Relations : agrégats relationnels du joueur principal (WR historique perso +
	// timeline de duels) réutilisés par les donuts + l'écart de frags cumulé de la
	// section « matchs joués ensemble ». nil → donuts sans repère + graphe masqué.
	Relations ExplorerRelationsProvider
	// LocalBannerPool : pool de bannières (nameplates) connues localement —
	// banner_image_url des joueurs suivis. Résolu PARESSEUSEMENT : appelé
	// uniquement quand la cible n'a ni bannière ni backdrop (cas non-local sans
	// customisation publique résolue). Sert à donner une nameplate de repli
	// déterministe par xuid. nil → pas de repli aléatoire (on garde le vide).
	LocalBannerPool func(ctx context.Context) []string
	TitleSlug       string
}

// NewExplorerService crée un ExplorerService.
func NewExplorerService(repo port.ExplorerRepository, xuid string) *ExplorerService {
	return &ExplorerService{repo: repo, xuid: xuid}
}

// WithDataAdapter injecte un games.TitleDataAdapter optionnel (Phase 2 plan
// finition multi-titres). Permet de logger le statut des capabilities et
// d'amorcer la bascule vers la couche canonique.
func (s *ExplorerService) WithDataAdapter(a games.TitleDataAdapter) *ExplorerService {
	s.dataAdapter = a
	return s
}

// WithTargetProfileProviders injecte les dépendances de l'encart "Profil joueur
// cible". Retourne le service pour chainer.
func (s *ExplorerService) WithTargetProfileProviders(deps ExplorerTargetProfileDeps) *ExplorerService {
	s.deps = deps
	return s
}

// WithLiveGamertagResolver injecte le résolveur live gamertag→xuid (fallback pour
// un joueur jamais croisé). nil → no-op. Retourne le service pour chainer.
func (s *ExplorerService) WithLiveGamertagResolver(r GamertagXUIDResolver) *ExplorerService {
	s.liveResolver = r
	return s
}

// WithWeaponKillsRepo injecte le loader weapon_kills alimentant la « Répartition des
// frags » v2 (sunburst classe→rôle + « Outils de destruction ») de l'encart cible —
// MIROIR de Synthesis/Sessions. Optionnel (nil → repli donut kill-type côté front). Le
// repo concret DuckDB fournit aussi les mécaniques natives H5 (LoadKillMechanicsAggregated)
// via type-assertion. Retourne le service pour chainer.
func (s *ExplorerService) WithWeaponKillsRepo(repo port.WeaponKillsRepository) *ExplorerService {
	s.weaponKillsRepo = repo
	return s
}

// logCapabilityIfMissing log un warning si la capability est absente du
// DataAdapter injecté. No-op si pas de DataAdapter.
func (s *ExplorerService) logCapabilityIfMissing(ctx context.Context, cap games.CapabilityKey, caller string) {
	if s.dataAdapter == nil {
		return
	}
	if !s.dataAdapter.Capabilities().Has(cap) {
		slog.WarnContext(ctx, "capability_not_supported",
			"title_slug", s.dataAdapter.TitleSlug(),
			"capability", string(cap),
			"caller", caller,
		)
	}
}

// GetCommonMatches retourne l'historique paginé de matchs communs avec un autre
// joueur, enrichi des badges encounter (ally_plus, tough_enemy, ordinal).
// page est 1-indexé ; PageSizeCommonMatches = 20 éléments par page.
// loadPlayerIntersection centralise la résolution repo/adapter de l'intersection
// 2-joueurs (HIGH-B). Adapter-first via LoadPlayerIntersection ; fallback sur les 2
// appels repo avec la gestion d'erreur EXACTE de l'historique (matchs communs =
// fatal, kills croisés = dégradation gracieuse).
func (s *ExplorerService) loadPlayerIntersection(ctx context.Context, otherXUID string) ([]domain.CommonMatchRaw, domain.KillerVictimAggregate, error) {
	if s.dataAdapter != nil {
		inter, err := s.dataAdapter.LoadPlayerIntersection(ctx, s.xuid, otherXUID)
		if err == nil {
			return commonMatchesFromCanonical(inter.Matches), killerVictimFromCanonical(inter.CrossKills), nil
		}
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil, domain.KillerVictimAggregate{}, fmt.Errorf("ExplorerService: matchs communs: %w", err)
		}
	}
	rawMatches, err := s.repo.GetCommonMatches(ctx, s.xuid, otherXUID)
	if err != nil {
		return nil, domain.KillerVictimAggregate{}, fmt.Errorf("ExplorerService: matchs communs: %w", err)
	}
	kv, kvErr := s.repo.GetKillerVictimBetween(ctx, s.xuid, otherXUID)
	if kvErr != nil {
		// Dégradation gracieuse : les badges tough_enemy ne seront pas calculés,
		// mais le reste de la réponse reste valide.
		slog.WarnContext(ctx, "explorer_kv_between_failed",
			"xuid1", s.xuid, "xuid2", otherXUID, "err", kvErr)
		kv = domain.KillerVictimAggregate{}
	}
	return rawMatches, kv, nil
}

// commonMatchesFromCanonical projette []canonical.CommonMatchRow → []domain.CommonMatchRaw
// (Self/Other → Player1/Player2 ; team IDs deep-copiés ; nil pour vide).
func commonMatchesFromCanonical(cs []canonical.CommonMatchRow) []domain.CommonMatchRaw {
	if len(cs) == 0 {
		return nil
	}
	out := make([]domain.CommonMatchRaw, 0, len(cs))
	for _, c := range cs {
		r := domain.CommonMatchRaw{
			MatchID:        c.MatchID,
			StartTime:      c.StartTime,
			MapUI:          c.MapUI,
			ModeUI:         c.ModeUI,
			Player1Outcome: c.SelfOutcomeCode,
			Player1Kills:   c.SelfKills,
			Player1Deaths:  c.SelfDeaths,
			Player1KDA:     c.SelfKDA,
		}
		if c.SelfTeamID != nil {
			v := *c.SelfTeamID
			r.Player1TeamID = &v
		}
		if c.OtherTeamID != nil {
			v := *c.OtherTeamID
			r.Player2TeamID = &v
		}
		out = append(out, r)
	}
	return out
}

// killerVictimFromCanonical projette canonical.CrossKillTally → domain.KillerVictimAggregate.
func killerVictimFromCanonical(c canonical.CrossKillTally) domain.KillerVictimAggregate {
	return domain.KillerVictimAggregate{KillsDealt: c.KillsBySelf, DeathsSuffered: c.KillsByOther}
}

func (s *ExplorerService) GetCommonMatches(
	ctx context.Context,
	otherGamertag string,
	otherXUID string,
	page int,
) (domain.ExplorerPlayerQueryResponse, error) {
	s.logCapabilityIfMissing(ctx, games.CapMatchHistory, "explorer_service.GetCommonMatches")

	// xuid déjà connu (ex. ligne du Classement) → on saute la résolution
	// gamertag→xuid locale, qui échouerait pour un joueur absent des données
	// locales. Le profil live (buildTargetProfile) reste servi pour tout xuid ;
	// l'intersection « matchs communs » sera simplement vide pour un inconnu.
	if otherXUID == "" {
		resolved, err := s.repo.ResolveXUIDByGamertag(ctx, otherGamertag)
		switch {
		case err == nil:
			otherXUID = resolved
		case errors.Is(err, sql.ErrNoRows) && s.liveResolver != nil:
			// Joueur JAMAIS croisé (absent de v_gamertag_lookup) : fallback live
			// (PeopleHub→profil Xbox). Sur succès, le profil cible PUBLIC est servi
			// (buildTargetProfile fonctionne pour tout xuid) ; l'intersection « matchs
			// communs » est simplement vide.
			live, lerr := s.liveResolver.ResolveXUID(ctx, otherGamertag)
			if lerr != nil {
				return domain.ExplorerPlayerQueryResponse{},
					fmt.Errorf("ExplorerService: gamertag %q introuvable (local + live): %w", otherGamertag, lerr)
			}
			otherXUID = live
			slog.InfoContext(ctx, "explorer_gamertag_resolved_live", "gamertag", otherGamertag, "xuid", live)
		default:
			return domain.ExplorerPlayerQueryResponse{},
				fmt.Errorf("ExplorerService: résolution gamertag %q: %w", otherGamertag, err)
		}
	}

	rawMatches, kv, err := s.loadPlayerIntersection(ctx, otherXUID)
	if err != nil {
		return domain.ExplorerPlayerQueryResponse{}, err
	}

	totalCount := len(rawMatches)
	if page < 1 {
		page = 1
	}
	pageSize := domain.PageSizeCommonMatches
	offset := (page - 1) * pageSize
	end := offset + pageSize
	if end > totalCount {
		end = totalCount
	}

	var pageMatches []domain.CommonMatchRaw
	if offset < totalCount {
		pageMatches = rawMatches[offset:end]
	}

	rows := convertCommonMatches(pageMatches)

	stats := buildEncounterStats(otherXUID, otherGamertag, rawMatches, kv)
	badges := narrative.ComputeEncounterBadges(stats, totalCount)
	wins, losses := countWinsLosses(rawMatches)
	encounterStats := convertEncounterStatsToExplorer(stats, totalCount)
	// Enrichissement best-effort de la section « matchs joués ensemble » : repère
	// « moyenne perso » des donuts + courbe d'écart de frags cumulé (duels). Réutilise
	// les requêtes du hub Relations (WR historique + timeline de duels), déjà testées.
	s.enrichEncounterRelations(ctx, encounterStats, otherXUID)
	activityHeatmap := analysis.ComputeActivityHeatmapFromCommonMatches(rawMatches)

	// Encart "Profil joueur cible" : 4 sources fetch en parallèle (best-effort).
	targetProfile := s.buildTargetProfile(ctx, otherXUID, otherGamertag, rawMatches)

	slog.DebugContext(ctx, "explorer_common_matches",
		"xuid", s.xuid, "other_xuid", otherXUID,
		"total", totalCount, "page", page, "badges", len(badges),
		"heatmap_cells", len(activityHeatmap),
		"target_profile_built", targetProfile != nil,
		"auth_available", targetProfile != nil && targetProfile.AuthAvailable,
		"career_served", targetProfile != nil && targetProfile.CareerStats != nil)

	return domain.ExplorerPlayerQueryResponse{
		TargetGamertag:  otherGamertag,
		TargetXUID:      otherXUID,
		CommonMatches:   rows,
		Badges:          convertEncounterBadges(badges),
		EncounterStats:  encounterStats,
		Total:           len(rows),
		TotalCount:      totalCount,
		WinsTogether:    wins,
		LossesTogether:  losses,
		Page:            page,
		PageSize:        pageSize,
		ActivityHeatmap: activityHeatmap,
		TargetProfile:   targetProfile,
	}, nil
}

// buildTargetProfile orchestre les sources de l'encart "Profil joueur cible" :
//   - identité Spartan : DB locale (si joueur suivi) sinon live (xuid arbitraire,
//     career rank/emblem/backdrop) — bannière fallback sur le backdrop
//   - carrière + médailles : service record live (stats + temps de jeu + top
//     médailles lifetime) — un seul appel, borné + caché
//   - CSR saison : classements ranked live de la saison courante (tout xuid)
//   - sample stats : agrégat local sur les matchs communs
//
// Aucune privacy n'est fetchée. Tout est best-effort : une source qui échoue
// rend nil sans bloquer les autres. Les fetchs live sont bornés par
// explorerTargetLiveBudget ; les sources locales (sample) ne le sont pas.
func (s *ExplorerService) buildTargetProfile(
	ctx context.Context,
	targetXUID, targetGamertag string,
	rawMatches []domain.CommonMatchRaw,
) *domain.ExplorerTargetProfile {
	tokens := ctxkeys.HaloTokens(ctx)
	hasAuth := tokens != nil && tokens.SpartanToken != ""

	var (
		identityRaw        *domain.HomeSpartanIdentityRow
		careerStats        *domain.NormalizedPlayerStats
		topMedals          []domain.MedalDigestItem
		seasonCSRs         []domain.CareerPlaylistCSR
		matchsPerSea       []domain.SeasonMatchCount
		sampleStats        *domain.ExplorerTargetSampleStats
		combatProfileLive  []domain.ExplorerTargetRecentMatch
		combatProfileLocal []domain.ExplorerTargetRecentMatch

		// Statuts par section (A3 — fin de la dégradation muette). Chaque goroutine
		// statue SA section ; jamais de valeur vide en sortie de buildTargetProfile.
		identityStatus   domain.ExplorerLiveSectionStatus
		careerStatus     domain.ExplorerLiveSectionStatus
		seasonCSRsStatus domain.ExplorerLiveSectionStatus
		seasonsStatus    domain.ExplorerLiveSectionStatus
		combatStatus     domain.ExplorerLiveSectionStatus
	)
	g, gctx := errgroup.WithContext(ctx)

	// Budget de latence sur les fetchs live (identité live / carrière / CSR).
	// Les calculs locaux (sample, matchs par saison) restent sur gctx, NON bornés.
	liveCtx, cancelLive := context.WithTimeout(gctx, explorerTargetLiveBudget)
	defer cancelLive()

	g.Go(func() error {
		identityRaw, identityStatus = s.fetchTargetIdentityRaw(liveCtx, targetXUID, targetGamertag, hasAuth)
		return nil
	})
	g.Go(func() error {
		// Service record lifetime → carrière + top médailles + saisons jouées
		// (Subqueries.SeasonIds) + playlists engagées (PlaylistAssetIds), puis
		// breakdown par saison (séquentiel car dépendant). Un seul appel lifetime.
		var seasonIDs, engagedPlaylists []string
		careerStats, topMedals, seasonIDs, engagedPlaylists, careerStatus = s.fetchTargetServiceRecord(liveCtx, targetGamertag, hasAuth)
		matchsPerSea, seasonsStatus = s.computeSeasonBreakdown(liveCtx, targetXUID, targetGamertag, hasAuth, seasonIDs, engagedPlaylists)
		return nil
	})
	g.Go(func() error {
		seasonCSRs, seasonCSRsStatus = s.fetchTargetCSR(liveCtx, targetXUID, hasAuth)
		return nil
	})
	g.Go(func() error { sampleStats = s.computeTargetSampleStats(gctx, targetXUID, rawMatches); return nil })
	// Profil de combat : deux sources servies en parallèle pour alimenter le toggle.
	// LIVE (défaut) = ~20 derniers matchs via l'API (liveCtx borné) ; LOCAL = matchs
	// de la cible en base (gctx, non borné, gratuit).
	g.Go(func() error {
		combatProfileLive, combatStatus = s.computeTargetCombatProfileLive(liveCtx, targetXUID, hasAuth)
		return nil
	})
	g.Go(func() error {
		combatProfileLocal = s.computeTargetCombatProfileLocal(gctx, targetXUID)
		return nil
	})

	_ = g.Wait()

	if hasAuth && careerStats == nil && liveCtx.Err() == context.DeadlineExceeded {
		slog.WarnContext(ctx, "explorer_target_live_budget_exceeded",
			"gamertag", targetGamertag, "budget", explorerTargetLiveBudget.String())
	}

	// Sérialise l'identité en DTO snake_case (career_rank imbriqué + adornment)
	// comme la Home, au lieu de la struct brute PascalCase. C'est le cœur du fix :
	// le front lit identity.banner_image_url / career_rank.rank_title. nil-safe
	// (Ranks peut être nil → fallback RankName ; identityRaw nil → DTO nil).
	identity := analysis.BuildSpartanIdentity(identityRaw, ctxkeys.Locale(ctx), s.deps.Ranks)

	return &domain.ExplorerTargetProfile{
		Identity:           identity,
		CareerStats:        careerStats,
		TopMedals:          topMedals,
		SeasonCSRs:         seasonCSRs,
		MatchesPerSeason:   matchsPerSea,
		SampleStats:        sampleStats,
		CombatProfile:      combatProfileLive,
		CombatProfileLocal: combatProfileLocal,
		AuthAvailable:      hasAuth,
		LiveStatus: domain.ExplorerLiveStatus{
			Identity:   identityStatus,
			Career:     careerStatus,
			SeasonCSRs: seasonCSRsStatus,
			Seasons:    seasonsStatus,
			CombatLive: combatStatus,
		},
	}
}

// explorerFragGapTimelineLimit : nombre de duels (matchs en ennemi) conservés
// pour la courbe « écart de frags cumulé » de la section « matchs joués ensemble »
// — aligné sur momentsTimelineLimit du hub Relations (mêmes N derniers duels).
const explorerFragGapTimelineLimit = 20

// enrichEncounterRelations complète best-effort la section « matchs joués
// ensemble » : PlayerWinRate (repère « moyenne perso » des donuts, WR historique
// du joueur) + FragGapSeries (écart de frags cumulé duel par duel contre la cible).
// no-op si stats nil (aucun match commun) ou provider non injecté. Chaque source
// est indépendante : un échec est loggé puis ignoré (dégradation gracieuse), la
// réponse reste servie sans le repère / le graphe.
func (s *ExplorerService) enrichEncounterRelations(ctx context.Context, stats *domain.ExplorerEncounterStats, otherXUID string) {
	if stats == nil || s.deps.Relations == nil {
		return
	}
	// WR historique perso (tout-temps) — repère des donuts. GetCoreEngagement avec
	// noyau vide ne calcule QUE le WR (la forme récente court-circuite).
	if eng, err := s.deps.Relations.GetCoreEngagement(ctx, nil, nil, 0); err != nil {
		slog.WarnContext(ctx, "explorer_player_win_rate_failed", "xuid", s.xuid, "err", err)
	} else {
		stats.PlayerWinRate = eng.PlayerWinRate
	}
	// Écart de frags cumulé — timeline des duels (matchs en ennemi contre la cible).
	if otherXUID != "" {
		duels, err := s.deps.Relations.GetRivalTimeline(ctx, otherXUID, nil, explorerFragGapTimelineLimit)
		if err != nil {
			slog.WarnContext(ctx, "explorer_frag_gap_timeline_failed", "other_xuid", otherXUID, "err", err)
		} else {
			stats.FragGapSeries = buildExplorerFragGapSeries(duels)
		}
	}
}

// explorerCombatProfileLimit : nombre de matchs PvP récents de la cible exposés
// dans les graphes "profil de combat" (décision plan). Filtre PvP fait en SQL.
const explorerCombatProfileLimit = 20

// computeTargetCombatProfile charge les N derniers matchs PvP de la cible
// (calcul local DuckDB, best-effort comme computeTargetSampleStats). nil en cas
// d'erreur (logguée) ou de cible sans match PvP — la section graphes est alors
// masquée côté front.
