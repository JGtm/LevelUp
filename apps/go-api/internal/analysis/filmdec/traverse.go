package filmdec

// Keyframe/delta entity traversal: drives the mask-gated component loop
// (FUN_14076cb60) over an archetype's ordered component list (from the registry),
// dispatching each present component to its ported bit-consumer. The goal is to
// reach the held-weapon component (weapon-state-type-info) and read its variant-name.
//
// STATUS: iterative validation harness. consumeByName wires only the desers that
// are bit-exact-confirmed; unconfirmed ones return ported=false so the traversal
// stops cleanly (DesyncAt) instead of silently mis-aligning — that tells us exactly
// which deser to port next.

// noVariant is the engine sentinel for an unread/absent variant-name.
const noVariant uint32 = 0xFFFFFFFF

// CompResult is one decoded component slot in a record.
type CompResult struct {
	Index    int    // archetype iterator index (= mask bit)
	Name     string // component name from the registry
	Variant  uint32 // variant-name for obje/weapon components (else noVariant)
	Ported   bool   // false => no bit-exact deser; traversal must stop here
	StartBit int
	// Payload porte la VALEUR décodée du composant, pour les seuls composants de
	// captureNames (cf. capture.go) : BodyVitality, ShieldVitality, RespawnTimer,
	// RoundTimer. nil partout ailleurs — le décodeur reste un sauteur de bits par défaut.
	Payload any
}

// EntityTrace is the result of traversing one new-entity record.
type EntityTrace struct {
	TypeIndex   uint32
	DefaultBits int
	Gate        bool
	Mask        uint64
	Comps       []CompResult
	Dead        *DeadState // captured object-dead-state heavy form (nil if no dead-state component present)
	DesyncAt    int        // iterator index of the first un-ported present component (-1 if all consumed)
	EndBit      int
}

// recordStateParam is the engine's per-component param_4 (the actor-tick /
// weapon-set count returned by the descriptor's vtable[0] in FUN_14076cb60). It is
// NOT read from the bitstream: it is an external descriptor property computed per
// component before its deser runs. Three biped components branch on it:
//
//	unit-actor-control  (i19): gates the optional 2nd slot id (param_4 > 1).
//	unit-actor-state    (i20): selects the width-table read (1->8,2->10,3->11,>=4->12,
//	                           default 12 when param_4 == 0).
//	unit-malleable-property (i23): gates a few R(1) flags (param_4 > 2, > 3).
//
// The default (0) is the conservative shape: actor-state reads width 12, and the
// optional slot / flag reads are absent. When a real descriptor count is known it
// must be supplied (it cannot be recovered from the bits alone).
//
// Exposed as a package var (not const) so a calibration harness can sweep {0,1,2,3}
// to find the value that re-synchronises the component AFTER a recordStateParam-
// dependent one (i10 object-parent-state, i19/i20/i23 unit-*). Verdict from the
// deser-fix workflow: i10's keyframe desync is THIS parameter, not a code bug.
var recordStateParam uint32 = 0

// SetRecordStateParam lets a harness sweep the runtime actor-tick/weapon-set count
// (param_4) that varies the bit width of i10/i19/i20/i23. See recordStateParam.
func SetRecordStateParam(v uint32) { recordStateParam, recordStateParamOverride = v, true }

// paramByComponent porte le VRAI param_4, par composant, tel que la capture live le
// mesure (colonne `param4` de .ai/V7.5/dumps/ce_capture_delta.csv). Extraction sur
// l'archétype bipède (ti=35) : la valeur est CONSTANTE pour un composant donné —
// aucun composant n'y présente deux valeurs sur 464 010 mesures.
//
//	i13 object-maximum-vitalities  -> 3       i23 unit-malleable-property     -> 4
//	i15 object-low-frequency       -> 2       i43..i46 weapon-state-type-info -> 2
//	i17 object-frame-configuration -> 0       i53 biped-malleable-property    -> 2
//	i18 unit-control               -> 2       i57/i59 biped-spartan-ability*  -> 2
//	tout le reste                  -> 1
//
// AVANT ce correctif, une SEULE variable globale valant 0 servait tous les composants —
// c'est-à-dire la valeur juste pour le seul i17, et fausse pour tous les autres.
// La table ne change QUE les composants dont le déser branche sur param_4 (i10, i19,
// i20, i23 et flock-destination) ; elle est neutre partout ailleurs.
var paramByComponent = map[string]uint32{
	"object-maximum-vitalities-component":  3,
	"object-low-frequency-component":       2,
	"object-frame-configuration-component": 0,
	"unit-control-component":               2,
	"unit-malleable-property-component":    4,
	compWeaponStateTypeInfo:                2,
	"biped-malleable-property":             2,
	"biped-malleable-property-component":   2,
	"biped-spartan-ability":                2,
	"biped-spartan-ability-component":      2,
	// i59 manquait à la table (2026-08-16, plan PLAN_GRAPPIN_LIGNE) : le commentaire
	// ci-dessus disait « i57/i59 -> 2 » mais seules les clés d'i57 existaient, donc la
	// queue R(3) d'i59 (FUN_140fc147c, param_4>1) n'était JAMAIS lue offline. Mesure :
	// chaque record i59 finissait à 3 bits exactement du record suivant (écarts
	// p10=p50=p90=3, n=988, TestI59AnchorWalkProof) ; avec la clé, l'écart tombe à 0.
	"biped-spartan-ability-non-predicted-state":           2,
	"biped-spartan-ability-non-predicted-state-component": 2,
}

// paramForComponent rend le param_4 du composant `name`. Défaut 1 : c'est la valeur
// mesurée pour l'écrasante majorité des composants (0 était un choix « conservateur »
// jamais mesuré, et faux).
func paramForComponent(name string) uint32 {
	if v, ok := paramByComponent[name]; ok {
		return v
	}
	if recordStateParamOverride {
		return recordStateParam // un harnais de balayage a forcé la valeur
	}
	return 1
}

// recordStateParamOverride passe à true dès qu'un harnais appelle SetRecordStateParam :
// le balayage manuel garde alors la main sur les composants hors table.
var recordStateParamOverride = false

// TraversalPrecision est le descripteur de quantification de position employé quand le
// déser d'i0 (object-position-dynamic-precision-component) se déclenche. Les tables de
// l'exécutable lisent 0 statiquement : les vraies largeurs viennent de la config de
// réplication du film.
//
// LE 6/6/6 HISTORIQUE ÉTAIT FAUX, ET C'ÉTAIT LA FAUTE CENTRALE DU DÉCODEUR (2026-07-26).
// Il donne un i0 de 5 + 6+6+6 = 23 bits. La vérité terrain — 134 767 relevés du curseur
// réel sur le désérialiseur du jeu — donne **47 bits, à 100 %**. Déficit de 24 bits dès le
// PREMIER composant de 100 % des records : tout ce qui suit était lu à côté, ce qui explique
// les comptes de grenades à 255 sur un champ borné à 2.
//
//	47 = 5 (3 spine + 1 useDefault + 1 index de région) + 13 + 13 + 14 + 2 de queue
//
// CE CHAMP N'EST PAS LE LEVIER — piège vérifié le 2026-07-26, à ne pas refaire.
// `AxisW` est la largeur des DELTAS de position (petits écarts image à image : 6 bits
// suffisent). Les 13/13/14 sont les largeurs des positions ABSOLUES, et le déser en tient
// une seconde, distincte, via `absAxisW`/`SetAbsoluteAxisW`. Y écrire 13/13/14 allonge le
// chemin delta — qui est le chemin DOMINANT — et dégrade au lieu de corriger : mesuré,
// i22 passe de 90,02 % à 92,83 % de comptes impossibles. Le 6/6/6 est donc conservé ici.
//
// Le vrai correctif d'i0 doit distinguer les deux largeurs le long de chaque branche de
// FUN_1406cfe44 (keep-baseline 101 bits · absolu · delta prédit), et non régler une globale.
// `PositionCalibratedSkip` court-circuite déjà le problème en sautant 47 ou 101 bits selon
// le premier bit — mais c'est un banc de calibration propre à Cliffhanger, pas un décodeur.
var TraversalPrecision = PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{6, 6, 6}}

// WorldObjectPrecision est le descripteur du chemin WORLD-OBJECT d'i0
// (`object-position-component`) : projectiles ti=41, armes au sol ti=42, équipement ti=37,
// corps rigides ti=38. À la différence du bipède, ces archétypes n'envoient PAS de delta de
// position : ils envoient la position ABSOLUE quantifiée à chaque image. Les largeurs sont
// donc celles de la CARTE — les mêmes que l'absolu du bipède — et non un 6/6/6 de delta.
//
// 45 bits au total = 1 precHigh + 1 index-sel + 1 index de région + 13 + 13 + 14 + 2 de queue.
// Mesuré sur 000d5950 : le triplet de porte vaut (0,0,0) dans 6 922 des 6 924 records des
// 70 trajectoires de grenade.
//
// PROPRE À LA CARTE, et CE DÉFAUT EST L'ENTRÉE `cliffhanger` DU CATALOGUE : `{13,13,14}` est
// exactement `map_quant_bounds.json` -> `cliffhanger`.`axisWidths` (module `ridgeline`), ce que
// `map_bounds_test.go` vérifie. Le défaut n'est donc pas un repli neutre : c'est UNE carte.
//
// QUI L'INSTALLE, ET DEPUIS QUAND (2026-08-15). `replay.BuildFromFilm` installe les largeurs de
// la carte du match pour toute la durée du décodage, sous `LockProcessDecode`, et les restaure
// au retour (`replay.installWorldObjectPrecision`). Elles viennent de `MapQuantEntry.AxisWidths`
// — la MÊME entrée de catalogue qui fournit les bornes, jamais un second réglage à armer à part.
//
// AVANT cette date, AUCUN chemin de production ne l'écrasait : toutes les cartes autres que
// Cliffhanger déquantifiaient leurs objets du monde aux largeurs de Cliffhanger. Mesuré sur
// 7 films / 7 cartes, dont 6 autres que Cliffhanger (part d'échantillons de projectile dans l'emprise du nuage des bipèdes du
// même film, coordonnées normalisées de l'AABB) : 0,09 · 0,51 · 28,46 · 31,31 · 65,21 % avec le
// défaut, 98,96 à 99,79 % avec les largeurs du catalogue — et 92,11 % des DEUX côtés sur
// Cliffhanger, où le correctif ne change rien par construction.
//
// `DetectI0Layout` n'est PAS la source de ces largeurs : c'est le CONTRÔLE que réclame le
// commentaire d'`AxisWidths`. Accord catalogue <-> découpage lu dans le film : 7 films sur 7.
var WorldObjectPrecision = PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{13, 13, 14}}

