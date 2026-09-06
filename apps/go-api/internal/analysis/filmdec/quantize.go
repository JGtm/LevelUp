package filmdec

import "math/bits"

// quantCenter is the mid-bucket dequantization offset (engine constant
// DAT_143cd84b0 = 0.5f): a quantized value q maps to min + step*(q + 0.5).
// Mirrors FUN_140c1e978.
const quantCenter = 0.5

// AxisRange is the [Min, Max] dequantization range for one axis.
type AxisRange struct {
	Min, Max float32
}

// Vec3Range is the per-axis range for a quantized 3-vector, selected by a
// precision index. Mirrors the engine range table DAT_143b8c6f0 (stride 0x18 =
// 3 axes x {min, max}).
type Vec3Range [3]AxisRange

// Quantization precision ranges recovered from DAT_143b8c6f0.
var (
	// QuantRangeUnit3 — precision 0: +/-3.0 per axis (direction / velocity).
	QuantRangeUnit3 = Vec3Range{{-3, 3}, {-3, 3}, {-3, 3}}
	// QuantRangeNorm — precision 1: +/-0.7 per axis (normalized vector / quaternion component).
	QuantRangeNorm = Vec3Range{{-0.7, 0.7}, {-0.7, 0.7}, {-0.7, 0.7}}
	// QuantRangeWorld100 — precision 2: +/-100.0 per axis (world position).
	QuantRangeWorld100 = Vec3Range{{-100, 100}, {-100, 100}, {-100, 100}}
	// QuantRangeCliffhanger — WRONG per-axis bounds attributed to DAT_14462cbe0[0] on a
	// prior read. It scattered absolutes to X~-928..-676 (hundreds of units off-box).
	// KEPT only as the documented "before" of the range-is-the-bug proof. NOT the runtime
	// value: a live Cheat Engine capture of DAT_14462cbe0[0] gives the SMALL map-local
	// biped box below (span X~113 => 113/2^13 = 0.0138 = the exact oracle quantum).
	QuantRangeCliffhanger = Vec3Range{{-973.867, 179.377}, {-361.439, 1047.008}, {-86.552, 489.092}}
	// QuantRangeCEBiped — the TRUE runtime dequant range for i0 biped absolute positions,
	// captured live from DAT_14462cbe0 index 0 (Cheat Engine, 2026-07-11): a small
	// map-local box X[-41.10,72.11] Y[-56.61,57.21] Z[-84.37,53.18]. The oracle player box
	// x[-6.33,35.70] y[-25.14,27.50] z[-4.20,7.08] is a strict subset. Per-film replication
	// config (map-specific) — a universal decoder must read entry 0 of the film's range
	// table; hardcoded here for map 000d5950 to validate the corrected scale.
	QuantRangeCEBiped = Vec3Range{{-41.10318, 72.10963}, {-56.60697, 57.212566}, {-84.37078, 53.18034}}
)

// ReadQuantizedVec3 reads a 3-component vector, each component on bits bits
// (MSB-first), then dequantizes through rng. Mirrors FUN_140c1e9d4 (read) +
// FUN_140c1e978 (dequantize).
func (b *BitReader) ReadQuantizedVec3(bits uint, rng Vec3Range) [3]float32 {
	var out [3]float32
	scale := float32(uint64(1) << bits)
	for i := 0; i < 3; i++ {
		q := float32(b.ReadBits(bits))
		step := (rng[i].Max - rng[i].Min) / scale
		out[i] = q*step + rng[i].Min + step*quantCenter
	}
	return out
}

// BitLenExport expose bitLen pour les outils de calibration.
func BitLenExport(v uint32) int { return bitLen(v) }

// bitLen mirrors FUN_1406d310c: bit-length helper (NOT a reader). bitLen(6)=3.
//
// DEPLACE ICI le 2026-09-05 (lot E, item E.2) depuis `entity.go`, supprime avec
// `entity_quant.go` : les deux decodeurs de record qu'ils portaient n'avaient aucun
// appelant et visaient, de l'aveu du depot (`components_batch7.go:6-8`), une AUTRE
// fonction du jeu. Trois helpers seulement etaient vivants : celui-ci, `readQuantStat`
// et `quantStatDefaultWidth`.
func bitLen(x uint32) int {
	if x == 0 {
		return 0
	}
	h := 31 - bits.LeadingZeros32(x)
	if x&((1<<uint(h))-1) != 0 {
		return h + 1
	}
	return h
}

// quantStatDefaultWidth is the value-field width for the default config range
// DAT_144706100 = 0x1fff (bitLen 13). Used by the nested parent-mode reads in
// FUN_140c9e990 where the runtime table slot is statically zero.
const quantStatDefaultWidth = 13

// readQuantStat mirrors FUN_1406d3140: a variable-width quantized integer read.
// param3 selects the runtime table slot; statWidth is the value-field width W
// (= bitLen(range)). Bit cost: [probe 1 bit when param3==1] + W value bits + 2
// trailing bits. Result layout: bits[31:30] = trailing2, bits[29:0] = base+value
// (base is DAT_1451f98d0[param3*2], statically 0).
//
// POURQUOI `param3` RESTE UN PARAMETRE alors que les quatre sites de production passent 1 :
// il MODELISE `param_3` de FUN_1406d3140, dont depend le COUT EN BITS de la lecture (la sonde
// d'un bit n'est depensee que pour param3==1). Le figer effacerait cette part de la grammaire
// portee. Le second lecteur, qui passait une autre valeur, vivait dans `entity_quant.go` —
// supprime le 2026-09-05 (lot E, item E.2) parce qu'il n'avait aucun appelant ; c'est cette
// suppression, et elle seule, qui rend le parametre uniforme aujourd'hui.
//
//nolint:unparam // cf. le paragraphe ci-dessus : param_3 est une grandeur de la grammaire.
func (b *BitReader) readQuantStat(param3 int, statWidth uint) uint32 {
	// param3==1 always spends 1 probe bit (the && chain tests param3==1 first).
	if param3 == 1 {
		_ = b.ReadBit() // probe (FUN_1406cf008); selects a special range slot at runtime
	}
	var value uint32
	if statWidth >= 1 {
		value = uint32(b.ReadBits(statWidth)) // W value bits MSB-first
	}
	top2 := uint32(b.ReadBits(2)) // always 2 trailing bits
	const base = 0                // DAT_1451f98d0[param3*2], statically 0
	return (top2 << 30) | (uint32(base) + value)
}
