package filmdec

// Batch-4 ECS component desers (shared per-frame entities: game-engine/spawn-filter/
// weapon/objective/physics). RE'd via workflow port-ecs-batch4 (Ghidra), confident=true.
//
// Retrait 2026-08-01 (lot C, PLAN_DETTE_AVANT_MERGE) de trois ports orphelins :
// `consumeGameEngineAlliance` (supplanté par le port INLINE de traverse.go pour
// `game-engine-alliance-component`, ti=0/1/2 i10), `consumeSupplyLinesBusy` et
// `consumeLowFrequency` (aucune adresse FUN_ à leur commentaire, donc aucune provenance
// vérifiable ; leurs archétypes — ti=26 i32 derrière un trou à i0, ti=3 hors World — ne
// sont pas décodés et ne sont planifiés nulle part).

// statborg-round-outcomes-component (FUN_142ed71a4): 32x R(2).
func consumeStatborgRoundOutcomes(br *BitReader) {
	for i := 0; i < 32; i++ {
		br.ReadBits(2)
	}
}
