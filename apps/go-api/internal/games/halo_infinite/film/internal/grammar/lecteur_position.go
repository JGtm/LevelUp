package grammar

import "levelup/go-api/internal/games/halo_infinite/film/internal/profile"

// lecteur_position.go — LE PORTAGE UNIQUE DE `FUN_14076e524`, LE LECTEUR DE POSITION QUANTIFIEE,
// ET DE SES DEUX ENVELOPPES (lot J6.3 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, DU-1).
//
// # CE QUE LE JEU LIT
//
// Releve Ghidra du 2026-09-27 (`HaloInfinite.exe`, base 0x140000000, lecture seule ;
// `.ai/V7.5/film_re/RELEVES_J6_GHIDRA_2026-09-27.md`) :
//
//	FUN_14076e524(dst, lecteur, indexOut, NIVEAU)
//	    R(1) porte ; si 0 -> index de plage sur DAT_144632be0 bits
//	    index == -1 -> ligne NIVEAU de la table DEFAUT     DAT_1445cc9e0 + NIVEAU*0xc
//	    index >= 0  -> ligne NIVEAU de la table PAR INDEX   DAT_1445ccbe0 + (index*0x20 + NIVEAU)*0xc
//	    puis FUN_140cc5128 : les trois axes, aux largeurs de la ligne
//	FUN_14076e494(lecteur, dst, NIVEAU, p4, p5, p6)     MOV R9D,R8D en 14076e49a
//	    FUN_14076f91c() vrai -> FUN_1411b259c = R(96) brut
//	    sinon p6 == 0        -> FUN_14076e524(NIVEAU)
//	    sinon                -> FUN_141f85880(dst, lecteur, p6, NIVEAU) : la loi au NIVEAU sur les
//	                            bornes p6, puis trois axes (sans porte ni index)
//	FUN_14076e420(lecteur, dst, NIVEAU, p4)             MOV ESI,R8D en 14076e43c
//	    precHigh = R(1), puis FUN_14076e494(NIVEAU, p6 = precHigh ? &DAT_143b8c6d0 : 0)
//	DAT_143b8c6d0 = +/-100 sur les trois axes (read_memory, `0000c8c2 0000c842` x3)
//
// Le NIVEAU ne choisit qu une LIGNE de table ; c est un IMMEDIAT du site d appel, jamais le
// niveau du registre de `chunk_00` (aucun site ne le transmet au lecteur, releve §1 bilan). Les
// deux tables sont remplies par la MEME loi (`profile.LargeursAxeDuNiveau`, `FUN_140be9b88`) :
// sur les bornes +/-20000 du build pour la table DEFAUT, sur celles de la plage pour la table PAR
// INDEX. `DAT_144632be0` n est PAS une constante : `FUN_140be9a14` la pose a 1 quand la carte
// declare une plage, a `ceilLog2(compte)` sinon (GA2-3) ; le profil la porte
// (`Mouvement.WorldObject.IndexW`, `profile.LargeurIndexDePlage`).
//
// # UN SEUL PORTAGE, TROIS POINTS D ENTREE
//
//	lireE524   la lecture nue (porte, index, trois axes) — le repli de `FUN_14076f3ec`
//	lireE494   la garde de pleine precision puis `lireE524` — la plupart des sites
//	lireE420   le bit precHigh puis l enveloppe a bornes — world-object i0, et les deux sites qui
//	           la recopient en ligne (`FUN_1406cfe44` branche absolue, `FUN_140f7ea14` ->
//	           `FUN_14076e4ec`)
//
// Chaque site appelle l un d eux avec l IMMEDIAT RELEVE CHEZ LE JEU, et la table des sites
// (`lecteur_position_ratchet_test.go`) le verifie par l AST : un site qui relit la position a la
// main, ou avec un autre niveau que le sien, rougit. Avant ce lot, la meme fonction du jeu avait
// NEUF portages concurrents (`consumeSimStateHandleTail`, `consumeAbsoluteWithGate`,
// `consumeAbsolutePayload`, `consumeQuantVec3`, `consumeQuantVec3WithGate`,
// `consumeE524PositionBody`, `consumeQuat16`, `readTranslocVec`, la branche en ligne de
// `dispatch_object.go`) : largeur 6 + niveau du registre, largeurs du descripteur de traversee,
// index fige a 1, largeurs de la carte sous la porte posee, 16 bits plats — chacun faux ailleurs.
//
// ONZE SITES RESTENT HORS DU PORTAGE, en EXCEPTIONS DATEES (decision du superviseur du
// 2026-09-27 ; lot J6-bis du 2026-09-28 ; lot R3 du 2026-09-29) : flock-position, world-object
// i0, `ti=38 i18`, flock-destination, tacmap-poiicon, player-desired-respawn-location,
// tacmap-displayasset, tacmap-areaofinterest, tacmap-cooptetherarea, crew-order et la branche
// precHigh = 1 de l i0 absolu du bipede gardent leur ancien lecteur dans
// `lecteur_position_exceptions.go`, parce que la lecture du jeu y fait baisser la fermeture des
// bobines. Les chiffres et le critere de retrait sont dans ce fichier et dans la table des sites.

