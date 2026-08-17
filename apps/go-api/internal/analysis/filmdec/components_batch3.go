package filmdec

// Batch-3 ECS component desers (death-slot archetype chains ti=12/37/11/27/30/0/6).
// RE'd via workflow port-ecs-deathchains (Ghidra), confident=true only; the polymorphic/
// version-gated navpoint-filter components are left unported (they desync only if present,
// rare for the death-frame slots).

// objective formatted-text + secondary share the same shape (R(1) presence + value + tagged list).
// Lecteur de DEUX composants de l'archétype OBJECTIFS (ti=11), couvert 0/34 par le dispatch :
// `managed-objective-formatted-text-component` (i2) et son jumeau
// `managed-objective-secondary-formatted-text-component` (i9). Les deux noms sont écrits EN
// ENTIER parce que la table ECS les déclare `deser_non_cable` en pointant ce fichier, et que
// son garde-rail G1 exige que le nom du composant y apparaisse (`checkCodeSource`).
//
// GARDÉE SANS APPELANT le 2026-08-01 (lot C, PLAN_DETTE_AVANT_MERGE) : ti=11 est PLANIFIÉ
// mais non décodé — le traverseur s'arrête à i0 (traverse.go:1187), la brancher seule ne
// changerait donc rien, et le gate du lot exige des artefacts identiques.
// CONDITION DE RETRAIT : branchée ou supprimée quand ti=11 sera décodé (master plan J6-A,
// ordre interne §4 « Objectifs étape 3 » / PLAN_OBJECTIFS_TEMPS_REEL).
//
//nolint:unused // grammaire d'un composant de ti=11 — voir la condition de retrait ci-dessus.
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

// (consumeEquipmentActivated a rejoint equipment_state.go le 2026-08-15 : il y publie
// désormais la valeur qu'il consommait sans la rendre — même grammaire, mêmes bits.)

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
