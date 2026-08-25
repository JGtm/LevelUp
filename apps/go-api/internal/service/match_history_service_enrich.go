// Package service - match_history_service_enrich.go : toFilterMatchRow +
// computeMapWinRates + enrichRows/enrichRow + sortItems/compareRows +
// paginate + helpers de format (outcomeLabel, formatDateFR,
// formatLifeSeconds, buildPeriodLabel, ptr/cmp helpers).
// Decoupe de match_history_service.go (god-file split, refactor 2026-05-27).
package service

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

func toFilterMatchRow(r domain.MatchHistoryRawRow) domain.FilterMatchRow {
	// PlaylistName est déjà FR-préféré (COALESCE(playlist_name_fr, playlist_name),
	// cf. domain.MatchHistoryRawRow) → cohérent avec la Value FR des options.
	// GameVariantName/FR + PlaylistNameEN sont remplis pour H5 par
	// applyMatchHistoryFRTranslations : GameVariant* alimente le fallback mode
	// (titres sans pair), PlaylistNameEN sert de clé de migration cascade.
	return domain.FilterMatchRow{
		MatchID:           r.MatchID,
		StartTime:         r.StartTime,
		MapName:           r.MapName,
		MapNameFR:         r.MapNameFR,
		PairName:          r.PairName,
		PairNameFR:        r.PairNameFR,
		PlaylistName:      r.PlaylistName,
		PlaylistNameEN:    r.PlaylistNameEN,
		GameVariantName:   r.GameVariantName,
		GameVariantNameFR: r.GameVariantNameFR,
		IsFirefight:       r.IsFirefight,
		IsRanked:          r.IsRanked,
		SessionID:         r.SessionID,
		SessionLabel:      r.SessionLabel,
		IsWithFriends:     r.IsWithFriends,
	}
}

// ---------------------------------------------------------------------------
// Win rate par carte
// ---------------------------------------------------------------------------

func computeMapWinRates(rows []domain.MatchHistoryRawRow) map[string][2]int {
	m := make(map[string][2]int)
	for _, r := range rows {
		name := derefStr(r.MapName)
		if name == "" {
			continue
		}
		entry := m[name]
		entry[1]++ // total
		if r.Outcome == domain.OutcomeWin {
			entry[0]++ // wins
		}
		m[name] = entry
	}
	return m
}

// ---------------------------------------------------------------------------
// Enrichissement
// ---------------------------------------------------------------------------

// rowFormatters regroupe les résolveurs title-agnostic injectés dans l'enrichissement
// d'une ligne : URL de page publique du match (F3) et libellé d'outcome via le titre
// (F4). Champs nil → dégradation gracieuse (URL vide ; outcome via le fallback FR dur).
type rowFormatters struct {
	matchURL      func(matchID string) string
	outcomeLabel  func(code int) string
	playlistLabel func(rawFR string) string
	// regulation : table `game_variant_name → temps réglementaire (s)` du titre
	// courant (regulation.toml). Nil/vide → aucun flag « Prolongation », jamais
	// d'erreur. Injectée par MatchHistoryService.WithRegulation.
	regulation map[string]int
	// skillBadgeURL : résolveur d'URL d'image de badge de palier du TITRE courant
	// (même contrat que la home : (tier capitalisé, sous-palier 0=Onyx/1..6) → URL).
	// Nil → aucune image, le front retombe sur le libellé texte du palier.
	skillBadgeURL func(tierEN string, subTier int) string
	// replays : ensemble des matchs ayant un artefact de rejeu 2D, construit UNE FOIS
	// par requête (un listing de dossier) et consulté en O(1) par ligne. Vide → aucune
	// ligne ne porte de rejeu, ce qui est l'état d'un titre sans film cuit.
	replays port.ReplayAvailability
}

// skillRankImageURLFor résout l'image du badge de palier d'une ligne brute via le
// chokepoint unique analysis.SkillBadgeURL (normalisation partagée avec la home).
// nil sans résolveur injecté ou sans palier exploitable → dégradation sur le texte.
func (f rowFormatters) skillRankImageURLFor(r domain.MatchHistoryRawRow) *string {
	return analysis.SkillBadgeURL(derefStr(r.SkillTier), r.SubTier, f.skillBadgeURL)
}

// overtimeFor résout le flag « Prolongation » d'une ligne brute via le
// chokepoint unique analysis.ResolveOvertime (même résolution que la Match View).
func (f rowFormatters) overtimeFor(r domain.MatchHistoryRawRow) (bool, int) {
	return analysis.ResolveOvertime(derefStr(r.GameVariantName), r.ElapsedSeconds, f.regulation)
}

