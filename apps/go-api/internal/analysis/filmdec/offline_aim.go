package filmdec

// Capture OFFLINE des directions (cap de visée / vélocité) DANS LE MÊME record biped que
// la position absolue i0 — c'est ce qui rend la validation possible : même slot, même
// instant, même paquet.
//
// Après i0, la boucle de composants poursuit dans l'ordre CROISSANT des index déclarés
// par le masque. Deux composants portent une direction empaquetée cubemap :
//
//	i1 object-translational-velocity (FUN_14076d45c -> FUN_14076d4d0) :
//	   R(1) outer ; outer==1 -> R(96) copie brute [keep] ;
//	   outer==0 -> R(1) absent-flag ; si 0 -> R(19) dir cubemap + R(10) magnitude log/exp.
//	i2 object-forward-and-up (FUN_14076e278 -> FUN_140c5fa84) :
//	   R(1) gate ; si gate==0 -> R(19) dir cubemap (le FACING du biped) ; puis R(8).
//
// i2 est notre CAP : la direction vers laquelle le corps du joueur est tourné, à la même
// date que la position. i1 sert d'ORACLE de contrôle (une direction de vélocité doit
// coïncider avec la différence finie des positions, à ~0°).
//
// Le balayage s'arrête au premier index de composant non modélisé : au-delà, la position
// du curseur n'est plus garantie, et décoder du bruit serait pire que ne rien décoder.

// aimDirBits est la largeur des directions empaquetées des composants mouvement (0x13).
const aimDirBits uint = 19

// velScaleBits est la largeur du mot de magnitude log/exp de la vélocité i1 (0xa).
const velScaleBits = 10

// i0TailBits : la QUEUE du composant i0, après le vec3 quantifié — 2 bits sur le chemin
// dominant (handleSel puis regionPresent, tous deux à 0 : cf. consumePositionHandleTail).
// MESURÉE, pas supposée : le balayage d'offset de cmd/tmp_aimsweep place la direction de
// vélocité i1 à i0+45+2+2 (les 2 derniers bits étant l'en-tête d'i1), et cette lecture
// donne un écart médian de 4,0° avec la direction de déplacement contre 90° au hasard.
// Cohérent avec le total i0 = 47 bits mesuré par capture Cheat Engine sur le chemin delta.
const i0TailBits = 2

// componentDirs porte les directions capturées dans un record biped.
type componentDirs struct {
	HasAim   bool   // i2 présent ET direction présente (gate==0)
	AimRaw   uint32 // R(19) brut du forward (cubemap)
	HasVel   bool   // i1 présent, chemin dynamic-precision avec direction
	VelRaw   uint32 // R(19) brut de la direction de vélocité
	VelScale uint32 // R(10) magnitude quantifiée
	HasYaw   bool   // i21 unit-desired-aiming-vector présent
	YawRaw   uint32 // R(12) : cap de VISÉE quantifié sur le tour complet
	PitchRaw uint32 // R(11) : élévation de visée quantifiée
}

// aimYawBits / aimPitchBits : largeurs des deux scalaires de FUN_14076e0ec (i21
// unit-desired-aiming-vector), 0xc puis 0xb.
const (
	aimYawBits   = 12
	aimPitchBits = 11
)

// AimHeadingDeg renvoie le CAP DE VISÉE en degrés dans le plan XY du repère monde
// ([0,360[), et sa validité. Convention MESURÉE sur le film (cf. cmd/tmp_aimcheck) :
// yaw = 360 * (q + 0.5) / 4096, même origine et même sens que atan2(Y, X) des positions
// déquantifiées — la moyenne circulaire de l'écart au cap de déplacement est nulle à
// moins de 2°, donc aucun décalage de convention n'est à appliquer.
func (p BipedPosition) AimHeadingDeg() (float32, bool) {
	if !p.HasYaw {
		return 0, false
	}
	return float32(360 * (float64(p.YawRaw) + 0.5) / (1 << aimYawBits)), true
}

// AimVector renvoie le cap unitaire décodé (cubemap 19 bits) et sa validité.
func (p BipedPosition) AimVector() ([3]float32, bool) {
	if !p.HasAim {
		return [3]float32{}, false
	}
	return DecodeAimVectorChecked(p.AimRaw, aimDirBits)
}

// VelocityVector renvoie la vélocité (direction cubemap × magnitude log/exp) et sa validité.
func (p BipedPosition) VelocityVector() ([3]float32, bool) {
	if !p.HasVel {
		return [3]float32{}, false
	}
	d, ok := DecodeAimVectorChecked(p.VelRaw, aimDirBits)
	if !ok {
		return [3]float32{}, false
	}
	m := DecodeVelocityMagnitude(uint64(p.VelScale), velScaleBits)
	return [3]float32{d[0] * m, d[1] * m, d[2] * m}, true
}

