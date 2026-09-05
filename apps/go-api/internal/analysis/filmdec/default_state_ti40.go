package filmdec

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
// LA SEULE FEUILLE NON ETABLIE STATIQUEMENT est la 4 (le quaternion `FUN_14076e494`), derriere la
// porte de flux `bVar14`. Sa largeur depend de globaux de configuration runtime : l'index
// `DAT_144632be0` (FUN_14076e524) et les trois largeurs per-axe `DAT_1445cc9e0` (FUN_140cc5128).
// C'est EXACTEMENT le bloc media-frame du BIPEDE (consumeBipedDefaultStateMediaFrame,
// default_state.go:299), lui aussi modelise ABSENT. On le modelise absent ici de la meme facon
// (vehicleMediaFrameBits, defaut 0) : bit-exact tant que `bVar14 == 0` (le cas nominal d'un spawn),
// desaligne sinon. La part de records a `bVar14 == 1` se MESURE (oracle de position), elle ne se
// suppose pas.
//
// POURQUOI ti=40 N'EST PAS DANS defaultStateDeserByTI. Par la regle de default_state_arch.go:30-32
// (« un archetype dont UNE largeur de feuille n'est pas etablie statiquement n'est PAS inscrit »),
// et parce que la feuille 4 est config-dependante, `ti=40` reste HORS de la table. L'inscription
// est la meme etape post-oracle que pour ti=42 : elle attend la confirmation que le port atterrit
// sur i0 (donc que bVar14 est nominalement 0). Elle est laissee au superviseur (cache Go partage).
//
// L'IDENTITE DU CHASSIS voyage dans MPPWord32 (feuille 2), lue AVANT toute position et toute porte
// optionnelle : elle est donc lisible meme sans decoder i0 (dont la grammaire dyn.-prec. diverge
// de la voie world-object, cf. § 6 du dossier RE).

// VehicleTypeIndex est l'archetype (typeIndex) des VEHICULES. Comme GroundWeaponTypeIndex, c'est
// un index de build a VERIFIER par le nom des composants du registre DU FILM, pas une constante du
// format (cadrage § 1.5 : six empreintes de registre, mais ti=40 stable a 48 composants).
const VehicleTypeIndex = 40

// vehicleMediaFrameBits est la largeur du bloc quaternion `FUN_14076e494` (feuille 4), atteint
// quand la porte de flux `bVar14` vaut 1. Cette largeur depend de globaux de config runtime
// (DAT_1445cc9e0 axis widths, DAT_144632be0 index) non recuperables statiquement — voir le § 3 du
// dossier RE. Defaut 0 : le bloc est modelise ABSENT, comme le media-frame du bipede
// (bipedDefaultStateTailBits). Expose pour qu'un harnais de calibration puisse sonder l'alternative.
var vehicleMediaFrameBits = 0

// SetVehicleMediaFrameBits fixe la largeur modelisee du bloc quaternion de la feuille 4.
func SetVehicleMediaFrameBits(n int) { vehicleMediaFrameBits = n }

// consumeDefaultStateTI40 porte FUN_1410a5a74 (archetype 40, « vehicule »).
//
// ATTENTION : la feuille 4 (quaternion, porte bVar14) n'est pas etablie statiquement ; ce port la
// modelise absente (vehicleMediaFrameBits = 0). Le deser est donc bit-exact sur le CHEMIN NOMINAL
// (bVar14 == 0) et NON inscrit dans defaultStateDeserByTI (cf. l'en-tete de fichier).
func consumeDefaultStateTI40(br *BitReader) {
	consumeVersionPrefix(br)              // 1. V : R(1) ; si 1 -> R(8)
	consumeMultiplayerPropertiesBlock(br) // 2. FUN_14080cfe8 : bloc MPP (publie MPPWord32)
	if br.ReadBit() {                     // 3. porte bVar14 -> DST+0x60 (R(1) inconditionnel)
		consumeVehicleMediaFrame(br) // 4. quaternion + FUN_140c1e79c : CONFIG-DEPENDANT
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
// `FUN_14076e494` (-> DST+0x64) puis `FUN_140c1e79c`. Sa largeur reelle depend de globaux de config
// runtime (cf. § 3 du dossier RE), donc elle est modelisee par vehicleMediaFrameBits (defaut 0 ->
// bloc absent). Le meme motif que consumeBipedDefaultStateTail (default_state.go:494).
func consumeVehicleMediaFrame(br *BitReader) {
	if vehicleMediaFrameBits > 0 {
		br.Skip(vehicleMediaFrameBits)
	}
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
