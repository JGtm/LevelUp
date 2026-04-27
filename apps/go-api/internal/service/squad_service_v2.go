// Package service — squad_service_v2.go : nouvelle version de la page Squad
// construite sur les fondations Phase 0 (PLAN_META_FOUNDATIONS_GO).
//
// Vit en parallèle de squad_service.go (legacy, mono-coéquipier) jusqu'à
// migration des consommateurs frontend (cf. PLAN_SQUAD_GO_PORTAGE).
//
// Phase 1 chunk S1 : ce fichier livre uniquement le squelette du service avec
// l'intersection des matchs de N coéquipiers (1..3) sur match_id. Les sections
// riches (KPI, score d'équipe, charts synergies, impact 8 rôles, radar...)
// seront greffées par les chunks S2-S11 sans toucher cette base.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// SquadV2Loader est l'interface minimale consommée par SquadServiceV2 pour
// charger les matchs d'un joueur. Permet d'injecter un mock en test sans
// dépendre de PlayerDB / pool concret.
//
// L'implémentation production résout (slug, gamertag) -> PlayerDB via
// pool.GetOrOpen + duckdb.NewPlayerMatchesRepo (adapter à fournir au handler
// dans un chunk ultérieur).
type SquadV2Loader interface {
	LoadFor(
		ctx context.Context,
		slug string,
		gamertag string,
		filters port.PlayerMatchFilters,
	) ([]canonical.PlayerMatchRow, error)
}

// SquadServiceV2 orchestre la page Squad V2.
type SquadServiceV2 struct {
	loader SquadV2Loader
}

// NewSquadServiceV2 construit le service avec un loader injecté.
func NewSquadServiceV2(loader SquadV2Loader) *SquadServiceV2 {
	return &SquadServiceV2{loader: loader}
}

// MaxTeammates est la borne haute du nombre de coéquipiers acceptés (cohérent
// avec la version Python : sélection 1..3).
const MaxTeammates = 3

// GetSquadPage charge les matchs du joueur principal + chacun des coéquipiers
// (parallèle), calcule l'intersection sur match_id, et retourne le DTO V2.
//
// Capability gating : si un joueur retourne games.ErrCapabilityNotSupported,
// il est exclu de l'intersection (le DTO ne le mentionne pas dans Players)
// et un CapabilityGap est ajouté à Capabilities. Si le joueur principal lui-même
// a la capability absente, la page est vide (SharedMatches=nil) avec un gap
// "fatal".
//
// Erreurs autres que ErrCapabilityNotSupported propagées comme une erreur
// 500 par le handler.
func (s *SquadServiceV2) GetSquadPage(
	ctx context.Context,
	slug string,
	mainGT string,
	teammateGTs []string,
	period temporal.Period,
) (*domain.SquadPageV2Response, error) {
	if mainGT == "" {
		return nil, errors.New("SquadServiceV2.GetSquadPage: mainGT requis")
	}
	if len(teammateGTs) > MaxTeammates {
		return nil, fmt.Errorf("SquadServiceV2.GetSquadPage: max %d coéquipiers, %d fournis",
			MaxTeammates, len(teammateGTs))
	}

	filters := port.PlayerMatchFilters{}
	if period != "" {
		filters.Period = &period
	}
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("SquadServiceV2.GetSquadPage: filters: %w", err)
	}

	perPlayer, capGaps, err := s.loadAllPlayers(ctx, slug, mainGT, teammateGTs, filters)
	if err != nil {
		return nil, err
	}

	resp := &domain.SquadPageV2Response{
		MainPlayer:   mainGT,
		Teammates:    teammateGTs,
		Period:       string(period),
		Capabilities: capGaps,
	}

	if _, hasMain := perPlayer[mainGT]; !hasMain {
		// Joueur principal indisponible : page vide mais capability gap signalé.
		slog.WarnContext(ctx, "squad: capability absente pour le joueur principal",
			"player", mainGT, "title_slug", slug)
		return resp, nil
	}

	resp.SharedMatches = intersectByMatchID(perPlayer)
	resp.SharedMatchesCount = len(resp.SharedMatches)
	resp.Header = buildSquadHeader(mainGT, perPlayer, resp.SharedMatches)
	return resp, nil
}

