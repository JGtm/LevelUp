package replaydiff

// polarite_table.go — LA POLARITE DECLAREE DE CHAQUE MESURE DE COUVERTURE (lot R4, 2026-09-28).
//
// Une ligne par bloc de couverture ; chaque bloc range TOUTES ses feuilles numeriques dans
// l'une des quatre classes de `Polarite` (polarite.go). Les feuilles sont les cles JSON telles
// que `aplatir` les pose, sous le prefixe du bloc :
//
//	`x`            une feuille numerique ;
//	`x/n`          la longueur d'un tableau ;
//	`m.*`          toute cle d'une map (compte par famille, par cause...) ;
//	`m.cle`        une cle NOMMEE de cette map, qui l'emporte sur `m.*` ;
//	`m.*/suffixe`  les cles composees d'une map (`<famille>/<origine>`) qui finissent ainsi.
//
// LA TABLE EST TOTALE ET C'EST UN RATCHET QUI LE TIENT : `polarite_ratchet_test.go` derive
// l'inventaire des feuilles de la forme du document (`film/replay/testdata/document_shape.golden`)
// et rougit sur toute feuille sans polarite, comme sur toute entree qui ne designe plus rien.
// Ajouter un compteur a la couverture oblige donc a le classer ici, dans le commit qui l'ajoute.
//
// Les listes sont des chaines separees par des espaces (une par classe et par bloc) : une table
// de donnees repete par nature ses noms de feuille (`published` a vingt blocs), et une chaine par
// classe garde la ligne lisible sans cent litteraux repetes.
//
// CRITERE DE CLASSE, applique feuille par feuille sur le commentaire de son champ
// (`internal/games/halo_infinite/film/replay`) :
//
//	echec       ce que la cuisson N'A PAS su faire : rejet, trou, ambiguite, contradiction,
//	            abstention, repli qu'une lecture fait baisser. Plus = pire.
//	succes      ce qui a ete LU, publie, nomme, confirme, rattache. Plus = mieux.
//	neutre      population, total, denominateur, fait du jeu ou de l'oracle (captures, prises
//	            du statborg, catalogue), VENTILATION d'un total publie par nature ou par VOIE
//	            (`namedBy*`, `closedBy*`, `taken`/`spent`...). Ni gain ni perte : un changement.
//	telemetrie  un reglage ou une forme de l'outil, pas une mesure du match. Un changement que
//	            le verdict du gate de corpus affiche sans le compter.

// blocPolarites classe les feuilles d'un ou plusieurs blocs de meme forme.
type blocPolarites struct {
	// Blocs : les prefixes auxquels la ligne s'applique, point final compris.
	Blocs []string
	// Une liste par classe, feuilles separees par des espaces.
	Echecs, Succes, Neutres, Telemetrie string
}

// Prefixes de bloc repetes (goconst) : la couverture de tete, et celle du calque d'Assaut.
const (
	racineCouverture = "coverage."
	racineAssaut     = "bombStats.coverage."
)

