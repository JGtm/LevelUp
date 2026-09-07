package replay

// pont_muet_test.go — LE PONT NE DOIT PAS SE TAIRE PARCE QUE LA PARTIE COMMENCE TARD.
//
// Ce que ces tests figent (lot `feat/v2-pont-muet`, 2026-09-07) :
//
//   - le calage du fil des morts se cherche sur la plage que LES DONNÉES imposent, plus sur
//     une marge amont de 60 s qui supposait la première mort dans la première minute ;
//   - un film déjà bien calé retient le MÊME entier qu'avant — la re-cuisson ne déplace rien ;
//   - un joueur qui ne meurt JAMAIS entre au roster quand la feuille de match est fournie, et
//     lui seul : sans feuille, le rejeu garde le roster du fil des morts, à l'octet près.
//
// Toutes les fixtures sont synthétiques : aucun film n'est lu, la CI les joue.

import (
	"testing"
)

// pontVie fabrique une vie qui se termine à `finMS` sur l'horloge du film.
func pontVie(slot uint32, finMS int64) lifeSpan {
	return lifeSpan{slot: slot, from: (finMS - 20_000) * 1000, to: finMS * 1000}
}

// pontFixture bâtit un film dont la partie démarre à `originMS` sur l'horloge du film et dont
// la première mort tombe à `premiereMortMS` sur l'horloge du MATCH. Chaque mort termine une
// vie, à `residuMS` près (le bruit d'horloge que la fenêtre d'appariement absorbe).
func pontFixture(originMS, premiereMortMS int64, n int) ([]lifeSpan, []Death) {
	var lives []lifeSpan
	var deaths []Death
	for i := 0; i < n; i++ {
		tMatch := premiereMortMS + int64(i)*9_000
		residu := int64(i%3) * 12 // 0, 12, 24 ms : sous la fenêtre de 150
		lives = append(lives, pontVie(uint32(500+i), originMS+tMatch+residu))
		deaths = append(deaths, Death{XUID: uint64(1000 + i), TimeMS: tMatch})
	}
	return lives, deaths
}

// TestUnFilmDontLaPremiereMortTombeApresUneMinuteEstQuandMemeCale — LE TEST DE MUTATION.
//
// Rouge si la plage redevient `[min(fins) − 60 000, max(fins)]` : le calage vrai (l'origine
// du film) tombe alors 40 s sous la borne basse et rien ne se nomme.
//
// La configuration est celle de `51ebbc0f`, mesurée : les joueurs rejoignent après 51 s de
// mise en place et la première mort tombe à 71,3 s.
func TestUnFilmDontLaPremiereMortTombeApresUneMinuteEstQuandMemeCale(t *testing.T) {
	const origine, premiereMort = 2_800_000, 100_000
	lives, deaths := pontFixture(origine, premiereMort, 12)

	off, apparies := bestDeathOffset(lives, deaths)

	if apparies != len(deaths) {
		t.Fatalf("morts appariées = %d, attendu %d : le calage n'a pas été trouvé "+
			"(la première mort tombe à %d ms, au-delà de la marge amont supprimée)",
			apparies, len(deaths), premiereMort)
	}
	if ecart := off - origine; ecart < -deathMatchWindowMS || ecart > deathMatchWindowMS {
		t.Fatalf("calage retenu %d, attendu %d à %d ms près (écart %d)",
			off, origine, deathMatchWindowMS, ecart)
	}
	if n := nameLivesByDeaths(lives, deaths, off); n != len(deaths) {
		t.Fatalf("vies nommées = %d, attendu %d", n, len(deaths))
	}
}

