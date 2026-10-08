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
	// ti=12 i20 a i27 : l ecrivain (142edb178) ecrit la presence ; present : le mot etat+0x850, puis
	// FUN_142c94dd4 : le bloc de filtres de FUN_142c7023c (masque sur quatre bits, drapeau d un bit,
	// par filtre present le tag sur quatre bits, le R(1) commun et la charge du tag), le mot, un mot
	// par filtre present, puis une entree d ordre de trois bits par filtre present.
	{"V20a", compNavpointVisualStateGroups0, 12, 1, "0"},
	{"V20b", compNavpointVisualStateGroups0, 12, 1, "1 | 00000000000000000000000000001011 | 0000 0 | 11111111111111111111111111111111"},
	{"V21", compNavpointVisualStateGroups1, 12, 1, "1 | 10000000000000000000000000000001 | 0101 1 | 0001 1 0 | 1001 0 00000000000000000000000000000111 | " +
		"00000000000000000000000000000010 | 00000000000000000000000000000011 | 11111111111111111111111111111100 | 001 000"},
	{"V27", compNavpointVisualStateGroups7, 12, 1, "1 | 00000000000000000000000000000000 | 1000 1 | 0000 | 01010101010101010101010101010101 | 11111111111111111111111111111111 | 111"},
	// ti=10 i22 (niveau 2) : l ecrivain (142edb250 -> FUN_142c7023c) ecrit le masque, le drapeau d un
	// bit, puis par filtre present le tag sur quatre bits, le R(1) commun et la charge du tag.
	{"I22a", compManagedObjectInteractionFilter, 10, 2, "0000 1"},
	{"I22b", compManagedObjectInteractionFilter, 10, 2, "0011 0 | 0100 1 101010101 | 1000 0 10101010"},
	// ti=11 i4 (niveau 2) : l ecrivain (142edb5cc -> FUN_142c7023c) ecrit le meme bloc ; ici un filtre
	// de tag 6 (liste de references : R(4) de compte, par entree une porte puis la reference) et un de
	// tag 11 (index derriere une porte inversee).
	{"O4a", compObjectiveInteractionFilter, 11, 2, "0000 0"},
	{"O4b", compObjectiveInteractionFilter, 11, 2, "1001 1 | 0110 1 0001 0 | 1011 0 0 10101"},
	// ti=10 i23 : l ecrivain (142edb23c -> FUN_142ed0ec8) ecrit les deux bits de l octet etat+0x170.
	{"O23a", compManagedObjectFlags, 10, 1, "10"},
	{"O23b", compManagedObjectFlags, 10, 1, "01"},
	// ti=12 i17 : l ecrivain (142edb084 -> FUN_1407edaf4) ecrit le mot etat+0x710 sur 32 bits.
	{"N17a", compNavpointObjectMarker, 12, 1, "00000000000000000000010011010010"},
	{"N17b", compNavpointObjectMarker, 12, 1, "11111111111111111111111111111111"},
	// ti=10 i18 a i21 : l ecrivain (142edb3a4) ecrit le mot etat+0x54+4*index sur 32 bits.
	{"O18a", compManagedObjectNetworkedProperty, 10, 1, "10000000000000000000000000000011"},
	{"O18b", compManagedObjectNetworkedProperty, 10, 1, "00000000000000000000000000000000"},
	// ti=45 i1 : l ecrivain (142edbda4) ecrit etat+0x14 & 0x3f sur six bits, puis etat+0x18 & 0xf sur
	// quatre.
	{"S1a", compMatchflowFocusData, 45, 1, "111110 | 0011"},
	{"S1b", compMatchflowFocusData, 45, 1, "000000 | 1000"},
	// ti=12 i13 : l ecrivain (142edb134 -> FUN_142ed18e8) quantifie etat+0x700 sur huit bits.
	{"N13a", compNavpointTopProgress, 12, 1, "10000000"},
	{"N13b", compNavpointTopProgress, 12, 1, "11111111"},
	// ti=12 i15 : l ecrivain (142edadf0 -> FUN_142ed18e8) quantifie etat+0x708 sur huit bits.
	{"N15a", compNavpointBottomProgress, 12, 1, "01111111"},
	{"N15b", compNavpointBottomProgress, 12, 1, "00000000"},
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
