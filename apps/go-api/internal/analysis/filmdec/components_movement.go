package filmdec

// Object MOVEMENT bit-consumers (BIPED archetype #35, shared with #40 i0..i9):
//   i0 object-position-dynamic-precision          (FUN_1406cfe44)
//   i1 object-translational-velocity-dyn-precision (FUN_14076d45c -> FUN_14076d4d0)
//   i3 object-angular-velocity-dyn-precision        (FUN_140d87740 -> FUN_14076e1c8)
//
// All three follow the dynamic-precision quantized-vector family. i1 and i3 are
// STATICALLY bit-exact (fixed widths confirmed by asm at the call sites). i0's
// VALUE widths come from a runtime precision descriptor (FUN_140be9a14 populates
// DAT_1445cc9e0 axis widths + DAT_144632be0 index width at map load), so its
// payload widths are passed in via PrecisionDescriptor; only the control spine is
// fixed.
//
// Reader-primitive cross-reference for this file (MSB-first):
//   FUN_1406cf008 / inline R(1) refill -> ReadBit()
//   FUN_14076d528(...,mag,scale)       -> R(1) present-flag; if present R(mag)+R(scale)
//   FUN_14076d6dc(...,scale)           -> R(scale)  (log/exp scale word)
//   FUN_1406d8288                       -> unpack (0 bits; consumes the R(mag) already read)
//   FUN_1406d676c(...,0x60)            -> R(96)  (raw full-precision vec3 = 3xfloat32)
//   FUN_14076e524                      -> R(1) gate + R(idxW) + 3xR(axisW) (absolute position)

// rawVec3Bits is the bit width of the full-precision "keep" vec3 copy taken by
// FUN_1406d676c when called with 0x60: 96 bits = three raw float32 components.
const rawVec3Bits = 0x60

// velocityMagBits / velocityScaleBits are the dynamic-precision packed-direction
// magnitude width and log/exp scale width for object-translational-velocity, read
// by FUN_14076d528(...,0xa,0x13): mag = 0x13 (19), scale = 0xa (10). Confirmed by
// the call-site asm at FUN_14076d4d0 (mov [rsp+0x30],0x13 ; mov [rsp+0x28],0xa).
const (
	velocityMagBits   uint = 0x13 // 19
	velocityScaleBits uint = 0x0a // 10
)

// angularMagBits / angularScaleBits are the same family for object-angular-velocity,
// read by FUN_14076d528(...,8,0x13): mag = 19, scale = 8. Confirmed by the call-site
// asm at FUN_14076e1c8 (mov [rsp+0x30],0x13 ; mov [rsp+0x28],8).
const (
	angularMagBits   uint = 0x13 // 19
	angularScaleBits uint = 0x08 // 8
)

// PrecisionDescriptor carries the runtime-populated position quantization widths
// (DAT_144632be0 index width + the 3 per-axis widths from DAT_1445cc9e0..) that
// FUN_140be9a14 installs at map/replication-config load. The film's own precision
// block is the source of truth; statically the .exe tables read 0, so a real
// decode MUST supply these from the film header. AxisW are the FUN_140cc5128
// per-axis widths; IndexW is the FUN_14076e524 gate-path index width.
type PrecisionDescriptor struct {
	IndexW uint
	AxisW  [3]uint
}

// consumeDynPrecVec3 mirrors FUN_14076d528: a leading R(1) present flag (bit==0
// => present), then on the present path R(mag) packed direction + R(scale) log/exp
// magnitude. bit==1 => absent (vector is the engine constant, no further bits).
// dynPrecHook, si non-nil, reçoit chaque vec3 dynamic-precision décodé (vélocité i1/i3) :
// présent, valeur brute direction packée + scale, et le bit de départ (== StartBit du
// composant). Additif : ne change AUCUNE consommation de bits.
var dynPrecHook func(present bool, packedDir, scale uint64, bitpos int)

// SetDynPrecHook installe (ou efface, nil) le hook de capture vélocité dynamic-precision.
func SetDynPrecHook(h func(present bool, packedDir, scale uint64, bitpos int)) { dynPrecHook = h }

