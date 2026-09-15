package filmdec

// dispatch_biped.go — SIXIEME ET SEPTIEME MAILLONS : les composants captes, le bipede, et la
// fin de la chaine.
//
// Deplacement pur depuis `traverse.go` au lot 2.7 ; chaine et exemption de longueur
// documentees en tete de `dispatch_object.go`.

// consumeCaptureAndBipedComponent porte les QUATRE composants dont la valeur est CAPTEE et
// remontee a l'appelant (obje, arme tenue, vitalites de corps et de bouclier, etat de mort —
// cf. `capture.go`), puis les composants d'ARME, de BIPEDE (i47 a i63) et d'ETAT DE
// SIMULATION. C'est le seul maillon qui rend un `variant` non nul et un `dead` non nil.
func consumeCaptureAndBipedComponent(br *BitReader, name string, typeIndex uint32, level uint32) (variant uint32, dead *DeadState, ported bool) {
	variant = noVariant
	switch name {
	case compObjectMultiplayerProperties: // i9 = the 'obje' (FUN_1407d4c94 TLV blob)
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
	case compObjectShieldVitality:
		consumeObjectShieldVitality(br)
		return variant, nil, true
	case deadStateComponentName:
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
	case compBipedAbilitySetAlt, compBipedAbilitySet: // i48 (FUN_1406d0ff0)
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
	case abilityEnergyNameAlt, abilityEnergyName: // i56 (FUN_140fc1410)
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
	case abilityPredictedName: // i57 (FUN_142f02810 -> FUN_142f268c4)
		// Rend ported=false sur la seule branche non determinable (tag brut == 3 ->
		// FUN_142f262d4, gate par des octets d'etat runtime) : desync propre plutot que
		// desalignement silencieux.
		return variant, nil, consumeBipedSpartanAbility(br)
	case grappleComponentNameAlt, grappleComponentName: // i59 (FUN_142f02994)
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
	default:
		return consumeManagedAndObjectiveComponent(br, name, level)
	}
}

// consumeManagedAndObjectiveComponent est le DERNIER maillon : objet gere (ti=10/12/13),
// objectif (ti=11), unite (ti=35 hors bipede) et les arms isoles. Sa branche `default` est
// celle du `switch` d'origine, mot pour mot : un composant qu'aucun maillon ne reconnait n'est
// pas porte, et la traversee s'arrete proprement sur lui (DesyncAt).
//
// Il ne prend PAS `typeIndex` : aucun de ses arms ne le lit, et il n'a plus de maillon a qui
// le passer.
func consumeManagedAndObjectiveComponent(br *BitReader, name string, level uint32) (variant uint32, dead *DeadState, ported bool) {
	variant = noVariant
	switch name {
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
	// ti=11 — l'archétype des objectifs gérés (components_managed_objective.go). Toutes les
	// largeurs viennent du sérialiseur réseau du descripteur de composant (`+0x38`), recette R7-d.
	// SEUL i4 `interaction-filter` reste dehors : sa queue est un appel virtuel de largeur
	// inconnue, et le porter à moitié désynchroniserait au lieu d'arrêter proprement.
	case compObjectiveTimers: // ti=11 i0 (FUN_142edbac8) — 2 x R(7), publie
		consumeObjectiveTimers(br)
		return variant, nil, true
	case compObjectiveColor: // ti=11 i1 (FUN_142edb548) — 4 x R(8)
		consumeObjectiveColor(br)
		return variant, nil, true
	case compObjectiveFormattedText, compObjectiveSecondaryFormattedText: // ti=11 i2 et i9
		consumeObjectiveFormattedText(br)
		return variant, nil, true
	case compObjectiveObjectReference: // ti=11 i3 (FUN_142edb6a4) — R(32), publie
		consumeObjectiveObjectReference(br)
		return variant, nil, true
	case compObjectiveType: // ti=11 i5 (FUN_142edbb00) — R(32), publie
		consumeObjectiveType(br)
		return variant, nil, true
	case compObjectiveEnabled, compObjectiveIsNewAndUnseen,
		compObjectiveIsOnlyOneItemUnlocked, compObjectiveForcedUpdate: // ti=11 i6/i10/i11/i33 — R(1)
		consumeObjectiveBool(br)
		return variant, nil, true
	case compObjectivePriority: // ti=11 i7 (FUN_142edb820) — R(8)
		consumeObjectivePriority(br)
		return variant, nil, true
	case compObjectiveMessageType: // ti=11 i8 (FUN_142edb604) — R(4)
		consumeObjectiveMessageType(br)
		return variant, nil, true
	case compObjectiveProgress: // ti=11 i12 (FUN_142edb8c0) — R(32) LA JAUGE, publie
		consumeObjectiveProgress(br)
		return variant, nil, true
	case compObjectiveRequiredProgress: // ti=11 i13 (FUN_142edb960) — R(32) LE SEUIL, publie
		consumeObjectiveRequiredProgress(br)
		return variant, nil, true
	case compObjectiveState: // ti=11 i14 (FUN_142edba10) — R(3), publie
		consumeObjectiveState(br)
		return variant, nil, true
	case compObjectiveParentObjective, compObjectiveSubObjectiveEntities: // ti=11 i15 et i16..i31 — R(32)
		consumeObjectiveEntityRef(br)
		return variant, nil, true
	case compObjectiveOutroPhaseDuration: // ti=11 i32 (FUN_142edb740) — R(8) quantifié
		consumeObjectiveOutroPhaseDuration(br)
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
	case "game-engine-team-mapping", compGameEngineTeamMapping: // typeIdx=0 i0 (FUN_140f58200)
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
		// L = niveau de precision du composant dans chunk_00, la largeur venant de la table
		// DAT_1445cc9e0 indexee par le niveau (largeur = 6+L, verifie sur le dump
		// ce_prec_widths_1445cc9e0.bin).
		//
		// ti44 i0 EST A L1, PAS A L0 (lot 1.2, 2026-09-14) : cette ligne disait « L0 -> 6 bits
		// par axe » parce que le registre se lisait un cran trop tot et servait le niveau du
		// composant PRECEDENT. Sous le cadrage du jeu le niveau est 1, donc 7 bits par axe, et
		// le budget passe de 5 x (1+1+3x6) = 100 bits a 5 x (1+1+3x7) = 115.
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
