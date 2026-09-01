// Package service - synthesis_service.go : orchestration de la page Synthese.
//
// Sprint 55 D1 : extrait de squad_service.go - SynthesisService devient autonome,
// implemente port.SynthesisService.
//
// Sprint 55 D2 : period et filters du SynthesisRequest sont reellement appliques.
//
// Le code est decoupe en fichiers thematiques pour respecter la limite des
// 500 lignes par fichier (CLAUDE.md). Ce fichier contient le type service,
// le constructeur, les Withers et GetSynthesisPage (entry point) +
// loaders helpers. Les autres responsabilites vivent dans :
//
//   - synthesis_service_legacy.go    : builders legacy SynthesisMatchRow /
//     SynthesisHeatmapRow (filterByPeriod,
//     buildScopeDescription,
//     buildHighlightsPreview, buildRivalriesPreview,
//     buildBreakdowns, sortMap/ModeEntries)
//   - synthesis_service_canonical.go : filtres + best refs + overview
//     (filterSynthesisByPeriodCanonical,
//     bestTracker, computeSynthesisBestRefs,
//     buildSynthesisOverviewCanonical)
//   - synthesis_service_builders.go  : builders canonical (highlights,
//     detailed stats, top weapons, combat
//     profile, fun stats from awards)
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/fragdist"
)

// SynthesisService orchestre les donnÃ©es de la page SynthÃ¨se.
type SynthesisService struct {
	repo port.SynthesisRepository
	// dataAdapter (optionnel, Phase 2 plan finition multi-titres) :
	// quand fourni, GetSynthesisPage mesure la capability match.history pour
	// loguer une Ã©ventuelle dÃ©gradation.
	dataAdapter games.TitleDataAdapter
	// playerMatchesRepo (P4.1+P4.3, ADR 0011) : source canonical-aware. Quand
	// fournie avec titleSlug+gamertag, GetSynthesisPage charge directement
	// `[]canonical.PlayerMatchRow` et appelle les analyses *FromCanonical sans
	// converter. Le path legacy (s.repo.LoadSynthesisMatches) reste pour
	// rÃ©trocompatibilitÃ© tant que la DI cabling n'est pas mise Ã  jour partout.
	playerMatchesRepo port.PlayerMatchesRepository
	// personalScoreAwardsRepo (P9) : charge les fun stats (betrayals, suicides,
	// vehicles_destroyed, hijacks) depuis personal_score_awards.
	personalScoreAwardsRepo port.PersonalScoreAwardsRepository
	// weaponKillsRepo : charge les kills agrégés par arme depuis shared.weapon_kills.
	// Quand nil, le champ TopWeaponKills est omis de la réponse.
	weaponKillsRepo port.WeaponKillsRepository
	// weaponAccuracyRepo : charge la précision agrégée par arme depuis la table
	// weapon_accuracy (Halo 5 natif). Quand nil OU titre sans table (Infinite),
	// le champ WeaponAccuracy est omis de la réponse.
	weaponAccuracyRepo port.WeaponAccuracyRepository
	// vehicleDestructionRepo : source ALTERNATIVE (par titre) des compteurs
	// « véhicules détruits » / « vol à la tire ». Câblé UNIQUEMENT pour les titres à
	// commendations NATIVES (Halo 5, capability commendations.native — cf. registry
	// SynthesisCtx, jamais slug==). Quand fourni, il PRIME sur personal_score_awards
	// pour ces deux compteurs (personal_score_awards est vide pour ces titres). nil
	// (Infinite) → comportement inchangé : les deux compteurs viennent des awards.
	vehicleDestructionRepo port.VehicleDestructionStatsRepository
	// objectiveStatsRepo : cumul (SUM) des stats objectifs (CTF/Zones/Oddball) sur le
	// scope. Câblé UNIQUEMENT pour les titres à capability match.objective.stats
	// (Infinite ; nil pour Halo 5 → bloc objective_stats omis). Best-effort.
	objectiveStatsRepo port.ObjectiveStatsRepository
	// titleSlug est nécessaire pour appeler PlayerMatchesRepo.LoadPlayerMatches.
	// Si "" et playerMatchesRepo != nil, fallback sur le repo legacy.
	titleSlug  string
	gamertag   string
	playerXUID string
}

