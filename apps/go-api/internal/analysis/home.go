// Package analysis â€” home.go : faÃ§ade minimale du package home.
//
// Ce fichier centralise les helpers math/agrÃ©gation transverses utilisÃ©s par
// l'ensemble des sous-modules home (home_kpis, home_highlights, home_recent,
// home_sessions, home_canonical_*) ainsi que la projection des mÃ©dias rÃ©cents
// (BuildRecentMedia), qui n'a pas de dÃ©pendance forte avec un sous-module.
//
// Les autres responsabilitÃ©s ont Ã©tÃ© extraites :
//   - home_locale.go     : constantes outcome/color/tone, helpers locale, labels,
//     narrative badges, score label, normalizeHomeModeLabel,
//     regex UUID, copyOptionalString, optionalStringValue.
//   - home_kpis.go       : ComputeKPIs, ComputeTrend, BuildHeroCard,
//     BuildSpartanIdentity + helpers de rank/skill peak.
//   - home_highlights.go : selectHighlightWindow, BuildHighlights, tuiles
//     MaÃ®trise/PerMinute/SÃ©rie + helpers de couleur/sÃ©lection.
//   - home_recent.go     : BuildRecentMatches*, mapImageURLFromRegistry,
//     mmrDelta, float64PtrVal, intPtrIfPos.
//   - home_sessions.go   : BuildSessionSummaries, BuildSessionSummary,
//     distinctSessionLabels, latestSessionLabel,
//     earliestStartTime, latestEndTime.
//
// Fonctions stateless : entrÃ©e = slices de domain rows, sortie = blocs JSON.
// Aucun accÃ¨s DB, aucun import Streamlit.
package analysis

import (
	"math"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// ---------------------------------------------------------------------------
// BuildRecentMedia â€” mÃ©dias rÃ©cents
// ---------------------------------------------------------------------------

// BuildRecentMedia transforme les lignes DuckDB en items de mÃ©dias rÃ©cents.
func BuildRecentMedia(media []domain.HomeMediaRow, limit int) []domain.RecentMediaItem {
	if len(media) == 0 {
		return nil
	}
	if len(media) > limit {
		media = media[:limit]
	}
	items := make([]domain.RecentMediaItem, 0, len(media))
	for _, m := range media {
		if m.FileName == "" {
			continue
		}
		items = append(items, domain.RecentMediaItem{
			Basename:       m.FileName,
			MatchID:        m.MatchID,
			MatchStartTime: m.MatchStartTime,
		})
	}
	return items
}

// ---------------------------------------------------------------------------
// Helpers math transverses (utilisÃ©s par home_kpis, home_sessions et tests)
// ---------------------------------------------------------------------------

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round3(v float64) float64 { return math.Round(v*1000) / 1000 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }

func meanRatio(matches []legacymatch.HomeMatchRow) *float64 {
	var sum, count float64
	for _, m := range matches {
		if m.Ratio != nil {
			sum += *m.Ratio
			count++
		}
	}
	if count == 0 {
		return nil
	}
	v := sum / count
	return &v
}

func meanAccuracy(matches []legacymatch.HomeMatchRow) *float64 {
	var sum, count float64
	for _, m := range matches {
		if m.Accuracy != nil {
			sum += *m.Accuracy
			count++
		}
	}
	if count == 0 {
		return nil
	}
	v := sum / count
	return &v
}

func winRate(matches []legacymatch.HomeMatchRow) float64 {
	if len(matches) == 0 {
		return 0
	}
	var wins int
	for _, m := range matches {
		if m.Outcome == homeOutcomeWin {
			wins++
		}
	}
	return WinRate(wins, len(matches))
}

func bestRatioMatch(matches []legacymatch.HomeMatchRow) *legacymatch.HomeMatchRow {
	var best *legacymatch.HomeMatchRow
	for i := range matches {
		if matches[i].Ratio == nil {
			continue
		}
		if best == nil || *matches[i].Ratio > *best.Ratio {
			best = &matches[i]
		}
	}
	return best
}
