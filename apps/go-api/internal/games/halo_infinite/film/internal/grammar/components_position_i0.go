package grammar

import "levelup/go-api/internal/games/halo_infinite/film/internal/profile"

// components_position_i0.go — LE DESERIALISEUR DE POSITION i0 (FUN_1406cfe44) ET SES TROIS
// CHEMINS DE CHARGE UTILE : delta predit, absolu quantifie, copie pleine precision ; plus la
// queue de poignee et les lecteurs de vec3 quantifies qu ils partagent.
//
// Sorti de `components_movement.go` par deplacement pur au lot 2.7 (scission des fichiers de
// plus de 500 lignes) : aucune ligne de logique n'a change. La frontiere est celle du fichier
// d origine : les composants de mouvement A LARGEUR FIXE (i1 vitesse, i3 rotation) et les
// drapeaux de precision restent la-bas ; i0, dont les largeurs viennent du descripteur de
// precision, vient ici avec tout ce qu il appelle.

// consumeObjectPositionDynamicPrecisionD (i0) mirrors FUN_1406cfe44, bit-exactly.
//
//	bUsePred = R(1) ; bDelta = R(1)  (header)
//
// Three mutually-exclusive payload paths + the shared bHandle tail. AxisW/IndexW
// from the runtime profile.PrecisionDescriptor (pd); `Lecteur.fullPrecision` = the
// FUN_14076f91c runtime gate (received, not read from the stream).
func consumeObjectPositionDynamicPrecisionD(br *Lecteur, pd profile.PrecisionDescriptor) {
	br.cap.startBit = br.BitPos()  // bit d entree (== StartBit du composant) pour l attribution
	br.cap.slot = br.cap.accumSlot // slot du record courant (attribution multi-entites)
	if br.calibratedSkip() {
		skipCalibratedPosition(br)
		return
	}
	bUsePred := br.ReadBit() // FUN_1406cfe44 R(1)
	bDelta := br.ReadBit()   // FUN_1406cfe44 R(1)

	// bUsePred==1: KEEP-BASELINE. bHandle, then a raw vec3 copy, then the tail.
	if bUsePred {
		bHandle := br.ReadBit()                    // FUN_1406cf008 R(1)
		readRawVec3(br)                            // FUN_1406d676c(...,0x60) = R(96) : AVANCE le curseur
		br.keepBaseline()                          // réutilisation baseline : ré-émet prev, JAMAIS les 96 bits bruts
		consumePositionHandleTail(br, bHandle, pd) // same tail whether bDelta 0 or 1
		return
	}

	// bUsePred==0, bDelta==0: ABSOLUTE -> FUN_14076e524, then tail (no fresh handle bit).
	if !bDelta {
		if br.p.Grammaire.GrammaireEcrivainI0 {
			// Grammaire de l'ÉCRIVAIN d'état complet (FUN_14320696c / FUN_14076e420) : le
			// bit h ne supprime rien, il garde la QUEUE, et le champ de 2 bits vient APRÈS.
			h := br.ReadBit()
			if fullPrecisionGate(br) {
				br.ReadBits(rawVec3Bits) // FUN_1407eb61c sous DAT_144e61ea0 : vec3 BRUT
			} else {
				consumeAbsolutePayload(br)
			}
			consumePositionHandleTail(br, h, pd)
			br.ReadBits(2) // FUN_14076e304, EN DERNIER
			return
		}
		consumeAbsoluteWithGate(br, pd)
		consumePositionHandleTail(br, false, pd)
		return
	}

	// bUsePred==0, bDelta==1: PREDICTED-DELTA (FUN_1406cfe44 else-branch).
	// CORRECTED grammar (direct Ghidra decompile of FUN_1406cfe44): exactly ONE
	// control bit (predFlag) precedes the predicted reader. The earlier port read a
	// spurious 2nd "prec-select" bit here AND gated the tail on it — both wrong.
	//   predFlag==0 (dominant): read FUN_14076f3ec (== consumePredictedDelta).
	//   predFlag==1 (rare): FUN_140f7ea14 special path, width unmodeled.
	// The handle tail is gated by the RUNTIME descriptor field bVar16 = (precIndex !=
	// -1), NOT a bitstream bit (`Lecteur.deltaHasHandleTail`; default false = the
	// dominant precIndex==-1 case). FUN_14076f91c full-precision gate =
	// `fullPrecisionGate` (DAT_144e61ea0 OU DAT_145121140). Both runtime gates are confirmed
	// via the CE delta capture.
	predFlag := br.ReadBit() // FUN_1406cfe44 inline R(1)
	if !predFlag {
		// LAB_1406cff18 : la garde de contexte est ICI, sur ce seul chemin (relu le
		// 2026-08-17, lot R7-c — elle ne couvre PAS la branche predFlag==1, qui porte la
		// sienne dans FUN_14076e4ec).
		if fullPrecisionGate(br) {
			readRawVec3(br) // FUN_1406d676c(...,0x60) = R(96) : AVANCE le curseur (keep, pas une coord)
			br.keepBaseline()
		} else {
			consumePredictedDelta(br, pd) // FUN_14076f3ec
		}
	} else {
		consumePredictedAbsolute(br) // FUN_140f7ea14
	}
	if br.deltaHasHandleTail() { // runtime bVar16 = (precIndex != -1)
		if br.ReadBit() { // FUN_1406cf008 -> FUN_1408f0ac4 handle resolve
			consume1408f0ac4(br, 0) // FUN_1408f0ac4(...,0)
		}
		if br.ReadBit() { // FUN_1406cf008 region present
			if br.ReadBit() { // FUN_1406cf008 region ext
				br.ReadBits(11) // R(0xb) region word
			}
		}
	}
}

