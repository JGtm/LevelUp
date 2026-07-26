//go:build integration

// Tests d'intégration end-to-end coach_advisor + Prestige (ADR 0020 Phase 1-9).
//
// Couvre :
//   - GenerateProposals avec un vrai catalog template (match catalogue)
//   - GenerateProposals avec synthèse (template synthétisé + persisté)
//   - AcceptProposal qui invoque prestige.CreateChallenge avec Source="coach"
//   - Vérification que le challenge créé apparaît bien dans la table challenge
//
// Utilise un stub BaselineProvider pour éviter de seeder shared_matches_v2.

package prestige

import (
	"context"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
	"levelup/go-api/internal/progression/coach_advisor"
)

// stubBaselineProvider retourne une baseline constante — suffit pour valider
// le câblage. Les tests Prestige internes ont déjà la couverture du calcul.
type stubBaselineProvider struct {
	matches []prestige.MatchData
}

func (p *stubBaselineProvider) RecentMatches(_ context.Context, _, _ string, window int) ([]prestige.MatchData, error) {
	if window > 0 && len(p.matches) > window {
		return p.matches[:window], nil
	}
	return p.matches, nil
}

func (p *stubBaselineProvider) PopulationPercentile(_ context.Context, _, _ string, _ float64) (float64, int, error) {
	return 50.0, 1000, nil
}

// e2eEnv regroupe tous les artefacts construits pour un test E2E.
type e2eEnv struct {
	playerDB     *duckdb.DB
	metadataDB   *duckdb.DB
	sharedDB     *duckdb.DB
	templateRepo *PrestigeTemplateRepo
	prestigeSvc  prestige.Service
	advisorSvc   coach_advisor.Service
	proposalRepo *duckdb.CoachProposalRepo
}

// newE2EEnv construit le full stack : 3 DBs migrées, prestige.Service réel,
// coach_advisor.Service réel branchés ensemble. Stub BaselineProvider avec
// quelques matchs accuracy ~0.42 pour permettre une création de challenge.
func newE2EEnv(t *testing.T) *e2eEnv {
	t.Helper()
	playerDB := setupPrestigeDB(t, migration.TargetPlayer)
	metadataDB := setupPrestigeDB(t, migration.TargetMetadata)
	sharedDB := setupPrestigeDB(t, migration.TargetSharedSocial)

	templateRepo := NewPrestigeTemplateRepo(metadataDB)
	socialRepo := NewPrestigeSocialRepo(sharedDB)
	squadRepo := NewPrestigeSquadRepo(sharedDB)
	squadChallRepo := NewPrestigeSquadChallengeRepo(sharedDB)
	presetArcRepo := NewPrestigePresetArcRepo(metadataDB)

	baselineMatches := make([]prestige.MatchData, 20)
	for i := range baselineMatches {
		baselineMatches[i] = prestige.MatchData{
			MatchID:     "m_" + string(rune('a'+i)),
			MetricValue: 0.42, // accuracy baseline 42%
			StartedAt:   time.Now().Add(-time.Duration(i) * 24 * time.Hour).UTC(),
		}
	}

	prestigeSvc := prestige.NewService(prestige.Deps{
		Tuning:           prestige.DefaultTuning(),
		Challenges:       NewPrestigeChallengeRepo(playerDB),
		Arcs:             NewPrestigeArcRepo(playerDB),
		Moments:          NewPrestigeMomentCardRepo(playerDB),
		Prestige:         socialRepo,
		Telemetry:        NewPrestigeTelemetryRepo(playerDB),
		BaselineState:    NewPrestigeBaselineStateRepo(playerDB),
		Templates:        templateRepo,
		PresetArcs:       presetArcRepo,
		SquadChallenges:  squadChallRepo,
		Squads:           squadRepo,
		BaselineProvider: &stubBaselineProvider{matches: baselineMatches},
	})

	proposalRepo := duckdb.NewCoachProposalRepo(playerDB)
	// Grammar inline minimale pour les tests
	grammar := coach_advisor.DefaultSynthesisGrammar()
	synth := coach_advisor.NewSynthesizer(grammar, coach_advisor.DefaultSynthesisConfig())

	advisorSvc := coach_advisor.NewService(coach_advisor.ServiceDeps{
		Repo:        proposalRepo,
		Templates:   templateRepo,
		Synthesizer: synth,
		Prestige:    prestigeSvc,
		Now:         func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
	})

	return &e2eEnv{
		playerDB:     playerDB,
		metadataDB:   metadataDB,
		sharedDB:     sharedDB,
		templateRepo: templateRepo,
		prestigeSvc:  prestigeSvc,
		advisorSvc:   advisorSvc,
		proposalRepo: proposalRepo,
	}
}

