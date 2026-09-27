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
			// bit h est precHigh, il garde la QUEUE, et le champ de 2 bits vient APRÈS.
			// h = 1 lit `FUN_141f85880` (trois axes sur +/-100), pas la plage de la carte
			// (lot J6.3, relevé du 2026-09-27 : `FUN_14076e29c` -> `FUN_14076e420(0x10)`).
			h, pos := lireE420(br, niveauPosition)
			semerPositionAbsolue(br, pos, PosKindAbsolute)
			consumePositionHandleTail(br, h, pd)
			br.ReadBits(2) // FUN_14076e304, EN DERNIER
			return
		}
		consumeAbsoluteWithGate(br)
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
	// +3×14(axes) = 47, index absent. (Lot J6.3 : la décomposition date de la largeur uniforme de
	// 14 bits, retirée au lot 3.4.1 ; porte posée, la table DÉFAUT lit 22/22/22 au niveau 0x10.)
	//
	// La GARDE DE CONTEXTE vit dans FUN_14076e4ec, pas dans FUN_140f7ea14 : le R(1) cVar1
	// est lu D'ABORD dans tous les cas, puis `FUN_14076e4ec(dst, br, 0x10, cVar1 ? &DAT_143b8c6d0 : 0)`
	// choisit R(96) brut (pleine précision), `FUN_141f85880` (cVar1 != 0) ou le lecteur
	// quantifié. Ajouté le 2026-08-17 (lot R7-c) : ce site ignorait la garde.
	//
	// C'EST LA FORME DE `FUN_14076e420` (lot J6.3) : un R(1), puis l'enveloppe à bornes. `cVar1 = 1`
	// lit `FUN_141f85880` — trois axes sur +/-100 au niveau 0x10, 42 bits —, et non « le vecteur
	// par défaut, 0 bit » que ce site lisait jusqu'au 2026-09-27.
	_, pos := lireE420(br, niveauPosition)         // FUN_140f7ea14 -> FUN_14076e4ec(0x10), CALL 140f7ea5c
	semerPositionAbsolue(br, pos, PosKindAbsolute) // predFlag==1 = absolue = seed d'accumulation
}

// consumePredictedDelta mirrors FUN_14076f3ec -> FUN_14076f550 (taken when
// FUN_14076f91c is false): a leading present flag, then EITHER 3 fixed signed 8-bit
// deltas (dominant) OR 3 axis-width words, OR an absolute fallback.
func consumePredictedDelta(br *Lecteur, pd profile.PrecisionDescriptor) {
	if br.ReadBit() { // FUN_14076f3ec R(1); set => predicted absent -> absolute fallback
		// LE REPLI EST `FUN_14076e524(0x10)` NU (CALL 14226a6c7, relevé du 2026-09-27) : ni bit
		// precHigh, ni garde — la garde est celle de l'appelant (`LAB_1406cff18`). Le portage
		// passait par `consumeAbsoluteWithGate` et lisait un precHigh de trop (lot J6.3).
		br.cap.viaRepli = true
		semerPositionAbsolue(br, lireE524(br, niveauPosition), kindDuCheminAbsolu(br))
		br.cap.viaRepli = false
		// Le R(2) de `FUN_14076e304`, lu par `FUN_1406cfe44` en `LAB_1406cffd7` pour tous ses
		// chemins sous un prédicat de finitude (0 bit). Le portage ne le lit que sur ce repli ;
		// son absence sur les autres chemins du delta est une découverte du lot J6.3, non traitée.
		br.ReadBits(2)
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

// consumeAbsoluteWithGate porte la BRANCHE ABSOLUE de `FUN_1406cfe44` (bUsePred = 0, bDelta = 0) :
// precHigh R(1), la garde de pleine precision, puis `FUN_14076e524(0x10)` (CALL 1406d009d, niveau
// en 1406d008a) ou, precHigh a 1, `FUN_141f85880` — c est la forme de `FUN_14076e420`, recopiee
// en ligne par le jeu, et le portage unique la lit ([lireE420]).
//
// PRECHIGH A 1 LIT TROIS AXES (lot J6.3, releve du 2026-09-27, §2.c) : `FUN_141f85880` n est
// PAS « le vecteur par defaut, 0 bit » que ce lecteur disait — elle applique la loi au niveau
// 0x10 aux bornes +/-100 de `DAT_143b8c6d0`, soit 3 x 14 = 42 bits. La position qu elle ecrit
// est finie : le R(2) de queue suit.
func consumeAbsoluteWithGate(br *Lecteur) {
	_, pos := lireE420(br, niveauPosition)
	if pos.brute {
		// FUN_1411b259c = R(96) BRUT. Le R(2) de `LAB_1406cffd7` depend de la finitude des
		// flottants lus ; ce chemin ne le lit pas (comportement conserve, hors production).
		return
	}
	semerPositionAbsolue(br, pos, kindDuCheminAbsolu(br))
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

// kindDuCheminAbsolu rend la forme de la graine : le repli du delta predit se distingue.
func kindDuCheminAbsolu(br *Lecteur) PosKind {
	if br.cap.viaRepli {
		return PosKindAbsFallback
	}
	return PosKindAbsolute
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