// skipCalibratedPosition est le BANC DE CALIBRATION du chemin i0 : il saute le total de bits
// mesure sur la capture du jeu au lieu de derouler la grammaire. Sorti du corps de
// `consumeObjectPositionDynamicPrecisionD` au lot 2.7 (2026-09-16) par deplacement pur — pas
// une ligne ne change, le bloc est recopie desindente d une tabulation. Il n a rien a faire
// dans le deserialiseur : c est un harnais, garde par un drapeau, et le sortir rend au
// deserialiseur la seule grammaire.
func skipCalibratedPosition(br *Lecteur) {
	// CALIBRATION INTELLIGENTE (largeurs CE constantes par chemin, Cliffhanger) : le 1er bit
	// bUsePred discrimine keep-baseline ragdoll (101 bits) vs absolu/predicted (47 bits). Les
	// largeurs d'axe runtime (pd.AxisW) n'étant pas sourcées statiquement, on saute au total
	// CE mesuré. Map-spécifique ; harness de validation, pas chemin de prod général.
	start := br.BitPos()
	w := 47
	if br.ReadBit() { // bUsePred
		w = 101
	}
	if d := start + w - br.BitPos(); d > 0 {
		br.Skip(d)
	}
}

// consumePredictedAbsolute porte FUN_140f7ea14 -> FUN_14076e4ec -> FUN_14076e524 : la branche
// `predFlag == 1` de FUN_1406cfe44, qui lit une POSITION ABSOLUE quantifiee la ou le chemin
// dominant lit un delta predit.
//
// SORTIE DU CORPS DE `consumeObjectPositionDynamicPrecisionD` au lot 2.7 (2026-09-16), par
// deplacement pur : pas une ligne de logique ne change, le bloc est recopie desindente d une
// tabulation. La frontiere n est pas un quota de lignes, c est celle du JEU — ce bloc est
// exactement une fonction de l executable, comme `consumePredictedDelta` (FUN_14076f3ec),
// `consumeAbsoluteWithGate` et `consumePositionHandleTail` le sont deja pour les leurs. Le
// fichier gardait donc une fonction du moteur inline au milieu d une autre.
func consumePredictedAbsolute(br *Lecteur) {
	// predFlag==1: FUN_140f7ea14 -> FUN_14076e4ec -> FUN_14076e524 = lecteur de POSITION
	// ABSOLUE quantisée. Ancien port : "width unmodeled" (0 bit) = LE bug i0 delta (lisait 3
	// bits au lieu de 47, mesuré par capture CE). Grammaire (FUN_140f7ea14 + FUN_14076e524) :
	//   R(1) cVar1 ; si cVar1==0 -> R(1) index-sel [si 0 -> R(IndexW)] + 3×R(AxisW).
	// CE delta capture (Cliffhanger) : total i0 = 47 => 3(en-tête) +1(cVar1) +1(index-sel)
	// +3×14(axes) = 47, index absent. Même pd.AxisW que le chemin absolu keyframe (Hydra OK).
	//
	// La GARDE DE CONTEXTE vit dans FUN_14076e4ec, pas dans FUN_140f7ea14 : le R(1) cVar1
	// est lu D'ABORD dans tous les cas, puis `FUN_14076e4ec(dst, br, 0x10, cVar1 ? &DAT_143b8c6d0 : 0)`
	// choisit R(96) brut (pleine précision), le vecteur par défaut (cVar1 != 0, 0 bit) ou le
	// lecteur quantifié. Ajouté le 2026-08-17 (lot R7-c) : ce site ignorait la garde.
	cVar1 := br.ReadBit() // FUN_140f7ea14 cVar1 (FUN_1406cf008)
	if fullPrecisionGate(br) {
		br.ReadBits(rawVec3Bits) // FUN_14076e4ec -> FUN_1411b259c
	} else if !cVar1 { // 0 -> lit la position absolue quantifiée
		pidx := -1
		if !br.ReadBit() { // FUN_14076e524 index-sel ; 0 -> lit l'index
			// MEME LARGEUR D'INDEX QUE PARTOUT AILLEURS (`DAT_144632be0`, lot 3.4.1) :
			// c'est une donnée de la CARTE, pas du descripteur de l'appelant.
			pidx = int(br.ReadBits(br.worldObjectPrecision().IndexW))
		}
		var v [3]float32
		for i := 0; i < 3; i++ {
			w := absAxisWFor(br, pidx, i)                           // table DEFAUT ou table PAR INDEX
			v[i] = dequantWorldAxis(br, pidx, br.ReadBits(w), w, i) // FUN_140cc5128 axe i
		}
		// MEME REGLE QUE `consumeAbsolutePayload` (correctif D1 (3.4)) : seule la plage
		// CATALOGUEE porte des bornes connues. `pidx == -1` est la boite monde du build, un
		// autre index une plage dont on n a pas l AABB — dans les deux cas la coordonnee
		// serait fausse en silence.
		if pidx >= 0 && uint32(pidx) == br.worldObjectPrecision().Region {
			br.seedAbsolute(PosKindAbsolute, v) // predFlag==1 = absolue = seed d'accumulation
		}
	}
}

