package types

// grammar_equipement.go — LES TYPES DE CONTRAT DE L EQUIPEMENT (couche `grammar`).
//
// Deplaces depuis `film/internal/grammar` au lot 2.6.2 (volet grammaire / rejeu) SANS
// REECRITURE : les commentaires sont ceux des declarations d origine, et un renvoi de godoc y
// designe encore un symbole de la couche qui produit.

// MPPFieldCount est le nombre de champs du bloc `object-multiplayer-properties`.
//
// LA DIMENSION FAIT PARTIE DE LA FORME : `EquipmentCreation` porte deux tableaux de cette
// longueur, donc un consommateur d une autre couche la voit. Elle vit ici pour cette raison, et
// l enumeration `grammar.MPPField` — qui porte une methode, donc une regle — la REPREND
// (`MPPFieldCount = types.MPPFieldCount`) : une seule source, pas deux litteraux.
const MPPFieldCount = 4

// EquipmentChangeKind qualifie un changement d'équipement porté.
type EquipmentChangeKind string

const (
	// EquipmentTaken : le joueur porte désormais cet équipement. C'est un RAMASSAGE.
	EquipmentTaken EquipmentChangeKind = "taken"
	// EquipmentSpent : le joueur n'en porte plus. Il l'a CONSOMMÉ (mesuré : jamais à la mort).
	EquipmentSpent EquipmentChangeKind = "spent"
	// EquipmentSpawned : première émission d'une vie, contemporaine de la naissance du
	// bipède — le joueur RÉAPPARAÎT avec cet équipement. Ce n'est PAS un ramassage.
	EquipmentSpawned EquipmentChangeKind = "spawned"
)

// EquipmentChange est UN changement d'équipement porté, daté et attribué.
type EquipmentChange struct {
	// TimestampUS est l'horodatage du paquet — MÊME horloge que BipedPosition.TimestampUS.
	TimestampUS uint64
	// Chunk / PacketIndex localisent l'événement dans le film.
	Chunk, PacketIndex int
	// Slot est le slot du bipède : il désigne une VIE, pas un joueur.
	Slot uint32
	// Counter est le compteur de rotation R(3). Il est conservé parce qu'il est LE témoin de
	// complétude : un pas différent de 1 (modulo 8) entre deux émissions d'une même vie
	// dénonce des émissions manquées.
	Counter uint32
	// Rank est le rang de palette de l'équipement désormais porté, dans la même convention
	// qu'AbilityRank.Rank. Vaut AbilitySetNoRank sur EquipmentSpent.
	Rank int
	// Previous est le rang précédent sur cette vie, quand il est connu. AbilitySetNoRank
	// sinon — y compris à la première émission, dont l'état d'avant n'est pas lisible.
	Previous int
	// Kind qualifie le changement.
	Kind EquipmentChangeKind
	// Recovered dit que cette émission vient de la RÉCUPÉRATION GATÉE (equipment_recovery.go)
	// et non du balayage strict : ses octets existent dans le film sous une forme que la
	// production rejette par construction, et son compteur comble exactement un saut annoncé.
	// La provenance reste dite — un `from` redevenu fiable grâce à elle n'est pas un `from`
	// lu par le chemin nominal.
	Recovered bool
	// Gap est le saut de compteur RÉSIDUEL constaté depuis l'émission précédente de la même
	// vie, APRÈS récupération : 0 = chaîne saine (pas de 1), n > 0 = n émissions manquent
	// encore juste avant celle-ci — son champ Previous n'est alors PAS une identité fiable.
	// La première émission d'une vie porte 0 (pas d'émission précédente) ; l'incomplétude de
	// tête se lit dans EquipmentChangeStats.LivesFirstOffSpec.
	Gap int
}

// EquipmentChangeStats dit ce que le balayage a vu ET ce qu'il a MANQUÉ. Le second est la
// raison d'être de la structure : c'est le seul canal du rejeu qui sache s'auto-mesurer.
type EquipmentChangeStats struct {
	// Walk porte les dénominateurs du balayage (records, lectures, portes ouvertes).
	Walk AbilityRankStats
	// Lives est le nombre de vies ayant émis au moins une fois.
	Lives int
	// Repeats compte les transitions dont le compteur ne bouge PAS. Une valeur non nulle
	// contredirait la propriété qui fonde ce fichier — le composant ne devrait entrer au
	// masque QUE sur changement.
	Repeats int
	// CounterJumps compte les transitions dont le compteur avance d'autre chose que 1
	// (modulo 8), et MissedEstimate le nombre d'émissions que ces sauts impliquent.
	CounterJumps, MissedEstimate int
	// LivesFirstOffSpec compte les vies dont la PREMIÈRE émission n'a pas le compteur
	// attendu : des émissions antérieures ont été manquées, ou le slot a été mal ancré.
	LivesFirstOffSpec int
	// Spawned / Taken / Spent ventilent les changements rendus.
	Spawned, Taken, Spent int
	// Recovered compte les émissions issues de la récupération gatée (equipment_recovery.go),
	// À PART des lues par le balayage strict. Les compteurs ci-dessus (CounterJumps,
	// MissedEstimate, LivesFirstOffSpec) décrivent la chaîne FINALE, récupération comprise :
	// ce qui reste manquant après elle — le témoin mesure ce qui est publié, pas un état
	// intermédiaire.
	Recovered int
}

