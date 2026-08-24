// Package handlers_test — replay_test.go : tests unitaires ReplayHandler.
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/port"
)

// mockReplayService implémente port.ReplayService.
type mockReplayService struct {
	doc replay.ReplayDocument
	err error
	// bg / image / bgErr : le fond de carte, indépendant de l'artefact — un rejeu peut
	// exister sans fond, et le contraire n'a pas de sens.
	bg    *replay.MapBackground
	image []byte
	bgErr error
	// callouts / calloutsErr : les zones nommées, indépendantes du fond ET de l'artefact.
	callouts    *replay.MapCalloutsEntry
	calloutsErr error
}

func (m *mockReplayService) GetReplay(_ context.Context, _ string) (replay.ReplayDocument, error) {
	return m.doc, m.err
}

func (m *mockReplayService) MapBackground(_ context.Context, _ string) (*replay.MapBackground, error) {
	return m.bg, m.bgErr
}

func (m *mockReplayService) MapBackgroundImage(_ context.Context, _ string) ([]byte, error) {
	return m.image, m.bgErr
}

func (m *mockReplayService) MapCallouts(_ context.Context, _ string) (*replay.MapCalloutsEntry, error) {
	return m.callouts, m.calloutsErr
}

// IsAvailable : le mock rend « disponible » exactement quand GetReplay rendrait un
// document — la présence et la lecture ne peuvent pas diverger dans un test.
func (m *mockReplayService) IsAvailable(_ context.Context, _ string) bool {
	return m.err == nil
}

// AvailableSet : même règle que IsAvailable — le mock ne connaît qu'un match, il
// rend l'ensemble vide (aucun tableau de matchs n'est servi par ce handler).
func (m *mockReplayService) AvailableSet(_ context.Context) (port.ReplayAvailability, error) {
	return port.ReplayAvailability{}, nil
}

func newReplayRouter(factory handlers.ServiceFactory[port.ReplayService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewReplayHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		// Le garde local est un middleware de transport, pas une branche du handler :
		// le monter ici reproduit exactement le montage de server_apiv1.go, et laisse
		// les tests du garde (adresse d'appel non locale → 404) porter sur la chaîne réelle.
		r.Use(handlers.LocalOnlyReplay)
		h.Mount(r)
	})
	return r
}

// doReplayGet émet une requête depuis la BOUCLE LOCALE.
//
// C'EST NÉCESSAIRE, ET C'EST LE SUJET : le rejeu n'est servi qu'en local (cf.
// replay_local_gate.go). `httptest.NewRequest` pose `192.0.2.1:1234` comme adresse d'appel —
// une adresse de documentation, donc non locale — et le garde répondait 404 à tous ces tests.
// Ils testaient le garde sans le savoir, au lieu de tester le handler.
func doReplayGet(r *chi.Mux, slug, matchID string) *httptest.ResponseRecorder {
	return doReplayGetFrom(r, slug, matchID, "127.0.0.1:54321")
}

// doReplayGetFrom permet de choisir l'adresse d'appel, pour éprouver le garde lui-même.
func doReplayGetFrom(r *chi.Mux, slug, matchID, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/players/"+slug+"/matches/"+matchID+"/replay", nil)
	req.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestReplayHandler_OK(t *testing.T) {
	mock := &mockReplayService{doc: replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion, MatchID: "000d5950", TitleSlug: "halo_infinite",
		FrameCount: 2,
		Tracks:     []replay.Track{{Slot: 665, Team: -1, Points: []replay.Point{{T: 0, X: 1, Y: 2}}}},
	}}
	factory := func(_ context.Context, slug string) (port.ReplayService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	w := doReplayGet(newReplayRouter(factory), testPlayerSlug, "000d5950")
	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", w.Code, w.Body.String())
	}
	var got replay.ReplayDocument
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("réponse illisible: %v", err)
	}
	if got.MatchID != "000d5950" || len(got.Tracks) != 1 || got.Tracks[0].Slot != 665 {
		t.Errorf("document inattendu: %+v", got)
	}
}

func TestReplayHandler_NotAvailable(t *testing.T) {
	mock := &mockReplayService{err: port.ErrReplayNotAvailable}
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return mock, nil }
	w := doReplayGet(newReplayRouter(factory), testPlayerSlug, "sans-artefact")
	if w.Code != http.StatusNotFound {
		t.Fatalf("attendu 404, obtenu %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "replay_not_available") {
		t.Errorf("code d'erreur attendu replay_not_available, body=%s", w.Body.String())
	}
}

func TestReplayHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) {
		return nil, errors.New("player_not_found")
	}
	w := doReplayGet(newReplayRouter(factory), "inconnu", "000d5950")
	if w.Code != http.StatusNotFound {
		t.Fatalf("attendu 404, obtenu %d", w.Code)
	}
}

func TestReplayHandler_ServiceError(t *testing.T) {
	mock := &mockReplayService{err: errors.New("boom")}
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return mock, nil }
	w := doReplayGet(newReplayRouter(factory), testPlayerSlug, "000d5950")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("attendu 500, obtenu %d", w.Code)
	}
}

// TestReplayHandler_RefusesRemoteCaller vérifie que LA ROUTE applique le garde local.
//
// CE QU'IL AJOUTE AUX TESTS DU GARDE. `TestReplayGate_*` éprouve la RÈGLE (quelle adresse est
// locale, et qu'un en-tête ne peut rien contre elle) ; celui-ci éprouve son BRANCHEMENT — si
// l'appel à `allowReplay` disparaissait du handler, les tests de la règle resteraient verts et
// le rejeu serait servi à tout le monde.
//
// Il répare aussi le défaut qui l'a fait écrire : le garde a été posé sans que les tests du
// handler soient adaptés, et ceux-ci tombaient dessus en annonçant un handler cassé.
func TestReplayHandler_RefusesRemoteCaller(t *testing.T) {
	mock := &mockReplayService{doc: replay.ReplayDocument{MatchID: "000d5950"}}
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return mock, nil }
	w := doReplayGetFrom(newReplayRouter(factory), testPlayerSlug, "000d5950", "203.0.113.7:443")
	if w.Code != http.StatusNotFound {
		t.Fatalf("un appelant distant doit recevoir 404, obtenu %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "replay_not_available") {
		t.Errorf("code d'erreur attendu replay_not_available, body=%s", w.Body.String())
	}
}

// doReplayPathFrom émet un GET sur un sous-chemin du rejeu, depuis l'adresse donnée.
func doReplayPathFrom(r *chi.Mux, slug, matchID, suffixe, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet,
		"/players/"+slug+"/matches/"+matchID+"/replay"+suffixe, nil)
	req.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// fondMock rend un service dont le fond de carte est celui de Cliffhanger.
func fondMock() *mockReplayService {
	return &mockReplayService{
		bg: &replay.MapBackground{
			SchemaVersion: replay.MapBackgroundSchemaVersion,
			Module:        "ridgeline",
			Image:         "ridgeline.png",
			Calibration: replay.MapBackgroundCalibration{
				MetersPerPixel: 0.092, OriginX: -57.3, OriginY: 78.87,
				WidthPx: 1633, HeightPx: 1627,
			},
		},
		image: []byte("\x89PNG\r\n\x1a\nfaux"),
	}
}

// TestReplayBackground_OK — le calage voyage entier : sans lui l'image ne se pose nulle part.
func TestReplayBackground_OK(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return fondMock(), nil }
	w := doReplayPathFrom(newReplayRouter(factory), testPlayerSlug, "000d5950", "/background", "127.0.0.1:5432")
	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", w.Code, w.Body.String())
	}
	var got replay.MapBackground
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("réponse illisible: %v", err)
	}
	if got.Module != "ridgeline" || got.Calibration.WidthPx != 1633 || got.Calibration.MetersPerPixel != 0.092 {
		t.Errorf("calage incomplet: %+v", got)
	}
}

// TestReplayBackgroundImage_OK — les octets sortent tels quels, avec leur type.
func TestReplayBackgroundImage_OK(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return fondMock(), nil }
	w := doReplayPathFrom(newReplayRouter(factory), testPlayerSlug, "000d5950", "/background.png", "127.0.0.1:5432")
	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, attendu image/png", ct)
	}
	if !strings.HasPrefix(w.Body.String(), "\x89PNG") {
		t.Errorf("les octets servis ne sont pas ceux du service: %q", w.Body.String())
	}
}

// TestReplayBackground_NotAvailable — une carte sans fond figé est un cas NORMAL : 404
// nommé, sur les deux routes, jamais un 500.
func TestReplayBackground_NotAvailable(t *testing.T) {
	mock := &mockReplayService{bgErr: port.ErrMapBackgroundNotAvailable}
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return mock, nil }
	for _, suffixe := range []string{"/background", "/background.png"} {
		w := doReplayPathFrom(newReplayRouter(factory), testPlayerSlug, "000d5950", suffixe, "127.0.0.1:5432")
		if w.Code != http.StatusNotFound {
			t.Errorf("%s: attendu 404, obtenu %d", suffixe, w.Code)
		}
		if !strings.Contains(w.Body.String(), "map_background_not_available") {
			t.Errorf("%s: code attendu map_background_not_available, body=%s", suffixe, w.Body.String())
		}
	}
}

