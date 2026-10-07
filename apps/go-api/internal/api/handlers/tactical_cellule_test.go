package handlers_test

// tactical_cellule_test.go — POST /players/{player_slug}/tactical/{map_id}/cellule (lot M1,
// Tactique S.1). Meme famille de tests que tactical_test.go (raster) : la couche handler ne
// fait que decoder, deleguer, traduire les refus — c'est deja verifie ligne a ligne pour
// Raster ; ici on prouve juste le meme branchement pour Cellule.

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// TestTacticalHandler_CelluleNominal : 200, le corps (perimetre + lecture + cellule)
// atteint le service tel quel, et la reponse (contributions) traverse le contrat ; le compte
// des matchs non ouvrables n'en fait pas partie (journal du service seulement).
func TestTacticalHandler_CelluleNominal(t *testing.T) {
	svc := &fakeTacticalSvc{cellule: domain.TacticalCelluleReponse{
		Contributions: []domain.TacticalContribution{
			{MatchID: "m1", InstantMs: 4200, XUID: "2533274000000001", Clock: domain.TacticalClockMatch},
		},
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
	if strings.Contains(w.Body.String(), "matchs_non_ouvrables") {
		t.Errorf("la reponse publie matchs_non_ouvrables : %s", w.Body.String())
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

// TestTacticalHandler_CelluleMiniTuile : les champs de la mini-tuile « Rejeu » et le nom de la zone
// traversent le contrat en snake_case.
func TestTacticalHandler_CelluleMiniTuile(t *testing.T) {
	distance := 12.5
	svc := &fakeTacticalSvc{cellule: domain.TacticalCelluleReponse{
		Contributions: []domain.TacticalContribution{{
			MatchID: "m1", InstantMs: 4200, XUID: "2533274000000001", Clock: domain.TacticalClockMatch,
			Face: domain.TacticalFaceMort, AutreGamertag: "Rival", ArmeLabel: "Fusil de combat", ArmeLabelEN: "BR75",
			Placement: &domain.TacticalPlacement{Seul: true, DistanceM: &distance},
			ModeLabel: "Assassin", ScoreLabel: "50 - 42", ScoreKind: "points", ReplayAvailable: true,
		}},
		Zone: &domain.TacticalZoneNom{NomFR: "Nid blindé", NomEN: "Armored Nest"},
	}}
	r := newTacticalRouter(tacticalFactory(svc, nil))
	w := appelPost(t, r, "/players/JGtm/tactical/streets/cellule", `{"match_ids":["m1"],"cellule":{"col":4,"lig":6}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	for _, cle := range []string{
		`"face":"mort"`, `"autre_gamertag":"Rival"`, `"arme_label":"Fusil de combat"`, `"arme_label_en":"BR75"`,
		`"mode_label":"Assassin"`, `"score_label":"50 - 42"`, `"score_kind":"points"`,
		`"placement":{"seul":true,"distance_m":12.5}`, `"replay_available":true`,
		`"zone":{"nom_fr":"Nid blindé","nom_en":"Armored Nest"}`,
	} {
		if !strings.Contains(w.Body.String(), cle) {
			t.Errorf("clef %s absente du JSON servi : %s", cle, w.Body.String())
		}
	}
}