// EquipmentCreation est UN record de création d'objet d'équipement, lu jusqu'à sa position.
type EquipmentCreation struct {
	// Slot et Gen identifient la vie de l'objet — LA PAIRE, comme partout ailleurs : le pool de
	// slots reboucle et la génération ne fait que 2 bits.
	Slot, Gen uint32
	// Chunk / PacketIndex / TimestampUS localisent la lecture (même horloge que BipedPosition).
	Chunk, PacketIndex int
	TimestampUS        uint64
	// BitPos est la position de l'en-tête dans le payload, en bits (traçabilité).
	BitPos int
	// HasRef / Ref : la référence d'entité 5 bits, si la porte l'a transmise.
	HasRef bool
	Ref    uint32
	// HasID / AbilityID : l'identifiant 32 bits « ability-enabled-id », si transmis.
	HasID     bool
	AbilityID uint32
	// MPP porte les quatre champs du bloc `object-multiplayer-properties` du MÊME record
	// (cf. MPPField). Ils sont là parce que le default-state de ti=37 les contient : chercher
	// l'identité de l'objet ailleurs que dans son propre record de création n'aurait pas de sens.
	MPPPresent [MPPFieldCount]bool
	MPPVal     [MPPFieldCount]uint64
	// X, Y, Z est la position du composant i0 du MÊME record : le lieu de la pose.
	X, Y, Z float32
	// Mask est la liste des index de composant du masque, strictement croissante.
	Mask []int
	// MaskFull dit que le masque était la branche PLEINE (R(64)) et non la branche éparse.
	MaskFull bool
	// MaskHasI0 dit que le masque annonçait i0 explicitement. Faux = i0 a été décodé quand
	// même, parce que le masque par défaut {i0} est OR'd en amont par le moteur
	// (vtable[0xa0], cf. consumeBipedDefaultStateMovement) : la position est alors le PREMIER
	// composant décodé sans figurer au masque du flux.
	MaskHasI0 bool
	// DefaultStateBits est le nombre de bits qu'a consommés consumeDefaultStateTI37 sur CE
	// record. Publié parce qu'un déserialiseur mal porté se voit à une largeur qui s'éparpille.
	DefaultStateBits int
	// HasAmmo / Ammo : les MUNITIONS de l'objet à sa naissance, lues dans le composant i20
	// `weapon-ammo-component` du MÊME record (arme au sol uniquement — cf. ground_weapon_ammo.go,
	// qui porte la mesure et la RÉSERVE DE LECTURE qui borne `HasAmmo`). Faux partout ailleurs :
	// aucun autre archétype de cette marche ne demande la lecture.
	HasAmmo bool
	Ammo    GroundWeaponAmmo
	// AfterBit est la position du premier bit après le composant i0 (traçabilité du balayage).
	AfterBit int
}

// EquipmentCreationStats compte ce que le balayage a rencontré. Sans ces dénominateurs, une
// distribution de valeurs ne se juge pas — et sans le détail des rejets, on ne sait pas si le
// balayage rate des records ou en invente.
type EquipmentCreationStats struct {
	// Slots est le nombre de slots de la bande passée au balayage.
	Slots int
	// Anchors est le nombre d'en-têtes NEW ti=37 reconnus (les quatre constantes + la bande).
	Anchors int
	// Overflow : le default-state déborde du payload — le curseur n'est plus digne de confiance.
	Overflow int
	// MaskBad : compte hors bornes, index non croissants, ou index >= nombre de composants.
	MaskBad int
	// PosBad : position rejetée (porte non nulle, ou quantum saturé — cf. decodeWorldObjectPos).
	PosBad int
	// Accepted est le nombre de records rendus.
	Accepted int
	// MaskSparse / MaskFull : branche du masque des records acceptés.
	MaskSparse, MaskFull int
	// NoI0 : records acceptés dont le masque du flux n'annonçait PAS i0 (position décodée par
	// le masque par défaut OR'd en amont). Comptés à part : ce sont les moins sûrs.
	NoI0 int
	// WithRef / WithID : records acceptés dont la porte a transmis la valeur du champ.
	WithRef, WithID int
	// WithAmmo : records acceptés dont les MUNITIONS ont pu être lues (composant i20 au masque
	// ET marche prouvée bit-exacte — cf. ground_weapon_ammo.go). L'écart avec `Accepted` n'est
	// pas une anomalie : il MESURE la réserve de lecture, et c'est lui qui doit tomber le jour
	// où le portage d'i9 sera corrigé.
	WithAmmo int
}

