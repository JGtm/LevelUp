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
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
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

// doutesApres rend les doutes d'une entite du temoin a TOUTES les images-cles porteuses qui
// suivent sa derniere lecture : son depart n'est prouve nulle part.
func doutesApres(scan grammar.PlayerEntityScan, slot int) []grammar.DouteDAbsence {
	var out []grammar.DouteDAbsence
	for _, e := range scan.Entities {
		if e.Slot != slot {
			continue
		}
		for r := e.LastKF + 1; r < len(scan.KeyframesUS); r++ {
			out = append(out, grammar.DouteDAbsence{Rang: r, Slot: slot})
		}
	}
	return out
}

// posesSousDoute pose le temoin `b1ad85eb` dont les departs de FairyNectar (slot 1299) et de Hanover
// Cat (slot 1610) ne sont PROUVES par aucune image-cle, et rend le roster, la couverture et le
// compteur de replis. `sansBrewDog` retire le remplacant de FairyNectar.
func posesSousDoute(t *testing.T, sansBrewDog bool) ([]RosterEntry, SeatCoverage, *fallback.Compteur) {
	t.Helper()
	roster, tracks, occIn, placeIn := temoinB1ad85eb()
	var doutes []grammar.DouteDAbsence
	doutes = append(doutes, doutesApres(occIn.scan, 1299)...)
	doutes = append(doutes, doutesApres(occIn.scan, 1610)...)
	sort.Slice(doutes, func(i, j int) bool {
		if doutes[i].Rang != doutes[j].Rang {
			return doutes[i].Rang < doutes[j].Rang
		}
		return doutes[i].Slot < doutes[j].Slot
	})
	occIn.scan.Doutes = doutes
	if sansBrewDog {
		occIn.bots = occIn.bots[:2]
		roster = roster[:len(roster)-1]
		tracks = tracks[:len(tracks)-1]
		var ents []grammar.PlayerEntity
		for _, e := range occIn.scan.Entities {
			if e.Slot != 2145 {
				ents = append(ents, e)
			}
		}
		occIn.scan.Entities = ents
	}
	fb := fallback.NouveauCompteur()
	placeIn.horloge.fb = fb
	occ := lierLesOccupants(roster, tracks, occIn)
	var pub teamPublication
	pub.poserEquipesParEntree(roster, occ)
	cov := poserLesSieges(roster, occ, placeIn)
	return roster, cov, fb
}

// verifierLaCapacite tient la regle des places a chaque frame : jamais plus de 4 fiches affichees
// par equipe (le temoin est un 4 contre 4).
func verifierLaCapacite(t *testing.T, roster []RosterEntry) {
	t.Helper()
	for f := 0; f < temoinFrames; f++ {
		for eq, noms := range affichesA(roster, f) {
			if len(noms) > 4 {
				t.Fatalf("frame %d : l'equipe %d affiche %d fiches %v — jamais plus que ses 4 places",
					f, eq, len(noms), noms)
			}
		}
	}
}

// TestUnDepartDouteuxEstBorneParLeSuccesseurDeSaPlace (constat DFIX-R7) : le depart non prouve
// recule jusqu'au bout du film, puis la regle des places le borne a la veille de l'arrivee du
// remplacant sur SA place — FairyNectar a f3154 (Brew Dog, place 1), Hanover Cat a f773 (343
// PardonMy, place 5). Le repli nomme compte les deux bornes differees.
func TestUnDepartDouteuxEstBorneParLeSuccesseurDeSaPlace(t *testing.T) {
	roster, cov, fb := posesSousDoute(t, false)
	attendu := map[string]int{"FairyNectar5788": 3154, "Hanover Cat": 773}
	for _, e := range roster {
		fin, ok := attendu[e.Name]
		if !ok {
			continue
		}
		if len(e.Presence) != 1 || e.Presence[0].ToMax == nil || *e.Presence[0].ToMax != fin {
			t.Errorf("%s : presence %+v, attendu affichee jusqu'a %d (veille du successeur)", e.Name,
				e.Presence, fin)
		}
	}
	verifierLaCapacite(t, roster)
	if cov.BornesDifferees != 2 || fb.Compte(fallback.NomBorneDePresenceDiffereeSurDoute) != 2 {
		t.Errorf("bornes differees %d, repli compte %d : attendu 2 et 2", cov.BornesDifferees,
			fb.Compte(fallback.NomBorneDePresenceDiffereeSurDoute))
	}
	if cov.Depassements != 0 || cov.Chevauchements != 0 {
		t.Errorf("couverture %+v : ni depassement ni chevauchement sous doute", cov)
	}
}

// TestUnDepartDouteuxSansSuccesseurCourtJusquAuBout (constat DFIX-R7) : sans remplacant sur sa
// place, le depart non prouve de FairyNectar court jusqu'au bout du film — sa part CERTAINE reste
// sa derniere vie — et la regle des places tient (son equipe garde 4 fiches au plus).
func TestUnDepartDouteuxSansSuccesseurCourtJusquAuBout(t *testing.T) {
	roster, cov, _ := posesSousDoute(t, true)
	for _, e := range roster {
		if e.Name != "FairyNectar5788" {
			continue
		}
		if len(e.Presence) != 1 || e.Presence[0].ToMax == nil || *e.Presence[0].ToMax != temoinFrames-1 ||
			e.Presence[0].To >= temoinFrames-1 {
			t.Errorf("FairyNectar : presence %+v, attendu affichee jusqu'a %d, certaine avant", e.Presence,
				temoinFrames-1)
		}
	}
	verifierLaCapacite(t, roster)
	if cov.Depassements != 0 {
		t.Errorf("couverture %+v : aucun depassement de capacite", cov)
	}
}
