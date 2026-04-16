// Package analysis — squad_timeseries.go : série temporelle des performances escouade.
package analysis

import (
	"fmt"
	"math"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
)

// =============================================================================
// Série temporelle escouade — miroir de compute_squad_timeseries()
// =============================================================================

// ComputeSquadTimeseries calcule la série temporelle des performances escouade.
//
// Groupe par session_id si disponible, sinon par période temporelle (semaine → mois).
// maxBuckets limite le nombre de points retournés.
// Miroir de Python compute_squad_timeseries() dans _performance_squad.py.
func ComputeSquadTimeseries(rows []domain.SquadMatchRow, maxBuckets int) []domain.SquadTimeseriesPoint {
	if len(rows) == 0 {
		return nil
	}

	// Essayer le groupement par session_id.
	bySession := groupSquadBySession(rows)
	if len(bySession) > 0 {
		return bySession
	}

	// Fallback : groupement temporel.
	return groupSquadByTime(rows, maxBuckets)
}

// groupSquadBySession regroupe les matchs par session_id.
func groupSquadBySession(rows []domain.SquadMatchRow) []domain.SquadTimeseriesPoint {
	// Vérifier que des sessions existent.
	hasSession := false
	for _, r := range rows {
		if r.SessionID != nil {
			hasSession = true
			break
		}
	}
	if !hasSession {
		return nil
	}

	type sessionAgg struct {
		label        string
		sortKey      int
		totalPerf    float64
		totalMMR     float64
		wins, losses int
		count        int
		countPerf    int
	}
	bySession := make(map[int]*sessionAgg)

	for _, r := range rows {
		if r.SessionID == nil {
			continue
		}
		sid := *r.SessionID
		agg, ok := bySession[sid]
		if !ok {
			label := fmt.Sprintf("%d", sid)
			if r.SessionLabel != nil && len(*r.SessionLabel) >= 5 {
				label = (*r.SessionLabel)[:5]
			}
			agg = &sessionAgg{label: label, sortKey: sid}
			bySession[sid] = agg
		}
		if r.PerformanceScore != nil {
			agg.totalPerf += *r.PerformanceScore
			agg.countPerf++
		}
		agg.totalMMR += r.TeamMMR
		agg.count++
		if r.Outcome == domain.OutcomeWin {
			agg.wins++
		} else if r.Outcome == domain.OutcomeLoss {
			agg.losses++
		}
	}

	// Trier par sortKey.
	ids := make([]int, 0, len(bySession))
	for id := range bySession {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	points := make([]domain.SquadTimeseriesPoint, 0, len(ids))
	for _, id := range ids {
		agg := bySession[id]
		var perfPtr *float64
		if agg.countPerf > 0 {
			v := math.Round((agg.totalPerf/float64(agg.countPerf))*10) / 10
			perfPtr = &v
		}
		mmrAvg := agg.totalMMR / float64(agg.count)
		p := domain.SquadTimeseriesPoint{
			BucketLabel: agg.label,
			SquadPerf:   perfPtr,
			TeamMMRAvg:  &mmrAvg,
			MatchCount:  agg.count,
		}
		total := agg.wins + agg.losses
		if total > 0 {
			wr := math.Round(float64(agg.wins)/float64(total)*1000) / 10
			p.WinRate = &wr
		}
		points = append(points, p)
	}
	return points
}

// groupSquadByTime regroupe les matchs par période temporelle (semaine → mois).
func groupSquadByTime(rows []domain.SquadMatchRow, maxBuckets int) []domain.SquadTimeseriesPoint {
	// Tenter week, puis month.
	for _, trunc := range []string{"week", "month"} {
		points := bucketByTime(rows, trunc)
		if len(points) <= maxBuckets {
			return points
		}
	}
	return bucketByTime(rows, "month")
}

// bucketByTime groupe les rows par période (week ou month).
func bucketByTime(rows []domain.SquadMatchRow, period string) []domain.SquadTimeseriesPoint {
	type bucketAgg struct {
		bucketTime   time.Time
		label        string
		totalPerf    float64
		totalMMR     float64
		wins, losses int
		count        int
		countPerf    int
	}
	byBucket := make(map[time.Time]*bucketAgg)

	for _, r := range rows {
		var bucket time.Time
		var label string
		if period == "week" {
			// Tronquer au lundi de la semaine.
			wd := int(r.StartTime.Weekday())
			if wd == 0 {
				wd = 7
			}
			bucket = r.StartTime.AddDate(0, 0, -(wd - 1)).Truncate(24 * time.Hour)
			label = bucket.Format("02/01")
		} else {
			bucket = time.Date(r.StartTime.Year(), r.StartTime.Month(), 1, 0, 0, 0, 0, time.UTC)
			label = bucket.Format("01/2006")
		}

		agg, ok := byBucket[bucket]
		if !ok {
			agg = &bucketAgg{bucketTime: bucket, label: label}
			byBucket[bucket] = agg
		}
		if r.PerformanceScore != nil {
			agg.totalPerf += *r.PerformanceScore
			agg.countPerf++
		}
		agg.totalMMR += r.TeamMMR
		agg.count++
		if r.Outcome == domain.OutcomeWin {
			agg.wins++
		} else if r.Outcome == domain.OutcomeLoss {
			agg.losses++
		}
	}

	// Trier par date.
	buckets := make([]time.Time, 0, len(byBucket))
	for b := range byBucket {
		buckets = append(buckets, b)
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].Before(buckets[j]) })

	points := make([]domain.SquadTimeseriesPoint, 0, len(buckets))
	for _, b := range buckets {
		agg := byBucket[b]
		var perfPtr *float64
		if agg.countPerf > 0 {
			v := math.Round((agg.totalPerf/float64(agg.countPerf))*10) / 10
			perfPtr = &v
		}
		mmrAvg := agg.totalMMR / float64(agg.count)
		p := domain.SquadTimeseriesPoint{
			BucketLabel: agg.label,
			SquadPerf:   perfPtr,
			TeamMMRAvg:  &mmrAvg,
			MatchCount:  agg.count,
		}
		total := agg.wins + agg.losses
		if total > 0 {
			wr := math.Round(float64(agg.wins)/float64(total)*1000) / 10
			p.WinRate = &wr
		}
		points = append(points, p)
	}
	return points
}
