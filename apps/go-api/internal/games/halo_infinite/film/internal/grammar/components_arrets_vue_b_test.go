package grammar

import (
	"strings"
	"testing"
)

// components_arrets_vue_b_test.go — les lecteurs des composants ou la marche depuis la fin de la
// vue A s arretait (`ti=12`, `ti=45`, `ti=10`), confrontes a des vecteurs ECRITS COMME L ECRIVAIN
// DU JEU LES ECRIT : bits dans l ordre d ecriture, chaque champ poids fort en tete. Un vecteur est
// tenu quand le lecteur de production, appele par la chaine de dispatch avec l archetype du
// composant, consomme EXACTEMENT ses bits ; une queue de 64 uns suit chaque vecteur
// ([tamponDeVecteur]), qu un lecteur qui lirait trop consommerait.

// vecteurArret : un etat ecrit par l ecrivain du jeu, l archetype et le niveau du composant.
type vecteurArret struct {
	id, composant string
	ti, niveau    uint32
	bits          string // '0' et '1' ; tout autre caractere est un separateur de lecture
}

// vecteursArrets : un vecteur par composant et par forme de l ecrivain.
var vecteursArrets = []vecteurArret{
	// ti=12 i16 : l ecrivain (142ed0e2c) ecrit les cinq bits du mot etat+0x70c.
	{"N16a", compNavpointOverrideFlags, 12, 1, "10110"},
	{"N16b", compNavpointOverrideFlags, 12, 1, "00000"},
	// ti=45 i0 : l ecrivain (FUN_142edbf94) ecrit l index plus 1 sur quatre bits (FUN_1407ebac4), puis
	// les mots etat+0xc, +0x10, +0x4, +0x8 sur 32 bits chacun.
	{"S0a", compMatchflowSequenceData, 45, 1, "0011 | 00000000000000000000000000000001 | 11111111111111111111111111111111 | 10000000000000000000000000000000 | 01010101010101010101010101010101"},
	{"S0b", compMatchflowSequenceData, 45, 1, "0000 | 00000000000000000000000000000000 | 00000000000000000000000000000000 | 00000000000000000000000000000000 | 00000000000000000000000000000000"},
	// ti=10 i2 a i17 : l ecrivain (142edb304) ecrit le mot etat+0x14+4*index sur 32 bits.
	{"O2a", compManagedObjectNavpoint, 10, 1, "11111111111111111111111111111110"},
	{"O2b", compManagedObjectNavpoint, 10, 1, "00000000000000000000000000000000"},
}

// TestLesArretsDeLaVueBLisentCeQueLEcrivainEcrit : chaque vecteur est consomme au bit pres.
func TestLesArretsDeLaVueBLisentCeQueLEcrivainEcrit(t *testing.T) {
	for _, v := range vecteursArrets {
		buf, n := tamponDeVecteur(v.bits)
		br := LecteurSur(buf)
		_, _, porte := consumeByName(br, v.composant, v.ti, v.niveau)
		if !porte {
			t.Errorf("%s (%s) : non porte", v.id, v.composant)
			continue
		}
		if got := br.BitPos(); got != n {
			t.Errorf("%s (%s) : %d bits consommes, l ecrivain en a ecrit %d (%s)", v.id, v.composant,
				got, n, strings.TrimSpace(v.bits))
		}
	}
}

// TestLeDecalageDuMarqueurLitSousLaGarde : `ti=12 i18` (`FUN_140f04f68`) garde sa position par
// `FUN_14076f91c` ; sous la garde (portee de reference), `FUN_1411b259c` lit `R(96)` brut, que
// l ecrivain (`FUN_1407eb61c`) pose dans la meme portee.
func TestLeDecalageDuMarqueurLitSousLaGarde(t *testing.T) {
	buf, total := ecrireFlux(seul(fixe(96)))
	br := LecteurSur(buf)
	p := br.Profil()
	p.Grammaire.PorteeBaseline = true
	br.PoserProfil(p)
	if _, _, ok := consumeByName(br, compNavpointPositionOffset, 12, 1); !ok {
		t.Fatal("ti=12 i18 : non porte")
	}
	if got := br.BitPos(); got != total {
		t.Fatalf("ti=12 i18 sous la garde : %d bits lus, l ecrivain en pose %d", got, total)
	}
}
