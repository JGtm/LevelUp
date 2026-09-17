package grammar

// dispatch_item.go — DEUXIEME MAILLON : equipement, objet pose, projectile, tacmap.
//
// Deplacement pur depuis `traverse.go` au lot 2.7 ; chaine et exemption de longueur
// documentees en tete de `dispatch_object.go`.

// consumeItemAndTacmapComponent porte les composants d'EQUIPEMENT et d'OBJET POSE (ti=37,
// ti=42), de PROJECTILE (ti=41) et de TACMAP (ti=32/33/34), plus les arms de scene et de
// corps rigide portes dans les memes lots.
func consumeItemAndTacmapComponent(br *Lecteur, name string, typeIndex uint32, level uint32) (variant uint32, dead *DeadState, ported bool) {
	variant = noVariant
	switch name {
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
	case "equipment-control-signal-component": // ti=37 i22 (FUN_14101cd94)
		// GRAMMAIRE RELUE le 2026-09-15 (lot 1.9.1 bis, pas 2 bis) : le deser est
		// `FUN_14101d200` (R(4), 14101d21d) puis `FUN_1408f0ac4(dst+0x58c, br, 4)`
		// (14101cdc3). La CATEGORIE est 4 : pas de bit de sonde, et 9 bits de valeur —
		// le portage precedent lisait `readQuantStat(1, 13)`, soit une sonde de trop ET
		// quatre bits de valeur de trop, cinq bits sur chaque record qui ouvre la porte.
		br.ReadBits(4)
		consume1408f0ac4(br, 4)
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
	case compWeaponAmmo: // ti=42 i20 (FUN_140fc3028) — R(8)+R(11)+R(12)
		consumeWeaponAmmo(br)
		return variant, nil, true
	case "branch-script-results-component": // ti=16 i1 (FUN_142ed3dcc) — R(6)+R(32)
		br.ReadBits(6)
		br.ReadBits(32)
		return variant, nil, true
	case compHighFrequency: // ti=4 i0 — variante FRAME (FUN_14076d034) = R(8), sonde
		br.obs.publishProbe(typeIndex, ProbeHighFrequency, br.ReadBits(8))
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
	default:
		return consumePlayerAndSceneComponent(br, name, typeIndex, level)
	}
}
