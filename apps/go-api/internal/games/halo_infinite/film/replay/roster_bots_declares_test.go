package replay

// roster_bots_declares_test.go — LE BOT SANS ENTITE DONT LES DECLARATIONS ONT NOMME UN CORPS ENTRE AU
// ROSTER (roster_bots_successeurs.go, [admettreLesBotsNommesParDeclaration]).
//
//	A-NOMME        gabarit de `bf2a9f05` (Mickey) : l'humain de l'index 3 part (absence prouvee a f50),
//	               le bot sans entite est declare de f58 a la fin et une piste porte son nom : il entre,
//	               sur la place du partant ;
//	A-SANS-PISTE   aucune piste ne porte son nom : il reste dehors ;
//	A-SIMULTANE    sa declaration touche la fenetre large de l'entite de l'humain : il reste dehors ;
//	A-ENTITE       le bot a une entite : cette lecture-ci ne decide pas (la lecture par entites le fait).

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// bfScrubDeTest : `successionDeTest` sans l'entite du bot (slot 5) — un bot dont toute la presence
// tombe entre deux images-cles n'en a pas.
func bfScrubDeTest() (rosterInitial []RosterEntry, bots []BotIdentity, scan grammar.PlayerEntityScan,
	tracks []Track) {
	idx, bots, scan, tracks := successionDeTest(false)
	scan.Entities = scan.Entities[:4]
	roster, _ := rosterDesOccupants(idx, nil, bots, scan, teamPublication{})
	return roster, bots, scan, tracks
}

func TestBotSansEntiteNommeParSaDeclarationEntreAuRoster(t *testing.T) { // A-NOMME
	roster, bots, scan, tracks := bfScrubDeTest()
	if len(roster) != 4 {
		t.Fatalf("roster de depart %d entrees, attendu 4 (le bot sans entite n'y est pas)", len(roster))
	}
	roster, admis := admettreLesBotsNommesParDeclaration(roster, bots, scan, teamPublication{}, tracks)
	if admis != 1 || len(roster) != 5 {
		t.Fatalf("admis %d, %d entrees : le bot dont une piste porte le nom entre au roster", admis, len(roster))
	}
	in := entreesDeTest(scan)
	in.bots = bots
	occ := lierLesOccupants(roster, tracks, in)
	cov := poserLesSieges(t.Context(), roster, occ, entreesDesPlaces{table: tableDeDebut(0, 1, 2, 3),
		horloge: horlogeDeSieges(nil)})
	for _, e := range roster {
		if e.Bot && e.Seat != 3 {
			t.Errorf("bot : place %d (%s), attendu 3 — la place du partant", e.Seat, e.SeatSource)
		}
	}
	if cov.SansPlace != 0 || cov.PlacesEnTrop != 0 || cov.Depassements != 0 || cov.IdentitesHorsRoster != 0 {
		t.Errorf("couverture %+v : rien de trop, aucune identite hors roster", cov)
	}
}

func TestBotSansPisteNommeeResteDehors(t *testing.T) { // A-SANS-PISTE
	roster, bots, scan, tracks := bfScrubDeTest()
	if _, admis := admettreLesBotsNommesParDeclaration(roster, bots, scan, teamPublication{}, tracks[:4]); admis != 0 {
		t.Fatalf("admis %d : aucune piste ne porte le nom du bot", admis)
	}
}

func TestBotDontLaDeclarationToucheLHumainResteDehors(t *testing.T) { // A-SIMULTANE
	roster, bots, scan, tracks := bfScrubDeTest()
	bots[0].Declarations = [][2]uint64{{4_000_000, 0}} // f40 : avant l'image-cle qui prouve le depart (f50)
	if _, admis := admettreLesBotsNommesParDeclaration(roster, bots, scan, teamPublication{}, tracks); admis != 0 {
		t.Fatalf("admis %d : la declaration touche la fenetre large de l'humain, rien ne prouve qu'ils se "+
			"succedent", admis)
	}
}

func TestBotAvecEntiteNEntrePasParSaDeclaration(t *testing.T) { // A-ENTITE
	idx, bots, scan, tracks := successionDeTest(true) // l'entite du bot partage une image-cle avec l'humain
	roster, _ := rosterDesOccupants(idx, nil, bots, scan, teamPublication{})
	if _, admis := admettreLesBotsNommesParDeclaration(roster, bots, scan, teamPublication{}, tracks); admis != 0 {
		t.Fatalf("admis %d : un bot qui a une entite se lit par ses entites, pas par cette lecture", admis)
	}
}
