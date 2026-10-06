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
//	              n'est pas reprise ;
//	S-PRIORITE    la place du bot que l'humain remplace passe avant la place liberee le plus tot ;
//	S-RELAIS-DATE la place liberee par un relais a la frame est datee de ce relais ;
//	S-COTOIE-DEBUT un humain lu des le debut de la declaration du bot ne lui succede pas ;
//	S-TIRS        le successeur lit la place du bot par ses tirs ;
//	S-PARTI-AVANT un humain parti avant le retrait du bot ne lui succede pas ;
//	S-LIAISON     le gabarit de `43e96765` de bout en bout, liaison aux entites et a BOT_METADATA
//	              comprise.

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
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

// TestLaPlaceDuBotRemplacePasseAvantLaPlaceLiberee (S-PRIORITE) : le bot tient la place 0 par
// lecture (il reprend l'index du partant) ; la place 3 a ete liberee plus tot. L'humain qui succede
// au bot prend SA place, pas la plus anciennement liberee.
func TestLaPlaceDuBotRemplacePasseAvantLaPlaceLiberee(t *testing.T) {
	roster := []RosterEntry{entree(0, "100"), entree(3, "130"), entree(2, "120"),
		{FilmIndex: 0, Name: "343 PardonMy [bot]"}, entree(9, "190")}
	occ := occupantsFabriques([]int{0, 0, 0, 0, 0}, []intervalleDePresence{{de: 0, a: 40, aMax: 44}},
		iv(0, 20), iv(0, siegeFrames-1), iv(45, 70), iv(50, siegeFrames-1))
	poserDesBots(roster, &occ, 3)
	occ.parEntree[4].vies = [][2]int{{75, siegeFrames - 1}}
	poserLesSieges(context.Background(), roster, occ, entreesDesPlaces{table: tableDeDebut(0, 3, 2),
		horloge: horlogeDeSieges(fallback.NouveauCompteur())})

	if roster[3].Seat != 0 || roster[3].SeatSource != SeatSourceLu {
		t.Fatalf("bot : place %d (%s), attendu 0 lue", roster[3].Seat, roster[3].SeatSource)
	}
	if roster[4].Seat != 0 || roster[4].SeatSource != SeatSourceApparie {
		t.Fatalf("humain : place %d (%s), attendu 0 — celle du bot qu'il remplace, avant la place 3 "+
			"liberee plus tot", roster[4].Seat, roster[4].SeatSource)
	}
}

// TestLeRelaisDateLaLiberation (S-RELAIS-DATE) : la place 7 est liberee A la frame d'arrivee du bot
// (relais) ; la place 3, de son equipe, n'a pas d'occupant anterieur. La liberation par relais
// compte : le bot prend la place du partant, pas la place 3 qu'un autre tiendra plus tard.
func TestLeRelaisDateLaLiberation(t *testing.T) {
	roster := []RosterEntry{entree(7, "170"), entree(3, "130"), entree(2, "120"),
		{FilmIndex: 8, Name: "343 Bachici [bot]"}}
	occ := occupantsFabriques([]int{0, 0, 0, 0}, iv(0, 58), iv(70, siegeFrames-1), iv(0, siegeFrames-1),
		iv(58, 60))
	poserDesBots(roster, &occ, 3)
	poserLesSieges(context.Background(), roster, occ, entreesDesPlaces{table: tableDeDebut(7, 3, 2),
		horloge: horlogeDeSieges(fallback.NouveauCompteur())})

	if roster[3].Seat != 7 {
		t.Fatalf("bot : place %d (%s), attendu 7 — la place que son relais libere", roster[3].Seat,
			roster[3].SeatSource)
	}
}

