package grammar

// i59_etiquette_loi_test.go — LA LOI DE L ETIQUETTE DU CORPS `i59` ET LE PERIMETRE DU PORT
// (lot 5.3.3-c, 2026-09-21).
//
// # CE QUE L ECRIVAIN DIT, LU A L OCTET
//
// `FUN_142f21c0c` — le lecteur d etiquette du corps `tag==3` d `i59` (`FUN_142f25e90`) — lit
// EXACTEMENT TROIS BITS et range `brut + 1` :
//
//	*(param_1 + 0x2c) += 3            // le compteur de bits avance de 3, et de 3 seulement
//	*param_3 = (octet de tete >> 5) + 1
//
// `FUN_142f25e90` dispatche ensuite sur cette valeur RANGEE, de 1 a 6, et **rend la main sans
// lire un bit de plus au-dela de 6**. Trois consequences, et chacune corrige une lecture du
// depot :
//
//	(1) L ETIQUETTE VAUT `brut + 1`. Le champ `AbilityNonPredictedState.Inner` du port porte le
//	    BRUT ; ses deux constantes (`anchorInnerLight` = 1, `anchorInnerHeavy` = 2) designent
//	    donc les etiquettes 2 et 3 de l ecrivain, pas 1 et 2.
//	(2) LA BRANCHE `etiquette == 0` DE L ECRIVAIN EST INATTEIGNABLE : `brut + 1 >= 1`. Le
//	    `if (*pcVar1 == '\0')` de tete est du code mort du point de vue du flux.
//	(3) LES BRUTS 6 ET 7 (etiquettes 7 et 8) NE PORTENT AUCUNE CHARGE PROPRE : l ecrivain sort
//	    du `switch` par son `return`. Le port les compte aujourd hui en desync.
//
// # CE QUE CE TEST FIGE, ET POURQUOI IL FIGE AUSSI LE MANQUE
//
// Il epingle, pour les HUIT valeurs brutes, la consommation de bits du port ET son verdict. Les
// six valeurs que le port ne modelise pas y sont figees comme NON PORTEES : un lot qui en
// portera une devra mettre ce tableau a jour DELIBEREMENT, et la mesure du cout (5.3.3-c :
// 38 records desynchronises sur 31 530, soit 0,12 %) restera lisible a cote.
//
// LA MESURE QUI JUSTIFIE DE NE PAS PORTER PLUS LOIN, ICI : les branches restantes exigent la
// largeur bit-exacte de `FUN_142f26e40`, de `FUN_1408f0ac4` (categories 0 et 5) et de
// `FUN_1407f08bc`, qu aucune lecture n a encore etablies ; et le port actuel lit sa porte
// `FUN_1407f08bc` AVANT le `switch`, la ou l ecrivain la lit DANS ses etiquettes 1 et 2. Un
// port partiel deplacerait donc le curseur sur les corps qui aboutissent aujourd hui — ceux
// qui publient `grappleLines[]` au document. Report consigne.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// i59LoiBits est la largeur de l etiquette chez l ecrivain : trois bits, pas un de plus.
const i59LoiBits = 3

// i59LoiEtiquette applique la loi de l ecrivain a une valeur brute.
func i59LoiEtiquette(brut uint64) uint64 { return brut + 1 }

