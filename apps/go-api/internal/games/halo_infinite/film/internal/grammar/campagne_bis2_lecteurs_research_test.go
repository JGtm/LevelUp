//go:build research && campagne_overlay

package grammar

// campagne_bis2_lecteurs_research_test.go — compagnon de `campagne_bis2_vehicules_research_test.go`
// (MESURES BIS 2 DE LA CAMPAGNE DE GRAMMAIRE, 2026-10-01) : les lecteurs de recherche des
// composants `ti=40` (note T6 §4) et `ti=43` (note T7 §5), et le crochet d interception qui les
// branche ([bis2Intercepteur]). Deplacement pur, decoupe pour le seuil de 500 lignes par fichier ;
// aucun comportement change. Exige la meme surcouche de recherche (tag `campagne_overlay`).

// --- les lecteurs de recherche ------------------------------------------------------------------

// b2vPorte : la porte `+0x818` que le crochet applique a i33/i34 (vrai = posee).
var b2vPorte bool

// b2vLireTi40 lit les composants `ti=40` non portes (note T6 §4). `pris` faux : la production lit.
func b2vLireTi40(br *Lecteur, name string) (pris, porte bool) {
	switch name {
	case "vehicle-auto-turret-triggers-component": // i30 FUN_142f04994
		br.ReadBits(3)
	case "vehicle-auto-turret-aiming-vector-component": // i31 FUN_14115f33c
		br.ReadBits(19)
	case "vehicle-transformed-or-desired-open-state-changed-component": // i32 FUN_142f04b70
		br.ReadBits(1)
		br.ReadBits(8)
	case "vehicle-type-state-component": // i33 FUN_142f02474 -> FUN_14320c4c8, sous la porte
		if b2vPorte {
			if v := br.ReadBits(2); v == 1 || v == 3 {
				br.ReadBits(6)
			}
		}
	case compVehicleTypePhysics: // i34 FUN_142f02498, sous la porte
		if b2vPorte {
			consumeVehicleTypePhysics(br)
		}
	case "vehicle-auto-turret-target-component": // i35 FUN_1408f0ac4(cat 1)
		consume1408f0ac4(br, 1)
	case "vehicle-sentry-state-component": // i36
		br.ReadBits(3)
		br.ReadBits(1)
	case "vehicle-weapon-set-component": // i38 -> FUN_1406d01fc
		br.ReadBits(3)
		for k := 0; k < 2; k++ {
			if !br.ReadBit() {
				br.ReadBits(2)
			}
		}
	case "vehicle-auto-turret-component": // i39
		br.ReadBits(2)
	case "vehicle-equipment-turret-parent-component": // i40 FUN_1408f0ac4(cat 0)
		consume1408f0ac4(br, 0)
	case "vehicle-seats-override-pitch-component", "vehicle-seats-override-yaw-component": // i41, i42
		br.ReadBits(16)
	case "air-drop-flight-component": // i45
		br.ReadBits(2)
		br.ReadBits(14)
		br.ReadBits(8)
	case "warp-component": // i46
		a, b := br.ReadBit(), br.ReadBit()
		if a {
			br.ReadBits(8)
		}
		if b {
			br.ReadBits(8)
		}
	case "vehicle-low-frequency-component": // i47
		if !br.ReadBit() {
			br.ReadBits(5)
		}
	default:
		return false, false
	}
	return true, true
}

// b2vLireTi43 lit les composants `device-*` non portes (note T7 §5). Le compte d i31 au-dela de 8
// fait rendre 0 au lecteur du jeu, qui tue le record : desynchronisation declaree.
func b2vLireTi43(br *Lecteur, name string) (pris, porte bool) {
	switch name {
	case "device-position-animation-name-component": // i19
		br.ReadBits(32)
		br.ReadBits(10)
	case "device-position-animation-control-component": // i20
		for k := 0; k < 4; k++ {
			br.ReadBits(64)
		}
	case "device-position-group-component": // i21
		br.ReadBits(32)
		if br.ReadBit() {
			br.ReadBits(8)
		}
	case "device-power-component", "device-power-group-component": // i22, i23
		br.ReadBits(14)
	case "device-interaction-in-progress-component", "device-exclusive-user-component": // i24, i29
		consume1408f0ac4(br, 1)
	case "device-interaction-hold-time-component", "device-interaction-start-time-override-component": // i25, i40
		br.ReadBits(8)
	case "device-control-action-string-override-component": // i26
		br.ReadBits(32)
		br.ReadBits(32)
	case "device-health-station-charges-component": // i27
		br.ReadBits(1)
		br.ReadBits(6)
	case "device-health-station-in-use-component", "device-in-primary-mode-component",
		"device-dispenser-require-los-component": // i28, i30, i33
		br.ReadBits(1)
	case "device-dispenser-monitors-changed-component": // i31
		n := br.ReadBits(8)
		if n > 8 {
			return true, false
		}
		for k := uint64(0); k < n; k++ {
			consume1408f0ac4(br, 1)
		}
	case "device-dispenser-state-flags-component": // i32
		br.ReadBits(5)
	case "device-animation-layer-settings-component": // i34
		if br.ReadBit() {
			for k := 0; k < 8; k++ {
				if br.ReadBit() {
					br.ReadBits(32)
					br.ReadBits(14)
					br.ReadBits(14)
					br.ReadBits(10)
					br.ReadBits(14)
					br.ReadBits(2)
				}
			}
		}
	case "device-animation-layer-state-component": // i35
		if br.ReadBit() {
			for k := 0; k < 8; k++ {
				if br.ReadBit() {
					if br.ReadBits(10) != 0 {
						br.ReadBits(14)
					}
				}
			}
		}
	case "device-dispenser-state-component": // i36
		for k := 0; k < 2; k++ {
			if br.ReadBit() {
				consume1408f0ac4(br, 0)
			}
			br.ReadBits(3)
		}
	case "device-object-dispenser-timer-component": // i37
		for k := 0; k < 2; k++ {
			if br.ReadBit() {
				br.ReadBits(10)
				br.ReadBits(10)
				br.ReadBits(5)
				br.ReadBits(10)
			}
		}
	case "device-position-transition-velocity-component": // i38
		br.ReadBits(18)
	case "device-machine-flags-component": // i39
		br.ReadBits(9)
	default:
		return false, false
	}
	return true, true
}

// b2vCrochet rend le crochet d interception : ti=40 et/ou ti=43.
func b2vCrochet(ti40, ti43 bool) func(br *Lecteur, name string, typeIndex, level uint32) (bool, bool) {
	return func(br *Lecteur, name string, typeIndex, _ uint32) (bool, bool) {
		switch {
		case ti40 && typeIndex == 40:
			return b2vLireTi40(br, name)
		case ti43 && typeIndex == 43:
			return b2vLireTi43(br, name)
		}
		return false, false
	}
}