// buildSquadHeader construit le SquadHeader (KPIs personnels + score equipe +
// cartes joueurs) depuis les rows par joueur et l'intersection des matchs
// partages.
//
// SoloKPIs : agreges depuis les rows complets du joueur principal (scope
// courant filtre par period). AllTimeKPIs nil pour S2 (a remplir dans un
// chunk dedie quand on cablera la tendance ▲▼).
//
// PlayerCards : 1 carte par joueur sur les matchs PARTAGES (intersection),
// pas sur l'historique solo. C'est aligne avec Python qui calcule le score
// d'equipe sur les matchs en escouade.
func buildSquadHeader(
	mainGT string,
	perPlayer map[string][]canonical.PlayerMatchRow,
	shared []domain.SquadSharedMatch,
) *domain.SquadHeader {
	header := &domain.SquadHeader{}

	if mainRows, ok := perPlayer[mainGT]; ok && len(mainRows) > 0 {
		soloKPIs := analysis.ComputeKPIStats(mainRows)
		header.SoloKPIs = &soloKPIs
	}

	if len(shared) == 0 {
		return header
	}

	// Carte par joueur : agreger les rows partages depuis SharedMatches.
	rowsByPlayer := projectSharedRows(shared)
	cards := buildPlayerScoreCards(rowsByPlayer)
	header.PlayerCards = cards

	// Score d'equipe : agreger les cartes individuelles + bonus.
	header.SquadScore = buildSquadScoreCard(cards)

	return header
}

// projectSharedRows extrait des SharedMatches une vue par joueur :
// gamertag -> liste des PlayerMatchRow sur les matchs partages.
func projectSharedRows(shared []domain.SquadSharedMatch) map[string][]canonical.PlayerMatchRow {
	out := make(map[string][]canonical.PlayerMatchRow)
	for _, sm := range shared {
		for gt, row := range sm.Players {
			out[gt] = append(out[gt], row)
		}
	}
	return out
}

// buildPlayerScoreCards construit une carte par joueur a partir de ses rows
// sur les matchs partages.
//
// Score : moyenne des Enrichment.PerformanceScore non nil. Si tous nil,
// fallback sur une approximation lineaire (50 + delta KD * 10 clampe).
//
// Comparison ▲▼ : calcule apres avoir agrege tous les scores (passe ulterieur).
func buildPlayerScoreCards(rowsByPlayer map[string][]canonical.PlayerMatchRow) []domain.PlayerScoreCard {
	gts := make([]string, 0, len(rowsByPlayer))
	for gt := range rowsByPlayer {
		gts = append(gts, gt)
	}
	sort.Strings(gts) // ordre stable

	cards := make([]domain.PlayerScoreCard, 0, len(gts))
	var sumScore float64
	for _, gt := range gts {
		card := computeOneCard(gt, rowsByPlayer[gt])
		cards = append(cards, card)
		sumScore += card.Score
	}
	if len(cards) == 0 {
		return cards
	}
	avgScore := sumScore / float64(len(cards))
	// Passe Comparison ▲▼.
	for i := range cards {
		cards[i].Comparison = compareToAvg(cards[i].Score, avgScore)
	}
	return cards
}

// computeOneCard agrege les stats d'un joueur sur ses rows partages.
func computeOneCard(gt string, rows []canonical.PlayerMatchRow) domain.PlayerScoreCard {
	card := domain.PlayerScoreCard{Gamertag: gt}
	if len(rows) == 0 {
		return card
	}
	var totalKills, totalDeaths int
	var wins, losses int
	var perfSum, accSum float64
	var perfSamples, accSamples int

	for _, r := range rows {
		if r.Self.Kills != nil {
			totalKills += *r.Self.Kills
		}
		if r.Self.Deaths != nil {
			totalDeaths += *r.Self.Deaths
		}
		switch r.Self.Outcome {
		case canonical.OutcomeWin:
			wins++
		case canonical.OutcomeLoss:
			losses++
		}
		if r.Enrichment.PerformanceScore != nil {
			perfSum += *r.Enrichment.PerformanceScore
			perfSamples++
		}
		if r.Self.Accuracy != nil {
			accSum += *r.Self.Accuracy
			accSamples++
		}
	}

	card.Kills = totalKills
	if totalDeaths > 0 {
		card.KDRatio = float64(totalKills) / float64(totalDeaths)
	} else if totalKills > 0 {
		card.KDRatio = float64(totalKills) // pas de mort = K/D = nombre de kills
	}
	if wins+losses > 0 {
		card.WinRate = float64(wins) / float64(wins+losses)
	}
	if accSamples > 0 {
		card.Accuracy = accSum / float64(accSamples)
	}
	if perfSamples > 0 {
		card.Score = perfSum / float64(perfSamples)
	} else {
		// Fallback : score approximatif borne 0..100 base sur K/D.
		card.Score = math.Max(0, math.Min(100, 50+10*(card.KDRatio-1)))
	}
	card.Score = math.Round(card.Score*10) / 10
	card.Label = scoreLabel(card.Score)
	return card
}

