package timeline

import (
	"testing"
	"time"
)

func atMs(baseMs int64) time.Time { return time.UnixMilli(baseMs).UTC() }

func TestComputeT0_NoData(t *testing.T) {
	start := atMs(1_000_000)
	// Que des bots / absents au début.
	parts := []ParticipationT0Input{
		{FirstJoinedTime: atMs(1_028_000), PresentAtBeginning: false},
		{FirstJoinedTime: atMs(1_028_000), PresentAtBeginning: true, IsBot: true},
		{PresentAtBeginning: true}, // FirstJoinedTime zero
	}
	t0, q := ComputeT0(parts, start)
	if q != T0QualityNoData || t0 != 0 {
		t.Errorf("got (%d, %s), want (0, no_data)", t0, q)
	}
	if q.Computed() {
		t.Error("no_data must not be Computed()")
	}
}

func TestComputeT0_SingleSource(t *testing.T) {
	start := atMs(1_000_000)
	parts := []ParticipationT0Input{
		{FirstJoinedTime: atMs(1_028_000), PresentAtBeginning: true},
		{FirstJoinedTime: atMs(1_050_000), PresentAtBeginning: false}, // latecomer ignoré
	}
	t0, q := ComputeT0(parts, start)
	if q != T0QualitySingleSource || t0 != 28_000 {
		t.Errorf("got (%d, %s), want (28000, single_source)", t0, q)
	}
}

func TestComputeT0_OK_TakesMin(t *testing.T) {
	start := atMs(1_000_000)
	// 4 joueurs présents au début, spread 500ms (≤ 2s) → OK, T0 = min.
	parts := []ParticipationT0Input{
		{FirstJoinedTime: atMs(1_028_200), PresentAtBeginning: true},
		{FirstJoinedTime: atMs(1_028_000), PresentAtBeginning: true},
		{FirstJoinedTime: atMs(1_028_500), PresentAtBeginning: true},
		{FirstJoinedTime: atMs(1_028_100), PresentAtBeginning: true},
	}
	t0, q := ComputeT0(parts, start)
	if q != T0QualityOK || t0 != 28_000 {
		t.Errorf("got (%d, %s), want (28000, ok)", t0, q)
	}
}

func TestComputeT0_SpreadHigh_TakesMedian(t *testing.T) {
	start := atMs(1_000_000)
	// Spread 5s (> 2s) → SpreadHigh, T0 = médiane.
	// joins: 28000, 30000, 33000 → median 30000.
	parts := []ParticipationT0Input{
		{FirstJoinedTime: atMs(1_028_000), PresentAtBeginning: true},
		{FirstJoinedTime: atMs(1_033_000), PresentAtBeginning: true},
		{FirstJoinedTime: atMs(1_030_000), PresentAtBeginning: true},
	}
	t0, q := ComputeT0(parts, start)
	if q != T0QualitySpreadHigh || t0 != 30_000 {
		t.Errorf("got (%d, %s), want (30000, spread_high)", t0, q)
	}
}

func TestComputeT0_Negative_Rejected(t *testing.T) {
	start := atMs(1_000_000)
	// first_joined avant start (timezone bug) → negative.
	parts := []ParticipationT0Input{
		{FirstJoinedTime: atMs(993_000), PresentAtBeginning: true},
		{FirstJoinedTime: atMs(993_100), PresentAtBeginning: true},
	}
	t0, q := ComputeT0(parts, start)
	if q != T0QualityNegative || t0 != 0 {
		t.Errorf("got (%d, %s), want (0, negative)", t0, q)
	}
	if q.Computed() {
		t.Error("negative must not be Computed()")
	}
}

func TestComputeT0_SuspiciousHigh_Rejected(t *testing.T) {
	start := atMs(1_000_000)
	// T0 = 7200s (timezone +2h bug) → suspicious_high.
	parts := []ParticipationT0Input{
		{FirstJoinedTime: atMs(1_000_000 + 7_200_000), PresentAtBeginning: true},
		{FirstJoinedTime: atMs(1_000_000 + 7_200_100), PresentAtBeginning: true},
	}
	t0, q := ComputeT0(parts, start)
	if q != T0QualitySuspiciousHigh || t0 != 0 {
		t.Errorf("got (%d, %s), want (0, suspicious_high)", t0, q)
	}
}

func TestComputeT0_BotsExcludedFromSpread(t *testing.T) {
	start := atMs(1_000_000)
	// Un bot avec un join très tardif ne doit pas polluer le spread.
	parts := []ParticipationT0Input{
		{FirstJoinedTime: atMs(1_028_000), PresentAtBeginning: true},
		{FirstJoinedTime: atMs(1_028_300), PresentAtBeginning: true},
		{FirstJoinedTime: atMs(1_090_000), PresentAtBeginning: true, IsBot: true},
	}
	t0, q := ComputeT0(parts, start)
	if q != T0QualityOK || t0 != 28_000 {
		t.Errorf("bot should be excluded; got (%d, %s), want (28000, ok)", t0, q)
	}
}

func TestComputeT0_FortressReference(t *testing.T) {
	// Reproduit le match Fortress : start 21:58:18.523Z, joueurs à 21:58:46.454
	// (≈ +27.9s). Attendu : OK, T0 ≈ 27931ms.
	start := time.Date(2026, 3, 31, 21, 58, 18, 523_000_000, time.UTC)
	join := time.Date(2026, 3, 31, 21, 58, 46, 454_000_000, time.UTC)
	parts := []ParticipationT0Input{
		{FirstJoinedTime: join, PresentAtBeginning: true},
		{FirstJoinedTime: join.Add(5 * time.Millisecond), PresentAtBeginning: true},
	}
	t0, q := ComputeT0(parts, start)
	if q != T0QualityOK {
		t.Errorf("quality = %s, want ok", q)
	}
	if t0 < 27_900 || t0 > 27_950 {
		t.Errorf("T0 = %dms, want ~27931ms", t0)
	}
}

func TestT0Quality_Computed(t *testing.T) {
	computed := []T0Quality{T0QualityOK, T0QualitySingleSource, T0QualitySpreadHigh}
	rejected := []T0Quality{T0QualityNoData, T0QualityNegative, T0QualitySuspiciousHigh}
	for _, q := range computed {
		if !q.Computed() {
			t.Errorf("%s should be Computed()", q)
		}
	}
	for _, q := range rejected {
		if q.Computed() {
			t.Errorf("%s should not be Computed()", q)
		}
	}
}
