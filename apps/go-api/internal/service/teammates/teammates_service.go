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
//     loadTeammatesCanonical +
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
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

// FriendGamertagsResolver retourne la liste courante des amis DU JOUEUR
// consulté (data/global/player_friends.json). Appelé à chaque requête pour refléter les
// PATCH settings sans redémarrage.
type FriendGamertagsResolver func(ctx context.Context) []string

// TeammatesService calcule les stats coéquipiers au format FastAPI.
type TeammatesService struct {
	repo            port.SquadRepository
	friendGamertags FriendGamertagsResolver
	// profils / lireAmis : les sources des coéquipiers connus (pool de la composition stricte),
	// cf. coequipiers_connus.go.
	profils  ProfilsDuTitre
	lireAmis FriendXUIDsReader
	// playerMatchesRepo (P4.3 finale) : loader canonical-only. Câblé en DI
	// universellement via registry.go (ServiceRegistry.playerMatchesAdapterFor).
	// IMPORTANT : cet adapteur est BOUND au gamertag du joueur principal (ignore
	// l'arg gamertag). Pour charger les canonical rows d'un coequipier different,
	// utiliser squadLoader.LoadFor (resolution dynamique par gamertag).
	playerMatchesRepo port.PlayerMatchesRepository
	titleSlug         string
	// roundsDecide : variantes dont le RESULTAT se lit en manches (regulation.toml).
	// Nil/absente -> lecture en points.
	roundsDecide map[string]bool
	gamertag     string
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
	// replaySvc (optionnel) : service de rejeu 2D, appelé UNE FOIS par requête pour
	// lister les matchs ayant un artefact (colonne « Rejeu » du tableau historique de
	// l'escouade). Nil → aucune ligne ne porte de rejeu (dégradation gracieuse).
	replaySvc port.ReplayService
	// radarRange / placementRepo (optionnels) : la table des portées de radar du titre et le
	// lecteur du placement des vies, les deux entrées du bloc « Groupés ou isolés » de
	// l'Emprise, cf. teammates_service_emprise_placement.go.
	radarRange    map[string]int
	placementRepo port.SquadLifePlacementRepository
	// sessionUsageRepo (optionnel) : le résumé d'usage (vues _latest) du bloc
	// « servi ou gâché » de l'équipement (étape E6.1bis). Câblé gated par
	// film.usage_summary ; nil → bloc servi avec Available=false et raison
	// machine. Cf. teammates_service_usage.go.
	sessionUsageRepo port.SessionUsageRepository
	// formesUsageRepo / formesObjectiveRepo / repoRoot (lot D2) : les deux sources du bloc
	// « formes retenues » et la racine du dépôt (catalogue d'armes), cf. teammates_service_formes.go.
	formesUsageRepo     port.SessionUsageRepository
	formesObjectiveRepo port.SquadFormesObjectiveRepository
	repoRoot            string
	objectiveModeEcarte func(pairName string) bool  // D6, cf. teammates_service_objective_history.go
	empriseRepo         port.SquadEmpriseRepository // feuille de match de l'Emprise, cf. teammates_service_emprise.go
	vehicleRepo         port.SquadVehicleRepository // ressource véhicules de l'Emprise, cf. teammates_service_emprise_vehicles.go
	// matchRangeRepo (optionnel) : le lecteur de portee de frag de TOUT le lobby, par
	// match (lot N2, D22-5). Sans lui, pas de referentiel : le bloc « roles de portee »
	// est omis. Cf. teammates_squad_range.go.
	matchRangeRepo port.MatchRangeRepository
	trends         TrendsDeps // page Tendances, vue Escouade : teammates_service_trends.go
}

// NewTeammatesService crée un TeammatesService.
//
// friendGamertags : optionnel. Si nil, le filtre amis-only est désactivé
// (top retourné brut). Quand fourni, le top dropdown est restreint aux amis
// configurés, et ces amis comptent parmi les coéquipiers connus de la
// composition stricte (coequipiers_connus.go).
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

