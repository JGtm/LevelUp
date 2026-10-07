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
	compNavpointOverrideFlags  = "managed-navpoint-override-flags"
	compNavpointPositionOffset = "managed-navpoint-position-offset"
)

// Largeurs lues dans le jeu.
const (
	// navpointOverrideFlagsBits : `i16`, `FUN_140ebf834` -> `FUN_140ebf854` (`ADD [flux+0x2c], 5`,
	// `SHR R9, 0x3b`) vers `etat + 0x70c` ; l ecrivain (`142ed0e2c`) ecrit les cinq bits du meme mot.
	navpointOverrideFlagsBits = 5

	// niveauPositionOffset : `i18`, l immediat que `FUN_140f04f68` passe a `FUN_14076e524`
	// (`MOV R9D, 0x10` en `140f04f80`).
	niveauPositionOffset = 0x10
)

// consumeNavpointOverrideFlags (ti=12 i16) — `FUN_140ebf834` : `R(5)` plat, sans porte.
func consumeNavpointOverrideFlags(br *Lecteur) { br.ReadBits(navpointOverrideFlagsBits) }

// consumeNavpointPositionOffset (ti=12 i18) — `FUN_140f04f68` : la garde de pleine precision
// (`FUN_14076f91c`, aucun bit lu) choisit la position brute (`FUN_1411b259c`, `R(96)`) ou la position
// quantifiee (`FUN_14076e524` au niveau `0x10`, `CALL 140f04f8b`) : la forme de `FUN_14076e494`,
// portee par [lireE494]. La position est rangee a `etat + 0x714` ; aucun consommateur ne la lit.
func consumeNavpointPositionOffset(br *Lecteur) { lireE494(br, niveauPositionOffset) }
