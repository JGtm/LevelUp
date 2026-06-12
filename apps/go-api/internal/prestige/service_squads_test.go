package prestige

import (
	"context"
	"errors"
	"testing"
)

func TestService_CreateSquad_CreatesAndAddsMembers(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sq, err := svc.CreateSquad(context.Background(), CreateSquadRequest{
		Name:      "Trio",
		CreatedBy: "alice",
		Members: []SquadMember{
			{Xuid: "x1", UserID: "alice"},
			{Xuid: "x2", UserID: "bob"},
			{Xuid: "x3"}, // ami hors-app (pas de user_id)
		},
	})
	if err != nil {
		t.Fatalf("CreateSquad: %v", err)
	}
	if sq.ID == "" || sq.Name != "Trio" || sq.CreatedBy != "alice" {
		t.Errorf("squad mal formé: %+v", sq)
	}
	if len(sqRepo.created) != 1 {
		t.Errorf("created=%d, want 1", len(sqRepo.created))
	}
	if len(sqRepo.added) != 3 {
		t.Fatalf("added=%d, want 3", len(sqRepo.added))
	}
	for _, m := range sqRepo.added {
		if m.SquadID != sq.ID {
			t.Errorf("membre %q squadID=%q, want %q", m.Xuid, m.SquadID, sq.ID)
		}
	}
}

func TestService_CreateSquad_SkipsMembersWithoutXUID(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	_, err := svc.CreateSquad(context.Background(), CreateSquadRequest{
		Name:      "Duo",
		CreatedBy: "alice",
		Members: []SquadMember{
			{Xuid: "x1", UserID: "alice"},
			{Xuid: "", UserID: "ghost"}, // clé invalide → ignoré
		},
	})
	if err != nil {
		t.Fatalf("CreateSquad: %v", err)
	}
	if len(sqRepo.added) != 1 {
		t.Errorf("added=%d, want 1 (membre sans xuid ignoré)", len(sqRepo.added))
	}
}

func TestService_CreateSquad_RequiresNameAndCreator(t *testing.T) {
	svc, _, _, _, _, _ := buildFullService()
	if _, err := svc.CreateSquad(context.Background(), CreateSquadRequest{Name: "", CreatedBy: "alice"}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("name vide: want ErrInvalidInput, got %v", err)
	}
	if _, err := svc.CreateSquad(context.Background(), CreateSquadRequest{Name: "X", CreatedBy: ""}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("created_by vide: want ErrInvalidInput, got %v", err)
	}
}

func TestService_AddSquadMember_RequiresMemberUser(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sqRepo.members = []SquadMember{{Xuid: "x1", UserID: "alice"}} // alice = membre-user

	// outsider (non membre-user) rejeté
	if err := svc.AddSquadMember(context.Background(), "sq1", SquadMember{Xuid: "x9"}, "outsider"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("outsider doit être rejeté, got %v", err)
	}
	if len(sqRepo.added) != 0 {
		t.Errorf("aucun ajout attendu après rejet, got %d", len(sqRepo.added))
	}

	// alice (membre-user) autorisée
	if err := svc.AddSquadMember(context.Background(), "sq1", SquadMember{Xuid: "x9", UserID: "carol"}, "alice"); err != nil {
		t.Errorf("membre-user doit pouvoir ajouter, got %v", err)
	}
	if len(sqRepo.added) != 1 || sqRepo.added[0].Xuid != "x9" || sqRepo.added[0].SquadID != "sq1" {
		t.Errorf("ajout inattendu: %+v", sqRepo.added)
	}
}

func TestService_AddSquadMember_RequiresXUID(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sqRepo.members = []SquadMember{{Xuid: "x1", UserID: "alice"}}
	if err := svc.AddSquadMember(context.Background(), "sq1", SquadMember{Xuid: ""}, "alice"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("xuid vide: want ErrInvalidInput, got %v", err)
	}
}

func TestService_RemoveSquadMember_RequiresMemberUser(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sqRepo.members = []SquadMember{{Xuid: "x1", UserID: "alice"}}

	if err := svc.RemoveSquadMember(context.Background(), "sq1", "x1", "outsider"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("outsider doit être rejeté, got %v", err)
	}
	if err := svc.RemoveSquadMember(context.Background(), "sq1", "x1", "alice"); err != nil {
		t.Errorf("membre-user doit pouvoir retirer, got %v", err)
	}
	if len(sqRepo.removed) != 1 || sqRepo.removed[0] != "x1" {
		t.Errorf("retrait inattendu: %+v", sqRepo.removed)
	}
}

