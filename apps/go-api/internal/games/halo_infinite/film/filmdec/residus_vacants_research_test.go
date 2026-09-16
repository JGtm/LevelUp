package filmdec

// residus_vacants_research_test.go — PHASE 5b, RESIDU 1 : LES DEUX ECARTS ABERRANTS DE LA TABLE
// DES SLOTS, ET CE QU'ILS CONTIENNENT.
//
// LE RESIDU. La phase 4 (`NOTE_PROFIL_PAR_BUILD`, C.5 n°1) a ferme 1 890 ecarts sur 1 892 apres
// avoir corrige le critere `slot+0x08`. Deux ecarts restaient « non tranches » : `b1bcbe24`
// (38 852 bits) et `1c5c10cc` (27 382 bits). Les branches ecrites d'avance, sans en choisir une
// (methode, erreur E) : (a) un enregistrement VACANT que le balayage ne voit pas, (b) un
// enregistrement de joueur que ses valeurs de champ font rejeter — bot, joueur parti avant le
// debut, champ hors plage.
//
// LA BRANCHE (a) EST TESTABLE SANS AUCUNE MESURE PREALABLE, parce que la grammaire PREDIT la
// longueur d'un enregistrement entierement a zero : c'est `rsVide`, une somme de largeurs lues
// dans l'executable. Si l'ecart aberrant vaut la longueur predite plus un nombre ENTIER de fois
// cette valeur, et si le predicat grammatical `rsVacant` passe a chacune des positions ainsi
// designees, la branche (a) est etablie et la (b) tombe.
//
// CE QUE CE RESIDU FERME EN PLUS. La longueur exacte d'un enregistrement de slot VIDE etait la
// question ouverte n°3 de la phase 1 (« 15 700 a 17 900 bits selon le film, non determine »).
//
// Gardes CHUNK00_FILMS. Lecture seule, aucun code de production touche.

import (
	"path/filepath"
	"testing"
)

// rsVide rend la longueur, en bits, d'un enregistrement de slot ENTIEREMENT A ZERO sur un build
// dont la transposition vaut `delta`. Elle est CALCULEE terme a terme depuis la grammaire, pas
// ajustee sur une mesure :
//
//	  85  en-tete du slot (1+1+1+32+2+48)        FUN_1407ecb08
//	+ 64  le XUID, nul                           FUN_1406d6498
//	+ 11  prefixe du masque (rang du bit haut)    FUN_1424ccf94
//	+  1  le masque reduit a un bit               masque vide -> rang 0 -> 1 bit ecrit
//	+ 12  N, nul                                  FUN_1411b1a24
//	+  8  M, nul                                  FUN_1411b198c
//	+ 832 le bloc de 104 octets                   sub+0xc48
//	+ 16  la chaine reduite a son NUL             FUN_1407ece18 s'arrete APRES l'unite nulle
//	+ 128 le bloc de 16 octets                    sub+0xc38
//	+ 32  `desired-representation`                sub+0xcb0
//	+ 64  le champ de 64 bits                     sub+0xcb8
//	+ 46  les six champs courts (10+14+6+8+7+1)
//	= 1 299 bits, puis le bloc de personnalisation, le bloc de queue et le u32 final.
const rsVideHorsBlocs = 1299

func rsVide(delta int) int { return rsVideHorsBlocs + s3sPersoBits + delta + s3sBloc44 + 32 }

// rsVacant dit si un enregistrement de slot VACANT commence au bit `p`. Le predicat est
// GRAMMATICAL et sans seuil : chaque champ est lu a sa place et doit valoir zero. Un tel
// enregistrement est doublement INVISIBLE au balayage — son booleen de tete vaut 0 (le balayage
// exige `1/0/0`) et son XUID tombe hors de la plage Xbox — et c'est la raison mecanique pour
// laquelle il DOUBLE l'ecart entre ses deux voisins.
//
// Les champs laisses libres sont ceux dont la mesure montre qu'ils ne sont PAS nuls dans un
// enregistrement vacant : `sub+0xcb8` (64 bits) et le champ de 6 bits `sub+0xc35`.
func rsVacant(d []byte, p int) bool {
	if s3rBit(d, p, 3) != 0 || s3rBit(d, p+3, 32) != 0 || s3rBit(d, p+35, 2) != 0 {
		return false
	}
	if s3rBit(d, p+37, 48) != 0 || s3rBit(d, p+s3rEnteteBits, 64) != 0 {
		return false
	}
	q := p + s3rEnteteBits + 64
	if s3rBit(d, q, s3sPrefixeMasque+1) != 0 { // prefixe nul, donc un seul bit de masque, nul
		return false
	}
	q += s3sPrefixeMasque + 1
	if s3rBit(d, q, s3sLargeurN) != 0 || s3rBit(d, q+s3sLargeurN, s3sLargeurM) != 0 {
		return false
	}
	q += s3sLargeurN + s3sLargeurM
	for k := 0; k < s3sBloc104; k += 64 { // les 104 octets de sub+0xc48
		if s3rBit(d, q+k, 64) != 0 {
			return false
		}
	}
	q += s3sBloc104
	if s3rBit(d, q, 16) != 0 { // la chaine reduite a son terminateur
		return false
	}
	q += 16
	for k := 0; k < s3sBloc16+32; k += 32 { // sub+0xc38 puis `desired-representation`
		if s3rBit(d, q+k, 32) != 0 {
			return false
		}
	}
	return true
}