// TestDejaLaAuDebutDuBotNeLuiSuccedePas (S-COTOIE-DEBUT) : l'humain est lu des la frame ou le bot est
// declare (et n'a aucune vie avant son retrait) : il ne remplace pas le bot, il le cotoie.
func TestDejaLaAuDebutDuBotNeLuiSuccedePas(t *testing.T) {
	roster := []RosterEntry{entree(0, "100"), entree(2, "120"), {FilmIndex: 8, Name: "343 PardonMy [bot]"},
		entree(9, "190")}
	occ := occupantsFabriques([]int{0, 0, 0, 0}, []intervalleDePresence{{de: 0, a: 40, aMax: 44}},
		iv(0, siegeFrames-1), iv(45, 70), iv(45, siegeFrames-1))
	poserDesBots(roster, &occ, 2)
	occ.parEntree[3].vies = [][2]int{{75, siegeFrames - 1}}
	poserLesSieges(context.Background(), roster, occ, entreesDesPlaces{table: tableDeDebut(0, 2),
		horloge: horlogeDeSieges(fallback.NouveauCompteur())})

	if roster[3].Seat == 0 {
		t.Fatalf("humain : place 0 (%s) — present des le debut du bot, il ne lui succede pas", roster[3].SeatSource)
	}
}

// TestLeSuccesseurLitLaPlaceDuBotParSesTirs (S-TIRS) : le bot tient la place 0 par lecture ; l'humain
// qui lui succede tire, apres le retrait, sous l'index 0. Sa place se LIT dans ses tirs : le bot date
// auquel il succede ne la tient pas (Q23), et sa presence s'ouvre au lendemain du retrait.
func TestLeSuccesseurLitLaPlaceDuBotParSesTirs(t *testing.T) {
	roster := []RosterEntry{entree(0, "100"), entree(2, "120"), {FilmIndex: 0, Name: "343 PardonMy [bot]"},
		entree(9, "190")}
	occ := occupantsFabriques([]int{0, 0, 0, 0}, []intervalleDePresence{{de: 0, a: 40, aMax: 44}},
		iv(0, siegeFrames-1), iv(45, 70), iv(50, siegeFrames-1))
	poserDesBots(roster, &occ, 2)
	occ.parEntree[3].vies = [][2]int{{75, siegeFrames - 1}}
	tirs := []FireEventRef{{FilmIndex: 0, TimestampUS: 8_000_000}, {FilmIndex: 0, TimestampUS: 9_000_000}}
	cov := poserLesSieges(context.Background(), roster, occ, entreesDesPlaces{table: tableDeDebut(0, 2), fire: tirs,
		tireurs: []string{"190", "190"}, horloge: horlogeDeSieges(fallback.NouveauCompteur())})

	if roster[3].Seat != 0 || roster[3].SeatSource != SeatSourceTirs || cov.TirsContestes != 0 {
		t.Fatalf("humain : place %d (%s), contestes %d — attendu 0 lue par ses tirs", roster[3].Seat,
			roster[3].SeatSource, cov.TirsContestes)
	}
	if p := roster[3].Presence; len(p) != 1 || p[0].From != 71 {
		t.Errorf("humain : presence %+v, attendu un debut au lendemain du retrait du bot (71)", p)
	}
}

