// Package handlers_test — groups_test.go : tests GroupsHandler (CRUD + invitations).
//
// La session est injectée via un cookie signé pointant sur une session pré-sauvée
// (identité Halo liée → xuid). Les routes sont montées SANS RequireAuth (testé
// ailleurs) ; le handler fait sa propre garde d'identité (401) + ownership (403).
package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/groupstore"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
)

func newGroupsRouter(t *testing.T) (*chi.Mux, *session.Store, *groupstore.GroupStore, *userstore.InviteStore) {
	t.Helper()
	dir := t.TempDir()
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	groups := groupstore.NewGroupStore(filepath.Join(dir, "groups.json"))
	invites := userstore.NewInviteStore(filepath.Join(dir, "invites.json"))
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	h := handlers.NewGroupsHandler(groups, invites, users)

	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	// Routes montées via Huma (V72-01 / H5). Le handler fait sa propre garde
	// d'identité (401) + ownership (403) ; RequireAuth n'est PAS branché ici (testé
	// ailleurs). Chemins ABSOLUS /groups (plus de trailing slash du r.Route/"/" chi).
	h.Mount(r)
	return r, sessStore, groups, invites
}

// authCookie pré-sauve une session avec identité Halo liée (xuid) et retourne le
// cookie signé correspondant.
func authCookie(t *testing.T, sessStore *session.Store, xuid, gamertag string) *http.Cookie {
	t.Helper()
	sess := sessStore.New()
	sess.LinkedHaloIdentity = &domain.HaloIdentity{XUID: xuid, Gamertag: gamertag}
	if err := sessStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}
	return &http.Cookie{Name: session.CookieName, Value: sessStore.SignCookie(sess.SessionID)}
}

func TestGroups_CreateAndList(t *testing.T) {
	r, sessStore, _, _ := newGroupsRouter(t)
	cookie := authCookie(t, sessStore, "alice-x", "Alice")

	// Create.
	req := httptest.NewRequest(http.MethodPost, "/groups", strings.NewReader(`{"name":"Famille"}`))
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", w.Code, w.Body.String())
	}
	var g domain.Group
	if err := json.Unmarshal(w.Body.Bytes(), &g); err != nil {
		t.Fatalf("decode group: %v", err)
	}
	if g.Name != "Famille" || !g.IsOwner("alice-x") {
		t.Fatalf("groupe inattendu: %+v", g)
	}

	// List → contient le groupe.
	lreq := httptest.NewRequest(http.MethodGet, "/groups", nil)
	lreq.AddCookie(cookie)
	lw := httptest.NewRecorder()
	r.ServeHTTP(lw, lreq)
	if lw.Code != http.StatusOK {
		t.Fatalf("list status = %d", lw.Code)
	}
	var list []domain.Group
	if err := json.Unmarshal(lw.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 || list[0].ID != g.ID {
		t.Fatalf("liste inattendue: %+v", list)
	}
}