// recordMaskHook (DEBUG) reçoit, pour chaque record biped ÉMIS, la liste des index de
// composants de son masque, le payload du paquet et le bit qui suit i0. Sert au
// diagnostic « quel champ, à quel offset, porte la direction » ; nil en production (même
// convention que dynPrecHook / repTraceHook). Les appels sont dans le MÊME ordre que les
// positions renvoyées (filtres de post-traitement mis à part).
var recordMaskHook func(idx []int, payload []byte, afterI0 int)

// SetRecordMaskHook installe (ou efface, nil) le hook de diagnostic du masque.
func SetRecordMaskHook(h func(idx []int, payload []byte, afterI0 int)) { recordMaskHook = h }

// ReadBitsAtForDiag expose la lecture MSB-first de n bits (n <= 32) à une position bit
// absolue : réservée aux harnais de diagnostic qui balaient un payload de paquet.
func ReadBitsAtForDiag(b []byte, pos, n int) uint32 { return readBitsAt(b, pos, n) }

// scanRecordDirs lit les composants qui SUIVENT i0 dans un record biped et en extrait les
// directions empaquetées. `at` est le bit juste après i0 ; `idx` la liste des index de
// composants déclarés par le masque (idx[0] == 0 == i0). Aucune I/O, aucun état global :
// arrêt net au premier index non modélisé.
func scanRecordDirs(pay []byte, at, total int, idx []int) componentDirs {
	var out componentDirs
	at += i0TailBits // queue d'i0 (handleSel + regionPresent)
	for _, id := range idx[1:] {
		switch id {
		case 1:
			var ok bool
			at, ok = readVelocityComponent(pay, at, total, &out)
			if !ok {
				return out
			}
		case 2:
			var ok bool
			at, ok = readForwardComponent(pay, at, total, &out)
			if !ok {
				return out
			}
		case 21:
			readAimingVectorComponent(pay, at, total, &out)
			return out // i21 capturé : la suite du record ne nous intéresse pas
		default:
			return out // composant non modélisé -> curseur non fiable, on s'arrête
		}
	}
	return out
}

// readVelocityComponent consomme i1 (object-translational-velocity) et capture la
// direction quand elle est présente. Renvoie le bit suivant et false si le record est
// tronqué.
func readVelocityComponent(pay []byte, at, total int, out *componentDirs) (int, bool) {
	if at+1 > total {
		return at, false
	}
	if readBitsAt(pay, at, 1) == 1 { // outer==1 : copie brute R(96)
		at++
		if at+rawVec3Bits > total {
			return at, false
		}
		return at + rawVec3Bits, true
	}
	at++
	if at+1 > total {
		return at, false
	}
	if readBitsAt(pay, at, 1) == 1 { // absent : le moteur garde sa constante
		return at + 1, true
	}
	at++
	if at+int(aimDirBits)+velScaleBits > total {
		return at, false
	}
	out.HasVel = true
	out.VelRaw = readBitsAt(pay, at, int(aimDirBits))
	out.VelScale = readBitsAt(pay, at+int(aimDirBits), velScaleBits)
	return at + int(aimDirBits) + velScaleBits, true
}

// readForwardComponent consomme i2 (object-forward-and-up : R(1) gate ; si 0 -> R(19)
// direction cubemap ; puis R(8) inconditionnel) et capture la direction si présente.
func readForwardComponent(pay []byte, at, total int, out *componentDirs) (int, bool) {
	if at+1 > total {
		return at, false
	}
	gate := readBitsAt(pay, at, 1)
	at++
	if gate == 0 {
		if at+int(aimDirBits) > total {
			return at, false
		}
		out.HasAim = true
		out.AimRaw = readBitsAt(pay, at, int(aimDirBits))
		at += int(aimDirBits)
	}
	if at+8 > total {
		return at, false
	}
	return at + 8, true
}

// readAimingVectorComponent consomme i21 (unit-desired-aiming-vector, FUN_14076df7c) et
// capture le couple (cap, élévation) de visée :
//
//	R(1) flag0 ; FUN_14076e0ec = R(12) cap + R(11) élévation ; ...
//
// OFFSET MESURÉ, pas supposé : le balayage de cmd/tmp_aimsweep2 place le champ d'angle
// exactement 1 bit après le début du composant (donc juste après flag0), avec une
// concentration circulaire R = 0,84 contre 0,03 pour le bruit de fond.
func readAimingVectorComponent(pay []byte, at, total int, out *componentDirs) {
	if at+1+aimYawBits+aimPitchBits > total {
		return
	}
	at++ // flag0
	out.HasYaw = true
	out.YawRaw = readBitsAt(pay, at, aimYawBits)
	out.PitchRaw = readBitsAt(pay, at+aimYawBits, aimPitchBits)
}
