package filmdec

// components_managed_property.go — LES PROPRIETES RESEAU D'UN OBJET GERE : ti=13
// `managed-object-property-*`. Deserialiseurs et hook de publication.
//
// CE QUE ti=13 EST. Un sac de proprietes NOMMEES attachees a un objet gere par le mode. Le
// lecteur d'image-cle `FUN_140ce55e8` en donne la structure d'ensemble, et elle explique tout le
// decoupage de l'archetype :
//
//	R(1) [; si 1 -> R(8)]                          selecteur
//	FUN_14080dec4(rdr, "propertyName", etat)       R(32) : le NOM de la propriete   -> i0
//	R(1) : la propriete est-elle masquee par joueur ?
//	   si 0 -> UN SEUL variant                     la valeur scalaire               -> i1
//	   si 1 -> BOUCLE DE 32 variants               une valeur PAR JOUEUR            -> i2..i33
//
// LA GRAMMAIRE VIENT DU DESERIALISEUR DU JEU (lot C-bis phase 0,
// `.ai/V7.5/replay2d/registre_film/LOTCBIS_PHASE0.md` sections 2 a 4). Recette R7-d habituelle :
// chaine `.rdata` -> getter de nom -> case de vtable `+0x08` -> lecteur en `+0x30`. Les deux
// composants partagent leur coeur, `FUN_140ce59bc`, qui lit `R(4)` puis dispatche sur
// `FUN_140ce5aa4`.
//
// LE POINT QUI AVAIT FAIT ECHOUER LA PREMIERE LECTURE (lot C phase 1a : « variant a 11 branches
// plus conteneur de longueur variable », STOP). Le contexte passe au dispatcheur porte trois
// champs, et le troisieme est un INDEX DE CHAMP, pas un compteur de bits restants :
//
//	140ce555f: OR   dword ptr [RSP + 0x30],0xffffffff   ; i1      -> index = 0xFFFFFFFF
//	140ce5980: MOV  EAX,dword ptr [R9 + 0x8]            ; i2..i33 -> index = descripteur+8, 0..31
//
// Le dispatcheur teste `index < 0x20`, et les deux moities du variant sont DISJOINTES : les
// branches 1 a 6 ne lisent que si l'index est hors bornes (mode A, i1), les branches 7 a 15 ne
// lisent que s'il est dans les bornes (mode B, i2..i33). Il n'y a donc AUCUN conteneur de
// longueur variable : le tampon local de 136 octets de `FUN_140ce593c` est la copie SBO de la
// valeur composite, et chaque instance de composant y ecrit UN champ.
//
// POURQUOI `consumeByName` SUFFIT, alors qu'il ne recoit pas l'index du composant. La grammaire
// ne depend de l'index QUE par le choix de la moitie, et ce choix se deduit du NOM :
// `managed-object-property-component` est toujours le mode A, `managed-object-player-masked-
// property-component` toujours le mode B (ses 32 instances ont toutes un index dans [0, 0x20[).
// L'index de joueur lui-meme se reconstitue chez l'appelant depuis l'ORDRE DU MASQUE, ce que le
// scanner fait — le meme partage des roles que les quatre `rtpc` de ti=10.
//
// LES VALEURS SONT PUBLIEES BRUTES, comme au lot C : le hook rend le TAG et le QUANTUM, et les
// convertisseurs exportes ci-dessous portent la convention de dequantification RETENUE (milieu
// d'intervalle), qui reste une convention et non une mesure.

// Etiquettes de registre des composants portes par ce fichier.
const (
	compManagedObjectProperty             = "managed-object-property-component"
	compManagedObjectPlayerMaskedProperty = "managed-object-player-masked-property-component"
)

// Les douze alternatives du variant. Le tag est lu sur 4 bits, donc 0..15 ; le `switch` du jeu
// n'a que douze cas et tout tag superieur ou egal a 11 tombe sur le douzieme (`FUN_141fce2f0`).
//
// LA MOITIE BASSE ET LA MOITIE HAUTE SONT LE MEME JEU DE TYPES : 3 contre 7 (meme flottant
// quantifie), 5 contre 9 (meme identifiant de chaine), 2 contre 10 (meme booleen), 1 contre 11
// (meme enumere), 4 contre 8 (meme R(32)). Le tag dit le TYPE de la propriete ; le mode dit si
// l'on lit la valeur entiere ou l'element d'UN joueur.
const (
	ManagedPropertyTagVide      = 0  // aucun bit : la propriete n'a pas de valeur
	ManagedPropertyTagEnum      = 1  // mode A : R(4) puis -1 (FUN_1407ef804)
	ManagedPropertyTagBool      = 2  // mode A : R(1) (FUN_1406cf008)
	ManagedPropertyTagQuant     = 3  // mode A : R(24) quantifie sur [-100, +100] (FUN_1406d84b4)
	ManagedPropertyTagU32       = 4  // mode A : R(32) en ligne (FUN_140ce5720)
	ManagedPropertyTagStringID  = 5  // mode A : R(32) « string-id-value » (FUN_14080dec4)
	ManagedPropertyTagU32Bis    = 6  // mode A : R(32) (FUN_141d0f344)
	ManagedPropertyTagQuantJ    = 7  // mode B : R(24) quantifie sur [-100, +100] (FUN_142ee59e0)
	ManagedPropertyTagU32J      = 8  // mode B : R(32) (FUN_142ecf464)
	ManagedPropertyTagStringIDJ = 9  // mode B : R(32) « participant-string-id-value »
	ManagedPropertyTagBoolJ     = 10 // mode B : R(1) (FUN_1406cf008)
	ManagedPropertyTagEnumJ     = 11 // mode B : R(4) puis -1 ; les tags 12..15 y tombent aussi

	managedPropertyTagBits      = 4
	managedPropertyQuantBits    = 24
	managedPropertyPlayerCount  = 32
	managedPropertyIndexPremier = 2 // i2 porte l'element du joueur 0 : l'appelant retire ce biais
)