// Les immediats de niveau des sites d appel portes (releve J6.1, 2026-09-27, §1 et §6).
const (
	// niveauPosition est l immediat 0x10 : `MOV R9D,0x10` en 1406d008a, 140ee7288, 140f04dd5,
	// 140f04fe5, 140f05018, 140fb8b33, 14226a6b8 ; `LEA R8D,[R9+0x10]` en 14076e2bc ; et 0x10
	// passe a `FUN_14076e494` / `FUN_1424e0e38` par tous les autres sites portes.
	niveauPosition = 0x10
	// niveauTransformDActif est l immediat 0x1E de `FUN_142ed9530` (CALL 142ed9556), appele cinq
	// fois par le lecteur d `asset-transform-component` (`FUN_142ed3c64`).
	niveauTransformDActif = 0x1e
)

// bornePrecHaut est la demi-etendue de `DAT_143b8c6d0` (+/-100 sur les trois axes, read_memory du
// 2026-09-27) : les bornes que `FUN_14076e420` passe a `FUN_141f85880` quand precHigh vaut 1.
const bornePrecHaut = float32(100)

// positionQuantifiee est ce qu une lecture a rendu. Une seule des trois formes :
//
//	brute     la garde de pleine precision a lu R(96) : pas de quanta
//	precHaut  `FUN_141f85880` : trois axes sur +/-100, sans porte ni index (idx = -1)
//	sinon     `FUN_14076e524` : idx (-1 = porte posee), trois quanta et leurs largeurs
type positionQuantifiee struct {
	idx      int
	q        [3]uint64
	w        [3]uint
	brute    bool
	precHaut bool
}

// tablesDePosition est ce que `FUN_14076e524` lit HORS du flux : la largeur d index de plage
// (`DAT_144632be0`) et la table PAR INDEX de la plage cataloguee.
type tablesDePosition struct {
	// indexW est `DAT_144632be0`.
	indexW uint
	// axesCarte est la ligne 0x10 de la table par index — les largeurs du catalogue (controle
	// 79/79 par `TestLaLoiRendLesLargeursDuCatalogue`).
	axesCarte [3]uint
	// bornesCarte sont les bornes de la plage, pour les lignes d un autre niveau que 0x10.
	bornesCarte [3][2]float32
	// indexLisible est faux quand la table par index est inconnue (entree de catalogue absente) :
	// un index lu ne peut alors pas l etre, et la lecture s arrete sur la porte.
	indexLisible bool
}

// tablesDuProfil rend les tables que porte le profil du lecteur — celles de la carte du match
// quand l appelant les a installees (`replay.installWorldObjectPrecision`).
func (b *Lecteur) tablesDuProfil() tablesDePosition {
	d := b.worldObjectPrecision()
	return tablesDePosition{indexW: d.IndexW, axesCarte: d.AxisW,
		bornesCarte: bornesDeLaPlage(b.worldPositionRange()), indexLisible: true}
}

// bornesDeLaPlage convertit une plage en bornes {min, max} par axe.
func bornesDeLaPlage(r profile.Vec3Range) [3][2]float32 {
	var out [3][2]float32
	for axe := 0; axe < 3; axe++ {
		out[axe] = [2]float32{r[axe].Min, r[axe].Max}
	}
	return out
}

// largeursDeLaLigne rend la ligne `niveau` de la table que l index choisit.
func largeursDeLaLigne(t tablesDePosition, idx, niveau int) [3]uint {
	switch {
	case idx < 0:
		return profile.LargeursAxeParDefautDuBuild(niveau)
	case niveau == profile.NiveauPositionDObjet:
		return t.axesCarte
	default:
		return profile.LargeursAxeDuNiveau(t.bornesCarte, niveau)
	}
}

