package replay

// film_inputs.go — CE QUE L'ETAGE DE BALAYAGE REND, ET RIEN D'AUTRE.
//
// # POURQUOI CE TYPE EXISTE (lot 1.0, decouverte D7 du PLAN_DECODEUR_FILM)
//
// `BuildFromFilm` enchaine une trentaine de balayages puis assemble. Jusqu'au 2026-09-14, cette
// SEQUENCE ne vivait qu'a un endroit — le corps de `BuildFromFilm` — et le fixture d'entrees
// (`golden_inputs_film_test.go`) la RECOPIAIT a la main pour figer les memes entrees. Les deux
// copies ont diverge : cinq canaux que la production cable n'existaient pas du tout au fixture
// (`WeaponChanges`, `Pickups`, `EquipmentChanges`, `Vehicles`, `BipedCreations`), la lunette non
// plus, et les largeurs d'axe du chemin world-object s'installaient par un AUTRE geste — sans la
// largeur d'index de region ni la region cataloguee.
//
// Le decoupage est donc : `scanFilmInputs` BALAIE et rend un `FilmInputs` ; `BuildFromFilm` =
// cet etage, puis `BuildFromPositions`. Le fixture appelle le MEME etage. Il n'y a plus de
// sequence a recopier.
//
// # CE QUE `FilmInputs` PORTE, ET POURQUOI PAS PLUS
//
// Exactement les champs que l'assemblage CONSOMME : les positions et les tirs (les deux
// parametres de `BuildFromPositions`) plus les champs d'`Options` que les balayages remplissent.
// Les STATISTIQUES qui ne servent qu'a l'observateur ou au journal n'y sont pas : elles restent
// locales au balayage qui les produit, et l'observateur les voit passer la.
//
// # LA FRONTIERE AVEC `Options`
//
// `Options` melange du REGLAGE (pas de temps, seuil de publication), des entrees fournies par
// l'APPELANT (catalogues de zones, roster, faits de la base) et des entrees LUES DANS LE FILM.
// `FilmInputs` est exactement le troisieme tiers. `applyTo` les repose sur les `Options` que
// l'assemblage recoit — c'est la seule ecriture, et elle est ecrite une fois.

import "levelup/go-api/internal/games/halo_infinite/film/grammar"

