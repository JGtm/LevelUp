package types

// grammar_monde.go — LES TYPES DE CONTRAT DES OBJETS DU MONDE (couche `grammar`).
//
// Deplaces depuis `film/internal/grammar` au lot 2.6.2 (volet grammaire / rejeu) SANS
// REECRITURE : les commentaires sont ceux des declarations d origine, et un renvoi de godoc y
// designe encore un symbole de la couche qui produit.

// DeadState holds the captured fields of the BIPED dead-state heavy form
// (FUN_140c1dd44), the kill-feed death event recorded on the victim's
// object-dead-state component. These are THE candidate weapon/damage-source
// fields established by the RE of the kill feed mechanism:
//
//	Mort   = comp+0x70  death flag (R(1) before the anim block).
//	EnumA  = comp+0x04  R(5) tag-table enum (damage-type / method candidate).
//	EnumB  = comp+0x08  R(5) tag-table enum (damage-type / method candidate).
//	Val0c  = comp+0x0c  R(4) value.
//	Val0e  = comp+0x0e  R(3) value.
//	HasRef = the leading present bit of the +0x10 block was set.
//	GIDPresent = the inner global-id present bit was set.
//	GlobalID   = comp+0x10 source R(32) global-id (resolved via GetLocalHandleFromGlobalId,
//	             table DAT_144b404f0) — SAME mechanism as the WST weapon handle.
//	             This is the strongest WEAPON / damage-source candidate.
//	Val14, Val18 = comp+0x14 (R(3)), comp+0x18 (R(1)+optR(6)) — only read when HasRef.
//
// The -1 / 0xFFFFFFFF sentinels mean "absent" (the engine writes them on the
// not-present branches).
type DeadState struct {
	Mort       bool
	EnumA      int32
	EnumB      int32
	Val0c      uint8
	Val0e      uint8
	HasRef     bool
	GIDPresent bool
	GlobalID   uint32 // raw R(32) global-id (0xFFFFFFFF if absent) — the weapon/source ref
	Val14      uint8
	Val18      int8
	// SrcTag0 (comp+0x00) and SrcTag4c (comp+0x4c) are the two readOpt(1+32) reads at
	// the HEAD and TAIL of FUN_140c1dd44, both via FUN_14080d69c. The 2026-07-10g
	// Ghidra grammar analysis hypothesised these were the persisted damage SOURCE tag
	// (melee/grenade cause) in the firearm family tag space. EMPIRICALLY REFUTED
	// (2026-07-10, tmp_dmgsource on film 000d5950): 0/786 resolve to any weapon family
	// — the values are RUNTIME entity/anim handles, because FUN_14080d69c reads a
	// local-handle, NOT the build-time tag string-id (that is FUN_14080dec4, used for
	// the firearm family and the WST held-weapon variant). Captured for future
	// world-binding resolution (a handle CAN be resolved to a source entity/tag once
	// the entity store is bound), NOT as a direct weapon name. 0xFFFFFFFF == absent.
	SrcTag0  uint32 // comp+0x00 readOpt(1+32) head handle (FUN_14080d69c) — runtime handle, not a tag
	SrcTag4c uint32 // comp+0x4c readOpt(1+32) tail handle (FUN_14080d69c) — runtime handle, not a tag
}

// ObjectDeath est UNE mort écrite : le composant dead-state d'une entité, porté à `Mort`.
type ObjectDeath struct {
	// TimestampUS est l'instant du PAQUET qui la porte, sur l'horloge du film.
	TimestampUS uint64
	// Slot / Gen identifient la VIE de l'entité (le pool de slots reboucle, la génération fait
	// 2 bits) ; TypeIndex est son archétype.
	Slot, Gen, TypeIndex uint32
	// Dead est le composant capturé. Chez le bipède il porte le couple victime / tueur ; sur
	// les autres archétypes seul le drapeau `Mort` est établi (les champs restent lus, leur
	// SENS ne l'est pas — cf. la réserve 1 de la note V13).
	Dead DeadState
	// TailDesync dit que le record a rompu APRÈS le dead-state : la tête est lue au bon
	// endroit, la queue du record n'est pas modélisée. Compté à part, jamais confondu avec un
	// record entièrement porté.
	TailDesync bool
}

// NavpointRadialRead est UNE lecture de `managed-navpoint-radial-progress` (ti=12 i14), datee
// sur l'horloge du MANIFESTE (la meme que `objectives.StatRecords`, donc que les
// explosions du statborg).
type NavpointRadialRead struct {
	// Slot identifie le point de navigation. Les navpoints vont par paires (+12, un par camp).
	Slot uint32
	// TMS est l'instant en millisecondes de MATCH (horloge du manifeste).
	TMS int32
	// Q est le quantum de progression, plage R(8) : 0..255.
	Q uint8
	// Chained dit que le record porteur se termine sur un en-tete de record valide — le seul
	// temoin de fiabilite PAR LECTURE que le balayage possede.
	Chained bool
}

// ProjectileSample est une position de projectile à un instant.
type ProjectileSample struct {
	TimestampUS uint64
	Chunk       int
	X, Y, Z     float32
	// AtRest signale que ce record porte `projectile-at-rest-state` : le vol est fini.
	AtRest bool
}

