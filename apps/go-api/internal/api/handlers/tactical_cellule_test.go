package handlers_test

// tactical_cellule_test.go — POST /players/{player_slug}/tactical/{map_id}/cellule (lot M1,
// Tactique S.1). Meme famille de tests que tactical_test.go (raster) : la couche handler ne
// fait que decoder, deleguer, traduire les refus — c'est deja verifie ligne a ligne pour
// Raster ; ici on prouve juste le meme branchement pour Cellule.

import (
	"encoding/json"
	"net/http"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// TestTacticalHandler_CelluleNominal : 200, le corps (perimetre + lecture + cellule)
// atteint le service tel quel, et la reponse (contributions + matchs_non_ouvrables)
// traverse le contrat.
func TestTacticalHandler_CelluleNominal(t *testing.T) {
	svc := &fakeTacticalSvc{cellule: domain.TacticalCelluleReponse{
		Contributions: []domain.TacticalContribution{
			{MatchID: "m1", InstantMs: 4200, XUID: "2533274000000001", Clock: domain.TacticalClockMatch},
		},
		MatchsNonOuvrables: 1,
	}}
	r := newTacticalRouter(tacticalFactory(svc, nil))

	w := appelPost(t, r, "/players/JGtm/tactical/streets/cellule",
		`{"match_ids":["m1","m2"],"coequipiers":["xuid(7)"],"question":"kills","qui":"escouade","cellule":{"col":4,"lig":6}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if svc.vuCelluleReq.MapID != "streets" || svc.vuCelluleReq.Question != "kills" || svc.vuCelluleReq.Qui != "escouade" {
		t.Errorf("parametres transmis = %+v", svc.vuCelluleReq)
	}
	if svc.vuCelluleReq.Col != 4 || svc.vuCelluleReq.Lig != 6 {
		t.Errorf("cellule transmise = col=%d lig=%d, want 4/6", svc.vuCelluleReq.Col, svc.vuCelluleReq.Lig)
	}
	if len(svc.vuCelluleReq.Scope.MatchIDs) != 2 || len(svc.vuCelluleReq.Scope.Coequipiers) != 1 {
		t.Errorf("perimetre transmis = %+v", svc.vuCelluleReq.Scope)
	}

	var got domain.TacticalCelluleReponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Contributions) != 1 || got.Contributions[0].MatchID != "m1" || got.Contributions[0].InstantMs != 4200 {
		t.Errorf("contributions = %+v", got.Contributions)
	}
	if got.Contributions[0].Clock != domain.TacticalClockMatch {
		t.Errorf("clock = %q, want %q (traverse le contrat, lot M1b)", got.Contributions[0].Clock, domain.TacticalClockMatch)
	}
	if got.MatchsNonOuvrables != 1 {
		t.Errorf("matchs_non_ouvrables = %d, want 1", got.MatchsNonOuvrables)
	}
}

// TestTacticalHandler_CelluleDefauts : question/qui par defaut, comme le raster.
func TestTacticalHandler_CelluleDefauts(t *testing.T) {
	svc := &fakeTacticalSvc{}
	r := newTacticalRouter(tacticalFactory(svc, nil))
	w := appelPost(t, r, "/players/JGtm/tactical/streets/cellule", `{"match_ids":["m1"],"cellule":{"col":1,"lig":1}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if svc.vuCelluleReq.Question != domain.TacticalQuestionMorts || svc.vuCelluleReq.Qui != domain.TacticalQuiMoi {
		t.Errorf("defauts = %q/%q, want morts/moi", svc.vuCelluleReq.Question, svc.vuCelluleReq.Qui)
	}
}

// TestTacticalHandler_CelluleCarteInconnue404 : meme refus type que le raster.
func TestTacticalHandler_CelluleCarteInconnue404(t *testing.T) {
	svc := &fakeTacticalSvc{errCell: domain.ErrTacticalCarteInconnue}
	w := appelPost(t, newTacticalRouter(tacticalFactory(svc, nil)),
		"/players/JGtm/tactical/inconnue/cellule", `{"cellule":{"col":1,"lig":1}}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("carte inconnue -> 404, got %d (%s)", w.Code, w.Body.String())
	}
}

// TestTacticalHandler_CelluleCapabilityNotSupported503 : meme degradation propre que le
// raster — jamais un 500 ni une panique.
func TestTacticalHandler_CelluleCapabilityNotSupported503(t *testing.T) {
	svc := &fakeTacticalSvc{errCell: games.ErrCapabilityNotSupported}
	w := appelPost(t, newTacticalRouter(tacticalFactory(svc, nil)),
		"/players/JGtm/tactical/streets/cellule", `{"cellule":{"col":1,"lig":1}}`)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("capability absente -> 503, got %d (%s)", w.Code, w.Body.String())
	}
}

// La validation stricte du map_id (MapIDValide) est testee de facon exhaustive dans
// tactical_mapid_test.go (TestTacticalMapID_CelluleRefuseLesChemins), sur la meme table
// de formes hostiles que le raster et le fond de carte.
