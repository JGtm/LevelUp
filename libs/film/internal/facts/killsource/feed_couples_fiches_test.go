package killsource

// feed_couples_fiches_test.go — UNE FICHE DE KILL APPARTIENT AU KILL LE PLUS PROCHE DE SON TUEUR
// (feed_couples_fiches.go).

import "testing"

// rosterDesFiches : U (1), C (6) et M (7) a la table du film, un bot sur l indice 9 tout le film.
func rosterDesFiches() *roster {
	return &roster{
		names:   []string{"U", "C", "M", "343 Ellis" + BotSuffix},
		pin:     map[int]int{1: 0, 6: 1, 7: 2, 9: 3},
		seatPin: map[int]bool{1: true, 6: true, 7: true},
		bots:    botsDuFilmEntier(9),
		nPlay:   10,
	}
}

// TestLaFicheVaAuKillLePlusProche — le temoin du parc `23aff586` : C tue un bot a 339 296 (le film
// n ecrit pas de fiche pour ce kill), U tue C a 340 997, puis C tue M a 341 197. Depuis le premier
// kill, la fenetre de 2,5 s voit la fiche du troisieme (341 193 : C -> M). Elle appartient au kill de
// 341 197, a 4 ms, pas a celui de 339 296, a 1 897 ms : le premier kill reste sans lecture (sa mort
// de bot se lit ensuite au dead-state) et le troisieme lit M, en accord avec la mort voisine.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : [resolveurDeCouples.ficheDUnAutreKill] qui rend toujours faux
// (le premier kill publie (C, M) en contradiction, le troisieme retombe sur une mort de bot, et la
// mort de M revient en orpheline).
func TestLaFicheVaAuKillLePlusProche(t *testing.T) {
	kf := &killFeed{
		events: []feedEvent{
			{timeMS: 339_296, killer: "C"},
			{timeMS: 340_997, killer: "U"}, {timeMS: 340_998, victim: "C", victimXUID: 6},
			{timeMS: 341_197, killer: "C"}, {timeMS: 341_198, victim: "M", victimXUID: 7},
		},
		xuidDe: map[string]uint64{"C": 6, "M": 7},
		names:  []string{"U", "C", "M"},
	}
	st := kf.resoudreCouples([]killEventRec{coupleRec(340_994, 1, 6), coupleRec(341_193, 6, 7)},
		rosterDesFiches(), nil)

	if len(kf.lus) != 2 || kf.lus[1].timeMS != 341_197 || kf.lus[1].victim != "M" {
		t.Fatalf("couples lus = %+v, attendu (U, C) a 340 997 et (C, M) a 341 197", kf.lus)
	}
	if len(kf.orphK) != 1 || kf.orphK[0].timeMS != 339_296 {
		t.Errorf("kills sans victime = %+v, attendu le seul kill de 339 296 (sa mort de bot se lit au "+
			"dead-state)", kf.orphK)
	}
	if st.Contradiction != 0 || st.Accord != 2 || len(kf.orphD) != 0 {
		t.Errorf("accord %d / contradiction %d / morts orphelines %d, attendu 2 / 0 / 0",
			st.Accord, st.Contradiction, len(kf.orphD))
	}
}

// TestUneLectureContrediteVaAuKillDontLaVictimeEstVoisine — la fiche (C -> M) a 1 900 est plus proche
// du kill de 1 000 (900 ms) que de celui de 3 000 (1 100 ms), mais aucun voisin du premier ne porte
// la mort de M, qui est au voisin immediat du second : elle appartient au second.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer la premiere regle (la restriction aux kills voisins) de
// [resolveurDeCouples.ficheDUnAutreKill] (le premier kill publie (C, M) en contradiction, le second
// recolle la mort de M : deux couples pour une mort).
func TestUneLectureContrediteVaAuKillDontLaVictimeEstVoisine(t *testing.T) {
	kf := &killFeed{
		events: []feedEvent{
			{timeMS: 1000, killer: "C"},
			{timeMS: 1500, killer: "U", victim: "C", victimXUID: 6},
			{timeMS: 3000, killer: "C"}, {timeMS: 3001, victim: "M", victimXUID: 7},
		},
		xuidDe: map[string]uint64{"C": 6, "M": 7},
		names:  []string{"U", "C", "M"},
	}
	st := kf.resoudreCouples([]killEventRec{coupleRec(1500, 1, 6), coupleRec(1900, 6, 7)},
		rosterDesFiches(), nil)

	if len(kf.lus) != 1 || kf.lus[0].timeMS != 3000 || kf.lus[0].victim != "M" {
		t.Fatalf("couples lus = %+v, attendu le seul (C, M) a 3 000", kf.lus)
	}
	if len(kf.fab) != 0 || st.Contradiction != 0 || st.Accord != 1 {
		t.Errorf("recolles %d / contradiction %d / accord %d, attendu 0 / 0 / 1",
			len(kf.fab), st.Contradiction, st.Accord)
	}
}
