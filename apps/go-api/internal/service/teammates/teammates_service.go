// Package service - TeammatesService : endpoint POST /pages/teammates (contrat FastAPI).
//
// Sprint 33 : adapte les donnees SquadRepository vers le contrat TeammatesPageResponse.
// Reutilise les memes queries Q29-Q31 que SquadService mais expose le format FastAPI.
//
// Le code est decoupe en fichiers thematiques pour respecter la limite des
// 500 lignes par fichier (CLAUDE.md). Ce fichier contient le type service,
// les types/aliases, le constructor, les Withers et GetPage. Les autres
// responsabilites vivent dans :
//
//   - teammates_service_briefing.go : briefing header +
//     loadTeammatesCanonicalParallel +
//     filtres synthesis (cascade, period,
//     picked sessions, session,
//     experience labels)
//   - teammates_service_kpis.go     : teammate options + row builders + KPIs
//     squad/synthesis + safeDiv/round2 +
//     enrichMapBreakdown + squadStatsToWinTotal
//   - teammates_service_assets.go   : asset enrichment + collectUniqueIDs +
//     modeLabel + computeMapBreakdown +
//     collectModeENs + buildSquadMatchHistory +
//     buildMatchSeries
package teammates

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

// FriendGamertagsResolver retourne la liste courante des amis configurÃƒÂ©s
// (app_settings.friend_gamertags). AppelÃƒÂ© ÃƒÂ  chaque requÃƒÂªte pour reflÃƒÂ©ter les
// PATCH settings sans redÃƒÂ©marrage.
type FriendGamertagsResolver func(ctx context.Context) []string

// TeammatesService calcule les stats coÃƒÂ©quipiers au format FastAPI.
type TeammatesService struct {
	repo            port.SquadRepository
	friendGamertags FriendGamertagsResolver
	// playerMatchesRepo (P4.3 finale) : loader canonical-only. CÃƒÂ¢blÃƒÂ© en DI
	// universellement via registry.go (ServiceRegistry.playerMatchesAdapterFor).
	// IMPORTANT : cet adapteur est BOUND au gamertag du joueur principal (ignore
	// l'arg gamertag). Pour charger les canonical rows d'un coequipier different,
	// utiliser squadLoader.LoadFor (resolution dynamique par gamertag).
	playerMatchesRepo port.PlayerMatchesRepository
	titleSlug         string
	gamertag          string
	// squadLoader (optionnel) : utilise pour charger les canonical rows des
	// coequipiers (mode squad du SessionBriefing). Si nil, le briefing degrade
	// en mode solo (SoloKPIs uniquement, pas de squad verdict).
	squadLoader squadagg.SquadV2Loader
	// medalDefs (optionnel) : résout les labels/descriptions anglais des médailles
	// depuis metadata.medal_definitions. Si nil, le digest est retourné sans
	// labels (medal_id et count seulement).
	medalDefs port.MedalDefinitionsRepository
	// weaponAccuracyRepo (optionnel) : loader agrégé weapon_accuracy (précision native
	// par arme). Table SHARED par titre → un repo lié à la player DB du main charge la
	// précision de tous les xuids de l'escouade en 1 appel (filtre MatchIDs + XUIDs).
	// Nil ou capability absente (Halo Infinite) → comparaison précision omise (best-effort).
	weaponAccuracyRepo port.WeaponAccuracyRepository
	// objectiveIndex (optionnel) : agrégats objectifs par famille de mode
	// (match_objective_stats_latest, SHARED → couvre aussi les coéquipiers non
	// suivis) pour l'axe « Objectifs » par opportunité du radar synergie. Câblé
	// gated par la capability match.objective.stats ; nil → axe retiré du radar.
	objectiveIndex port.ObjectiveIndexRepository
}

// NewTeammatesService crÃƒÂ©e un TeammatesService.
//
// friendGamertags : optionnel. Si nil, le filtre amis-only est dÃƒÂ©sactivÃƒÂ©
// (top retournÃƒÂ© brut, ancien comportement). Quand fourni, le top dropdown
// est restreint aux amis configurÃƒÂ©s.
func NewTeammatesService(repo port.SquadRepository, friendGamertags FriendGamertagsResolver) *TeammatesService {
	return &TeammatesService{repo: repo, friendGamertags: friendGamertags}
}

// WithPlayerMatchesRepo (P4.3 finale, ADR 0011) injecte le loader canonical-aware.
func (s *TeammatesService) WithPlayerMatchesRepo(repo port.PlayerMatchesRepository, titleSlug, gamertag string) *TeammatesService {
	s.playerMatchesRepo = repo
	s.titleSlug = titleSlug
	s.gamertag = gamertag
	return s
}

