package objectiveevents

// named_bornes_test.go — LES DEUX BORNES DE PRUDENCE DU DEROULAGE (plan §4 lot 4b, D13).
//
// Ce que ces tests verrouillent, et pourquoi chacun existe :
//
//	serie saine INTACTE       la borne par pas ne doit toucher AUCUN film sain. Le cas limite
//	                          n'est pas invente : 3 est le pire deroulage d'un pas SAIN mesure
//	                          sur les 68 artefacts du parc confrontes a l'oracle API et a la
//	                          feuille de match (lot 6.7-B1, item 6). Il PASSE.
//	pas ABERRANT refuse       64 est le plus petit pas que l'oracle refute (`cde26226`,
//	                          comp 24 A : 65 vols publies pour UN reel). Il est REJETE, et le
//	                          vol reel survit — c'est la separation que la borne mesure.
//	premier terme ENORME      `prev` part de zero et n'est jamais reinitialise : la grandeur
//	                          qui explose est `pts[0].Value - 0`, pas un ecart entre deux
//	                          echantillons. C'est la forme exacte des quatre bombes.
//	saut INTERMEDIAIRE        la meme borne au milieu d'une serie par ailleurs saine — et la
//	                          preuve que `prev` AVANCE malgre le rejet (sinon le point suivant
//	                          rejouerait le meme ecart geant et la borne n'aurait rien borne).
//	budget EPUISE             le plafond total, atteint en cours de serie : le deroulage
//	                          s'arrete la et la passe n'emet plus rien.
//	budget qui TRAVERSE       note N-AV : un plafond verifie seulement ENTRE les series
//	                          laisserait un appel isole allouer des gigaoctets. Le solde doit
//	                          descendre DANS l'appel.

import (
	"testing"
)

// pt abrege la construction d'un point de serie.
func pt(ms int, v int64) ScorePoint { return ScorePoint{TimeMS: ms, Slot: 12, Value: v} }

// TestIncrementTimesSerieSaineIntacte — le pire pas SAIN mesure passe sans etre touche.
//
// LA REFERENCE A CHANGE AU LOT 6.7-B1 (item 6). Elle valait 17 306, « pire deroulage d'un film
// SAIN » — une qualification que l'oracle API refute : le MEME emplacement (comp 20 B) produit
// 84 assistances de capture pour ZERO reelle sur `a0c36016`. Le pire pas reellement sain du
// parc vaut 3 (`kills`, un joueur), et 376 des 379 triples d'action d'objectif valent 1.
func TestIncrementTimesSerieSaineIntacte(t *testing.T) {
	const pireSaut = 3 // parc de 68 artefacts, 2026-09-11 : le plus gros pas confirme par l'oracle
	pts := []ScorePoint{pt(100, 1), pt(200, 2), pt(345931, 2+pireSaut)}
	b := newEventBudget("test")
	out := incrementTimes(pts, statSlotKey{20, sideB}, b)

	if len(out) != 2+pireSaut {
		t.Fatalf("%d evenements, attendu %d — la borne a coupe un film SAIN", len(out), 2+pireSaut)
	}
	if out[0] != 100 || out[1] != 200 || out[2] != 345931 || out[len(out)-1] != 345931 {
		t.Fatalf("instants mal dates : %d %d %d ... %d", out[0], out[1], out[2], out[len(out)-1])
	}
	if b.rejetes != 0 || b.tronque {
		t.Fatalf("rejets=%d tronque=%v — aucun des deux ne doit se declencher ici", b.rejetes, b.tronque)
	}
	if want := maxNamedEventsPerFilm - (2 + pireSaut); b.reste != want {
		t.Fatalf("solde %d, attendu %d", b.reste, want)
	}
}

