package prestige

import (
	"context"
	"errors"
	"testing"
	"time"
)

// stubPresetArcRepo : PresetArcRepo configurable pour les tests d'adoption.
type stubPresetArcRepo struct {
	byID       map[string]PresetArc
	stepsByID  map[string][]PresetArcStep
	listResult []PresetArc
}

func (r *stubPresetArcRepo) ListByTitle(_ context.Context, _ string) ([]PresetArc, error) {
	return r.listResult, nil
}
func (r *stubPresetArcRepo) GetByID(_ context.Context, id string) (PresetArc, error) {
	p, ok := r.byID[id]
	if !ok {
		return PresetArc{}, ErrArcNotFound
	}
	return p, nil
}
func (r *stubPresetArcRepo) GetSteps(_ context.Context, presetArcID string) ([]PresetArcStep, error) {
	return r.stepsByID[presetArcID], nil
}
func (r *stubPresetArcRepo) Replace(_ context.Context, _ string, _ []PresetArc, _ []PresetArcStep) error {
	return nil
}

func buildPresetService(preset *stubPresetArcRepo) (*service, *fakeChallengeRepo, *fakeArcRepo, *fakeTemplateRepo) {
	chRepo := &fakeChallengeRepo{}
	arcRepo := &fakeArcRepo{}
	tplRepo := &fakeTemplateRepo{}
	deps := Deps{
		Tuning:           DefaultTuning(),
		Challenges:       chRepo,
		Arcs:             arcRepo,
		Templates:        tplRepo,
		PresetArcs:       preset,
		Telemetry:        &fakeNoOpTelemetryRepo{},
		Prestige:         &fakeNoOpPrestigeRepo{},
		BaselineProvider: &fakeBaselineProvider{},
		Now:              func() time.Time { return time.Now().UTC() },
	}
	return NewService(deps).(*service), chRepo, arcRepo, tplRepo
}

func TestService_ListArcPresets_HydratesSteps(t *testing.T) {
	preset := &stubPresetArcRepo{
		listResult: []PresetArc{{ID: "p1", TitleSlug: "halo_infinite", TitleFR: "Ascension"}},
		stepsByID: map[string][]PresetArcStep{
			"p1": {
				{PresetArcID: "p1", Position: 1, TemplateID: "t-kda", TargetTier: TierNormal},
				{PresetArcID: "p1", Position: 2, TemplateID: "t-kills", TargetTier: TierHeroic},
			},
		},
	}
	svc, _, _, _ := buildPresetService(preset)

	out, err := svc.ListArcPresets(context.Background(), "u1", "halo_infinite")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d presets, want 1", len(out))
	}
	if len(out[0].Steps) != 2 {
		t.Errorf("preset steps not hydrated: got %d, want 2", len(out[0].Steps))
	}
}

