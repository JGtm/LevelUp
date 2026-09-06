package filmdec

// Position capture (ADDITIVE, PARTIE 1 victim-position pipeline). This file lets a
// probe observe the i0 object-position-dynamic-precision payload WITHOUT changing
// any bit-consumption: consumeObjectPositionDynamicPrecisionD reads exactly the same
// bits as before; it merely *also* reports the decoded value through a package-level
// hook when one is installed. Default (hook nil) = zero behaviour change.
//
// The i0 deser (FUN_1406cfe44) has three payload encodings (cf components_movement.go):
//   - KEEP-BASELINE (bUsePred==1): raw vec3 = 3 float32 from a 96-bit copy (PosKindRaw).
//   - ABSOLUTE      (bUsePred==0,bDelta==0): 3 quantized axes -> dequant world pos
//     (PosKindAbsolute). Absent/default-vector paths report nothing.
//   - PREDICTED-DELTA(bUsePred==0,bDelta==1): a present flag then EITHER 3 signed
//     8-bit deltas (PosKindDelta8) OR 3 axis-width quantized words (PosKindDeltaAxis),
//     OR an absolute fallback (PosKindAbsolute). The caller accumulates deltas onto
//     the per-slot baseline.
//
// WORLD RANGE for the absolute/keyframe quantized path: the i0 axis widths come from
// TraversalPrecision (IndexW=1, AxisW=6/6/6, measured via Cheat Engine). The dequant
// RANGE is the engine world-position range DAT_143b8c6f0 precision-2 = +/-100 per axis
// (QuantRangeWorld100, validated by quantize_test.go::TestReadQuantizedVec3_World100).
// Halo Infinite ships maps inside a normalized [-100,100]^3 replication box for the
// dynamic-precision position component, so QuantRangeWorld100 is the right table slot
// for 000d5950. The raw (96-bit) path is in absolute engine world units (un-quantized
// float32), a different magnitude scale than the quantized box — they are NOT mixed.

// PosKind tags how a captured i0 sample was encoded in the stream.
type PosKind int

const (
	PosKindNone        PosKind = iota
	PosKindRaw                 // 96-bit raw vec3 (3 float32), keep-baseline copy
	PosKindAbsolute            // DIRECT quantized absolute world pos (bUsePred=0,bDelta=0)
	PosKindAbsFallback         // quantized absolute via the predicted-delta absent fallback
	PosKindDelta8              // 3 signed 8-bit deltas (quantized world step)
	PosKindDeltaAxis           // 3 axis-width quantized delta words
)

func (k PosKind) String() string {
	switch k {
	case PosKindRaw:
		return "raw"
	case PosKindAbsolute:
		return "abs"
	case PosKindAbsFallback:
		return "absfb"
	case PosKindDelta8:
		return "d8"
	case PosKindDeltaAxis:
		return "dax"
	default:
		return "none"
	}
}

// PositionSample is one decoded i0 position payload. For absolute/raw paths Vec is the
// world position; for delta paths Vec holds the per-axis delta (in dequantized world
// units for Delta8, in raw quantized steps the caller must accumulate for DeltaAxis).
// BitPos is the stream bit position at which the i0 component STARTED (== comps[0].
// StartBit of its record), letting a probe attribute the sample to the right record.
type PositionSample struct {
	Kind PosKind
	Vec  [3]float32 // ABSOLUE RÉSOLUE (seed OU prev+delta accumulé) quand un World accumulateur
	// est installé ; sinon delta brut pour les chemins delta (borné, PAS une coordonnée monde).
	BitPos int
	Slot   uint32 // slot du record en cours de décodage (attribution multi-entités)
}

// posCaptureStartBit is the stream bit position at which the currently-decoding i0
// component began, set by consumeObjectPositionDynamicPrecisionD on entry so emitPos
// can stamp the sample for record attribution by the probe.
var posCaptureStartBit int

