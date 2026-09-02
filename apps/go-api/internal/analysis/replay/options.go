package replay

// options.go — LE REGLAGE ET LES ENTREES DE DONNEES DE L'ASSEMBLAGE.
//
// DEPLACEMENT PUR depuis `build.go` (lot 1 de PLAN_CUISSON_PERF, 2026-09-02) : `Options` et ses
// deux accesseurs de defaut y occupaient 205 lignes — presque un quart d'un fichier de 928 —
// alors qu'ils ne sont pas de l'assemblage mais son CONTRAT D'ENTREE. Le lot 0 avait pousse
// `build.go` de 875 a 922 lignes par l'observateur (note N-G du plan), et la migration vers un
// film deja charge l'aurait encore alourdi. Le struct, ses commentaires et les deux methodes
// sont repris TELS QUELS : aucune ligne de logique n'a change.
//
// CE QUE `Options` MELANGE, ET POURQUOI CE N'EST PAS UN DEFAUT : du REGLAGE (pas de temps, seuil
// de publication) et des ENTREES DE DONNEES (loadouts, grenades, projectiles, zones...). Les
// secondes vivent ici plutot qu'en parametres pour garder `BuildFromPositions` sous la limite de
// cinq arguments du depot — c'est ecrit champ par champ, a chaque fois que le cas se pose.

import (
	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
)

