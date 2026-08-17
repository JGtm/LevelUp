package filmdec

// default_state_ti42.go — L'ETAT PAR DEFAUT de l'archetype ARME AU SOL (ti=42), porte
// bit a bit depuis `FUN_1407f0c68`.
//
// CHAINE DE RESOLUTION (celle de default_state_arch.go, rejouee pour ti=42 par le lot R5 et
// consignee dans .ai/V7.5/killweapon/WALK_PORT_NOTES.md § IMAGE-CLE §4) :
//
//	FUN_140e453b4 (registrar) -> FUN_140e45fc4(world, 0x2a, &PTR_PTR_144701780) @0x140e4578f
//	xref [WRITE] sur 0x144701780 -> FUN_1403721d0 : PTR_PTR_144701780 = &PTR_LAB_1436fd790
//	vtable 0x1436fd790 , *(vtable + 0x60) = 0x1407f0c68
//
// GRAMMAIRE, feuille par feuille (chaque feuille touche `reader+0x2c`) :
//
//	1  V                     R(1) ; si 1 -> R(8)              FUN_1406cf008 + bloc inline +8
//	2  FUN_1407f2224         = consumeDefaultStateTI36        MOV EDX,0x60 @0x1407f0cd1
//	                           (V + bloc multiplayer-properties FUN_14080cfe8)
//	3  R(12)  -> dst+0x60                                     bloc inline +0xc
//	4  R(7)   -> dst+0x64                                     largeur figee par
//	                                                          `MOV dword [RSP+0x20],7` @0x1407f0d30
//	                                                          avant CALL 0x1406d84b4 @0x1407f0d38
//	5  FUN_1407f2494(dst+0x68, reader)                        CALL @0x1407f0d49
//	6  ECS_ReadEntityRefIndex5 = FUN_1407f2058                R(1) ; si 0 -> R(5) -> dst+0xa4
//
// LE POINT 5 NE S'ECRIT PAS DEUX FOIS. `FUN_1407f2494` avait deja ete porte puis supprime le
// 2026-07-26 comme doublon de `consumeWeaponMagazineList` (aujourd'hui dans
// components_object.go, appele par `consumeWeaponStateTypeInfoVariant`). Le doublon a ete
// retranche sur pieces : la grammaire decompilee (porte R(1) ; si 0 -> FUN_14080d69c
// [R(1) ; si 1 -> R(32)] ; sinon n=R(4) puis n x [R(1) ; si 1 -> R(32)]) est celle de
// `consumeWeaponMagazineList`, feuille pour feuille et porte pour porte. On REUTILISE, on ne
// recopie pas : une seconde copie re-divergerait au premier correctif (regle des 2 copies).
//
// L'IDENTITE DE L'ARME SORT PAR LE MEME CHEMIN QUE CELLE DE L'EQUIPEMENT : le bloc
// `object-multiplayer-properties` du point 2 publie son mot de 32 bits inconditionnel par
// `mppHook` (MPPWord32). Pour ti=37 ce mot est le GlobalID du tag `eqip` ; l'hypothese pour
// ti=42 est le tag `weap`, et elle se MESURE en croisant ce mot avec la famille high-32 lue
// aux images-cles (deux chaines independantes), jamais en le supposant.
//
// LARGEUR DU CHEMIN MINIMAL (toutes portes au plus court, largeurs MPP 9/5) :
// 1 + (1 + 56) + 12 + 7 + 2 + 1 = 80 bits — a comparer aux 60 bits de ti=37, dont ti=42
// reprend le bloc MPP a l'identique.

// consumeDefaultStateTI42 porte FUN_1407f0c68 (archetype 42, « arme au sol »).
func consumeDefaultStateTI42(br *BitReader) {
	consumeVersionPrefix(br)      // 1. V
	consumeDefaultStateTI36(br)   // 2. FUN_1407f2224 : V + bloc MPP (FUN_14080cfe8)
	br.ReadBits(12)               // 3. inline R(12) -> dst+0x60
	br.ReadBits(7)                // 4. FUN_1406d84b4 R(7) -> dst+0x64
	consumeWeaponMagazineList(br) // 5. FUN_1407f2494 (deja porte, cf. components_object.go)
	consumeGate0R(br, 5)          // 6. ECS_ReadEntityRefIndex5 = FUN_1407f2058
}

// GroundWeaponDefaultStateMinBits est la largeur du chemin minimal de consumeDefaultStateTI42
// aux largeurs MPP par defaut. Publiee parce que l'oracle d'offset la COMPARE au flux : c'est
// la prediction que l'histogramme des distances doit retrouver, ou refuter.
const GroundWeaponDefaultStateMinBits = 80
