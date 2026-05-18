package streaks

import (
	"context"
	"errors"
	"testing"
	"time"
)

// evaluator_test.go — tests unitaires de l'évaluateur de streaks.
//
// Stratégie : Repo fake en mémoire pour isoler la logique d'orchestration.
// Tous les tests utilisent un horodatage fixe pour rendre les buckets déterministes.

// ─── Repo fake ──────────────────────────────────────────────────────────────

type fakeRepo struct {
	streaks map[string]Streak // key = userID|titleSlug|type
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{streaks: map[string]Streak{}}
}

func (r *fakeRepo) key(userID, titleSlug string, st StreakType) string {
	return userID + "|" + titleSlug + "|" + string(st)
}

func (r *fakeRepo) GetActive(_ context.Context, userID, titleSlug string, st StreakType) (*Streak, error) {
	s, ok := r.streaks[r.key(userID, titleSlug, st)]
	if !ok {
		return nil, nil
	}
	// Considérer broken comme non-actif.
	if s.Status == StreakStatusBroken {
		return nil, nil
	}
	out := s
	return &out, nil
}

func (r *fakeRepo) Upsert(_ context.Context, s Streak) error {
	if s.ID == "" {
		return errors.New("streak ID required")
	}
	r.streaks[r.key(s.UserID, s.TitleSlug, s.Type)] = s
	return nil
}

func (r *fakeRepo) List(_ context.Context, userID, titleSlug string) ([]Streak, error) {
	out := []Streak{}
	for _, s := range r.streaks {
		if s.UserID == userID && s.TitleSlug == titleSlug {
			out = append(out, s)
		}
	}
	return out, nil
}

// ─── Helpers tests ──────────────────────────────────────────────────────────

// fixedDate retourne une date UTC sans surprise de fuseau.
func fixedDate(year int, month time.Month, day, hour int) time.Time {
	return time.Date(year, month, day, hour, 0, 0, 0, time.UTC)
}

func newEvaluator(repo Repo) *Evaluator {
	counter := 0
	return NewEvaluator(repo).WithIDGen(func() string {
		counter++
		return "test-id"
	})
}

func evalOnce(t *testing.T, ev *Evaluator, input EvaluateInput) EvaluationResult {
	t.Helper()
	results, err := ev.Evaluate(context.Background(), input)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	return results[0]
}

// ─── Tests : démarrage de streak ────────────────────────────────────────────

func TestEvaluate_StartedOnFirstMatch_DailyPlay(t *testing.T) {
	repo := newFakeRepo()
	ev := newEvaluator(repo)
	now := fixedDate(2026, 5, 18, 14)

	res := evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		Matches:    []MatchActivity{{PlayedAt: fixedDate(2026, 5, 18, 12)}},
		Thresholds: map[StreakType]float64{StreakTypeDailyPlay: 0},
	})

	if res.Transition != TransitionStarted {
		t.Fatalf("expected Started, got %s", res.Transition)
	}
	if res.Streak.CurrentLength != 1 {
		t.Errorf("CurrentLength = %d, want 1", res.Streak.CurrentLength)
	}
	if res.Streak.Status != StreakStatusActive {
		t.Errorf("Status = %s, want active", res.Streak.Status)
	}
}

func TestEvaluate_NoMatchesToday_NoStart(t *testing.T) {
	repo := newFakeRepo()
	ev := newEvaluator(repo)
	now := fixedDate(2026, 5, 18, 14)

	res := evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		Matches:    []MatchActivity{{PlayedAt: fixedDate(2026, 5, 17, 12)}}, // hier
		Thresholds: map[StreakType]float64{StreakTypeDailyPlay: 0},
	})

	if res.Transition != TransitionNone {
		t.Fatalf("expected None, got %s", res.Transition)
	}
}

// ─── Tests : incrément + idempotence ────────────────────────────────────────

func TestEvaluate_IncrementOnNextDay(t *testing.T) {
	repo := newFakeRepo()
	ev := newEvaluator(repo)

	// J0 : démarre la streak.
	day0 := fixedDate(2026, 5, 17, 14)
	evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: day0,
		Matches:    []MatchActivity{{PlayedAt: day0}},
		Thresholds: map[StreakType]float64{StreakTypeDailyPlay: 0},
	})

	// J1 : nouvelle activité.
	day1 := fixedDate(2026, 5, 18, 12)
	res := evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: day1,
		Matches: []MatchActivity{
			{PlayedAt: day0},
			{PlayedAt: day1},
		},
		Thresholds: map[StreakType]float64{StreakTypeDailyPlay: 0},
	})

	if res.Transition != TransitionIncremented {
		t.Fatalf("expected Incremented, got %s", res.Transition)
	}
	if res.Streak.CurrentLength != 2 {
		t.Errorf("CurrentLength = %d, want 2", res.Streak.CurrentLength)
	}
	if res.Streak.BestLength != 2 {
		t.Errorf("BestLength = %d, want 2", res.Streak.BestLength)
	}
}