func (f rowFormatters) matchURLFor(matchID string) string {
	if f.matchURL == nil {
		return ""
	}
	return f.matchURL(matchID)
}

// playlistLabelFor résout le libellé d'affichage de la playlist via le chokepoint
// unique injecté (strip + override). nil → renvoie le brut (dégradation gracieuse).
func (f rowFormatters) playlistLabelFor(rawFR string) string {
	if f.playlistLabel == nil {
		return rawFR
	}
	return f.playlistLabel(rawFR)
}

// hasReplayFor dit si le match a un artefact de rejeu 2D. Lookup O(1) dans l'ensemble
// construit une fois par requête : aucun accès disque ici.
func (f rowFormatters) hasReplayFor(matchID string) bool {
	return f.replays.Has(matchID)
}

func (f rowFormatters) outcomeLabelFor(code int) string {
	if f.outcomeLabel == nil {
		return outcomeLabel(code) // failsafe : libellés FR canoniques Halo
	}
	return f.outcomeLabel(code)
}

func enrichRows(rows []domain.MatchHistoryRawRow, mapWR map[string][2]int, fmts rowFormatters) []domain.MatchHistoryRow {
	out := make([]domain.MatchHistoryRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, enrichRow(r, mapWR, fmts))
	}
	return out
}

// enrichMapWinRate calcule le taux de victoire historique du joueur sur une map
// (arrondi 1 déc.) + le total de matchs, depuis la table mapWR pré-agrégée.
// (nil, nil) si map inconnue ou 0 match. Extrait d'enrichRow (L3 : funlen).
func enrichMapWinRate(mapName string, mapWR map[string][2]int) (*float64, *int) {
	if mapName == "" {
		return nil, nil
	}
	entry, ok := mapWR[mapName]
	if !ok || entry[1] <= 0 {
		return nil, nil
	}
	v := math.Round(float64(entry[0])/float64(entry[1])*100*10) / 10
	total := entry[1]
	return &v, &total
}

func enrichRow(r domain.MatchHistoryRawRow, mapWR map[string][2]int, fmts rowFormatters) domain.MatchHistoryRow {
	// mode_ui : pair prioritaire, sinon fallback game_variant (titres sans pair_name,
	// ex. Halo 5). Sans ce repli, mode_ui resterait vide sur la liste Explorer/
	// Historique pour H5. Convention centralisée — cf. analysis.ResolveModeUIWithVariant.
	modeUI := analysis.ResolveModeUIWithVariant(r.PairName, r.PairNameFR, r.GameVariantName, r.GameVariantNameFR)
	mapU := coalesceStr(r.MapNameFR, r.MapName)
	playlist := fmts.playlistLabelFor(coalesceStr(r.PlaylistName))

	var startTime time.Time
	if r.StartTime != nil {
		startTime = *r.StartTime
	}
	label := formatDateFR(startTime)

	var deltaMMR *float64
	if r.TeamMMR != nil && r.EnemyMMR != nil {
		v := *r.TeamMMR - *r.EnemyMMR
		deltaMMR = &v
	}

	winRate, winRateTotal := enrichMapWinRate(derefStr(r.MapName), mapWR)

	matchURL := fmts.matchURLFor(r.MatchID)

	var perfScore *int
	var perfTier int
	if r.PerformanceScore != nil {
		v := int(math.Round(*r.PerformanceScore))
		perfScore = &v
		perfTier = int(analysis.PerfTier(*r.PerformanceScore))
	}

	var kda *float64
	if r.KDA != nil {
		kda = r.KDA
	}

	scoreLabel := "-"
	if r.MyTeamScore != nil && r.EnemyTeamScore != nil {
		scoreLabel = fmt.Sprintf("%d - %d", *r.MyTeamScore, *r.EnemyTeamScore)
	}

	isOvertime, overtimeSeconds := fmts.overtimeFor(r)

	return domain.MatchHistoryRow{
		MatchID:                  r.MatchID,
		StartTime:                startTime,
		StartTimeLabel:           label,
		OutcomeCode:              r.Outcome,
		OutcomeLabel:             fmts.outcomeLabelFor(r.Outcome),
		ScoreLabel:               scoreLabel,
		MapUI:                    ptrStr(mapU),
		ModeUI:                   modeUI,
		PlaylistLabel:            ptrStr(playlist),
		TeamMMR:                  r.TeamMMR,
		EnemyMMR:                 r.EnemyMMR,
		DeltaMMR:                 deltaMMR,
		WinRateHist:              winRate,
		WinRateHistTotal:         winRateTotal,
		PerformanceScoreRelative: perfScore,
		PerfTier:                 perfTier,
		KDA:                      kda,
		Kills:                    r.Kills,
		Deaths:                   r.Deaths,
		Assists:                  r.Assists,
		SkillTierLabel:           r.SkillTierLabel,
		SkillRatingType:          r.SkillRatingType,
		SkillRankImageURL:        fmts.skillRankImageURLFor(r),
		PersonalScore:            r.PersonalScore,
		ExpectedWinProb:          r.SkillExpectedWinProb,
		PlacementDone:            r.PlacementDone,
		PlacementTotal:           r.PlacementTotal,
		PerfPlacementDone:        r.PerfPlacementDone,
		PerfPlacementTotal:       r.PerfPlacementTotal,
		AverageLifeMMSS:          formatLifeSeconds(r.AverageLifeSeconds),
		DurationSeconds:          r.TimePlayedSeconds,
		IsOvertime:               isOvertime,
		OvertimeSeconds:          overtimeSeconds,
		MatchURL:                 matchURL,
		HasReplay:                fmts.hasReplayFor(r.MatchID),
		IsExcluded:               r.IsExcluded,
		IsWithFriends:            r.IsWithFriends,
		ExperienceTypeLabel:      explorerExperienceType(r),
		DominanceFlag:            r.DominanceFlag,
	}
}