// Options règle l'assemblage du document de rejeu.
type Options struct {
	// Scoped (facultatif) rend le PALIER DE LUNETTE d'un slot a un instant. Cf. zoom_state.go.
	Scoped func(slot uint32, tsUS uint64) int
	// FrameIntervalMS : pas de temps de la grille ; 0 -> DefaultFrameIntervalMS.
	FrameIntervalMS int
	// MinPoints : seuil de publication d'une track ; 0 -> DefaultMinPoints.
	MinPoints int
	// Geometry : props Forge optionnels (repères contextuels, pas le fond de carte).
	Geometry []MapObject
	// Structure : emprises de la géométrie structurelle de la carte (le vrai fond de
	// carte, cf. structure.go). Optionnelle : une carte sans fichier figé donne un rejeu
	// sans fond, pas une erreur.
	Structure []Surface
	// Loadouts : armes portées décodées des keyframes (cf. loadouts.go). Entrée de DONNÉES
	// et non de réglage — elle vit ici plutôt qu'en paramètre pour ne pas pousser
	// BuildFromPositions au-delà de 5 arguments. Absente = rejeu sans armes portées.
	Loadouts []filmdec.KeyframeLoadout
	// Grenades : lancers de grenade décodés des paquets delta (cf. grenades.go). Comme
	// Loadouts, c'est une entrée de DONNÉES. Absente = rejeu sans lancers. Le rattachement
	// à un slot passe par le pont du fil des morts : sans morts lisibles, les lancers décodés
	// ne sont pas publiés (on refuse de les poser sur le mauvais joueur).
	Grenades []filmdec.GrenadeThrow
	// Projectiles : trajectoires de projectile decodees des paquets delta (cf. projectiles.go).
	// Entree de DONNEES, comme Loadouts et Grenades. Absente = rejeu sans trajectoires.
	Projectiles []filmdec.ProjectileTrack
	// Inventory : inventaire complet lu aux memes images-cles que les armes portees
	// (cf. inventory.go). Entree de DONNEES. Absente = rejeu sans grenades ni munitions.
	Inventory []KeyframeInventory
	// InventoryDeltas sont les lectures d'inventaire des paquets DELTA (grenades). Absentes =
	// le film n'en transmet pas, ou le balayage a echoue : l'axe des grenades retombe alors sur
	// les seules images-cles.
	InventoryDeltas []filmdec.InventoryDelta
	// InventoryDeltaAmmoRefused reporte la porte du scanner : le canal MUNITIONS de ce film a
	// ete refuse en bloc. Pure telemetrie — les grenades ne sont pas concernees.
	InventoryDeltaAmmoRefused bool
	// AbilityRanks : les identites de capacite transmises par i48 dans les paquets DELTA
	// (cf. abilities.go). Entree de DONNEES, comme Inventory. C'est le canal qui voit TOUTE
	// la palette ; celui des images-cles, porte par Inventory, n'en voit que la fenetre
	// 16..23. Absente = rejeu dont les capacites se limitent a cette fenetre.
	AbilityRanks []filmdec.AbilityRank
	// CamoStates : les transmissions de la voie d'etat du camouflage (i28 queue[1], cf.
	// filmdec/camo_state.go). Entree de DONNEES, comme AbilityRanks. Absente = rejeu sans
	// episodes de camouflage — le surbouclier, lui, voyage dans les positions (Shield.Q).
	CamoStates []filmdec.CamoRead
	// GrappleReads : les evenements de grappin lus dans le corps tag==3 d'i59 (cf.
	// filmdec/grapple_state.go). Entree de DONNEES, comme CamoStates. Absente = rejeu sans
	// tractions de grappin — jamais des tractions devinees.
	GrappleReads []filmdec.GrappleRead
	// Placements / PlacementStats : les POSES d'objets d'equipement lues dans les records de
	// CREATION de l'archetype 37 (cf. filmdec/equipment_placements.go). Entree de DONNEES,
	// comme GrappleReads. Absente = rejeu sans poses — jamais des poses devinees.
	//
	// LES STATISTIQUES VOYAGENT AVEC LA LISTE, et il le faut : elles portent le decoupage de
	// bloc CALIBRE sur ce film. Une liste vide sans elles serait indistinguable d'un film
	// sans equipement, alors que ce peut etre un film dont la calibration a echoue.
	// WeaponChanges : les PRISES ET LACHERS d'arme lus dans le flux delta (cf.
	// filmdec/held_weapon_changes.go). Entree de DONNEES, comme GrappleReads. Absente =
	// rejeu sans ramassages — jamais des ramassages devines.
	WeaponChanges []filmdec.HeldWeaponChange
	// Pickups / PickupStats : les RAMASSAGES NATIFS lus dans la liste d'evenements des paquets
	// delta (evenement `biped_pickup`, cf. filmdec/biped_pickups.go). Entree de DONNEES, comme
	// WeaponChanges. Absente = rejeu sans ramassages natifs — jamais des ramassages devines.
	//
	// LES STATISTIQUES VOYAGENT AVEC LA LISTE, et il le faut : elles portent le compte des
	// listes MULTIPLES, c'est-a-dire la mesure de ce que le canal ne peut PAS voir (un
	// ramassage en 2e position d'une liste lui echappe). Une liste vide sans elles serait
	// indistinguable d'un film sans ramassage.
	Pickups     []filmdec.BipedPickup
	PickupStats filmdec.BipedPickupStats
	// EquipmentChanges / EquipmentChangeStats : les RAMASSAGES ET CONSOMMATIONS d'equipement
	// lus dans le flux delta (cf. filmdec/equipment_changes.go). Entree de DONNEES, comme
	// WeaponChanges. Les stats voyagent avec parce qu'elles portent le TEMOIN DE COMPLETUDE
	// (compteur de rotation) : sans elles, la couverture ne saurait pas dire ce qui manque.
	EquipmentChanges     []filmdec.EquipmentChange
	EquipmentChangeStats filmdec.EquipmentChangeStats
	Placements           []filmdec.EquipmentPlacement
	PlacementStats       filmdec.EquipmentPlacementStats
	// Pads : ce que le film rend sur les SOCLES — armes au sol (`ti=42`) et power-ups (`ti=37`),
	// TROIS lectures chacun, `Scanned` disant qu'elles ont abouti (cf. build_ground_weapons.go).
	// Entree de DONNEES, comme Placements. Absente = rejeu sans socles — jamais des socles devines.
	Pads PadScans
	// SpawnPoints : les points d'apparition d'objet ramassable NON-ARME de la CARTE, lus au
	// catalogue fige par l'appelant (cf. cmd/replay-build). Entree de DONNEES, pas de reglage.
	//
	// POURQUOI L'APPELANT ET PAS LE BUILDER : la generation d'artefact est HORS LIGNE et le
	// reste. Le builder ne va rien chercher — on lui donne ce que la carte declare, ou rien.
	SpawnPoints []MapSpawnPoint
	// SpawnPointsState dit CE QUE VAUT l'absence d'un point : carte absente du catalogue,
	// carte connue dont les points ne sont PAS ETABLIS, ou points etablis (fut-ce a zero).
	// Les trois valeurs et leur raison d'etre sont documentees sur
	// `PickupCoverage.SpawnPointsState`, qui les publie. Vide = carte absente.
	SpawnPointsState string
	// Deaths : le fil des morts du film (chunk highlight), qui NOMME les vies et fonde TOUT le
	// rattachement (cf. lives.go). Entrée de DONNÉES comme les précédentes.
	//
	// SANS ELLE, AUCUN TIR NI LANCER N'EST PUBLIÉ — et c'est voulu. Il n'existe plus de repli :
	// les deux méthodes qui faisaient élire un propriétaire de slot ont été retirées le
	// 2026-07-28. Un rejeu muet se voit ; un rejeu qui pose des tirs sur le mauvais joueur ne
	// se voit pas.
	Deaths []Death
	// PlayerIndices est la table identité -> index de joueur, LUE dans le film (cf.
	// player_index.go). Second maillon du pont, et lui aussi une lecture. Absente, aucun tir
	// ni lancer n'est publié.
	PlayerIndices PlayerIndexTable
	// Objectives : les actions d'objectif NOMMÉES ET IDENTIFIÉES PAR MANCHE (cf. objectives.go).
	// Entrée de DONNÉES, comme Loadouts et Grenades.
	//
	// POURQUOI DÉJÀ IDENTIFIÉES, et pas décodées ici : le NOMMAGE et le pont d'identité sont un
	// second décodage du statborg que l'appelant fait UNE fois (cf. replaybuild/matchfacts.go),
	// et qu'il fait servir aussi à la courbe de score — les refaire ici rejouerait ce décodage.
	// Le pont est PAR MANCHE, par les seuls INSTANTS DE MORT (aucune base, JUSTE en multi-manche
	// où le slot d'entité est réattribué) — comme la couronne VIP et le drapeau vivant, à cette
	// nuance près qu'eux se résolvent dans ce paquet parce qu'ils lisent `opt.Deaths` déjà scanné
	// ici. Absente = rejeu sans calque d'objectifs.
	Objectives []objectiveevents.IdentifiedEvent
	// Score : de quoi construire LA COURBE DE SCORE (entrée de DONNÉES comme Objectives ; cf. score_timeline.go et build_score.go). Nil = ni calque ni couverture de score.
	Score *ScoreInput
	// Flag : de quoi construire LA VIE DES DRAPEAUX de CTF (entrée de DONNÉES comme Score ; cf.
	// build_objectives_live.go). `Scanned` faux = ni calque ni couverture de drapeau.
	Flag FlagInput
	// Zone : de quoi construire L'ETAT DES ZONES (entrée de DONNÉES comme Flag ; cf.
	// build_zones.go). Le CATALOGUE de zones vient de l'appelant — c'est lui qui sait joindre la
	// carte du match — et il commande le balayage : sans zones, `ti=13` n'est pas lu.
	Zone ZoneInput
	// Vip : de quoi construire LA COURONNE VIP (entrée de DONNÉES comme Flag ; cf. vip_crown.go).
	// `Scanned` faux = ni couronne ni couverture. La GARDE DE MODE est chez l'appelant : `comp
	// 22 A` vaut `flag_grabs` en CTF, donc seul un appelant qui reconnaît le match VIP par
	// `game_variant_name` le pose — ce paquet ne devine aucun mode.
	Vip VipInput
	// Skull : de quoi construire LE PORTEUR DU CRANE d'Oddball (entrée de DONNÉES comme Vip ; cf.
	// skull_carries.go). `Scanned` faux = ni calque ni couverture. La GARDE DE MODE est chez
	// l'appelant : `comp 0 A` est le score de mode de tout mode, donc seul un appelant qui
	// reconnaît le match Oddball (par `game_variant_name`) le pose — ce paquet ne devine aucun mode.
	Skull SkullInput
	// Bomb : de quoi construire L'ARMEMENT DE LA BOMBE d'Assaut (entrée de DONNÉES comme Skull ;
	// cf. bomb_armings.go). `Scanned` faux = ni balayage ni calque ni couverture. La GARDE DE
	// MODE est chez l'appelant : le canal n'est prouvé que sur Neutral Bomb et Husky Raid,
	// jamais One Bomb — ce paquet ne devine aucun mode.
	Bomb BombInput
	// NeutralDeaths : les morts que personne ne revendique, AVEC LEUR TYPE DÉJÀ RÉSOLU
	// (cf. NeutralDeath). Entrée de DONNÉES comme Deaths et Objectives.
	//
	// POURQUOI DÉJÀ RÉSOLUES, et pas décodées ici : la source du dégât fatal se lit dans le
	// dead-state du film, et ce décodage a UN seul propriétaire dans le dépôt (le paquet
	// `killsource` du titre). Le redécoder ici en ferait un second décodeur du même fait, et
	// deux décodeurs du même fait divergent — c'est la règle qui gouverne déjà Objectives et
	// les couples de `killpos.go`. L'appelant décode, résout le pictogramme du titre, fournit.
	// Absente = rejeu dont les lignes de mort neutres gardent leur repère générique.
	NeutralDeaths []NeutralDeath
	// Kills : les frags/assistances RÉSOLUS EN IDENTITÉ pour la jointure avec les épisodes
	// d'état actif (camo, surbouclier — cf. equipment_episode_kills.go). Entrée de DONNÉES
	// comme NeutralDeaths, MÊME RAISON : la source de dégât/crédit a un seul propriétaire
	// dans le dépôt (`killsource`), et le redécoder ici en ferait un second décodeur du même
	// fait. `Kills.Read=false` (repli zéro) publie `Coverage.Equipment.KillsRead=false` —
	// jamais un `EquipmentEpisode.K/A` à zéro qui se lirait comme une mesure.
	Kills KillsInput
	// FilmClockOriginUS est l'horodatage moteur du PREMIER PAQUET du film, c'est-à-dire le
	// zéro de l'horloge sur laquelle les highlight events sont datés (cf. origin.go). Entrée
	// de DONNÉES, comme Loadouts et Deaths. Zéro = origine incalculable : le document ne
	// publie alors aucune origine, et le client retombe sur son appariement.
	FilmClockOriginUS uint64
	// Scan : réglages du décodage offline ; zéro -> filmdec.DefaultScanFilmOptions().
	Scan *filmdec.ScanFilmOptions
	// Labels : le catalogue de libellés DU TITRE (armes, grenades, capacités), chargé
	// depuis config/titles/{slug}/mappings/ par l'appelant hors ligne (cf. catalog.go).
	// Absent = document sans table de libellés : le client affiche les identifiants
	// bruts, ce qui reste vrai — contrairement à un nom approché.
	Labels LabelCatalog
	// MapQuant : l'ENTRÉE DE CATALOGUE de la carte du match (cf. filmdec.MapQuantCatalog).
	// OBLIGATOIRE : sans elle le décodeur ne produit que des quanta, et BuildFromFilm refuse
	// d'émettre un document plutôt que des coordonnées fausses (elles l'étaient jusqu'ici :
	// les bornes de Cliffhanger étaient appliquées à toutes les cartes, et le filtre de
	// téléportation en m/s décalibré d'autant).
	//
	// POURQUOI L'ENTRÉE ENTIÈRE ET NON `*filmdec.Vec3Range` (correctif du 2026-08-15) : les
	// BORNES et les LARGEURS D'AXE sont deux faces de la même entrée de catalogue, et jusqu'ici
	// seules les bornes descendaient. Les largeurs restaient au défaut de paquet — celles de
	// Cliffhanger — sur toutes les autres cartes. Les porter dans un second champ aurait laissé
	// armer l'une sans l'autre : un seul champ, donc, et l'oubli devient impossible.
	MapQuant *filmdec.MapQuantEntry
	// Observe recoit chaque etape de BuildFromFilm et sa sortie (cf. observe.go). Nil = rien —
	// mais EN PRODUCTION IL N'EST JAMAIS NIL : `replaybuild.BuildBytes` passe toujours sa
	// methode `b.observe`, qui teste elle-meme si un observateur est branche (cf. observe.go).
	Observe Observer
	// clock date la fin du balayage precedent, pour la duree Debug par balayage (cf. observe.go).
	// NON EXPORTE ET SANS REGLAGE : c'est BuildFromFilm qui l'arme, au moment ou le decodage
	// commence — un appelant qui le fournirait daterait le premier balayage depuis sa propre
	// preparation. Nil (BuildFromPositions, tests) = aucune mesure, aucun cout.
	clock *stepClock
}

func (o Options) frameIntervalMS() int {
	if o.FrameIntervalMS > 0 {
		return o.FrameIntervalMS
	}
	return DefaultFrameIntervalMS
}

func (o Options) minPoints() int {
	if o.MinPoints > 0 {
		return o.MinPoints
	}
	return DefaultMinPoints
}
