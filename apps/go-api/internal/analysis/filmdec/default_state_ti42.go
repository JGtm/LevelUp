package filmdec

// default_state_ti42.go — L'ETAT PAR DEFAUT DE L'ARCHETYPE `ti=42` (arme au sol), porte
// 100 % STATIQUEMENT (lot R5, phase 1.4).
//
// POURQUOI IL MANQUAIT. `default_state_arch.go` porte vingt archetypes et s'interdit
// d'inscrire ceux dont une largeur de feuille n'est pas etablie statiquement. `ti=42`
// n'y figurait pas, et c'est la CONDITION DE REPRISE ecrite au registre le 2026-08-12 pour
// la position des armes au sol : « default-state de `ti=42` resolu ».
//
// LA CHAINE DE RESOLUTION, rejouee (elle est celle de `default_state_arch.go:5-18`) :
//
//	registrar FUN_140e453b4 -> FUN_140e45fc4(world, 0x2a, &PTR_PTR_144701780)  @0x140e4578f
//	unique xref [WRITE] sur 0x144701780 -> FUN_1403721d0 : vtable = 0x1436fd790
//	*(0x1436fd790 + 0x60) = 0x1407f0c68
//
// LA GRAMMAIRE de `FUN_1407f0c68`, feuille par feuille (chacune touche `reader+0x2c`) :
//
//	V                        R(1) ; si 1 -> R(8)
//	FUN_1407f2224            = V + FUN_14080cfe8 (bloc multiplayer-properties) — c'est-a-dire
//	                           EXACTEMENT consumeDefaultStateTI36, deja porte bit-exact
//	R(12)                    -> dst+0x60
//	R(7)                     FUN_1406d84b4, largeur figee par `MOV dword [RSP+0x20],7`
//	                           @0x1407f0d30, juste avant le CALL @0x1407f0d38
//	FUN_1407f2494            bloc de liste (ci-dessous)                     -> dst+0x68
//	ECS_ReadEntityRefIndex5  FUN_1407f2058 = R(1) ; si 0 -> R(5)            -> dst+0xa4
//
// CE QUE CE PORT N'A PAS. Aucun oracle ne le valide : le seul oracle disponible etait la
// marche bit-exacte au record d'IMAGE-CLE, et la phase 2 du lot R5 a REFUTE que le corps
// d'un record d'image-cle soit un record NEW (0 marche exacte sur 1 226 records `ti=37` et
// 9 460 records `ti=38`, seize lectures de corps, cent vingt-huit decalages). Il entre donc
// dans la table pour la MEME raison que les vingt autres — toutes ses largeurs sont etablies
// statiquement — et sa provenance est ecrite ci-dessus, adresse par adresse.
//
// Detail du journal RE : `.ai/V7.5/killweapon/WALK_PORT_NOTES.md`, section « IMAGE-CLE » §4.

// consumeDefaultStateTI42 porte FUN_1407f0c68 (archetype 42, « arme au sol »).
func consumeDefaultStateTI42(br *BitReader) {
	consumeVersionPrefix(br)    // V
	consumeDefaultStateTI36(br) // FUN_1407f2224 = V + bloc multiplayer-properties
	br.ReadBits(12)             // bloc inline +0xc -> dst+0x60
	br.ReadBits(7)              // FUN_1406d84b4, largeur 7 figee @0x1407f0d30
	consume1407f2494(br)        // bloc de liste -> dst+0x68
	consumeGate0R(br, 5)        // ECS_ReadEntityRefIndex5 (FUN_1407f2058)
}

// consume1407f2494 porte FUN_1407f2494, le bloc de liste de `ti=42` :
//
//	porte = R(1)                       FUN_1406cf008 @0x1407f24c1
//	si porte == 0 : FUN_14080d69c      R(1) ; si 1 -> R(32)
//	                (le reader est bien passe : `MOV RDX,RBP` avant le CALL @0x1407f24dc)
//	sinon         : n = R(4)           FUN_1424e1d48
//	                n fois : R(1) ; si 1 -> R(32)   (FUN_1406cf008 + FUN_14080d6f0)
func consume1407f2494(br *BitReader) {
	if !br.ReadBit() {
		consumeOpt32(br) // FUN_14080d69c
		return
	}
	n := int(br.ReadBits(4)) // FUN_1424e1d48 = R(4)
	for i := 0; i < n; i++ {
		consumeOpt32(br) // R(1) ; si 1 -> R(32)
	}
}
