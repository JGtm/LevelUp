// Package service â€” SessionPageService : page dÃ©tail de session avec suggestion de comparaison.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// errCodeSessionNotFound est le code machine renvoyé quand une session
// explicitement demandée est introuvable (ADR 0029, Couche B).
const errCodeSessionNotFound = "session_not_found"

// SessionPageService construit la page de dÃ©tail d'une session.
type SessionPageService struct {
	statsRepo port.StatsRepository
	// playerMatchesRepo (P4.1, ADR 0011) : loader canonical-aware optionnel.
	// TODO P4.3 : retirer le converter quand les analyses session_page
	// (extractSessionLabels, buildCompareEntry, buildSessionDetailRows, etc.)
	// consommeront canonical.
	playerMatchesRepo port.PlayerMatchesRepository
	titleSlug         string
	gamertag          string
	// weaponKillsRepo (P5) : loader weapon_kills agrégé pour la répartition des frags
	// (sunburst v2) de la session. Optionnel — nil → FragDistribution best-effort
	// (classes API servies, ventilation gun retombant dans « Non attribué »).
	weaponKillsRepo port.WeaponKillsRepository
	// weaponAccuracyRepo : loader weapon_accuracy agrégé pour le graphe « Précision
	// par arme » de la session (Halo 5 natif, MIROIR de weaponKillsRepo). Optionnel —
	// nil ou capability absente (Infinite) → WeaponAccuracy best-effort nil (le front
	// retombe sur « Détails des frags »).
	weaponAccuracyRepo port.WeaponAccuracyRepository
	// csrThreshold (optionnel) : résolveur season_id → seuil placement CSR (5 ou 10).
	// Sans lui, applyMatchPlacements retombe sur le défaut (5). Cf. match_history_placement.go.
	csrThreshold CSRThresholdResolver
	// expectedAssistsModels / expectedAssistsCoefs (optionnels) : résolution des
	// assists attendus (chaîne personnel → populationnel) pour l'écart cumulé au FDA
	// attendu. nil → AssistsExpected nil (l'attendu dégrade en K/D pur).
	expectedAssistsModels assistsModelReader
	expectedAssistsCoefs  assistsCoefReader
}

// NewSessionPageService crÃ©e un SessionPageService.
func NewSessionPageService(statsRepo port.StatsRepository) *SessionPageService {
	return &SessionPageService{statsRepo: statsRepo}
}

// WithPlayerMatchesRepo (P4.1, ADR 0011) injecte le loader canonical-aware.
func (s *SessionPageService) WithPlayerMatchesRepo(repo port.PlayerMatchesRepository, titleSlug, gamertag string) *SessionPageService {
	s.playerMatchesRepo = repo
	s.titleSlug = titleSlug
	s.gamertag = gamertag
	return s
}

// WithCSRThresholds injecte le résolveur de seuil placement CSR (cf. l'Explorer /
// match-history). Permet à la colonne "Rang" d'afficher "X/Y" en phase de placement.
func (s *SessionPageService) WithCSRThresholds(resolver CSRThresholdResolver) *SessionPageService {
	s.csrThreshold = resolver
	return s
}

// WithWeaponKillsRepo injecte le loader weapon_kills (P5) alimentant la répartition
// hiérarchique des frags (sunburst v2) par session. Optionnel.
func (s *SessionPageService) WithWeaponKillsRepo(repo port.WeaponKillsRepository) *SessionPageService {
	s.weaponKillsRepo = repo
	return s
}

// WithWeaponAccuracyRepo injecte le loader weapon_accuracy alimentant le graphe
// « Précision par arme » de la session (Halo 5 natif, MIROIR de WithWeaponKillsRepo).
// Optionnel — nil / capability absente (Infinite) → WeaponAccuracy best-effort nil.
func (s *SessionPageService) WithWeaponAccuracyRepo(repo port.WeaponAccuracyRepository) *SessionPageService {
	s.weaponAccuracyRepo = repo
	return s
}

