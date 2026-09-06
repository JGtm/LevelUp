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
	vuScope    domain.TacticalScope
	vuAppele   bool
}

func (f *fakeTacticalSvc) MapsPlayed(_ context.Context, scope domain.TacticalScope) (domain.TacticalMapsPage, error) {
	f.vuScope, f.vuAppele = scope, true
	return f.page, f.errMaps
}

func (f *fakeTacticalSvc) Raster(_ context.Context, req domain.TacticalRasterRequest) (domain.TacticalRaster, error) {
	f.vuCarte, f.vuQuestion, f.vuQui = req.MapID, req.Question, req.Qui
	f.vuScope, f.vuAppele = req.Scope, true
	return f.raster, f.errRast
}

// newTacticalRouter monte l'onglet avec un service de rejeu qui n'a AUCUN fond : les tests
// des lectures tactiques ne doivent rien devoir au fond de carte. Les tests du fond
// utilisent newTacticalRouterAvecFond (tactical_background_test.go).
func newTacticalRouter(factory handlers.ServiceFactory[port.TacticalService]) *chi.Mux {
	return newTacticalRouterAvecFond(factory,
		tacticalReplayFactory(&mockReplayService{bgMapErr: port.ErrMapBackgroundNotAvailable}, nil))
}

func newTacticalRouterAvecFond(
	factory handlers.ServiceFactory[port.TacticalService],
	replay handlers.ServiceFactory[port.ReplayService],
) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(sub chi.Router) {
		handlers.NewTacticalHandler(factory, replay).Mount(sub)
	})
	return r
}

func tacticalReplayFactory(svc port.ReplayService, err error) handlers.ServiceFactory[port.ReplayService] {
	return func(context.Context, string) (port.ReplayService, error) { return svc, err }
}

func tacticalFactory(svc port.TacticalService, err error) handlers.ServiceFactory[port.TacticalService] {
	return func(context.Context, string) (port.TacticalService, error) { return svc, err }
}

// appel : les deux routes du FOND restent des GET (une image et son calage n'ont pas
// de perimetre).
func appel(t *testing.T, r *chi.Mux, url string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
	return w
}

// appelRaster / appelMaps : les deux LECTURES sont des POST depuis la phase 4 bis —
// leur perimetre est une liste de match_id, qui ne tient pas dans une query string.
// Corps `{}` = perimetre vide, ce qui suffit partout ou seul le routage ou le refus
// est observe.
func appelPost(t *testing.T, r *chi.Mux, url, corps string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(corps))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
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
	w := appelPost(t, newTacticalRouter(tacticalFactory(svc, nil)),
		"/players/JGtm/tactical/maps", `{"match_ids":["m1","m2"]}`)
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

