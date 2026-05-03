// Package service Ã¢â‚¬â€ TeammatesService : endpoint POST /pages/teammates (contrat FastAPI).
//
// Sprint 33 : adapte les donnÃƒÂ©es SquadRepository vers le contrat TeammatesPageResponse.
// RÃƒÂ©utilise les mÃƒÂªmes queries Q29-Q31 que SquadService mais expose le format FastAPI.
package service

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
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
	squadLoader SquadV2Loader
	// medalDefs (optionnel) : résout les labels/descriptions anglais des médailles
	// depuis metadata.medal_definitions. Si nil, le digest est retourné sans
	// labels (medal_id et count seulement).
	medalDefs port.MedalDefinitionsRepository
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
func (s *TeammatesService) WithSquadLoader(loader SquadV2Loader) *TeammatesService {
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

// GetPage retourne la page Teammates avec options, comparaisons et solo ref.
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
	var allSquadRows []domain.SquadMatchRow
	var allSquadRowsForTimeline []domain.SquadMatchRow
	matchSeries := map[string][]domain.SquadMatchSeriesPoint{}

	for _, gt := range req.SelectedGamertags {
		row, squadMatches, allSquadMatchesTm, err := s.buildTeammateRowWithMatches(ctx, playerXUID, gt, topRows, filteredMatches, sessionMatchIDs)
		if err != nil {
			slog.WarnContext(ctx, "teammates: erreur buildTeammateRow", "gamertag", gt, "err", err)
			continue // skip teammate on error
		}
		if row != nil {
			teammates = append(teammates, *row)
			allSquadRows = append(allSquadRows, squadMatches...)
			allSquadRowsForTimeline = append(allSquadRowsForTimeline, allSquadMatchesTm...)
			matchSeries[gt] = buildMatchSeries(squadMatches)
		}
	}

	// Timeseries + MapBreakdown sur l'union des matchs escouade.
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
	var firstEvents *domain.SquadFirstEvents
	var medalDigest []domain.MedalDigestEntry
	if len(allSquadRows) > 0 {
		enrichSquadMatchAssets(ctx, s.repo, allSquadRows)
		modeFR, err := s.repo.LoadModeTranslationsFR(ctx, collectModeENs(allSquadRows))
		if err != nil {
			slog.WarnContext(ctx, "teammates: LoadModeTranslationsFR failed", "err", err)
		}
		timeseries = analysis.ComputeSquadTimeseries(allSquadRows, 20)
		mapBreakdown = computeMapBreakdown(allSquadRows)
		historicalWR := computeHistoricalMapWRByLabel(canonicalRows)
		mapBreakdown = enrichMapBreakdownWithHistory(mapBreakdown, historicalWR)
		historicalPerf := computeHistoricalMapPerfByLabel(canonicalRows)
		mapBreakdown = enrichMapBreakdownWithHistoricalPerf(mapBreakdown, historicalPerf)
		matchHistory = buildSquadMatchHistory(allSquadRows, modeFR)
		sessionTimeline = buildSquadSessionTimeline(allSquadRowsForTimeline)
		mapHeatmap = s.buildSquadMapHeatmap(ctx, allSquadRows, req.SelectedGamertags, sessionMatchIDs)
		impactMatrix = s.buildSquadImpactMatrix(ctx, allSquadRows, s.gamertag, req.SelectedGamertags)
		perMinuteStats = s.buildSquadPerMinuteStats(ctx, allSquadRows, s.gamertag, req.SelectedGamertags, sessionMatchIDs)
		synergyRadar = s.buildSquadSynergyRadar(ctx, allSquadRows, s.gamertag, req.SelectedGamertags)
		intensityProfile = s.buildSquadIntensityProfile(ctx, allSquadRows, s.gamertag, req.SelectedGamertags, "all")
		performanceSeries = s.buildSquadPerformanceSeries(ctx, allSquadRows, s.gamertag, req.SelectedGamertags)
		weaponKills = s.buildSquadWeaponKills(ctx, allSquadRows, s.gamertag, playerXUID, teammates)
		firstEvents = s.buildSquadFirstEvents(ctx, allSquadRows, s.gamertag, playerXUID, teammates)
		medalDigest = s.buildMedalDigest(ctx, allSquadRows, s.gamertag, playerXUID, teammates, req.Locale)
	}

	// Header (SessionBriefing) — alimente le composant <SessionBriefing> dans
	// SquadLayout. Mode solo (SoloKPIs uniquement) si aucun coequipier
	// selectionne ; mode squad complet sinon.
	mainFilteredCanonical := filterCanonicalByMatchIDsSet(canonicalRows, filteredMatches)
	header := s.buildBriefingHeaderForTeammatesPage(
		ctx, mainFilteredCanonical, req.SelectedGamertags, req.Filters, sessionMatchIDs,
	)

	return domain.TeammatesPageResponse{
		Options:           options,
		Teammates:         teammates,
		TotalMatches:      totalMatches,
		SessionLabels:     sessionLabels,
		FriendsCount:      len(friendGTs),
		Timeseries:        timeseries,
		MapBreakdown:      mapBreakdown,
		MatchSeries:       matchSeries,
		MatchHistory:      matchHistory,
		SessionTimeline:   sessionTimeline,
		MapHeatmap:        mapHeatmap,
		ImpactMatrix:      impactMatrix,
		PerMinuteStats:    perMinuteStats,
		SynergyRadar:      synergyRadar,
		IntensityProfile:  intensityProfile,
		PerformanceSeries: performanceSeries,
		WeaponKills:       weaponKills,
		FirstEvents:       firstEvents,
		Header:            header,
		MainPlayer:        s.gamertag,
		MedalDigest:       medalDigest,
	}, nil
}

