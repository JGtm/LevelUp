package grammar

// lecteur_jeu_darmes_guard_test.go — LE GARDE-RAIL DU LECTEUR DE JEU D ARMES (regle des deux
// copies, CLAUDE.md n. 6). La sequence de `FUN_1406d01fc` — R(3) puis deux `FUN_1406d00ec`
// (R(1) ; si 0, R(2)) — n existe qu une fois, dans [lireJeuDArmes] (`unit_weaponstate.go`), que
// lisent le bipede (`i42`), le vehicule (`ti=40 i38`) et la queue de l arme tenue
// ([consumeWeaponStateTail]). Ce test interdit qu elle revienne en ligne ailleurs dans la
// production du paquet, sous ses deux formes : `ReadBits(3)` suivi, dans le meme bloc, de deux
// appels identiques (deux [consumeID2], ou deux appels d un doublon de [consumeID2] sous un autre
// nom, comme le `consumeOpt2` qui vivait dans `components_object.go`), ou `ReadBits(3)` suivi des
// deux `FUN_1406d00ec` deplies (`ReadBit`, `ReadBits(2)`, deux fois). La suite des appels est lue
// dans l arbre syntaxique (`sequence_appels_test.go`) : la mise en page n y compte pas.

import "testing"

// lectureBrute : les appels qui ne sont pas un lecteur de `FUN_1406d00ec` (lectures directes du
// [Lecteur], appel sans nom) : `R(3)` suivi de deux `R(32)` n est pas la sequence.
var lectureBrute = map[string]bool{"ReadBits": true, "ReadBit": true, "Skip": true, "?": true}

// copiesDeJeuDArmes compte, dans la suite d appels `seq` d une fonction, les lectures en ligne de
// `FUN_1406d01fc`.
func copiesDeJeuDArmes(seq []appelLu) int {
	n := 0
	for i, a := range seq {
		if !estAppel(a, "ReadBits", "3") {
			continue
		}
		suite := seq[i+1:]
		if len(suite) >= 2 && suite[0] == suite[1] && suite[0].bloc == a.bloc && !lectureBrute[suite[0].nom] {
			n++
			continue
		}
		if len(suite) >= 4 && estAppel(suite[0], "ReadBit") && estAppel(suite[1], "ReadBits", "2") &&
			estAppel(suite[2], "ReadBit") && estAppel(suite[3], "ReadBits", "2") {
			n++
		}
	}
	return n
}

// TestLecteurDeJeuDArmesUnique interdit les copies hors du fichier hote (`_test.go` exclus) ; le
// fichier hote porte la sequence exactement une fois, sans quoi le garde-rail garde un fantome.
func TestLecteurDeJeuDArmesUnique(t *testing.T) {
	hote := 0
	for _, f := range appelsDeLaProduction(t) {
		copies := 0
		for _, seq := range f.fonctions {
			copies += copiesDeJeuDArmes(seq)
		}
		if f.nom == "unit_weaponstate.go" {
			hote = copies
			continue
		}
		if copies > 0 {
			t.Errorf("%s : %d lecture(s) en ligne de FUN_1406d01fc — appeler lireJeuDArmes "+
				"(unit_weaponstate.go)", f.nom, copies)
		}
	}
	if hote != 1 {
		t.Errorf("unit_weaponstate.go porte %d fois la sequence de FUN_1406d01fc, 1 attendue — "+
			"deplacer le garde-rail avec le lecteur", hote)
	}
}

// TestGardeRailJeuDArmesVoitLesDeuxFormes : la copie se voit appelee ou depliee, sur une ligne
// comme sur plusieurs ; un R(3) suivi d autre chose n est pas la sequence.
func TestGardeRailJeuDArmesVoitLesDeuxFormes(t *testing.T) {
	for _, c := range []struct {
		corps  string
		copies int
	}{
		{"br.ReadBits(3); consumeID2(br); consumeID2(br)", 1},
		{"_ = uint32(br.ReadBits(3)) // emplacement\n\tconsumeID2(br)\n\tconsumeID2(br)", 1},
		{"br.ReadBits(3)\n\tif !br.ReadBit() {\n\t\tbr.ReadBits(2)\n\t}\n\tif !br.ReadBit() { br.ReadBits(2) }", 1},
		{"br.ReadBits(3)\n\tconsumeOpt2(br) // doublon de consumeID2\n\tconsumeOpt2(br)", 1},
		{"br.ReadBits(3); consumeID2(br); consumeOpt5(br)", 0},
		{"br.ReadBits(3); br.ReadBits(32); br.ReadBits(32)", 0},
	} {
		src := "package p\n\nfunc f(br *Lecteur) {\n\t" + c.corps + "\n}\n"
		n := 0
		for _, seq := range appelsDuSource(t, "vecteur.go", []byte(src)) {
			n += copiesDeJeuDArmes(seq)
		}
		if n != c.copies {
			t.Errorf("%q : %d copie(s), attendu %d", c.corps, n, c.copies)
		}
	}
}