// SetWorldObjectPrecisionFromLayout installe les largeurs d'axe de la CARTE pour le chemin
// world-object. Les axes sont partagés avec l'absolu du bipède : c'est le même AABB de BSP qui
// les fixe — hypothèse enfin vérifiée par ses conséquences le 2026-08-15 (cf. ci-dessus).
//
// SOURCE ATTENDUE : `MapQuantEntry.AxisWidths`, déduit des bornes par la loi du moteur. Le
// découpage lu dans le film (`DetectI0Layout`) sert de contrôle : s'il contredit le catalogue,
// ce sont les BORNES qui sont fausses.
//
// L'APPELANT DOIT DÉTENIR `LockProcessDecode` et restaurer la valeur précédente : c'est un
// global de paquet. Le seul appelant de production est `replay.installWorldObjectPrecision`.
func SetWorldObjectPrecisionFromLayout(l I0Layout) {
	if l.AxisW[0] == 0 || l.AxisW[1] == 0 || l.AxisW[2] == 0 {
		return // layout non détecté : garder le défaut plutôt qu'installer des zéros
	}
	WorldObjectPrecision.AxisW = l.AxisW
	// La largeur de l'INDEX DE RÉGION est elle aussi une constante par carte
	// (ceilLog2(nb de régions) — 2 bits sur Live Fire, lot C catalogues 2026-08-27). Un
	// layout sans gate (appels historiques qui ne posent que AxisW) laisse le défaut.
	// LIMITE ASSUMÉE : le lecteur world-object déquantifie tous les records aux largeurs
	// de LA région cataloguée ; un record d'une autre région (rarissime — l'ordre des
	// 3/291 288 de Cliffhanger) consommerait des largeurs différentes et désalignerait
	// SON record. La table par région (SetAbsPerIndexAxisW) existe pour le chemin
	// sim-state ; l'y étendre ici attendra une carte où le cas pèse.
	if l.GateBits > i0SpineBits+i0UseDefaultBits {
		WorldObjectPrecision.IndexW = uint(l.GateBits - i0SpineBits - i0UseDefaultBits)
	}
}

