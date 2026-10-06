package grammar

import (
	"slices"
	"testing"
)

// vue_a_lecture_test.go — les vecteurs de la lecture de la vue A (lots LN et VA), ecrits d apres
// l ecrivain de la vue A (`FUN_140bbd474` : `1`, genre R(7), trois references gardees, charge ;
// terminateur `0` de `FUN_142f2c3b0`) et d apres les lecteurs des charges.

// grammaireDeTest rend la grammaire de la vue A d un film dont la table des genres est de classe
// `classe` et de cardinal `genres`.
func grammaireDeTest(classe classeDeLaVueA, genres int) grammaireDeLaVueA {
	return grammaireDeLaVueA{classe: classe, genres: genres}
}

// grammaireRecente est la grammaire d un film recent : sa table des genres est la table native.
func grammaireRecente() grammaireDeLaVueA { return grammaireDeTest(vueAEgale, GenresVueA) }

// lireSous lit la vue A d un payload (bit de configuration au bit 0) sous le profil par defaut et la
// grammaire `g`.
func lireSous(pay []byte, g grammaireDeLaVueA) FluxVueA {
	return lireLaVueA(pay, 1, ProfilDeBalayageParDefaut(), g)
}

// ecrireEnTeteDeMessage ecrit la continuation, le genre et trois gardes de reference fermees.
func (w *bitWriter) ecrireEnTeteDeMessage(genre uint64) {
	w.bit(1)
	w.bits(genre, 7) // FUN_14080a9d4 : R(7)
	w.bits(0, 3)
}

// TestLaTableDesGenresEstCelleDuJeu : les genres vides sont les treize dont `vtable + 0x10` rend 0
// (et dont `DAT_14474cd90` porte une taille nulle) ; les genres que la production decode deja ont
// le nom et les domaines que leurs decodeurs relisent.
func TestLaTableDesGenresEstCelleDuJeu(t *testing.T) {
	var vides []int
	for g := range GenresVueA {
		if _, vide := descripteurDuGenre(g); vide {
			vides = append(vides, g)
		}
	}
	if want := []int{3, 4, 23, 24, 25, 26, 33, 49, 54, 57, 59, 92, 103}; !slices.Equal(vides, want) {
		t.Fatalf("genres vides %v, attendu %v", vides, want)
	}
	for _, c := range []struct {
		genre int
		nom   string
		dom   [3]int
	}{
		{EventBipedBoardVehicle, "biped_board_vehicle", [3]int{2, 3, 7}},
		{EventUnitExitVehicle, "unit_exit_vehicle", [3]int{1, 1, 7}},
		{TypeTirArme, "action_weapon_fire", [3]int{1, 8, 7}},
		{21, "unit_zoom", [3]int{4, 8, 7}},
	} {
		dom, _ := descripteurDuGenre(c.genre)
		if genresReleves[c.genre].nom != c.nom || dom != c.dom {
			t.Errorf("genre %d : %q %v, attendu %q %v", c.genre, genresReleves[c.genre].nom, dom, c.nom, c.dom)
		}
	}
}

// TestLaVueALueRendLeDebutDeLaVueB : un zoom dont la reference 0 (domaine 4, 9 bits) est presente,
// puis un rechargement (porte de `FUN_1407f2058` a 0 : R(5) lu), puis le terminateur. La fin est le
// bit qui suit le terminateur, et les deux genres sont rendus dans l ordre. MUTATION : la polarite de
// [consumeGate0R] inversee dans [chargeRechargement] -> cinq bits de moins, ROUGE.
func TestLaVueALueRendLeDebutDeLaVueB(t *testing.T) {
	var w bitWriter
	w.bit(1) // configuration
	w.bit(1)
	w.bits(21, 7)
	w.bit(1)           // garde de la reference 0
	w.bits(0x1ab, 9+2) // domaine 4 : R(9), puis R(2)
	w.bits(0, 2)       // gardes des references 1 et 2
	w.bits(2, 2)       // FUN_14080cb98
	w.ecrireEnTeteDeMessage(38)
	w.bits(0xa, 4)  // quatre R(1)
	w.bit(0)        // FUN_1407f2058 : porte a 0
	w.bits(0x15, 5) // R(5)
	w.bit(0)        // terminateur
	fin := w.n
	w.bits(0x3f, 6) // la vue B
	a := lireSous(w.buf, grammaireRecente())
	if !a.Porte || a.Vide || a.Fin != fin || !slices.Equal(a.Genres, []int{21, 38}) || a.finDeTete() != 9 {
		t.Fatalf("vue A %+v (tete jusqu au bit %d), attendu portee jusqu au bit %d, genres [21 38]", a,
			a.finDeTete(), fin)
	}
}

