package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/prestige"
)

// mockPrestigeService est l'implémentation in-memory pour tester le handler.
type mockPrestigeService struct {
	createResp      prestige.Challenge
	createErr       error
	getResp         prestige.Challenge
	getErr          error
	listResp        []prestige.Challenge
	listErr         error
	updateResp      prestige.Challenge
	updateErr       error
	abandonErr      error
	suggestNextResp []prestige.Template
	suggestNextErr  error
	mePrestige      prestige.UserPrestige
	meErr           error
	suggestTplResp  []prestige.Template
	suggestTplErr   error
	createSquadErr  error
	joinSquadErr    error
	deleteArcErr    error
	listPresetsResp []prestige.PresetArc
	listPresetsErr  error
	adoptResp       prestige.Arc
	adoptErr        error

	lastCreate       prestige.CreateChallengeRequest
	lastUpdate       prestige.UpdateChallengePatch
	lastUpdateID     string
	lastAbandonID    string
	lastSuggestNext  string
	lastDeleteArcID  string
	lastDeleteArcUID string
	lastDeleteArcCsc bool
	lastAdoptID      string
	lastAdoptUID     string

	lastCreateSquad    prestige.CreateSquadRequest
	listSquadsResp     []prestige.Squad
	lastAddSquadID     string
	lastAddSquadMember prestige.SquadMember
	lastAddSquadReqBy  string
	evalSquadResp      []prestige.SquadParticipantProgress
	lastEvalSquadID    string
	lastEvalSquadReqBy string
	orientationResp    string

	lastRenameSquadID    string
	lastRenameSquadName  string
	lastRenameSquadReqBy string
	renameSquadErr       error
	lastDeleteSquadID    string
	lastDeleteSquadReqBy string
	deleteSquadErr       error
	usualPlaylistsResp   []string
	usualModesResp       []string
}

func (m *mockPrestigeService) CreateChallenge(ctx context.Context, req prestige.CreateChallengeRequest) (prestige.Challenge, error) {
	m.lastCreate = req
	return m.createResp, m.createErr
}

func (m *mockPrestigeService) UpdateChallenge(ctx context.Context, id string, patch prestige.UpdateChallengePatch) (prestige.Challenge, error) {
	m.lastUpdateID = id
	m.lastUpdate = patch
	return m.updateResp, m.updateErr
}

func (m *mockPrestigeService) AbandonChallenge(ctx context.Context, id string) error {
	m.lastAbandonID = id
	return m.abandonErr
}

func (m *mockPrestigeService) GetChallenge(ctx context.Context, id string) (prestige.Challenge, error) {
	return m.getResp, m.getErr
}

func (m *mockPrestigeService) ListActiveChallenges(ctx context.Context, userID, titleSlug string) ([]prestige.Challenge, error) {
	return m.listResp, m.listErr
}

func (m *mockPrestigeService) EvaluateForUser(ctx context.Context, userID, titleSlug string) ([]prestige.EvaluationOutcome, error) {
	return nil, nil
}

func (m *mockPrestigeService) GetUserPrestige(ctx context.Context, userID, titleSlug string) (prestige.UserPrestige, error) {
	return m.mePrestige, m.meErr
}

func (m *mockPrestigeService) SuggestTemplates(ctx context.Context, userID, titleSlug string, count int) ([]prestige.Template, error) {
	return m.suggestTplResp, m.suggestTplErr
}

func (m *mockPrestigeService) SuggestNext(ctx context.Context, completedID string) ([]prestige.Template, error) {
	m.lastSuggestNext = completedID
	return m.suggestNextResp, m.suggestNextErr
}

