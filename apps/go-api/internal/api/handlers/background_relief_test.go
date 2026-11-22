package handlers_test

// background_relief_test.go — LE RELIEF SE SERT PAR `?variante=relief` (image) ET PAR
// `background.relief.json` (données), sur les deux préfixes : par match et par carte.
//
// Ce que ces tests cadenassent : le relief est servi par ses propres méthodes du service, jamais par
// celles du fond ni du plan (les doubles rendent des octets distincts) ; un relief absent est
// l'absence normale d'un fond (404 propre, même code) ; une variante inconnue reste refusée (400) ;
// les données ont le cache des images (ETag, 304, `private`) ; un map_id hors vocabulaire n'atteint
// jamais le service.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/port"
)

func TestFondParMatch_VarianteServiLImageRelief(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return fondMock(), nil }
	r := newReplayRouter(factory)

	w := doReplayPath(r, testPlayerSlug, "000d5950", "/background.png?variante=relief")
	if w.Code != http.StatusOK || w.Body.String() != "relief-match" {
		t.Fatalf("variante=relief : %d %q, attendu 200 \"relief-match\"", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/webp" {
		t.Errorf("Content-Type = %q, attendu image/webp", ct)
	}
	if w.Header().Get("ETag") == "" || w.Header().Get("Cache-Control") != "private, max-age=3600" {
		t.Errorf("en-tetes de cache absents ou changes : %v", w.Header())
	}
	if w = doReplayPath(r, testPlayerSlug, "000d5950", "/background.png?variante=plan"); w.Body.String() != "plan-match" {
		t.Errorf("variante=plan : %q, attendu le plan et non le relief", w.Body.String())
	}
}

func TestFondParMatch_ReliefAbsentEstUnFondAbsent(t *testing.T) {
	mock := fondMock()
	mock.reliefErr = port.ErrMapBackgroundNotAvailable
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return mock, nil }
	r := newReplayRouter(factory)
	for _, suffixe := range []string{"/background.png?variante=relief", "/background.relief.json"} {
		w := doReplayPath(r, testPlayerSlug, "000d5950", suffixe)
		if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "map_background_not_available") {
			t.Errorf("%s : %d %s, attendu 404 map_background_not_available", suffixe, w.Code, w.Body.String())
		}
	}
	// Le fond, lui, reste servi : seul le relief manque.
	if w := doReplayPath(r, testPlayerSlug, "000d5950", "/background.png"); w.Code != http.StatusOK {
		t.Errorf("fond sans relief : %d, attendu 200", w.Code)
	}
}

// Ajouter `relief` aux variantes ne rouvre aucune forme refusée : une valeur inconnue, une casse
// fausse du nom du paramètre ou de la valeur, une liste de variantes restent des 400.
func TestFondParMatch_VarianteReliefMalEcriteRefusee(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return fondMock(), nil }
	for _, q := range []string{"?variante=Relief", "?Variante=relief", "?variante=relief,plan", "?variante=reliefs", "?variante="} {
		w := doReplayPath(newReplayRouter(factory), testPlayerSlug, "000d5950", "/background.png"+q)
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid_request") {
			t.Errorf("%s : %d %s, attendu 400 invalid_request", q, w.Code, w.Body.String())
		}
	}
}

func TestFondParMatch_DonneesDuRelief(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return fondMock(), nil }
	w := doReplayPath(newReplayRouter(factory), testPlayerSlug, "000d5950", "/background.relief.json")
	if w.Code != http.StatusOK || w.Body.String() != `{"donnees":"match"}` {
		t.Fatalf("données du relief : %d %q", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, attendu application/json", ct)
	}
	if w.Header().Get("ETag") == "" || w.Header().Get("Cache-Control") != "private, max-age=3600" {
		t.Errorf("en-tetes de cache absents ou changes : %v", w.Header())
	}
}

func TestFondParMatch_DonneesDuReliefEnPanne(t *testing.T) {
	mock := fondMock()
	mock.reliefErr = context.DeadlineExceeded
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return mock, nil }
	w := doReplayPath(newReplayRouter(factory), testPlayerSlug, "000d5950", "/background.relief.json")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("panne : %d, attendu 500", w.Code)
	}
}

