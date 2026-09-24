package replay

// sieges_revue_test.go — LES CONSTATS DE LA REVUE ADVERSE DU LOT M2 (2026-09-24), UN TEST CHACUN.
//
//	R1-SUCCESSEUR  un bot et un humain qui se succedent sur un MEME index ont chacun leur entree,
//	               et la meme place (4f77afc1 : 343 Doomfruit puis Narotlcs sur l'index 26) — jamais
//	               une fiche de plus ;
//	R1-SIMULTANE   deux occupants du meme index a la meme image-cle : la contradiction d'origine,
//	               le bot reste dehors ; sa vie nommee se COMPTE hors roster ;
//	R5-PARVIES     sur un film balaye, l'entree qu'aucune entite ne porte tient sa place jusqu'a la
//	               veille de l'image-cle qui suit sa derniere vie — mourir n'est pas partir —, et
//	               le repli se compte ;
//	R6-CAPACITE    le depassement se mesure contre la CAPACITE, pas contre les places posees ;
//	R7-COUPDENVOI  l'occupant que la table du debut nomme tient sa place des la frame 0, meme sans
//	               entite ; l'arrivant qui reprend son index, non.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// successionDeTest : un 2 contre 2 (index 0..3, xuids 100..130), images-cles porteuses aux frames
// 10, 30, 50, 70, 90. L'humain d'index 3 part apres f30 ; le bot `343 Robot [bot]` est declare sur
// l'index 3 de f58 a la fin, son entite lue de f70 a f90. `simultane` fait lire l'humain jusqu'a
// f70 : les deux occupants de l'index 3 partagent alors une image-cle.
func successionDeTest(simultane bool) (PlayerIndexTable, []BotIdentity, grammar.PlayerEntityScan, []Track) {
	idx := PlayerIndexTable{ByXUID: map[uint64]int{100: 0, 110: 1, 120: 2, 130: 3}}
	bots := []BotIdentity{{FilmIndex: 3, Name: "343 Robot [bot]", BotID: 7,
		Declarations: [][2]uint64{{5_800_000, 0}}}}
	finHumain := 1
	if simultane {
		finHumain = 3
	}
	scan := scanDeTest([]int{10, 30, 50, 70, 90},
		grammar.PlayerEntity{Slot: 1, Index: 0, Team: 0, LastKF: 4, Seen: 5},
		grammar.PlayerEntity{Slot: 2, Index: 1, Team: 0, LastKF: 4, Seen: 5},
		grammar.PlayerEntity{Slot: 3, Index: 2, Team: 1, LastKF: 4, Seen: 5},
		grammar.PlayerEntity{Slot: 4, Index: 3, Team: 1, LastKF: finHumain, Seen: finHumain + 1},
		grammar.PlayerEntity{Slot: 5, Index: 3, Team: 1, FirstKF: 3, LastKF: 4, Seen: 2})
	tracks := []Track{vieDe("100", 5, 99), vieDe("110", 5, 99), vieDe("120", 5, 99), vieDe("130", 5, 35),
		{Slot: 9, Bot: "343 Robot [bot]", StartFrame: 62, EndFrame: 99}}
	return idx, bots, scan, tracks
}

// poserLaSuccession deroule la chaine de production : roster, liaison, places.
func poserLaSuccession(idx PlayerIndexTable, bots []BotIdentity, scan grammar.PlayerEntityScan,
	tracks []Track) ([]RosterEntry, SeatCoverage, int) {
	roster, admis := rosterDesOccupants(idx, nil, bots, scan, teamPublication{})
	in := entreesDeTest(scan)
	in.bots = bots
	occ := lierLesOccupants(roster, tracks, in)
	var equipes teamPublication
	equipes.poserEquipesParEntree(roster, occ)
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0, 1, 2, 3),
		horloge: horlogeDeSieges(nil)})
	cov.BotsSuccesseurs = admis
	return roster, cov, admis
}

func TestBotSuccesseurDUnHumainTientSaPlace(t *testing.T) {
	idx, bots, scan, tracks := successionDeTest(false)
	roster, cov, admis := poserLaSuccession(idx, bots, scan, tracks)

	var bot, partant *RosterEntry
	for i := range roster {
		switch {
		case roster[i].Bot:
			bot = &roster[i]
		case roster[i].XUID == "130":
			partant = &roster[i]
		}
	}
	if bot == nil || admis != 1 || cov.BotsSuccesseurs != 1 {
		t.Fatalf("admis %d, couverture %d, roster %+v : le bot qui succede a l'humain d'index 3 doit "+
			"avoir son entree", admis, cov.BotsSuccesseurs, roster)
	}
	if bot.Seat != 3 || bot.SeatSource != SeatSourceLu || bot.Team == nil || *bot.Team != 1 {
		t.Errorf("bot : place %d (%s), equipe %v — attendu la place 3 du partant, lue, equipe 1",
			bot.Seat, bot.SeatSource, deref(bot.Team))
	}
	fin := partant.Presence[len(partant.Presence)-1]
	if fin.ToMax == nil || *fin.ToMax >= bot.Presence[0].From {
		t.Errorf("partant %+v, bot %+v : l'affichage du partant s'arrete avant l'arrivee du bot",
			partant.Presence, bot.Presence)
	}
	if cov.IdentitesHorsRoster != 0 || cov.Sieges != 4 || cov.PlacesEnTrop != 0 || cov.Depassements != 0 {
		t.Errorf("couverture %+v : aucune identite hors roster, quatre places, rien de trop", cov)
	}
}

