package filmdec

// World-object / crew archetype component deserializers (ti=14 crew, ti=6 statborg,
// ti=20 spawn-filter, etc.). Ported to advance the FRAME walk past the world entities
// so their slots bind (the World can't reach the bipeds otherwise). Gradient-driven.

// ---- Batch ECS world/object component desers (RE'd via workflow port-ecs-chunked, EXE-verified) ----

// effect-state-data-component (ti=18 x32, FUN_142ed438c).
func consumeEffectStateData(br *BitReader) {
	if br.ReadBits(1) != 0 {
		br.ReadBits(32)
	}
	consume1408f0ac4(br)
	br.ReadBits(32)
	if br.ReadBits(1) != 0 {
		consume1408f0ac4(br)
		br.ReadBits(32)
	}
}

// change-scene-component (FUN_142ed3fcc): R(6) + R(12)=N + N-bit blob (N self-describing).
func consumeChangeScene(br *BitReader) {
	br.ReadBits(6)
	n := uint(br.ReadBits(12))
	for rem := n; rem > 0; {
		take := rem
		if take > 32 {
			take = 32
		}
		br.ReadBits(take)
		rem -= take
	}
}

// spawn-filter-type-component (FUN_142ed708c -> FUN_142ecf744): 2-bit tag dispatch.
// version-gated tag1 (confident=false) modeled as v<2 handle path.
func consumeSpawnFilterType(br *BitReader) {
	switch br.ReadBits(2) {
	case 0:
		// none
	case 1:
		if br.ReadBits(1) == 0 {
			br.ReadBits(5)
		}
	case 2:
		br.readQuantStat(1, 13)
		br.ReadBits(6)
	default:
		br.ReadBits(32)
		consumeE524PositionBody(br)
		br.ReadBits(3)
		count := int(br.ReadBits(4))
		for i := 0; i < count; i++ {
			br.ReadBits(32)
		}
		if br.ReadBits(1) == 0 {
			br.ReadBits(5)
		}
	}
}

// statborg-current-round-value-stat-component (FUN_140c18794): 2x R(5) header +
// 2x var-width signed + 2x R(1) flag + conditional var-width.
func consumeStatborgValueStat(br *BitReader) {
	br.ReadBits(5)
	br.ReadBits(5)
	br.ReadSignedVarWidth()
	br.ReadSignedVarWidth()
	flagA := br.ReadBit()
	flagB := br.ReadBit()
	if flagA {
		br.ReadSignedVarWidth()
	}
	if flagB {
		br.ReadSignedVarWidth()
	}
}

// music-state-component (FUN_142ed5ef0).
func consumeMusicState(br *BitReader) {
	br.ReadBits(1)
	br.ReadBits(32)
	br.ReadBits(32) // tagA id
	br.ReadBits(1)  // tagA present
	br.ReadBits(32) // tagB id
	br.ReadBits(1)  // tagB present
	if br.ReadBits(1) != 0 {
		br.ReadBits(1)
		br.ReadBits(16)
		br.ReadBits(16)
	}
	if br.ReadBits(1) != 0 {
		for i := 0; i < 6; i++ {
			br.ReadBits(16)
		}
	}
}

// tacmap-poiicon (FUN_142ed8418).
func consumeTacmapPoiicon(br *BitReader) {
	br.ReadBits(32)
	br.ReadBits(32)
	br.ReadBits(1)
	br.ReadBits(3)
	br.ReadBits(32)
	br.ReadBits(32)
	br.ReadBits(9)
	br.ReadBits(9)
	consumeE524PositionBody(br)
	br.ReadBits(32)
	br.ReadBits(1)
	br.ReadBits(8)
	br.ReadBits(8)
	br.ReadBits(8)
	br.ReadBits(8)
}

// tacmap-areaofinterest (FUN_142ed3c50): R(32)+R(3)+pos+R(12).
func consumeTacmapAreaOfInterest(br *BitReader) {
	br.ReadBits(32)
	br.ReadBits(3)
	consumeE524PositionBody(br)
	br.ReadBits(12)
}

// tacmap-displayasset (FUN_142ed433c): R(32)+R(32)+R(2)+pos+R(96)+R(96)+R(1).
func consumeTacmapDisplayAsset(br *BitReader) {
	br.ReadBits(32)
	br.ReadBits(32)
	br.ReadBits(2)
	consumeE524PositionBody(br)
	br.ReadBits(64)
	br.ReadBits(32)
	br.ReadBits(64)
	br.ReadBits(32)
	br.ReadBits(1)
}

