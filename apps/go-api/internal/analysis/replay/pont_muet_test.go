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
	"sort"
	"testing"
)

// pontVie fabrique une vie qui se termine à `finMS` sur l'horloge du film.
func pontVie(slot uint32, finMS int64) lifeSpan {
	return lifeSpan{slot: slot, from: (finMS - 20_000) * 1000, to: finMS * 1000}
}

// pontFixture bâtit un film dont la partie démarre à `originMS` sur l'horloge du film et dont
// la première mort tombe à `premiereMortMS` sur l'horloge du MATCH. Chaque mort termine une
// vie, à `residuMS` près (le bruit d'horloge que la fenêtre d'appariement absorbe).
//
// LES MORTS NE SONT PAS PÉRIODIQUES, et ce n'est pas un détail : avec un pas constant, décaler
// le calage d'exactement une période apparie chaque mort à la fin de vie SUIVANTE et rend un
// second candidat presque aussi bon — une ambiguïté de la fixture, pas du film. Aucun match
// réel n'a des morts également espacées.
func pontFixture(originMS, premiereMortMS int64, n int) ([]lifeSpan, []Death) {
	var lives []lifeSpan
	var deaths []Death
	tMatch := premiereMortMS
	for i := 0; i < n; i++ {
		residu := int64(i%3) * 12 // 0, 12, 24 ms : sous la fenêtre de 150
		lives = append(lives, pontVie(uint32(500+i), originMS+tMatch+residu))
		deaths = append(deaths, Death{XUID: uint64(1000 + i), TimeMS: tMatch})
		tMatch += 3_000 + int64(i*2_777)%9_000 // pas irrégulier, déterministe
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

	off, apparies, _ := bestDeathOffset(lives, deaths)

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

	off, apparies, _ := bestDeathOffset(lives, deaths)

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

			off, n, _ := bestDeathOffset(lives, deaths)

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

	off, apparies, _ := bestDeathOffset(lives, deaths)

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
	if off, n, _ := bestDeathOffset(nil, deaths); off != 0 || n != 0 {
		t.Fatalf("sans vie : (%d, %d), attendu (0, 0)", off, n)
	}
	if off, n, _ := bestDeathOffset(lives, nil); off != 0 || n != 0 {
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

// pontFixtureAmas ajoute à un film régulier un AMAS : `kFins` fins de vie simultanées (une fin
// de manche ou de film, que `buildLifeSpans` produit dans un même cycle de réplication) et
// `mMorts` morts simultanées (un multi-kill), placées pour que tous leurs couples désignent le
// même écart faux, à `decalage` du vrai calage.
func pontFixtureAmas(originMS, premiereMortMS int64, n, kFins, mMorts int, decalage int64) ([]lifeSpan, []Death) {
	lives, deaths := pontFixture(originMS, premiereMortMS, n)
	const finDeMancheMS = 300_000
	for i := 0; i < kFins; i++ {
		// Toutes dans le même cycle de réplication (~16 ms), très en deçà d'un panier.
		lives = append(lives, pontVie(uint32(900+i), originMS+finDeMancheMS+int64(i%2)*16))
	}
	for j := 0; j < mMorts; j++ {
		deaths = append(deaths, Death{
			XUID:   uint64(9000 + j),
			TimeMS: finDeMancheMS - decalage + int64(j%2)*8,
		})
	}
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].TimeMS < deaths[j].TimeMS })
	return lives, deaths
}

// TestUnAmasDeFinsSimultaneesNEmportePasLeVote — LE TEST DE MUTATION du constat PONT-R1/C1.
//
// Rouge si le vote redevient un comptage par COUPLES : `kFins × mMorts` voix tombent alors dans
// un seul panier de bruit (96 pour 24 fins et 4 morts) et passent devant le vrai calage, que
// l'affinage n'explore jamais — un calage faux, sans aucun signal.
//
// L'oracle est le balayage exhaustif d'avant : la première mort tombe à 30 s, donc DANS son
// ancienne plage. Il n'y a pas d'ambiguïté sur la bonne réponse.
func TestUnAmasDeFinsSimultaneesNEmportePasLeVote(t *testing.T) {
	for _, cas := range []struct {
		nom                  string
		morts, kFins, mMorts int
	}{
		{"90 morts, 24 fins x 4 morts", 90, 24, 4},
		{"60 morts, 24 fins x 4 morts", 60, 24, 4},
		{"20 morts, 8 fins x 4 morts", 20, 8, 4},
		{"12 morts, 8 fins x 4 morts", 12, 8, 4},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			lives, deaths := pontFixtureAmas(5_000_000, 30_000, cas.morts, cas.kFins, cas.mMorts, 400_040)
			attOff, attN := pontCalageHistorique(lifeEndsMS(lives), deaths)

			off, n, second := bestDeathOffset(lives, deaths)

			if off != attOff || n != attN {
				t.Fatalf("calage %d (%d appariées), le balayage exhaustif rend %d (%d) : "+
					"l'amas de %d fins simultanées x %d morts simultanées a emporté le vote",
					off, n, attOff, attN, cas.kFins, cas.mMorts)
			}
			if second >= n {
				t.Fatalf("le second candidat apparie %d contre %d au calage retenu : "+
					"la marge publiée serait mensongère", second, n)
			}
		})
	}
}