// seedCatalogTemplateAccuracy insère un template catalogue accuracy/combat.
func (e *e2eEnv) seedCatalogTemplateAccuracy(t *testing.T) {
	t.Helper()
	tpl := prestige.Template{
		ID:              "halo_test.accuracy",
		TitleSlug:       "halo_infinite",
		Metric:          "accuracy",
		WindowType:      prestige.WindowType("rolling_days"),
		WindowValue:     "14",
		Cadence:         prestige.Cadence("weekly"),
		EvalType:        prestige.EvalType("threshold"),
		ModeFilter:      "universal",
		LabelEN:         "Improve accuracy",
		LabelFR:         "Améliore la précision",
		DescriptionEN:   "Push accuracy over 14 days",
		DescriptionFR:   "Pousse la précision sur 14 jours",
		NormalTarget:    0.45,
		HeroicTarget:    0.50,
		LegendaryTarget: 0.55,
		MythicTarget:    0.60,
		LUSRComponents:  []string{"accuracy_delta"},
		RadarAxes:       []string{"combat"},
		Source:          "catalog",
		SchemaVersion:   1,
		UpdatedAt:       time.Now().UTC(),
	}
	if err := e.templateRepo.Replace(context.Background(), "halo_infinite", []prestige.Template{tpl}); err != nil {
		t.Fatalf("seed template: %v", err)
	}
}

// strongAccuracySignal retourne un Signal LOWESS positif accuracy/combat fort.
func strongAccuracySignal() coach_advisor.Signal {
	return coach_advisor.Signal{
		Kind:          coach_advisor.SignalLOWESSPositive,
		Metric:        "accuracy",
		LUSRComponent: "accuracy_delta",
		RadarAxis:     "combat",
		Strength:      0.85,
	}
}

// ─── Tests ───

func TestCoachAdvisor_E2E_CatalogMatch_ChallengePersistedInPrestige(t *testing.T) {
	env := newE2EEnv(t)
	env.seedCatalogTemplateAccuracy(t)
	ctx := context.Background()

	// 1. Générer une proposal sur signal accuracy fort → matche le catalog
	props, err := env.advisorSvc.GenerateProposals(ctx, coach_advisor.GenerateInput{
		UserID:           "test_user",
		TitleSlug:        "halo_infinite",
		ProactiveEnabled: true,
		Signals:          []coach_advisor.Signal{strongAccuracySignal()},
	})
	if err != nil {
		t.Fatalf("GenerateProposals: %v", err)
	}
	if len(props) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(props))
	}
	p := props[0]
	if p.Kind != coach_advisor.ProposalKindChallenge {
		t.Errorf("Kind: got %q, want challenge", p.Kind)
	}
	if p.Origin != coach_advisor.OriginCatalog {
		t.Errorf("Origin: got %q, want catalog", p.Origin)
	}
	if p.TemplateID != "halo_test.accuracy" {
		t.Errorf("TemplateID: got %q, want halo_test.accuracy", p.TemplateID)
	}

	// 2. Vérif proposal persistée
	stored, err := env.proposalRepo.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("proposal Get: %v", err)
	}
	if stored.Status != coach_advisor.ProposalPending {
		t.Errorf("stored Status: got %q, want pending", stored.Status)
	}

	// 3. Accept → doit créer un challenge dans la DB Prestige
	res, err := env.advisorSvc.AcceptProposal(ctx, p.ID)
	if err != nil {
		t.Fatalf("AcceptProposal: %v", err)
	}
	if res.ChallengeID == "" {
		t.Fatal("expected ChallengeID after accept")
	}

	// 4. Vérif que le challenge existe vraiment dans la table challenge
	challengeRepo := NewPrestigeChallengeRepo(env.playerDB)
	ch, err := challengeRepo.Get(ctx, res.ChallengeID)
	if err != nil {
		t.Fatalf("challenge Get: %v", err)
	}
	if ch.UserID != "test_user" || ch.TitleSlug != "halo_infinite" {
		t.Errorf("challenge identity mismatch: %+v", ch)
	}
	if ch.TemplateID != "halo_test.accuracy" {
		t.Errorf("challenge.TemplateID: got %q", ch.TemplateID)
	}
	if ch.Metric != "accuracy" {
		t.Errorf("challenge.Metric: got %q", ch.Metric)
	}
	if ch.Status != prestige.StatusActive {
		t.Errorf("challenge.Status: got %q, want active", ch.Status)
	}

	// 5. Vérif proposal marquée accepted
	finalProp, _ := env.proposalRepo.Get(ctx, p.ID)
	if finalProp.Status != coach_advisor.ProposalAccepted {
		t.Errorf("proposal final Status: got %q, want accepted", finalProp.Status)
	}
	if finalProp.ResolvedRef != res.ChallengeID {
		t.Errorf("proposal ResolvedRef: got %q, want %q", finalProp.ResolvedRef, res.ChallengeID)
	}
}