// lireE524Sur porte `FUN_14076e524` sur des tables donnees. ok est faux quand la porte ouvre un
// index que `t` ne sait pas lire : le curseur s arrete alors apres la porte.
func lireE524Sur(br *Lecteur, niveau int, t tablesDePosition) (positionQuantifiee, bool) {
	pos := positionQuantifiee{idx: -1}
	if !br.ReadBit() { // FUN_1406cf008 : porte a 0 -> l index est present
		if !t.indexLisible {
			return pos, false
		}
		pos.idx = int(br.ReadBits(t.indexW)) // DAT_144632be0
	}
	br.obs.compterIndexAbsolu(pos.idx)
	pos.w = largeursDeLaLigne(t, pos.idx, niveau)
	for axe := range pos.q {
		pos.q[axe] = br.ReadBits(pos.w[axe]) // FUN_140cc5128 axe
	}
	return pos, true
}

// lireE524 porte `FUN_14076e524(NIVEAU)` sur les tables du profil : la lecture NUE.
func lireE524(br *Lecteur, niveau int) positionQuantifiee {
	pos, _ := lireE524Sur(br, niveau, br.tablesDuProfil()) // les tables du profil lisent tout index
	return pos
}

// lireE494Sur porte `FUN_14076e494(NIVEAU, p6 = 0)` sur des tables donnees : la garde de pleine
// precision (`FUN_14076f91c`, 0 bit), puis `FUN_14076e524`.
func lireE494Sur(br *Lecteur, niveau int, t tablesDePosition) (positionQuantifiee, bool) {
	if fullPrecisionGate(br) {
		br.ReadBits(rawVec3Bits) // FUN_1411b259c -> FUN_1406d676c(..., 0x60)
		return positionQuantifiee{idx: -1, brute: true}, true
	}
	return lireE524Sur(br, niveau, t)
}

// lireE494 porte `FUN_14076e494(NIVEAU, p6 = 0)` sur les tables du profil.
func lireE494(br *Lecteur, niveau int) positionQuantifiee {
	pos, _ := lireE494Sur(br, niveau, br.tablesDuProfil()) // les tables du profil lisent tout index
	return pos
}

// lireE420 porte `FUN_14076e420(NIVEAU)` : R(1) precHigh, puis `FUN_14076e494` avec les bornes
// +/-100 quand il vaut 1. Rend precHigh — `FUN_14076e29c` le passe a la queue de poignee.
//
//nolint:unparam // le niveau est l IMMEDIAT DU SITE, lu par la table des sites (lecteur_position_ratchet_test.go) : aujourd hui 0x10 a tous les sites portes de FUN_14076e420, mais c est au site de le dire (2026-09-27, lot J6.3).
func lireE420(br *Lecteur, niveau int) (bool, positionQuantifiee) {
	precHigh := br.ReadBit() // FUN_1406cf008
	if fullPrecisionGate(br) {
		br.ReadBits(rawVec3Bits) // FUN_1411b259c
		return precHigh, positionQuantifiee{idx: -1, brute: true}
	}
	if !precHigh {
		return false, lireE524(br, niveau)
	}
	return true, lireF85880(br, niveau)
}

// lireF85880 porte `FUN_141f85880(dst, lecteur, &DAT_143b8c6d0, NIVEAU)` : la loi au NIVEAU sur les
// bornes +/-100 (`FUN_140be9b88`), puis trois axes (`FUN_1424cbed4` -> `FUN_140cc5128`). Au niveau
// 0x10 : 3 x 14 = 42 bits — et non « le vecteur par defaut, 0 bit », que le depot lisait jusqu ici.
func lireF85880(br *Lecteur, niveau int) positionQuantifiee {
	b := [2]float32{-bornePrecHaut, bornePrecHaut}
	pos := positionQuantifiee{idx: -1, precHaut: true,
		w: profile.LargeursAxeDuNiveau([3][2]float32{b, b, b}, niveau)}
	for axe := range pos.q {
		pos.q[axe] = br.ReadBits(pos.w[axe])
	}
	return pos
}

// semerPositionAbsolue emet la position d un chemin absolu d i0 comme graine d accumulation — SI
// ET SEULEMENT SI l index designe la plage CATALOGUEE (`Region`, correctif D1 (3.4)) : une autre
// plage n a pas de bornes connues, la porte posee designe la boite monde du build, et les formes
// brute et precHaut ne portent pas de coordonnee de carte. Dans chacun de ces cas la coordonnee
// serait fausse en silence ; l histogramme [Observation.IndexAbsolus] les compte.
func semerPositionAbsolue(br *Lecteur, pos positionQuantifiee, kind PosKind) {
	if pos.brute || pos.precHaut || pos.idx < 0 || uint32(pos.idx) != br.worldObjectPrecision().Region {
		return
	}
	var v [3]float32
	for axe := range v {
		v[axe] = dequantWorldAxis(br, pos.idx, pos.q[axe], pos.w[axe], axe)
	}
	br.seedAbsolute(kind, v)
}