// EquipmentLifeKey identifie une vie d'objet du monde : LA PAIRE (slot, génération), jamais le
// slot seul — le pool de slots reboucle et la génération ne fait que 2 bits.
type EquipmentLifeKey struct{ Slot, Gen uint32 }

// EquipmentPlacement est UNE pose d'objet d'équipement, telle que le film la porte.
type EquipmentPlacement struct {
	// Life identifie la vie d'objet (slot, génération) — la clé qui relie le record de création
	// à la trajectoire décodée des paquets delta.
	Life EquipmentLifeKey
	// T0US est l'instant du record de création : la pose. T1US est le dernier point de la vie
	// décodée, c'est-à-dire l'instant où l'objet cesse de bouger — une BORNE INFÉRIEURE de sa
	// durée de vie, jamais sa disparition (cf. l'en-tête de ce fichier). Les deux sur l'horloge
	// des paquets, celle des positions de bipède.
	T0US, T1US uint64
	// X, Y, Z est la position du record de création, en coordonnées MONDE (bornes de la carte).
	X, Y, Z float32
	// GlobalID est le GlobalID du tag `eqip` de l'objet : son IDENTITÉ, telle que le jeu la
	// définit. Le nom se résout par le manifeste du titre, jamais ici.
	GlobalID uint32
	// Points est le nombre d'échantillons de la vie décodée — le dénominateur d'une trajectoire.
	Points int
}

// EquipmentSpawnEvent est UNE occurrence du type 103, avec la vie d'objet qu'elle désigne.
type EquipmentSpawnEvent struct {
	// Chunk / PacketIndex localisent l'événement dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet — MÊME horloge que
	// [EquipmentPlacement.T0US] et [EquipmentCreation.TimestampUS], donc croisable sans
	// recalage.
	TimestampUS uint64
	// Spawned est la vie d'objet ENGENDRÉE, telle que la référence 1 la désigne : la paire
	// (slot, génération), exactement la clé qu'un record de création écrit. Ne vaut que si
	// [EquipmentSpawnEvent.SpawnedValid].
	Spawned EquipmentLifeKey
	// SpawnedValid : la référence 1 portait sa garde. Faux sur 6 occurrences sur 931.
	SpawnedValid bool
	// Source est la PREMIÈRE référence, rendue BRUTE et non interprétée : elle désigne un
	// `ti=37` de longue durée que les images-clés voient et qu'aucune création delta ne porte.
	// PISTE pour l'équipement source d'un déploiement, non instruite (table D du registre 0.E).
	Source EquipmentLifeKey
	// SourceValid : la référence 0 portait sa garde (929 sur 931).
	SourceValid bool
	// Ref2Present : la TROISIÈME référence portait sa garde. Comptée et JAMAIS LUE : la mesure
	// du parc en trouve 3 sur 931, donc rien ne permet d'établir ce qu'elle désigne. Le compte
	// laisse un futur lot voir si un build la pose davantage, sans qu'aucune décision ne repose
	// dessus aujourd'hui.
	Ref2Present bool
}

// EquipmentSpawnStats dit ce que le balayage a vu — les dénominateurs sans lesquels un compte
// de zéro ne se distingue pas d'un film muet.
type EquipmentSpawnStats struct {
	// Chunks est le nombre de chunks lus ; Packets le nombre de paquets delta traversés.
	Chunks, Packets int
	// Lists est le nombre de paquets dont la liste d'événements n'est PAS vide — le
	// dénominateur de la lecture de tête.
	Lists int
	// Events est le nombre d'occurrences du type 103 lues en tête de liste.
	Events int
	// WithSpawned / WithSource comptent les références présentes ; Ref2 compte les troisièmes
	// références posées (attendu : ~3 sur 931, cf. l'en-tête).
	WithSpawned, WithSource, Ref2 int
}

// GroundWeaponAmmo porte les deux champs PROUVES du composant `weapon-ammo-component`.
type GroundWeaponAmmo struct {
	// Mag est le CHARGEUR de l'arme au moment ou elle a touche le sol (champ A, R(8)).
	Mag uint32
	// Res est la RESERVE a ce meme instant (champ B, R(11)).
	Res uint32
}