// ---------------------------------------------------------------------------
// Tri + pagination
// ---------------------------------------------------------------------------

func sortItems(items []domain.MatchHistoryRow, field, dir string) {
	if field == "" {
		field = "start_time"
	}
	if dir == "" {
		dir = "desc"
	}
	descending := dir == "desc"

	sort.SliceStable(items, func(i, j int) bool {
		less := compareMatchHistoryRows(items[i], items[j], field)
		if descending {
			return !less
		}
		return less
	})
}

func compareMatchHistoryRows(a, b domain.MatchHistoryRow, field string) bool {
	switch field {
	case "outcome_code":
		return a.OutcomeCode < b.OutcomeCode
	case "team_mmr":
		return cmpNullFloat(a.TeamMMR, b.TeamMMR)
	case "delta_mmr":
		return cmpNullFloat(a.DeltaMMR, b.DeltaMMR)
	case "win_rate_hist":
		return cmpNullFloat(a.WinRateHist, b.WinRateHist)
	case "performance_score_relative":
		return cmpNullInt(a.PerformanceScoreRelative, b.PerformanceScoreRelative)
	case "kda":
		return cmpNullFloat(a.KDA, b.KDA)
	case scopeKills:
		return a.Kills < b.Kills
	default: // start_time
		return a.StartTime.Before(b.StartTime)
	}
}

func paginate(items []domain.MatchHistoryRow, req domain.PaginationRequest) (domain.PaginationMeta, []domain.MatchHistoryRow) {
	total := len(items)
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 25
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	// Init à [] plutôt que nil : un slice nil sérialise en JSON `null` et crashe
	// le front sur .filter() / .map(). Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
	pageItems := []domain.MatchHistoryRow{}
	if start < total {
		pageItems = items[start:end]
	}
	meta := domain.PaginationMeta{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasNext:  page < totalPages,
		HasPrev:  page > 1,
	}
	return meta, pageItems
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func outcomeLabel(code int) string {
	if lbl, ok := outcomeLabels[code]; ok {
		return lbl
	}
	return "-"
}

func formatDateFR(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("02/01/2006 15:04")
}

// zeroDuration est le format MM:SS retourné quand la durée est nulle ou invalide.
const zeroDuration = "0:00"

func formatLifeSeconds(secs *float64) string {
	if secs == nil {
		return zeroDuration
	}
	total := int(*secs)
	if total < 0 {
		return zeroDuration
	}
	mm := total / 60
	ss := total % 60
	return fmt.Sprintf("%d:%02d", mm, ss)
}

func buildPeriodLabel(f domain.FilterContextInput) *string {
	if f.FilterMode == scopeSessions {
		return nil
	}
	p := f.Period
	if p.StartDate == nil && p.EndDate == nil {
		return nil
	}
	var parts []string
	if p.StartDate != nil {
		parts = append(parts, p.StartDate.Format("02/01/2006"))
	}
	if p.EndDate != nil {
		parts = append(parts, p.EndDate.Format("02/01/2006"))
	}
	lbl := strings.Join(parts, " → ")
	return &lbl
}

func ptrStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func cmpNullFloat(a, b *float64) bool {
	if a == nil {
		return b != nil
	}
	if b == nil {
		return false
	}
	return *a < *b
}

func cmpNullInt(a, b *int) bool {
	if a == nil {
		return b != nil
	}
	if b == nil {
		return false
	}
	return *a < *b
}