func TestService_ListArcPresets_RequiresTitle(t *testing.T) {
	svc, _, _, _ := buildPresetService(&stubPresetArcRepo{})
	_, err := svc.ListArcPresets(context.Background(), "u1", "")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_AdoptPresetArc_OK(t *testing.T) {
	preset := &stubPresetArcRepo{
		byID: map[string]PresetArc{
			"p1": {ID: "p1", TitleSlug: "halo_infinite", TitleFR: "Ascension du Spartan"},
		},
		stepsByID: map[string][]PresetArcStep{
			"p1": {
				{PresetArcID: "p1", Position: 1, TemplateID: "t-kda", TargetTier: TierNormal},
				{PresetArcID: "p1", Position: 2, TemplateID: "t-kills", TargetTier: TierHeroic},
			},
		},
	}
	svc, chRepo, arcRepo, tplRepo := buildPresetService(preset)
	tplRepo.templates = []Template{
		{ID: "t-kda", TitleSlug: "halo_infinite", Metric: "FieldKDA",
			WindowType: WindowSession, Cadence: CadenceFree, EvalType: EvalThreshold,
			NormalTarget: 1.1, HeroicTarget: 1.35, LegendaryTarget: 1.6, MythicTarget: 2.0},
		{ID: "t-kills", TitleSlug: "halo_infinite", Metric: "FieldKills",
			WindowType: WindowSession, Cadence: CadenceFree, EvalType: EvalThreshold,
			NormalTarget: 50, HeroicTarget: 80, LegendaryTarget: 120, MythicTarget: 200},
	}

	arc, err := svc.AdoptPresetArc(context.Background(), "u1", "halo_infinite", "p1")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !arc.IsPreset || arc.PresetID != "p1" {
		t.Errorf("arc should be a preset adoption: %+v", arc)
	}
	if arc.Title != "Ascension du Spartan" {
		t.Errorf("arc title not from preset: %q", arc.Title)
	}
	if len(arcRepo.created) != 1 {
		t.Errorf("expected 1 arc created, got %d", len(arcRepo.created))
	}
	if chRepo.createCount != 2 {
		t.Errorf("expected 2 objectives created, got %d", chRepo.createCount)
	}
	// Origine : l'adoption d'un preset arc vient du joueur → source "user" (ADR 0020).
	for _, c := range chRepo.createdChallenges {
		if c.Source != ChallengeSourceUser {
			t.Errorf("preset objective %s: source=%q want %q", c.ID, c.Source, ChallengeSourceUser)
		}
	}
}

func TestService_AdoptPresetArc_WrongTitle(t *testing.T) {
	preset := &stubPresetArcRepo{
		byID: map[string]PresetArc{"p1": {ID: "p1", TitleSlug: "other_title"}},
	}
	svc, _, arcRepo, _ := buildPresetService(preset)

	_, err := svc.AdoptPresetArc(context.Background(), "u1", "halo_infinite", "p1")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
	if len(arcRepo.created) != 0 {
		t.Error("no arc should be created on title mismatch")
	}
}

func TestService_AdoptPresetArc_NotFound(t *testing.T) {
	svc, _, _, _ := buildPresetService(&stubPresetArcRepo{byID: map[string]PresetArc{}})
	_, err := svc.AdoptPresetArc(context.Background(), "u1", "halo_infinite", "missing")
	if !errors.Is(err, ErrArcNotFound) {
		t.Errorf("expected ErrArcNotFound, got %v", err)
	}
}

// Une étape référençant un template inconnu est ignorée (best-effort) : l'arc
// est tout de même créé avec les objectifs valides.
func TestService_AdoptPresetArc_SkipsMissingTemplate(t *testing.T) {
	preset := &stubPresetArcRepo{
		byID: map[string]PresetArc{"p1": {ID: "p1", TitleSlug: "halo_infinite", TitleFR: "Arc"}},
		stepsByID: map[string][]PresetArcStep{
			"p1": {
				{PresetArcID: "p1", Position: 1, TemplateID: "t-kda", TargetTier: TierNormal},
				{PresetArcID: "p1", Position: 2, TemplateID: "t-unknown", TargetTier: TierHeroic},
			},
		},
	}
	svc, chRepo, arcRepo, tplRepo := buildPresetService(preset)
	tplRepo.templates = []Template{
		{ID: "t-kda", TitleSlug: "halo_infinite", Metric: "FieldKDA",
			WindowType: WindowSession, Cadence: CadenceFree, EvalType: EvalThreshold,
			NormalTarget: 1.1, HeroicTarget: 1.35, LegendaryTarget: 1.6, MythicTarget: 2.0},
	}

	arc, err := svc.AdoptPresetArc(context.Background(), "u1", "halo_infinite", "p1")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(arcRepo.created) != 1 {
		t.Errorf("arc should still be created, got %d", len(arcRepo.created))
	}
	if chRepo.createCount != 1 {
		t.Errorf("only the valid step should create an objective, got %d", chRepo.createCount)
	}
	_ = arc
}

func TestService_AdoptPresetArc_RequiresArgs(t *testing.T) {
	svc, _, _, _ := buildPresetService(&stubPresetArcRepo{})
	for _, tc := range []struct{ uid, title, preset string }{
		{"", "halo_infinite", "p1"},
		{"u1", "", "p1"},
		{"u1", "halo_infinite", ""},
	} {
		_, err := svc.AdoptPresetArc(context.Background(), tc.uid, tc.title, tc.preset)
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("AdoptPresetArc(%q,%q,%q): expected ErrInvalidInput, got %v", tc.uid, tc.title, tc.preset, err)
		}
	}
}
