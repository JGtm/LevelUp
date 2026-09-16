package grammar

import (
	"levelup/go-api/internal/games/halo_infinite/film/profile"
	"math/bits"
)

// quantCenter is the mid-bucket dequantization offset (engine constant
// DAT_143cd84b0 = 0.5f): a quantized value q maps to min + step*(q + 0.5).
// Mirrors FUN_140c1e978.
const quantCenter = 0.5

// ReadQuantizedVec3 reads a 3-component vector, each component on bits bits
// (MSB-first), then dequantizes through rng. Mirrors FUN_140c1e9d4 (read) +
// FUN_140c1e978 (dequantize).
func (b *Lecteur) ReadQuantizedVec3(bits uint, rng profile.Vec3Range) [3]float32 {
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
// fonction du jeu. Deux helpers seulement sont vivants : celui-ci et `readQuantStat`
// (la constante `quantStatDefaultWidth` est morte le 2026-09-15 : la largeur se derive
// desormais de la CATEGORIE, cf. `varwidth.go`).
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

// readQuantStat porte FUN_1406d3140 quand l appelant veut la VALEUR : meme lecture que
// `readVarWidthInt`, rendue au format du jeu.
//
// `param3` est la CATEGORIE (`param_3` du jeu) : elle choisit la plage dans la table de
// `FUN_140d10bb0`, donc la largeur du champ de valeur (cf. `varwidth.go`). Cout en bits :
// [sonde R(1) si param3 == 1] + `varWidthBits` bits de valeur + R(2) de queue. Quand la sonde
// de la categorie 1 rend 1, le jeu BASCULE sur l entree 4 et ne lit plus que 9 bits de valeur.
//
// Format rendu : bits[31:30] = les deux bits de queue, bits[29:0] = la valeur.
//
// LA BASE DE LA TABLE N EST PAS AJOUTEE, ET C EST DELIBERE : `FUN_1406d3140` rend
// `(queue << 30) | (base + valeur)` avec une base de 0x200 / 0x300 / 0x400 selon la categorie.
// Elle ne change AUCUN bit lu, elle decale des IDENTIFIANTS publies — un changement qui se juge
// au gate de decodage, indisponible pour ce lot. Consigne au §4 du PLAN_DECODEUR_FILM.
func (b *Lecteur) readQuantStat(param3 int) uint32 {
	if param3 == varWidthProbeCategory && b.ReadBit() { // sonde : param_3 == 1 SEULEMENT
		param3 = varWidthProbeSlot
	}
	var value uint32
	if w := varWidthBits(param3); w >= 1 {
		value = uint32(b.ReadBits(w)) // W bits de valeur, MSB en tete
	}
	top2 := uint32(b.ReadBits(2)) // toujours deux bits de queue
	return (top2 << 30) | value
}