// WithSquadLoader injecte le loader per-gamertag utilise pour le SessionBriefing
// mode squad (chargement des canonical rows de chaque coequipier en parallele
// via TitlePlayerResolver). Si non cable, le briefing degrade en mode solo.
func (s *TeammatesService) WithSquadLoader(loader squadagg.SquadV2Loader) *TeammatesService {
	s.squadLoader = loader
	return s
}

// WithMedalDefs injecte le repo de définitions médailles (labels + descriptions
// anglaises depuis metadata.medal_definitions). Si non câblé, le MedalDigest
// est retourné avec medal_id + count uniquement (sans labels ni images).
func (s *TeammatesService) WithMedalDefs(repo port.MedalDefinitionsRepository) *TeammatesService {
	s.medalDefs = repo
	return s
}

// WithWeaponAccuracyRepo injecte le loader agrégé weapon_accuracy (précision native par
// arme). Miroir de WithWeaponKillsRepo côté Synthesis/Sessions. Si non câblé (ou capability
// absente sur le titre), la comparaison « Précision par arme » est omise (best-effort nil).
func (s *TeammatesService) WithWeaponAccuracyRepo(repo port.WeaponAccuracyRepository) *TeammatesService {
	s.weaponAccuracyRepo = repo
	return s
}

// WithObjectiveIndexRepo injecte le repo des agrégats objectifs par famille
// (match_objective_stats_latest) pour l'axe « Objectifs » par opportunité du
// radar synergie. Optionnel — capability-gated au wiring (match.objective.stats) ;
// nil → l'axe Objectif est retiré de TOUTES les séries du radar.
func (s *TeammatesService) WithObjectiveIndexRepo(repo port.ObjectiveIndexRepository) *TeammatesService {
	s.objectiveIndex = repo
	return s
}