// managedPropertyPayloadBits rend la largeur de la charge utile d'une valeur de variant, TAG
// EXCLU, pour le mode demande. C'est la table de largeurs lue dans le desassemblage, et l'unique
// endroit ou elle est ecrite — le decodeur des vecteurs de test l'appelle aussi, de sorte que les
// vecteurs figes testent CETTE table et non une copie.
//
// AUCUNE DESYNCHRONISATION N'EST POSSIBLE : la largeur est entierement determinee par 4 bits lus
// dans le flux, chaque branche est integralement consommee, et les deux lecteurs du jeu finissent
// sur `MOV AL,0x1`. C'est pourquoi le statut de table est `porte` et non `partiel` — meme
// raisonnement qu'au lot C phase 1b pour les `rtpc`.
func managedPropertyPayloadBits(tag int, modeA bool) int {
	if modeA {
		switch tag {
		case ManagedPropertyTagEnum:
			return 4
		case ManagedPropertyTagBool:
			return 1
		case ManagedPropertyTagQuant:
			return managedPropertyQuantBits
		case ManagedPropertyTagU32, ManagedPropertyTagStringID, ManagedPropertyTagU32Bis:
			return 32
		}
		return 0 // tag 0 et tags 7..15 : la garde `index < 0x20` les rend muets en mode A
	}
	switch tag {
	case ManagedPropertyTagQuantJ:
		return managedPropertyQuantBits
	case ManagedPropertyTagU32J, ManagedPropertyTagStringIDJ:
		return 32
	case ManagedPropertyTagBoolJ:
		return 1
	}
	if tag >= ManagedPropertyTagEnumJ {
		return 4
	}
	return 0 // tags 0..6 : la meme garde les rend muets en mode B
}

// ManagedPropertyField designe le champ publie de ti=13. Enumeration STABLE et NOMMEE, jamais un
// index de registre : le lot 0 a mesure que le decoupage du registre CHANGE AVEC LE BUILD, donc
// un index de composant ne designe pas la meme chose d'un film a l'autre.
type ManagedPropertyField int

// Les champs publies, et leur compte.
const (
	ManagedPropertyScalar     ManagedPropertyField = iota // i1      : le variant en mode A
	ManagedPropertyPerPlayer                              // i2..i33 : le variant en mode B
	ManagedPropertyFieldCount = 2
)

// String rend l'etiquette de registre du champ.
func (f ManagedPropertyField) String() string {
	switch f {
	case ManagedPropertyScalar:
		return compManagedObjectProperty
	case ManagedPropertyPerPlayer:
		return compManagedObjectPlayerMaskedProperty
	}
	return champInconnu
}

// managedPropertyHook, si non nil, recoit chaque lecture d'un champ de ti=13.
//
// PAS DE `present` ICI : aucun des deux composants n'a de porte de tete — le tag EST la valeur de
// tete, et il est publie. Global de paquet : l'appelant detient `LockProcessDecode`.
//
// FORME DES VALEURS : `values[0]` est toujours le TAG ; `values[1]`, present seulement quand la
// branche lit, est le quantum BRUT. Une branche muette publie donc un seul element — et c'est une
// information, pas un manque : elle dit que la propriete existe et que ce record n'en porte pas
// la valeur.
var managedPropertyHook func(f ManagedPropertyField, values []uint64)

// SetManagedPropertyHook installe (ou retire, avec nil) la sonde des composants de ti=13.
//
// UN HOOK SEPARE de ceux de ti=10 et ti=12, pour la meme raison qui les separait entre eux : les
// archetypes sont distincts, leurs slots sont disjoints, et un consommateur qui suit l'etat d'une
// propriete n'ecoute pas les memes objets qu'un consommateur qui suit une bordure ou un marqueur.
func SetManagedPropertyHook(h func(f ManagedPropertyField, values []uint64)) {
	managedPropertyHook = h
}

func publishManagedProperty(f ManagedPropertyField, values ...uint64) {
	if managedPropertyHook != nil {
		managedPropertyHook(f, values)
	}
}

