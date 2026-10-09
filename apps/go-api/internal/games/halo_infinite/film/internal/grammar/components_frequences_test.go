package grammar

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// Vecteurs des deux archetypes « frequence », ecrits d apres les ECRIVAINS du jeu :
// FUN_142eda938 (`low-frequency`, `ti=3 i0`), FUN_142eda744 (`high-frequency`, `ti=3 i1`),
// FUN_142eda680 (`high-frequency`, `ti=4 i0`). Chaque vecteur est suivi d un temoin de 13 bits :
// le lecteur doit s arreter exactement au bout du composant, et le temoin se relire intact.

const (
	temoinFrequence      = 0x1b5d // 13 bits
	largeurTemoinFrequen = 13
)

// ecrirePositionSansIndex ecrit une position par la porte « index absent » (`R(1) = 1`), sur les
// largeurs par defaut du niveau 0x10 : la forme que lit `lireE494` sous le profil par defaut.
func ecrirePositionSansIndex(w *bitWriter, axes [3]uint64) {
	w.bit(1)
	larg := profile.LargeursAxeParDefautDuBuild(niveauPosition)
	for k, v := range axes {
		w.bits(v, int(larg[k]))
	}
}

// ecrireAvantHaut ecrit l orientation de FUN_141f86118 (mode 0) : porte, direction de 19 bits si
// la porte est nulle, roulis de 8 bits.
func ecrireAvantHaut(w *bitWriter, direction uint64, avecDirection bool, roulis uint64) {
	if avecDirection {
		w.bit(0)
		w.bits(direction, 19)
	} else {
		w.bit(1)
	}
	w.bits(roulis, 8)
}

// entreeBasseFrequence est une entree de la liste de FUN_142eda938 : drapeaux `+0x27` (3 bits),
// mot `+0x24` (16 bits), code `+0x26` (5 bits) ; `direction` choisit la branche de la porte de
// l orientation (ecrite si le bit 2 des drapeaux est pose).
type entreeBasseFrequence struct {
	drapeaux  uint64
	mot       uint64
	code      uint64
	direction bool
}

// ecrireBasseFrequence suit FUN_142eda938 champ par champ ; `teteAvecDirection` choisit la branche
// de la porte de l orientation de tete.
func ecrireBasseFrequence(w *bitWriter, teteAvecDirection bool, entrees []entreeBasseFrequence) {
	ecrirePositionSansIndex(w, [3]uint64{5, 9, 3})
	ecrireAvantHaut(w, 0x4a1b2, teteAvecDirection, 0x7f)
	w.bits(0xbeef, 16) // +0x52c
	w.bits(0xa5, 8)    // +0x52e
	w.bits(2, 2)       // +0x52f
	w.bits(uint64(len(entrees)), largeurEntreesBasseFrequence)
	for _, e := range entrees {
		w.bits(e.drapeaux, 3) // FUN_142b67fe8
		if e.drapeaux&1 != 0 {
			ecrirePositionSansIndex(w, [3]uint64{1, 2, 3})
		}
		if e.drapeaux&2 != 0 {
			ecrireAvantHaut(w, 0x2c3d4, e.direction, 0x11)
		}
		w.bits(e.mot, 16)
		w.bits(e.code, 5) // FUN_142af2af0
	}
}

// lireAuTemoin rend le lecteur positionne sur `buf` et le nombre de bits du composant ecrit.
func lireAuTemoin(w *bitWriter) (*Lecteur, int) {
	fin := w.n
	w.bits(temoinFrequence, largeurTemoinFrequen)
	return lecteurDInstrument(append(w.buf, make([]byte, 16)...)), fin
}

func verifierTemoin(t *testing.T, br *Lecteur, fin int, quoi string) {
	t.Helper()
	if br.BitPos() != fin {
		t.Fatalf("%s : le lecteur s arrete au bit %d, l ecrivain a ecrit %d bits", quoi, br.BitPos(), fin)
	}
	if v := br.ReadBits(largeurTemoinFrequen); v != temoinFrequence {
		t.Fatalf("%s : temoin relu %#x, %#x attendu", quoi, v, temoinFrequence)
	}
}

// casBasseFrequence : un composant `low-frequency` a ecrire, la branche de la porte d orientation
// de tete et la liste d entrees.
type casBasseFrequence struct {
	teteAvecDirection bool
	entrees           []entreeBasseFrequence
}

