package killsource

// feed_couples_test.go — LE KILL-EVENT 85 DECIDE LE COUPLE, ET LE RECOLLAGE NE PEUT PLUS
// RATTRAPER EN SILENCE (lot 1.9.3).
//
// # CE QUE CES TESTS TIENNENT, ET POURQUOI PAR MUTATION
//
// Un test qui verifierait seulement « le couple publie est (C, E) » resterait VERT si la lecture
// cessait de servir et que le repli rendait par hasard le meme couple — c est exactement ce qui
// arrive sur 198 des 281 kills sans mort en face du corpus, ou lecture et recollage s accordent.
// Les temoins ci-dessous sont donc construits pour que les DEUX rendent des reponses
// DIFFERENTES : echanger tueur et victime dans l enregistrement fait basculer le couple publie,
// donc rougir. C est la seule forme qui prouve que la lecture DECIDE.

import "testing"

// coupleTemoin : le decor commun. Trois joueurs epingles par la table du film (C, D, E) et un
// BOT epingle par BOT_METADATA, plus un feed ou le recollage et la lecture DIVERGENT :
//
//	t=2000  kill de C, sans mort en face   <- l instant a decider
//	t=2050  mort de D                      <- ce que le RECOLLAGE prendrait (voisin immediat)
//	t=2100  mort de E                      <- ce que la LECTURE nomme
func coupleTemoin() (*killFeed, *roster) {
	kf := &killFeed{
		events: []feedEvent{
			{timeMS: 2000, killer: "C"},
			{timeMS: 2050, victim: "D", victimXUID: 44},
			{timeMS: 2100, victim: "E", victimXUID: 55},
		},
		xuidDe: map[string]uint64{"D": 44, "E": 55},
		names:  []string{"C", "D", "E"},
	}
	r := &roster{
		names:   []string{"C", "D", "E", "Bob" + BotSuffix},
		pin:     map[int]int{1: 0, 2: 1, 3: 2, 9: 3},
		seatPin: map[int]bool{1: true, 2: true, 3: true},
		nPlay:   10,
	}
	return kf, r
}

// coupleRec : un kill-event 85 localise a `ms`, tel que [scanKillEvents] le rendrait.
func coupleRec(ms, tueur, victime int) killEventRec {
	return killEventRec{ms: ms, chain: minChain,
		fields: killEventFields{killer: tueur, victim: victime, assist: -1, end: 128}}
}

// TestLeCoupleVientDuKillEvent85 : LE TEMOIN PRINCIPAL. Le film nomme E ; le voisin immediat
// porte la mort de D. C est E qui est publie, et la mort de D reste disponible pour les temps
// qui cherchent un tueur bot ou une mort que personne ne revendique.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : echanger `killer` et `victim` dans [coupleRec] — le film
// dit alors « E a tue C », plus aucun enregistrement ne nomme C en tueur, la lecture se tait et
// le repli republie (C, D). Jouee et restauree au lot 1.9.3.
func TestLeCoupleVientDuKillEvent85(t *testing.T) {
	kf, r := coupleTemoin()
	st := kf.resoudreCouples([]killEventRec{coupleRec(2000, 1, 3)}, r)

	if len(kf.pairs) != 1 || kf.pairs[0].victim != "E" {
		t.Fatalf("couples = %+v, attendu le seul (C, E) — le film ECRIT E, le voisin porte D", kf.pairs)
	}
	if kf.pairs[0].victimXUID != 55 {
		t.Errorf("xuid de la victime = %d, attendu 55 (celui de E, pris a la mort consommee)",
			kf.pairs[0].victimXUID)
	}
	if len(kf.lus) != 1 || len(kf.fab) != 0 {
		t.Errorf("lus = %d / recolles = %d, attendu 1 / 0 : le repli n avait rien a faire ici",
			len(kf.lus), len(kf.fab))
	}
	if len(kf.orphD) != 1 || kf.orphD[0].victim != "D" {
		t.Errorf("morts sans tueur = %+v, attendu la seule mort de D — le recollage la consommait "+
			"a tort", kf.orphD)
	}
	if st.Lus != 1 || st.Recolles != 0 || st.Accord != 1 || st.Contradiction != 0 {
		t.Errorf("compteurs = %+v, attendu 1 lu / 0 recolle / 1 accord / 0 contradiction", st)
	}
}