func TestEvaluate_SameDay_Idempotent(t *testing.T) {
	repo := newFakeRepo()
	ev := newEvaluator(repo)
	day0 := fixedDate(2026, 5, 18, 10)

	evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: day0,
		Matches:    []MatchActivity{{PlayedAt: day0}},
		Thresholds: map[StreakType]float64{StreakTypeDailyPlay: 0},
	})

	// 2 heures plus tard, même jour.
	res := evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: day0.Add(2 * time.Hour),
		Matches:    []MatchActivity{{PlayedAt: day0}},
		Thresholds: map[StreakType]float64{StreakTypeDailyPlay: 0},
	})

	if res.Transition != TransitionNone {
		t.Fatalf("expected None (idempotent), got %s", res.Transition)
	}
	if res.Streak.CurrentLength != 1 {
		t.Errorf("CurrentLength = %d, want 1", res.Streak.CurrentLength)
	}
}

// ─── Tests : shields ────────────────────────────────────────────────────────

func TestEvaluate_OneMissedDay_ShieldSaves(t *testing.T) {
	repo := newFakeRepo()
	ev := newEvaluator(repo)

	// J0 : démarre.
	day0 := fixedDate(2026, 5, 18, 14)
	evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: day0,
		Matches:    []MatchActivity{{PlayedAt: day0}},
		Thresholds: map[StreakType]float64{StreakTypeDailyPlay: 0},
	})

	// J2 : saute J1. Shield doit absorber + on incrémente sur J2.
	day2 := fixedDate(2026, 5, 20, 14)
	res := evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: day2,
		Matches: []MatchActivity{
			{PlayedAt: day0},
			{PlayedAt: day2},
			// pas de match J1 = jour manqué
		},
		Thresholds: map[StreakType]float64{StreakTypeDailyPlay: 0},
	})

	if res.Transition != TransitionShielded {
		t.Fatalf("expected Shielded, got %s", res.Transition)
	}
	if res.Streak.ShieldsUsed != 1 {
		t.Errorf("ShieldsUsed = %d, want 1", res.Streak.ShieldsUsed)
	}
	if res.Streak.CurrentLength != 2 {
		t.Errorf("CurrentLength = %d, want 2 (J0+J2 with shield on J1)", res.Streak.CurrentLength)
	}
	if res.Streak.Status != StreakStatusActive {
		t.Errorf("Status = %s, want active (recovered after shield)", res.Streak.Status)
	}
}

func TestEvaluate_TwoMissedDays_Breaks(t *testing.T) {
	repo := newFakeRepo()
	ev := newEvaluator(repo)

	// J0 : démarre.
	day0 := fixedDate(2026, 5, 18, 14)
	evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: day0,
		Matches:    []MatchActivity{{PlayedAt: day0}},
		Thresholds: map[StreakType]float64{StreakTypeDailyPlay: 0},
	})

	// J3 : saute J1 ET J2 = 2 jours manqués, 1 shield ne suffit pas.
	day3 := fixedDate(2026, 5, 21, 14)
	res := evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: day3,
		Matches:    []MatchActivity{{PlayedAt: day0}, {PlayedAt: day3}},
		Thresholds: map[StreakType]float64{StreakTypeDailyPlay: 0},
	})

	if res.Transition != TransitionBroken {
		t.Fatalf("expected Broken, got %s", res.Transition)
	}
	if res.Streak.Status != StreakStatusBroken {
		t.Errorf("Status = %s, want broken", res.Streak.Status)
	}
	if res.Streak.BrokenAt == nil {
		t.Errorf("BrokenAt should be set")
	}
}

func TestEvaluate_ShieldsRegenerate_AcrossMonths(t *testing.T) {
	// Streak active fin mai avec shield consommé, on évalue début juin avec
	// 1 seul jour manqué (mai 31) → shield doit régénérer puis se consommer.
	repo := newFakeRepo()
	ev := newEvaluator(repo)

	mayInc := fixedDate(2026, 5, 30, 14)
	pre := Streak{
		ID: "test-id", UserID: "u1", TitleSlug: "halo_infinite",
		Type: StreakTypeDailyPlay, StartedAt: fixedDate(2026, 5, 24, 0),
		CurrentLength: 7, BestLength: 7,
		LastIncrementAt:  &mayInc,
		ShieldsUsed:      1, // consommé en mai
		ShieldsAvailable: MaxShieldsPerMonth,
		Status:           StreakStatusActive,
	}
	if err := repo.Upsert(context.Background(), pre); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Juin 1 : saute mai 31 → 1 jour manqué. Shield doit régénérer (juin != mai)
	// puis se consommer pour absorber mai 31.
	jun1 := fixedDate(2026, 6, 1, 14)
	res := evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: jun1,
		Matches:    []MatchActivity{{PlayedAt: mayInc}, {PlayedAt: jun1}},
		Thresholds: map[StreakType]float64{StreakTypeDailyPlay: 0},
	})

	if res.Transition != TransitionShielded {
		t.Fatalf("expected Shielded (regen across months), got %s", res.Transition)
	}
	if res.Streak.ShieldsUsed != 1 {
		t.Errorf("ShieldsUsed = %d, want 1 (regenerated then consumed)", res.Streak.ShieldsUsed)
	}
	if res.Streak.CurrentLength != 8 {
		t.Errorf("CurrentLength = %d, want 8 (J7 + J jun1 with shield on mai 31)", res.Streak.CurrentLength)
	}
}

