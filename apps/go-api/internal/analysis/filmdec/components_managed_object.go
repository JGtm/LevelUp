package filmdec

// components_managed_object.go — LES OBJETS SCRIPTES DU MODE : ti=10 `managed-object-*` et
// ti=12 `managed-navpoint-*`. Deserialiseurs et hooks de publication.
//
// POURQUOI CE FICHIER EXISTE. La phase 0 du lot C a mesure que ces deux archetypes sont la
// machinerie vivante du mode : presents sur les 10 films a objectif du corpus, ABSENTS du temoin
// Slayer pour ti=10, et deux de leurs composants portent a eux seuls l'essentiel du trafic de
// leur archetype (`managed-navpoint-radial-progress` : 74,7 % et 93,1 % des records ti=12 des
// deux films Strongholds ; `managed-object-rtpc-component` : 17 a 53 % des records ti=10).
// La phase 1a a lu leur grammaire dans le binaire, adresse par adresse
// (`.ai/V7.5/replay2d/registre_film/LOTC_PHASE1A.md` sections 3 et 4).
//
// LA GRAMMAIRE VIENT DU DESERIALISEUR DU JEU, PAS D'UN AJUSTEMENT. Chaque largeur ci-dessous est
// un immediat lu dans le desassemblage (par exemple `MOV dword ptr [RSP + 0x20],0x8` a
// `140fc8d3b` pour la largeur de `radial-progress`), et la recette qui mene du nom de composant
// au deserialiseur est celle de R7-d : chaine `.rdata` -> getter de nom -> case de vtable
// `+0x08` -> lecteur en `+0x30`. Le thunk `+0x28` (`FUN_14076ce9c`) est un PUR forwarder vers
// `+0x30` : il ne consomme aucun bit, donc la charge utile d'un composant commence exactement la
// ou le masque finit — c'est ce qui rend les vecteurs de test de la phase 1a exploitables.
//
// LES VALEURS SONT PUBLIEES BRUTES. Les hooks rendent le QUANTUM, jamais un flottant : la
// convention de dequantification du jeu (milieu d'intervalle contre bornes incluses) n'est PAS
// etablie — `FUN_1406d84b4` rend son resultat en XMM0 et la decompilation ne montre pas le
// calcul. L'ecart vaut un demi-quantum. Les convertisseurs ci-dessous portent la convention
// RETENUE, documentee et testable, et l'appelant reste libre de lire le quantum.

// Etiquettes de registre des composants portes par ce fichier. Des constantes, pas des
// litteraux repetes : elles servent au routage de `consumeByName`, au `String()` des champs et
// aux garde-rails.
const (
	compManagedObjectBoundaryVisibility = "managed-object-boundary-visibility-component"
	compManagedObjectBoundaryColor      = "managed-object-boundary-color-component"
	compManagedObjectRTPC               = "managed-object-rtpc-component"
	compNavpointRadialProgress          = "managed-navpoint-radial-progress"
)

// -----------------------------------------------------------------------------------------
// ti=10 — L OBJET SCRIPTE DU MODE
// -----------------------------------------------------------------------------------------

// ManagedObjectField designe le champ publie de ti=10. Enumeration STABLE et NOMMEE, jamais un
// index de registre : le lot 0 a mesure que le decoupage du registre CHANGE AVEC LE BUILD
// (`06dfe6d9` : 116 blocs / 1 031 slots contre 118 / 1 067 ailleurs), donc un index de composant
// ne designe pas la meme chose d'un film a l'autre.
type ManagedObjectField int

// Les champs publies, et leur compte.
const (
	ManagedObjectBoundaryVisibility ManagedObjectField = iota // i0  : 32 drapeaux plats
	ManagedObjectBoundaryColor                                // i1  : 4 x R(8) -> RGBA
	ManagedObjectRTPC                                         // i26..i29 : R(32) id [+ R(22)]
	ManagedObjectFieldCount         = 3
)

// String rend l'etiquette de registre du champ.
func (f ManagedObjectField) String() string {
	switch f {
	case ManagedObjectBoundaryVisibility:
		return compManagedObjectBoundaryVisibility
	case ManagedObjectBoundaryColor:
		return compManagedObjectBoundaryColor
	case ManagedObjectRTPC:
		return compManagedObjectRTPC
	}
	return "champ inconnu"
}

