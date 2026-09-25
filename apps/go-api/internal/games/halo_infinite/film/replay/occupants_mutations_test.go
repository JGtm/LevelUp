package replay

// occupants_mutations_test.go — LES REGLES QUE QUATRE MUTATIONS LAISSAIENT VERTES (revue adverse du
// lot M2, constat M2-R3, 2026-09-24). Chaque test nomme la mutation qu'il fait rougir.
//
//	G11  « AtEnd ignore »                l'occupant lu a la DERNIERE image-cle porteuse est la
//	                                     jusqu'a la fin du document, pas jusqu'a cette image-cle ;
//	G4b  « humain lie sans condition     deux humains qui se relaient sur un index ont chacun
//	       de temps »                    LEUR entite : celle dont la fenetre contient un debut de vie ;
//	G12  « memeEquipe toujours vrai »    un arrivant ne lit pas dans ses tirs la place d'une AUTRE
//	                                     equipe ;
//	G17  « retirerLesAffichagesVides     un intervalle que la borne au successeur a vide ne se
//	       desactive »                   publie pas.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// G11 : l'entite est lue de la premiere a la derniere image-cle porteuse (f10..f90) ; le document
// dure 100 frames. L'occupant est CERTAIN jusqu'a la fin — f99 —, pas seulement jusqu'a f90.
func TestEntiteLueALaDerniereImageCleTientJusquALaFin(t *testing.T) {
	scan := scanDeTest([]int{10, 50, 90}, grammar.PlayerEntity{Slot: 7, Index: 2, Team: 0, LastKF: 2, Seen: 3})
	occ := lierLesOccupants([]RosterEntry{entree(2, "100")}, []Track{vieDe("100", 12, 60)}, entreesDeTest(scan))
	p := occ.parEntree[0].presence
	if len(p) != 1 || p[0] != (intervalleDePresence{de: 0, a: siegeFrames - 1, aMax: siegeFrames - 1}) {
		t.Fatalf("presence %+v : lu au coup d'envoi ET a la derniere image-cle, il est la de 0 a %d",
			p, siegeFrames-1)
	}
}

// G4b : l'index 3 est tenu par l'humain 100 (entite de f10 a f30), puis REPRIS par l'humain 200
// (entite de f70 a f90). Chacun se lie a l'entite dont la fenetre large contient le debut d'une de
// ses vies — sans condition de temps, les deux revendiqueraient les deux entites.
func TestDeuxHumainsDUnMemeIndexOntChacunLeurEntite(t *testing.T) {
	scan := scanDeTest([]int{10, 30, 50, 70, 90},
		grammar.PlayerEntity{Slot: 7, Index: 3, Team: 0, LastKF: 1, Seen: 2},
		grammar.PlayerEntity{Slot: 8, Index: 3, Team: 0, FirstKF: 3, LastKF: 4, Seen: 2})
	roster := []RosterEntry{entree(3, "100"), entree(3, "200")}
	tracks := []Track{vieDe("100", 12, 28), vieDe("200", 72, 99)}
	occ := lierLesOccupants(roster, tracks, entreesDeTest(scan))
	if occ.entitesContestees != 0 || len(occ.parEntree[0].entites) != 1 || len(occ.parEntree[1].entites) != 1 ||
		occ.parEntree[0].entites[0] == occ.parEntree[1].entites[0] {
		t.Fatalf("contestees %d, entites %v / %v : chaque humain a SON entite, celle de son temps",
			occ.entitesContestees, occ.parEntree[0].entites, occ.parEntree[1].entites)
	}
}

// G12 : l'arrivant d'index 10 est de l'equipe 1 ; ses tirs portent l'index 5, une place de
// l'equipe 0. Elle n'est pas a lui : ses tirs ne votent pas, et le chainage le pose dans SON
// equipe.
func TestTirsNeDonnentPasLaPlaceDUneAutreEquipe(t *testing.T) {
	roster := []RosterEntry{entree(0, "100"), entree(1, "110"), entree(5, "500"), entree(10, "310")}
	occ := occupantsFabriques([]int{0, 1, 0, 1}, iv(0, 99), iv(0, 15), iv(0, 10), iv(20, 99))
	tirs := []FireEventRef{{FilmIndex: 5, TimestampUS: 3_000_000}, {FilmIndex: 5, TimestampUS: 5_000_000}}
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0, 1, 5), fire: tirs,
		horloge: horlogeDeSieges(nil)})
	if roster[3].Seat != 1 || roster[3].SeatSource != SeatSourceApparie || cov.PlacesTirs != 0 {
		t.Errorf("arrivant de l'equipe 1 : place %d (%s), places lues par les tirs %d — attendu la "+
			"place 1 de SON equipe, par chainage", roster[3].Seat, roster[3].SeatSource, cov.PlacesTirs)
	}
}

// G17 : deux occupants de la place 0 commencent a la meme frame ; le premier n'y est certain a
// aucune frame (`a < de`, vu seulement avant l'origine). La borne au successeur VIDE son
// affichage : il n'est affiche nulle part, et ne se publie pas.
func TestIntervalleVideParLaBorneNeSePubliePas(t *testing.T) {
	roster := []RosterEntry{entree(0, "100"), entree(0, "200")}
	occ := occupantsFabriques([]int{0, 0}, []intervalleDePresence{{de: 0, a: -1, aMax: 30}}, iv(0, 99))
	poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0), horloge: horlogeDeSieges(nil)})
	if len(roster[0].Presence) != 0 {
		t.Errorf("presence %+v : un affichage vide par la borne ne se publie pas", roster[0].Presence)
	}
	if len(roster[1].Presence) != 1 || roster[1].Presence[0].From != 0 {
		t.Errorf("successeur %+v : il tient la place des f0", roster[1].Presence)
	}
}
