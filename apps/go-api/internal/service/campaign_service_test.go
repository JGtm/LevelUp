package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/campaign"
	"levelup/go-api/internal/port"
)

// fakeCampaignRepo : dépôt de campagne en mémoire (campaign.Repo).
type fakeCampaignRepo struct {
	byID     map[string]campaign.ImprovementCampaign
	inserted []campaign.ImprovementCampaign
	gotUser  string
	gotTitle string
}

func (f *fakeCampaignRepo) Insert(_ context.Context, c campaign.ImprovementCampaign) error {
	f.inserted = append(f.inserted, c)
	if f.byID == nil {
		f.byID = map[string]campaign.ImprovementCampaign{}
	}
	f.byID[c.ID] = c
	return nil
}

func (f *fakeCampaignRepo) GetByID(_ context.Context, id string) (campaign.ImprovementCampaign, error) {
	c, ok := f.byID[id]
	if !ok {
		return campaign.ImprovementCampaign{}, campaign.ErrNotFound
	}
	return c, nil
}

func (f *fakeCampaignRepo) GetActive(_ context.Context, user, title string) (campaign.ImprovementCampaign, error) {
	f.gotUser, f.gotTitle = user, title
	return campaign.ImprovementCampaign{}, campaign.ErrNotFound
}

func (f *fakeCampaignRepo) ListEnded(_ context.Context, user, title string) ([]campaign.ImprovementCampaign, error) {
	f.gotUser, f.gotTitle = user, title
	return []campaign.ImprovementCampaign{{ID: "old"}}, nil
}

func (f *fakeCampaignRepo) UpdateStatus(_ context.Context, id string, st campaign.CampaignStatus, _ *time.Time) error {
	c := f.byID[id]
	c.Status = st
	f.byID[id] = c
	return nil
}

func (f *fakeCampaignRepo) UpdateEvaluation(context.Context, string, campaign.Evaluation) error {
	return nil
}

func (f *fakeCampaignRepo) LinkedChallengeIDs(context.Context, string) ([]string, error) {
	return nil, nil
}

func (f *fakeCampaignRepo) LinkChallenge(context.Context, string, string) error { return nil }

// fakeSamples : échantillons d'axe en mémoire (campaign.SampleProvider).
type fakeSamples struct{}

func (fakeSamples) LoadAxisSamples(context.Context, string, string, string, campaign.AxisKind, string, time.Time, time.Time) ([]float64, error) {
	return []float64{1, 2, 3}, nil
}

func newCampaignSvc(repo *fakeCampaignRepo) *CampaignService {
	return NewCampaignService(func(_ context.Context, slug string) (port.CampaignScope, error) {
		if slug != "jg" {
			return port.CampaignScope{}, errors.New("inconnu")
		}
		return port.CampaignScope{XUID: "xuid-jg", Repo: repo, Samples: fakeSamples{}}, nil
	})
}

func TestCampaignService_JoueurIntrouvableSurToutesLesOperations(t *testing.T) {
	svc := newCampaignSvc(&fakeCampaignRepo{})
	ctx, now := context.Background(), time.Now()
	_, e1 := svc.Start(ctx, "x", campaign.StartParams{}, now)
	_, e2 := svc.Active(ctx, "x", "halo_infinite")
	_, e3 := svc.Ended(ctx, "x", "halo_infinite")
	_, e4 := svc.ByID(ctx, "x", "c1")
	errs := []error{e1, e2, e3, e4, svc.Pause(ctx, "x", "c1"), svc.Resume(ctx, "x", "c1"), svc.Close(ctx, "x", "c1", now), svc.Abandon(ctx, "x", "c1", now)}
	for i, err := range errs {
		var pe *port.ProgressionError
		if !errors.As(err, &pe) || pe.Kind != port.ProgErrPlayerNotFound {
			t.Errorf("opération %d : %v ; attendu ProgErrPlayerNotFound", i, err)
		}
	}
}

func TestCampaignService_LesLecturesPortentLeJoueurResolu(t *testing.T) {
	repo := &fakeCampaignRepo{}
	svc := newCampaignSvc(repo)
	if _, err := svc.Active(context.Background(), "jg", "halo_5"); !errors.Is(err, campaign.ErrNotFound) {
		t.Fatalf("Active = %v ; le refus du dépôt doit passer tel quel", err)
	}
	if repo.gotUser != "xuid-jg" || repo.gotTitle != "halo_5" {
		t.Errorf("Active lit (%q, %q) ; attendu (xuid-jg, halo_5)", repo.gotUser, repo.gotTitle)
	}
	list, err := svc.Ended(context.Background(), "jg", "halo_infinite")
	if err != nil || len(list) != 1 || repo.gotUser != "xuid-jg" {
		t.Fatalf("Ended = %v, %v, utilisateur %q", list, err, repo.gotUser)
	}
}

func TestCampaignService_StartImposeLeJoueurResolu(t *testing.T) {
	repo := &fakeCampaignRepo{}
	svc := newCampaignSvc(repo)
	c, err := svc.Start(context.Background(), "jg", campaign.StartParams{
		UserID: "usurpe", TitleSlug: "halo_infinite", Axis: "combat", AxisKind: campaign.AxisKindRadar,
	}, time.Now())
	if err != nil {
		t.Fatalf("Start : %v", err)
	}
	if c.UserID != "xuid-jg" || len(repo.inserted) != 1 || repo.inserted[0].UserID != "xuid-jg" {
		t.Fatalf("la campagne doit appartenir au joueur résolu : %+v", c)
	}
	_, err = svc.Start(context.Background(), "jg", campaign.StartParams{TitleSlug: "halo_infinite", Axis: "", AxisKind: campaign.AxisKindRadar}, time.Now())
	if !errors.Is(err, campaign.ErrInvalidAxis) {
		t.Fatalf("axe vide : %v ; attendu campaign.ErrInvalidAxis", err)
	}
}

func TestCampaignService_TransitionDeLaCampagne(t *testing.T) {
	repo := &fakeCampaignRepo{byID: map[string]campaign.ImprovementCampaign{
		"c1": {ID: "c1", Status: campaign.StatusActive},
	}}
	svc := newCampaignSvc(repo)
	ctx := context.Background()
	if err := svc.Pause(ctx, "jg", "c1"); err != nil || repo.byID["c1"].Status != campaign.StatusPaused {
		t.Fatalf("Pause : %v, statut %q", err, repo.byID["c1"].Status)
	}
	if err := svc.Resume(ctx, "jg", "c1"); err != nil || repo.byID["c1"].Status != campaign.StatusActive {
		t.Fatalf("Resume : %v, statut %q", err, repo.byID["c1"].Status)
	}
	if err := svc.Close(ctx, "jg", "c1", time.Now()); err != nil || repo.byID["c1"].Status != campaign.StatusCompleted {
		t.Fatalf("Close : %v, statut %q", err, repo.byID["c1"].Status)
	}
	if err := svc.Pause(ctx, "jg", "c1"); !errors.Is(err, campaign.ErrInvalidStatus) {
		t.Fatalf("Pause d'une campagne close : %v ; attendu ErrInvalidStatus", err)
	}
	if _, err := svc.ByID(ctx, "jg", "absente"); !errors.Is(err, campaign.ErrNotFound) {
		t.Fatalf("ByID absente : %v", err)
	}
}
