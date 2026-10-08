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

	// `i20` a `i27` : huit descripteurs d une meme table (accesseur de nom `14064c620`, table
	// `143d07f00`, index en `descripteur + 8`), un seul lecteur ([consumeNavpointVisualStateGroup]).
	compNavpointVisualStateGroups0 = "managed-navpoint-visual-state-groups-component-0"
	compNavpointVisualStateGroups1 = "managed-navpoint-visual-state-groups-component-1"
	compNavpointVisualStateGroups2 = "managed-navpoint-visual-state-groups-component-2"
	compNavpointVisualStateGroups3 = "managed-navpoint-visual-state-groups-component-3"
	compNavpointVisualStateGroups4 = "managed-navpoint-visual-state-groups-component-4"
	compNavpointVisualStateGroups5 = "managed-navpoint-visual-state-groups-component-5"
	compNavpointVisualStateGroups6 = "managed-navpoint-visual-state-groups-component-6"
	compNavpointVisualStateGroups7 = "managed-navpoint-visual-state-groups-component-7"
)

// Largeurs du groupe d etats visuels (`i20` a `i27`), lues dans `FUN_140dbe1bc` et ses appeles.
const (
	// groupeEtatsVisuelsMotBits : les mots `R(32)` du groupe (`FUN_14080dec4`) : l identifiant
	// (`etat + 0x850` pour le groupe 0), le mot qui suit le jeu de filtres (`+ 0x108`) et le mot de
	// chaque filtre present (`+ 0x10c + 4 * i`).
	groupeEtatsVisuelsMotBits = 32
	// groupeEtatsVisuelsVersion : le `v` que `FUN_140dbe1bc` passe au bloc de filtres et a l ordre
	// (`MOV R8D, 1` en `140dbe1f4`, recopie dans `R9D` puis `R8D` par `FUN_140dbe218` et
	// `FUN_140dbe25c`) : drapeau d un bit, pas d octet legacy, ordre sur trois bits.
	groupeEtatsVisuelsVersion = true
)

// consumeNavpointVisualStateGroup (ti=12 `i20` a `i27`) — `FUN_140dbe1bc` :
//
//	R(1) presence (`FUN_1406cf008`) ; absent : le champ vaut -1, rien d autre n est lu
//	R(32) identifiant (`FUN_140dbe218` -> `FUN_14080dec4`, `etat + 0x728 + 0x130 * index + 0x128`)
//	`FUN_140dbe25c(groupe, flux, v = 1)` : bloc de filtres ([consumeFilterSet]) ; R(32) ;
//	  un R(32) par filtre present ; K entrees d ordre de trois bits ([consumeNavpointFilterOrder])
//
// L ecrivain (`142edb178`) ecrit la presence (`FUN_1406d49c4`, champ different de -1), le mot
// (`FUN_141d12268`), puis `FUN_142c94dd4` : le bloc de filtres (`FUN_142c7023c`), le mot, un mot
// par filtre present et les K entrees d ordre sur trois bits. Le seul echec est celui du bloc de
// filtres (tag 15, qui ne revient pas chez le jeu).
func consumeNavpointVisualStateGroup(br *Lecteur) bool {
	if !br.ReadBit() {
		return true
	}
	br.ReadBits(groupeEtatsVisuelsMotBits)
	mask, ok := consumeFilterSet(br, groupeEtatsVisuelsVersion)
	if !ok {
		return false
	}
	br.ReadBits(groupeEtatsVisuelsMotBits)
	k := navpointFilterCount(mask)
	for range k {
		br.ReadBits(groupeEtatsVisuelsMotBits)
	}
	consumeNavpointFilterOrder(br, k, groupeEtatsVisuelsVersion)
	return true
}

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

// compNavpointObjectMarker : l etiquette de registre de `ti=12 i17`.
const compNavpointObjectMarker = "managed-navpoint-object-marker"

// navpointObjectMarkerBits : `i17`, `FUN_141169e68` -> `FUN_14080dec4` (`+0x2c += 0x20`) vers
// `etat + 0x710` ; l ecrivain (`142edb084` -> `FUN_1407edaf4`) ecrit les 32 bits du meme mot.
const navpointObjectMarkerBits = 32

// consumeNavpointObjectMarker (ti=12 i17) — `FUN_141169e68` : `R(32)` plat, sans porte.
func consumeNavpointObjectMarker(br *Lecteur) { br.ReadBits(navpointObjectMarkerBits) }

// compNavpointTopProgress : l etiquette de registre de `ti=12 i13`.
const compNavpointTopProgress = "managed-navpoint-top-progress"

// navpointBarreBits : la largeur que `i13` (`FUN_142ed51d8`, vers `etat + 0x700`) passe a
// `FUN_1406d84b4` (`MOV dword [RSP+0x20], 8`), bornes `DAT_143cd84ec` / `DAT_143cd8374` ; l ecrivain
// (`142edb134` -> `FUN_142ed18e8`) quantifie la meme valeur sur huit bits.
const navpointBarreBits = 8

// consumeNavpointBarreDeProgression (ti=12 i13 et i15) — `R(8)` quantifie, sans porte ; la valeur n est pas
// publiee.
func consumeNavpointBarreDeProgression(br *Lecteur) { br.ReadBits(navpointBarreBits) }

// compNavpointBottomProgress : l etiquette de registre de `ti=12 i15`. Son lecteur (`FUN_142ed4fe4`,
// vers `etat + 0x708`) passe la meme largeur et les memes bornes a `FUN_1406d84b4` que celui de
// `i13` ; l ecrivain (`142edadf0` -> `FUN_142ed18e8`) aussi : [consumeNavpointBarreDeProgression].
const compNavpointBottomProgress = "managed-navpoint-bottom-progress"