// consumeByName dispatches a component to its ported bit-consumer. It returns the
// variant-name for variant-bearing components (obje, held-weapon), the captured
// object-dead-state (non-nil only for the dead-state component), and ported=false
// for components whose deser is not yet bit-exact.
func consumeByName(br *BitReader, name string, typeIndex uint32, level uint32) (variant uint32, dead *DeadState, ported bool) {
	variant = noVariant
	switch name {
	case "object-position-dynamic-precision-component": // i0
		consumeObjectPositionDynamicPrecisionD(br, TraversalPrecision)
		return variant, nil, true
	case "object-translational-velocity-dynamic-precision-component": // i1
		consumeObjectTranslationalVelocity(br)
		return variant, nil, true
	case "object-angular-velocity-dynamic-precision-component", "object-angular-velocity-component": // i3
		// VRAI deser biped i3 (table ECS live) = FUN_140d70998 = FUN_14076d528 = un SEUL gate
		// dynamic-precision : R(1) present ; si 0 -> R(19) dir + R(8) magnitude. L'ancien routage
		// (consumeObjectAngularVelocity=FUN_140d87740) ajoutait un gate EXTERNE parasite + R(96)
		// keep -> sur-lecture de 96 bits qui engloutissait les records des joueurs suivants
		// (oracle: 1 seul biped-delta/frame au lieu de ~8). Validé par tmp_framedump (biped-deltas
		// par frame). Toggle useLegacyAngularVel pour l'ancien comportement.
		if useLegacyAngularVel {
			consumeObjectAngularVelocity(br)
		} else {
			consumeDynPrecVec3(br, angularMagBits, angularScaleBits)
		}
		return variant, nil, true
	case "object-region-state-component": // i6
		consumeObjectRegionState(br)
		return variant, nil, true
	case "object-damage-sections-component": // i7
		consumeObjectDamageSections(br)
		return variant, nil, true
	case "object-constraint-component": // i8
		consumeObjectConstraint(br)
		return variant, nil, true
	case "object-parent-state-component": // i10 (1st desync on typeIndex=40)
		consumeObjectParentState(br, paramForComponent(name), typeIndex)
		return variant, nil, true
	case "object-scale-component": // i12
		consumeObjectScale(br)
		return variant, nil, true
	case "object-maximum-vitalities-component": // i13
		consumeObjectMaximumVitalities(br)
		return variant, nil, true
	case "object-dissolver-component": // i14
		consumeObjectDissolver(br)
		return variant, nil, true
	case "object-low-frequency-component": // i15 = FUN_1407ef088 (validé, matche la table live)
		consumeObjectLowFrequency(br)
		return variant, nil, true
	case "object-physics-flags-component": // i16
		consumeObjectPhysicsFlags(br)
		return variant, nil, true
	case "object-frame-configuration-component": // i17
		consumeObjectFrameConfiguration(br)
		return variant, nil, true
	case "unit-actor-control-component":
		consumeUnitActorControl(br, paramForComponent(name))
		return variant, nil, true
	case "unit-actor-state-component":
		consumeUnitActorState(br, paramForComponent(name))
		return variant, nil, true
	case "unit-malleable-property-component":
		consumeUnitMalleableProperty(br, paramForComponent(name))
		return variant, nil, true
	case "biped-spartan-ability-malleable-property-component": // i58 (FUN_140fea4c0)
		consumeBipedSpartanAbilityMalleableProperty(br)
		return variant, nil, true
	case "object-position-component": // world-object i0 (FUN_14076e29c)
		// Structure RE bit-exacte (FUN_14076e420 precHigh R(1) -> FUN_14076e524 index-sel+index+3axes
		// / FUN_141f85880 AABB ; FUN_14076e3e4 handle-tail gated precHigh ; FUN_14076e304 R(2) finite).
		//
		// DÉCOUPAGE CORRIGÉ le 2026-07-26 — et c'est le bug le plus retors rencontré ici.
		// L'ancien portage lisait `R(2)` d'index puis 13/13/13. Le vrai découpage est `R(1)`
		// d'index puis 13/13/**14**. LE TOTAL EST LE MÊME (45 bits) : aucun test de longueur,
		// aucune mesure de désynchronisation ne pouvait le voir. Seule une mesure de CONTENU le
		// révèle — ici le profil de bascule bit à bit sur 6 794 paires de records consécutifs
		// d'une même entité, qui place les frontières à 16 / 29 / 43.
		//
		// L'ancien commentaire affirmait « largeurs world-object UNIVERSELLES, PAS map-specific ».
		// C'est RÉFUTÉ : ce sont les largeurs DE LA CARTE, les mêmes que celles du bipède
		// (13/13/14 sur Cliffhanger), et les bornes de déquantification sont le même AABB de BSP.
		// Seule la PORTE diffère du bipède : 3 bits ici (precHigh + index-sel + index de région)
		// contre 5 là-bas. C'est ce qui explique la contradiction « i0 45 vs 47 bits » qui
		// traînait dans les notes : ce ne sont pas deux mesures du même champ, ce sont deux
		// archétypes différents.
		if br.ReadBit() { // precHigh (FUN_14076e420 R(1))
			br.ReadBits(59) // precHigh=1 : FUN_141f85880 AABB + handle-tail + R(2) (total 60 mesuré)
		} else {
			if !br.ReadBit() { // FUN_14076e524 index-sel ; si 0 -> lit l'index de région
				br.ReadBits(WorldObjectPrecision.IndexW)
			}
			for a := 0; a < 3; a++ {
				br.ReadBits(WorldObjectPrecision.AxisW[a]) // FUN_140cc5128 axe a
			}
			br.ReadBits(2) // FUN_14076e304 R(2) finite (handle-tail = 0 bit quand precHigh=0)
		}
		return variant, nil, true
	case "object-translational-velocity-component": // world-object i1 (FUN_14076e228)
		consume14076d528(br) // R(1)[+R(19)+R(10)]
		return variant, nil, true
	case "object-forward-and-up-dynamic-precision-component": // ti=38 i2 (reuse biped i2 deser)
		consumeObjectForwardAndUp(br)
		return variant, nil, true
	case "change-scene-component": // ti=16 i0 (FUN_142ed3fcc)
		consumeChangeScene(br)
		return variant, nil, true
	case "spawn-filter-type-component": // ti=20 i0 (FUN_142ed708c)
		consumeSpawnFilterType(br)
		return variant, nil, true
	case "statborg-current-round-value-stat-component": // ti=6 i0 (FUN_140c18794)
		consumeStatborgValueStat(br)
		return variant, nil, true
	case "managed-object-participant-respawn-block-component": // ti=29 i0 (FUN_142ed6a20)
		consumeRespawnBlock(br)
		return variant, nil, true
	case "item-ignore-player-component": // ti=37 i19 (FUN_141101120)
		consumeItemIgnorePlayer(br)
		return variant, nil, true
	case "projectile-at-rest-state": // ti=41 i18 (FUN_141076264) — R(1)+R(1)[si1:R(19)]
		br.ReadBit()
		if br.ReadBit() {
			br.ReadBits(19)
		}
		return variant, nil, true
	case "projectile-command_tick": // ti=41 i20 (FUN_140cec080) — R(1)[si1:R(8)]
		if br.ReadBit() {
			br.ReadBits(8)
		}
		return variant, nil, true
	case "projectile-tether-state": // ti=41 i19 (FUN_142f04850) — R(1) flag
		br.ReadBit()
		return variant, nil, true
	case "projectile-deceleration-disabled-state": // ti=41 i21 (FUN_142f04630) — R(1) flag
		br.ReadBit()
		return variant, nil, true
	case "item-at-rest-component": // ti=37/42 i18 (FUN_140ff91e4) — R(1) flag
		br.ReadBit()
		return variant, nil, true
	case compEquipmentDeployed: // ti=37 i20 (FUN_142ed4618) — R(1) flag
		consumeEquipmentDeployed(br)
		return variant, nil, true
	case "tacmap-iconlodthresholds": // ti=34 i2 (FUN_142ed4848) — R(5)=N + N×R(12)
		for n := br.ReadBits(5); n > 0; n-- {
			br.ReadBits(12)
		}
		return variant, nil, true
	case "tacmap-mapscale": // ti=34 i0 (FUN_142ed5dd8) — R(32)
		br.ReadBits(32)
		return variant, nil, true
	case "tacmap-settingstag": // ti=34 i1 (FUN_142ed6d0c) — R(32)
		br.ReadBits(32)
		return variant, nil, true
	case compEquipmentActivated: // ti=37 i21 (FUN_*)
		consumeEquipmentActivated(br)
		return variant, nil, true
	case "statborg-finalized-rounds-values-stat-component": // ti=6 i28
		consumeStatborgFinalized(br)
		return variant, nil, true
	case "tacmap-areaofinterest": // ti=32 i0 (FUN_142ed3c50)
		consumeTacmapAreaOfInterest(br)
		return variant, nil, true
	case "game-engine-shared-team-lives-component": // ti=0 i1 (FUN_142f03600)
		consumeGameEngineSharedTeamLives(br)
		return variant, nil, true
	case "generic-rigid-body-transforms-component": // ti=38 i18 (FUN_142f036f0)
		consumeGenericRigidBodyTransforms(br)
		return variant, nil, true
	case "equipment-control-signal-component": // ti=37 i22 (FUN_14101cd94) — R(4)+R(1)[+readQuantStat]
		br.ReadBits(4)
		if br.ReadBit() {
			br.readQuantStat(1, quantStatDefaultWidth)
		}
		return variant, nil, true
	case compEquipmentCreator: // ti=37 i23 (FUN_142ed45f4) — R(1)[si0:R(5)]
		consumeEquipmentCreator(br)
		return variant, nil, true
	case compEquipmentEnergy: // ti=37 i24 (FUN_141087bec) — R(14)
		consumeEquipmentEnergy(br)
		return variant, nil, true
	case "tacmap-missionmarkerstate": // ti=34 i8 (FUN_142ed5e90) — R(32)+R(32)
		br.ReadBits(32)
		br.ReadBits(32)
		return variant, nil, true
	case "tacmap-lockedlights": // ti=34 i5 (FUN_142ed4f84) — 4×(R(32)+R(96))
		for i := 0; i < 4; i++ {
			br.ReadBits(32)
			br.ReadBits(96)
		}
		return variant, nil, true
	case "tacmap-cameraheading": // ti=34 i3 (FUN_141168208) — R(12)
		br.ReadBits(12)
		return variant, nil, true
	case "tacmap-waypointstate": // ti=34 i7 (FUN_140f04d74) — R(1)+R(32)+pos e524 (version-gate R(1) externe omis)
		br.ReadBit()
		br.ReadBits(32)
		consumeE524PositionBody(br)
		return variant, nil, true
	case "tacmap-dungeonstate": // ti=34 i4 (FUN_142ed4350) — R(1)+R(96)
		br.ReadBit()
		br.ReadBits(96)
		return variant, nil, true
	case "weapon-ammo-component": // ti=42 i20 (FUN_140fc3028) — R(8)+R(11)+R(12)
		br.ReadBits(8)
		br.ReadBits(11)
		br.ReadBits(12)
		return variant, nil, true
	case "branch-script-results-component": // ti=16 i1 (FUN_142ed3dcc) — R(6)+R(32)
		br.ReadBits(6)
		br.ReadBits(32)
		return variant, nil, true
	case compHighFrequency: // ti=4 i0 — variante FRAME (FUN_14076d034) = R(8), sonde
		publishProbe(typeIndex, ProbeHighFrequency, br.ReadBits(8))
		return variant, nil, true
	case "animated-mesh-dynamic-state-component": // ti=38 i19 (FUN_142f0258c) — R(8)+R(1)+R(16)
		br.ReadBits(8)
		br.ReadBit()
		br.ReadBits(16)
		return variant, nil, true
	// ---- batch3 (workflow port-worldobject-batch3) ----
	case "tacmap-fasttravelstate": // ti=34 i6 (FUN_14116fc90) — R(1)
		br.ReadBit()
		return variant, nil, true
	case "tacmap-missioncount": // ti=34 i16 (FUN_142ed5e68) — R(9)
		br.ReadBits(9)
		return variant, nil, true
	case "equipment-being-hacked-component": // ti=37 i25 (FUN_142ed441c) — R(8)
		br.ReadBits(8)
		return variant, nil, true
	case compEquipmentEnergyDelay: // ti=37 i26 (FUN_140dda128) — R(10), publie
		consumeEquipmentEnergyDelay(br)
		return variant, nil, true
	case compEquipmentCharges: // ti=37 i27 (FUN_142ed4518) — R(8), publie
		consumeEquipmentCharges(br)
		return variant, nil, true
	case "player-waypoint-component": // ti=5 i0 (FUN_1410665dc) — R(3)
		br.ReadBits(3)
		return variant, nil, true
	case "player-respawn-timer-component": // ti=5 i1 (FUN_140f3fb8c) — R(1)+R(10)+R(10) = 21 bits
		// Grammaire dans decodePlayerRespawnTimer (vitality.go) — copie unique.
		_ = decodePlayerRespawnTimer(br)
		return variant, nil, true
	case compPlayerSoftKillTimer: // ti=5 i2 (FUN_140d580a8->FUN_140d580d0) — publie
		consumePlayerSoftKillTimer(br)
		return variant, nil, true
	case compPlayerTargetTracking: // ti=5 i3 (FUN_142f044f0) — publie
		consumePlayerTargetTracking(br)
		return variant, nil, true
	case "player-unsafe-respawn-timer-component": // ti=5 i4 (FUN_142f0452c) — R(10)
		br.ReadBits(10)
		return variant, nil, true
	case "player-respawn-safety-component": // ti=5 i5 (FUN_1410e3f30->FUN_140ebf854) — R(5)
		br.ReadBits(5)
		return variant, nil, true
	case compPlayerDesiredRespawnPlayer: // ti=5 i6 (FUN_1410f7330) — publie
		consumePlayerDesiredRespawnPlayer(br)
		return variant, nil, true
	// --- lot workflow filmdec-port-component-desers (2026-06-30, RE Ghidra parallele) ---
	// HIGH-confidence : largeurs fixes verifiees bit-exact.
	case "crew-orders-off-flags-component": // ti=14 i2 (FUN_142ed428c) — R(8)
		br.ReadBits(8)
		return variant, nil, true
	case "statborg-entry-index-and-type-component": // ti=6 i57 (FUN_1410be614) — R(32)+R(8)
		br.ReadBits(32)
		br.ReadBits(8)
		return variant, nil, true
	case "physics-state-component": // ti=22 i0 (FUN_142ed6c20) — R(32)+R(1)[si1:R(32)]
		br.ReadBits(32)
		if br.ReadBit() {
			br.ReadBits(32)
		}
		return variant, nil, true
	case "flock-emitting-component": // ti=21 i0 (FUN_142ed46c8) — R(1)
		br.ReadBit()
		return variant, nil, true
	case "music-variables-component": // ti=17 i0 (FUN_142ed66bc) — 32x{R(1); si bit==0: R(2)+R(32)+R(32)}
		for i := 0; i < 32; i++ {
			if !br.ReadBit() {
				br.ReadBits(2)
				br.ReadBits(32)
				br.ReadBits(32)
			}
		}
		return variant, nil, true
	case "managed-player-team-designator-component": // ti=9 i0 (FUN_140f581e8) — R(4)
		br.ReadBits(4)
		return variant, nil, true
	case compGameEngineCurrentState: // ti=0 i2 (FUN_14116d1d0) — publie
		consumeGameEngineCurrentState(br)
		return variant, nil, true
	case "game-engine-game-finished-component": // ti=2 i3 (FUN_142f035e0) — R(1)
		br.ReadBit()
		return variant, nil, true
	case compManagedObjectPropName: // ti=13 i0 (FUN_142ed69d8) — R(32), sonde
		publishProbe(typeIndex, ProbeManagedObjectPropertyName, br.ReadBits(32))
		return variant, nil, true
	case "managed-navpoint-sub-type-component": // ti=12 i0 (FUN_1410e0cac) — R(32)
		br.ReadBits(32)
		return variant, nil, true
	case "player-early-respawn-requested-component": // ti=5 i8 (FUN_142f04034) — R(1)
		br.ReadBit()
		return variant, nil, true
	case "player-engine-loadout-index-component": // ti=5 i9 (FUN_142f04054) — R(3)
		br.ReadBits(3)
		return variant, nil, true
	case "biped-emp-timer-component": // ti=35 i51 (FUN_142f02830) — R(8) (timer quant 0..10s)
		publishEmpTimer(br.ReadBits(8)) // cf. emp_timer.go — largeur inchangée, valeur publiée
		return variant, nil, true
	case compSplashMessageDynamic: // ti=47 i1 (FUN_140daebd0) — R(24), sonde
		publishProbe(typeIndex, ProbeSplashDynamic, br.ReadBits(24))
		return variant, nil, true
	// LOW-confidence : largeur quantifiee runtime data-dependent sur la branche gate==1.
	// On porte le cas commun (gate==0) et on desync PROPREMENT sur la branche data-dependent
	// (jamais de bit-guess = pas de corruption silencieuse en aval).
	case "crew-order-component": // ti=14 i0 (FUN_142ed4274) — R(3)+R(1)gate1[si1: vec3 quant 6+level]
		br.ReadBits(3)
		if br.ReadBit() { // gate1 == présence du vecteur
			consumeQuantVec3(br, quantAxisWidth(uint(level))) // PISTE 1 : largeur = 6+flags(registre)
		}
		return variant, nil, true
	case "tacmap-poiiconoffset": // ti=30 i1 — vec3 quant pur (6+level)
		consumeQuantVec3(br, quantAxisWidth(uint(level)))
		return variant, nil, true
	case "tacmap-poiicon": // ti=30 i0 (FUN_142ed8418) — bloc + vec3 quant(6+level) au milieu
		br.ReadBits(32) // icon-id
		br.ReadBits(32) // icon-missionid
		br.ReadBit()
		br.ReadBits(3)
		br.ReadBits(32) // icon-text
		br.ReadBits(32) // icon-bitmapbg
		br.ReadBits(9)
		br.ReadBits(9)
		consumeQuantVec3(br, quantAxisWidth(uint(level))) // vec3
		br.ReadBits(32)                                   // string-id
		br.ReadBit()
		br.ReadBits(8)
		br.ReadBits(8)
		br.ReadBits(8)
		br.ReadBits(8)
		return variant, nil, true
	case "flock-destination-component": // ti=21 i2-i11 — R(1)flag + vec3 quant(6+level) + R(2) si rsp>1
		br.ReadBit()
		consumeQuantVec3(br, quantAxisWidth(uint(level)))
		if paramForComponent(name) > 1 {
			br.ReadBits(2)
		}
		return variant, nil, true
	case "crew-marked-objects-component": // ti=14 i1 (FUN_142ed421c) — R(1); gate==1 -> R(W)+R(2) W runtime
		if br.ReadBit() {
			return variant, nil, false
		}
		return variant, nil, true
	case "nav-cutscene-flag-component": // ti=15 i0 (FUN_142ed69b8) — R(1); present==1 -> gros bloc vec3 runtime
		if br.ReadBit() {
			return variant, nil, false
		}
		return variant, nil, true
	case "player-desired-respawn-seat-component": // ti=5 i7 (FUN_142f03f34) — R(1)[si1:R(W)+R(2) runtime]+R(7)
		if br.ReadBit() {
			return variant, nil, false
		}
		br.ReadBits(7)
		return variant, nil, true
	case "effect-state-data-component": // ti=18 i0 (FUN_142ed438c) — gates R(1)/R(32) ; markers R(W) runtime -> desync
		if br.ReadBit() {
			br.ReadBits(32) // tag-ref
		}
		if br.ReadBit() {
			return variant, nil, false // markerA -> R(W)+R(2) runtime
		}
		br.ReadBits(32) // marker-name
		if br.ReadBit() {
			return variant, nil, false // 2e marker -> R(W) runtime
		}
		return variant, nil, true
	// --- lot 2 workflow filmdec-port-component-desers (2026-06-30) ---
	// HIGH-confidence : largeurs fixes verifiees bit-exact.
	case "tacmap-poiisgoldenpath": // ti=30 i2 (FUN_142ed4800) — R(1)+R(1)
		br.ReadBit()
		br.ReadBit()
		return variant, nil, true
	case compPlayerEngineLoadout: // ti=5 i11 (FUN_141044428) — 8xR(8)=64, publie
		consumePlayerEngineLoadout(br)
		return variant, nil, true
	case "player-fade-properties-component": // ti=5 i13 (FUN_141020bac) — 6xR(12)=72
		for i := 0; i < 6; i++ {
			br.ReadBits(12)
		}
		return variant, nil, true
	case compPlayerLivesRemaining: // ti=5 i14 (FUN_141055734) — R(7), publie
		consumePlayerLivesRemaining(br)
		return variant, nil, true
	case "managed-player-color-override-component": // ti=9 i1 (FUN_142ed5b54) — 8xR(8)=64 (2 couleurs RGBA quant)
		for i := 0; i < 8; i++ {
			br.ReadBits(8)
		}
		return variant, nil, true
	case "managed-player-flags-component": // ti=9 i2 (FUN_142ed5bac) — R(4)
		br.ReadBits(4)
		return variant, nil, true
	case "managed-player-back-button-scoreboard-flair-component": // ti=9 i3 (FUN_142ed5af4) — R(32)
		br.ReadBits(32)
		return variant, nil, true
	case "music-state-component": // ti=17 i1 (FUN_142ed5ef0) — R1+R32+R33+R33+gate1{R1+R16+R16}+gate2{6xR16}
		br.ReadBit()
		br.ReadBits(32)
		br.ReadBits(32)
		br.ReadBit()
		br.ReadBits(32)
		br.ReadBit()
		if br.ReadBit() {
			br.ReadBit()
			br.ReadBits(16)
			br.ReadBits(16)
		}
		if br.ReadBit() {
			br.ReadBits(16)
			br.ReadBits(16)
			br.ReadBits(16)
			br.ReadBits(16)
			br.ReadBits(16)
			br.ReadBits(16)
		}
		return variant, nil, true
	case "flock-destroying-component": // ti=21 i1 (FUN_142ed468c) — R(1)
		br.ReadBit()
		return variant, nil, true
	case "flock-forced-respawn-component": // ti=21 i12 (FUN_142ed4740) — R(1)
		br.ReadBit()
		return variant, nil, true
	case "flock-relevancy-component": // ti=21 i13 (FUN_14110d2fc) — R(8) (float quant 0..1)
		br.ReadBits(8)
		return variant, nil, true
	case "game-engine-round-timer-component": // ti=0 i5 (FUN_1407ee790) — R(16)+R(16)+R(5)
		// Grammaire dans decodeGameEngineRoundTimer (vitality.go) — copie unique.
		_ = decodeGameEngineRoundTimer(br)
		return variant, nil, true
	case compGameEngineSuddenDeath: // ti=0 i6 (FUN_14116d3a4) — R(16)+R(16)+R(5), publie
		consumeGameEngineSuddenDeath(br)
		return variant, nil, true
	case "game-engine-screen-sequence-component": // ti=0 i9 (FUN_14101d118) — R(4)+R(8)
		br.ReadBits(4)
		br.ReadBits(8)
		return variant, nil, true
	// LOW safe-prefix : cas dominant (gate==0 = absent) avance ; gate==1 (largeur runtime) desync propre.
	case "player-primary-respawn-object-component": // ti=5 i10 (FUN_142f043e8) — R(1)[si1:R(W)+R(2) runtime]
		if br.ReadBit() {
			return variant, nil, false
		}
		return variant, nil, true
	case compPlayerDesiredRespawnLoc: // ti=5 i12 — R(1)[si1: vec3 quant(6+level) + R(19)], publie
		consumePlayerDesiredRespawnLocation(br, level)
		return variant, nil, true
	// --- lot 3 workflow filmdec-port-component-desers (2026-06-30) ---
	case compPlayerLastBetrayer: // ti=5 i15 (FUN_142f04158) — R(6), publie
		consumePlayerLastBetrayer(br)
		return variant, nil, true
	case "player-vehicle-entrance-ban-component": // ti=5 i16 (FUN_142f04610) — R(1)
		br.ReadBit()
		return variant, nil, true
	case compPlayerControlAiming: // ti=5 i17 (FUN_142f03ea4) — R(19) direction cubemap, publie
		consumePlayerControlAiming(br)
		return variant, nil, true
	case compPlayerActiveInGame: // ti=5 i18 (FUN_1411615d8) — R(1), publie
		consumePlayerActiveInGame(br)
		return variant, nil, true
	case compPlayerPendingJoinInProgress: // ti=5 i19 (FUN_1411615b8) — R(1), publie
		consumePlayerPendingJoinInProgress(br)
		return variant, nil, true
	case compPlayerMalleableProperties: // ti=5 i20 (FUN_1407f0518) — publie, bits de porte compris
		consumePlayerMalleableProperties(br)
		return variant, nil, true
	case "player-representation-component": // ti=5 i21 (FUN_14111ec64) — R(32)
		br.ReadBits(32)
		return variant, nil, true
	case "player-power-frame-points-component": // ti=5 i23 (FUN_142f04240) — R(16)+R(16)
		br.ReadBits(16)
		br.ReadBits(16)
		return variant, nil, true
	case "player-supply-lines-currency-simulation-component": // ti=5 i25 (FUN_142f04408) — R(16)
		br.ReadBits(16)
		return variant, nil, true
	case "player-allowed-to-quit-component": // ti=5 i26 (FUN_142f04138) — R(1)
		br.ReadBit()
		return variant, nil, true
	case "managed-player-active-mission-name-component": // ti=9 i5 (FUN_142ed5ab0) — R(32)+R(32)
		br.ReadBits(32)
		br.ReadBits(32)
		return variant, nil, true
	case "managed-player-show-active-mission-name-in-hud-component": // ti=9 i6 (FUN_142ed5d68) — R(1)
		br.ReadBit()
		return variant, nil, true
	case "managed-player-campaign-progress-component": // ti=9 i7 (FUN_142ed5b18) — R(8)
		br.ReadBits(8)
		return variant, nil, true
	case "managed-player-current-season-component": // ti=9 i8 (FUN_142ed5b88) — R(32)
		br.ReadBits(32)
		return variant, nil, true
	case "managed-player-custom-input-prompt-widget": // ti=9 i9 (FUN_141fcf160) — gates+R(32)+R(3)count[si>0:union -> desync]
		if !br.ReadBit() { // gate_present==0 -> stop
			return variant, nil, true
		}
		br.ReadBits(2)
		if !br.ReadBit() { // gate_sub==0 -> stop
			return variant, nil, true
		}
		br.ReadBits(32)
		if br.ReadBits(3) != 0 { // count>0 -> boucle variant taggee, desync propre
			return variant, nil, false
		}
		return variant, nil, true
	case compGameEngineGracePeriod: // ti=0 i7 (FUN_141165d24) — R(16)+R(16)+R(5), publie
		consumeGameEngineGracePeriod(br)
		return variant, nil, true
	case compGameEngineRoundConditionFlags: // ti=0 i8 (FUN_141132dc0) — R(10), publie
		consumeGameEngineRoundConditionFlags(br)
		return variant, nil, true
	case "game-engine-alliance-component": // ti=0 i10 (FUN_140a24968) — R(32) mask + popcount*(R(32)+R(32))
		mask := uint32(br.ReadBits(32))
		for i := 0; i < 32; i++ {
			if (mask>>uint(i))&1 != 0 {
				br.ReadBits(32)
				br.ReadBits(32)
			}
		}
		return variant, nil, true
	case compGameEngineCurrentRound: // ti=0 i4 (FUN_14116fc70) — R(1)[si0:R(5)], publie
		consumeGameEngineCurrentRound(br)
		return variant, nil, true
	case "tacmap-backmenu-openoverride": // ti=34 i13 (FUN_142ed3d64)
		consumeTacmapBackmenuOpenoverride(br)
		return variant, nil, true
	case "tacmap-queuedreplaymission": // ti=34 i9 (FUN_1407f24f8)
		consumeTacmapQueuedReplayMission(br)
		return variant, nil, true
	case "tacmap-cooptetherarea": // ti=34 i11 (FUN_142ed4198)
		consumeTacmapCoopTetherArea(br)
		return variant, nil, true
	case "tacmap-displayasset": // ti=33 i0 (FUN_142ed433c)
		consumeTacmapDisplayAsset(br)
		return variant, nil, true
	case "equipment-tracked-object-handles-stack-component": // ti=37 i28 (FUN_140f72dec)
		consumeEquipmentTrackedStack2(br)
		return variant, nil, true
	case "equipment-command-tick-component": // ti=37 i29 (FUN_140e0a564, table ECS live)
		consumeEquipmentCommandTick(br)
		return variant, nil, true
	case "equipment-has-infinite-uses-component": // ti=37 i30 (FUN_142ed4640, table ECS live) — R(1)
		br.ReadBit()
		return variant, nil, true
	case "track-frame-component": // ti=16 i5 (FUN_142ed740c)
		consumeTrackFrameComponent(br)
		return variant, nil, true
	case "statborg-round-outcomes-component": // ti=6 i56 (FUN_142ed71a4) — 32×R(2)
		consumeStatborgRoundOutcomes(br)
		return variant, nil, true
	case compSplashMessageStatic: // ti=47 i0 (FUN_141085d50) — sonde sur le R(24) inconditionnel
		publishProbe(typeIndex, ProbeSplashStatic, consumeManagedSplashMessage(br))
		return variant, nil, true
	case "object-multiplayer-properties-component": // i9 = the 'obje' (FUN_1407d4c94 TLV blob)
		consumeObjectMultiplayerProperties(br)
		return variant, nil, true
	case compWeaponStateTypeInfo: // held weapon (FUN_1407f06bc)
		v := consumeWeaponStateTypeInfoVariant(br)
		return v, nil, true
	case "object-forward-and-up-component":
		consumeObjectForwardAndUp(br)
		return variant, nil, true
	case compObjectBodyVitality:
		consumeObjectBodyVitality(br)
		return variant, nil, true
	case "object-shield-vitality-component":
		consumeObjectShieldVitality(br)
		return variant, nil, true
	case "object-dead-state-component":
		// FUN_140c1dce0 : le corps lourd (FUN_140c1dd44) n'est lu QUE si typeIndex vaut
		// 0x23 (35, biped) ou 0x28 (40) ; le R(1) de queue, QUE si 0x23. Les autres
		// archetypes porteurs du composant (a36..a39, a41..a43, ti6/38/42 mesures sur
		// ce_capture_delta) ne lisent qu'UN bit : appliquer la forme lourde partout
		// sur-consommait ~130 bits et desynchronisait leur record.
		if typeIndex == 0x23 || typeIndex == 0x28 {
			ds := consumeObjectDeadStateBipedTI(br, typeIndex)
			return variant, &ds, true
		}
		ds := DeadState{Mort: consumeObjectDeadState(br), GlobalID: 0xFFFFFFFF,
			EnumA: -1, EnumB: -1, Val18: -1, SrcTag0: 0xFFFFFFFF, SrcTag4c: 0xFFFFFFFF}
		return variant, &ds, true
	case "weapon-state-ammo":
		consumeWeaponStateAmmo(br)
		return variant, nil, true
	case "weapon-state-rounds-inventory":
		consumeWeaponStateRoundsInventory(br)
		return variant, nil, true
	case "weapon-state-overheated":
		consumeWeaponStateOverheated(br)
		return variant, nil, true
	case "biped-desired-weapon-set":
		consumeBipedDesiredWeaponSet(br)
		return variant, nil, true
	case "biped-desired-grenade-set", "biped-desired-grenade-set-component": // i47 (FUN_140c6a638)
		consumeBipedDesiredGrenadeSet(br)
		return variant, nil, true
	case "biped-desired-ability-set", "biped-desired-ability-set-component": // i48 (FUN_1406d0ff0)
		consumeBipedDesiredAbilitySet(br)
		return variant, nil, true
	case "biped-control-context", "biped-control-context-component": // i49 (FUN_14107166c)
		consumeBipedControlContext(br)
		return variant, nil, true
	case "biped-map-editor-flag", "biped-map-editor-flag-component": // i50 (FUN_142f02854)
		consumeBipedMapEditorFlag(br)
		return variant, nil, true
	case "biped-low-frequency-data", "biped-low-frequency-data-component": // i52 (FUN_140fc91e0)
		consumeBipedLowFrequencyData(br)
		return variant, nil, true
	case "biped-malleable-property", "biped-malleable-property-component": // i53 (FUN_140ff6764)
		consumeBipedMalleableProperty(br)
		return variant, nil, true
	case "biped-mobility-action", "biped-mobility-action-component": // i54 (FUN_1408f0264)
		consumeBipedMobilityAction(br)
		return variant, nil, true
	case "biped-spartan-ability-energy", "biped-spartan-ability-energy-component": // i56 (FUN_140fc1410)
		consumeBipedSpartanAbilityEnergy(br)
		return variant, nil, true
	case "flock-position-component": // ti21 i16 (FUN_140ee7270 -> FUN_14076e524)
		consumeFlockPosition(br, uint(level))
		return variant, nil, true
	case "flock-current-destination-component": // ti21 i17 (FUN_142ed4668 -> FUN_142ed0674)
		consumeFlockCurrentDestination(br)
		return variant, nil, true
	case "flock-remembered-danger-component": // ti21 i15 (FUN_142ed477c)
		consumeFlockRememberedDanger(br)
		return variant, nil, true
	case "flock-fleeing-component": // ti21 i14 (FUN_142ed4704)
		consumeFlockFleeing(br)
		return variant, nil, true
	case "biped-spartan-ability-component": // i57 (FUN_142f02810 -> FUN_142f268c4)
		// Rend ported=false sur la seule branche non determinable (tag brut == 3 ->
		// FUN_142f262d4, gate par des octets d'etat runtime) : desync propre plutot que
		// desalignement silencieux.
		return variant, nil, consumeBipedSpartanAbility(br)
	case "biped-spartan-ability-non-predicted-state", "biped-spartan-ability-non-predicted-state-component": // i59 (FUN_142f02994)
		// Corps tag==3 (FUN_142f25e90, ancre du grappin) porté le 2026-08-16 : rend
		// ported=false sur les seules valeurs internes jamais observées — désync propre,
		// même contrat qu'i57 ci-dessus. param_4 vient de paramForComponent (i59 -> 2,
		// la queue R(3) est lue — l'ancien global brut valait 0 et la sautait).
		return variant, nil, consumeBipedSpartanAbilityNonPredictedState(br, paramForComponent(name))
	case "simulation-state", "simulation-state-component": // i60 (thunk 142f02434 -> FUN_142ED6D88, vérifié live)
		// GRAMMAIRE COMPLÈTE depuis le 2026-08-17 (lot R7-b) : structure connue (flag +
		// 2×gate5 + 8×R16 + 2×R2 + R1[R19]+R8) PLUS la queue FUN_14076e494, dont le prédicat
		// de garde s'est révélé vrai par construction (cf. consumeSimulationState).
		// Le drapeau `simStateComplete` reste la porte : ce n'est plus la GRAMMAIRE qui manque,
		// c'est la SOURCE DES LARGEURS D'AXE de la queue en production (cf. simStateComplete).
		consumeSimulationState(br)
		return variant, nil, simStateComplete
	case "simulation-state-playback", "simulation-state-playback-component": // i61 (thunk 142f02454 -> FUN_142ed6d20, vérifié live)
		consumeSimulationStatePlayback(br)
		return variant, nil, true
	case "biped-slide", "biped-slide-component": // i62 (FUN_142f02978 -> FUN_142f26ce8)
		consumeBipedSlide(br)
		return variant, nil, true
	case "biped-action", "biped-action-component": // i63 (FUN_142f027f4 -> FUN_142f26a20)
		// Returns ported=false on the value-gated loop1 dispatch (count>0) so the
		// traversal desyncs cleanly instead of mis-aligning. Common case: 196 bits.
		return variant, nil, consumeBipedAction(br)
	case compManagedObjectBoundaryVisibility: // ti=10 i0 (FUN_141169e90 -> FUN_14080ae28) — 32xR(1), publie
		consumeManagedObjectBoundaryVisibility(br)
		return variant, nil, true
	case compManagedObjectBoundaryColor: // ti=10 i1 (FUN_142ed52b4) — 4xR(8) RGBA, publie
		consumeManagedObjectBoundaryColor(br)
		return variant, nil, true
	case compManagedObjectRTPC: // ti=10 i26..i29 (FUN_140796d38) — R(32) id [+R(22)], publie
		consumeManagedObjectRTPC(br)
		return variant, nil, true
	case compNavpointRadialProgress: // ti=12 i14 (FUN_140fc8d14) — R(8), publie
		consumeNavpointRadialProgress(br)
		return variant, nil, true
	case compManagedObjectProperty: // ti=13 i1 (FUN_140ce5554 -> FUN_140ce59bc) — variant mode A, publie
		consumeManagedObjectProperty(br)
		return variant, nil, true
	case compManagedObjectPlayerMaskedProperty: // ti=13 i2..i33 (FUN_140ce593c -> FUN_140ce59bc) — variant mode B, publie
		consumeManagedObjectPlayerMaskedProperty(br)
		return variant, nil, true
	case compManagedObjectiveTimers: // ti=11 i0 (FUN_142ed5a6c -> FUN_1410d9088) — 2xR(7), publie
		consumeManagedObjectiveTimers(br)
		return variant, nil, true
	case compManagedObjectiveColor: // ti=11 i1 (FUN_142ed544c) — 4xR(8) RGBA, publie
		consumeManagedObjectiveColor(br)
		return variant, nil, true
	case compManagedObjectiveObjectRef: // ti=11 i3 (FUN_142ed5550) — R(32) GlobalID (LE PORTEUR), publie
		consumeManagedObjectiveU32(br, ManagedObjectiveObjectRef)
		return variant, nil, true
	case compManagedObjectiveType: // ti=11 i5 (FUN_1410fc4a4) — R(32), publie
		consumeManagedObjectiveU32(br, ManagedObjectiveType)
		return variant, nil, true
	case compManagedObjectiveProgress: // ti=11 i12 (FUN_142ed575c) — R(32), publie
		consumeManagedObjectiveU32(br, ManagedObjectiveProgress)
		return variant, nil, true
	case compManagedObjectiveRequired: // ti=11 i13 (FUN_142ed5844) — R(32), publie
		consumeManagedObjectiveU32(br, ManagedObjectiveRequired)
		return variant, nil, true
	case compManagedObjectiveState: // ti=11 i14 (FUN_142ed5948) — R(3), publie
		consumeManagedObjectiveState(br)
		return variant, nil, true
	case compManagedObjectiveParent: // ti=11 i15 (FUN_142ed5674) — R(32) GlobalID, publie
		consumeManagedObjectiveU32(br, ManagedObjectiveParent)
		return variant, nil, true
	case compManagedObjectiveSubEntity: // ti=11 i16..i31 (FUN_142ed5974) — R(32) GlobalID, publie
		consumeManagedObjectiveU32(br, ManagedObjectiveSubEntity)
		return variant, nil, true
	case "device-position-component": // ti43 (FUN_140bef320) — R(14)+R(1)
		consumeDevicePosition(br)
		return variant, nil, true
	case "game-engine-campaign-timer-component": // ti2 (FUN_1407ee764) — R(16)+R(16)+R(5)
		consumeGameEngineCampaignTimer(br)
		return variant, nil, true
	case "biped-posture-physics-component": // ti35 i55 (FUN_142f0293c -> FUN_142f1f630) — R(2)
		consumeBipedPosturePhysics(br)
		return variant, nil, true
	case "game-engine-team-mapping", "game-engine-team-mapping-component": // typeIdx=0 i0 (FUN_140f58200)
		consumeGameEngineTeamMapping(br)
		return variant, nil, true
	case "unit-control-component":
		consumeUnitControl(br)
		return variant, nil, true
	case "unit-grenade-counts-component":
		consumeUnitGrenadeCounts(br)
		return variant, nil, true
	case "unit-equipment-component":
		consumeUnitEquipment(br)
		return variant, nil, true
	case "unit-crouch-component":
		consumeUnitCrouch(br)
		return variant, nil, true
	case "unit-active-camo-state-component":
		consumeUnitActiveCamoState(br)
		return variant, nil, true
	case "unit-command-tick-component":
		consumeUnitCommandTick(br)
		return variant, nil, true
	case "unit-low-frequency-component":
		consumeUnitLowFrequency(br)
		return variant, nil, true
	case "unit-stun-component":
		consumeUnitStun(br)
		return variant, nil, true
	case "unit-desired-aiming-vector-component":
		consumeUnitDesiredAimingVector(br)
		return variant, nil, true
	case "asset-transform-component":
		// ti44 i0 (deser FUN_142ed3c64, resolu STATIQUEMENT : chaine .rdata 143c949b0 ->
		// getName 141178040 -> descripteur 143d08c18 -> +0x20 thunk -> +0x28 = FUN_142ed3c64).
		// Corps = 5 x FUN_142ed9530, chacune = FUN_14076e494(...,0x1e,0,0,0) -> FUN_14076e524 :
		//   R(1) gate ; si gate==0 -> R(DAT_144632be0 = 1) index ; puis 3 x R(6+L) (FUN_140cc5128).
		// L = niveau de precision du composant dans chunk_00 (ti44 i0 = L0 -> 6 bits/axe), la
		// largeur venant de la table DAT_1445cc9e0 indexee par le niveau (largeur = 6+L, verifie
		// sur le dump ce_prec_widths_1445cc9e0.bin).
		for i := 0; i < 5; i++ {
			consumeQuantVec3WithGate(br, quantAxisWidth(uint(level)))
		}
		return variant, nil, true
	case "spawn-filter-weight-component":
		// CÂBLÉ 2026-07-25. Deser confirmé STATIQUEMENT (nom -> chaîne .rdata 143c96530 ->
		// getName 141177840 -> descripteur 143d05e00 -> +0x20 = thunk FUN_14076ce9c ->
		// +0x28 = FUN_142ed70b8), qui est un unique FUN_1406d84b4 de largeur 0x10 = R(16).
		// Ce composant était le 1er COUPABLE mesuré contre l'oracle de position (4970 des
		// 6885 paquets confrontés rompaient dessus) : il était RE'd mais jamais dispatché.
		consumeSpawnFilterWeight(br)
		return variant, nil, true
	// ---- Batch8 (workflow filmdec-port-top-components, 2026-06-14) : NON CÂBLÉ ----
	// Les 16 desers RE'd (components_batch8.go) sont EXE-vérifiés en isolation, MAIS les câbler
	// dans le World-seed persistant FAIT BAISSER le gradient (63%->25%) : ils complètent des
	// recNew à un bit erroné (largeurs runtime / erreurs subtiles) -> frames false-clean ->
	// bindings garbage qui persistent (rollback ne couvre que les frames en erreur) -> cascade
	// de slots non-bindés (group-2 forge ti=48 = 14184 unbound). Le component-grind via seed NE
	// CONVERGE PAS sur le binding gap. Gardés comme référence ; câblage en attente d'un chemin
	// de binding robuste (replay propre depuis keyframe = mur deser default-state, cf handoff L3).
	default:
		// Un-ported (object position/velocity/angular/region/damage/constraint/parent/
		// scale/..., unit-actor-control/state/malleable, biped-* tail): stop cleanly.
		return variant, nil, false
	}
}

