package replay

// sieges_successions_test.go — LE BOT BOUCHE-TROU PREND LA PLACE DU PARTANT, ET L'HUMAIN QUI ARRIVE
// LUI SUCCEDE (lot « toute entree du roster a l'equipe que le film ecrit », 2026-10-06 ; regle des
// places, Q23).
//
//	S-CHAINE      gabarit de `43716616` : le partant de la place 5 sort, le bot (equipe lue dans sa
//	              declaration) prend SA place par chainage, l'humain qui arrive ensuite la reprend ;
//	              aucune place 8, aucun depassement ;
//	S-RELAIS      gabarit de `572e236b` : la derniere vie du partant finit a la frame ou le bot est
//	              declare — un relais a la frame, pas un chevauchement ;
//	S-RELAIS-LU   la meme frame partagee par deux presences lues aux images-cles (aucun bot date) :
//	              deux occupants a la meme image-cle, la contradiction reste ;
//	S-SUCCESSION  gabarit de `43e96765` : l'humain est lu a l'image-cle pendant la declaration du bot
//	              et n'a aucune vie avant son retrait : il lui succede, sa presence commence au
//	              lendemain du retrait ;
//	S-COTOIE      le meme humain avec une vie AVANT le retrait du bot : ils se cotoient, la place
//	              n'est pas reprise.

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// poserDesBots marque comme BOTS DECLARES les entrees donnees (presence datee par BOT_METADATA).
func poserDesBots(roster []RosterEntry, occ *occupants, entrees ...int) {
	for _, i := range entrees {
		roster[i].Bot = true
		occ.parEntree[i].declaree = true
	}
}

func TestBotSurLaPlaceDuPartantPuisLHumain(t *testing.T) { // S-CHAINE
	roster := []RosterEntry{entree(5, "150"), entree(2, "120"), {FilmIndex: 8, Name: "343 Sandwolf [bot]"},
		entree(9, "190")}
	occ := occupantsFabriques([]int{0, 0, 0, 0}, []intervalleDePresence{{de: 0, a: 11, aMax: 31}},
		iv(0, siegeFrames-1), iv(24, 28), iv(32, siegeFrames-1))
	poserDesBots(roster, &occ, 2)
	fb := fallback.NouveauCompteur()
	cov := poserLesSieges(context.Background(), roster, occ, entreesDesPlaces{table: tableDeDebut(5, 2),
		horloge: horlogeDeSieges(fb)})

	if roster[2].Seat != 5 || roster[2].SeatSource != SeatSourceApparie {
		t.Fatalf("bot : place %d (%s), attendu 5 (apparie) — la place du partant", roster[2].Seat, roster[2].SeatSource)
	}
	if roster[3].Seat != 5 {
		t.Fatalf("humain arrive : place %d (%s), attendu 5 — il reprend la place apres le bot", roster[3].Seat,
			roster[3].SeatSource)
	}
	if p := roster[0].Presence; len(p) != 1 || p[0].ToMax == nil || *p[0].ToMax != 23 {
		t.Errorf("partant : presence %+v, attendu un affichage borne a 23 (veille du bot)", p)
	}
	if cov.SansPlace != 0 || cov.Depassements != 0 || cov.PlacesEnTrop != 0 || cov.Sieges != 2 ||
		cov.Chevauchements != 0 {
		t.Errorf("couverture %+v : deux places, aucune entree sans place ni depassement", cov)
	}
}

func TestRelaisALaFrameDuBot(t *testing.T) { // S-RELAIS
	roster := []RosterEntry{entree(7, "170"), entree(2, "120"), {FilmIndex: 8, Name: "343 Bachici [bot]"},
		entree(9, "190")}
	occ := occupantsFabriques([]int{0, 0, 0, 0}, []intervalleDePresence{{de: 0, a: 58, aMax: 69}},
		iv(0, siegeFrames-1), iv(58, 60), iv(70, siegeFrames-1))
	poserDesBots(roster, &occ, 2)
	cov := poserLesSieges(context.Background(), roster, occ, entreesDesPlaces{table: tableDeDebut(7, 2),
		horloge: horlogeDeSieges(fallback.NouveauCompteur())})

	if roster[2].Seat != 7 || roster[2].SeatSource != SeatSourceApparie {
		t.Fatalf("bot : place %d (%s), attendu 7 (apparie) — le relais a la frame 58 libere la place",
			roster[2].Seat, roster[2].SeatSource)
	}
	if p := roster[0].Presence; len(p) != 1 || p[0].To != 57 {
		t.Errorf("partant : presence %+v, attendu une fin certaine bornee a 57", p)
	}
	if cov.Chevauchements != 0 || cov.SansPlace != 0 || cov.Depassements != 0 {
		t.Errorf("couverture %+v : un relais a la frame n'est pas un chevauchement", cov)
	}
}