// consumePredictedDelta mirrors FUN_14076f3ec -> FUN_14076f550 (taken when
// FUN_14076f91c is false): a leading present flag, then EITHER 3 fixed signed 8-bit
// deltas (dominant) OR 3 axis-width words, OR an absolute fallback.
func consumePredictedDelta(br *Lecteur, pd profile.PrecisionDescriptor) {
	if br.ReadBit() { // FUN_14076f3ec R(1); set => predicted absent -> absolute fallback
		br.cap.viaRepli = true
		consumeAbsoluteWithGate(br, pd)
		br.cap.viaRepli = false
		return
	}
	if br.ReadBit() { // FUN_14076f550 mask; set => fixed signed 8-bit deltas (dominant)
		// 3 signed 8-bit deltas = un NOMBRE DE CRANS signé par axe. Le pas physique est
		// DeltaQuantum (propre au delta, PAS la range absolue/2^axisW qui donnait ~18 u = faux).
		var d [3]float32
		for i := 0; i < 3; i++ {
			n := signed8(br.ReadBits(8))
			d[i] = float32(n) * br.deltaQuantum()
		}
		br.applyDelta(PosKindDelta8, d)
		return
	}
	// mask clear => FUN_1424cbed4 -> FUN_140cc5128 : delta axis-width. C'est un delta SIGNÉ
	// centré-zéro (pas une pseudo-absolue qui soustrairait Min) : q ∈ [0,2^AxisW) est recentré
	// sur [-2^(AxisW-1), 2^(AxisW-1)) et mis à l'échelle DeltaQuantum (range delta propre =
	// DeltaQuantum*2^AxisW). AxisW = pd.AxisW[i] (6 par défaut ; ambiguïté 6 vs 14 sweepable).
	var d [3]float32
	for i := 0; i < 3; i++ {
		w := deltaAxisW(br, pd, i)
		q := br.ReadBits(w)
		half := float32(uint64(1) << (w - 1))
		d[i] = (float32(q) - half) * br.deltaQuantum() // multiple ENTIER de Q, centré (q==half -> 0)
	}
	br.applyDelta(PosKindDeltaAxis, d)
}

