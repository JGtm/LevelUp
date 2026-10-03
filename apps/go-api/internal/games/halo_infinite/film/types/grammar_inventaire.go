package types

// grammar_inventaire.go — L INVENTAIRE D UN BIPED A UNE IMAGE-CLE, ET LE COMPTE DE SA LECTURE
// (couche `grammar`).
//
// Deplaces depuis `film/replay` au lot J4.2 (2026-09-26) SANS REECRITURE, avec
// `inventory_decode.go` qui les produit (cf. grammar_pont.go pour la regle du deplacement).

// KeyframeInventory est l'inventaire d'un biped à l'instant d'une image-clé.
type KeyframeInventory struct {
	// TimestampUS est l'horodatage du paquet — MÊME horloge que BipedPosition.TimestampUS.
	TimestampUS uint64
	// Chunk / PacketIndex localisent l'image-clé dans le film.
	Chunk, PacketIndex int
	// Slot est le slot du biped porteur (celui des trajectoires).
	Slot uint32
	// Grenades porte le compteur de chaque type, par rang (0 Fragmentation, 1 Plasma,
	// 2 Dynamo, 3 Spike). Nulle et GrenadesRead faux = non lu, jamais « zéro grenade ».
	Grenades     [InventorySlotCount]uint32
	GrenadesRead bool
	// GrenadesByPosition dit par QUELLE VOIE les compteurs ont été lus : faux = R2a, par
	// l'ancre de capacité ; vrai = R2b, par la position relative au bloc de munitions (cf.
	// inventory_grenades_rules.go). Sans conséquence sur la valeur publiée — les deux voies
	// lisent le même champ, et sur les 1 167 records où les deux s'appliquent elles rendent la
	// même position 1 167 fois. C'est une TÉLÉMÉTRIE : elle alimente KeyframeInventoryStats,
	// pour qu'une dérive du repli se voie au lieu de se fondre dans le total.
	GrenadesByPosition bool
	// SelectedGrenadeRank est le rang de grenade SÉLECTIONNÉ (i47), ou -1 non lu. C'est le
	// type qui partira au prochain lancer. Publié seulement si le masque lu recoupe
	// exactement les compteurs i22 et si la sélection est unanime dans la fenêtre (cf.
	// invGrenadeSelLo) : à défaut, -1 — une sélection ne se devine pas.
	SelectedGrenadeRank int
	// AbilityRank est le RANG de palette de la capacité portée, ou -1 non lu.
	//
	// C'EST UN RANG, PAS UN INDEX — il l'est depuis le 2026-08-14 (cf. invAbilityRankHigh), et
	// le champ a changé de nom parce qu'il a changé de grandeur. Ce canal ne voit QUE la
	// fenêtre 16..23 de la palette : hors d'elle, l'ancre ne matche pas et la lecture n'existe
	// pas. Le rang complet, sur toute la palette, vient d'i48 (grammar.ScanFilmAbilityRanks).
	//
	// Le NOM ne se décide pas ici : la table est partielle ET propre à la palette du match,
	// et la nommer est le travail de la couche qui possède le catalogue.
	AbilityRank int
	// Ammo est l'état des quatre emplacements décrits par la carte mémoire. Seuls les deux
	// premiers portent une arme ; les deux autres sont vides, et cette vacuité fait partie du
	// critère de parse (44 bits nuls).
	Ammo     [InventorySlotCount]SlotAmmo
	AmmoRead bool
	// DrawnSlot est le sélecteur i42 : 0 ou 1 = cet emplacement est dégainé, 2 = aucune arme
	// dégainée, -1 = non lu. LE 2 EST UNE VALEUR : au premier keyframe le match n'a pas
	// commencé et les huit joueurs ont leurs armes rangées.
	DrawnSlot int
	// AmmoCandidates est le nombre de débuts de bloc qui satisfaisaient le critère. 1 = lecture
	// unique ; au-delà, le plus long a été retenu et ce nombre dit que le départage a eu lieu.
	AmmoCandidates int
}

// KeyframeInventoryStats compte ce que ScanFilmKeyframeInventory (inventory_decode.go) a
// rencontré. Sans ces dénominateurs, une fiche clairsemée ne se diagnostique pas : rien ne
// distingue « peu de keyframes dans le film » de « chunks corrompus » (audit
// AUDIT_AVAL_INVENTAIRE_2026-08-24.md, point 3). Même vocabulaire que les scanners frères
// (types.AbilityRankStats, CamoStateStats, GrappleStats). Vit ici, avec InventoryCoverage
// qu'elle alimente, et non dans inventory_decode.go (seuil de taille du dépôt, CLAUDE.md n°5).
type KeyframeInventoryStats struct {
	// Chunks est le nombre total de chunks du film (CountFilmChunks).
	Chunks int
	// ChunksUnread est le nombre de chunks dont la lecture disque a échoué — un `continue`
	// nu ne les révélait auparavant nulle part, ni compteur ni log.
	ChunksUnread int
	// Keyframes est le nombre de paquets d'image-clé parcourus, tous chunks confondus.
	Keyframes int
	// Records est le nombre de records de biped (ti=invBipedTI) rencontrés dans ces
	// images-clés. `keyframeInventories` n'en écarte AUCUN — chaque record produit une
	// lecture, lue ou non — donc ce compte est AUSSI le nombre de lectures rendues : la même
	// grandeur n'est pas dupliquée sous deux noms.
	Records int
	// GrenadesByAnchor / GrenadesByPosition comptent les lectures de compteurs de grenade PAR
	// VOIE : R2a, ancrée sur la capacité, et R2b, le repli positionnel livré le 2026-08-25
	// (cf. inventory_grenades_rules.go). Leur somme est le nombre de records dont les grenades
	// ont été lues ; `Records` en reste le dénominateur.
	//
	// LES DEUX SONT COMPTÉES SÉPARÉMENT PARCE QU'ELLES N'ONT PAS LE MÊME STATUT. R2a est exacte
	// par construction — l'ancre borne le champ. R2b repose sur une LOI DE POSITION mesurée sur
	// 24 films ; fondue dans un total, une dérive du repli sur un film d'une autre version du
	// jeu ne se verrait nulle part.
	GrenadesByAnchor   int
	GrenadesByPosition int
}
