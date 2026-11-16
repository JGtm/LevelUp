// Package handlers_test — replay_spawn_forecast_test.go : la route
// GET /players/{slug}/matches/{id}/replay/reapparitions (prevision de reapparition).
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

func previsionMock(mock *mockReplayService) func(context.Context, string) (port.ReplayService, error) {
	return func(_ context.Context, _ string) (port.ReplayService, error) { return mock, nil }
}

func TestReplaySpawnForecast_Servie(t *testing.T) {
	zone := 3
	mock := &mockReplayService{forecast: replaydoc.SpawnForecast{
		Status: replaydoc.SpawnForecastStatusReady, Family: "assassin", Model: "assassin",
		MatchesOnMap: 12, Candidates: 80, SampleStepMs: 1000,
		Zones: []replaydoc.SpawnForecastZone{{VolumeIndex: 3, Name: "Fer à cheval"}},
		Deaths: []replaydoc.SpawnForecastDeath{{
			XUID: "xuid(1)", DeathFrame: 100, RespawnFrame: 400, ActualZone: &zone,
			Samples: []replaydoc.SpawnForecastSample{{Frame: 100, Zones: []replaydoc.SpawnForecastZoneChance{{Zone: 3, P: 0.6}}}},
		}},
	}}
	w := doReplayPath(newReplayRouter(previsionMock(mock)), testPlayerSlug, "000d5950", "/reapparitions")
	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", w.Code, w.Body.String())
	}
	var got replaydoc.SpawnForecast
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("réponse illisible: %v", err)
	}
	if got.Status != "ready" || len(got.Deaths) != 1 || got.Deaths[0].XUID != "xuid(1)" ||
		got.Deaths[0].ActualZone == nil || *got.Deaths[0].ActualZone != 3 || got.Zones[0].Name != "Fer à cheval" {
		t.Errorf("prévision incomplète: %+v", got)
	}
}

// Une prévision qui n'a pas lieu d'être est une RÉPONSE : 200, statut unavailable, sa raison.
func TestReplaySpawnForecast_IndisponibleEstUneReponse(t *testing.T) {
	mock := &mockReplayService{forecast: replaydoc.SpawnForecast{
		Status: replaydoc.SpawnForecastStatusUnavailable, Reason: replaydoc.SpawnForecastReasonNotEnough,
		Message: "carte pauvre", Zones: []replaydoc.SpawnForecastZone{}, Deaths: []replaydoc.SpawnForecastDeath{},
	}}
	w := doReplayPath(newReplayRouter(previsionMock(mock)), testPlayerSlug, "000d5950", "/reapparitions")
	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", w.Code, w.Body.String())
	}
	for _, attendu := range []string{`"status":"unavailable"`, `"reason":"not_enough_matches"`, `"deaths":[]`, `"zones":[]`} {
		if !strings.Contains(w.Body.String(), attendu) {
			t.Errorf("la réponse devrait contenir %s : %s", attendu, w.Body.String())
		}
	}
}

func TestReplaySpawnForecast_Erreurs(t *testing.T) {
	cas := []struct {
		nom    string
		err    error
		statut int
		code   string
	}{
		{"sans rejeu", port.ErrReplayNotAvailable, http.StatusNotFound, "replay_not_available"},
		{"capability absente", games.ErrCapabilityNotSupported, http.StatusServiceUnavailable, "capability_not_supported"},
		{"panne", errors.New("disque illisible /secret/chemin"), http.StatusInternalServerError, "replay_error"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			mock := &mockReplayService{forecastErr: c.err}
			w := doReplayPath(newReplayRouter(previsionMock(mock)), testPlayerSlug, "000d5950", "/reapparitions")
			if w.Code != c.statut || !strings.Contains(w.Body.String(), c.code) {
				t.Errorf("attendu %d %s, obtenu %d: %s", c.statut, c.code, w.Code, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "/secret/chemin") {
				t.Errorf("le texte de l'erreur ne part pas au client : %s", w.Body.String())
			}
		})
	}
}

func TestReplaySpawnForecast_JoueurInconnu(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) {
		return nil, errors.New("player_not_found")
	}
	w := doReplayPath(newReplayRouter(factory), "inconnu", "000d5950", "/reapparitions")
	if w.Code != http.StatusNotFound {
		t.Errorf("attendu 404, obtenu %d", w.Code)
	}
}
