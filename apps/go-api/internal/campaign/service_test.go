package campaign

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

// service_test.go — tests unitaires avec fake repo + fake samples.

// ─── Fakes ─────────────────────────────────────────────────────────────────

type fakeRepo struct {
	byID    map[string]ImprovementCampaign
	active  map[string]ImprovementCampaign // key = userID|titleSlug
	linked  map[string][]string
	updates int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		byID:   map[string]ImprovementCampaign{},
		active: map[string]ImprovementCampaign{},
		linked: map[string][]string{},
	}
}

func (r *fakeRepo) key(u, t string) string { return u + "|" + t }

func (r *fakeRepo) Insert(_ context.Context, c ImprovementCampaign) error {
	r.byID[c.ID] = c
	if c.Status == StatusActive {
		r.active[r.key(c.UserID, c.TitleSlug)] = c
	}
	return nil
}
func (r *fakeRepo) GetByID(_ context.Context, id string) (ImprovementCampaign, error) {
	if c, ok := r.byID[id]; ok {
		return c, nil
	}
	return ImprovementCampaign{}, ErrNotFound
}
func (r *fakeRepo) GetActive(_ context.Context, userID, titleSlug string) (ImprovementCampaign, error) {
	if c, ok := r.active[r.key(userID, titleSlug)]; ok && c.Status == StatusActive {
		return c, nil
	}
	return ImprovementCampaign{}, ErrNotFound
}
func (r *fakeRepo) ListEnded(_ context.Context, userID, titleSlug string) ([]ImprovementCampaign, error) {
	var out []ImprovementCampaign
	for _, c := range r.byID {
		if c.UserID == userID && c.TitleSlug == titleSlug && c.IsEnded() {
			out = append(out, c)
		}
	}
	return out, nil
}
func (r *fakeRepo) UpdateStatus(_ context.Context, id string, status CampaignStatus, endedAt *time.Time) error {
	c, ok := r.byID[id]
	if !ok {
		return ErrNotFound
	}
	c.Status = status
	c.EndedAt = endedAt
	r.byID[id] = c
	if status != StatusActive {
		delete(r.active, r.key(c.UserID, c.TitleSlug))
	}
	return nil
}
func (r *fakeRepo) UpdateEvaluation(_ context.Context, id string, _ Evaluation) error {
	if _, ok := r.byID[id]; !ok {
		return ErrNotFound
	}
	r.updates++
	return nil
}
func (r *fakeRepo) LinkedChallengeIDs(_ context.Context, campaignID string) ([]string, error) {
	return r.linked[campaignID], nil
}
func (r *fakeRepo) LinkChallenge(_ context.Context, challengeID, campaignID string) error {
	if campaignID == "" {
		for cid, ids := range r.linked {
			filtered := ids[:0]
			for _, id := range ids {
				if id != challengeID {
					filtered = append(filtered, id)
				}
			}
			r.linked[cid] = filtered
		}
		return nil
	}
	r.linked[campaignID] = append(r.linked[campaignID], challengeID)
	return nil
}

type fakeSamples struct {
	values []float64
}

func (s *fakeSamples) LoadAxisSamples(_ context.Context, _, _, _ string, _ AxisKind, _ string, _, _ time.Time) ([]float64, error) {
	return s.values, nil
}

// ─── Tests ─────────────────────────────────────────────────────────────────

func TestStartCampaign_Success(t *testing.T) {
	repo := newFakeRepo()
	samples := &fakeSamples{values: []float64{1, 2, 3, 4, 5}}
	svc := NewService(repo, samples)

	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	c, err := svc.StartCampaign(context.Background(), StartParams{
		UserID:    "u1",
		TitleSlug: "halo",
		Axis:      "survival",
		AxisKind:  AxisKindRadar,
	}, now)
	if err != nil {
		t.Fatalf("StartCampaign: %v", err)
	}
	if c.ID == "" {
		t.Error("ID empty")
	}
	if c.SnapshotValue != 3 {
		t.Errorf("Snapshot mean: got %.2f, want 3.0", c.SnapshotValue)
	}
	if c.SnapshotSample != 5 {
		t.Errorf("SnapshotSample: got %d, want 5", c.SnapshotSample)
	}
	if c.Status != StatusActive {
		t.Errorf("Status: got %q, want active", c.Status)
	}
	if c.PlaylistGroup != "all" {
		t.Errorf("PlaylistGroup default: got %q, want all", c.PlaylistGroup)
	}
}