// TestSuccessionParLaLiaison (S-LIAISON) : le gabarit de `43e96765`, de bout en bout — la liaison aux
// entites et a BOT_METADATA (l'equipe du bot vient de sa declaration, sa presence est datee) puis la
// pose des places. Le bot prend la place du partant, l'humain lui succede par le chainage, et sa
// presence s'ouvre au lendemain du retrait.
func TestSuccessionParLaLiaison(t *testing.T) {
	scan := scanDeTest([]int{0, 20, 40, 60, 80},
		grammar.PlayerEntity{Slot: 1, Index: 0, Team: 0, FirstKF: 0, LastKF: 1, Seen: 2},
		grammar.PlayerEntity{Slot: 2, Index: 2, Team: 0, FirstKF: 0, LastKF: 4, Seen: 5},
		grammar.PlayerEntity{Slot: 9, Index: 9, Team: 0, FirstKF: 3, LastKF: 4, Seen: 2})
	equipe := 0
	nom := "343 PardonMy [bot]"
	roster := []RosterEntry{entree(0, "100"), entree(2, "120"), {FilmIndex: 8, Name: nom, Bot: true},
		entree(9, "190")}
	in := entreesDeTest(scan)
	in.bots = []BotIdentity{{FilmIndex: 8, Name: nom, Team: &equipe,
		Declarations: [][2]uint64{{3_500_000, 7_000_000}}}}
	tracks := []Track{vieDe("100", 0, 30), vieDe("120", 0, 99), vieDe("190", 75, 99)}
	occ := lierLesOccupants(roster, tracks, in)
	cov := poserLesSieges(context.Background(), roster, occ, entreesDesPlaces{table: tableDeDebut(0, 2),
		horloge: horlogeDeSieges(fallback.NouveauCompteur())})

	if eq := occ.parEntree[2].equipe; eq == nil || *eq != 0 || !occ.parEntree[2].declaree {
		t.Fatalf("bot : equipe %v, declaree %v — attendu 0 lue dans sa declaration, presence datee",
			deref(eq), occ.parEntree[2].declaree)
	}
	if roster[2].Seat != 0 || roster[3].Seat != 0 {
		t.Fatalf("places bot %d (%s), humain %d (%s) : attendu 0 et 0", roster[2].Seat, roster[2].SeatSource,
			roster[3].Seat, roster[3].SeatSource)
	}
	if p := roster[3].Presence; len(p) != 1 || p[0].From != 70 {
		t.Errorf("humain : presence %+v, attendu un debut au lendemain du retrait du bot (70)", p)
	}
	if cov.Depassements != 0 || cov.SansPlace != 0 || cov.Chevauchements != 0 {
		t.Errorf("couverture %+v : ni depassement, ni entree sans place, ni chevauchement", cov)
	}
}

// TestPartiAvantLeRetraitNeSuccedePas (S-PARTI-AVANT) : X est lu a une seule image-cle pendant la
// declaration du bot et sa presence certaine s'acheve AVANT le retrait — il ne tiendrait la place a
// aucune frame : il ne succede pas. H, lu ensuite et encore la apres le retrait, lui succede. Aucune
// frame de la place 0 n'a deux occupants.
func TestPartiAvantLeRetraitNeSuccedePas(t *testing.T) {
	roster := []RosterEntry{entree(0, "100"), entree(2, "120"), {FilmIndex: 8, Name: "343 PardonMy [bot]"},
		entree(9, "190"), entree(10, "200")}
	occ := occupantsFabriques([]int{0, 0, 0, 0, 0}, []intervalleDePresence{{de: 0, a: 40, aMax: 44}},
		iv(0, siegeFrames-1), iv(45, 70), []intervalleDePresence{{de: 55, a: 55, aMax: 64}},
		iv(65, siegeFrames-1))
	poserDesBots(roster, &occ, 2)
	occ.parEntree[4].vies = [][2]int{{80, siegeFrames - 1}}
	poserLesSieges(context.Background(), roster, occ, entreesDesPlaces{table: tableDeDebut(0, 2),
		horloge: horlogeDeSieges(fallback.NouveauCompteur())})

	if roster[3].Seat == 0 {
		t.Fatalf("X : place 0 (%s) — parti avant le retrait du bot, il ne lui succede pas", roster[3].SeatSource)
	}
	if roster[4].Seat != 0 || len(roster[4].Presence) != 1 || roster[4].Presence[0].From != 71 {
		t.Fatalf("H : place %d (%s), presence %+v — attendu 0, ouverte au lendemain du retrait (71)",
			roster[4].Seat, roster[4].SeatSource, roster[4].Presence)
	}
	par := map[int]int{}
	for _, e := range roster {
		if e.Seat != 0 {
			continue
		}
		for _, p := range e.Presence {
			fin := p.To
			if p.ToMax != nil {
				fin = *p.ToMax
			}
			for f := p.From; f <= fin; f++ {
				par[f]++
				if par[f] > 1 {
					t.Fatalf("place 0 : deux occupants affiches a la frame %d", f)
				}
			}
		}
	}
}