// bipedDefaultStateTypeIndex is the keyframe typeIndex of the biped archetype
// (#35), the one whose default-state is deserialised by FUN_140F44C38
// (consumeBipedDefaultState). For this typeIndex TraverseEntity uses the bit-exact
// deser instead of a fixed Skip(defaultStateBits).
const bipedDefaultStateTypeIndex = 35

// objectArchetypeCount est le nombre d'archétypes OBJET enregistrés dans la table ECS du .exe
// (DAT_144e61d88, peuplée par le registrar FUN_140e453b4 : ti 0x00..0x31, COUNT=0x32=50). Un record
// NEW avec R(6) typeIndex >= 50 est IMPOSSIBLE (FUN_1408f1aa4 retourne erreur 3 sur descripteur NULL).
// ⇒ garde-rail : un ti >= 50 est la PREUVE d'une désync (curseur en zone de données), pas un vrai record.
// (Découverte autoritative .exe, 2026-07-03.)
const objectArchetypeCount = 50

// useBipedDefaultStateDeser routes typeIndex==35 records through the bit-exact
// consumeBipedDefaultState (FUN_140F44C38) for the part of the default-state it
// actually covers, then skips the measured residue up to defaultStateBits. It is a
// package var (default true) so the calibration harness can A/B it against the
// pure fixed-Skip path. See consumeBipedDefaultState / default_state.go.
//
// MEASURED FACT (cmd/tmp_defstate_measure, cmd/tmp_mpp): on the Hydra record
// FUN_140F44C38 (vtable[0x60], the ONLY vtable slot that receives the bitreader RSI
// — confirmed by asm at FUN_141f86704 @141f868c2 `MOV R9, RSI`) consumes 120 bits of
// the 380-bit default-state, NOT 32. The earlier 32-bit figure was wrong because the
// port treated FUN_14080cfe8 (the object-multiplayer-properties block) as 0 bits;
// the asm at FUN_140F44C38 @140f44d0e (`MOV RDX, RDI` = bitreader) proves it consumes
// the stream. consumeMultiplayerPropertiesBlock now ports it bit-exact (+88 bits).
//
// The remaining 260 bits are read by FUN_140F44C38 leaf paths whose widths are
// populated at map-load from the film's replication/precision config (DAT_1445cc9e0
// per-axis widths, DAT_144632be0 index width, DAT_145121140) and read 0 statically —
// the same limitation already affecting TraversalPrecision / PrecisionDescriptor. On
// the calibration record those config-gated paths form a regular 96-bit block (a
// quantized vec3/quat, "0x3FC,0" x4 @bit194256) plus dense words: NOT a free loop,
// but driven by runtime widths not recoverable from the .exe. vtable[0x88] gets no
// bitreader and the FUN_141f86704 config tail is <=2 bits, so they cannot host it.
// Until those runtime widths are sourced from the film header, the 120-bit bit-exact
// prefix runs and the residue is skipped to preserve the calibrated total. Set
// useBipedDefaultStateDeser=false for the legacy pure-Skip behaviour.
var useBipedDefaultStateDeser = true

