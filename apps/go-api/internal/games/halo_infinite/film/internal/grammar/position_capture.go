package grammar

import "levelup/go-api/internal/games/halo_infinite/film/internal/profile"

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
// le descripteur de traversée (IndexW=1, AxisW=6/6/6, measured via Cheat Engine). The dequant
// RANGE is the engine world-position range DAT_143b8c6f0 precision-2 = +/-100 per axis
// (profile.QuantRangeWorld100, validated by quantize_test.go::TestReadQuantizedVec3_World100).
// Halo Infinite ships maps inside a normalized [-100,100]^3 replication box for the
// dynamic-precision position component, so profile.QuantRangeWorld100 is the right table slot
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

// captureDePosition : L ETAT DE CAPTURE D UN BALAYAGE, porte par le lecteur de bits.
//
// C ETAIENT SIX VARIABLES DE PAQUET JUSQU AU LOT 2.3 (`posCaptureStartBit`, `posCaptureSlot`,
// `accumWorld`, `accumSlot`, `absViaFallback`, plus la portee de `setAccumSlot`). Elles
// decrivent UN record en cours de decodage : deux balayages simultanes n ont rien a partager
// la-dedans, et c est l une des raisons pour lesquelles le decodage passait sous un verrou.
type captureDePosition struct {
	// startBit : position de bit ou le composant i0 EN COURS a commence — estampillee sur
	// chaque echantillon pour que la sonde l attribue au bon record.
	startBit int
	// slot : le slot du record en cours (== accumSlot au moment ou i0 decode).
	slot uint32
	// accum / accumSlot : contexte d ACCUMULATION. Quand `accum` n est pas nil, le deser i0
	// RESOUT chaque image en position absolue : les chemins absolus posent le seed (SetPos),
	// les chemins delta lisent prev (PosOf) + delta et reecrivent, le keep-baseline re-emet
	// prev. `accum` nil (le defaut) = pas d accumulation : le correctif SEMANTIQUE reste actif
	// (jamais d emission des 96 bits keep-baseline bruts = fin de l aberrant ~1e28) mais les
	// deltas sont emis bruts (bornes).
	//
	// PLUS AUCUN INSTALLATEUR depuis le 2026-09-05 (lot E, item E.2) : `SetPositionAccumulator`
	// n avait aucun appelant. Le champ est conserve parce que l accumulation par World est la
	// semantique PORTEE du deser i0, et qu un harnais interne au paquet la retablirait en une
	// ligne — sur SON lecteur desormais, pas sur le processus.
	accum     *World
	accumSlot uint32
	// viaRepli dit que la lecture absolue en cours est atteinte par le REPLI du delta predit
	// absent (et non par le chemin absolu direct), pour que l echantillon soit etiquete
	// `PosKindAbsFallback`. Les deux chemins n ont pas la meme fiabilite en pratique.
	viaRepli bool
}

// poserSlotDeCapture fixe le slot cible pour le record courant (appele par les decodeurs).
func (b *Lecteur) poserSlotDeCapture(slot uint32) { b.cap.accumSlot = slot }

// emitPos reports a decoded i0 sample to the hook if one is installed.
func (b *Lecteur) emitPos(kind PosKind, v [3]float32) {
	if b.obs != nil && b.obs.PosCaptureHook != nil {
		b.obs.PosCaptureHook(PositionSample{
			Kind: kind, Vec: v, BitPos: b.cap.startBit, Slot: b.cap.slot})
	}
}

// seedAbsolute pose une position ABSOLUE fraîche (keyframe / predFlag==1 / fallback) : c'est le
// point d'ancrage à partir duquel les deltas ultérieurs s'accumulent. Écrit dans le World
// accumulateur si présent, puis émet.
func (b *Lecteur) seedAbsolute(kind PosKind, v [3]float32) {
	if b.cap.accum != nil {
		b.cap.accum.SetPos(b.cap.accumSlot, v)
	}
	b.emitPos(kind, v)
}

