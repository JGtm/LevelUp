package filmdec

// Batch-4 ECS component desers (shared per-frame entities: game-engine/spawn-filter/
// weapon/objective/physics). RE'd via workflow port-ecs-batch4 (Ghidra), confident=true.

// game-engine-alliance-component (FUN_140a24968): 32-bit mask + per set bit 2x R(32).
func consumeGameEngineAlliance(br *BitReader) {
	var mask uint32
	for i := uint(0); i < 32; i++ {
		if br.ReadBit() {
			mask |= 1 << i
		}
	}
	for i := uint(0); i < 32; i++ {
		if (mask>>i)&1 != 0 {
			br.ReadBits(32)
			br.ReadBits(32)
		}
	}
}

// statborg-round-outcomes-component (FUN_142ed71a4): 32x R(2).
func consumeStatborgRoundOutcomes(br *BitReader) {
	for i := 0; i < 32; i++ {
		br.ReadBits(2)
	}
}

// supply-lines-busy-state-component: R(5)+1 = count, then count x R(1).
func consumeSupplyLinesBusy(br *BitReader) {
	count := uint(br.ReadBits(5)) + 1
	for i := uint(0); i < count; i++ {
		br.ReadBit()
	}
}

// low-frequency (FUN-low-frequency rich variant): position body + rotation + count loop.
func consumeLowFrequency(br *BitReader) {
	consumeE524PositionBody(br)
	if !br.ReadBit() { // rotation default flag
		br.ReadBits(19)
	}
	br.ReadBits(8)
	br.ReadBits(16)
	br.ReadBits(8)
	br.ReadBits(2)
	count := br.ReadBits(6)
	for i := uint64(0); i < count; i++ {
		flags := br.ReadBits(3)
		if flags&1 != 0 {
			consumeE524PositionBody(br)
		}
		if flags&2 != 0 {
			if !br.ReadBit() {
				br.ReadBits(19)
			}
			br.ReadBits(8)
		}
		br.ReadBits(16)
		br.ReadBits(5)
	}
}