// SetUseBipedDefaultStateDeser toggles the bit-exact biped default-state deser
// prefix (vs a pure Skip(defaultStateBits)).
func SetUseBipedDefaultStateDeser(v bool) { useBipedDefaultStateDeser = v }

// TraverseEntity decodes one new-entity record header (R6 typeIndex + default-state
// + R1 gate + presence mask) and walks the archetype's present components via
// consumeByName. It stops at the first un-ported present component (DesyncAt). The
// held-weapon variant is captured when a weapon-state-type-info slot is reached.
//
// For the biped archetype (typeIndex==35) the leading part of the default-state is
// deserialised bit-exact via consumeBipedDefaultState (FUN_140F44C38) + the
// config-gated FUN_141f86704 tail; any residue up to defaultStateBits is skipped
// (the residue deser is not yet identified — see useBipedDefaultStateDeser). For
// other archetypes the legacy fixed Skip(defaultStateBits) is kept.
func TraverseEntity(br *BitReader, reg *Registry, defaultStateBits int) EntityTrace {
	t := EntityTrace{DesyncAt: -1, DefaultBits: defaultStateBits}
	t.TypeIndex = uint32(br.ReadBits(6))
	if t.TypeIndex >= objectArchetypeCount {
		// typeIndex objet cappé < 50 (.exe) : au-delà = désync (curseur en zone de données), pas un record.
		t.DesyncAt = 0
		t.EndBit = br.BitPos()
		return t
	}
	if useBipedDefaultStateDeser && t.TypeIndex == bipedDefaultStateTypeIndex {
		// VALIDÉ BIT-EXACT en live (CE breakpoint sur FUN_140f44c38 : rep biped = 166 ou 198
		// bits selon la donnée ; consumeBipedDefaultState consomme EXACTEMENT 198 sur le record
		// capturé). Le default-state = la longueur EXACTE du deser, PAS un skip vers le "380"
		// (qui était un faux-propre incluant des bits de la boucle de composants). Le résidu-skip
		// historique est SUPPRIMÉ.
		consumeBipedDefaultState(br)     // FUN_140F44C38, self-délimité, bit-exact
		consumeBipedDefaultStateTail(br) // FUN_141f86704 config-gated tail (default 0)
	} else if db, ok := defaultStateBitsByTI[t.TypeIndex]; ok {
		br.Skip(db) // surcharge de calibration (harness keyframe) : prioritaire sur le port
	} else if fn, ok := defaultStateDeserByTI[t.TypeIndex]; ok && useArchDefaultStateDeser {
		fn(br) // deser vtable[0x60] porté bit-exact (cf. default_state_arch.go)
	} else {
		br.Skip(defaultStateBits) // fallback : stub 0-bit (défaut) ou largeur globale de calibration
	}
	// t.Gate : R(1) réel AVANT le masque dans le record NEW (pré-boucle, cf FUN_1408f1aa4).
	// N'existe PAS dans le path DELTA (decodeDelta appelle consumeMask seul). Le retirer casse
	// le décodage (désync) — c'est un vrai bit, pas un double-gate.
	t.Gate = br.ReadBit()
	t.Mask = consumeMask(br)

	arch, ok := reg.Archetype(int(t.TypeIndex))
	if !ok {
		t.DesyncAt = 0
		t.EndBit = br.BitPos()
		return t
	}
	traverseComponentLoop(br, arch, &t)
	if t.DesyncAt == -1 && newRecordTailBits > 0 {
		br.Skip(newRecordTailBits) // tail terminal per-record NEW (calibration keyframe)
	}
	t.EndBit = br.BitPos()
	return t
}

