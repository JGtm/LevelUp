package filmdec

// ability_energy.go — LA JAUGE D'UTILISATION DE LA CAPACITÉ D'ARMURE (i56), extraite de
// components_biped_ability.go le 2026-08-15 : ce fichier dépassait déjà le seuil de 500
// lignes, et cette section venait de gagner un hook de publication. Elle est autonome — un
// composant, sa rétro-ingénierie, sa grammaire, sa sonde — comme ability_rank.go l'est
// pour i48.

// ---------------------------------------------------------------------------
// i56 biped-spartan-ability-energy-component  (deser FUN_140fc1410 -> FUN_140fc147c)
//   Registry string "biped-spartan-ability-energy-component" @143c98de0.
//   Name-thunk 141177540 (lea rax,[143c98de0]; ret).
//   Descriptor @143d0ce60 ; deser thunk (6e ptr, base+0x28) @140fc1410.
// ---------------------------------------------------------------------------

// consumeBipedSpartanAbilityEnergy mirrors FUN_140fc1410 : R(3) MASQUE, puis 7 BITS PAR
// CHARGE ARMEE.
//
// ATTENTION — le DECOMPILE MENT ICI. Ghidra emet
// `WARNING: Removing unreachable block (ram, 0x14246a410)` et supprime justement le bloc
// qui lit les 7 bits ; le port precedent en avait conclu « 3 bits, CONFIRMED bit-exact »,
// ce qui etait FAUX. Seul le DESASSEMBLAGE fait foi (relu le 2026-07-26) :
//
//	140fc1432 CALL 140fc147c        masque = R(3), rendu par out-param [RSP+0x40]
//	                                (140fc14a1 ADD [RCX+0x2c],0x3 ; SHR R9,0x3d = 3 bits de tete)
//	140fc143e boucle i = 0..2 :
//	140fc1446   EDX = 1 << i
//	140fc1448   TEST EAX,EDX
//	140fc144a   JNZ 0x14246a410     bit i arme -> bloc froid externalise
//	140fc1450   R9B = 0x7f          bit i a 0 -> valeur par defaut, AUCUN bit lu
//	140fc1459   [RDI + i + 0x12ea] = R9B
//	14246a425 ADD [RBX+0x2c],0x7    bloc froid : +7 bits
//	14246a438 SHR R9,0x39           ( >>57 = les 7 bits de tete )
//	14246a4d2 JMP 140fc1453         retour dans la boucle, aucun autre bit
//
// Cout total : 3 + 7 * popcount(masque), soit 3 a 24 bits (et non 3).
// 0x7F est la valeur par defaut (charge pleine) quand le bit du masque est a 0.
// LES VALEURS NE SONT PLUS JETÉES (2026-08-15) : comme i48 le 14/08 et comme les quatre
// composants de ti=37 (equipment_state.go), ce déser consommait ses bits pour rester aligné
// et abandonnait ce qu'il lisait. `abilityEnergyHook` les publie ; le parcours de bits est
// inchangé, et 0x7F reste la valeur par défaut NON transmise d'un emplacement non armé.
func consumeBipedSpartanAbilityEnergy(br *BitReader) {
	mask := br.ReadBits(3) // FUN_140fc147c
	var ch [AbilityEnergyCharges]int
	for i := uint(0); i < AbilityEnergyCharges; i++ {
		ch[i] = AbilityEnergyUnarmed
		if mask&(1<<i) != 0 {
			ch[i] = int(br.ReadBits(7)) // bloc froid 0x14246a410
		}
	}
	if abilityEnergyHook != nil {
		abilityEnergyHook(uint32(mask), ch)
	}
}

// AbilityEnergyCharges est le nombre d'emplacements de charge décrits par le masque R(3).
const AbilityEnergyCharges = 3

// AbilityEnergyUnarmed est publié pour un emplacement dont le bit de masque vaut 0 : le film
// ne transmet RIEN pour cet emplacement. Ce n'est pas la valeur 0, et ce n'est pas non plus
// 0x7F — le moteur pose 0x7F en RAM, mais l'assimiler à une lecture fabriquerait des chutes.
const AbilityEnergyUnarmed = -1

// abilityEnergyHook, si non nil, reçoit d'i56 le masque R(3) et les trois valeurs 7 bits
// (AbilityEnergyUnarmed pour un emplacement non armé). Le déser reste inchangé bit pour bit.
var abilityEnergyHook func(mask uint32, ch [AbilityEnergyCharges]int)

// SetAbilityEnergyHook installe (ou retire, avec nil) la sonde de lecture d'i56.
func SetAbilityEnergyHook(h func(mask uint32, ch [AbilityEnergyCharges]int)) {
	abilityEnergyHook = h
}