func TestService_ListSquadsForUser(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sqRepo.squadsByUser = []Squad{{ID: "sq1", Name: "Trio"}}

	got, err := svc.ListSquadsForUser(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListSquadsForUser: %v", err)
	}
	if len(got) != 1 || got[0].ID != "sq1" {
		t.Errorf("got=%+v, want [sq1]", got)
	}
	if _, err := svc.ListSquadsForUser(context.Background(), ""); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("user vide: want ErrInvalidInput, got %v", err)
	}
}

// fakeSquadMatchProvider renvoie des SquadMatchMetric figés (sans DB).
type fakeSquadMatchProvider struct {
	matches []SquadMatchMetric
	err     error
}

func (f *fakeSquadMatchProvider) SquadMatchMetrics(_ context.Context, _ []string, _, _ string, _ int) ([]SquadMatchMetric, error) {
	return f.matches, f.err
}

func TestService_EvaluateSquadChallenge_AggregatesAndPersists(t *testing.T) {
	svc, _, _, scRepo, sqRepo, tplRepo := buildFullService()
	svc.deps.SquadMatches = &fakeSquadMatchProvider{matches: []SquadMatchMetric{
		{MatchID: "m1", Xuids: []string{xA, xB}, Values: map[string]float64{xA: 6, xB: 2}},
		{MatchID: "m2", Xuids: []string{xA, xB}, Values: map[string]float64{xA: 5, xB: 1}},
	}}
	scRepo.getResp = SquadChallenge{
		ID: "sc1", SquadID: "sq1", TemplateID: "tpl1", TitleSlug: "halo_infinite",
		TargetPerMember: 10, WindowType: WindowLastNMatches, WindowValue: "5",
	}
	tplRepo.templates = []Template{{ID: "tpl1", Metric: "kills"}}
	sqRepo.members = []SquadMember{
		{SquadID: "sq1", Xuid: xA, UserID: "alice"},
		{SquadID: "sq1", Xuid: xB, UserID: "bob"},
	}

	progress, err := svc.EvaluateSquadChallenge(context.Background(), "sc1", "alice")
	if err != nil {
		t.Fatalf("EvaluateSquadChallenge: %v", err)
	}
	byX := progressByXUID(progress)
	if byX[xA].Value != 11 || !byX[xA].Completed {
		t.Errorf("A: %+v, want value=11 completed", byX[xA])
	}
	if byX[xB].Value != 3 || byX[xB].Completed {
		t.Errorf("B: %+v, want value=3 non-completed", byX[xB])
	}
	// Les 2 membres-app ont une progression persistée.
	if len(scRepo.progressUpdates) != 2 {
		t.Fatalf("progressUpdates=%d, want 2", len(scRepo.progressUpdates))
	}
}

func TestService_EvaluateSquadChallenge_RejectsNonMember(t *testing.T) {
	svc, _, _, scRepo, sqRepo, _ := buildFullService()
	svc.deps.SquadMatches = &fakeSquadMatchProvider{}
	scRepo.getResp = SquadChallenge{ID: "sc1", SquadID: "sq1", TemplateID: "tpl1"}
	sqRepo.members = []SquadMember{{SquadID: "sq1", Xuid: xA, UserID: "alice"}}

	if _, err := svc.EvaluateSquadChallenge(context.Background(), "sc1", "outsider"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("non-membre doit être rejeté, got %v", err)
	}
}

func TestService_EvaluateSquadChallenge_RequiresTemplate(t *testing.T) {
	svc, _, _, scRepo, sqRepo, _ := buildFullService()
	svc.deps.SquadMatches = &fakeSquadMatchProvider{}
	scRepo.getResp = SquadChallenge{ID: "sc1", SquadID: "sq1", TemplateID: ""} // pas de template
	sqRepo.members = []SquadMember{{SquadID: "sq1", Xuid: xA, UserID: "alice"}}

	if _, err := svc.EvaluateSquadChallenge(context.Background(), "sc1", "alice"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("défi sans template doit être rejeté, got %v", err)
	}
}