// posCaptureSlot est le slot du record en cours (== accumSlot au moment où i0 décode), estampillé
// sur chaque PositionSample pour l'attribution par slot côté probe/outil de trajectoire.
var posCaptureSlot uint32

// accumWorld / accumSlot : contexte d'ACCUMULATION de position. Quand accumWorld != nil, le deser
// i0 RÉSOUT chaque frame en position absolue : les chemins absolus posent le seed (SetPos), les
// chemins delta lisent prev (PosOf) + delta et réécrivent (SetPos), le keep-baseline ré-émet prev.
// accumSlot est renseigné par le décodeur de record (decodeDelta / DecodeFrameRecords) AVANT la
// boucle de composants. accumWorld == nil (défaut) = pas d'accumulation : le fix SÉMANTIQUE reste
// actif (jamais d'émission des 96 bits keep-baseline bruts = fin de l'aberrant ~1e28) mais les
// deltas sont émis bruts (bornés).
//
// PLUS AUCUN INSTALLATEUR depuis le 2026-09-05 (lot E, item E.2) : `SetPositionAccumulator`
// n'avait aucun appelant et a été supprimé avec les 21 autres réglages inatteignables.
// accumWorld reste donc nil sur tous les chemins ; la variable est conservée parce que
// l'accumulation par World est la sémantique portée du deser i0, et qu'un harnais interne au
// paquet la rétablirait en une ligne — pas par une surface publique que personne n'appelle.
var (
	accumWorld *World
	accumSlot  uint32
)

// setAccumSlot fixe le slot cible d'accumulation pour le record courant (appelé par les décodeurs).
func setAccumSlot(slot uint32) { accumSlot = slot }

// absViaFallback marks that the current absolute read is reached via the predicted-
// delta absent fallback (vs the direct absolute path), so emitPos can tag the sample
// PosKindAbsFallback. The two paths have very different reliability in practice.
var absViaFallback bool

// posCaptureHook, when non-nil, receives every i0 position payload the deser decodes.
// Nil by default (no-op, no behaviour change). Not safe for concurrent use across
// goroutines (single-frame decode is sequential).
//
// INSTALLÉ DEPUIS LE PAQUET, PLUS DE L'EXTÉRIEUR : le seul installateur est
// `scanForTargetDelta` (frame_records.go), qui capture la position i0 de chaque record
// d'essai. Le réglage public `SetPositionCaptureHook` a été supprimé le 2026-09-05
// (lot E, item E.2) : il n'avait aucun appelant.
var posCaptureHook func(PositionSample)

// emitPos reports a decoded i0 sample to the hook if one is installed.
func emitPos(kind PosKind, v [3]float32) {
	if posCaptureHook != nil {
		posCaptureHook(PositionSample{Kind: kind, Vec: v, BitPos: posCaptureStartBit, Slot: posCaptureSlot})
	}
}

// seedAbsolute pose une position ABSOLUE fraîche (keyframe / predFlag==1 / fallback) : c'est le
// point d'ancrage à partir duquel les deltas ultérieurs s'accumulent. Écrit dans le World
// accumulateur si présent, puis émet.
func seedAbsolute(kind PosKind, v [3]float32) {
	if accumWorld != nil {
		accumWorld.SetPos(accumSlot, v)
	}
	emitPos(kind, v)
}

// applyDelta accumule un delta signé (centré-zéro) sur la dernière position résolue du slot.
// Sans World accumulateur : émet le delta brut (borné, PAS une coordonnée). Avec World mais sans
// seed préalable : n'émet RIEN (trou attendu — deltas antérieurs à la 1re absolue d'un slot).
func applyDelta(kind PosKind, d [3]float32) {
	if accumWorld == nil {
		emitPos(kind, d)
		return
	}
	prev, ok := accumWorld.PosOf(accumSlot)
	if !ok {
		return // delta sans seed : pas encore de position pour ce slot
	}
	np := [3]float32{prev[0] + d[0], prev[1] + d[1], prev[2] + d[2]}
	accumWorld.SetPos(accumSlot, np)
	emitPos(kind, np)
}