// selectable-zone-data-component (FUN_142ed6cec): R(32)+pos+R(1)[+R(5)].
func consumeSelectableZoneData(br *BitReader) {
	br.ReadBits(32)
	consumeE524PositionBody(br)
	if br.ReadBits(1) == 0 {
		br.ReadBits(5)
	}
}

// managed-object-participant-respawn-block-component (FUN_142ed6a20): R(32)+R(1)[+R(5)].
func consumeRespawnBlock(br *BitReader) {
	br.ReadBits(32)
	if br.ReadBits(1) == 0 {
		br.ReadBits(5)
	}
}

// item-ignore-player-component (FUN_141101120): R(1)[+R(5)].
func consumeItemIgnorePlayer(br *BitReader) {
	if br.ReadBits(1) == 0 {
		br.ReadBits(5)
	}
}

// state-broker-state-changed-data-component (FUN_142c346d8): R(32)+R(2)+switch.
func consumeStateBroker(br *BitReader) {
	br.ReadBits(32)
	switch br.ReadBits(2) {
	case 0:
		br.ReadBits(1)
	case 1:
		br.ReadBits(16)
	case 2:
		br.ReadBits(32)
	default:
		br.ReadBits(32)
	}
}

// consumeGameEngineSharedTeamLives mirrors FUN_142f03600 (ti=0 i1): 8x R(8) = 64 bits
// (one lives byte per team, unconditional).
func consumeGameEngineSharedTeamLives(br *BitReader) {
	for i := 0; i < 8; i++ {
		br.ReadBits(8)
	}
}

// consumeE524PositionBody mirrors the FUN_14076e524 absolute-position read WITHOUT a
// leading gate (the caller supplies its own gate): R(1) index-select; if 0 -> R(idxW)
// index; then 3 axes of R(axisW). Widths from TraversalPrecision (runtime, derivable).
func consumeE524PositionBody(br *BitReader) {
	if !br.ReadBit() { // FUN_14076e524 index-present select
		br.ReadBits(TraversalPrecision.IndexW)
	}
	for i := 0; i < 3; i++ {
		br.ReadBits(TraversalPrecision.AxisW[i]) // FUN_140cc5128 axis i
	}
}

// consumeCompressedDir140c1e79c mirrors FUN_140c1e79c: R(1) sign; if 0 -> R(19) packed
// magnitude (FUN_1406d8288); then R(8) scale (FUN_1406d84b4 width 0x8 @140c1e80f).
func consumeCompressedDir140c1e79c(br *BitReader) {
	if !br.ReadBit() { // sign==0 -> packed magnitude present
		br.ReadBits(19) // FUN_1406d8288 (0x13)
	}
	br.ReadBits(8) // FUN_1406d84b4 width 8
}

// consumeGenericRigidBodyTransforms mirrors FUN_142f036f0 (generic-rigid-body-transforms,
// i18 of item/object archetypes ti=36/38/39…):
//
//	mask = R(8) ; for each set bit (0..7): FUN_140c1e79c (compressed dir) +
//	FUN_1404fdcb4 (local float, 0 bits) + FUN_14076e494 (e524 absolute position).
func consumeGenericRigidBodyTransforms(br *BitReader) {
	mask := br.ReadBits(8)
	for i := uint(0); i < 8; i++ {
		if mask&(1<<i) != 0 {
			consumeCompressedDir140c1e79c(br)
			consumeE524PositionBody(br)
		}
	}
}

// consumeCrewMarkedObjects mirrors FUN_142ed421c (crew-marked-objects-component, ti=14 i1):
//
//	R(1) gate ; if SET -> FUN_1406d3140 (quant int) + FUN_140809d20 (handle resolver, 0 bits).
func consumeCrewMarkedObjects(br *BitReader) {
	if br.ReadBit() { // gate
		br.readQuantStat(1, quantStatDefaultWidth) // FUN_1406d3140 (probe + 13 + 2)
	}
}

// consumeCrewOrder mirrors FUN_142ed4274 -> FUN_142ed9120 (crew-order-component, ti=14 i0):
//
//	FUN_142b1cf3c = R(3)
//	R(1) gate (FUN_1406cf008) ; if SET -> FUN_14076e494 = e524 absolute position.
func consumeCrewOrder(br *BitReader) {
	br.ReadBits(3)    // FUN_142b1cf3c
	if br.ReadBit() { // gate; if set -> position
		consumeE524PositionBody(br)
	}
}