// TestIncrementTimesPasAberrantDeSoixanteQuatreRejete — LE POINT DE L'ITEM 6, sur la forme
// EXACTE que l'oracle refute : `cde26226`, comp 24 A, un joueur qui publie 65 vols de drapeau
// pour UN a l'oracle. Le vol reel precede le saut ; apres rejet il survit, seul.
//
// MUTATION : remettre `maxUnrollPerStep` a 100 000 rougit ce test — les 64 unites ressortent.
func TestIncrementTimesPasAberrantDeSoixanteQuatreRejete(t *testing.T) {
	const aberrant = 64 // parc de 68 artefacts : le PLUS PETIT pas que l'oracle refute
	pts := []ScorePoint{pt(100, 1), pt(200, 1+aberrant)}
	b := newEventBudget("test")
	out := incrementTimes(pts, statSlotKey{24, sideA}, b)

	if len(out) != 1 || out[0] != 100 {
		t.Fatalf("%v, attendu le SEUL vol reel date de 100 ms", out)
	}
	if b.rejetes != 1 {
		t.Fatalf("rejets=%d, attendu 1", b.rejetes)
	}
}

// TestIncrementTimesAssistsExplosivesRamenentLaFeuilleDeMatch — la mesure de bout en bout des
// trois films a `assists` explosives, reproduite a l'unite : six assistances reelles, puis un
// pas de 9 482 (`16ea3668`, comp 3 A). L'oracle de la feuille dit SIX.
func TestIncrementTimesAssistsExplosivesRamenentLaFeuilleDeMatch(t *testing.T) {
	pts := []ScorePoint{pt(100, 1), pt(200, 2), pt(300, 3), pt(400, 4), pt(500, 5), pt(600, 6),
		pt(700, 9488)}
	b := newEventBudget("test")
	out := incrementTimes(pts, statSlotKey{3, sideA}, b)

	if len(out) != 6 {
		t.Fatalf("%d assistances publiees, attendu 6 (la feuille de match)", len(out))
	}
	if b.rejetes != 1 {
		t.Fatalf("rejets=%d, attendu 1", b.rejetes)
	}
}

// TestIncrementTimesPremierTermeEnormeRejete — la forme exacte de `51101d1d` : le PREMIER
// point porte deja le deroulage geant, `prev` valant zero.
func TestIncrementTimesPremierTermeEnormeRejete(t *testing.T) {
	const bombe = 2163333610 // mesure 4b.1 : `51101d1d`, comp 20 B, slot 24, t = 136 636 ms
	pts := []ScorePoint{pt(136636, bombe), pt(200000, bombe+3)}
	b := newEventBudget("test")
	out := incrementTimes(pts, statSlotKey{20, sideB}, b)

	// Le point geant n'emet rien ; `prev` a saute a sa valeur, donc le point suivant ne
	// deroule que SON propre ecart (3), et non les 2,1 milliards depuis zero.
	want := []int{200000, 200000, 200000}
	if len(out) != len(want) {
		t.Fatalf("%d evenements, attendu %d", len(out), len(want))
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("evenement %d date %d, attendu %d", i, out[i], want[i])
		}
	}
	if b.rejetes != 1 || b.tronque {
		t.Fatalf("rejets=%d tronque=%v, attendu 1 et false", b.rejetes, b.tronque)
	}
}

// TestIncrementTimesSautIntermediaireRejete — la meme borne au milieu d'une serie saine.
func TestIncrementTimesSautIntermediaireRejete(t *testing.T) {
	const bombe = 537698416 // mesure 4b.1 : la PLUS PETITE bombe (`1c4c63c2`, comp 22 A)
	pts := []ScorePoint{pt(100, 1), pt(200, 2), pt(300, 2+bombe), pt(400, 2+bombe+2)}
	b := newEventBudget("test")
	out := incrementTimes(pts, statSlotKey{22, sideA}, b)

	want := []int{100, 200, 400, 400}
	if len(out) != len(want) {
		t.Fatalf("%d evenements, attendu %d : %v", len(out), len(want), out)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("evenement %d date %d, attendu %d", i, out[i], want[i])
		}
	}
	if b.rejetes != 1 {
		t.Fatalf("rejets=%d, attendu 1", b.rejetes)
	}
}

