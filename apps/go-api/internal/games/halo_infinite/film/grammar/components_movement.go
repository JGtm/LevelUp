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
func consumeObjectTranslationalVelocity(br *Lecteur) {
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
// CORRECTION DE MODÈLE, lot R7-c (2026-08-17). Ce champ portait auparavant les DEUX
// globaux de `FUN_14076f91c` à la fois, ce qui était faux : `DAT_144e61ea0` est une
// PORTÉE (cf. keyframeBaselineScope) et `DAT_145121140` un réglage de process, et ils
// ne gardent PAS les mêmes lecteurs. Sous la seule portée baseline, `i49`
// (`FUN_14107166c`), `i2 forward-and-up` (`FUN_140c5f938`) et le bloc MPP
// (`FUN_14080cfe8`) — qui lisent `DAT_145121140` SEUL — ne bougent pas.
//
// C'ÉTAIT LA VARIABLE DE PAQUET `PositionFullPrecision` JUSQU'AU LOT 2.2.a : elle vient
// désormais du PROFIL que le lecteur porte ([Lecteur.poserMouvement]).
func (b *Lecteur) fullPrecision() bool { return b.p.Mouvement.FullPrecision }

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
// C'ÉTAIT LA VARIABLE DE PAQUET `keyframeBaselineScope` JUSQU'AU LOT 2.3 : la portée vit dans
// [GrammaireBalayage.PorteeBaseline], que le lecteur porte.

// fullPrecisionGate porte `FUN_14076f91c` : `DAT_144e61ea0 != 0 || DAT_145121140 == 1`.
// Zéro bit consommé — c'est un prédicat de CONTEXTE, jamais un bit du flux.
//
// LA PORTÉE ET LE RÉGLAGE VIENNENT DE DEUX ENDROITS DU PROFIL, et c'est voulu : la portée
// (`DAT_144e61ea0`) est une BASCULE DE GRAMMAIRE que les lecteurs d'état complet lèvent autour
// d'un appel ; le réglage (`DAT_145121140`) est une valeur de MOUVEMENT, arrivée au lot 2.2.a.
func fullPrecisionGate(br *Lecteur) bool {
	return br.p.Grammaire.PorteeBaseline || br.fullPrecision()
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

// keyframeWriterI0Grammar route le chemin ABSOLU d'i0 sur la grammaire que l'ECRIVAIN d'état
// complet du jeu pose, et que le lecteur du jeu relit — les deux disent la même chose CONTRE
// le port (lot R7-d, `WALK_PORT_NOTES.md` section « i0 — LE LECTEUR DU JEU DIT LA MÊME CHOSE
// QUE L'ÉCRIVAIN ») :
//
//	écrivain FUN_14320678c -> FUN_14320696c -> FUN_142e2d86c ; lecteur FUN_14076e29c ->
//	FUN_14076e420 : le 3e bit ne SUPPRIME pas la charge utile (il choisit la table de plage
//	DAT_143b8c6d0 = ±100) et il est la PORTE DE LA QUEUE DE HANDLE ; le champ de 2 bits est
//	INCONDITIONNEL et vient EN DERNIER, après la queue.
//
// Le port actuel fait `if precHigh { return }` (0 bit de charge) et force la queue à false :
// une sous-lecture de `1 + [idxW] + 3 x axisW` bits dès que ce bit vaut 1.
//
// KILL-SWITCH — défaut OFF posé le 2026-08-17 (lot R7-e) : la correction n'est pas encore
// prouvée bit-exacte sur l'oracle de frontière, et le chemin absolu d'i0 sert la trajectoire
// de PRODUCTION du rejeu 2D. Critère de bascule du défaut : atterrissage bit-exact en hausse
// sur les 591 records `ti=35` bornés ET non-régression delta verte. Retrait de la bascule (une
// seule grammaire, celle du jeu) visé à la clôture du chantier image-clé, au plus tard le
// 2026-10-31 — si le critère n'est pas tenu d'ici là, c'est le port qu'il faut rouvrir, pas la
// bascule qu'il faut prolonger.
// C'ÉTAIT LA VARIABLE DE PAQUET `keyframeWriterI0Grammar` JUSQU'AU LOT 2.3 : la bascule vit
// dans [GrammaireBalayage.GrammaireEcrivainI0], que le lecteur porte.
