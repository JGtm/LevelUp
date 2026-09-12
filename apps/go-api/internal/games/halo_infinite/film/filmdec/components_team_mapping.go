package filmdec

// Non-biped archetype component desers reached by the FRAME record loop walk. Porting
// these lets the loop advance PAST the first biped (slot 519) to reach the other
// player bipeds (512-518) and the world entities — the loop stops at the first
// un-ported present component, so any archetype interleaved before a biped blocks it.
//
// All widths here are LITERAL (verified via Ghidra): none of these leaf readers loads
// a map-load width table. See the verdict in the L3 handoff.

// ---------------------------------------------------------------------------
// typeIndex 0, i0  game-engine-team-mapping-component  (deser FUN_140f58200)
//   Registry string "game-engine-team-mapping-component" @143c985c0.
//   Descriptor @143d0f7a8 ; deser thunk (vtable+0x28) @140f58200.
//   THE #1 non-biped blocker of the record-loop walk (it sits early in the loop and
//   gates reaching the player bipeds 512-518).
// ---------------------------------------------------------------------------

// consumeGameEngineTeamMapping mirrors FUN_140f58200 (order verified from the disasm,
// the destination offsets RBX+{0,2,4,6,8,a} make the field widths unambiguous):
//
//	FUN_140f582d0 = R(8)   -> state+0   (field A)
//	FUN_140f58324 = R(9)   -> state+2   (field B)
//	FUN_140f58324 = R(9)   -> state+4   (field C)
//	FUN_140f582d0 = R(8)   -> state+6   (MASK M; the loop tests `word ptr [state+6]`)
//	FUN_140f582d0 = R(8)   -> state+8   (field D)
//	FUN_140f582d0 = R(8)   -> state+a   (field E)
//	for i in 0..7: if (M >> i) & 1: FUN_1407ef804 = R(4)   (per-team value, value-1)
//
// Leaf widths CONFIRMED bit-exact: FUN_140f582d0 = flat R(8), FUN_140f58324 = flat R(9),
// FUN_1407ef804 = flat R(4). The mask is the 4th read (R(8) at state+6), NOT one of the
// R(9) fields — the disasm `LEA R14,[RBX+0x6]` then `TEST word ptr [R14],AX` proves it.
func consumeGameEngineTeamMapping(br *BitReader) {
	br.ReadBits(8)         // FUN_140f582d0 = R(8) field A (state+0)
	br.ReadBits(9)         // FUN_140f58324 = R(9) field B (state+2)
	br.ReadBits(9)         // FUN_140f58324 = R(9) field C (state+4)
	mask := br.ReadBits(8) // FUN_140f582d0 = R(8) MASK M (state+6)
	br.ReadBits(8)         // FUN_140f582d0 = R(8) field D (state+8)
	br.ReadBits(8)         // FUN_140f582d0 = R(8) field E (state+a)
	for i := uint(0); i < 8; i++ {
		if (mask>>i)&1 != 0 {
			br.ReadBits(4) // FUN_1407ef804 = R(4) per-present-team value
		}
	}
}
