// Package service - teammates_service_assets.go : asset enrichment +
// collectUniqueIDs + modeLabel + computeMapBreakdown + collectModeENs +
// buildSquadMatchHistory + buildMatchSeries. Decoupe de teammates_service.go
// (god-file split, refactor 2026-05-27).
package service

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

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
			key = tsLabelUnknown
		}
		if _, ok := m[key]; !ok {
			lbl := r.MapUI
			if lbl == "" {
				lbl = tsLabelUnknown
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
//
// mapWR : (wins, total) par MapID sur l'historique complet du joueur
// principal — sert à injecter le taux historique par carte. Si nil ou clé
// absente, WinRateHist reste nil (la cellule front affiche "—").
func buildSquadMatchHistory(matches []domain.SquadMatchRow, modeFR map[string]string, mapWR map[string][2]int) []domain.SquadMatchHistoryRow {
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
		var winRate *float64
		var winRateTotal *int
		if mapWR != nil {
			key := m.MapID
			if key == "" {
				key = m.MapName
			}
			if key != "" {
				if entry, ok := mapWR[key]; ok && entry[1] > 0 {
					v := round2(float64(entry[0]) / float64(entry[1]))
					winRate = &v
					total := entry[1]
					winRateTotal = &total
				}
			}
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
			DurationSeconds:  m.DurationSeconds,
			GameplayDurationSeconds: m.GameplayDurationSeconds,
			WinRateHist:      winRate,
			WinRateHistTotal: winRateTotal,
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
