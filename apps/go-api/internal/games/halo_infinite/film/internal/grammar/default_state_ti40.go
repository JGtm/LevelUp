package grammar

// default_state_ti40.go — L'ETAT PAR DEFAUT de l'archetype VEHICULE (ti=40), porte feuille par
// feuille depuis `FUN_1410A5A74` (= vtable[0x60] du descripteur d'archetype 40, cf.
// KEYFRAME_ARCHETYPE_DEFAULTSTATE_TABLE.md:30, classe REAL). Dossier complet :
// .ai/V7.5/film_re/RE_DEFAULTSTATE_TI40_2026-08-31.md.
//
// CONVENTION DU BITREADER (param_4 dans l'exe) : chaque feuille qui lit fait `*(param_4+0x2c) += N`.
//
// GRAMMAIRE, feuille par feuille (adresse-source entre crochets) :
//
//	1  V                     R(1) ; si 1 -> R(8)              CALL 0x1406cf008 @0x1410a5a96 ;
//	                                                          R10D=8 @0x1410a5aa5, ADD @0x1410a5abf
//	2  bloc MPP              FUN_14080cfe8                    CALL 0x14080cfe8 @0x1410a5ad4
//	                         (publie MPPWord32)               (RDX=RSI = bitreader)
//	3  porte bVar14 -> +0x60 R(1)                             inline ADD [RSI+0x2c],1 @0x1410a5aed
//	4  si bVar14 : quat      NON ETABLIE (config runtime)     JNZ 0x1424a3a02 @0x1410a5b09 ;
//	   + FUN_140c1e79c                                        FUN_14076e524/FUN_140cc5128 (cf. § 3 RE)
//	5  FUN_14076dc04 -> +0x88 R(19)                           R9D=0x13 @0x1410a5b16, CALL @0x1410a5b1f
//	6  porte cVar3 -> +0xac  R(1)                             CALL 0x1406cf008 @0x1410a5b46
//	7a si cVar3==0 : opt32   R(1) ; si 1 -> R(32)             CALL 0x14080d69c @0x1410a5b63 (RDX=RSI)
//	7b si cVar3!=0 : liste   R(2) count ; count x opt32       ADD [RSI+0x2c],2 @0x1424a3a4c ;
//	                                                          boucle CALL 0x1406cf008 @0x1424a3afb,
//	                                                          MOV RDX,RSI @0x1424a3b09 + CALL
//	                                                          0x14080d6f0 @0x1424a3b0c
//
// LES CINQ FEUILLES SONT LUES (2026-09-18, lot 5.1.7-b). La quatrieme — le quaternion
// `FUN_14076e494` derriere la porte de flux `bVar14` — a longtemps porte la mention « largeur non
// etablie statiquement, globaux de configuration runtime » et se modelisait ABSENTE. LA MENTION
// ETAIT PERIMEE : les deux globaux qu elle nommait (l index `DAT_144632be0` de `FUN_14076e524`, les
// trois largeurs per-axe `DAT_1445cc9e0` de `FUN_140cc5128`) entrent par le CATALOGUE DE LA CARTE
// depuis le lot 3.4.1, et les deux fonctions de la feuille sont portees depuis le lot R7-b. La
// feuille se lit donc a la largeur de la carte du match — cf. [consumeVehicleMediaFrame] pour la
// mesure qui l etablit et son temoin negatif.
//
// `ti=40` EST DONC INSCRIT dans `defaultStateDeserByTI`, et la regle de `default_state_arch.go`
// est tenue, pas contournee : plus aucune feuille n est devinee.
//
// L IDENTITE DU CHASSIS voyage dans MPPWord32 (feuille 2), lue AVANT toute position et toute porte
// optionnelle : elle est donc lisible meme sans decoder i0 (dont la grammaire dyn.-prec. diverge
// de la voie world-object, cf. § 6 du dossier RE).

// VehicleTypeIndex est l'archetype (typeIndex) des VEHICULES. Comme GroundWeaponTypeIndex, c'est
// un index de build a VERIFIER par le nom des composants du registre DU FILM, pas une constante du
// format (cadrage § 1.5 : six empreintes de registre, mais ti=40 stable a 48 composants).
const VehicleTypeIndex = 40