// scoreLabel mappe un score 0..100 vers une categorie qualitative.
// Aligne avec Python get_score_label().
func scoreLabel(score float64) string {
	switch {
	case score >= 80:
		return "excellent"
	case score >= 65:
		return "good"
	case score >= 50:
		return "average"
	case score >= 35:
		return "poor"
	}
	return "bad"
}

// compareToAvg retourne "above" / "below" / "near" selon ecart au moyen squad.
// Seuil 5 points (au-dela = above/below, sinon near).
func compareToAvg(score, avg float64) string {
	const threshold = 5.0
	if score >= avg+threshold {
		return "above"
	}
	if score <= avg-threshold {
		return "below"
	}
	return "near"
}

// buildSquadScoreCard agrege les cartes individuelles en un score d'equipe.
// Reproduit la logique compute_squad_performance_score() Python (cf.
// analysis/squad_score.go pour la version legacy mais adaptee au DTO V2).
func buildSquadScoreCard(cards []domain.PlayerScoreCard) *domain.SquadScoreCard {
	if len(cards) == 0 {
		return nil
	}
	var scores, kds, killsList []float64
	var winSum float64
	var winSamples int
	for _, c := range cards {
		scores = append(scores, c.Score)
		kds = append(kds, c.KDRatio)
		killsList = append(killsList, float64(c.Kills))
		if c.WinRate > 0 || c.Kills > 0 { // sample si le joueur a au moins joue
			winSum += c.WinRate
			winSamples++
		}
	}
	var sumScores float64
	for _, s := range scores {
		sumScores += s
	}
	baseAvg := sumScores / float64(len(scores))

	teamWR := 0.0
	if winSamples > 0 {
		teamWR = winSum / float64(winSamples)
	}
	bonusWR := 0
	if teamWR > 0.6 {
		bonusWR = 5
	}

	bonusKD := 0
	minKD := math.Inf(1)
	for _, k := range kds {
		if k < minKD {
			minKD = k
		}
	}
	if !math.IsInf(minKD, 1) && minKD > 1.0 {
		bonusKD = 5
	}

	bonusBalance := 0
	stddev := stdDev(killsList)
	if len(killsList) >= 2 && stddev < 3.0 {
		bonusBalance = 3
	}

	score := math.Max(0, math.Min(100, baseAvg+float64(bonusWR+bonusKD+bonusBalance)))
	return &domain.SquadScoreCard{
		Score:        math.Round(score*10) / 10,
		Grade:        squadGrade(score),
		BaseAvg:      math.Round(baseAvg*10) / 10,
		BonusWinRate: bonusWR,
		BonusMinKD:   bonusKD,
		BonusBalance: bonusBalance,
		TeamWinRate:  teamWR,
		MinKD:        minKDOrZero(minKD),
		KillsStdDev:  stddev,
	}
}

// minKDOrZero remplace +Inf par 0 (cas aucun joueur).
func minKDOrZero(v float64) float64 {
	if math.IsInf(v, 1) {
		return 0
	}
	return v
}

// stdDev calcule l'ecart-type d'une serie. Retourne 0 si <2 elements.
func stdDev(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	mean := sum / float64(len(xs))
	var variance float64
	for _, x := range xs {
		diff := x - mean
		variance += diff * diff
	}
	variance /= float64(len(xs))
	return math.Sqrt(variance)
}

// squadGrade mappe un score 0..100 vers une lettre. Aligne avec analysis.resolveSquadGrade.
func squadGrade(score float64) string {
	switch {
	case score >= 90:
		return "S"
	case score >= 80:
		return "A"
	case score >= 65:
		return "B"
	case score >= 50:
		return "C"
	case score >= 35:
		return "D"
	}
	return "F"
}

