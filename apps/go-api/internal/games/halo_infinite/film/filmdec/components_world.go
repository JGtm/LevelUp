package filmdec

// World-object / crew archetype component deserializers (ti=14 crew, ti=6 statborg,
// ti=20 spawn-filter, etc.). Ported to advance the FRAME walk past the world entities
// so their slots bind (the World can't reach the bipeds otherwise). Gradient-driven.

// ---- Batch ECS world/object component desers (RE'd via workflow port-ecs-chunked, EXE-verified) ----
//
// Retrait 2026-08-01 (lot C, PLAN_DETTE_AVANT_MERGE) de six ports orphelins de ce fichier
// (`effect-state-data`, `music-state`, `tacmap-poiicon`, `crew-marked-objects`, `crew-order`,
// `state-broker`). Les cinq premiers sont SUPPLANTÉS par le port INLINE de traverse.go, qui
// est le port CORRIGÉ : il lit les vec3 en quantification 6+level (flags du registre) là où
// ceux d'ici employaient les largeurs fixes de TraversalPrecision, et il DÉSYNCHRONISE
// proprement sur les branches de largeur runtime au lieu de deviner des bits (décision de
// méthode inscrite en tête du bloc `crew-order` de traverse.go). Les rebrancher, ce serait
// réintroduire l'approche écartée. `state-broker` visait ti=46, ni décodé ni planifié.

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
// C'est le lecteur des composants i0..i31 de ti=23 — l'archétype ZONES, couvert 0/33 par le
// dispatch : ce seul lecteur, branché, le porterait à 32/33.
//
// GARDÉE SANS APPELANT le 2026-08-01 (lot C, PLAN_DETTE_AVANT_MERGE) : ti=23 est PLANIFIÉ
// mais non décodé, et absent du World des trois films de référence (0 slot) — la brancher ne
// peut rien changer aujourd'hui, et le gate du lot exige des artefacts identiques.
// CONDITION DE RETRAIT : branchée ou supprimée quand ti=23 sera décodé (master plan J6-A §4 —
// l'état des zones par images-clés + footer type-3).
//
//nolint:unused // grammaire d'un composant de ti=23 — voir la condition de retrait ci-dessus.
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

// Les ports `crew-marked-objects-component` (ti=14 i1) et `crew-order-component` (ti=14 i0)
// vivaient ici ; retirés le 2026-08-01 (voir la note de retrait en tête de fichier) — le
// dispatch de traverse.go les porte, avec la quantification 6+level et la désynchronisation
// propre sur la branche de largeur runtime.