// traverseComponentLoop walks an archetype's ORDERED component list and decodes each
// present component (mask bit set) via consumeByName, in iteration order. It is the
// shared body between the keyframe/NEW path (TraverseEntity, which prepends the
// R6 typeIndex + default-state + gate header) and the FRAME delta path (decodeDelta,
// which has NO header — just mask + components). Stops cleanly at the first un-ported
// present component (DesyncAt) and captures the held-weapon variant when reached.
// filmComponentCorruptionCheck : en mode FILM/replay (FUN_1404f2b4c()==2, actif quand on
// décode un film), FUN_14076cb60 lit APRÈS chaque composant présent un R(1) garde ; si le
// bit ==1, un R(32) sentinel (marqueur 0xbcddcba "entity component corrupt"). Le décodeur
// supposait ce mode OFF → il sautait tous ces bits (validé live CE : le record NEW biped
// fait 1924 bits ; sans ce check mon décodeur en consommait 1863 = -61). Découverte workflow
// biped-record-61bit-gap (agents FUN_1408f1aa4 + FUN_14076cb60 convergents).
var filmComponentCorruptionCheck = false

// useLegacyAngularVel routes i3 through the OLD (wrong) double-gated angular-velocity
// deser (FUN_140d87740, +R(96) over-read). Default false = the correct single-gate
// FUN_14076d528. Kept as a toggle for A/B against the biped-deltas-per-frame oracle.
var useLegacyAngularVel = false

