package grammar

// rcomb2_leviers.go — SURCOUCHE DE MESURE R-COMB-2 (campagne de grammaire, 2026-10-02). Fichier
// AJOUTE par la surcouche unique (il n existe pas dans le depot) ; outillage de MESURE seulement.
//
// Deux bascules, INERTES par defaut (aucune variable d environnement : lecture de production) :
//
//	rc2L7                    L7, « NEW sur slot occupe » : un NEW traverse proprement qui contredit
//	                         une entite vivante est LIE au lieu d etre refuse (frame_infer.go de la
//	                         surcouche). Posee par la sonde, ou par CAMPAGNE_RCOMB2_L7=1 (binaires).
//	CAMPAGNE_RCOMB2_KS       liste de leviers de composant poses AU CHARGEMENT du paquet, pour les
//	                         binaires (cmd/killsource) qui n ont pas de crochet de test : L8, L2,
//	                         L3a, L4a, L6b, WO (site world-object-i0 au jeu), L6a (avec
//	                         CAMPAGNE_RCOMB2_KS_BORNES = `idx:minx,miny,minz,maxx,maxy,maxz;...`).
//
// Les lecteurs ci-dessous sont des RECOPIES TEXTUELLES des lecteurs des sondes (seuls les noms
// changent) : b2vLireTi40 / b2vLireTi43 (campagne_bis2_lecteurs_research_test.go),
// b3CrochetTi3(true, true) et b3LireBasseFrequence (campagne_bis3_ti3_research_test.go), le
// variant « moteur » de rl3Crochet (r_comp_l3_research_test.go) et reap37LireVolumes /
// reap37LireLetterbox / reap37LireBassin (reapparition_37_bassin_film_lecteurs_test.go, bits
// consommes seulement).

import (
	"os"
	"strconv"
	"strings"
)

// rc2L7 : bascule L7 (cf. l en-tete).
var rc2L7 = os.Getenv("CAMPAGNE_RCOMB2_L7") == "1"

// rc2ksPorte : la porte `+0x818` des composants i33/i34 de `ti=40` (L4a la pose).
var rc2ksPorte bool

func init() {
	brut := os.Getenv("CAMPAGNE_RCOMB2_KS")
	if brut == "" {
		return
	}
	on := map[string]bool{}
	for _, x := range strings.Split(brut, ",") {
		on[strings.TrimSpace(x)] = true
	}
	rc2ksPorte = on["L4a"]
	if on["L8"] || on["L2"] || on["L4a"] || on["L3a"] {
		bis2Intercepteur = func(br *Lecteur, name string, typeIndex, level uint32) (bool, bool) {
			switch {
			case on["L8"] && typeIndex == 3:
				return rc2ksTi3(br, name)
			case on["L2"] && typeIndex == 43:
				return rc2ksTi43(br, name)
			case on["L4a"] && typeIndex == 40:
				return rc2ksTi40(br, name)
			case on["L3a"]:
				return rc2ksMoteur(br, name, level)
			}
			return false, false
		}
	}
	bis2SitesJeu[bis2SiteFlockPosition] = on["L6b"]
	bis2SitesJeu[bis2SiteDisplayAsset] = on["L6b"]
	bis2SitesJeu[bis2SiteWorldObject] = on["WO"]
	if on["L6a"] {
		if b := rc2ksBornes(os.Getenv("CAMPAGNE_RCOMB2_KS_BORNES")); len(b) > 0 {
			bis2C3 = &bis2EtatC3{parIndex: true, bornes: b}
		}
	}
}

// rc2ksBornes : recopie de b2pBornes (campagne_bis2_positions_research_test.go).
func rc2ksBornes(brut string) map[int][3][2]float32 {
	if brut == "" {
		return nil
	}
	out := map[int][3][2]float32{}
	for _, morceau := range strings.Split(brut, ";") {
		idx, reste, ok := strings.Cut(morceau, ":")
		if !ok {
			continue
		}
		i, err := strconv.Atoi(strings.TrimSpace(idx))
		if err != nil {
			continue
		}
		var v [6]float32
		for k, x := range strings.Split(reste, ",") {
			if k < 6 {
				f, _ := strconv.ParseFloat(strings.TrimSpace(x), 32)
				v[k] = float32(f)
			}
		}
		out[i] = [3][2]float32{{v[0], v[3]}, {v[1], v[4]}, {v[2], v[5]}}
	}
	return out
}

// rc2ksTi3 : b3CrochetTi3(true, true) sur `ti=3` (L8).
func rc2ksTi3(br *Lecteur, name string) (bool, bool) {
	switch {
	case name == "low-frequency":
		return true, rc2ksBasseFrequence(br)
	case name == compHighFrequency:
		br.ReadBits(16)
		br.ReadBits(8)
		br.ReadBits(2)
		return true, true
	}
	return false, false
}

// rc2ksBasseFrequence : b3LireBasseFrequence (FUN_142ed4aec).
func rc2ksBasseFrequence(br *Lecteur) bool {
	lireE494(br, niveauPosition)
	consumeObjectForwardAndUp(br)
	br.ReadBits(16)
	br.ReadBits(8)
	br.ReadBits(2)
	n := br.ReadBits(6)
	for k := uint64(0); k < n; k++ {
		f := br.ReadBits(3)
		if f&1 != 0 {
			lireE494(br, niveauPosition)
		}
		if f&2 != 0 {
			consumeObjectForwardAndUp(br)
		}
		br.ReadBits(16)
		br.ReadBits(5)
	}
	return true
}

