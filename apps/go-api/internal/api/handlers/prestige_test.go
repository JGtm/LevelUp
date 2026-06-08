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

	lastCreate       prestige.CreateChallengeRequest
	lastUpdate       prestige.UpdateChallengePatch
	lastUpdateID     string
	lastAbandonID    string
	lastSuggestNext  string
	lastDeleteArcID  string
	lastDeleteArcUID string
	lastDeleteArcCsc bool
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
func (m *mockPrestigeService) EnablePilotMode(ctx context.Context, _, _ string) (prestige.PilotModeAttribution, error) {
	return prestige.PilotModeAttribution{}, nil
}
func (m *mockPrestigeService) DisablePilotMode(ctx context.Context, _, _ string) error {
	return nil
}

// ─────────── Helpers ───────────

func newRouter(svc prestige.Service) *chi.Mux {
	h := NewPrestigeHandler(svc)
	r := chi.NewRouter()
	r.Post("/challenges", h.CreateChallenge)
	r.Get("/challenges", h.ListActiveChallenges)
	r.Get("/challenges/{id}", h.GetChallenge)
	r.Patch("/challenges/{id}", h.UpdateChallenge)
	r.Delete("/challenges/{id}", h.AbandonChallenge)
	r.Post("/challenges/{id}/suggest-next", h.SuggestNext)
	r.Delete("/arcs/{id}", h.DeleteArc)
	r.Get("/prestige/me", h.GetMyPrestige)
	r.Get("/templates/suggest", h.SuggestTemplates)
	return r
}

// ─────────── Tests ───────────

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
	h := NewPrestigeHandler(svc)
	r := chi.NewRouter()
	r.Post("/squads/{squad_id}/challenges", h.CreateSquadChallenge)
	r.Post("/squad-challenges/{id}/join", h.JoinSquadChallenge)
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