// WithRoundsDecide injecte la table `game_variant_name -> le resultat se lit en MANCHES`
// (regulation.toml [rounds_decide], ADR 0032). Sans injection, le tableau de l'escouade
// affiche le score de l'API — jamais une regression, mais une incoherence avec la vue match
// si le titre en declare : c'est pourquoi le wiring l'injecte toujours.
func (s *TeammatesService) WithRoundsDecide(roundsDecide map[string]bool) *TeammatesService {
	s.roundsDecide = roundsDecide
	return s
}

// WithSquadLoader injecte le loader per-gamertag utilise pour le SessionBriefing
// mode squad (canonical rows de chaque coequipier via TitlePlayerResolver, lues une
// fois par requete — lecturesDeLaPage). Si non cable, le briefing degrade en mode solo.
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

// WithReplay injecte le service de rejeu 2D — MÊME service que l'endpoint /replay
// (une seule résolution de chemin dans le dépôt). Seul AvailableSet est appelé : un
// listing de dossier par requête, jamais un accès disque par ligne du tableau.
// Sans injection : has_replay reste faux sur toutes les lignes.
func (s *TeammatesService) WithReplay(svc port.ReplayService) *TeammatesService {
	s.replaySvc = svc
	return s
}

// replayAvailability liste les matchs ayant un artefact de rejeu. Service non câblé
// ou listing en échec (déjà journalisé côté service de rejeu) : ensemble vide — la
// page se sert sans la colonne plutôt qu'en erreur.
func (s *TeammatesService) replayAvailability(ctx context.Context) port.ReplayAvailability {
	if s.replaySvc == nil {
		return nil
	}
	set, err := s.replaySvc.AvailableSet(ctx)
	if err != nil {
		return nil
	}
	return set
}