// TestLaMargeDuCalageEstPubliee — la paire (retenu, suivant) doit sortir de `bestDeathOffset`,
// sinon rien ne permet de voir un vote trompé. Sur un film sain la marge est franche.
func TestLaMargeDuCalageEstPubliee(t *testing.T) {
	lives, deaths := pontFixtureAmas(5_000_000, 30_000, 40, 12, 4, 400_040)

	_, n, second := bestDeathOffset(lives, deaths)

	if n != 40 {
		t.Fatalf("appariements = %d, attendu 40", n)
	}
	if second == 0 {
		t.Fatalf("second candidat = 0 : l'amas de bruit devrait en former un, "+
			"la marge ne serait pas mesurée (retenu %d)", n)
	}
	if n < deathOffsetMargeMin*second {
		t.Fatalf("marge %d/%d sous le seuil de %d sur une fixture pourtant nette",
			n, second, deathOffsetMargeMin)
	}
}

// TestUnCalageSansConcurrentNAPasDeSecond — sans amas, il n'y a qu'un candidat : le second
// compte est nul, et la garde de marge ne doit pas se déclencher pour autant.
func TestUnCalageSansConcurrentNAPasDeSecond(t *testing.T) {
	lives, deaths := pontFixture(2_000_000, 20_000, 16)

	_, n, second := bestDeathOffset(lives, deaths)

	if n != 16 {
		t.Fatalf("appariements = %d, attendu 16", n)
	}
	if second*deathOffsetMargeMin > n {
		t.Fatalf("second = %d contre %d : la garde s'alarmerait sur un film sain", second, n)
	}
}

// TestSansFeuilleLeRosterEstCeluiDeLAncienCorps — LA CONTRE-ÉPREUVE, refaite après le constat
// PONT-R1/C3 : comparer `rosterOf(d, nil)` à `rosterFromDeaths(d)` ne prouvait plus rien, celui-ci
// étant devenu un délégué d'une ligne — les deux membres étaient le MÊME appel.
//
// L'oracle est désormais une copie LITTÉRALE de l'ancien corps (`pontRosterHistorique`), et la
// comparaison porte sur 300 tirages de xuids réels. C'est la frontière du CLI hors ligne et de
// l'ouvrier sans faits : sans feuille de match, le roster doit être celui d'avant, à l'octet.
func TestSansFeuilleLeRosterEstCeluiDeLAncienCorps(t *testing.T) {
	graine := uint64(20260907)
	suivant := func() uint64 { // xorshift : reproductible, aucune dépendance
		graine ^= graine << 13
		graine ^= graine >> 7
		graine ^= graine << 17
		return graine
	}
	for tirage := 0; tirage < 300; tirage++ {
		deaths := make([]Death, 0, 24)
		for i := 0; i < 1+int(suivant()%24); i++ {
			// Des xuids du domaine réel (]2e15, 3e15[), certains répétés.
			deaths = append(deaths, Death{
				XUID:   2_000_000_000_000_000 + suivant()%1_000_000_000_000_000,
				TimeMS: int64(suivant() % 600_000),
			})
		}
		if len(deaths) > 3 {
			deaths[len(deaths)-1].XUID = deaths[0].XUID // au moins un doublon
		}

		got, want := rosterOf(deaths, nil), pontRosterHistorique(deaths)

		if len(got) != len(want) {
			t.Fatalf("tirage %d : roster de %d entrées, l'ancien corps en rend %d",
				tirage, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("tirage %d, rang %d : %d contre %d — « vide = comportement d'avant » "+
					"n'est plus tenu", tirage, i, got[i], want[i])
			}
		}
	}
}

// pontRosterHistorique est le corps de `rosterFromDeaths` d'avant le 2026-09-07, recopié à
// l'identique. ORACLE DE TEST UNIQUEMENT — il ne doit jamais déléguer au code de production,
// c'est tout son intérêt.
func pontRosterHistorique(deaths []Death) []uint64 {
	seen := map[uint64]bool{}
	out := make([]uint64, 0, 8)
	for _, d := range deaths {
		if !seen[d.XUID] {
			seen[d.XUID] = true
			out = append(out, d.XUID)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// TestLeVoteNeDesignePasLAmasEnPREMIER — LE TEST DE MUTATION du comptage lui-même.
//
// `bestDeathOffset` affine plusieurs candidats : ce filet rattrape un vote trompé, et masque
// donc la faute dans un test de bout en bout. Celui-ci vise le vote SEUL, et exige que le vrai
// calage sorte en TÊTE. Rouge dès qu'une voix est déposée par COUPLE plutôt que par mort :
// `kFins × mMorts` voix passent alors devant les `n` du vrai calage.
func TestLeVoteNeDesignePasLAmasEnPREMIER(t *testing.T) {
	for _, cas := range []struct {
		nom                  string
		morts, kFins, mMorts int
	}{
		{"90 morts, 24 fins x 4 morts", 90, 24, 4},
		{"20 morts, 8 fins x 4 morts", 20, 8, 4},
		{"12 morts, 8 fins x 4 morts", 12, 8, 4},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			const origine = 5_000_000
			lives, deaths := pontFixtureAmas(origine, 30_000, cas.morts, cas.kFins, cas.mMorts, 400_040)

			candidats := voteDeathOffsets(lifeEndsMS(lives), deaths, deathOffsetCandidats)

			if len(candidats) == 0 {
				t.Fatalf("le vote ne désigne aucun candidat")
			}
			if ecart := absI64(candidats[0] - origine); ecart > 2*deathMatchWindowMS {
				t.Fatalf("premier candidat %d, à %d ms du vrai calage %d : l'amas de %d fins "+
					"simultanées x %d morts simultanées a emporté le vote (%d voix par couples "+
					"contre %d morts au vrai calage)",
					candidats[0], ecart, origine, cas.kFins, cas.mMorts,
					cas.kFins*cas.mMorts, cas.morts)
			}
		})
	}
}