// TestUneVictimeBotNeFabriquePlusDeCouple : la fabrication que le lot supprime.
//
// Le film nomme un BOT en victime. Le kill-feed etant HUMAIN-SEUL, il n existe aucune mort en
// face : l ancien recollage prenait celle du voisin et publiait un couple qui n a jamais eu lieu.
// La victime est desormais NOMMEE a la source (« les vies anonymes n existent pas »), l instant
// part vers la population des morts de bot, et AUCUNE mort de voisin n est consommee.
func TestUneVictimeBotNeFabriquePlusDeCouple(t *testing.T) {
	kf, r := coupleTemoin()
	st := kf.resoudreCouples([]killEventRec{coupleRec(2000, 1, 9)}, r)

	if len(kf.pairs) != 0 {
		t.Fatalf("couples = %+v, attendu AUCUN : la victime est un bot, il n y a pas de couple "+
			"du kill-feed a publier", kf.pairs)
	}
	if len(kf.botLus) != 1 || kf.botLus[0].victime != 9 || kf.botLus[0].ev.victim != "Bob"+BotSuffix {
		t.Fatalf("morts de bot lues = %+v, attendu la seule victime 9 nommee Bob%s", kf.botLus, BotSuffix)
	}
	if len(kf.orphD) != 2 {
		t.Errorf("morts sans tueur = %d, attendu 2 (D et E) : aucune n est consommee par un couple "+
			"fabrique", len(kf.orphD))
	}
	if st.VictimesBotLues != 1 || st.Recolles != 0 {
		t.Errorf("compteurs = %+v, attendu 1 victime de bot lue et 0 recollage", st)
	}
}

// TestLeRepliNeSertQueLeSilence : sans kill-event, la decomposition est EXACTEMENT celle d avant
// le lot — mecanisme inchange, et le repli se compte.
func TestLeRepliNeSertQueLeSilence(t *testing.T) {
	kf, r := coupleTemoin()
	st := kf.resoudreCouples(nil, r)

	if len(kf.pairs) != 1 || kf.pairs[0].victim != "D" || kf.pairs[0].victimXUID != 44 {
		t.Fatalf("couples = %+v, attendu le seul (C, D) recolle sur le voisin immediat", kf.pairs)
	}
	if len(kf.fab) != 1 || len(kf.lus) != 0 || st.Recolles != 1 || st.Muet != 1 {
		t.Errorf("recolles = %d / lus = %d / compteurs = %+v, attendu 1 recollage sur un silence",
			len(kf.fab), len(kf.lus), st)
	}
}

// TestLeRepliNeSertPasUneLectureAMBIGUE : deux enregistrements non consommes nomment le meme
// tueur et DES VICTIMES DIFFERENTES. La lecture ne tranche pas — donc elle ne decide pas —, le
// repli reprend, et le compteur d ambiguite le dit. C est D14 (b) : le repli ne se declenche
// jamais sur un desaccord avec la lecture, seulement sur son silence ou son indecision.
func TestLeRepliNeSertPasUneLectureAMBIGUE(t *testing.T) {
	kf, r := coupleTemoin()
	st := kf.resoudreCouples([]killEventRec{coupleRec(2000, 1, 3), coupleRec(2400, 1, 2)}, r)

	if len(kf.fab) != 1 || kf.pairs[0].victim != "D" {
		t.Fatalf("couples = %+v / recolles = %d, attendu le repli sur (C, D)", kf.pairs, len(kf.fab))
	}
	if st.Ambigu != 1 || st.Muet != 0 || st.Lus != 0 {
		t.Errorf("compteurs = %+v, attendu 1 ambigu, 0 muet, 0 lu", st)
	}
}

// TestLeCoupleDuMemeInstantConsommeSonEnregistrement : LE PREMIER TEMPS, et il est le resultat.
//
// Le couple que le FEED ecrit au meme instant prend le kill-event dont le couple entier
// correspond ; sans ce temps, ce meme enregistrement serait disponible pour le kill orphelin
// voisin et lui donnerait une victime qui appartient a une autre mort.
func TestLeCoupleDuMemeInstantConsommeSonEnregistrement(t *testing.T) {
	kf, r := coupleTemoin()
	// Une mort ecrite au meme instant : A(1) tue E(3) a 1900.
	kf.events = append([]feedEvent{{timeMS: 1900, killer: "C", victim: "E", victimXUID: 55}}, kf.events...)

	st := kf.resoudreCouples([]killEventRec{coupleRec(1900, 1, 3)}, r)

	if st.MemeInstant != 1 {
		t.Fatalf("couples au meme instant = %d, attendu 1", st.MemeInstant)
	}
	if st.Lus != 0 || st.Recolles != 1 {
		t.Errorf("compteurs = %+v, attendu 0 lu et 1 recolle : le seul enregistrement etait "+
			"deja consomme par le couple du meme instant", st)
	}
}
