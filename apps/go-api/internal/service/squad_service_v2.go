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

// SquadV2Loader est l'interface consommee par SquadServiceV2 pour charger
// toutes les donnees necessaires a la page Squad V2 (matchs, events, kills
// armes, medailles).
//
// Permet d'injecter un mock en test sans dependre de PlayerDB / pool concret.
// L'implementation production resout (slug, gamertag) -> *PlayerDB via le
// pool global puis delegue aux repos concrets.
//
// Capability gating : chaque methode peut retourner games.ErrCapabilityNotSupported
// pour signaler que la source n'est pas disponible. Le service degrade gracieusement
// en omettant la section concernee + ajoutant un CapabilityGap.
type SquadV2Loader interface {
	// LoadFor charge les matchs d'un joueur (chunk S1).
	LoadFor(
		ctx context.Context,
		slug string,
		gamertag string,
		filters port.PlayerMatchFilters,
	) ([]canonical.PlayerMatchRow, error)

	// LoadHighlightEvents charge les events filmes pour les matchs partages
	// (chunks S5+S6). Implementation prod : duckdb.HighlightEventsRepo.
	LoadHighlightEvents(
		ctx context.Context,
		slug string,
		filters port.HighlightEventFilters,
	) ([]canonical.HighlightEvent, error)

	// LoadWeaponKills charge les kills aggreges par arme (chunk S9).
	// Implementation prod : duckdb.WeaponKillsRepo.
	LoadWeaponKills(
		ctx context.Context,
		slug string,
		filters port.WeaponKillFilters,
	) ([]port.WeaponKillRow, error)

	// LoadMedals charge les medailles par (xuid, match) (chunk S9).
	// Implementation prod : duckdb.MedalsByXUIDRepo.
	LoadMedals(
		ctx context.Context,
		slug string,
		filters port.MedalsByXUIDFilters,
	) ([]port.MedalRow, error)

	// LoadEmblemURLs retourne l'URL de l'emblème Spartan de chaque gamertag
	// (depuis career_progression.emblem_image_url). Dégradation silencieuse :
	// les joueurs sans DB ou sans données retournent une entrée vide.
	LoadEmblemURLs(
		ctx context.Context,
		slug string,
		gamertags []string,
	) map[string]string
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
	experienceTypes []string,
	playlists []string,
	maps []string,
	modes []string,
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

	// Appliquer le filtre cascade (experience_types, playlists, maps, modes) sur les rows de
	// chaque joueur avant l'intersection : seuls les matchs satisfaisant tous les critères
	// sont conservés.
	if len(experienceTypes) > 0 || len(playlists) > 0 || len(maps) > 0 || len(modes) > 0 {
		for gt, rows := range perPlayer {
			perPlayer[gt] = filterRowsByCascade(rows, experienceTypes, playlists, maps, modes)
		}
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

	// gtToXUID est utilise par buildSquadHeader (KPIsByXUID drill-down) ET
	// par les charts/tables ci-dessous (squadXUIDs). Calcul fait une seule fois.
	squadOrder := buildSquadOrder(mainGT, teammateGTs)
	squadXUIDs := extractSquadXUIDs(squadOrder, perPlayer)

	resp.Header = buildSquadHeader(ctx, mainGT, perPlayer, squadXUIDs, resp.SharedMatches)

	// Si pas de matchs partages, retourner sans charger les sections lourdes.
	if len(resp.SharedMatches) == 0 {
		return resp, nil
	}

	// Composition des charts + tableaux. Charge les sources externes
	// (events, weapons, medals) en parallele puis appelle les 16 builders.
	rowsBySharedPlayer := projectSharedRows(resp.SharedMatches)

	historicalMain, _ := s.loadHistoricalMain(ctx, slug, mainGT)
	events, eventsCapGap := s.loadSharedEvents(ctx, slug, resp.SharedMatches, squadXUIDs)
	weapons, weaponsCapGap := s.loadWeapons(ctx, slug, resp.SharedMatches, squadXUIDs)
	medals, medalsCapGap := s.loadMedals(ctx, slug, resp.SharedMatches, squadXUIDs)

	resp.Charts = buildSquadCharts(buildSquadChartsInput{
		mainGT:         mainGT,
		squadOrder:     squadOrder,
		squadXUIDs:     squadXUIDs,
		rowsByPlayer:   rowsBySharedPlayer,
		mainHistorical: historicalMain,
		events:         events,
		sharedMatches:  resp.SharedMatches,
	})
	resp.Tables = buildSquadTables(buildSquadTablesInput{
		sharedMatches: resp.SharedMatches,
		rowsByPlayer:  rowsBySharedPlayer,
		squadOrder:    squadOrder,
		squadXUIDs:    squadXUIDs,
		weapons:       weapons,
		medals:        medals,
	})

	for _, gap := range []*canonical.CapabilityGap{eventsCapGap, weaponsCapGap, medalsCapGap} {
		if gap != nil {
			resp.Capabilities = append(resp.Capabilities, *gap)
		}
	}
	return resp, nil
}

// buildSquadOrder construit l'ordre stable des gamertags du squad : main puis
// coequipiers dans l'ordre d'arrivee.
func buildSquadOrder(mainGT string, teammates []string) []string {
	out := make([]string, 0, 1+len(teammates))
	out = append(out, mainGT)
	out = append(out, teammates...)
	return out
}

// extractSquadXUIDs derive le mapping gamertag -> xuid en regardant la
// premiere PlayerMatchRow disponible pour chaque joueur. Si un joueur n'a
// aucun match (capability absente), il est omis.
func extractSquadXUIDs(squadOrder []string, perPlayer map[string][]canonical.PlayerMatchRow) map[string]string {
	out := make(map[string]string, len(squadOrder))
	for _, gt := range squadOrder {
		rows := perPlayer[gt]
		if len(rows) == 0 {
			continue
		}
		if xuid := rows[0].Self.Identity.XUID; xuid != "" {
			out[gt] = xuid
		}
	}
	return out
}

// loadHistoricalMain charge l'historique complet du joueur principal (sans
// filtre period) pour les charts BulletWinrate / PerfVsHistorical (S3).
// Capability absente -> retourne nil (les charts dependants seront omis).
func (s *SquadServiceV2) loadHistoricalMain(ctx context.Context, slug, mainGT string) ([]canonical.PlayerMatchRow, error) {
	rows, err := s.loader.LoadFor(ctx, slug, mainGT, port.PlayerMatchFilters{})
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil, nil
		}
		slog.WarnContext(ctx, "squad: loadHistoricalMain echec", "err", err, "player", mainGT)
		return nil, nil
	}
	return rows, nil
}

