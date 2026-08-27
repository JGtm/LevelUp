package filmdec

// components_managed_objective.go — LE SUIVI D'OBJECTIF DU HUD : ti=11 `managed-objective-*`.
// Deserialiseurs des feuilles TRIVIALES resolues au Ghidra (docs de conception
// `.ai/V7.5/replay2d/PISTE_A_ti11_color.md` et `PISTE_B_ti11_sous_entites.md`) et hook de
// publication nomme par famille d'archetype.
//
// CE QUE ti=11 EST. `managed-objective` est le DESCRIPTEUR d'objectif du HUD (34 composants),
// pas l'objet physique : aucun composant ne porte de position (lot R4). Les feuilles cablees
// ici sont les seules RESOLUES bit-exact ; toute la structure est un ARBRE parent/enfant
// (i15 parent / i16-31 enfants) borne a 16 slots (offset d'i32 prouve la borne).
//
// LA GRAMMAIRE VIENT DU DESERIALISEUR DU JEU, adresse par adresse (PISTE_A/PISTE_B) :
//
//	i0  managed-objective-timers-component            FUN_142ed5a6c -> FUN_1410d9088  2x R(7)
//	i1  managed-objective-color-component             FUN_142ed544c                   4x R(8) RGBA
//	i3  managed-objective-object-reference-component  FUN_142ed5550                   R(32) GlobalID
//	i5  managed-objective-type-component              FUN_1410fc4a4                   R(32)
//	i12 managed-objective-progress-component          FUN_142ed575c                   R(32)
//	i13 managed-objective-required-progress-component FUN_142ed5844                   R(32)
//	i14 managed-objective-state-component             FUN_142ed5948                   R(3)
//	i15 managed-objective-parent-objective-component  FUN_142ed5674                   R(32) GlobalID
//	i16..i31 managed-objective-sub-objective-entities FUN_142ed5974                   R(32) GlobalID
//
// Toutes triviales : largeur immediate, aucune porte, aucune quantification (sauf le
// dequant [0,1] de i1, publie BRUT ici), aucun vec3, aucun compte data-dependant. Chaque
// deser rend toujours vrai cote jeu => aucune desynchronisation possible (statut `porte`).
//
// LES VALEURS SONT PUBLIEES BRUTES (quantum), meme convention que ti=10/12/13. Les 16 slots
// sub-objective PARTAGENT leur nom : `consumeByName` ne recoit pas l'index, l'appelant
// reconstruit l'occurrence depuis l'ORDRE DU MASQUE (helper `ManagedObjectiveSubEntitySlot`),
// exactement le partage de roles des `rtpc` ti=10 et des `masked-property` ti=13.

// Etiquettes de registre des composants portes par ce fichier. Des constantes, pas des
// litteraux repetes : elles servent au routage de `consumeByName` et au `String()` du champ.
const (
	compManagedObjectiveTimers    = "managed-objective-timers-component"                 // i0
	compManagedObjectiveColor     = "managed-objective-color-component"                  // i1
	compManagedObjectiveObjectRef = "managed-objective-object-reference-component"       // i3
	compManagedObjectiveType      = "managed-objective-type-component"                   // i5
	compManagedObjectiveProgress  = "managed-objective-progress-component"               // i12
	compManagedObjectiveRequired  = "managed-objective-required-progress-component"      // i13
	compManagedObjectiveState     = "managed-objective-state-component"                  // i14
	compManagedObjectiveParent    = "managed-objective-parent-objective-component"       // i15
	compManagedObjectiveSubEntity = "managed-objective-sub-objective-entities-component" // i16..i31
)

// Largeurs des feuilles (immediats du desassemblage, cf. en-tete).
const (
	managedObjectiveTimerBits    = 7  // i0 : 2 lectures
	managedObjectiveColorBits    = 8  // i1 : 4 lectures RGBA
	managedObjectiveRefBits      = 32 // i3/i5/i12/i13/i15/i16-31 : R(32)
	managedObjectiveStateBits    = 3  // i14 : R(3)
	managedObjectiveSubEntityMax = 16 // borne DURE du tableau (offset d'i32 le prouve)
)

// ManagedObjectiveField designe le champ publie de ti=11. Enumeration STABLE et NOMMEE, jamais
// un index de registre : le decoupage du registre CHANGE AVEC LE BUILD (lot 0), donc un index
// de composant ne designe pas la meme chose d'un film a l'autre.
type ManagedObjectiveField int

// Les champs publies, et leur compte.
const (
	ManagedObjectiveTimers     ManagedObjectiveField = iota // i0      : 2x R(7)
	ManagedObjectiveColor                                   // i1      : 4x R(8) RGBA
	ManagedObjectiveObjectRef                               // i3      : R(32) GlobalID (LE PORTEUR)
	ManagedObjectiveType                                    // i5      : R(32)
	ManagedObjectiveProgress                                // i12     : R(32)
	ManagedObjectiveRequired                                // i13     : R(32)
	ManagedObjectiveState                                   // i14     : R(3)
	ManagedObjectiveParent                                  // i15     : R(32) GlobalID
	ManagedObjectiveSubEntity                               // i16..i31: R(32) GlobalID
	ManagedObjectiveFieldCount = 9
)

