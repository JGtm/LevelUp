// Package service - synthesis_service_canonical.go : filtres + best refs +
// overview canonical pour la page Synthese. Decoupe de synthesis_service.go
// (god-file split, refactor 2026-05-27).
package service

import (
	"fmt"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func filterSynthesisByPeriodCanonical(
	rows []canonical.PlayerMatchRow,
	period string,
	startDate string,
	endDate string,
) ([]canonical.PlayerMatchRow, []string, []string) {
	applied := []string{}
	ignored := []string{}
	const dayFmt = "2006-01-02"

	if startDate != "" || endDate != "" {
		var start, end *time.Time
		if startDate != "" {
			if t, err := time.Parse(dayFmt, startDate); err == nil {
				start = &t
			}
		}
		if endDate != "" {
			if t, err := time.Parse(dayFmt, endDate); err == nil {
				endOfDay := t.Add(24*time.Hour - time.Second)
				end = &endOfDay
			}
		}
		if start != nil || end != nil {
			applied = append(applied, fmt.Sprintf("periode=%s>%s", startDate, endDate))
			filtered := make([]canonical.PlayerMatchRow, 0, len(rows))
			for _, r := range rows {
				at := r.Summary.StartedAtUTC
				if start != nil && at.Before(*start) {
					continue
				}
				if end != nil && at.After(*end) {
					continue
				}
				filtered = append(filtered, r)
			}
			return filtered, applied, ignored
		}
	}

	var cutoff *time.Time
	now := time.Now().UTC()
	switch period {
	case "1w":
		t := now.AddDate(0, 0, -7)
		cutoff = &t
		applied = append(applied, fmt.Sprintf("periode=%s", period))
	case "1m":
		t := now.AddDate(0, -1, 0)
		cutoff = &t
		applied = append(applied, fmt.Sprintf("periode=%s", period))
	case "1y":
		t := now.AddDate(-1, 0, 0)
		cutoff = &t
		applied = append(applied, fmt.Sprintf("periode=%s", period))
	case "2y":
		t := now.AddDate(-2, 0, 0)
		cutoff = &t
		applied = append(applied, fmt.Sprintf("periode=%s", period))
	default:
	}

	if cutoff == nil {
		return rows, applied, ignored
	}
	filtered := make([]canonical.PlayerMatchRow, 0, len(rows))
	for _, r := range rows {
		if !r.Summary.StartedAtUTC.Before(*cutoff) {
			filtered = append(filtered, r)
		}
	}
	return filtered, applied, ignored
}

// bestTracker garde en mémoire le match record (match_id + valeur max) d'une
// métrique au fil de l'itération sur les rows canonical. En cas d'égalité,
// le premier match rencontré est conservé (rows pré-triés par date).
type bestTracker struct {
	matchID string
	value   float64
	seen    bool
}

func (b *bestTracker) update(matchID string, v float64) {
	if !b.seen || v > b.value {
		b.matchID = matchID
		b.value = v
		b.seen = true
	}
}

func (b *bestTracker) toRef() *domain.BestMatchRef {
	if !b.seen || b.value <= 0 {
		return nil
	}
	return &domain.BestMatchRef{MatchID: b.matchID, Value: b.value}
}

// synthesisBestRefs agrège les "Top X" cliquables exposés par SynthesisOverview.
type synthesisBestRefs struct {
	kills, kda, perf, accuracy, damage, killingSpree *domain.BestMatchRef
	headshots, personalScore                         *domain.BestMatchRef
}

// computeSynthesisBestRefs identifie le match record pour chaque métrique
// exposée comme carte "Top X" / "Meilleur X" côté front (Synthesis page).
func computeSynthesisBestRefs(rows []canonical.PlayerMatchRow, provideSpree bool) synthesisBestRefs {
	var trK, trKDA, trPerf, trAcc, trDmg, trSpree, trHS, trPS bestTracker
	for _, r := range rows {
		// « Meilleures stats » = records PvP exploitables : on exclut les matchs
		// non terminés (DNF / abandon, stats tronquées) et le PvE/Firefight (barème
		// d'IA non comparable au PvP). Les totaux de l'overview, eux, comptent tout.
		if r.Self.Outcome == canonical.OutcomeDNF {
			continue
		}
		if r.Summary.IsPvE != nil && *r.Summary.IsPvE {
			continue
		}
		id := r.Summary.MatchID
		if r.Self.Kills != nil {
			trK.update(id, float64(*r.Self.Kills))
		}
		if r.Self.KDA != nil {
			trKDA.update(id, *r.Self.KDA)
		}
		if r.Enrichment.PerformanceScore != nil {
			trPerf.update(id, *r.Enrichment.PerformanceScore)
		}
		if r.Self.Accuracy != nil {
			trAcc.update(id, *r.Self.Accuracy)
		}
		if r.Self.DamageDealt != nil {
			trDmg.update(id, float64(*r.Self.DamageDealt))
		}
		// MaxKillingSpree : ignorée quand le titre ne la porte pas (Halo 5) → la carte
		// « meilleur max killing spree » est masquée (killingSpree reste nil) plutôt
		// que de fabriquer une valeur. Cf. games.ProvidesMaxKillingSpree.
		if provideSpree && r.Self.MaxKillingSpree != nil {
			trSpree.update(id, float64(*r.Self.MaxKillingSpree))
		}
		if r.Self.HeadshotKills != nil {
			trHS.update(id, float64(*r.Self.HeadshotKills))
		}
		if r.Self.PersonalScore != nil {
			trPS.update(id, float64(*r.Self.PersonalScore))
		}
	}
	return synthesisBestRefs{
		kills:         trK.toRef(),
		kda:           trKDA.toRef(),
		perf:          trPerf.toRef(),
		accuracy:      trAcc.toRef(),
		damage:        trDmg.toRef(),
		killingSpree:  trSpree.toRef(),
		headshots:     trHS.toRef(),
		personalScore: trPS.toRef(),
	}
}

// buildSynthesisOverviewCanonical est la variante canonical de
// buildSynthesisOverview. Lit Self.Kills/Deaths/Outcome/KDA depuis
// canonical au lieu de SynthesisMatchRow.{Kills,Deaths,Outcome,KDA}.
func buildSynthesisOverviewCanonical(rows []canonical.PlayerMatchRow, soloKPIs domain.SynthesisKPIs, provideSpree bool) domain.SynthesisOverview {
	var totalKills, totalDeaths, totalAssists, totalWins, totalLosses, totalTies, totalDNF int
	var winStreak, maxStreak int

	for _, r := range rows {
		k := 0
		if r.Self.Kills != nil {
			k = *r.Self.Kills
		}
		d := 0
		if r.Self.Deaths != nil {
			d = *r.Self.Deaths
		}
		if r.Self.Assists != nil {
			totalAssists += *r.Self.Assists
		}
		totalKills += k
		totalDeaths += d
		switch r.Self.Outcome {
		case canonical.OutcomeWin:
			totalWins++
			winStreak++
			if winStreak > maxStreak {
				maxStreak = winStreak
			}
		case canonical.OutcomeLoss:
			totalLosses++
			winStreak = 0
		case canonical.OutcomeTie:
			totalTies++
			winStreak = 0
		default:
			totalDNF++
			winStreak = 0
		}
	}

	n := len(rows)
	ov := domain.SynthesisOverview{
		TotalMatches:     n,
		TotalWins:        totalWins,
		TotalLosses:      totalLosses,
		TotalTies:        totalTies,
		TotalDNF:         totalDNF,
		TotalKills:       totalKills,
		TotalDeaths:      totalDeaths,
		TotalAssists:     totalAssists,
		WinRate:          soloKPIs.WinRate,
		LongestWinStreak: maxStreak,
	}
	if soloKPIs.KDRatio != nil {
		ov.AvgKDA = soloKPIs.KDRatio
	}
	if n > 0 {
		avgKills := float64(totalKills) / float64(n)
		avgDeaths := float64(totalDeaths) / float64(n)
		ov.AvgKills = &avgKills
		ov.AvgDeaths = &avgDeaths
		totalKDR := analysis.KDR(totalKills, totalDeaths)
		ov.TotalKDR = &totalKDR
	}
	if soloKPIs.PerformanceScore != nil {
		ov.AvgPerfScore = soloKPIs.PerformanceScore
	}

	applyBestRefsToOverview(&ov, computeSynthesisBestRefs(rows, provideSpree))
	return ov
}

// applyBestRefsToOverview projette les refs calculés vers l'overview et
// alimente aussi les champs scalaires legacy (best_kills_match / best_kda_match)
// pour préserver le contrat OpenAPI existant.
func applyBestRefsToOverview(ov *domain.SynthesisOverview, refs synthesisBestRefs) {
	ov.BestKillsRef = refs.kills
	ov.BestKDARef = refs.kda
	ov.BestPerfRef = refs.perf
	ov.BestAccuracyRef = refs.accuracy
	ov.BestDamageRef = refs.damage
	ov.BestKillingSpreeRef = refs.killingSpree
	ov.BestHeadshotsRef = refs.headshots
	ov.BestPersonalScoreRef = refs.personalScore
	if refs.kills != nil {
		k := int(refs.kills.Value)
		ov.BestKillsMatch = &k
	}
	if refs.kda != nil {
		v := refs.kda.Value
		ov.BestKDAMatch = &v
	}
}

// buildHighlightsPreviewCanonical est la variante canonical de
// buildHighlightsPreview. Top/pire matchs sur les mÃªmes critÃ¨res
