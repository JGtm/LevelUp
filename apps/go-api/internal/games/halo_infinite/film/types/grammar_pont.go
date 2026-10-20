package types

// grammar_pont.go — LES TYPES DE CONTRAT DES LECTURES DU PONT D IDENTITE ET DE L INVENTAIRE
// D IMAGE-CLE (couche `grammar`).
//
// Deplaces depuis `film/replay` au lot J4.2 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25
// (2026-09-26, decision DU-3 = S1) SANS REECRITURE, avec les cinq fichiers de lecture qui les
// produisent (`deaths_source.go`, `player_index.go`, `origin.go`, `film_player_table.go`,
// `inventory_decode.go`, descendus en `grammar`) : les commentaires sont ceux des declarations
// d origine, et un renvoi de godoc y designe encore un symbole de la couche qui produit.
//
// `FilmPlayerTable` N EST PAS ICI, et c est la regle du paquet (doc.go) : elle porte une methode
// (`Lue`, une regle). Elle vit en `grammar`, avec sa lecture. `SlotAmmo` et
// `KeyframeInventoryStats` sont venus PAR CLOTURE : le premier est le type d un champ de
// `KeyframeInventory`, le second le compte que rend sa lecture.

// Death est une mort du fil, telle que le film la porte : une identité et un instant.
// L'identité est le XUID — jamais un index (cf. la règle « un ordre n'est pas une
// identité », qui a déjà produit une fausse découverte dans ce chantier).
type Death struct {
	// XUID identifie la victime. Stable, global, indépendant de tout tri.
	XUID uint64
	// Gamertag est le nom porté PAR LE FILM lui-même, dans le même enregistrement que le xuid
	// (32 octets UTF-16LE). Il n'est pas obligatoire au rattachement — celui-ci ne travaille
	// que sur le xuid — mais il rend le rejeu lisible SANS base de données, ce qui est la
	// propriété que tout ce pipeline cherche à préserver. Vide si l'enregistrement ne le porte
	// pas ; l'identité reste alors le xuid.
	Gamertag string
	// TimeMS est l'instant de la mort sur l'horloge du MATCH (origine = début du match),
	// qui n'est pas celle du film. Le décalage entre les deux est résolu par mesure.
	TimeMS int64
}

// PlayerIndexTable est le lien identité -> index de joueur du film, lu et contrôlé.
type PlayerIndexTable struct {
	// ByXUID est la table elle-même.
	ByXUID map[uint64]int
	// Readings est le nombre de chunks de réplication qui ont produit la table.
	Readings int
	// Disagreements est le nombre de xuids pour lesquels deux chunks ont lu des index
	// DIFFÉRENTS. Non nul, la table n'est pas publiée : ce n'est pas un désaccord à
	// arbitrer, c'est le symptôme d'une lecture fausse.
	Disagreements int
}

// InventorySlotCount est le nombre de types de grenade décrits par i22, et aussi le nombre
// d'emplacements d'arme décrits par la carte mémoire (0x7F0 + s*0x90, quatre entrées).
//
// LA DIMENSION FAIT PARTIE DE LA FORME : `KeyframeInventory` porte deux tableaux de cette
// longueur. Meme rangement que `MPPFieldCount` : `grammar` la REPREND (`invGrenadeSlots =
// types.InventorySlotCount`), une seule source.
const InventorySlotCount = 4

// SlotAmmo est l'état de munitions d'UN emplacement d'arme.
//
// LES TROIS CAS NE SE CONFONDENT PAS, et c'est tout l'intérêt des pointeurs :
//   - Mag non nil    : arme à chargeur (chargeur + réserve) ;
//   - Gauge non nil  : arme à jauge de charge, fraction dans [0,1] sur 4096 niveaux ;
//   - les deux nil   : le film n'écrit RIEN pour cet emplacement. Pour une arme à charge, cela
//     veut dire PLEIN — le flux est différentiel et le plein est la valeur par défaut, donc il
//     n'est jamais transmis. Ce n'est PAS « zéro » : publier 0 affirmerait un chargeur vide.
type SlotAmmo struct {
	Mag   *uint32
	Res   *uint32
	Gauge *float64
	// Overheat et Flags sont lus mais non interprétés : ils bornent le parse (leur largeur
	// entre dans le critère d'atterrissage) sans qu'on prétende savoir ce qu'ils disent.
	Overheat uint32
	Flags    uint32
}
