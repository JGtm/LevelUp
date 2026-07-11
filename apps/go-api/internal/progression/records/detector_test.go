package records

import (
	"context"
	"testing"
	"time"
)

// detector_test.go — tests unitaires du détecteur de PB.
//
// Repos fakes en mémoire pour isoler la logique. Date fixe pour rendre les
// cutoffs de fenêtre déterministes.

// ─── Fakes ──────────────────────────────────────────────────────────────────

type fakePBRepo struct {
	pbs map[string]PersonalRecord // key = xuid|metric|period
}

func newFakePBRepo() *fakePBRepo { return &fakePBRepo{pbs: map[string]PersonalRecord{}} }

func (r *fakePBRepo) key(xuid, metric string, p RecordPeriod) string {
	return xuid + "|" + metric + "|" + string(p)
}

func (r *fakePBRepo) Get(_ context.Context, xuid, metric string, p RecordPeriod) (*PersonalRecord, error) {
	pb, ok := r.pbs[r.key(xuid, metric, p)]
	if !ok {
		return nil, nil
	}
	out := pb
	return &out, nil
}

func (r *fakePBRepo) Upsert(_ context.Context, pb PersonalRecord) error {
	r.pbs[r.key(pb.XUID, pb.Metric, pb.Period)] = pb
	return nil
}

func (r *fakePBRepo) ListByXUID(_ context.Context, xuid string) ([]PersonalRecord, error) {
	var out []PersonalRecord
	for _, pb := range r.pbs {
		if pb.XUID == xuid {
			out = append(out, pb)
		}
	}
	return out, nil
}

type fakeHistoryRepo struct {
	entries []RecordHistory
}

func newFakeHistoryRepo() *fakeHistoryRepo { return &fakeHistoryRepo{} }

func (r *fakeHistoryRepo) Append(_ context.Context, h RecordHistory) error {
	r.entries = append(r.entries, h)
	return nil
}