// TestUnFilmQuiDemarreTresTardEstCaleAussi — la borne n'est pas déplacée, elle a disparu :
// une mise en place de plus de deux minutes (les BTB mesurés : 136 s sur `4f77afc1`) ne doit
// pas plus échouer qu'une d'une minute.
func TestUnFilmQuiDemarreTresTardEstCaleAussi(t *testing.T) {
	const origine, premiereMort = 7_200_000, 140_000
	lives, deaths := pontFixture(origine, premiereMort, 20)

	off, apparies := bestDeathOffset(lives, deaths)

	if apparies != len(deaths) {
		t.Fatalf("morts appariées = %d, attendu %d", apparies, len(deaths))
	}
	if ecart := off - origine; ecart < -deathMatchWindowMS || ecart > deathMatchWindowMS {
		t.Fatalf("calage retenu %d, attendu %d à %d ms près", off, origine, deathMatchWindowMS)
	}
}

// TestUnFilmDejaCaleRetientLeMemeEntier — LA CONTRE-ÉPREUVE, et c'est elle qui autorise la
// re-cuisson du parc : sur un film dont la première mort tombe DANS la première minute — le
// cas que l'ancienne plage traitait déjà —, l'entier retenu doit être exactement celui que
// le balayage linéaire d'avant rendait.
//
// Le témoin est le balayage d'avant, réécrit ici tel quel (`pontCalageHistorique`). Ce n'est
// pas une copie de production : c'est l'oracle contre lequel la neutralité se prouve.
func TestUnFilmDejaCaleRetientLeMemeEntier(t *testing.T) {
	for _, cas := range []struct {
		nom                   string
		origine, premiereMort int64
		n                     int
	}{
		{"mort a 12 s", 400_000, 12_000, 10},
		{"mort a 52 s", 6_000_000, 52_000, 24},
		{"mort a 59 s", 1_234_567, 59_000, 8},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			lives, deaths := pontFixture(cas.origine, cas.premiereMort, cas.n)
			attendu, attenduN := pontCalageHistorique(lifeEndsMS(lives), deaths)

			off, n := bestDeathOffset(lives, deaths)

			if off != attendu || n != attenduN {
				t.Fatalf("calage %d (%d appariées), le balayage d'avant rendait %d (%d) : "+
					"la neutralité sur les films déjà calés n'est plus tenue",
					off, n, attendu, attenduN)
			}
		})
	}
}

// pontCalageHistorique est le balayage linéaire d'avant le 2026-09-07, à l'identique : plage
// `[min(fins) − 60 000, max(fins)]`, pas de 10 ms, plateau centré. ORACLE DE TEST UNIQUEMENT.
func pontCalageHistorique(ends []int64, deaths []Death) (int64, int) {
	lo, hi := ends[0], ends[0]
	for _, e := range ends {
		lo, hi = minI64(lo, e), maxI64(hi, e)
	}
	bestN := -1
	var plateau []int64
	for off := lo - 60_000; off <= hi; off += 10 {
		if n := countDeathMatches(ends, deaths, off); n > bestN {
			bestN, plateau = n, []int64{off}
		} else if n == bestN {
			plateau = append(plateau, off)
		}
	}
	return plateau[len(plateau)/2], bestN
}

// TestLeVoteDesigneLePicEtPasLeBruit — rouge si le vote retient le panier le plus peuplé sans
// distinguer un vrai calage d'un amas de coïncidences : on ajoute des vies qui ne terminent
// aucune mort, et le calage doit rester celui des morts.
func TestLeVoteDesigneLePicEtPasLeBruit(t *testing.T) {
	const origine, premiereMort = 3_000_000, 80_000
	lives, deaths := pontFixture(origine, premiereMort, 15)
	// Douze vies parasites groupées ailleurs : elles votent toutes pour le même écart faux.
	for i := 0; i < 12; i++ {
		lives = append(lives, pontVie(uint32(900+i), origine+premiereMort-400_000+int64(i)))
	}

	off, apparies := bestDeathOffset(lives, deaths)

	if apparies != len(deaths) {
		t.Fatalf("morts appariées = %d, attendu %d : le bruit a emporté le vote", apparies, len(deaths))
	}
	if ecart := off - origine; ecart < -deathMatchWindowMS || ecart > deathMatchWindowMS {
		t.Fatalf("calage retenu %d, attendu %d à %d ms près", off, origine, deathMatchWindowMS)
	}
}

