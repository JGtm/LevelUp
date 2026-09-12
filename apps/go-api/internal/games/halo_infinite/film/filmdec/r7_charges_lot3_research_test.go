package filmdec

// r7_charges_lot3_research_test.go — suite : la famille PROJECTILES / DEGATS (types 5, 6, 7,
// 1, 2, 106), derivee de l'executable pendant ce lot.
//
// DEUX POINTS QUI COMPTENT :
//
//  1. La granularite du vecteur quantifie DIFFERE par type — c'est le 4e argument de
//     `FUN_14076e524` au site d'appel : k=15 pour le type 5 (`MOV R9D,0xf`), k=12 pour les
//     types 6 et 106 (`MOV R9D,0xc`), k=16 pour le 117. Sur les bornes par defaut cela donne
//     21, 18 et 22 bits par axe. Une largeur unique aurait desynchronise deux types sur trois.
//  2. Le lecteur du type 7 (`0x142f17474`) n'est PAS une fonction : c'est un thunk de 11
//     octets `MOV RDX,R9 ; MOV RCX,R8 ; JMP 0x142f1c6cc`. Le vrai lecteur est `FUN_142f1c6cc`,
//     et il est entierement FERME (vecteur local a 3 x R(12), table de bornes STATIQUE
//     `DAT_143b8c6f0` — aucune dependance a la carte ni au drapeau de session).

// r7T5DirA / r7T5DirB : les deux largeurs `FUN_14076dc04` du type 5. Elles ne se lisent pas
// au site d'appel et la calibration sur l'oracle (TestR7CalibreType5) est NON CONCLUANTE :
// le meilleur couple (24, 24) rend 1,413 records/paquet contre 1,328 au deuxieme — un ecart
// de 6 %, tres en dessous des 30 % exiges d'avance. Les valeurs ci-dessous sont donc le
// MEILLEUR CANDIDAT MESURE, pas une derivation ; le type 5 reste hors de la liste des
// largeurs validees (`r7TypesJustes`).
var (
	r7T5DirA = 24
	r7T5DirB = 24
)

// r7Porte32 consomme `R(1) g ; si g : R(32)` (primitive 0x14080d69c).
func r7Porte32(br *BitReader) {
	if br.ReadBit() {
		br.Skip(32)
	}
}

// r7BlocVariante consomme le bloc `si g==0 : [R(1);si 1:R(32)] + R(32) variant-name`
// (POLARITE INVERSEE) commun aux types 5, 6 et 7.
func r7BlocVariante(br *BitReader) {
	if !br.ReadBit() {
		r7Porte32(br)
		br.Skip(32) // variant-name
	}
}

// r7SkipChargeLot3 est la suite de r7SkipChargeLot2. Rend false pour tout type non ferme.
func r7SkipChargeLot3(br *BitReader, typ int, ctx r7Ctx) bool {
	switch typ {
	// --- 5 projectile_detonate (lecteur 0x1408096f8) ---
	//
	// RELECTURE DU LECTEUR, apres le verdict FAUX de TestR7Largeur : la fin du record avait
	// ete transcrite a tort en alternative. Le decompile donne une SUITE INCONDITIONNELLE :
	// `FUN_1406cf008` (R(1)) puis `FUN_14076dc04` puis `FUN_1424cd2fc` (R(2)).
	// Deux largeurs (`r7T5DirA`, `r7T5DirB`) ne se lisent pas au site d'appel : les seuls
	// immediats du corps de la fonction sont `MOV R9D,0xF` (le vecteur quantifie) et
	// `MOV R9D,0x8`. Elles sont CALIBREES par TestR7CalibreType5 sur l'oracle de trame.
	case 5:
		br.Skip(6) // R(6) type/enum (FUN_140809454)
		r7BlocVariante(br)
		r7Porte32(br)
		if !r7VecteurQuantifie(br, ctx, 15) { // MOV R9D,0xf au site d'appel
			return false
		}
		br.Skip(r7T5DirA) // FUN_14076dc04, largeur non lisible au site d'appel
		br.Skip(5)        // scalaire quantifie (FUN_1406d84b4)
		br.Skip(1)        // FUN_1406cf008
		br.Skip(9)        // R(9) puis valeur-1 (inline, +0x2c += 9)
		if br.ReadBit() { // inline, +0x2c += 1
			br.Skip(10 + 8) // FUN_140809530
		}
		br.Skip(1)        // FUN_1406cf008 — MANQUAIT dans la premiere transcription
		br.Skip(r7T5DirB) // FUN_14076dc04
		br.Skip(2)        // FUN_1424cd2fc
		return true

	// --- 6 projectile_impact_effect (lecteur 0x1410f03b4) ---
	case 6:
		r7BlocVariante(br)
		br.Skip(7 + 7)                        // deux R(7) (FUN_1406d84b4, [RSP+0x20]=7)
		br.Skip(19)                           // direction compressee
		if !r7VecteurQuantifie(br, ctx, 12) { // MOV R9D,0xc au site d'appel
			return false
		}
		br.Skip(19)    // direction compressee
		br.Skip(9 + 1) // R(9) valeur-1 puis R(1)
		return true

	// --- 7 projectile_object_impact_effect (vrai lecteur 0x142f1c6cc) : FERME ---
	case 7:
		r7BlocVariante(br)
		br.Skip(7 + 7)  // deux R(7)
		br.Skip(19)     // R(19)
		br.Skip(2)      // R(2) selecteur de bornes
		br.Skip(3 * 12) // vecteur LOCAL 3 x R(12) (FUN_140c1e924, bornes statiques)
		br.Skip(19)     // R(19)
		br.Skip(9 + 16) // R(9) valeur-1 puis R(16)
		br.Skip(1 + 1)  // deux R(1)
		return true

	// --- 1 damage_section_response (lecteur 0x140968368) : FERME, 10/14/29/33 bits ---
	case 1:
		br.Skip(5)
		if !br.ReadBit() { // FUN_1409684dc, POLARITE INVERSEE
			br.Skip(4)
		}
		br.Skip(3) // FUN_1424d0f48
		if br.ReadBit() {
			br.Skip(19)
		}
		return true

	// --- 2 restore_damage_section (lecteur 0x142ef90a4) : FERME, 3 ou 6 bits ---
	case 2:
		if br.ReadBit() {
			br.Skip(5) // ceil_log2(32)
		} else {
			br.Skip(2) // ceil_log2(3)
		}
		return true

	// --- 106 ObjectCollisionDamage (lecteur 0x14112134c) ---
	case 106:
		if !r7VecteurQuantifie(br, ctx, 12) {
			return false
		}
		br.Skip(10) // direction compressee (FUN_14076dc04, R9D=0xa)
		br.Skip(9)  // R(9) valeur-1
		return true
	}
	return r7SkipChargeLot4(br, typ, ctx)
}