// TestI59LoiDeLEtiquetteEstBrutPlusUn epingle la loi elle-meme : elle est le seul fait de ce
// fichier dont depend la lecture des six autres branches.
func TestI59LoiDeLEtiquetteEstBrutPlusUn(t *testing.T) {
	for brut := uint64(0); brut < 8; brut++ {
		if got, veut := i59LoiEtiquette(brut), brut+1; got != veut {
			t.Fatalf("brut %d : etiquette %d, attendu %d", brut, got, veut)
		}
	}
	// L ETIQUETTE 0 EST INATTEIGNABLE, et c est le point (2) de l en-tete.
	for brut := uint64(0); brut < 8; brut++ {
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

// TestI59PerimetreDuPortParEtiquette fige la consommation de bits du port pour les huit valeurs
// brutes, et NOMME les six qu il ne modelise pas.
func TestI59PerimetreDuPortParEtiquette(t *testing.T) {
	// LE PROFIL EST EXPLICITE : la position absolue du corps se lit aux largeurs d axe de la
	// CARTE, donc un profil implicite rendrait ce test dependant d un defaut de catalogue.
	prof := ProfilDeBalayageParDefaut()
	prof.PoserLargeursObjetDuMondeDepuisDecoupage(i59LoiDecoupage())
	axes := prof.LargeursObjetDuMonde().AxisW
	somme := int(axes[0] + axes[1] + axes[2])

	// enTete : etiquette(3) + Zero3(3) + trois axes + Mid7(7) + la porte(1), gate a 0.
	enTete := i59LoiBits + anchorZeroBits + somme + anchorMidBits + 1

	cas := []struct {
		brut  uint64
		porte bool
		bits  int
		dit   string
	}{
		{0, false, enTete, "etiquette 1 de l ecrivain : NON PORTEE (le port sort sur `default`)"},
		{1, true, enTete, "etiquette 2 : portee — le corps LEGER, aucune charge de plus"},
		{2, true, enTete + 3*1 + anchorPackedBits + anchorTailBits,
			"etiquette 3 : portee — trois vecteurs (portes a 1, donc 1 bit chacun), R(24), R(9)"},
		{3, false, enTete, "etiquette 4 : NON PORTEE"},
		{4, false, enTete, "etiquette 5 : NON PORTEE"},
		{5, false, enTete, "etiquette 6 : NON PORTEE"},
		{6, false, enTete, "etiquette 7 : NON PORTEE — l ecrivain n y lit AUCUNE charge propre"},
		{7, false, enTete, "etiquette 8 : NON PORTEE — idem"},
	}
	for _, c := range cas {
		br := LecteurSur(i59LoiCorps(c.brut, somme))
		br.PoserProfil(prof)
		var st AbilityNonPredictedState
		st.Inner = -1
		got := consumeAbilityAnchorBody(br, &st)
		if got != c.porte {
			t.Errorf("brut %d (%s) : porte = %v, attendu %v", c.brut, c.dit, got, c.porte)
		}
		if br.BitPos() != c.bits {
			t.Errorf("brut %d (%s) : %d bits consommes, attendu %d",
				c.brut, c.dit, br.BitPos(), c.bits)
		}
		if st.Inner != int(c.brut) {
			t.Errorf("brut %d : Inner = %d — le champ porte le BRUT, pas l etiquette",
				c.brut, st.Inner)
		}
	}
}

// i59LoiDecoupage rend un decoupage d axes VALIDE et explicite pour ce test.
func i59LoiDecoupage() profile.I0Layout {
	return profile.I0Layout{GateBits: 5, AxisW: [3]uint{13, 13, 14}}
}

// i59LoiCorps fabrique un corps synthetique : l etiquette, `Zero3 = 0`, les trois axes a zero,
// `Mid7 = 0`, la porte a 0, puis de quoi nourrir les trois vecteurs (portes a 1) et les deux
// champs de queue de l etiquette 3.
func i59LoiCorps(brut uint64, sommeAxes int) []byte {
	var w bitWriter
	w.bits(brut, i59LoiBits)
	w.bits(0, anchorZeroBits)
	w.bits(0, sommeAxes)
	w.bits(0, anchorMidBits)
	w.bit(0) // porte FUN_1407f08bc fermee
	for i := 0; i < 3; i++ {
		w.bit(1) // vecteur constant : aucune charge
	}
	w.bits(0, anchorPackedBits)
	w.bits(0, anchorTailBits)
	w.bits(0, 8) // marge : le lecteur ne doit jamais manquer d octets
	return w.buf
}
