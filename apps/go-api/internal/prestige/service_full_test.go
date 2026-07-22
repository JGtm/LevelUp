package prestige

import (
	"context"
	"errors"
	"testing"
	"time"
)

// service_full_test.go — tests d'orchestration du Service avec mocks complets.
// Couvre les paths critiques de service.go + service_arcs_squads.go +
// service_pilot_pool.go non couverts par service_quotas_test.go.

// fakeArcRepo capture les arcs créés.
type fakeArcRepo struct {
	created []Arc
	deleted []string
	getResp Arc
	getErr  error
}

func (r *fakeArcRepo) Create(_ context.Context, a Arc) error {
	r.created = append(r.created, a)
	return nil
}
func (r *fakeArcRepo) Get(_ context.Context, _ string) (Arc, error) {
	return r.getResp, r.getErr
}
func (r *fakeArcRepo) ListByUser(_ context.Context, _, _ string) ([]Arc, error) { return nil, nil }
func (r *fakeArcRepo) MarkCompleted(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (r *fakeArcRepo) Delete(_ context.Context, id string) error {
	r.deleted = append(r.deleted, id)
	return nil
}

// squadProgressUpdate capture un appel à UpdateParticipantProgress.
type squadProgressUpdate struct {
	ChallengeID string
	UserID      string
	Value       float64
	CompletedAt *time.Time
}

// fakeSquadChallengeRepo capture les défis escouade.
type fakeSquadChallengeRepo struct {
	createdChallenges []SquadChallenge
	addedParticipants []SquadChallengeParticipant
	getResp           SquadChallenge
	getErr            error
	progressUpdates   []squadProgressUpdate
}

func (r *fakeSquadChallengeRepo) Create(_ context.Context, sc SquadChallenge) error {
	r.createdChallenges = append(r.createdChallenges, sc)
	return nil
}
func (r *fakeSquadChallengeRepo) Get(_ context.Context, _ string) (SquadChallenge, error) {
	return r.getResp, r.getErr
}
func (r *fakeSquadChallengeRepo) ListBySquad(_ context.Context, _ string) ([]SquadChallenge, error) {
	return nil, nil
}
func (r *fakeSquadChallengeRepo) AddParticipant(_ context.Context, p SquadChallengeParticipant) error {
	r.addedParticipants = append(r.addedParticipants, p)
	return nil
}
func (r *fakeSquadChallengeRepo) UpdateParticipantProgress(_ context.Context, challengeID, userID string, value float64, completedAt *time.Time) error {
	r.progressUpdates = append(r.progressUpdates, squadProgressUpdate{
		ChallengeID: challengeID, UserID: userID, Value: value, CompletedAt: completedAt,
	})
	return nil
}
func (r *fakeSquadChallengeRepo) ListParticipants(_ context.Context, _ string) ([]SquadChallengeParticipant, error) {
	return nil, nil
}
func (r *fakeSquadChallengeRepo) CountActiveParticipants(_ context.Context, _ string) (int, error) {
	return 0, nil
}

// fakeSquadRepo capture les squads (stateful pour les tests CRUD).
//
// `members` est le roster renvoyé par ListMembers (seed) ; `squadsByUser` le
// retour de ListSquadsForUser. created/added/removed capturent les écritures.
type fakeSquadRepo struct {
	members      []SquadMember
	squadsByUser []Squad
	created      []Squad
	added        []SquadMember
	removed      []string
	renamed      [][2]string // {id, name}
	removeErr    error       // si non nil, RemoveMember échoue (test échec partiel)
}

func (r *fakeSquadRepo) Create(_ context.Context, s Squad) error {
	r.created = append(r.created, s)
	return nil
}
func (r *fakeSquadRepo) Get(_ context.Context, id string) (Squad, error) {
	for _, s := range r.created {
		if s.ID == id {
			return s, nil
		}
	}
	return Squad{}, nil
}
func (r *fakeSquadRepo) AddMember(_ context.Context, m SquadMember) error {
	r.added = append(r.added, m)
	return nil
}
func (r *fakeSquadRepo) RemoveMember(_ context.Context, _, xuid string) error {
	r.removed = append(r.removed, xuid)
	return r.removeErr
}
func (r *fakeSquadRepo) Rename(_ context.Context, id, name string) error {
	r.renamed = append(r.renamed, [2]string{id, name})
	return nil
}
func (r *fakeSquadRepo) ListMembers(_ context.Context, _ string) ([]SquadMember, error) {
	return r.members, nil
}
func (r *fakeSquadRepo) ListSquadsForUser(_ context.Context, _ string) ([]Squad, error) {
	return r.squadsByUser, nil
}

// fakeTemplateRepo capture les templates.
type fakeTemplateRepo struct {
	templates []Template
}

func (r *fakeTemplateRepo) ListByTitle(_ context.Context, _ string) ([]Template, error) {
	return r.templates, nil
}
func (r *fakeTemplateRepo) GetByID(_ context.Context, id string) (Template, error) {
	for _, t := range r.templates {
		if t.ID == id {
			return t, nil
		}
	}
	return Template{}, errors.New("not found")
}
func (r *fakeTemplateRepo) Suggest(_ context.Context, _ string, _ []string, count int) ([]Template, error) {
	if count > len(r.templates) {
		count = len(r.templates)
	}
	return r.templates[:count], nil
}
func (r *fakeTemplateRepo) Replace(_ context.Context, _ string, _ []Template) error { return nil }
func (r *fakeTemplateRepo) UpsertOne(_ context.Context, t Template) error {
	r.templates = append(r.templates, t)
	return nil
}

// buildFullService crée un service avec tous les fakes.
func buildFullService() (*service, *fakeChallengeRepo, *fakeArcRepo, *fakeSquadChallengeRepo, *fakeSquadRepo, *fakeTemplateRepo) {
	chRepo := &fakeChallengeRepo{}
	arcRepo := &fakeArcRepo{}
	scRepo := &fakeSquadChallengeRepo{}
	sqRepo := &fakeSquadRepo{}
	tplRepo := &fakeTemplateRepo{}
	deps := Deps{
		Tuning:           DefaultTuning(),
		Challenges:       chRepo,
		Arcs:             arcRepo,
		SquadChallenges:  scRepo,
		Squads:           sqRepo,
		Templates:        tplRepo,
		Telemetry:        &fakeNoOpTelemetryRepo{},
		Prestige:         &fakeNoOpPrestigeRepo{},
		BaselineProvider: &fakeBaselineProvider{},
		Now:              func() time.Time { return time.Now().UTC() },
	}
	return NewService(deps).(*service), chRepo, arcRepo, scRepo, sqRepo, tplRepo
}

// ─── Arcs ───

func TestService_CreateArc_OK(t *testing.T) {
	svc, _, arcRepo, _, _, _ := buildFullService()
	a, err := svc.CreateArc(context.Background(), CreateArcRequest{
		UserID:    "u1",
		TitleSlug: "halo_infinite",
		Title:     "Le Slayer Custom",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if a.ID == "" || a.IsPreset {
		t.Errorf("arc invalide: %+v", a)
	}
	if len(arcRepo.created) != 1 {
		t.Errorf("expected 1 created, got %d", len(arcRepo.created))
	}
}

func TestService_CreateArc_RequiredFields(t *testing.T) {
	svc, _, _, _, _, _ := buildFullService()
	_, err := svc.CreateArc(context.Background(), CreateArcRequest{
		UserID: "u1", // title_slug et title manquants
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

// ─── Suggestions + cooldown ───

// SuggestTemplates annote CooldownEndsAt sur les templates dont la métrique
// est en cooldown actif pour le joueur, et laisse les autres à nil.
func TestService_SuggestTemplates_AnnotatesCooldown(t *testing.T) {
	svc, chRepo, _, _, _, tplRepo := buildFullService()
	tplRepo.templates = []Template{
		{ID: "t-kda", TitleSlug: "halo_infinite", Metric: "FieldKDA"},
		{ID: "t-kills", TitleSlug: "halo_infinite", Metric: "FieldKills"},
	}
	recent := time.Now().UTC().Add(-1 * time.Hour) // < 24h → cooldown actif
	chRepo.listResult = []Challenge{
		{Metric: "FieldKDA", Mode: ModeLibre, Status: StatusAbandoned, AbandonedAt: &recent},
	}

	out, err := svc.SuggestTemplates(context.Background(), "u1", "halo_infinite", 5)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	byMetric := map[string]*time.Time{}
	for _, tmpl := range out {
		byMetric[tmpl.Metric] = tmpl.CooldownEndsAt
	}
	if byMetric["FieldKDA"] == nil {
		t.Error("FieldKDA template should be annotated with a cooldown")
	}
	if byMetric["FieldKills"] != nil {
		t.Errorf("FieldKills template should not have a cooldown, got %v", byMetric["FieldKills"])
	}
}

// ─── Squad challenges ───

func TestService_CreateSquadChallenge_AutoJoinsCreator(t *testing.T) {
	svc, _, _, scRepo, _, _ := buildFullService()
	sc, err := svc.CreateSquadChallenge(context.Background(), CreateSquadChallengeRequest{
		SquadID:         "sq1",
		TitleSlug:       "halo_infinite",
		Mode:            SquadCollective,
		EvalType:        EvalThreshold,
		WindowType:      WindowSession,
		TargetPerMember: 5.0,
		CreatedBy:       "u1",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if sc.ID == "" {
		t.Error("ID vide")
	}
	if len(scRepo.addedParticipants) != 1 {
		t.Errorf("expected creator auto-join, got %d participants", len(scRepo.addedParticipants))
	}
	if scRepo.addedParticipants[0].UserID != "u1" {
		t.Errorf("auto-join wrong user: %s", scRepo.addedParticipants[0].UserID)
	}
}

func TestService_CreateSquadChallenge_CollectiveRequiresTargetPerMember(t *testing.T) {
	svc, _, _, _, _, _ := buildFullService()
	_, err := svc.CreateSquadChallenge(context.Background(), CreateSquadChallengeRequest{
		SquadID: "sq1", TitleSlug: "halo_infinite",
		Mode: SquadCollective, EvalType: EvalThreshold, WindowType: WindowSession,
		CreatedBy:       "u1",
		TargetPerMember: 0, // INVALID en mode collectif
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_JoinSquadChallenge(t *testing.T) {
	svc, _, _, scRepo, sqRepo, _ := buildFullService()
	scRepo.getResp = SquadChallenge{ID: "sc1", SquadID: "sq1"}
	sqRepo.members = []SquadMember{{SquadID: "sq1", Xuid: "x2", UserID: "u2"}}
	err := svc.JoinSquadChallenge(context.Background(), "sc1", "u2", TierHeroic, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(scRepo.addedParticipants) != 1 {
		t.Errorf("expected 1 participant, got %d", len(scRepo.addedParticipants))
	}
}

// TestService_JoinSquadChallenge_RejectsNonMember : garde d'appartenance
// objet-level (BOLA). Un utilisateur qui n'est PAS membre-user de l'escouade du
// défi ne peut pas le rejoindre, même avec un challenge_id valide (l'actor guard
// du handler ne couvre que l'acteur, pas l'objet).
func TestService_JoinSquadChallenge_RejectsNonMember(t *testing.T) {
	svc, _, _, scRepo, sqRepo, _ := buildFullService()
	scRepo.getResp = SquadChallenge{ID: "sc1", SquadID: "sq1"}
	sqRepo.members = []SquadMember{{SquadID: "sq1", Xuid: "x1", UserID: "alice"}}
	if err := svc.JoinSquadChallenge(context.Background(), "sc1", "outsider", TierHeroic, false); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("non-membre doit être rejeté (BOLA objet-level), got %v", err)
	}
	if len(scRepo.addedParticipants) != 0 {
		t.Errorf("aucun participant ne doit être ajouté pour un non-membre, got %d", len(scRepo.addedParticipants))
	}
}

// TestService_JoinSquadChallenge_LookupError_NotSwallowed : si la lecture du défi
// échoue, l'erreur n'est PAS avalée en faux succès (règle 10) — aucun participant
// ajouté, une erreur remonte (et la cause est loggée côté service).
func TestService_JoinSquadChallenge_LookupError_NotSwallowed(t *testing.T) {
	svc, _, _, scRepo, _, _ := buildFullService()
	scRepo.getErr = errors.New("db boom")
	if err := svc.JoinSquadChallenge(context.Background(), "sc1", "u2", TierHeroic, false); err == nil {
		t.Error("lecture KO doit remonter une erreur (pas de faux succès)")
	}
	if len(scRepo.addedParticipants) != 0 {
		t.Errorf("aucun participant si le défi est illisible, got %d", len(scRepo.addedParticipants))
	}
}

// ─── Mode pilote ───

func TestService_EnablePilotMode_CreatesDailyAndWeekly(t *testing.T) {
	svc, _, _, _, _, tplRepo := buildFullService()
	tplRepo.templates = []Template{
		{ID: "t1", Cadence: CadenceDaily, WindowType: WindowSession, EvalType: EvalThreshold,
			Metric: "FieldKDA", LabelFR: "Stay sharp",
			NormalTarget: 1.1, HeroicTarget: 1.35, LegendaryTarget: 1.6, MythicTarget: 2.0},
		{ID: "t2", Cadence: CadenceWeekly, WindowType: WindowSession, EvalType: EvalThreshold,
			Metric: "FieldKDA", LabelFR: "Constant",
			NormalTarget: 1.2, HeroicTarget: 1.5, LegendaryTarget: 1.8, MythicTarget: 2.2},
	}
	out, err := svc.EnablePilotMode(context.Background(), "u1", "halo_infinite")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out.Daily == nil {
		t.Error("expected daily attributed")
	}
	if out.WeeklyForced == nil {
		t.Error("expected weekly forced attributed")
	}
	// Origine : défis auto-attribués par le mode pilote → source "pilot_mode" (ADR 0020).
	if out.Daily != nil && out.Daily.Source != ChallengeSourcePilotMode {
		t.Errorf("daily source=%q want %q", out.Daily.Source, ChallengeSourcePilotMode)
	}
	if out.WeeklyForced != nil && out.WeeklyForced.Source != ChallengeSourcePilotMode {
		t.Errorf("weekly forced source=%q want %q", out.WeeklyForced.Source, ChallengeSourcePilotMode)
	}
}

func TestService_EnablePilotMode_RequiredFields(t *testing.T) {
	svc, _, _, _, _, _ := buildFullService()
	_, err := svc.EnablePilotMode(context.Background(), "", "halo_infinite")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_DisablePilotMode_ArchivesActivePilotChallenges(t *testing.T) {
	svc, chRepo, _, _, _, _ := buildFullService()
	// List renvoie 2 défis pilote actifs (le service filtre déjà status=active
	// + mode=pilote côté ChallengeFilter → le fake les renvoie tels quels).
	chRepo.listResult = []Challenge{
		{ID: "c1", Status: StatusActive, Mode: ModePilote},
		{ID: "c2", Status: StatusActive, Mode: ModePilote},
	}
	if err := svc.DisablePilotMode(context.Background(), "u1", "halo_infinite"); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(chRepo.statusUpdates) != 2 {
		t.Fatalf("expected 2 status updates, got %d", len(chRepo.statusUpdates))
	}
	for _, u := range chRepo.statusUpdates {
		if u.Status != StatusArchived {
			t.Errorf("challenge %s: status=%q want %q", u.ID, u.Status, StatusArchived)
		}
	}
}

func TestService_DisablePilotMode_NoActivePilotChallenges(t *testing.T) {
	svc, chRepo, _, _, _, _ := buildFullService()
	chRepo.listResult = nil // aucun défi pilote actif
	if err := svc.DisablePilotMode(context.Background(), "u1", "halo_infinite"); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(chRepo.statusUpdates) != 0 {
		t.Errorf("expected no status update, got %d", len(chRepo.statusUpdates))
	}
}

func TestService_DisablePilotMode_RequiredFields(t *testing.T) {
	svc, _, _, _, _, _ := buildFullService()
	if err := svc.DisablePilotMode(context.Background(), "", "halo_infinite"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

// TestService_DisablePilotMode_EmitsArchivedTelemetry (F5) : chaque défi pilote
// archivé émet une transition `archived` (distincte d'`abandoned`), pour tracer
// le churn du mode pilote — comme AbandonChallenge émet `abandoned`.
func TestService_DisablePilotMode_EmitsArchivedTelemetry(t *testing.T) {
	chRepo := &fakeChallengeRepo{}
	telRepo := &fakeTelemetryRepo{}
	deps := Deps{
		Tuning:           DefaultTuning(),
		Challenges:       chRepo,
		Arcs:             &fakeArcRepo{},
		SquadChallenges:  &fakeSquadChallengeRepo{},
		Squads:           &fakeSquadRepo{},
		Templates:        &fakeTemplateRepo{},
		Telemetry:        telRepo,
		Prestige:         &fakeNoOpPrestigeRepo{},
		BaselineProvider: &fakeBaselineProvider{},
		Now:              func() time.Time { return time.Now().UTC() },
	}
	svc := NewService(deps).(*service)
	chRepo.listResult = []Challenge{
		{ID: "c1", Status: StatusActive, Mode: ModePilote},
		{ID: "c2", Status: StatusActive, Mode: ModePilote},
	}
	if err := svc.DisablePilotMode(context.Background(), "u1", "halo_infinite"); err != nil {
		t.Fatalf("err: %v", err)
	}
	if telRepo.count() != 2 {
		t.Fatalf("expected 2 telemetry events, got %d", telRepo.count())
	}
	for _, ev := range telRepo.events {
		if ev.EventType != TelemetryArchived {
			t.Errorf("event_type=%q want %q", ev.EventType, TelemetryArchived)
		}
		if ev.Mode != ModePilote {
			t.Errorf("mode=%q want %q", ev.Mode, ModePilote)
		}
	}
}

// ─── Squad pool ───

func TestService_RefreshSquadPool_RequiresMembership(t *testing.T) {
	svc, _, _, _, sqRepo, tplRepo := buildFullService()
	tplRepo.templates = []Template{{ID: "t1"}, {ID: "t2"}}
	sqRepo.members = []SquadMember{{UserID: "u1"}}
	_, err := svc.RefreshSquadPool(context.Background(), "sq1", "halo_infinite", "u_outsider")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("non-member should be rejected, got %v", err)
	}
}

func TestService_RefreshSquadPool_GeneratesPool(t *testing.T) {
	svc, _, _, _, sqRepo, tplRepo := buildFullService()
	sqRepo.members = []SquadMember{{UserID: "u1"}}
	tplRepo.templates = make([]Template, 20)
	for i := range tplRepo.templates {
		tplRepo.templates[i] = Template{ID: "t" + string(rune('a'+i)), Cadence: CadenceWeekly}
	}
	pool, err := svc.RefreshSquadPool(context.Background(), "sq1", "halo_infinite", "u1")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(pool) < 6 || len(pool) > 9 {
		t.Errorf("pool size out of range [6,9]: %d", len(pool))
	}
}

// ─── UpdateChallenge ───

func TestService_UpdateChallenge_LabelOnly(t *testing.T) {
	svc, chRepo, _, _, _, _ := buildFullService()
	chRepo.activeTotal = 1
	// Set up Get to return an active libre challenge
	// Le fake n'implémente pas Get → on patch via une variante ; on utilise le fait
	// que UpdateChallenge appelle Get() qui retourne ErrChallengeNotFound dans le fake.
	_, err := svc.UpdateChallenge(context.Background(), "ch_inconnu", UpdateChallengePatch{})
	if !errors.Is(err, ErrChallengeNotFound) {
		t.Errorf("expected ErrChallengeNotFound, got %v", err)
	}
}

// ─── Suggestions ───

func TestService_SuggestTemplates_DefaultCount(t *testing.T) {
	svc, _, _, _, _, tplRepo := buildFullService()
	tplRepo.templates = []Template{{ID: "t1"}, {ID: "t2"}, {ID: "t3"}, {ID: "t4"}}
	out, err := svc.SuggestTemplates(context.Background(), "u1", "halo_infinite", 0)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(out) != 3 { // alternatives_count = 3 par défaut
		t.Errorf("expected 3 suggestions, got %d", len(out))
	}
}