// loadSharedEvents charge les events filmes des matchs partages (squad XUIDs).
// Capability absente -> retourne nil + CapabilityGap pour signaler S5/S6 omis.
func (s *SquadServiceV2) loadSharedEvents(
	ctx context.Context,
	slug string,
	shared []domain.SquadSharedMatch,
	squadXUIDs map[string]string,
) ([]canonical.HighlightEvent, *canonical.CapabilityGap) {
	if len(shared) == 0 || len(squadXUIDs) == 0 {
		return nil, nil
	}
	matchIDs := matchIDsOf(shared)
	xuids := xuidsOf(squadXUIDs)
	// Pour valider les filtres : MatchIDs requis (pas besoin de PlayerXUID
	// dans ce cas, le repo filtrera client-side via squadXUIDs).
	filters := port.HighlightEventFilters{MatchIDs: matchIDs}
	if err := filters.Validate(); err != nil {
		slog.WarnContext(ctx, "squad: HighlightEventFilters invalides", "err", err)
		return nil, nil
	}
	events, err := s.loader.LoadHighlightEvents(ctx, slug, filters)
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil, &canonical.CapabilityGap{
				CapabilityKey: "match.detail.events",
				ReasonCode:    "events_unsupported",
				Severity:      "info",
				Message:       "Events filmes non disponibles : Impact + Cadence + Intensite omis.",
			}
		}
		slog.ErrorContext(ctx, "squad: LoadHighlightEvents echec", "err", err)
		return nil, nil
	}
	// Filtrer client-side aux squad xuids (repo retourne tous events des matchs).
	xuidSet := make(map[string]bool, len(xuids))
	for _, x := range xuids {
		xuidSet[x] = true
	}
	filtered := events[:0]
	for _, ev := range events {
		if isEventInSquad(ev, xuidSet) {
			filtered = append(filtered, ev)
		}
	}
	return filtered, nil
}

func isEventInSquad(ev canonical.HighlightEvent, xuidSet map[string]bool) bool {
	if ev.KillerXUID != nil && xuidSet[*ev.KillerXUID] {
		return true
	}
	if ev.VictimXUID != nil && xuidSet[*ev.VictimXUID] {
		return true
	}
	if ev.PlayerXUID != nil && xuidSet[*ev.PlayerXUID] {
		return true
	}
	return false
}