// fakeSquadProfileProvider renvoie un profil 6-axes figé (sans DB).
type fakeSquadProfileProvider struct {
	axes []map[string]float64
}

func (f *fakeSquadProfileProvider) SquadAxes(_ context.Context, _ []string, _ string) ([]map[string]float64, error) {
	return f.axes, nil
}

func TestService_RefreshSquadPool_CoachBiasFavorsWeakAxis(t *testing.T) {
	svc, _, _, _, sqRepo, tplRepo := buildFullService()
	sqRepo.members = []SquadMember{{SquadID: "sq1", Xuid: xA, UserID: "alice"}}
	// Profil faible sur "support" → axe focal du coach.
	svc.deps.SquadProfile = &fakeSquadProfileProvider{axes: []map[string]float64{
		{"combat": 0.9, "support": 0.2, "survival": 0.7},
	}}
	// Un seul template "support", noyé parmi des combat/survival.
	tplRepo.templates = []Template{
		{ID: "c1", Metric: "kills_total", Cadence: CadenceWeekly},
		{ID: "c2", Metric: "kills_total", Cadence: CadenceWeekly},
		{ID: "s1", Metric: "assists", Cadence: CadenceWeekly},
		{ID: "d1", Metric: "deaths_total", Cadence: CadenceWeekly},
	}
	pool, err := svc.RefreshSquadPool(context.Background(), "sq1", "halo_infinite", "alice")
	if err != nil {
		t.Fatalf("RefreshSquadPool: %v", err)
	}
	if len(pool) == 0 {
		t.Fatal("pool vide")
	}
	// Le biais place le template de l'axe faible (support) en tête.
	if MetricToRadarAxis(pool[0].Metric) != "support" {
		t.Errorf("pool[0] axe = %q (metric %q), want support",
			MetricToRadarAxis(pool[0].Metric), pool[0].Metric)
	}
}

func TestService_RefreshSquadPool_NoProfileNoBias(t *testing.T) {
	// Sans SquadProfile provider, RefreshSquadPool reste un shuffle (pas d'erreur).
	svc, _, _, _, sqRepo, tplRepo := buildFullService()
	sqRepo.members = []SquadMember{{SquadID: "sq1", Xuid: xA, UserID: "alice"}}
	tplRepo.templates = []Template{{ID: "c1", Metric: "kills_total", Cadence: CadenceWeekly}}
	if _, err := svc.RefreshSquadPool(context.Background(), "sq1", "halo_infinite", "alice"); err != nil {
		t.Errorf("RefreshSquadPool sans profil: %v", err)
	}
}

func TestService_SquadOrientation_WeakestAxis(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sqRepo.members = []SquadMember{{SquadID: "sq1", Xuid: xA, UserID: "alice"}}
	svc.deps.SquadProfile = &fakeSquadProfileProvider{axes: []map[string]float64{
		{"combat": 0.8, "survival": 0.2, "score": 0.6}, // survival = le plus faible
	}}
	axis, err := svc.SquadOrientation(context.Background(), "sq1", "alice")
	if err != nil {
		t.Fatalf("SquadOrientation: %v", err)
	}
	if axis != "survival" {
		t.Errorf("axis = %q, want survival (le plus faible)", axis)
	}
}

func TestService_SquadOrientation_RejectsNonMember(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sqRepo.members = []SquadMember{{SquadID: "sq1", Xuid: xA, UserID: "alice"}}
	if _, err := svc.SquadOrientation(context.Background(), "sq1", "outsider"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("non-membre doit être rejeté, got %v", err)
	}
}

func TestService_SquadOrientation_NoProfileReturnsEmpty(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sqRepo.members = []SquadMember{{SquadID: "sq1", Xuid: xA, UserID: "alice"}}
	// Pas de SquadProfile provider → axe "" (pas d'orientation, pas d'erreur).
	axis, err := svc.SquadOrientation(context.Background(), "sq1", "alice")
	if err != nil {
		t.Fatalf("SquadOrientation: %v", err)
	}
	if axis != "" {
		t.Errorf("axis = %q, want vide (pas de provider)", axis)
	}
}
