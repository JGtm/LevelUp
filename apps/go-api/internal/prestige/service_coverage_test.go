package prestige

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// service_coverage_test.go — tests ciblés pour ramener la couverture
// du package au-dessus du seuil 80 % imposé par IMPL_PRESTIGE.md.
//
// Couvre :
//   - service.go : AbandonChallenge / GetChallenge / ListActiveChallenges /
//                  GetUserPrestige / SuggestNext / recomputeTier (via UpdateChallenge target)
//   - service_arcs_squads.go : ListArcs / GetArc / GetSquadChallenge / ListSquadChallenges
//   - service_evaluate.go : EvaluateForUser / evaluateOne / applyTransition / creditCompletion
//   - catalog_loader.go : LoadTemplatesFromTOML / LoadPresetArcsFromTOML / validateTemplateEntry
//   - enums.go : méthodes String() restantes
//   - lifecycle.go : CanAbandon

// ─── Fakes étendus (avec Get qui peut retourner un challenge réel) ───

type stubChallengeRepo struct {
	stored       map[string]Challenge
	listResult   []Challenge
	listErr      error
	updateStatus []ChallengeStatus
	updateLabel  []string
	updateTarget []float64
	createCalled bool
	activeByCad  map[Cadence]int
	activeTotal  int
	getErr       error
}

func newStubChallengeRepo() *stubChallengeRepo {
	return &stubChallengeRepo{stored: map[string]Challenge{}, activeByCad: map[Cadence]int{}}
}

func (r *stubChallengeRepo) Create(_ context.Context, c Challenge) error {
	r.createCalled = true
	r.stored[c.ID] = c
	return nil
}
func (r *stubChallengeRepo) Get(_ context.Context, id string) (Challenge, error) {
	if r.getErr != nil {
		return Challenge{}, r.getErr
	}
	c, ok := r.stored[id]
	if !ok {
		return Challenge{}, ErrChallengeNotFound
	}
	return c, nil
}
func (r *stubChallengeRepo) List(_ context.Context, f ChallengeFilter) ([]Challenge, error) {
	// Filtre par arc : lit depuis `stored` pour refléter les UpdateStatus
	// (utilisé par enrichArcReward / maybeCompleteArc). Le chemin sans ArcID
	// conserve le comportement historique (retourne listResult tel quel).
	if f.ArcID != nil {
		if r.listErr != nil {
			return nil, r.listErr
		}
		var out []Challenge
		for _, c := range r.stored {
			if c.ArcID == *f.ArcID {
				out = append(out, c)
			}
		}
		return out, nil
	}
	return r.listResult, r.listErr
}
func (r *stubChallengeRepo) UpdateStatus(_ context.Context, id string, s ChallengeStatus, _ time.Time) error {
	r.updateStatus = append(r.updateStatus, s)
	c := r.stored[id]
	c.Status = s
	r.stored[id] = c
	return nil
}
func (r *stubChallengeRepo) UpdateLabel(_ context.Context, _, l string) error {
	r.updateLabel = append(r.updateLabel, l)
	return nil
}
func (r *stubChallengeRepo) UpdateTarget(_ context.Context, _ string, target float64, _ Tier, _ DataTier, _ time.Time) error {
	r.updateTarget = append(r.updateTarget, target)
	return nil
}
func (r *stubChallengeRepo) CountActiveByCadence(_ context.Context, _, _ string, c Cadence) (int, error) {
	return r.activeByCad[c], nil
}
func (r *stubChallengeRepo) CountActiveTotal(_ context.Context, _, _ string) (int, error) {
	return r.activeTotal, nil
}
func (r *stubChallengeRepo) CountCreatedSince(_ context.Context, _, _ string, _ ChallengeMode, _ time.Time) (int, error) {
	return 0, nil
}
func (r *stubChallengeRepo) DetachFromArc(_ context.Context, arcID string) error {
	for id, c := range r.stored {
		if c.ArcID == arcID {
			c.ArcID = ""
			r.stored[id] = c
		}
	}
	return nil
}
func (r *stubChallengeRepo) DeleteByArc(_ context.Context, arcID string) error {
	for id, c := range r.stored {
		if c.ArcID == arcID {
			delete(r.stored, id)
		}
	}
	return nil
}

type stubArcRepo struct {
	stored map[string]Arc
	list   []Arc
	getErr error
}

func newStubArcRepo() *stubArcRepo { return &stubArcRepo{stored: map[string]Arc{}} }

func (r *stubArcRepo) Create(_ context.Context, a Arc) error { r.stored[a.ID] = a; return nil }
func (r *stubArcRepo) Get(_ context.Context, id string) (Arc, error) {
	if r.getErr != nil {
		return Arc{}, r.getErr
	}
	a, ok := r.stored[id]
	if !ok {
		return Arc{}, ErrArcNotFound
	}
	return a, nil
}
func (r *stubArcRepo) ListByUser(_ context.Context, _, _ string) ([]Arc, error) {
	return r.list, nil
}
func (r *stubArcRepo) MarkCompleted(_ context.Context, _ string, _ time.Time) error { return nil }
func (r *stubArcRepo) Delete(_ context.Context, id string) error {
	delete(r.stored, id)
	return nil
}

type stubSquadChallengeRepo struct {
	stored      map[string]SquadChallenge
	listBySquad []SquadChallenge
	getErr      error
}

func newStubSquadChallengeRepo() *stubSquadChallengeRepo {
	return &stubSquadChallengeRepo{stored: map[string]SquadChallenge{}}
}

