// Package handlers_test — notifications_test.go : tests du NotificationsHandler.
//
// Stratégie : fake Repository in-memory pour piloter le Service réel à travers
// le handler. Couvre les 8 endpoints (List, UnreadCount, MarkRead, MarkAllRead,
// MarkUnread, Delete, GetPreferences, UpdatePreferences) + cas d'erreur.
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/dblease"
)

// fakeNotifRepo : impl Repository en mémoire pour les tests.
type fakeNotifRepo struct {
	items    []notifications.Notification
	prefs    map[notifications.Category]notifications.Preference
	listErr  error
	insertOK func(*notifications.Notification) error
}

func newFakeNotifRepo() *fakeNotifRepo {
	return &fakeNotifRepo{
		prefs: map[notifications.Category]notifications.Preference{},
	}
}

func (r *fakeNotifRepo) Insert(_ context.Context, n *notifications.Notification) error {
	if r.insertOK != nil {
		if err := r.insertOK(n); err != nil {
			return err
		}
	}
	r.items = append([]notifications.Notification{*n}, r.items...)
	return nil
}

func (r *fakeNotifRepo) List(_ context.Context, f notifications.ListFilter) (notifications.ListResult, error) {
	if r.listErr != nil {
		return notifications.ListResult{}, r.listErr
	}
	out := []notifications.Notification{}
	for _, it := range r.items {
		if f.UnreadOnly && it.ReadAt != nil {
			continue
		}
		if f.Category != "" && it.Category != f.Category {
			continue
		}
		if f.BeforeID > 0 && it.ID >= f.BeforeID {
			continue
		}
		out = append(out, it)
		if f.Limit > 0 && len(out) >= f.Limit {
			break
		}
	}
	res := notifications.ListResult{Items: out}
	if f.Limit > 0 && len(out) == f.Limit {
		last := out[len(out)-1].ID
		res.NextCursor = &last
	}
	return res, nil
}

func (r *fakeNotifRepo) UnreadCount(_ context.Context) (notifications.UnreadCount, error) {
	out := notifications.UnreadCount{ByCategory: map[string]int{}}
	for _, it := range r.items {
		if it.ReadAt != nil {
			continue
		}
		out.Count++
		out.ByCategory[string(it.Category)]++
	}
	return out, nil
}

func (r *fakeNotifRepo) MarkRead(_ context.Context, ids []int64) (int, error) {
	idSet := map[int64]bool{}
	for _, id := range ids {
		idSet[id] = true
	}
	n := 0
	for i := range r.items {
		if idSet[r.items[i].ID] && r.items[i].ReadAt == nil {
			t := timeNowForTest()
			r.items[i].ReadAt = &t
			n++
		}
	}
	return n, nil
}

func (r *fakeNotifRepo) MarkUnread(_ context.Context, id int64) error {
	for i := range r.items {
		if r.items[i].ID == id {
			r.items[i].ReadAt = nil
			return nil
		}
	}
	return notifications.ErrNotFound
}

func (r *fakeNotifRepo) MarkAllRead(_ context.Context, cat notifications.Category) (int, error) {
	n := 0
	for i := range r.items {
		if r.items[i].ReadAt != nil {
			continue
		}
		if cat != "" && r.items[i].Category != cat {
			continue
		}
		t := timeNowForTest()
		r.items[i].ReadAt = &t
		n++
	}
	return n, nil
}

func (r *fakeNotifRepo) Delete(_ context.Context, id int64) error {
	for i, it := range r.items {
		if it.ID == id {
			r.items = append(r.items[:i], r.items[i+1:]...)
			return nil
		}
	}
	return notifications.ErrNotFound
}

func (r *fakeNotifRepo) CapAndSweep(_ context.Context, _ int) error              { return nil }
func (r *fakeNotifRepo) SweepStaleInfoRead(_ context.Context, _ time.Time) error { return nil }