func TestCoachAdvisor_E2E_DisabledShortCircuits_NoProposal(t *testing.T) {
	env := newE2EEnv(t)
	env.seedCatalogTemplateAccuracy(t)
	ctx := context.Background()

	props, err := env.advisorSvc.GenerateProposals(ctx, coach_advisor.GenerateInput{
		UserID:           "test_user",
		TitleSlug:        "halo_infinite",
		ProactiveEnabled: false, // ← short-circuit
		Signals:          []coach_advisor.Signal{strongAccuracySignal()},
	})
	if err != nil {
		t.Fatalf("GenerateProposals: %v", err)
	}
	if len(props) != 0 {
		t.Errorf("expected no proposals when disabled, got %d", len(props))
	}

	// Vérif rien en DB
	all, _ := env.proposalRepo.ListByUser(ctx, "test_user", "halo_infinite", "")
	if len(all) != 0 {
		t.Errorf("no proposal should be persisted, got %d", len(all))
	}
}

func TestCoachAdvisor_E2E_NoCatalogMatch_NoSynthAvailable_Skipped(t *testing.T) {
	env := newE2EEnv(t)
	// pas de catalog seedé + grammaire vide → pas de synthèse
	ctx := context.Background()

	props, err := env.advisorSvc.GenerateProposals(ctx, coach_advisor.GenerateInput{
		UserID:           "test_user",
		TitleSlug:        "halo_infinite",
		ProactiveEnabled: true,
		Signals:          []coach_advisor.Signal{strongAccuracySignal()},
	})
	if err != nil {
		t.Fatalf("GenerateProposals: %v", err)
	}
	if len(props) != 0 {
		t.Errorf("expected no proposals (no catalog, empty grammar), got %d", len(props))
	}
}

func TestCoachAdvisor_E2E_AcceptedProposal_CannotBeAcceptedAgain(t *testing.T) {
	env := newE2EEnv(t)
	env.seedCatalogTemplateAccuracy(t)
	ctx := context.Background()

	props, _ := env.advisorSvc.GenerateProposals(ctx, coach_advisor.GenerateInput{
		UserID:           "test_user",
		TitleSlug:        "halo_infinite",
		ProactiveEnabled: true,
		Signals:          []coach_advisor.Signal{strongAccuracySignal()},
	})
	if len(props) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(props))
	}
	id := props[0].ID
	if _, err := env.advisorSvc.AcceptProposal(ctx, id); err != nil {
		t.Fatalf("first AcceptProposal: %v", err)
	}
	_, err := env.advisorSvc.AcceptProposal(ctx, id)
	if err == nil {
		t.Error("second AcceptProposal should fail (already accepted)")
	}
}

func TestCoachAdvisor_E2E_Dismiss_TransitionsAndStaysIdempotent(t *testing.T) {
	env := newE2EEnv(t)
	env.seedCatalogTemplateAccuracy(t)
	ctx := context.Background()

	props, _ := env.advisorSvc.GenerateProposals(ctx, coach_advisor.GenerateInput{
		UserID:           "test_user",
		TitleSlug:        "halo_infinite",
		ProactiveEnabled: true,
		Signals:          []coach_advisor.Signal{strongAccuracySignal()},
	})
	id := props[0].ID

	if err := env.advisorSvc.DismissProposal(ctx, id); err != nil {
		t.Fatalf("DismissProposal: %v", err)
	}
	stored, _ := env.proposalRepo.Get(ctx, id)
	if stored.Status != coach_advisor.ProposalDismissed {
		t.Errorf("Status after dismiss: got %q, want dismissed", stored.Status)
	}

	// 2e appel : idempotent (pas d'erreur, status reste dismissed)
	if err := env.advisorSvc.DismissProposal(ctx, id); err != nil {
		t.Fatalf("DismissProposal idempotent: %v", err)
	}
	stored2, _ := env.proposalRepo.Get(ctx, id)
	if stored2.Status != coach_advisor.ProposalDismissed {
		t.Errorf("Status after 2nd dismiss: got %q", stored2.Status)
	}
}