// GetPage retourne la page Teammates avec options, comparaisons et solo ref.
//
// boucle gamertags → calcule timeseries/heatmap/impact. Splitter en sous-fonctions
// nécessiterait 5+ params chacune et perdrait la vue d'ensemble du flow.
//
// Annulation (D2.7) : ctx.Err() est vérifié entre les sections ; une requête annulée rend
// l'erreur du contexte (requeteAnnulee), jamais une page partielle.
//
//nolint:funlen // Orchestrateur séquentiel : charge top → filtre → applique cascade →
func (s *TeammatesService) GetPage(
	ctx context.Context,
	playerXUID string,
	req domain.TeammatesQueryRequest,
) (domain.TeammatesPageResponse, error) {
	s, lectures := s.pourLaRequete() // lectures partagées par les blocs (teammates_service_loads.go)
	stop := timing.FromContext(ctx).Section("top_teammates")
	topRows, err := s.repo.LoadTopTeammates(ctx, playerXUID)
	stop()
	if err != nil {
		return domain.TeammatesPageResponse{}, fmt.Errorf("TeammatesService: %w", err)
	}

	// §3 plan Squad/Sessions : filtre top dropdown aux amis configurés
	// (liste d'amis du joueur). Hors amis = exclus du dropdown mais
	// toujours requêtables explicitement via SelectedGamertags + alias.
	var friendGTs []string
	if s.friendGamertags != nil {
		friendGTs = s.friendGamertags(ctx)
	}
	dropdownRows := topRows
	if friendGTs != nil {
		dropdownRows = filterTopRowsToFriends(topRows, friendGTs)
	}

	// Options (liste des coéquipiers fréquents — limitée aux amis si configuré).
	options := buildTeammateOptions(dropdownRows)

	// P4.3 finale (ADR 0011) : load canonical via PlayerMatchesRepo, convert
	// vers SynthesisMatchRow pour les helpers internes (extractSynthesisSessionLabels,
	// filterSynthesisByCascade, etc.).
	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		return domain.TeammatesPageResponse{}, fmt.Errorf("TeammatesService: PlayerMatchesRepo non câblé (P4.3 finale exige le wiring DI)")
	}
	if err := ctx.Err(); err != nil {
		return requeteAnnulee(err)
	}
	stop = timing.FromContext(ctx).Section("player_matches")
	canonicalRows, err := s.playerMatchesRepo.LoadPlayerMatches(
		ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
	)
	stop()
	if err != nil {
		return domain.TeammatesPageResponse{}, fmt.Errorf("TeammatesService synthesis: %w", err)
	}
	allMatches := analysis.SynthesisMatchRowsFromCanonical(canonicalRows)

	// Extraire les session_labels disponibles (solo / escouade).
	sessionLabels := extractSynthesisSessionLabels(allMatches)

	// Filtrer les matchs selon les sessions sélectionnées.
	filteredMatches := filterSynthesisBySession(allMatches, req.PickedSoloSessions, req.PickedSquadSessions)

	// Appliquer les filtres cascade (experience_types, playlists) si présents.
	if req.Filters != nil {
		filteredMatches = filterSynthesisByCascade(filteredMatches, req.Filters.Cascade)
		// Period (rail nav, PeriodePill) — sans cela la navigation periode n'a aucun
		// effet sur les charts/tableaux Escouade. Filtre par StartTime [start, end+1j-1s].
		filteredMatches = filterSynthesisByPeriodInput(filteredMatches, req.Filters.Period)
		// picked_sessions du filterContext (rail nav, FilterOmnibar SessionPill).
		// Vit en parallele de PickedSquadSessions/PickedSoloSessions ; intersection
		// volontaire pour le cas multi-select + nav, en pratique l'un est vide quand
		// l'autre est pose donc l'effet net est equivalent a "remplacement".
		filteredMatches = filterSynthesisByPickedSessions(filteredMatches, req.Filters.Sessions.PickedSessions, canonicalRows)
	}

	totalMatches := len(filteredMatches)

	// Set d'IDs de session si une session est piquée, par l'un OU l'autre chemin (D2.5).
	// Nil = pas de filtre = tous les matchs escouade retournés.
	sessionMatchIDs := sessionMatchIDsDeLaPage(req, filteredMatches)

	// Calculs détaillés pour les gamertags sélectionnés.
	teammates := make([]domain.TeammateRow, 0, len(req.SelectedGamertags))
	matchSeries := map[string][]domain.SquadMatchSeriesPoint{}

	// Sets par coéquipier : chaque set = matchs communs (main ∩ ce coéquipier).
	// L'INTERSECTION de ces sets = matchs joués par le joueur principal ET TOUS
	// les coéquipiers sélectionnés (composition exacte). Avant ce fix on faisait
	// une union (un match joué sans un coéquipier survivait), d'où le bug
	// "coéquipier ajouté à une session qu'il n'a pas jouée".
	var setsFiltered, setsAllForTimeline [][]domain.SquadMatchRow

	// Dégradations best-effort de CETTE requête : loggées en erreur et remontées
	// dans la réponse (cf. teammates_data_issues.go).
	issues := &dataIssues{}

	for _, gt := range req.SelectedGamertags {
		if err := ctx.Err(); err != nil {
			return requeteAnnulee(err)
		}
		row, squadMatches, allSquadMatchesTm, err := s.buildTeammateRowWithMatches(ctx, playerXUID, gt, topRows, filteredMatches, sessionMatchIDs)
		if err != nil {
			// Le coéquipier disparaît de la population commune : la page reste
			// affichable mais ses nombres sont amputés → l'UI doit le dire.
			issues.add(ctx, domain.DataIssueTeammateMatches, gt, err)
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
	if err := ctx.Err(); err != nil {
		return requeteAnnulee(err)
	}

	// Population canonique du contexte escouade : « matchs commencés ensemble » =
	// intersection du roster (matchs joués par le main ET tous les sélectionnés).
	// C'est la SEULE population consommée par les compteurs, charts, heatmap et
	// briefing — aucun consommateur ne re-filtre pour son compte.
	allSquadRows := intersectSquadRowsByMatchID(setsFiltered)
	allSquadRowsForTimeline := intersectSquadRowsByMatchID(setsAllForTimeline)

	// rosterRowsForTimeline : population AVANT l'option composition exacte
	// (intersection du roster, historique complet) — publiée telle quelle en
	// MatchCountRoster par session (ADR 0033 critère 3), indépendamment du
	// résultat du filtre ci-dessous.
	rosterRowsForTimeline := allSquadRowsForTimeline

	// Équipe alliée du main par match (Q32b) : chargée UNE fois sur l'union des
	// matchs et partagée par ses trois consommateurs (filtre composition exacte,
	// matrice d'impact, courbe « équipe » du profil d'intensité) — cf. loadMainTeamAllies.
	selectedXUIDs := collectSelectedXUIDs(teammates)
	// Coéquipiers connus (amis déclarés et profils suivis, coequipiers_connus.go) : le pool de
	// l'option, lu sous l'option seulement — hors option aucun match n'est écarté.
	var connus map[string]string
	if req.FilterExactComposition && len(selectedXUIDs) > 0 {
		connus = s.coequipiersConnus(ctx, friendGTs)
	}
	extraPool := buildExtraPoolXUIDs(connus, selectedXUIDs, playerXUID)
	allies, mainTeamByMatch := s.loadMainTeamAllies(
		ctx, playerXUID, collectMatchIDs(allSquadRowsForTimeline, allSquadRows),
		req.FilterExactComposition && len(selectedXUIDs) > 0, issues)

	// Option « composition exacte » (req.FilterExactComposition) : restreint en plus aux
	// matchs où AUCUN autre coéquipier connu (extraPool) n'était sur l'équipe alliée du main.
	// Alliés non chargés : intersection du roster gardée (dégradation gracieuse), filtre
	// briefing désactivé (exactTeamByMatch nil), échec remonté à l'UI par loadMainTeamAllies.
	var exactTeamByMatch map[string]map[string]struct{}
	var excludedForTimeline []domain.SquadMatchRow
	if req.FilterExactComposition && len(selectedXUIDs) > 0 && mainTeamByMatch != nil {
		exactTeamByMatch = mainTeamByMatch
		allSquadRows, _ = filterExactComposition(allSquadRows, mainTeamByMatch, extraPool, selectedXUIDs)
		allSquadRowsForTimeline, excludedForTimeline = filterExactComposition(allSquadRowsForTimeline, mainTeamByMatch, extraPool, selectedXUIDs)
	}

	// Sections de la population escouade (composition exacte comprise) : tableaux puis graphes,
	// chacune sautée dès que la requête est annulée (teammates_service_sections.go).
	siVivante(ctx, func() { lectures.precharger(ctx, req.SelectedGamertags, allSquadRows) })
	sec := s.sectionsDeLaPopulation(ctx, populationEscouade{
		playerXUID: playerXUID, req: req, rows: allSquadRows, rowsTimeline: allSquadRowsForTimeline,
		teammates: teammates, allies: allies, mainTeamByMatch: mainTeamByMatch,
		sessionMatchIDs: sessionMatchIDs, selectedXUIDs: selectedXUIDs, extraPool: extraPool,
		issues: issues,
	})
	if err := ctx.Err(); err != nil {
		return requeteAnnulee(err)
	}

	// Header (SessionBriefing) de SquadLayout : mode solo (SoloKPIs uniquement) sans coéquipier
	// sélectionné, mode squad complet sinon.
	mainFilteredCanonical := filterCanonicalByMatchIDsSet(canonicalRows, filteredMatches)
	compFilter := &exactCompositionFilter{
		teamByMatch:   exactTeamByMatch,
		extraPool:     extraPool,
		selectedXUIDs: selectedXUIDs,
	}
	header := s.buildBriefingHeaderForTeammatesPage(
		ctx, mainFilteredCanonical, req.SelectedGamertags, req.Filters, sessionMatchIDs, compFilter,
	)

	// Sessions de la composition exacte : dérivées de l'intersection NON filtrée
	// par session (historique complet de la composition). Alimentent le
	// SessionMultiSelect et le ré-ancrage front. MatchCount reste le compte
	// POST-filtre (SOURCE UNIQUE, ADR 0033) ; MatchCountRoster (compte AVANT le
	// filtre exclusif) et ExcludedByExactComposition (matchs écartés + coéquipier
	// responsable nommé) publient l'écart sous l'option composition exacte
	// (critère 3). Sans coéquipier sélectionné, on reprend les sessions squad du
	// joueur principal (exploration inchangée) — pas d'écart à publier.
	var compositionSessions []domain.CompositionSessionEntry
	var latestCompositionSession string
	if len(req.SelectedGamertags) > 0 {
		stop = timing.FromContext(ctx).Section("composition_sessions")
		compositionSessions = buildCompositionSessionEntries(
			allSquadRowsForTimeline, rosterRowsForTimeline, excludedForTimeline,
			mainTeamByMatch, extraPool, connus,
		)
		stop()
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
		// Dénominateur de l'écart composition exacte (ADR 0033 critère 3) : combien
		// de sessions publient un écart, combien de matchs gardés/écartés, combien
		// de coéquipiers distincts en sont responsables. Seulement si le filtre a
		// effectivement écarté quelque chose (sinon rien à tracer).
		if len(excludedForTimeline) > 0 {
			slog.InfoContext(ctx, "teammates.exact_composition_gap",
				"player", s.gamertag,
				"sessions", len(compositionSessions),
				"kept_matches", len(allSquadRowsForTimeline),
				"excluded_matches", len(excludedForTimeline),
				"distinct_culprits", countDistinctCulpritXUIDs(excludedForTimeline, mainTeamByMatch, extraPool),
			)
		}
	} else {
		compositionSessions = wrapSessionLabelsAsComposition(sessionLabels.Squad)
	}

	// Blocs d'usage (servi ou gâché, formes, historique d'objectif, Emprise) : best-effort, sur le
	// périmètre D2 — composition exacte ∩ filteredMatches, ou filteredMatches seul (teammates_service_usage.go).
	usage := s.loadUsageBlocks(ctx, playerXUID, porteeUsage{
		filtered: filteredMatches, squadRows: allSquadRows, timelineRows: allSquadRowsForTimeline,
		mainTeamByMatch: mainTeamByMatch, history: sec.matchHistory, pairNames: pairNamesOf(canonicalRows, allSquadRowsForTimeline),
		compositionSessions: compositionSessions,
	}, req)
	if err := ctx.Err(); err != nil {
		return requeteAnnulee(err)
	}

	return domain.TeammatesPageResponse{
		Options:             options,
		Teammates:           teammates,
		TotalMatches:        totalMatches,
		SessionLabels:       sessionLabels,
		FriendsCount:        len(friendGTs),
		Timeseries:          sec.timeseries,
		MapBreakdown:        sec.mapBreakdown,
		MatchSeries:         matchSeries,
		MatchHistory:        sec.matchHistory,
		SessionTimeline:     sec.sessionTimeline,
		MapHeatmap:          sec.mapHeatmap,
		ImpactMatrix:        sec.impactMatrix,
		PerMinuteStats:      sec.perMinuteStats,
		SynergyRadar:        sec.synergyRadar,
		IntensityProfile:    sec.intensityProfile,
		PerformanceSeries:   sec.performanceSeries,
		FragClasses:         sec.fragClasses,
		WeaponTools:         sec.weaponTools,
		WeaponAccuracy:      sec.weaponAccuracy,
		NativeKillMechanics: sec.nativeKillMechanics,
		FirstBlood:          sec.firstBlood,
		AssistPairs:         sec.assistPairs,
		RangeProfiles:       sec.rangeProfiles,
		Header:              header,
		MainPlayer:          s.gamertag,
		MedalDigest:         sec.medalDigest,

		CompositionSessions:      compositionSessions,
		LatestCompositionSession: latestCompositionSession,
		DataIssues:               issues.list(),
		SquadFormes:              usage.formes,
		SquadObjectiveHistory:    usage.objectif,
		SquadEmprise:             usage.emprise,
	}, nil
}

// buildBriefingHeaderForTeammatesPage construit le SquadHeader pour la page
// Teammates. Mode solo si selectedGamertags vide ; mode squad complet sinon
// (relit les canonical rows de chaque teammate, lues une fois par requete, puis appelle
// le builder existant squadagg.BuildSquadHeader).
//
// Degradation gracieuse : si le chargement des teammates echoue (capability
// absente, erreur DB), retourne au moins le SoloKPIs du joueur principal pour
// que le briefing reste utile en mode degrade.
