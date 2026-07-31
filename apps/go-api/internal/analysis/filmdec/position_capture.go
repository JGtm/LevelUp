package filmdec

import "math"

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
// deltas sont émis bruts (bornés). Seul l'outil de trajectoire persistante installe accumWorld.
var (
	accumWorld *World
	accumSlot  uint32
)

// SetPositionAccumulator installe (ou retire, nil) le World qui accumule les positions i0 à
// travers les records/paquets. Le World doit être CELUI passé au décodeur séquentiel
// (DecodeFrameRecords) : les seeds absolus et l'accumulation de delta y sont écrits par slot.
// Ne pas l'utiliser avec les décodeurs à essais/inférence (ils re-décodent en spéculatif et
// pollueraient l'accumulation).
func SetPositionAccumulator(w *World) { accumWorld = w }

// setAccumSlot fixe le slot cible d'accumulation pour le record courant (appelé par les décodeurs).
func setAccumSlot(slot uint32) { accumSlot = slot }

// absViaFallback marks that the current absolute read is reached via the predicted-
// delta absent fallback (vs the direct absolute path), so emitPos can tag the sample
// PosKindAbsFallback. The two paths have very different reliability in practice.
var absViaFallback bool

// posCaptureHook, when non-nil, receives every i0 position payload the deser decodes.
// Installed by the probe via SetPositionCaptureHook; nil by default (no-op, no
// behaviour change). Not safe for concurrent use across goroutines (single-frame
// decode is sequential).
var posCaptureHook func(PositionSample)

// SetPositionCaptureHook installs (or clears, with nil) the i0 position capture hook.
func SetPositionCaptureHook(h func(PositionSample)) { posCaptureHook = h }

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
var absDequantMode = AbsDequantRange

// SetAbsDequantMode règle la forme de déquant absolu (harness de calibration contre l'oracle).
func SetAbsDequantMode(m AbsDequantMode) { absDequantMode = m }

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

// absAxisW retourne la largeur d'axe effective d'un chemin absolu (override ou pd.AxisW).
func absAxisW(pd PrecisionDescriptor, i int) uint {
	if absoluteAxisW > 0 {
		return absoluteAxisW
	}
	return pd.AxisW[i]
}

// absPerIndexAxisW — LARGEURS D'AXE PAR INDEX DE PLAGE DE REPLICATION (7ter.54 axe 3).
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
// nil = comportement historique (largeur uniforme). Cle -1 = chemin sans index.
var absPerIndexAxisW map[int][3]uint

// SetAbsPerIndexAxisW installe (nil = efface) la table de largeurs par index de plage.
func SetAbsPerIndexAxisW(t map[int][3]uint) { absPerIndexAxisW = t }

// absAxisWFor retourne la largeur de l'axe i pour l'index de plage idx (repli : largeur uniforme).
func absAxisWFor(pd PrecisionDescriptor, idx, i int) uint {
	if i == 0 {
		absIdxHist[idx]++
	}
	if absPerIndexAxisW != nil {
		if w, ok := absPerIndexAxisW[idx]; ok && w[i] > 0 {
			return w[i]
		}
	}
	return absAxisW(pd, i)
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

// rawVec3 reads a 96-bit raw vec3 (3 IEEE-754 float32, MSB-first per the film
// bit-reader) without dequantization. Mirrors FUN_1406d676c(...,0x60).
//
// NB (2026-06-30, Ghidra FUN_1406cfe44) : le chemin keep-baseline (bUsePred=1) NE
// réécrit PAS le champ position (il garde la baseline) — ces 96 bits ne sont donc PAS
// une position joueur. PosKindRaw ne doit pas être interprété comme une coordonnée.
func readRawVec3(br *BitReader) [3]float32 {
	var v [3]float32
	for i := 0; i < 3; i++ {
		v[i] = math.Float32frombits(uint32(br.ReadBits(32)))
	}
	return v
}