// rc2ksMoteur : rl3Crochet, variante « moteur » (bassin et moteur, sans decalage ni hypothese).
func rc2ksMoteur(br *Lecteur, name string, level uint32) (bool, bool) {
	switch {
	case name == "managed-engine-timers-component":
		rc2ksBassin(br)
		return true, true
	case name == "game-engine-soft-ceilings-component":
		br.Skip(128)
	case name == "game-engine-disabled-kill-volume-flags-component":
		n := br.ReadBits(13)
		for range n {
			br.ReadBit()
		}
	case name == "GameEngineComposerLetterboxComponent":
		if level < 2 {
			return true, false
		}
		br.ReadBit()
		br.ReadBits(16)
		for range 4 {
			if !br.ReadBit() {
				br.ReadBits(7)
			}
		}
		for range 4 {
			if br.ReadBit() {
				br.ReadBits(16)
			}
		}
	case name == "scenario-intro-component":
		br.ReadBits(7)
		br.ReadBits(1)
	case name == "matchflow-isplaying-flags-component":
		br.ReadBits(8)
	default:
		return false, false
	}
	return true, true
}

// rc2ksBassin : reap37LireBassin, bits consommes seulement.
func rc2ksBassin(br *Lecteur) {
	masque := br.ReadBits(64)
	for k := range 64 {
		if masque&(uint64(1)<<uint(k)) == 0 {
			continue
		}
		tag := br.ReadBits(2)
		if tag == 0 {
			continue
		}
		br.ReadBits(16)
		br.ReadBits(16)
		br.ReadBits(5)
		if tag == 1 {
			br.ReadBits(16)
		}
	}
}

// rc2ksTi40 : b2vLireTi40 (L4a), porte `+0x818` = rc2ksPorte.
func rc2ksTi40(br *Lecteur, name string) (bool, bool) {
	switch name {
	case "vehicle-auto-turret-triggers-component":
		br.ReadBits(3)
	case "vehicle-auto-turret-aiming-vector-component":
		br.ReadBits(19)
	case "vehicle-transformed-or-desired-open-state-changed-component":
		br.ReadBits(1)
		br.ReadBits(8)
	case "vehicle-type-state-component":
		if rc2ksPorte {
			if v := br.ReadBits(2); v == 1 || v == 3 {
				br.ReadBits(6)
			}
		}
	case compVehicleTypePhysics:
		if rc2ksPorte {
			consumeVehicleTypePhysics(br)
		}
	case "vehicle-auto-turret-target-component":
		consume1408f0ac4(br, 1)
	case "vehicle-sentry-state-component":
		br.ReadBits(3)
		br.ReadBits(1)
	case "vehicle-weapon-set-component":
		br.ReadBits(3)
		for k := 0; k < 2; k++ {
			if !br.ReadBit() {
				br.ReadBits(2)
			}
		}
	case "vehicle-auto-turret-component":
		br.ReadBits(2)
	case "vehicle-equipment-turret-parent-component":
		consume1408f0ac4(br, 0)
	case "vehicle-seats-override-pitch-component", "vehicle-seats-override-yaw-component":
		br.ReadBits(16)
	case "air-drop-flight-component":
		br.ReadBits(2)
		br.ReadBits(14)
		br.ReadBits(8)
	case "warp-component":
		a, b := br.ReadBit(), br.ReadBit()
		if a {
			br.ReadBits(8)
		}
		if b {
			br.ReadBits(8)
		}
	case "vehicle-low-frequency-component":
		if !br.ReadBit() {
			br.ReadBits(5)
		}
	default:
		return false, false
	}
	return true, true
}

// rc2ksTi43 : b2vLireTi43 (L2).
func rc2ksTi43(br *Lecteur, name string) (bool, bool) {
	switch name {
	case "device-position-animation-name-component":
		br.ReadBits(32)
		br.ReadBits(10)
	case "device-position-animation-control-component":
		for k := 0; k < 4; k++ {
			br.ReadBits(64)
		}
	case "device-position-group-component":
		br.ReadBits(32)
		if br.ReadBit() {
			br.ReadBits(8)
		}
	case "device-power-component", "device-power-group-component":
		br.ReadBits(14)
	case "device-interaction-in-progress-component", "device-exclusive-user-component":
		consume1408f0ac4(br, 1)
	case "device-interaction-hold-time-component", "device-interaction-start-time-override-component":
		br.ReadBits(8)
	case "device-control-action-string-override-component":
		br.ReadBits(32)
		br.ReadBits(32)
	case "device-health-station-charges-component":
		br.ReadBits(1)
		br.ReadBits(6)
	case "device-health-station-in-use-component", "device-in-primary-mode-component",
		"device-dispenser-require-los-component":
		br.ReadBits(1)
	case "device-dispenser-monitors-changed-component":
		n := br.ReadBits(8)
		if n > 8 {
			return true, false
		}
		for k := uint64(0); k < n; k++ {
			consume1408f0ac4(br, 1)
		}
	case "device-dispenser-state-flags-component":
		br.ReadBits(5)
	case "device-animation-layer-settings-component":
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
	case "device-animation-layer-state-component":
		if br.ReadBit() {
			for k := 0; k < 8; k++ {
				if br.ReadBit() {
					if br.ReadBits(10) != 0 {
						br.ReadBits(14)
					}
				}
			}
		}
	case "device-dispenser-state-component":
		for k := 0; k < 2; k++ {
			if br.ReadBit() {
				consume1408f0ac4(br, 0)
			}
			br.ReadBits(3)
		}
	case "device-object-dispenser-timer-component":
		for k := 0; k < 2; k++ {
			if br.ReadBit() {
				br.ReadBits(10)
				br.ReadBits(10)
				br.ReadBits(5)
				br.ReadBits(10)
			}
		}
	case "device-position-transition-velocity-component":
		br.ReadBits(18)
	case "device-machine-flags-component":
		br.ReadBits(9)
	default:
		return false, false
	}
	return true, true
}
