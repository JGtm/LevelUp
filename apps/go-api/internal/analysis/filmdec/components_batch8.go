package filmdec

// Batch8 FRAME component desers — RE'd via workflow filmdec-port-top-components (2026-06-14,
// ghidra-verified, top desync-count components of the clean-frame histogram). Goal: raise the
// per-frame clean rate so the World binding completes (the death chains block on unbound
// world-object slots, a symptom of the 63% gradient). All widths LITERAL unless noted.
//
// IMPORTANT recipe nuance discovered here: for NAME-INDEXED descriptor tables (equipment-*,
// biped-emp-timer), the read/write codec is at descriptor+0x40, NOT +0x28 (+0x28 is a
// 'return 1' predicate). The +0x28 recipe only holds for the component-replication descriptors.

// crew-orders-off-flags-component (ti=14, FUN_142ed9e80 -> FUN_142c74154): one byte of flags.
func consumeCrewOrdersOffFlags(br *BitReader) {
	br.ReadBits(8)
}

// music-variables-component (FUN_142ed66bc): 32 fixed slots, each a presence bit (0 => present)
// then R(2)+R(32)+R(32). All literal widths.
func consumeMusicVariables(br *BitReader) {
	for i := 0; i < 32; i++ {
		if !br.ReadBit() { // flag==0 => present
			br.ReadBits(2)
			br.ReadBits(32)
			br.ReadBits(32)
		}
	}
}

// sound-placement-state-data-component (FUN_142ed700c): optional R(20) (gate R(1), present when
// bit==0) then a trailing R(1) flag. Widths literal (0x14, 1).
func consumeSoundPlacementStateData(br *BitReader) {
	if !br.ReadBit() { // present
		br.ReadBits(20)
	}
	br.ReadBit()
}

// equipment-command-tick-component: déjà porté dans components_batch3.go
// (consumeEquipmentCommandTick) ; la RE batch8 confirme la même grammaire (eq bit + champs
// optU8). Seul le câblage manquait.

// equipment-has-infinite-uses-component (codec @+0x40 = FUN_142eda3f8): a single boolean bit.
func consumeEquipmentHasInfiniteUses(br *BitReader) {
	br.ReadBit()
}

// biped-emp-timer-component (codec @+0x40 = FUN_142f05250 -> FUN_1420352a0): quantized float
// packed in exactly 8 bits (0x100 buckets). Dequant range is a compile-time constant that does
// NOT affect the bit count.
func consumeBipedEmpTimer(br *BitReader) {
	br.ReadBits(8)
}

// game-engine-current-state-component (ti=0, FUN_14116d1d0): single inlined R(3) (state enum).
func consumeGameEngineCurrentState(br *BitReader) {
	br.ReadBits(3)
}

// game-engine-game-finished-component (ti=0, FUN_142f035e0): single R(1) boolean flag.
func consumeGameEngineGameFinished(br *BitReader) {
	br.ReadBit()
}

// statborg-entry-index-and-type-component (FUN_1410be614): R(32) entry-index then R(8) type.
func consumeStatborgEntryIndexAndType(br *BitReader) {
	br.ReadBits(32)
	br.ReadBits(8)
}

// spawn-filter-weight-component (FUN_142ed70b8 -> FUN_1406d84b4 width 0x10): quantized float,
// 16 bits consumed (dequant bounds are globals that don't change the bit count).
func consumeSpawnFilterWeight(br *BitReader) {
	br.ReadBits(16)
}

// state-broker-state-changed-data-component (FUN_142c346d8 -> FUN_142c357d8): R(32) key, R(2)
// type tag, then a tag-dictated value (all literal widths, tag from stream).
func consumeStateBrokerStateChangedData(br *BitReader) {
	br.ReadBits(32) // key (string-id)
	switch br.ReadBits(2) {
	case 0:
		br.ReadBit() // bool
	case 1:
		br.ReadBits(16)
	case 2:
		br.ReadBits(32)
	default: // 3
		br.ReadBits(32) // stringid-value
	}
}

// forge-player-data-edited-objects-ids-component (FUN_142f0308c): a leading entry, then an
// 8-bit count of additional entries. Each entry = R(2) tag then {0|16|13} bits by tag.
func consumeForgePlayerDataEditedObjectsIDs(br *BitReader) {
	forgeEditedEntry(br)
	count := br.ReadBits(8)
	for i := uint64(0); i < count; i++ {
		forgeEditedEntry(br)
	}
}

// forgeEditedEntry mirrors FUN_142a6e8c4: R(2) tag (raw-1) then tag2 -> R(16), tag3 -> R(13).
func forgeEditedEntry(br *BitReader) {
	switch br.ReadBits(2) {
	case 2:
		br.ReadBits(16)
	case 3:
		br.ReadBits(13)
	}
}