// TestLeCalageResteVideSansMortNiVie — les deux sorties anticipées restent des sorties.
func TestLeCalageResteVideSansMortNiVie(t *testing.T) {
	lives, deaths := pontFixture(1_000_000, 30_000, 4)
	if off, n := bestDeathOffset(nil, deaths); off != 0 || n != 0 {
		t.Fatalf("sans vie : (%d, %d), attendu (0, 0)", off, n)
	}
	if off, n := bestDeathOffset(lives, nil); off != 0 || n != 0 {
		t.Fatalf("sans mort : (%d, %d), attendu (0, 0)", off, n)
	}
}

// TestUnJoueurQuiNeMeurtJamaisEntreAuRosterParLaFeuille — LE TEST DE MUTATION du second
// correctif. Rouge si `rosterOf` ignore son complément : le joueur à 0 mort n'a alors aucun
// index de joueur, aucune de ses pistes n'est rattachable, et il manque au roster publié.
//
// Configuration mesurée sur `3372e7eb` : 8 joueurs à la feuille, 6 au roster publié, et les
// deux manquants sont exactement ceux qui finissent à 0 mort (6 et 8 frags).
func TestUnJoueurQuiNeMeurtJamaisEntreAuRosterParLaFeuille(t *testing.T) {
	deaths := []Death{{XUID: 11, TimeMS: 1000}, {XUID: 13, TimeMS: 2000}, {XUID: 11, TimeMS: 3000}}
	feuille := []uint64{11, 12, 13, 14} // 12 et 14 ne meurent jamais

	got := rosterOf(deaths, feuille)

	attendu := []uint64{11, 12, 13, 14}
	if len(got) != len(attendu) {
		t.Fatalf("roster = %v, attendu %v : les joueurs à 0 mort n'y entrent pas", got, attendu)
	}
	for i, x := range attendu {
		if got[i] != x {
			t.Fatalf("roster = %v, attendu %v (ordre stable, croissant)", got, attendu)
		}
	}
}

// TestSansFeuilleLeRosterResteCeluiDuFilDesMorts — LA CONTRE-ÉPREUVE : le complément est une
// ENTRÉE de l'appelant, pas une invitation à deviner. Le CLI hors ligne et l'ouvrier sans
// faits doivent rendre exactement ce que le fil des morts donne.
func TestSansFeuilleLeRosterResteCeluiDuFilDesMorts(t *testing.T) {
	deaths := []Death{{XUID: 13, TimeMS: 2000}, {XUID: 11, TimeMS: 1000}, {XUID: 13, TimeMS: 3000}}

	sansFeuille := rosterOf(deaths, nil)
	historique := rosterFromDeaths(deaths)

	if len(sansFeuille) != 2 || sansFeuille[0] != 11 || sansFeuille[1] != 13 {
		t.Fatalf("roster sans feuille = %v, attendu [11 13]", sansFeuille)
	}
	if len(historique) != len(sansFeuille) {
		t.Fatalf("rosterFromDeaths = %v, rosterOf(deaths, nil) = %v : les deux doivent coïncider",
			historique, sansFeuille)
	}
}

// TestLeRosterNAdmetNiZeroNiDoublon — un xuid nul (une mort dont l'identité n'a pas été lue)
// ne doit pas devenir une entrée de roster, et un joueur présent des deux côtés ne compte
// qu'une fois.
func TestLeRosterNAdmetNiZeroNiDoublon(t *testing.T) {
	deaths := []Death{{XUID: 0, TimeMS: 10}, {XUID: 21, TimeMS: 20}}

	got := rosterOf(deaths, []uint64{21, 0, 22})

	if len(got) != 2 || got[0] != 21 || got[1] != 22 {
		t.Fatalf("roster = %v, attendu [21 22]", got)
	}
}