// ProjectileTrack est la vie d'un projectile : un slot, une génération, une suite de positions.
type ProjectileTrack struct {
	// Slot et Gen identifient la vie. LA PAIRE, pas le slot seul : le pool de slots reboucle
	// et un même slot sert plusieurs projectiles au cours du match.
	Slot uint32
	Gen  uint32
	Pts  []ProjectileSample
}

// TranslocatorTeleport est UNE téléportation exécutée : quand, par quel bipède — et, quand la
// charge a pu être lue, d'où à où.
type TranslocatorTeleport struct {
	// TimestampUS : l'horodatage du paquet porteur — MÊME horloge que
	// BipedPosition.TimestampUS (l'horloge MOTEUR des paquets, cf. le piège documenté au
	// rapport R1 §0 : elle n'est PAS la timeline de l'artefact).
	TimestampUS uint64
	// Slot : le slot du bipède qui se téléporte, directement comparable à
	// BipedPosition.Slot.
	Slot uint32
	// From / To : le DÉPART et l'ARRIVÉE du saut en coordonnées monde (X, Y, Z), déquantifiés
	// aux bornes VRAIES de la carte. Valides SEULEMENT si HasPositions ; à lire ensemble,
	// jamais l'une sans l'autre.
	From, To [3]float32
	// HasPositions dit que la charge a été lue ET déquantifiée. Faux = l'instant et le slot
	// restent mesurés, les positions ne sont pas connues (carte hors catalogue, entrée sans
	// largeurs, région étrangère, ou charge non conforme au layout). Un appelant qui
	// lirait From/To sans ce témoin publierait l'origine du monde comme une position.
	HasPositions bool
}

// VehicleEvent est un embarquement (board) ou une sortie (exit) décodé depuis la liste
// d'événements d'un paquet delta.
type VehicleEvent struct {
	// Kind vaut EventBipedBoardVehicle ou EventUnitExitVehicle.
	Kind int
	// Chunk / PacketIndex localisent l'événement dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'INSTANT de l'événement, en microsecondes (horloge du film) — même
	// horloge que BipedPosition/FireEvent, donc directement croisable.
	TimestampUS uint64

	// OccupantPresent : la référence 0 (l'unité) est présente.
	OccupantPresent bool
	// OccupantSonde : la sonde de la réf 0 de la SORTIE (domaine 1). 1 = index bipède-relatif
	// (9 bits), 0 = slot absolu (13 bits). Toujours 0 pour un EMBARQUEMENT : ses réfs sont en
	// domaines 2/3/7, et `FUN_1406d3140` ne lit la sonde que pour le domaine 1.
	OccupantSonde int
	// OccupantSlot est le slot de l'unité : base bipède + index si sonde=1, index brut sinon.
	OccupantSlot uint32
	// OccupantInBand : le slot tombe dans la bande de slots bipèdes du film (contrôle).
	OccupantInBand bool

	// VehicleSlot est le slot du VÉHICULE que l'événement NOMME : la RÉFÉRENCE 1 de la SORTIE,
	// lue en domaine 1 exactement comme l'occupant.
	//
	// POURQUOI LE MÊME DOMAINE POUR DEUX CHOSES DIFFÉRENTES — c'est l'acquis du lot V7 (§ 6 du
	// rapport `V7_DESTRUCTION_EVENEMENT_2026-09-02.md`) : LE DOMAINE 1 EST CELUI DES UNITÉS, et
	// dans la taxonomie Halo le BIPÈDE et le VÉHICULE sont deux spécialisations d'UNITÉ. La base
	// est la même (le minimum de la bande bipède) et l'index de 9 bits porte au-delà, jusqu'à la
	// bande `ti=40`. Mesure : sur les références de domaine 1 dont l'index SORT de la bande
	// bipède, 99,6 à 100,0 % tombent dans la bande `ti=40` (types 0/1/7/36, 12 films) là où le
	// hasard en mettrait 3 à 16 %. Le lot V6 avait cherché le véhicule en domaine 7 et l'y avait
	// réfuté à raison : il est en domaine 1.
	//
	// POUR LA SORTIE, LA MESURE EST SANS RESTE : 105 / 105 sorties de 12 films, 100,0 % en bande
	// `ti=40`, zéro bipède, zéro hors bande (V7 § 7). C'est cette référence-là que le calque de
	// rejeu emploie pour résoudre le véhicule d'un épisode d'occupation, la géométrie n'étant plus
	// que le repli.
	VehicleSlot uint32
	// VehicleSlotValid : la référence du véhicule était PRÉSENTE (bit de garde posé). Faux pour un
	// EMBARQUEMENT : ses trois références sont en domaines 2/3/7 et AUCUNE ne résout un slot
	// `ti=40` (mesure au § 2 du rapport V8).
	VehicleSlotValid bool
	// VehicleGen est la GÉNÉRATION (2 bits) du handle du véhicule, lue dans la même référence.
	// Elle n'est PAS la clé de vie employée par le calque — celle-ci se résout par la fenêtre
	// temporelle du recensement —, mais elle en est le contrôle indépendant (V8 § 2).
	VehicleGen uint32

	// Seat est le siège (R(6)) lu en fin de charge. SeatValid=false si le payload est trop court.
	Seat      uint32
	SeatValid bool
}
