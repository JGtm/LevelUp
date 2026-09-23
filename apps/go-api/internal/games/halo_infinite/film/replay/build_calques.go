package replay

// build_calques.go — LES QUATRE PASSES DE CALQUES DE L ASSEMBLAGE : le grappin et les poses
// d equipement, les prises et les socles, les armes au sol et les vehicules, puis les objectifs
// vivants (drapeau, couronne, crane, bombe, zones).
//
// DEPLACEMENT PUR depuis le corps de `BuildFromPositions` (lot 2.7 volet publication,
// 2026-09-16) : chaque passe est le bloc d instructions qu elle etait, dans le meme ordre, avec
// ses commentaires de mesure ; seule la designation des variables a change (`doc` -> `a.doc`).
// Voir `build.go` pour l ordre des passes et ce qu il protege.

import "log/slog"

// poserGrappinEtPoses publie les tractions de grappin (qui exigent les bornes de la carte) et
// les poses d equipement.
func (a *assemblage) poserGrappinEtPoses() {
	// Les TRACTIONS de grappin : fenetre mesuree par vie + ancre en coordonnees monde
	// (cf. grapple_lines.go). L'ancre exige les bornes de la carte : sans MapQuant,
	// aucune traction (regle map_bounds.go — pas de bornes, pas de coordonnee monde).
	switch {
	case a.opt.MapQuant != nil:
		var grapCov *GrappleCoverage
		a.doc.GrappleLines, grapCov = buildGrappleLines(a.opt.GrappleReads, *a.opt.MapQuant, a.origin, a.step, a.doc.Tracks)
		a.doc.Coverage.Grapple = grapCov
		slog.Info("rejeu : tractions de grappin",
			"tirs", grapCov.LightReads, "accroches", grapCov.HeavyReads,
			"tractions", grapCov.Pulls, "vies", grapCov.PullLives,
			"rates", grapCov.UnpairedFires, "corpsCasses", grapCov.BrokenBodies)
	case len(a.opt.GrappleReads) > 0:
		slog.Warn("rejeu : lectures de grappin sans bornes de carte — aucune traction publiee",
			"lectures", len(a.opt.GrappleReads))
	}
	// Les POSES d'equipement : famille par le manifeste du titre, poseur et cap MESURES sur le
	// nuage NON decime (une pose dure quelques dizaines de millisecondes ; la decimation
	// perdrait le record contemporain qui designe le poseur). La FIN OBSERVEE (schema 28)
	// vient du recensement ti=37 deja lu par la chaine des socles (opt.Pads.Powerups).
	a.doc.EquipmentPlacements, a.doc.Coverage.Placements = buildEquipmentPlacements(
		equipmentInputs{
			Raw: a.opt.Placements, Stats: a.opt.PlacementStats, Positions: a.sorted,
			Census: a.opt.Pads.Powerups.Keyframes,
			Spawns: a.opt.SpawnEvents, SpawnStats: a.opt.SpawnStats,
			// LES DEUX SIGNAUX ECRITS DE L'ORIGINE (lot 1.9.1, D13) : la MORT du poseur, que le
			// registre d'identite apparie aux vies (lots 1.6 / 1.8, seul producteur de liens),
			// et sa PRISE d'equipement (`equipmentChanges.taken`). Ni l'une ni l'autre ne se
			// reconstruit ici : on passe la lecture deja faite.
			Lives: a.reg.Vies(), Changes: a.opt.EquipmentChanges,
		},
		replayClock{origin: a.origin, step: a.step, frames: a.doc.FrameCount, fb: a.opt.Fallbacks,
			families: a.opt.Labels.EquipmentFamilies})
	logPlacementCoverage(a.doc.Coverage.Placements)
}