// FilmInputs porte ce que l'etage de balayage d'un film rend a l'assemblage.
//
// L'ORDRE DES CHAMPS SUIT L'ORDRE DES BALAYAGES (cf. build_from_film.go) : c'est la seule facon
// de lire le type et la sequence cote a cote sans se demander ce qui manque.
type FilmInputs struct {
	// FilmMajorVersion est la version du film, lue dans l'en-tete de son registre. nil = le film
	// ne porte pas son registre (le fil des morts retombe alors sur le decoupage historique du
	// gamertag).
	FilmMajorVersion *int
	// Translocations sont les teleportations du translocateur (evenements type 117). Elles
	// servent DEUX fois : le calque du document, et l'exemption du filtre de vitesse des
	// positions (decision D2) — d'ou leur lecture AVANT les positions.
	Translocations []grammar.TranslocatorTeleport
	// Positions est le nuage NON decime des positions de bipede : le premier parametre de
	// `BuildFromPositions`.
	Positions []grammar.BipedPosition
	// BipedCreations sont les records de CREATION de bipede — le lien direct corps -> joueur.
	BipedCreations []grammar.BipedCreation
	// Fire sont les evenements de tir : le second parametre de `BuildFromPositions`.
	Fire []grammar.FireEvent
	// Loadouts sont les armes portees relevees aux images-cles.
	Loadouts []grammar.KeyframeLoadout
	// WeaponChanges sont les prises et lachers d'arme lus dans le flux delta.
	WeaponChanges []grammar.HeldWeaponChange
	// Pickups / PickupStats sont les ramassages NATIFS (evenement `biped_pickup`) et la mesure
	// de ce que ce canal ne peut PAS voir (listes multiples, refus).
	Pickups     []grammar.BipedPickup
	PickupStats grammar.BipedPickupStats
	// Inventory est l'inventaire complet lu aux memes images-cles que les armes portees.
	Inventory []KeyframeInventory
	// InventoryDeltas sont les lectures d'inventaire des paquets DELTA (grenades, jeu
	// selectionne) ; InventoryDeltaAmmoRefused est le VERDICT de la porte du canal munitions,
	// publie tel quel dans la couverture.
	InventoryDeltas           []grammar.InventoryDelta
	InventoryDeltaAmmoRefused bool
	// AbilityRanks sont les identites de capacite portees (i48).
	AbilityRanks []grammar.AbilityRank
	// EquipmentChanges / EquipmentChangeStats sont les ramassages et consommations d'equipement,
	// avec le TEMOIN DE COMPLETUDE (compteur de rotation) que la couverture publie.
	EquipmentChanges     []grammar.EquipmentChange
	EquipmentChangeStats grammar.EquipmentChangeStats
	// CamoStates sont les transmissions de la voie d'etat du camouflage (i28 queue[1]).
	CamoStates []grammar.CamoRead
	// GrappleReads sont les evenements de grappin (corps tag==3 d'i59).
	GrappleReads []grammar.GrappleRead
	// AbilityImpulses / AbilityImpulseStats sont les impulsions de capacite (tag==1 d'i57/i59).
	AbilityImpulses     []grammar.AbilityImpulse
	AbilityImpulseStats grammar.AbilityImpulseStats
	// AbilityCharges / AbilityChargeStats sont les charges d'equipement restantes (i56).
	AbilityCharges     []grammar.AbilityCharge
	AbilityChargeStats grammar.AbilityChargeStats
	// ZoomEvents sont les bascules de LUNETTE lues dans la liste d'evenements. Elles entrent ici
	// BRUTES, et non deja reduites en `Options.Scoped` : c'est `applyTo` qui reconstruit le
	// palier a l'instant (cf. sa note), pour qu'un fixture n'ait qu'une LISTE a serialiser la ou
	// `Options.Scoped` est une fermeture, qui ne se fige pas.
	ZoomEvents []grammar.ZoomEvent
	// Placements / PlacementStats sont les POSES d'equipement (creations ti=37) et la CALIBRATION
	// du bloc de replication mesuree sur ce film — celle dont les socles et les vehicules
	// heritent leurs largeurs MPP.
	Placements     []grammar.EquipmentPlacement
	PlacementStats grammar.EquipmentPlacementStats
	// SpawnEvents / SpawnStats sont les evenements de liste type 103 `EquipmentSpawnedObject`
	// — « une PIECE a ete engendree » — et les denominateurs de leur balayage. C'est LE SIGNAL
	// ECRIT de l'origine d'une pose de panneau (lot 1.9.1, D13).
	SpawnEvents []grammar.EquipmentSpawnEvent
	SpawnStats  grammar.EquipmentSpawnStats
	// Pads sont les deux voies des SOCLES (armes au sol ti=42, power-ups ti=37).
	Pads PadScans
	// Vehicles est le calque des VEHICULES (ti=40) : recensement, creations, nuage de positions,
	// evenements d'embarquement et visees des occupants.
	Vehicles VehicleScan
	// FlagMarks est le CONTROLE independant du calque du drapeau. Vide hors CTF : le balayage est
	// garde par ce que l'appelant a fourni (`Options.Flag`).
	FlagMarks grammar.CarrierMarkScan
	// ZoneReads / ZoneScanned sont l'etat des zones (ti=13). Le CATALOGUE de zones vient de
	// l'appelant et commande le balayage : sans zones, rien n'est lu et `ZoneScanned` est faux.
	ZoneReads   []grammar.ManagedPropertyRead
	ZoneScanned bool
	// BombReads est l'anneau d'armement de la bombe (ti=12), sur les seuls matchs que l'appelant
	// reconnait Assaut armable.
	BombReads []grammar.NavpointRadialRead
	// Grenades sont les lancers de grenade des paquets delta.
	Grenades []grammar.GrenadeThrow
	// Projectiles sont les trajectoires de projectile.
	Projectiles []grammar.ProjectileTrack
	// Deaths est le fil des morts : il NOMME les vies et fonde tout le rattachement.
	Deaths []Death
	// PlayerIndices est la table identite -> index de joueur, LUE dans le film.
	//
	// ELLE N'EST LUE QUE SI LE FIL DES MORTS EST NON VIDE (sans roster, il n'y a rien a
	// chercher) ; sinon elle reste vide. Avant le lot 1.0, `BuildFromFilm` laissait dans ce cas
	// `Options.PlayerIndices` INTACT — ce qui revient au meme : aucun appelant de `BuildFromFilm`
	// ne la fournit (verifie le 2026-09-14 ; les seuls remplisseurs de ce champ passent par
	// `BuildFromPositions`, cf. `killcollector/positions_identity_entree.go`).
	PlayerIndices PlayerIndexTable
	// FilmTable est la TABLE DES JOUEURS QUE LE FILM ECRIT (`chunk_00`, section 2 et corps) :
	// le lien DIRECT `index <-> xuid <-> gamertag`, lu par [ScanFilmPlayerTable].
	//
	// ELLE NE REMPLACE PAS `PlayerIndices`, ELLE LA PRECEDE : la table du film est celle du
	// DEBUT du film, et un joueur qui rejoint en cours de partie n'y a pas de siege (mesure du
	// 2026-09-14 : jusqu'a 5 sur un BTB). Le registre d'identite pose les sieges du film, puis
	// COMPLETE par la lecture des chunks pour les xuids dont la table est muette.
	FilmTable FilmPlayerTable
	// PlayerTeams est l'EQUIPE DE CHAQUE JOUEUR, lue dans la trame d'etat par
	// [grammar.ScanPlayerTeams] : `index de joueur -> designateur` (`-1` = aucune equipe).
	// C'est la SEULE source d'equipe du document (V4) ; la base ne fait que controler.
	PlayerTeams map[int]int
	// TeamScan est le rapport de cette lecture. Il voyage avec la table parce qu'une table vide
	// et une lecture refusee ne disent pas la meme chose.
	TeamScan grammar.TeamScanReport
	// FilmClockOriginUS est l'horodatage moteur du PREMIER paquet du film. Zero = origine
	// incalculable : le document sort sans origine.
	FilmClockOriginUS uint64
}