// NewSynthesisService crÃ©e un SynthesisService avec le repository injectÃ©.
func NewSynthesisService(repo port.SynthesisRepository) *SynthesisService {
	return &SynthesisService{repo: repo}
}

// WithDataAdapter injecte un games.TitleDataAdapter optionnel (Phase 2 plan
// finition multi-titres). Permet de logger le statut des capabilities et
// d'amorcer la bascule fonctionnelle vers la couche canonique.
func (s *SynthesisService) WithDataAdapter(a games.TitleDataAdapter) *SynthesisService {
	s.dataAdapter = a
	return s
}

// WithPlayerMatchesRepo (P4.1+P4.3, ADR 0011) injecte le loader canonical-aware.
// Quand fourni avec titleSlug+gamertag, GetSynthesisPage charge depuis le
// loader unifiÃ© et appelle les analyses *FromCanonical (pas de converter).
func (s *SynthesisService) WithPlayerMatchesRepo(
	repo port.PlayerMatchesRepository,
	titleSlug, gamertag string,
) *SynthesisService {
	s.playerMatchesRepo = repo
	s.titleSlug = titleSlug
	s.gamertag = gamertag
	return s
}

// WithPersonalScoreAwardsRepo (P9) injecte le loader pour les fun stats.
func (s *SynthesisService) WithPersonalScoreAwardsRepo(
	repo port.PersonalScoreAwardsRepository,
	playerXUID string,
) *SynthesisService {
	s.personalScoreAwardsRepo = repo
	s.playerXUID = playerXUID
	return s
}

func (s *SynthesisService) WithWeaponKillsRepo(repo port.WeaponKillsRepository) *SynthesisService {
	s.weaponKillsRepo = repo
	return s
}

// WithWeaponAccuracyRepo injecte le loader pour le classement précision par arme.
func (s *SynthesisService) WithWeaponAccuracyRepo(repo port.WeaponAccuracyRepository) *SynthesisService {
	s.weaponAccuracyRepo = repo
	return s
}

// WithVehicleDestructionStatsRepo injecte la source PAR TITRE des compteurs
// « véhicules détruits » / « vol à la tire » (Halo 5 : commendations natives pour les
// véhicules détruits, médailles Hijack/Skyjack pour le vol à la tire). Réutilise
// s.playerXUID (posé par WithPersonalScoreAwardsRepo, câblé en amont dans SynthesisCtx).
func (s *SynthesisService) WithVehicleDestructionStatsRepo(repo port.VehicleDestructionStatsRepository) *SynthesisService {
	s.vehicleDestructionRepo = repo
	return s
}

// WithObjectiveStatsRepo injecte la source du cumul des stats objectifs (CTF/Zones/
// Oddball) sur le scope. Câblé gated par la capability match.objective.stats
// (SynthesisCtx) ; nil → bloc objective_stats omis de la réponse.
func (s *SynthesisService) WithObjectiveStatsRepo(repo port.ObjectiveStatsRepository) *SynthesisService {
	s.objectiveStatsRepo = repo
	return s
}

