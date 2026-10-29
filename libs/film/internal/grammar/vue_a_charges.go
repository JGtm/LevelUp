package grammar

// vue_a_charges.go — LES CHARGES DES MESSAGES DE LA VUE A, PORTEES DEPUIS LEUR LECTEUR DU JEU (lots
// LN et VA de la campagne de grammaire).
//
// Une charge est le lecteur `vtable + 0x68` d un descripteur de [tableDesGenresVueA], appele par
// `FUN_14080a9d4` avec `param_5 = 1` : toute branche `param_5 == 0` d un lecteur est morte dans le
// film. Chaque fonction ci-dessous porte UN lecteur du jeu, nomme par son adresse ; les largeurs
// sont celles du lecteur (increments du compteur `+0x2c` du flux) ou l immediat du site d appel
// (`R9D` pour `FUN_14076dc04`, cinquieme argument pour `FUN_1406d84b4`). Une charge rend faux
// quand le jeu lirait une valeur qui ne se lit ni dans l executable ni dans le film.
//
// Les primitives du jeu et leur portage :
//
//	FUN_1406cf008  R(1)                               br.ReadBit
//	FUN_14080d69c  R(1) g ; si g : R(32)              consumeGateR(br, 32)
//	FUN_1407f2058  R(1) g ; si g == 0 : R(5)          consumeGate0R(br, 5)
//	FUN_14076dc04  R(n), n = R9D au site d appel       br.Skip(n)
//	FUN_1406d84b4  R(n), n = cinquieme argument       br.Skip(n)
//	FUN_14080dec4  R(32)                              br.Skip(32)
//	FUN_1406d3140  reference de domaine               readVarWidthInt

// largeurDirection est l immediat `R9D = 0x13` de `FUN_14076dc04` aux sites qui lisent une
// direction unitaire quantifiee (`FUN_1406d8288` la decode sans lire de bit).
const largeurDirection = 19

// chargeDuGenre rend le lecteur de la charge d un genre ; nil : charge non portee. La version d un
// genre (`FUN_141102ed0`) y est la version native : la vue A ne se lit que sur un film qui la
// declare ([classeDesGenres]).
func chargeDuGenre(genre int) func(*Lecteur) bool { //nolint:gocyclo // un case par genre de message porte (aiguillage)
	switch genre {
	case 0:
		return chargeDegatsApres
	case 1:
		return chargeReponseDeSection
	case 2:
		return chargeSectionRestauree
	case 7:
		return chargeImpactSurObjet
	case 8, 53:
		return chargeEmbarquement
	case 9:
		return chargeRamassage
	case 15:
		return chargeScript
	case 21:
		return chargeZoom
	case 22:
		return chargeSortieDeVehicule
	case TypeTirArme:
		return chargeTirArme
	case 38:
		return chargeRechargement
	case 39:
		return chargeLancerInitie
	case 75:
		return chargeDialogueIA
	case 76:
		return chargeDialogue2D
	case 80:
		return chargeEffetDIA
	case 88:
		return chargeRetourDeCarte
	case 109:
		return chargeCycleDeVieIA
	case GenreJoueurTue:
		return chargeJoueurTue
	case 116:
		return chargeEffetsDeTeleportation
	case 120:
		return chargeAppelDeJoueur
	}
	if c := chargeDArme(genre); c != nil {
		return c
	}
	return chargeDEvenementDeJeu(genre)
}

// degatsF58AvecOctet est la valeur du R(4) range en [+0x58] a laquelle `FUN_1407f15a4` lit un R(8)
// apres la porte de `FUN_141015740` (`*(int *)(param_3 + 0x58) == 1`).
const degatsF58AvecOctet = 1

