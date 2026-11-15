package service

import (
	"strconv"
	"testing"
	"time"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

// Témoins de caractérisation des cadences par minute du package service : valeurs exactes (au bit près) sur des
// entrées fixes dont une durée non multiple de 60, une durée nulle et une durée absente.

func temoinF(t *testing.T, label string, v float64) {
	t.Helper()
	t.Logf("%s = %s", label, strconv.FormatFloat(v, 'g', -1, 64))
}

func temoinRows() []legacymatch.StatsMatchRow {
	ip := func(v int) *int { return &v }
	base := time.Date(2026, 3, 2, 18, 0, 0, 0, time.UTC)
	return []legacymatch.StatsMatchRow{
		{StartTime: base, PersonalScore: ip(1234), TimePlayedSeconds: ip(937)},
		{StartTime: base.Add(time.Hour), PersonalScore: ip(987), TimePlayedSeconds: ip(1051)},
		{StartTime: base.Add(2 * time.Hour), PersonalScore: ip(600), TimePlayedSeconds: ip(60)},
		{StartTime: base.Add(3 * time.Hour), PersonalScore: ip(50), TimePlayedSeconds: ip(300)},
		{StartTime: base.Add(4 * time.Hour), PersonalScore: ip(500), TimePlayedSeconds: ip(0)},
		{StartTime: base.Add(5 * time.Hour), PersonalScore: ip(300)},
		{StartTime: base.Add(6 * time.Hour), TimePlayedSeconds: ip(777)},
	}
}

func TestTemoinMinute_OngletPrecision(t *testing.T) {
	got := buildAccuracyTab(temoinRows())
	t.Logf("precision.len = %d", len(got.ScorePerMin))
	for i, v := range got.ScorePerMin {
		temoinF(t, "precision.spm"+strconv.Itoa(i), v)
	}
}

func TestTemoinMinute_BucketsScoreParMinute(t *testing.T) {
	got := buildScorePerMinBuckets(temoinRows())
	t.Logf("buckets.len = %d", len(got))
	for _, b := range got {
		t.Logf("buckets [%v,%v) = %d", b.BucketLower, b.BucketUpper, b.Count)
	}
}

func TestTemoinMinute_OngletIntensite(t *testing.T) {
	got := buildIntensityTab(temoinRows())
	t.Logf("intensite.len = %d", len(got.ScorePerMinData))
	for _, p := range got.ScorePerMinData {
		temoinF(t, "intensite.spm"+strconv.Itoa(p.Index), p.Value)
	}
}

func TestTemoinMinute_ContributionsParMinute(t *testing.T) {
	ip := func(v int) *int { return &v }
	mk := func(secs *int, k, d, a int) canonical.PlayerMatchRow {
		var r canonical.PlayerMatchRow
		r.Summary.DurationSeconds = secs
		r.Self.Kills, r.Self.Deaths, r.Self.Assists = ip(k), ip(d), ip(a)
		return r
	}
	got := BuildPerMinuteStats(map[string][]canonical.PlayerMatchRow{
		"alpha": {mk(ip(937), 11, 7, 3), mk(ip(1051), 23, 13, 5), mk(ip(59), 4, 2, 1), mk(ip(0), 9, 9, 9), mk(nil, 8, 8, 8)},
		"bravo": {mk(ip(0), 9, 4, 2)},
		"carol": {mk(ip(601), 12, 6, 4)},
	})
	t.Logf("contrib.len = %d", len(got.Datapoints))
	for _, p := range got.Datapoints {
		for _, k := range []string{"kills_per_min", "deaths_per_min", "assists_per_min"} {
			temoinF(t, "contrib."+p.Category+"."+k, p.Components[k])
		}
	}
}
