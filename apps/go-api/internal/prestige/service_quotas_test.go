package prestige

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Tests du quota guard (CheckQuotas) sur le service.
// Vérifient que le mode pilote applique les plafonds et que le mode libre passe.

// fakeChallengeRepo permet de simuler les counts sans DB.
type fakeChallengeRepo struct {
	activeByCadence   map[Cadence]int
	activeTotal       int
	createdSince      int
	createCalled      bool
	createCount       int
	createdChallenges []Challenge    // capture des défis persistés (assertions Source, etc.)
	listResult        []Challenge    // renvoyé tel quel par List (pour tests cooldown)
	detachedArc       string         // dernier arcID passé à DetachFromArc
	deletedArc        string         // dernier arcID passé à DeleteByArc
	statusUpdates     []statusUpdate // capture des UpdateStatus (id + statut)
}

// statusUpdate capture un appel UpdateStatus pour assertion (mode pilote disable…).
type statusUpdate struct {
	ID     string
	Status ChallengeStatus
}

func (r *fakeChallengeRepo) Create(_ context.Context, c Challenge) error {
	r.createCalled = true
	r.createCount++
	r.createdChallenges = append(r.createdChallenges, c)
	return nil
}
func (r *fakeChallengeRepo) Get(_ context.Context, _ string) (Challenge, error) {
	return Challenge{}, ErrChallengeNotFound
}
func (r *fakeChallengeRepo) List(_ context.Context, _ ChallengeFilter) ([]Challenge, error) {
	return r.listResult, nil
}
func (r *fakeChallengeRepo) DetachFromArc(_ context.Context, arcID string) error {
	r.detachedArc = arcID
	return nil
}
func (r *fakeChallengeRepo) DeleteByArc(_ context.Context, arcID string) error {
	r.deletedArc = arcID
	return nil
}
func (r *fakeChallengeRepo) UpdateStatus(_ context.Context, id string, s ChallengeStatus, _ time.Time) error {
	r.statusUpdates = append(r.statusUpdates, statusUpdate{ID: id, Status: s})
	return nil
}
func (r *fakeChallengeRepo) UpdateLabel(_ context.Context, _, _ string) error { return nil }
func (r *fakeChallengeRepo) UpdateTarget(_ context.Context, _ string, _ float64, _ Tier, _ DataTier, _ time.Time) error {
	return nil
}
func (r *fakeChallengeRepo) CountActiveByCadence(_ context.Context, _, _ string, c Cadence) (int, error) {
	return r.activeByCadence[c], nil
}
func (r *fakeChallengeRepo) CountActiveTotal(_ context.Context, _, _ string) (int, error) {
	return r.activeTotal, nil
}
func (r *fakeChallengeRepo) CountCreatedSince(_ context.Context, _, _ string, _ ChallengeMode, _ time.Time) (int, error) {
	return r.createdSince, nil
}

// fakeBaselineProvider évite la dépendance à un PlayerDB réel.
type fakeBaselineProvider struct{}

func (f *fakeBaselineProvider) RecentMatches(_ context.Context, _, _, _ string, _ int) ([]MatchData, error) {
	return []MatchData{
		{MatchID: "m1", MetricValue: 1.0, StartedAt: time.Now()},
		{MatchID: "m2", MetricValue: 1.0, StartedAt: time.Now()},
		{MatchID: "m3", MetricValue: 1.0, StartedAt: time.Now()},
		{MatchID: "m4", MetricValue: 1.0, StartedAt: time.Now()},
		{MatchID: "m5", MetricValue: 1.0, StartedAt: time.Now()},
		{MatchID: "m6", MetricValue: 1.0, StartedAt: time.Now()},
		{MatchID: "m7", MetricValue: 1.0, StartedAt: time.Now()},
		{MatchID: "m8", MetricValue: 1.0, StartedAt: time.Now()},
		{MatchID: "m9", MetricValue: 1.0, StartedAt: time.Now()},
		{MatchID: "m10", MetricValue: 1.0, StartedAt: time.Now()},
	}, nil
}
func (f *fakeBaselineProvider) PopulationPercentile(_ context.Context, _, _ string, _ float64) (float64, int, error) {
	return 0.95, 100, nil // au-dessus de p90 → mythic éligible
}

// buildServiceForQuotaTests assemble un service minimal avec un fake repo.
//
// On utilise NewService pour bénéficier de l'emitter télémétrie initialisé
// (sinon nil-deref dans CreateChallenge → EmitCreated).
func buildServiceForQuotaTests(repo *fakeChallengeRepo) *service {
	telRepo := &fakeNoOpTelemetryRepo{}
	deps := Deps{
		Tuning:           DefaultTuning(),
		Challenges:       repo,
		Telemetry:        telRepo,
		Prestige:         &fakeNoOpPrestigeRepo{},
		BaselineProvider: &fakeBaselineProvider{},
		Now:              func() time.Time { return time.Now().UTC() },
	}
	return NewService(deps).(*service)
}