// TestLaVueANeDevineRien : un genre dont la charge depend d une valeur que le film ne declare pas,
// un genre au-dela du cardinal declare, un film dont la table ne se lit pas, un message tronque, un
// genre non vide dont la charge n est pas portee : la lecture s arrete apres le genre du message,
// qu elle rend, et ne lit rien au-dela. MUTATIONS — le cardinal du film ignore ; un genre sans
// charge portee accepte : ROUGE.
func TestLaVueANeDevineRien(t *testing.T) {
	var w bitWriter
	w.bit(1)
	w.ecrireEnTeteDeMessage(15) // Script : le film n a pas declare la simulation de son enregistreur
	w.bits(0, 40)
	if a := lireSous(w.buf, grammaireRecente()); a.Porte || a.Fin != 9 || !slices.Equal(a.Genres, []int{15}) {
		t.Fatalf("genre 15 sans options de partie : %+v", a)
	}
	var z bitWriter
	z.bit(1)
	z.ecrireEnTeteDeMessage(21)
	z.bits(0, 2)
	z.bit(0)
	finZ := z.n
	z.bits(0, 8)
	for _, c := range []struct {
		nom string
		g   grammaireDeLaVueA
		pay []byte
	}{
		{"genre 21 d un film qui en declare 21", grammaireDeTest(vueAPrefixe, 21), z.buf},
		{"film sans table de genres", grammaireDeLaVueA{}, z.buf},
	} {
		if a := lireSous(c.pay, c.g); a.Porte || a.Fin != 9 || !slices.Equal(a.Genres, []int{21}) {
			t.Errorf("%s : %+v, attendu arretee au bit 9 apres le genre 21", c.nom, a)
		}
	}
	var r bitWriter
	r.bit(1)
	r.ecrireEnTeteDeMessage(38) // weapon_reload : dix bits de charge, au-dela des deux octets
	if a := lireSous(r.buf[:2], grammaireRecente()); a.Porte || a.Fin != 9 || !slices.Equal(a.Genres, []int{38}) {
		t.Errorf("charge tronquee : %+v, attendu arretee au bit 9 apres le genre 38", a)
	}
	if a := lireSous(z.buf[:1], grammaireRecente()); a.Porte || a.Fin != 2 || len(a.Genres) != 0 {
		t.Errorf("genre tronque : %+v, attendu arretee au bit 2 sans genre", a)
	}
	if a := lireSous(z.buf, grammaireDeTest(vueAPrefixe, 22)); !a.Porte || a.Fin != finZ {
		t.Fatalf("genre 21 d un film qui en declare 22 : fin %d (%v), attendu %d", a.Fin, a.Porte, finZ)
	}
	var k bitWriter
	k.bit(1)
	k.ecrireEnTeteDeMessage(85) // PlayerKilledEvent : genre non vide, charge refusee sans variante lue
	k.bit(0)                    // lu comme un terminateur si le message passait sans sa charge
	k.bits(0, 16)
	if a := lireSous(k.buf, grammaireRecente()); a.Porte || a.Fin != 9 || !slices.Equal(a.Genres, []int{85}) {
		t.Errorf("genre 85 sans variante lue : %+v, attendu arretee au bit 9 apres le genre 85", a)
	}
}

// TestLeControleDeCorruptionSuitLaCharge : quand le film declare le controle de corruption,
// `FUN_14080a9d4` lit R(1) apres la charge, puis R(32) s il vaut 1 ; jamais apres un genre vide.
func TestLeControleDeCorruptionSuitLaCharge(t *testing.T) {
	bal := ProfilDeBalayageParDefaut()
	bal.Grammaire.ControleDeCorruption = true
	var w bitWriter
	w.bit(1)
	w.ecrireEnTeteDeMessage(21)
	w.bits(1, 2)
	w.bit(1)
	w.bits(0x0bcddcba, 32)
	w.ecrireEnTeteDeMessage(103) // vide : ni charge ni controle
	w.bit(0)
	fin := w.n
	w.bits(0, 8)
	if a := lireLaVueA(w.buf, 1, bal, grammaireRecente()); !a.Porte || a.Fin != fin {
		t.Fatalf("fin %d (%v), attendu %d", a.Fin, a.Porte, fin)
	}
}

// TestLaTableDuFilmSeRangeEnDeuxClasses : la table du film EGALE a la table native (film recent),
// PREFIXE STRICT de la table native (film ancien), et les tables que la grammaire de l executable ne
// lit pas — une version differente, plus longue, vide. Les versions des dix genres dont le lecteur
// consulte la sienne, et les deux plus hautes, sont celles de `DAT_14474cd90`. MUTATIONS : egalite
// lue comme un prefixe, comparaison de version retiree, table native reduite a la version du genre
// 0 -> ROUGE.
func TestLaTableDuFilmSeRangeEnDeuxClasses(t *testing.T) {
	for genre, v := range map[int]uint32{35: 4, 36: 3, 40: 2, 48: 2, 61: 6, 81: 5, 89: 2, 90: 3, 91: 3, 93: 2, 97: 4, 114: 1} {
		if got := versionNative(genre); got != v {
			t.Errorf("version native du genre %d : %d, attendu %d", genre, got, v)
		}
	}
	natives := make([]uint32, GenresVueA)
	for g := range natives {
		natives[g] = versionNative(g)
	}
	autre := slices.Clone(natives)
	autre[TypeTirArme]--
	for _, c := range []struct {
		nom      string
		table    []uint32
		classe   classeDeLaVueA
		cardinal int
	}{
		{"table native", natives, vueAEgale, GenresVueA},
		{"prefixe de 122 genres", natives[:122], vueAPrefixe, 122},
		{"prefixe de 121 genres", natives[:121], vueAPrefixe, 121},
		{"version differente du genre 36", autre, vueAIllisible, 0},
		{"prefixe a version differente", autre[:121], vueAIllisible, 0},
		{"table de 124 genres", append(slices.Clone(natives), 1), vueAIllisible, 0},
		{"table vide", nil, vueAIllisible, 0},
	} {
		if classe, cardinal := classeDesGenres(c.table); classe != c.classe || cardinal != c.cardinal {
			t.Errorf("%s : classe %d, cardinal %d ; attendu %d, %d", c.nom, classe, cardinal, c.classe, c.cardinal)
		}
	}
}

