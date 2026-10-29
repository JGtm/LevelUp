package grammar

// Object MOVEMENT bit-consumers (BIPED archetype #35, shared with #40 i0..i9):
//   i0 object-position-dynamic-precision          (FUN_1406cfe44)
//   i1 object-translational-velocity-dyn-precision (FUN_14076d45c -> FUN_14076d4d0)
//   i3 object-angular-velocity-dyn-precision        (FUN_140d87740 -> FUN_14076e1c8)
//
// All three follow the dynamic-precision quantized-vector family. i1 and i3 are
// STATICALLY bit-exact (fixed widths confirmed by asm at the call sites). i0's
// VALUE widths come from a runtime precision descriptor (FUN_140be9a14 populates
// DAT_1445cc9e0 axis widths + DAT_144632be0 index width at map load), so its
// payload widths are passed in via profile.PrecisionDescriptor; only the control spine is
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

// consumeDynPrecVec3 mirrors FUN_14076d528: a leading R(1) present flag (bit==0
// => present), then on the present path R(mag) packed direction + R(scale) log/exp
// magnitude. bit==1 => absent (vector is the engine constant, no further bits).
//
// LE CROCHET DE CAPTURE `dynPrecHook` A ETE SUPPRIME le 2026-09-05 (lot E, item E.2), avec son
// setter `SetDynPrecHook` et les blocs de sauvegarde/restauration qui le promenaient : aucun
// site du depot ne l'installait non-nil, tests compris. Il etait donc prouvablement toujours
// nil, et sa capture — additive, sans effet sur la consommation de bits — n'emettait rien.
// La consommation de bits de ce deser est inchangee, a la ligne pres.
func consumeDynPrecVec3(br *Lecteur, mag, scale uint) { //nolint:unparam // magnitude de grammaire ecrite au site d appel ; le lot 2.2 la porte au profil (2026-09-17, fusion 2.7g : la scission a sorti ce site de la baseline lint)
	if br.ReadBit() { // FUN_14076d528 leading R(1); JNZ -> absent (0 payload bits)
		return
	}
	br.ReadBits(mag)   // packed direction (feeds FUN_1406d8288 unpack, 0 extra bits)
	br.ReadBits(scale) // FUN_14076d6dc log/exp scale word
}

// consumeObjectTranslationalVelocity (i1) mirrors FUN_14076d45c -> FUN_14076d4d0.
//
//	outer = R(1)
//	if outer == 1 : R(96)  (FUN_1406d676c 0x60 raw full-precision vec3)  [keep path]
//	if outer == 0 : consumeDynPrecVec3(mag=19, scale=10)                 [delta path]
//
// Bit cost: outer==1 -> 97 ; outer==0 & present -> 31 ; outer==0 & absent -> 2.
// LES QUATRE VALEURS SONT PUBLIEES DEPUIS LE LOT 5.3.4 (2026-09-21) : la vitesse a l instant est
// ce qui distingue une montee (composante verticale) d un deplacement au sol, et ce qui ferait
// voir un sprint (seconde bosse de l histogramme de vitesse au sol). La consommation de bits est
// INCHANGEE (cf. `etats_mouvement_hooks.go` pour la forme de la tranche).
func consumeObjectTranslationalVelocity(br *Lecteur) {
	if br.ReadBit() { // FUN_14076d45c R(1); set -> FUN_14076d4d0 mode 2 (keep)
		br.ReadBits(rawVec3Bits) // FUN_1406d676c(...,0x60) = R(96)
		br.publishEtatMouvement(EtatVitesse, 1, 0, 0, 0)
		return
	}
	porte, dir, ech := consumeDynPrecVec3Lu(br, velocityMagBits, velocityScaleBits)
	br.publishEtatMouvement(EtatVitesse, 0, bit2u(porte), dir, ech)
}

// consumeDynPrecVec3Lu est [consumeDynPrecVec3] qui REND ce qu il a lu : la porte, la direction
// empaquetee et le mot d echelle. Meme consommation de bits, au bit pres.
func consumeDynPrecVec3Lu(br *Lecteur, mag, scale uint) (porte bool, dir, ech uint64) {
	if porte = br.ReadBit(); porte { // FUN_14076d528 leading R(1); JNZ -> absent
		return porte, 0, 0
	}
	dir = br.ReadBits(mag)   // packed direction (feeds FUN_1406d8288 unpack, 0 extra bits)
	ech = br.ReadBits(scale) // FUN_14076d6dc log/exp scale word
	return porte, dir, ech
}