func (r *stubSquadChallengeRepo) Create(_ context.Context, sc SquadChallenge) error {
	r.stored[sc.ID] = sc
	return nil
}
func (r *stubSquadChallengeRepo) Get(_ context.Context, id string) (SquadChallenge, error) {
	if r.getErr != nil {
		return SquadChallenge{}, r.getErr
	}
	sc, ok := r.stored[id]
	if !ok {
		return SquadChallenge{}, errors.New("not found")
	}
	return sc, nil
}
func (r *stubSquadChallengeRepo) ListBySquad(_ context.Context, _ string) ([]SquadChallenge, error) {
	return r.listBySquad, nil
}
func (r *stubSquadChallengeRepo) AddParticipant(_ context.Context, _ SquadChallengeParticipant) error {
	return nil
}
func (r *stubSquadChallengeRepo) UpdateParticipantProgress(_ context.Context, _, _ string, _ float64, _ *time.Time) error {
	return nil
}
func (r *stubSquadChallengeRepo) ListParticipants(_ context.Context, _ string) ([]SquadChallengeParticipant, error) {
	return nil, nil
}
func (r *stubSquadChallengeRepo) CountActiveParticipants(_ context.Context, _ string) (int, error) {
	return 0, nil
}

type stubPrestigeRepo struct {
	emitted     []PrestigeEvent
	upsert      []UserPrestige
	user        UserPrestige
	userCross   UserPrestige
	emitErr     error
	getErr      error
	getCrossErr error
}

func (r *stubPrestigeRepo) EmitEvent(_ context.Context, ev PrestigeEvent) error {
	if r.emitErr != nil {
		return r.emitErr
	}
	r.emitted = append(r.emitted, ev)
	return nil
}
func (r *stubPrestigeRepo) GetUserPrestige(_ context.Context, _, _ string) (UserPrestige, error) {
	return r.user, r.getErr
}
func (r *stubPrestigeRepo) GetUserPrestigeCrossTitle(_ context.Context, _ string) (UserPrestige, error) {
	return r.userCross, r.getCrossErr
}
func (r *stubPrestigeRepo) UpsertUserPrestige(_ context.Context, up UserPrestige) error {
	r.upsert = append(r.upsert, up)
	return nil
}
func (r *stubPrestigeRepo) ListEvents(_ context.Context, _, _ string, _ time.Time) ([]PrestigeEvent, error) {
	return nil, nil
}
func (r *stubPrestigeRepo) GetLeaderboard(_ context.Context, _ []string, _ *string, _ time.Time) ([]LeaderboardEntry, error) {
	return nil, nil
}

// matchesProvider permet de varier les matchs selon le test.
type matchesProvider struct {
	matches []MatchData
	err     error
	pop     float64
	popSize int
}

func (p *matchesProvider) RecentMatches(_ context.Context, _, _, _ string, _ int) ([]MatchData, error) {
	return p.matches, p.err
}
func (p *matchesProvider) PopulationPercentile(_ context.Context, _, _ string, _ float64) (float64, int, error) {
	return p.pop, p.popSize, nil
}

func buildCoverageService() (*service, *stubChallengeRepo, *stubArcRepo, *stubSquadChallengeRepo, *stubPrestigeRepo, *matchesProvider) {
	chRepo := newStubChallengeRepo()
	arcRepo := newStubArcRepo()
	scRepo := newStubSquadChallengeRepo()
	prRepo := &stubPrestigeRepo{}
	bp := &matchesProvider{matches: tenMatches(), pop: 0.95, popSize: 100}
	deps := Deps{
		Tuning:           DefaultTuning(),
		Challenges:       chRepo,
		Arcs:             arcRepo,
		SquadChallenges:  scRepo,
		Squads:           &fakeSquadRepo{},
		Templates:        &fakeTemplateRepo{},
		Telemetry:        &fakeNoOpTelemetryRepo{},
		Prestige:         prRepo,
		BaselineProvider: bp,
		Now:              func() time.Time { return time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC) },
	}
	return NewService(deps).(*service), chRepo, arcRepo, scRepo, prRepo, bp
}

func tenMatches() []MatchData {
	out := make([]MatchData, 10)
	now := time.Now()
	for i := range out {
		out[i] = MatchData{MatchID: "m", MetricValue: 1.0, StartedAt: now}
	}
	return out
}

// ─── service.go ───

func TestService_AbandonChallenge_OK(t *testing.T) {
	svc, chRepo, _, _, _, _ := buildCoverageService()
	chRepo.stored["ch1"] = Challenge{
		ID: "ch1", UserID: "u1", Status: StatusActive, Mode: ModeLibre,
	}
	if err := svc.AbandonChallenge(context.Background(), "ch1"); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := chRepo.stored["ch1"].Status; got != StatusAbandoned {
		t.Errorf("expected abandoned, got %s", got)
	}
}

func TestService_AbandonChallenge_NotFound(t *testing.T) {
	svc, _, _, _, _, _ := buildCoverageService()
	err := svc.AbandonChallenge(context.Background(), "missing")
	if !errors.Is(err, ErrChallengeNotFound) {
		t.Errorf("expected ErrChallengeNotFound, got %v", err)
	}
}

// ─── DeleteArc (Lot A) ───