// WithExpectedAssists injecte les résolveurs du modèle d'assists attendus (personnel
// via player DB + populationnel via metadata) pour l'écart cumulé au FDA attendu de
// la session. Optionnel — nil → AssistsExpected nil (l'attendu dégrade en K/D pur).
func (s *SessionPageService) WithExpectedAssists(models assistsModelReader, coefs assistsCoefReader) *SessionPageService {
	s.expectedAssistsModels = models
	s.expectedAssistsCoefs = coefs
	return s
}

// GetPage retourne la page dÃ©tail d'une session avec suggestion de comparaison.
func (s *SessionPageService) GetPage(
	ctx context.Context,
	req domain.SessionPageRequest,
) (domain.SessionPageResponse, error) {
	if err := req.Validate(); err != nil {
		return domain.SessionPageResponse{}, fmt.Errorf("SessionPageService.GetPage validate: %w", err)
	}

	// P4.3 finale (ADR 0011) : path canonical exclusif.
	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		return domain.SessionPageResponse{}, fmt.Errorf("SessionPageService: PlayerMatchesRepo non cÃ¢blÃ© (P4.3 finale exige le wiring DI)")
	}
	canonicalRows, err := s.playerMatchesRepo.LoadPlayerMatches(
		ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
	)
	if err != nil {
		return domain.SessionPageResponse{}, fmt.Errorf("SessionPageService.GetPage: %w", err)
	}
	hp := games.EffectiveHpToKill(s.titleSlug)
	rows := analysis.StatsMatchRowsFromCanonical(canonicalRows, hp)
	// Placement X/Y : calculé sur TOUS les matchs (le LUSR a besoin de l'ordre global
	// par chaîne) ; appliqué ensuite par match dans buildSessionDetailRows.
	placements := s.computeSessionPlacements(ctx, rows)

	filtered := filterStatsMatchRows(rows, req.Filters)

	// Deep-link (ADR 0029, Couche B) : une session explicitement demandée doit
	// exister dans le périmètre filtré (TOUTES sessions, mono-match incluses) —
	// sinon 404 explicite au lieu d'une page vide 200 trompeuse.
	allLabels := extractSessionLabels(filtered)
	if lbl := derefString(req.SessionLabel); lbl != "" && indexOfSessionLabel(allLabels, lbl) == -1 {
		slog.InfoContext(ctx, "session page: session demandée introuvable", "requested_session", lbl)
		return domain.SessionPageResponse{}, &domain.APIError{
			Code:    errCodeSessionNotFound,
			Message: "session introuvable : " + lbl,
		}
	}

	// Sessions d'un seul match exclues de la liste/navigation (cf. minListedSessionMatches).
	// Comptage sur `rows` (historique complet) = taille BRUTE de la session : une session de
	// 2 matchs resserrée à 1 par le filtre période reste listée. Un deep-link vers une session
	// d'un seul match reste résoluble via req.SessionLabel (lastOrNil court-circuite la liste).
	labels := keepMultiMatchSessionLabels(allLabels, rows)
	if len(labels) == 0 {
		slog.InfoContext(ctx, "session page: no sessions after filtering")
		return domain.SessionPageResponse{
			AvailableSessions: []string{},
			Matches:           []domain.SessionDetailMatchRow{},
			CompareMatches:    []domain.SessionDetailMatchRow{},
			CompareMetrics:    []domain.SessionCompareMetricRow{},
		}, nil
	}

	provideSpree := games.ProvidesMaxKillingSpree(s.titleSlug)
	currentLabel := lastOrNil(labels, req.SessionLabel)
	currentMatches := filterBySession(filtered, currentLabel)
	currentEntry := buildCompareEntryWithObjectives(currentMatches, currentLabel, s.objectiveScores(ctx, currentMatches), hp, provideSpree)
	if currentEntry == nil {
		slog.WarnContext(ctx, "session page: current session not found after filtering",
			"requested_session", derefString(req.SessionLabel),
			"resolved_session", currentLabel,
		)
		return domain.SessionPageResponse{
			AvailableSessions: labels,
			Matches:           []domain.SessionDetailMatchRow{},
			CompareMatches:    []domain.SessionDetailMatchRow{},
			CompareMetrics:    []domain.SessionCompareMetricRow{},
		}, nil
	}

	// Vivier de comparaison ÉLARGI : sessions de même catégorie, hors isolation
	// période/session (cf. comparePoolFilters). Le bouton Comparer, le dropdown et
	// la suggestion s'appuient dessus → ils restent disponibles même quand le filtre
	// L2 a été resserré sur UNE seule session pour la vue principale.
	compareScope := filterStatsMatchRows(rows, comparePoolFilters(req.Filters))
	compareLabels := keepMultiMatchSessionLabels(extractSessionLabels(compareScope), rows)
	suggestion, candidateCount := buildSessionCompareSuggestion(compareLabels, currentLabel, compareScope)
	compareLabel := resolveRequestedCompareLabel(req, suggestion)
	compareEnabled := req.EnableCompare && compareLabel != "" && compareLabel != currentLabel

	// Navigation prev/next : reste sur le périmètre resserré (la vue principale).
	prevLabel, nextLabel := neighboringSessionLabels(labels, currentLabel)

	// Assists attendus (is_me) résolus une fois par mode — écart cumulé au FDA attendu
	// (le cumul est fait côté front sur les rows de la session courante).
	currentAssistsExpected := computeExpectedAssistsBatch(ctx, s.expectedAssistsModels, s.expectedAssistsCoefs, currentMatches)

	resp := domain.SessionPageResponse{
		CurrentSession:       currentEntry,
		AvailableSessions:    compareLabels,
		Matches:              buildSessionDetailRows(currentMatches, currentEntry.DominantCategory, req.Locale, currentAssistsExpected),
		SuggestedCompare:     suggestion,
		CompareEnabled:       compareEnabled,
		CompareMatches:       []domain.SessionDetailMatchRow{},
		CompareMetrics:       []domain.SessionCompareMetricRow{},
		PreviousSessionLabel: prevLabel,
		NextSessionLabel:     nextLabel,
	}

	// Répartition des frags (sunburst v2) de la session courante — nouveau chemin de
	// données P5 (weapon_kills + compteurs kill-type canoniques du scope session).
	s.attachSessionFragDistribution(ctx, resp.CurrentSession, canonicalRows, matchIDsFromStatsRows(currentMatches))

	if compareEnabled {
		// Matchs de la session comparée : depuis le vivier élargi, pour qu'une session
		// hors du filtre resserré ait bien ses matchs (sinon filterBySession sur le
		// périmètre resserré renverrait vide).
		compareMatches := filterBySession(compareScope, compareLabel)
		resp.CompareSession = buildCompareEntryWithObjectives(compareMatches, compareLabel, s.objectiveScores(ctx, compareMatches), hp, provideSpree)
		if resp.CompareSession != nil {
			resp.CompareMetrics = buildCompareMetrics(currentMatches, compareMatches)
			resp.CompareMatches = buildSessionDetailRows(compareMatches, resp.CompareSession.DominantCategory, req.Locale,
				computeExpectedAssistsBatch(ctx, s.expectedAssistsModels, s.expectedAssistsCoefs, compareMatches))
			s.attachSessionFragDistribution(ctx, resp.CompareSession, canonicalRows, matchIDsFromStatsRows(compareMatches))
		} else {
			resp.CompareEnabled = false
			slog.WarnContext(ctx, "session page: compare session missing after filtering",
				"current_session", currentLabel,
				"compare_session", compareLabel,
			)
		}
	}

	// Placement X/Y dans la colonne Rang (matchs de placement) — appliqué aux deux
	// tableaux (session + comparée), comme l'Explorer.
	applyPlacementsToRows(resp.Matches, placements)
	applyPlacementsToRows(resp.CompareMatches, placements)

	// Taille de lobby (joueurs présents à la fin, bots inclus) pour le breakdown
	// des placements — best-effort, dégrade gracieusement si le repo ne le fournit pas.
	s.attachLobbySizes(ctx, resp.Matches, resp.CompareMatches)

	slog.InfoContext(ctx, "session page generated",
		"resolved_session", currentLabel,
		"view_sessions", len(labels),
		"compare_pool_sessions", len(compareLabels),
		"current_match_count", len(currentMatches),
		"suggestion", suggestionLabel(suggestion),
		"suggestion_candidates", candidateCount,
		"compare_enabled", resp.CompareEnabled,
		"compare_session", compareLabel,
		"compare_match_count", len(resp.CompareMatches),
		"previous_session_label", derefString(prevLabel),
		"next_session_label", derefString(nextLabel),
	)

	return resp, nil
}

