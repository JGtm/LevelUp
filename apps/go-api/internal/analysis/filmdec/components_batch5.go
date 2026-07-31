package filmdec

// Batch-5 ECS component desers (game-engine/statborg/track/state-checksum). RE'd via
// workflow port-ecs-batch5 (Ghidra), confident=true.

// NOTE 2026-07-25 : l'ancien consumeGameEngineCampaignTimer (attribue a FUN_14076e744) etait
// MORT (aucun case ne l'appelait) et pointait sur la mauvaise fonction. Le deserialiseur reel du
// composant, resolu par la chaine statique nom -> getName -> descripteur -> bloc+0x40, est
// FUN_1407ee764 ; il est porte dans components_walk_batch9.go.

// game-engine-disabled-kill-volume-flags-component (FUN_142f03498): R(13) count + count x R(1).
func consumeGameEngineDisabledKillVolume(br *BitReader) {
	count := br.ReadBits(13)
	for i := uint64(0); i < count; i++ {
		br.ReadBit()
	}
}

// track-frame-component (FUN_142ED740C): R(6) + R(1) + R(1) + [flagA==0: R(12)=N + N bits].
func consumeTrackFrame(br *BitReader) {
	br.ReadBits(6)
	flagA := br.ReadBit()
	br.ReadBit() // flagB
	if !flagA {
		n := uint(br.ReadBits(12))
		for n > 0 {
			t := n
			if t > 64 {
				t = 64
			}
			br.ReadBits(t)
			n -= t
		}
	}
}

// tacmap-category (FUN): R(32) + R(1)[+R(4)].
func consumeTacmapCategory(br *BitReader) {
	br.ReadBits(32)
	if !br.ReadBit() {
		br.ReadBits(4)
	}
}
