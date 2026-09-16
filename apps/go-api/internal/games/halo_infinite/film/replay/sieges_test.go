package replay

// sieges_test.go — CE QUE LA POSE DES SIEGES GARANTIT (lot 1.9.14).
//
// Chaque test nomme la regle qu'il tient, et une seule :
//
//	T-LU        le film REUTILISE l'index : les deux occupants partagent le siege, la source
//	            reste `lu`, et le repli ordinal ne se declenche PAS ;
//	T-APPARIE   le film ecrit un index NEUF : le siege vient du repli, compte une fois ;
//	T-ORDRE     un arrivant ne reprend pas un siege ENCORE OCCUPE a son arrivee ;
//	T-SANSTABLE un film sans table de depart n'apparie RIEN et le publie ;
//	T-TRONQUE   plus d'arrivants que de sieges liberes : aucune panique, aucun chainage de trop ;
//	T-PRESENCE  les intervalles de presence d'un partant et d'un arrivant ne se recouvrent pas,
//	            et le nombre d'occupants SIMULTANES est celui de l'effectif, pas du roster.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/facts/fallback"
)

// siegeFrames : la longueur des documents de ces tests. Les bornes t1 et t2 s'y rapportent.
const siegeFrames = 100

// tableDeDebut fabrique la table du film pour les index donnes — le roster du DEBUT.
func tableDeDebut(index ...int) FilmPlayerTable {
	t := FilmPlayerTable{Occupied: len(index), Vacant: 32 - len(index)}
	for _, i := range index {
		t.Seats = append(t.Seats, FilmPlayerSeat{FilmIndex: i, XUID: uint64(1000 + i), Gamertag: "j"})
	}
	return t
}

// entree fabrique une entree de roster humaine d'index donne, dans le camp donne.
func entree(idx, camp int, xuid string) RosterEntry {
	c := camp
	return RosterEntry{XUID: xuid, FilmIndex: idx, Name: "j" + xuid, Team: &c}
}

// vie fabrique une piste d'un xuid sur l'intervalle de frames donne.
func vie(xuid string, debut, fin int) Track {
	return Track{Slot: 1, XUID: xuid, StartFrame: debut, EndFrame: fin,
		Points: []Point{{T: debut}, {T: fin}}}
}

// TestSiegeRepriseEcriteEstLue (T-LU, T-PRESENCE) : le partant sort a t1, l'arrivant entre a t2
// SUR LE MEME INDEX. Le document publie le meme siege pour les deux, par LECTURE, et les deux
// presences sont disjointes.
func TestSiegeRepriseEcriteEstLue(t *testing.T) {
	const t1, t2 = 40, 60
	roster := []RosterEntry{entree(0, 1, "100"), entree(0, 1, "200")}
	tracks := []Track{vie("100", 0, t1), vie("200", t2, siegeFrames-1)}
	fb := fallback.NouveauCompteur()
	cov := poserLesSieges(roster, tracks, tableDeDebut(0), siegeFrames, fb)

	if roster[0].Seat != roster[1].Seat {
		t.Fatalf("sieges %d et %d : le film REUTILISE l'index, les deux occupants le partagent",
			roster[0].Seat, roster[1].Seat)
	}
	for i := range roster {
		if roster[i].SeatSource != SeatSourceLu {
			t.Errorf("entree %d : source %q, attendu %q — le film a ECRIT la reprise",
				i, roster[i].SeatSource, SeatSourceLu)
		}
	}
	if n := fb.Compte(fallback.NomSiegeDuRemplacantParAppariementOrdinal); n != 0 {
		t.Errorf("repli declenche %d fois, attendu 0 : il ne passe JAMAIS devant une lecture", n)
	}
	if cov.ReprisesEcrites != 1 || cov.Apparies != 0 || cov.Lus != 2 {
		t.Errorf("couverture %+v : attendu 1 reprise ecrite, 0 appariee, 2 lues", cov)
	}
	if cov.OccupantsMax != 1 {
		t.Errorf("occupants simultanes %d, attendu 1 : les deux presences sont DISJOINTES",
			cov.OccupantsMax)
	}
}