// tablePolarites — relevee sur pieces le 2026-09-28 (forme du document au schema 76).
var tablePolarites = []blocPolarites{
	{Blocs: []string{racineCouverture},
		Echecs:  "fallbacks/n",
		Neutres: "filmMajorVersion"},
	{Blocs: []string{"coverage.shots.", "coverage.grenades.", "coverage.objectives."},
		Echecs:  "noSlot ambiguous outOfWindow unpublished refusedByRoster unitOtherIndex",
		Succes:  "attached byUnit",
		Neutres: "available"},
	{Blocs: []string{"coverage.abilities."},
		Echecs: "scanNoise unpublished",
		Succes: "reads published"},
	{Blocs: []string{"coverage.abilityCharges."},
		Echecs:  "unpublished noIdentity noResolver",
		Succes:  "reads published",
		Neutres: "beforeOrigin otherFamily"},
	{Blocs: []string{"coverage.abilityImpulses."},
		Echecs:  "unpublished noIdentity noResolver scan.unread",
		Succes:  "reads published scan.records scan.read scan.tag1",
		Neutres: "episodes beforeOrigin otherFamily scan.withI57 scan.withI59"},
	{Blocs: []string{"coverage.birthLoadouts."},
		Echecs:  "desync overflow unconfirmed noWeaponComponent noLife noDisplayable",
		Succes:  "closed read published",
		Neutres: "creations snapped beforeOrigin nonWeapon unarmedGrants"},
	{Blocs: []string{"coverage.bombArmings."},
		Echecs:  "outOfWindow",
		Succes:  "reads rises armed published detonationsCovered",
		Neutres: "belowFull pairMerged detonations"},
	{Blocs: []string{"coverage.bombCarries."},
		Echecs:  "open noBridge outOfWindow carrierAbsent",
		Succes:  "events carries closed",
		Neutres: "periods byDeath"},
	{Blocs: []string{"coverage.bridge."},
		Echecs: "bridgeNamedLives closedContested closedRefused discordant indexDisagreements " +
			"slotCollisions unnamedLives unnamedLivesContested",
		Succes: "bodiesWithCreation concordant deathOffsetMatched directByCreation " +
			"directByCreationPropagated fromReading indexReadings livesNamed",
		Neutres: "closedByRespawn closedByShot deathOffsetMs deathOffsetRunnerUp livesTotal " +
			"namedByNextLife namedByPreviousLife namedBySlotBridge slots"},
	{Blocs: []string{"coverage.continuousFire."},
		Echecs: "holes holeRuns holesUnlocated holesOpenViewB holesOverflow holesKind holesBlockBC " +
			"holesCap holesNotClosing burstsWithHole innerHoles heldHoleMs noPlayer ambiguous " +
			"noTrack weaponUnknown",
		Succes: "reached closed entries withAction firing burstsRead published shots",
		Neutres: "packets onVehicle onFoot otherInput vehicleNoWeapon notContinuous empty byPlace " +
			"clippedToMount"},
	{Blocs: []string{"coverage.deathsPaths.walk.", "coverage.deathsPaths.directScan."},
		Succes:  "matched published",
		Neutres: "population"},
	{Blocs: []string{"coverage.decoder.registry."},
		Neutres: "blocks namedSlots"},
	{Blocs: []string{"coverage.equipment."},
		Succes:  "camoEpisodes camoLives overshieldEpisodes overshieldLives",
		Neutres: "tracksTotal"},
	{Blocs: []string{"coverage.equipmentChanges."},
		Echecs:  "counterJumps livesFirstOffSpec missedEstimate repeats",
		Succes:  "decoded lives published",
		Neutres: "beforeOrigin recovered spawned spent taken"},
	{Blocs: []string{"coverage.flagCarries."},
		Echecs: "ambiguousCarrierKills ambiguousHomecomings ambiguousReturns ambiguousSlot " +
			"carrierTeamUnknown closedOverlaps noBridge noTrack open outOfWindow overlaps " +
			"filmBaseContradict ownFlagRefused unjudgedCarrierKills unresolved",
		Succes: "carries closed dropsRepositioned filmBaseAgree gaugePaired gaugePoints gaugeReads gaugeSlots " +
			"gaugeSpans markerConfirmed objectLives openConfirmed",
		Neutres: "assignedByPlay bursts captures closedByHandoff closedByHome closedByObject " +
			"closedByReturn dropsWithheld homeByObject markerObserved neutralBirths openObserved " +
			"filmBases openings spawns spawnsFromFilm steals teamBirths"},
	{Blocs: []string{"coverage.grapple."},
		Echecs:  "brokenBodies",
		Succes:  "heavyReads lightReads pullLives pulls",
		Neutres: "unpairedFires"},
	{Blocs: []string{"coverage.grenadeReads."},
		Echecs: "unpublished",
		Succes: "fromDelta fromKeyframe"},
	{Blocs: []string{"coverage.groundWeaponItems."},
		Echecs:  "endOpen",
		Succes:  "ammoRead dropperNamed endPickup endSeen objects pickupLinked published",
		Neutres: "atRest takesTotal"},
	{Blocs: []string{"coverage.groundWeapons."},
		Echecs: "unknown",
		Succes: "accepted cycles dated kept pads powerupAccepted powerupKept powerupPads",
		Neutres: "anchors atRest clusters dropped never objectives occupancies rejected slots " +
			"spawned"},
	{Blocs: []string{"coverage.inventory."},
		Echecs:  "unpublished",
		Succes:  "decoded published",
		Neutres: "droppedBeforeOrigin"},
	{Blocs: []string{"coverage.keyframes."},
		Echecs:  "contradictoryProofs framedAbsentBipeds refutations slides",
		Succes:  "bipeds records",
		Neutres: "elections jumps keyframes neighbors resyncs"},
	{Blocs: []string{"coverage.objectiveObjects."},
		Echecs:  "outOfAxis",
		Succes:  "lives points",
		Neutres: "declared motionless"},
	{Blocs: []string{"coverage.padDating."},
		Echecs:  "ambiguous uncovered",
		Succes:  "dated named",
		Neutres: "occupations powerupOccupations"},
	{Blocs: []string{"coverage.pickups."},
		Echecs: "multiEvent originUnknown refused unknownFamilies spawnerByPointKind.unknown",
		Succes: "decoded named originGround originSpawner published",
		Neutres: "beforeOrigin items mapCatalogPoints unarmedGrants weapons " +
			"spawnerByPointKind.*"},
	{Blocs: []string{"coverage.placements."},
		Echecs: "endOpen unknown byCause.both byCause.none byCause.no_owner byFamilyOrigin.*/unknown",
		Succes: "confirmed endSeen lives named placements spawnEvents spawnLists withHeading " +
			"withOwner byFamily.* byCause.spawn_event byCause.death_written byCause.taken_written",
		Neutres: "anchors deployed dropped other byCause.* byCause.manifest_piece byFamilyOrigin.*"},
	{Blocs: []string{"coverage.projectiles."},
		Echecs: "truncated",
		Succes: "published tracks"},
	{Blocs: []string{"coverage.score."},
		Echecs:  "roundsContradicted/n roundsContradictedRecords",
		Succes:  "points",
		Neutres: "rounds roundsWritten/n"},
	{Blocs: []string{"coverage.seats."},
		Echecs: "bornesDifferees chevauchements depassements entitesContestees entitesNonLiees " +
			"identitesHorsRoster imagesClesDouteuses placesEnTrop placesOuvertes " +
			"presencesParLesVies sansEquipe sansPlace sansPresence tirsContestes trousDEntite",
		Succes: "botsSuccesseurs lus placesTirs relaisBornes tirsParPlace",
		Neutres: "apparies arrivants capacite entrees occupantsMax presencesCloses " +
			"reprisesEcrites sieges"},
	{Blocs: []string{"coverage.skullCarries."},
		Echecs:  "carrierAbsent noBridge open outOfWindow",
		Succes:  "carries closed",
		Neutres: "grabs trains"},
	{Blocs: []string{"coverage.stances."},
		Echecs: "desyncs dropped eventPacketsUnlocated forgottenBindings refusedNews " +
			"refusedNewFalseReads refusedNewLostCreations refusedNewUndecided",
		Succes:     "intervals jumpsDerived lives reads records",
		Neutres:    "byKind.* jumpEpisodes tracksTotal",
		Telemetrie: "mapWidths/n"},
	{Blocs: []string{"coverage.t0Film."},
		Neutres: "burst marginMs moving tracks"},
	{Blocs: []string{"coverage.teams."},
		Echecs:  "contradiction divergences rejected tracksSlotAmbiguous unread",
		Succes:  "accord film tracksNamed",
		Neutres: "noTeam records silence tracks"},
	{Blocs: []string{"coverage.tracks."},
		Echecs:     "gapMs gaps horsEmprise refusedMinPoints refusedPoints slotsDesarmes",
		Succes:     "published publishedPoints",
		Neutres:    "avantCreation slotsArmes viesAvantPremiereCreation",
		Telemetrie: "minPoints"},
	{Blocs: []string{"coverage.translocations."},
		Echecs:  "unpublished",
		Succes:  "events positioned published",
		Neutres: "beforeOrigin"},
	{Blocs: []string{"coverage.vehicles."},
		Echecs: "ambiguous cycleMissing deathsTailDesync deathsUnmatched " +
			"echantillonsAuTraversDUnSilence echantillonsHorsEmprise endUnknown familyUnknown " +
			"noPosition samplesAfterEnd shotsAmbiguous shotsNoRide shotsUnplaced " +
			"silencesNonTranches spawnsHorsEmprise turretCarrierBirthMismatch " +
			"turretRidesAlreadyAboard turretRidesDropped " +
			"turretRidesOutOfWindow unknownChassis.*",
		Succes: "aimReads aimSamples cycleGaps cycles deathsMatched deathsRead endDestroyed " +
			"familyResolved lives published rides ridesNamed ridesRead ridesWithAim ridesWithSeat " +
			"samples shots shotsByUnit shotsVehicleWeapon turretRides turrets turretsOnCarrier " +
			"variants vehiclesRidden withChassis withHeading withSpawn",
		Neutres: "aimRideFrames cycleLocations endFilmEnd merged ridesProximity shotsByUnitNoRide " +
			"shotsOnCarrier"},
	{Blocs: []string{"coverage.vipCrown."},
		Echecs:  "noBridge open outOfWindow",
		Succes:  "closed periods",
		Neutres: "closedByDeath closedBySelection selections"},
	{Blocs: []string{"coverage.weaponChanges."},
		Succes:  "decoded published",
		Neutres: "beforeOrigin dropped restated swapped taken unarmedGrants"},
	{Blocs: []string{"coverage.zones."},
		Echecs:  "ambiguousZone capturerElectionDisagreed noPosition ownerUnpaired ownerVoteDisagreed unknownOwner unpaired",
		Succes:  "attributed capturerNamed gaugePoints letters ownerAgreed ownerNamed paired spans",
		Neutres: "captures catalog hillPeriods outside ownerChecked slots"},
	{Blocs: []string{racineAssaut},
		Echecs: "armingsAmbiguous armingsNoBridge armingsNoCarrier armingsNoClock periodsNoBridge " +
			"periodsOpen",
		Succes: "armingsAttributed killsOnCarrier",
		Neutres: "armings armingsByActiveCarry armingsByDrop detonations kills periods " +
			"periodsByDeath players"},
}

// polaritesHeritees — LES FEUILLES QUE LA FORME COURANTE NE PORTE PLUS, et que portent encore les
// artefacts de REFERENCE d'un gate (cuits a un schema anterieur). Leur disparition est attendue :
// la classer evite qu'elle sorte en perte. Chaque entree dit quand et pourquoi elle est partie ;
// le ratchet refuse qu'une feuille heritee figure aussi dans la forme courante.
//
//   - `vehicles.ridesFromEvent` / `ridesMixed` / `ridesFromGap` : ventilation des episodes par
//     PRECISION, retiree au schema 67 (5.10.6, l'occupation est LUE, `ridesRead` /
//     `ridesProximity`). Mesure 2026-09-28 : 63 artefacts du parc local la portent.
//   - `vehicles.turretRidesNotRideable` : refus « porteur non pilotable », nul depuis le schema 69
//     (lot M7b, branche retiree), champ retire au schema 77 (2026-10-02, chantier Falcon de
//     Behemoth). Tout artefact 69 a 76 le porte, a 0.
var polaritesHeritees = []blocPolarites{
	{Blocs: []string{"coverage.vehicles."},
		Neutres: "ridesFromEvent ridesMixed ridesFromGap turretRidesNotRideable"},
}