// SetUseLegacyAngularVel toggles the legacy i3 angular-velocity deser.
func SetUseLegacyAngularVel(v bool) { useLegacyAngularVel = v }

// SetFilmComponentCorruptionCheck (dé)active le corruption-check per-composant du mode film.
func SetFilmComponentCorruptionCheck(v bool) { filmComponentCorruptionCheck = v }

// newRecordTailBits : bits terminaux consommés APRÈS la boucle de composants d'un record NEW
// (candidat : le tail de FUN_1408f1aa4 après FUN_14076cb60, non encore identifié bit-exact).
// Toggle de calibration keyframe (cmd/tmp_kfdecode) : l'agent Ghidra a mesuré que record0
// (ti=22, default-state 0 bit) doit finir 1 bit plus loin pour aligner record1 = NEW slot1.
// Défaut 0 = comportement historique (le path delta n'appelle pas TraverseEntity, non affecté).
var newRecordTailBits = 0

// SetNewRecordTailBits fixe le tail terminal per-record NEW (calibration keyframe).
func SetNewRecordTailBits(n int) { newRecordTailBits = n }

// defaultStateBitsByTI : largeur FIXE (bits) du default-state (vtable[0x60]) par typeIndex, pour
// les archétypes non-biped dont le default-state est un deser court à largeur constante (résolus
// statiquement DAT_144e61d88[ti]→vtable[0x60], agent Ghidra). Absent = stub 0-bit (0x1408d8220).
// Le biped (ti=35) est traité à part (consumeBipedDefaultState). Vide par défaut = comportement
// historique (fallback br.Skip(defaultStateBits)).
var defaultStateBitsByTI = map[uint32]int{}

// SetDefaultStateBitsForTI enregistre la largeur de default-state d'un archétype (calibration keyframe).
func SetDefaultStateBitsForTI(ti uint32, bits int) { defaultStateBitsByTI[ti] = bits }

// useArchDefaultStateDeser route les archétypes non-biped vers leur déserialiseur
// vtable[0x60] porté (default_state_arch.go) au lieu du Skip(0) historique. Défaut true ;
// le passer à false rend le comportement d'avant le portage pour un A/B mesuré.
var useArchDefaultStateDeser = true

// SetUseArchDefaultStateDeser (dé)active les desers de default-state par archétype.
func SetUseArchDefaultStateDeser(v bool) { useArchDefaultStateDeser = v }

// consumeCorruptionCheck lit le sentinel per-composant du mode film : R(1) garde ; si 1, R(32).
func consumeCorruptionCheck(br *BitReader) {
	if filmComponentCorruptionCheck && br.ReadBit() {
		br.ReadBits(32) // sentinel attendu 0xbcddcba
	}
}

// simStateComplete : si true, i60 (simulation-state) est déclaré ENTIÈREMENT décodé et la
// traversée continue vers i61-63. DÉFAUT false — et depuis le 2026-08-17 ce n'est PLUS pour
// la raison historique (« grammaire de la queue non résoluble offline ») : la grammaire est
// établie (consumeSimStateHandleTail, lot R7-b, décompile + prédicat prouvé vrai par
// construction). Ce qui manque est la SOURCE DES LARGEURS D'AXE de cette queue sur le chemin
// de production : `absAxisWFor` retombe sur `absoluteAxisW`, un UNIFORME 14 qui n'est la
// largeur d'aucune carte (Cliffhanger 13/13/14, Bazaar 17/17/16, Illusion 18/18/17 —
// mesurés par DetectI0Layout le 2026-08-17). Continuer la marche avec une queue mal
// dimensionnée propagerait un désalignement au lieu d'un désync propre.
//
// KILL-SWITCH — bascule du défaut à `true` conditionnée à UN critère mesurable : que le
// chemin absolu d'i0 tire ses trois largeurs de la carte du match (comme
// `replay.installWorldObjectPrecision` le fait déjà pour `WorldObjectPrecision`) au lieu de
// l'uniforme `absoluteAxisW`. Retrait cible du drapeau : à la bascule. Témoin de détection
// connu : `TestGoldenMiniBobine` (killsource) passe de 0 à 2 « source appartenant à la
// victime » PROPOSÉES, 0 publiée dans les deux cas — mesuré le 2026-08-17.
var simStateComplete = false

// SetSimStateComplete (dé)active le mode « i60 complet » (instruments de mesure ; défaut off).
func SetSimStateComplete(v bool) { simStateComplete = v }

