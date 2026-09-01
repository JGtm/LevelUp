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
// BRANCHÉE le 2026-09-01 : la condition de retrait posée le 2026-08-01 (« branchée ou supprimée
// quand ti=11 sera décodé ») est LEVÉE — `consumeByName` route désormais i2 et i9 vers cette
// fonction (`components_managed_objective.go` porte les trente-et-un autres composants de
// l'archétype). Le `nolint:unused` qui accompagnait la garde est retiré avec elle.
//
// LA GRAMMAIRE EST RECOUPÉE AU BINAIRE depuis ce jour, ce qu'elle n'était pas : le sérialiseur
// réseau du descripteur est `FUN_142edb5bc` pour i2 et `FUN_142edba00` pour i9, tous deux vers
// `FUN_142c70d5c` — R(1) de présence, puis R(32) (FUN_1407edaf4), puis R(3) de compte, puis
// autant d'éléments taggés sur R(3) (FUN_142c70e88). La forme portée depuis le workflow
// port-ecs-deathchains coïncide ; seule la charge par tag reste issue de ce workflow.
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