func TestStartCampaign_AlreadyActive(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, &fakeSamples{})
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)

	_, err := svc.StartCampaign(context.Background(), StartParams{
		UserID: "u1", TitleSlug: "halo", Axis: "combat", AxisKind: AxisKindRadar,
	}, now)
	if err != nil {
		t.Fatalf("first start: %v", err)
	}
	_, err = svc.StartCampaign(context.Background(), StartParams{
		UserID: "u1", TitleSlug: "halo", Axis: "survival", AxisKind: AxisKindRadar,
	}, now)
	if !errors.Is(err, ErrAlreadyActive) {
		t.Errorf("second start: got err=%v, want ErrAlreadyActive", err)
	}
}

func TestStartCampaign_InvalidAxis(t *testing.T) {
	svc := NewService(newFakeRepo(), &fakeSamples{})
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	_, err := svc.StartCampaign(context.Background(), StartParams{
		UserID: "u1", TitleSlug: "halo", Axis: "", AxisKind: AxisKindRadar,
	}, now)
	if !errors.Is(err, ErrInvalidAxis) {
		t.Errorf("empty axis: got %v, want ErrInvalidAxis", err)
	}
	_, err = svc.StartCampaign(context.Background(), StartParams{
		UserID: "u1", TitleSlug: "halo", Axis: "combat", AxisKind: "unknown",
	}, now)
	if !errors.Is(err, ErrInvalidAxis) {
		t.Errorf("invalid kind: got %v, want ErrInvalidAxis", err)
	}
}

func TestTransitions_PauseResumeCloseAbandon(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, &fakeSamples{})
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)

	c, err := svc.StartCampaign(context.Background(), StartParams{
		UserID: "u1", TitleSlug: "halo", Axis: "combat", AxisKind: AxisKindRadar,
	}, now)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := svc.PauseCampaign(context.Background(), c.ID); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if repo.byID[c.ID].Status != StatusPaused {
		t.Errorf("status: got %q, want paused", repo.byID[c.ID].Status)
	}
	if err := svc.ResumeCampaign(context.Background(), c.ID); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if repo.byID[c.ID].Status != StatusActive {
		t.Errorf("status: got %q, want active", repo.byID[c.ID].Status)
	}
	if err := svc.CloseCampaign(context.Background(), c.ID, now.Add(24*time.Hour)); err != nil {
		t.Fatalf("close: %v", err)
	}
	if repo.byID[c.ID].Status != StatusCompleted {
		t.Errorf("status: got %q, want completed", repo.byID[c.ID].Status)
	}
	// Close again → invalid transition
	if err := svc.CloseCampaign(context.Background(), c.ID, now); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("re-close: got %v, want ErrInvalidStatus", err)
	}
	// Abandon a separate campaign
	c2, _ := svc.StartCampaign(context.Background(), StartParams{
		UserID: "u2", TitleSlug: "halo", Axis: "survival", AxisKind: AxisKindRadar,
	}, now)
	if err := svc.AbandonCampaign(context.Background(), c2.ID, now); err != nil {
		t.Fatalf("abandon: %v", err)
	}
	if repo.byID[c2.ID].Status != StatusAbandoned {
		t.Errorf("status: got %q, want abandoned", repo.byID[c2.ID].Status)
	}
}

func TestLinkChallenge_RoundTrip(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, &fakeSamples{})
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	c, _ := svc.StartCampaign(context.Background(), StartParams{
		UserID: "u1", TitleSlug: "halo", Axis: "combat", AxisKind: AxisKindRadar,
	}, now)

	_ = svc.LinkChallenge(context.Background(), "ch-1", c.ID)
	_ = svc.LinkChallenge(context.Background(), "ch-2", c.ID)
	got, _ := svc.GetByID(context.Background(), c.ID)
	if len(got.LinkedChallengeIDs) != 2 {
		t.Errorf("linked: got %v, want 2 entries", got.LinkedChallengeIDs)
	}
	// Unlink ch-1
	_ = svc.LinkChallenge(context.Background(), "ch-1", "")
	got, _ = svc.GetByID(context.Background(), c.ID)
	if len(got.LinkedChallengeIDs) != 1 {
		t.Errorf("after unlink: got %v, want 1", got.LinkedChallengeIDs)
	}
	if got.LinkedChallengeIDs[0] != "ch-2" {
		t.Errorf("remaining: got %v, want [ch-2]", got.LinkedChallengeIDs)
	}
}

