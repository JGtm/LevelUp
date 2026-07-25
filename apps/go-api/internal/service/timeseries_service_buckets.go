// Package service — timeseries_service_buckets.go : builders des
// distribution buckets (Accuracy, ScorePerMin, Life, PersonalScore,
// MaxKillingSpree, PerfScore, RollingWR). Decoupe de timeseries_service.go
// (god-file split, refactor 2026-05-27).
package service

import (
	"context"
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

func buildAccuracyBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const binWidth = 5.0      // 5 % par bin
	counts := make([]int, 21) // bins 0-5, 5-10, …, 95-100, 100+

	for _, m := range matches {
		if m.Accuracy == nil {
			continue
		}
		pct := *m.Accuracy
		if pct < 0 {
			pct = 0
		}
		idx := int(pct / binWidth)
		if idx >= len(counts) {
			idx = len(counts) - 1
		}
		counts[idx]++
	}

	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for i, c := range counts {
		if c == 0 {
			continue
		}
		start := float64(i) * binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start, BucketUpper: start + binWidth, Count: c,
		})
	}
	return buckets
}

// buildScorePerMinBuckets crÃ©e des buckets de 10 pts/min pour la distribution score/min.
func buildScorePerMinBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const binWidth = 10.0
	counts := make(map[int]int)

	for _, m := range matches {
		if m.PersonalScore == nil || m.TimePlayedSeconds == nil || *m.TimePlayedSeconds == 0 {
			continue
		}
		spm := float64(*m.PersonalScore) / (float64(*m.TimePlayedSeconds) / 60.0)
		idx := int(spm / binWidth)
		counts[idx]++
	}

	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for idx, c := range counts {
		start := float64(idx) * binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start, BucketUpper: start + binWidth, Count: c,
		})
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].BucketLower < buckets[j].BucketLower })
	return buckets
}

// matchAvgLifeSeconds retourne la durée de vie moyenne du match et la source
// utilisée. Ordre de préférence (B1, 2026-07-25) :
//
//  1. AvgLifeSeconds — valeur RÉELLE de l'API (match_participants.avg_life_seconds),
//     seule mesure fiable : elle tient compte des respawns effectifs, pas d'une
//     division par (morts+1).
//  2. Repli `time_played / (morts + 1)` — proxy historique, conservé pour les
//     matchs antérieurs à la colonne (et les titres qui ne la fournissent pas).
//
// `isReal` distingue les deux : le caller compte les replis et les trace (jamais de
// dégradation muette). `ok` est faux quand aucune des deux sources n'est exploitable.
func matchAvgLifeSeconds(m legacymatch.StatsMatchRow) (life float64, isReal bool, ok bool) {
	if m.AvgLifeSeconds != nil && *m.AvgLifeSeconds > 0 {
		return *m.AvgLifeSeconds, true, true
	}
	if m.TimePlayedSeconds != nil && *m.TimePlayedSeconds > 0 {
		return float64(*m.TimePlayedSeconds) / float64(m.Deaths+1), false, true
	}
	return 0, false, false
}

// buildLifeBuckets crée des buckets de 5s pour la distribution de la durée de vie
// moyenne. Source = `avg_life_seconds` (valeur réelle de l'API) ; repli sur le
// proxy `time_played/(morts+1)` quand elle est absente (matchs anciens). Le
// nombre de replis est tracé en Debug — un histogramme majoritairement issu du
// proxy est un symptôme de backfill manquant, pas un état normal.
func buildLifeBuckets(ctx context.Context, matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const binWidth = 5.0
	maxLife := 0.0
	fallbacks, used := 0, 0
	for _, m := range matches {
		life, isReal, ok := matchAvgLifeSeconds(m)
		if !ok {
			continue
		}
		used++
		if !isReal {
			fallbacks++
		}
		if life > maxLife {
			maxLife = life
		}
	}
	if fallbacks > 0 {
		slog.DebugContext(ctx, "timeseries: durée de vie estimée par repli (proxy temps_joué/(morts+1))",
			"matches_fallback", fallbacks, "matches_used", used, "matches_total", len(matches))
	}
	if maxLife == 0 {
		return []domain.DistributionBucket{}
	}
	numBins := int(maxLife/binWidth) + 1
	counts := make(map[int]int)
	for _, m := range matches {
		life, _, ok := matchAvgLifeSeconds(m)
		if !ok {
			continue
		}
		idx := int(life / binWidth)
		if idx > numBins {
			idx = numBins
		}
		counts[idx]++
	}
	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for idx, c := range counts {
		start := float64(idx) * binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start, BucketUpper: start + binWidth, Count: c,
		})
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].BucketLower < buckets[j].BucketLower })
	return buckets
}

