package grammar

// i59_etiquette_loi_test.go — LA LOI DE L ETIQUETTE DU CORPS `i59` ET LES HUIT ETIQUETTES DU
// PORT (lot 5.3.3-c, 2026-09-21 ; port complet le 2026-10-08, lot des arrets de la vue B, suite).
//
// # CE QUE L ECRIVAIN DIT, LU A L OCTET
//
// `FUN_142f21c0c` — le lecteur d etiquette du corps `tag==3` d `i59` (`FUN_142f25e90`) — lit
// EXACTEMENT TROIS BITS et range `brut + 1` :
//
//	*(param_1 + 0x2c) += 3            // le compteur de bits avance de 3, et de 3 seulement
//	*param_3 = (octet de tete >> 5) + 1
//
// `FUN_142f25e90` lit ensuite le prefixe (`FUN_142f26e40`) et six drapeaux (`FUN_14297ea84`),
// puis dispatche sur la valeur RANGEE, de 1 a 6, et **rend la main sans lire un bit de plus
// au-dela de 6**. L ecrivain (`FUN_142f272ac`) ecrit les memes champs dans le meme ordre.
//
// # CE QUE CE TEST FIGE
//
// Pour les HUIT valeurs brutes, un corps ECRIT COMME L ECRIVAIN L ECRIT (portes ouvertes et
// fermees melangees, pour qu une porte lue a la place d une autre se voie), que le port doit
// consommer au bit pres ; et la position du prefixe, rendue aux largeurs de la carte quand la
// reference est absente et que la plage est indexee.

import "testing"

// i59LoiBits est la largeur de l etiquette chez l ecrivain : trois bits, pas un de plus.
const i59LoiBits = 3

// i59LoiEtiquette applique la loi de l ecrivain a une valeur brute.
func i59LoiEtiquette(brut uint64) uint64 { return brut + 1 }

// TestI59LoiDeLEtiquetteEstBrutPlusUn epingle la loi elle-meme : elle est le seul fait de ce
// fichier dont depend la lecture des six autres branches.
func TestI59LoiDeLEtiquetteEstBrutPlusUn(t *testing.T) {
	for brut := range uint64(8) {
		if got, veut := i59LoiEtiquette(brut), brut+1; got != veut {
			t.Fatalf("brut %d : etiquette %d, attendu %d", brut, got, veut)
		}
	}
	// L ETIQUETTE 0 EST INATTEIGNABLE : la branche morte de l ecrivain.
	for brut := range uint64(8) {
		if i59LoiEtiquette(brut) == 0 {
			t.Fatalf("brut %d rend l etiquette 0 : la branche morte de l ecrivain serait "+
				"atteignable, et la lecture de FUN_142f21c0c serait fausse", brut)
		}
	}
	// LES BRUTS 6 ET 7 SONT LES SEULS HORS DU `switch` de l ecrivain (1 a 6).
	for _, brut := range []uint64{6, 7} {
		if e := i59LoiEtiquette(brut); e <= 6 {
			t.Fatalf("brut %d rend l etiquette %d : elle tomberait DANS le switch", brut, e)
		}
	}
}

// Les morceaux de flux de l ecrivain du corps, en largeurs du jeu.
var (
	// i59RefAbsente : `FUN_1408f0ac4` porte fermee.
	i59RefAbsente = seul(bit(false))
	// i59RefCategorie5 / i59RefCategorie0 : porte ouverte, l entier de la categorie (8 et 13 bits),
	// la queue R(2).
	i59RefCategorie5 = seul(bit(true), fixe(8), fixe(2))
	i59RefCategorie0 = seul(bit(true), fixe(13), fixe(2))
	// i59VecPlein / i59VecConstant : `FUN_142f26e9c`, porte a 0 (R(24) + R(12)) ou a 1.
	i59VecPlein    = seul(bit(false), fixe(24), fixe(12))
	i59VecConstant = seul(bit(true))
)

// i59Prefixe : `FUN_142f27da4` sans reference — la porte fermee, puis la position de
// `FUN_1407eb600` au niveau 0x10, a l index 0 de la plage (largeurs de la carte) — et les six
// drapeaux de `FUN_1429980a0`.
func i59Prefixe() []champDeFlux {
	return concat(i59RefAbsente, e524(0, 1, axesCarteNiveau16), seul(fixe(6)))
}

