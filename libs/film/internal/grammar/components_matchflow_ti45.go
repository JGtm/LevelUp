package grammar

// components_matchflow_ti45.go — L ARCHETYPE `matchflow-*` (ti=45), composants `i0` et `i1`.
//
// Lu dans le jeu (HaloInfinite.exe HI_1_13_0, Ghidra, lecture seule) : nom -> accesseur de nom
// (141177bc0) -> descripteur (slot 143d07be0) -> lecteur `FUN_14101cdd8`, ecrivain `FUN_142edbf94`.
//
//	lecteur   FUN_14101d200 : R(4), rend la valeur - 1 -> etat+0x0 ; puis quatre R(32) bruts
//	          (`ADD [flux+0x2c], 0x20`) -> etat+0xc, +0x10, +0x4, +0x8, dans cet ordre
//	ecrivain  FUN_1407ebac4 : l octet etat+0x0 plus 1 sur quatre bits ; puis les quatre mots de 32
//	          bits dans le meme ordre
//
// Aucune porte, aucune dependance au niveau : 132 bits, toujours.

// compMatchflowSequenceData : l etiquette de registre de `ti=45 i0`.
const compMatchflowSequenceData = "matchflow-sequence-data-component"

// Largeurs lues dans le lecteur du jeu.
const (
	largeurIndexDeSequence uint = 4  // FUN_14101d200 : `ADD [flux+0x2c], 4`, `SHR R9, 0x3c`
	largeurMotDeSequence   uint = 32 // `ADD [flux+0x2c], EBP` avec EBP = 0x20
	motsDeSequence              = 4  // etat+0xc, +0x10, +0x4, +0x8
)

// consumeMatchflowSequenceData (ti=45 i0) — `FUN_14101cdd8` : `R(4)` puis quatre `R(32)`.
func consumeMatchflowSequenceData(br *Lecteur) {
	br.ReadBits(largeurIndexDeSequence)
	for range motsDeSequence {
		br.ReadBits(largeurMotDeSequence)
	}
}

// compMatchflowFocusData : l etiquette de registre de `ti=45 i1`.
const compMatchflowFocusData = "matchflow-focus-data-component"

// Largeurs du lecteur `FUN_141167744` (accesseur de nom 141177bb0, descripteur 143d07b90) : deux
// entiers signes, etendus par leur bit de poids fort. L ecrivain (`142edbda4`) ecrit `etat+0x14 & 0x3f`
// sur six bits puis `etat+0x18 & 0xf` sur quatre.
const (
	largeurFocusA uint = 6 // `+0x2c += 6`, signe par le bit 0x20 -> etat+0x14
	largeurFocusB uint = 4 // `+0x2c += 4`, signe par le bit 0x8 -> etat+0x18
)

// consumeMatchflowFocusData (ti=45 i1) — `FUN_141167744` : `R(6)` puis `R(4)`, sans porte.
func consumeMatchflowFocusData(br *Lecteur) {
	br.ReadBits(largeurFocusA)
	br.ReadBits(largeurFocusB)
}