// TestSiegeIndexNeufEstApparie (T-APPARIE) : le film ecrit un index NEUF pour l'arrivant. Le
// chainage sur le siege du partant est un repli, compte une fois.
func TestSiegeIndexNeufEstApparie(t *testing.T) {
	const t1, t2 = 40, 60
	roster := []RosterEntry{entree(0, 1, "100"), entree(1, 1, "200")}
	tracks := []Track{vie("100", 0, t1), vie("200", t2, siegeFrames-1)}
	fb := fallback.NouveauCompteur()
	cov := poserLesSieges(roster, tracks, tableDeDebut(0), siegeFrames, fb)

	if roster[1].Seat != roster[0].Seat {
		t.Fatalf("siege de l'arrivant %d, attendu %d (celui du partant)",
			roster[1].Seat, roster[0].Seat)
	}
	if roster[1].SeatSource != SeatSourceApparie {
		t.Errorf("source %q, attendu %q — le film n'a pas ecrit cette reprise",
			roster[1].SeatSource, SeatSourceApparie)
	}
	if roster[1].FilmIndex != 1 {
		t.Errorf("FilmIndex %d : l'index LU ne doit jamais etre ecrase par le siege",
			roster[1].FilmIndex)
	}
	if n := fb.Compte(fallback.NomSiegeDuRemplacantParAppariementOrdinal); n != 1 {
		t.Errorf("repli declenche %d fois, attendu 1", n)
	}
	if cov.Apparies != 1 || cov.Arrivants != 1 || cov.PresencesCloses != 1 {
		t.Errorf("couverture %+v : attendu 1 appariee, 1 arrivant, 1 presence close", cov)
	}
}

// TestSiegeOccupeNEstPasRepris (T-ORDRE) : un arrivant ne prend pas le siege d'un joueur qui est
// encore la quand il arrive — un remplacement remplace quelqu'un.
func TestSiegeOccupeNEstPasRepris(t *testing.T) {
	roster := []RosterEntry{entree(0, 1, "100"), entree(1, 1, "200")}
	// Le titulaire joue jusqu'a la frame 80 ; l'arrivant entre a la frame 60.
	tracks := []Track{vie("100", 0, 80), vie("200", 60, siegeFrames-1)}
	fb := fallback.NouveauCompteur()
	cov := poserLesSieges(roster, tracks, tableDeDebut(0), siegeFrames, fb)

	if roster[1].Seat != 1 || roster[1].SeatSource != SeatSourceLu {
		t.Fatalf("siege %d (%s) : le siege 0 est ENCORE OCCUPE a la frame 60, il ne se reprend pas",
			roster[1].Seat, roster[1].SeatSource)
	}
	if n := fb.Compte(fallback.NomSiegeDuRemplacantParAppariementOrdinal); n != 0 {
		t.Errorf("repli declenche %d fois, attendu 0", n)
	}
	if cov.OccupantsMax != 2 {
		t.Errorf("occupants simultanes %d, attendu 2 : les presences SE RECOUVRENT", cov.OccupantsMax)
	}
}

// TestSiegeSansTableDuFilmNApparieRien (T-SANSTABLE) : sans la table du debut, arrivants et
// origines ne se distinguent pas. On s'ABSTIENT, et on le publie.
func TestSiegeSansTableDuFilmNApparieRien(t *testing.T) {
	roster := []RosterEntry{entree(0, 1, "100"), entree(1, 1, "200")}
	tracks := []Track{vie("100", 0, 40), vie("200", 60, siegeFrames-1)}
	fb := fallback.NouveauCompteur()
	cov := poserLesSieges(roster, tracks, FilmPlayerTable{}, siegeFrames, fb)

	if !cov.SansTableDuFilm {
		t.Fatal("la couverture doit DIRE que la table manque : une abstention tue est un silence")
	}
	if cov.Apparies != 0 || roster[1].Seat != 1 {
		t.Errorf("siege %d, apparies %d : sans table, aucun chainage n'est decidable",
			roster[1].Seat, cov.Apparies)
	}
	if n := fb.Compte(fallback.NomSiegeDuRemplacantParAppariementOrdinal); n != 0 {
		t.Errorf("repli declenche %d fois : s'abstenir n'est PAS se replier", n)
	}
}

