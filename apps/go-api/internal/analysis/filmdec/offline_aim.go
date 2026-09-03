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

// componentVitals porte la VITALITÉ capturée dans le même record biped que la position.
//
// Même principe que componentDirs : le balayage repart du bit qui suit i0 et consomme les
// composants du masque dans l'ordre. i4 et i5 sont deux crans après les directions ; ils
// n'exigent donc qu'un composant de plus (i3, angular-velocity) pour être atteints.
type componentVitals struct {
	HasBody   bool // i4 object-body-vitality présent ET atteint sans désync
	Body      BodyVitality
	HasShield bool // i5 object-shield-vitality présent ET atteint sans désync
	Shield    ShieldVitality
}

// dirsGrammar dit QUELLE grammaire d'orientation le balayage offline applique à i2 et i3.
// Le zéro (tout faux) est la grammaire du BIPÈDE — c'est le comportement historique et
// celui de tous les appelants existants. Les deux drapeaux sont SÉPARÉS pour qu'un
// instrument puisse isoler la contribution de chacun (i2 seul, i3 seul, les deux) ; la
// production, elle, ne pose qu'un seul choix (ScanFilmOptions.DynPrecOrientation), parce
// que l'archétype décide des DEUX à la fois.
type dirsGrammar struct {
	fwdUpDynPrec  bool   // i2 : FUN_140c5f7ec au lieu de FUN_14076e278
	angVelDynPrec bool   // i3 : FUN_140d87740 au lieu de FUN_140d70998
	fwdUpParam    uint32 // arg5 de FUN_140c5f7ec : >= 2 ajoute le bit de porte C
}

// dynPrecOrientationGrammar est la grammaire des archétypes ti=38/39/40/43.
func dynPrecOrientationGrammar() dirsGrammar {
	return dirsGrammar{
		fwdUpDynPrec:  true,
		angVelDynPrec: true,
		fwdUpParam:    paramForComponent(compForwardUpDynPrec),
	}
}

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
	return aimHeadingDegFromRaw(p.YawRaw), true
}

// aimHeadingDegFromRaw est le SEUL détenteur de la conversion « quantum de cap -> degrés ».
// `BipedPosition.AimHeadingDeg` (record porteur de position) et `BipedAim.AimHeadingDeg`
// (record SANS position, cf. offline_aim_only.go) l'appellent tous deux : la convention est
// mesurée une fois et écrite une fois.
func aimHeadingDegFromRaw(yaw uint32) float32 {
	return float32(360 * (float64(yaw) + 0.5) / (1 << aimYawBits))
}

// AimPitchDeg renvoie l'ÉLÉVATION DE VISÉE en degrés (positif = vers le haut, 0 = à plat), et
// sa validité. Elle vient du second scalaire d'i21, le R(11) qui suit le cap.
//
// CONVENTION MESURÉE, pas supposée (item E.0.1 du plan d'exploitation du registre, mesure du
// 2026-08-18 sur `000d5950` Cliffhanger, `530820e5` Catalyst et `7344d24f` Vagabond) :
//
//	ENCODAGE      binaire décalé, centré sur 1024 : le mode de la distribution tombe à 1024
//	              (Catalyst), 1013 (Vagabond) et 1006 (Cliffhanger) — un joueur vise à plat, ou
//	              quelques degrés sous l'horizontale (il vise des corps, pas des yeux).
//	QUANTUM       360/2048 = 0,17578 deg par pas, soit DEUX FOIS celui du cap. Établi par
//	              l'oracle du kill : au moment du tir le réticule est sur la victime, donc
//	              l'angle visé est atan2(dz, distance horizontale) entre les deux bipèdes.
//	              L'ajustement dz = dxy·tan(c·pas) − h rend c = 0,1706 / 0,1385 / 0,1685 deg/pas
//	              sur les trois films (R² 0,90 / 0,72 / 0,92), et le candidat 360/2048 est à
//	              1,01 / 1,26 / 1,03 fois la meilleure somme des carrés, quand le quantum du cap
//	              (180/2048) est à 3,34 / 1,38 / 4,06 — REFUTÉ.
//	SIGNE         au-dessus de 1024 = vers le HAUT : 56 accords de signe sur 58 kills à
//	              |dz| >= 1 m (100 % / 91,7 % / 100 %), témoin par permutation 86,7 / 58,3 /
//	              47,4 %, plancher du prédicteur constant 93,3 / 66,7 / 63,2 %.
//
// RÉSERVE HONNÊTE. Les valeurs observées tiennent dans [537, 1490] sur les trois films, donc
// dans la MOITIÉ centrale du champ ([512, 1536] = ±90°). Cette mesure ne peut donc pas
// distinguer « le champ couvre ±180° et le jeu borne le tangage à ±90° » de « le champ ne code
// que ±90° sur la moitié de ses valeurs » : les deux donnent EXACTEMENT les mêmes degrés sur
// tout ce que le film transmet. Le jour où une valeur sortirait de cette moitié, c'est cette
// formule-ci (±180° sur tout le champ) qui la rendrait, et il faudra la revérifier.
func (p BipedPosition) AimPitchDeg() (float32, bool) {
	if !p.HasYaw {
		return 0, false
	}
	return aimPitchDegFromRaw(p.PitchRaw), true
}

