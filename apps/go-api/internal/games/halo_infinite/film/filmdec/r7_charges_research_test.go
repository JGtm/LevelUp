package filmdec

// r7_charges_research_test.go — lot R7 : les LARGEURS DE CHARGE par type, sourcees de
// l'executable (lecteur `vtable+0x68` de chaque descripteur, decompile). C'est LE verrou que
// R6 n'avait pas fait sauter : sans la largeur de charge on ne sait pas ou commence
// l'evenement suivant, donc on ne lit que la TETE de chaque paquet.
//
// METHODE DE DERIVATION (identique pour tous). Le flux de bits porte son compteur de bits
// consommes en `*(int *)(flux + 0x2c)` : toute primitive qui fait `+0x2c += N` consomme N
// bits sur ce chemin ; une fonction sans `+0x2c` et sans appel de lecteur consomme 0 bit.
// Les primitives (verifiees sur pieces ce lot, sauf mention) :
//
//	0x1406cf008 R(1) · 0x1424e1d48 R(4) · 0x142ed0674 R(4)-1 · 0x142ed0abc R(4)+1
//	0x1407f0094 R(16) · 0x14080d6f0 / 0x142ef49b8 / 0x14080dec4 R(32) · 0x14080bd28 R(15)
//	0x14080d69c [R(1) g ; si g : R(32)]
//	0x1407f2058 [R(1) g ; si g==0 : R(5)]   POLARITE INVERSEE (verifie : +0x2c += 5 sous g==0)
//	0x1406d00ec [R(1) g ; si g==0 : R(2)]   POLARITE INVERSEE
//	0x14080cb98 R(2)-1 · 0x14080cb50 [R(1) g ; si g : R(n)] · 0x1406d84b4 R(n) (n = 5e arg)
//	0x1406d3140 reference var-int · 0x1408d8220 lecteur VIDE, 0 bit
//	0x14076dc04 R(n) (n = R9D au site d'appel) · 0x1406d310c ceil_log2, 0 bit (calcul)
//	0x141102ed0(id) / 0x1404f25f4 / 0x14076f91c drapeaux de build ou de session (0 bit)
//
// CORRECTION A LA GRAMMAIRE DE 2026-08-30 : `0x1406d676c` n'est PAS un `R(64)` fixe. C'est un
// `R(n)` GENERIQUE dont n, en bits, est le 4e argument (`for (; 0x3f < n; n -= 0x40)
// +0x2c += 0x40 ; puis +0x2c += n`). C'est ce qui rend le type 15 auto-delimite.
//
// STATUT PAR TYPE : `r7SkipCharge` ne traite QUE les types FERMES (on sait avancer a coup
// sur). Tout type non traite arrete la marche et est compte comme opaque — c'est le resultat
// honnete, jamais une largeur devinee.
//
// LECTURE SEULE, skip par defaut.

import "math"

// r7BornesDefaut : demi-etendue des bornes par defaut du moteur (DAT_143b8c6b8, +/-20000),
// utilisee quand le vecteur quantifie ne designe pas de region.
const r7BornesDefaut = 40000.0

// r7Ctx porte ce dont une charge peut avoir besoin en plus du flux : les ETENDUES de la
// carte du film (bornes vraies de quantification) et la largeur runtime de l'index de region.
//
// Pour MARCHER on ne consomme que des bits, mais la largeur d'un axe depend de l'etendue ET
// de la granularite k du type : 117 lit a k=16, le type 5 a k=15, les types 6 et 106 a k=12.
// On garde donc les etendues et on applique la formule de l'exe, jamais une constante.
type r7Ctx struct {
	etendues   [3]float64 // max[i] - min[i] de la carte, en metres
	regionBits uint       // largeur de l'index de region (DAT_144632be0)
	hasMap     bool
}

// r7VarPosBrute : le predicat `FUN_14076f91c()` (= `DAT_144e61ea0 != 0 || DAT_145121140 == 1`)
// bascule TOUTES les positions de la quantification vers 96 bits bruts. Les deux globales
// valent zero dans l'image mais sont ecrites a l'execution : la valeur est CALIBREE SUR
// PIECES par TestR7Variantes, jamais devinee.
var r7VarPosBrute = false