func TestEvaluate_NoData_NoOp(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, &fakeSamples{values: nil})
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)

	c, _ := svc.StartCampaign(context.Background(), StartParams{
		UserID: "u1", TitleSlug: "halo", Axis: "combat", AxisKind: AxisKindRadar,
	}, now)
	// Evaluate happens without samples — should not error.
	if err := svc.Evaluate(context.Background(), c, now.Add(time.Hour)); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if repo.updates != 1 {
		t.Errorf("UpdateEvaluation called %d times, want 1", repo.updates)
	}
}

func TestEvaluate_InactiveCampaign_NoUpdate(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, &fakeSamples{values: []float64{1, 2, 3, 4, 5}})
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)

	c, _ := svc.StartCampaign(context.Background(), StartParams{
		UserID: "u1", TitleSlug: "halo", Axis: "combat", AxisKind: AxisKindRadar,
	}, now)
	_ = svc.PauseCampaign(context.Background(), c.ID)
	updated := repo.byID[c.ID]
	if err := svc.Evaluate(context.Background(), updated, now); err != nil {
		t.Fatalf("Evaluate inactive: %v", err)
	}
	if repo.updates != 0 {
		t.Errorf("paused campaign: UpdateEvaluation called %d times, want 0", repo.updates)
	}
}

func TestEvaluateActive_NoActiveCampaign_NoError(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, &fakeSamples{})
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	if err := svc.EvaluateActive(context.Background(), "no-such-user", "halo", now); err != nil {
		t.Errorf("EvaluateActive no campaign: %v", err)
	}
}

func TestMean_Helper(t *testing.T) {
	if mean(nil) != 0 {
		t.Errorf("mean(nil) = %.2f, want 0", mean(nil))
	}
	if got := mean([]float64{1, 2, 3, 4, 5}); got != 3 {
		t.Errorf("mean([1..5]) = %.2f, want 3", got)
	}
}

func TestImprovementCampaign_FinalValueAndDelta(t *testing.T) {
	raw := 1.2
	lowess := 1.5
	// LOWESS prioritaire sur raw.
	c := ImprovementCampaign{SnapshotValue: 1.0, CurrentValueRaw: &raw, CurrentValueLOWESS: &lowess}
	if fv := c.FinalValue(); fv == nil || *fv != 1.5 {
		t.Errorf("FinalValue (lowess prioritaire): got %v, want 1.5", fv)
	}
	if d := c.Delta(); d == nil || *d != 0.5 {
		t.Errorf("Delta: got %v, want 0.5", d)
	}
	// Sans LOWESS → repli sur raw.
	c2 := ImprovementCampaign{SnapshotValue: 2.0, CurrentValueRaw: &raw}
	if fv := c2.FinalValue(); fv == nil || *fv != 1.2 {
		t.Errorf("FinalValue (repli raw): got %v, want 1.2", fv)
	}
	if d := c2.Delta(); d == nil || *d != -0.8 {
		t.Errorf("Delta (repli raw): got %v, want -0.8", d)
	}
	// Jamais évaluée → nil.
	c3 := ImprovementCampaign{SnapshotValue: 1.0}
	if c3.FinalValue() != nil {
		t.Errorf("FinalValue (non évaluée): want nil, got %v", c3.FinalValue())
	}
	if c3.Delta() != nil {
		t.Errorf("Delta (non évaluée): want nil, got %v", c3.Delta())
	}
}

func TestService_ListEnded_FiltersAndScopes(t *testing.T) {
	repo := newFakeRepo()
	ended := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	repo.byID["done"] = ImprovementCampaign{ID: "done", UserID: "u1", TitleSlug: "halo", Status: StatusCompleted, EndedAt: &ended}
	repo.byID["quit"] = ImprovementCampaign{ID: "quit", UserID: "u1", TitleSlug: "halo", Status: StatusAbandoned, EndedAt: &ended}
	repo.byID["live"] = ImprovementCampaign{ID: "live", UserID: "u1", TitleSlug: "halo", Status: StatusActive}
	repo.byID["other"] = ImprovementCampaign{ID: "other", UserID: "u1", TitleSlug: "halo_5", Status: StatusCompleted, EndedAt: &ended}

	got, err := NewService(repo, &fakeSamples{}).ListEnded(context.Background(), "u1", "halo")
	if err != nil {
		t.Fatalf("ListEnded: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("attendu 2 closes (scope halo), obtenu %d: %+v", len(got), got)
	}
	for _, c := range got {
		if !c.IsEnded() || c.TitleSlug != "halo" {
			t.Errorf("campagne inattendue dans l'historique: %+v", c)
		}
	}
}

// Silence unused: sql import is only for the test fakeRepo's Evaluation param.
var _ = sql.NullFloat64{}