// TestSiegePlusDArrivantsQueDeLiberations (T-TRONQUE) : la borne de la boucle d'appariement tient
// quand les arrivants sont plus nombreux que les sieges liberes.
func TestSiegePlusDArrivantsQueDeLiberations(t *testing.T) {
	roster := []RosterEntry{
		entree(0, 1, "100"), entree(1, 1, "200"), entree(2, 1, "300"), entree(3, 1, "400"),
	}
	tracks := []Track{
		vie("100", 0, 30),             // le SEUL siege libere
		vie("200", 0, siegeFrames-1),  // titulaire qui reste
		vie("300", 40, siegeFrames-1), // arrivant 1
		vie("400", 50, siegeFrames-1), // arrivant 2 : plus rien a reprendre
	}
	fb := fallback.NouveauCompteur()
	cov := poserLesSieges(roster, tracks, tableDeDebut(0, 1), siegeFrames, fb)

	if roster[2].Seat != 0 || roster[2].SeatSource != SeatSourceApparie {
		t.Errorf("arrivant 1 : siege %d (%s), attendu 0 (apparie)",
			roster[2].Seat, roster[2].SeatSource)
	}
	if roster[3].Seat != 3 || roster[3].SeatSource != SeatSourceLu {
		t.Errorf("arrivant 2 : siege %d (%s), attendu 3 (lu) — il n'y a plus de siege a reprendre",
			roster[3].Seat, roster[3].SeatSource)
	}
	if cov.Apparies != 1 {
		t.Errorf("apparies %d, attendu 1", cov.Apparies)
	}
}

// TestSiegeSansPresenceNEstPasAppariee (T-PRESENCE) : une entree qu'aucune vie ne couvre n'a de
// fiche a AUCUN instant. Elle ne libere ni ne prend de siege, et le document la COMPTE.
func TestSiegeSansPresenceNEstPasAppariee(t *testing.T) {
	roster := []RosterEntry{entree(0, 1, "100"), entree(1, 1, "200")}
	tracks := []Track{vie("100", 0, siegeFrames-1)} // "200" n'a aucune vie
	fb := fallback.NouveauCompteur()
	cov := poserLesSieges(roster, tracks, tableDeDebut(0), siegeFrames, fb)

	if cov.SansPresence != 1 {
		t.Errorf("sansPresence %d, attendu 1", cov.SansPresence)
	}
	if roster[1].SeatSource != SeatSourceLu || cov.Apparies != 0 {
		t.Errorf("une entree sans presence ne s'apparie pas : %q, apparies %d",
			roster[1].SeatSource, cov.Apparies)
	}
	if cov.OccupantsMax != 1 {
		t.Errorf("occupants simultanes %d, attendu 1", cov.OccupantsMax)
	}
}

// TestSiegeDUnBotSeJointParLeNom : un bot n'a pas de xuid (schema 36) ; sa presence se joint par
// le nom, des deux cotes, sans quoi tous les bots seraient « sans presence ».
func TestSiegeDUnBotSeJointParLeNom(t *testing.T) {
	camp := 0
	roster := []RosterEntry{{FilmIndex: 4, Name: "343 Razzle [bot]", Bot: true, Team: &camp}}
	tracks := []Track{{Slot: 9, Bot: "343 Razzle [bot]", StartFrame: 10, EndFrame: 90,
		Points: []Point{{T: 10}, {T: 90}}}}
	cov := poserLesSieges(roster, tracks, tableDeDebut(4), siegeFrames, nil)

	if cov.SansPresence != 0 {
		t.Fatalf("sansPresence %d : la vie d'un bot se joint par son NOM", cov.SansPresence)
	}
	if cov.OccupantsMax != 1 {
		t.Errorf("occupants simultanes %d, attendu 1", cov.OccupantsMax)
	}
}