// TestLesDegatsLisentLaPorteInverseeDuJeu : `FUN_1407f15a4` lit `FUN_1407f2058` (R(1) ; R(5) si
// 0). MUTATION : `consumeGateR` au lieu de `consumeGate0R` -> cinq bits decales, ROUGE.
func TestLesDegatsLisentLaPorteInverseeDuJeu(t *testing.T) {
	var w bitWriter
	w.bit(0)               // FUN_14080d69c : porte fermee
	w.bit(0)               // FUN_1407f2058 : porte a 0, R(5) suit
	w.bits(0x1f, 5)        //
	w.bits(0, 0x13)        // FUN_14076dc04, R9D = 0x13
	w.bit(0)               // [+0x20]
	w.bits(0, 5+5+6)       // trois FUN_1406d84b4
	w.bit(0)               // R(1) du bloc froid
	w.bits(0, 15)          // quinze R(1), le dernier a 0
	w.bits(0, 1+3+5+5+1+4) // bit 19, R(3), deux scalaires, echelle, FUN_1407f1f24
	w.bit(1)               // FUN_1407f1e4c : porte a 1, rien ne suit
	w.bits(1, 4)           // [+0x58] = 1
	w.bit(0)               // FUN_141015740 absent
	w.bits(0, 8)           // R(8) garde par [+0x58] == 1
	w.bits(0, 4)           // FUN_1406d310c(10) = 4
	w.bit(0)               // victime absente
	fin := w.n
	w.bits(0x7f, 7)
	br := LecteurSur(w.buf)
	if !chargeDegatsApres(br) || br.BitPos() != fin {
		t.Fatalf("fin %d, attendu %d", br.BitPos(), fin)
	}
}

// TestLeTirCourtSArreteApresSaDirection : `FUN_14080c1f8` sur la branche courte (premier bit a 1)
// lit la tete puis R(0xa), et s arrete.
func TestLeTirCourtSArreteApresSaDirection(t *testing.T) {
	var w bitWriter
	w.bit(1)           // court
	w.bit(0)           // bloc
	w.bits(0, 7+1)     // FUN_141fcf670 : R(7), R(1)
	w.bit(1)           // FUN_1407f2058 : sentinelle
	w.bit(1)           // FUN_1406d00ec : sentinelle
	w.bit(0)           // FUN_14080d69c : absent
	w.bits(0, 32+2)    // variante, [0x1d], [2]
	w.bits(0x3ff, 0xa) // FUN_14076dc04 : R9D = 0xa (14080c465)
	fin := w.n
	w.bits(0, 16)
	br := LecteurSur(w.buf)
	if !chargeTirArme(br) || br.BitPos() != fin {
		t.Fatalf("fin %d, attendu %d", br.BitPos(), fin)
	}
}

// TestLeBitDeConfigurationAZeroArreteLaVueA : `FUN_142987460` lit `DAT_144706104` avant la vue A ;
// a 0, `FUN_1406d3140` lit toutes ses references a la plage `DAT_144706100`, que les charges portees
// ne lisent pas : la lecture s arrete apres la tete, qu elle lit a l identique. Le meme paquet, bit
// a 1, se lit jusqu a son terminateur. MUTATION : le bit saute au lieu d etre lu -> le paquet a 0
// lu, ROUGE.
func TestLeBitDeConfigurationAZeroArreteLaVueA(t *testing.T) {
	for _, configuration := range []uint64{0, 1} {
		var w bitWriter
		w.bits(configuration, 1)
		w.bit(1)
		w.bits(21, 7) // unit_zoom
		w.bit(1)      // garde de la reference 0 : domaine 4, R(9) puis R(2) a la table par categorie
		w.bits(0, 9+2)
		w.bits(0, 2) // gardes des references 1 et 2
		w.bits(0, 2) // FUN_14080cb98
		w.bit(0)     // terminateur
		fin := w.n
		w.bits(0, 16)
		a := lireSous(w.buf, grammaireRecente())
		if configuration == 0 && (a.Porte || a.Fin != 9 || !slices.Equal(a.Genres, []int{21})) {
			t.Fatalf("bit de configuration a 0 : %+v, attendu la tete seule", a)
		}
		if configuration == 1 && (!a.Porte || a.Fin != fin) {
			t.Fatalf("fin %d (%v), attendu %d", a.Fin, a.Porte, fin)
		}
	}
}
