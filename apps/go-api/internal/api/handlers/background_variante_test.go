package handlers_test

// background_variante_test.go — L'IMAGE PLAN SE SERT PAR `?variante=plan`, sur les deux routes
// d'image de fond (par match et par carte).
//
// Ce que ces tests cadenassent : la variante `plan` sert l'image plan et JAMAIS le fond (les doubles
// rendent des octets distincts) ; sans variante le fond est servi comme avant ; une variante
// inconnue est refusee (400) plutot que repondue par le fond ; un plan absent est l'absence
// normale d'un fond (404 propre, meme code).

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"levelup/go-api/internal/port"
)

func TestFondParMatch_VarianteServiLImagePlan(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return fondMock(), nil }
	r := newReplayRouter(factory)

	w := doReplayPath(r, testPlayerSlug, "000d5950", "/background.png?variante=plan")
	if w.Code != http.StatusOK || w.Body.String() != "plan-match" {
		t.Fatalf("variante=plan : %d %q, attendu 200 \"plan-match\"", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/webp" {
		t.Errorf("Content-Type = %q, attendu image/webp", ct)
	}
	if w.Header().Get("ETag") == "" || !strings.Contains(w.Header().Get("Cache-Control"), "private") {
		t.Errorf("en-tetes de cache absents : %v", w.Header())
	}

	w = doReplayPath(r, testPlayerSlug, "000d5950", "/background.png")
	if w.Code != http.StatusOK || w.Body.String() == "plan-match" {
		t.Errorf("sans variante : %d, attendu le fond et non le plan", w.Code)
	}
}

func TestFondParMatch_VarianteInconnueRefusee(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return fondMock(), nil }
	w := doReplayPath(newReplayRouter(factory), testPlayerSlug, "000d5950", "/background.png?variante=autre")
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid_request") {
		t.Fatalf("variante inconnue : %d %s, attendu 400 invalid_request", w.Code, w.Body.String())
	}
}

// Une valeur vide et un nom de paramètre d'une autre casse sont refusés comme une valeur
// inconnue : seule l'absence du paramètre sert le fond.
// Mutation jouee : traiter `?variante=` comme l'absence du paramètre.
func TestFondParMatch_VarianteVideOuMalEcriteRefusee(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return fondMock(), nil }
	for _, q := range []string{"?variante=", "?variante", "?Variante=plan", "?VARIANTE="} {
		w := doReplayPath(newReplayRouter(factory), testPlayerSlug, "000d5950", "/background.png"+q)
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid_request") {
			t.Errorf("%s : %d %s, attendu 400 invalid_request", q, w.Code, w.Body.String())
		}
	}
}

func TestFondParMatch_PlanAbsentEstUnFondAbsent(t *testing.T) {
	mock := fondMock()
	mock.planErr = port.ErrMapBackgroundNotAvailable
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return mock, nil }
	w := doReplayPath(newReplayRouter(factory), testPlayerSlug, "000d5950", "/background.png?variante=plan")
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "map_background_not_available") {
		t.Fatalf("plan absent : %d %s, attendu 404 map_background_not_available", w.Code, w.Body.String())
	}
}

func TestFondParCarte_VarianteServiLImagePlan(t *testing.T) {
	mock := &mockReplayService{imageMap: []byte("fond-carte")}
	r := routeurFond(mock)

	w := appel(t, r, "/players/JGtm/tactical/streets/background.png?variante=plan")
	if w.Code != http.StatusOK || w.Body.String() != "plan-carte" {
		t.Fatalf("variante=plan : %d %q, attendu 200 \"plan-carte\"", w.Code, w.Body.String())
	}
	if mock.vuMapID != "streets" {
		t.Errorf("map_id transmis = %q, attendu streets", mock.vuMapID)
	}
	if w = appel(t, r, "/players/JGtm/tactical/streets/background.png"); w.Body.String() != "fond-carte" {
		t.Errorf("sans variante : %q, attendu le fond", w.Body.String())
	}
	if w = appel(t, r, "/players/JGtm/tactical/streets/background.png?variante=x"); w.Code != http.StatusBadRequest {
		t.Errorf("variante inconnue : %d, attendu 400", w.Code)
	}
	mock.planErr = port.ErrMapBackgroundNotAvailable
	if w = appel(t, r, "/players/JGtm/tactical/streets/background.png?variante=plan"); w.Code != http.StatusNotFound {
		t.Errorf("plan absent : %d, attendu 404", w.Code)
	}
}
