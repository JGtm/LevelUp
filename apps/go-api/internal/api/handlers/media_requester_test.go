// Package handlers_test — media_requester_test.go : le demandeur des écritures de
// médias est le profil POSSÉDÉ par l'utilisateur de la session, jamais le joueur
// courant de la session (que POST /session/context laisse choisir).
//
// Scénario d'attaque (registre sécurité C4-1) : Mallory fixe le joueur courant de sa
// session sur sa victime, puis supprime ou like un média. Le demandeur doit rester
// Mallory : la suppression est jugée sur SON identité (refusée si le média est d'un
// autre) et le like lui est attribué.
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// requesterIs : le résolveur câblé par le routeur, réduit à une valeur fixe.
func requesterIs(slug string) handlers.RequesterSlugResolver {
	return func(context.Context) string { return slug }
}

// forgedSession : la session de Mallory, joueur courant fixé sur la victime.
func forgedSession(victim string) *domain.SessionData {
	role := "user"
	return &domain.SessionData{SessionID: "s-mallory", CurrentPlayerSlug: &victim, Role: &role}
}

func newRequesterMediaRouter(svc port.MediaService, requester string) *chi.Mux {
	factory := func(context.Context, string) (port.MediaService, error) { return svc, nil }
	h := handlers.NewMediaHandler(factory, nil, "").WithAuthEnforced(true).WithRequester(requesterIs(requester))
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) { h.Mount(r) })
	return r
}

func TestMediaDelete_RequesterIsOwnedProfileNotSessionPlayer(t *testing.T) {
	mock := &mockMediaService{}
	r := newRequesterMediaRouter(mock, "Mallory")
	w := deleteMedia(r, "/api/v1/players/"+testPlayerSlug+"/media/files/JGtm/clip.mp4", forgedSession("JGtm"))
	if w.Code != http.StatusOK {
		t.Fatalf("statut %d : %s", w.Code, w.Body.String())
	}
	if mock.delReq == nil || mock.delReq.RequesterSlug != "Mallory" {
		t.Fatalf("demandeur transmis au service = %+v, attendu Mallory (jamais le joueur courant forgé)", mock.delReq)
	}
}

func TestMediaDelete_NoOwnedProfileIs401EvenWithSessionPlayer(t *testing.T) {
	mock := &mockMediaService{}
	r := newRequesterMediaRouter(mock, "")
	w := deleteMedia(r, "/api/v1/players/"+testPlayerSlug+"/media/files/JGtm/clip.mp4", forgedSession("JGtm"))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("statut %d, attendu 401 : %s", w.Code, w.Body.String())
	}
	if mock.delReq != nil {
		t.Fatal("suppression transmise au service sans demandeur")
	}
}

func likeWithSession(r http.Handler, sess *domain.SessionData) *httptest.ResponseRecorder {
	body, _ := json.Marshal(domain.MediaLikeRequest{FilePath: "JGtm/clip.mp4", Liked: true, LikerSlug: "JGtm"})
	req := httptest.NewRequest(http.MethodPatch, "/players/"+testPlayerSlug+"/media/likes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.InjectSession(req.Context(), sess))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestMediaLike_LikerIsOwnedProfileNotSessionPlayer(t *testing.T) {
	spy := &spyLikeService{}
	r := newRequesterMediaRouter(spy, "Mallory")
	w := likeWithSession(r, forgedSession("JGtm"))
	if w.Code != http.StatusOK {
		t.Fatalf("statut %d : %s", w.Code, w.Body.String())
	}
	if spy.capturedReq == nil || spy.capturedReq.LikerSlug != "Mallory" {
		t.Fatalf("liker transmis = %+v, attendu Mallory (ni le joueur courant forgé, ni le corps)", spy.capturedReq)
	}
}

func TestMediaLike_NoOwnedProfileIs401EvenWithSessionPlayer(t *testing.T) {
	spy := &spyLikeService{}
	r := newRequesterMediaRouter(spy, "")
	w := likeWithSession(r, forgedSession("JGtm"))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("statut %d, attendu 401 : %s", w.Code, w.Body.String())
	}
	if spy.capturedReq != nil {
		t.Fatal("like transmis au service sans liker")
	}
}

// spyAssociateService : le service refuse le média d'autrui (règle CanModifyMedia),
// et capture la requête pour vérifier l'identité transmise.
type spyAssociateService struct {
	mockMediaService
	got *domain.MediaAssociateRequest
}

func (s *spyAssociateService) AssociateMediaToMatch(_ context.Context, req domain.MediaAssociateRequest) (*domain.MediaAssociateResponse, error) {
	s.got = &req
	if req.RequesterSlug != "JGtm" {
		return nil, domain.ErrForbidden("propriétaire seulement")
	}
	return &domain.MediaAssociateResponse{FilePath: req.FilePath, MatchID: req.MatchID}, nil
}

// TestMediaAssociate_ForeignMediaIs403 : Mallory (joueur courant forgé sur JGtm)
// associe le clip de JGtm → 403, avec Mallory transmis comme demandeur.
func TestMediaAssociate_ForeignMediaIs403(t *testing.T) {
	spy := &spyAssociateService{}
	r := newRequesterMediaRouter(spy, "Mallory")
	body, _ := json.Marshal(domain.MediaAssociateRequest{FilePath: "JGtm/clip.mp4", MatchID: "m-autre"})
	req := httptest.NewRequest(http.MethodPost, "/players/"+testPlayerSlug+"/media/associate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.InjectSession(req.Context(), forgedSession("JGtm")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("statut %d, attendu 403 : %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("media_associate_forbidden")) {
		t.Fatalf("code attendu media_associate_forbidden : %s", w.Body.String())
	}
	if spy.got == nil || spy.got.RequesterSlug != "Mallory" || !spy.got.AuthEnforced {
		t.Fatalf("requête transmise = %+v, attendu demandeur Mallory, auth appliquée", spy.got)
	}
}
