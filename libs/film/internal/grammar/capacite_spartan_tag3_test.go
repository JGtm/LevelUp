package grammar

// capacite_spartan_tag3_test.go — LA BRANCHE `tag == 3` D `i57` (`biped-spartan-ability`) AU BIT PRES
// (plan marche complete, A2.A ; jeu : `FUN_142f262d4`, `FUN_14297ea84`, `FUN_142f04664`).
//
// Chaque cas fabrique le flux que l ecrivain pose et exige que le lecteur en consomme exactement la
// longueur, sans toucher la queue de bits a 1 que le flux porte derriere lui. Le jeu lit :
//
//	a = R(1) ; si a : flags = R(6) ; si flags & 1 : c = R(1) ;
//	  c == 0 : une position ; c == 1 : un entier a largeur variable de categorie 0 (13 bits + 2 de queue)
//	           puis, si flags & 0x10, R(2), 3 x R(13), R(1)[R(16)] ;
//	b = R(1) ; si b : une seconde position.
//
// LES POSITIONS SONT LUES SUR LA TABLE PAR INDEX DE LA CARTE (`lecteurSousLaCarte`) : la region 3 de la
// carte d essai porte 18/18/16 bits, que la region jouee (12/12/12) ne porte pas.

import "testing"

// positionRegion3 : la porte a 0, l index 3 sur deux bits, les trois axes de la region 3.
func positionRegion3() []champDeFlux { return e524(3, 2, [3]uint{18, 18, 16}) }

func TestLaCapaciteTag3LitLaSuiteDuJeu(t *testing.T) {
	flags := func(v uint64) []champDeFlux { return seul(champDeFlux{v, 6}) }
	// reference : l entier de categorie 0, 13 bits de valeur et 2 de queue (15 bits, et non 13).
	reference := seul(fixe(13), champDeFlux{3, 2})
	// corps de FUN_142f04664 avec c = 1 : R(2), trois R(13), R(1)[R(16)].
	corps := func(avecSuite bool) []champDeFlux {
		out := seul(fixe(2), fixe(13), fixe(13), fixe(13))
		if avecSuite {
			return append(out, bit(true), fixe(16))
		}
		return append(out, bit(false))
	}
	cas := []struct {
		nom    string
		champs []champDeFlux
	}{
		{"a = 0, b = 0", seul(bit(false), bit(false))},
		{"a = 0, b = 1 : la seconde position", concat(seul(bit(false), bit(true)), positionRegion3())},
		{"a = 1, flags & 1 = 0 : la porte reste fermee", concat(seul(bit(true)), flags(0x20), seul(bit(false)))},
		{"a = 1, flags & 1 = 0 meme avec flags & 0x10", concat(seul(bit(true)), flags(0x1e), seul(bit(false)))},
		{"c = 0 : la position", concat(seul(bit(true)), flags(0x01), seul(bit(false)), positionRegion3(), seul(bit(false)))},
		{"c = 0, flags & 0x10 sans effet", concat(seul(bit(true)), flags(0x11), seul(bit(false)), positionRegion3(), seul(bit(false)))},
		{"c = 1 sans flags & 0x10 : 15 bits et rien d autre", concat(seul(bit(true)), flags(0x01), seul(bit(true)), reference, seul(bit(false)))},
		{"c = 1 avec flags & 0x10, sans suite", concat(seul(bit(true)), flags(0x11), seul(bit(true)), reference, corps(false), seul(bit(false)))},
		{"c = 1 avec flags & 0x10, suite R(16)", concat(seul(bit(true)), flags(0x31), seul(bit(true)), reference, corps(true), seul(bit(false)))},
		{"c = 0 puis la seconde position", concat(seul(bit(true)), flags(0x01), seul(bit(false)), positionRegion3(), seul(bit(true)), positionRegion3())},
		{"c = 1 puis la seconde position", concat(seul(bit(true)), flags(0x01), seul(bit(true)), reference, seul(bit(true)), positionRegion3())},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			buf, total := ecrireFlux(c.champs)
			br := lecteurSousLaCarte(buf, entreeAQuatreRegions(true))
			consumeSpartanAbilityTag3(br)
			if got := br.BitPos(); got != total {
				t.Fatalf("%d bits lus, l ecrivain en pose %d", got, total)
			}
		})
	}
}

// Le tag est lu par `consumeBipedSpartanAbility` : tag 3 = deux bits, puis la suite ci-dessus.
func TestLeTag3DeLaCapaciteSpartanCommenceApresSesDeuxBits(t *testing.T) {
	champs := concat(seul(champDeFlux{3, 2}, bit(true), champDeFlux{0x01, 6}, bit(false)), positionRegion3(), seul(bit(false)))
	buf, total := ecrireFlux(champs)
	br := lecteurSousLaCarte(buf, entreeAQuatreRegions(true))
	consumeBipedSpartanAbility(br)
	if got := br.BitPos(); got != total {
		t.Fatalf("%d bits lus, l ecrivain en pose %d", got, total)
	}
}
