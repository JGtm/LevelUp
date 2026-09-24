package replay

// sieges_capacite_test.go — LA CAPACITE D'UNE EQUIPE ET LES PLACES OUVERTES (lot M2.3,
// 2026-09-23). Une equipe a un nombre FINI de places — la taille d'equipe du mode, que le film
// n'ecrit pas (ou que rien ne lit) : la pose l'ESTIME par trois bornes basses, et un arrivant qui ne
// trouve aucune place libre en OUVRE une tant que son equipe est sous cette capacite.
//
//	P-OUVERTE        la table du debut a un joueur de moins dans une equipe (`e5adf7b2` : 23
//	                 sieges pour 12 contre 12) : l'arrivant de cette equipe ouvre la place qui
//	                 manquait, par le repli compte ; une equipe pleine n'en ouvre pas ;
//	P-NUMERO         une place ouverte prend l'index de son arrivant, sauf si une place le porte
//	                 deja : elle prend alors un numero au-dela des 64 index ;
//	P-CAPACITE-TABLE une equipe a qui il manque DEUX joueurs a la table garde la taille de la plus
//	                 grande equipe de la table, pas la moyenne des sieges ;
//	P-CAPACITE-LUE   les deux equipes incompletes a la table : la capacite monte au plus grand
//	                 nombre d'entites que le film montre ensemble dans l'equipe, et pas au-dela.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// TestPlaceOuverteQuandLaTableEstIncomplete (P-OUVERTE).
func TestPlaceOuverteQuandLaTableEstIncomplete(t *testing.T) {
	roster := []RosterEntry{
		entree(0, "100"),  // equipe 0
		entree(1, "110"),  // equipe 0 : deux places
		entree(2, "200"),  // equipe 1 : une seule a la table du debut
		entree(9, "900"),  // arrive a 30 dans l'equipe 1
		entree(10, "910"), // arrive a 40 dans l'equipe 0, pleine
	}
	occ := occupantsFabriques([]int{0, 0, 1, 1, 0}, iv(0, 99), iv(0, 99), iv(0, 99), iv(30, 99), iv(40, 99))
	fb := fallback.NouveauCompteur()
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0, 1, 2),
		horloge: horlogeDeSieges(fb)})

	if roster[3].Seat != 9 || roster[3].SeatSource != SeatSourceOuverte {
		t.Fatalf("arrivant de l'equipe 1 : place %d (%s), attendu 9 (%s) — son equipe n'avait qu'une "+
			"place sur deux", roster[3].Seat, roster[3].SeatSource, SeatSourceOuverte)
	}
	if roster[4].SeatSource != SeatSourceIndex {
		t.Errorf("arrivant de l'equipe 0 : %q — son equipe tient deja ses deux places", roster[4].SeatSource)
	}
	if n := fb.Compte(fallback.NomPlaceOuverteSousLaCapaciteEstimee); n != 1 {
		t.Errorf("repli declenche %d fois, attendu 1", n)
	}
	if cov.PlacesOuvertes != 1 || cov.SansPlace != 1 || cov.Apparies != 0 {
		t.Errorf("couverture %+v : attendu 1 place ouverte, 1 sans place, 0 appariee", cov)
	}
}

// TestPlaceOuverteNeReprendPasUnNumeroDePlace (P-NUMERO) : deux arrivants de deux equipes portent
// le meme index 9 (des bots qui se relaient) ; le second ne peut pas tenir la place 9, tenue par
// l'autre equipe — il ouvre la place 64.
func TestPlaceOuverteNeReprendPasUnNumeroDePlace(t *testing.T) {
	roster := []RosterEntry{entree(0, "100"), entree(1, "200"), entree(9, "900"), entree(9, "910")}
	occ := occupantsFabriques([]int{0, 1, 0, 1}, iv(0, 99), iv(0, 99), iv(10, 99), iv(20, 99))
	occ.simultanees = map[int]int{0: 2, 1: 2}
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0, 1),
		horloge: horlogeDeSieges(nil)})

	if roster[2].Seat != 9 || roster[3].Seat != premierePlaceHorsIndex {
		t.Fatalf("places %d et %d : attendu 9 puis %d (au-dela des index)", roster[2].Seat,
			roster[3].Seat, premierePlaceHorsIndex)
	}
	if cov.PlacesOuvertes != 2 || cov.Depassements != 0 || cov.Sieges != 4 {
		t.Errorf("couverture %+v : 2 places ouvertes, 4 places distinctes, aucun depassement", cov)
	}
}

// TestCapaciteDeLaPlusGrandeEquipeDeLaTable (P-CAPACITE-TABLE) : quatre contre quatre a la table,
// moins deux joueurs de l'equipe 1 (6 sieges) — `ceil(6 / 2)` vaudrait 3 ; la capacite est 4, et
// les deux arrivants de l'equipe 1 ouvrent chacun sa place.
func TestCapaciteDeLaPlusGrandeEquipeDeLaTable(t *testing.T) {
	roster := []RosterEntry{
		entree(0, "100"), entree(1, "110"), entree(2, "120"), entree(3, "130"), // equipe 0
		entree(4, "200"), entree(5, "210"), // equipe 1
		entree(9, "900"), entree(10, "910"), // arrivants de l'equipe 1
	}
	occ := occupantsFabriques([]int{0, 0, 0, 0, 1, 1, 1, 1}, iv(0, 99), iv(0, 99), iv(0, 99),
		iv(0, 99), iv(0, 99), iv(0, 99), iv(20, 99), iv(30, 99))
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0, 1, 2, 3, 4, 5),
		horloge: horlogeDeSieges(nil)})

	if roster[6].SeatSource != SeatSourceOuverte || roster[7].SeatSource != SeatSourceOuverte {
		t.Fatalf("arrivants : %q et %q, attendu deux places ouvertes — l'equipe 1 a la taille de "+
			"l'equipe 0", roster[6].SeatSource, roster[7].SeatSource)
	}
	if cov.SansPlace != 0 || cov.Depassements != 0 {
		t.Errorf("couverture %+v : aucun arrivant sans place, aucun depassement", cov)
	}
}

// TestCapaciteLueDansLesEntites (P-CAPACITE-LUE) : un contre un a la table, deux contre deux dans
// le film. L'equipe 0 montre deux entites ensemble : son arrivant ouvre une place. L'equipe 1 n'en
// montre qu'une : son arrivant, dont la presence chevauche celle du titulaire, n'en a pas.
func TestCapaciteLueDansLesEntites(t *testing.T) {
	roster := []RosterEntry{entree(0, "100"), entree(1, "200"), entree(9, "900"), entree(10, "910")}
	occ := occupantsFabriques([]int{0, 1, 0, 1}, iv(0, 99), iv(0, 99), iv(20, 99), iv(30, 99))
	occ.simultanees = map[int]int{0: 2, 1: 1}
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0, 1),
		horloge: horlogeDeSieges(nil)})

	if roster[2].SeatSource != SeatSourceOuverte {
		t.Errorf("arrivant de l'equipe 0 : %q, attendu %q (deux entites lues ensemble)",
			roster[2].SeatSource, SeatSourceOuverte)
	}
	if roster[3].SeatSource != SeatSourceIndex {
		t.Errorf("arrivant de l'equipe 1 : %q, attendu %q — le film ne montre jamais deux "+
			"occupants de l'equipe 1 ensemble", roster[3].SeatSource, SeatSourceIndex)
	}
	if cov.PlacesOuvertes != 1 || cov.SansPlace != 1 {
		t.Errorf("couverture %+v : 1 place ouverte, 1 sans place", cov)
	}
}