func TestService_DeleteArc_Detach(t *testing.T) {
	svc, chRepo, arcRepo, _, _, _ := buildCoverageService()
	arcRepo.stored["arc1"] = Arc{ID: "arc1", UserID: "u1", TitleSlug: "halo_infinite",
		CreatedAt: time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)}
	chRepo.stored["ch1"] = Challenge{ID: "ch1", UserID: "u1", ArcID: "arc1", Status: StatusActive, Mode: ModeLibre}

	if err := svc.DeleteArc(context.Background(), "u1", "arc1", DeleteArcOptions{CascadeObjectives: false}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if _, ok := arcRepo.stored["arc1"]; ok {
		t.Error("arc should be deleted")
	}
	if got := chRepo.stored["ch1"].ArcID; got != "" {
		t.Errorf("objective should be detached (arc_id vide), got %q", got)
	}
	if chRepo.stored["ch1"].Status != StatusActive {
		t.Error("detached objective should stay active")
	}
}

// Arc créé < 1h → exemption → objectifs supprimés physiquement (zéro cooldown).
func TestService_DeleteArc_CascadeExempt_HardDeletes(t *testing.T) {
	svc, chRepo, arcRepo, _, _, _ := buildCoverageService()
	arcRepo.stored["arc1"] = Arc{ID: "arc1", UserID: "u1",
		CreatedAt: time.Date(2026, 4, 25, 11, 30, 0, 0, time.UTC)} // now - 30min
	chRepo.stored["ch1"] = Challenge{ID: "ch1", UserID: "u1", ArcID: "arc1", Status: StatusActive, Mode: ModeLibre}

	if err := svc.DeleteArc(context.Background(), "u1", "arc1", DeleteArcOptions{CascadeObjectives: true}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if _, ok := chRepo.stored["ch1"]; ok {
		t.Error("objective should be hard-deleted under exemption")
	}
	if _, ok := arcRepo.stored["arc1"]; ok {
		t.Error("arc should be deleted")
	}
}

// Arc créé ≥ 1h → pas d'exemption → objectifs actifs abandonnés (cooldown).
func TestService_DeleteArc_CascadeAbandons(t *testing.T) {
	svc, chRepo, arcRepo, _, _, _ := buildCoverageService()
	arcRepo.stored["arc1"] = Arc{ID: "arc1", UserID: "u1",
		CreatedAt: time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)} // now - 2h
	chRepo.stored["ch1"] = Challenge{ID: "ch1", UserID: "u1", ArcID: "arc1", Status: StatusActive, Mode: ModeLibre}

	if err := svc.DeleteArc(context.Background(), "u1", "arc1", DeleteArcOptions{CascadeObjectives: true}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := chRepo.stored["ch1"].Status; got != StatusAbandoned {
		t.Errorf("objective should be abandoned, got %s", got)
	}
	if _, ok := arcRepo.stored["arc1"]; ok {
		t.Error("arc should be deleted")
	}
}

func TestService_DeleteArc_Forbidden(t *testing.T) {
	svc, _, arcRepo, _, _, _ := buildCoverageService()
	arcRepo.stored["arc1"] = Arc{ID: "arc1", UserID: "owner",
		CreatedAt: time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)}
	err := svc.DeleteArc(context.Background(), "intruder", "arc1", DeleteArcOptions{CascadeObjectives: true})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
	if _, ok := arcRepo.stored["arc1"]; !ok {
		t.Error("arc must NOT be deleted on forbidden")
	}
}

func TestService_DeleteArc_NotFound(t *testing.T) {
	svc, _, _, _, _, _ := buildCoverageService()
	err := svc.DeleteArc(context.Background(), "u1", "missing", DeleteArcOptions{})
	if !errors.Is(err, ErrArcNotFound) {
		t.Errorf("expected ErrArcNotFound, got %v", err)
	}
}

func TestService_AbandonChallenge_Terminal(t *testing.T) {
	svc, chRepo, _, _, _, _ := buildCoverageService()
	chRepo.stored["ch1"] = Challenge{ID: "ch1", Status: StatusCompleted}
	err := svc.AbandonChallenge(context.Background(), "ch1")
	if !errors.Is(err, ErrAlreadyTerminal) {
		t.Errorf("expected ErrAlreadyTerminal, got %v", err)
	}
}

func TestService_GetChallenge_OK(t *testing.T) {
	svc, chRepo, _, _, _, _ := buildCoverageService()
	chRepo.stored["ch1"] = Challenge{ID: "ch1"}
	got, err := svc.GetChallenge(context.Background(), "ch1")
	if err != nil || got.ID != "ch1" {
		t.Fatalf("unexpected: %v %+v", err, got)
	}
}