// r7BitsAxe rend le nombre de bits d'un axe pour une etendue et une granularite k.
// Source : FUN_140be9b88 + FUN_140be9c78 (granularite de base DAT_143cd9758 = 1/120 m,
// seuil DAT_143cd837c = 1e-4, plafond DAT_143cd975c = 2^22, plafond de largeur 26).
//
//	e(k)  = (1/120) * 2^(16-k)
//	n     = min(2^22, ceil(etendue / (2*e)))
//	bits  = min(26, ceil(log2(n)))
//
// CONTROLE : cette formule reproduit les axisWidths du catalogue de production
// `map_quant_bounds.json` a k=16 (verifie a R6 : aquarius 77,8/46,2/18,1 m -> 13/12/11) et
// rend, sur les bornes par defaut +/-20000, exactement 22 bits a k=16, 21 a k=15, 18 a k=12
// — les trois valeurs relevees independamment au desassemblage des sites d'appel.
func r7BitsAxe(etendue float64, k int) uint {
	e := (1.0 / 120.0) * math.Pow(2, float64(16-k))
	if e < 1e-4 {
		return 26
	}
	n := math.Ceil(etendue / (2 * e))
	if n < 1 {
		n = 1
	}
	if lim := math.Pow(2, 22); n > lim {
		n = lim
	}
	b := int(math.Ceil(math.Log2(n)))
	if b > 26 {
		b = 26
	}
	if b < 0 {
		b = 0
	}
	return uint(b)
}

// r7VecteurQuantifie consomme un vecteur quantifie (primitive 0x14076e524) a la granularite k.
// Porte INVERSEE : bit 0 -> index de region R(wr) puis 3 axes aux bornes de LA region ;
// bit 1 -> bornes par defaut du moteur. Sous r7VarPosBrute, la position est un R(96) brut et
// aucune de ces largeurs ne s'applique.
func r7VecteurQuantifie(br *BitReader, ctx r7Ctx, k int) bool {
	if r7VarPosBrute {
		br.Skip(96)
		return true
	}
	if !br.ReadBit() {
		if !ctx.hasMap {
			return false // sans la carte du film, la largeur est inconnue : on ne devine pas
		}
		br.Skip(int(ctx.regionBits))
		for i := 0; i < 3; i++ {
			br.Skip(int(r7BitsAxe(ctx.etendues[i], k)))
		}
		return true
	}
	br.Skip(3 * int(r7BitsAxe(r7BornesDefaut, k)))
	return true
}

// r7SkipCharge consomme la charge du type. Rend false si le type n'a pas de grammaire fermee
// (marche impossible au-dela) ou si la charge exige une carte absente.
func r7SkipCharge(br *BitReader, typ int, ctx r7Ctx) bool {
	switch typ {
	// --- lecteur VIDE 0x1408d8220 : 0 bit (annexe A) ---
	case 3, 4, 23, 24, 25, 26, 33, 49, 54, 57, 59, 92, 103:
		return true

	// --- 21 unit_zoom : R(2) puis valeur-1 (lecteur 0x141168b28, PROUVE lot E) ---
	case 21:
		br.Skip(2)
		return true

	// --- 118 repair_complete : [R(1) g ; si g : R(32)] (lecteur 0x142ef9074 = 14080d69c seul) ---
	case 118:
		if br.ReadBit() {
			br.Skip(32)
		}
		return true

	// --- 108 NavpointRequest : R(32) + R(32) (PROUVE lot E4) ---
	case 108:
		br.Skip(64)
		return true

	// --- 0 damage_aftermath : grammaire bit-exacte deja en production ---
	case 0:
		lot1DecodeDamageAftermath(br)
		return true

	// --- 100 PowerUpApplied : [R(1);si 1:R(32)] + R(32) + [R(1);si 1:R(32)] (PROUVE lot E) ---
	case 100:
		if br.ReadBit() {
			br.Skip(32)
		}
		br.Skip(32)
		if br.ReadBit() {
			br.Skip(32)
		}
		return true

	// --- 117 : [R(1) g ; si g : R(32)] + 2 vecteurs quantifies k=16 (valide 18/18, R6) ---
	case 117:
		if br.ReadBit() {
			br.Skip(32)
		}
		for p := 0; p < 2; p++ {
			if !r7VecteurQuantifie(br, ctx, 16) {
				return false
			}
		}
		return true
	}
	return r7SkipChargeLot2(br, typ, ctx)
}