// consumeQuantVec3WithGate porte l'epine de FUN_14076e524 (lecteur de vec3 quantifie
// generique, partage par tous les composants "transform") :
//
//	R(1) gate (FUN_1406cf008) ; si gate==0 -> R(DAT_144632be0 = 1) index de table ;
//	puis 3 x R(axisW) (FUN_140cc5128), INCONDITIONNELLEMENT (les deux branches du gate
//	rejoignent LAB_14076e5f3).
//
// axisW vient de la table DAT_1445cc9e0 indexee par le niveau de precision : largeur = 6+L
// (dump memoire ce_prec_widths_1445cc9e0.bin : L0->6, L7->13, L8->14, L9->15...).
//
// PIEGE DE NOMMAGE, statue le 2026-08-01 (lot C) : malgre son suffixe, cette fonction a UNE
// PORTE DE MOINS que `consumeQuantVec3` (plus bas dans ce fichier), qui commence par la porte
// `precHigh` a sortie immediate (0 bit). Les deux ne sont donc PAS interchangeables et les
// fusionner changerait un compte de bits. Chacune a ses appelants vivants : celle-ci pour le
// flock (components_flock.go), l'autre pour les composants a vec3 quantifie du dispatch.
func consumeQuantVec3WithGate(br *Lecteur, axisW uint) {
	if !br.ReadBit() { // FUN_1406cf008 ; bit==0 -> l'index est present
		br.ReadBits(1) // DAT_144632be0 = 1
	}
	br.ReadBits(axisW)
	br.ReadBits(axisW)
	br.ReadBits(axisW)
}

// DeltaAxisWidth = largeur d'axe (bits) du chemin DELTA axis-width de i0
// (FUN_1424cbed4 -> FUN_140cc5128). Elle ne vient PAS du niveau de précision du
// registre chunk_00 (i0 y est L0) mais du descripteur de précision RUNTIME installé
// au chargement de map (FUN_140be9a14 -> DAT_1445cc9e0), qui lit 0 statiquement dans
// l'.exe — même limitation que le descripteur de traversée.
//
// SOURCE : la table DAT_1445cc9e0 dumpée en mémoire vive
// (.ai/V7.5/dumps/ce_prec_widths_1445cc9e0.bin) est un tableau [niveau][3 axes] de
// largeurs = 6+L : L0->6/6/6, L7->13/13/13, **L8->14/14/14**, L9->15/15/15.
//
// MESURE (non circulaire, cmd/tmp_deadstate mode `solvechain`, film 000d5950) : sur les
// 15 529 records du masque {i0,i1,i21,i25} dont l'oracle de POSITION Rosette donne la
// longueur vraie (113 bits), la seule largeur de i0 pour laquelle les desers PORTÉS de i1
// et de i21 consomment exactement leurs largeurs vraies (31 et 25 bits, obtenues par
// différence de masques) est **47 bits = 2+1+1+1+3x14** : i1 tombe juste sur 100.0% des
// records et i21 sur 100.0%. Toute autre largeur retombe à 0-52%. 47 recoupe en outre la
// mesure Cheat Engine indépendante de §7ter.27 (i0=47, 134767/134767).
// C'ÉTAIT UNE VARIABLE DE PAQUET JUSQU'AU LOT 2.2.b : la largeur vit dans le PROFIL que le
// lecteur porte (`Movement.DeltaAxisWidth`).
//
// deltaAxisW retourne la largeur d'axe du chemin delta (celle du profil si > 0, sinon pd).
func deltaAxisW(br *Lecteur, pd profile.PrecisionDescriptor, i int) uint {
	if w := br.p.Mouvement.DeltaAxisWidth; w > 0 {
		return w
	}
	return pd.AxisW[i]
}

