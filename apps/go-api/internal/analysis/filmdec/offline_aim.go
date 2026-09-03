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
	// QUEUE D'i21 — les trois drapeaux et le SECOND vecteur, jusqu'ici lus puis jetés.
	// Ils sont capturés sous la même option CaptureDirs, dans le même record : cf.
	// readAimingVectorComponent pour la grammaire et pour ce que le moteur en fait.
	AimFlag0 bool // +0x724 bit0 : les DEUX directions coïncident -> une seule est transmise
	AimFlag1 bool // +0x724 bit1 : drapeau accolé à la direction A. Valide si HasYaw.
	HasAimB  bool // le SECOND vecteur est présent (donc AimFlag0 == false) et lu en entier
	// YawRawB / PitchRawB : le second couple (cap, élévation), même largeurs que le premier.
	YawRawB, PitchRawB uint32
	AimFlag2           bool // +0x5f8 bit0 : drapeau accolé à la direction B. Valide si HasAimB.
	// MaskBits : le MASQUE DE COMPOSANTS du record, un bit par index déclaré (bit i = index i).
	// Il voyage DANS le record, et c'est tout son intérêt : le hook de diagnostic
	// (recordMaskHook) tire AVANT les filtres de post-traitement (DropIsolated,
	// DropTeleports), donc un appelant qui appaire ses appels aux positions renvoyées
	// s'appaire à côté dès qu'un record est écarté. Renseigné sous CaptureDirs.
	MaskBits uint64
	// MaskOver signale qu'un index >= 64 a été déclaré et n'a donc pas pu entrer dans MaskBits.
	// Jamais vu sur les archétypes bipèdes (i0..i58) ; sans ce drapeau, un masque tronqué se
	// lirait comme un masque complet.
	MaskOver bool
}