// buildPersonalScoreBuckets crée des buckets de 250 points pour la
// distribution du score personnel par match (PersonalScore — synced).
func buildPersonalScoreBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const binWidth = 250.0
	counts := make(map[int]int)
	for _, m := range matches {
		if m.PersonalScore == nil {
			continue
		}
		score := float64(*m.PersonalScore)
		if score < 0 {
			score = 0
		}
		idx := int(score / binWidth)
		counts[idx]++
	}
	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for idx, c := range counts {
		start := float64(idx) * binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start, BucketUpper: start + binWidth, Count: c,
		})
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].BucketLower < buckets[j].BucketLower })
	return buckets
}

// buildMaxKillingSpreeBuckets crée des buckets entiers (binWidth=1) pour la
// distribution du max killing spree par match. Skip si nil ou ≤0.
func buildMaxKillingSpreeBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	counts := make(map[int]int)
	for _, m := range matches {
		if m.MaxKillingSpree == nil || *m.MaxKillingSpree <= 0 {
			continue
		}
		counts[*m.MaxKillingSpree]++
	}
	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for v, c := range counts {
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: float64(v), BucketUpper: float64(v + 1), Count: c,
		})
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].BucketLower < buckets[j].BucketLower })
	return buckets
}

// buildPerfScoreBuckets crée des buckets de 5 points pour la distribution du
// performance_score (PerfScoreComputed). Range attendue [0, 100]. Les matchs
// sans score (sync incomplet) sont skippés.
func buildPerfScoreBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const binWidth = 5.0
	counts := make([]int, 21) // 0-5, 5-10, …, 95-100, 100+
	for _, m := range matches {
		if m.PerfScoreComputed == nil {
			continue
		}
		score := *m.PerfScoreComputed
		if score < 0 {
			score = 0
		}
		idx := int(score / binWidth)
		if idx >= len(counts) {
			idx = len(counts) - 1
		}
		counts[idx]++
	}
	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for i, c := range counts {
		if c == 0 {
			continue
		}
		start := float64(i) * binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start, BucketUpper: start + binWidth, Count: c,
		})
	}
	return buckets
}

// buildRollingWRBuckets crÃ©e des buckets de 5 % pour la distribution du win-rate glissant (fenÃªtre 14).
func buildRollingWRBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const (
		window   = 14
		binWidth = 5.0
	)
	counts := make([]int, 21) // bins 0-5, 5-10, â€¦, 95-100

	for i := range matches {
		start := i - window + 1
		if start < 0 {
			start = 0
		}
		wins := 0
		for j := start; j <= i; j++ {
			if matches[j].Outcome != nil && *matches[j].Outcome == analysis.OutcomeWin {
				wins++
			}
		}
		total := i - start + 1
		// TODO(expiry:2026-12-31) P4 ADR 0006 : retirer *100 (convention API canonique 0..1).
		wr := analysis.WinRate(wins, total) * 100
		idx := int(wr / binWidth)
		if idx >= len(counts) {
			idx = len(counts) - 1
		}
		counts[idx]++
	}

	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for i, c := range counts {
		if c == 0 {
			continue
		}
		start := float64(i) * binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start, BucketUpper: start + binWidth, Count: c,
		})
	}
	return buckets
}
