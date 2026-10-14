package types

// grammar_inventaire.go — L INVENTAIRE D UN BIPED A UNE IMAGE-CLE, ET LE COMPTE DE SA LECTURE
// (couche `grammar`).
//
// Deplaces depuis `film/replay` au lot J4.2 (2026-09-26) SANS REECRITURE (cf. grammar_pont.go pour
// la regle du deplacement).
//
// DEUX PRODUCTEURS, UN SEUL TYPE (2.7.d1, 2026-10-09) : la lecture de l etat complet du bipede par
// la grammaire sur les records que sa regle d admission retient
// (`grammar/keyframe_etat_complet_admission.go`), et, derriere elle, la fenetre de bits
// (`grammar/inventory_decode.go`) sur les autres, sous le repli `repli_fenetre_inventaire_image_cle`.
// Les champs ci-dessous disent ce que vaut chaque valeur selon son producteur.

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
	// SelectedGrenadeRank est le rang de grenade SÉLECTIONNÉ (i47), EN BASE 0 (0 = le premier
	// rang de Grenades), ou -1 : non lu, ou aucun type désigné. C'est le type qui partira au
	// prochain lancer. Lu par la grammaire : la sélection R(3) d'i47, en base 1 dans le film,
	// moins un (décision U-3 du 2026-10-08), sur un record dont le masque d'i47 égale la bitmap des
	// compteurs i22 (règle d'admission). Rendu par la fenêtre : publié seulement si le masque lu
	// recoupe exactement les compteurs i22 et si la sélection est unanime dans la fenêtre (cf.
	// invGrenadeSelLo) — une sélection ne se devine pas.
	SelectedGrenadeRank int
	// AbilityRank est le RANG de palette de la capacité portée, ou -1 non lu.
	//
	// C'EST UN RANG, PAS UN INDEX — il l'est depuis le 2026-08-14 (cf. invAbilityRankHigh), et
	// le champ a changé de nom parce qu'il a changé de grandeur. Il n'est publié que dans la
	// fenêtre 16..23 de la palette : la fenêtre de bits ne voit qu'elle (hors d'elle, l'ancre ne
	// matche pas), et la grammaire, qui lit le rang complet d'i48 dans l'image-clé, COMPTE et ne
	// publie pas un rang hors de 16..23 (décision U-3 du 2026-10-08, `CapaciteHorsDomaine`). Le
	// rang complet, sur toute la palette, vient d'i48 des paquets delta (grammar.ScanFilmAbilityRanks).
	//
	// Le NOM ne se décide pas ici : la table est partielle ET propre à la palette du match,
	// et la nommer est le travail de la couche qui possède le catalogue.
	AbilityRank int
	// Ammo est l'état des quatre emplacements décrits par la carte mémoire. Seuls les deux
	// premiers portent une arme ; les deux autres sont vides, et cette vacuité fait partie du
	// critère de parse (44 bits nuls).
	Ammo     [InventorySlotCount]SlotAmmo
	AmmoRead bool
	// DrawnSlot est l'EMPLACEMENT D'ARME DÉSIRÉ EN MAIN PRINCIPALE (i42, param[1] de
	// `FUN_1406d01fc` : unité +0x389), un index d'emplacement 0..3 — égal à l'emplacement
	// dégainé hors d'un changement d'arme en cours —, ou -1 : absent (porte de `FUN_1406d00ec`
	// levée) ou non lu ; un -1 est publié comme une ABSENCE (décisions U-3 et U-4 du 2026-10-08).
	// Aucun lecteur relu du jeu ne donne à une valeur le sens « aucune arme dégainée ». Rendu par
	// la fenêtre, c'est le DERNIER R(2) présent du bloc (param[2], seconde main, en ambidextrie).
	DrawnSlot int
	// AmmoCandidates est le nombre de débuts de bloc qui satisfaisaient le critère de la fenêtre.
	// 1 = lecture unique (toujours 1 pour une lecture de la grammaire) ; au-delà, le plus long a
	// été retenu et ce nombre dit que le départage a eu lieu.
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
