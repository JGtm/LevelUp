package replay

// sieges_origine_test.go — UN OCCUPANT PARTI AVANT L'ORIGINE DU DOCUMENT N'Y TIENT AUCUNE PLACE
// (lot M2.3, 2026-09-23 ; defaut trouve par le gate de corpus sur `396cfc92`).
//
// Gabarit mesure sur ce film : la premiere image-clé porteuse tombe 28 s AVANT la frame 0 du
// document, la deuxieme 8 s avant. Un joueur (index 4) n'est lu qu'a la premiere ; le bot qui le
// remplace est declare par BOT_METADATA des la deuxieme et retire 18,4 s plus tard ; un humain
// (index 9) arrive a la troisieme. Bornees d'abord a la grille, les deux premieres presences
// tombaient toutes deux a la frame 0 — « certaines » ensemble —, le bot n'avait pas de place et
// son equipe en affichait cinq.
//
//	O-ORIGINE  le partant d'avant l'origine n'a AUCUNE presence ; le bot tient la place du
//	           partant des la frame 0, l'humain la reprend ; quatre places, aucun depassement ;
//	O-INCERTAIN vu pour la derniere fois avant l'origine, peut-etre encore la apres : aucune
//	           frame certaine (`to < from`), affiche jusqu'a l'image-cle suivante.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// origineDuDocument : l'origine de la grille, en microsecondes de film — 28 s apres la premiere
// image-cle porteuse (images-cles a 0, 20, 40 s... du film).
const origineDuDocument = 28_000_000

func horlogeDOrigine() replayClock {
	return replayClock{origin: origineDuDocument, step: 100_000, frames: 1000}
}

// scanDOrigine : cinq images-cles porteuses, toutes les 20 s depuis l'instant 0 du film.
func scanDOrigine(ents ...grammar.PlayerEntity) grammar.PlayerEntityScan {
	s := grammar.PlayerEntityScan{Scanned: true, Entities: ents}
	for k := uint64(0); k < 5; k++ {
		s.KeyframesUS = append(s.KeyframesUS, k*20_000_000)
	}
	return s
}

func TestPartiAvantLOrigineNeTientAucunePlace(t *testing.T) {
	scan := scanDOrigine(
		grammar.PlayerEntity{Slot: 1, Index: 0, Team: 1, FirstKF: 0, LastKF: 4, Seen: 5},
		grammar.PlayerEntity{Slot: 2, Index: 1, Team: 1, FirstKF: 0, LastKF: 4, Seen: 5},
		grammar.PlayerEntity{Slot: 3, Index: 2, Team: 1, FirstKF: 0, LastKF: 4, Seen: 5},
		grammar.PlayerEntity{Slot: 4, Index: 4, Team: 1, FirstKF: 0, LastKF: 0, Seen: 1}, // parti
		grammar.PlayerEntity{Slot: 5, Index: 8, Team: 1, FirstKF: 1, LastKF: 1, Seen: 1}, // le bot
		grammar.PlayerEntity{Slot: 6, Index: 9, Team: 1, FirstKF: 2, LastKF: 4, Seen: 3},
	)
	roster := []RosterEntry{entree(0, "100"), entree(1, "110"), entree(2, "120"), entree(4, "140"),
		{FilmIndex: 8, Name: "343 PardonMy [bot]", Bot: true}, entree(9, "190")}
	bots := []BotIdentity{{FilmIndex: 8, Name: "343 PardonMy [bot]",
		Declarations: [][2]uint64{{20_000_000, 38_400_000}}}}
	tracks := []Track{vieDe("190", 125, 999)}
	in := entreesDesOccupants{scan: scan, bots: bots, horloge: horlogeDOrigine()}
	occ := lierLesOccupants(roster, tracks, in)
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0, 1, 2, 4),
		horloge: horlogeDOrigine()})

	if len(roster[3].Presence) != 0 {
		t.Errorf("le partant d'avant l'origine : presence %+v, attendu aucune", roster[3].Presence)
	}
	bot, humain := roster[4], roster[5]
	if bot.Seat != 4 || len(bot.Presence) != 1 || bot.Presence[0].From != 0 || bot.Presence[0].To != 103 {
		t.Errorf("le bot : place %d (%s), presence %+v — attendu la place 4 de 0 a 103",
			bot.Seat, bot.SeatSource, bot.Presence)
	}
	if humain.Seat != 4 {
		t.Errorf("l'arrivant : place %d (%s), attendu 4 — il reprend la place du bot", humain.Seat,
			humain.SeatSource)
	}
	if cov.Depassements != 0 || cov.SansPlace != 0 || cov.Sieges != 4 || cov.SansPresence != 1 {
		t.Errorf("couverture %+v : quatre places, aucun depassement, aucun sans place, un partant "+
			"sans presence", cov)
	}
}

func TestVuAvantLOrigineSeulementNEstCertainNullePart(t *testing.T) {
	// Lu aux deux premieres images-cles (-280, -80), plus a la troisieme (+120) : il est parti
	// entre -80 et +120 — peut-etre apres la frame 0.
	scan := scanDOrigine(grammar.PlayerEntity{Slot: 4, Index: 4, Team: 1, FirstKF: 0, LastKF: 1, Seen: 2})
	occ := lierLesOccupants([]RosterEntry{entree(4, "140")}, nil, entreesDesOccupants{scan: scan,
		horloge: horlogeDOrigine()})
	p := occ.parEntree[0].presence
	if len(p) != 1 || p[0] != (intervalleDePresence{de: 0, a: -1, aMax: 119}) {
		t.Fatalf("presence %+v : attendu de 0, AUCUNE frame certaine (a = -1), affiche jusqu'a 119", p)
	}
}
