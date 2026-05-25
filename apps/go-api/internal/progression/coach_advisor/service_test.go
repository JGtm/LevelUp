package coach_advisor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/progression/coach_advisor"
)

// fakeRepo est un Repo en mémoire pour tests unitaires du Service.
type fakeRepo struct {
	proposals map[string]coach_advisor.Proposal
	// errors injectables pour simuler les pannes
	errOnMarkObsoleted error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{proposals: map[string]coach_advisor.Proposal{}}
}

func (r *fakeRepo) Create(_ context.Context, p coach_advisor.Proposal) error {
	if _, exists := r.proposals[p.ID]; exists {
		return errors.New("duplicate")
	}
	r.proposals[p.ID] = p
	return nil
}

func (r *fakeRepo) Get(_ context.Context, id string) (coach_advisor.Proposal, error) {
	p, ok := r.proposals[id]
	if !ok {
		return coach_advisor.Proposal{}, coach_advisor.ErrProposalNotFound
	}
	return p, nil
}

func (r *fakeRepo) ListByUser(_ context.Context, userID, titleSlug string, status coach_advisor.ProposalStatus) ([]coach_advisor.Proposal, error) {
	var out []coach_advisor.Proposal
	for _, p := range r.proposals {
		if p.UserID != userID || p.TitleSlug != titleSlug {
			continue
		}
		if status != "" && p.Status != status {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *fakeRepo) ListPendingBySignalScope(_ context.Context, userID, titleSlug, metric, axis string) ([]coach_advisor.Proposal, error) {
	var out []coach_advisor.Proposal
	for _, p := range r.proposals {
		if p.UserID != userID || p.TitleSlug != titleSlug || p.Status != coach_advisor.ProposalPending {
			continue
		}
		if p.SourceMetric == metric || p.RadarAxis == axis {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *fakeRepo) ListPendingByAxis(_ context.Context, userID, titleSlug, axis string) ([]coach_advisor.Proposal, error) {
	var out []coach_advisor.Proposal
	for _, p := range r.proposals {
		if p.UserID == userID && p.TitleSlug == titleSlug && p.Status == coach_advisor.ProposalPending && p.RadarAxis == axis {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *fakeRepo) MarkAccepted(_ context.Context, id, ref string, now time.Time) error {
	p, ok := r.proposals[id]
	if !ok {
		return coach_advisor.ErrProposalNotFound
	}
	p.Status = coach_advisor.ProposalAccepted
	p.ResolvedAt = &now
	p.ResolvedRef = ref
	r.proposals[id] = p
	return nil
}

func (r *fakeRepo) MarkDismissed(_ context.Context, id string, now time.Time) error {
	p, ok := r.proposals[id]
	if !ok {
		return coach_advisor.ErrProposalNotFound
	}
	p.Status = coach_advisor.ProposalDismissed
	p.ResolvedAt = &now
	r.proposals[id] = p
	return nil
}

func (r *fakeRepo) MarkSuperseded(_ context.Context, id, newID string, now time.Time) error {
	p, ok := r.proposals[id]
	if !ok || p.Status != coach_advisor.ProposalPending {
		return nil
	}
	p.Status = coach_advisor.ProposalSuperseded
	p.SupersededBy = newID
	p.SupersededAt = &now
	r.proposals[id] = p
	return nil
}

func (r *fakeRepo) MarkObsoleted(_ context.Context, id string, now time.Time) error {
	if r.errOnMarkObsoleted != nil {
		return r.errOnMarkObsoleted
	}
	p, ok := r.proposals[id]
	if !ok || p.Status != coach_advisor.ProposalPending {
		return nil
	}
	p.Status = coach_advisor.ProposalObsoleted
	p.ObsoletedAt = &now
	r.proposals[id] = p
	return nil
}

// ─── Tests ───

func mkPending(id, userID, axis, metric string) coach_advisor.Proposal {
	return coach_advisor.Proposal{
		ID:           id,
		UserID:       userID,
		TitleSlug:    "halo_infinite",
		Kind:         coach_advisor.ProposalKindChallenge,
		SourceSignal: coach_advisor.SignalLOWESSPositive,
		SourceMetric: metric,
		RadarAxis:    axis,
		Origin:       coach_advisor.OriginCatalog,
		Status:       coach_advisor.ProposalPending,
	}
}

func TestService_ListProposals_RequiresUserAndTitle(t *testing.T) {
	svc := coach_advisor.NewService(coach_advisor.ServiceDeps{Repo: newFakeRepo()})
	if _, err := svc.ListProposals(context.Background(), "", "halo_infinite", ""); err == nil {
		t.Error("expected error on empty userID")
	}
	if _, err := svc.ListProposals(context.Background(), "u1", "", ""); err == nil {
		t.Error("expected error on empty titleSlug")
	}
}

func TestService_ListProposals_FiltersByStatus(t *testing.T) {
	repo := newFakeRepo()
	ctx := context.Background()
	_ = repo.Create(ctx, mkPending("p1", "u1", "combat", "accuracy"))
	_ = repo.Create(ctx, mkPending("p2", "u1", "combat", "kda"))
	_ = repo.MarkDismissed(ctx, "p2", time.Now())

	svc := coach_advisor.NewService(coach_advisor.ServiceDeps{Repo: repo})
	pending, err := svc.ListProposals(ctx, "u1", "halo_infinite", coach_advisor.ProposalPending)
	if err != nil {
		t.Fatalf("ListProposals: %v", err)
	}
	if len(pending) != 1 || pending[0].ID != "p1" {
		t.Errorf("expected 1 pending (p1), got %v", pending)
	}
}

func TestService_DismissProposal_RequiresID(t *testing.T) {
	svc := coach_advisor.NewService(coach_advisor.ServiceDeps{Repo: newFakeRepo()})
	if err := svc.DismissProposal(context.Background(), ""); err == nil {
		t.Error("expected error on empty id")
	}
}

func TestService_DismissProposal_TransitionsStatus(t *testing.T) {
	repo := newFakeRepo()
	ctx := context.Background()
	_ = repo.Create(ctx, mkPending("p1", "u1", "combat", "accuracy"))

	svc := coach_advisor.NewService(coach_advisor.ServiceDeps{Repo: repo})
	if err := svc.DismissProposal(ctx, "p1"); err != nil {
		t.Fatalf("DismissProposal: %v", err)
	}
	got, _ := repo.Get(ctx, "p1")
	if got.Status != coach_advisor.ProposalDismissed {
		t.Errorf("Status: got %q, want dismissed", got.Status)
	}
	if got.ResolvedAt == nil {
		t.Error("ResolvedAt should be set after dismiss")
	}
}

func TestService_ObsoletePendingForAxis_BatchAndCount(t *testing.T) {
	repo := newFakeRepo()
	ctx := context.Background()
	_ = repo.Create(ctx, mkPending("p1", "u1", "combat", "accuracy"))
	_ = repo.Create(ctx, mkPending("p2", "u1", "combat", "kda"))
	_ = repo.Create(ctx, mkPending("p3", "u1", "score", "win_rate"))

	svc := coach_advisor.NewService(coach_advisor.ServiceDeps{Repo: repo})
	n, err := svc.ObsoletePendingForAxis(ctx, "u1", "halo_infinite", "combat")
	if err != nil {
		t.Fatalf("ObsoletePendingForAxis: %v", err)
	}
	if n != 2 {
		t.Errorf("count: got %d, want 2", n)
	}
	got1, _ := repo.Get(ctx, "p1")
	got2, _ := repo.Get(ctx, "p2")
	got3, _ := repo.Get(ctx, "p3")
	if got1.Status != coach_advisor.ProposalObsoleted || got2.Status != coach_advisor.ProposalObsoleted {
		t.Errorf("p1/p2 not obsoleted: %q / %q", got1.Status, got2.Status)
	}
	if got3.Status != coach_advisor.ProposalPending {
		t.Errorf("p3 should stay pending (different axis), got %q", got3.Status)
	}
}

func TestService_ObsoletePendingForAxis_RequiresAllParams(t *testing.T) {
	svc := coach_advisor.NewService(coach_advisor.ServiceDeps{Repo: newFakeRepo()})
	if _, err := svc.ObsoletePendingForAxis(context.Background(), "", "halo_infinite", "combat"); err == nil {
		t.Error("expected error on empty userID")
	}
	if _, err := svc.ObsoletePendingForAxis(context.Background(), "u1", "", "combat"); err == nil {
		t.Error("expected error on empty titleSlug")
	}
	if _, err := svc.ObsoletePendingForAxis(context.Background(), "u1", "halo_infinite", ""); err == nil {
		t.Error("expected error on empty axis")
	}
}

func TestService_ObsoletePendingForAxis_RepoFailure_LoggedNotPropagated(t *testing.T) {
	repo := newFakeRepo()
	ctx := context.Background()
	_ = repo.Create(ctx, mkPending("p1", "u1", "combat", "accuracy"))
	_ = repo.Create(ctx, mkPending("p2", "u1", "combat", "kda"))

	repo.errOnMarkObsoleted = errors.New("simulated repo failure")
	svc := coach_advisor.NewService(coach_advisor.ServiceDeps{Repo: repo})

	// Best-effort : retourne (0, nil) — erreurs individuelles loggées sans interrompre
	n, err := svc.ObsoletePendingForAxis(ctx, "u1", "halo_infinite", "combat")
	if err != nil {
		t.Fatalf("err should be nil (best-effort): %v", err)
	}
	if n != 0 {
		t.Errorf("count should be 0 on all-fail, got %d", n)
	}
}