// GetSynthesisPage construit la rÃ©ponse de la page SynthÃ¨se.
// Sprint 55 D2 : applique period et filters depuis le SynthesisRequest.
func (s *SynthesisService) GetSynthesisPage(
	ctx context.Context,
	playerXUID string,
	req domain.SynthesisRequest,
) (*domain.SynthesisPageV2Response, error) {
	period := req.Period
	if period == "" {
		period = "all"
	}

	// Phase 2 plan finition multi-titres : log de la capability match.history
	// quand un DataAdapter est injectÃ©. Sert Ã  mesurer la dÃ©gradation potentielle
	// avant la bascule fonctionnelle (le Synthesis lit aujourd'hui depuis le repo
	// legacy car canonical.PlayerStats ne couvre pas encore SynthesisMatch).
	if s.dataAdapter != nil {
		caps := s.dataAdapter.Capabilities()
		if !caps.Has(games.CapMatchHistory) {
			slog.WarnContext(ctx, "capability_not_supported",
				"title_slug", s.dataAdapter.TitleSlug(),
				"capability", string(games.CapMatchHistory),
				"caller", "synthesis_service.GetSynthesisPage",
			)
		}
	}

	// P4.3 finale (ADR 0011) : path canonical exclusif. Le legacy fallback
	// path a Ã©tÃ© supprimÃ© â€" playerMatchesRepo + titleSlug + gamertag sont
	// dÃ©sormais REQUIS (wirÃ©s universellement en DI via registry.go).
	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		return nil, fmt.Errorf("SynthesisService: PlayerMatchesRepo non cÃ¢blÃ© (P4.3 finale exige le wiring DI)")
	}
	canonicalRows, err := s.loadAndEnrichCanonicalRows(ctx)
	if err != nil {
		return nil, err
	}
	filteredCanon, filtersApplied, filtersIgnored := filterSynthesisByPeriodCanonical(canonicalRows, period, req.StartDate, req.EndDate)
	c := req.Filters.Cascade
	filteredCanon = filterRowsByCascade(filteredCanon, c.ExperienceTypes, c.Playlists, c.Maps, c.Modes)

	hp := games.EffectiveHpToKill(s.titleSlug)
	soloKPIs := analysis.ComputeSynthesisKPIsFromCanonical(filteredCanon, false, hp)
	squadKPIs := analysis.ComputeSynthesisKPIsFromCanonical(filteredCanon, true, hp)
	topWeeks := analysis.ComputeSynthesisTopWeeksFromCanonical(filteredCanon)
	heatmap := analysis.ComputeTemporalHeatmapFromCanonical(filteredCanon)
	provideSpree := games.ProvidesMaxKillingSpree(s.titleSlug)
	overview := buildSynthesisOverviewCanonical(filteredCanon, soloKPIs, provideSpree)
	slog.DebugContext(ctx, "synthesis: best refs detected",
		"player_xuid", playerXUID,
		"matches", len(filteredCanon),
		"kills_ref", overview.BestKillsRef != nil,
		"kda_ref", overview.BestKDARef != nil,
		"perf_ref", overview.BestPerfRef != nil,
		"accuracy_ref", overview.BestAccuracyRef != nil,
		"damage_ref", overview.BestDamageRef != nil,
		"killing_spree_ref", overview.BestKillingSpreeRef != nil,
		"headshots_ref", overview.BestHeadshotsRef != nil,
		"personal_score_ref", overview.BestPersonalScoreRef != nil,
	)
	highlights := buildHighlightsPreviewCanonical(filteredCanon)
	matchCount := len(filteredCanon)
	comparison := analysis.ComputeComparisonMetrics(soloKPIs, squadKPIs)

	// D7 : breakdowns map/mode depuis les rows canoniques filtrés (period-aware).
	// Les Labels["fr"] des AssetReference ont été hydratés par
	// EnrichCanonicalAssetTranslations en amont, donc buildBreakdownsFromCanonical
	// lit directement les noms FR.
	breakdowns := buildBreakdownsFromCanonical(filteredCanon)

	// P9 : stats détaillées (combat, tir, dégâts, fun)
	detailedStats := buildSynthesisDetailedStatsFromCanonical(filteredCanon, provideSpree)

	// P9 : fun stats depuis personal_score_awards (requete separee, erreur non fatale)
	s.applyFunStatsToDetailedStats(ctx, &detailedStats, filteredCanon)

	// Frags par arme + par rôle (registre) + répartition hiérarchique classe→rôle
	// (sunburst v2) : best-effort, ignoré si repo absent.
	topWeaponKills, fragDistribution := s.loadTopWeaponKills(ctx, filteredCanon, detailedStats, overview.TotalKills)

	// Précision par arme (Halo 5 natif) : best-effort, nil si repo absent ou
	// titre sans table weapon_accuracy (Infinite) → champ omis.
	weaponAccuracy := s.loadWeaponAccuracy(ctx, filteredCanon)

	// KPI objectifs (cumul CTF/Zones/Oddball sur le scope) : best-effort, nil si repo
	// absent (capability match.objective.stats non déclarée — Halo 5) ou scope sans
	// match à objectif → bloc omis.
	objectiveStats := s.loadObjectiveStats(ctx, filteredCanon)

	scope := domain.SynthesisScope{
		Period:         period,
		MatchCount:     matchCount,
		FiltersApplied: filtersApplied,
		FiltersIgnored: filtersIgnored,
		Description:    buildScopeDescription(period, matchCount),
		ComputedAt:     time.Now().UTC(),
	}

	combatProfile := buildCombatProfileFromCanonical(filteredCanon, hp)
	if combatProfile != nil {
		slog.DebugContext(ctx, "synthesis: combat profile computed",
			"matches", combatProfile.MatchCount,
			"avg_oc", combatProfile.AvgOC,
			"avg_dr", combatProfile.AvgDR,
			"has_pace_ratio", combatProfile.AvgPaceRatio != nil,
			"style_activity", combatProfile.StyleActivity)
	}

	return &domain.SynthesisPageV2Response{
		Scope:             scope,
		Overview:          overview,
		SoloKPIs:          soloKPIs,
		SquadKPIs:         squadKPIs,
		ComparisonMetrics: comparison,
		HeatmapData:       heatmap,
		TopWeeks:          topWeeks,
		HighlightsPreview: highlights,
		Breakdowns:        breakdowns,
		DetailedStats:     detailedStats,
		TopWeaponKills:    topWeaponKills,
		FragDistribution:  fragDistribution,
		WeaponAccuracy:    weaponAccuracy,
		CombatProfile:     combatProfile,
		ObjectiveStats:    objectiveStats,
	}, nil
}