// poserPrisesEtSocles publie les prises et lachers d arme, les ramassages natifs, les socles,
// puis la datation des occupations de socle par l evenement natif.
func (a *assemblage) poserPrisesEtSocles() {
	// LES PRISES ET LES LACHERS d'arme, sur l'axe de frames du document. Les re-annonces d'une
	// arme deja portee au spawn sont ECARTEES ici : ce ne sont pas des ramassages.
	var wcCov WeaponChangeCoverage
	a.doc.WeaponChanges, wcCov = buildWeaponChanges(a.opt.WeaponChanges, a.origin, a.step)
	a.doc.Coverage.WeaponChanges = &wcCov
	slog.Info("rejeu : prises et lachers d arme",
		"decodes", wcCov.Decoded, "publies", wcCov.Published,
		"prises", wcCov.Taken, "lachers", wcCov.Dropped, "echanges", wcCov.Swapped,
		"reannonces", wcCov.Restated, "avantOrigine", wcCov.BeforeOrigin)
	// LES RAMASSAGES NATIFS, dates a la milliseconde et ATTRIBUES par le pont slot -> joueur
	// deja construit ci-dessus. Publies AVANT les socles : `attachWeaponPads` s'en sert pour
	// dater les occupations de socle qui restaient en intervalle de vingt secondes.
	var pkCov PickupCoverage
	judge := newPickupOriginJudge(a.opt, a.pos, a.doc.EquipmentPlacements)
	a.doc.Pickups, pkCov = buildPickups(a.opt.Pickups,
		replayClock{origin: a.origin, step: a.step, frames: a.doc.FrameCount, fb: a.opt.Fallbacks,
			families: a.opt.Labels.EquipmentFamilies},
		pickupInputs{occupant: a.reg.XUIDNumAt, st: a.opt.PickupStats,
			weaponKeys: a.opt.Labels.Keys, judge: judge})
	a.doc.Coverage.Pickups = &pkCov
	slog.Info("rejeu : ramassages natifs",
		"decodes", pkCov.Decoded, "publies", pkCov.Published, "nommes", pkCov.Named,
		"armes", pkCov.Weapons, "objets", pkCov.Items,
		"origineSocle", pkCov.OriginSpawner, "origineSol", pkCov.OriginGround,
		"origineInconnue", pkCov.OriginUnknown, "etatPoints", pkCov.SpawnPointsState,
		"pointsCatalogue", pkCov.MapCatalogPoints,
		"socleParNature", pkCov.SpawnerByPointKind,
		"famillesInconnues", pkCov.UnknownFamilies,
		"avantOrigine", pkCov.BeforeOrigin, "listesMultiples", pkCov.MultiEvent,
		"refuses", pkCov.Refused)
	// Les SOCLES — armes au sol ET power-ups —, sur le meme nuage NON decime (build_ground_weapons.go).
	a.gwObjs = attachWeaponPads(&a.doc, a.opt.Pads, a.sorted,
		replayClock{origin: a.origin, step: a.step, frames: a.doc.FrameCount, fb: a.opt.Fallbacks}, a.opt.Labels)
	// DATATION DES OCCUPATIONS DE SOCLE par l'evenement natif : l'intervalle de vingt secondes
	// devient un instant, et `xuid` cesse d'etre `null`, QUAND un ramassage natif de la meme
	// famille tombe dans la fenetre. Rien n'est efface : une occupation non couverte garde son
	// intervalle intact (pad_pickup_dating.go).
	padDating := datePadPickups(a.doc.WeaponPads, a.doc.PadPickups, a.doc.Pickups)
	a.doc.Coverage.PadDating = &padDating
	slog.Info("rejeu : datation des occupations de socle",
		"occupations", padDating.Occupations, "datees", padDating.Dated, "nommees", padDating.Named,
		"ambigues", padDating.Ambiguous, "nonCouvertes", padDating.Uncovered)
}

// poserArmesAuSolEtVehicules publie les armes au sol objet par objet, les vehicules, et la
// SECONDE porte des tirs — celle des joueurs embarques, qui exige les deux precedents.
func (a *assemblage) poserArmesAuSolEtVehicules() {
	// LES ARMES AU SOL INDIVIDUELLES (schema 27) : la meme chaine que les socles, publiee objet
	// par objet, LIEE aux lachers et aux prises du flux delta, et bornee par l observation —
	// jamais par une table de durees (document_ground_weapon_items.go).
	var gwiCov GroundWeaponItemsCoverage
	a.doc.GroundWeapons, gwiCov = buildGroundWeaponItems(a.gwObjs, a.opt.WeaponChanges, a.sorted,
		replayClock{origin: a.origin, step: a.step, frames: a.doc.FrameCount, fb: a.opt.Fallbacks})
	a.doc.Coverage.GroundWeaponItems = &gwiCov
	logGroundWeaponItems(gwiCov)
	// LES VEHICULES, sur le MEME nuage NON decime de bipedes (ce sont ses TROUS qui portent les
	// episodes d'occupation) et le MEME pont slot -> xuid que les tirs — cf. build_vehicles.go.
	// Pose APRES la couverture : il publie la sienne.
	// LA PORTE DES POSITIONS S APPLIQUE AUSSI AUX VEHICULES (lot M1 des retours du rejeu) : la
	// MEME emprise que les joueurs, mesuree une fois par `passerLaPorte` (cf.
	// positions_porte_vehicules.go).
	vehicules, horsEmprise, naissancesHorsEmprise := ecarterVehiculesHorsEmprise(a.opt.Vehicles, a.emprise, a.opt.Fallbacks)
	attachVehicles(&a.doc, vehicules, a.sorted, a.reg,
		replayClock{origin: a.origin, step: a.step, frames: a.doc.FrameCount, fb: a.opt.Fallbacks})
	if c := a.doc.Coverage.Vehicles; c != nil {
		c.EchantillonsHorsEmprise, c.SpawnsHorsEmprise = horsEmprise, naissancesHorsEmprise
	}
	// LES TIRS DES JOUEURS EMBARQUES : la SECONDE porte des tirs, celle que la premiere ne
	// pouvait pas franchir (un occupant attache ne replique plus sa position de bipede, donc
	// `slotFor` n'a rien a poser sur la carte). Elle exige les episodes d'occupation ET les
	// trajectoires de vehicule : elle vient donc APRES `attachVehicles`, et elle met a jour la
	// couverture des tirs deja publiee — cf. vehicle_shots.go.
	attachVehicleShots(&a.doc, a.shotOrphans, a.reg,
		replayClock{origin: a.origin, step: a.step, frames: a.doc.FrameCount, fb: a.opt.Fallbacks})
}