// chargeDegatsApres porte `FUN_1407f15a4` (`damage_aftermath`).
//
// EXEMPTION A LA REGLE DES DEUX COPIES (CLAUDE.md n. 6), 2026-10-05 : c est la troisieme lecture de
// production de `FUN_1407f15a4`, apres [evBody0] (`chaine_d_evenements_corps.go`, le rattrapage des kills) et
// [lot1DecodeDamageAftermath] (`weapon_hits_decode.go`). Les trois ne sont pas centralisees ici :
// [lot1DecodeDamageAftermath] lit la porte de `FUN_1407f2058` a polarite inversee (« si 1 : R(5) »,
// D-LN-2 de `LOT_LN.md`), et les reunir changerait la sortie de `weapon_hits` et toucherait le
// rattrapage des kills. Retrait : le lot dedie qui centralise `FUN_1407f15a4` et corrige D-LN-2.
func chargeDegatsApres(br *Lecteur) bool {
	consumeGateR(br, 32)      // FUN_14080d69c
	consumeGate0R(br, 5)      // FUN_1407f2058
	br.Skip(largeurDirection) // FUN_14076dc04, R9D = EBP + 0x14 = 0x13 (1407f15e8)
	if br.ReadBit() {         // [+0x20]
		br.Skip(largeurDirection) // FUN_14076dc04, R9D = 0x13 (1407f1a66)
		br.Skip(12)               // FUN_1406d84b4, cinquieme argument 0xc (1407f1a89)
	}
	br.Skip(5 + 5 + 6)  // FUN_1406d84b4 x 3, cinquiemes arguments 5, 5, 6 (1407f1631..1407f166a)
	consumeGateR(br, 5) // R(1) puis FUN_1406d84b4 (5) au bloc froid 1423260c0
	var dernier bool
	for range drapeauxDeDegats { // les quinze R(1) de [+0x44]
		dernier = br.ReadBit()
	}
	if dernier { // le quinzieme (bit 28) garde un R(32)
		br.Skip(32)
	}
	br.Skip(1 + 3)        // R(1) bit 19 de [+0x44], R(3) de [+0x5c]
	br.Skip(5 + 5)        // FUN_1424cd17c, FUN_1424cd150 : FUN_1406d84b4 a cinq bits
	br.Skip(1 + 4)        // R(1) d echelle, FUN_1407f1f24 R(4)
	consumeGate0R(br, 10) // FUN_1407f1e4c
	f58 := br.ReadBits(4) // [+0x58]
	consumeGateR(br, 32)  // R(1) puis FUN_141015740 R(32)
	if f58 == degatsF58AvecOctet {
		br.Skip(8)
	}
	br.Skip(bitLen(10)) // FUN_1406d310c(10)
	if br.ReadBit() {
		readVarWidthInt(br, 0) // FUN_1406d3140(..., 0, ...)
	}
	return true
}

// drapeauxDeDegats est le nombre de R(1) que `FUN_1407f15a4` range dans [+0x44] avant le R(32)
// garde par le dernier (bits 1 a 6, 9, 7, 8, 11, 20, 21, 23, 30, 28).
const drapeauxDeDegats = 15

// chargeReponseDeSection porte `FUN_140968368` (`damage_section_response`).
func chargeReponseDeSection(br *Lecteur) bool {
	br.Skip(5)           // R(5)
	consumeGate0R(br, 4) // FUN_1409684dc
	br.Skip(3)           // FUN_1424d0f48
	if br.ReadBit() {
		br.Skip(largeurDirection) // FUN_14076dc04, R9D = 0x13 (1409683f1)
	}
	return true
}

// chargeSectionRestauree porte `FUN_142ef90a4` (`restore_damage_section`) : R(1) puis un index de
// `FUN_1406d310c(3)` bits si 0, de `FUN_1406d310c(0x20)` bits si 1.
func chargeSectionRestauree(br *Lecteur) bool {
	if br.ReadBit() {
		br.Skip(bitLen(0x20))
	} else {
		br.Skip(bitLen(3))
	}
	return true
}

// chargeImpactSurObjet porte `FUN_142f1c6cc`, atteint par le thunk `FUN_142f17474`
// (`projectile_object_impact_effect` : `MOV RDX,R9 ; MOV RCX,R8 ; JMP 0x142f1c6cc`).
func chargeImpactSurObjet(br *Lecteur) bool {
	lireVarianteSiAbsente(br)
	br.Skip(7 + 7)            // FUN_1406d84b4 x 2, cinquieme argument EBX = 7 (142f1c746)
	br.Skip(largeurDirection) // FUN_14076dc04, R9D = EBX + 0xc (142f1c77f)
	br.Skip(2)                // R(2)
	consume140c1e9d4(br, 12)  // FUN_140c1e924 -> FUN_140c1e9d4, R9D = 0xc
	br.Skip(largeurDirection) // FUN_14076dc04, R9D = R14D (142f1c871)
	br.Skip(9 + 16 + 1 + 1)   // R(9), R(16), R(1), R(1)
	return true
}

// chargeEmbarquement porte `FUN_142f168c0`, lecteur commun de `biped_board_vehicle` et de
// `unit_enter_vehicle` : R(6).
func chargeEmbarquement(br *Lecteur) bool {
	br.Skip(6)
	return true
}