func consumeDynPrecVec3(br *BitReader, mag, scale uint) {
	start := br.BitPos()
	if br.ReadBit() { // FUN_14076d528 leading R(1); JNZ -> absent (0 payload bits)
		if dynPrecHook != nil {
			dynPrecHook(false, 0, 0, start)
		}
		return
	}
	dir := br.ReadBits(mag)  // packed direction (feeds FUN_1406d8288 unpack, 0 extra bits)
	sc := br.ReadBits(scale) // FUN_14076d6dc log/exp scale word
	if dynPrecHook != nil {
		dynPrecHook(true, dir, sc, start)
	}
}

// consumeObjectTranslationalVelocity (i1) mirrors FUN_14076d45c -> FUN_14076d4d0.
//
//	outer = R(1)
//	if outer == 1 : R(96)  (FUN_1406d676c 0x60 raw full-precision vec3)  [keep path]
//	if outer == 0 : consumeDynPrecVec3(mag=19, scale=10)                 [delta path]
//
// Bit cost: outer==1 -> 97 ; outer==0 & present -> 31 ; outer==0 & absent -> 2.
func consumeObjectTranslationalVelocity(br *BitReader) {
	if br.ReadBit() { // FUN_14076d45c R(1); set -> FUN_14076d4d0 mode 2 (keep)
		br.ReadBits(rawVec3Bits) // FUN_1406d676c(...,0x60) = R(96)
		return
	}
	consumeDynPrecVec3(br, velocityMagBits, velocityScaleBits)
}

// consumeObjectAngularVelocity (i3) mirrors FUN_140d87740 -> FUN_14076e1c8. Same
// shape as translational velocity; the scale width is 8 (not 10).
//
//	outer = R(1)
//	if outer == 1 : R(96)                                  [keep path]
//	if outer == 0 : consumeDynPrecVec3(mag=19, scale=8)    [delta path]
//
// Bit cost: outer==1 -> 97 ; outer==0 & present -> 29 ; outer==0 & absent -> 2.
func consumeObjectAngularVelocity(br *BitReader) {
	if br.ReadBit() { // FUN_140d87740 R(1); set -> FUN_14076e1c8 mode 2 (keep)
		br.ReadBits(rawVec3Bits) // FUN_1406d676c(...,0x60) = R(96)
		return
	}
	consumeDynPrecVec3(br, angularMagBits, angularScaleBits)
}

// PositionFullPrecision mirroite le SEUL global `DAT_145121140` : la configuration
// « réplication haute précision » du process. NOT a bitstream bit. Default false
// (retail high-prec path inactive).
//
// CORRECTION DE MODÈLE, lot R7-c (2026-08-17). Ce champ portait auparavant les DEUX
// globaux de `FUN_14076f91c` à la fois, ce qui était faux : `DAT_144e61ea0` est une
// PORTÉE (cf. keyframeBaselineScope) et `DAT_145121140` un réglage de process, et ils
// ne gardent PAS les mêmes lecteurs. Sous la seule portée baseline, `i49`
// (`FUN_14107166c`), `i2 forward-and-up` (`FUN_140c5f938`) et le bloc MPP
// (`FUN_14080cfe8`) — qui lisent `DAT_145121140` SEUL — ne bougent pas.
var PositionFullPrecision = false

// keyframeBaselineScope mirroite `DAT_144e61ea0` : une PORTÉE, pas un réglage. Les huit
// lecteurs d'état complet du groupe `142e2*`/`142e3*` (dont `FUN_142e2bfd0`) le lèvent à 1
// juste AVANT l'appel `vtable[0x60]` (état par défaut) et le remettent à 0 juste APRÈS.
// Pendant cette portée, `FUN_14076f91c()` est vrai et tous les lecteurs de position du
// moteur passent du quantifié au BRUT 96 bits.
//
// KILL-SWITCH — défaut `false` depuis le 2026-08-17. La bascule du défaut est conditionnée
// à UN critère mesurable : que la lecture d'un corps d'image-clé sous cette portée fasse
// remonter l'atterrissage bit-exact des 591 records `ti=35` bornés au-dessus de 50 %
// (mesure `TestKF35CBaselineScope`). Retrait cible du drapeau : à la bascule.
var keyframeBaselineScope = false

// SetKeyframeBaselineScope (dé)active la portée baseline et rend l'ancienne valeur
// (instruments de mesure ; défaut off, restauration en `defer`).
func SetKeyframeBaselineScope(v bool) bool {
	prev := keyframeBaselineScope
	keyframeBaselineScope = v
	return prev
}

