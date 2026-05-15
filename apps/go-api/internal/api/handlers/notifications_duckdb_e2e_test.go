// Package handlers_test — notifications_duckdb_e2e_test.go : tests HTTP
// E2E des endpoints /notifications avec un VRAI NotificationsRepo DuckDB.
//
// Les tests `notifications_test.go` utilisent un fakeRepo en mémoire qui ne
// reproduit pas les contraintes DuckDB (PK, indexes secondaires, NULL handling).
// Ces tests-ci couvrent le chemin exact du log d'erreur 2026-05-14 :
//
//   handler.List → svc.List → repo.List → DuckDB shared_social
//
// Ils valident que les opérations HTTP qui ont déclenché le bug ART/NULL
// (MarkRead, MarkAllRead, Delete) sont saines post-migration.

package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
)

// newDuckDBServiceForHandler construit un (Service, factory) adossé à un vrai
// NotificationsRepo DuckDB pour un slug donné.
func newDuckDBServiceForHandler(t *testing.T, slug string) (*notifications.Service, func(context.Context, string) (*notifications.Service, error)) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shared_social.duckdb")
	rw, err := duckdb.OpenReadWrite(dbPath)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	if err := migration.RunForDB(rw.SQLDb(), migration.TargetSharedSocial); err != nil {
		rw.Close()
		t.Fatalf("RunForDB(TargetSharedSocial): %v", err)
	}
	t.Cleanup(func() { rw.Close() })

	pdb := &duckdb.PlayerDB{
		SharedSocial: rw,
		XUID:         "xuid-" + slug,
		Gamertag:     slug,
	}
	repo := duckdb.NewNotificationsRepo(pdb)
	svc := notifications.NewService(repo)

	factory := func(_ context.Context, requestedSlug string) (*notifications.Service, error) {
		if requestedSlug != slug {
			return nil, fmt.Errorf("player not found: %s", requestedSlug)
		}
		return svc, nil
	}
	return svc, factory
}

// seedNotif insère N notifs non-lues via le service (donc IDs auto-générés).
func seedNotifs(t *testing.T, svc *notifications.Service, cat notifications.Category, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := svc.Emit(context.Background(), notifications.EmitInput{
			Category: cat,
			Severity: notifications.SeverityWarn,
			TitleKey: "notif." + string(cat) + ".title",
			Source:   "test_seed",
		}); err != nil {
			t.Fatalf("Emit seed #%d: %v", i, err)
		}
	}
}

// ─── 1. GET /notifications — chemin exact du log d'erreur ────────────────────