// filterCanonicalByMatchIDsSet ne garde que les canonical rows dont le match_id
// figure dans le slice de SynthesisMatchRow filtré (post cascade + sessions).
// Sert de pont entre la pipeline legacy SynthesisMatchRow et les builders
// canoniques (ComputeKPIStats, buildSquadHeader).
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
// existant buildSquadHeader).
//
// Degradation gracieuse : si le chargement des teammates echoue (capability
// absente, erreur DB), retourne au moins le SoloKPIs du joueur principal pour
// que le briefing reste utile en mode degrade.
func (s *TeammatesService) buildBriefingHeaderForTeammatesPage(
	ctx context.Context,
	mainFiltered []canonical.PlayerMatchRow,
	selectedGamertags []string,
	filters *domain.FilterContextInput,
	sessionMatchIDs map[string]bool,
) *domain.SquadHeader {
	// Mode solo : SoloKPIs uniquement (pas de verdict squad).
	// Egalement le cas si squadLoader pas cable (degradation gracieuse).
	if len(selectedGamertags) == 0 || s.squadLoader == nil {
		if len(mainFiltered) == 0 {
			return nil
		}
		kpis := analysis.ComputeKPIStats(mainFiltered)
		return &domain.SquadHeader{SoloKPIs: &kpis}
	}

	// Mode squad : charge canonical rows par teammate en parallele.
	teammateRows, err := s.loadTeammatesCanonicalParallel(ctx, selectedGamertags)
	if err != nil {
		slog.WarnContext(ctx, "teammates_briefing.load_failed",
			"err", err.Error(), "selected_count", len(selectedGamertags))
		// Degradation : juste SoloKPIs.
		if len(mainFiltered) == 0 {
			return nil
		}
		kpis := analysis.ComputeKPIStats(mainFiltered)
		return &domain.SquadHeader{SoloKPIs: &kpis}
	}

	// Construire perPlayer : main + chaque teammate (filtres appliques).
	perPlayer := map[string][]canonical.PlayerMatchRow{s.gamertag: mainFiltered}
	for gt, rows := range teammateRows {
		filtered := rows
		if filters != nil {
			c := filters.Cascade
			filtered = filterRowsByCascade(filtered, c.ExperienceTypes, c.Playlists, c.Maps, c.Modes)
		}
		if len(sessionMatchIDs) > 0 {
			kept := make([]canonical.PlayerMatchRow, 0, len(filtered))
			for _, r := range filtered {
				if sessionMatchIDs[r.Summary.MatchID] {
					kept = append(kept, r)
				}
			}
			filtered = kept
		}
		perPlayer[gt] = filtered
	}

	squadOrder := buildSquadOrder(s.gamertag, selectedGamertags)
	gtToXUID := extractSquadXUIDs(squadOrder, perPlayer)
	sharedMatches := intersectByMatchID(perPlayer)

	return buildSquadHeader(ctx, s.gamertag, perPlayer, gtToXUID, sharedMatches)
}

