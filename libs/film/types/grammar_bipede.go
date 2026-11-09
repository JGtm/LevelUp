package types

// grammar_bipede.go — LES TYPES DE CONTRAT DU BIPEDE ET DE SON INVENTAIRE (couche `grammar`).
//
// Deplaces depuis `film/internal/grammar` au lot 2.6.2 (volet grammaire / rejeu) SANS
// REECRITURE : les commentaires sont ceux des declarations d origine, et un renvoi de godoc y
// designe encore un symbole de la couche qui produit.

// BipedCreationStats compte ce que le balayage a rencontré. Sans ces dénominateurs, une
// couverture ne se juge pas — et sans le détail des rejets, on ne sait pas si le balayage rate
// des records ou en invente.
type BipedCreationStats struct {
	// Slots est le nombre de slots de la bande passée au balayage.
	Slots int
	// Anchors est le nombre d'en-têtes NEW `ti=35` reconnus (trois constantes de format + la
	// bande de slots). C'est le DÉNOMINATEUR du gate, pas un nombre de records.
	Anchors int
	// Truncated : le prologue déborde du payload — le curseur n'est plus digne de confiance.
	Truncated int
	// ShapeBad : l'ancre n'a pas la forme d'une création de bipède (porte de version fermée,
	// version ≠ 13, ou porte de représentation fermée). Le rejet ORDINAIRE de l'ancrage bit à
	// bit ; il n'alarme rien.
	ShapeBad int
	// SignatureMismatch : l'ancre a toute la FORME d'une création de bipède, mais son mot de
	// 32 bits n'est pas [BipedRepresentationName]. C'est LE compteur à surveiller — cf.
	// l'en-tête du fichier pour son plancher de bruit et pour l'alarme que l'appelant arme.
	SignatureMismatch int
	// OtherWord est le mot de 32 bits alternatif le plus fréquent parmi les
	// `SignatureMismatch`, et OtherWordCount son compte. Zéro quand il n'y en a aucun.
	OtherWord      uint32
	OtherWordCount int
	// GateClosed : signature reconnue, mais la porte INVERSÉE de l'index s'est FERMÉE — le
	// record ne porte pas d'index. Mesuré à ZÉRO sur 529 records ; s'il devient non nul, c'est
	// un fait à instruire, pas un défaut de lecture.
	GateClosed int
	// Accepted est le nombre de records rendus (signature reconnue ET index transmis).
	Accepted int
}

// BipedPickup est UN ramassage, daté, attribué et nommé.
type BipedPickup struct {
	// TimestampUS est l'horodatage du paquet — MÊME horloge que BipedPosition.TimestampUS.
	TimestampUS uint64
	// Chunk localise l'événement dans le film.
	Chunk int
	// Slot est le slot du bipède RAMASSEUR : il désigne une VIE, pas un joueur. Même espace
	// de slots que HeldWeaponChange.Slot — c'est celui que l'assemblage relie au joueur.
	Slot uint32
	// CatalogID est l'identifiant de CATALOGUE de l'objet ramassé (le R(32) de la charge).
	// Ce n'est PAS un handle du monde : la même valeur se retrouve d'un match à l'autre.
	// Pour les armes, il vaut la FAMILLE d'arme telle que HeldWeaponChange.Family la publie —
	// mesuré : 100 % des familles vues par i43..i46 sont dans l'ensemble des CatalogID.
	CatalogID uint32
	// Class est le R(3) de tête de charge. Il sépare les ramassages d'ARME des autres : les
	// classes 0 et 1 portent une famille d'arme d'i43..i46 dans 63 à 72 % des cas, les classes
	// 2 et 3 dans 0,0 % — sur 118 événements de deux films. Voir BipedPickupIsWeaponClass.
	Class uint8
}

// BipedPickupStats dit ce que le balayage a vu ET ce qu'il a REFUSÉ. Le second compte autant :
// la largeur d'index du domaine 2 est une valeur de runtime, et un film qui la porterait
// différente produirait des slots hors de la bande de bipèdes. Ces rejets sont la sentinelle.
type BipedPickupStats struct {
	// Packets est le nombre de paquets delta dont l'octet de tête vaut 0xC4.
	Packets int
	// Type9 / Type8 / OtherType ventilent le type lu en tête de liste.
	Type9, Type8, OtherType int
	// Published est le nombre de ramassages rendus.
	Published int
	// MultiEvent compte les listes qui portent un AUTRE événement après le type 9. Il mesure
	// ce que ce balayage ne peut pas voir : un type 9 en deuxième position d'une liste ouverte
	// par une autre famille lui échappe entièrement.
	MultiEvent int
	// RefusedNoRef / RefusedNoCatalog / RefusedOffBand comptent les rejets. Aucun des trois
	// n'a jamais été observé non nul sur le corpus de référence — une valeur non nulle est un
	// signal, pas un détail.
	RefusedNoRef, RefusedNoCatalog, RefusedOffBand int
	// UnexpectedWideRef compte les événements dont ref1 ou ref2 est présente. Jamais observé :
	// une valeur non nulle dénonce un cadrage faux ou un build différent.
	UnexpectedWideRef int
}