// consumeManagedPropertyVariant lit un variant dans le mode demande et publie (tag [, quantum]).
// C'est le coeur partage, image de `FUN_140ce59bc` puis `FUN_140ce5aa4`.
func consumeManagedPropertyVariant(br *BitReader, f ManagedPropertyField, modeA bool) {
	tag := br.ReadBits(managedPropertyTagBits)
	n := managedPropertyPayloadBits(int(tag), modeA)
	if n == 0 {
		publishManagedProperty(f, tag)
		return
	}
	publishManagedProperty(f, tag, br.ReadBits(uint(n)))
}

// consumeManagedObjectProperty (ti=13 i1) — lecteur `FUN_140ce5554` puis `FUN_140ce59bc`.
//
// MODE A. Le lecteur pose `index = 0xFFFFFFFF` avant de descendre dans le variant, donc les
// branches 1 a 6 lisent et les branches 7 a 15 ne consomment rien. Largeur totale, tag compris :
// 4 (tags 0 et 7..15), 5 (tag 2), 8 (tag 1), 28 (tag 3), 36 (tags 4, 5, 6).
//
// CE QUE LA MESURE DIT DEJA DE CE CANAL (phase 0, 5 films a objectif) : un slot porte UN SEUL
// type de propriete — la coherence du tag par slot vaut 99,9 a 100 % contre 15,9 a 51,7 % sur la
// bande fantome — et le type dominant est le tag 3, un flottant quantifie, a 85,7-95,4 %. Le
// record CHAINE (un en-tete de record valide commence exactement au bit de fin calcule) a 87,0 a
// 99,3 %, contre 94-97 % pour le meilleur temoin du corpus et 2-3 % pour la bande fantome : la
// largeur n'est pas une hypothese, le flux la confirme.
func consumeManagedObjectProperty(br *BitReader) {
	consumeManagedPropertyVariant(br, ManagedPropertyScalar, true)
}

// consumeManagedObjectPlayerMaskedProperty (ti=13 i2..i33) — lecteur `FUN_140ce593c` puis
// `FUN_140ce59bc`.
//
// MODE B. L'index vient du descripteur et vaut 0..31, donc les branches 7 a 15 lisent et les
// branches 0 a 6 ne consomment rien — l'exact miroir du mode A. Largeur totale, tag compris :
// 4 (tags 0..6), 5 (tag 10), 8 (tags 11..15), 28 (tag 7), 36 (tags 8, 9).
//
// LE MODE N'EST PAS DEDUIT DU BINAIRE SEUL, IL EST MESURE : la valeur que le registre pose dans
// le descripteur ne se lit pas dans le code (elle est ecrite a la construction de l'archetype).
// Le chainage a departage les deux hypotheses sur la donnee — 81 a 95 % sous le mode B contre
// 2 a 8 % sous le mode A, sur onze composants et deux films KOTH.
//
// RESERVE UTILE A L'APPELANT : en Strongholds, le trafic apparent de ces composants est de la
// CONTAMINATION d'ancrage (0 % de chainage, sous la bande fantome). Ils ne parlent reellement
// qu'en KOTH sur le corpus mesure.
func consumeManagedObjectPlayerMaskedProperty(br *BitReader) {
	consumeManagedPropertyVariant(br, ManagedPropertyPerPlayer, false)
}

// ManagedPropertyFilmIndex rend l'index de joueur porte par le composant d'index `i` du masque
// (i2 porte le joueur 0), tel que LE FILM numerote les joueurs. Rend -1 hors de la plage des 32
// instances declarees.
//
// LE NOM DIT SON STATUT. Il s'appelait `ManagedPropertyPlayerIndex` : cet index est un ORDRE
// interne au film, jamais une identite — la jointure entre joueurs passe par le XUID (garde-rail
// `archlint/no_player_index_identity_test.go`, precedent `FireEvent.FilmIndex`).
//
// POURQUOI CHEZ L'APPELANT ET NON DANS LE DESER : `consumeByName` ne recoit pas l'index du
// composant, et le jeu le lit dans le descripteur. L'appelant, lui, a le masque — c'est le meme
// partage des roles que pour les quatre `rtpc` de ti=10.
func ManagedPropertyFilmIndex(i int) int {
	n := i - managedPropertyIndexPremier
	if n < 0 || n >= managedPropertyPlayerCount {
		return -1
	}
	return n
}

// ManagedPropertyQuantValue dequantifie la valeur des tags 3 et 7 (24 bits) dans leur plage
// [-100.0, +100.0], constantes `0x143cd8f84` (-100.0f) et `0x143cd84a8` (+100.0f).
func ManagedPropertyQuantValue(q uint64) float32 {
	return dequantMidpoint(q, managedPropertyQuantBits, -100, 100)
}

// ManagedPropertyEnumValue rend la valeur d'un enumere (tags 1 et 11..15). `FUN_1407ef804` lit
// R(4) puis range `valeur - 1` dans un OCTET SIGNE : la valeur 0 du flux signifie donc
// « absent » et vaut -1, et non 255.
func ManagedPropertyEnumValue(q uint64) int8 { return int8(q) - 1 }