func TestGroups_Unauthenticated_401(t *testing.T) {
	r, _, _, _ := newGroupsRouter(t)
	// Aucun cookie → session anonyme (pas d'identité) → 401.
	req := httptest.NewRequest(http.MethodPost, "/groups", strings.NewReader(`{"name":"X"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (identity_required)", w.Code)
	}
}

func TestGroups_InviteMemberVsStranger(t *testing.T) {
	r, sessStore, _, invites := newGroupsRouter(t)
	owner := authCookie(t, sessStore, "alice-x", "Alice")
	stranger := authCookie(t, sessStore, "bob-x", "Bob")

	// Alice crée un groupe.
	creq := httptest.NewRequest(http.MethodPost, "/groups", strings.NewReader(`{"name":"Fam"}`))
	creq.AddCookie(owner)
	cw := httptest.NewRecorder()
	r.ServeHTTP(cw, creq)
	var g domain.Group
	_ = json.Unmarshal(cw.Body.Bytes(), &g)

	// Étranger → 403.
	sreq := httptest.NewRequest(http.MethodPost, "/groups/"+g.ID+"/invites", nil)
	sreq.AddCookie(stranger)
	sw := httptest.NewRecorder()
	r.ServeHTTP(sw, sreq)
	if sw.Code != http.StatusForbidden {
		t.Fatalf("stranger invite status = %d, want 403", sw.Code)
	}

	// Membre (owner) → 201 + invite portant le GroupID.
	mreq := httptest.NewRequest(http.MethodPost, "/groups/"+g.ID+"/invites", nil)
	mreq.AddCookie(owner)
	mw := httptest.NewRecorder()
	r.ServeHTTP(mw, mreq)
	if mw.Code != http.StatusCreated {
		t.Fatalf("owner invite status = %d, body = %s", mw.Code, mw.Body.String())
	}
	var inv domain.InviteCode
	_ = json.Unmarshal(mw.Body.Bytes(), &inv)
	if inv.GroupID != g.ID {
		t.Fatalf("invite.GroupID = %q, want %q", inv.GroupID, g.ID)
	}
	// Le code est bien validable dans le store.
	if err := invites.Validate(inv.Code); err != nil {
		t.Fatalf("invite générée invalide: %v", err)
	}
}

func TestGroups_OwnerOnlyDelete(t *testing.T) {
	r, sessStore, _, _ := newGroupsRouter(t)
	owner := authCookie(t, sessStore, "alice-x", "Alice")
	stranger := authCookie(t, sessStore, "bob-x", "Bob")

	creq := httptest.NewRequest(http.MethodPost, "/groups", strings.NewReader(`{"name":"Fam"}`))
	creq.AddCookie(owner)
	cw := httptest.NewRecorder()
	r.ServeHTTP(cw, creq)
	var g domain.Group
	_ = json.Unmarshal(cw.Body.Bytes(), &g)

	// Étranger ne peut pas supprimer → 403.
	dreq := httptest.NewRequest(http.MethodDelete, "/groups/"+g.ID, nil)
	dreq.AddCookie(stranger)
	dw := httptest.NewRecorder()
	r.ServeHTTP(dw, dreq)
	if dw.Code != http.StatusForbidden {
		t.Fatalf("stranger delete status = %d, want 403", dw.Code)
	}

	// Propriétaire supprime → 204.
	oreq := httptest.NewRequest(http.MethodDelete, "/groups/"+g.ID, nil)
	oreq.AddCookie(owner)
	ow := httptest.NewRecorder()
	r.ServeHTTP(ow, oreq)
	if ow.Code != http.StatusNoContent {
		t.Fatalf("owner delete status = %d, want 204", ow.Code)
	}
}

func TestGroups_RemoveAndLeaveMember(t *testing.T) {
	r, sessStore, groups, _ := newGroupsRouter(t)
	owner := authCookie(t, sessStore, "alice-x", "Alice")
	bob := authCookie(t, sessStore, "bob-x", "Bob")

	// Alice crée un groupe, Bob en devient membre (ajout direct via le store).
	creq := httptest.NewRequest(http.MethodPost, "/groups", strings.NewReader(`{"name":"Fam"}`))
	creq.AddCookie(owner)
	cw := httptest.NewRecorder()
	r.ServeHTTP(cw, creq)
	var g domain.Group
	_ = json.Unmarshal(cw.Body.Bytes(), &g)
	if err := groups.AddMember(g.ID, "bob-x", "Bob"); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	// Le propriétaire ne peut pas quitter → 409.
	leaveOwner := httptest.NewRequest(http.MethodDelete, "/groups/"+g.ID+"/members/me", nil)
	leaveOwner.AddCookie(owner)
	low := httptest.NewRecorder()
	r.ServeHTTP(low, leaveOwner)
	if low.Code != http.StatusConflict {
		t.Fatalf("owner leave status = %d, want 409", low.Code)
	}

	// Bob quitte (self) → 204.
	leaveBob := httptest.NewRequest(http.MethodDelete, "/groups/"+g.ID+"/members/me", nil)
	leaveBob.AddCookie(bob)
	lbw := httptest.NewRecorder()
	r.ServeHTTP(lbw, leaveBob)
	if lbw.Code != http.StatusNoContent {
		t.Fatalf("bob leave status = %d, want 204", lbw.Code)
	}
	if got, _ := groups.Get(g.ID); got.HasMember("bob-x") {
		t.Fatal("bob aurait dû quitter le groupe")
	}

	// Ré-ajout de Bob, puis retrait par le propriétaire → 204.
	_ = groups.AddMember(g.ID, "bob-x", "Bob")
	rem := httptest.NewRequest(http.MethodDelete, "/groups/"+g.ID+"/members/bob-x", nil)
	rem.AddCookie(owner)
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, rem)
	if rw.Code != http.StatusNoContent {
		t.Fatalf("owner remove member status = %d, want 204", rw.Code)
	}

	// Un étranger ne peut pas retirer (pas propriétaire) → 403.
	_ = groups.AddMember(g.ID, "bob-x", "Bob")
	stRem := httptest.NewRequest(http.MethodDelete, "/groups/"+g.ID+"/members/bob-x", nil)
	stRem.AddCookie(bob) // bob = membre mais pas propriétaire
	stw := httptest.NewRecorder()
	r.ServeHTTP(stw, stRem)
	if stw.Code != http.StatusForbidden {
		t.Fatalf("non-owner remove status = %d, want 403", stw.Code)
	}
}