// i59CorpsParEtiquette rend, par valeur brute, ce que l ecrivain pose apres le prefixe.
func i59CorpsParEtiquette() map[uint64][]champDeFlux {
	return map[uint64][]champDeFlux{
		0: seul(bit(true), fixe(8)),                   // etiquette 1 : FUN_142ecf8a0
		1: concat(i59RefCategorie5, seul(bit(false))), // etiquette 2 : le tir
		2: concat(i59RefCategorie0, i59RefAbsente, i59VecPlein, i59VecConstant, i59VecPlein,
			seul(fixe(24), fixe(9))), // etiquette 3 : l accroche
		3: concat(i59RefAbsente, i59VecConstant, e524(-1, 1, axesDefautNiveau16), seul(fixe(24), fixe(9))),
		4: concat(i59RefCategorie5, i59VecPlein, e524(0, 1, axesCarteNiveau16), seul(fixe(24), fixe(9))),
		5: concat(seul(bit(false)), i59RefCategorie5, i59RefAbsente, i59VecConstant, i59VecPlein,
			seul(bit(true), fixe(24))), // etiquette 6 : porte fermee -> categorie 5
		6: nil, // etiquette 7 : aucune charge
		7: nil, // etiquette 8 : aucune charge
	}
}

// TestI59LesHuitEtiquettesLisentCeQueLEcrivainEcrit : chaque corps est consomme au bit pres, et la
// position du prefixe est rendue aux largeurs de la carte.
func TestI59LesHuitEtiquettesLisentCeQueLEcrivainEcrit(t *testing.T) {
	for brut, corps := range i59CorpsParEtiquette() {
		flux := concat(seul(champDeFlux{brut, i59LoiBits}), i59Prefixe(), corps)
		buf, total := ecrireFlux(flux)
		br := LecteurSur(buf)
		st := AbilityNonPredictedState{Inner: -1}
		consumeAbilityAnchorBody(br, &st)
		if got := br.BitPos(); got != total {
			t.Errorf("brut %d (etiquette %d) : %d bits consommes, l ecrivain en pose %d", brut,
				i59LoiEtiquette(brut), got, total)
		}
		if st.Inner != int(brut) {
			t.Errorf("brut %d : Inner = %d — le champ porte le BRUT, pas l etiquette", brut, st.Inner)
		}
		if !st.PosCarte || st.Reference {
			t.Errorf("brut %d : position du prefixe non rendue (PosCarte %v, Reference %v)", brut,
				st.PosCarte, st.Reference)
		}
		if st.PosQ[2] != uint32(motif(axesCarteNiveau16[2])) {
			t.Errorf("brut %d : PosQ[2] = %d, l ecrivain a pose %d", brut, st.PosQ[2],
				motif(axesCarteNiveau16[2]))
		}
	}
}

// TestI59LePrefixeAReferenceNeRendPasDePosition : la reference presente (categorie 1, sonde a 0)
// fait lire a `FUN_142f04664` sa branche `c` — R(2), trois R(13), R(1)[R(16)] — et aucune position.
func TestI59LePrefixeAReferenceNeRendPasDePosition(t *testing.T) {
	flux := concat(seul(champDeFlux{1, i59LoiBits}),
		seul(bit(true), bit(false), fixe(13), fixe(2)),
		seul(fixe(2), fixe(13), fixe(13), fixe(13), bit(true), fixe(16)),
		seul(fixe(6)), i59CorpsParEtiquette()[1])
	buf, total := ecrireFlux(flux)
	br := LecteurSur(buf)
	st := AbilityNonPredictedState{Inner: -1}
	consumeAbilityAnchorBody(br, &st)
	if got := br.BitPos(); got != total {
		t.Fatalf("%d bits consommes, l ecrivain en pose %d", got, total)
	}
	if !st.Reference || st.PosCarte {
		t.Fatalf("Reference %v, PosCarte %v : attendu une reference et aucune position", st.Reference,
			st.PosCarte)
	}
}
