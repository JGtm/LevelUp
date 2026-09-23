package replay

// sieges_test.go — CE QUE LA POSE DES PLACES GARANTIT (lot 1.9.14 ; refondu au lot M2.3,
// 2026-09-23, sur la regle des places de l'utilisateur).
//
// LES TESTS D'AVANT LE LOT M2.3 ONT ETE REECRITS AVEC LA REGLE QU'ILS VERROUILLAIENT : ils
// fixaient l'appariement ORDINAL des seuls partants de la table ayant une equipe et des vies
// (`repli_siege_du_remplacant_par_appariement_ordinal`, retire). Chaque test nomme la regle qu'il
// tient, et une seule :
//
//	P-LU        le film REUTILISE l'index : les deux occupants tiennent la meme place, par lecture ;
//	P-CHAINE    un arrivant d'index NEUF prend la place que son equipe libere, par le repli compte ;
//	P-OCCUPEE   une place encore tenue a son arrivee ne se reprend pas ;
//	P-CAPACITE  plus d'arrivants que de places : le surplus reste sans place, compte, et c'est le
//	            seul chemin par lequel une equipe depasse ses places (`depassements`) ;
//	P-SANSTABLE sans table du debut, aucune place n'est decidable : abstention publiee ;
//	P-TIRS      un arrivant qui tire lit sa place a l'unanimite de ses tirs ; deux places votees
//	            = contestee, et le chainage reprend ;
//	P-BORNE     l'affichage d'un occupant s'arrete la veille de l'arrivee du suivant ;
//	P-VIES      sans entite, le dernier occupant d'une place reste jusqu'a la fin (repli compte).

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// siegeFrames : la longueur des documents de ces tests.
const siegeFrames = 100

// tableDeDebut fabrique la table du film pour les index donnes — le roster du DEBUT.
func tableDeDebut(index ...int) FilmPlayerTable {
	t := FilmPlayerTable{Occupied: len(index), Vacant: 32 - len(index)}
	for _, i := range index {
		t.Seats = append(t.Seats, FilmPlayerSeat{FilmIndex: i, XUID: uint64(1000 + i), Gamertag: "j"})
	}
	return t
}

// entree fabrique une entree de roster humaine d'index donne.
func entree(idx int, xuid string) RosterEntry {
	return RosterEntry{XUID: xuid, FilmIndex: idx, Name: "j" + xuid}
}

// occupantsFabriques porte les occupants d'un test : une presence LUE (comme si le film l'avait
// donnee) et une equipe par entree, dans l'ordre du roster.
func occupantsFabriques(camps []int, presences ...[]intervalleDePresence) occupants {
	occ := occupants{parEntree: make([]occupantDuRoster, len(presences)), balaye: true}
	for i, p := range presences {
		c := camps[i]
		occ.parEntree[i] = occupantDuRoster{presence: p, equipe: &c, lue: true}
	}
	return occ
}

// iv fabrique un intervalle de presence dont l'affichage egale le certain.
func iv(de, a int) []intervalleDePresence {
	return []intervalleDePresence{{de: de, a: a, aMax: a}}
}

func horlogeDeSieges(fb *fallback.Compteur) replayClock {
	return replayClock{origin: 0, step: 100_000, frames: siegeFrames, fb: fb}
}