// applyDelta accumule un delta signé (centré-zéro) sur la dernière position résolue du slot.
// Sans World accumulateur : émet le delta brut (borné, PAS une coordonnée). Avec World mais sans
// seed préalable : n'émet RIEN (trou attendu — deltas antérieurs à la 1re absolue d'un slot).
func (b *Lecteur) applyDelta(kind PosKind, d [3]float32) {
	if b.cap.accum == nil {
		b.emitPos(kind, d)
		return
	}
	prev, ok := b.cap.accum.PosOf(b.cap.accumSlot)
	if !ok {
		return // delta sans seed : pas encore de position pour ce slot
	}
	np := [3]float32{prev[0] + d[0], prev[1] + d[1], prev[2] + d[2]}
	b.cap.accum.SetPos(b.cap.accumSlot, np)
	b.emitPos(kind, np)
}

// keepBaseline traite le chemin KEEP-BASELINE (bUsePred==1) et le keep pleine-précision : les
// 96 bits lus NE SONT PAS une coordonnée (réutilisation de la baseline, cf FUN_1406cfe44). On
// ré-émet la position courante résolue du slot (si connue) au lieu du float garbage (fin de
// l'aberrant ~1e28). Sans World accumulateur : rien à ré-émettre.
func (b *Lecteur) keepBaseline() {
	if b.cap.accum == nil {
		return
	}
	if prev, ok := b.cap.accum.PosOf(b.cap.accumSlot); ok {
		b.emitPos(PosKindRaw, prev)
	}
}

// WorldPositionRange is the dequant range used for the i0 absolute/keyframe quantized
// path. Exposed (var, not const) so a probe can A/B alternative range-table slots
// (Unit3 / Norm / World100 / Cliffhanger) against known sane coordinates. DEFAULT =
// profile.QuantRangeCEBiped, the live-captured DAT_14462cbe0[0] small map-local biped box (span
// X~113 => 0.0138 oracle quantum). The old profile.QuantRangeCliffhanger [-974,179]... scattered
// absolutes hundreds of units off-box (the range WAS the bug); it stays selectable for
// the before/after proof.
// C'ÉTAIT UNE VARIABLE DE PAQUET JUSQU'AU LOT 2.2.b : la range vit dans le PROFIL que le
// lecteur porte (`Movement.Range`), et l'A/B de sonde se fait en posant un profil sur le
// lecteur, plus en écrivant dans le processus.
func (b *Lecteur) worldPositionRange() profile.Vec3Range { return b.p.Mouvement.Range }

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