// consumeObjectAngularVelocity (i3) mirrors FUN_140d87740 -> FUN_14076e1c8. Same
// shape as translational velocity; the scale width is 8 (not 10).
//
//	outer = R(1)
//	if outer == 1 : R(96)                                  [keep path]
//	if outer == 0 : consumeDynPrecVec3(mag=19, scale=8)    [delta path]
//
// Bit cost: outer==1 -> 97 ; outer==0 & present -> 29 ; outer==0 & absent -> 2.
func consumeObjectAngularVelocity(br *Lecteur) {
	if br.ReadBit() { // FUN_140d87740 R(1); set -> FUN_14076e1c8 mode 2 (keep)
		br.ReadBits(rawVec3Bits) // FUN_1406d676c(...,0x60) = R(96)
		return
	}
	consumeDynPrecVec3(br, angularMagBits, angularScaleBits)
}

// fullPrecision mirroite le SEUL global `DAT_145121140` : la configuration
// « réplication haute précision » du process. NOT a bitstream bit. Default false
// (retail high-prec path inactive).
//
// CE N'EST PAS LA PORTÉE `DAT_144e61ea0` ([Lecteur.portee]), et les deux ne gardent PAS les
// mêmes lecteurs : `i49` (`FUN_14107166c`), `i2 forward-and-up` (`FUN_140c5f938`) et le bloc
// MPP (`FUN_14080cfe8`) lisent `DAT_145121140` SEUL — la portée ne les change pas.
//
// C'ÉTAIT LA VARIABLE DE PAQUET `PositionFullPrecision` JUSQU'AU LOT 2.2.a : elle vient
// désormais du PROFIL que le lecteur porte ([Lecteur.poserMouvement]).
func (b *Lecteur) fullPrecision() bool { return b.p.Mouvement.FullPrecision }

// fullPrecisionGate porte `FUN_14076f91c` : `DAT_144e61ea0 != 0 || DAT_145121140 == 1`, soit la
// PORTÉE de la lecture d'état complet ([Lecteur.portee]), que le lecteur porte, ou le RÉGLAGE
// de process ([Lecteur.fullPrecision]), que son profil porte. Zéro bit consommé — c'est un
// prédicat de CONTEXTE, jamais un bit du flux. Vrai, les lecteurs de position du moteur lisent le
// vecteur BRUT de 96 bits (`FUN_1411b259c` = `FUN_1406d676c(.., 0x60)`) au lieu du quantifié.
func fullPrecisionGate(br *Lecteur) bool {
	return br.portee || br.fullPrecision()
}

// deltaHasHandleTail mirrors the runtime field bVar16 = (precIndex != -1)
// that gates the i0 predicted-delta handle tail in FUN_1406cfe44. It is NOT a
// bitstream bit: precIndex lives at precDesc+0x10 in the entity's previous position
// state (RAM), populated at map load from the film's replication config (reads 0/-1
// statically — same limitation as the traversal descriptor). Default false = the dominant
// precIndex==-1 case (no tail). The CE delta capture confirms whether it ever fires.
//
// C'ÉTAIT LA VARIABLE DE PAQUET `PositionDeltaHasHandleTail` JUSQU'AU LOT 2.2.a.
func (b *Lecteur) deltaHasHandleTail() bool { return b.p.Mouvement.DeltaHasHandleTail }

// calibratedSkip active la calibration intelligente d'i0 (saut au total CE 47/101 selon
// bUsePred) au lieu du deser dont la précision d'axe runtime n'est pas sourcée statiquement.
// Harness de validation map-spécifique (Cliffhanger). Default false.
//
// C'ÉTAIT LA VARIABLE DE PAQUET `PositionCalibratedSkip` JUSQU'AU LOT 2.2.a.
func (b *Lecteur) calibratedSkip() bool { return b.p.Mouvement.CalibratedSkip }

// DeltaQuantum est le pas (unité monde) d'UN cran de position répliqué en DELTA par i0. La
// famille delta (signed-8 ou axis-width) code un NOMBRE DE CRANS signé ; le pas physique est ce
// quantum, PROPRE au chemin delta et DISTINCT de la range absolue (Cliffhanger ~[-974,179] / 2^6
// = 18 u = FAUX pour un delta). Valeur par défaut = quantum de grille MESURÉ sur l'oracle CE
// (différence minimale non nulle entre positions consécutives, identique sur X/Y/Z et tous les
// slots). Le reglage public `SetDeltaQuantum` a ete supprime le 2026-09-05 (lot E, item E.2) :
// aucun appelant. La range delta pour le chemin axis-width vaut DeltaQuantum * 2^AxisW
// (centree 0).
// C'ÉTAIT UNE VARIABLE DE PAQUET JUSQU'AU LOT 2.2.b : le quantum vit dans le PROFIL que le
// lecteur porte (`Movement.DeltaQuantum`).
func (b *Lecteur) deltaQuantum() float32 { return b.p.Mouvement.DeltaQuantum }