// TestPlaceRepriseEcriteEstLue (P-LU) : le partant sort a 40, l'arrivant entre a 60 SUR LE MEME
// INDEX. Les deux tiennent la meme place, par LECTURE, et le repli ne se declenche pas.
func TestPlaceRepriseEcriteEstLue(t *testing.T) {
	roster := []RosterEntry{entree(0, "100"), entree(0, "200")}
	occ := occupantsFabriques([]int{1, 1}, iv(0, 40), iv(60, siegeFrames-1))
	fb := fallback.NouveauCompteur()
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0), horloge: horlogeDeSieges(fb)})

	if roster[0].Seat != 0 || roster[1].Seat != 0 {
		t.Fatalf("places %d et %d : le film REUTILISE l'index, les deux occupants la partagent",
			roster[0].Seat, roster[1].Seat)
	}
	for i := range roster {
		if roster[i].SeatSource != SeatSourceLu {
			t.Errorf("entree %d : source %q, attendu %q", i, roster[i].SeatSource, SeatSourceLu)
		}
	}
	if n := fb.Compte(fallback.NomPlaceDuRemplacantParChainageDEquipe); n != 0 {
		t.Errorf("repli declenche %d fois, attendu 0 : il ne passe JAMAIS devant une lecture", n)
	}
	if cov.ReprisesEcrites != 1 || cov.Apparies != 0 || cov.Lus != 2 || cov.OccupantsMax != 1 {
		t.Errorf("couverture %+v : attendu 1 reprise ecrite, 0 appariee, 2 lues, 1 occupant", cov)
	}
}

// TestPlaceIndexNeufEstChainee (P-CHAINE) : l'arrivant porte un index NEUF ; il prend la place que
// son equipe a liberee, par le repli, compte une fois. Son index LU n'est jamais ecrase.
func TestPlaceIndexNeufEstChainee(t *testing.T) {
	roster := []RosterEntry{entree(0, "100"), entree(1, "200"), entree(9, "300")}
	occ := occupantsFabriques([]int{1, 0, 1}, iv(0, 40), iv(0, siegeFrames-1), iv(60, siegeFrames-1))
	fb := fallback.NouveauCompteur()
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0, 1), horloge: horlogeDeSieges(fb)})

	if roster[2].Seat != 0 || roster[2].SeatSource != SeatSourceApparie {
		t.Fatalf("arrivant : place %d (%s), attendu 0 (apparie) — la place LIBEREE de son equipe",
			roster[2].Seat, roster[2].SeatSource)
	}
	if roster[2].FilmIndex != 9 {
		t.Errorf("FilmIndex %d : l'index LU ne doit jamais etre ecrase par la place", roster[2].FilmIndex)
	}
	if n := fb.Compte(fallback.NomPlaceDuRemplacantParChainageDEquipe); n != 1 {
		t.Errorf("repli declenche %d fois, attendu 1", n)
	}
	if cov.Apparies != 1 || cov.Arrivants != 1 || cov.PresencesCloses != 1 || cov.Depassements != 0 {
		t.Errorf("couverture %+v : attendu 1 appariee, 1 arrivant, 1 presence close, 0 depassement", cov)
	}
}

// TestPlaceOccupeeNEstPasReprise (P-OCCUPEE) : le titulaire joue jusqu'a 80, l'arrivant entre a
// 60 — la place n'est pas libre ; faute d'autre place de son equipe, il n'en a pas.
func TestPlaceOccupeeNEstPasReprise(t *testing.T) {
	roster := []RosterEntry{entree(0, "100"), entree(9, "200")}
	occ := occupantsFabriques([]int{1, 1}, iv(0, 80), iv(60, siegeFrames-1))
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0),
		horloge: horlogeDeSieges(nil)})

	if roster[1].Seat != 9 || roster[1].SeatSource != SeatSourceIndex {
		t.Fatalf("place %d (%s) : la place 0 est ENCORE TENUE a la frame 60, elle ne se reprend pas",
			roster[1].Seat, roster[1].SeatSource)
	}
	if cov.SansPlace != 1 || cov.Depassements == 0 {
		t.Errorf("couverture %+v : l'arrivant sans place se COMPTE, et son equipe depasse ses places", cov)
	}
}

