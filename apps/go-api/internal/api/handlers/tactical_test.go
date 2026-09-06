package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// fakeTacticalSvc double port.TacticalService et retient ce qu'on lui passe.
type fakeTacticalSvc struct {
	page    domain.TacticalMapsPage
	raster  domain.TacticalRaster
	errMaps error
	errRast error

	vuCarte    string
	vuQuestion string
	vuQui      string
	vuFiltre   *domain.MatchFilterSpec
}

func (f *fakeTacticalSvc) MapsPlayed(_ context.Context, filtre *domain.MatchFilterSpec) (domain.TacticalMapsPage, error) {
	f.vuFiltre = filtre
	return f.page, f.errMaps
}

func (f *fakeTacticalSvc) Raster(_ context.Context, carte, question, qui string, filtre *domain.MatchFilterSpec) (domain.TacticalRaster, error) {
	f.vuCarte, f.vuQuestion, f.vuQui, f.vuFiltre = carte, question, qui, filtre
	return f.raster, f.errRast
}

func newTacticalRouter(factory handlers.ServiceFactory[port.TacticalService]) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(sub chi.Router) {
		handlers.NewTacticalHandler(factory).Mount(sub)
	})
	return r
}

func tacticalFactory(svc port.TacticalService, err error) handlers.ServiceFactory[port.TacticalService] {
	return func(context.Context, string) (port.TacticalService, error) { return svc, err }
}

func appel(t *testing.T, r *chi.Mux, url string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
	return w
}

// TestTacticalHandler_Maps : la grille d'entree, servie telle quelle.
func TestTacticalHandler_Maps(t *testing.T) {
	svc := &fakeTacticalSvc{page: domain.TacticalMapsPage{
		PlancherMatchs: domain.PlancherMatchsParCarte,
		Cartes: []domain.TacticalMapCard{
			{MapID: "streets", MapName: "Streets", Matchs: 12, Victoires: 7, Defaites: 5},
			{MapID: "bazaar", MapName: "Bazaar", Matchs: 3, SousPlancher: true},
		},
	}}
	w := appel(t, newTacticalRouter(tacticalFactory(svc, nil)), "/players/JGtm/tactical/maps")
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got domain.TacticalMapsPage
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Cartes) != 2 || got.PlancherMatchs != domain.PlancherMatchsParCarte {
		t.Fatalf("payload inattendu: %+v", got)
	}
	if !got.Cartes[1].SousPlancher {
		t.Errorf("le drapeau sous_plancher doit traverser le contrat: %+v", got.Cartes[1])
	}
}

// TestTacticalHandler_RasterNominal : 200, et les parametres arrivent au service
// tels qu'ils ont ete demandes.
func TestTacticalHandler_RasterNominal(t *testing.T) {
	svc := &fakeTacticalSvc{raster: domain.TacticalRaster{
		MapID: "streets", Question: domain.TacticalQuestionKills, Qui: domain.TacticalQuiEscouade,
		MatchsRetenus: 20, PasM: 0.5,
		Cellules: []domain.CelluleTactique{{Col: 4, Lig: 4, Valeur: 0.5, Brut: 10, Matchs: 5}},
	}}
	r := newTacticalRouter(tacticalFactory(svc, nil))

	w := appel(t, r, "/players/JGtm/tactical/streets/raster?question=kills&qui=escouade")
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if svc.vuCarte != "streets" || svc.vuQuestion != "kills" || svc.vuQui != "escouade" {
		t.Errorf("parametres transmis = %q/%q/%q", svc.vuCarte, svc.vuQuestion, svc.vuQui)
	}
	var got domain.TacticalRaster
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.MatchsRetenus != 20 || len(got.Cellules) != 1 || got.PasM != 0.5 {
		t.Errorf("payload inattendu: %+v", got)
	}
	if got.Echange != nil {
		t.Errorf("Echange absent doit etre OMIS du JSON, pas rendu a zero: %+v", got.Echange)
	}
}

// TestTacticalHandler_RasterDefauts : sans parametre, la lecture par defaut est
// « ou je meurs », axe « moi ».
func TestTacticalHandler_RasterDefauts(t *testing.T) {
	svc := &fakeTacticalSvc{}
	r := newTacticalRouter(tacticalFactory(svc, nil))
	if w := appel(t, r, "/players/JGtm/tactical/streets/raster"); w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if svc.vuQuestion != domain.TacticalQuestionMorts || svc.vuQui != domain.TacticalQuiMoi {
		t.Errorf("defauts = %q/%q, want morts/moi", svc.vuQuestion, svc.vuQui)
	}
	if svc.vuFiltre != nil {
		t.Errorf("aucun filtre demande : spec = %+v, want nil", svc.vuFiltre)
	}
}