func TestService_GetChallenge_NotFound(t *testing.T) {
	svc, _, _, _, _, _ := buildCoverageService()
	_, err := svc.GetChallenge(context.Background(), "missing")
	if !errors.Is(err, ErrChallengeNotFound) {
		t.Errorf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestService_ListActiveChallenges_DelegatesFilter(t *testing.T) {
	svc, chRepo, _, _, _, _ := buildCoverageService()
	chRepo.listResult = []Challenge{{ID: "a"}, {ID: "b"}}
	out, err := svc.ListActiveChallenges(context.Background(), "u1", "halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Errorf("expected 2, got %d", len(out))
	}
}

func TestService_ListChallenges_TerminalNotEnriched(t *testing.T) {
	svc, chRepo, _, _, _, _ := buildCoverageService()
	// Un défi terminal (completed) avec un palier : l'enrichissement PP/valeur
	// courante ne doit PAS s'appliquer (sa fenêtre est passée).
	chRepo.listResult = []Challenge{{ID: "done", Status: StatusCompleted, Tier: TierHeroic, DataTier: DataFull}}
	out, err := svc.ListChallenges(context.Background(), "u1", "halo_infinite", []ChallengeStatus{StatusCompleted})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1, got %d", len(out))
	}
	if out[0].PPReward != 0 {
		t.Errorf("défi terminal enrichi à tort : PPReward=%d, want 0", out[0].PPReward)
	}
	if out[0].CurrentValue != 0 {
		t.Errorf("défi terminal enrichi à tort : CurrentValue=%v, want 0", out[0].CurrentValue)
	}
}

func TestService_GetUserPrestige_TitleSpecific(t *testing.T) {
	svc, _, _, _, prRepo, _ := buildCoverageService()
	prRepo.user = UserPrestige{UserID: "u1", TitleSlug: "halo_infinite", TotalPP: 500}
	got, err := svc.GetUserPrestige(context.Background(), "u1", "halo_infinite")
	if err != nil || got.TotalPP != 500 {
		t.Errorf("unexpected: %v %+v", err, got)
	}
}

func TestService_GetUserPrestige_CrossTitle(t *testing.T) {
	svc, _, _, _, prRepo, _ := buildCoverageService()
	prRepo.userCross = UserPrestige{UserID: "u1", TotalPP: 1500}
	got, err := svc.GetUserPrestige(context.Background(), "u1", "")
	if err != nil || got.TotalPP != 1500 {
		t.Errorf("unexpected: %v %+v", err, got)
	}
}

func TestService_SuggestNext_OK(t *testing.T) {
	svc, chRepo, _, _, _, _ := buildCoverageService()
	chRepo.stored["ch1"] = Challenge{ID: "ch1", TitleSlug: "halo_infinite", TemplateID: "t1"}
	tplRepo := svc.deps.Templates.(*fakeTemplateRepo)
	tplRepo.templates = []Template{{ID: "t2"}, {ID: "t3"}, {ID: "t4"}}
	out, err := svc.SuggestNext(context.Background(), "ch1")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Errorf("expected 3, got %d", len(out))
	}
}

