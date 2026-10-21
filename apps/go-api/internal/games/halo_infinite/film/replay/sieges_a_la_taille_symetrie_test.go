package replay

// sieges_a_la_taille_symetrie_test.go — LA REGLE DES PLACES EST SYMETRIQUE (decision utilisateur du
// 2026-10-21 : « impossible d'avoir 5 joueurs en meme temps, meme s'il y a un bot » ; un bot qui
// remplace un joueur parti prend SA fiche, comme un humain qui remplace un bot prend celle du bot).
// Equipe de deux, cf. l'en-tete de sieges_a_la_taille.go :
//
//	T-RETOUR      un joueur qui revient alors que sa fiche a ete reprise prend une fiche LIBEREE de son
//	              equipe, pour cet intervalle seulement (`presence[].seat`, `400bb628`) ;
//	T-ABSENCE     un bot qui arrive pendant l'absence d'un humain (aucune vie) tient SA fiche le temps
//	              de l'absence ; l'humain la retrouve apres (`f235e2f1`) ;
//	T-RELEVE      un humain qui arrive dans une equipe pleine releve le bot, qui perd sa fiche la veille ;
//	              une vie du bot apres l'arrivee est comptee (`botsRelevesEnJeu`).
//
// Les trois verifient l'invariant : a chaque frame, au plus deux occupants, sur deux fiches distinctes.

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// poserALaTailleDeDeux pose les places d'une equipe de deux.
func poserALaTailleDeDeux(roster []RosterEntry, occ occupants, table ...int) SeatCoverage {
	return poserLesSieges(context.Background(), roster, occ, entreesDesPlaces{table: tableDeDebut(table...),
		horloge: horlogeDeSieges(fallback.NouveauCompteur()), tailleDEquipe: 2})
}

// verifierLInvariant echoue quand une frame montre plus de `taille` occupants, ou deux sur une fiche.
func verifierLInvariant(t *testing.T, roster []RosterEntry, taille int) {
	t.Helper()
	for f := range siegeFrames {
		fiches := map[int]string{}
		for _, e := range roster {
			for _, p := range e.Presence {
				fin := p.To
				if p.ToMax != nil {
					fin = *p.ToMax
				}
				if f < p.From || f > fin {
					continue
				}
				place := e.Seat
				if p.Seat != nil {
					place = *p.Seat
				}
				if autre, pris := fiches[place]; pris {
					t.Fatalf("frame %d : fiche %d tenue par %s et %s", f, place, autre, e.Name)
				}
				fiches[place] = e.Name
			}
		}
		if len(fiches) > taille {
			t.Fatalf("frame %d : %d occupants %v pour %d places", f, len(fiches), fiches, taille)
		}
	}
}

// TestUnRetourPrendUneFicheLiberee (T-RETOUR).
func TestUnRetourPrendUneFicheLiberee(t *testing.T) {
	roster := []RosterEntry{
		entree(0, "100"),  // A, part a 50
		entree(1, "110"),  // R, part a 30, revient a 70
		entree(40, "140"), // H, arrive a 40 : la fiche de R est libre
	}
	retour := []intervalleDePresence{{de: 0, a: 30, aMax: 30}, {de: 70, a: 99, aMax: 99}}
	occ := occupantsFabriques([]int{0, 0, 0}, iv(0, 50), retour, iv(40, 99))
	cov := poserALaTailleDeDeux(roster, occ, 0, 1)

	if roster[2].Seat != 1 {
		t.Errorf("H : fiche %d, attendu celle de R (1)", roster[2].Seat)
	}
	if p := roster[1].Presence; len(p) != 2 || p[0].Seat != nil || p[1].Seat == nil || *p[1].Seat != 0 {
		t.Errorf("R : presence %+v, attendu son retour sur la fiche liberee par A (0)", p)
	}
	if roster[1].Seat != 1 {
		t.Errorf("R : fiche d'entree %d, attendu celle de son premier intervalle (1)", roster[1].Seat)
	}
	if cov.PlacesEnTrop != 0 {
		t.Errorf("couverture %+v : aucune place en trop", cov)
	}
	verifierLInvariant(t, roster, 2)
}

// TestUnBotTientLAbsenceDUnHumain (T-ABSENCE).
func TestUnBotTientLAbsenceDUnHumain(t *testing.T) {
	roster := []RosterEntry{
		entree(0, "100"), // A, la du debut a la fin
		entree(1, "110"), // E, sans corps de 21 a 59
		{Bot: true, Name: "343 B", FilmIndex: 30},
	}
	occ := occupantsFabriques([]int{0, 0, 0}, iv(0, 99), iv(0, 99), iv(25, 50))
	occ.parEntree[0].vies = [][2]int{{0, 99}}
	occ.parEntree[1].vies = [][2]int{{0, 20}, {60, 99}}
	occ.parEntree[2].vies = [][2]int{{25, 50}}
	cov := poserALaTailleDeDeux(roster, occ, 0, 1)

	if roster[2].Seat != 1 || roster[2].SeatSource != SeatSourceTaille {
		t.Errorf("bot : fiche %d (%s), attendu celle de E (1, %s)", roster[2].Seat, roster[2].SeatSource, SeatSourceTaille)
	}
	if p := roster[1].Presence; len(p) != 2 || p[0].To != 24 || p[1].From != 51 || p[1].To != 99 {
		t.Errorf("E : presence %+v, attendu [0,24] puis [51,99] autour du bot", p)
	}
	if cov.AbsencesTenuesParUnBot != 1 || cov.PlacesEnTrop != 0 {
		t.Errorf("couverture %+v : une absence tenue, aucune place en trop", cov)
	}
	verifierLInvariant(t, roster, 2)
}

// TestUnHumainReleveLeBot (T-RELEVE).
func TestUnHumainReleveLeBot(t *testing.T) {
	roster := []RosterEntry{
		entree(0, "100"), // A, la du debut a la fin
		{Bot: true, Name: "343 B", FilmIndex: 30},
		entree(40, "140"), // H, arrive a 30 dans une equipe pleine
	}
	occ := occupantsFabriques([]int{0, 0, 0}, iv(0, 99), iv(0, 99), iv(30, 99))
	occ.parEntree[1].vies = [][2]int{{0, 20}, {22, 45}} // une vie du bot passe l'arrivee de H
	occ.parEntree[2].vies = [][2]int{{31, 99}}          // H joue des son arrivee
	cov := poserALaTailleDeDeux(roster, occ, 0)

	if roster[2].Seat != roster[1].Seat {
		t.Errorf("H : fiche %d, attendu celle du bot (%d)", roster[2].Seat, roster[1].Seat)
	}
	if p := roster[1].Presence; len(p) != 1 || p[0].To != 29 {
		t.Errorf("bot releve : presence %+v, attendu jusqu'a la veille de l'arrivee de H (29)", p)
	}
	if cov.BotsRelevesALaTaille != 1 || cov.BotsRelevesEnJeu != 1 || cov.PlacesEnTrop != 0 {
		t.Errorf("couverture %+v : un bot releve, encore en jeu, aucune place en trop", cov)
	}
	verifierLInvariant(t, roster, 2)
}
