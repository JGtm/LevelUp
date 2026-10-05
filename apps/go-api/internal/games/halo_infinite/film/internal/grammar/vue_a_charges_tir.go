package grammar

import "levelup/go-api/internal/games/halo_infinite/film/internal/source"

// vue_a_charges_tir.go — LA CHARGE DU TIR `action_weapon_fire` (genre 36), PORTEE DEPUIS SON
// LECTEUR `FUN_14080c1f8` (lot LN de la campagne de grammaire).
//
// La tete (jusqu aux deux R(1) qui suivent l arme) est celle que [lireEnteteTir36] lit dans un
// paquet ; ici elle est lue sur le lecteur de la vue A, a la position du message. Les sous-lecteurs
// deja portes sont appeles, jamais recopies : [consume142f26740] (`FUN_140c9e4d8`, troisieme
// argument 0), [consume1408eff64], [lireVecteur1431a0cbc], [consumeOpt1431a0abc], [lireE494].

// largeurTirCourt est l immediat `R9D = 0xa` de `FUN_14076dc04` sur la branche courte du tir
// (14080c465).
const largeurTirCourt = 10

// Les largeurs de la boucle des composantes : `local_98 = 0xc` quand le premier compte vaut 1,
// 4 sinon, et `FUN_14102bd24` les plafonne a 6 quand la cible designee porte le genre 1.
const (
	largeurComposanteSeule    = 12
	largeurComposantesMulti   = 4
	largeurComposanteGenreUn  = 6
	ciblesAIndexCourt         = 3 // en dessous, l index de cible est R(1) ; sinon R(4)
	largeurIndexCibleCourt    = 1
	largeurIndexCibleLong     = 4
	largeurComptesDuTir       = 4  // FUN_14080cc68 : R(4) quand le compte n est pas 0 ou 1
	largeurPoidsDeComposante  = 16 // R(0x10) de la boucle des composantes
	largeurGenreDeCible       = 2  // R(2) de la boucle des cibles
	largeurEnteteComposante   = 4  // R(4) en tete de chaque composante
	versionTirQueueHorodatage = 1  // FUN_141102ed0(0x24) > 1 : FUN_1406d84b4 a quatre bits
	largeurQueueHorodatage    = 4
)

// chargeTirArme porte `FUN_14080c1f8` (`action_weapon_fire`).
func chargeTirArme(br *Lecteur) bool {
	court, bloc := br.ReadBit(), br.ReadBit() // [0], [0x1c]
	br.Skip(largeurNumeroDeTir + 1)           // FUN_141fcf670 : R(7) puis R(1)
	consumeGate0R(br, 5)                      // FUN_1407f2034 -> FUN_1407f2058
	consumeID2(br)                            // FUN_1406d00ec
	consumeGateR(br, 32)                      // FUN_14080d69c
	br.Skip(32 + 1 + 1)                       // FUN_14080dec4 « variant_name », [0x1d], [2]
	horodatage := false
	if bloc {
		br.Skip(1)                // [0x2dc]
		horodatage = br.ReadBit() // [0x2dd]
	}
	if horodatage {
		consumeOpt1431a0abc(br)
	}
	if court {
		br.Skip(largeurTirCourt) // FUN_14076dc04, puis fin du message
		return true
	}
	if lireCiblesEtComposantes(br) {
		ouvert := source.BitAt(br.Octets(), br.BitPos()) == 1 // la garde que FUN_140c9e4d8 rend
		consume142f26740(br)                                  // FUN_140c9e4d8(..., 0, &garde)
		consume1408eff64(br, true)
		if !ouvert {
			br.Skip(int(FireAimBits)) // R(0x1e) : la visee
		}
	}
	lireQueueDuTir(br, bloc, horodatage)
	return true
}

// lireCiblesEtComposantes lit les comptes (`FUN_14080cc68`), la boucle des cibles puis celle des
// composantes, et dit si les deux lecteurs composites et la visee suivent (`bVar14` : vrai sauf
// quand la derniere composante presente porte un R(3) nul).
func lireCiblesEtComposantes(br *Lecteur) bool {
	nComp, nCib := lireComptesDuTir(br)
	var genreUn [1 << largeurComptesDuTir]bool
	for i := range nCib {
		genreUn[i] = br.ReadBits(largeurGenreDeCible) == 1
		br.Skip(1)
		readVarWidthInt(br, varWidthProbeCategory) // FUN_1406d3140(..., 1, ...) : param_5 = 1
	}
	base := largeurComposantesMulti
	if nComp == 1 {
		base = largeurComposanteSeule
	}
	suite := true
	for range nComp {
		br.Skip(largeurEnteteComposante)
		if !br.ReadBit() {
			continue
		}
		suite = br.ReadBits(uint(bitLen(6))) != 0 // FUN_1406d310c(6)
		largeurIndex := uint(largeurIndexCibleLong)
		if nCib < ciblesAIndexCourt {
			largeurIndex = largeurIndexCibleCourt
		}
		idx := br.ReadBits(largeurIndex)
		br.Skip(largeurPoidsDeComposante)
		w := base
		if genreUn[idx] {
			w = min(w, largeurComposanteGenreUn) // FUN_14102bd24
		}
		br.Skip(3 * w) // FUN_140c1e924 -> FUN_140c1e9d4 : trois R(w)
	}
	return suite
}

// lireComptesDuTir porte `FUN_14080cc68` et rend (premier compte, second compte) : le premier
// compte les composantes, le second les cibles (dont la boucle est lue la premiere).
func lireComptesDuTir(br *Lecteur) (nComp, nCib int) {
	if br.ReadBit() {
		return 0, 0
	}
	nComp = 1
	if !br.ReadBit() {
		nComp = int(br.ReadBits(largeurComptesDuTir))
	}
	if br.ReadBit() {
		return nComp, 0
	}
	nCib = 1
	if !br.ReadBit() {
		nCib = int(br.ReadBits(largeurComptesDuTir))
	}
	return nComp, nCib
}

// lireQueueDuTir lit la fin de `FUN_14080c1f8`, apres l etiquette `LAB_14080c88d`.
func lireQueueDuTir(br *Lecteur, bloc, horodatage bool) {
	if !horodatage {
		consumeGateR(br, 6) // R(1) ; si 1 : R(6)
		br.Skip(6)
	}
	if bloc {
		lireVecteur1431a0cbc(br)
		if versionNative(TypeTirArme) > versionTirQueueHorodatage { // FUN_141102ed0(0x24)
			br.Skip(largeurQueueHorodatage)
		}
	} else {
		br.Skip(2)          // FUN_14080cb98
		a2 := br.ReadBit()  // FUN_14080cb50
		consumeGateR(br, 4) // FUN_14080cb50 : R(1) ; si 1 : FUN_1406d84b4 a 4 bits
		if a2 {
			consumeGateR(br, 12) // FUN_14320c36c : R(1) ; si 1 : deux FUN_1406d84b4 a 6 bits
			consumeGateR(br, 12) // FUN_142a40f18 : idem
		}
	}
	br.Skip(6)
	consumeGateR(br, 7) // R(1) ; si 1 : FUN_1406d84b4 a 7 bits
	if br.ReadBit() {
		lireE494(br, niveauPosition) // FUN_14076e494(..., 0x10, 0, 1, 0)
	}
}