// GetPage retourne la page Teammates avec options, comparaisons et solo ref.
//
// boucle gamertags → calcule timeseries/heatmap/impact. Splitter en sous-fonctions
// nécessiterait 5+ params chacune et perdrait la vue d'ensemble du flow.
//
//nolint:funlen // Orchestrateur séquentiel : charge top → filtre → applique cascade →
func (s *TeammatesService) GetPage(
	ctx context.Context,
	playerXUID string,
	req domain.TeammatesQueryRequest,
) (domain.TeammatesPageResponse, error) {
	topRows, err := s.repo.LoadTopTeammates(ctx, playerXUID)
	if err != nil {
		return domain.TeammatesPageResponse{}, fmt.Errorf("TeammatesService: %w", err)
	}

	// Ã‚Â§3 plan Squad/Sessions : filtre top dropdown aux amis configurÃƒÂ©s
	// (settings.friend_gamertags). Hors amis = exclus du dropdown mais
	// toujours requÃƒÂªtables explicitement via SelectedGamertags + alias.
	var friendGTs []string
	if s.friendGamertags != nil {
		friendGTs = s.friendGamertags(ctx)
	}
	dropdownRows := topRows
	if friendGTs != nil {
		dropdownRows = filterTopRowsToFriends(topRows, friendGTs)
	}

	// Options (liste des coÃƒÂ©quipiers frÃƒÂ©quents Ã¢â‚¬â€ limitÃƒÂ©e aux amis si configurÃƒÂ©).
	options := buildTeammateOptions(dropdownRows)

	// P4.3 finale (ADR 0011) : load canonical via PlayerMatchesRepo, convert
	// vers SynthesisMatchRow pour les helpers internes (extractSynthesisSessionLabels,
	// filterSynthesisByCascade, etc.).
	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		return domain.TeammatesPageResponse{}, fmt.Errorf("TeammatesService: PlayerMatchesRepo non cÃƒÂ¢blÃƒÂ© (P4.3 finale exige le wiring DI)")
	}
	canonicalRows, err := s.playerMatchesRepo.LoadPlayerMatches(
		ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
	)
	if err != nil {
		return domain.TeammatesPageResponse{}, fmt.Errorf("TeammatesService synthesis: %w", err)
	}
	allMatches := analysis.SynthesisMatchRowsFromCanonical(canonicalRows)

	// Extraire les session_labels disponibles (solo / escouade).
	sessionLabels := extractSynthesisSessionLabels(allMatches)

	// Filtrer les matchs selon les sessions sÃƒÂ©lectionnÃƒÂ©es.
	filteredMatches := filterSynthesisBySession(allMatches, req.PickedSoloSessions, req.PickedSquadSessions)

	// Appliquer les filtres cascade (experience_types, playlists) si prÃƒÂ©sents.
	if req.Filters != nil {
		filteredMatches = filterSynthesisByCascade(filteredMatches, req.Filters.Cascade)
		// Period (rail nav, PeriodePill) — sans cela la navigation periode n'a aucun
		// effet sur les charts/tableaux Escouade. Filtre par StartTime [start, end+1j-1s].
		filteredMatches = filterSynthesisByPeriodInput(filteredMatches, req.Filters.Period)
		// picked_sessions du filterContext (rail nav, FilterOmnibar SessionPill).
		// Vit en parallele de PickedSquadSessions/PickedSoloSessions ; intersection
		// volontaire pour le cas multi-select + nav, en pratique l'un est vide quand
		// l'autre est pose donc l'effet net est equivalent a "remplacement".
		filteredMatches = filterSynthesisByPickedSessions(filteredMatches, req.Filters.Sessions.PickedSessions)
	}

	totalMatches := len(filteredMatches)

	// Construire le set d'IDs de session si un filtre de session est actif.
	// Nil = pas de filtre = tous les matchs escouade retournés.
	var sessionMatchIDs map[string]bool
	if len(req.PickedSoloSessions) > 0 || len(req.PickedSquadSessions) > 0 {
		sessionMatchIDs = make(map[string]bool, len(filteredMatches))
		for _, m := range filteredMatches {
			sessionMatchIDs[m.MatchID] = true
		}
	}

	// Calculs détaillés pour les gamertags sélectionnés.
	teammates := make([]domain.TeammateRow, 0, len(req.SelectedGamertags))
	matchSeries := map[string][]domain.SquadMatchSeriesPoint{}

	// Sets par coéquipier : chaque set = matchs communs (main ∩ ce coéquipier).
	// L'INTERSECTION de ces sets = matchs joués par le joueur principal ET TOUS
	// les coéquipiers sélectionnés (composition exacte). Avant ce fix on faisait
	// une union (un match joué sans un coéquipier survivait), d'où le bug
	// "coéquipier ajouté à une session qu'il n'a pas jouée".
	var setsFiltered, setsAllForTimeline [][]domain.SquadMatchRow

	for _, gt := range req.SelectedGamertags {
		row, squadMatches, allSquadMatchesTm, err := s.buildTeammateRowWithMatches(ctx, playerXUID, gt, topRows, filteredMatches, sessionMatchIDs)
		if err != nil {
			slog.WarnContext(ctx, "teammates: erreur buildTeammateRow", "gamertag", gt, "err", err)
			continue // skip teammate on error
		}
		if row != nil {
			teammates = append(teammates, *row)
			setsFiltered = append(setsFiltered, squadMatches)
			setsAllForTimeline = append(setsAllForTimeline, allSquadMatchesTm)
			// matchSeries reste per-coéquipier (trajectoire de CE coéquipier vs main) :
			// volontairement non intersecté.
			matchSeries[gt] = buildMatchSeries(squadMatches)
		}
	}

	// Composition exacte : intersection sur tous les coéquipiers sélectionnés.
	allSquadRows := intersectSquadRowsByMatchID(setsFiltered)
	allSquadRowsForTimeline := intersectSquadRowsByMatchID(setsAllForTimeline)

	// Exclusivité de composition : l'intersection ci-dessus garantit que le main a
	// joué AVEC tous les sélectionnés, mais PAS qu'aucun autre coéquipier connu
	// n'était présent. On charge l'équipe alliée du main par match puis on écarte
	// les matchs où un coéquipier connu hors sélection (extraPool) figure sur cette
	// équipe. Best-effort : si le chargement échoue, on garde l'intersection brute
	// (dégradation gracieuse) et le filtre briefing est désactivé (mainTeamByMatch nil).
	selectedXUIDs := collectSelectedXUIDs(teammates)
	friendXUIDs := resolveFriendXUIDs(friendGTs, topRows)
	extraPool := buildExtraPoolXUIDs(topRows, friendXUIDs, selectedXUIDs, playerXUID)
	var mainTeamByMatch map[string]map[string]struct{}
	if len(allSquadRowsForTimeline) > 0 && len(selectedXUIDs) > 0 {
		allies, err := s.repo.LoadMainTeamParticipants(
			ctx, playerXUID, collectMatchIDs(allSquadRowsForTimeline, allSquadRows))
		if err != nil {
			slog.WarnContext(ctx, "teammates: LoadMainTeamParticipants failed (exact composition filter skipped)", "err", err)
		} else {
			mainTeamByMatch = buildMainTeamXUIDSet(allies)
			allSquadRows = filterExactComposition(allSquadRows, mainTeamByMatch, extraPool, selectedXUIDs)
			allSquadRowsForTimeline = filterExactComposition(allSquadRowsForTimeline, mainTeamByMatch, extraPool, selectedXUIDs)
		}
	}

	// Timeseries + MapBreakdown sur l'intersection des matchs escouade (composition exacte).
	var timeseries []domain.SquadTimeseriesPoint
	var mapBreakdown []domain.MapBreakdownRow
	var matchHistory []domain.SquadMatchHistoryRow
	var sessionTimeline []domain.SquadSessionPoint
	var mapHeatmap *domain.SquadMapHeatmap
	var impactMatrix *domain.SquadImpactMatrix
	var perMinuteStats []domain.SquadPerMinuteEntry
	var synergyRadar []domain.SquadSynergyRadarSeries
	var intensityProfile *domain.SquadIntensityProfile
	var performanceSeries map[string][]domain.SquadPerformanceSeriesPoint
	var weaponKills *domain.SquadWeaponKills
	var weaponAccuracy *domain.SquadWeaponAccuracy
	var fragClasses map[string][]domain.FragClassEntry
	var nativeKillMechanics *domain.SquadKillMechanics
	var firstBlood []domain.FirstBloodPlayerSeries
	var medalDigest []domain.MedalDigestEntry
	if len(allSquadRows) > 0 {
		// Résout map/playlist/mode FR sur les rows (mode via la cascade
		// canonique asset_translations + mode_name_tr, cf. enrichSquadMatchAssets).
		enrichSquadMatchAssets(ctx, s.repo, allSquadRows)
		timeseries = analysis.ComputeSquadTimeseries(allSquadRows, 20)
		mapBreakdown = computeMapBreakdown(allSquadRows)

		// Historique "avec cette escouade EXACTE" par carte. squadXUIDs = coéquipiers
		// sélectionnés (selectedXUIDs) ; excludeXUIDs = autres coéquipiers connus
		// (extraPool) → anti-join qui écarte les matchs où l'un d'eux était sur
		// l'équipe du main. Aucun filtre période/session : référence historique
		// complète, mais exclusive à la composition (parité avec allSquadRows).
		squadStats, err := s.repo.LoadMapStatsForSquad(ctx, playerXUID, selectedXUIDs, sortedXUIDSlice(extraPool))
		if err != nil {
			slog.WarnContext(ctx, "teammates: LoadMapStatsForSquad failed", "err", err)
		}
		mapBreakdown = enrichMapBreakdownWithSquadStats(mapBreakdown, squadStats)
		matchHistory = buildSquadMatchHistory(allSquadRows, squadStatsToWinTotal(squadStats), s.titleSlug)
		sessionTimeline = buildSquadSessionTimeline(allSquadRowsForTimeline)
		mapHeatmap = s.buildSquadMapHeatmap(ctx, allSquadRows, req.SelectedGamertags, sessionMatchIDs)
		impactMatrix = s.buildSquadImpactMatrix(ctx, allSquadRows, playerXUID, s.gamertag, req.SelectedGamertags)
		perMinuteStats = s.buildSquadPerMinuteStats(ctx, allSquadRows, s.gamertag, req.SelectedGamertags, sessionMatchIDs)
		synergyRadar = s.buildSquadSynergyRadar(ctx, allSquadRows, s.gamertag, req.SelectedGamertags)
		intensityProfile = s.buildSquadIntensityProfile(ctx, allSquadRows, s.gamertag, req.SelectedGamertags, "all")
		performanceSeries = s.buildSquadPerformanceSeries(ctx, allSquadRows, s.gamertag, playerXUID, req.SelectedGamertags, teammates)
		weaponKills, fragClasses = s.buildSquadWeaponKills(ctx, allSquadRows, s.gamertag, playerXUID, teammates, performanceSeries)
		weaponAccuracy = s.buildSquadWeaponAccuracy(ctx, allSquadRows, s.gamertag, playerXUID, teammates)
		nativeKillMechanics = s.buildSquadKillMechanics(ctx, allSquadRows, s.gamertag, playerXUID, teammates)
		firstBlood = s.buildSquadFirstBlood(ctx, allSquadRows, s.gamertag, playerXUID, teammates)
		medalDigest = s.buildMedalDigest(ctx, allSquadRows, s.gamertag, playerXUID, teammates, req.Locale)
	}

	// Header (SessionBriefing) — alimente le composant <SessionBriefing> dans
	// SquadLayout. Mode solo (SoloKPIs uniquement) si aucun coequipier
	// selectionne ; mode squad complet sinon.
	mainFilteredCanonical := filterCanonicalByMatchIDsSet(canonicalRows, filteredMatches)
	compFilter := &exactCompositionFilter{
		teamByMatch:   mainTeamByMatch,
		extraPool:     extraPool,
		selectedXUIDs: selectedXUIDs,
	}
	header := s.buildBriefingHeaderForTeammatesPage(
		ctx, mainFilteredCanonical, req.SelectedGamertags, req.Filters, sessionMatchIDs, compFilter,
	)

	// Sessions de la composition exacte : dérivées de l'intersection NON filtrée
	// par session (historique complet de la composition). Alimentent le
	// SessionMultiSelect et le ré-ancrage front. Sans coéquipier sélectionné, on
	// reprend les sessions squad du joueur principal (exploration inchangée).
	var compositionSessions []domain.SessionLabelEntry
	var latestCompositionSession string
	if len(req.SelectedGamertags) > 0 {
		compositionSessions = buildCompositionSessionLabels(allSquadRowsForTimeline)
		if len(compositionSessions) > 0 {
			latestCompositionSession = compositionSessions[0].Label
		}
		// Diagnostic : permet de tracer "pourquoi la session est vide / re-ancrée"
		// pour une composition donnée (cf. ./logs/general.log).
		slog.DebugContext(ctx, "teammates.composition_resolved",
			"player", s.gamertag,
			"selected_count", len(req.SelectedGamertags),
			"shared_matches", len(allSquadRowsForTimeline),
			"composition_sessions", len(compositionSessions),
			"latest_session", latestCompositionSession,
		)
	} else {
		compositionSessions = sessionLabels.Squad
	}

	return domain.TeammatesPageResponse{
		Options:             options,
		Teammates:           teammates,
		TotalMatches:        totalMatches,
		SessionLabels:       sessionLabels,
		FriendsCount:        len(friendGTs),
		Timeseries:          timeseries,
		MapBreakdown:        mapBreakdown,
		MatchSeries:         matchSeries,
		MatchHistory:        matchHistory,
		SessionTimeline:     sessionTimeline,
		MapHeatmap:          mapHeatmap,
		ImpactMatrix:        impactMatrix,
		PerMinuteStats:      perMinuteStats,
		SynergyRadar:        synergyRadar,
		IntensityProfile:    intensityProfile,
		PerformanceSeries:   performanceSeries,
		FragClasses:         fragClasses,
		WeaponKills:         weaponKills,
		WeaponAccuracy:      weaponAccuracy,
		NativeKillMechanics: nativeKillMechanics,
		FirstBlood:          firstBlood,
		Header:              header,
		MainPlayer:          s.gamertag,
		MedalDigest:         medalDigest,

		CompositionSessions:      compositionSessions,
		LatestCompositionSession: latestCompositionSession,
	}, nil
}