// CamoRead est UNE transmission de la voie d'état du camouflage, localisée dans le film.
type CamoRead struct {
	// Slot est l'identifiant bas du biped porteur — le même que celui des trajectoires,
	// donc UNE VIE et non un joueur (le slot migre aux réapparitions).
	Slot uint32
	// Chunk / PacketIndex localisent la lecture dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet porteur — MÊME horloge que BipedPosition.
	TimestampUS uint64
	// Q est le quantum brut R(12) de queue[1]. Binaire mesuré : CamoInactiveQ ou
	// CamoActiveQ.
	Q uint16
}

// GrappleRead est UNE lecture d'événement de grappin, localisée dans le film.
type GrappleRead struct {
	// Slot est l'identifiant bas du biped porteur — le même que celui des trajectoires,
	// donc UNE VIE et non un joueur (le slot migre aux réapparitions).
	Slot uint32
	// Chunk / PacketIndex localisent la lecture dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet porteur — MÊME horloge que BipedPosition.
	TimestampUS uint64
	// Heavy dit si la lecture est le corps LOURD (l'accroche, second membre de la paire).
	// Le corps léger est le tir.
	Heavy bool
	// PosQ : les trois quanta de la position de l'ancre, aux largeurs d'axe de la carte
	// installées au moment du balayage (WorldObjectPrecision.AxisW).
	PosQ [3]uint32
}

// HeldWeaponChangeKind qualifie un changement d'arme en main.
type HeldWeaponChangeKind string

const (
	// HeldWeaponTaken : l'emplacement était vide (ou l'arme absente du loadout de spawn) et
	// porte désormais une arme.
	HeldWeaponTaken HeldWeaponChangeKind = "taken"
	// HeldWeaponDropped : l'emplacement passe à vide. C'est le cas NON AMBIGU.
	HeldWeaponDropped HeldWeaponChangeKind = "dropped"
	// HeldWeaponSwapped : l'emplacement passe d'une arme à une autre.
	HeldWeaponSwapped HeldWeaponChangeKind = "swapped"
	// HeldWeaponRestated : l'arme était déjà portée au spawn ; le flux ne fait que la
	// ré-annoncer (changement d'emplacement). Ce n'est PAS un ramassage, et le distinguer
	// est ce qui empêche de compter des prises qui n'ont pas eu lieu.
	HeldWeaponRestated HeldWeaponChangeKind = "restated"
)

// HeldWeaponChange est UN changement d'arme en main, daté et attribué.
type HeldWeaponChange struct {
	// TimestampUS est l'horodatage du paquet — MÊME horloge que BipedPosition.TimestampUS.
	TimestampUS uint64
	// Chunk localise l'événement dans le film.
	Chunk int
	// Slot est le slot du bipède porteur : il désigne une VIE, pas un joueur.
	Slot uint32
	// SlotIndex est l'emplacement d'arme concerné (l'index du composant dans le masque).
	SlotIndex int
	// Emplacement est le RANG de cet emplacement parmi les composants `weapon-state-type-info`
	// de l'archétype du film (0 = le premier, i43 sur les films mesurés) — lot M3.2. C'est la
	// clé qu'une dotation de naissance partage avec le flux delta ; elle vient des NOMS du
	// registre, jamais d'un index de composant en dur.
	Emplacement int
	// Family est la moitié HAUTE de l'identifiant 64 bits : l'identité de l'arme, celle que
	// le catalogue nomme. `noVariant` quand l'emplacement devient vide.
	//
	// C'EST LA MOITIÉ HAUTE ET PAS LA BASSE, et le point a été payé : le déserialiseur lit
	// deux R(32) et le port ne rendait que le second, qui ne résout RIEN au catalogue (cinq
	// valeurs distinctes sur trente et une émissions, dont un suffixe partagé).
	Family uint32
	// Low est la moitié basse (la variante cosmétique), gardée pour le diagnostic.
	Low uint32
	// Previous est la famille précédente sur cet emplacement, quand elle est connue.
	Previous uint32
	// Kind qualifie le changement.
	Kind HeldWeaponChangeKind
}