// consumeDefaultStateTI40 porte FUN_1410a5a74 (archetype 40, « vehicule »).
//
// LES CINQ FEUILLES SONT LUES, ET L ARCHETYPE EST INSCRIT dans `defaultStateDeserByTI` depuis le
// 2026-09-18 (lot 5.1.7-b). La feuille 4 ne se modelise plus absente : elle se LIT (cf.
// [consumeVehicleMediaFrame]).
func consumeDefaultStateTI40(br *Lecteur) {
	consumeVersionPrefix(br)              // 1. V : R(1) ; si 1 -> R(8)
	consumeMultiplayerPropertiesBlock(br) // 2. FUN_14080cfe8 : bloc MPP (publie MPPWord32)
	if br.ReadBit() {                     // 3. porte bVar14 -> DST+0x60 (R(1) inconditionnel)
		consumeVehicleMediaFrame(br) // 4. quaternion + FUN_140c1e79c, aux largeurs de la carte
	}
	br.ReadBits(19)    // 5. FUN_14076dc04 : R(19), largeur R9D=0x13
	if !br.ReadBit() { // 6. porte cVar3 -> DST+0xac ; branche selon la valeur
		consumeOpt32(br) // 7a. cVar3==0 : FUN_14080d69c = R(1) ; si 1 -> R(32)
	} else {
		n := br.ReadBits(2) // 7b. cVar3!=0 : R(2) count
		for i := uint64(0); i < n; i++ {
			consumeOpt32(br) // count x [R(1) ; si 1 -> R(32) FUN_14080d6f0 sur flux film]
		}
	}
}

// consumeVehicleMediaFrame porte la feuille 4 (bloc froid @0x1424a3a02) : le quaternion
// `FUN_14076e494` (-> DST+0x64) puis `FUN_140c1e79c`.
//
// ELLE ETAIT MODELISEE ABSENTE JUSQU AU 2026-09-18, sous la mention « largeur config-dependante,
// non etablie statiquement ». LA MENTION ETAIT PERIMEE : les deux globaux qu elle nommait —
// l index `DAT_144632be0` et les trois largeurs per-axe `DAT_1445cc9e0` — entrent par le CATALOGUE
// DE LA CARTE depuis le lot 3.4.1, et les DEUX fonctions de la feuille sont portees depuis le lot
// R7-b : `FUN_14076e494` par [consumeSimStateHandleTail] et `FUN_140c1e79c` par [consume140c1e79c].
// La feuille se lit donc, a la largeur de la carte du match, comme le chemin world-object.
//
// MESURE QUI L ETABLIT (`4f77afc1`, 1 140 records `ti=40` d image-cle, 2026-09-18) : la porte
// `bVar14` vaut 1 sur **470 records (41,2 %)** — elle n est donc pas negligeable. Feuille LUE, les
// deux populations butent au MEME rang, sans exception : `bVar14 == 0` 661/661 a `i30`,
// `bVar14 == 1` 470/470 a `i30`. TEMOIN NEGATIF, feuille modelisee absente : les 470 partent de
// travers et rendent `DesyncAt == -1` — la boucle de composants ne tourne pas, ce qui est
// exactement le faux « aucun bloquant » que le golden 0.A.3 portait sur `ti=40`.
func consumeVehicleMediaFrame(br *Lecteur) {
	consumeSimStateHandleTail(br) // FUN_14076e494
	consume140c1e79c(br)          // FUN_140c1e79c
}

// VehicleDefaultStateMinBits est la largeur du chemin minimal de consumeDefaultStateTI40 (toutes
// portes fermees, bVar14 == 0 donc quat absent, defaut MPP 9/5) :
//
//	1 (V) + 56 (MPP) + 1 (porte bVar14) + 19 (R19) + 1 (porte cVar3) + 1 (opt32 ferme) = 79.
//
// A comparer aux 80 bits de ti=42. Publiee parce que l'oracle d'offset la COMPARE au flux : c'est
// la prediction du plancher que l'histogramme des distances doit retrouver (portes ouvertes = pic
// au-dessus).
const VehicleDefaultStateMinBits = 79