// ─── Tests : multi-bucket walk ──────────────────────────────────────────────

func TestEvaluate_ConsecutiveMatches_MultipleIncrementsInOneEval(t *testing.T) {
	// J0 : démarre. J1, J2, J3 : joué chaque jour mais pas d'éval entre temps.
	// Eval à J3 doit incrémenter 3x d'un coup.
	repo := newFakeRepo()
	ev := newEvaluator(repo)

	day0 := fixedDate(2026, 5, 18, 14)
	evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: day0,
		Matches:    []MatchActivity{{PlayedAt: day0}},
		Thresholds: map[StreakType]float64{StreakTypeDailyPlay: 0},
	})

	day3 := fixedDate(2026, 5, 21, 14)
	res := evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: day3,
		Matches: []MatchActivity{
			{PlayedAt: day0},
			{PlayedAt: fixedDate(2026, 5, 19, 14)},
			{PlayedAt: fixedDate(2026, 5, 20, 14)},
			{PlayedAt: day3},
		},
		Thresholds: map[StreakType]float64{StreakTypeDailyPlay: 0},
	})

	if res.Transition != TransitionIncremented {
		t.Fatalf("expected Incremented, got %s", res.Transition)
	}
	if res.Streak.CurrentLength != 4 {
		t.Errorf("CurrentLength = %d, want 4 (J0+J1+J2+J3)", res.Streak.CurrentLength)
	}
}

// ─── Tests : types weekly ──────────────────────────────────────────────────

func TestEvaluate_WeeklyPlay_FiveMatches_Satisfies(t *testing.T) {
	repo := newFakeRepo()
	ev := newEvaluator(repo)
	// Lundi 18 mai 2026 = début semaine ISO. On joue 5 fois lundi.
	now := fixedDate(2026, 5, 18, 20)
	matches := []MatchActivity{}
	for i := 0; i < 5; i++ {
		matches = append(matches, MatchActivity{PlayedAt: fixedDate(2026, 5, 18, 10+i)})
	}

	res := evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		Matches:    matches,
		Thresholds: map[StreakType]float64{StreakTypeWeeklyPlay: 0},
	})

	if res.Transition != TransitionStarted {
		t.Fatalf("expected Started (5 matches sat weekly), got %s", res.Transition)
	}
}

func TestEvaluate_WeeklyPlay_FourMatches_Insufficient(t *testing.T) {
	repo := newFakeRepo()
	ev := newEvaluator(repo)
	now := fixedDate(2026, 5, 18, 20)
	matches := []MatchActivity{}
	for i := 0; i < 4; i++ {
		matches = append(matches, MatchActivity{PlayedAt: fixedDate(2026, 5, 18, 10+i)})
	}

	res := evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		Matches:    matches,
		Thresholds: map[StreakType]float64{StreakTypeWeeklyPlay: 0},
	})

	if res.Transition != TransitionNone {
		t.Fatalf("expected None (4 matches < 5), got %s", res.Transition)
	}
}

// ─── Tests : types perf-based ──────────────────────────────────────────────

func TestEvaluate_DailyPerf_KDAAboveThreshold(t *testing.T) {
	repo := newFakeRepo()
	ev := newEvaluator(repo)
	now := fixedDate(2026, 5, 18, 14)

	res := evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		Matches: []MatchActivity{
			{PlayedAt: now, Stats: map[string]float64{"kda": 1.8}},
		},
		Thresholds: map[StreakType]float64{StreakTypeDailyPerf: 1.5},
	})

	if res.Transition != TransitionStarted {
		t.Fatalf("expected Started (KDA 1.8 > threshold 1.5), got %s", res.Transition)
	}
}

func TestEvaluate_DailyPerf_KDABelowThreshold_NoStart(t *testing.T) {
	repo := newFakeRepo()
	ev := newEvaluator(repo)
	now := fixedDate(2026, 5, 18, 14)

	res := evalOnce(t, ev, EvaluateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		Matches: []MatchActivity{
			{PlayedAt: now, Stats: map[string]float64{"kda": 1.2}},
		},
		Thresholds: map[StreakType]float64{StreakTypeDailyPerf: 1.5},
	})

	if res.Transition != TransitionNone {
		t.Fatalf("expected None (KDA 1.2 < threshold 1.5), got %s", res.Transition)
	}
}

// ─── Tests : PP multiplier (couverture rapide) ──────────────────────────────

func TestPPMultiplier_AllBrackets(t *testing.T) {
	cases := []struct {
		length int
		want   float64
	}{
		{0, 1.00}, {1, 1.00}, {3, 1.00},
		{4, 1.10}, {7, 1.10},
		{8, 1.25}, {14, 1.25},
		{15, 1.50}, {29, 1.50},
		{30, 1.75}, {365, 1.75},
	}
	for _, c := range cases {
		got := PPMultiplier(c.length)
		if got != c.want {
			t.Errorf("PPMultiplier(%d) = %.2f, want %.2f", c.length, got, c.want)
		}
	}
}