// InventoryDelta est UNE transmission d'inventaire de grenades, localisée dans le film.
type InventoryDelta struct {
	// Slot est l'identifiant bas du biped porteur — le même que celui des trajectoires, donc
	// UNE VIE et non un joueur (le slot migre aux réapparitions).
	Slot uint32
	// Chunk / PacketIndex localisent la lecture dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet porteur — MÊME horloge que BipedPosition.
	TimestampUS uint64
	// Grenades porte le compteur de chaque rang (i22). NIL = i22 n'était pas au masque de ce
	// record, ou sa lecture violait une borne : jamais « zéro grenade ».
	Grenades []uint32
	// Mask est le masque R(6) des types portés (i47), valide seulement si SelRead.
	Mask uint32
	// Sel est le rang SÉLECTIONNÉ en base 0, ou InventoryDeltaNoSel si i47 n'en désigne
	// aucun. Valide seulement si SelRead.
	Sel int
	// SelRead dit si i47 a été lu sur ce record. Faux = non transmis ou non atteint.
	SelRead bool
	// Ammo porte l'état de munitions des emplacements que CE record transmet — jamais les
	// quatre par défaut. Vide = aucun composant de munitions au masque.
	Ammo []InventoryDeltaAmmo
}

// InventoryDeltaAmmo est l'état de munitions d'UN emplacement d'arme, tel qu'un paquet delta
// le transmet. Les trois grandeurs sont indépendamment optionnelles : elles viennent de DEUX
// composants distincts, qu'un record peut annoncer séparément.
type InventoryDeltaAmmo struct {
	// WeaponSlot est le rang de l'emplacement dans l'archétype (0 et 1 portent une arme).
	WeaponSlot int
	// Mag est le chargeur, ou nil si le film n'écrit rien pour ce champ.
	Mag *uint32
	// FracQ est le quantum R(12) BRUT de la fraction, ou nil. Brut, parce que la
	// déquantification appartient à la couche qui sait ce qu'elle affiche.
	//
	// ELLE COMPTE CE QUI A ÉTÉ CONSOMMÉ, pas ce qui reste — deux témoins concordants, cf.
	// AmmoSlot.Gauge (replay/inventory.go). Un client qui dessine une charge RESTANTE doit
	// donc afficher le complément.
	FracQ *uint32
	// Res est la réserve (R(11)), ou nil si `weapon-state-rounds-inventory` n'était pas au
	// masque de ce record.
	Res *uint32
}

// KeyframeLoadout est l'ensemble des identifiants de FAMILLE d'arme trouvés dans le record
// biped d'un slot, à l'instant d'un keyframe.
type KeyframeLoadout struct {
	// TimestampUS est l'horodatage du paquet keyframe — MÊME horloge que
	// BipedPosition.TimestampUS et FireEvent.TimestampUS.
	TimestampUS uint64
	// Chunk / PacketIndex localisent le keyframe dans le film.
	Chunk, PacketIndex int
	// Slot est le slot du biped porteur (celui des trajectoires).
	Slot uint32
	// Families liste les familles (high-32 du weapon-id) dans l'ORDRE DES BITS du record.
	// Les alias ne sont PAS repliés ici : deux familles distinctes peuvent désigner le même
	// canon. Le repli est une question de NOMMAGE, il appartient à la couche qui possède le
	// catalogue d'armes (cf. replay/loadouts.go).
	Families []uint32
}

// BirthWeapon est UN emplacement d'arme lu dans le record NEW de naissance d'un bipède.
type BirthWeapon struct {
	// Emplacement est le rang de l'emplacement parmi les composants `weapon-state-type-info`
	// de l'archétype (même clé que [HeldWeaponChange.Emplacement]).
	Emplacement int
	// Family est la moitié HAUTE de l'identifiant (l'arme) ; la sentinelle d'emplacement vide
	// quand la porte de présence est fermée.
	Family uint32
	// Low est la moitié basse (la variante cosmétique), gardée pour le diagnostic.
	Low uint32
}

// BirthLoadout est la DOTATION DE NAISSANCE d'une vie : les emplacements d'arme que le record
// NEW du bipède transmet à sa création (lot M3.2). Elle ne se lit que quand le record entier se
// FERME — traversé sans désynchronisation, et suivi d'un record confirmé (cf.
// `grammar/birth_loadouts.go`) ; sinon rien n'est rendu et le refus est compté.
type BirthLoadout struct {
	// TimestampUS est l'horodatage du paquet porteur — MÊME horloge que les autres lectures.
	TimestampUS uint64
	// Chunk / PacketIndex localisent le record dans le film.
	Chunk, PacketIndex int
	// Slot et Generation désignent LA VIE (même clé que la création de bipède).
	Slot, Generation uint32
	// Weapons liste les emplacements annoncés au masque du record, dans l'ordre des composants.
	Weapons []BirthWeapon
}