// keepBaseline traite le chemin KEEP-BASELINE (bUsePred==1) et le keep pleine-précision : les
// 96 bits lus NE SONT PAS une coordonnée (réutilisation de la baseline, cf FUN_1406cfe44). On
// ré-émet la position courante résolue du slot (si connue) au lieu du float garbage (fin de
// l'aberrant ~1e28). Sans World accumulateur : rien à ré-émettre.
func keepBaseline() {
	if accumWorld == nil {
		return
	}
	if prev, ok := accumWorld.PosOf(accumSlot); ok {
		emitPos(PosKindRaw, prev)
	}
}

// WorldPositionRange is the dequant range used for the i0 absolute/keyframe quantized
// path. Exposed (var, not const) so a probe can A/B alternative range-table slots
// (Unit3 / Norm / World100 / Cliffhanger) against known sane coordinates. DEFAULT =
// QuantRangeCEBiped, the live-captured DAT_14462cbe0[0] small map-local biped box (span
// X~113 => 0.0138 oracle quantum). The old QuantRangeCliffhanger [-974,179]... scattered
// absolutes hundreds of units off-box (the range WAS the bug); it stays selectable for
// the before/after proof.
var WorldPositionRange = QuantRangeCEBiped

// AbsDequantMode sélectionne la FORME de déquantification d'un axe absolu i0.
type AbsDequantMode int

const (
	// AbsDequantRange : min + step*(q+0.5) via WorldPositionRange (formule range Cliffhanger,
	// FUN_140c1e978). C'est le comportement historique — miscalibré pour les positions joueur
	// (span ~1150 u sur 6 bits => 18 u/cran = téléportation).
	AbsDequantRange AbsDequantMode = iota
	// AbsDequantCenteredQuantum : (q - 2^(w-1)) * DeltaQuantum. Grille fine centrée sur 0, MÊME
	// quantum que le chemin delta (0.0138). L'axe couvre ±2^(w-1)·quantum ; à w=14 => ±113 u,
	// qui contient largement la boîte oracle joueur (X[-6..36] Y[-25..27] Z[-4..7]).
	AbsDequantCenteredQuantum
)

// absDequantMode : forme de déquant des chemins absolus i0 (défaut = range Cliffhanger historique).
// Le réglage public `SetAbsDequantMode` (harnais de calibration) a été supprimé le 2026-09-05
// (lot E, item E.2) : aucun appelant. La valeur de production est celle du défaut ci-dessous.
// PROVENANCE : forme de dequant MESUREE du chemin absolu i0 sur le film 000d5950 (Cliffhanger).
// Constante depuis le 2026-09-06 (lot E, item E.8) : plus aucun ecrivain depuis le retrait des
// 22 reglages morts, et c est la valeur que la production decode.
const absDequantMode = AbsDequantRange

// absoluteAxisW, si > 0, OVERRIDE la largeur d'axe des CHEMINS ABSOLUS i0 (consumeAbsoluteWithGate
// + predFlag==1) — distincte de pd.AxisW (qui garde 6/6/6 pour le default-state et le delta
// axis-width). La capture CE mesure 3×14 sur predFlag==1 (total i0 predicted = 47 bits). 0 =
// utilise pd.AxisW[i] (comportement historique). Changer cette largeur CHANGE la consommation de
// bits des chemins absolus — intentionnel (le juge devient la boîte oracle, pas offline-vs-CE).
//
// VALEUR PAR DEFAUT 14 (2026-07-25) : mesuree contre l'oracle de POSITION Rosette du film
// 000d5950 (cmd/tmp_deadstate, modes `split` et `solvechain`). Le chemin i0 d'un record
// DELTA biped est le chemin absolu (bUsePred=0, bDelta=0 sur 3090/3090 records) et sa
// largeur vraie est 47 bits = 2 + 1(precHigh) + 1(index-sel) + 1(IndexW) + 3x14. La table
// de precision du jeu (DAT_1445cc9e0, dumpee dans ce_prec_widths_1445cc9e0.bin) donne
// 14/14/14 au niveau 8 (largeur = 6+L) : 14 est donc une entree REELLE de la table du .exe,
// pas une constante ad hoc. Verification croisee : a i0=47, les desers PORTES de i1 et i21
// consomment exactement leurs largeurs vraies sur 100.0% de 15 529 records, et le deser
// porte de i25 finit exactement a la fin vraie sur 100.0% de 3 090 records.
var absoluteAxisW uint = 14