// TestTacticalHandler_MapsPerimetre : la grille d'entree lit le MEME perimetre que la
// lecture de placement — liste blanche ET composition — et le transmet au service.
func TestTacticalHandler_MapsPerimetre(t *testing.T) {
	svc := &fakeTacticalSvc{}
	r := newTacticalRouter(tacticalFactory(svc, nil))
	w := appelPost(t, r, "/players/JGtm/tactical/maps",
		`{"match_ids":["m1","m2","m3"],"coequipiers":["xuid(42)"]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if len(svc.vuScope.MatchIDs) != 3 || svc.vuScope.MatchIDs[0] != "m1" {
		t.Errorf("match_ids transmis = %v, want les trois du corps", svc.vuScope.MatchIDs)
	}
	if len(svc.vuScope.Coequipiers) != 1 || svc.vuScope.Coequipiers[0] != "xuid(42)" {
		t.Errorf("coequipiers transmis = %v", svc.vuScope.Coequipiers)
	}
}

// TestTacticalHandler_MapsSansMatchIDs : un corps sans `match_ids` vaut PERIMETRE
// VIDE, pas « tout l'historique ». Le service recoit une liste vide et decide ; le
// handler n'invente aucun defaut permissif.
func TestTacticalHandler_MapsSansMatchIDs(t *testing.T) {
	svc := &fakeTacticalSvc{}
	r := newTacticalRouter(tacticalFactory(svc, nil))
	if w := appelPost(t, r, "/players/JGtm/tactical/maps", `{}`); w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !svc.vuAppele {
		t.Fatal("le service n'a pas ete appele")
	}
	if len(svc.vuScope.MatchIDs) != 0 {
		t.Errorf("match_ids = %v, want vide", svc.vuScope.MatchIDs)
	}
}

// TestTacticalHandler_RasterNominal : 200, et les parametres arrivent au service
// tels qu'ils ont ete demandes.
func TestTacticalHandler_RasterNominal(t *testing.T) {
	svc := &fakeTacticalSvc{raster: domain.TacticalRaster{
		MapID: "streets", Question: domain.TacticalQuestionKills, Qui: domain.TacticalQuiEscouade,
		MatchsRetenus: 20, PasM: 0.5,
		EvenementsJournal: 30, EvenementsLocalises: 24,
		Cellules: []domain.CelluleTactique{{Col: 4, Lig: 4, Valeur: 0.5, Brut: 10, Matchs: 5}},
	}}
	r := newTacticalRouter(tacticalFactory(svc, nil))

	w := appelPost(t, r, "/players/JGtm/tactical/streets/raster",
		`{"match_ids":["m1"],"coequipiers":["xuid(7)"],"question":"kills","qui":"escouade"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if svc.vuCarte != "streets" || svc.vuQuestion != "kills" || svc.vuQui != "escouade" {
		t.Errorf("parametres transmis = %q/%q/%q", svc.vuCarte, svc.vuQuestion, svc.vuQui)
	}
	if len(svc.vuScope.MatchIDs) != 1 || len(svc.vuScope.Coequipiers) != 1 {
		t.Errorf("perimetre transmis = %+v", svc.vuScope)
	}
	var got domain.TacticalRaster
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.MatchsRetenus != 20 || len(got.Cellules) != 1 || got.PasM != 0.5 {
		t.Errorf("payload inattendu: %+v", got)
	}
	// La couverture de localisation traverse le contrat : sans elle le pied de carte
	// ne peut pas dire « 30 evenements, 24 localises ».
	if got.EvenementsJournal != 30 || got.EvenementsLocalises != 24 {
		t.Errorf("couverture = %d/%d, want 30/24", got.EvenementsJournal, got.EvenementsLocalises)
	}
	// Les tags JSON sont snake_case sur TOUT le contrat (R4) — un PascalCase isole
	// se verrait ici.
	for _, cle := range []string{`"matchs_retenus"`, `"evenements_journal"`, `"centre_x"`, `"n_cellules"`} {
		if !strings.Contains(w.Body.String(), cle) {
			t.Errorf("clef %s absente du JSON servi : %s", cle, w.Body.String())
		}
	}
	if got.Echange != nil {
		t.Errorf("Echange absent doit etre OMIS du JSON, pas rendu a zero: %+v", got.Echange)
	}
}

// TestTacticalHandler_RasterDefauts : sans question ni axe, la lecture par defaut est
// « ou je meurs », axe « moi ».
func TestTacticalHandler_RasterDefauts(t *testing.T) {
	svc := &fakeTacticalSvc{}
	r := newTacticalRouter(tacticalFactory(svc, nil))
	if w := appelPost(t, r, "/players/JGtm/tactical/streets/raster", `{"match_ids":["m1"]}`); w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if svc.vuQuestion != domain.TacticalQuestionMorts || svc.vuQui != domain.TacticalQuiMoi {
		t.Errorf("defauts = %q/%q, want morts/moi", svc.vuQuestion, svc.vuQui)
	}
}

// TestTacticalHandler_PerimetreOrdonneEtIntegral : la liste blanche traverse le
// contrat SANS PERDRE NI REORDONNER un identifiant. Un decodage qui tronquerait ou
// dedoublonnerait ne se verrait qu'a la lecture des chiffres, page par page.
func TestTacticalHandler_PerimetreOrdonneEtIntegral(t *testing.T) {
	svc := &fakeTacticalSvc{}
	r := newTacticalRouter(tacticalFactory(svc, nil))
	w := appelPost(t, r, "/players/JGtm/tactical/streets/raster",
		`{"match_ids":["m3","m1","m2","m1"]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	want := []string{"m3", "m1", "m2", "m1"}
	if len(svc.vuScope.MatchIDs) != len(want) {
		t.Fatalf("match_ids = %v, want %v", svc.vuScope.MatchIDs, want)
	}
	for i := range want {
		if svc.vuScope.MatchIDs[i] != want[i] {
			t.Fatalf("match_ids = %v, want %v (ordre et integralite du corps)", svc.vuScope.MatchIDs, want)
		}
	}
}

// TestTacticalHandler_CarteInconnue404 : une carte que ce joueur n'a pas jouee dans
// ce perimetre est un 404 — jamais une lecture vide qui se lirait comme « rien ne s'y
// passe ».
func TestTacticalHandler_CarteInconnue404(t *testing.T) {
	svc := &fakeTacticalSvc{errRast: domain.ErrTacticalCarteInconnue}
	w := appelPost(t, newTacticalRouter(tacticalFactory(svc, nil)),
		"/players/JGtm/tactical/inconnue/raster", `{}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("carte inconnue -> 404, got %d (%s)", w.Code, w.Body.String())
	}
}

// TestTacticalHandler_QuestionInconnue400 / AxeInconnu400 : refus types du service.
func TestTacticalHandler_QuestionInconnue400(t *testing.T) {
	svc := &fakeTacticalSvc{errRast: domain.ErrTacticalQuestionInconnue}
	w := appelPost(t, newTacticalRouter(tacticalFactory(svc, nil)),
		"/players/JGtm/tactical/streets/raster", `{"question":"temps"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("question inconnue -> 400, got %d (%s)", w.Code, w.Body.String())
	}
	if svc.vuQuestion != "temps" {
		t.Errorf("une question hors vocabulaire ne doit PAS etre remplacee par le defaut: %q", svc.vuQuestion)
	}
}

func TestTacticalHandler_AxeInconnu400(t *testing.T) {
	svc := &fakeTacticalSvc{errRast: domain.ErrTacticalQuiInconnu}
	w := appelPost(t, newTacticalRouter(tacticalFactory(svc, nil)),
		"/players/JGtm/tactical/streets/raster", `{"qui":"tout-le-monde"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("axe inconnu -> 400, got %d (%s)", w.Code, w.Body.String())
	}
}

// TestTacticalHandler_EscouadeSansComposition400 : l'axe « escouade » demande sans
// coequipiers est un 400 PROPRE, avec son PROPRE code — l'axe existe, c'est la
// composition qui manque. `tactical_axis_unknown` enverrait le client corriger le
// mauvais parametre.
func TestTacticalHandler_EscouadeSansComposition400(t *testing.T) {
	svc := &fakeTacticalSvc{errRast: domain.ErrTacticalEscouadeSansComposition}
	w := appelPost(t, newTacticalRouter(tacticalFactory(svc, nil)),
		"/players/JGtm/tactical/streets/raster", `{"qui":"escouade"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("escouade sans composition -> 400, got %d (%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "tactical_squad_axis_without_composition") {
		t.Errorf("code = %s, want tactical_squad_axis_without_composition", w.Body.String())
	}
}

// TestTacticalHandler_CapabilityNotSupported503 : titre sans `film.kill_positions`
// — 503 propre par le helper central, jamais un 500 ni une panique.
func TestTacticalHandler_CapabilityNotSupported503(t *testing.T) {
	svc := &fakeTacticalSvc{errRast: games.ErrCapabilityNotSupported}
	w := appelPost(t, newTacticalRouter(tacticalFactory(svc, nil)),
		"/players/JGtm/tactical/streets/raster", `{}`)
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
		if w := appelPost(t, r, url, `{}`); w.Code != http.StatusNotFound {
			t.Errorf("%s: joueur inconnu -> 404, got %d (%s)", url, w.Code, w.Body.String())
		}
	}
}

// TestTacticalHandler_ErreurInterne500 : toute autre panne reste un 500.
func TestTacticalHandler_ErreurInterne500(t *testing.T) {
	svc := &fakeTacticalSvc{errMaps: errors.New("base illisible")}
	w := appelPost(t, newTacticalRouter(tacticalFactory(svc, nil)),
		"/players/JGtm/tactical/maps", `{}`)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("panne -> 500, got %d (%s)", w.Code, w.Body.String())
	}
}

// TestTacticalHandler_CompositionInvalide400 : une composition hors bornes ou mal
// formee est un 400 PROPRE, avec son code — et la valeur refusee est nommee (c'est
// ce qui rend le 400 utile : l'appelant sait quoi corriger).
func TestTacticalHandler_CompositionInvalide400(t *testing.T) {
	svc := &fakeTacticalSvc{errRast: domain.ErrTacticalCompositionInvalide, errMaps: domain.ErrTacticalCompositionInvalide}
	r := newTacticalRouter(tacticalFactory(svc, nil))

	for _, url := range []string{
		"/players/JGtm/tactical/streets/raster",
		"/players/JGtm/tactical/maps",
	} {
		w := appelPost(t, r, url, `{"coequipiers":["1","2","3","4"]}`)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s : status=%d, want 400 — body=%s", url, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "tactical_composition_invalid") {
			t.Errorf("%s : code = %s, want tactical_composition_invalid", url, w.Body.String())
		}
	}
}

// TestTacticalHandler_QuestionTemps : la question `temps` (occupation) traverse le
// contrat comme les trois autres — le handler ne connait aucun vocabulaire, il transmet.
func TestTacticalHandler_QuestionTemps(t *testing.T) {
	svc := &fakeTacticalSvc{raster: domain.TacticalRaster{
		MapID: "streets", Question: domain.TacticalQuestionTemps, Qui: domain.TacticalQuiMoi,
		MatchsFiltres: 4, MatchsRetenus: 3, PasM: 0.5,
		Cellules: []domain.CelluleTactique{{Col: 2, Lig: 3, Valeur: 1.5, Brut: 18, Matchs: 3}},
	}}
	r := newTacticalRouter(tacticalFactory(svc, nil))
	w := appelPost(t, r, "/players/JGtm/tactical/streets/raster",
		`{"match_ids":["m1","m2","m3","m4"],"question":"temps"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if svc.vuQuestion != domain.TacticalQuestionTemps {
		t.Fatalf("question transmise = %q, attendu %q", svc.vuQuestion, domain.TacticalQuestionTemps)
	}
	var got domain.TacticalRaster
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Les DEUX denominateurs traversent : sans eux le pied de carte ne peut pas dire
	// « 3 matchs mesures sur 4 », qui est toute la couverture d'une lecture d'occupation.
	if got.MatchsFiltres != 4 || got.MatchsRetenus != 3 {
		t.Fatalf("denominateurs = %d/%d, attendu 3 sur 4", got.MatchsRetenus, got.MatchsFiltres)
	}
	if len(got.Cellules) != 1 || got.Cellules[0].Valeur != 1.5 || got.Cellules[0].Brut != 18 {
		t.Fatalf("cellules = %+v : la valeur (secondes) et le brut (echantillons) doivent traverser ensemble",
			got.Cellules)
	}
}

// TestTacticalHandler_TempsSansCapability : un titre qui ne produit pas d'artefact de
// rejeu rend 503, jamais 200 avec une carte vide.
func TestTacticalHandler_TempsSansCapability(t *testing.T) {
	svc := &fakeTacticalSvc{errRast: games.ErrCapabilityNotSupported}
	r := newTacticalRouter(tacticalFactory(svc, nil))
	w := appelPost(t, r, "/players/JGtm/tactical/streets/raster", `{"match_ids":["m1"],"question":"temps"}`)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s, attendu 503", w.Code, w.Body.String())
	}
}