// String rend l'etiquette de registre du champ.
func (f ManagedObjectiveField) String() string {
	switch f {
	case ManagedObjectiveTimers:
		return compManagedObjectiveTimers
	case ManagedObjectiveColor:
		return compManagedObjectiveColor
	case ManagedObjectiveObjectRef:
		return compManagedObjectiveObjectRef
	case ManagedObjectiveType:
		return compManagedObjectiveType
	case ManagedObjectiveProgress:
		return compManagedObjectiveProgress
	case ManagedObjectiveRequired:
		return compManagedObjectiveRequired
	case ManagedObjectiveState:
		return compManagedObjectiveState
	case ManagedObjectiveParent:
		return compManagedObjectiveParent
	case ManagedObjectiveSubEntity:
		return compManagedObjectiveSubEntity
	}
	return "champ inconnu"
}

// managedObjectiveHook, si non nil, recoit chaque lecture d'un champ de ti=11.
//
// PAS DE `present` ICI : aucun de ces composants n'a de porte de tete (lectures
// inconditionnelles). Global de paquet : l'appelant detient `LockProcessDecode`. Un hook
// SEPARE de ceux de ti=10/12/13, meme raison qu'entre eux : archetype distinct, slots
// disjoints.
var managedObjectiveHook func(f ManagedObjectiveField, values []uint64)

// SetManagedObjectiveHook installe (ou retire, avec nil) la sonde des composants de ti=11.
func SetManagedObjectiveHook(h func(f ManagedObjectiveField, values []uint64)) {
	managedObjectiveHook = h
}

func publishManagedObjective(f ManagedObjectiveField, values ...uint64) {
	if managedObjectiveHook != nil {
		managedObjectiveHook(f, values)
	}
}

// consumeManagedObjectiveU32 lit un R(32) et le publie sous le champ demande. Feuille partagee
// par i3/i5/i12/i13/i15 et les 16 slots i16-31 (tous FUN_142ed55.. : R(32) pur, seule la
// destination change cote jeu). Un seul helper, pas six copies du meme `ReadBits(32)`.
func consumeManagedObjectiveU32(br *BitReader, f ManagedObjectiveField) {
	publishManagedObjective(f, br.ReadBits(managedObjectiveRefBits))
}

// consumeManagedObjectiveTimers (ti=11 i0) — FUN_142ed5a6c -> FUN_1410d9088 : 2x R(7).
// La valeur native = quantum lu (le deser rend quantum-1 ; le biais est publie tel quel).
func consumeManagedObjectiveTimers(br *BitReader) {
	t0 := br.ReadBits(managedObjectiveTimerBits)
	t1 := br.ReadBits(managedObjectiveTimerBits)
	publishManagedObjective(ManagedObjectiveTimers, t0, t1)
}

// consumeManagedObjectiveColor (ti=11 i1) — FUN_142ed544c : 4x R(8) dequant [0,1] = RGBA.
// Publie les quatre quantums bruts (convention de dequant non figee, cf. ti=10 boundary-color).
func consumeManagedObjectiveColor(br *BitReader) {
	r := br.ReadBits(managedObjectiveColorBits)
	g := br.ReadBits(managedObjectiveColorBits)
	b := br.ReadBits(managedObjectiveColorBits)
	a := br.ReadBits(managedObjectiveColorBits)
	publishManagedObjective(ManagedObjectiveColor, r, g, b, a)
}

// consumeManagedObjectiveState (ti=11 i14) — FUN_142ed5948 : R(3) (conteste / neutre / tenu).
func consumeManagedObjectiveState(br *BitReader) {
	publishManagedObjective(ManagedObjectiveState, br.ReadBits(managedObjectiveStateBits))
}

// ManagedObjectiveSubEntitySlot rend le slot 0..15 porte par la i-eme occurrence du composant
// sub-objective-entities dans l'ORDRE DU MASQUE (l'appelant compte les occurrences ; le deser,
// comme le jeu, ne connait pas son index). Rend -1 hors de la plage des 16 slots declares.
//
// MEME PARTAGE DE ROLES que `ManagedPropertyFilmIndex` (ti=13) et les `rtpc` (ti=10) : la
// grammaire ne depend pas de l'index, seule l'identite de slot en depend, et l'ordre de lecture
// est fixe => l'etiquette d'occurrence est stable image-par-image et film-par-film.
func ManagedObjectiveSubEntitySlot(occurrence int) int {
	if occurrence < 0 || occurrence >= managedObjectiveSubEntityMax {
		return -1
	}
	return occurrence
}

// ManagedObjectiveColorValue dequantifie une composante de couleur i1 (8 bits) dans [0, 1],
// meme convention (milieu d'intervalle) que ti=10 boundary-color.
func ManagedObjectiveColorValue(q uint64) float32 {
	return dequantMidpoint(q, managedObjectiveColorBits, 0, 1)
}
