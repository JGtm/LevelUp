package coach_advisor_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/prestige"
	"levelup/go-api/internal/progression/coach_advisor"
)

// ─── Fakes spécifiques Phase 7 ───

// fakeTemplateRepo : in-memory implementation of prestige.TemplateRepo
type fakeTemplateRepo struct {
	templates map[string]prestige.Template
}

func newFakeTemplateRepo() *fakeTemplateRepo {
	return &fakeTemplateRepo{templates: map[string]prestige.Template{}}
}

func (r *fakeTemplateRepo) ListByTitle(_ context.Context, slug string) ([]prestige.Template, error) {
	var out []prestige.Template
	for _, t := range r.templates {
		if t.TitleSlug == slug {
			out = append(out, t)
		}
	}
	return out, nil
}

func (r *fakeTemplateRepo) GetByID(_ context.Context, id string) (prestige.Template, error) {
	t, ok := r.templates[id]
	if !ok {
		return prestige.Template{}, errors.New("template not found")
	}
	return t, nil
}

func (r *fakeTemplateRepo) Suggest(_ context.Context, _ string, _ []string, _ int) ([]prestige.Template, error) {
	return nil, nil
}

func (r *fakeTemplateRepo) Replace(_ context.Context, _ string, ts []prestige.Template) error {
	for _, t := range ts {
		r.templates[t.ID] = t
	}
	return nil
}

func (r *fakeTemplateRepo) UpsertOne(_ context.Context, t prestige.Template) error {
	r.templates[t.ID] = t
	return nil
}

// fakePrestigeWriter : in-memory implementation of coach_advisor.PrestigeWriter
type fakePrestigeWriter struct {
	createdChallenges []prestige.CreateChallengeRequest
	createdArcs       []prestige.CreateArcRequest
	chCounter         int
	arcCounter        int
}

func (w *fakePrestigeWriter) CreateChallenge(_ context.Context, req prestige.CreateChallengeRequest) (prestige.Challenge, error) {
	w.createdChallenges = append(w.createdChallenges, req)
	w.chCounter++
	id := fakeChallengeID(w.chCounter)
	return prestige.Challenge{ID: id, UserID: req.UserID, TitleSlug: req.TitleSlug, TemplateID: req.TemplateID, ArcID: req.ArcID}, nil
}

func (w *fakePrestigeWriter) CreateArc(_ context.Context, req prestige.CreateArcRequest) (prestige.Arc, error) {
	w.createdArcs = append(w.createdArcs, req)
	w.arcCounter++
	id := fakeArcID(w.arcCounter)
	return prestige.Arc{ID: id, UserID: req.UserID, TitleSlug: req.TitleSlug, Title: req.Title}, nil
}

func fakeChallengeID(n int) string { return "ch_test_" + intToStr(n) }
func fakeArcID(n int) string       { return "arc_test_" + intToStr(n) }

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// catalogWithAccuracy ajoute un template accuracy → axis combat dans le repo.
func catalogWithAccuracy(repo *fakeTemplateRepo) {
	repo.templates["catalog_accuracy"] = prestige.Template{
		ID:              "catalog_accuracy",
		TitleSlug:       "halo_infinite",
		Metric:          "accuracy",
		WindowType:      "rolling_days",
		WindowValue:     "14",
		Cadence:         "weekly",
		EvalType:        "threshold",
		ModeFilter:      "universal",
		LabelEN:         "Accuracy",
		LabelFR:         "Accuracy",
		NormalTarget:    0.45,
		HeroicTarget:    0.50,
		LegendaryTarget: 0.55,
		MythicTarget:    0.60,
		LUSRComponents:  []string{"accuracy_delta"},
		RadarAxes:       []string{"combat"},
		Source:          "catalog",
	}
}

// loadGrammar charge le sampleGrammarTOML défini dans synthesizer_test.go
func loadGrammar(t *testing.T) coach_advisor.SynthesisGrammar {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "g.toml")
	if err := os.WriteFile(path, []byte(sampleGrammarTOML), 0o644); err != nil {
		t.Fatalf("write grammar: %v", err)
	}
	g, err := coach_advisor.LoadSynthesisGrammar(path)
	if err != nil {
		t.Fatalf("LoadSynthesisGrammar: %v", err)
	}
	return g
}

