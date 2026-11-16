package handlers_test

// tactical_reapparitions_test.go — la route « reapparitions adverses » : le corps traverse jusqu'au service,
// chaque refus type a son statut et son code, une lecture vide reste un 200 qui dit pourquoi.

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

const urlReapparitions = "/players/JGtm/tactical/streets/reapparitions-adverses"

// reapparitionsFake est la part du double du port tactique qui sert la lecture ; il est embarque dans
// fakeTacticalSvc (tactical_test.go).
type reapparitionsFake struct {
	reapparitions    domain.TacticalReapparitionsAdverses
	errReapparitions error
	vuReapparitions  domain.TacticalReapparitionsAdversesRequest
}

func (f *reapparitionsFake) ReapparitionsAdverses(_ context.Context,
	req domain.TacticalReapparitionsAdversesRequest) (domain.TacticalReapparitionsAdverses, error) {
	f.vuReapparitions = req
	return f.reapparitions, f.errReapparitions
}

func TestTacticalHandler_ReapparitionsAdversesTransmetLeCorps(t *testing.T) {
	svc := &fakeTacticalSvc{}
	w := appelPost(t, newTacticalRouter(tacticalFactory(svc, nil)), urlReapparitions,
		`{"match_ids":["m1","m2"],"coequipiers":["2533274000000002"],"volume_index":7}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	req := svc.vuReapparitions
	if req.MapID != "streets" || req.VolumeIndex != 7 ||
		len(req.Scope.MatchIDs) != 2 || len(req.Scope.Coequipiers) != 1 {
		t.Fatalf("demande vue par le service = %+v", req)
	}
}

// La reponse d'une lecture vide reste un 200 : etat, raison, zones en liste (jamais null).
func TestTacticalHandler_ReapparitionsAdversesVideEstUn200(t *testing.T) {
	svc := &fakeTacticalSvc{reapparitionsFake: reapparitionsFake{reapparitions: domain.TacticalReapparitionsAdverses{
		Etat: domain.TacticalEtatVide, Raison: domain.TacticalRaisonAucunMatchMesure,
		Reference: domain.TacticalRepartitionZones{Zones: []domain.TacticalPartDeZone{}},
		Tenue:     domain.TacticalRepartitionZones{Zones: []domain.TacticalPartDeZone{}},
	}}}
	w := appelPost(t, newTacticalRouter(tacticalFactory(svc, nil)), urlReapparitions, `{"match_ids":["m1"],"volume_index":1}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(got["etat"]) != `"vide"` || string(got["raison"]) != `"aucun_match_mesure"` {
		t.Errorf("etat/raison = %s / %s", got["etat"], got["raison"])
	}
	for _, repartition := range []string{"reference", "tenue"} {
		var r struct {
			Zones *[]json.RawMessage `json:"zones"`
		}
		if err := json.Unmarshal(got[repartition], &r); err != nil || r.Zones == nil {
			t.Errorf("%s : zones doit etre une liste, pas null (%s)", repartition, got[repartition])
		}
	}
}

func TestTacticalHandler_ReapparitionsAdversesRefus(t *testing.T) {
	cas := []struct {
		nom    string
		err    error
		statut int
		code   string
	}{
		{"zone inconnue", domain.ErrTacticalZoneInconnue, http.StatusNotFound, "tactical_zone_unknown"},
		{"carte inconnue", domain.ErrTacticalCarteInconnue, http.StatusNotFound, "tactical_map_unknown"},
		{"composition invalide", domain.ErrTacticalCompositionInvalide, http.StatusBadRequest, "tactical_composition_invalid"},
		{"capability absente", games.ErrCapabilityNotSupported, http.StatusServiceUnavailable, ""},
	}
	for _, c := range cas {
		svc := &fakeTacticalSvc{reapparitionsFake: reapparitionsFake{errReapparitions: c.err}}
		w := appelPost(t, newTacticalRouter(tacticalFactory(svc, nil)), urlReapparitions, `{"match_ids":["m1"],"volume_index":1}`)
		if w.Code != c.statut || !strings.Contains(w.Body.String(), c.code) {
			t.Errorf("%s : status=%d body=%s, attendu %d %q", c.nom, w.Code, w.Body.String(), c.statut, c.code)
		}
	}
}

// Un map_id hors vocabulaire est refuse AVANT le service, avec le 404 d'une carte inconnue.
func TestTacticalHandler_ReapparitionsAdversesMapInvalide(t *testing.T) {
	svc := &fakeTacticalSvc{}
	w := appelPost(t, newTacticalRouter(tacticalFactory(svc, nil)),
		"/players/JGtm/tactical/..%5Cx/reapparitions-adverses", `{"volume_index":1}`)
	if w.Code != http.StatusNotFound || svc.vuAppele {
		t.Fatalf("status=%d appele=%v body=%s, attendu 404 sans appel du service", w.Code, svc.vuAppele, w.Body.String())
	}
}

func TestTacticalHandler_ReapparitionsAdversesJoueurInconnu(t *testing.T) {
	r := newTacticalRouter(tacticalFactory(nil, context.DeadlineExceeded))
	w := appelPost(t, r, urlReapparitions, `{"volume_index":1}`)
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "player_not_found") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

// Sur un titre sans `film.spawn_forecast`, le vrai service refuse AVANT toute lecture : le handler le rend
// en 503 propre (jamais un 200 vide qui se lirait « personne ne reapparait ici »).
func TestTacticalHandler_ReapparitionsAdversesRefuseSurUnTitreSansSpawnForecast(t *testing.T) {
	_, ce, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	caps, err := games.LoadCapabilityMap(filepath.Join(filepath.Dir(ce), "..", "..", "..", "..", ".."), "synthetic_title_b")
	if err != nil {
		t.Fatalf("LoadCapabilityMap: %v", err)
	}
	svc := service.NewTacticalService(depotQuiNeDoitPasEtreLu{}, caps, "2533274000000001")
	r := newTacticalRouter(func(context.Context, string) (port.TacticalService, error) { return svc, nil })
	w := appelPost(t, r, urlReapparitions, `{"match_ids":["m1"],"volume_index":1}`)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s, attendu 503", w.Code, w.Body.String())
	}
}