func TestService_SuggestNext_NotFound(t *testing.T) {
	svc, _, _, _, _, _ := buildCoverageService()
	_, err := svc.SuggestNext(context.Background(), "missing")
	if !errors.Is(err, ErrChallengeNotFound) {
		t.Errorf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestService_UpdateChallenge_TargetRecomputesTier(t *testing.T) {
	svc, chRepo, _, _, _, _ := buildCoverageService()
	chRepo.stored["ch1"] = Challenge{
		ID: "ch1", UserID: "u1", TitleSlug: "halo_infinite",
		Metric: "FieldKDA", Target: 1.5,
		Mode: ModeLibre, Status: StatusActive, Tier: TierHeroic,
	}
	newTarget := 2.0
	updated, err := svc.UpdateChallenge(context.Background(), "ch1", UpdateChallengePatch{
		Target: &newTarget,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if updated.Target != 2.0 {
		t.Errorf("target not updated: %v", updated.Target)
	}
	if len(chRepo.updateTarget) != 1 || chRepo.updateTarget[0] != 2.0 {
		t.Errorf("updateTarget not invoked: %v", chRepo.updateTarget)
	}
}

func TestService_UpdateChallenge_LabelPatchInvokesRepo(t *testing.T) {
	svc, chRepo, _, _, _, _ := buildCoverageService()
	chRepo.stored["ch1"] = Challenge{ID: "ch1", Mode: ModeLibre, Status: StatusActive}
	newLabel := "Nouveau libellé"
	_, err := svc.UpdateChallenge(context.Background(), "ch1", UpdateChallengePatch{Label: &newLabel})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(chRepo.updateLabel) != 1 || chRepo.updateLabel[0] != newLabel {
		t.Errorf("updateLabel not invoked: %v", chRepo.updateLabel)
	}
}

func TestService_UpdateChallenge_TargetRefusedOnPilote(t *testing.T) {
	svc, chRepo, _, _, _, _ := buildCoverageService()
	chRepo.stored["ch1"] = Challenge{
		ID: "ch1", Mode: ModePilote, Status: StatusActive,
	}
	newTarget := 2.0
	_, err := svc.UpdateChallenge(context.Background(), "ch1", UpdateChallengePatch{Target: &newTarget})
	if !errors.Is(err, ErrNotEditable) {
		t.Errorf("expected ErrNotEditable, got %v", err)
	}
}

// ─── service_arcs_squads.go ───

func TestService_ListArcs_RequiresFields(t *testing.T) {
	svc, _, _, _, _, _ := buildCoverageService()
	_, err := svc.ListArcs(context.Background(), "", "halo_infinite")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_ListArcs_DelegatesToRepo(t *testing.T) {
	svc, _, arcRepo, _, _, _ := buildCoverageService()
	arcRepo.list = []Arc{{ID: "a1"}, {ID: "a2"}}
	out, err := svc.ListArcs(context.Background(), "u1", "halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Errorf("expected 2, got %d", len(out))
	}
}

func TestService_GetArc_OK(t *testing.T) {
	svc, _, arcRepo, _, _, _ := buildCoverageService()
	arcRepo.stored["a1"] = Arc{ID: "a1", Title: "Slayer Custom"}
	got, err := svc.GetArc(context.Background(), "a1")
	if err != nil || got.Title != "Slayer Custom" {
		t.Errorf("unexpected: %v %+v", err, got)
	}
}

func TestService_GetArc_NotFound(t *testing.T) {
	svc, _, _, _, _, _ := buildCoverageService()
	_, err := svc.GetArc(context.Background(), "missing")
	if !errors.Is(err, ErrArcNotFound) {
		t.Errorf("expected ErrArcNotFound, got %v", err)
	}
}

func TestService_GetArc_OtherErrorWrapped(t *testing.T) {
	svc, _, arcRepo, _, _, _ := buildCoverageService()
	arcRepo.getErr = errors.New("db down")
	_, err := svc.GetArc(context.Background(), "a1")
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, ErrArcNotFound) {
		t.Errorf("should wrap, not return ErrArcNotFound")
	}
}

func TestService_GetSquadChallenge_OK(t *testing.T) {
	svc, _, _, scRepo, _, _ := buildCoverageService()
	scRepo.stored["sc1"] = SquadChallenge{ID: "sc1", SquadID: "sq1"}
	got, err := svc.GetSquadChallenge(context.Background(), "sc1")
	if err != nil || got.SquadID != "sq1" {
		t.Errorf("unexpected: %v %+v", err, got)
	}
}

func TestService_GetSquadChallenge_NotFound(t *testing.T) {
	svc, _, _, _, _, _ := buildCoverageService()
	_, err := svc.GetSquadChallenge(context.Background(), "missing")
	if err == nil {
		t.Error("expected error")
	}
}

func TestService_ListSquadChallenges_RequiresSquad(t *testing.T) {
	svc, _, _, _, _, _ := buildCoverageService()
	_, err := svc.ListSquadChallenges(context.Background(), "", "alice")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_ListSquadChallenges_OK(t *testing.T) {
	svc, _, _, scRepo, _, _ := buildCoverageService()
	svc.deps.Squads.(*fakeSquadRepo).members = []SquadMember{{Xuid: "x1", UserID: "alice"}}
	scRepo.listBySquad = []SquadChallenge{{ID: "sc1"}, {ID: "sc2"}}
	out, err := svc.ListSquadChallenges(context.Background(), "sq1", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Errorf("expected 2, got %d", len(out))
	}
}

// TestService_ListSquadChallenges_RejectsNonMember : garde d'appartenance
// objet-level (BOLA). « outsider » possède son propre slug (ownershipMW le
// laisserait passer) mais n'est PAS membre-user de sq1 → aucun défi renvoyé.
func TestService_ListSquadChallenges_RejectsNonMember(t *testing.T) {
	svc, _, _, scRepo, _, _ := buildCoverageService()
	svc.deps.Squads.(*fakeSquadRepo).members = []SquadMember{{Xuid: "x1", UserID: "alice"}}
	scRepo.listBySquad = []SquadChallenge{{ID: "sc1"}}
	out, err := svc.ListSquadChallenges(context.Background(), "sq1", "outsider")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("non-membre doit être rejeté (BOLA objet-level), got %v", err)
	}
	if out != nil {
		t.Errorf("aucune donnée ne doit fuir vers un non-membre, got %+v", out)
	}
}

// ─── service_evaluate.go ───

func TestService_EvaluateForUser_NoActive(t *testing.T) {
	svc, _, _, _, _, _ := buildCoverageService()
	out, err := svc.EvaluateForUser(context.Background(), "u1", "halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Errorf("expected 0 outcomes, got %d", len(out))
	}
}

func TestService_EvaluateForUser_TargetReachedCreditsPP(t *testing.T) {
	svc, chRepo, _, _, prRepo, bp := buildCoverageService()
	// Matchs avec moyenne 2.0 → atteint la cible 1.5.
	bp.matches = make([]MatchData, 12)
	for i := range bp.matches {
		bp.matches[i] = MatchData{MetricValue: 2.0, StartedAt: time.Now()}
	}
	chRepo.listResult = []Challenge{{
		ID: "ch1", UserID: "u1", TitleSlug: "halo_infinite",
		Metric: "FieldKDA", Target: 1.5,
		Status: StatusActive, EvalType: EvalThreshold, WindowType: WindowSession,
		Tier: TierHeroic, DataTier: DataFull, Mode: ModeLibre,
	}}
	chRepo.stored["ch1"] = chRepo.listResult[0]

	outcomes, err := svc.EvaluateForUser(context.Background(), "u1", "halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 {
		t.Fatalf("expected 1 outcome, got %d", len(outcomes))
	}
	if outcomes[0].NewStatus != StatusCompleted {
		t.Errorf("expected completed, got %s", outcomes[0].NewStatus)
	}
	if outcomes[0].PPCredited <= 0 {
		t.Errorf("expected PP credited, got %d", outcomes[0].PPCredited)
	}
	if len(prRepo.emitted) != 1 {
		t.Errorf("expected 1 emitted event, got %d", len(prRepo.emitted))
	}
}

func TestService_EvaluateForUser_DeadlineExpired(t *testing.T) {
	svc, chRepo, _, _, _, bp := buildCoverageService()
	bp.matches = []MatchData{}
	chRepo.listResult = []Challenge{{
		ID: "ch1", UserID: "u1", TitleSlug: "halo_infinite",
		Metric: "FieldKDA", Target: 5.0,
		Status: StatusActive, EvalType: EvalThreshold,
		WindowType:  WindowDeadline,
		WindowValue: "2020-01-01",
		Tier:        TierMythic, DataTier: DataFull,
	}}
	chRepo.stored["ch1"] = chRepo.listResult[0]

	outcomes, err := svc.EvaluateForUser(context.Background(), "u1", "halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 {
		t.Fatalf("expected 1 outcome, got %d", len(outcomes))
	}
	if outcomes[0].NewStatus != StatusExpired {
		t.Errorf("expected expired, got %s", outcomes[0].NewStatus)
	}
}

func TestService_EvaluateForUser_FetchErrorReturnsUnchanged(t *testing.T) {
	svc, chRepo, _, _, _, bp := buildCoverageService()
	bp.err = errors.New("boom")
	chRepo.listResult = []Challenge{{
		ID: "ch1", Status: StatusActive, EvalType: EvalThreshold, Metric: "FieldKDA", Target: 2.0,
		WindowType: WindowSession,
	}}
	outcomes, err := svc.EvaluateForUser(context.Background(), "u1", "halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 {
		t.Fatalf("expected 1, got %d", len(outcomes))
	}
	if outcomes[0].NewStatus != StatusActive {
		t.Errorf("status should be unchanged, got %s", outcomes[0].NewStatus)
	}
}

// ─── catalog_loader.go ───

func writeTempTOML(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

type captureTemplateRepo struct {
	titleSlug string
	templates []Template
	err       error
}

func (r *captureTemplateRepo) ListByTitle(_ context.Context, _ string) ([]Template, error) {
	return nil, nil
}
func (r *captureTemplateRepo) GetByID(_ context.Context, _ string) (Template, error) {
	return Template{}, nil
}
func (r *captureTemplateRepo) Suggest(_ context.Context, _ string, _ []string, _ int) ([]Template, error) {
	return nil, nil
}
func (r *captureTemplateRepo) Replace(_ context.Context, ts string, tpl []Template) error {
	r.titleSlug = ts
	r.templates = tpl
	return r.err
}
func (r *captureTemplateRepo) UpsertOne(_ context.Context, t Template) error {
	r.titleSlug = t.TitleSlug
	r.templates = []Template{t}
	return r.err
}

func TestLoadTemplatesFromTOML_OK(t *testing.T) {
	body := `
[meta]
schema_version = 1
title_slug = "halo_infinite"

[[templates]]
id = "t_kda_session"
metric = "FieldKDA"
window_type = "session"
window_value = "1"
cadence = "daily"
eval_type = "threshold"
mode_filter = "pvp"
label_en = "KDA boost"
label_fr = "Booste ton KDA"
description_en = "Average KDA"
description_fr = "KDA moyen"
normal_target = 1.1
heroic_target = 1.35
legendary_target = 1.6
mythic_target = 2.0
`
	path := writeTempTOML(t, "templates.toml", body)
	repo := &captureTemplateRepo{}
	count, err := LoadTemplatesFromTOML(context.Background(), repo, path)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
	if repo.titleSlug != "halo_infinite" {
		t.Errorf("title_slug: %s", repo.titleSlug)
	}
	if repo.templates[0].ModeFilter != "pvp" {
		t.Errorf("mode_filter: %s", repo.templates[0].ModeFilter)
	}
}

// TestLoadTemplatesFromTOML_TaggingFields (V1 PlayerProfile §5.1) : vérifie
// que les nouveaux champs lusr_components, radar_axes, is_long_term sont
// correctement parsés depuis le TOML et propagés dans le Template.
func TestLoadTemplatesFromTOML_TaggingFields(t *testing.T) {
	body := `
[meta]
schema_version = 1
title_slug = "halo_infinite"

[[templates]]
id = "t_kda_rolling"
metric = "FieldKDA"
window_type = "rolling_days"
window_value = "7"
cadence = "weekly"
eval_type = "threshold"
mode_filter = "pvp"
label_en = "KDA consistency"
label_fr = "Régularité KDA"
description_en = "Maintain a strong KDA over 7 days"
description_fr = "Maintiens un bon KDA sur 7 jours"
normal_target = 1.1
heroic_target = 1.4
legendary_target = 1.7
mythic_target = 2.2
lusr_components = ["kills_vs_expected", "deaths_vs_expected"]
radar_axes = ["combat", "survival"]
is_long_term = true

[[templates]]
id = "t_session_no_tags"
metric = "FieldKills"
window_type = "session"
window_value = "1"
cadence = "daily"
eval_type = "threshold"
mode_filter = "universal"
label_en = "Quick kills"
label_fr = "Tueries rapides"
description_en = "Get kills in a single session"
description_fr = "Élimine en une session"
normal_target = 5.0
heroic_target = 10.0
legendary_target = 15.0
mythic_target = 20.0
`
	path := writeTempTOML(t, "templates.toml", body)
	repo := &captureTemplateRepo{}
	count, err := LoadTemplatesFromTOML(context.Background(), repo, path)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 templates, got %d", count)
	}

	// 1er template : tags présents.
	tag := repo.templates[0]
	if len(tag.LUSRComponents) != 2 || tag.LUSRComponents[0] != "kills_vs_expected" || tag.LUSRComponents[1] != "deaths_vs_expected" {
		t.Errorf("LUSRComponents = %v, want [kills_vs_expected deaths_vs_expected]", tag.LUSRComponents)
	}
	if len(tag.RadarAxes) != 2 || tag.RadarAxes[0] != "combat" || tag.RadarAxes[1] != "survival" {
		t.Errorf("RadarAxes = %v, want [combat survival]", tag.RadarAxes)
	}
	if !tag.IsLongTerm {
		t.Errorf("IsLongTerm = false, want true")
	}

	// 2e template : pas de tags → valeurs zero.
	noTag := repo.templates[1]
	if len(noTag.LUSRComponents) != 0 {
		t.Errorf("LUSRComponents = %v, want empty", noTag.LUSRComponents)
	}
	if len(noTag.RadarAxes) != 0 {
		t.Errorf("RadarAxes = %v, want empty", noTag.RadarAxes)
	}
	if noTag.IsLongTerm {
		t.Errorf("IsLongTerm = true, want false (default)")
	}
}

func TestLoadTemplatesFromTOML_ModeFilterDefaultsUniversal(t *testing.T) {
	body := `
[meta]
schema_version = 1
title_slug = "halo_infinite"

[[templates]]
id = "t1"
metric = "FieldKDA"
window_type = "session"
cadence = "daily"
eval_type = "threshold"
label_en = "X"
label_fr = "Y"
normal_target = 1.0
heroic_target = 1.2
legendary_target = 1.5
mythic_target = 1.8
`
	path := writeTempTOML(t, "templates.toml", body)
	repo := &captureTemplateRepo{}
	if _, err := LoadTemplatesFromTOML(context.Background(), repo, path); err != nil {
		t.Fatalf("err: %v", err)
	}
	if repo.templates[0].ModeFilter != "universal" {
		t.Errorf("default mode_filter: %s", repo.templates[0].ModeFilter)
	}
}

func TestLoadTemplatesFromTOML_MissingFile(t *testing.T) {
	repo := &captureTemplateRepo{}
	_, err := LoadTemplatesFromTOML(context.Background(), repo, "/no/such/file.toml")
	if err == nil {
		t.Error("expected error on missing file")
	}
}

func TestLoadTemplatesFromTOML_ParseError(t *testing.T) {
	path := writeTempTOML(t, "broken.toml", "not = [valid")
	repo := &captureTemplateRepo{}
	_, err := LoadTemplatesFromTOML(context.Background(), repo, path)
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestLoadTemplatesFromTOML_MissingTitleSlug(t *testing.T) {
	body := `
[meta]
schema_version = 1
`
	path := writeTempTOML(t, "no_slug.toml", body)
	repo := &captureTemplateRepo{}
	_, err := LoadTemplatesFromTOML(context.Background(), repo, path)
	if err == nil {
		t.Error("expected error")
	}
}

func TestLoadTemplatesFromTOML_InvalidEnumOrPaliers(t *testing.T) {
	cases := map[string]string{
		"bad_cadence": `
[meta]
schema_version=1
title_slug="halo_infinite"
[[templates]]
id="t"
metric="FieldKDA"
window_type="session"
cadence="hourly"
eval_type="threshold"
label_en="A"
label_fr="B"
normal_target=1.0
heroic_target=1.2
legendary_target=1.5
mythic_target=1.8
`,
		"bad_paliers": `
[meta]
schema_version=1
title_slug="halo_infinite"
[[templates]]
id="t"
metric="FieldKDA"
window_type="session"
cadence="daily"
eval_type="threshold"
label_en="A"
label_fr="B"
normal_target=2.0
heroic_target=1.5
legendary_target=1.0
mythic_target=0.5
`,
		"missing_label": `
[meta]
schema_version=1
title_slug="halo_infinite"
[[templates]]
id="t"
metric="FieldKDA"
window_type="session"
cadence="daily"
eval_type="threshold"
label_en=""
label_fr=""
normal_target=1.0
heroic_target=1.2
legendary_target=1.5
mythic_target=1.8
`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeTempTOML(t, name+".toml", body)
			repo := &captureTemplateRepo{}
			_, err := LoadTemplatesFromTOML(context.Background(), repo, path)
			if err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

type capturePresetArcRepo struct {
	titleSlug string
	arcs      []PresetArc
	steps     []PresetArcStep
}

func (r *capturePresetArcRepo) ListByTitle(_ context.Context, _ string) ([]PresetArc, error) {
	return nil, nil
}
func (r *capturePresetArcRepo) GetByID(_ context.Context, _ string) (PresetArc, error) {
	return PresetArc{}, nil
}
func (r *capturePresetArcRepo) GetSteps(_ context.Context, _ string) ([]PresetArcStep, error) {
	return nil, nil
}
func (r *capturePresetArcRepo) Replace(_ context.Context, ts string, arcs []PresetArc, steps []PresetArcStep) error {
	r.titleSlug = ts
	r.arcs = arcs
	r.steps = steps
	return nil
}

func TestLoadPresetArcsFromTOML_OK(t *testing.T) {
	body := `
[meta]
schema_version = 1
title_slug = "halo_infinite"

[[arcs]]
id = "arc_slayer"
title_en = "The Slayer"
title_fr = "Le Tueur"
description_en = "5 steps to climb the slayer ladder"
description_fr = "5 étapes pour monter en slayer"

  [[arcs.steps]]
  position = 1
  template_id = "t_kda_session"
  target_tier = "normal"

  [[arcs.steps]]
  position = 2
  template_id = "t_kda_week"
  target_tier = "heroic"
`
	path := writeTempTOML(t, "presets.toml", body)
	repo := &capturePresetArcRepo{}
	count, err := LoadPresetArcsFromTOML(context.Background(), repo, path)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
	if len(repo.steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(repo.steps))
	}
}

func TestLoadPresetArcsFromTOML_InvalidTier(t *testing.T) {
	body := `
[meta]
schema_version=1
title_slug="halo_infinite"

[[arcs]]
id="a"
title_en="A"
title_fr="A"

  [[arcs.steps]]
  position=1
  template_id="t1"
  target_tier="diamond"
`
	path := writeTempTOML(t, "presets_bad_tier.toml", body)
	repo := &capturePresetArcRepo{}
	_, err := LoadPresetArcsFromTOML(context.Background(), repo, path)
	if err == nil {
		t.Error("expected error")
	}
}

func TestLoadPresetArcsFromTOML_MissingFile(t *testing.T) {
	repo := &capturePresetArcRepo{}
	_, err := LoadPresetArcsFromTOML(context.Background(), repo, "/missing.toml")
	if err == nil {
		t.Error("expected error")
	}
}

func TestLoadPresetArcsFromTOML_MissingTitleSlug(t *testing.T) {
	path := writeTempTOML(t, "presets_noslug.toml", "[meta]\nschema_version=1\n")
	repo := &capturePresetArcRepo{}
	_, err := LoadPresetArcsFromTOML(context.Background(), repo, path)
	if err == nil {
		t.Error("expected error")
	}
}

func TestLoadPresetArcsFromTOML_StepRequiresPosition(t *testing.T) {
	body := `
[meta]
schema_version=1
title_slug="halo_infinite"

[[arcs]]
id="a"
title_en="A"
title_fr="A"

  [[arcs.steps]]
  position=0
  template_id="t1"
  target_tier="normal"
`
	path := writeTempTOML(t, "presets_bad_pos.toml", body)
	repo := &capturePresetArcRepo{}
	_, err := LoadPresetArcsFromTOML(context.Background(), repo, path)
	if err == nil {
		t.Error("expected error")
	}
}

func TestLoadPresetArcsFromTOML_ArcRequiresTitle(t *testing.T) {
	body := `
[meta]
schema_version=1
title_slug="halo_infinite"

[[arcs]]
id="a"
title_en=""
title_fr=""
`
	path := writeTempTOML(t, "presets_no_title.toml", body)
	repo := &capturePresetArcRepo{}
	_, err := LoadPresetArcsFromTOML(context.Background(), repo, path)
	if err == nil {
		t.Error("expected error")
	}
}

// ─── enums.go : couvre les String() restants ───

func TestEnumStringers(t *testing.T) {
	cases := []struct {
		got, want string
	}{
		{StatusActive.String(), "active"},
		{StatusCompleted.String(), "completed"},
		{TierHeroic.String(), "heroic"},
		{CadenceWeekly.String(), "weekly"},
		{EvalCumulative.String(), "cumulative"},
		{WindowDeadline.String(), "deadline"},
		{ModePilote.String(), "pilote"},
		{DataEstimated.String(), "estimated"},
		{SquadCompetitive.String(), "competitive"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("String(): got %q want %q", tc.got, tc.want)
		}
	}
}

// ─── lifecycle.go : CanAbandon ───

func TestCanAbandon(t *testing.T) {
	cases := map[ChallengeStatus]bool{
		StatusActive:    true,
		StatusDraft:     false,
		StatusCompleted: false,
		StatusExpired:   false,
		StatusAbandoned: false,
		StatusArchived:  false,
	}
	for s, want := range cases {
		if got := CanAbandon(Challenge{Status: s}); got != want {
			t.Errorf("CanAbandon(%s) = %v, want %v", s, got, want)
		}
	}
}

// ─── tuning.go : LoadTuning paths supplémentaires ───

func TestLoadTuning_MissingFallsBack(t *testing.T) {
	got := LoadTuning("/no/such/tuning.toml")
	if got.SchemaVersion == 0 {
		t.Error("expected DefaultTuning fallback (schema_version != 0)")
	}
}

func TestLoadTuning_ParseErrorFallsBack(t *testing.T) {
	path := writeTempTOML(t, "broken.toml", "garbage = [")
	got := LoadTuning(path)
	if got.AntiSmurf.MinStretch == 0 {
		t.Error("expected DefaultTuning fallback")
	}
}

func TestLoadTuning_InvalidValidationFallsBack(t *testing.T) {
	// Stretch non monotones → Validate() échoue → DefaultTuning.
	body := `
schema_version = 1

[anti_smurf]
min_stretch = 0.08
population_min_threshold = 50
baseline_stale_days = 60
recovery_matches = 10

[stretch_thresholds]
normal = 1.50
heroic = 1.40
legendary = 1.30
mythic = 1.20

[pp_amounts]
normal = 50
heroic = 75
legendary = 125
mythic = 200

[data_tier]
full = 1.0
estimated = 0.5
tracking = 0.0

[levels]
thresholds = [0, 500]
names = ["A", "B"]

[baseline]
window_matches = 20
matches_full = 10
matches_estimated = 5

[squad_pool]
size_min = 6
size_max = 9
`
	path := writeTempTOML(t, "invalid.toml", body)
	got := LoadTuning(path)
	// Devrait être DefaultTuning : Stretch.Normal = 1.08 dans default.
	if got.Stretch.Normal != 1.08 {
		t.Errorf("expected DefaultTuning fallback after Validate, got %v", got.Stretch.Normal)
	}
}

func TestTuning_WinRateMinForWindow_AllBranches(t *testing.T) {
	tn := DefaultTuning()
	cases := []struct {
		wt   WindowType
		val  string
		want int
	}{
		{WindowSession, "", tn.WinRateMin.Session},
		{WindowRollingDays, "7", tn.WinRateMin.Rolling7d},
		{WindowRollingDays, "14", tn.WinRateMin.Rolling14d},
		{WindowRollingDays, "30", tn.WinRateMin.Rolling30d},
		{WindowRollingDays, "999", 0},
		{WindowDeadline, "", 0},
	}
	for _, tc := range cases {
		if got := tn.WinRateMinForWindow(tc.wt, tc.val); got != tc.want {
			t.Errorf("WinRateMinForWindow(%s,%q)=%d want %d", tc.wt, tc.val, got, tc.want)
		}
	}
}

func TestTuning_PopulationCapTier_AllBranches(t *testing.T) {
	tn := DefaultTuning()
	cases := []struct {
		pct  float64
		want Tier
	}{
		{0.10, TierNormal},
		{0.60, TierHeroic},
		{0.80, TierLegendary},
		{0.95, TierMythic},
	}
	for _, tc := range cases {
		if got := tn.PopulationCapTier(tc.pct); got != tc.want {
			t.Errorf("PopulationCapTier(%v)=%s want %s", tc.pct, got, tc.want)
		}
	}
}
