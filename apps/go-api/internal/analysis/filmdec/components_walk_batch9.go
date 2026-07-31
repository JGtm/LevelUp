package filmdec

// components_walk_batch9.go — desers ajoutes pour debloquer la MARCHE SEQUENTIELLE des
// paquets type-0 (chantier dead-state i11, 2026-07-25). Chaque entree a ete resolue par la
// meme chaine statique, sans CheatEngine :
//
//	nom du composant (chunk_00) -> chaine .rdata -> getName() (vtable+0x18 du bloc de 0x50 o)
//	-> bloc de descripteur -> deserialiseur = bloc+0x40 (le serialiseur est en bloc+0x28).
//
// Le bloc de descripteur se localise par le thunk partage FUN_14076ce9c, present en bloc+0x38.

// consumeManagedObjectBoundaryVisibility (ti10 i?) — deser FUN_141169e90 -> FUN_14080ae28 :
// boucle de 32 iterations qui lit UN bit par iteration (FUN_1406cf008) et le range dans un
// u32 de flags. Largeur = 32 bits PLATS, sans gate.
// C'etait le 1er bloqueur non-biped de la marche apres game-engine-team-mapping.
func consumeManagedObjectBoundaryVisibility(br *BitReader) { br.Skip(32) }

// consumeDevicePosition (ti43) — deser FUN_140bef320 :
//
//	FUN_1406d84b4(reader, ..., width=0xe, ...)  = R(14) scalaire dequantifie
//	FUN_1406cf008                               = R(1) flag (state+0x582 bit 4)
func consumeDevicePosition(br *BitReader) { br.Skip(15) }

// consumeGameEngineCampaignTimer (ti2) — deser FUN_1407ee764 -> FUN_140d580d0(dst, reader,
// width=0x10, table) = R(16) + R(16) + FUN_1407f0354 = R(5). Meme forme que
// game-engine-round-timer-component (FUN_1407ee790), deja porte en R(16)+R(16)+R(5).
func consumeGameEngineCampaignTimer(br *BitReader) { br.Skip(37) }

// consumeBipedPosturePhysics (ti35 i55) — deser FUN_142f0293c -> FUN_142f1f630 : R(2) puis
// FUN_141fd997c (resolution d'etat, 0 bit lu).
func consumeBipedPosturePhysics(br *BitReader) { br.Skip(2) }
