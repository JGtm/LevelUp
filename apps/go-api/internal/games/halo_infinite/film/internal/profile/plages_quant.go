package profile

// plages_quant.go — LES PLAGES DE DEQUANTIFICATION, COMME VALEURS.
//
// EXTRAITES DE `grammar/quantize.go` AU LOT 2.5.b. La table de plages du moteur
// (`DAT_143b8c6f0`, stride 0x18 = 3 axes x {min, max}) est une DONNEE relue dans le jeu ; le
// LECTEUR qui s en sert (`Lecteur.ReadQuantizedVec3`, sa constante de centre de casier et les
// helpers de largeur variable) reste en `grammar`.
//
// POURQUOI ELLES DESCENDENT : [MapQuantEntry.Range] et [MovementProfile.Range] sont de ce type.
// Sans lui, la couche `profile` aurait importe `grammar` — l import vers le haut que R1 du
// ratchet des couches refuse, et exactement le blocage que le lot leve.

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

// QuantRangeParDefautDuBuild rend la plage de la table DEFAUT du moteur : les bornes
// `DAT_143b8c6b8` de `.rdata` (`+/-20000` sur les trois axes), recopiees en `DAT_1445cc9c8` par
// le remplisseur `FUN_140be9a14`.
//
// C EST LA PLAGE DU CHEMIN `index == -1`, ET D AUCUN AUTRE. `FUN_14076e524` ne la choisit que
// lorsque le bit de porte est pose ; tout `index >= 0` adresse `DAT_14462cbe0 + index*0x18`,
// c est-a-dire une plage REELLE de la carte. La boite de 40 000 unites est celle des objets
// hors carte, pas un repli neutre : dequantifier une position de carte dedans la deplacerait de
// plusieurs milliers d unites.
//
// Elle est une FONCTION et non une `var` : une plage exportee par valeur mutable serait un etat
// de paquet que n importe qui pourrait reecrire, et la couche `profile` porte des DONNEES.
func QuantRangeParDefautDuBuild() Vec3Range {
	b := BornesParDefautDuBuild()
	var r Vec3Range
	for axe := 0; axe < 3; axe++ {
		r[axe] = AxisRange{Min: b[axe][0], Max: b[axe][1]}
	}
	return r
}
