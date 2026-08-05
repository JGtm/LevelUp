package filmdec

// Batch8 FRAME component desers — RE'd via workflow filmdec-port-top-components (2026-06-14,
// ghidra-verified, top desync-count components of the clean-frame histogram). Goal: raise the
// per-frame clean rate so the World binding completes (the death chains block on unbound
// world-object slots, a symptom of the 63% gradient). All widths LITERAL unless noted.
//
// IMPORTANT recipe nuance discovered here: for NAME-INDEXED descriptor tables (equipment-*,
// biped-emp-timer), the read/write codec is at descriptor+0x40, NOT +0x28 (+0x28 is a
// 'return 1' predicate). The +0x28 recipe only holds for the component-replication descriptors.
//
// Retrait 2026-08-01 (lot C, PLAN_DETTE_AVANT_MERGE) de dix ports orphelins de ce lot.
// Sept sont SUPPLANTÉS : leur nom de composant est déjà dispatché par traverse.go, qui porte
// la grammaire en INLINE depuis la table ECS live (J0.4) — et cette table corrige au passage
// des `ti=` devinés ici (`game-engine-game-finished` est ti=2 i3, pas ti=0). Trois visaient
// des archétypes ni décodés ni planifiés : `sound-placement-state-data` (ti=19),
// `state-broker-state-changed-data` (ti=46), `forge-player-data-edited-objects-ids` (ti=48).
// Les grandeurs mesurées à l'appui de ces verdicts sont au journal du plan de dette.

// equipment-command-tick-component: déjà porté dans components_batch3.go
// (consumeEquipmentCommandTick) ; la RE batch8 confirme la même grammaire (eq bit + champs
// optU8). Seul le câblage manquait.

// spawn-filter-weight-component (FUN_142ed70b8 -> FUN_1406d84b4 width 0x10): quantized float,
// 16 bits consumed (dequant bounds are globals that don't change the bit count).
func consumeSpawnFilterWeight(br *BitReader) {
	br.ReadBits(16)
}