// TestReplayBackground_RefusesRemoteCaller — LE POINT QUI COMPTE POUR LA PROD.
//
// Le rejeu n'est servi qu'en local ; ses deux nouvelles routes doivent hériter du MÊME
// garde, y compris la route chi nue de l'image — c'est précisément celle qui pourrait
// échapper au montage sans que rien ne le dise.
func TestReplayBackground_RefusesRemoteCaller(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return fondMock(), nil }
	for _, suffixe := range []string{"/background", "/background.png"} {
		w := doReplayPathFrom(newReplayRouter(factory), testPlayerSlug, "000d5950", suffixe, "203.0.113.7:443")
		if w.Code != http.StatusNotFound {
			t.Errorf("%s: un appelant distant doit recevoir 404, obtenu %d", suffixe, w.Code)
		}
		if strings.HasPrefix(w.Body.String(), "\x89PNG") {
			t.Errorf("%s: l'image a été servie à un appelant distant", suffixe)
		}
	}
}

// calloutsMock rend un service dont les zones nommées sont celles de Cliffhanger.
func calloutsMock() *mockReplayService {
	return &mockReplayService{
		callouts: &replay.MapCalloutsEntry{
			Module:     "ridgeline",
			Provenance: replay.CalloutsProvenanceDecoupe,
			Zones: []replay.CalloutZone{{
				VolumeIndex: 10, Name: "ridgeline horses", EN: "Horseshoe", FR: "Fer à cheval",
				X: 19, Y: 10, Z: 1, ZBottom: -0.2, ZTop: 11,
				Polygon: [][2]float64{{14.8, 7.5}, {23.8, 7.5}, {23.8, 15.3}},
			}},
		},
	}
}

// TestReplayCallouts_OK — l'entrée voyage entière : polygones monde + libellés FR/EN.
func TestReplayCallouts_OK(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return calloutsMock(), nil }
	w := doReplayPathFrom(newReplayRouter(factory), testPlayerSlug, "000d5950", "/callouts", "127.0.0.1:5432")
	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", w.Code, w.Body.String())
	}
	var got replay.MapCalloutsEntry
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("réponse illisible: %v", err)
	}
	if got.Module != "ridgeline" || len(got.Zones) != 1 || got.Zones[0].FR != "Fer à cheval" ||
		len(got.Zones[0].Polygon) != 3 {
		t.Errorf("entrée incomplète: %+v", got)
	}
}

// TestReplayCallouts_NotAvailable — une carte sans zones nommées (Forge) est un cas
// NORMAL : 404 nommé, jamais un 500.
func TestReplayCallouts_NotAvailable(t *testing.T) {
	mock := &mockReplayService{calloutsErr: port.ErrMapCalloutsNotAvailable}
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return mock, nil }
	w := doReplayPathFrom(newReplayRouter(factory), testPlayerSlug, "000d5950", "/callouts", "127.0.0.1:5432")
	if w.Code != http.StatusNotFound {
		t.Fatalf("attendu 404, obtenu %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "map_callouts_not_available") {
		t.Errorf("code attendu map_callouts_not_available, body=%s", w.Body.String())
	}
}

// TestReplayCallouts_RefusesRemoteCaller — la route hérite du même garde local que le
// reste du rejeu.
func TestReplayCallouts_RefusesRemoteCaller(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return calloutsMock(), nil }
	w := doReplayPathFrom(newReplayRouter(factory), testPlayerSlug, "000d5950", "/callouts", "203.0.113.7:443")
	if w.Code != http.StatusNotFound {
		t.Fatalf("un appelant distant doit recevoir 404, obtenu %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "Horseshoe") {
		t.Error("les zones ont été servies à un appelant distant")
	}
}

// TestReplayBackground_PlayerNotFound — la résolution du joueur précède tout le reste.
func TestReplayBackground_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) {
		return nil, errors.New("player_not_found")
	}
	for _, suffixe := range []string{"/background", "/background.png"} {
		w := doReplayPathFrom(newReplayRouter(factory), "inconnu", "000d5950", suffixe, "127.0.0.1:5432")
		if w.Code != http.StatusNotFound {
			t.Errorf("%s: attendu 404, obtenu %d", suffixe, w.Code)
		}
	}
}