func (m *mockPrestigeService) CreateArc(ctx context.Context, _ prestige.CreateArcRequest) (prestige.Arc, error) {
	return prestige.Arc{}, nil
}
func (m *mockPrestigeService) ListArcs(ctx context.Context, _, _ string) ([]prestige.Arc, error) {
	return nil, nil
}
func (m *mockPrestigeService) ListArcPresets(ctx context.Context, _, _ string) ([]prestige.PresetArc, error) {
	return m.listPresetsResp, m.listPresetsErr
}
func (m *mockPrestigeService) AdoptPresetArc(ctx context.Context, userID, _, presetID string) (prestige.Arc, error) {
	m.lastAdoptUID = userID
	m.lastAdoptID = presetID
	return m.adoptResp, m.adoptErr
}
func (m *mockPrestigeService) GetArc(ctx context.Context, _ string) (prestige.Arc, error) {
	return prestige.Arc{}, nil
}
func (m *mockPrestigeService) DeleteArc(ctx context.Context, userID, id string, opts prestige.DeleteArcOptions) error {
	m.lastDeleteArcUID = userID
	m.lastDeleteArcID = id
	m.lastDeleteArcCsc = opts.CascadeObjectives
	return m.deleteArcErr
}
func (m *mockPrestigeService) CreateSquadChallenge(ctx context.Context, _ prestige.CreateSquadChallengeRequest) (prestige.SquadChallenge, error) {
	return prestige.SquadChallenge{}, m.createSquadErr
}
func (m *mockPrestigeService) JoinSquadChallenge(ctx context.Context, _, _ string, _ prestige.Tier, _ bool) error {
	return m.joinSquadErr
}
func (m *mockPrestigeService) GetSquadChallenge(ctx context.Context, _ string) (prestige.SquadChallenge, error) {
	return prestige.SquadChallenge{}, nil
}
func (m *mockPrestigeService) ListSquadChallenges(ctx context.Context, _ string) ([]prestige.SquadChallenge, error) {
	return nil, nil
}
func (m *mockPrestigeService) RefreshSquadPool(ctx context.Context, _, _, _ string) ([]prestige.Template, error) {
	return nil, nil
}
func (m *mockPrestigeService) CreateSquad(ctx context.Context, req prestige.CreateSquadRequest) (prestige.Squad, error) {
	m.lastCreateSquad = req
	return prestige.Squad{ID: "sq_test", Name: req.Name, CreatedBy: req.CreatedBy}, nil
}
func (m *mockPrestigeService) ListSquadsForUser(ctx context.Context, _ string) ([]prestige.Squad, error) {
	return m.listSquadsResp, nil
}
func (m *mockPrestigeService) GetSquad(ctx context.Context, _ string) (prestige.Squad, error) {
	return prestige.Squad{}, nil
}
func (m *mockPrestigeService) ListSquadMembers(ctx context.Context, _ string) ([]prestige.SquadMember, error) {
	return nil, nil
}
func (m *mockPrestigeService) AddSquadMember(ctx context.Context, squadID string, member prestige.SquadMember, requestedBy string) error {
	m.lastAddSquadID = squadID
	m.lastAddSquadMember = member
	m.lastAddSquadReqBy = requestedBy
	return nil
}
func (m *mockPrestigeService) RemoveSquadMember(ctx context.Context, _, _, _ string) error {
	return nil
}
func (m *mockPrestigeService) RenameSquad(ctx context.Context, squadID, name, requestedBy string) error {
	m.lastRenameSquadID = squadID
	m.lastRenameSquadName = name
	m.lastRenameSquadReqBy = requestedBy
	return m.renameSquadErr
}
func (m *mockPrestigeService) DeleteSquad(ctx context.Context, squadID, requestedBy string) error {
	m.lastDeleteSquadID = squadID
	m.lastDeleteSquadReqBy = requestedBy
	return m.deleteSquadErr
}
func (m *mockPrestigeService) SquadUsualContexts(ctx context.Context, _ []string, _ string) ([]string, []string, error) {
	return m.usualPlaylistsResp, m.usualModesResp, nil
}
func (m *mockPrestigeService) EvaluateSquadChallenge(ctx context.Context, squadChallengeID, requestedBy string) ([]prestige.SquadParticipantProgress, error) {
	m.lastEvalSquadID = squadChallengeID
	m.lastEvalSquadReqBy = requestedBy
	return m.evalSquadResp, nil
}
func (m *mockPrestigeService) SquadOrientation(ctx context.Context, _, _ string) (string, error) {
	return m.orientationResp, nil
}
func (m *mockPrestigeService) EnablePilotMode(ctx context.Context, _, _ string) (prestige.PilotModeAttribution, error) {
	return prestige.PilotModeAttribution{}, nil
}
func (m *mockPrestigeService) DisablePilotMode(ctx context.Context, _, _ string) error {
	return nil
}

// ─────────── Helpers ───────────

// testAppPlayers simule l'annuaire db_profiles pour les tests roster.
func testAppPlayers(_ context.Context) ([]domain.PlayerSummary, error) {
	return []domain.PlayerSummary{
		{PlayerSlug: "alice", Gamertag: "Alice", XUID: "xAlice"},
		{PlayerSlug: "bob", Gamertag: "Bob", XUID: "xBob"},
	}, nil
}

func newRouter(svc prestige.Service) *chi.Mux {
	h := NewPrestigeHandler(svc, testAppPlayers)
	r := chi.NewRouter()
	h.Mount(r)
	return r
}

