package grammar

// components_navpoint_suite.go — L ARCHETYPE `managed-navpoint` (ti=12) APRES `i14` : les
// composants ou la marche d une trame partie de la fin de la vue A s arretait.
//
// # CE QUI FONDE CHAQUE LARGEUR
//
// Chaque lecteur est celui du jeu (`HaloInfinite.exe` HI_1_13_0) : nom -> `getName` -> descripteur
// -> deserialiseur (le slot qui suit le thunk `FUN_14076ce9c`), son adresse sur le `case` du
// maillon [consumeNavpointComponent]. L ecrivain (le slot `+0x10` du descripteur) est relu pour
// chaque champ : il ecrit ce que le lecteur lit, sans porte.

// Les etiquettes de registre des composants portes ici.
const (
	compNavpointOverrideFlags = "managed-navpoint-override-flags"
)

// Largeurs lues dans le jeu.
const (
	// navpointOverrideFlagsBits : `i16`, `FUN_140ebf834` -> `FUN_140ebf854` (`ADD [flux+0x2c], 5`,
	// `SHR R9, 0x3b`) vers `etat + 0x70c` ; l ecrivain (`142ed0e2c`) ecrit les cinq bits du meme mot.
	navpointOverrideFlagsBits = 5
)

// consumeNavpointOverrideFlags (ti=12 i16) — `FUN_140ebf834` : `R(5)` plat, sans porte.
func consumeNavpointOverrideFlags(br *Lecteur) { br.ReadBits(navpointOverrideFlagsBits) }