// buildFullDeps : un Service avec toutes les deps Phase 7.
func buildFullDeps(t *testing.T) (*fakeRepo, *fakeTemplateRepo, *fakePrestigeWriter, coach_advisor.ServiceDeps) {
	t.Helper()
	repo := newFakeRepo()
	tplRepo := newFakeTemplateRepo()
	pw := &fakePrestigeWriter{}
	deps := coach_advisor.ServiceDeps{
		Repo:        repo,
		Templates:   tplRepo,
		Synthesizer: coach_advisor.NewSynthesizer(loadGrammar(t), coach_advisor.DefaultSynthesisConfig()),
		Prestige:    pw,
		Now:         func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
	}
	return repo, tplRepo, pw, deps
}

// ─── Tests GenerateProposals ───

func TestGenerateProposals_ShortCircuitsWhenDisabled(t *testing.T) {
	_, _, _, deps := buildFullDeps(t)
	svc := coach_advisor.NewService(deps)
	out, err := svc.GenerateProposals(context.Background(), coach_advisor.GenerateInput{
		UserID:           "u1",
		TitleSlug:        "halo_infinite",
		ProactiveEnabled: false,
		Signals: []coach_advisor.Signal{
			{Kind: coach_advisor.SignalLOWESSPositive, Metric: "accuracy", Strength: 0.9},
		},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected no proposals when disabled, got %d", len(out))
	}
}

func TestGenerateProposals_RequiresUserAndTitle(t *testing.T) {
	_, _, _, deps := buildFullDeps(t)
	svc := coach_advisor.NewService(deps)
	_, err := svc.GenerateProposals(context.Background(), coach_advisor.GenerateInput{
		UserID:           "",
		TitleSlug:        "halo_infinite",
		ProactiveEnabled: true,
	})
	if err == nil {
		t.Error("expected error on empty userID")
	}
}

func TestGenerateProposals_CatalogMatch_CreatesChallengeProposal(t *testing.T) {
	repo, tplRepo, _, deps := buildFullDeps(t)
	catalogWithAccuracy(tplRepo)
	svc := coach_advisor.NewService(deps)

	out, err := svc.GenerateProposals(context.Background(), coach_advisor.GenerateInput{
		UserID:           "u1",
		TitleSlug:        "halo_infinite",
		ProactiveEnabled: true,
		Signals: []coach_advisor.Signal{
			{
				Kind:          coach_advisor.SignalLOWESSPositive,
				Metric:        "accuracy",
				LUSRComponent: "accuracy_delta",
				RadarAxis:     "combat",
				Strength:      0.75,
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(out))
	}
	p := out[0]
	if p.Kind != coach_advisor.ProposalKindChallenge {
		t.Errorf("Kind: got %q, want challenge", p.Kind)
	}
	if p.TemplateID != "catalog_accuracy" {
		t.Errorf("TemplateID: got %q, want catalog_accuracy", p.TemplateID)
	}
	if p.Origin != coach_advisor.OriginCatalog {
		t.Errorf("Origin: got %q, want catalog", p.Origin)
	}
	if _, err := repo.Get(context.Background(), p.ID); err != nil {
		t.Errorf("proposal not persisted: %v", err)
	}
}

func TestGenerateProposals_SynthesisFallback_CreatesSynthesizedProposal(t *testing.T) {
	repo, tplRepo, _, deps := buildFullDeps(t)
	// Pas de catalog → fallback synthesis (accuracy est dans la grammaire test)
	svc := coach_advisor.NewService(deps)

	out, err := svc.GenerateProposals(context.Background(), coach_advisor.GenerateInput{
		UserID:           "u1",
		TitleSlug:        "halo_infinite",
		ProactiveEnabled: true,
		Signals: []coach_advisor.Signal{
			{
				Kind:          coach_advisor.SignalLOWESSPositive,
				Metric:        "accuracy",
				LUSRComponent: "accuracy_delta",
				RadarAxis:     "combat",
				Strength:      0.75,
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(out) != 1 || out[0].Origin != coach_advisor.OriginSynthesized {
		t.Fatalf("expected 1 synthesized proposal, got %v", out)
	}
	// Le template synthétisé doit être dans le repo
	if len(tplRepo.templates) != 1 {
		t.Errorf("synthesized template should be persisted, got %d templates", len(tplRepo.templates))
	}
	for _, tmpl := range tplRepo.templates {
		if tmpl.Source != "coach_synthesized" {
			t.Errorf("template source: got %q, want coach_synthesized", tmpl.Source)
		}
	}
	if _, err := repo.Get(context.Background(), out[0].ID); err != nil {
		t.Errorf("proposal not persisted: %v", err)
	}
}

func TestGenerateProposals_WeakSignalNoCatalog_Skipped(t *testing.T) {
	_, _, _, deps := buildFullDeps(t)
	svc := coach_advisor.NewService(deps)

	out, _ := svc.GenerateProposals(context.Background(), coach_advisor.GenerateInput{
		UserID:           "u1",
		TitleSlug:        "halo_infinite",
		ProactiveEnabled: true,
		Signals: []coach_advisor.Signal{
			{
				Kind:     coach_advisor.SignalLOWESSPositive,
				Metric:   "accuracy",
				Strength: 0.3, // < synthesis_min_strength (0.6)
			},
		},
	})
	if len(out) != 0 {
		t.Errorf("expected no proposal for weak signal, got %d", len(out))
	}
}

func TestGenerateProposals_ConvergentSignals_ComposesArc(t *testing.T) {
	_, _, _, deps := buildFullDeps(t)
	svc := coach_advisor.NewService(deps)

	out, err := svc.GenerateProposals(context.Background(), coach_advisor.GenerateInput{
		UserID:           "u1",
		TitleSlug:        "halo_infinite",
		ProactiveEnabled: true,
		Signals: []coach_advisor.Signal{
			{Kind: coach_advisor.SignalLOWESSPositive, Metric: "accuracy", LUSRComponent: "accuracy_delta", RadarAxis: "combat", Strength: 0.85},
			{Kind: coach_advisor.SignalCombatPatternActive, Metric: "kda", RadarAxis: "combat", Strength: 0.75},
		},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// 2 signaux convergents sur "combat" → 1 arc proposal (challenges individuels filtrés)
	var arcs, challenges int
	for _, p := range out {
		if p.Kind == coach_advisor.ProposalKindArc {
			arcs++
		} else {
			challenges++
		}
	}
	if arcs != 1 {
		t.Errorf("expected 1 arc proposal, got %d", arcs)
	}
	if challenges != 0 {
		t.Errorf("expected 0 challenge proposals (covered by arc), got %d", challenges)
	}
}

func TestGenerateProposals_SupersedeOlderWeakerProposal(t *testing.T) {
	repo, tplRepo, _, deps := buildFullDeps(t)
	catalogWithAccuracy(tplRepo)
	ctx := context.Background()

	// Sync 1 : signal faible 0.6 → proposal P_old (strength=0.6)
	svc := coach_advisor.NewService(deps)
	out1, _ := svc.GenerateProposals(ctx, coach_advisor.GenerateInput{
		UserID:           "u1",
		TitleSlug:        "halo_infinite",
		ProactiveEnabled: true,
		Signals: []coach_advisor.Signal{
			{Kind: coach_advisor.SignalLOWESSPositive, Metric: "accuracy", LUSRComponent: "accuracy_delta", RadarAxis: "combat", Strength: 0.6},
		},
	})
	if len(out1) != 1 {
		t.Fatalf("sync1: expected 1 proposal, got %d", len(out1))
	}
	oldID := out1[0].ID

	// Sync 2 : signal fort 0.95 → P_new supersède P_old
	out2, _ := svc.GenerateProposals(ctx, coach_advisor.GenerateInput{
		UserID:           "u1",
		TitleSlug:        "halo_infinite",
		ProactiveEnabled: true,
		Signals: []coach_advisor.Signal{
			{Kind: coach_advisor.SignalLOWESSPositive, Metric: "accuracy", LUSRComponent: "accuracy_delta", RadarAxis: "combat", Strength: 0.95},
		},
	})
	if len(out2) != 1 {
		t.Fatalf("sync2: expected 1 proposal, got %d", len(out2))
	}
	newID := out2[0].ID

	old, _ := repo.Get(ctx, oldID)
	if old.Status != coach_advisor.ProposalSuperseded {
		t.Errorf("old status: got %q, want superseded", old.Status)
	}
	if old.SupersededBy != newID {
		t.Errorf("SupersededBy: got %q, want %q", old.SupersededBy, newID)
	}
}

func TestGenerateProposals_CappedAtMaxProposalsPerSync(t *testing.T) {
	_, tplRepo, _, deps := buildFullDeps(t)
	catalogWithAccuracy(tplRepo)
	deps.Tuning = coach_advisor.Tuning{
		MinCatalogMatchScore:       0.4,
		MaxProposalsPerSync:        2, // cap dur à 2
		SupersessionStrengthUplift: 1.10,
	}
	svc := coach_advisor.NewService(deps)

	// 4 signaux différents — devrait produire 4 proposals avant cap, capé à 2
	out, _ := svc.GenerateProposals(context.Background(), coach_advisor.GenerateInput{
		UserID:           "u1",
		TitleSlug:        "halo_infinite",
		ProactiveEnabled: true,
		Signals: []coach_advisor.Signal{
			{Kind: coach_advisor.SignalLOWESSPositive, Metric: "accuracy", LUSRComponent: "accuracy_delta", RadarAxis: "combat", Strength: 0.9},
			{Kind: coach_advisor.SignalLOWESSPositive, Metric: "kda", RadarAxis: "combat", Strength: 0.8},
			{Kind: coach_advisor.SignalMilestoneApproach, Metric: "win_rate", RadarAxis: "score", Strength: 0.75},
			{Kind: coach_advisor.SignalRecordApproach, Metric: "performance_score", RadarAxis: "score", Strength: 0.7},
		},
	})
	if len(out) > 2 {
		t.Errorf("expected cap at 2, got %d", len(out))
	}
}

// ─── Tests AcceptProposal ───

func TestAcceptProposal_NotFound_ReturnsError(t *testing.T) {
	_, _, _, deps := buildFullDeps(t)
	svc := coach_advisor.NewService(deps)
	_, err := svc.AcceptProposal(context.Background(), "nonexistent")
	if !errors.Is(err, coach_advisor.ErrProposalNotFound) {
		t.Errorf("expected ErrProposalNotFound, got %v", err)
	}
}

func TestAcceptProposal_AlreadyAccepted_ReturnsNotAcceptable(t *testing.T) {
	repo, _, _, deps := buildFullDeps(t)
	ctx := context.Background()
	p := mkPending("p1", "u1", "combat", "accuracy")
	p.Status = coach_advisor.ProposalAccepted
	_ = repo.Create(ctx, p)

	svc := coach_advisor.NewService(deps)
	_, err := svc.AcceptProposal(ctx, "p1")
	if !errors.Is(err, coach_advisor.ErrProposalNotAcceptable) {
		t.Errorf("expected ErrProposalNotAcceptable, got %v", err)
	}
}

func TestAcceptProposal_Challenge_HappyPath(t *testing.T) {
	repo, tplRepo, pw, deps := buildFullDeps(t)
	catalogWithAccuracy(tplRepo)
	ctx := context.Background()

	p := mkPending("p1", "u1", "combat", "accuracy")
	p.TemplateID = "catalog_accuracy"
	_ = repo.Create(ctx, p)

	svc := coach_advisor.NewService(deps)
	res, err := svc.AcceptProposal(ctx, "p1")
	if err != nil {
		t.Fatalf("AcceptProposal: %v", err)
	}
	if res.ChallengeID == "" {
		t.Error("expected ChallengeID, got empty")
	}
	if len(pw.createdChallenges) != 1 {
		t.Fatalf("expected 1 challenge created, got %d", len(pw.createdChallenges))
	}
	req := pw.createdChallenges[0]
	if req.Source != "coach" {
		t.Errorf("Source: got %q, want coach", req.Source)
	}
	if req.TemplateID != "catalog_accuracy" {
		t.Errorf("TemplateID: got %q", req.TemplateID)
	}
	// Vérif transition status
	got, _ := repo.Get(ctx, "p1")
	if got.Status != coach_advisor.ProposalAccepted {
		t.Errorf("Status: got %q, want accepted", got.Status)
	}
	if got.ResolvedRef != res.ChallengeID {
		t.Errorf("ResolvedRef: got %q, want %q", got.ResolvedRef, res.ChallengeID)
	}
}

func TestAcceptProposal_Arc_CreatesArcAndChallenges(t *testing.T) {
	repo, tplRepo, pw, deps := buildFullDeps(t)
	catalogWithAccuracy(tplRepo)
	// Ajouter 2 autres templates pour les steps de l'arc
	tplRepo.templates["catalog_kda"] = prestige.Template{
		ID: "catalog_kda", TitleSlug: "halo_infinite", Metric: "kda",
		LUSRComponents: nil, RadarAxes: []string{"combat"}, Source: "catalog",
	}
	ctx := context.Background()

	// Manuellement créer une proposal arc avec ChallengesSpec valide
	arcProp := coach_advisor.Proposal{
		ID:        "arc1",
		UserID:    "u1",
		TitleSlug: "halo_infinite",
		Kind:      coach_advisor.ProposalKindArc,
		ChallengesSpec: `[
			{"position":1,"template_id":"catalog_accuracy","suggested_tier":"normal","signal_kind":"lowess_positive"},
			{"position":2,"template_id":"catalog_kda","suggested_tier":"heroic","signal_kind":"combat_pattern_active"}
		]`,
		ReasonParams: `{"title_fr":"Combat","description_fr":"Arc test"}`,
		RadarAxis:    "combat",
		Status:       coach_advisor.ProposalPending,
		Origin:       coach_advisor.OriginCatalog,
		SourceSignal: coach_advisor.SignalLOWESSPositive,
	}
	_ = repo.Create(ctx, arcProp)

	svc := coach_advisor.NewService(deps)
	res, err := svc.AcceptProposal(ctx, "arc1")
	if err != nil {
		t.Fatalf("AcceptProposal: %v", err)
	}
	if res.ArcID == "" {
		t.Error("expected ArcID, got empty")
	}
	if len(res.ChallengeIDs) != 2 {
		t.Errorf("expected 2 ChallengeIDs, got %d", len(res.ChallengeIDs))
	}
	if len(pw.createdArcs) != 1 {
		t.Fatalf("expected 1 arc created, got %d", len(pw.createdArcs))
	}
	if pw.createdArcs[0].Source != "coach" {
		t.Errorf("arc Source: got %q, want coach", pw.createdArcs[0].Source)
	}
	if len(pw.createdChallenges) != 2 {
		t.Fatalf("expected 2 challenges created, got %d", len(pw.createdChallenges))
	}
	for i, ch := range pw.createdChallenges {
		if ch.ArcID != res.ArcID {
			t.Errorf("challenge %d ArcID: got %q, want %q", i, ch.ArcID, res.ArcID)
		}
		if ch.Position != i+1 {
			t.Errorf("challenge %d Position: got %d, want %d", i, ch.Position, i+1)
		}
	}
}

func TestAcceptProposal_RequiresPrestigeDeps(t *testing.T) {
	repo := newFakeRepo()
	ctx := context.Background()
	p := mkPending("p1", "u1", "combat", "accuracy")
	_ = repo.Create(ctx, p)

	// Build service WITHOUT Prestige + Templates
	svc := coach_advisor.NewService(coach_advisor.ServiceDeps{Repo: repo})
	_, err := svc.AcceptProposal(ctx, "p1")
	if err == nil {
		t.Error("expected error when Prestige/Templates deps missing")
	}
}