// rsEcarts rend les enregistrements a nom imprimable du balayage CORRIGE de la phase 4 (critere
// `slot+0x08` leve, filtre de parasite par le champ de nom) et l'ecart au suivant. Partir du
// balayage d'origine melangerait deux causes d'ecart double : le champ de 2 bits (phase 4) et

// TestResidusSlotAberrants execute R1 : les deux ecarts aberrants de la phase 4, sur pieces.
//
// Le test ne suppose rien de la cause. Il publie, autour de chaque ecart qui ne vaut pas la
// longueur predite plus la constante du film : les deux enregistrements qui l'encadrent champ
// par champ, ce que la decoupe du bloc de queue en dit, et si un enregistrement SUPPLEMENTAIRE
// se lit dans l'intervalle (ce qui signerait un slot que le balayage ne voit pas).
func TestResidusSlotAberrants(t *testing.T) {
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		build, _ := s3bBuild(d)
		delta, _, _ := rsDelta(d)
		es, ecarts := rsEcarts(d)
		t.Logf("=== %s === build %q, delta %+d, %d enregistrement(s)",
			filepath.Base(dir), build, delta, len(es))
		for i, e := range es {
			if ecarts[i] == 0 || ecarts[i]-s3sPredite(e) == delta {
				continue
			}
			rsAberrant(t, d, es, ecarts, i, delta)
		}
	}
}

// rsAberrant publie tout ce qu'on peut dire d'un ecart aberrant, sans rien conclure.
func rsAberrant(t *testing.T, d []byte, es []*s3sEnr, ecarts []int, i, delta int) {
	t.Helper()
	e := es[i]
	surplus := (ecarts[i] - s3sPredite(e)) - delta
	c := rsDecoupe(d, e, ecarts[i])
	t.Logf("  ABERRANT slot %2d bit %9d xuid %19d gt %-16q : ecart %6d, predite %6d, "+
		"surplus %+d au-dela du delta", i, e.debut, e.xuid, e.gamertag, ecarts[i],
		s3sPredite(e), surplus)
	t.Logf("    champs : masque %4d/%4d N %4d M %3d b2 %d f1 %d f6 %d f10 %d f14 %d "+
		"repr %d q64 %016x jeton %012x u32Tete %d", e.popMasque, e.compteMasque, e.n, e.m,
		e.b2, e.f1, e.f6, e.f10, e.f14, e.repr, e.q64, e.jeton48, e.u32Tete)
	t.Logf("    decoupe : AVANT %d, APRES %d (reference %d), plus longue suite nulle %d, "+
		"%d touche(s) du nom petit-boutiste", c.avant, c.apres, rsApresRef, c.zeros, c.touches)
	t.Logf("    queue lue petit-boutiste %q ; suivant : bit %9d xuid %19d gt %q",
		s3sTexte16(e.bloc44, true), es[i+1].debut, es[i+1].xuid, es[i+1].gamertag)
	rsIntervalle(t, d, e.debut+s3sPredite(e)+delta, es[i+1].debut, delta)
}