// TestTacticalHandler_FiltreExplorateur : le vocabulaire de l'Explorateur est lu
// et transmis ; un axe MAL FORME est ignore, jamais un 400 (un lien partage ne doit
// pas casser sur une valeur douteuse).
func TestTacticalHandler_FiltreExplorateur(t *testing.T) {
	svc := &fakeTacticalSvc{}
	r := newTacticalRouter(tacticalFactory(svc, nil))
	w := appel(t, r, "/players/JGtm/tactical/streets/raster?"+
		"playlist=Ranked%20Arena,Quick%20Play&mode=Assassin&outcome=win"+
		"&from=2026-08-01T00:00:00Z&with_player=pas-un-xuid")
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	f := svc.vuFiltre
	if f == nil {
		t.Fatal("filtre non transmis")
	}
	if len(f.PlaylistNames) != 2 || f.PlaylistNames[0] != "Ranked Arena" {
		t.Errorf("playlists = %v", f.PlaylistNames)
	}
	if len(f.ModeCategories) != 1 || f.ModeCategories[0] != "Assassin" {
		t.Errorf("modes = %v", f.ModeCategories)
	}
	if f.Outcome == nil || *f.Outcome != "win" {
		t.Errorf("outcome = %v", f.Outcome)
	}
	if f.DateFrom == nil || !f.DateFrom.Equal(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("from = %v", f.DateFrom)
	}
	if f.WithPlayerXuid != nil {
		t.Errorf("with_player mal forme doit etre IGNORE, pas transmis: %v", *f.WithPlayerXuid)
	}
}

// TestTacticalHandler_CarteInconnue404 : une carte que ce joueur n'a pas jouee sous
// ce filtre est un 404 — jamais une lecture vide qui se lirait comme « rien ne s'y
// passe ».
func TestTacticalHandler_CarteInconnue404(t *testing.T) {
	svc := &fakeTacticalSvc{errRast: domain.ErrTacticalCarteInconnue}
	w := appel(t, newTacticalRouter(tacticalFactory(svc, nil)), "/players/JGtm/tactical/inconnue/raster")
	if w.Code != http.StatusNotFound {
		t.Errorf("carte inconnue -> 404, got %d (%s)", w.Code, w.Body.String())
	}
}

// TestTacticalHandler_QuestionInconnue400 / AxeInconnu400 : refus types du service.
func TestTacticalHandler_QuestionInconnue400(t *testing.T) {
	svc := &fakeTacticalSvc{errRast: domain.ErrTacticalQuestionInconnue}
	w := appel(t, newTacticalRouter(tacticalFactory(svc, nil)), "/players/JGtm/tactical/streets/raster?question=temps")
	if w.Code != http.StatusBadRequest {
		t.Errorf("question inconnue -> 400, got %d (%s)", w.Code, w.Body.String())
	}
	if svc.vuQuestion != "temps" {
		t.Errorf("une question hors vocabulaire ne doit PAS etre remplacee par le defaut: %q", svc.vuQuestion)
	}
}

func TestTacticalHandler_AxeInconnu400(t *testing.T) {
	svc := &fakeTacticalSvc{errRast: domain.ErrTacticalQuiInconnu}
	w := appel(t, newTacticalRouter(tacticalFactory(svc, nil)), "/players/JGtm/tactical/streets/raster?qui=tout-le-monde")
	if w.Code != http.StatusBadRequest {
		t.Errorf("axe inconnu -> 400, got %d (%s)", w.Code, w.Body.String())
	}
}

// TestTacticalHandler_CapabilityNotSupported503 : titre sans `film.kill_positions`
// — 503 propre par le helper central, jamais un 500 ni une panique.
func TestTacticalHandler_CapabilityNotSupported503(t *testing.T) {
	svc := &fakeTacticalSvc{errRast: games.ErrCapabilityNotSupported}
	w := appel(t, newTacticalRouter(tacticalFactory(svc, nil)), "/players/JGtm/tactical/streets/raster")
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("capability absente -> 503, got %d (%s)", w.Code, w.Body.String())
	}
}

// TestTacticalHandler_PlayerNotFound404 : joueur inconnu.
func TestTacticalHandler_PlayerNotFound404(t *testing.T) {
	r := newTacticalRouter(tacticalFactory(nil, errors.New("no such player")))
	for _, url := range []string{
		"/players/inconnu/tactical/maps",
		"/players/inconnu/tactical/streets/raster",
	} {
		if w := appel(t, r, url); w.Code != http.StatusNotFound {
			t.Errorf("%s: joueur inconnu -> 404, got %d (%s)", url, w.Code, w.Body.String())
		}
	}
}

// TestTacticalHandler_ErreurInterne500 : toute autre panne reste un 500.
func TestTacticalHandler_ErreurInterne500(t *testing.T) {
	svc := &fakeTacticalSvc{errMaps: errors.New("base illisible")}
	w := appel(t, newTacticalRouter(tacticalFactory(svc, nil)), "/players/JGtm/tactical/maps")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("panne -> 500, got %d (%s)", w.Code, w.Body.String())
	}
}