// loadObjectiveStats agrège (SUM) les stats objectifs du joueur sur le scope filtré.
// Best-effort : nil si repo non câblé (capability absente), joueur inconnu, scope vide,
// erreur SQL, ou aucun match à objectif dans le scope (bloc omis de la réponse).
func (s *SynthesisService) loadObjectiveStats(
	ctx context.Context, filteredCanon []canonical.PlayerMatchRow,
) *domain.ObjectiveAggregate {
	if s.objectiveStatsRepo == nil || s.playerXUID == "" || len(filteredCanon) == 0 {
		return nil
	}
	matchIDs := make([]string, 0, len(filteredCanon))
	for _, r := range filteredCanon {
		matchIDs = append(matchIDs, r.Summary.MatchID)
	}
	byXUID, err := s.objectiveStatsRepo.LoadAggregatedByXUID(ctx, matchIDs, []string{s.playerXUID})
	if err != nil {
		slog.WarnContext(ctx, "synthesis: objective stats query failed (best-effort)",
			"player_xuid", s.playerXUID, "match_count", len(matchIDs), "err", err)
		return nil
	}
	return byXUID[s.playerXUID]
}

// loadAndEnrichCanonicalRows charge les canonical rows et applique
// EnrichCanonicalAssetTranslations + log diagnostic FR. Best-effort sur enrich.
func (s *SynthesisService) loadAndEnrichCanonicalRows(ctx context.Context) ([]canonical.PlayerMatchRow, error) {
	canonicalRows, err := s.playerMatchesRepo.LoadPlayerMatches(
		ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
	)
	if err != nil {
		return nil, fmt.Errorf("SynthesisService load: %w", err)
	}
	slog.DebugContext(ctx, "synthesis: loaded canonical",
		"rows", len(canonicalRows), "title_slug", s.titleSlug)
	if err := s.repo.EnrichCanonicalAssetTranslations(ctx, canonicalRows); err != nil {
		slog.WarnContext(ctx, "synthesis: EnrichCanonicalAssetTranslations failed", "err", err)
		return canonicalRows, nil
	}
	mapFR, pairFR := 0, 0
	for _, r := range canonicalRows {
		if r.Summary.Map != nil && r.Summary.Map.Labels["fr"] != "" {
			mapFR++
		}
		if r.Summary.PairMode != nil && r.Summary.PairMode.Labels["fr"] != "" {
			pairFR++
		}
	}
	slog.InfoContext(ctx, "synthesis: canonical FR enrichment",
		"rows", len(canonicalRows), "map_fr_count", mapFR, "pair_fr_count", pairFR)
	return canonicalRows, nil
}

