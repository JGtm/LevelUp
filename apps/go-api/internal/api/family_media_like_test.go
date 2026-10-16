package api

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

// likeSpy : service médias qui ne sert que le like et capture la requête reçue.
type likeSpy struct {
	port.MediaService
	got *domain.MediaLikeRequest
}

func (s *likeSpy) SetMediaLike(_ context.Context, req domain.MediaLikeRequest) (*domain.MediaLikeResponse, error) {
	s.got = &req
	return &domain.MediaLikeResponse{FilePath: req.FilePath, Liked: req.Liked}, nil
}

// TestFamilyMemberLikesFriendMedia_AttributedToOwnProfile : Mallory, co-membre du groupe
// de JGtm, like un média depuis le profil de JGtm. La garde de propriété la laisse passer
// (exception nommée de l'accès famille en lecture seule) et le like est attribué au
// profil que Mallory possède, jamais à JGtm ni au corps de la requête.
func TestFamilyMemberLikesFriendMedia_AttributedToOwnProfile(t *testing.T) {
	cfg := requesterConfig(t, "password")
	users := requesterUsers{"mallory": {Username: "mallory", Role: domain.RoleUser, XUID: "222"}}
	access := middleware.NewPlayerAccess(false, "password", playerOwnershipXUIDResolver(cfg), users,
		func(context.Context) map[string]bool { return map[string]bool{"222": true, "999": true} })

	spy := &likeSpy{}
	factory := func(context.Context, string) (port.MediaService, error) { return spy, nil }
	h := handlers.NewMediaHandler(factory, nil, "").
		WithAuthEnforced(true).
		WithRequester(requesterPlayerSlugResolver(cfg, users))
	r := chi.NewRouter()
	r.Route("/api/v1/players/{player_slug}", func(r chi.Router) {
		r.Use(middleware.RequirePlayerOwnership(access))
		h.Mount(r)
	})

	body, _ := json.Marshal(domain.MediaLikeRequest{FilePath: "JGtm/clip.mp4", Liked: true, LikerSlug: "JGtm"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/players/JGtm/media/likes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	username, current := "mallory", "JGtm"
	sess := &domain.SessionData{SessionID: "s", Username: &username, CurrentPlayerSlug: &current}
	req = req.WithContext(middleware.InjectSession(req.Context(), sess))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("statut %d, attendu 200 : %s", w.Code, w.Body.String())
	}
	if spy.got == nil || spy.got.LikerSlug != "Mallory" {
		t.Fatalf("like transmis = %+v, attendu liker Mallory (profil possédé du co-membre)", spy.got)
	}
}
