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
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

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
	rows := analysis.StatsMatchRowsFromCanonical(canonicalRows)

	filtered := filterStatsMatchRows(rows, req.Filters)
	labels := extractSessionLabels(filtered)
	if len(labels) == 0 {
		slog.InfoContext(ctx, "session page: no sessions after filtering")
		return domain.SessionPageResponse{
			AvailableSessions: []string{},
			Matches:           []domain.SessionDetailMatchRow{},
			CompareMatches:    []domain.SessionDetailMatchRow{},
			CompareMetrics:    []domain.SessionCompareMetricRow{},
		}, nil
	}

	currentLabel := lastOrNil(labels, req.SessionLabel)
	currentMatches := filterBySession(filtered, currentLabel)
	currentEntry := buildCompareEntryWithObjectives(currentMatches, currentLabel, s.objectiveScores(ctx, currentMatches))
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

	suggestion, candidateCount := buildSessionCompareSuggestion(labels, currentLabel, filtered)
	compareLabel := resolveRequestedCompareLabel(req, suggestion)
	compareEnabled := req.EnableCompare && compareLabel != "" && compareLabel != currentLabel

	prevLabel, nextLabel := neighboringSessionLabels(labels, currentLabel)

	resp := domain.SessionPageResponse{
		CurrentSession:       currentEntry,
		AvailableSessions:    labels,
		Matches:              buildSessionDetailRows(currentMatches, currentEntry.DominantCategory, req.Locale),
		SuggestedCompare:     suggestion,
		CompareEnabled:       compareEnabled,
		CompareMatches:       []domain.SessionDetailMatchRow{},
		CompareMetrics:       []domain.SessionCompareMetricRow{},
		PreviousSessionLabel: prevLabel,
		NextSessionLabel:     nextLabel,
	}

	if compareEnabled {
		compareMatches := filterBySession(filtered, compareLabel)
		resp.CompareSession = buildCompareEntryWithObjectives(compareMatches, compareLabel, s.objectiveScores(ctx, compareMatches))
		if resp.CompareSession != nil {
			resp.CompareMetrics = buildCompareMetrics(currentMatches, compareMatches)
			resp.CompareMatches = buildSessionDetailRows(compareMatches, resp.CompareSession.DominantCategory, req.Locale)
		} else {
			resp.CompareEnabled = false
			slog.WarnContext(ctx, "session page: compare session missing after filtering",
				"current_session", currentLabel,
				"compare_session", compareLabel,
			)
		}
	}

	// Taille de lobby (joueurs présents à la fin, bots inclus) pour le breakdown
	// des placements — best-effort, dégrade gracieusement si le repo ne le fournit pas.
	s.attachLobbySizes(ctx, resp.Matches, resp.CompareMatches)

	slog.InfoContext(ctx, "session page generated",
		"resolved_session", currentLabel,
		"available_sessions", len(labels),
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
	Count    int
	Index    int
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

func buildSessionDetailRows(
	rows []legacymatch.StatsMatchRow,
	dominantCategory *string,
	locale string,
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
		} else {
			modeUI = derefString(analysis.ResolveModeUI(&row.PairName, nil))
		}
		out = append(out, domain.SessionDetailMatchRow{
			MatchID:          row.MatchID,
			StartTime:        row.StartTime,
			Outcome:          row.Outcome,
			PlaylistName:     playlist,
			PairName:         row.PairName,
			IsRanked:         row.IsRanked,
			Kills:            row.Kills,
			Deaths:           row.Deaths,
			Assists:          row.Assists,
			KDA:              effectiveKDA(row),
			Accuracy:         row.Accuracy,
			PersonalScore:    row.PersonalScore,
			PerformanceScore: row.PerfScoreComputed,
			SessionLabel:     row.SessionLabel,
			DominantCategory: dominantCategory,
			OffensiveConv:    row.OffensiveConversion,
			DefensiveResist:  row.DefensiveResistance,
			DamageDealt:      row.DamageDealt,
			DamageTaken:      row.DamageTaken,
			Placement:        row.Rank,
			MapName:          mapName,
			DurationSeconds:  row.TimePlayedSeconds,
			TeamMMR:          row.TeamMMR,
			EnemyMMR:         row.EnemyMMR,
			DeltaMMR:         deltaMMR,
			PerfTier:         perfTier,
			SkillRatingType:  row.SkillRatingType,
			SkillRatingValue: row.SkillRatingValue,
			SkillRatingDelta: row.SkillRatingDelta,
			ModeUI:           modeUI,
		})
	}
	return out
}

func makeSessionCandidate(label string, rows []legacymatch.StatsMatchRow, index int) sessionCandidate {
	return sessionCandidate{
		Label:    label,
		Category: dominantSessionCategory(rows),
		IsRanked: sessionIsRanked(rows),
		Count:    len(rows),
		Index:    index,
	}
}

func scoreSessionCandidate(current, candidate sessionCandidate) int {
	score := 0
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
	parts := make([]string, 0, 3)
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
