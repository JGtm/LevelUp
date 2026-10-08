package grammar

// components_frequences.go — LES DEUX ARCHETYPES « FREQUENCE » (`ti=3` et `ti=4`) ET LE SEUL
// COMPOSANT HOMONYME DE GRAMMAIRE DU JEU, `high-frequency`.
//
// LE DISPATCH ROUTE PAR NOM, ET UN NOM DESIGNE D ORDINAIRE UNE TABLE. Le jeu enregistre chaque
// composant d un archetype par `FUN_14064dd28(archetype + 8, index, &objet)` ; l objet porte la
// table (vtable) dont `+0x28` (thunk `FUN_14076ce9c` vers `+0x30`) est le lecteur que les deux
// boucles de composants appellent, delta (`FUN_14076cb60`, CALL 14076cd19) comme image-cle
// (`FUN_142e2c690`, CALL 142e2c7c9). Un seul nom de composant est enregistre sous deux tables de
// grammaires differentes : `high-frequency`. Pour lui, et pour lui seul, le lecteur se choisit par
// l archetype que le record porte dans son en-tete (`TypeIndex`, la valeur que la fonction
// d enregistrement ecrit en `archetype + 0x4754`).
//
//	archetype 3  FUN_140e460fc  i0 low-frequency   objet 0x144746e68, table 0x143d07b40,
//	                                               lecteur FUN_142ed4aec, ecrivain FUN_142eda938
//	                            i1 high-frequency  objet 0x144746e60, table 0x143d07af0,
//	                                               lecteur FUN_142ed4880, ecrivain FUN_142eda744
//	archetype 4  FUN_140e462d8  i0 high-frequency  objet 0x144746d38, table 0x143d06a60,
//	                                               lecteur FUN_14076d034, ecrivain FUN_142eda680
//
// Un archetype qui declarerait `high-frequency` sans etre l un de ces deux n a pas de table
// connue : le composant n y est pas porte (la traversee s arrete), plutot que lu sous la
// grammaire d un autre archetype. Le garde-fou de cette regle est
// `TestG6LesHomonymesSeRoutentParTable` (`ecs_dispatch_table_guard_test.go`).
//
// `low-frequency` NE SE LIT QUE DANS UN RECORD A MASQUE (DELTA, NEW). La boucle de l etat complet
// d image-cle, `FUN_142e2c690`, pose `DAT_144e61ea0 = 1` a son entree (142e2c6b8) et le remet a 0
// a sa sortie commune (142e2c76a) ; chaque lecteur de composant (CALL 142e2c7c9) s y execute sous
// cette portee, ou `FUN_14076f91c` rend vrai et `FUN_14076e494` lit la position BRUTE
// (`FUN_1411b259c` = `FUN_1406d676c(..., 0x60)`, R(96)) : la tete et chaque entree de
// FUN_142ed4aec y ont une autre largeur. La boucle delta (`FUN_14076cb60`) ne pose pas la portee.
// La marche d etat complet du depot pose cette portee ([Lecteur.portee]), sous laquelle
// [lireE494] lit comme le jeu ; dans un etat complet ([Lecteur.etatComplet]), le composant reste
// pourtant non porte (la traversee s arrete) : sa lecture n y a pas ete mesuree.

// Les archetypes qui enregistrent `high-frequency`, lus dans leurs fonctions d enregistrement.
const (
	archetypeFrequences     = 3 // FUN_140e460fc : *(archetype + 0x4754) = 3
	archetypeHauteFrequence = 4 // FUN_140e462d8 : *(archetype + 0x4754) = 4
)

// compLowFrequency : `ti=3 i0`, un nom a table unique (chaine 0x143c957c8, accesseur 0x141177bd0).
const compLowFrequency = "low-frequency"

// largeurEntreesBasseFrequence : le compte d entrees de `low-frequency`, `R(6)` (FUN_142ed4aec
// le lit sur 6 bits, FUN_142eda938 l ecrit sur 6 bits).
const largeurEntreesBasseFrequence = 6

// consumeLowFrequency porte FUN_142ed4aec (`ti=3 i0`, table 0x143d07b40), dont l ecrivain est
// FUN_142eda938 :
//
//	position     FUN_1424e0e38 -> FUN_14076e494(niveau 0x10, 0)        (+0x508)
//	avant / haut FUN_140c5f938(mode 0) -> FUN_140c5fa84                (+0x514, +0x520)
//	R(16) +0x52c ; R(8) +0x52e ; R(2) +0x52f
//	n = R(6)                                                           (+0x0)
//	n fois : f = R(3) (FUN_1424d9a30, octet +0x27 de l entree) ;
//	         si f & 1 : position (FUN_1424e0e38, niveau 0x10) ;
//	         si f & 2 : avant / haut (FUN_140c5f938, mode 0) ;
//	         R(16) (+0x24) ; R(5) (FUN_1424ccc74, +0x26)
//
// Rend faux, sans lire un bit, dans un etat complet d image-cle ([Lecteur.etatComplet], cf. l en-tete).
func consumeLowFrequency(br *Lecteur) bool {
	if br.etatComplet {
		return false
	}
	lireE494(br, niveauPosition)
	consumeObjectForwardAndUp(br)
	consumeChampsDeFrequence(br)
	for n := br.ReadBits(largeurEntreesBasseFrequence); n > 0; n-- {
		f := br.ReadBits(3)
		if f&1 != 0 {
			lireE494(br, niveauPosition)
		}
		if f&2 != 0 {
			consumeObjectForwardAndUp(br)
		}
		br.ReadBits(16)
		br.ReadBits(5)
	}
	return true
}

// consumeChampsDeFrequence lit les trois champs que les deux lecteurs de l archetype 3 partagent,
// aux memes adresses : R(16) en +0x52c, R(8) en +0x52e, R(2) en +0x52f (FUN_142ed4aec apres la
// position et l orientation ; FUN_142ed4880 en entier).
func consumeChampsDeFrequence(br *Lecteur) {
	br.ReadBits(16)
	br.ReadBits(8)
	br.ReadBits(2)
}

// consumeHighFrequency route `high-frequency` vers le lecteur de la table que l archetype du
// record a enregistree (cf. l en-tete). Rend faux pour un archetype sans table connue.
func consumeHighFrequency(br *Lecteur, typeIndex uint32) bool {
	switch typeIndex {
	case archetypeFrequences: // table 0x143d07af0, FUN_142ed4880 : R(16) + R(8) + R(2)
		consumeChampsDeFrequence(br)
		return true
	case archetypeHauteFrequence: // table 0x143d06a60, FUN_14076d034 : R(8), sonde
		br.obs.publishProbe(typeIndex, ProbeHighFrequency, br.ReadBits(8))
		return true
	}
	return false
}
