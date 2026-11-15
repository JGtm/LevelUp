package service

import (
	"strconv"
	"testing"
	"time"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

// Témoins de caractérisation des cadences par minute du package service : valeurs attendues au bit près
// (chaîne 'g', -1) sur des entrées fixes dont une durée non multiple de 60, une durée nulle et une durée
// absente. Les valeurs sont celles que les sites rendaient avant leur passage par analysis.PerMinute.

func temoinF(t *testing.T, label string, v float64, want string) {
	t.Helper()
	got := strconv.FormatFloat(v, 'g', -1, 64)
	t.Logf("%s = %s", label, got)
	if got != want {
		t.Errorf("%s = %s, attendu %s", label, got, want)
	}
}

func temoinN(t *testing.T, label string, got, want int) {
	t.Helper()
	t.Logf("%s = %d", label, got)
	if got != want {
		t.Errorf("%s = %d, attendu %d", label, got, want)
	}
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
	temoinN(t, "precision.len", len(got.ScorePerMin), 4)
	for i, want := range []string{"79", "56.3", "600", "10"} {
		if i < len(got.ScorePerMin) {
			temoinF(t, "precision.spm"+strconv.Itoa(i), got.ScorePerMin[i], want)
		}
	}
}

func TestTemoinMinute_BucketsScoreParMinute(t *testing.T) {
	got := buildScorePerMinBuckets(temoinRows())
	temoinN(t, "buckets.len", len(got), 4)
	for i, lower := range []float64{10, 50, 70, 600} {
		if i >= len(got) {
			break
		}
		b := got[i]
		if b.BucketLower != lower || b.BucketUpper != lower+10 {
			t.Errorf("bucket %d = [%v,%v), attendu [%v,%v)", i, b.BucketLower, b.BucketUpper, lower, lower+10)
		}
		t.Logf("buckets [%v,%v) = %d", b.BucketLower, b.BucketUpper, b.Count)
		if b.Count != 1 {
			t.Errorf("bucket %d : %d matchs, attendu 1", i, b.Count)
		}
	}
}

func TestTemoinMinute_OngletIntensite(t *testing.T) {
	got := buildIntensityTab(temoinRows())
	temoinN(t, "intensite.len", len(got.ScorePerMinData), 7)
	for i, want := range []string{"79.02", "56.35", "600", "10", "0", "0", "0"} {
		if i < len(got.ScorePerMinData) {
			p := got.ScorePerMinData[i]
			temoinF(t, "intensite.spm"+strconv.Itoa(p.Index), p.Value, want)
		}
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
	temoinN(t, "contrib.len", len(got.Datapoints), 2)
	// Somme des secondes puis une division : sur ces entrées, le résultat est celui de l'ancienne somme des
	// minutes (l'écart au dernier bit admis ne s'y manifeste pas).
	attendu := map[string][3]string{
		"alpha": {"1.1138251099169516", "0.6448461162677088", "0.26380068392769906"},
		"carol": {"1.1980033277870215", "0.5990016638935107", "0.39933444259567386"},
	}
	for _, p := range got.Datapoints {
		want, ok := attendu[p.Category]
		if !ok {
			t.Errorf("catégorie inattendue %q", p.Category)
			continue
		}
		for i, k := range []string{"kills_per_min", "deaths_per_min", "assists_per_min"} {
			temoinF(t, "contrib."+p.Category+"."+k, p.Components[k], want[i])
		}
	}
}

// Somme des minutes par match puis une division : 25 frags en 300 s + 344 s font 2,3291925465838506 par
// minute ; une somme des secondes divisée par 60 rendrait ...851 au dernier bit.
func TestTemoinMinute_ContributionsSommeDesMinutes(t *testing.T) {
	ip := func(v int) *int { return &v }
	mk := func(secs, k, d, a int) canonical.PlayerMatchRow {
		var r canonical.PlayerMatchRow
		r.Summary.DurationSeconds = ip(secs)
		r.Self.Kills, r.Self.Deaths, r.Self.Assists = ip(k), ip(d), ip(a)
		return r
	}
	got := BuildPerMinuteStats(map[string][]canonical.PlayerMatchRow{
		"alpha": {mk(300, 20, 6, 4), mk(344, 5, 4, 3)},
	})
	temoinN(t, "contrib2.len", len(got.Datapoints), 1)
	if len(got.Datapoints) == 1 {
		c := got.Datapoints[0].Components
		temoinF(t, "contrib2.kills_per_min", c["kills_per_min"], "2.3291925465838506")
		temoinF(t, "contrib2.deaths_per_min", c["deaths_per_min"], "0.9316770186335402")
		temoinF(t, "contrib2.assists_per_min", c["assists_per_min"], "0.6521739130434782")
	}
}

// Une durée négative est exclue des tranches de score par minute, comme une durée nulle ou absente.
func TestTemoinMinute_BucketsDureeNegativeExclue(t *testing.T) {
	ip := func(v int) *int { return &v }
	got := buildScorePerMinBuckets([]legacymatch.StatsMatchRow{
		{PersonalScore: ip(500), TimePlayedSeconds: ip(-60)},
		{PersonalScore: ip(500), TimePlayedSeconds: ip(0)},
	})
	temoinN(t, "buckets.negatif.len", len(got), 0)
}