// loadAllPlayers charge les matchs du joueur principal + des coéquipiers en
// parallèle. Capability absente -> exclu de perPlayer + ajouté à capGaps.
func (s *SquadServiceV2) loadAllPlayers(
	ctx context.Context,
	slug, mainGT string,
	teammateGTs []string,
	filters port.PlayerMatchFilters,
) (map[string][]canonical.PlayerMatchRow, []canonical.CapabilityGap, error) {
	allGTs := append([]string{mainGT}, teammateGTs...)

	g, gctx := errgroup.WithContext(ctx)
	var mu sync.Mutex
	perPlayer := make(map[string][]canonical.PlayerMatchRow, len(allGTs))
	var capGaps []canonical.CapabilityGap

	for _, gt := range allGTs {
		gt := gt
		g.Go(func() error {
			rows, err := s.loader.LoadFor(gctx, slug, gt, filters)
			if err != nil {
				if errors.Is(err, games.ErrCapabilityNotSupported) {
					mu.Lock()
					capGaps = append(capGaps, canonical.CapabilityGap{
						CapabilityKey: string(games.CapMatchHistory),
						ReasonCode:    "match_history_unsupported",
						Severity:      "warning",
						Message:       fmt.Sprintf("match.history non supporté pour %s", gt),
					})
					mu.Unlock()
					return nil
				}
				return fmt.Errorf("LoadFor(%s): %w", gt, err)
			}
			mu.Lock()
			perPlayer[gt] = rows
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}
	return perPlayer, capGaps, nil
}

// intersectByMatchID retourne les matchs présents chez TOUS les joueurs de
// perPlayer. Trié par StartedAt DESC (match le plus récent en premier).
//
// Si perPlayer est vide ou contient des slices vides, retourne nil.
func intersectByMatchID(perPlayer map[string][]canonical.PlayerMatchRow) []domain.SquadSharedMatch {
	if len(perPlayer) == 0 {
		return nil
	}

	indexed := make(map[string]map[string]canonical.PlayerMatchRow, len(perPlayer))
	for gt, rows := range perPlayer {
		idx := make(map[string]canonical.PlayerMatchRow, len(rows))
		for _, r := range rows {
			idx[r.Summary.MatchID] = r
		}
		indexed[gt] = idx
	}

	// Trouver l'intersection : un match_id présent chez tous.
	sharedIDs := matchIDsPresentInAll(indexed)
	if len(sharedIDs) == 0 {
		return nil
	}

	out := make([]domain.SquadSharedMatch, 0, len(sharedIDs))
	for _, id := range sharedIDs {
		sm := buildSharedMatch(id, indexed)
		out = append(out, sm)
	}

	// Tri par StartedAt DESC, fallback alphabétique sur MatchID si égalité.
	sort.Slice(out, func(i, j int) bool {
		if !out[i].StartedAt.Equal(out[j].StartedAt) {
			return out[i].StartedAt.After(out[j].StartedAt)
		}
		return out[i].MatchID < out[j].MatchID
	})
	return out
}

// matchIDsPresentInAll retourne la liste des match_id présents dans toutes les
// maps (intersection ensembliste sur les clés).
func matchIDsPresentInAll(indexed map[string]map[string]canonical.PlayerMatchRow) []string {
	if len(indexed) == 0 {
		return nil
	}
	// Choisir la map la plus petite comme base pour minimiser le travail.
	var smallestGT string
	smallestSize := -1
	for gt, m := range indexed {
		if smallestSize == -1 || len(m) < smallestSize {
			smallestSize = len(m)
			smallestGT = gt
		}
	}

	var out []string
	for id := range indexed[smallestGT] {
		present := true
		for gt, m := range indexed {
			if gt == smallestGT {
				continue
			}
			if _, ok := m[id]; !ok {
				present = false
				break
			}
		}
		if present {
			out = append(out, id)
		}
	}
	return out
}

// buildSharedMatch hydrate un SquadSharedMatch depuis les rows de chaque joueur.
// Les champs niveau-match (Map, Mode, Outcome, StartedAt) sont pris du joueur
// principal sortable (premier dans l'ordre alphabétique des gamertags pour
// reproductibilité).
func buildSharedMatch(matchID string, indexed map[string]map[string]canonical.PlayerMatchRow) domain.SquadSharedMatch {
	sm := domain.SquadSharedMatch{
		MatchID: matchID,
		Players: make(map[string]canonical.PlayerMatchRow, len(indexed)),
	}
	gts := make([]string, 0, len(indexed))
	for gt := range indexed {
		gts = append(gts, gt)
	}
	sort.Strings(gts)
	for _, gt := range gts {
		row := indexed[gt][matchID]
		sm.Players[gt] = row
		if sm.StartedAt.IsZero() {
			sm.StartedAt = row.Summary.StartedAtUTC
			sm.Map = row.Summary.Map
			sm.Mode = row.Summary.GameVariant
			sm.Playlist = row.Summary.Playlist
			sm.Outcome = row.Summary.Outcome
		}
	}
	return sm
}