// loadTeammatesCanonicalParallel charge les canonical PlayerMatchRow pour
// chaque gamertag en parallele via errgroup. Capability absente est ignoree
// silencieusement (le teammate sera juste absent du resultat).
//
// Utilise squadLoader.LoadFor (resolution dynamique par gamertag) plutot que
// playerMatchesRepo (qui est bound au main et ignore l'arg gamertag).
// Si squadLoader est nil, retourne une map vide → mode solo dans le briefing.
func (s *TeammatesService) loadTeammatesCanonicalParallel(
	ctx context.Context,
	gamertags []string,
) (map[string][]canonical.PlayerMatchRow, error) {
	if s.squadLoader == nil {
		return map[string][]canonical.PlayerMatchRow{}, nil
	}
	g, gctx := errgroup.WithContext(ctx)
	var mu sync.Mutex
	out := make(map[string][]canonical.PlayerMatchRow, len(gamertags))
	for _, gt := range gamertags {
		gt := gt
		g.Go(func() error {
			rows, err := s.squadLoader.LoadFor(gctx, s.titleSlug, gt, port.PlayerMatchFilters{})
			if err != nil {
				if errors.Is(err, games.ErrCapabilityNotSupported) {
					return nil
				}
				return fmt.Errorf("LoadFor(%s): %w", gt, err)
			}
			mu.Lock()
			out[gt] = rows
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

// filterTopRowsToFriends garde uniquement les lignes dont le gamertag matche
// (case-insensitive, trim) un ami de friendGamertags. Liste vide = aucune
// ligne (le user doit ajouter des amis pour peupler le dropdown).
func filterTopRowsToFriends(rows []domain.TopTeammateRow, friendGamertags []string) []domain.TopTeammateRow {
	if len(friendGamertags) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(friendGamertags))
	for _, gt := range friendGamertags {
		k := strings.ToLower(strings.TrimSpace(gt))
		if k != "" {
			allowed[k] = struct{}{}
		}
	}
	out := make([]domain.TopTeammateRow, 0, len(rows))
	for _, r := range rows {
		if _, ok := allowed[strings.ToLower(r.Gamertag)]; ok {
			out = append(out, r)
		}
	}
	return out
}

// extractSynthesisSessionLabels collecte les sessions uniques en sÃƒÂ©parant solo / escouade,
// calcule les bornes temporelles, agrÃƒÂ¨ge les expÃƒÂ©riences et playlists prÃƒÂ©sentes, et trie par StartedAt DESC.
func extractSynthesisSessionLabels(matches []legacymatch.SynthesisMatchRow) domain.SessionLabelsList {
	type meta struct {
		startedAt   time.Time
		endedAt     time.Time
		experiences map[string]struct{}
		playlists   map[string]struct{}
	}
	soloMap := map[string]*meta{}
	squadMap := map[string]*meta{}

	for _, m := range matches {
		if m.SessionLabel == nil || *m.SessionLabel == "" {
			continue
		}
		label := *m.SessionLabel
		t := m.StartTime
		var em map[string]*meta
		if m.IsWithFriends {
			em = squadMap
		} else {
			em = soloMap
		}
		entry, ok := em[label]
		if !ok {
			entry = &meta{
				startedAt:   t,
				endedAt:     t,
				experiences: map[string]struct{}{},
				playlists:   map[string]struct{}{},
			}
			em[label] = entry
		}
		if t.Before(entry.startedAt) {
			entry.startedAt = t
		}
		if t.After(entry.endedAt) {
			entry.endedAt = t
		}
		entry.experiences[synthesisExperienceLabel(m)] = struct{}{}
		if m.PlaylistName != "" {
			entry.playlists[m.PlaylistName] = struct{}{}
		}
	}

	toSlice := func(m map[string]*meta) []domain.SessionLabelEntry {
		out := make([]domain.SessionLabelEntry, 0, len(m))
		for label, entry := range m {
			exps := make([]string, 0, len(entry.experiences))
			for e := range entry.experiences {
				exps = append(exps, e)
			}
			slices.Sort(exps)
			pls := make([]string, 0, len(entry.playlists))
			for p := range entry.playlists {
				pls = append(pls, p)
			}
			slices.Sort(pls)
			out = append(out, domain.SessionLabelEntry{
				Label:       label,
				StartedAt:   entry.startedAt,
				EndedAt:     entry.endedAt,
				Experiences: exps,
				Playlists:   pls,
			})
		}
		slices.SortFunc(out, func(a, b domain.SessionLabelEntry) int {
			return cmp.Compare(b.StartedAt.Unix(), a.StartedAt.Unix())
		})
		return out
	}

	return domain.SessionLabelsList{
		Solo:  toSlice(soloMap),
		Squad: toSlice(squadMap),
	}
}

// synthesisExperienceLabel dÃƒÂ©rive le label d'expÃƒÂ©rience d'un match (miroir de filters_service.go).
func synthesisExperienceLabel(m legacymatch.SynthesisMatchRow) string {
	if m.IsFirefight {
		return "PVE"
	}
	if m.IsRanked {
		return "PVP classé"
	}
	return "PVP non classé"
}

// filterSynthesisByCascade applique les filtres experience_types et playlists sur les matchs.
func filterSynthesisByCascade(matches []legacymatch.SynthesisMatchRow, c domain.CascadeFilter) []legacymatch.SynthesisMatchRow {
	if len(c.ExperienceTypes) == 0 && len(c.Playlists) == 0 {
		return matches
	}
	expSet := make(map[string]struct{}, len(c.ExperienceTypes))
	for _, e := range c.ExperienceTypes {
		expSet[e] = struct{}{}
	}
	plSet := make(map[string]struct{}, len(c.Playlists))
	for _, p := range c.Playlists {
		plSet[p] = struct{}{}
	}
	out := matches[:0:0]
	for _, m := range matches {
		if len(expSet) > 0 {
			if _, ok := expSet[synthesisExperienceLabel(m)]; !ok {
				continue
			}
		}
		if len(plSet) > 0 {
			if _, ok := plSet[m.PlaylistName]; !ok {
				continue
			}
		}
		out = append(out, m)
	}
	return out
}

// filterSynthesisBySession filtre les matchs selon les sessions sÃƒÂ©lectionnÃƒÂ©es (union des labels).
// Slices vides Ã¢â€ â€™ tous les matchs retournÃƒÂ©s sans filtre.
func filterSynthesisBySession(
	matches []legacymatch.SynthesisMatchRow,
	pickedSolo []string,
	pickedSquad []string,
) []legacymatch.SynthesisMatchRow {
	if len(pickedSolo) == 0 && len(pickedSquad) == 0 {
		return matches
	}
	filtered := make([]legacymatch.SynthesisMatchRow, 0, len(matches))
	for _, m := range matches {
		label := ""
		if m.SessionLabel != nil {
			label = *m.SessionLabel
		}
		if len(pickedSolo) > 0 && !m.IsWithFriends && slices.Contains(pickedSolo, label) {
			filtered = append(filtered, m)
			continue
		}
		if len(pickedSquad) > 0 && m.IsWithFriends && slices.Contains(pickedSquad, label) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// buildTeammateOptions convertit les TopTeammateRow en TeammateOption.
func buildTeammateOptions(rows []domain.TopTeammateRow) []domain.TeammateOption {
	opts := make([]domain.TeammateOption, 0, len(rows))
	for _, r := range rows {
		xuid := r.XUID
		opts = append(opts, domain.TeammateOption{
			Gamertag:       r.Gamertag,
			XUID:           &xuid,
			EncounterCount: r.GamesTogether,
		})
	}
	return opts
}

// buildTeammateRowWithMatches construit les KPIs avec/sans pour un coéquipier.
// Retourne (row, filteredMatches, allMatches, error) :
//   - filteredMatches : matchs restreints à la session active (utilisés pour KPIs, history, breakdown)
//   - allMatches      : tous les matchs communs sans filtre de session (utilisés pour la timeline)
func (s *TeammatesService) buildTeammateRowWithMatches(
	ctx context.Context,
	playerXUID, gamertag string,
	topRows []domain.TopTeammateRow,
	allMatches []legacymatch.SynthesisMatchRow,
	sessionMatchIDs map[string]bool,
) (*domain.TeammateRow, []domain.SquadMatchRow, []domain.SquadMatchRow, error) {
	// Ãƒâ€°tape 1 : chercher le gamertag dans le top 50 escouade Ã¢â‚¬â€ case-insensitive
	// pour absorber les variations de casse entre la saisie user et la valeur en
	// DB (Halo API renvoie tantÃƒÂ´t "Madina97294" tantÃƒÂ´t "madina97294").
	var teammateXUID string
	var encounterCount int
	for _, r := range topRows {
		if strings.EqualFold(r.Gamertag, gamertag) {
			teammateXUID = r.XUID
			encounterCount = r.GamesTogether
			break
		}
	}

	// Ãƒâ€°tape 2 : fallback Ã¢â‚¬â€ rÃƒÂ©soudre via global.xuid_aliases pour les gamertags
	// hors top 50 (utilisateur qui a 50+ coÃƒÂ©quipiers rÃƒÂ©guliers OU saisie libre
	// dans la combobox). encounterCount reste 0 Ã¢â‚¬â€ recalculÃƒÂ© depuis squadMatches
	// plus bas si on charge effectivement les matchs.
	if teammateXUID == "" {
		resolved, found, err := s.repo.LookupXUIDByGamertag(ctx, gamertag)
		if err != nil {
			slog.WarnContext(ctx, "teammates_gamertag_lookup_failed",
				"player_xuid", playerXUID,
				"gamertag", gamertag,
				"err", err.Error(),
			)
			return nil, nil, nil, nil
		}
		if !found {
			// Vraiment inconnu de tous les aliases Ã¢â‚¬â€ on log et on drop.
			slog.WarnContext(ctx, "teammates_gamertag_not_found",
				"player_xuid", playerXUID,
				"gamertag", gamertag,
				"top_rows_count", len(topRows),
			)
			return nil, nil, nil, nil
		}
		teammateXUID = resolved
	}

	// Charger tous les matchs communs (sans filtre de session — nécessaire pour la timeline).
	allSquadMatches, err := s.repo.LoadSquadMatches(ctx, playerXUID, teammateXUID)
	if err != nil {
		slog.ErrorContext(ctx, "teammates_load_squad_matches_failed",
			"player_xuid", playerXUID, "teammate_xuid", teammateXUID,
			"gamertag", gamertag, "err", err.Error())
		return nil, nil, nil, fmt.Errorf("buildTeammateRowWithMatches LoadSquadMatches: %w", err)
	}

	// Restreindre aux matchs de la session sélectionnée pour les KPIs/historique.
	squadMatches := allSquadMatches
	if len(sessionMatchIDs) > 0 {
		filtered := make([]domain.SquadMatchRow, 0, len(allSquadMatches))
		for _, m := range allSquadMatches {
			if sessionMatchIDs[m.MatchID] {
				filtered = append(filtered, m)
			}
		}
		squadMatches = filtered
	}

	withKPIs := computeKPIsFromSquadMatches(squadMatches)

	// KPIs "sans" = matchs qui ne sont PAS dans les matchs communs.
	commonIDs := make(map[string]bool, len(squadMatches))
	for _, m := range squadMatches {
		commonIDs[m.MatchID] = true
	}
	withoutKPIs := computeKPIsFromSynthesisExcluding(allMatches, commonIDs)

	xuid := teammateXUID
	var lastSeen *time.Time
	if len(squadMatches) > 0 {
		t := squadMatches[0].StartTime
		for _, m := range squadMatches {
			if m.StartTime.After(t) {
				t = m.StartTime
			}
		}
		lastSeen = &t
	}

	if encounterCount == 0 {
		encounterCount = len(allSquadMatches)
	}

	return &domain.TeammateRow{
		Gamertag:       gamertag,
		XUID:           &xuid,
		EncounterCount: encounterCount,
		LastSeenAt:     lastSeen,
		WithKPIs:       withKPIs,
		WithoutKPIs:    &withoutKPIs,
	}, squadMatches, allSquadMatches, nil
}

// computeKPIsFromSquadMatches calcule les KPIs depuis les matchs communs.
func computeKPIsFromSquadMatches(matches []domain.SquadMatchRow) domain.TeammateKPIs {
	n := len(matches)
	if n == 0 {
		return domain.TeammateKPIs{}
	}
	wins := 0
	totalKills, totalDeaths, totalAssists := 0, 0, 0
	totalHS, totalPK := 0, 0
	accSum, accCount := 0.0, 0
	for _, m := range matches {
		if m.Outcome == analysis.OutcomeWin {
			wins++
		}
		totalKills += m.Kills
		totalDeaths += m.Deaths
		totalAssists += m.Assists
		totalHS += m.HeadshotKills
		totalPK += m.PerfectKills
		if m.Accuracy != nil {
			accSum += *m.Accuracy
			accCount++
		}
	}
	kd := safeDiv(float64(totalKills), float64(totalDeaths))
	kpg := float64(totalKills) / float64(n)
	apg := float64(totalAssists) / float64(n)
	hspg := float64(totalHS) / float64(n)
	pkpg := float64(totalPK) / float64(n)
	var acc *float64
	if accCount > 0 {
		v := round2(accSum / float64(accCount) * 100)
		acc = &v
	}
	return domain.TeammateKPIs{
		MatchCount:           n,
		Wins:                 wins,
		KDRatio:              &kd,
		WinRate:              analysis.WinRate(wins, n),
		Accuracy:             acc,
		KillsPerGame:         &kpg,
		AssistsPerGame:       &apg,
		HeadshotKillsPerGame: &hspg,
		PerfectKillsPerGame:  &pkpg,
	}
}

// computeKPIsFromSynthesisExcluding calcule les KPIs en excluant certains matchs.
func computeKPIsFromSynthesisExcluding(
	matches []legacymatch.SynthesisMatchRow,
	exclude map[string]bool,
) domain.TeammateKPIs {
	var filtered []legacymatch.SynthesisMatchRow
	for _, m := range matches {
		if !exclude[m.MatchID] {
			filtered = append(filtered, m)
		}
	}
	n := len(filtered)
	if n == 0 {
		return domain.TeammateKPIs{}
	}
	wins := 0
	totalKills, totalDeaths := 0, 0
	for _, m := range filtered {
		if m.Outcome == analysis.OutcomeWin {
			wins++
		}
		totalKills += m.Kills
		totalDeaths += m.Deaths
	}
	kd := safeDiv(float64(totalKills), float64(totalDeaths))
	kpg := float64(totalKills) / float64(n)
	return domain.TeammateKPIs{
		MatchCount:   n,
		Wins:         wins,
		KDRatio:      &kd,
		WinRate:      analysis.WinRate(wins, n),
		KillsPerGame: &kpg,
	}
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return a
	}
	return round2(a / b)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// computeHistoricalMapWRByLabel calcule le win rate par carte sur TOUT l'historique
// du joueur principal (canonicalRows non filtrés). Clé = DefaultLabel (= map_name SQL).
func computeHistoricalMapWRByLabel(rows []canonical.PlayerMatchRow) map[string]float64 {
	type stats struct{ count, wins int }
	m := map[string]*stats{}
	for _, r := range rows {
		if r.Summary.Map == nil {
			continue
		}
		key := r.Summary.Map.ID
		if key == "" {
			key = r.Summary.Map.DefaultLabel
		}
		if key == "" {
			continue
		}
		if _, ok := m[key]; !ok {
			m[key] = &stats{}
		}
		m[key].count++
		if r.Self.Outcome == canonical.OutcomeWin {
			m[key].wins++
		}
	}
	result := make(map[string]float64, len(m))
	for k, s := range m {
		if s.count > 0 {
			result[k] = round2(float64(s.wins) / float64(s.count))
		}
	}
	return result
}

// enrichMapBreakdownWithHistory injecte HistoricalWinRate depuis la map d'historique.
func enrichMapBreakdownWithHistory(rows []domain.MapBreakdownRow, historical map[string]float64) []domain.MapBreakdownRow {
	for i := range rows {
		key := rows[i].MapID
		if key == "" {
			key = rows[i].MapUI
		}
		if wr, ok := historical[key]; ok {
			v := wr
			rows[i].HistoricalWinRate = &v
		}
	}
	return rows
}

// enrichMapBreakdownWithHistoricalPerf injecte HistoricalPerformanceAvg depuis
// la map d'historique perf (alimentée par computeHistoricalMapPerfByLabel).
func enrichMapBreakdownWithHistoricalPerf(rows []domain.MapBreakdownRow, historical map[string]float64) []domain.MapBreakdownRow {
	for i := range rows {
		key := rows[i].MapID
		if key == "" {
			key = rows[i].MapUI
		}
		if perf, ok := historical[key]; ok {
			v := perf
			rows[i].HistoricalPerformanceAvg = &v
		}
	}
	return rows
}

// enrichSquadMatchAssets enrichit MapUI et PlaylistName des rows avec les traductions FR
// depuis metadata.asset_translations (calqué sur home_repo.enrichHomeMatchTranslations).
func enrichSquadMatchAssets(ctx context.Context, repo port.SquadRepository, rows []domain.SquadMatchRow) {
	mapIDs := collectUniqueIDs(rows, func(r domain.SquadMatchRow) string { return r.MapID })
	playlistIDs := collectUniqueIDs(rows, func(r domain.SquadMatchRow) string { return r.PlaylistID })

	mapFR, err := repo.LoadAssetTranslationsFR(ctx, "map", mapIDs)
	if err != nil {
		slog.WarnContext(ctx, "teammates: LoadAssetTranslationsFR map failed", "err", err)
	}
	playlistFR, err := repo.LoadAssetTranslationsFR(ctx, "playlist", playlistIDs)
	if err != nil {
		slog.WarnContext(ctx, "teammates: LoadAssetTranslationsFR playlist failed", "err", err)
	}

	for i := range rows {
		if fr := strings.TrimSpace(mapFR[rows[i].MapID]); fr != "" {
			rows[i].MapUI = fr
		}
		if fr := strings.TrimSpace(playlistFR[rows[i].PlaylistID]); fr != "" {
			rows[i].PlaylistName = fr
		}
	}
}

func collectUniqueIDs(rows []domain.SquadMatchRow, idOf func(domain.SquadMatchRow) string) []string {
	seen := make(map[string]struct{}, len(rows))
	result := make([]string, 0, len(rows))
	for _, r := range rows {
		id := idOf(r)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}

// modeLabel retourne le label FR du mode si disponible dans modeFR, sinon le label EN normalisé.
func modeLabel(pairName, mapUI string, modeFR map[string]string) string {
	en := analysis.NormalizeModeLabel(pairName, mapUI)
	if fr, ok := modeFR[en]; ok && fr != "" {
		return fr
	}
	return en
}

// computeMapBreakdown agrège les stats par carte depuis les matchs escouade.
// PerformanceAvg = moyenne des PerformanceScore non nil ; nil si aucun.
func computeMapBreakdown(matches []domain.SquadMatchRow) []domain.MapBreakdownRow {
	type stats struct {
		mapUI       string
		count, wins int
		perfSum     float64
		perfCount   int
	}
	m := map[string]*stats{}
	for _, r := range matches {
		// Clé interne = UUID si dispo (language-agnostic), sinon label d'affichage.
		key := r.MapID
		if key == "" {
			key = r.MapUI
		}
		if key == "" {
			key = "Unknown"
		}
		if _, ok := m[key]; !ok {
			lbl := r.MapUI
			if lbl == "" {
				lbl = "Unknown"
			}
			m[key] = &stats{mapUI: lbl}
		}
		m[key].count++
		if r.Outcome == analysis.OutcomeWin {
			m[key].wins++
		}
		if r.PerformanceScore != nil {
			m[key].perfSum += *r.PerformanceScore
			m[key].perfCount++
		}
	}
	result := make([]domain.MapBreakdownRow, 0, len(m))
	for mapKey, s := range m {
		row := domain.MapBreakdownRow{
			MapID:      mapKey,
			MapUI:      s.mapUI,
			MatchCount: s.count,
			WinRate:    round2(float64(s.wins) / float64(s.count)),
		}
		if s.perfCount > 0 {
			avg := round2(s.perfSum / float64(s.perfCount))
			row.PerformanceAvg = &avg
		}
		result = append(result, row)
	}
	return result
}

// computeHistoricalMapPerfByLabel calcule la moyenne de performance_score
// par carte sur TOUT l'historique du joueur principal (canonicalRows non
// filtrés). Clé = DefaultLabel (= MapUI côté SquadMatchRow). Carte ignorée
// si aucun match avec score.
func computeHistoricalMapPerfByLabel(rows []canonical.PlayerMatchRow) map[string]float64 {
	type stats struct {
		sum   float64
		count int
	}
	m := map[string]*stats{}
	for _, r := range rows {
		if r.Summary.Map == nil {
			continue
		}
		if r.Enrichment.PerformanceScore == nil {
			continue
		}
		key := r.Summary.Map.ID
		if key == "" {
			key = r.Summary.Map.DefaultLabel
		}
		if key == "" {
			continue
		}
		if _, ok := m[key]; !ok {
			m[key] = &stats{}
		}
		m[key].sum += *r.Enrichment.PerformanceScore
		m[key].count++
	}
	result := make(map[string]float64, len(m))
	for k, s := range m {
		if s.count > 0 {
			result[k] = round2(s.sum / float64(s.count))
		}
	}
	return result
}

// collectModeENs retourne les noms de modes EN normalisés uniques depuis les matchs squad.
// Utilisé pour le batch-lookup mode_name_tr FR.
func collectModeENs(matches []domain.SquadMatchRow) []string {
	seen := make(map[string]struct{}, 16)
	result := make([]string, 0, 16)
	for _, m := range matches {
		en := analysis.NormalizeModeLabel(m.PairName, m.MapUI)
		if en == "" {
			continue
		}
		if _, ok := seen[en]; !ok {
			seen[en] = struct{}{}
			result = append(result, en)
		}
	}
	return result
}

// buildSquadMatchHistory construit la table historique pour teammates.11 :
// une ligne par match unique, triée par StartTime DESC. Pas de cap serveur —
// la pagination (20/page) est gérée côté client (TanStack Table).
func buildSquadMatchHistory(matches []domain.SquadMatchRow, modeFR map[string]string) []domain.SquadMatchHistoryRow {
	seen := make(map[string]struct{}, len(matches))
	rows := make([]domain.SquadMatchHistoryRow, 0, len(matches))
	for _, m := range matches {
		if m.MatchID == "" {
			continue
		}
		if _, dup := seen[m.MatchID]; dup {
			continue
		}
		seen[m.MatchID] = struct{}{}
		var deltaMMR *float64
		if m.EnemyMMR != nil {
			d := m.TeamMMR - *m.EnemyMMR
			deltaMMR = &d
		}
		var scoreLabel string
		if m.MyTeamScore != nil && m.EnemyTeamScore != nil {
			scoreLabel = fmt.Sprintf("%d - %d", *m.MyTeamScore, *m.EnemyTeamScore)
		}
		rows = append(rows, domain.SquadMatchHistoryRow{
			MatchID:          m.MatchID,
			StartTime:        m.StartTime.Format("2006-01-02T15:04:05Z"),
			MapUI:            m.MapUI,
			PlaylistName:     m.PlaylistName,
			PairName:         m.PairName,
			ModeUI:           modeLabel(m.PairName, m.MapUI, modeFR),
			Outcome:          m.Outcome,
			Kills:            m.Kills,
			Deaths:           m.Deaths,
			Assists:          m.Assists,
			Accuracy:         m.Accuracy,
			PerformanceScore: m.PerformanceScore,
			TeamMMRAvg:       m.TeamMMR,
			EnemyMMRAvg:      m.EnemyMMR,
			DeltaMMR:         deltaMMR,
			ScoreLabel:       scoreLabel,
			SessionLabel:     m.SessionLabel,
		})
	}
	slices.SortFunc(rows, func(a, b domain.SquadMatchHistoryRow) int {
		return cmp.Compare(b.StartTime, a.StartTime) // DESC
	})
	return rows
}

// buildMatchSeries construit la sÃƒÂ©rie temporelle des matchs pour un coÃƒÂ©quipier.
func buildMatchSeries(matches []domain.SquadMatchRow) []domain.SquadMatchSeriesPoint {
	series := make([]domain.SquadMatchSeriesPoint, 0, len(matches))
	for _, m := range matches {
		series = append(series, domain.SquadMatchSeriesPoint{
			MatchID:          m.MatchID,
			StartTime:        m.StartTime.Format("2006-01-02T15:04:05Z"),
			Outcome:          m.Outcome,
			PerformanceScore: m.PerformanceScore,
			TeamMMRAvg:       m.TeamMMR,
			SessionLabel:     m.SessionLabel,
		})
	}
	return series
}
