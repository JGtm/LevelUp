package replay

// occupants_doutes_test.go — UNE IMAGE-CLE DOUTEUSE NE CONCLUT NI UNE ARRIVEE TARDIVE NI UN
// DEPART (lot D-fix des retours du rejeu, 2026-09-24).
//
// Le lot M2 posait l'arrivee d'un occupant a la premiere image-cle qui lit son entite et son
// depart a la premiere qui ne la lit plus. Quand la marche d'image-cle perdait le record (une
// fausse ancre elue), l'occupant ARRIVAIT plus tard (`000d5950` : absent 16,3 s au depart). Une
// absence que la marche ne prouve pas (`grammar.PlayerEntityScan.Doutes`) ne borne plus rien : la
// presence recule jusqu'a la premiere absence prouvee, ou jusqu'au bord du film — et se compte.
//
//	D-ARRIVEE  entite lue aux images-cles 1 et 2 d'un film a trois ; image-cle 0 douteuse pour
//	           son slot : la presence part de la frame 0 (sans le doute : de la frame 50) ;
//	D-DEPART   entite lue a l'image-cle 0 ; l'image-cle 1 douteuse, la 2 prouve l'absence :
//	           affichee jusqu'a la veille de la 2 (frame 89 ; sans le doute : 49) ;
//	D-FIN      les images-cles 1 et 2 douteuses : affichee jusqu'au bout, certaine jusqu'a sa
//	           derniere lecture ;
//	D-COMPTE   les deux compteurs de `coverage.seats` : images douteuses et bornes differees.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// presenceDuSeulOccupant lie un humain d'index 4, sans vie, a l'unique entite du balayage et rend
// sa presence et la liaison.
func presenceDuSeulOccupant(t *testing.T, scan grammar.PlayerEntityScan) (intervalleDePresence, occupants) {
	t.Helper()
	occ := lierLesOccupants([]RosterEntry{entree(4, "100")}, nil, entreesDeTest(scan))
	p := occ.parEntree[0].presence
	if len(p) != 1 || !occ.parEntree[0].lue {
		t.Fatalf("presence %+v (lue %v) : attendu UNE presence lue", p, occ.parEntree[0].lue)
	}
	return p[0], occ
}

func TestUneImageCleDouteuseNeConclutPasUneArriveeTardive(t *testing.T) {
	e := grammar.PlayerEntity{Slot: 7, Index: 4, Team: 0, FirstKF: 1, LastKF: 2, Seen: 2}
	sans, _ := presenceDuSeulOccupant(t, scanDeTest([]int{10, 50, 90}, e))
	if sans.de != 50 {
		t.Fatalf("sans doute : presence depuis %d, attendu 50 (l'image-cle 0 prouve l'absence)", sans.de)
	}
	douteux := scanDeTest([]int{10, 50, 90}, e)
	douteux.Doutes = []grammar.DouteDAbsence{{Rang: 0, Slot: 7}}
	avec, occ := presenceDuSeulOccupant(t, douteux)
	if avec.de != 0 {
		t.Fatalf("image-cle 0 douteuse : presence depuis %d, attendu 0 — une absence non prouvee n'est "+
			"pas une arrivee tardive", avec.de)
	}
	if occ.imagesDouteuses != 1 || occ.bornesDifferees != 1 {
		t.Fatalf("images douteuses %d, bornes differees %d : attendu 1 et 1", occ.imagesDouteuses,
			occ.bornesDifferees)
	}
}

func TestUneImageCleDouteuseNeConclutPasUnDepart(t *testing.T) {
	e := grammar.PlayerEntity{Slot: 7, Index: 4, Team: 0, FirstKF: 0, LastKF: 0, Seen: 1}
	sans, _ := presenceDuSeulOccupant(t, scanDeTest([]int{10, 50, 90}, e))
	if sans.aMax != 49 {
		t.Fatalf("sans doute : affichee jusqu'a %d, attendu 49 (veille de l'image-cle 1)", sans.aMax)
	}
	douteux := scanDeTest([]int{10, 50, 90}, e)
	douteux.Doutes = []grammar.DouteDAbsence{{Rang: 1, Slot: 7}}
	avec, _ := presenceDuSeulOccupant(t, douteux)
	if avec.aMax != 89 || avec.a != 10 {
		t.Fatalf("image-cle 1 douteuse : presence %+v, attendu certaine jusqu'a 10 et affichee jusqu'a 89 "+
			"(veille de la premiere absence prouvee)", avec)
	}
	jusquAuBout := scanDeTest([]int{10, 50, 90}, e)
	jusquAuBout.Doutes = []grammar.DouteDAbsence{{Rang: 1, Slot: 7}, {Rang: 2, Slot: 7}}
	fin, occ := presenceDuSeulOccupant(t, jusquAuBout)
	if fin.aMax != siegeFrames-1 || fin.a != 10 {
		t.Fatalf("aucune absence prouvee apres : presence %+v, attendu certaine jusqu'a 10 et affichee "+
			"jusqu'a %d", fin, siegeFrames-1)
	}
	if occ.imagesDouteuses != 2 || occ.bornesDifferees != 1 {
		t.Fatalf("images douteuses %d, bornes differees %d : attendu 2 et 1", occ.imagesDouteuses,
			occ.bornesDifferees)
	}
}

func TestLesDoutesSeComptentDansLaCouvertureDesPlaces(t *testing.T) {
	e := grammar.PlayerEntity{Slot: 7, Index: 4, Team: 0, FirstKF: 1, LastKF: 2, Seen: 2}
	douteux := scanDeTest([]int{10, 50, 90}, e)
	douteux.Doutes = []grammar.DouteDAbsence{{Rang: 0, Slot: 7}}
	roster := []RosterEntry{entree(4, "100")}
	occ := lierLesOccupants(roster, nil, entreesDeTest(douteux))
	cov := poserLesSieges(roster, occ, entreesDesPlaces{horloge: replayClock{step: 100_000, frames: siegeFrames}})
	if cov.ImagesClesDouteuses != 1 || cov.BornesDifferees != 1 {
		t.Fatalf("coverage.seats : images douteuses %d, bornes differees %d — attendu 1 et 1",
			cov.ImagesClesDouteuses, cov.BornesDifferees)
	}
}