// applyTo repose les entrees lues dans le film sur les `Options` de l'assemblage.
//
// C'EST LA SEULE ECRITURE, et c'est le point : tant que `BuildFromFilm` remplissait `opt` champ
// par champ au fil des balayages, tout autre chemin qui voulait les MEMES entrees devait
// recopier la liste — et l'a fait, incompletement (cf. l'en-tete du fichier). Un champ ajoute a
// `FilmInputs` sans ligne ici ne parvient a l'assemblage par AUCUN chemin, ni en production ni au
// fixture : l'oubli se voit des le premier golden.
//
// LES CHAMPS IMBRIQUES SE POSENT UN A UN (`Flag.Marks`, `Zone.Reads`, `Bomb.Reads`) : ces trois
// structures portent AUSSI des entrees de l'appelant (catalogue de zones, garde de mode), qu'un
// remplacement en bloc effacerait.
//
// LA LUNETTE SE RECONSTRUIT ICI, et non au balayage : `buildScopedLookup` est une reduction PURE
// des evenements de lunette et des vies lues dans les positions — les deux sont dans
// `FilmInputs`. La faire ici est ce qui permet a un fixture de porter une liste la ou
// `Options.Scoped` est une fermeture. Cout deplace, pas ajoute : la reduction ne lit aucun octet
// de film (elle etait auparavant imputee a l'etape `zoomEvents`, cf. build_from_film.go).
func (in FilmInputs) applyTo(opt *Options) {
	opt.FilmMajorVersion = in.FilmMajorVersion
	opt.Translocations = in.Translocations
	opt.BipedCreations = in.BipedCreations
	opt.Loadouts = in.Loadouts
	opt.WeaponChanges = in.WeaponChanges
	opt.Pickups, opt.PickupStats = in.Pickups, in.PickupStats
	opt.Inventory = in.Inventory
	opt.InventoryDeltas = in.InventoryDeltas
	opt.InventoryDeltaAmmoRefused = in.InventoryDeltaAmmoRefused
	opt.AbilityRanks = in.AbilityRanks
	opt.EquipmentChanges, opt.EquipmentChangeStats = in.EquipmentChanges, in.EquipmentChangeStats
	opt.CamoStates = in.CamoStates
	opt.GrappleReads = in.GrappleReads
	opt.AbilityImpulses, opt.AbilityImpulseStats = in.AbilityImpulses, in.AbilityImpulseStats
	opt.AbilityCharges, opt.AbilityChargeStats = in.AbilityCharges, in.AbilityChargeStats
	opt.Scoped = buildScopedLookup(in.ZoomEvents,
		buildLifeSpans(indexBySlot(in.Positions)), zoomHoldUS)
	opt.Placements, opt.PlacementStats = in.Placements, in.PlacementStats
	opt.SpawnEvents, opt.SpawnStats = in.SpawnEvents, in.SpawnStats
	opt.Pads = in.Pads
	opt.Vehicles = in.Vehicles
	opt.Flag.Marks = in.FlagMarks
	opt.Zone.Reads, opt.Zone.Scanned = in.ZoneReads, in.ZoneScanned
	opt.Bomb.Reads = in.BombReads
	opt.Grenades = in.Grenades
	opt.Projectiles = in.Projectiles
	opt.Deaths = in.Deaths
	opt.PlayerIndices = in.PlayerIndices
	opt.FilmTable = in.FilmTable
	opt.PlayerTeams, opt.TeamScan = in.PlayerTeams, in.TeamScan
	opt.FilmClockOriginUS = in.FilmClockOriginUS
}