// consumeAbsoluteWithGate mirrors the absolute-reader spine (prec-select, the
// FUN_14076f91c runtime gate, then FUN_14076e524 index+vec3).
func consumeAbsoluteWithGate(br *Lecteur, pd profile.PrecisionDescriptor) {
	precHigh := br.ReadBit() // FUN_1406cf008
	if fullPrecisionGate(br) {
		// FUN_1411b259c = FUN_1406d676c(br, br, dst, 0x60) : R(96) BRUT, pas 0 bit.
		// L'ancien commentaire (« NaN/keep fill, 0 payload bits ») lisait le RÉSULTAT
		// (le vecteur écrit est un NaN de conservation) et non le CURSEUR — corrigé
		// sur pièce le 2026-08-17 (lot R7-c).
		br.ReadBits(rawVec3Bits)
		return
	}
	if precHigh {
		return // default vector (FUN_141f85880), 0 payload bits
	}
	consumeAbsolutePayload(br)
	// Champ « fini » de 2 bits — FUN_14076e304, appelé en LAB_1406cffd7 sous un prédicat
	// (FUN_140492128) qui ne consomme AUCUN bit : il est donc lu systématiquement.
	//
	// AJOUTÉ le 2026-07-27. Le chemin world-object le lisait déjà (traverse.go, `[2 finite]`
	// de son total de 45 bits) ; le chemin dynamic-precision du bipède ne l'a jamais lu. Les
	// deux portent pourtant le MÊME lecteur absolu — c'est la double implémentation qui les
	// a laissés diverger. Sans ces 2 bits le compte tombait à 45 au lieu des 47 mesurés par
	// la capture CE (100 % de 154 158 dispatches, aucune variance).
	//
	// PLACE DU CHAMP : ici il est lu AVANT la queue de handle, qui est de toute façon éteinte
	// sur ce chemin (`consumePositionHandleTail(br, false, ...)`). L'écrivain du jeu le pose
	// APRÈS la queue — l'ordre n'est donc observable que sous `keyframeWriterI0Grammar`.
	br.ReadBits(2)
}