func TestFondParMatch_DonneesDuReliefCycleETag(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return fondMock(), nil }
	r := newReplayRouter(factory)
	verifieCycleETag(t, func(ifNoneMatch string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet,
			"/players/"+testPlayerSlug+"/matches/000d5950/replay/background.relief.json", nil)
		req.RemoteAddr = "127.0.0.1:5432"
		if ifNoneMatch != "" {
			req.Header.Set("If-None-Match", ifNoneMatch)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	})
}

func TestFondParCarte_VarianteServiLImageRelief(t *testing.T) {
	mock := &mockReplayService{imageMap: []byte("fond-carte")}
	r := routeurFond(mock)

	w := appel(t, r, "/players/JGtm/tactical/streets/background.png?variante=relief")
	if w.Code != http.StatusOK || w.Body.String() != "relief-carte" {
		t.Fatalf("variante=relief : %d %q, attendu 200 \"relief-carte\"", w.Code, w.Body.String())
	}
	if mock.vuMapID != "streets" {
		t.Errorf("map_id transmis = %q, attendu streets", mock.vuMapID)
	}
	if w = appel(t, r, "/players/JGtm/tactical/streets/background.png"); w.Body.String() != "fond-carte" {
		t.Errorf("sans variante : %q, attendu le fond", w.Body.String())
	}
	if w = appel(t, r, "/players/JGtm/tactical/streets/background.png?variante=Relief"); w.Code != http.StatusBadRequest {
		t.Errorf("casse fausse : %d, attendu 400", w.Code)
	}
	mock.reliefErr = port.ErrMapBackgroundNotAvailable
	if w = appel(t, r, "/players/JGtm/tactical/streets/background.png?variante=relief"); w.Code != http.StatusNotFound {
		t.Errorf("relief absent : %d, attendu 404", w.Code)
	}
}

func TestFondParCarte_DonneesDuRelief(t *testing.T) {
	mock := &mockReplayService{}
	r := routeurFond(mock)

	w := appel(t, r, "/players/JGtm/tactical/streets/background.relief.json")
	if w.Code != http.StatusOK || w.Body.String() != `{"donnees":"carte"}` {
		t.Fatalf("données du relief : %d %q", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, attendu application/json", ct)
	}
	if w.Header().Get("ETag") == "" || w.Header().Get("Cache-Control") != "private, max-age=3600" {
		t.Errorf("en-tetes de cache absents ou changes : %v", w.Header())
	}
	if mock.vuMapID != "streets" {
		t.Errorf("map_id transmis = %q, attendu streets", mock.vuMapID)
	}

	mock.reliefErr = port.ErrMapBackgroundNotAvailable
	w = appel(t, r, "/players/JGtm/tactical/streets/background.relief.json")
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "map_background_not_available") {
		t.Errorf("relief absent : %d %s, attendu 404 map_background_not_available", w.Code, w.Body.String())
	}
	mock.reliefErr = context.DeadlineExceeded
	if w = appel(t, r, "/players/JGtm/tactical/streets/background.relief.json"); w.Code != http.StatusInternalServerError {
		t.Errorf("panne : %d, attendu 500", w.Code)
	}
}

func TestFondParCarte_DonneesDuReliefCycleETag(t *testing.T) {
	r := routeurFond(&mockReplayService{})
	verifieCycleETag(t, func(ifNoneMatch string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/players/JGtm/tactical/streets/background.relief.json", nil)
		if ifNoneMatch != "" {
			req.Header.Set("If-None-Match", ifNoneMatch)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	})
}

// Un map_id hors vocabulaire est refusé comme pour l'image du fond, avec le message de l'absence, et
// le service n'est jamais appelé : le double servirait des données si on l'appelait.
func TestFondParCarte_DonneesDuReliefRefusentLesChemins(t *testing.T) {
	for _, hostile := range mapIDsHostilesURL {
		mock := &mockReplayService{}
		w := appel(t, routeurFond(mock), "/players/JGtm/tactical/"+hostile+"/background.relief.json")
		if w.Code != http.StatusNotFound || mock.vuMapID != "" {
			t.Errorf("map_id %q : status=%d, map_id vu=%q, attendu 404 sans appel au service",
				hostile, w.Code, mock.vuMapID)
		}
	}
}