// TestIncrementTimesBudgetEpuiseEnCoursDeRoute — le plafond TOTAL, atteint au milieu d'une
// serie dont chaque pas tient pourtant sous la borne par pas.
func TestIncrementTimesBudgetEpuiseEnCoursDeRoute(t *testing.T) {
	b := newEventBudget("test")
	b.reste = 5 // solde reduit : le plafond reel demanderait un million d'evenements pour rien
	pts := []ScorePoint{pt(100, 3), pt(200, 6), pt(300, 20)}
	out := incrementTimes(pts, statSlotKey{21, sideA}, b)

	// Le premier pas (3) tient ; le second (3) ne tient plus dans les 2 restants : le
	// deroulage s'arrete AVANT lui, il n'est pas emis a moitie.
	if len(out) != 3 {
		t.Fatalf("%d evenements, attendu 3 : %v", len(out), out)
	}
	if !b.tronque {
		t.Fatal("la passe doit etre marquee TRONQUEE")
	}
	if b.reste != 2 {
		t.Fatalf("solde %d, attendu 2 (le pas refuse n'est pas debite)", b.reste)
	}
	// Une passe tronquee n'emet plus rien du tout, meme une serie qui tiendrait.
	if reste := incrementTimes([]ScorePoint{pt(400, 1)}, statSlotKey{2, sideA}, b); reste != nil {
		t.Fatalf("appel apres troncature : %v, attendu nil", reste)
	}
}

// TestBudgetTraverseLesAppels — note N-AV : le solde doit descendre DANS l'appel, sinon un
// appel isole peut allouer des gigaoctets avant que qui que ce soit ait la main.
func TestBudgetTraverseLesAppels(t *testing.T) {
	b := newEventBudget("test")
	b.reste = 10

	if n := len(incrementTimes([]ScorePoint{pt(100, 8)}, statSlotKey{2, sideA}, b)); n != 8 {
		t.Fatalf("premier appel : %d evenements, attendu 8", n)
	}
	if b.reste != 2 {
		t.Fatalf("solde apres le premier appel : %d, attendu 2", b.reste)
	}
	// Le SECOND appel voit le solde du premier : son pas de 5 n'y tient pas.
	if n := len(incrementTimes([]ScorePoint{pt(200, 5)}, statSlotKey{3, sideA}, b)); n != 0 {
		t.Fatalf("second appel : %d evenements, attendu 0 (solde epuise)", n)
	}
	if !b.tronque {
		t.Fatal("le second appel doit avoir marque la passe tronquee")
	}
}

// TestNamedEventsFromNeutraliseUneBombe — bout en bout : un enregistrement fautif greffe sur
// un corpus sain ne change RIEN a la sortie (le deroulage geant est refuse) et ne fait pas
// exploser la memoire. C'est la propriete qui rend les quatre films-bombes cuisables.
func TestNamedEventsFromNeutraliseUneBombe(t *testing.T) {
	sain := onePassCorpus()
	avant := NamedEventsFrom(sain, ObjectiveTypeFlag)
	if len(avant) == 0 {
		t.Fatal("corpus sans evenement : le test serait vacant")
	}

	// `comp 20 B` = flag_capture_assists, l'emplacement fautif de `51101d1d`. La valeur est
	// posee sur le slot 12, qui porte deja une serie saine du meme emplacement.
	bombe := append(append([]StatRecord{}, sain...),
		StatRecord{TimeMS: 9000, Slot: 12, Round: 0,
			Comps: map[int]StatValue{20: {A: 0, B: 2163333610}}})

	apres := NamedEventsFrom(bombe, ObjectiveTypeFlag)
	if len(apres) != len(avant) {
		t.Fatalf("%d evenements avec la bombe, %d sans — le deroulage aberrant a ete emis",
			len(apres), len(avant))
	}
	for i := range avant {
		if apres[i] != avant[i] {
			t.Fatalf("evenement %d : %+v, attendu %+v", i, apres[i], avant[i])
		}
	}
}
