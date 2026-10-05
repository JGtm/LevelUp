package profile

// mpp_declare.go — LE DECOUPAGE DU BLOC MPP QUE LE FILM DECLARE, PAR LA TAILLE DE L ETAT DE
// CREATION DE SES OBJETS (ADR 0037 IR-7).
//
// # CE QUE LE FILM DECLARE
//
// Chaque record d image-cle range, juste avant l etat de creation de son objet, un mot `n1` : la
// TAILLE de la structure d etat de creation de l archetype, que l ecrivain du jeu y met
// (`FUN_142e2d08c` range `vtable+0x20` du descripteur). Le lecteur courant ne s en sert que comme
// garde (`FUN_142e2bfd0` : `if (0 < (int)n1)`). Le decodeur s en sert comme CLE du decoupage du
// bloc `object-multiplayer-properties` (`FUN_14080cfe8`), que l etat de creation des neuf
// archetypes de [tailleEtatDeCreationCourante] lit, directement ou par `FUN_1407f2224`.
//
// # LA REGLE ([MPPPourTailleDeclaree])
//
//	n1 == taille courante       -> le decoupage que lit l executable courant (MPPRelu)
//	n1 == taille courante - 4   -> 8/3, PRESUME PAR MESURE (MPPPresumeParMesure)
//	autre                       -> rien n est declare : l appelant garde son chemin, et le compte
//
// # PROVENANCES
//
// TAILLES COURANTES : RELUES dans l executable (Ghidra, lecture seule), `vtable+0x20` du
// descripteur de chaque archetype ; le getter est cite ligne a ligne.
//
// 8/3 : PRESUME PAR MESURE. L executable courant lit le bloc par des largeurs litterales et ne
// porte aucune condition qui le lise autrement : les films dont `n1` vaut la taille courante moins
// 4 ont ete ECRITS avec trois bits de moins dans ce bloc. Mesure du 2026-10-05 sur les sept bobines
// versionnees (`replay/testdata/minifilm_*`) : `n1` = taille courante - 4 sur les cinq bobines des
// formats 21, 24 et 25, pour les sept archetypes de [archetypeCleDuDecoupage] ; sous 8/3, l oracle
// `n2` est modal a 0,974-1,000 (0,02-0,54 sous 9/5) ; les memes identifiants de 32 bits sortent un
// bit plus tot (le champ de tete fait 8 bits) ; les deux autres bits tombent dans une plage de
// zeros (R(2), index, compte), sans effet sur les valeurs lues. Liste gelee :
// `TestDecoupagesPresumesParMesureGeles`.
//
// Ce que la lecture du bloc ajoute (`FUN_14080cfe8`, `FUN_142f1bc2c`) : le champ de tete est un
// champ de DRAPEAUX (valeurs {0, 2, 4, 64, 66}, jamais au-dela de 255 sur le format courant), dont
// les anciens builds n ont que huit. La structure du bloc fait 0x60 octets, et un seul retrait la
// raccourcit d exactement 4 octets : celui de R(2), un octet en +0x08 suivi du bourrage jusqu a
// l entier en +0x0c. Le decoupage le plus fidele est donc « R(2) absent, index R(5) » ; il lit les
// memes bits et les memes valeurs que 8/3 sur tous les records authentiques observes, ou R(2),
// l index et le compte valent 0 (ou 1 pour l index).

// ProvenanceMPP dit d ou vient un decoupage du bloc MPP.
type ProvenanceMPP uint8

// Les provenances d un decoupage MPP.
const (
	// MPPNonDeclare : aucun decoupage — la taille lue n en designe aucun, ou rien n a ete lu.
	MPPNonDeclare ProvenanceMPP = iota
	// MPPRelu : le decoupage que lit l executable courant, largeurs litterales de `FUN_14080cfe8`.
	MPPRelu
	// MPPPresumeParMesure : un decoupage que l executable courant ne lit pas, presume par mesure.
	MPPPresumeParMesure
)

// String rend l etiquette de la provenance, pour les journaux.
func (p ProvenanceMPP) String() string {
	switch p {
	case MPPRelu:
		return "relu"
	case MPPPresumeParMesure:
		return "presume_par_mesure"
	}
	return "non_declare"
}

// ecartDeTailleDuDecoupageAncien : les films ecrits sous 8/3 declarent une structure d etat de
// creation plus petite de 4 octets que celle de l executable courant (mesure, cf. l en-tete).
const ecartDeTailleDuDecoupageAncien = 4

// mppAncienPresumeParMesure rend le decoupage des films dont `n1` vaut la taille courante moins 4.
func mppAncienPresumeParMesure() MPPWidths { return MPPWidths{Lead: 8, Index: 3} }

// tailleEtatDeCreationCourante : la taille, en octets, que l executable courant declare pour l etat
// de creation des archetypes dont l etat de creation lit le bloc MPP. RELUE en `vtable+0x20`.
var tailleEtatDeCreationCourante = map[uint32]int32{
	35: 0x98, // FUN_14117f9b0 (vtable 0x143737178), etat de creation FUN_140f44c38
	36: 0x60, // FUN_141184270 (vtable 0x1436fec28), etat de creation FUN_1407f2224
	37: 0x68, // FUN_14119da10 (vtable 0x1436fea88), etat de creation FUN_1407f105c
	38: 0x68, // FUN_14119da10 (vtable 0x143c996f0), etat de creation FUN_1408f0b48
	39: 0x68, // FUN_14119da10 (vtable 0x14374cfc0), etat de creation FUN_1408f0b48
	40: 0xb0, // FUN_141175c40 (vtable 0x143736fd8), etat de creation FUN_1410a5a74
	41: 0xd4, // FUN_1408effc0 (vtable 0x1436fd5f0), etat de creation FUN_1408efb58
	42: 0xa8, // FUN_14119e5b0 (vtable 0x1436fd790), etat de creation FUN_1407f0c68
	43: 0x60, // FUN_141184270 (vtable 0x1436fd450), etat de creation FUN_140fe7630
}

// archetypeCleDuDecoupage dit si la regle est MESUREE sur l archetype `ti`. 36 et 39 ont leur
// taille relue mais aucune de leurs images-cles n est dans les bobines de la mesure : ils ne
// decident pas du decoupage d un film.
func archetypeCleDuDecoupage(ti uint32) bool {
	switch ti {
	case 35, 37, 38, 40, 41, 42, 43:
		return true
	}
	return false
}

// CleDuDecoupageMPP dit si un record d archetype `ti` porte la cle du decoupage MPP : sa taille
// courante est relue et la regle y est mesuree.
func CleDuDecoupageMPP(ti uint32) bool {
	_, ok := tailleEtatDeCreationCourante[ti]
	return ok && archetypeCleDuDecoupage(ti)
}

// MPPPourTailleDeclaree rend le decoupage du bloc MPP que declare un record d archetype `ti` dont
// le mot de taille d etat de creation vaut `n1`, et sa provenance. Faux quand l archetype ne porte
// pas la cle ([CleDuDecoupageMPP]) ou que la taille ne designe aucun decoupage.
func MPPPourTailleDeclaree(ti uint32, n1 int32) (MPPWidths, ProvenanceMPP, bool) {
	if !CleDuDecoupageMPP(ti) {
		return MPPWidths{}, MPPNonDeclare, false
	}
	switch courante := tailleEtatDeCreationCourante[ti]; n1 {
	case courante:
		return MPPParDefaut(), MPPRelu, true
	case courante - ecartDeTailleDuDecoupageAncien:
		return mppAncienPresumeParMesure(), MPPPresumeParMesure, true
	}
	return MPPWidths{}, MPPNonDeclare, false
}
