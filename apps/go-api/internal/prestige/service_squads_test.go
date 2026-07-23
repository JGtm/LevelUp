package prestige

import (
	"context"
	"errors"
	"testing"
	"time"
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

func TestService_RenameSquad_RequiresMemberUser(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sqRepo.members = []SquadMember{{Xuid: "x1", UserID: "alice"}}

	if err := svc.RenameSquad(context.Background(), "sq1", "Nouveau", "outsider"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("outsider doit être rejeté, got %v", err)
	}
	if len(sqRepo.renamed) != 0 {
		t.Errorf("aucun rename attendu après rejet, got %d", len(sqRepo.renamed))
	}
	if err := svc.RenameSquad(context.Background(), "sq1", "Nouveau", "alice"); err != nil {
		t.Errorf("membre-user doit pouvoir renommer, got %v", err)
	}
	if len(sqRepo.renamed) != 1 || sqRepo.renamed[0] != [2]string{"sq1", "Nouveau"} {
		t.Errorf("rename inattendu: %+v", sqRepo.renamed)
	}
}

func TestService_RenameSquad_RequiresName(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sqRepo.members = []SquadMember{{Xuid: "x1", UserID: "alice"}}
	if err := svc.RenameSquad(context.Background(), "sq1", "", "alice"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("name vide: want ErrInvalidInput, got %v", err)
	}
}

func TestService_DeleteSquad_RemovesAllMembers(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sqRepo.members = []SquadMember{{Xuid: "x1", UserID: "alice"}, {Xuid: "x2"}}

	if err := svc.DeleteSquad(context.Background(), "sq1", "outsider"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("outsider doit être rejeté, got %v", err)
	}
	if len(sqRepo.removed) != 0 {
		t.Errorf("aucun retrait attendu après rejet, got %d", len(sqRepo.removed))
	}
	if err := svc.DeleteSquad(context.Background(), "sq1", "alice"); err != nil {
		t.Fatalf("DeleteSquad: %v", err)
	}
	if len(sqRepo.removed) != 2 {
		t.Errorf("removed=%d, want 2 (tous les membres retirés)", len(sqRepo.removed))
	}
}

func TestService_DeleteSquad_PartialFailure_ReturnsError(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sqRepo.members = []SquadMember{{Xuid: "x1", UserID: "alice"}, {Xuid: "x2"}}
	sqRepo.removeErr = errors.New("db locked")

	err := svc.DeleteSquad(context.Background(), "sq1", "alice")
	if err == nil {
		t.Fatal("retrait en échec doit remonter une erreur (pas un faux succès)")
	}
	// Best-effort : tous les retraits sont tentés malgré l'échec (idempotent au retry).
	if len(sqRepo.removed) != 2 {
		t.Errorf("removed=%d, want 2 (tous tentés)", len(sqRepo.removed))
	}
}

func TestService_SquadUsualContexts_DelegatesToProvider(t *testing.T) {
	svc, _, _, _, _, _ := buildFullService()
	svc.deps.SquadMatches = &fakeSquadMatchProvider{
		usualPlaylists: []string{"Classé", "Grande équipe"},
		usualModes:     []string{"Slayer"},
	}
	pls, mds, err := svc.SquadUsualContexts(context.Background(), []string{"x1", "x2"}, "halo_infinite")
	if err != nil {
		t.Fatalf("SquadUsualContexts: %v", err)
	}
	if len(pls) != 2 || pls[0] != "Classé" || len(mds) != 1 || mds[0] != "Slayer" {
		t.Errorf("got pls=%v mds=%v", pls, mds)
	}
	// roster vide → nil sans erreur (best-effort)
	if pls2, _, _ := svc.SquadUsualContexts(context.Background(), nil, "halo_infinite"); pls2 != nil {
		t.Errorf("roster vide: want nil, got %v", pls2)
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
	matches        []SquadMatchMetric
	err            error
	usualPlaylists []string
	usualModes     []string
	lastSince      time.Time // capture la borne basse reçue (Lot 4)
}

func (f *fakeSquadMatchProvider) SquadMatchMetrics(_ context.Context, _ []string, _, _ string, _ int, since time.Time) ([]SquadMatchMetric, error) {
	f.lastSince = since
	return f.matches, f.err
}

func (f *fakeSquadMatchProvider) SquadUsualContexts(_ context.Context, _ []string, _ string, _ int) ([]string, []string, error) {
	return f.usualPlaylists, f.usualModes, nil
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

// TestSquadEvalSince (Lot 4, item 4.1/4.2) : la borne basse est created_at, sauf
// rolling_days où la plus récente de (created_at, now-N j) l'emporte.
func TestSquadEvalSince(t *testing.T) {
	created := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name       string
		windowType WindowType
		windowVal  string
		want       time.Time
	}{
		{"last_n_matches → created_at", WindowLastNMatches, "5", created},
		{"session → created_at", WindowSession, "3", created},
		{"rolling_days récent > created", WindowRollingDays, "5", now.AddDate(0, 0, -5)},
		{"rolling_days ancien < created → created", WindowRollingDays, "60", created},
		{"rolling_days invalide → created", WindowRollingDays, "abc", created},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sc := SquadChallenge{CreatedAt: created, WindowType: tc.windowType, WindowValue: tc.windowVal}
			if got := squadEvalSince(sc, now); !got.Equal(tc.want) {
				t.Errorf("squadEvalSince = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestService_EvaluateSquadChallenge_BoundsSinceToCreatedAt (Lot 4, item 4.1) :
// EvaluateSquadChallenge transmet une borne basse = created_at au provider (plus
// de complétion rétroactive avec l'historique antérieur à la création).
func TestService_EvaluateSquadChallenge_BoundsSinceToCreatedAt(t *testing.T) {
	svc, _, _, scRepo, sqRepo, tplRepo := buildFullService()
	prov := &fakeSquadMatchProvider{matches: []SquadMatchMetric{
		{MatchID: "m1", Xuids: []string{xA, xB}, Values: map[string]float64{xA: 6, xB: 2}},
	}}
	svc.deps.SquadMatches = prov
	created := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	scRepo.getResp = SquadChallenge{
		ID: "sc1", SquadID: "sq1", TemplateID: "tpl1", TitleSlug: "halo_infinite",
		TargetPerMember: 10, WindowType: WindowLastNMatches, WindowValue: "5", CreatedAt: created,
	}
	tplRepo.templates = []Template{{ID: "tpl1", Metric: "kills"}}
	sqRepo.members = []SquadMember{{SquadID: "sq1", Xuid: xA, UserID: "alice"}, {SquadID: "sq1", Xuid: xB, UserID: "bob"}}

	if _, err := svc.EvaluateSquadChallenge(context.Background(), "sc1", "alice"); err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !prov.lastSince.Equal(created) {
		t.Errorf("borne since = %v, want created_at %v", prov.lastSince, created)
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