// SetAbsoluteAxisW règle la largeur d'axe des chemins absolus i0 (0 = pd.AxisW). Harness de sweep.
func SetAbsoluteAxisW(w uint) { absoluteAxisW = w }

// absAxisW retourne la largeur d'axe effective d'un chemin ABSOLU i0.
//
// CORRECTION DU 2026-07-27 — le repli était `pd.AxisW`, c'est-à-dire 6/6/6, et c'était FAUX.
//
// `pd.AxisW` est la largeur du chemin DELTA. Le chemin ABSOLU lit la table de région
// (`FUN_140cc5128` après `FUN_14076e524`), dont les largeurs sont 13/13/14 — les mêmes que
// celles du chemin world-object, qui les porte correctement depuis toujours
// (`WorldObjectPrecision`, cf. traverse.go). Les deux chemins lisent la MÊME table ; c'est
// leur double implémentation qui les avait laissés diverger.
//
// LA MESURE QUI TRANCHE : la capture CE du dispatch donne i0 du bipède à **47 bits, une
// seule valeur distincte, 100 % de 154 158 dispatches**. Le compte se ferme à l'unité :
//
//	1 bUsePred + 1 bDelta + 1 precHigh + 1 indexSel + 1 IndexW + (13+13+14) + 2 finite = 47
//
// Avec 6/6/6 et sans le champ « finite » on consommait 23 bits — 24 de moins, sur un
// composant présent dans 96,8 % des records et lu EN PREMIER. Le déficit décalait donc
// l'en-tête du record SUIVANT, d'où des masques lus n'importe où et un i22 vu 63 fois trop
// souvent. Chercher la faute dans les grammaires de composants ne pouvait rien donner :
// elles étaient justes.
//
// POURQUOI L'ESSAI PRÉCÉDENT AVAIT ÉCHOUÉ : régler `TraversalPrecision.AxisW` à 13/13/14
// changeait AUSSI la largeur du delta — le chemin dominant — et dégradait la mesure. Le
// commentaire de traverse.go le disait déjà : « le vrai correctif doit distinguer les deux
// largeurs le long de chaque branche, et non régler une globale. »
// (Le descripteur de précision n'entre PAS dans ce choix : la largeur absolue vient soit
// du réglage global, soit de WorldObjectPrecision — jamais du descripteur de l'appelant.)
func absAxisW(i int) uint {
	if absoluteAxisW > 0 {
		return absoluteAxisW
	}
	return WorldObjectPrecision.AxisW[i]
}