// fullPrecisionGate porte `FUN_14076f91c` : `DAT_144e61ea0 != 0 || DAT_145121140 == 1`.
// Zéro bit consommé — c'est un prédicat de CONTEXTE, jamais un bit du flux.
func fullPrecisionGate() bool { return keyframeBaselineScope || PositionFullPrecision }

// PositionDeltaHasHandleTail mirrors the runtime field bVar16 = (precIndex != -1)
// that gates the i0 predicted-delta handle tail in FUN_1406cfe44. It is NOT a
// bitstream bit: precIndex lives at precDesc+0x10 in the entity's previous position
// state (RAM), populated at map load from the film's replication config (reads 0/-1
// statically — same limitation as TraversalPrecision). Default false = the dominant
// precIndex==-1 case (no tail). The CE delta capture confirms whether it ever fires.
var PositionDeltaHasHandleTail = false

// PositionCalibratedSkip active la calibration intelligente d'i0 (saut au total CE 47/101 selon
// bUsePred) au lieu du deser dont la précision d'axe runtime n'est pas sourcée statiquement.
// Harness de validation map-spécifique (Cliffhanger). Default false.
var PositionCalibratedSkip = false

// DeltaQuantum est le pas (unité monde) d'UN cran de position répliqué en DELTA par i0. La
// famille delta (signed-8 ou axis-width) code un NOMBRE DE CRANS signé ; le pas physique est ce
// quantum, PROPRE au chemin delta et DISTINCT de la range absolue (Cliffhanger ~[-974,179] / 2^6
// = 18 u = FAUX pour un delta). Valeur par défaut = quantum de grille MESURÉ sur l'oracle CE
// (différence minimale non nulle entre positions consécutives, identique sur X/Y/Z et tous les
// slots). Réglable via SetDeltaQuantum (l'outil de validation le DÉRIVE de l'oracle plutôt que
// de le figer). La range delta pour le chemin axis-width vaut DeltaQuantum * 2^AxisW (centrée 0).
var DeltaQuantum float32 = 0.01383

// SetDeltaQuantum règle le pas du chemin delta d'i0 (dérivé de l'oracle par l'outil de validation).
func SetDeltaQuantum(q float32) { DeltaQuantum = q }