// applyFunStatsToDetailedStats charge les fun stats et les fusionne dans DetailedStats.
// Source par défaut = personal_score_awards (Infinite : betrayals/suicides/véhicules/
// hijacks). Pour les titres à commendations NATIVES (Halo 5), véhicules détruits + vol
// à la tire PRIMENT sur les awards (vides pour ce titre) — véhicules détruits depuis
// match_commendations, vol à la tire depuis les médailles Hijack/Skyjack de
// medals_earned (aucune commendation « Grand Theft » n'existe côté H5, cf. doc
// internal/platform/duckdb/vehicle_commendation_stats_repo.go) — branchement par
// capability câblé en amont (vehicleDestructionRepo nil pour Infinite). Best-effort :
// une source en erreur ne casse pas la page (log, dégradation).
func (s *SynthesisService) applyFunStatsToDetailedStats(
	ctx context.Context, detailedStats *domain.SynthesisDetailedStats, filteredCanon []canonical.PlayerMatchRow,
) {
	if s.playerXUID == "" {
		return
	}
	matchIDs := make([]string, 0, len(filteredCanon))
	for _, r := range filteredCanon {
		matchIDs = append(matchIDs, r.Summary.MatchID)
	}
	if len(matchIDs) == 0 {
		return
	}

	if s.personalScoreAwardsRepo != nil {
		funStats, _ := buildSynthesisFunStatsFromAwards(ctx, s.personalScoreAwardsRepo, s.titleSlug, matchIDs, s.playerXUID)
		detailedStats.TotalBetrayals = funStats.TotalBetrayals
		detailedStats.TotalSuicides = funStats.TotalSuicides
		detailedStats.TotalVehiclesDestroyed = funStats.TotalVehiclesDestroyed
		detailedStats.TotalHijacks = funStats.TotalHijacks
	}

	// Halo 5 (commendations.native) : véhicules détruits (commendations natives) et
	// vol à la tire (médailles Hijack/Skyjack, medals_earned) PRIMENT sur les awards
	// (vides pour ce titre).
	if s.vehicleDestructionRepo != nil {
		vd, err := s.vehicleDestructionRepo.LoadVehicleDestructionStats(ctx, s.titleSlug, matchIDs, s.playerXUID)
		if err != nil {
			slog.WarnContext(ctx, "synthesis: vehicle destruction stats query failed (best-effort)",
				"title", s.titleSlug, "player_xuid", s.playerXUID, "match_count", len(matchIDs), "err", err)
		} else {
			detailedStats.TotalVehiclesDestroyed = vd.VehiclesDestroyed
			detailedStats.TotalHijacks = vd.Hijacks
		}
	}
}

// loadTopWeaponKills agrège, sur le scope filtré, les frags par arme (top 20) ET la
// répartition hiérarchique classe→rôle (sunburst v2). Une seule requête
// (ResolveRoles=true renseigne row.Role + row.Class). Best-effort : la FragDistribution
// est construite même si le repo d'armes est absent ou en erreur (les classes API
// melee/grenade/spartan + total restent servies ; les frags d'arme non résolus
// retombent dans « Non attribué »). nil partout si le scope est vide (total 0 → le
// front rend null).
func (s *SynthesisService) loadTopWeaponKills(
	ctx context.Context,
	filteredCanon []canonical.PlayerMatchRow,
	detailedStats domain.SynthesisDetailedStats,
	totalKills int,
) ([]domain.SynthesisWeaponKillEntry, *domain.FragDistribution) {
	if len(filteredCanon) == 0 || totalKills <= 0 {
		return nil, nil
	}
	rows := s.loadWeaponKillRows(ctx, filteredCanon)
	matchIDs := make([]string, 0, len(filteredCanon))
	for _, r := range filteredCanon {
		matchIDs = append(matchIDs, r.Summary.MatchID)
	}
	hasMechanics := titleHasNativeKillMechanics(s.titleSlug)
	counts := domain.FragKillTypeCounts{
		Melee:         detailedStats.TotalMeleeKills,
		Grenade:       detailedStats.TotalGrenadeKills,
		Assassination: detailedStats.TotalAssassinations,
		GroundPound:   detailedStats.TotalGroundPoundKills,
		ShoulderBash:  detailedStats.TotalShoulderBashKills,
		Total:         totalKills,
	}
	fd := fragdist.Build(rows, counts, hasMechanics)
	logFragDistribution(ctx, "synthesis", s.titleSlug, s.gamertag, fd)
	return buildTopWeaponKills(rows, synthesisWeaponChartTopN), &fd
}