// consumeAbsolutePayload lit la CHARGE UTILE du lecteur absolu (FUN_14076e494) : le sélecteur
// d'index de plage, son mot éventuel, puis les trois axes quantisés. Il ne lit NI le bit de
// tête, NI le champ de 2 bits de queue — les deux appelants ne les posent pas au même endroit
// (cf. `keyframeWriterI0Grammar`).
// CORRECTIF D1 (3.4), 2026-09-17 — « index 1 / no-index = ±20000 » ÉTAIT FAUX, ET LE FILTRE
// `if idx != 0 { return }` AVEC LUI.
//
// Le désassemblage de `FUN_14076e524` (note 3.4 §1.3) ne donne les bornes `±20000`
// (`DAT_1445cc9c8`, copie de `DAT_143b8c6b8`) QUE pour `index == -1`, c'est-à-dire quand le bit
// de porte est posé ; tout `index >= 0` adresse `DAT_14462cbe0 + index*0x18`, une plage RÉELLE
// de la carte. Le catalogue le disait déjà de son côté : `live fire` porte `region = 1` et
// `regionIndexBits = 2` (4 régions déclarées par `ds/globals/common`, l'arène est la 1). Sur
// cette carte le filtre gardait donc exactement ce qu'il fallait jeter et jetait ce qu'il
// fallait garder — 59 376 des 59 377 records i0 de ses deux films portent l'index 01.
//
// LA RÈGLE EST DONC CELLE DU CATALOGUE, PAS UN ZÉRO EN DUR : la position n'est émise que si
// l'index lu désigne la plage CATALOGUÉE (`Region`, nulle sur 78 cartes sur 79). Un index
// d'une autre plage n'a pas de bornes connues — le déquantifier avec celles de l'arène
// produirait une coordonnée fausse silencieuse — et `index == -1` désigne la boîte monde du
// build, pas une position de carte. Les deux se comptent à l'histogramme
// [Observation.IndexAbsolus] : jamais un zéro muet.
func consumeAbsolutePayload(br *Lecteur) {
	// L'index sélectionne la plage de déquantification (`DAT_14462cbe0 + index*0x18`) ; la
	// porte posée (pas d'index) sélectionne la boîte monde du build (`DAT_1445cc9c8`).
	//
	// SA LARGEUR EST CELLE DE LA CARTE, PAS CELLE DU DESCRIPTEUR DE L'APPELANT (lot 3.4.1) :
	// `DAT_144632be0` est UNE valeur, posée une fois au chargement de la carte
	// (`ceilLog2(nb de plages)`, 2 bits sur Live Fire) et lue par TOUS les chemins de
	// `FUN_14076e524` — le chemin world-object la lisait déjà ainsi
	// (`consumeSimStateHandleTail`, `dispatch_object.go`), le chemin du bipède prenait celle
	// du descripteur de traversée, que la calibration de `killsource` balayait à l'aveugle.
	idx := -1
	if !br.ReadBit() {
		idx = int(br.ReadBits(br.worldObjectPrecision().IndexW)) // DAT_144632be0
	}
	var v [3]float32
	for i := 0; i < 3; i++ {
		// LARGEUR ET BORNES SUIVENT LE MÊME INDEX, et c'est le point : elles sortent de la
		// même AABB (`FUN_140be9b88` dérive les unes des autres). Les dissocier était la
		// forme que prenait le défaut corrigé ici.
		w := absAxisWFor(br, idx, i)                           // table DEFAUT ou table PAR INDEX
		v[i] = dequantWorldAxis(br, idx, br.ReadBits(w), w, i) // FUN_140cc5128 axis i
	}
	if idx < 0 || uint32(idx) != br.worldObjectPrecision().Region {
		return
	}
	kind := PosKindAbsolute
	if br.cap.viaRepli {
		kind = PosKindAbsFallback
	}
	br.seedAbsolute(kind, v) // absolue in-map = seed d'accumulation pour les deltas ultérieurs
}

// consumePositionHandleTail mirrors the bHandle-gated tail shared by FUN_1406cfe44
// (inline) and FUN_14076e3e4: if bHandle clear the field is 0xFFFFFFFF (0 bits);
// else a handle-resolve word (R(IndexW)+R(2)) and an optional 11-bit region word.
// FUN_1406cb0cc reads 0 bits (runtime validity predicate only).
func consumePositionHandleTail(br *Lecteur, bHandle bool, pd profile.PrecisionDescriptor) {
	if !bHandle {
		return // field = 0xFFFFFFFF, 0 bits
	}
	if br.ReadBit() { // handleSel == 1 -> FUN_1408f0ac4
		if br.ReadBit() { // FUN_1408f0ac4 R(1) present
			br.ReadBits(pd.IndexW) // FUN_1406d3140 index word (bitlen(handleCount))
			br.ReadBits(2)         // FUN_1406d3140 trailing fixed 2-bit word
		}
	}
	if br.ReadBit() { // FUN_1406cf008 regionPresent
		if br.ReadBit() { // FUN_1406cf008 regionExt
			br.ReadBits(11) // R(0xb) region word
		}
	}
}

