package handlers_test

// tactical_background_test.go — LE FOND DE CARTE DE LA GRILLE, servi PAR CARTE.
//
// Ce que ces tests cadenassent :
//   - l'image arrive telle quelle, avec son type et son cache — le meme que le fond par
//     match, parce que c'est la MEME donnee de reference versionnee ;
//   - le map_id de l'URL est bien celui transmis au service (une vignette qui affiche le
//     fond d'une AUTRE carte est indetectable a l'oeil) ;
//   - une carte sans fond et une carte inconnue rendent toutes deux un 404 PROPRE : les
//     deux sont des absences normales, jamais une panne de page ;
//   - le fond N'EST PAS derriere le garde local du rejeu (une requete depuis une adresse
//     non locale est servie) — sans quoi la grille serait vide en production.

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/port"
)

// appelDepuis emet une requete GET depuis une adresse d'appel choisie.
func appelDepuis(t *testing.T, r *chi.Mux, url, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func routeurFond(mock *mockReplayService) *chi.Mux {
	return newTacticalRouterAvecFond(
		tacticalFactory(&fakeTacticalSvc{page: domain.TacticalMapsPage{}}, nil),
		tacticalReplayFactory(mock, nil))
}

// TestTacticalBackgroundImage_OK : 200, octets intacts, en-tetes de cache, map_id transmis.
func TestTacticalBackgroundImage_OK(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0x42}
	mock := &mockReplayService{imageMap: png}
	w := appel(t, routeurFond(mock), "/players/JGtm/tactical/streets/background.png")
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := w.Body.Bytes(); string(got) != string(png) {
		t.Fatalf("octets alteres: %v", got)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type=%q, attendu image/png", ct)
	}
	if cl := w.Header().Get("Content-Length"); cl != "9" {
		t.Errorf("Content-Length=%q, attendu 9", cl)
	}
	// Le meme cache que le fond par match : donnee de reference versionnee, derriere
	// l'ownership joueur — `private`, jamais `public`.
	if cc := w.Header().Get("Cache-Control"); cc != "private, max-age=3600" {
		t.Errorf("Cache-Control=%q", cc)
	}
	if mock.vuMapID != "streets" {
		t.Errorf("map_id transmis=%q, attendu streets", mock.vuMapID)
	}
}

// TestTacticalBackgroundImage_SansFond : la carte existe, elle n'a pas de fond fige — 404
// PROPRE, avec le code d'erreur du contrat. C'est une absence NORMALE : toutes les cartes
// n'ont pas d'image cuite.
func TestTacticalBackgroundImage_SansFond(t *testing.T) {
	mock := &mockReplayService{bgMapErr: port.ErrMapBackgroundNotAvailable}
	w := appel(t, routeurFond(mock), "/players/JGtm/tactical/aquarius/background.png")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d, attendu 404 — body=%s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); !strings.Contains(body, "map_background_not_available") {
		t.Errorf("code d'erreur absent du corps: %s", body)
	}
}

// TestTacticalBackgroundImage_CarteInconnue : un map_id que la base ne resout pas rend la
// MEME sentinelle d'absence (le service ne sait pas nommer la carte, donc pas de fond) —
// 404, jamais un 500.
func TestTacticalBackgroundImage_CarteInconnue(t *testing.T) {
	mock := &mockReplayService{bgMapErr: port.ErrMapBackgroundNotAvailable}
	w := appel(t, routeurFond(mock), "/players/JGtm/tactical/carte-qui-n-existe-pas/background.png")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d, attendu 404 — body=%s", w.Code, w.Body.String())
	}
	if mock.vuMapID != "carte-qui-n-existe-pas" {
		t.Errorf("map_id transmis=%q", mock.vuMapID)
	}
}

// TestTacticalBackgroundImage_Erreur : toute AUTRE erreur reste un 500 — une lecture
// disque en echec n'est pas une absence, et la confondre avec un 404 masquerait la panne.
func TestTacticalBackgroundImage_Erreur(t *testing.T) {
	mock := &mockReplayService{bgMapErr: errors.New("disque illisible")}
	w := appel(t, routeurFond(mock), "/players/JGtm/tactical/streets/background.png")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, attendu 500 — body=%s", w.Code, w.Body.String())
	}
}

// TestTacticalBackgroundImage_JoueurInconnu : 404 player_not_found, avant toute lecture.
func TestTacticalBackgroundImage_JoueurInconnu(t *testing.T) {
	r := newTacticalRouterAvecFond(
		tacticalFactory(&fakeTacticalSvc{}, nil),
		tacticalReplayFactory(nil, errors.New("no such player")))
	w := appel(t, r, "/players/inconnu/tactical/streets/background.png")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d, attendu 404 — body=%s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); !strings.Contains(body, "player_not_found") {
		t.Errorf("code d'erreur absent du corps: %s", body)
	}
}

// TestTacticalBackgroundImage_HorsBoucleLocale : le fond de la grille N'EST PAS sous le
// garde local du rejeu. Ce garde protege les trajectoires decodees du film ; l'image d'une
// carte est une donnee de reference versionnee. Monter la route dans le groupe garde
// aurait vide la grille en production sans rien proteger — d'ou ce test, qui appelle
// depuis une adresse NON locale et attend 200.
func TestTacticalBackgroundImage_HorsBoucleLocale(t *testing.T) {
	mock := &mockReplayService{imageMap: []byte{1, 2, 3}}
	w := appelDepuis(t, routeurFond(mock),
		"/players/JGtm/tactical/streets/background.png", "203.0.113.7:4242")
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, attendu 200 hors boucle locale — body=%s", w.Code, w.Body.String())
	}
}

// TestTacticalBackground_Calage : le calage voyage avec l'image (metres par pixel, origine
// monde) — l'un sans l'autre ne se pose nulle part.
func TestTacticalBackground_Calage(t *testing.T) {
	mock := &mockReplayService{bgMap: &replaydoc.MapBackground{
		Module: "streets",
		Calibration: replaydoc.MapBackgroundCalibration{
			MetersPerPixel: 0.25, WidthPx: 1024, HeightPx: 1024,
		},
	}}
	w := appel(t, routeurFond(mock), "/players/JGtm/tactical/streets/background")
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); !strings.Contains(body, "streets") {
		t.Errorf("calage sans sa cle de carte: %s", body)
	}
	if mock.vuMapID != "streets" {
		t.Errorf("map_id transmis=%q", mock.vuMapID)
	}
}

// TestTacticalBackground_CalageAbsent : 404 propre, meme sentinelle que l'image.
func TestTacticalBackground_CalageAbsent(t *testing.T) {
	mock := &mockReplayService{bgMapErr: port.ErrMapBackgroundNotAvailable}
	w := appel(t, routeurFond(mock), "/players/JGtm/tactical/aquarius/background")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d, attendu 404 — body=%s", w.Code, w.Body.String())
	}
}