// ─── Tests ───

func TestCheckQuotas_LibreModeAlwaysPasses(t *testing.T) {
	repo := &fakeChallengeRepo{activeTotal: 100} // bien au-dessus du plafond
	svc := buildServiceForQuotaTests(repo)
	err := svc.checkQuotas(context.Background(), CreateChallengeRequest{
		Mode:    ModeLibre,
		Cadence: CadenceDaily,
	})
	if err != nil {
		t.Errorf("libre mode should bypass quotas, got %v", err)
	}
}

func TestCheckQuotas_PiloteRespectsTotalCap(t *testing.T) {
	repo := &fakeChallengeRepo{activeTotal: 12} // = TotalActiveMax
	svc := buildServiceForQuotaTests(repo)
	err := svc.checkQuotas(context.Background(), CreateChallengeRequest{
		Mode:    ModePilote,
		Cadence: CadenceDaily,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput at total cap, got %v", err)
	}
}

func TestCheckQuotas_PiloteRespectsCadenceCap(t *testing.T) {
	cases := []struct {
		cadence    Cadence
		count      int
		shouldFail bool
	}{
		{CadenceDaily, 3, true}, // = DailyMax
		{CadenceDaily, 2, false},
		{CadenceWeekly, 5, true}, // = WeeklyMax
		{CadenceWeekly, 4, false},
		{CadenceMonthly, 2, true}, // = MonthlyMax
		{CadenceMonthly, 1, false},
	}
	for _, tc := range cases {
		repo := &fakeChallengeRepo{
			activeTotal:     1,
			activeByCadence: map[Cadence]int{tc.cadence: tc.count},
		}
		svc := buildServiceForQuotaTests(repo)
		err := svc.checkQuotas(context.Background(), CreateChallengeRequest{
			Mode:    ModePilote,
			Cadence: tc.cadence,
		})
		if tc.shouldFail && err == nil {
			t.Errorf("%s at cap %d should fail", tc.cadence, tc.count)
		}
		if !tc.shouldFail && err != nil {
			t.Errorf("%s at %d should pass, got %v", tc.cadence, tc.count, err)
		}
	}
}

func TestCheckQuotas_FreeCadencePilote(t *testing.T) {
	// CadenceFree avec mode pilote n'a pas de cap dédié (cadenceQuotaMax → 0)
	// → seul le total cap s'applique.
	repo := &fakeChallengeRepo{
		activeTotal:     5,
		activeByCadence: map[Cadence]int{CadenceFree: 100},
	}
	svc := buildServiceForQuotaTests(repo)
	err := svc.checkQuotas(context.Background(), CreateChallengeRequest{
		Mode:    ModePilote,
		Cadence: CadenceFree,
	})
	if err != nil {
		t.Errorf("free cadence should not have per-cadence cap, got %v", err)
	}
}

// Vérifie que CreateChallenge() refuse réellement quand le quota est plein.
func TestCreateChallenge_RejectedAtQuota(t *testing.T) {
	repo := &fakeChallengeRepo{activeTotal: 12}
	svc := buildServiceForQuotaTests(repo)
	_, err := svc.CreateChallenge(context.Background(), CreateChallengeRequest{
		UserID:     "u1",
		TitleSlug:  "halo_infinite",
		Metric:     "FieldKDA",
		Target:     1.5,
		WindowType: WindowSession,
		Cadence:    CadenceDaily,
		EvalType:   EvalThreshold,
		Mode:       ModePilote,
	})
	if err == nil {
		t.Fatal("expected error from quota cap")
	}
	if repo.createCalled {
		t.Error("repo.Create should not have been called when quota is full")
	}
}

func TestCreateChallenge_LibreNoQuotaCheck(t *testing.T) {
	repo := &fakeChallengeRepo{activeTotal: 1000} // énorme
	svc := buildServiceForQuotaTests(repo)

	_, err := svc.CreateChallenge(context.Background(), CreateChallengeRequest{
		UserID:     "u1",
		TitleSlug:  "halo_infinite",
		Metric:     "FieldKDA",
		Target:     1.5,
		WindowType: WindowSession,
		Cadence:    CadenceFree,
		EvalType:   EvalThreshold,
		Mode:       ModeLibre,
	})
	if err != nil {
		t.Fatalf("libre mode should pass quota, got %v", err)
	}
	if !repo.createCalled {
		t.Error("repo.Create should have been called")
	}
}

// ─── Cooldown anti-farming (tous modes, depuis 2026-06-08) ───

// Un abandon récent sur la même métrique bloque la recréation (libre inclus).
func TestCreateChallenge_CooldownBlocks(t *testing.T) {
	now := time.Now().UTC()
	abandonedRecent := now.Add(-1 * time.Hour) // < 24h → cooldown actif
	repo := &fakeChallengeRepo{
		listResult: []Challenge{
			{Metric: "FieldKDA", Mode: ModeLibre, Status: StatusAbandoned, AbandonedAt: &abandonedRecent},
		},
	}
	svc := buildServiceForQuotaTests(repo)
	_, err := svc.CreateChallenge(context.Background(), CreateChallengeRequest{
		UserID:     "u1",
		TitleSlug:  "halo_infinite",
		Metric:     "FieldKDA",
		Target:     1.5,
		WindowType: WindowSession,
		Cadence:    CadenceFree,
		EvalType:   EvalThreshold,
		Mode:       ModeLibre,
	})
	if !errors.Is(err, ErrCooldownActive) {
		t.Fatalf("expected ErrCooldownActive, got %v", err)
	}
	if repo.createCalled {
		t.Error("repo.Create should not have been called under cooldown")
	}
}

// Un abandon ancien (> 24h) n'empêche plus la recréation.
func TestCreateChallenge_CooldownExpired_OK(t *testing.T) {
	now := time.Now().UTC()
	abandonedOld := now.Add(-48 * time.Hour) // > 24h → cooldown expiré
	repo := &fakeChallengeRepo{
		listResult: []Challenge{
			{Metric: "FieldKDA", Mode: ModeLibre, Status: StatusAbandoned, AbandonedAt: &abandonedOld},
		},
	}
	svc := buildServiceForQuotaTests(repo)
	_, err := svc.CreateChallenge(context.Background(), CreateChallengeRequest{
		UserID:     "u1",
		TitleSlug:  "halo_infinite",
		Metric:     "FieldKDA",
		Target:     1.5,
		WindowType: WindowSession,
		Cadence:    CadenceFree,
		EvalType:   EvalThreshold,
		Mode:       ModeLibre,
	})
	if err != nil {
		t.Fatalf("expired cooldown should allow creation, got %v", err)
	}
	if !repo.createCalled {
		t.Error("repo.Create should have been called once cooldown expired")
	}
}

// La complétion ne crée pas de cooldown → recréation immédiate possible.
func TestCreateChallenge_CompletedNoCooldown(t *testing.T) {
	now := time.Now().UTC()
	completedRecent := now.Add(-1 * time.Minute)
	repo := &fakeChallengeRepo{
		listResult: []Challenge{
			{Metric: "FieldKDA", Mode: ModeLibre, Status: StatusCompleted, CompletedAt: &completedRecent},
		},
	}
	svc := buildServiceForQuotaTests(repo)
	_, err := svc.CreateChallenge(context.Background(), CreateChallengeRequest{
		UserID:     "u1",
		TitleSlug:  "halo_infinite",
		Metric:     "FieldKDA",
		Target:     1.5,
		WindowType: WindowSession,
		Cadence:    CadenceFree,
		EvalType:   EvalThreshold,
		Mode:       ModeLibre,
	})
	if err != nil {
		t.Fatalf("completion should not trigger cooldown, got %v", err)
	}
}

// ─── Fakes minimaux ───

type fakeNoOpPrestigeRepo struct{}

func (f *fakeNoOpPrestigeRepo) EmitEvent(_ context.Context, _ PrestigeEvent) error { return nil }
func (f *fakeNoOpPrestigeRepo) GetUserPrestige(_ context.Context, _, _ string) (UserPrestige, error) {
	return UserPrestige{}, nil
}
func (f *fakeNoOpPrestigeRepo) GetUserPrestigeCrossTitle(_ context.Context, _ string) (UserPrestige, error) {
	return UserPrestige{}, nil
}
func (f *fakeNoOpPrestigeRepo) UpsertUserPrestige(_ context.Context, _ UserPrestige) error {
	return nil
}
func (f *fakeNoOpPrestigeRepo) ListEvents(_ context.Context, _, _ string, _ time.Time) ([]PrestigeEvent, error) {
	return nil, nil
}
func (f *fakeNoOpPrestigeRepo) GetLeaderboard(_ context.Context, _ []string, _ *string, _ time.Time) ([]LeaderboardEntry, error) {
	return nil, nil
}

type fakeNoOpTelemetryRepo struct{}

func (f *fakeNoOpTelemetryRepo) Emit(_ context.Context, _ PrestigeTelemetry) error { return nil }
