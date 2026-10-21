package replay

// sieges_a_la_taille_test.go — AUCUNE EQUIPE NE TIENT PLUS DE PLACES QUE SA TAILLE (controle n° 8
// des controles de la v8). Le temoin reel est `4f77afc1` (BTB, Eagle : 13 places pour 12) ; ces
// cas le reduisent a une equipe de deux :
//
//	T-TAILLE   la table du debut porte trois index pour une equipe de deux : un partant d'avant le
//	           coup d'envoi (index 2), un joueur absent qui reviendra (index 1, tenu en attendant par un
//	           bot), et l'arrivant que le jeu met sur l'index du partant. L'arrivant releve le bot sur
//	           la place gardee, le partant n'est plus affiche : deux fiches ;
//	T-ESTIMEE  sans taille de playlist, rien ne bouge : la capacite estimee vaut les trois places.

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// rosterDeLaTaille rend le roster et les occupants du cas T-TAILLE.
func rosterDeLaTaille() ([]RosterEntry, occupants) {
	roster := []RosterEntry{
		entree(0, "100"), // A, la du debut a la fin
		entree(1, "110"), // T, absent puis de retour a 80
		entree(2, "120"), // L, parti avant le coup d'envoi
		{Bot: true, Name: "343 R", FilmIndex: 30},
		entree(40, "140"), // H, arrive a 30 alors que l'equipe est pleine
	}
	partant := []intervalleDePresence{{de: 0, a: -1, aMax: 10}}
	occ := occupantsFabriques([]int{0, 0, 0, 0, 0}, iv(0, 99), iv(80, 99), partant, iv(0, 60), iv(30, 75))
	occ.parEntree[4].vies = [][2]int{{32, 75}} // H joue des son arrivee : il releve le bot
	return roster, occ
}

// TestEquipeRameneeASaTaille (T-TAILLE).
func TestEquipeRameneeASaTaille(t *testing.T) {
	roster, occ := rosterDeLaTaille()
	fb := fallback.NouveauCompteur()
	cov := poserLesSieges(context.Background(), roster, occ, entreesDesPlaces{table: tableDeDebut(0, 1, 2),
		horloge: horlogeDeSieges(fb), tailleDEquipe: 2})

	places := map[int]bool{}
	for _, e := range roster {
		if len(e.Presence) > 0 {
			places[e.Seat] = true
		}
	}
	if len(places) != 2 {
		t.Fatalf("places affichees %v : attendu 2 (la taille de l'equipe)", places)
	}
	if roster[4].Seat != roster[3].Seat || roster[4].SeatSource != SeatSourceTaille {
		t.Errorf("H : place %d (%s), attendu la place du bot %d (%s)", roster[4].Seat, roster[4].SeatSource,
			roster[3].Seat, SeatSourceTaille)
	}
	if p := roster[3].Presence; len(p) != 1 || p[0].To != 29 {
		t.Errorf("bot releve : presence %+v, attendu jusqu'a la veille de l'arrivee de H (29)", p)
	}
	if len(roster[2].Presence) != 0 {
		t.Errorf("partant d'avant le coup d'envoi encore affiche : %+v", roster[2].Presence)
	}
	if cov.CapaciteSource != capaciteDuMode || cov.RameneesALaTaille != 2 || cov.BotsRelevesALaTaille != 1 ||
		cov.PlacesEnTrop != 0 {
		t.Errorf("couverture %+v : source mode, 2 entrees ramenees, 1 bot releve, 0 place en trop", cov)
	}
	if fb.Compte(fallback.NomTailleDEquipeDuMode) != 1 || fb.Compte(fallback.NomPlaceRameneeALaTailleDEquipe) != 2 {
		t.Errorf("replis : taille %d, ramenees %d — attendu 1 et 2",
			fb.Compte(fallback.NomTailleDEquipeDuMode), fb.Compte(fallback.NomPlaceRameneeALaTailleDEquipe))
	}
}

// TestSansTailleDeLaPlaylistRienNeBouge (T-ESTIMEE).
func TestSansTailleDeLaPlaylistRienNeBouge(t *testing.T) {
	roster, occ := rosterDeLaTaille()
	fb := fallback.NouveauCompteur()
	cov := poserLesSieges(context.Background(), roster, occ, entreesDesPlaces{table: tableDeDebut(0, 1, 2),
		horloge: horlogeDeSieges(fb)})

	if cov.CapaciteSource != capaciteEstimee || cov.RameneesALaTaille != 0 {
		t.Errorf("couverture %+v : capacite estimee, aucune entree ramenee", cov)
	}
	for i, e := range roster {
		if e.SeatSource == SeatSourceTaille {
			t.Errorf("entree %d ramenee sans taille de playlist", i)
		}
	}
	if fb.Compte(fallback.NomTailleDEquipeDuMode) != 0 {
		t.Error("repli de la taille du mode declenche sans taille")
	}
}