// aimPitchDegFromRaw est le SEUL détenteur de la conversion « quantum d'élévation -> degrés »,
// réserve comprise (cf. le commentaire d'AimPitchDeg ci-dessus). Deux appelants :
// `BipedPosition.AimPitchDeg` et `BipedAim.AimPitchDeg`.
func aimPitchDegFromRaw(pitch uint32) float32 {
	return float32(360*(float64(pitch)+0.5)/(1<<aimPitchBits) - 180)
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
// directions empaquetées ET la vitalité (i4 santé, i5 bouclier). `at` est le bit juste
// après i0 ; `idx` la liste des index de composants déclarés par le masque (idx[0] == 0 ==
// i0). Aucune I/O, aucun état global : arrêt net au premier index non modélisé.
//
// i3 (angular-velocity) a été ajouté le 2026-07-26 non pour sa valeur — elle n'est pas
// capturée — mais parce qu'il SÉPARE les directions de la vitalité dans l'ordre du masque.
// Sans lui, tout record déclarant i3 s'arrêtait avant i4/i5 ET avant i21. Conséquence
// mesurée et publiée : la couverture du cap de visée CHANGE (elle augmente) ; c'est un
// effet de bord assumé d'un décodeur qui va plus loin, pas une modification de la position.
//
// `g` choisit la grammaire d'i2/i3 : zéro = celle du BIPÈDE
// (`object-forward-and-up-component` -> FUN_14076e278,
// `object-angular-velocity-component` -> FUN_140d70998) ; les variantes
// `-dynamic-precision-` que portent ti=38/39/40/43 sont QUATRE désérialiseurs distincts
// (FUN_140c5f7ec / FUN_140d87740), résolus statiquement le 2026-09-03 : voir
// components_dynprec_orientation.go. Sans elles, un balayage de la bande ti=40 lisait i2
// amputé de ses bits de tête et i3 amputé de son gate externe, donc atteignait i4
// (vitalité) avec un curseur faux — la cause racine du bruit mesuré au lot V2b.
func scanRecordDirs(pay []byte, at, total int, idx []int, g dirsGrammar) (componentDirs, componentVitals) {
	var out componentDirs
	var vit componentVitals
	at += i0TailBits // queue d'i0 (handleSel + regionPresent)
	for _, id := range idx[1:] {
		var ok bool
		switch id {
		case 1:
			at, ok = readVelocityComponent(pay, at, total, &out)
		case 2:
			if g.fwdUpDynPrec {
				at, ok = readForwardComponentDynPrec(pay, at, total, &out, g.fwdUpParam)
			} else {
				at, ok = readForwardComponent(pay, at, total, &out)
			}
		case 3:
			if g.angVelDynPrec {
				at, ok = readAngularVelocityComponentDynPrec(pay, at, total)
			} else {
				at, ok = readAngularVelocityComponent(pay, at, total)
			}
		case 4:
			at, ok = readBodyVitalityComponent(pay, at, total, &vit)
		case 5:
			at, ok = readShieldVitalityComponent(pay, at, total, &vit)
		case 21:
			readAimingVectorComponent(pay, at, total, &out)
			return out, vit // i21 capturé : la suite du record ne nous intéresse pas
		default:
			return out, vit // composant non modélisé -> curseur non fiable, on s'arrête
		}
		if !ok {
			return out, vit
		}
	}
	return out, vit
}

// readAngularVelocityComponent consomme i3 (object-angular-velocity, FUN_140d70998 =
// FUN_14076d528) : R(1) gate ; si gate == 0 -> R(19) direction + R(8) magnitude. Même
// famille que la vélocité i1, largeurs figées par angularMagBits / angularScaleBits.
// Aucune valeur capturée : ce composant n'est traversé que pour atteindre i4/i5.
func readAngularVelocityComponent(pay []byte, at, total int) (int, bool) {
	if at+1 > total {
		return at, false
	}
	gate := readBitsAt(pay, at, 1)
	at++
	if gate != 0 { // absent : le moteur garde sa constante, zéro bit de charge utile
		return at, true
	}
	n := int(angularMagBits + angularScaleBits)
	if at+n > total {
		return at, false
	}
	return at + n, true
}

// readBodyVitalityComponent consomme i4 et capture la santé. La GRAMMAIRE n'est pas
// réécrite ici : on positionne un BitReader sur le payload et on appelle le décodeur
// canonique (vitality.go), seul détenteur de la forme du composant.
func readBodyVitalityComponent(pay []byte, at, total int, vit *componentVitals) (int, bool) {
	if at+bodyVitalityBits > total {
		return at, false
	}
	br := NewBitReader(pay)
	br.SetBitPos(at)
	vit.Body = decodeObjectBodyVitality(br)
	vit.HasBody = true
	return br.BitPos(), true
}

// readShieldVitalityComponent consomme i5 et capture le bouclier. Largeur VARIABLE (29 à
// 55 bits selon deux portes internes) : on exige d'abord le minimum lisible, puis on
// vérifie après coup que le décodage n'a pas débordé — un décodage qui déborde rendrait des
// zéros de bourrage, c'est-à-dire un bouclier faux.
func readShieldVitalityComponent(pay []byte, at, total int, vit *componentVitals) (int, bool) {
	if at+shieldVitalityMinBits > total {
		return at, false
	}
	br := NewBitReader(pay)
	br.SetBitPos(at)
	s := decodeObjectShieldVitality(br)
	if br.BitPos() > total {
		return at, false
	}
	vit.Shield = s
	vit.HasShield = true
	return br.BitPos(), true
}

// Largeurs de contrôle des bornes du balayage offline (cf. TestDecodeShieldVitalityBitCost).
const (
	bodyVitalityBits      = 8 + 3      // i4 : quantum + 3 drapeaux
	shieldVitalityMinBits = 8 + 1 + 20 // i5 : chemin le plus court (portes fermées)
)

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

// readForwardComponentDynPrec consomme i2 `object-forward-and-up-DYNAMIC-PRECISION-component`
// (FUN_140c5f7ec, ti=38/39/40/43). La grammaire n'est PAS réécrite ici : on pose un
// BitReader et on appelle son unique détenteur (components_dynprec_orientation.go).
func readForwardComponentDynPrec(pay []byte, at, total int, out *componentDirs, param uint32) (int, bool) {
	br := NewBitReader(pay)
	br.SetBitPos(at)
	v, ok := decodeObjectForwardAndUpDynPrec(br, param)
	if !ok || br.BitPos() > total {
		return at, false
	}
	if v.HasDir {
		out.HasAim = true
		out.AimRaw = v.DirRaw
	}
	return br.BitPos(), true
}

// readAngularVelocityComponentDynPrec consomme i3
// `object-angular-velocity-DYNAMIC-PRECISION-component` (FUN_140d87740, ti=40) : un gate
// EXTERNE R(1) que le composant sans « dynamic-precision » n'a pas, puis soit R(96) de
// copie brute, soit le vec3 dyn.-préc. habituel. Aucune valeur capturée : ce composant
// n'est traversé que pour atteindre i4/i5.
func readAngularVelocityComponentDynPrec(pay []byte, at, total int) (int, bool) {
	br := NewBitReader(pay)
	br.SetBitPos(at)
	consumeObjectAngularVelocityDynPrec(br)
	if br.BitPos() > total {
		return at, false
	}
	return br.BitPos(), true
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