// consumeObjectPositionDynamicPrecisionD (i0) mirrors FUN_1406cfe44, bit-exactly.
//
//	bUsePred = R(1) ; bDelta = R(1)  (header)
//
// Three mutually-exclusive payload paths + the shared bHandle tail. AxisW/IndexW
// from the runtime PrecisionDescriptor (pd); PositionFullPrecision = the
// FUN_14076f91c runtime gate (received, not read from the stream).
func consumeObjectPositionDynamicPrecisionD(br *BitReader, pd PrecisionDescriptor) {
	posCaptureStartBit = br.BitPos() // entry bit (== component StartBit) for sample attribution
	posCaptureSlot = accumSlot       // slot du record courant (attribution multi-entités)
	if PositionCalibratedSkip {
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
		return
	}
	bUsePred := br.ReadBit() // FUN_1406cfe44 R(1)
	bDelta := br.ReadBit()   // FUN_1406cfe44 R(1)

	// bUsePred==1: KEEP-BASELINE. bHandle, then a raw vec3 copy, then the tail.
	if bUsePred {
		bHandle := br.ReadBit()                    // FUN_1406cf008 R(1)
		readRawVec3(br)                            // FUN_1406d676c(...,0x60) = R(96) : AVANCE le curseur
		keepBaseline()                             // réutilisation baseline : ré-émet prev, JAMAIS les 96 bits bruts
		consumePositionHandleTail(br, bHandle, pd) // same tail whether bDelta 0 or 1
		return
	}

	// bUsePred==0, bDelta==0: ABSOLUTE -> FUN_14076e524, then tail (no fresh handle bit).
	if !bDelta {
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
	// -1), NOT a bitstream bit (PositionDeltaHasHandleTail; default false = the
	// dominant precIndex==-1 case). FUN_14076f91c full-precision gate =
	// `fullPrecisionGate` (DAT_144e61ea0 OU DAT_145121140). Both runtime gates are confirmed
	// via the CE delta capture.
	predFlag := br.ReadBit() // FUN_1406cfe44 inline R(1)
	if !predFlag {
		// LAB_1406cff18 : la garde de contexte est ICI, sur ce seul chemin (relu le
		// 2026-08-17, lot R7-c — elle ne couvre PAS la branche predFlag==1, qui porte la
		// sienne dans FUN_14076e4ec).
		if fullPrecisionGate() {
			readRawVec3(br) // FUN_1406d676c(...,0x60) = R(96) : AVANCE le curseur (keep, pas une coord)
			keepBaseline()
		} else {
			consumePredictedDelta(br, pd) // FUN_14076f3ec
		}
	} else {
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
		if fullPrecisionGate() {
			br.ReadBits(rawVec3Bits) // FUN_14076e4ec -> FUN_1411b259c
		} else if !cVar1 { // 0 -> lit la position absolue quantifiée
			pidx := -1
			if !br.ReadBit() { // FUN_14076e524 index-sel ; 0 -> lit l'index
				pidx = int(br.ReadBits(pd.IndexW))
			}
			var v [3]float32
			for i := 0; i < 3; i++ {
				w := absAxisWFor(pidx, i)                     // largeur par index (7ter.54) ou uniforme
				v[i] = dequantWorldAxis(br.ReadBits(w), w, i) // FUN_140cc5128 axe i
			}
			seedAbsolute(PosKindAbsolute, v) // predFlag==1 = position absolue = seed d'accumulation
		}
	}
	if PositionDeltaHasHandleTail { // runtime bVar16 = (precIndex != -1)
		if br.ReadBit() { // FUN_1406cf008 -> FUN_1408f0ac4 handle resolve
			consume1408f0ac4(br)
		}
		if br.ReadBit() { // FUN_1406cf008 region present
			if br.ReadBit() { // FUN_1406cf008 region ext
				br.ReadBits(11) // R(0xb) region word
			}
		}
	}
}

// consumePredictedDelta mirrors FUN_14076f3ec -> FUN_14076f550 (taken when
// FUN_14076f91c is false): a leading present flag, then EITHER 3 fixed signed 8-bit
// deltas (dominant) OR 3 axis-width words, OR an absolute fallback.
func consumePredictedDelta(br *BitReader, pd PrecisionDescriptor) {
	if br.ReadBit() { // FUN_14076f3ec R(1); set => predicted absent -> absolute fallback
		absViaFallback = true
		consumeAbsoluteWithGate(br, pd)
		absViaFallback = false
		return
	}
	if br.ReadBit() { // FUN_14076f550 mask; set => fixed signed 8-bit deltas (dominant)
		// 3 signed 8-bit deltas = un NOMBRE DE CRANS signé par axe. Le pas physique est
		// DeltaQuantum (propre au delta, PAS la range absolue/2^axisW qui donnait ~18 u = faux).
		var d [3]float32
		for i := 0; i < 3; i++ {
			n := signed8(br.ReadBits(8))
			d[i] = float32(n) * DeltaQuantum
		}
		applyDelta(PosKindDelta8, d)
		return
	}
	// mask clear => FUN_1424cbed4 -> FUN_140cc5128 : delta axis-width. C'est un delta SIGNÉ
	// centré-zéro (pas une pseudo-absolue qui soustrairait Min) : q ∈ [0,2^AxisW) est recentré
	// sur [-2^(AxisW-1), 2^(AxisW-1)) et mis à l'échelle DeltaQuantum (range delta propre =
	// DeltaQuantum*2^AxisW). AxisW = pd.AxisW[i] (6 par défaut ; ambiguïté 6 vs 14 sweepable).
	var d [3]float32
	for i := 0; i < 3; i++ {
		w := deltaAxisW(pd, i)
		q := br.ReadBits(w)
		half := float32(uint64(1) << (w - 1))
		d[i] = (float32(q) - half) * DeltaQuantum // multiple ENTIER de Q, centré (q==half -> 0)
	}
	applyDelta(PosKindDeltaAxis, d)
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
func consumeQuantVec3WithGate(br *BitReader, axisW uint) {
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
// l'.exe — même limitation que TraversalPrecision.
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
var DeltaAxisWidth uint = 14

// deltaAxisW retourne la largeur d'axe du chemin delta (DeltaAxisWidth si > 0, sinon pd).
func deltaAxisW(pd PrecisionDescriptor, i int) uint {
	if DeltaAxisWidth > 0 {
		return DeltaAxisWidth
	}
	return pd.AxisW[i]
}

// consumeAbsoluteWithGate mirrors the absolute-reader spine (prec-select, the
// FUN_14076f91c runtime gate, then FUN_14076e524 index+vec3).
func consumeAbsoluteWithGate(br *BitReader, pd PrecisionDescriptor) {
	precHigh := br.ReadBit() // FUN_1406cf008
	if fullPrecisionGate() {
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
	// The index selects the dequant RANGE (DAT_14462cbe0): index 0 = the map replication
	// bounds (real in-map position) ; index 1 / no-index = ±20000 (off-map, non-player).
	idx := -1 // no index (index-select bit set) -> fallback ±20000
	if !br.ReadBit() {
		idx = int(br.ReadBits(pd.IndexW)) // DAT_144632be0 index width
	}
	var v [3]float32
	for i := 0; i < 3; i++ {
		// Les deux lignées se rejoignent ici sans se contredire : `absAxisWFor` cherche
		// d'abord la largeur PAR INDEX de plage (7ter.54), et retombe sur `absAxisW` —
		// la table de région 13/13/14 qui ferme le compte à 47 bits — quand aucune table
		// par index n'est installée, ce qui est le défaut. La mesure du rejeu est donc
		// préservée telle quelle, et le chemin par index reste disponible.
		w := absAxisWFor(idx, i)                      // par index (7ter.54), sinon région 13/13/14
		v[i] = dequantWorldAxis(br.ReadBits(w), w, i) // FUN_140cc5128 axis i
	}
	// Champ « fini » de 2 bits — FUN_14076e304, appelé en LAB_1406cffd7 sous un prédicat
	// (FUN_140492128) qui ne consomme AUCUN bit : il est donc lu systématiquement.
	//
	// AJOUTÉ le 2026-07-27. Le chemin world-object le lisait déjà (traverse.go, `[2 finite]`
	// de son total de 45 bits) ; le chemin dynamic-precision du bipède ne l'a jamais lu. Les
	// deux portent pourtant le MÊME lecteur absolu — c'est la double implémentation qui les
	// a laissés diverger. Sans ces 2 bits le compte tombait à 45 au lieu des 47 mesurés par
	// la capture CE (100 % de 154 158 dispatches, aucune variance).
	br.ReadBits(2)
	// Only index-0 positions are in the map bounds (real player positions). index!=0
	// dequantizes against ±20000 (off-map) and is noise for a trajectory -> don't emit.
	if idx != 0 {
		return
	}
	kind := PosKindAbsolute
	if absViaFallback {
		kind = PosKindAbsFallback
	}
	seedAbsolute(kind, v) // absolue in-map = seed d'accumulation pour les deltas ultérieurs
}

// consumePositionHandleTail mirrors the bHandle-gated tail shared by FUN_1406cfe44
// (inline) and FUN_14076e3e4: if bHandle clear the field is 0xFFFFFFFF (0 bits);
// else a handle-resolve word (R(IndexW)+R(2)) and an optional 11-bit region word.
// FUN_1406cb0cc reads 0 bits (runtime validity predicate only).
func consumePositionHandleTail(br *BitReader, bHandle bool, pd PrecisionDescriptor) {
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
//   - la source de L (le champ flags du registre chunk_00) reste une PISTE pour ces
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
// largeur d'axe PARAMÉTRÉE (PISTE 1) : axisW = 6+level, level = flags du registre. La
// largeur d'index reste en dur à 1 (DAT_144632be0), comme partout ailleurs dans ce
// paquet : aucun appelant n'en a jamais passé une autre. Décode pur (pas d'emitPos) pour
// les composants non-position porteurs d'un vec3 quantifié (crew-order, tacmap-offset,
// desired-respawn-location, ...).
func consumeQuantVec3(br *BitReader, axisW uint) {
	if br.ReadBit() { // precHigh == 1 -> vecteur défaut, 0 bit
		return
	}
	if !br.ReadBit() { // index-present select ; 0 -> lit l'index
		br.ReadBits(1) // DAT_144632be0 = 1
	}
	br.ReadBits(axisW)
	br.ReadBits(axisW)
	br.ReadBits(axisW)
}