// ─────────── Tests ───────────

func TestPrestigeHandler_SquadOrientation_OK(t *testing.T) {
	mock := &mockPrestigeService{orientationResp: "combat"}
	r := newRouter(mock)
	req := httptest.NewRequest(http.MethodGet, "/squads/sq1/orientation?requested_by=alice", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["axis"] != "combat" {
		t.Errorf("axis=%q, want combat", resp["axis"])
	}
}

func TestPrestigeHandler_SquadOrientation_RequiresRequestedBy(t *testing.T) {
	mock := &mockPrestigeService{}
	r := newRouter(mock)
	req := httptest.NewRequest(http.MethodGet, "/squads/sq1/orientation", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400 (requested_by requis)", w.Code)
	}
}

func TestPrestigeHandler_EvaluateSquadChallenge_OK(t *testing.T) {
	mock := &mockPrestigeService{
		evalSquadResp: []prestige.SquadParticipantProgress{{Xuid: "xAlice", Value: 12, Completed: true}},
	}
	r := newRouter(mock)
	req := httptest.NewRequest(http.MethodPost, "/squad-challenges/sc1/evaluate", bytes.NewBufferString(`{"requested_by":"alice"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.lastEvalSquadID != "sc1" || mock.lastEvalSquadReqBy != "alice" {
		t.Errorf("capturé: id=%q by=%q", mock.lastEvalSquadID, mock.lastEvalSquadReqBy)
	}
}

func TestPrestigeHandler_EvaluateSquadChallenge_RequiresRequestedBy(t *testing.T) {
	mock := &mockPrestigeService{}
	r := newRouter(mock)
	req := httptest.NewRequest(http.MethodPost, "/squad-challenges/sc1/evaluate", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400 (requested_by requis)", w.Code)
	}
}

func TestPrestigeHandler_CreateSquad_ResolvesCreatorAndTagsMembers(t *testing.T) {
	mock := &mockPrestigeService{}
	r := newRouter(mock)
	body := `{"name":"Trio","created_by":"alice","members":[{"xuid":"xBob"},{"xuid":"xFriend"}]}`
	req := httptest.NewRequest(http.MethodPost, "/squads", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d, want 201; body=%s", w.Code, w.Body.String())
	}
	got := mock.lastCreateSquad
	if got.Name != "Trio" || got.CreatedBy != "alice" {
		t.Errorf("req=%+v", got)
	}
	if len(got.Members) != 3 {
		t.Fatalf("members=%d, want 3 (créateur + 2)", len(got.Members))
	}
	userIDByXUID := map[string]string{}
	for _, m := range got.Members {
		userIDByXUID[m.Xuid] = m.UserID
	}
	if userIDByXUID["xAlice"] != "alice" {
		t.Errorf("créateur xAlice user_id=%q, want alice", userIDByXUID["xAlice"])
	}
	if userIDByXUID["xBob"] != "bob" {
		t.Errorf("xBob user_id=%q, want bob (membre-app tagué)", userIDByXUID["xBob"])
	}
	if userIDByXUID["xFriend"] != "" {
		t.Errorf("xFriend user_id=%q, want vide (ami hors-app)", userIDByXUID["xFriend"])
	}
}

func TestPrestigeHandler_CreateSquad_UnknownCreator_400(t *testing.T) {
	mock := &mockPrestigeService{}
	r := newRouter(mock)
	body := `{"name":"X","created_by":"nobody"}`
	req := httptest.NewRequest(http.MethodPost, "/squads", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400 (créateur inconnu)", w.Code)
	}
}

func TestPrestigeHandler_CreateSquad_CreatorSlugCaseInsensitive(t *testing.T) {
	mock := &mockPrestigeService{}
	r := newRouter(mock)
	// Slug créateur en casse différente de db_profiles ("alice") → doit résoudre
	// quand même, sinon unknown_creator 400 silencieux côté UI (clic « Enregistrer »
	// sans effet).
	body := `{"name":"Trio","created_by":"ALICE","members":[{"xuid":"xBob"}]}`
	req := httptest.NewRequest(http.MethodPost, "/squads", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d, want 201 (slug créateur insensible à la casse); body=%s", w.Code, w.Body.String())
	}
	hasCreator := false
	for _, m := range mock.lastCreateSquad.Members {
		if m.Xuid == "xAlice" && m.UserID == "alice" {
			hasCreator = true
		}
	}
	if !hasCreator {
		t.Errorf("créateur xAlice/alice absent du roster: %+v", mock.lastCreateSquad.Members)
	}
}

func TestPrestigeHandler_AddSquadMember_TagsAndForwards(t *testing.T) {
	mock := &mockPrestigeService{}
	r := newRouter(mock)
	body := `{"xuid":"xBob","requested_by":"alice"}`
	req := httptest.NewRequest(http.MethodPost, "/squads/sq1/members", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want 204; body=%s", w.Code, w.Body.String())
	}
	if mock.lastAddSquadID != "sq1" || mock.lastAddSquadMember.Xuid != "xBob" || mock.lastAddSquadReqBy != "alice" {
		t.Errorf("capturé: id=%q member=%+v by=%q", mock.lastAddSquadID, mock.lastAddSquadMember, mock.lastAddSquadReqBy)
	}
	if mock.lastAddSquadMember.UserID != "bob" {
		t.Errorf("xBob user_id=%q, want bob (tag membre-app)", mock.lastAddSquadMember.UserID)
	}
}

func TestPrestigeHandler_ListMySquads_RequiresUserID(t *testing.T) {
	mock := &mockPrestigeService{}
	r := newRouter(mock)
	req := httptest.NewRequest(http.MethodGet, "/squads", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400 (user_id requis)", w.Code)
	}
}

func TestPrestigeHandler_ListMySquads_OK(t *testing.T) {
	mock := &mockPrestigeService{listSquadsResp: []prestige.Squad{{ID: "sq1", Name: "Trio"}}}
	r := newRouter(mock)
	req := httptest.NewRequest(http.MethodGet, "/squads?user_id=alice", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestPrestigeHandler_ListMySquads_IncludesUsualContexts(t *testing.T) {
	mock := &mockPrestigeService{
		listSquadsResp:     []prestige.Squad{{ID: "sq1", Name: "Trio"}},
		usualPlaylistsResp: []string{"Ranked"},
		usualModesResp:     []string{"Slayer"},
	}
	r := newRouter(mock)
	req := httptest.NewRequest(http.MethodGet, "/squads?user_id=alice", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200; body=%s", w.Code, w.Body.String())
	}
	body := w.Body.Bytes()
	if !bytes.Contains(body, []byte(`"usual_playlists":["Ranked"]`)) || !bytes.Contains(body, []byte(`"usual_modes":["Slayer"]`)) {
		t.Errorf("indice dérivé absent de la réponse: %s", w.Body.String())
	}
}

func TestPrestigeHandler_RenameSquad_OK(t *testing.T) {
	mock := &mockPrestigeService{}
	r := newRouter(mock)
	body := `{"name":"Nouveau nom","requested_by":"alice"}`
	req := httptest.NewRequest(http.MethodPatch, "/squads/sq1", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want 204; body=%s", w.Code, w.Body.String())
	}
	if mock.lastRenameSquadID != "sq1" || mock.lastRenameSquadName != "Nouveau nom" || mock.lastRenameSquadReqBy != "alice" {
		t.Errorf("capturé: id=%q name=%q by=%q", mock.lastRenameSquadID, mock.lastRenameSquadName, mock.lastRenameSquadReqBy)
	}
}

func TestPrestigeHandler_RenameSquad_RequiresFields(t *testing.T) {
	mock := &mockPrestigeService{}
	r := newRouter(mock)
	req := httptest.NewRequest(http.MethodPatch, "/squads/sq1", bytes.NewBufferString(`{"requested_by":"alice"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("name manquant: status=%d, want 400", w.Code)
	}
}

func TestPrestigeHandler_DeleteSquad_OK(t *testing.T) {
	mock := &mockPrestigeService{}
	r := newRouter(mock)
	req := httptest.NewRequest(http.MethodDelete, "/squads/sq1?requested_by=alice", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want 204; body=%s", w.Code, w.Body.String())
	}
	if mock.lastDeleteSquadID != "sq1" || mock.lastDeleteSquadReqBy != "alice" {
		t.Errorf("capturé: id=%q by=%q", mock.lastDeleteSquadID, mock.lastDeleteSquadReqBy)
	}
}

func TestPrestigeHandler_DeleteSquad_RequiresRequestedBy(t *testing.T) {
	mock := &mockPrestigeService{}
	r := newRouter(mock)
	req := httptest.NewRequest(http.MethodDelete, "/squads/sq1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("requested_by manquant: status=%d, want 400", w.Code)
	}
}

// newRouterGuarded : comme newRouter mais avec une garde d'autorisation acteur
// (ADR 0029 sur routes squad top-level).
func newRouterGuarded(svc prestige.Service, guard ActorGuard) *chi.Mux {
	h := NewPrestigeHandler(svc, testAppPlayers).WithActorGuard(guard)
	r := chi.NewRouter()
	h.Mount(r)
	return r
}

// TestPrestigeHandler_SquadActorGuard_DeniesForeignActor : un appelant ne peut
// pas agir au nom d'un autre slug (created_by/requested_by/user_id) → 403. La
// garde n'autorise ici que « alice » ; toutes les requêtes prétendent « bob ».
func TestPrestigeHandler_SquadActorGuard_DeniesForeignActor(t *testing.T) {
	onlyAlice := func(_ context.Context, slug string) bool { return slug == "alice" }
	cases := []struct {
		name, method, path, body string
	}{
		{"create_squad", http.MethodPost, "/squads", `{"name":"X","created_by":"bob"}`},
		{"list_my_squads", http.MethodGet, "/squads?user_id=bob", ""},
		{"add_member", http.MethodPost, "/squads/sq1/members", `{"xuid":"xBob","requested_by":"bob"}`},
		{"remove_member", http.MethodDelete, "/squads/sq1/members/xBob?requested_by=bob", ""},
		{"rename_squad", http.MethodPatch, "/squads/sq1", `{"name":"X","requested_by":"bob"}`},
		{"delete_squad", http.MethodDelete, "/squads/sq1?requested_by=bob", ""},
		{"evaluate", http.MethodPost, "/squad-challenges/sc1/evaluate", `{"requested_by":"bob"}`},
		{"orientation", http.MethodGet, "/squads/sq1/orientation?requested_by=bob", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRouterGuarded(&mockPrestigeService{}, onlyAlice)
			var req *http.Request
			if tc.body == "" {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			} else {
				req = httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusForbidden {
				t.Errorf("status=%d, want 403 (acteur étranger refusé); body=%s", w.Code, w.Body.String())
			}
		})
	}
}

// TestPrestigeHandler_SquadActorGuard_AllowsOwnActor : l'acteur autorisé passe
// la garde (la route s'exécute normalement).
func TestPrestigeHandler_SquadActorGuard_AllowsOwnActor(t *testing.T) {
	onlyAlice := func(_ context.Context, slug string) bool { return slug == "alice" }
	r := newRouterGuarded(&mockPrestigeService{orientationResp: "combat"}, onlyAlice)
	req := httptest.NewRequest(http.MethodGet, "/squads/sq1/orientation?requested_by=alice", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200 (acteur autorisé); body=%s", w.Code, w.Body.String())
	}
}

func TestPrestigeHandler_ListArcPresets_OK(t *testing.T) {
	mock := &mockPrestigeService{
		listPresetsResp: []prestige.PresetArc{{ID: "p1", TitleSlug: "halo_infinite", TitleFR: "Ascension"}},
	}
	router := newRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/arcs/presets?user_id=u1&title_slug=halo_infinite", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestPrestigeHandler_ListArcPresets_MissingParams400(t *testing.T) {
	router := newRouter(&mockPrestigeService{})
	req := httptest.NewRequest(http.MethodGet, "/arcs/presets?user_id=u1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 (title_slug manquant), got %d", w.Code)
	}
}

func TestPrestigeHandler_AdoptPresetArc_Created201(t *testing.T) {
	mock := &mockPrestigeService{adoptResp: prestige.Arc{ID: "arc1", IsPreset: true, PresetID: "p1"}}
	router := newRouter(mock)

	body := `{"user_id":"u1","title_slug":"halo_infinite"}`
	req := httptest.NewRequest(http.MethodPost, "/arcs/presets/p1/adopt", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%s)", w.Code, w.Body.String())
	}
	if mock.lastAdoptID != "p1" || mock.lastAdoptUID != "u1" {
		t.Errorf("args not propagated: preset=%q uid=%q", mock.lastAdoptID, mock.lastAdoptUID)
	}
}

func TestPrestigeHandler_AdoptPresetArc_MissingUser400(t *testing.T) {
	router := newRouter(&mockPrestigeService{})
	body := `{"title_slug":"halo_infinite"}`
	req := httptest.NewRequest(http.MethodPost, "/arcs/presets/p1/adopt", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 (user_id manquant), got %d", w.Code)
	}
}

func TestPrestigeHandler_AdoptPresetArc_NotFound404(t *testing.T) {
	mock := &mockPrestigeService{adoptErr: prestige.ErrArcNotFound}
	router := newRouter(mock)

	body := `{"user_id":"u1","title_slug":"halo_infinite"}`
	req := httptest.NewRequest(http.MethodPost, "/arcs/presets/missing/adopt", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPrestigeHandler_CreateChallenge_Success(t *testing.T) {
	mock := &mockPrestigeService{
		createResp: prestige.Challenge{ID: "ch_1", UserID: "u1", Tier: prestige.TierHeroic},
	}
	router := newRouter(mock)

	body := `{"user_id":"u1","title_slug":"halo_infinite","metric":"FieldKDA","target":1.5,"window_type":"session","cadence":"weekly","eval_type":"threshold","mode":"libre"}`
	req := httptest.NewRequest(http.MethodPost, "/challenges", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%s)", w.Code, w.Body.String())
	}
	if mock.lastCreate.Metric != "FieldKDA" {
		t.Errorf("metric not propagated: %q", mock.lastCreate.Metric)
	}
	if mock.lastCreate.WindowType != prestige.WindowSession {
		t.Errorf("window_type not parsed: %q", mock.lastCreate.WindowType)
	}
}

func TestPrestigeHandler_CreateChallenge_TooEasy(t *testing.T) {
	// Erreur ErrInvalidInput contenant "stretch" → 400 challenge_too_easy
	mock := &mockPrestigeService{
		createErr: errors.New("prestige: invalid input: stretch=1.05 below min"),
	}
	router := newRouter(mock)

	body := `{"user_id":"u1","title_slug":"halo_infinite","metric":"FieldKDA","target":0.5,"window_type":"session","cadence":"weekly","eval_type":"threshold","mode":"libre"}`
	req := httptest.NewRequest(http.MethodPost, "/challenges", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPrestigeHandler_CreateChallenge_BadJSON(t *testing.T) {
	router := newRouter(&mockPrestigeService{})
	req := httptest.NewRequest(http.MethodPost, "/challenges", bytes.NewBufferString(`{not json`))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ─────────── DeleteArc (Lot A) ───────────

func TestPrestigeHandler_DeleteArc_Cascade204(t *testing.T) {
	mock := &mockPrestigeService{}
	router := newRouter(mock)

	req := httptest.NewRequest(http.MethodDelete, "/arcs/arc1?user_id=u1&objectives=delete", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d (body=%s)", w.Code, w.Body.String())
	}
	if mock.lastDeleteArcID != "arc1" || mock.lastDeleteArcUID != "u1" || !mock.lastDeleteArcCsc {
		t.Errorf("args not propagated: id=%q uid=%q cascade=%v",
			mock.lastDeleteArcID, mock.lastDeleteArcUID, mock.lastDeleteArcCsc)
	}
}

func TestPrestigeHandler_DeleteArc_Detach204(t *testing.T) {
	mock := &mockPrestigeService{}
	router := newRouter(mock)

	req := httptest.NewRequest(http.MethodDelete, "/arcs/arc1?user_id=u1&objectives=detach", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if mock.lastDeleteArcCsc {
		t.Error("detach should not cascade")
	}
}

func TestPrestigeHandler_DeleteArc_MissingObjectivesParam400(t *testing.T) {
	router := newRouter(&mockPrestigeService{})
	req := httptest.NewRequest(http.MethodDelete, "/arcs/arc1?user_id=u1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 (objectives manquant), got %d", w.Code)
	}
}

func TestPrestigeHandler_DeleteArc_MissingUser400(t *testing.T) {
	router := newRouter(&mockPrestigeService{})
	req := httptest.NewRequest(http.MethodDelete, "/arcs/arc1?objectives=delete", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 (user_id manquant), got %d", w.Code)
	}
}

func TestPrestigeHandler_DeleteArc_Forbidden403(t *testing.T) {
	mock := &mockPrestigeService{deleteArcErr: prestige.ErrForbidden}
	router := newRouter(mock)

	req := httptest.NewRequest(http.MethodDelete, "/arcs/arc1?user_id=intruder&objectives=delete", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestPrestigeHandler_DeleteArc_NotFound404(t *testing.T) {
	mock := &mockPrestigeService{deleteArcErr: prestige.ErrArcNotFound}
	router := newRouter(mock)

	req := httptest.NewRequest(http.MethodDelete, "/arcs/missing?user_id=u1&objectives=delete", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPrestigeHandler_GetChallenge_NotFound(t *testing.T) {
	mock := &mockPrestigeService{getErr: prestige.ErrChallengeNotFound}
	router := newRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/challenges/missing_id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPrestigeHandler_GetChallenge_OK(t *testing.T) {
	mock := &mockPrestigeService{
		getResp: prestige.Challenge{
			ID: "ch_1", Status: prestige.StatusActive,
			WindowType: prestige.WindowSession, Cadence: prestige.CadenceFree,
			EvalType: prestige.EvalThreshold, Mode: prestige.ModeLibre,
			DataTier: prestige.DataFull,
		},
	}
	router := newRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/challenges/ch_1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var got prestige.Challenge
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != "ch_1" {
		t.Errorf("got ID %q", got.ID)
	}
}

func TestPrestigeHandler_ListActiveChallenges(t *testing.T) {
	complete := func(id string) prestige.Challenge {
		return prestige.Challenge{
			ID: id, Status: prestige.StatusActive,
			WindowType: prestige.WindowSession, Cadence: prestige.CadenceFree,
			EvalType: prestige.EvalThreshold, Mode: prestige.ModeLibre,
			DataTier: prestige.DataFull,
		}
	}
	mock := &mockPrestigeService{
		listResp: []prestige.Challenge{complete("ch_a"), complete("ch_b")},
	}
	router := newRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/challenges?user_id=u1&title_slug=halo_infinite", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Challenges []prestige.Challenge `json:"challenges"`
		Count      int                  `json:"count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Count != 2 {
		t.Errorf("count got %d want 2", resp.Count)
	}
}

func TestPrestigeHandler_ListActiveChallenges_MissingParams(t *testing.T) {
	router := newRouter(&mockPrestigeService{})
	req := httptest.NewRequest(http.MethodGet, "/challenges", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPrestigeHandler_UpdateChallenge_NotEditable(t *testing.T) {
	mock := &mockPrestigeService{updateErr: prestige.ErrNotEditable}
	router := newRouter(mock)

	body := `{"target":1.7}`
	req := httptest.NewRequest(http.MethodPatch, "/challenges/ch_1", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestPrestigeHandler_UpdateChallenge_OK(t *testing.T) {
	target := 1.7
	mock := &mockPrestigeService{
		updateResp: prestige.Challenge{ID: "ch_1", Target: target},
	}
	router := newRouter(mock)

	body := `{"target":1.7}`
	req := httptest.NewRequest(http.MethodPatch, "/challenges/ch_1", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	if mock.lastUpdateID != "ch_1" {
		t.Errorf("ID not propagated: %q", mock.lastUpdateID)
	}
	if mock.lastUpdate.Target == nil || *mock.lastUpdate.Target != 1.7 {
		t.Errorf("target not propagated: %v", mock.lastUpdate.Target)
	}
}

func TestPrestigeHandler_AbandonChallenge_AlreadyTerminal(t *testing.T) {
	mock := &mockPrestigeService{abandonErr: prestige.ErrAlreadyTerminal}
	router := newRouter(mock)

	req := httptest.NewRequest(http.MethodDelete, "/challenges/ch_1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestPrestigeHandler_AbandonChallenge_OK(t *testing.T) {
	mock := &mockPrestigeService{}
	router := newRouter(mock)

	req := httptest.NewRequest(http.MethodDelete, "/challenges/ch_1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if mock.lastAbandonID != "ch_1" {
		t.Errorf("ID not propagated: %q", mock.lastAbandonID)
	}
}

func TestPrestigeHandler_SuggestNext(t *testing.T) {
	mock := &mockPrestigeService{
		suggestNextResp: []prestige.Template{
			{ID: "tpl_higher", LabelEN: "Push it"},
		},
	}
	router := newRouter(mock)

	req := httptest.NewRequest(http.MethodPost, "/challenges/ch_1/suggest-next", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if mock.lastSuggestNext != "ch_1" {
		t.Errorf("ID not propagated: %q", mock.lastSuggestNext)
	}
}

func TestPrestigeHandler_GetMyPrestige_PerTitle(t *testing.T) {
	mock := &mockPrestigeService{
		mePrestige: prestige.UserPrestige{UserID: "u1", TitleSlug: "halo_infinite", TotalPP: 1500, CurrentLevel: 2},
	}
	router := newRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/prestige/me?user_id=u1&title_slug=halo_infinite", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var got prestige.UserPrestige
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.TotalPP != 1500 {
		t.Errorf("got total_pp %d want 1500", got.TotalPP)
	}
}

func TestPrestigeHandler_SuggestTemplates(t *testing.T) {
	mock := &mockPrestigeService{
		suggestTplResp: []prestige.Template{{ID: "t1"}, {ID: "t2"}, {ID: "t3"}},
	}
	router := newRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/templates/suggest?user_id=u1&title_slug=halo_infinite&count=3", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ─────────── ErrDBLocked → 503 Retry-After (commit 2 db-concurrency) ───────────

// dbLockedErr simule l'erreur retournée par LazyPrestigeService quand le sync
// engine ou un autre handler tient déjà le lease player DB.
//
// On wrappe ErrDBLocked dans un fmt.Errorf comme le fait la couche dblease
// (cf. AcquireWriter), pour vérifier que le mapping handler dépend bien
// d'errors.Is et pas d'égalité directe.
func dbLockedErr() error {
	return fmt.Errorf("dblease: player lease timeout after 5s on test-path: %w", dblease.ErrDBLocked)
}

func TestPrestigeHandler_CreateChallenge_DBLocked_Returns503(t *testing.T) {
	mock := &mockPrestigeService{createErr: dbLockedErr()}
	router := newRouter(mock)

	body := `{"user_id":"u1","title_slug":"halo_infinite","metric":"FieldKDA","target":1.5,"window_type":"session","cadence":"weekly","eval_type":"threshold","mode":"libre"}`
	req := httptest.NewRequest(http.MethodPost, "/challenges", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body=%s)", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Retry-After"); got != "5" {
		t.Errorf("Retry-After header = %q, want %q", got, "5")
	}
	var body503 map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body503); err != nil {
		t.Fatalf("response body not JSON: %v", err)
	}
	if code, _ := body503["code"].(string); code != "db_busy" {
		t.Errorf("error code = %v, want db_busy", body503["code"])
	}
}

func TestPrestigeHandler_UpdateChallenge_DBLocked_Returns503(t *testing.T) {
	mock := &mockPrestigeService{updateErr: dbLockedErr()}
	router := newRouter(mock)

	body := `{"target":2.0}`
	req := httptest.NewRequest(http.MethodPatch, "/challenges/ch_42", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body=%s)", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Retry-After"); got != "5" {
		t.Errorf("Retry-After header = %q, want %q", got, "5")
	}
}

func TestPrestigeHandler_AbandonChallenge_DBLocked_Returns503(t *testing.T) {
	mock := &mockPrestigeService{abandonErr: dbLockedErr()}
	router := newRouter(mock)

	req := httptest.NewRequest(http.MethodDelete, "/challenges/ch_42", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body=%s)", w.Code, w.Body.String())
	}
}

// TestPrestigeHandler_NotFoundErrorsNotMistakenForDBLocked vérifie que les
// erreurs métier classiques (ChallengeNotFound) ne sont pas mal-mappées en 503
// — protection contre une régression du switch dans writeServiceError.
func TestPrestigeHandler_NotFoundErrorsNotMistakenForDBLocked(t *testing.T) {
	mock := &mockPrestigeService{getErr: prestige.ErrChallengeNotFound}
	router := newRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/challenges/ch_missing", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got != "" {
		t.Errorf("Retry-After should not be set on 404, got %q", got)
	}
}

// newSquadRouter étend newRouter avec les endpoints squad pour les tests
// du commit 3 (lease shared_social.duckdb).
func newSquadRouter(svc prestige.Service) *chi.Mux {
	h := NewPrestigeHandler(svc, testAppPlayers)
	r := chi.NewRouter()
	h.Mount(r)
	return r
}

func TestPrestigeHandler_CreateSquadChallenge_DBLocked_Returns503(t *testing.T) {
	mock := &mockPrestigeService{createSquadErr: dbLockedErr()}
	router := newSquadRouter(mock)

	body := `{"title_slug":"halo_infinite","mode":"collective","eval_type":"threshold","window_type":"session","target_per_member":5,"created_by":"u1"}`
	req := httptest.NewRequest(http.MethodPost, "/squads/squad_1/challenges", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body=%s)", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Retry-After"); got != "5" {
		t.Errorf("Retry-After header = %q, want %q", got, "5")
	}
}

func TestPrestigeHandler_JoinSquadChallenge_DBLocked_Returns503(t *testing.T) {
	mock := &mockPrestigeService{joinSquadErr: dbLockedErr()}
	router := newSquadRouter(mock)

	body := `{"user_id":"u1","chosen_tier":"heroic"}`
	req := httptest.NewRequest(http.MethodPost, "/squad-challenges/sc_1/join", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body=%s)", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Retry-After"); got != "5" {
		t.Errorf("Retry-After header = %q, want %q", got, "5")
	}
}
