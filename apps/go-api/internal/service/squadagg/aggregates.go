package squadagg

import (
	"context"
	"log/slog"
	"math"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

func BuildSquadHeader(
	ctx context.Context,
	mainGT string,
	gtToXUID map[string]string,
	shared []domain.SquadSharedMatch,
) *domain.SquadHeader {
	header := &domain.SquadHeader{}

	if len(shared) == 0 {
		return header
	}

	// Carte par joueur : agreger les rows partages depuis SharedMatches.
	rowsByPlayer := ProjectSharedRows(shared)
	hp := games.EffectiveHpToKill(ctxkeys.TitleSlug(ctx))

	// SoloKPIs : KPIs du joueur principal sur les matchs partages uniquement.
	// Le briefing reflete ainsi le scope escouade, pas l'historique solo.
	if mainRows, ok := rowsByPlayer[mainGT]; ok && len(mainRows) > 0 {
		soloKPIs := analysis.ComputeKPIStats(mainRows, hp)
		header.SoloKPIs = &soloKPIs
	}

	cards := buildPlayerScoreCards(rowsByPlayer, gtToXUID)
	header.PlayerCards = cards

	// Score d'equipe : agreger les cartes individuelles + bonus.
	header.SquadScore = buildSquadScoreCard(cards)

	// KPIs par xuid (drill-down SessionBriefing) + moyenne d'equipe (reference
	// pour trends ▲/▼). Calcul sur les memes rows que PlayerCards.
	kpisByXUID := make(map[string]*domain.KPIStats, len(rowsByPlayer))
	for gt, rows := range rowsByPlayer {
		xuid := gtToXUID[gt]
		if xuid == "" {
			continue
		}
		kpis := analysis.ComputeKPIStats(rows, hp)
		kpisByXUID[xuid] = &kpis
	}
	if len(kpisByXUID) > 0 {
		header.KPIsByXUID = kpisByXUID
		header.TeamAvgKPIs = analysis.ComputeTeamAvgKPIs(kpisByXUID)
	}
	slog.DebugContext(ctx, "squad_briefing.kpis_by_xuid_computed",
		"team_size", len(kpisByXUID),
		"shared_match_count", len(shared))

	return header
}

// ProjectSharedRows extrait des SharedMatches une vue par joueur :
// gamertag -> liste des PlayerMatchRow sur les matchs partages.
func ProjectSharedRows(shared []domain.SquadSharedMatch) map[string][]canonical.PlayerMatchRow {
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
//
// gtToXUID est utilise pour renseigner PlayerScoreCard.XUID (utilise par le
// SessionBriefing front pour le drill-down click). Si un gamertag n'a pas de
// xuid resolu, le card.XUID reste vide (le front fallback sur SoloKPIs).
func buildPlayerScoreCards(
	rowsByPlayer map[string][]canonical.PlayerMatchRow,
	gtToXUID map[string]string,
) []domain.PlayerScoreCard {
	gts := make([]string, 0, len(rowsByPlayer))
	for gt := range rowsByPlayer {
		gts = append(gts, gt)
	}
	sort.Strings(gts) // ordre stable

	cards := make([]domain.PlayerScoreCard, 0, len(gts))
	var sumScore float64
	for _, gt := range gts {
		card := computeOneCard(gt, rowsByPlayer[gt])
		card.XUID = gtToXUID[gt]
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
		card.WinRate = analysis.WinRate(wins, wins+losses)
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

// canonicalRowExperienceLabel dérive le label d'expérience d'une PlayerMatchRow.
// Miroir de synthesisExperienceLabel dans teammates_service.go.
func canonicalRowExperienceLabel(r canonical.PlayerMatchRow) string {
	if r.Summary.IsPvE != nil && *r.Summary.IsPvE {
		return ExpTypePVE
	}
	if r.Summary.IsRanked != nil && *r.Summary.IsRanked {
		return ExpTypePVPRanked
	}
	return ExpTypePVPUnranked
}

// FilterRowsByCascade filtre une slice de PlayerMatchRow selon les critères
// experience_types, playlists, maps et modes. Slices vides = pas de filtre sur ce critère.
//
// Modes : comparaison sur PairMode (pair_name_fr COALESCE pair_name) — même source
// que filtersResolve. Fallback sur DefaultLabel (EN) si Labels["fr"] absent.
func FilterRowsByCascade(rows []canonical.PlayerMatchRow, expTypes, playlists, maps, modes []string) []canonical.PlayerMatchRow {
	expSet := make(map[string]struct{}, len(expTypes))
	for _, e := range expTypes {
		expSet[e] = struct{}{}
	}
	plSet := make(map[string]struct{}, len(playlists))
	for _, p := range playlists {
		plSet[p] = struct{}{}
	}
	mapSet := make(map[string]struct{}, len(maps))
	for _, m := range maps {
		mapSet[m] = struct{}{}
	}
	modeSet := make(map[string]struct{}, len(modes))
	for _, m := range modes {
		modeSet[m] = struct{}{}
	}
	out := rows[:0:0]
	for _, r := range rows {
		if len(expSet) > 0 {
			if _, ok := expSet[canonicalRowExperienceLabel(r)]; !ok {
				continue
			}
		}
		if len(plSet) > 0 {
			if _, ok := plSet[assetLabelForFilter(r.Summary.Playlist)]; !ok {
				continue
			}
		}
		if len(mapSet) > 0 {
			if _, ok := mapSet[assetLabelForFilter(r.Summary.Map)]; !ok {
				continue
			}
		}
		if len(modeSet) > 0 {
			if _, ok := modeSet[assetLabelForFilter(r.Summary.PairMode)]; !ok {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

// assetLabelForFilter retourne le label d'un AssetReference utilisé comme clé de filtre.
// Préfère Labels["fr"] (COALESCE(name_fr, name) — même source que filtersResolve)
// sur DefaultLabel (anglais pur) pour alignement avec les valeurs du frontend.
func assetLabelForFilter(ref *canonical.AssetReference) string {
	if ref == nil {
		return ""
	}
	if fr, ok := ref.Labels["fr"]; ok && fr != "" {
		return fr
	}
	return ref.DefaultLabel
}

// loadAllPlayers charge les matchs du joueur principal + des coéquipiers en
// parallèle. Capability absente -> exclu de perPlayer + ajouté à capGaps.