// absAxisWFor retourne la largeur de l axe i pour l index de plage idx.
//
// LARGEURS D AXE PAR INDEX DE PLAGE DE REPLICATION (7ter.54 axe 3) — LE SAVOIR, GARDE ICI.
//
// SOURCE, DESASSEMBLAGE : `FUN_14076e524(out, reader, outIndexPtr, LEVEL)` choisit ses trois
// largeurs dans DEUX tables distinctes selon l'index lu au flux :
//
//	index == -1 (bit de gate pose)  ->  DAT_1445cc9e0 + LEVEL*0xc          (table << defaut >>)
//	index >= 0                     ->  DAT_1445ccbe0 + (index*0x20 + LEVEL)*0xc  (table << par index >>)
//
// et LEVEL est un IMMEDIAT STATIQUE 0x10 = 16 sur les trois sites d'appel du composant
// position (`MOV R9D,0x10` a 1406d008a dans FUN_1406cfe44 ; `MOV R8D,0x10` a 140f7ea50 dans
// FUN_140f7ea14, que FUN_14076e4ec deplace en R9 a 14076e505 ; 14226a6b8 dans FUN_14076f3ec).
// Les trois largeurs ne sont donc PAS uniformes et PAS les memes pour tous les index — ce que
// l'override uniforme `absoluteAxisW` suppose.
//
// LA TABLE `absPerIndexAxisW` QUI PORTAIT CE MODELE A ETE SUPPRIMEE le 2026-09-06 (lot E, item
// E.8) : elle etait nil et le restait — son unique installateur, le reglage public
// `SetAbsPerIndexAxisW`, n avait aucun appelant et est parti au lot E.2. Elle ne portait AUCUNE
// valeur mesuree, seulement le modele ci-dessus, qui reste donc ecrit ici, a l endroit ou un
// futur portage viendra le lire. La largeur rendue est celle du chemin uniforme, comme avant.
func absAxisWFor(idx, i int) uint {
	if i == 0 {
		absIdxHist[idx]++
	}
	return absAxisW(i)
}

// absIdxHist : histogramme des index de plage rencontres sur les chemins ABSOLUS de i0 (7ter.54
// axe 3). Purement observationnel : incremente sur l'axe 0 de chaque lecture, ne change AUCUNE
// consommation de bits. C'est la mesure qui dit si l'index dominant est 0 (bornes de la map) ou
// pas — donc quelle ligne de la table de largeurs pese reellement.
var absIdxHist = map[int]int{}

// AbsIndexHistogram rend (et remet a zero) l'histogramme des index de plage absolus.
func AbsIndexHistogram() map[int]int {
	out := make(map[int]int, len(absIdxHist))
	for k, v := range absIdxHist {
		out[k] = v
	}
	absIdxHist = map[int]int{}
	return out
}

// dequantWorldAxis dequantizes one absolute quantized axis word (width bits). Deux formes :
//   - AbsDequantRange (défaut) : min + step*(q+0.5) via WorldPositionRange (FUN_140c1e978).
//   - AbsDequantCenteredQuantum : (q - 2^(bits-1)) * DeltaQuantum — grille fine centrée sur 0.
func dequantWorldAxis(q uint64, bits uint, axis int) float32 {
	if absDequantMode == AbsDequantCenteredQuantum {
		half := float32(uint64(1) << (bits - 1))
		return (float32(q) - half) * DeltaQuantum
	}
	scale := float32(uint64(1) << bits)
	step := (WorldPositionRange[axis].Max - WorldPositionRange[axis].Min) / scale
	return float32(q)*step + WorldPositionRange[axis].Min + step*quantCenter
}

// signed8 reinterprets an 8-bit field as a signed delta count.
func signed8(b uint64) int32 {
	if b&0x80 != 0 {
		return int32(b) - 256
	}
	return int32(b)
}

// readRawVec3 consomme les 96 bits d'un vec3 brut (3 IEEE-754 float32, MSB-first au sens
// du lecteur de bits du film). Mirrors FUN_1406d676c(...,0x60).
//
// NB (2026-06-30, Ghidra FUN_1406cfe44) : le chemin keep-baseline (bUsePred=1) NE
// réécrit PAS le champ position (il garde la baseline) — ces 96 bits ne sont donc PAS
// une position joueur. PosKindRaw ne doit pas être interprété comme une coordonnée.
// C'EST POURQUOI RIEN N'EST RENDU : la fonction n'existe que pour AVANCER le curseur, et
// rendre un [3]float32 invitait à lire ces bits comme une coordonnée — ce que le NB
// ci-dessus interdit. Aucun appelant ne l'a jamais fait.
func readRawVec3(br *BitReader) {
	for i := 0; i < 3; i++ {
		br.ReadBits(32)
	}
}