func (r *fakeHistoryRepo) ListRecent(_ context.Context, userID, titleSlug string, limit int) ([]RecordHistory, error) {
	var out []RecordHistory
	for _, h := range r.entries {
		if h.UserID == userID && h.TitleSlug == titleSlug {
			out = append(out, h)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func fixedDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 12, 0, 0, 0, time.UTC)
}

func newDetector(pb *fakePBRepo, hist *fakeHistoryRepo) *Detector {
	counter := 0
	return NewDetector(pb, hist).WithIDGen(func() string {
		counter++
		return "test-id"
	})
}

func resultsByPeriod(results []DetectionResult, metric TrackedMetric) map[RecordPeriod]DetectionResult {
	out := map[RecordPeriod]DetectionResult{}
	for _, r := range results {
		if r.Metric == metric {
			out[r.Period] = r
		}
	}
	return out
}

// ─── Tests : skip si pas assez de matchs ───────────────────────────────────

func TestDetect_SkipsWhenInsufficientMatches(t *testing.T) {
	pb := newFakePBRepo()
	hist := newFakeHistoryRepo()
	d := newDetector(pb, hist)
	now := fixedDate(2026, 5, 18)

	// 5 matchs seulement (< MinMatchesForRecord = 10)
	matches := make([]MatchInput, 5)
	for i := range matches {
		matches[i] = MatchInput{
			MatchID:  "m" + string(rune('0'+i)),
			PlayedAt: now.AddDate(0, 0, -i),
			Metrics:  map[TrackedMetric]float64{MetricPerformanceScore: 70 + float64(i)},
		}
	}

	results, err := d.Detect(context.Background(), DetectInput{
		XUID: "x1", UserID: "x1", TitleSlug: "halo_infinite",
		Now: now, Matches: matches,
		Metrics: []TrackedMetric{MetricPerformanceScore},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	for _, r := range results {
		if r.NewPB {
			t.Errorf("expected no NewPB with %d matches < MinMatchesForRecord, got %+v", len(matches), r)
		}
	}
	if len(pb.pbs) != 0 {
		t.Errorf("PBRepo should be empty, got %d entries", len(pb.pbs))
	}
}

// ─── Tests : premier PB sur 3 fenêtres ──────────────────────────────────────

func TestDetect_FirstPB_PersistsAllThreePeriods(t *testing.T) {
	pb := newFakePBRepo()
	hist := newFakeHistoryRepo()
	d := newDetector(pb, hist)
	now := fixedDate(2026, 5, 18)

	// 12 matchs sur 12 jours consécutifs (≥ MinMatchesForRecord).
	// Le meilleur (90) est à J-2, le pire (50) à J-11.
	matches := make([]MatchInput, 12)
	for i := 0; i < 12; i++ {
		matches[i] = MatchInput{
			MatchID:  "m" + string(rune('a'+i)),
			PlayedAt: now.AddDate(0, 0, -i),
			Metrics:  map[TrackedMetric]float64{MetricPerformanceScore: 50 + float64(i)*3.5},
		}
	}
	// J-2 → 50 + 2*3.5 = 57. Le max est à J-11 = 50 + 11*3.5 = 88.5
	expected := 50 + 11*3.5

	results, err := d.Detect(context.Background(), DetectInput{
		XUID: "x1", UserID: "x1", TitleSlug: "halo_infinite",
		Now: now, Matches: matches,
		Metrics: []TrackedMetric{MetricPerformanceScore},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}

	byPeriod := resultsByPeriod(results, MetricPerformanceScore)
	for _, p := range AllRecordPeriods() {
		r, ok := byPeriod[p]
		if !ok {
			t.Errorf("missing result for period %s", p)
			continue
		}
		if !r.NewPB {
			t.Errorf("period %s: NewPB should be true on first PB", p)
		}
		if r.Value != expected {
			t.Errorf("period %s: Value = %.2f, want %.2f", p, r.Value, expected)
		}
		if r.PreviousValue != nil {
			t.Errorf("period %s: PreviousValue should be nil on first PB", p)
		}
	}
	// 3 entrées historique (1 par période)
	if len(hist.entries) != 3 {
		t.Errorf("history len = %d, want 3", len(hist.entries))
	}
}

// ─── Tests : nouveau PB qui bat l'ancien ───────────────────────────────────

func TestDetect_NewPBBeatsPrevious(t *testing.T) {
	pb := newFakePBRepo()
	hist := newFakeHistoryRepo()
	d := newDetector(pb, hist)
	now := fixedDate(2026, 5, 18)

	// Seed un PB existant à 80 sur all_time
	prevAchieved := now.AddDate(0, 0, -50)
	pb.pbs[pb.key("x1", string(MetricPerformanceScore), RecordPeriodAllTime)] = PersonalRecord{
		XUID: "x1", Metric: string(MetricPerformanceScore), Period: RecordPeriodAllTime,
		Value: 80, AchievedAt: &prevAchieved, UpdatedAt: prevAchieved,
	}

	// 10 matchs récents, dont un à 92
	matches := make([]MatchInput, 10)
	for i := 0; i < 10; i++ {
		v := 60.0 + float64(i)
		if i == 5 {
			v = 92
		}
		matches[i] = MatchInput{
			MatchID:  "m" + string(rune('a'+i)),
			PlayedAt: now.AddDate(0, 0, -i),
			Metrics:  map[TrackedMetric]float64{MetricPerformanceScore: v},
		}
	}

	results, err := d.Detect(context.Background(), DetectInput{
		XUID: "x1", UserID: "x1", TitleSlug: "halo_infinite",
		Now: now, Matches: matches,
		Metrics: []TrackedMetric{MetricPerformanceScore},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}

	allTime := resultsByPeriod(results, MetricPerformanceScore)[RecordPeriodAllTime]
	if !allTime.NewPB {
		t.Errorf("expected NewPB on all_time")
	}
	if allTime.Value != 92 {
		t.Errorf("Value = %.2f, want 92", allTime.Value)
	}
	if allTime.PreviousValue == nil || *allTime.PreviousValue != 80 {
		t.Errorf("PreviousValue = %v, want 80", allTime.PreviousValue)
	}
}

// ─── Tests : valeur courante < PB existant → no PB, mais near-miss ─────────

func TestDetect_NearMiss(t *testing.T) {
	pb := newFakePBRepo()
	hist := newFakeHistoryRepo()
	d := newDetector(pb, hist)
	now := fixedDate(2026, 5, 18)

	// PB existant à 100 sur les 3 fenêtres (sinon les fenêtres non seedées
	// seraient traitées comme « first PB » et persistées).
	prevAchieved := now.AddDate(0, 0, -60)
	for _, p := range AllRecordPeriods() {
		pb.pbs[pb.key("x1", string(MetricPerformanceScore), p)] = PersonalRecord{
			XUID: "x1", Metric: string(MetricPerformanceScore), Period: p,
			Value: 100, AchievedAt: &prevAchieved, UpdatedAt: prevAchieved,
		}
	}

	// Meilleur match récent à 96 = 96% du PB → dans la zone near-miss (>= 95)
	matches := make([]MatchInput, 10)
	for i := 0; i < 10; i++ {
		v := 70.0 + float64(i)
		if i == 3 {
			v = 96
		}
		matches[i] = MatchInput{
			MatchID:  "m" + string(rune('a'+i)),
			PlayedAt: now.AddDate(0, 0, -i),
			Metrics:  map[TrackedMetric]float64{MetricPerformanceScore: v},
		}
	}

	results, err := d.Detect(context.Background(), DetectInput{
		XUID: "x1", UserID: "x1", TitleSlug: "halo_infinite",
		Now: now, Matches: matches,
		Metrics: []TrackedMetric{MetricPerformanceScore},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}

	allTime := resultsByPeriod(results, MetricPerformanceScore)[RecordPeriodAllTime]
	if allTime.NewPB {
		t.Errorf("expected no NewPB (96 < 100)")
	}
	if !allTime.NearMiss {
		t.Errorf("expected NearMiss (96 >= 100 × 0.95)")
	}
	if len(hist.entries) != 0 {
		t.Errorf("history should remain empty on near-miss only")
	}
}

// ─── Tests : window 30d ignore matchs hors fenêtre ─────────────────────────

func TestDetect_Window30d_IgnoresOldMatches(t *testing.T) {
	pb := newFakePBRepo()
	hist := newFakeHistoryRepo()
	d := newDetector(pb, hist)
	now := fixedDate(2026, 5, 18)

	// 10 matchs récents (< 30j) à valeur 70, plus 1 match ancien (J-60) à 99.
	matches := []MatchInput{
		{MatchID: "old", PlayedAt: now.AddDate(0, 0, -60), Metrics: map[TrackedMetric]float64{MetricPerformanceScore: 99}},
	}
	for i := 0; i < 10; i++ {
		matches = append(matches, MatchInput{
			MatchID:  "m" + string(rune('a'+i)),
			PlayedAt: now.AddDate(0, 0, -i-1),
			Metrics:  map[TrackedMetric]float64{MetricPerformanceScore: 70},
		})
	}

	results, err := d.Detect(context.Background(), DetectInput{
		XUID: "x1", UserID: "x1", TitleSlug: "halo_infinite",
		Now: now, Matches: matches,
		Metrics: []TrackedMetric{MetricPerformanceScore},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}

	byP := resultsByPeriod(results, MetricPerformanceScore)
	if byP[RecordPeriod30d].Value != 70 {
		t.Errorf("30d Value = %.2f, want 70 (old match should be ignored)", byP[RecordPeriod30d].Value)
	}
	if byP[RecordPeriod90d].Value != 99 {
		t.Errorf("90d Value = %.2f, want 99 (J-60 included in 90d)", byP[RecordPeriod90d].Value)
	}
	if byP[RecordPeriodAllTime].Value != 99 {
		t.Errorf("all_time Value = %.2f, want 99", byP[RecordPeriodAllTime].Value)
	}
}

// ─── Tests : IsNearMiss edge cases ─────────────────────────────────────────

func TestIsNearMiss_Cases(t *testing.T) {
	// DP11 : bande SIGNIFICATIVE = target×0.95 <= current <= target×0.98.
	cases := []struct {
		current, target float64
		want            bool
		desc            string
	}{
		{97, 100, true, "3% sous le PB → near-miss (dans la bande [95,98])"},
		{95, 100, true, "exactement 95% → near-miss (borne basse)"},
		{98, 100, true, "exactement 98% → near-miss (borne haute = 1-NearMissMinGapRatio)"},
		{94.999, 100, false, "6% sous le PB (< 95%) → non (hors bande)"},
		{99, 100, false, "1% sous le PB (> 98%) → non (écart insignifiant, DP11)"},
		{99.99, 100, false, "quasi égal au PB → non (incident 73.33 vs 73.333336)"},
		{100, 100, false, "égal → PAS near-miss (on est déjà au record)"},
		{101, 100, false, "au-dessus → non (c'est un NewPB, pas un near-miss)"},
		{50, 0, false, "target 0 → dégénéré, pas de near-miss"},
		{0, 100, false, "current 0 → trop loin"},
	}
	for _, c := range cases {
		if got := IsNearMiss(c.current, c.target); got != c.want {
			t.Errorf("%s: IsNearMiss(%.3f, %.3f) = %v, want %v", c.desc, c.current, c.target, got, c.want)
		}
	}
}