// TestBasseFrequenceSuitSonEcrivain : `ti=3 i0` lit exactement ce que FUN_142eda938 ecrit, liste
// vide, liste de drapeaux varies (dont le bit 4, que ni l ecrivain ni le lecteur ne consultent)
// et liste de 63 entrees (le maximum du compte de 6 bits). Les deux branches de la porte de
// FUN_140c5fa84 (direction de 19 bits ecrite si la porte est nulle) sont jouees en tete comme en
// entree.
func TestBasseFrequenceSuitSonEcrivain(t *testing.T) {
	longue := make([]entreeBasseFrequence, 63)
	for k := range longue {
		longue[k] = entreeBasseFrequence{
			drapeaux: uint64(k % 8), mot: uint64(k * 977), code: uint64(k % 32), direction: (k/8)%2 == 1,
		}
	}
	varies := []entreeBasseFrequence{
		{drapeaux: 0, mot: 0x1234, code: 7},
		{drapeaux: 1, mot: 0xffff, code: 31},
		{drapeaux: 2, mot: 0, code: 0},
		{drapeaux: 2, mot: 0x0420, code: 3, direction: true},
		{drapeaux: 3, mot: 0x8001, code: 16, direction: true},
		{drapeaux: 3, mot: 0x7ffe, code: 15},
		{drapeaux: 4, mot: 0x0f0f, code: 9},
	}
	cas := map[string]casBasseFrequence{
		"liste vide, tete avec direction":        {teteAvecDirection: true},
		"liste vide, tete sans direction":        {teteAvecDirection: false},
		"drapeaux 0 a 4, tete avec direction":    {teteAvecDirection: true, entrees: varies},
		"drapeaux 0 a 4, tete sans direction":    {teteAvecDirection: false, entrees: varies},
		"entree seule avec direction":            {teteAvecDirection: true, entrees: varies[3:4]},
		"63 entrees, directions alternees par 8": {teteAvecDirection: true, entrees: longue},
	}
	for nom, c := range cas {
		w := &bitWriter{}
		ecrireBasseFrequence(w, c.teteAvecDirection, c.entrees)
		br, fin := lireAuTemoin(w)
		_, _, porte := consumeByName(br, compLowFrequency, archetypeFrequences, 0)
		if !porte {
			t.Fatalf("%s : low-frequency declare non porte", nom)
		}
		verifierTemoin(t, br, fin, nom)
	}
}

// TestBasseFrequenceNonPorteeDansUnEtatComplet : sous la boucle d etat complet (`FUN_142e2c690`,
// portee `DAT_144e61ea0` posee), `ti=3 i0` n est pas lu aux largeurs du delta ; le composant rend
// « non porte » sans consommer un bit, et le meme vecteur se lit en entier hors etat complet.
func TestBasseFrequenceNonPorteeDansUnEtatComplet(t *testing.T) {
	for _, etatComplet := range []bool{false, true} {
		w := &bitWriter{}
		ecrireBasseFrequence(w, true, []entreeBasseFrequence{{drapeaux: 3, mot: 0x8001, code: 16}})
		br, fin := lireAuTemoin(w)
		enEtatComplet(br, etatComplet)
		_, _, porte := consumeByName(br, compLowFrequency, archetypeFrequences, 0)
		if etatComplet {
			if porte || br.BitPos() != 0 {
				t.Fatalf("etat complet : porte=%v, %d bits lus ; attendu non porte, 0 bit", porte, br.BitPos())
			}
			continue
		}
		if !porte {
			t.Fatal("record a masque : low-frequency declare non porte")
		}
		verifierTemoin(t, br, fin, "record a masque")
	}
}

// TestHauteFrequenceSeLitParLaTableDeLArchetype : le meme nom, deux tables. `ti=3 i1` lit les
// 26 bits de FUN_142eda744 sans sonde ; `ti=4 i0` lit le R(8) de FUN_142eda680 et le publie a la
// sonde ; un archetype sans table connue ne lit rien et rend « non porte ».
func TestHauteFrequenceSeLitParLaTableDeLArchetype(t *testing.T) {
	clearAllHooks(t)
	var sondes []uint64
	observateur.ProbeHook = func(ti uint32, comp ProbeComponent, v []uint64) {
		if ti != archetypeHauteFrequence || comp != ProbeHighFrequency {
			t.Errorf("sonde inattendue ti=%d %v", ti, comp)
		}
		sondes = append(sondes, v...)
	}

	w := &bitWriter{}
	w.bits(0xbeef, 16)
	w.bits(0xa5, 8)
	w.bits(1, 2)
	br, fin := lireAuTemoin(w)
	if _, _, porte := consumeByName(br, compHighFrequency, archetypeFrequences, 0); !porte {
		t.Fatal("ti=3 i1 : non porte")
	}
	verifierTemoin(t, br, fin, "ti=3 i1")
	if len(sondes) != 0 {
		t.Fatalf("ti=3 i1 publie a la sonde de ti=4 : %v", sondes)
	}

	w = &bitWriter{}
	w.bits(0x5a, 8)
	br, fin = lireAuTemoin(w)
	if _, _, porte := consumeByName(br, compHighFrequency, archetypeHauteFrequence, 0); !porte {
		t.Fatal("ti=4 i0 : non porte")
	}
	verifierTemoin(t, br, fin, "ti=4 i0")
	if len(sondes) != 1 || sondes[0] != 0x5a {
		t.Fatalf("ti=4 i0 : sonde %v, [0x5a] attendu", sondes)
	}

	for _, ti := range []uint32{0, 5, 35, 47, 63} {
		br = lecteurDInstrument(make([]byte, 8))
		if _, _, porte := consumeByName(br, compHighFrequency, ti, 0); porte || br.BitPos() != 0 {
			t.Errorf("ti=%d : high-frequency lu (%d bits, porte=%v) sans table connue", ti, br.BitPos(), porte)
		}
	}
}

// archetypeDuHook rend l archetype sous lequel un composant de `hookedNames` se lit : celui qui en
// enregistre la table publiee pour `high-frequency` (`ti=4`), le bipede pour les autres, que le
// dispatch lit sous tout archetype.
func archetypeDuHook(nom string) uint32 {
	if nom == compHighFrequency {
		return archetypeHauteFrequence
	}
	return BipedTypeIndex
}