// TestPlacesLaCapaciteBorneLesPlacesJamaisTenues (P-CAPACITE) : en 2 contre 2 sur une table de 4
// sieges, le siege jamais tenu (celui d'un joueur parti avant le coup d'envoi) ne va qu'a
// l'equipe a qui il manque une place — jamais a une equipe deja pleine.
func TestPlacesLaCapaciteBorneLesPlacesJamaisTenues(t *testing.T) {
	roster := []RosterEntry{
		entree(0, "100"), entree(1, "110"), // equipe 0 : deux places tenues
		entree(2, "200"), // equipe 1 : une seule — le siege 3 n'a jamais joue
		{FilmIndex: 3, XUID: "300", Name: "parti avant le coup d'envoi"},
		entree(9, "900"),  // bot de l'equipe 1, present au depart : il prend le siege 3
		entree(10, "910"), // arrivant de l'equipe 0 : son equipe est pleine
	}
	occ := occupantsFabriques([]int{0, 0, 1, 1, 1, 0}, iv(0, 99), iv(0, 99), iv(0, 99), nil,
		iv(0, 99), iv(50, 99))
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0, 1, 2, 3),
		horloge: horlogeDeSieges(nil)})

	if roster[4].Seat != 3 || roster[4].SeatSource != SeatSourceApparie {
		t.Errorf("bot de l'equipe 1 : place %d (%s), attendu 3 (apparie)", roster[4].Seat, roster[4].SeatSource)
	}
	if roster[5].SeatSource != SeatSourceIndex {
		t.Errorf("arrivant de l'equipe 0 : %q — son equipe tient deja ses deux places", roster[5].SeatSource)
	}
	if cov.SansPresence != 1 || cov.SansPlace != 1 {
		t.Errorf("couverture %+v : 1 entree sans presence, 1 sans place", cov)
	}
}

// TestPlaceSansTableDuFilmNeDecideRien (P-SANSTABLE) : sans la table du debut, arrivants et
// origines ne se distinguent pas. On s'ABSTIENT, et on le publie.
func TestPlaceSansTableDuFilmNeDecideRien(t *testing.T) {
	roster := []RosterEntry{entree(0, "100"), entree(1, "200")}
	occ := occupantsFabriques([]int{1, 1}, iv(0, 40), iv(60, siegeFrames-1))
	fb := fallback.NouveauCompteur()
	cov := poserLesSieges(roster, occ, entreesDesPlaces{horloge: horlogeDeSieges(fb)})

	if !cov.SansTableDuFilm {
		t.Fatal("la couverture doit DIRE que la table manque : une abstention tue est un silence")
	}
	if cov.Apparies != 0 || roster[1].Seat != 1 || roster[1].SeatSource != SeatSourceLu {
		t.Errorf("place %d (%s), apparies %d : sans table, aucun chainage n'est decidable",
			roster[1].Seat, roster[1].SeatSource, cov.Apparies)
	}
	if n := fb.Compte(fallback.NomPlaceDuRemplacantParChainageDEquipe); n != 0 {
		t.Errorf("repli declenche %d fois : s'abstenir n'est PAS se replier", n)
	}
}

// TestPlaceLueDansLesTirs (P-TIRS) : l'arrivant d'index 10 tire sous l'index 5 — la place de la
// table que personne ne tient — et il la LIT, avant tout chainage.
func TestPlaceLueDansLesTirs(t *testing.T) {
	roster := []RosterEntry{entree(0, "100"), {FilmIndex: 5, XUID: "500"}, entree(10, "310")}
	occ := occupantsFabriques([]int{0, 0, 0}, iv(0, 99), nil, iv(20, 99))
	tirs := []FireEventRef{
		{FilmIndex: 5, TimestampUS: 3_000_000}, {FilmIndex: 5, TimestampUS: 5_000_000},
		{FilmIndex: 0, TimestampUS: 4_000_000}, // le tir du titulaire de la place 0 ne vote pas
	}
	fb := fallback.NouveauCompteur()
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0, 5), fire: tirs,
		horloge: horlogeDeSieges(fb)})

	if roster[2].Seat != 5 || roster[2].SeatSource != SeatSourceTirs {
		t.Fatalf("arrivant : place %d (%s), attendu 5 (tirs)", roster[2].Seat, roster[2].SeatSource)
	}
	if cov.PlacesTirs != 1 || cov.Apparies != 0 || fb.Compte(fallback.NomPlaceDuRemplacantParChainageDEquipe) != 0 {
		t.Errorf("couverture %+v : une place LUE ne passe pas par le repli", cov)
	}
}

