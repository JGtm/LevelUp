// Package service - teammates_service_kpis.go : teammate options + row
// builders + KPIs (squad/synthesis) + safeDiv/round2 + enrichMapBreakdown +
// squadStatsToWinTotal. Decoupe de teammates_service.go (god-file split,
// refactor 2026-05-27).
package teammates

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

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

	// Ãƒâ€°tape 2 : fallback Ã¢â‚¬â€ rÃƒÂ©soudre via shared.xuid_aliases (LookupXUIDByGamertag) pour les gamertags
	// hors top 50 (utilisateur qui a 50+ coÃƒÂ©quipiers rÃƒÂ©guliers OU saisie libre
	// dans la combobox). encounterCount reste 0 Ã¢â‚¬â€ recalculÃƒÂ© depuis squadMatches
	// plus bas si on charge effectivement les matchs.
	if teammateXUID == "" {
		resolved, found, err := s.repo.LookupXUIDByGamertag(ctx, gamertag)
		if err != nil {
			slog.WarnContext(ctx, "teammates_gamertag_lookup_failed",
				"player_xuid", playerXUID,
				"gamertag", gamertag,
				"err", err,
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
			"gamertag", gamertag, "err", err)
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
		// m.Accuracy est déjà en pourcentage 0..100 (match_participants.accuracy,
		// cf. sync/transforms.go). Le ×100 historique (quand l'accuracy était 0..1)
		// produisait du 0..10000 → radar plafonné. La moyenne reste en 0..100.
		v := round2(accSum / float64(accCount))
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

// enrichMapBreakdownWithSquadStats injecte HistoricalWinRate et
// HistoricalPerformanceAvg depuis la map de stats agrégées par le repo
// (LoadMapStatsForSquad). Une seule jointure par MapID — fallback MapUI si
// le map_id n'est pas exposé (cas dégradé).
//
// Aucune ligne n'est ajoutée : seules les MapBreakdownRow déjà présentes
// dans la session courante (computeMapBreakdown) reçoivent leur enrichissement.
func enrichMapBreakdownWithSquadStats(rows []domain.MapBreakdownRow, stats map[string]domain.MapSquadStats) []domain.MapBreakdownRow {
	if len(stats) == 0 {
		return rows
	}
	for i := range rows {
		key := rows[i].MapID
		if key == "" {
			key = rows[i].MapUI
		}
		s, ok := stats[key]
		if !ok || s.Total == 0 {
			continue
		}
		wr := round2(float64(s.Wins) / float64(s.Total))
		rows[i].HistoricalWinRate = &wr
		if s.PerfAvg != nil {
			v := *s.PerfAvg
			rows[i].HistoricalPerformanceAvg = &v
		}
	}
	return rows
}

// squadStatsToWinTotal convertit la map repo en format compatible avec
// buildSquadMatchHistory (signature historique : map[mapID][2]int{wins,total}).
func squadStatsToWinTotal(stats map[string]domain.MapSquadStats) map[string][2]int {
	if len(stats) == 0 {
		return nil
	}
	out := make(map[string][2]int, len(stats))
	for k, s := range stats {
		out[k] = [2]int{s.Wins, s.Total}
	}
	return out
}

// enrichSquadMatchAssets enrichit MapUI et PlaylistName des rows avec les traductions FR
// depuis metadata.asset_translations (calqué sur home_repo.enrichHomeMatchTranslations).