// managedObjectHook, si non nil, recoit chaque lecture d'un champ de ti=10.
//
// PAS DE `present` ICI : aucun de ces composants n'a de porte de tete. Global de paquet :
// l'appelant detient `LockProcessDecode`.
var managedObjectHook func(f ManagedObjectField, values []uint64)

// SetManagedObjectHook installe (ou retire, avec nil) la sonde des composants de ti=10.
func SetManagedObjectHook(h func(f ManagedObjectField, values []uint64)) { managedObjectHook = h }

func publishManagedObject(f ManagedObjectField, values ...uint64) {
	if managedObjectHook != nil {
		managedObjectHook(f, values)
	}
}

// consumeManagedObjectBoundaryColor (ti=10 i1) — lecteur `FUN_142ed52b4`.
//
// Quatre lectures de 8 bits, INCONDITIONNELLES, chacune dequantifiee dans [0.0, +1.0] et rangee
// a `etat+0x04`, `+0x08`, `+0x0c`, `+0x10` : c'est une couleur RGBA. Les quatre largeurs sont
// quatre immediats explicites (`142ed52dd`, `142ed52f6`, `142ed5315`, `142ed5334`) ; le minimum
// vient d'un `XORPS XMM2,XMM2` (donc 0.0) et le maximum de la constante `0x143cd8374` (+1.0f),
// tous deux charges UNE fois et jamais recharges entre les appels.
//
// Largeur totale : 32 bits. Aucune porte, aucune dependance a un etat runtime.
func consumeManagedObjectBoundaryColor(br *BitReader) {
	r := br.ReadBits(8)
	g := br.ReadBits(8)
	b := br.ReadBits(8)
	a := br.ReadBits(8)
	publishManagedObject(ManagedObjectBoundaryColor, r, g, b, a)
}

// consumeManagedObjectRTPC (ti=10 i26..i29) — lecteur `FUN_140796d38`.
//
// `R(32)` identifiant, puis — SEULEMENT si l'identifiant est non nul — `R(22)` dequantifie dans
// [-10000.0, +10000.0]. Quand l'identifiant est nul, le jeu ecrit une valeur sentinelle
// (`0x800000`) SANS lire un bit. La largeur du record depend donc de la DONNEE : 32 ou 54 bits.
//
// LE STATUT RESTE `porte` ET CE N'EST PAS UNE FACILITE : le branchement porte sur une valeur
// LUE DANS LE FLUX, les deux branches sont integralement consommees, et le cas rend toujours
// `true` — aucun desync n'est possible. `partiel` est reserve aux cas ou la traversee peut
// lacher (retour data-dependant), ce qui n'est pas le cas ici.
//
// LES QUATRE COMPOSANTS (i26..i29) PARTAGENT CE LECTEUR, et `consumeByName` ne recoit PAS
// l'index du composant — le jeu, lui, le lit dans le descripteur (`param_1 + 8`) pour choisir sa
// case d'etat (`etat + 0x17c + index*8`). Ce n'est pas une perte : l'IDENTIFIANT de 32 bits est
// la vraie identite du canal, il est constant par composant et IDENTIQUE d'un film a l'autre
// (`0x06854540` et `0x7CBF0066` mesures sur les deux films Strongholds, phase 1a section 5).
// Publier l'identifiant plutot qu'un index de registre est donc a la fois plus juste et conforme
// a la regle « enumeration nommee, jamais un index de registre ».
func consumeManagedObjectRTPC(br *BitReader) {
	id := br.ReadBits(32)
	if id == 0 {
		publishManagedObject(ManagedObjectRTPC, id)
		return
	}
	publishManagedObject(ManagedObjectRTPC, id, br.ReadBits(22))
}

// ManagedObjectRTPCValue dequantifie la valeur d'un canal RTPC (22 bits) dans sa plage
// [-10000.0, +10000.0], constantes `0x143cd8774` / `0x143cd8770`.
func ManagedObjectRTPCValue(q uint64) float32 { return dequantMidpoint(q, 22, -10000, 10000) }