func TestBotSimultaneAUnHumainResteDehors(t *testing.T) {
	idx, bots, scan, tracks := successionDeTest(true)
	roster, cov, admis := poserLaSuccession(idx, bots, scan, tracks)
	for _, e := range roster {
		if e.Bot {
			t.Fatalf("roster %+v : deux occupants de l'index 3 a la meme image-cle se contredisent, "+
				"le bot n'entre pas", roster)
		}
	}
	if admis != 0 || cov.IdentitesHorsRoster != 1 {
		t.Errorf("admis %d, hors roster %d : attendu 0 et 1 — sa vie nommee se COMPTE", admis,
			cov.IdentitesHorsRoster)
	}
}

func TestBotSansDeclarationDateeResteDehors(t *testing.T) {
	idx, bots, scan, _ := successionDeTest(false)
	bots[0].Declarations = nil
	if roster, admis := rosterDesOccupants(idx, nil, bots, scan, teamPublication{}); admis != 0 || len(roster) != 4 {
		t.Errorf("admis %d, %d entrees : sans declaration datee, rien ne dit que le bot SUCCEDE", admis,
			len(roster))
	}
}

func TestPresenceParLesViesSurFilmBalaye(t *testing.T) {
	// L'entite de l'index 3 est revendiquee par deux humains : liee a personne. L'humain 130 n'a
	// donc que ses vies ; la derniere finit a f35, l'image-cle porteuse suivante est a f50.
	scan := scanDeTest([]int{10, 30, 50, 70, 90},
		grammar.PlayerEntity{Slot: 4, Index: 3, Team: 1, LastKF: 4, Seen: 5})
	roster := []RosterEntry{entree(3, "130"), entree(3, "140")}
	tracks := []Track{vieDe("130", 12, 20), vieDe("130", 25, 35), vieDe("140", 15, 18)}
	occ := lierLesOccupants(roster, tracks, entreesDeTest(scan))
	fb := fallback.NouveauCompteur()
	cov := poserLesSieges(roster, occ, entreesDesPlaces{horloge: horlogeDeSieges(fb)})

	p := roster[0].Presence
	if len(p) != 1 || p[0].To != 35 || p[0].ToMax == nil || *p[0].ToMax != 49 {
		t.Fatalf("presence %+v : attendu to 35, affichee jusqu'a 49 (veille de l'image-cle f50)", p)
	}
	if cov.PresencesParLesVies != 2 || fb.Compte(fallback.NomPresenceDUneEntreeParSesVies) != 2 {
		t.Errorf("presencesParLesVies %d, repli %d : attendu 2 et 2", cov.PresencesParLesVies,
			fb.Compte(fallback.NomPresenceDUneEntreeParSesVies))
	}
}

func TestDepassementContreLaCapacite(t *testing.T) {
	// 1 contre 1 : la place 0 est tenue de bout en bout ; l'arrivant de la meme equipe, present en
	// meme temps, n'a pas de place (`index`) — sa tuile est EN TROP. Une entree sans equipe se
	// compte a part.
	roster := []RosterEntry{entree(0, "100"), entree(1, "110"), entree(9, "900"), entree(8, "800")}
	occ := occupantsFabriques([]int{0, 1, 0, 0}, iv(0, 99), iv(0, 99), iv(40, 99), iv(10, 20))
	occ.parEntree[3].equipe = nil
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0, 1), horloge: horlogeDeSieges(nil)})

	if cov.Capacite != 1 || cov.PlacesEnTrop != 1 || cov.Depassements != 60 || cov.SansEquipe != 1 {
		t.Errorf("couverture %+v : capacite 1, une place en trop, 60 frames au-dela, une entree sans "+
			"equipe", cov)
	}
}

func TestOccupantDeLaTableTenuDesLeCoupDEnvoi(t *testing.T) {
	// Sans entite : l'humain assis au siege 0 par la table apparait a f22 — il est la depuis f0.
	// L'arrivant qui reprend l'index 0 a f60 n'etait pas a la table : il commence a sa vie.
	roster := []RosterEntry{{XUID: "1000", FilmIndex: 0}, {XUID: "2000", FilmIndex: 0}}
	occ := occupantsFabriques([]int{1, 1}, iv(22, 40), iv(60, 99))
	occ.balaye = false
	for i := range occ.parEntree {
		occ.parEntree[i].lue = false
	}
	poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0), horloge: horlogeDeSieges(nil)})

	if roster[0].Presence[0].From != 0 || roster[1].Presence[0].From != 60 {
		t.Errorf("presences %+v / %+v : l'occupant de la table des f0, l'arrivant a sa vie",
			roster[0].Presence, roster[1].Presence)
	}
}