// loadWeaponKillRows charge les rows agrégées d'armes (ResolveRoles=true → Role+Class).
// Best-effort : nil si repo absent, gamertag vide, ou erreur (loggée, jamais avalée).
func (s *SynthesisService) loadWeaponKillRows(
	ctx context.Context, filteredCanon []canonical.PlayerMatchRow,
) []port.WeaponKillRow {
	if s.weaponKillsRepo == nil || s.gamertag == "" {
		return nil
	}
	matchIDs := make([]string, 0, len(filteredCanon))
	for _, r := range filteredCanon {
		matchIDs = append(matchIDs, r.Summary.MatchID)
	}
	wf := port.WeaponKillFilters{MatchIDs: matchIDs, Gamertag: s.gamertag, ResolveRoles: true}
	rows, err := s.weaponKillsRepo.LoadWeaponKillsAggregated(ctx, s.titleSlug, wf)
	if err != nil {
		// ErrCapabilityNotSupported = légitime (titre sans table weapon_kills) → Debug.
		// Toute autre erreur = anomalie (SQL invalide, conn timeout, etc.) → Warn,
		// pour éviter qu'un nouveau bug "shared.X prefix" passe inaperçu comme en
		// 2026-05-26 (chart breakdown armes vide pendant ~1 semaine).
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.DebugContext(ctx, "synthesis: weapon kills capability absente",
				"title", s.titleSlug, "gamertag", s.gamertag)
		} else {
			slog.WarnContext(ctx, "synthesis: weapon kills query failed (best-effort, fallback nil)",
				"title", s.titleSlug, "gamertag", s.gamertag,
				"match_count", len(matchIDs), "err", err)
		}
		return nil
	}
	return rows
}

// titleHasNativeKillMechanics indique si le titre fournit NATIVEMENT le détail des
// kills par mécanique (assassinats + compétences spartiate). Capability-gated (jamais
// de comparaison de slug — ratchet no_slug_comparison_test.go). Titre inconnu → false.
func titleHasNativeKillMechanics(slug string) bool {
	d := titlePkg.DefaultRegistry().Get(slug)
	return d != nil && d.HasCapability(titlePkg.CapNativeKillMechanics)
}

// loadWeaponAccuracy agrège, sur le scope filtré, la précision par arme (top N
// armes, même cap que « Frags par arme »). Calqué sur loadTopWeaponKills.
// Best-effort : nil si repo
// absent, gamertag/scope vide, ou capability manquante (titre sans table
// weapon_accuracy, ex. Infinite).
func (s *SynthesisService) loadWeaponAccuracy(
	ctx context.Context, filteredCanon []canonical.PlayerMatchRow,
) []domain.SynthesisWeaponAccuracyEntry {
	if s.weaponAccuracyRepo == nil || s.gamertag == "" || len(filteredCanon) == 0 {
		return nil
	}
	matchIDs := make([]string, 0, len(filteredCanon))
	for _, r := range filteredCanon {
		matchIDs = append(matchIDs, r.Summary.MatchID)
	}
	wf := port.WeaponAccuracyFilters{MatchIDs: matchIDs, Gamertag: s.gamertag}
	rows, err := s.weaponAccuracyRepo.LoadWeaponAccuracyAggregated(ctx, s.titleSlug, wf)
	if err != nil {
		// ErrCapabilityNotSupported = légitime (titre sans table weapon_accuracy,
		// ex. Infinite) → Debug. Toute autre erreur = anomalie → Warn (parité avec
		// loadTopWeaponKills, pour ne pas masquer un futur bug SQL/conn).
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.DebugContext(ctx, "synthesis: weapon accuracy capability absente",
				"title", s.titleSlug, "gamertag", s.gamertag)
		} else {
			slog.WarnContext(ctx, "synthesis: weapon accuracy query failed (best-effort, fallback nil)",
				"title", s.titleSlug, "gamertag", s.gamertag,
				"match_count", len(matchIDs), "err", err)
		}
		return nil
	}
	return buildWeaponAccuracy(rows, synthesisWeaponChartTopN)
}

// =============================================================================
// Helpers internes
// =============================================================================

// filterSynthesisByPeriod filtre les matchs SynthÃ¨se selon la pÃ©riode demandÃ©e.
// Retourne les matchs filtrÃ©s, les filtres appliquÃ©s et ceux ignorÃ©s.