// chargeRamassage porte `FUN_141037828` (`biped_pickup`).
func chargeRamassage(br *Lecteur) bool {
	br.Skip(3)           // R(3)
	consumeGateR(br, 32) // FUN_14080d69c
	return true
}

// chargeZoom porte `FUN_141168b28` (`unit_zoom`) -> `FUN_14080cb98` : R(2).
func chargeZoom(br *Lecteur) bool {
	br.Skip(2)
	return true
}

// chargeSortieDeVehicule porte `FUN_142f17b94` (`unit_exit_vehicle`).
func chargeSortieDeVehicule(br *Lecteur) bool {
	br.Skip(6 + 1 + 3) // R(6), FUN_1406cf008, FUN_140c1e31c R(3)
	return true
}

// chargeRechargement porte `FUN_1407f0ff8` (`weapon_reload`).
func chargeRechargement(br *Lecteur) bool {
	br.Skip(4)           // FUN_1406cf008 x 4
	consumeGate0R(br, 5) // FUN_1407f2058
	return true
}

// chargeDialogueIA porte `FUN_140f2e634` (`AIDialog`).
func chargeDialogueIA(br *Lecteur) bool {
	br.Skip(5)                   // FUN_140f2ea18
	consumeGateR(br, 32)         // FUN_1406cf008 puis FUN_14080d6f0
	consumeGate0R(br, 12)        // FUN_140f2e7a4
	consumeGate0R(br, 5)         // FUN_1407f2058
	n := int(br.ReadBits(1)) + 1 // FUN_1424e1464 : R(1) + 1
	br.Skip(n)                   // FUN_1406cf008 x n
	return true
}

// chargeDialogue2D porte `FUN_140f2e87c` (`Dialogue2D`).
func chargeDialogue2D(br *Lecteur) bool {
	consumeGateR(br, 32) // FUN_14080d69c
	consumeGateR(br, 32) // FUN_1406cf008 puis FUN_14080d6f0
	consumeGate0R(br, 5) // FUN_1407f2058
	br.Skip(32)          // FUN_1406ce648 : 32 x R(1)
	return true
}

// largeurModeIA est le R(2) de `FUN_142af27f8` et de `FUN_142ef15e0`.
const largeurModeIA = 2

// chargeEffetDIA porte `FUN_142ef8f74` (`networked_ai_effect`) : R(1) ; si 1 : `FUN_14080d6f0`,
// puis `FUN_142ef15e0` : R(2) mode ; 0 : une position de niveau 0x10 ; 1 : `FUN_142eefa5c`, trois
// positions de niveau 0x10 ; sinon rien.
func chargeEffetDIA(br *Lecteur) bool {
	consumeGateR(br, 32)
	positions := [1 << largeurModeIA]int{1, 3, 0, 0} // 0 : une ; 1 : FUN_142eefa5c, trois ; 2, 3 : aucune
	for range positions[br.ReadBits(largeurModeIA)] {
		lireE494(br, niveauPosition)
	}
	return true
}

// chargeRetourDeCarte porte `FUN_142ef8b5c` (`RevertMap`) : R(1).
func chargeRetourDeCarte(br *Lecteur) bool {
	br.Skip(1)
	return true
}

// chargeCycleDeVieIA porte `FUN_142c61310` (`PersonalAILifceycleEffect`) : `FUN_142af27f8` R(2),
// deux R(1) gardant chacun `FUN_14080d6f0`, puis `FUN_14080dec4` « marker-name ».
func chargeCycleDeVieIA(br *Lecteur) bool {
	br.Skip(largeurModeIA)
	consumeGateR(br, 32)
	consumeGateR(br, 32)
	br.Skip(32)
	return true
}

// chargeAppelDeJoueur porte `FUN_142f163e0` (`PlayerCalloutRequest`) : `FUN_1424e2f20` R(5), une
// position de niveau 0x10, puis deux listes comptees par `FUN_1424e1d48` (R(4)) de references
// `FUN_1406d3140` en categorie 1 (`MOV R8D,0x1` @142f16454) puis 0 (`XOR R8D,R8D` @142f1648a).
func chargeAppelDeJoueur(br *Lecteur) bool {
	br.Skip(5)
	lireE494(br, niveauPosition) // FUN_14076e494(..., 0x10, 1, 1, 0)
	for _, categorie := range [2]int{varWidthProbeCategory, 0} {
		for range br.ReadBits(4) {
			readVarWidthInt(br, categorie)
		}
	}
	return true
}