// filterCanonicalByMatchIDsSet ne garde que les canonical rows dont le match_id
// figure dans le slice de SynthesisMatchRow filtré (post cascade + sessions).
// Sert de pont entre la pipeline legacy SynthesisMatchRow et les builders
// canoniques (ComputeKPIStats, squadagg.BuildSquadHeader).
func filterCanonicalByMatchIDsSet(
	rows []canonical.PlayerMatchRow,
	filtered []legacymatch.SynthesisMatchRow,
) []canonical.PlayerMatchRow {
	if len(filtered) == 0 || len(rows) == 0 {
		return nil
	}
	keep := make(map[string]struct{}, len(filtered))
	for _, m := range filtered {
		keep[m.MatchID] = struct{}{}
	}
	out := make([]canonical.PlayerMatchRow, 0, len(filtered))
	for _, r := range rows {
		if _, ok := keep[r.Summary.MatchID]; ok {
			out = append(out, r)
		}
	}
	return out
}

// buildBriefingHeaderForTeammatesPage construit le SquadHeader pour la page
// Teammates. Mode solo si selectedGamertags vide ; mode squad complet sinon
// (charge les canonical rows par teammate en parallele puis appelle le builder
// existant squadagg.BuildSquadHeader).
//
// Degradation gracieuse : si le chargement des teammates echoue (capability
// absente, erreur DB), retourne au moins le SoloKPIs du joueur principal pour
// que le briefing reste utile en mode degrade.