func TestFramePartageeSansBotDateResteUneContradiction(t *testing.T) { // S-RELAIS-LU
	roster := []RosterEntry{entree(7, "170"), entree(2, "120"), entree(9, "190")}
	occ := occupantsFabriques([]int{0, 0, 0}, iv(0, 58), iv(0, siegeFrames-1), iv(58, siegeFrames-1))
	cov := poserLesSieges(context.Background(), roster, occ, entreesDesPlaces{table: tableDeDebut(7, 2),
		horloge: horlogeDeSieges(fallback.NouveauCompteur())})

	if roster[2].SeatSource != SeatSourceIndex || cov.SansPlace != 1 {
		t.Fatalf("arrivant : place %d (%s), sans place %d — deux occupants lus a la meme frame ne se "+
			"relaient pas", roster[2].Seat, roster[2].SeatSource, cov.SansPlace)
	}
}

func TestLHumainSuccedeAuBot(t *testing.T) { // S-SUCCESSION
	roster := []RosterEntry{entree(0, "100"), entree(2, "120"), {FilmIndex: 8, Name: "343 PardonMy [bot]"},
		entree(9, "190")}
	occ := occupantsFabriques([]int{0, 0, 0, 0}, []intervalleDePresence{{de: 0, a: 40, aMax: 44}},
		iv(0, siegeFrames-1), iv(45, 70), iv(50, siegeFrames-1))
	poserDesBots(roster, &occ, 2)
	occ.parEntree[3].vies = [][2]int{{75, siegeFrames - 1}}
	cov := poserLesSieges(context.Background(), roster, occ, entreesDesPlaces{table: tableDeDebut(0, 2),
		horloge: horlogeDeSieges(fallback.NouveauCompteur())})

	if roster[2].Seat != 0 || roster[3].Seat != 0 {
		t.Fatalf("places bot %d (%s), humain %d (%s) : attendu 0 et 0 — l'humain succede au bot", roster[2].Seat,
			roster[2].SeatSource, roster[3].Seat, roster[3].SeatSource)
	}
	if p := roster[3].Presence; len(p) != 1 || p[0].From != 71 {
		t.Errorf("humain : presence %+v, attendu un debut au lendemain du retrait du bot (71)", p)
	}
	if cov.Depassements != 0 || cov.SansPlace != 0 || cov.Chevauchements != 0 {
		t.Errorf("couverture %+v : une succession, ni depassement ni chevauchement", cov)
	}
}

func TestLHumainQuiJoueAvantLeRetraitCotoieLeBot(t *testing.T) { // S-COTOIE
	roster := []RosterEntry{entree(0, "100"), entree(2, "120"), {FilmIndex: 8, Name: "343 PardonMy [bot]"},
		entree(9, "190")}
	occ := occupantsFabriques([]int{0, 0, 0, 0}, []intervalleDePresence{{de: 0, a: 40, aMax: 44}},
		iv(0, siegeFrames-1), iv(45, 70), iv(50, siegeFrames-1))
	poserDesBots(roster, &occ, 2)
	occ.parEntree[3].vies = [][2]int{{60, siegeFrames - 1}}
	cov := poserLesSieges(context.Background(), roster, occ, entreesDesPlaces{table: tableDeDebut(0, 2),
		horloge: horlogeDeSieges(fallback.NouveauCompteur())})

	if p := roster[3].Presence; roster[3].Seat == 0 || len(p) != 1 || p[0].From != 50 {
		t.Fatalf("humain : place %d (%s), presence %+v — une vie avant le retrait du bot : ils se "+
			"cotoient, aucune succession (place non reprise, presence intacte)", roster[3].Seat,
			roster[3].SeatSource, roster[3].Presence)
	}
	if cov.SansPlace != 1 {
		t.Errorf("couverture %+v : l humain qui cotoie le bot reste sans place, et le compte le dit", cov)
	}
}