// lobbySizeProvider est une capability OPTIONNELLE du repo de matchs : compte les
// participants présents à la fin (present_at_completion) par match. Seul l'adapter
// DuckDB réel l'implémente ; les mocks de test ne l'implémentent pas → l'axe du
// breakdown de placements dégrade alors sur max(placement observé) côté front.
type lobbySizeProvider interface {
	LobbySizesAtCompletion(ctx context.Context, slug string, matchIDs []string) (map[string]int, error)
}

// objectiveScoreProvider est une capability OPTIONNELLE : somme des scores PSA
// "objective" par match (table personal_score_awards). Alimente les axes
// Objective/Score du profil de participation. Seul l'adapter DuckDB réel
// l'implémente ; sans elle (mocks) les axes dégradent à 0 / résiduel sans objectif.
type objectiveScoreProvider interface {
	ObjectiveScores(ctx context.Context, slug string, matchIDs []string) (map[string]int, error)
}

// objectiveScores récupère les scores PSA "objective" des matchs fournis via le
// provider optionnel. nil si le repo ne fournit pas la capability ou en cas d'erreur.
func (s *SessionPageService) objectiveScores(ctx context.Context, matches []legacymatch.StatsMatchRow) map[string]int {
	provider, ok := s.playerMatchesRepo.(objectiveScoreProvider)
	if !ok || len(matches) == 0 {
		return nil
	}
	ids := make([]string, 0, len(matches))
	for i := range matches {
		ids = append(ids, matches[i].MatchID)
	}
	scores, err := provider.ObjectiveScores(ctx, s.titleSlug, ids)
	if err != nil {
		slog.WarnContext(ctx, "session page: objective scores unavailable", "err", err)
		return nil
	}
	return scores
}