// quantAxisWidth retourne la largeur en bits d'un axe quantifié dans la TABLE PAR DÉFAUT
// du moteur — celle bâtie sur la BOÎTE MONDE (DAT_143b8c6b8, range ≈ 40000 unités par axe),
// et non sur une région de compression.
//
//	W(L) = min(26, ceilLog2(ceil(rangeMonde / (2·q(L)))))   q(L) = 2^(16-L)/120
//
// se réduit à min(26, 6+L) pour L ∈ [0,20] (FUN_140be9b88 ; forme fermée vérifiée terme à
// terme par TestQuantAxisWidthFormula contre la formule complète, pas par tautologie).
//
// PÉRIMÈTRE — ce que cette largeur N'EST PAS. Elle ne s'applique PAS à la position d'objet
// répliquée (composant i0). Celle-ci passe par la table PAR RÉGION, bâtie sur l'AABB du BSP
// de la carte : W = min(26, ceilLog2(ceil(60·extent))) au niveau 16 câblé au site d'appel.
// Cette largeur-là est propre à la carte (12/12/12 sur Streets, 18/19/17 sur Highpower...) ;
// elle est lue dans le film par DetectI0Layout et recoupée aux bornes du module de la carte
// (cf. i0_layout.go, map_bounds.go, cmd/mapquant-build) — confrontation réussie sur 13
// cartes / 13, largeurs identiques sur les 3 axes.
//
// POURQUOI ELLE RESTE VALABLE ICI. Les seuls appelants sont des composants NON-POSITION
// (crew-order, tacmap-poiicon/offset, flock-destination, desired-respawn-location) : leur
// vec3 n'est jamais émis comme coordonnée, il n'est consommé que pour rester aligné sur le
// bitstream. Deux réserves explicites :
//   - la source de L (le niveau de l'entrée du registre chunk_00) reste une PISTE pour ces
//     composants : le registre est bit-à-bit identique d'un film à l'autre, il ne peut donc
//     pas porter d'information par carte ; pour un vec3 borné par la boîte monde c'est
//     cohérent, mais aucun de ces composants n'a été validé au bit près ;
//   - si l'un d'eux s'avérait borné par une région, il faudrait la table par région, pas
//     celle-ci.
//
// Cf .ai/V7.5/film_re/HANDOFF_KEYFRAME_LIVE_CAPTURE.md « LES LARGEURS SONT OFFLINE ».
func quantAxisWidth(level uint) uint {
	if w := 6 + level; w < 26 {
		return w
	}
	return 26
}

// consumeQuantVec3 lit un vecteur 3D quantifié (FUN_14076e524, le coeur quantifié de
// FUN_14076e494) SANS capture : gate precHigh (1 -> vecteur défaut, 0 bit) ; sinon
// index-gate (0 -> R(1)) + 3×R(axisW). Mêmes gates que consumeAbsoluteWithGate, mais
// largeur d'axe PARAMÉTRÉE (PISTE 1) : axisW = 6+level, level = le niveau de l'entrée du
// registre (`Archetype.Levels`, lu en `entrée + 0x100`). La largeur d'index reste en dur à 1
// (DAT_144632be0), comme partout ailleurs dans ce paquet : aucun appelant n'en a jamais passé
// une autre. Décode pur (pas d'emitPos) pour les composants non-position porteurs d'un vec3
// quantifié (crew-order, tacmap-offset, desired-respawn-location, ...).
func consumeQuantVec3(br *Lecteur, axisW uint) {
	_, _, _, _ = consumeQuantVec3Values(br, axisW)
}

// consumeQuantVec3Values est le MÊME lecteur, qui rend ses trois quanta BRUTS. `ok` est faux
// quand la garde `precHigh` a valu 1 : le vecteur par défaut, zéro bit de charge utile — ce
// qui n'est pas un vecteur nul, et les confondre fabriquerait une position à l'origine.
//
// UNE SEULE COPIE DE LA GRAMMAIRE, et c'est la règle : `consumeQuantVec3` délègue ici au lieu
// de relire les mêmes largeurs à côté. Deux lecteurs du même champ divergent le jour où l'un
// des deux est corrigé.
func consumeQuantVec3Values(br *Lecteur, axisW uint) (x, y, z uint64, ok bool) {
	if br.ReadBit() { // precHigh == 1 -> vecteur défaut, 0 bit
		return 0, 0, 0, false
	}
	if !br.ReadBit() { // index-present select ; 0 -> lit l'index
		br.ReadBits(1) // DAT_144632be0 = 1
	}
	x = br.ReadBits(axisW)
	y = br.ReadBits(axisW)
	z = br.ReadBits(axisW)
	return x, y, z, true
}
