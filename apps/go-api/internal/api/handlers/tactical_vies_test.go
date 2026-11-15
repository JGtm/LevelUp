package handlers_test

// tactical_vies_test.go — le parametre `vies` traverse les deux lectures (raster et cellule),
// et une valeur hors vocabulaire est un 400 type.

import (
	"net/http"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
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
