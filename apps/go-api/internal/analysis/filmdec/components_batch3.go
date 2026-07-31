package filmdec

// Batch-3 ECS component desers (death-slot archetype chains ti=12/37/11/27/30/0/6).
// RE'd via workflow port-ecs-deathchains (Ghidra), confident=true only; the polymorphic/
// version-gated navpoint-filter components are left unported (they desync only if present,
// rare for the death-frame slots).

// managed-navpoint-formatted-text-component.
func consumeNavpointFormattedText(br *BitReader) {
	count := br.ReadBits(8)
	for i := uint64(0); i < count; i++ {
		br.ReadBits(32)
		if br.ReadBit() {
			br.ReadBits(32)
			subCount := br.ReadBits(3)
			for j := uint64(0); j < subCount; j++ {
				tag := br.ReadBits(3)
				switch tag {
				case 0:
				case 1:
					if !br.ReadBit() {
						br.ReadBits(5)
					}
				case 2:
					if br.ReadBit() {
						br.ReadBits(24)
					} else {
						br.ReadBits(32)
					}
				default:
					br.ReadBits(32)
				}
			}
		}
	}
}

// objective formatted-text + secondary share the same shape (R(1) presence + value + tagged list).
func consumeObjectiveFormattedText(br *BitReader) {
	if !br.ReadBit() {
		return
	}
	br.ReadBits(32)
	count := br.ReadBits(3)
	for i := uint64(0); i < count; i++ {
		tag := br.ReadBits(3)
		switch tag {
		case 0:
		case 1:
			if !br.ReadBit() {
				br.ReadBits(5)
			}
		case 2:
			if !br.ReadBit() {
				br.ReadBits(32)
			} else {
				br.ReadBits(24)
			}
		default:
			br.ReadBits(32)
		}
	}
}

// equipment-activated-component.
func consumeEquipmentActivated(br *BitReader) {
	if !br.ReadBit() {
		br.ReadBits(3)
	} else {
		consume1408f0ac4(br)
	}
}

// equipment-tracked-object-handles-stack-component: R(4) count + count x consume1408f0ac4.
func consumeEquipmentTrackedStack(br *BitReader) {
	count := br.ReadBits(4)
	for i := uint64(0); i < count; i++ {
		consume1408f0ac4(br)
	}
}

// equipment-command-tick-component: R(1) flag; if 0: 2x optU8; else: 1x optU8.
func consumeEquipmentCommandTick(br *BitReader) {
	if !br.ReadBit() {
		if br.ReadBit() {
			br.ReadBits(8)
		}
		if br.ReadBit() {
			br.ReadBits(8)
		}
	} else {
		if br.ReadBit() {
			br.ReadBits(8)
		}
	}
}

// statborg-finalized-rounds-values-stat-component: R(32) mask + per set bit 2x{R(1)[if0:varwidth]}.
func consumeStatborgFinalized(br *BitReader) {
	mask := uint32(br.ReadBits(32))
	for i := uint(0); i < 32; i++ {
		if (mask>>i)&1 != 0 {
			if !br.ReadBit() {
				br.ReadSignedVarWidth()
			}
			if !br.ReadBit() {
				br.ReadSignedVarWidth()
			}
		}
	}
}