// loadWeapons charge les kills aggregees par arme pour les matchs partages.
func (s *SquadServiceV2) loadWeapons(
	ctx context.Context,
	slug string,
	shared []domain.SquadSharedMatch,
	squadXUIDs map[string]string,
) ([]port.WeaponKillRow, *canonical.CapabilityGap) {
	if len(shared) == 0 || len(squadXUIDs) == 0 {
		return nil, nil
	}
	filters := port.WeaponKillFilters{
		MatchIDs:            matchIDsOf(shared),
		XUIDs:               xuidsOf(squadXUIDs),
		IncludeGrenadeMelee: true,
	}
	if err := filters.Validate(); err != nil {
		slog.WarnContext(ctx, "squad: WeaponKillFilters invalides", "err", err)
		return nil, nil
	}
	rows, err := s.loader.LoadWeaponKills(ctx, slug, filters)
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil, &canonical.CapabilityGap{
				CapabilityKey: "match.detail.weapon_kills",
				ReasonCode:    "weapon_kills_unsupported",
				Severity:      "info",
				Message:       "Kills par arme non disponibles : tableau armes omis.",
			}
		}
		slog.ErrorContext(ctx, "squad: LoadWeaponKills echec", "err", err)
		return nil, nil
	}
	return rows, nil
}

// loadMedals charge les medailles par (xuid, match) pour les matchs partages.
func (s *SquadServiceV2) loadMedals(
	ctx context.Context,
	slug string,
	shared []domain.SquadSharedMatch,
	squadXUIDs map[string]string,
) ([]port.MedalRow, *canonical.CapabilityGap) {
	if len(shared) == 0 || len(squadXUIDs) == 0 {
		return nil, nil
	}
	filters := port.MedalsByXUIDFilters{
		MatchIDs: matchIDsOf(shared),
		XUIDs:    xuidsOf(squadXUIDs),
	}
	if err := filters.Validate(); err != nil {
		slog.WarnContext(ctx, "squad: MedalsByXUIDFilters invalides", "err", err)
		return nil, nil
	}
	rows, err := s.loader.LoadMedals(ctx, slug, filters)
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil, &canonical.CapabilityGap{
				CapabilityKey: "match.detail.medals",
				ReasonCode:    "medals_unsupported",
				Severity:      "info",
				Message:       "Medailles non disponibles : galerie medailles omise.",
			}
		}
		slog.ErrorContext(ctx, "squad: LoadMedals echec", "err", err)
		return nil, nil
	}
	return rows, nil
}

func matchIDsOf(shared []domain.SquadSharedMatch) []string {
	out := make([]string, 0, len(shared))
	for _, sm := range shared {
		out = append(out, sm.MatchID)
	}
	return out
}

func xuidsOf(squadXUIDs map[string]string) []string {
	out := make([]string, 0, len(squadXUIDs))
	for _, x := range squadXUIDs {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

// buildSquadHeader construit le SquadHeader (KPIs personnels + score equipe +
// cartes joueurs + KPIs per-xuid pour drill-down SessionBriefing) depuis les
// rows par joueur et l'intersection des matchs partages.
//
// SoloKPIs : agreges depuis les rows complets du joueur principal (scope
// courant filtre par period). AllTimeKPIs nil pour S2 (a remplir dans un
// chunk dedie quand on cablera la tendance ▲▼).
//
// PlayerCards : 1 carte par joueur sur les matchs PARTAGES (intersection),
// pas sur l'historique solo. C'est aligne avec Python qui calcule le score
// d'equipe sur les matchs en escouade.
//
// KPIsByXUID + TeamAvgKPIs : agreges sur les matchs PARTAGES (meme scope que
// PlayerCards). Alimentent le SessionBriefing (drill-down click + reference
// trends ▲/▼). KPIsByXUID est cle par xuid (pas gamertag) pour matcher avec
// PlayerScoreCard.XUID cote front.
//
// Capability gating : si LoadFor a retourne ErrCapabilityNotSupported pour le
// joueur principal, perPlayer[mainGT] est absent et le caller GetSquadPage a
// deja court-circuite. Pas besoin de gate explicite ici.
func buildSquadHeader(
	ctx context.Context,
	mainGT string,
	perPlayer map[string][]canonical.PlayerMatchRow,
	gtToXUID map[string]string,
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
		kpis := analysis.ComputeKPIStats(rows)
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
		return "PVE"
	}
	if r.Summary.IsRanked != nil && *r.Summary.IsRanked {
		return "PVP classé"
	}
	return "PVP non classé"
}

// filterRowsByCascade filtre une slice de PlayerMatchRow selon les critères
// experience_types, playlists, maps et modes. Slices vides = pas de filtre sur ce critère.
//
// Modes : comparaison sur PairMode (pair_name_fr COALESCE pair_name) — même source
// que filtersResolve. Fallback sur DefaultLabel (EN) si Labels["fr"] absent.
func filterRowsByCascade(rows []canonical.PlayerMatchRow, expTypes, playlists, maps, modes []string) []canonical.PlayerMatchRow {
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
			sm.Mode = row.Summary.PairMode
			if sm.Mode == nil {
				sm.Mode = row.Summary.GameVariant
			}
			sm.Playlist = row.Summary.Playlist
			sm.Outcome = row.Summary.Outcome
		}
	}
	return sm
}
