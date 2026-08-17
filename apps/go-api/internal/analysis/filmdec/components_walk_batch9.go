package filmdec

// components_walk_batch9.go — desers ajoutes pour debloquer la MARCHE SEQUENTIELLE des
// paquets type-0 (chantier dead-state i11, 2026-07-25). Chaque entree a ete resolue par la
// meme chaine statique, sans CheatEngine :
//
//	nom du composant (chunk_00) -> chaine .rdata -> getName() (vtable+0x18 du bloc de 0x50 o)
//	-> bloc de descripteur -> deserialiseur = bloc+0x40 (le serialiseur est en bloc+0x28).
//
// Le bloc de descripteur se localise par le thunk partage FUN_14076ce9c, present en bloc+0x38.

// consumeManagedObjectBoundaryVisibility (ti=10 i0) — deser FUN_141169e90 -> FUN_14080ae28 :
// boucle de 32 iterations qui lit UN bit par iteration (FUN_1406cf008) et le range dans un
// u32 de flags. Largeur = 32 bits PLATS, sans gate.
// C'etait le 1er bloqueur non-biped de la marche apres game-engine-team-mapping.
//
// LE `Skip(32)` EST DEVENU UNE LECTURE (lot 0 item 0.6, 2026-08-17), et le u32 est assemble
// COMME LE JEU L'ASSEMBLE : l'iteration i pose son bit au rang i. Ce n'est pas la meme valeur
// que `ReadBits(32)`, qui rendrait le bit de l'iteration 0 au rang 31 — et c'est precisement
// pourquoi la boucle est ecrite bit par bit ici plutot que raccourcie. La CONSOMMATION est
// identique au bit (32 bits dans les deux cas) : `TestHooksConsumeSameBitsWithoutHook` le
// verifie.
//
// CE QUE CES 32 DRAPEAUX SONT, ET CE QU'ILS NE SONT PAS : la visibilite de bordure de l'objet
// SCRIPTE du mode (ti=10). Ce qu'ils signifient est la question du lot C ; ici, on les rend
// lisibles, sans les interpreter.
func consumeManagedObjectBoundaryVisibility(br *BitReader) {
	var flags uint64
	for i := 0; i < 32; i++ {
		if br.ReadBit() {
			flags |= 1 << uint(i)
		}
	}
	publishManagedObject(ManagedObjectBoundaryVisibility, flags)
}

// consumeDevicePosition (ti43) — deser FUN_140bef320 :
//
//	FUN_1406d84b4(reader, ..., width=0xe, ...)  = R(14) scalaire dequantifie
//	FUN_1406cf008                               = R(1) flag (state+0x582 bit 4)
func consumeDevicePosition(br *BitReader) { br.Skip(15) }

// consumeGameEngineCampaignTimer (ti2) — deser FUN_1407ee764 -> FUN_140d580d0(dst, reader,
// width=0x10, table) = R(16) + R(16) + FUN_1407f0354 = R(5). Meme forme que
// game-engine-round-timer-component (FUN_1407ee790), deja porte en R(16)+R(16)+R(5).
//
// NOTE 2026-07-25 (rapatriee ici le 2026-08-01 a la suppression de components_batch5.go) :
// un port anterieur attribuait ce composant a FUN_14076e744 — mauvaise fonction, et mort
// faute d'appelant. La chaine statique nom -> getName -> descripteur -> bloc+0x40 donne
// FUN_1407ee764, porte ci-dessous.
func consumeGameEngineCampaignTimer(br *BitReader) { br.Skip(37) }

// consumeBipedPosturePhysics (ti35 i55) — deser FUN_142f0293c -> FUN_142f1f630 : R(2) puis
// FUN_141fd997c (resolution d'etat, 0 bit lu).
func consumeBipedPosturePhysics(br *BitReader) { br.Skip(2) }

// -----------------------------------------------------------------------------------------
// LE HOOK DE L'OBJET SCRIPTE DU MODE (ti=10)
// -----------------------------------------------------------------------------------------

// ManagedObjectField designe le champ publie de ti=10. Enumeration STABLE, pas un index de
// registre (meme raison que `GameEngineField`). Un seul champ aujourd'hui : les 29 autres
// composants de l'archetype sont `non_porte`, et c'est le gisement du lot C.
type ManagedObjectField int

// Le champ publie, et son compte.
const (
	ManagedObjectBoundaryVisibility ManagedObjectField = iota // i0 : 32 drapeaux plats
	ManagedObjectFieldCount                            = 1
)

// String rend l'etiquette de registre du champ.
func (f ManagedObjectField) String() string {
	if f == ManagedObjectBoundaryVisibility {
		return "managed-object-boundary-visibility-component"
	}
	return "champ inconnu"
}

// managedObjectHook, si non nil, recoit chaque lecture d'un champ de ti=10.
//
// PAS DE `present` ICI : le composant n'a pas de porte de tete, ses 32 bits sont toujours lus.
// Global de paquet : l'appelant detient `LockProcessDecode`.
var managedObjectHook func(f ManagedObjectField, values []uint64)

// SetManagedObjectHook installe (ou retire, avec nil) la sonde des composants de ti=10.
func SetManagedObjectHook(h func(f ManagedObjectField, values []uint64)) { managedObjectHook = h }

func publishManagedObject(f ManagedObjectField, values ...uint64) {
	if managedObjectHook != nil {
		managedObjectHook(f, values)
	}
}
