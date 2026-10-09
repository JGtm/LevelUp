// Package handlers_test — presence_test.go : GET /api/v1/presence bout-en-bout
// (routage Huma + service réel), sur le cas qui doit rester silencieux : le
// watcher est éteint. Le shell interroge cet endpoint toutes les 30 s ; il doit
// alors recevoir une réponse vide en 200, jamais une erreur.
package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/service"
)

// presenceRouter monte l'endpoint avec le service RÉEL (aucun mock de service :
// c'est justement sa dégradation qu'on teste).
func presenceRouter(svc *service.PresenceService) *chi.Mux {
	r := chi.NewRouter()
	handlers.NewPresenceHandler(svc).Mount(r)
	return r
}

func decodePresence(t *testing.T, r *chi.Mux) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/presence", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu 200 : %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("JSON invalide : %v (%s)", err, w.Body.String())
	}
	return body
}

// Watcher désactivé (aucune source de présence) : 200 avec une liste VIDE (pas
// nulle) et un compteur à zéro.
func TestPresenceEndpoint_NoDaemon_EmptyPayload(t *testing.T) {
	svc := service.NewPresenceService(
		func(context.Context, *domain.SessionData) ([]domain.PlayerSummary, error) {
			return []domain.PlayerSummary{{PlayerSlug: "jgtm", Gamertag: "JGtm"}}, nil
		},
		nil, // watcher absent
	)

	body := decodePresence(t, presenceRouter(svc))
	players, ok := body["players"].([]any)
	if !ok {
		t.Fatalf("players = %v (%T), attendu un tableau", body["players"], body["players"])
	}
	if len(players) != 0 {
		t.Errorf("players = %d, attendu 0 (aucune présence connue)", len(players))
	}
	if body["friends_in_game"] != float64(0) {
		t.Errorf("friends_in_game = %v, attendu 0", body["friends_in_game"])
	}
}

// Service entièrement dépourvu de sources (démo, boot partiel) : même contrat,
// aucune panique.
func TestPresenceEndpoint_NoSourcesAtAll_EmptyPayload(t *testing.T) {
	body := decodePresence(t, presenceRouter(service.NewPresenceService(nil, nil)))
	if players, ok := body["players"].([]any); !ok || len(players) != 0 {
		t.Errorf("players = %v, attendu tableau vide", body["players"])
	}
}

// Cas nominal : la présence d'un joueur suivi remonte au contrat, avec son
// titre — c'est ce que lit le sélecteur de joueur du shell.
func TestPresenceEndpoint_InGamePlayerIsServed(t *testing.T) {
	svc := service.NewPresenceService(
		func(context.Context, *domain.SessionData) ([]domain.PlayerSummary, error) {
			return []domain.PlayerSummary{{PlayerSlug: "jgtm", Gamertag: "JGtm"}}, nil
		},
		func() []service.TrackedPresence {
			// LastEventAt est le témoin de vivacité du poll : sans lui le service
			// considère le titre comme périmé et ne le sert pas (borne de
			// fraîcheur, cf. service.presenceFreshnessWindow).
			return []service.TrackedPresence{
				{
					Gamertag:    "JGtm",
					TitleSlug:   "halo_infinite",
					TitleName:   "Halo Infinite",
					LastEventAt: time.Now(),
				},
			}
		},
	)

	body := decodePresence(t, presenceRouter(svc))
	players, _ := body["players"].([]any)
	if len(players) != 1 {
		t.Fatalf("players = %d, attendu 1", len(players))
	}
	p, _ := players[0].(map[string]any)
	if p["player_slug"] != "jgtm" || p["in_game"] != true || p["title_slug"] != "halo_infinite" {
		t.Errorf("joueur servi = %+v", p)
	}
}
