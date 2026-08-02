package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// deleteMedia exécute un DELETE sur la route de suppression, avec une session
// optionnelle injectée dans le contexte (nil = aucune session).
func deleteMedia(r chi.Router, urlPath string, sess *domain.SessionData) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete,
		"/players/"+testPlayerSlug+"/media?file_path="+url.QueryEscape(urlPath), nil)
	if sess != nil {
		req = req.WithContext(middleware.InjectSession(req.Context(), sess))
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// sessionFor construit une session portant un joueur courant et un rôle.
func sessionFor(playerSlug, role string) *domain.SessionData {
	sess := &domain.SessionData{SessionID: "sess-delete"}
	if playerSlug != "" {
		sess.CurrentPlayerSlug = &playerSlug
	}
	if role != "" {
		sess.Role = &role
	}
	return sess
}

// newDeleteRouter monte le handler média avec auth appliquée (multi-utilisateur).
func newDeleteRouter(mock *mockMediaService, authEnforced bool) *chi.Mux {
	factory := func(_ context.Context, slug string) (port.MediaService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := chi.NewRouter()
	h := handlers.NewMediaHandler(factory, nil, "").WithAuthEnforced(authEnforced)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})
	return r
}

// TestMediaHandler_DeleteMedia_Owner_200 : le propriétaire supprime son média.
// Vérifie aussi les deux contrats de chemin : le service reçoit le chemin
// STOCKÉ, le client reçoit en écho l'URL servable qu'il a envoyée (son cache est
// indexé par cette URL — cf. régression de l'item 1.5).
func TestMediaHandler_DeleteMedia_Owner_200(t *testing.T) {
	mock := &mockMediaService{}
	r := newDeleteRouter(mock, true)

	const urlPath = "/api/v1/players/" + testPlayerSlug + "/media/files/JGtm/clip.mp4"
	w := deleteMedia(r, urlPath, sessionFor("JGtm", "user"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if mock.delReq == nil {
		t.Fatal("DeleteMedia n'a pas été appelé")
	}
	if mock.delReq.FilePath != "JGtm/clip.mp4" {
		t.Errorf("service FilePath = %q, want %q (chemin stocké)", mock.delReq.FilePath, "JGtm/clip.mp4")
	}
	if mock.delReq.RequesterSlug != "JGtm" {
		t.Errorf("RequesterSlug = %q, want JGtm", mock.delReq.RequesterSlug)
	}
	if mock.delReq.RequesterIsAdmin {
		t.Error("RequesterIsAdmin = true pour un rôle 'user'")
	}
	if !mock.delReq.AuthEnforced {
		t.Error("AuthEnforced = false alors que le handler est en mode authentifié")
	}

	var resp domain.MediaDeleteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.FilePath != urlPath {
		t.Errorf("réponse file_path = %q, want %q (écho de la requête)", resp.FilePath, urlPath)
	}
	if !resp.Deleted {
		t.Error("réponse deleted=false")
	}
}

// TestMediaHandler_DeleteMedia_NoSession_401 : garde anti-silence. Sans joueur
// courant, l'échec doit dire « reconnecte-toi » (401), pas « tu n'as pas le
// droit » (403) — deux actions correctives différentes pour l'utilisateur.
func TestMediaHandler_DeleteMedia_NoSession_401(t *testing.T) {
	mock := &mockMediaService{}
	r := newDeleteRouter(mock, true)

	w := deleteMedia(r, "/api/v1/players/"+testPlayerSlug+"/media/files/JGtm/clip.mp4", nil)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if mock.delReq != nil {
		t.Error("DeleteMedia ne doit PAS être appelé sans identité opposable")
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("delete_requires_session")) {
		t.Errorf("corps sans code delete_requires_session : %s", w.Body.String())
	}
}

// TestMediaHandler_DeleteMedia_NotOwner_403 : un utilisateur authentifié qui
// n'est pas propriétaire reçoit 403. Le refus vient du service (il connaît le
// propriétaire réel du média) et le handler le traduit sans le réinterpréter.
func TestMediaHandler_DeleteMedia_NotOwner_403(t *testing.T) {
	mock := &mockMediaService{
		delErr: domain.ErrForbidden("Seul le propriétaire de ce média ou un administrateur peut le supprimer."),
	}
	r := newDeleteRouter(mock, true)

	w := deleteMedia(r, "/api/v1/players/"+testPlayerSlug+"/media/files/JGtm/clip.mp4",
		sessionFor("Chocoboflor", "user"))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("media_delete_forbidden")) {
		t.Errorf("corps sans code media_delete_forbidden : %s", w.Body.String())
	}
}

// TestMediaHandler_DeleteMedia_UnknownMedia_404 : chemin inconnu.
func TestMediaHandler_DeleteMedia_UnknownMedia_404(t *testing.T) {
	mock := &mockMediaService{delErr: domain.ErrNotFound("media", "JGtm/inconnu.mp4")}
	r := newDeleteRouter(mock, true)

	w := deleteMedia(r, "/api/v1/players/"+testPlayerSlug+"/media/files/JGtm/inconnu.mp4",
		sessionFor("JGtm", "user"))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestMediaHandler_DeleteMedia_MissingFilePath_400 : file_path est requis.
func TestMediaHandler_DeleteMedia_MissingFilePath_400(t *testing.T) {
	mock := &mockMediaService{}
	r := newDeleteRouter(mock, true)

	w := deleteMedia(r, "", sessionFor("JGtm", "user"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if mock.delReq != nil {
		t.Error("DeleteMedia ne doit PAS être appelé sans file_path")
	}
}

// TestMediaHandler_DeleteMedia_AdminRolePropagated : le rôle admin de la session
// arrive jusqu'au service, qui seul décide (domain.CanDeleteMedia).
func TestMediaHandler_DeleteMedia_AdminRolePropagated(t *testing.T) {
	mock := &mockMediaService{}
	r := newDeleteRouter(mock, true)

	w := deleteMedia(r, "/api/v1/players/"+testPlayerSlug+"/media/files/JGtm/clip.mp4",
		sessionFor("Chocoboflor", string(domain.RoleAdmin)))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if mock.delReq == nil || !mock.delReq.RequesterIsAdmin {
		t.Errorf("RequesterIsAdmin non propagé : %+v", mock.delReq)
	}
}

// TestMediaHandler_DeleteMedia_NoAuthEnforced_NoSession_Passes : en
// mono-utilisateur / démo, la suppression reste possible sans session — sinon
// la fonctionnalité serait inutilisable sur une instance locale, qui est le cas
// d'usage principal du dépôt.
func TestMediaHandler_DeleteMedia_NoAuthEnforced_NoSession_Passes(t *testing.T) {
	mock := &mockMediaService{}
	r := newDeleteRouter(mock, false)

	w := deleteMedia(r, "/api/v1/players/"+testPlayerSlug+"/media/files/JGtm/clip.mp4", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if mock.delReq == nil || mock.delReq.AuthEnforced {
		t.Errorf("AuthEnforced doit être false en mono-utilisateur : %+v", mock.delReq)
	}
}