// TestPlaceDesTirsContesteeRetombeSurLeChainage (P-TIRS) : les tirs de l'arrivant designent DEUX
// places — la lecture se tait, le compte le dit, et le chainage reprend.
func TestPlaceDesTirsContesteeRetombeSurLeChainage(t *testing.T) {
	roster := []RosterEntry{entree(0, "100"), {FilmIndex: 4, XUID: "400"}, {FilmIndex: 5, XUID: "500"},
		entree(10, "310")}
	occ := occupantsFabriques([]int{0, 0, 0, 0}, iv(0, 99), nil, nil, iv(20, 99))
	tirs := []FireEventRef{{FilmIndex: 4, TimestampUS: 3_000_000}, {FilmIndex: 5, TimestampUS: 5_000_000}}
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0, 4, 5), fire: tirs,
		horloge: horlogeDeSieges(nil)})

	if cov.TirsContestes != 1 || roster[3].SeatSource != SeatSourceApparie {
		t.Errorf("tirs contestes %d, source %q : attendu 1 contestation puis le chainage",
			cov.TirsContestes, roster[3].SeatSource)
	}
}

// TestPlaceBorneeAuSuccesseur (P-BORNE) : le partant peut encore etre la jusqu'a 70 (image-cle
// suivante), son remplacant arrive a 55 — l'affichage du partant s'arrete a 54.
func TestPlaceBorneeAuSuccesseur(t *testing.T) {
	roster := []RosterEntry{entree(0, "100"), entree(9, "900")}
	occ := occupantsFabriques([]int{1, 1}, []intervalleDePresence{{de: 0, a: 50, aMax: 70}},
		iv(55, siegeFrames-1))
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0), horloge: horlogeDeSieges(nil)})

	if roster[1].Seat != 0 {
		t.Fatalf("remplacant : place %d, attendu 0", roster[1].Seat)
	}
	p := roster[0].Presence
	if len(p) != 1 || p[0].To != 50 || p[0].ToMax == nil || *p[0].ToMax != 54 {
		t.Fatalf("presence du partant %+v : attendu to 50, toMax 54 (veille de l'arrivee)", p)
	}
	if cov.RelaisBornes != 1 || cov.OccupantsMax != 1 || cov.Depassements != 0 {
		t.Errorf("couverture %+v : 1 relais borne, jamais deux occupants sur une place", cov)
	}
}

// TestPresenceSansEntiteTientLeDernierJusquALaFin (P-VIES) : sans entite lue, le dernier occupant
// d'une place reste affiche jusqu'a la fin — mourir n'est pas partir —, et le repli se compte.
func TestPresenceSansEntiteTientLeDernierJusquALaFin(t *testing.T) {
	roster := []RosterEntry{entree(0, "100")}
	occ := occupantsFabriques([]int{1}, iv(0, 40))
	occ.balaye = false
	fb := fallback.NouveauCompteur()
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0), horloge: horlogeDeSieges(fb)})

	p := roster[0].Presence
	if len(p) != 1 || p[0].To != 40 || p[0].ToMax == nil || *p[0].ToMax != siegeFrames-1 {
		t.Fatalf("presence %+v : attendu to 40, affiche jusqu'a %d", p, siegeFrames-1)
	}
	if n := fb.Compte(fallback.NomPresenceParEnveloppeDesVies); n != 1 || cov.Presences != PresencesDesVies {
		t.Errorf("repli %d, presences %q : attendu 1 et %q", n, cov.Presences, PresencesDesVies)
	}
}