// rsIntervalle publie ce que contient l'intervalle laisse par un ecart aberrant. Les deux
// branches de l'alternative sont testees, aucune n'est choisie d'avance :
//
//	(a) UN SLOT VACANT QUE LE BALAYAGE NE VOIT PAS. Alors la taille de l'intervalle est un
//	    MULTIPLE ENTIER de `rsVide(delta)` — longueur CALCULEE, pas ajustee — et il contient
//	    exactement un bit non nul par enregistrement vacant (le booleen de tete `slot+0x00`).
//	(b) UN ENREGISTREMENT DE JOUEUR QUE LE BALAYAGE REJETTE. Alors l'intervalle porte un en-tete
//	    suivi d'un entier de la plage des XUID Xbox. Le test le cherche SANS aucun critere de
//	    valeur sur les autres champs.
func rsIntervalle(t *testing.T, d []byte, a, b, delta int) {
	t.Helper()
	trouves, nonNuls, premierNonNul := 0, 0, -1
	for p := a; p < b; p++ {
		if s3rBit(d, p, 1) != 0 {
			nonNuls++
			if premierNonNul < 0 {
				premierNonNul = p - a
			}
		}
	}
	for p := a; p+s3rEnteteBits+64 <= b; p++ {
		x := s3rBit(d, p+s3rEnteteBits, 64)
		if s3rBit(d, p, 3) != 4 || x < s3rXuidLo || x >= s3rXuidHi {
			continue
		}
		trouves++
		t.Logf("    intervalle (b) : en-tete a %9d, xuid %19d, u32 %d, 2 bits %d",
			p, x, s3rBit(d, p+3, 32), s3rBit(d, p+35, 2))
	}
	vide, reste := (b-a)/rsVide(delta), (b-a)%rsVide(delta)
	t.Logf("    intervalle [%d, %d) = %d bits : %d bit(s) non nul(s), le premier a +%d ; "+
		"(a) %d x rsVide(%+d)=%d, reste %d ; (b) %d en-tete + XUID de la plage Xbox",
		a, b, b-a, nonNuls, premierNonNul, vide, delta, rsVide(delta), reste, trouves)
	t.Logf("    a l'ouverture de l'intervalle : predicat VACANT (tous champs nuls, lus a leur "+
		"place) = %v ; champ de 6 bits `sub+0xc35` brut %d, champ de 64 bits `sub+0xcb8` %d",
		rsVacant(d, a), s3rBit(d, a+rsVideHorsBlocs-22, 6), s3rBit(d, a+rsVideHorsBlocs-110, 64))
}

// TestResidusSlotFermeture rejoue P1 de la phase 4 — « chaque ecart vaut la longueur PREDITE
// plus UNE SEULE constante par film » — avec la seule tolerance que R1 justifie : un ecart peut
// porter EN PLUS un nombre entier de slots VACANTS, et chacun doit alors satisfaire le predicat
// grammatical `rsVacant` A SA POSITION CALCULEE. C'est donc une tolerance verifiee, pas un jeu.
//
// La phase 4 fermait 1 890 ecarts sur 1 892. Le critere de ce test, ecrit d'avance : 1 892/1 892.
func TestResidusSlotFermeture(t *testing.T) {
	conf, aberr, vacTot, films, filmsBons := 0, 0, 0, 0, 0
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		delta, _, _ := rsDelta(d)
		c, a, v := rsFermetureFilm(d, delta)
		conf, aberr, vacTot = conf+c, aberr+a, vacTot+v
		films++
		if a == 0 {
			filmsBons++
		} else {
			t.Errorf("%s : %d ecart(s) ne ferment pas, meme en tolerant des slots vacants",
				filepath.Base(dir), a)
		}
		if v > 0 {
			t.Logf("%-10s delta %+6d : %d/%d ecart(s) fermes, %d slot(s) VACANT(s) verifie(s)",
				filepath.Base(dir), delta, c, c+a, v)
		}
	}
	t.Logf("=== BILAN R1 === %d/%d ecarts fermes sur %d films (%d films sans aucun ecart "+
		"aberrant) ; %d slot(s) vacant(s) trouve(s) et verifies par le predicat grammatical",
		conf, conf+aberr, films, filmsBons, vacTot)
}

// rsFermetureFilm mesure la fermeture d'un film : ecarts conformes, aberrants, slots vacants.
func rsFermetureFilm(d []byte, delta int) (conformes, aberrants, vacants int) {
	es, ecarts := rsEcarts(d)
	for i, e := range es {
		if ecarts[i] == 0 {
			continue
		}
		surplus := ecarts[i] - s3sPredite(e) - delta
		if surplus == 0 {
			conformes++
			continue
		}
		n, ok := rsVacantsDans(d, e.debut+s3sPredite(e)+delta, surplus, delta)
		if !ok {
			aberrants++
			continue
		}
		conformes++
		vacants += n
	}
	return conformes, aberrants, vacants
}

// rsVacantsDans dit si le surplus d'un ecart s'explique par un nombre entier de slots VACANTS
// consecutifs a partir de `a`, chacun verifie par le predicat grammatical a sa position.
func rsVacantsDans(d []byte, a, surplus, delta int) (n int, ok bool) {
	l := rsVide(delta)
	if surplus <= 0 || surplus%l != 0 {
		return 0, false
	}
	for k := 0; k < surplus/l; k++ {
		if !rsVacant(d, a+k*l) {
			return k, false
		}
	}
	return surplus / l, true
}