// maskBitsOf compacte la liste d'index de composants d'un record en un masque de 64 bits.
func maskBitsOf(idx []int) (uint64, bool) {
	var m uint64
	over := false
	for _, id := range idx {
		if id < 0 || id >= 64 {
			over = true
			continue
		}
		m |= 1 << uint(id)
	}
	return m, over
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
	return float32(360*(float64(p.PitchRaw)+0.5)/(1<<aimPitchBits) - 180), true
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
// Le lecteur `br` est FOURNI PAR L'APPELANT et REUTILISE : les deux composants de vitalite
// (i4 et i5) repositionnent le meme lecteur par `SetBitPos` au lieu d'en allouer un chacun.
// C'etaient deux allocations PAR RECORD BIPEDE, sur un balayage qui en reconnait des
// dizaines de milliers par film. `br` doit etre lie au payload balaye.
func scanRecordDirs(br *BitReader, at, total int, idx []int) (componentDirs, componentVitals) {
	pay := br.buf
	var out componentDirs
	var vit componentVitals
	at += i0TailBits // queue d'i0 (handleSel + regionPresent)
	for _, id := range idx[1:] {
		var ok bool
		switch id {
		case 1:
			at, ok = readVelocityComponent(pay, at, total, &out)
		case 2:
			at, ok = readForwardComponent(pay, at, total, &out)
		case 3:
			at, ok = readAngularVelocityComponent(pay, at, total)
		case 4:
			at, ok = readBodyVitalityComponent(br, at, total, &vit)
		case 5:
			at, ok = readShieldVitalityComponent(br, at, total, &vit)
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
func readBodyVitalityComponent(br *BitReader, at, total int, vit *componentVitals) (int, bool) {
	if at+bodyVitalityBits > total {
		return at, false
	}
	br.SetBitPos(at)
	vit.Body = decodeObjectBodyVitality(br)
	vit.HasBody = true
	return br.BitPos(), true
}

// readShieldVitalityComponent consomme i5 et capture le bouclier. Largeur VARIABLE (29 à
// 55 bits selon deux portes internes) : on exige d'abord le minimum lisible, puis on
// vérifie après coup que le décodage n'a pas débordé — un décodage qui déborde rendrait des
// zéros de bourrage, c'est-à-dire un bouclier faux.
func readShieldVitalityComponent(br *BitReader, at, total int, vit *componentVitals) (int, bool) {
	if at+shieldVitalityMinBits > total {
		return at, false
	}
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

// readAimingVectorComponent consomme i21 (unit-desired-aiming-vector, FUN_14076df7c) et
// capture le couple (cap, élévation) de visée, PUIS la queue du composant :
//
//	R(1) flag0 ; FUN_14076e0ec = R(12) cap + R(11) élévation ; R(1) flag1 ;
//	si flag0 == 0 : FUN_14076e0ec (second couple) + R(1) flag2.
//
// Soit 25 bits sur le chemin dominant et 49 bits sur l'autre — les deux largeurs relevées
// dans la table ECS.
//
// OFFSET MESURÉ, pas supposé : le balayage de cmd/tmp_aimsweep2 place le champ d'angle
// exactement 1 bit après le début du composant (donc juste après flag0), avec une
// concentration circulaire R = 0,84 contre 0,03 pour le bruit de fond.
//
// CE QUE LE MOTEUR MET DANS CES QUATRE CHAMPS — lu chez le PRODUCTEUR, FUN_142ee09a8, qui
// est la passe de collecte qui remplit l'état répliqué de l'unité (bloc de masque 0x200000
// = i21), et confirmé chez le CONSOMMATEUR, FUN_1404d4cb8, qui le recopie sur l'objet local :
//
//	flag0   = ! FUN_14071443c(unité). Vrai quand l'unité n'a pas de contrôleur, ou quand le
//	          bit 22 de unité+0x380 est posé. Le moteur s'en sert comme drapeau de
//	          COMPRESSION : à vrai, la seconde direction est identique à la première et
//	          FUN_1406c72e8 recopie le vecteur A dans les DEUX emplacements de destination
//	          (+0x348 et +0x3e4).
//	vect. A = unité+0x348 ; vect. B = unité+0x3e4. Deux directions de visée distinctes, de
//	          même forme, dont la seconde n'est transmise que si flag0 == 0.
//	flag1   = signe de (unité+0x354) x A, produit vectoriel 2D ; flag2 = signe de
//	          (unité+0x3d8) x B. Ce sont donc les SIGNES d'un axe compagnon, ce que
//	          l'encodage 23 bits d'une direction ne peut pas porter — de la géométrie, pas
//	          un état de jeu. Ni l'un ni l'autre n'est un bit de lunette : voir la mesure
//	          `visee_lunette_research_test.go` (paquet replay), qui le CHIFFRE au lieu de
//	          l'affirmer.
//
// Les bornes sont vérifiées champ par champ : une queue tronquée laisse le couple de tête
// publié (comportement inchangé) et n'arme simplement pas les nouveaux drapeaux.
func readAimingVectorComponent(pay []byte, at, total int, out *componentDirs) {
	if at+1+aimYawBits+aimPitchBits > total {
		return
	}
	out.AimFlag0 = readBitsAt(pay, at, 1) == 1
	at++
	out.HasYaw = true
	out.YawRaw = readBitsAt(pay, at, aimYawBits)
	out.PitchRaw = readBitsAt(pay, at+aimYawBits, aimPitchBits)
	at += aimYawBits + aimPitchBits
	if at+1 > total {
		return
	}
	out.AimFlag1 = readBitsAt(pay, at, 1) == 1
	at++
	if out.AimFlag0 { // les deux directions coïncident : rien d'autre n'est transmis
		return
	}
	if at+aimYawBits+aimPitchBits+1 > total {
		return
	}
	out.HasAimB = true
	out.YawRawB = readBitsAt(pay, at, aimYawBits)
	out.PitchRawB = readBitsAt(pay, at+aimYawBits, aimPitchBits)
	out.AimFlag2 = readBitsAt(pay, at+aimYawBits+aimPitchBits, 1) == 1
}