// absAxisWFor retourne la largeur de l axe i pour l index de plage idx.
//
// LARGEURS D AXE PAR INDEX DE PLAGE DE REPLICATION — LE SAVOIR, ET DESORMAIS LA LECTURE.
//
// SOURCE, DESASSEMBLAGE : `FUN_14076e524(out, reader, outIndexPtr, LEVEL)` choisit ses trois
// largeurs dans DEUX tables distinctes selon l'index lu au flux :
//
//	index == -1 (bit de porte pose)  ->  DAT_1445cc9e0 + LEVEL*0xc                (table DEFAUT)
//	index >= 0                       ->  DAT_1445ccbe0 + (index*0x20 + LEVEL)*0xc (table PAR INDEX)
//
// et LEVEL est un IMMEDIAT STATIQUE 0x10 = 16 aux neuf sites d'appel du composant de position
// (`MOV R9D,0x10` en 1406d008a, 140f04dd5, 140f04f32, 140f04f80, 140f04fe5, 140f05018,
// 140fb8b33, 140ee7288, 14226a6b8). Les deux tables sont remplies par la MEME loi
// (`FUN_140be9b88`, cf. `profile/loi_largeurs.go`) sur DEUX jeux de bornes : celles du BUILD
// (`+/-20000`, `.rdata`) pour la table defaut, celles de la CARTE pour la table par index.
//
// CE QUE LE LOT 3.4.1 CHANGE, ET C'EST LE LOT ENTIER. La largeur UNIFORME de 14 bits
// (`Movement.AbsoluteAxisW`), qui ecrasait les trois largeurs de la carte des qu'elle etait
// posee — c'est-a-dire toujours, en production — A DISPARU, et `idx` cesse d'etre jete :
//
//	idx == -1  la table DEFAUT au niveau du composant de position, soit `22/22/22` sur ce
//	           build ([profile.LargeursAxeParDefautDuBuild]) — et surtout PAS les largeurs de
//	           la carte ;
//	idx >= 0   la table PAR INDEX de la carte, portee par le descripteur absolu du profil
//	           ([profile.MapQuantEntry.PrecisionAbsolue], pose par `replay`). C'est la MEME
//	           table que le chemin world-object : les deux la lisent, et c'est leur double
//	           implantation qui les avait laisses diverger.
//
// LE COMPTE SE FERME A L'UNITE SUR LA MESURE QUI FAISAIT AUTORITE, et il ne se fermait pas
// avant : la capture CE du dispatch donne i0 du bipede a 47 bits, une seule valeur distincte,
// 100 % de 154 158 dispatches, et
//
//	1 bUsePred + 1 bDelta + 1 precHigh + 1 indexSel + 1 IndexW + (13+13+14) + 2 finite = 47
//
// avec les largeurs de Cliffhanger. Avec l'uniforme 14 on lisait 49.
//
// LE CATALOGUE NE PORTE QUE LA PLAGE JOUEE, et c'est une limite ASSUMEE, ecrite ici parce
// qu'elle se voit dans le compte : une carte qui declare plusieurs plages n'a d'entree que pour
// celle de l'arene (`Region`). Un record d'une AUTRE plage est donc lu aux largeurs de
// celle-la — le moins mauvais choix, et le seul qui garde l'alignement du record suivant — puis
// sa position n'est PAS emise (`consumeAbsolutePayload`). L'histogramme
// [Observation.IndexAbsolus] compte les index rencontres : jamais un zero muet.
func absAxisWFor(br *Lecteur, idx, i int) uint {
	if i == 0 {
		br.obs.compterIndexAbsolu(idx)
	}
	if idx < 0 {
		return profile.LargeursAxeParDefautDuBuild(profile.NiveauPositionDObjet)[i]
	}
	return br.worldObjectPrecision().AxisW[i]
}

// dequantWorldAxis dequantizes one absolute quantized axis word (width bits). Deux formes :
//   - AbsDequantRange (défaut) : min + step*(q+0.5) via la plage de l'index (FUN_140c1e978).
//   - AbsDequantCenteredQuantum : (q - 2^(bits-1)) * DeltaQuantum — grille fine centrée sur 0.
//
// LA PLAGE SUIT L'INDEX DEPUIS LE LOT 3.4.1, exactement comme la largeur (`absAxisWFor`) et
// comme chez `FUN_14076e524` : bornes et largeurs voyagent ENSEMBLE, elles sortent de la meme
// AABB. `idx == -1` (porte posee) dequantifie dans la boite monde du BUILD (`+/-20000`) ; tout
// `idx >= 0` dequantifie dans la plage de la CARTE que le profil porte.
func dequantWorldAxis(br *Lecteur, idx int, q uint64, bits uint, axis int) float32 {
	if absDequantMode == AbsDequantCenteredQuantum {
		half := float32(uint64(1) << (bits - 1))
		return (float32(q) - half) * br.deltaQuantum()
	}
	wr := br.worldPositionRange()
	if idx < 0 {
		wr = profile.QuantRangeParDefautDuBuild()
	}
	scale := float32(uint64(1) << bits)
	step := (wr[axis].Max - wr[axis].Min) / scale
	return float32(q)*step + wr[axis].Min + step*quantCenter
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
func readRawVec3(br *Lecteur) {
	for i := 0; i < 3; i++ {
		br.ReadBits(32)
	}
}