func (r *fakeNotifRepo) GetPreferences(_ context.Context) ([]notifications.Preference, error) {
	out := []notifications.Preference{}
	for _, p := range r.prefs {
		out = append(out, p)
	}
	return out, nil
}

func (r *fakeNotifRepo) UpsertPreferences(_ context.Context, prefs []notifications.Preference) error {
	for _, p := range prefs {
		r.prefs[p.Category] = p
	}
	return nil
}

func (r *fakeNotifRepo) IsCategoryEnabled(_ context.Context, c notifications.Category) (bool, error) {
	if p, ok := r.prefs[c]; ok {
		return p.Enabled, nil
	}
	return true, nil
}

// timeNowForTest fournit un timestamp stable pour les assertions.
func timeNowForTest() time.Time { return time.Now().UTC() }

// Helpers — routeur de test.
func newNotificationsRouter(factory handlers.NotificationsServiceFactory) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewNotificationsHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})
	return r
}

func makeFactory(svc *notifications.Service, slug string) handlers.NotificationsServiceFactory {
	return func(_ context.Context, s string) (*notifications.Service, error) {
		if s != slug {
			return nil, errors.New("player_not_found")
		}
		return svc, nil
	}
}

// ─── Tests ────────────────────────────────────────────────────────────────

func TestNotificationsHandler_List_Empty(t *testing.T) {
	repo := newFakeNotifRepo()
	svc := notifications.NewService(repo)
	r := newNotificationsRouter(makeFactory(svc, "p1"))

	req := httptest.NewRequest(http.MethodGet, "/players/p1/notifications", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body struct {
		Items []notifications.Notification `json:"items"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 0 {
		t.Errorf("expected empty items, got %d", len(body.Items))
	}
}

func TestNotificationsHandler_List_PlayerNotFound(t *testing.T) {
	repo := newFakeNotifRepo()
	svc := notifications.NewService(repo)
	r := newNotificationsRouter(makeFactory(svc, "p1"))

	req := httptest.NewRequest(http.MethodGet, "/players/unknown/notifications", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestNotificationsHandler_UnreadCount(t *testing.T) {
	repo := newFakeNotifRepo()
	svc := notifications.NewService(repo)
	// Émet 2 notifs (toutes non lues par défaut)
	_ = svc.Emit(context.Background(), notifications.EmitInput{
		Category: notifications.CategoryMatchSynced, TitleKey: "k", Source: "s",
	})
	_ = svc.Emit(context.Background(), notifications.EmitInput{
		Category: notifications.CategoryMediaAdded, TitleKey: "k", Source: "s",
	})

	r := newNotificationsRouter(makeFactory(svc, "p1"))
	req := httptest.NewRequest(http.MethodGet, "/players/p1/notifications/unread-count", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body notifications.UnreadCount
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body.Count != 2 {
		t.Errorf("expected count=2, got %d", body.Count)
	}
}

func TestNotificationsHandler_MarkRead(t *testing.T) {
	repo := newFakeNotifRepo()
	svc := notifications.NewService(repo)
	_ = svc.Emit(context.Background(), notifications.EmitInput{
		Category: notifications.CategoryMatchSynced, TitleKey: "k", Source: "s",
	})
	id := repo.items[0].ID

	r := newNotificationsRouter(makeFactory(svc, "p1"))
	body, _ := json.Marshal(map[string][]int64{"ids": {id}})
	req := httptest.NewRequest(http.MethodPost, "/players/p1/notifications/mark-read", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var res notifications.MarkResult
	_ = json.NewDecoder(w.Body).Decode(&res)
	if res.Updated != 1 {
		t.Errorf("expected updated=1, got %d", res.Updated)
	}
	if repo.items[0].ReadAt == nil {
		t.Error("expected item to be marked as read")
	}
}

func TestNotificationsHandler_MarkRead_EmptyIDs(t *testing.T) {
	repo := newFakeNotifRepo()
	svc := notifications.NewService(repo)
	r := newNotificationsRouter(makeFactory(svc, "p1"))

	body, _ := json.Marshal(map[string][]int64{"ids": {}})
	req := httptest.NewRequest(http.MethodPost, "/players/p1/notifications/mark-read", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestNotificationsHandler_MarkUnread_NotFound(t *testing.T) {
	repo := newFakeNotifRepo()
	svc := notifications.NewService(repo)
	r := newNotificationsRouter(makeFactory(svc, "p1"))

	req := httptest.NewRequest(http.MethodPatch, "/players/p1/notifications/999/unread", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown id, got %d", w.Code)
	}
}

func TestNotificationsHandler_MarkUnread_BadID(t *testing.T) {
	repo := newFakeNotifRepo()
	svc := notifications.NewService(repo)
	r := newNotificationsRouter(makeFactory(svc, "p1"))

	req := httptest.NewRequest(http.MethodPatch, "/players/p1/notifications/abc/unread", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid id, got %d", w.Code)
	}
}

func TestNotificationsHandler_MarkAllRead(t *testing.T) {
	repo := newFakeNotifRepo()
	svc := notifications.NewService(repo)
	for i := 0; i < 3; i++ {
		_ = svc.Emit(context.Background(), notifications.EmitInput{
			Category: notifications.CategoryMatchSynced, TitleKey: "k", Source: "s",
		})
	}

	r := newNotificationsRouter(makeFactory(svc, "p1"))
	req := httptest.NewRequest(http.MethodPost, "/players/p1/notifications/mark-all-read", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var res notifications.MarkResult
	_ = json.NewDecoder(w.Body).Decode(&res)
	if res.Updated != 3 {
		t.Errorf("expected 3 updated, got %d", res.Updated)
	}
}

func TestNotificationsHandler_Delete(t *testing.T) {
	repo := newFakeNotifRepo()
	svc := notifications.NewService(repo)
	_ = svc.Emit(context.Background(), notifications.EmitInput{
		Category: notifications.CategoryMatchSynced, TitleKey: "k", Source: "s",
	})
	id := repo.items[0].ID

	r := newNotificationsRouter(makeFactory(svc, "p1"))
	req := httptest.NewRequest(http.MethodDelete, "/players/p1/notifications/"+i64Str(id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if len(repo.items) != 0 {
		t.Errorf("expected items deleted, got %d", len(repo.items))
	}
}

func TestNotificationsHandler_Preferences_Roundtrip(t *testing.T) {
	repo := newFakeNotifRepo()
	svc := notifications.NewService(repo)
	r := newNotificationsRouter(makeFactory(svc, "p1"))

	// PATCH
	prefs := []notifications.Preference{
		{Category: notifications.CategoryMatchSynced, Enabled: false, Delivery: notifications.DeliveryOff},
	}
	body, _ := json.Marshal(map[string]any{"items": prefs})
	req := httptest.NewRequest(http.MethodPatch, "/players/p1/notifications/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}

	// GET
	req2 := httptest.NewRequest(http.MethodGet, "/players/p1/notifications/preferences", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d", w2.Code)
	}
	var body2 struct {
		Items []notifications.Preference `json:"items"`
	}
	_ = json.NewDecoder(w2.Body).Decode(&body2)
	if len(body2.Items) != 1 || body2.Items[0].Enabled {
		t.Errorf("expected 1 disabled pref, got %+v", body2.Items)
	}
}

func TestNotificationsHandler_PostTest_Emits(t *testing.T) {
	repo := newFakeNotifRepo()
	svc := notifications.NewService(repo)
	r := newNotificationsRouter(makeFactory(svc, "p1"))

	req := httptest.NewRequest(http.MethodPost, "/players/p1/notifications/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("PostTest expected 204, got %d (body=%s)", w.Code, w.Body.String())
	}
	if len(repo.items) != 1 {
		t.Fatalf("expected 1 emitted notif, got %d", len(repo.items))
	}
	if repo.items[0].Category != notifications.CategoryAppRelease {
		t.Errorf("expected category app_release, got %q", repo.items[0].Category)
	}
	if repo.items[0].Source != "test_button" {
		t.Errorf("expected source=test_button, got %q", repo.items[0].Source)
	}
}

func TestNotificationsHandler_PostTest_PlayerNotFound(t *testing.T) {
	repo := newFakeNotifRepo()
	svc := notifications.NewService(repo)
	r := newNotificationsRouter(makeFactory(svc, "p1"))

	req := httptest.NewRequest(http.MethodPost, "/players/unknown/notifications/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown player, got %d", w.Code)
	}
}

func TestNotificationsHandler_BadJSON(t *testing.T) {
	repo := newFakeNotifRepo()
	svc := notifications.NewService(repo)
	r := newNotificationsRouter(makeFactory(svc, "p1"))

	req := httptest.NewRequest(http.MethodPost, "/players/p1/notifications/mark-read",
		bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = 8
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d", w.Code)
	}
}

func i64Str(i int64) string {
	return strconv.FormatInt(i, 10)
}

// ─── ErrDBLocked → 503 (commit 4 db-concurrency) ───

// dbLockedAcquirer simule un *LeasedWriter saturé en retournant directement
// une erreur qui wrap ErrDBLocked, comme le ferait dblease.AcquireWriter en
// situation de timeout réel.
func dbLockedAcquirer() (*dblease.LeasedWriter, error) {
	return nil, fmt.Errorf("simulated lease timeout: %w", dblease.ErrDBLocked)
}

func TestNotificationsHandler_MarkRead_DBLocked_Returns503(t *testing.T) {
	repo := newFakeNotifRepo()
	svc := notifications.NewService(repo, notifications.WithWriterAcquirer(dbLockedAcquirer))
	r := newNotificationsRouter(makeFactory(svc, "p1"))

	body := `{"ids":[1,2,3]}`
	req := httptest.NewRequest(http.MethodPost, "/players/p1/notifications/mark-read",
		bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

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

func TestNotificationsHandler_Delete_DBLocked_Returns503(t *testing.T) {
	repo := newFakeNotifRepo()
	svc := notifications.NewService(repo, notifications.WithWriterAcquirer(dbLockedAcquirer))
	r := newNotificationsRouter(makeFactory(svc, "p1"))

	req := httptest.NewRequest(http.MethodDelete, "/players/p1/notifications/42", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body=%s)", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Retry-After"); got != "5" {
		t.Errorf("Retry-After header = %q, want %q", got, "5")
	}
}

func TestNotificationsHandler_UpdatePreferences_DBLocked_Returns503(t *testing.T) {
	repo := newFakeNotifRepo()
	svc := notifications.NewService(repo, notifications.WithWriterAcquirer(dbLockedAcquirer))
	r := newNotificationsRouter(makeFactory(svc, "p1"))

	body := `{"items":[{"category":"match_synced","enabled":true,"delivery":"toast"}]}`
	req := httptest.NewRequest(http.MethodPatch, "/players/p1/notifications/preferences",
		bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body=%s)", w.Code, w.Body.String())
	}
}

// TestNotificationsHandler_Emit_BestEffort_NoErrorOnLeaseBusy vérifie que
// PostTest (qui appelle Emit) retourne 204 même si le lease est saturé —
// preuve que le contrat best-effort de Emit est préservé même via HTTP.
func TestNotificationsHandler_Emit_BestEffort_NoErrorOnLeaseBusy(t *testing.T) {
	repo := newFakeNotifRepo()
	repo.prefs[notifications.CategoryAppRelease] = notifications.Preference{
		Category: notifications.CategoryAppRelease, Enabled: true, Delivery: notifications.DeliveryToast,
	}
	svc := notifications.NewService(repo, notifications.WithWriterAcquirer(dbLockedAcquirer))
	r := newNotificationsRouter(makeFactory(svc, "p1"))

	req := httptest.NewRequest(http.MethodPost, "/players/p1/notifications/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("PostTest should return 204 even on lease busy (best-effort), got %d (body=%s)",
			w.Code, w.Body.String())
	}
}
