package replay

// ground_weapon_pads_releve_test.go — LE RELEVE D UN SOCLE HORS DE L EMPRISE JOUEE.
//
// Chaque test nomme la mutation qui doit le faire rougir.

import (
	"slices"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// releveOrigine / releveStep : l horloge des temoins (frames de 100 ms).
const (
	releveOrigine = 1_000_000
	releveStep    = 100_000
)

// empriseDeTemoin : une emprise ARMEE sur une zone jouee de 30 x 30 m, z de 0 a 10 m.
func empriseDeTemoin() empriseJouee {
	var xs, ys, zs []float32
	for i := range 300 {
		xs, ys, zs = append(xs, float32(i%30)), append(ys, float32(i/10)), append(zs, float32(i%11))
	}
	slices.Sort(xs)
	slices.Sort(ys)
	slices.Sort(zs)
	return empriseDesAxes(xs, ys, zs)
}

// priseDatee : une occupation du socle 0 datee a la frame `t`, et le ramassage natif qui la date.
func priseDatee(t int, slot uint32) (PadPickup, types.BipedPickup) {
	ft := t
	return PadPickup{Pad: 0, T: &ft},
		types.BipedPickup{Slot: slot, CatalogID: 0xb533957e, TimestampUS: releveOrigine + uint64(t)*releveStep + 5_000}
}

// memePosition : deux socles a la meme position.
func memePosition(a, b WeaponPad) bool { return a.X == b.X && a.Y == b.Y && a.Z == b.Z }

// releveDeTemoin : la regle jouee sur un socle et des prises datees dont les ramasseurs se tiennent
// aux positions `lieux` (un par prise, dans l ordre).
func releveDeTemoin(pad WeaponPad, lieux [][3]float32) (WeaponPad, GroundWeaponCoverage) {
	var picks []PadPickup
	var natifs []types.BipedPickup
	posDe := map[uint32][3]float32{}
	for i, l := range lieux {
		slot := uint32(512 + i)
		k, n := priseDatee(10*(i+1), slot)
		picks, natifs, posDe[slot] = append(picks, k), append(natifs, n), l
	}
	pads := []WeaponPad{pad}
	var cov GroundWeaponCoverage
	releverLesSoclesHorsEmprise(pads, picks, entreesDuReleve{
		natifs: natifs,
		position: func(slot uint32, _ uint64) (x, y, z float32, ok bool) {
			p, ok := posDe[slot]
			return p[0], p[1], p[2], ok
		},
		emprise: empriseDeTemoin(),
		clock:   replayClock{origin: releveOrigine, step: releveStep},
	}, &cov)
	return pads[0], cov
}

// TestUnSocleHorsEmpriseSeReleveAuLieuDesPrises — le socle cree a 500 m sous la zone jouee se pose au
// centroide de ses deux ramasseurs.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer l affectation de la position dans
// [releverLesSoclesHorsEmprise].
func TestUnSocleHorsEmpriseSeReleveAuLieuDesPrises(t *testing.T) {
	got, cov := releveDeTemoin(WeaponPad{X: 0, Y: 0, Z: -500, Weapon: "0xB533957E"},
		[][3]float32{{10, 10, 5}, {10.4, 10.2, 5}})
	if got.X != 10.2 || got.Y != 10.1 || got.Z != 5 {
		t.Errorf("socle en (%v, %v, %v), attendu au centroide des prises (10.2, 10.1, 5)", got.X, got.Y, got.Z)
	}
	if cov.HorsEmprise != 1 || cov.Releves != 1 || cov.PlusieursLieux != 0 {
		t.Errorf("couverture %d / %d / %d, attendu 1 / 1 / 0", cov.HorsEmprise, cov.Releves, cov.PlusieursLieux)
	}
}

// TestDesPrisesEnPlusieursLieuxNeRelevePas — le meme point de creation sert deux socles de la carte :
// aucun releve, et la raison se compte.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer le controle `dist3(l, c) >= originDropMaxDist` de
// [centroideDUnSeulLieu].
func TestDesPrisesEnPlusieursLieuxNeRelevePas(t *testing.T) {
	pad := WeaponPad{X: 0, Y: 0, Z: -500, Weapon: "0xB533957E"}
	got, cov := releveDeTemoin(pad, [][3]float32{{10, 10, 5}, {25, 20, 5}})
	if !memePosition(got, pad) {
		t.Errorf("socle deplace en (%v, %v, %v) : deux lieux de prise ne designent pas un socle", got.X, got.Y, got.Z)
	}
	if cov.HorsEmprise != 1 || cov.Releves != 0 || cov.PlusieursLieux != 1 {
		t.Errorf("couverture %d / %d / %d, attendu 1 / 0 / 1", cov.HorsEmprise, cov.Releves, cov.PlusieursLieux)
	}
}

// TestUneSeulePriseNeReleveRien — une occupation datee ne fait pas une recurrence.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : abaisser le seuil a une prise.
func TestUneSeulePriseNeReleveRien(t *testing.T) {
	pad := WeaponPad{X: 0, Y: 0, Z: -500, Weapon: "0xB533957E"}
	got, cov := releveDeTemoin(pad, [][3]float32{{10, 10, 5}})
	if !memePosition(got, pad) || cov.HorsEmprise != 1 || cov.Releves != 0 {
		t.Errorf("socle (%v, %v, %v), couverture %+v : une seule prise ne localise pas un socle",
			got.X, got.Y, got.Z, cov)
	}
}

// TestUnSocleDansLEmpriseNeBougePas — la regle ne touche ni un socle dans la zone jouee, ni un socle
// de bonus.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer la garde d emprise (ou celle de la famille d arme).
func TestUnSocleDansLEmpriseNeBougePas(t *testing.T) {
	for _, pad := range []WeaponPad{
		{X: 12, Y: 12, Z: 5, Weapon: "0xB533957E"},
		{X: 0, Y: 0, Z: -500, Weapon: "powerup_camo"},
	} {
		got, cov := releveDeTemoin(pad, [][3]float32{{10, 10, 5}, {10.4, 10.2, 5}})
		if !memePosition(got, pad) || cov.HorsEmprise != 0 {
			t.Errorf("%s : socle (%v, %v, %v), couverture %+v — attendu intact et hors du compte",
				pad.Weapon, got.X, got.Y, got.Z, cov)
		}
	}
}
