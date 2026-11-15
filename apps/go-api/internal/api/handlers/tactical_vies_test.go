package handlers_test

// tactical_vies_test.go — le parametre `vies` traverse les deux lectures (raster et cellule),
// et une valeur hors vocabulaire est un 400 type.

import (
	"context"
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

// TestTacticalHandler_ViesTransmisAuRaster : sans cette assertion, l'oublier dans le decodage
// servirait la lecture « depart » sous une bascule « toutes les vies ».
func TestTacticalHandler_ViesTransmisAuRaster(t *testing.T) {
	svc := &fakeTacticalSvc{}
	r := newTacticalRouter(tacticalFactory(svc, nil))
	w := appelPost(t, r, "/players/JGtm/tactical/streets/raster",
		`{"match_ids":["m1"],"question":"routes","spawn":"s+1+1+c03","vies":"  toutes  "}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if svc.vuScope.Vies != domain.TacticalViesToutes {
		t.Fatalf("vies transmises = %q, attendu %q (valeur rognee)", svc.vuScope.Vies, domain.TacticalViesToutes)
	}
}

// TestTacticalHandler_ViesAbsentesRestentVides : le defaut (`depart`) se decide dans le service,
// pas dans le handler.
func TestTacticalHandler_ViesAbsentesRestentVides(t *testing.T) {
	svc := &fakeTacticalSvc{}
	r := newTacticalRouter(tacticalFactory(svc, nil))
	w := appelPost(t, r, "/players/JGtm/tactical/streets/raster", `{"match_ids":["m1"]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if svc.vuScope.Vies != "" {
		t.Fatalf("vies = %q, attendu vide quand le corps n'en porte pas", svc.vuScope.Vies)
	}
}

// TestTacticalHandler_ViesTransmisALaCellule : le detail d'une cellule suit la meme bascule.
func TestTacticalHandler_ViesTransmisALaCellule(t *testing.T) {
	svc := &fakeTacticalSvc{}
	r := newTacticalRouter(tacticalFactory(svc, nil))
	w := appelPost(t, r, "/players/JGtm/tactical/streets/cellule",
		`{"match_ids":["m1"],"question":"routes","spawn":"s+1+1+c03","vies":"toutes","cellule":{"col":1,"lig":1}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if svc.vuCelluleReq.Scope.Vies != domain.TacticalViesToutes {
		t.Fatalf("vies transmises = %q, attendu %q", svc.vuCelluleReq.Scope.Vies, domain.TacticalViesToutes)
	}
}

// TestTacticalHandler_ViesInconnues400 : un refus type avec son propre code, pour les deux
// lectures. La valeur n'est pas remplacee par le defaut avant d'arriver au service.
func TestTacticalHandler_ViesInconnues400(t *testing.T) {
	for _, tc := range []struct{ nom, url, corps string }{
		{"raster", "/players/JGtm/tactical/streets/raster", `{"vies":"autre"}`},
		{"cellule", "/players/JGtm/tactical/streets/cellule", `{"vies":"autre","cellule":{"col":1,"lig":1}}`},
	} {
		svc := &fakeTacticalSvc{errRast: domain.ErrTacticalViesInconnue, errCell: domain.ErrTacticalViesInconnue}
		w := appelPost(t, newTacticalRouter(tacticalFactory(svc, nil)), tc.url, tc.corps)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s : status=%d body=%s, attendu 400", tc.nom, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "tactical_lives_unknown") {
			t.Fatalf("%s : corps = %s, attendu le code tactical_lives_unknown", tc.nom, w.Body.String())
		}
	}
}

// depotQuiNeDoitPasEtreLu : le refus de `toutes` se decide AVANT toute lecture ; un appel au
// depot (methode du port non surchargee, donc nil) fait paniquer le test.
type depotQuiNeDoitPasEtreLu struct{ port.TacticalRepository }

// TestTacticalHandler_ToutesRefuseSurUnTitreSansSpawnForecast : le titre synthetique declare
// `film.replay_artifact` sans `film.spawn_forecast`, donc ses sidecars ne portent que la premiere
// vie. Le vrai service refuse `toutes` pour les deux lectures, et le handler le rend en 503
// propre (jamais un 200 de premieres vies, jamais un 500).
func TestTacticalHandler_ToutesRefuseSurUnTitreSansSpawnForecast(t *testing.T) {
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
	for _, tc := range []struct{ nom, url, corps string }{
		{"raster", "/players/JGtm/tactical/streets/raster", `{"match_ids":["m1"],"question":"routes","vies":"toutes"}`},
		{"cellule", "/players/JGtm/tactical/streets/cellule", `{"match_ids":["m1"],"question":"routes","vies":"toutes","cellule":{"col":1,"lig":1}}`},
	} {
		w := appelPost(t, r, tc.url, tc.corps)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s : status=%d body=%s, attendu 503", tc.nom, w.Code, w.Body.String())
		}
	}
}