// ManagedObjectBoundaryColorValue dequantifie une composante de couleur (8 bits) dans [0, 1].
func ManagedObjectBoundaryColorValue(q uint64) float32 { return dequantMidpoint(q, 8, 0, 1) }

// -----------------------------------------------------------------------------------------
// ti=12 — LE POINT DE NAVIGATION DU MODE
// -----------------------------------------------------------------------------------------

// NavpointField designe le champ publie de ti=12. Meme regle que `ManagedObjectField` :
// enumeration nommee, jamais un index de registre.
type NavpointField int

// Le champ publie, et son compte. Les 26 autres composants de l'archetype restent `non_porte`.
const (
	NavpointRadialProgress NavpointField = iota // i14 : R(8) -> progression radiale
	NavpointFieldCount                   = 1
)

// String rend l'etiquette de registre du champ.
func (f NavpointField) String() string {
	if f == NavpointRadialProgress {
		return compNavpointRadialProgress
	}
	return "champ inconnu"
}

// navpointHook, si non nil, recoit chaque lecture d'un champ de ti=12. Pas de `present` : le
// composant n'a pas de porte de tete. Global de paquet : l'appelant detient `LockProcessDecode`.
var navpointHook func(f NavpointField, values []uint64)

// SetNavpointHook installe (ou retire, avec nil) la sonde des composants de ti=12.
//
// UN HOOK SEPARE DE CELUI DE ti=10, et non un champ de plus dans `ManagedObjectField` : les deux
// archetypes sont distincts (`managed-object-*` contre `managed-navpoint-*`), leurs slots sont
// disjoints, et un consommateur qui suit une zone n'ecoute pas les memes objets qu'un
// consommateur qui suit un marqueur. C'est le modele deja etabli par le paquet — un hook nomme
// par famille d'archetype (`GameEngineField`, `EquipmentField`, `ManagedObjectField`).
func SetNavpointHook(h func(f NavpointField, values []uint64)) { navpointHook = h }

func publishNavpoint(f NavpointField, values ...uint64) {
	if navpointHook != nil {
		navpointHook(f, values)
	}
}

// consumeNavpointRadialProgress (ti=12 i14) — lecteur `FUN_140fc8d14`.
//
// UNE lecture de 8 bits, inconditionnelle, dequantifiee dans [-1.0, +1.0] et rangee a
// `etat+0x704`. Le lecteur tient en dix instructions et sa largeur est un immediat
// (`MOV dword ptr [RSP + 0x20],0x8` a `140fc8d3b`) ; il finit sur `MOV AL,0x1`, donc il rend
// toujours vrai.
//
// CE QUE LA MESURE DIT DEJA DE CE CANAL (phase 1a section 5, 17 244 et 17 612 records a masque
// singleton) : la distribution est lisse — 159 et 128 valeurs distinctes, aucune au-dela de
// 1,2 % — et centree sur le quantum 128, qui est le ZERO de la plage [-1, +1] ; les valeurs
// dominantes sont espacees d'environ 3. C'est une RAMPE, pas un enumere.
func consumeNavpointRadialProgress(br *BitReader) {
	publishNavpoint(NavpointRadialProgress, br.ReadBits(8))
}

// NavpointRadialProgressValue dequantifie la progression radiale (8 bits) dans sa plage
// [-1.0, +1.0], constantes `0x143cd84ec` / `0x143cd8374`.
func NavpointRadialProgressValue(q uint64) float32 { return dequantMidpoint(q, 8, -1, 1) }

// dequantMidpoint rend la valeur d'un quantum de `bits` bits dans [min, max], par la convention
// du MILIEU D'INTERVALLE — celle que le paquet emploie deja pour les positions d'objet du monde
// (`decodeWorldObjectPos`, projectiles.go). C'est une convention RETENUE, pas une convention
// mesuree : voir l'en-tete de ce fichier.
func dequantMidpoint(q uint64, bits uint, min, max float32) float32 {
	steps := float32(uint64(1) << bits)
	return min + (float32(q)+0.5)*(max-min)/steps
}