// consumeSimStateHandleTail porte FUN_14076e494(br, dst, LEVEL=0x10, 0, 0, param_6=0) — la
// QUEUE d'i60, RÉSOLUE le 2026-08-17 (lot R7-b) après avoir été longtemps portée « largeur
// inconnue, désync propre ».
//
//	cVar1 = FUN_14076f91c()   garde RUNTIME (DAT_144e61ea0 / DAT_145121140), 0 bit
//	                          = PositionFullPrecision, déjà modélisée ici.
//	cVar1 != 0 : FUN_1411b259c -> FUN_1406d676c(br, br, dst, 0x60)   = R(96) brut.
//	cVar1 == 0 (retail, dominant) : FUN_14076e524(dst, br, idxOut, LEVEL=0x10) =
//	          R(1) porte d'index ; si 0 -> R(DAT_144632be0) index de région ;
//	          puis FUN_140cc5128 = 3 axes aux largeurs de la ligne LEVEL=16.
//
// C'est EXACTEMENT le lecteur absolu de `consumeAbsoluteWithGate`, MOINS son bit precHigh
// (ici la garde est runtime, pas un bit du flux) et MOINS son R(2) « fini » de queue — que
// FUN_14076e494 n'appelle pas.
func consumeSimStateHandleTail(br *BitReader) {
	if fullPrecisionGate() { // FUN_14076f91c vrai -> copie brute
		br.ReadBits(rawVec3Bits) // FUN_1406d676c(..., 0x60)
		return
	}
	idx := -1
	if !br.ReadBit() { // FUN_14076e524 : porte d'index ; 0 -> lit l'index de région
		idx = int(br.ReadBits(WorldObjectPrecision.IndexW))
	}
	for i := 0; i < 3; i++ {
		br.ReadBits(absAxisWFor(idx, i)) // FUN_140cc5128 axe i
	}
}

// consume140c1e79c porte FUN_140c1e79c (direction+magnitude d'i60) :
//
//	R(1) gate ; si bit==0 -> R(19) packed dir (FUN_1406d8288 dequant, 0 bit)
//	PUIS TOUJOURS R(8) magnitude (FUN_1406d84b4 width=8 @140c1e80f ; FUN_1406d8678 dequant).
//
// CONFIRMÉ via disasm : le bloc LAB_140c1e7f2 charge `MOV dword [RSP+0x20],0x8` avant
// CALL 1406d84b4. L'ancien port oubliait cette magnitude R(8) (lumpée dans simStateExtra).
func consume140c1e79c(br *BitReader) {
	if !br.ReadBit() { // gate==0 -> packed dir
		br.ReadBits(19)
	}
	br.ReadBits(8) // magnitude R(8)
}

// consumeSimulationState porte i60 (FUN_142ED6D88, vérifié via décompile) :
//
//	R(1) flag (FUN_1406cf008) ; si 0 -> FUN_14058c250 (0 bit).
//	si 1 : 2×FUN_1407f2058 (R(1)[R5]) + 4×FUN_142ee2194 (R16) + 2×R(2) inline
//	       + 4×FUN_142ee2194 (R16) + FUN_140c1e79c (R1[R19]+R8)
//	       + queue : FUN_140501798 predicate (0 bit) puis FUN_14076e494.
//
// LE PREDICAT EST VRAI PAR CONSTRUCTION (établi le 2026-08-17, lot R7-b), et c'est ce qui
// débloque la queue. `FUN_140c1e79c(br, ?, out=RBP+0x2c, dir=RBP+0x38)` (disasm @142ed6f9b)
// décode une DIRECTION unitaire en `+0x38` puis appelle `FUN_1406d8678(dir, angle, out)`,
// qui construit en `+0x2c` un vecteur PERPENDICULAIRE à la direction (produit vectoriel avec
// un vecteur de base, normalisé, puis rotation de Rodrigues autour de `dir` par l'angle R(8),
// et normalisation finale FUN_1404fec88). `FUN_140501798(+0x2c, +0x38)` teste ensuite
// ‖v1‖²≈1, ‖v2‖²≈1 et v1·v2≈0 — constantes lues dans le binaire : DAT_143cd8374 = 1.0,
// DAT_143cd8370 = 0.0, tolérance DAT_143cd84bc = 1e-3. Une base orthonormée construite
// satisfait les trois : la queue est donc LUE, elle n'est pas conditionnelle en pratique.
func consumeSimulationState(br *BitReader) {
	if !br.ReadBit() { // FUN_1406cf008 flag ; 0 -> 0 bit
		return
	}
	consumeGate0R(br, 5) // FUN_1407f2058 #1
	consumeGate0R(br, 5) // FUN_1407f2058 #2
	for i := 0; i < 4; i++ {
		br.ReadBits(16) // FUN_142ee2194 = R(16)
	}
	br.ReadBits(2) // inline R(2) #1
	br.ReadBits(2) // inline R(2) #2
	for i := 0; i < 4; i++ {
		br.ReadBits(16) // FUN_142ee2194 = R(16)
	}
	consume140c1e79c(br)          // FUN_140c1e79c = R(1)[R19]+R8
	consumeSimStateHandleTail(br) // FUN_14076e494, predicat vrai par construction
}

func traverseComponentLoop(br *BitReader, arch Archetype, t *EntityTrace) {
	traverseComponentLoopFrom(br, arch, t, 0)
}

// traverseComponentLoopFrom walks the component loop starting at index `from` —
// the resume path of component-width inference (frame_chain_infer.go), which skips
// a failed component by a candidate width and re-decodes the remainder.
func traverseComponentLoopFrom(br *BitReader, arch Archetype, t *EntityTrace, from int) {
	for i := from; i < len(arch.Components); i++ {
		if t.Mask&(uint64(1)<<(uint(i)&63)) == 0 {
			continue // component absent from the mask: NO bits consumed.
		}
		start := br.BitPos()
		// CALIBRATION OVERRIDE: runtime-precision components (i0 position, i21 aiming,
		// ...) read per-axis widths populated at map-load and absent from the static
		// .exe, so their ported desers consume the wrong bit count in the delta
		// predicted-precision path. When a CE delta capture has measured the real
		// per-component width (CONSTANT per map), skip by it instead of calling the
		// buggy deser. The dead-state is intentionally NOT calibrated (must be decoded).
		// i5 object-shield-vitality était cité ici À TORT (corrigé le 2026-07-26) :
		// FUN_140d50cbc n'utilise que des largeurs littérales 8/16/12 et des bornes
		// .rdata constantes — il ne dépend d'aucune précision runtime.
		if w, ok := calibratedWidth[arch.Components[i]]; ok {
			br.Skip(w)
			consumeCorruptionCheck(br)
			t.Comps = append(t.Comps, CompResult{Index: i, Name: arch.Components[i], Ported: true, StartBit: start})
			continue
		}
		variant, dead, payload, ported := consumeByNameCapturing(br, arch.Components[i], t.TypeIndex, arch.Level(i))
		if dead != nil {
			t.Dead = dead
		}
		if !ported {
			// CALIBRATION HOOK: an un-ported component normally desyncs (we can't trust
			// the bit position past a component whose width we don't know). If a probe has
			// set a stub width for this component name, consume that many bits and keep
			// going — lets a harness brute-force a missing tail deser's width by record-
			// chaining (the only un-ported component on the delta-biped path is i63
			// biped-action-component, the LAST component, AFTER the weapon). Default: off.
			if w, ok := unportedStubWidth[arch.Components[i]]; ok {
				br.Skip(w)
				consumeCorruptionCheck(br)
				t.Comps = append(t.Comps, CompResult{Index: i, Name: arch.Components[i], Variant: variant, Ported: true, StartBit: start})
				continue
			}
			t.Comps = append(t.Comps, CompResult{Index: i, Name: arch.Components[i], Variant: variant, Ported: false, StartBit: start})
			t.DesyncAt = i
			return
		}
		consumeCorruptionCheck(br)
		t.Comps = append(t.Comps, CompResult{Index: i, Name: arch.Components[i], Variant: variant,
			Ported: ported, StartBit: start, Payload: payload})
	}
}

// calibratedWidth overrides a component's deser with a fixed bit-skip, for the
// runtime-precision components whose per-axis widths are map-load runtime values
// absent from the static .exe (i0 object-position, i21 unit-desired-aiming-vector,
// ...). i5 object-shield-vitality figurait ici À TORT : ses largeurs sont des
// littéraux 8/16/12 dans FUN_140d50cbc (corrigé le 2026-07-26, cf.
// consumeObjectShieldVitality). Widths come from a Cheat Engine delta
// capture (tools/ce/filmdec_delta_capture.lua, processed by cmd/tmp_deltacal) and
// are CONSTANT per map. This is a CALIBRATION harness keyed to a specific film/map,
// NOT a general decode path. Empty by default. The dead-state component must NOT be
// added here (it is decoded to read killer-absolute-participant-index).
var calibratedWidth = map[string]int{}

// SetCalibratedWidth sets (w>=0) or clears (w<0) the calibrated skip width for a
// component name. Takes precedence over consumeByName in traverseComponentLoop.
func SetCalibratedWidth(name string, w int) {
	if w < 0 {
		delete(calibratedWidth, name)
		return
	}
	calibratedWidth[name] = w
}

// unportedStubWidth lets a calibration harness assign a provisional bit width to a
// component whose deser is not yet ported, so the traversal can continue past it
// (used to brute-force a missing tail deser's width via record-chaining). Keyed by
// component name; empty by default (un-ported components desync). NOT a decode path
// for production — only the probe sets it (SetUnportedStubWidth).
var unportedStubWidth = map[string]int{}

// SetUnportedStubWidth sets (width>=0) or clears (width<0) the provisional stub width
// for an un-ported component name. Returns nothing; affects subsequent traversals.
func SetUnportedStubWidth(name string, width int) {
	if width < 0 {
		delete(unportedStubWidth, name)
		return
	}
	unportedStubWidth[name] = width
}

// consumeMask mirrors FUN_1406d7610: R(1) gate ; if 0 -> R(3) count + count×R(6)
// index (sparse set) ; if 1 -> R(64) dense mask.
func consumeMask(br *BitReader) uint64 {
	if !br.ReadBit() {
		count := uint32(br.ReadBits(3))
		var mask uint64
		for i := uint32(0); i < count; i++ {
			idx := uint(br.ReadBits(6))
			mask |= uint64(1) << (idx & 63)
		}
		return mask
	}
	return br.ReadBits(64)
}