// TestNotificationsHandler_DuckDB_GetList_ReturnsAllAfterMarkOps reproduit la
// séquence du log de prod : émission de notifs data_health_warning, MarkAllRead,
// puis GET /notifications. C'est sur ce dernier appel que le log montrait
// « database has been invalidated ».
func TestNotificationsHandler_DuckDB_GetList_ReturnsAllAfterMarkOps(t *testing.T) {
	svc, factory := newDuckDBServiceForHandler(t, "p1")
	seedNotifs(t, svc, notifications.CategoryDataHealthWarning, 3)
	r := newNotificationsRouter(factory)

	// Étape 1 : GET /notifications — la requête qui crashait en prod.
	req := httptest.NewRequest(http.MethodGet, "/players/p1/notifications", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET list initial: status %d, body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Items []notifications.Notification `json:"items"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 3 {
		t.Errorf("attendu 3 items, obtenu %d", len(body.Items))
	}

	// Étape 2 : POST /mark-all-read avec category=data_health_warning — le
	// scénario UI qui touchait l'ancien idx_pn_xuid_unread.
	mar := strings.NewReader(`{"category":"data_health_warning"}`)
	req = httptest.NewRequest(http.MethodPost, "/players/p1/notifications/mark-all-read", mar)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST mark-all-read: status %d, body=%s", w.Code, w.Body.String())
	}

	// Étape 3 : GET /notifications à nouveau — c'est exactement la séquence du
	// log de prod. Doit retourner 200 OK.
	req = httptest.NewRequest(http.MethodGet, "/players/p1/notifications", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET list après mark-all-read: status %d (régression bug ART/NULL ?), body=%s",
			w.Code, w.Body.String())
	}

	// Étape 4 : GET /unread-count doit retourner 0.
	req = httptest.NewRequest(http.MethodGet, "/players/p1/notifications/unread-count", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET unread-count: status %d", w.Code)
	}
	var count struct {
		Count int `json:"count"`
	}
	_ = json.NewDecoder(w.Body).Decode(&count)
	if count.Count != 0 {
		t.Errorf("unread-count: attendu 0, obtenu %d", count.Count)
	}
}

// ─── 2. POST /mark-read avec IDs : UPDATE bulk sur read_at NULL ─────────────

func TestNotificationsHandler_DuckDB_PostMarkRead_WithIDs(t *testing.T) {
	svc, factory := newDuckDBServiceForHandler(t, "p1")
	seedNotifs(t, svc, notifications.CategoryDataHealthWarning, 5)
	r := newNotificationsRouter(factory)

	// Récupère les IDs réels (auto-générés par le service).
	list, _ := svc.List(context.Background(), notifications.ListFilter{Limit: 10})
	ids := []int64{list.Items[0].ID, list.Items[1].ID}
	idsJSON, _ := json.Marshal(struct {
		IDs []int64 `json:"ids"`
	}{IDs: ids})

	req := httptest.NewRequest(http.MethodPost,
		"/players/p1/notifications/mark-read", bytes.NewReader(idsJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST mark-read: status %d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Updated int `json:"updated"`
	}
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Updated != 2 {
		t.Errorf("updated: attendu 2, obtenu %d", resp.Updated)
	}

	// La connexion DB doit rester valide pour la suite.
	req = httptest.NewRequest(http.MethodGet,
		"/players/p1/notifications?unread_only=true", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET ?unread_only=true: status %d", w.Code)
	}
	var body struct {
		Items []notifications.Notification `json:"items"`
	}
	_ = json.NewDecoder(w.Body).Decode(&body)
	if len(body.Items) != 3 {
		t.Errorf("attendu 3 non-lues restantes, obtenu %d", len(body.Items))
	}
}

// ─── 3. DELETE /notifications/{id} sur ligne read_at NULL ────────────────────

func TestNotificationsHandler_DuckDB_DeleteUnread_StaysHealthy(t *testing.T) {
	svc, factory := newDuckDBServiceForHandler(t, "p1")
	seedNotifs(t, svc, notifications.CategoryDataHealthWarning, 3)
	r := newNotificationsRouter(factory)

	list, _ := svc.List(context.Background(), notifications.ListFilter{Limit: 10})
	if len(list.Items) != 3 {
		t.Fatalf("seed: attendu 3, obtenu %d", len(list.Items))
	}
	targetID := list.Items[0].ID

	// DELETE
	req := httptest.NewRequest(http.MethodDelete,
		"/players/p1/notifications/"+strconv.FormatInt(targetID, 10), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE: status %d, body=%s", w.Code, w.Body.String())
	}

	// DELETE inconnu → 404
	req = httptest.NewRequest(http.MethodDelete,
		"/players/p1/notifications/99999999", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("DELETE id inconnu: attendu 404, obtenu %d", w.Code)
	}

	// GET final : la DB doit toujours répondre.
	req = httptest.NewRequest(http.MethodGet, "/players/p1/notifications", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET final: status %d (régression ?)", w.Code)
	}
	var body struct {
		Items []notifications.Notification `json:"items"`
	}
	_ = json.NewDecoder(w.Body).Decode(&body)
	if len(body.Items) != 2 {
		t.Errorf("attendu 2 items après DELETE, obtenu %d", len(body.Items))
	}
}

// ─── 4. PATCH /notifications/{id}/unread (UPDATE read_at → NULL) ────────────

func TestNotificationsHandler_DuckDB_PatchMarkUnread(t *testing.T) {
	svc, factory := newDuckDBServiceForHandler(t, "p1")
	seedNotifs(t, svc, notifications.CategoryDataHealthWarning, 2)
	r := newNotificationsRouter(factory)

	list, _ := svc.List(context.Background(), notifications.ListFilter{Limit: 10})
	id := list.Items[0].ID

	// MarkRead d'abord.
	mrBody, _ := json.Marshal(struct {
		IDs []int64 `json:"ids"`
	}{IDs: []int64{id}})
	req := httptest.NewRequest(http.MethodPost,
		"/players/p1/notifications/mark-read", bytes.NewReader(mrBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("mark-read: status %d", w.Code)
	}

	// PATCH /unread — UPDATE read_at → NULL (le chemin inverse, sensible aussi).
	req = httptest.NewRequest(http.MethodPatch,
		"/players/p1/notifications/"+strconv.FormatInt(id, 10)+"/unread", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("PATCH unread: status %d, body=%s", w.Code, w.Body.String())
	}

	// unread-count doit valoir 2 (les 2 sont non-lues).
	req = httptest.NewRequest(http.MethodGet,
		"/players/p1/notifications/unread-count", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var count struct {
		Count int `json:"count"`
	}
	_ = json.NewDecoder(w.Body).Decode(&count)
	if count.Count != 2 {
		t.Errorf("unread-count après PATCH unread: attendu 2, obtenu %d", count.Count)
	}
}

// ─── 5. Full HTTP cycle data_health_warning (scénario log de prod) ──────────

// TestNotificationsHandler_DuckDB_DataHealthWarning_HTTPFullCycle simule la
// séquence HTTP exacte qu'un user déclencherait pour interagir avec une notif
// data_health_warning : GET → POST mark-all-read → GET → DELETE → GET.
//
// C'est le plus haut niveau de validation : si ce test passe, la pipeline
// frontend → API → service → repo → DuckDB est saine bout-en-bout.
func TestNotificationsHandler_DuckDB_DataHealthWarning_HTTPFullCycle(t *testing.T) {
	svc, factory := newDuckDBServiceForHandler(t, "main_player")
	r := newNotificationsRouter(factory)

	// Simule 3 cycles HealthScheduler en émettant directement via le service.
	seedNotifs(t, svc, notifications.CategoryDataHealthWarning, 3)

	// 1. GET /notifications?category=data_health_warning
	req := httptest.NewRequest(http.MethodGet,
		"/players/main_player/notifications?category=data_health_warning", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("step1 GET: %d", w.Code)
	}

	// 2. POST /mark-all-read avec category=data_health_warning
	body := strings.NewReader(`{"category":"data_health_warning"}`)
	req = httptest.NewRequest(http.MethodPost,
		"/players/main_player/notifications/mark-all-read", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("step2 mark-all-read: %d, body=%s", w.Code, w.Body.String())
	}
	var mar struct {
		Updated int `json:"updated"`
	}
	_ = json.NewDecoder(w.Body).Decode(&mar)
	if mar.Updated != 3 {
		t.Errorf("updated: %d", mar.Updated)
	}

	// 3. GET /unread-count → 0
	req = httptest.NewRequest(http.MethodGet,
		"/players/main_player/notifications/unread-count", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var count struct {
		Count int `json:"count"`
	}
	_ = json.NewDecoder(w.Body).Decode(&count)
	if count.Count != 0 {
		t.Errorf("step3 unread-count: attendu 0, obtenu %d", count.Count)
	}

	// 4. DELETE de la 1ère notif lue
	list, _ := svc.List(context.Background(), notifications.ListFilter{Limit: 10})
	if len(list.Items) == 0 {
		t.Fatal("step4: list vide après cycle, attendu 3")
	}
	id := list.Items[0].ID
	req = httptest.NewRequest(http.MethodDelete,
		"/players/main_player/notifications/"+strconv.FormatInt(id, 10), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("step4 DELETE: %d", w.Code)
	}

	// 5. GET final — la pipeline doit avoir survécu à toutes les opérations.
	req = httptest.NewRequest(http.MethodGet, "/players/main_player/notifications", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("step5 GET final: %d (pipeline cassée)", w.Code)
	}
	var final struct {
		Items []notifications.Notification `json:"items"`
	}
	_ = json.NewDecoder(w.Body).Decode(&final)
	if len(final.Items) != 2 {
		t.Errorf("step5: attendu 2 items restants, obtenu %d", len(final.Items))
	}
}