// attachLobbySizes renseigne LobbySize sur chaque row à partir du provider optionnel.
// Best-effort : no-op si le repo ne fournit pas la capability ou si la requête échoue.
func (s *SessionPageService) attachLobbySizes(ctx context.Context, rowSets ...[]domain.SessionDetailMatchRow) {
	provider, ok := s.playerMatchesRepo.(lobbySizeProvider)
	if !ok {
		return
	}
	idSet := make(map[string]struct{})
	for _, rows := range rowSets {
		for i := range rows {
			idSet[rows[i].MatchID] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	sizes, err := provider.LobbySizesAtCompletion(ctx, s.titleSlug, ids)
	if err != nil {
		slog.WarnContext(ctx, "session page: lobby sizes unavailable", "err", err)
		return
	}
	for _, rows := range rowSets {
		for i := range rows {
			if n, ok := sizes[rows[i].MatchID]; ok && n > 0 {
				v := n
				rows[i].LobbySize = &v
			}
		}
	}
}

type sessionCandidate struct {
	Label    string
	Category string
	IsRanked bool
	// IsSquad : composition approximée (majorité de matchs avec des amis). On
	// préfère comparer une session escouade à une autre escouade (et solo↔solo).
	IsSquad bool
	Count   int
	Index   int
}

// comparePoolFilters dérive le périmètre du VIVIER de comparaison à partir des
// filtres de la page. On conserve les filtres de CATÉGORIE (cascade maps/modes/
// playlists + match_context solo/squad) mais on neutralise ce qui ISOLE une session
// (période + sélection de session). Sans ça, resserrer le filtre L2 pour afficher
// UNE session vide le vivier → le bouton Comparer disparaît alors que d'autres
// sessions comparables existent. La session affichée provient toujours du filtre
// resserré ; seul le vivier (dropdown + suggestion + visibilité bouton) est élargi.
func comparePoolFilters(f domain.FilterContextInput) domain.FilterContextInput {
	f.Period = domain.PeriodInput{}
	f.Sessions = domain.SessionsFilter{}
	return f
}

func buildSessionCompareSuggestion(
	labels []string,
	currentLabel string,
	rows []legacymatch.StatsMatchRow,
) (*domain.SessionCompareSuggestion, int) {
	currentIndex := indexOfSessionLabel(labels, currentLabel)
	if currentIndex == -1 {
		return nil, 0
	}

	currentMatches := filterBySession(rows, currentLabel)
	current := makeSessionCandidate(currentLabel, currentMatches, currentIndex)

	best := sessionCandidate{}
	bestScore := math.MinInt
	candidateCount := 0
	for i := currentIndex - 1; i >= 0; i-- {
		label := labels[i]
		candidate := makeSessionCandidate(label, filterBySession(rows, label), i)
		if candidate.Count == 0 {
			continue
		}
		candidateCount++
		score := scoreSessionCandidate(current, candidate)
		if score > bestScore {
			bestScore = score
			best = candidate
		}
	}

	if candidateCount == 0 {
		return nil, 0
	}

	strategy := "chronological-fallback"
	if best.Category == current.Category && best.IsRanked == current.IsRanked {
		strategy = "category-ranked-volume"
		if best.Count != current.Count {
			strategy = "category-ranked-close-volume"
		}
	}

	return &domain.SessionCompareSuggestion{
		SessionLabel: best.Label,
		Strategy:     strategy,
		Reason:       buildSuggestionReason(current, best),
	}, candidateCount
}

// sessionPlacement : progression placement X/Y d'un match.
type sessionPlacement struct{ done, total int }

// computeSessionPlacements calcule le placement X/Y par match en RÉUTILISANT la
// logique de l'Explorer (applyMatchPlacements) — cohérence garantie avec l'app. On
// convertit tous les matchs en MatchHistoryRawRow : le label CSR "Placement (N
// restants)" est synthétisé depuis MeasurementRemaining ; LUSR dérivé des 10 plus
// vieux par chaîne. `rows` DOIT contenir TOUS les matchs (pas que la session) pour
// que le calcul LUSR (chronologique global par chaîne) soit correct.
func (s *SessionPageService) computeSessionPlacements(ctx context.Context, rows []legacymatch.StatsMatchRow) map[string]sessionPlacement {
	if len(rows) == 0 {
		return nil
	}
	raw := make([]domain.MatchHistoryRawRow, len(rows))
	for i := range rows {
		src := &rows[i]
		st := src.StartTime
		mr := domain.MatchHistoryRawRow{MatchID: src.MatchID, StartTime: &st, SeasonID: src.SkillSeasonID}
		if src.PairName != "" {
			pn := src.PairName
			mr.PairName = &pn
		}
		if src.SkillRatingType != "" {
			rt := strings.ToUpper(src.SkillRatingType) // "csr"/"lusr" → "CSR"/"LUSR"
			mr.SkillRatingType = &rt
			// CSR en placement : applyCSRPlacements parse "Placement (N restants)" ;
			// on le synthétise depuis MeasurementRemaining (la valeur N).
			if rt == "CSR" && src.SkillMeasurementRemaining != nil && *src.SkillMeasurementRemaining > 0 {
				lbl := fmt.Sprintf("Placement (%d restants)", *src.SkillMeasurementRemaining)
				mr.SkillTierLabel = &lbl
			}
		}
		raw[i] = mr
	}
	applyMatchPlacements(ctx, raw, s.csrThreshold)
	out := make(map[string]sessionPlacement, 8)
	for i := range raw {
		if raw[i].PlacementDone != nil && raw[i].PlacementTotal != nil {
			out[raw[i].MatchID] = sessionPlacement{done: *raw[i].PlacementDone, total: *raw[i].PlacementTotal}
		}
	}
	return out
}

// applyPlacementsToRows renseigne PlacementDone/Total sur les lignes dont le match
// est en phase de placement (map calculée par computeSessionPlacements). Étape
// séparée de la construction des lignes → buildSessionDetailRows garde sa signature.
func applyPlacementsToRows(rows []domain.SessionDetailMatchRow, placements map[string]sessionPlacement) {
	if len(placements) == 0 {
		return
	}
	for i := range rows {
		if p, ok := placements[rows[i].MatchID]; ok {
			d, tot := p.done, p.total
			rows[i].PlacementDone, rows[i].PlacementTotal = &d, &tot
		}
	}
}

func buildSessionDetailRows(
	rows []legacymatch.StatsMatchRow,
	dominantCategory *string,
	locale string,
	assistsExpected map[string]*float64,
) []domain.SessionDetailMatchRow {
	// Locale-aware (aligné Home/Explorer) : FR par défaut, EN si locale == "en".
	// Sans ça les cartes/modes/playlists restaient figés (FR si trad présente,
	// sinon EN) quelle que soit la locale sélectionnée par l'utilisateur.
	frPreferred := locale != "en"
	out := make([]domain.SessionDetailMatchRow, 0, len(rows))
	for _, row := range rows {
		mapName := row.MapName
		if frPreferred && row.MapNameFR != "" {
			mapName = row.MapNameFR
		}
		playlist := row.PlaylistName
		if frPreferred && row.PlaylistNameFR != "" {
			playlist = row.PlaylistNameFR
		}
		var deltaMMR *float64
		if row.TeamMMR != nil && row.EnemyMMR != nil {
			d := *row.TeamMMR - *row.EnemyMMR
			deltaMMR = &d
		}
		perfTier := 0
		if row.PerfScoreComputed != nil {
			perfTier = int(analysis.PerfTier(*row.PerfScoreComputed))
		}
		// Mode : ResolveModeUI prend la trad FR si fournie ; en EN on passe nil
		// pour rester sur le sous-mode normalisé EN.
		var modeUI string
		if frPreferred {
			modeUI = derefString(analysis.ResolveModeUI(&row.PairName, &row.PairNameFR))
			// Repli GameVariant (localisé FR) quand la cascade pair n'a pas trouvé de FR
			// (pair_name_fr NULL + asset_translations[pair] non localisé). Même repli que
			// le converter Home. Ne se déclenche que si le GameVariant FR est réellement
			// localisé (≠ EN) → n'écrase jamais une résolution FR déjà correcte.
			if row.PairNameFR == "" && row.GameVariantNameFR != "" &&
				!strings.EqualFold(row.GameVariantNameFR, row.GameVariantName) {
				if fr := derefString(analysis.ResolveModeUI(&row.GameVariantName, &row.GameVariantNameFR)); fr != "" {
					modeUI = fr
				}
			}
		} else {
			modeUI = derefString(analysis.ResolveModeUI(&row.PairName, nil))
		}
		// Libellé du palier ("Or III", "Diamant V"…) construit comme l'Explorer —
		// la colonne "Rang" affiche le palier, pas la valeur brute. Nil si non rankée.
		skillTierLabel := analysis.BuildSkillTierLabel(row.SkillTierCode, row.SkillTierCodeFR, row.SkillSubTier, frPreferred)
		assistsExp := assistsExpected[row.MatchID]
		kdaExp := analysis.ExpectedFDA(row.KillsExpected, row.DeathsExpected, assistsExp)
		out = append(out, domain.SessionDetailMatchRow{
			MatchID:              row.MatchID,
			StartTime:            row.StartTime,
			Outcome:              row.Outcome,
			PlaylistName:         playlist,
			PairName:             row.PairName,
			IsRanked:             row.IsRanked,
			Kills:                row.Kills,
			Deaths:               row.Deaths,
			Assists:              row.Assists,
			KDA:                  effectiveKDA(row),
			Accuracy:             row.Accuracy,
			PersonalScore:        row.PersonalScore,
			PerformanceScore:     row.PerfScoreComputed,
			SessionLabel:         row.SessionLabel,
			DominantCategory:     dominantCategory,
			OffensiveConv:        row.OffensiveConversion,
			DefensiveResist:      row.DefensiveResistance,
			DamageDealt:          row.DamageDealt,
			DamageTaken:          row.DamageTaken,
			Placement:            row.Rank,
			MapName:              mapName,
			DurationSeconds:      row.TimePlayedSeconds,
			TeamMMR:              row.TeamMMR,
			EnemyMMR:             row.EnemyMMR,
			DeltaMMR:             deltaMMR,
			PerfTier:             perfTier,
			SkillRatingType:      row.SkillRatingType,
			SkillRatingValue:     row.SkillRatingValue,
			SkillRatingDelta:     row.SkillRatingDelta,
			SkillExpectedWinProb: row.SkillExpectedWinProb,
			SkillTierLabel:       skillTierLabel,
			ModeUI:               modeUI,
			KillsExpected:        row.KillsExpected,
			DeathsExpected:       row.DeathsExpected,
			AssistsExpected:      assistsExp,
			KdaExpected:          kdaExp,
		})
	}
	return out
}

func makeSessionCandidate(label string, rows []legacymatch.StatsMatchRow, index int) sessionCandidate {
	return sessionCandidate{
		Label:    label,
		Category: dominantSessionCategory(rows),
		IsRanked: sessionIsRanked(rows),
		IsSquad:  sessionIsSquad(rows),
		Count:    len(rows),
		Index:    index,
	}
}

func scoreSessionCandidate(current, candidate sessionCandidate) int {
	score := 0
	// Composition (escouade/solo) : dimension la plus déterminante pour comparer
	// une session escouade — on lui donne le poids le plus fort.
	if current.IsSquad == candidate.IsSquad {
		score += 8
	}
	if current.Category == candidate.Category {
		score += 6
	}
	if current.IsRanked == candidate.IsRanked {
		score += 3
	}
	diff := absInt(current.Count - candidate.Count)
	switch {
	case diff == 0:
		score += 4
	case diff == 1:
		score += 3
	case diff == 2:
		score += 2
	case diff <= 4:
		score += 1
	}
	indexGap := absInt(current.Index - candidate.Index)
	if indexGap == 1 {
		score += 2
	} else if indexGap <= 3 {
		score += 1
	}
	return score
}

func buildSuggestionReason(current, candidate sessionCandidate) string {
	parts := make([]string, 0, 4)
	if current.IsSquad == candidate.IsSquad {
		if candidate.IsSquad {
			parts = append(parts, "même composition (escouade)")
		} else {
			parts = append(parts, "même composition (solo)")
		}
	}
	if current.Category == candidate.Category {
		parts = append(parts, fmt.Sprintf("même catégorie %s", strings.ToLower(candidate.Category)))
	}
	if current.IsRanked == candidate.IsRanked {
		if candidate.IsRanked {
			parts = append(parts, "même statut classé")
		} else {
			parts = append(parts, "même statut social")
		}
	}
	diff := absInt(current.Count - candidate.Count)
	if diff == 0 {
		parts = append(parts, "même volume")
	} else {
		parts = append(parts, fmt.Sprintf("écart de %d match(s)", diff))
	}
	if len(parts) == 0 {
		return "session chronologiquement proche"
	}
	return strings.Join(parts, " · ")
}

func resolveRequestedCompareLabel(
	req domain.SessionPageRequest,
	suggestion *domain.SessionCompareSuggestion,
) string {
	if req.CompareSessionLabel != nil && *req.CompareSessionLabel != "" {
		return *req.CompareSessionLabel
	}
	if suggestion == nil {
		return ""
	}
	return suggestion.SessionLabel
}

func suggestionLabel(s *domain.SessionCompareSuggestion) string {
	if s == nil {
		return ""
	}
	return s.SessionLabel
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func indexOfSessionLabel(labels []string, target string) int {
	for index, label := range labels {
		if label == target {
			return index
		}
	}
	return -1
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// neighboringSessionLabels retourne les labels chronologiquement adjacents à
// `currentLabel` dans `labels`. labels[i-1] = session précédente (plus ancienne),
// labels[i+1] = session suivante (plus récente). Retourne (nil, nil) si la
// session courante est aux bornes ou absente.
func neighboringSessionLabels(labels []string, currentLabel string) (*string, *string) {
	idx := indexOfSessionLabel(labels, currentLabel)
	if idx == -1 {
		return nil, nil
	}
	var prev, next *string
	if idx-1 >= 0 {
		v := labels[idx-1]
		prev = &v
	}
	if idx+1 < len(labels) {
		v := labels[idx+1]
		next = &v
	}
	return prev, next
}