// poserObjectifsVivants publie les calques portes par les pistes (drapeau, couronne VIP, crane,
// bombe), les objets d objectif libres, l etat des zones, puis les faits de l Assaut.
func (a *assemblage) poserObjectifsVivants() {
	// La VIE DES DRAPEAUX, sur les pistes PUBLIEES (le drapeau porte est a la position de son
	// porteur, et c'est celle-la que le client dessine) — cf. build_objectives_live.go.
	attachFlagCarries(&a.doc, a.opt, a.reg, replayClock{origin: a.origin, step: a.step, frames: a.doc.FrameCount, fb: a.opt.Fallbacks},
		a.equipes)
	// LA COURONNE VIP, sur les pistes PUBLIEES (la couronne est a la position de son porteur) —
	// gardee de mode par l'appelant (opt.Vip.Scanned), cf. vip_crown.go.
	attachVipCrown(&a.doc, a.opt, a.reg, replayClock{origin: a.origin, step: a.step, frames: a.doc.FrameCount, fb: a.opt.Fallbacks})
	// LE PORTEUR DU CRANE d'Oddball, sur les pistes PUBLIEES (le crane est a la position de son
	// porteur) — garde de mode par l'appelant (opt.Skull.Scanned), cf. skull_carries.go. Le crane
	// LIBRE (attachObjectiveObjects, ci-dessous) reste la couche POSITION ; ce calque-ci est la
	// couche VIVANTE par-dessus.
	attachSkullCarries(&a.doc, a.opt, a.reg, replayClock{origin: a.origin, step: a.step, frames: a.doc.FrameCount, fb: a.opt.Fallbacks},
		a.unnamed.deduced)
	// LE PORTEUR DE LA BOMBE d'Assaut, sur les pistes PUBLIEES (la bombe est a la position de
	// son porteur) — garde de mode par l'appelant (opt.Bomb.CarryScanned, TOUTES les variantes
	// de la famille bomb), source : le canal des armes tenues DEJA balaye (opt.WeaponChanges),
	// cf. bomb_carries.go.
	bombCarry := attachBombCarries(&a.doc, a.opt, a.reg,
		replayClock{origin: a.origin, step: a.step, frames: a.doc.FrameCount, fb: a.opt.Fallbacks}, a.unnamed.deduced)
	// LES OBJETS D'OBJECTIF LIBRES SONT POSÉS HORS DE LA GARDE DE MODE DU DRAPEAU, et c'est
	// délibéré : ce calque ne lit ni le statborg ni le fil des morts, donc rien de ce que cette
	// garde protège. La placer devant l'éteindrait sur Oddball — là où il sert.
	attachObjectiveObjects(&a.doc, a.opt, replayClock{origin: a.origin, step: a.step, frames: a.doc.FrameCount, fb: a.opt.Fallbacks})
	// L'ETAT DES ZONES, sur la MEME horloge que les positions et sur les captures DEJA posees
	// (`doc.Objectives`) — cf. build_zones.go.
	attachZoneStates(&a.doc, a.opt, a.reg, replayClock{origin: a.origin, step: a.step, frames: a.doc.FrameCount, fb: a.opt.Fallbacks})
	// L'ARMEMENT DE LA BOMBE, sur la meme horloge que les actions d'objectif et confronte aux
	// explosions DEJA posees (`doc.Objectives`) — garde de mode par l'appelant
	// (opt.Bomb.Scanned), cf. bomb_armings.go.
	attachBombArmings(&a.doc, a.opt, a.clock)
	// LES CINQ STATISTIQUES D'OBJECTIF DE L'ASSAUT, et les faits datés qui les portent. Elles
	// se calculent ICI parce que c'est le seul endroit où leurs quatre sources vivent en pleine
	// fidélité — la chronologie de portage EN MILLISECONDES (rendue par `attachBombCarries`),
	// les armements DÉJÀ publiés (`attachBombArmings`, juste au-dessus), les actions d'objectif
	// nommées et le recalage d'horloge. Aucun balayage de plus, aucune étape observée de plus
	// (cf. bomb_stats_document.go).
	attachBombStats(&a.doc, a.opt, a.reg, bombCarry)
	slog.Info("rejeu : episodes d'equipement actif",
		"viesPubliees", a.doc.Coverage.Equipment.TracksTotal,
		"viesCamo", a.doc.Coverage.Equipment.CamoLives,
		"episodesCamo", a.doc.Coverage.Equipment.CamoEpisodes,
		"viesSurbouclier", a.doc.Coverage.Equipment.OvershieldLives,
		"episodesSurbouclier", a.doc.Coverage.Equipment.OvershieldEpisodes)
}