// BirthLoadoutStats compte ce que la lecture des dotations de naissance a vu et refusé.
type BirthLoadoutStats struct {
	// Creations est le nombre de créations de bipède soumises à la lecture.
	Creations int
	// Read est le nombre de dotations rendues (record fermé).
	Read int
	// Desync : la traversée s'est arrêtée sur un composant sans lecteur porté.
	Desync int
	// Overflow : la traversée a dépassé la fin du payload.
	Overflow int
	// Unconfirmed : le record se traverse, mais ce qui le suit n'est ni un delta propre sur un
	// slot lié, ni un record NEW dont le monde ou une image-clé ultérieure confirme l'archétype.
	Unconfirmed int
	// NoWeaponComponent : record fermé dont le masque n'annonce aucun emplacement d'arme.
	NoWeaponComponent int
	// ClosedByDelta / ClosedByBoundNew / ClosedByAnticipatedNew ventilent les fermetures par la
	// nature du record suivant.
	ClosedByDelta, ClosedByBoundNew, ClosedByAnticipatedNew int
}

// PlayerSlot : un slot OCCUPE de la table.
type PlayerSlot struct {
	// FilmIndex : le RANG du slot dans la table de 32, VACANTS COMPRIS. C'est l'index du
	// tableau que l'ecrivain parcourt (`enregistrement += 0x1450`), donc l'index du slot.
	//
	// CE QUE LA MESURE DIT, ET CE QU'ELLE NE DIT PAS. L'ordre des enregistrements EST le
	// `player_index` de production : `filmIndex - rang` est CONSTANT sur 76 films sur 76, et la
	// coincidence est totale sur 72 (2026-09-12, oracle = le `roster[].filmIndex` des documents
	// de rejeu). Mais aucun de ces 76 films ne porte de slot vacant INTERCALE, donc l'oracle ne
	// separe pas « rang absolu » de « index parmi les occupes » : les deux definitions y
	// coincident. Les 5 films du cache a vacant intercale (`07f6af1b`, `0d1dddfb`, `1c5c10cc`,
	// `b1bcbe24`, `c744aa29`, mesures le 2026-09-14) n'ont AUCUN document de rejeu — la question
	// n'est donc pas tranchable sur ce corpus, et le rapport porte `InterleavedVacant` pour que
	// le consommateur sache quand les deux lectures divergent.
	FilmIndex int
	XUID      uint64
	Gamertag  string
	// SessionToken : le champ de 48 bits de `slot+0x09`. Propre au couple (match, joueur) :
	// cinq valeurs differentes a forte entropie pour un meme XUID sur cinq films. Lecture la
	// plus economique : un jeton de session. NON PROUVE.
	SessionToken uint64
	// Bit / TotalBits : la position et la longueur de l'enregistrement dans le flux, pour qu'un
	// rapport ou un instrument puisse revenir dessus sans re-chercher.
	Bit       int
	TotalBits int
	Shorts    PlayerSlotShorts
}

// PlayerSlotShorts : les champs COURTS d'un enregistrement de slot.
//
// Ils sont publies parce qu'ils sont lus, et parce qu'un champ mesure CONSTANT qui se met a
// varier est le premier signe qu'une grammaire a bouge. Mesure du 2026-09-12 sur les 44
// enregistrements de six films : huit d'entre eux sont constants sur tout le corpus (`Deux`=0,
// `F10`=183, `F14`=0, `F6`=-1, `F8`=0, `F7`=0, `Repr`=0, `Tete`=0) ; le neuvieme, `F1`, prend
// deux valeurs et ce sont exactement les enregistrements a listes vides contre ceux a listes
// pleines — un drapeau de presence des listes. AUCUN ne porte l'equipe : c'est mesure, et ferme
// par la negative (aucun ne partage un roster en deux moities egales, sur 0/6 films).
type PlayerSlotShorts struct {
	Tete uint32 // slot+0x04, 32 bits
	Deux uint32 // slot+0x08, 2 bits (octet signe : domaine -2..1)
	Repr uint32 // sub+0xcb0, 32 bits, etiquete `desired-representation`
	Q64  uint64 // sub+0xcb8, 64 bits
	F10  uint32 // sub+0xc12, 10 bits
	F14  uint32 // sub+0xc36, 14 bits
	F6   int    // sub+0xc35, 6 bits, ecrit VALEUR+1 : rendu signe (-1 sur un slot occupe)
	F8   uint32 // sub+0xc10, 8 bits
	F7   uint32 // sub+0xc34, 7 bits
	F1   uint32 // sub+0xc11 & 1, 1 bit
}
