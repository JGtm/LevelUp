package replay

// film_scan.go — LES CINQ PHASES DE BALAYAGE D UN FILM, DANS L ORDRE DU FILM.
//
// SORTI DE `build_from_film.go` le 2026-09-14 (lot 1.0 du PLAN_DECODEUR_FILM) : le fichier
// depassait les 500 lignes du depot une fois la sequence extraite de `BuildFromFilm`. La coupure
// suit une responsabilite — la, le CHEMIN (verrou, largeurs, contexte du film, puis assemblage) ;
// ici, ce que chaque phase LIT. DEPLACEMENT PUR : aucune ligne de logique ne change.
//
// PRE-REQUIS DE TOUTES LES METHODES DE CE FICHIER : `filmdec.LockProcessDecode` tenu et les
// largeurs d axe de la carte installees. `scanFilmInputs` est leur unique appelant, et
// `BuildFromFilm` tient les deux (cf. build_from_film.go).
//
// GARDE-RAIL QUI LIT CE FICHIER : `observe_test.go` — il descend de `scanFilmInputs` dans ces
// methodes pour reconstituer la liste ORDONNEE des etapes observees.

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// balayerPositions lit la version du film, les teleportations, les positions, les creations de
// bipede, les tirs et les armes de depart. C'EST LA SEULE PHASE QUI PEUT ECHOUER : sans
// positions, il n'y a pas de rejeu.
func (s *filmScan) balayerPositions() error {
	// LA VERSION DU FILM VOYAGE AVEC L'ARTEFACT. Elle est lue dans l'en-tete du registre
	// (`chunk_00`), elle commande deja le decoupage du gamertag du fil des morts, et rien dans
	// l'artefact ne disait sous quelle grammaire il avait ete cuit.
	//
	// FILM SANS REGISTRE : nil dans la couverture, ET UN WARN ICI. C'est cette phase qui le
	// porte parce qu'elle est la seule du chemin a connaitre le `match_id` — `ScanDeaths`, qui
	// applique le decoupage historique dans ce cas, recoit un film deja charge et est appelee
	// deux fois par cuisson (revue adversariale du 2026-09-12, constat P2-4 : le WARN y etait
	// sans match et en double).
	if v, lue := filmdec.FilmMajorVersion(s.film); lue {
		s.in.FilmMajorVersion = &v
	} else {
		slog.Warn("rejeu : version de film illisible — le fil des morts retombe sur le decoupage "+
			"historique du gamertag", "match_id", s.matchID)
	}
	// TÉLÉPORTATIONS DU TRANSLOCATEUR : lues AVANT les positions, parce qu'elles servent
	// deux fois — le calque `translocations` du document, et l'EXEMPTION du filtre de
	// vitesse (décision D2) : une arrivée de téléportation part à 193-1540 m/s, le filtre à
	// 100 m/s la rejetait à tort (R3 : 51/51 rejets mesurés, tous à ±200 ms d'un événement
	// 117 du même slot). Sur un film sans tête 117, la liste est vide et le filtre est
	// bit à bit identique à l'actuel — invariance prouvée par test.
	//
	// L'ENTRÉE DE CATALOGUE Y DESCEND parce que la CHARGE de l'événement porte les deux
	// positions du va-et-vient, quantifiées aux bornes de la carte (R6 §1, validé 18/18) :
	// sans elle le scanner rendrait des quanta invérifiables, donc rien. Elle est garantie
	// non nulle ici (refus en tête de BuildFromFilm).
	s.in.Translocations = filmdec.ScanTranslocatorTeleports(s.film, s.opt.MapQuant)
	s.scan.TeleportExemptions = filmdec.TeleportExemptionsOf(s.in.Translocations)
	if len(s.in.Translocations) > 0 {
		slog.Info("translocateur : teleportations lues", "evenements", len(s.in.Translocations))
	}
	s.opt.observe("translocations", s.in.Translocations)
	positions, err := filmdec.ScanBipedPositions(s.film, s.scan)
	if err != nil {
		return err
	}
	s.in.Positions = positions
	s.opt.observe("positions", s.in.Positions)
	s.balayerCreations()
	// Les tirs sont décodés du MÊME film et sur la MÊME horloge que les positions ; leur
	// absence n'est pas fatale (un film sans event de tir reste un rejeu valide).
	shots, err := filmdec.ScanFireEvents(s.film)
	if err != nil {
		slog.Warn("events de tir illisibles — rejeu sans tirs", "err", err, "match_id", s.matchID)
		shots = nil
	}
	s.in.Fire = shots
	s.opt.observe("fire", s.in.Fire)
	// Armes portées : lues dans les keyframes du MÊME film, sur la MÊME horloge. Leur
	// absence n'est pas fatale (un rejeu sans armes reste un rejeu valide).
	loadouts, err := filmdec.ScanKeyframeLoadouts(s.film, loadoutFamilies())
	if err != nil {
		slog.Warn("keyframes illisibles — rejeu sans armes portées", "err", err, "match_id", s.matchID)
		loadouts = nil
	}
	s.in.Loadouts = loadouts
	s.opt.observe("loadouts", s.in.Loadouts)
	return nil
}

// balayerCreations lit les CRÉATIONS DE BIPÈDE : le lien DIRECT corps -> joueur, lu dans le
// default-state du record NEW `ti=35` (lot E2, 2026-09-08). MÊME bande de slots que les
// positions, pour que les deux lectures parlent des mêmes corps. Absence NON fatale — le
// registre dégrade alors sur le pont par morts et le PUBLIE
// (`coverage.bridge.bridgeNamedLives`).
func (s *filmScan) balayerCreations() {
	creations, creaStats, err := filmdec.ScanBipedCreations(s.fc)
	if err != nil {
		slog.Warn("creations de bipede illisibles — le registre degrade sur le pont par morts",
			"err", err, "match_id", s.matchID)
		creations, creaStats = nil, filmdec.BipedCreationStats{}
	} else {
		slog.Info("creation de bipede : records lus",
			"corps", creaStats.Slots, "ancres", creaStats.Anchors, "acceptes", creaStats.Accepted,
			"formeRefusee", creaStats.ShapeBad, "signatureEtrangere", creaStats.SignatureMismatch,
			"motAlternatif", creaStats.OtherWord, "motAlternatifCompte", creaStats.OtherWordCount,
			"porteFermee", creaStats.GateClosed, "tronques", creaStats.Truncated)
	}
	if creaStats.Anchors > 0 && creaStats.Accepted == 0 {
		// L'ALARME DU LECTEUR, ET C'EST LA SEULE : des ancres de la bonne FORME dont aucune ne
		// porte la constante de représentation. Un film dont les bipèdes portent un autre corps
		// que le Spartan multijoueur se lirait ainsi (cf. biped_creation.go).
		slog.Warn("creation de bipede : aucune signature reconnue sur des ancres presentes — "+
			"ce film porte-t-il une autre representation ?",
			"match_id", s.matchID, "ancres", creaStats.Anchors,
			"motAlternatifModal", creaStats.OtherWord, "compte", creaStats.OtherWordCount)
	}
	s.in.BipedCreations = creations
	s.opt.observe("bipedCreations", s.in.BipedCreations)
}

// balayerPortage lit ce qu'un joueur PORTE : changements d'arme, ramassages natifs, inventaire
// d'image-cle et inventaire delta.
func (s *filmScan) balayerPortage() {
	// PRISES ET LACHERS D'ARME : le composant d'identite d'arme n'entre au masque du flux
	// delta que lorsqu'un emplacement CHANGE (cf. filmdec/held_weapon_changes.go). Le
	// predicat de spawn vient des loadouts qu'on vient de lire : sans lui, la PREMIERE
	// emission d'un emplacement serait comptee comme une prise alors qu'elle peut n'etre que
	// la re-annonce d'une arme deja portee. Absence non fatale — le rejeu sort sans
	// ramassages, jamais avec des ramassages devines.
	weaponChanges, wStats, err := filmdec.ScanHeldWeaponChanges(s.fc, spawnSetFrom(s.in.Loadouts))
	if err != nil {
		slog.Warn("changements d arme illisibles — rejeu sans ramassages", "err", err, "match_id", s.matchID)
		weaponChanges = nil
	} else {
		slog.Info("ramassage : changements d arme lus",
			"recordsDelta", wStats.Records, "masquePorteur", wStats.WithComponent,
			"emissions", wStats.Emissions, "repetitions", wStats.Repeats)
	}
	s.in.WeaponChanges = weaponChanges
	s.opt.observe("heldWeaponChanges", s.in.WeaponChanges)
	s.opt.observe("heldWeaponChanges.stats", wStats)
	// RAMASSAGES NATIFS : l'evenement `biped_pickup` de la liste d'evenements, en tete des
	// paquets delta. AUTRE SOURCE que le canal ci-dessus (qui lit un composant du bipede
	// PENDANT la traversee d'un record) : celui-ci lit des bits que personne d'autre ne lit,
	// avant la trame. Il date a la milliseconde ET nomme le ramasseur. Absence non fatale.
	pickups, pStats, err := filmdec.ScanBipedPickups(s.fc)
	if err != nil {
		slog.Warn("ramassages natifs illisibles — rejeu sans ramassages natifs", "err", err, "match_id", s.matchID)
		pickups, pStats = nil, filmdec.BipedPickupStats{}
	} else {
		slog.Info("ramassage natif : evenements lus",
			"paquets", pStats.Packets, "type9", pStats.Type9, "type8", pStats.Type8,
			"publies", pStats.Published, "listesMultiples", pStats.MultiEvent,
			"refusesSansRef", pStats.RefusedNoRef, "refusesSansIdentifiant", pStats.RefusedNoCatalog,
			"refusesHorsBande", pStats.RefusedOffBand, "refLargeInattendue", pStats.UnexpectedWideRef)
	}
	s.in.Pickups, s.in.PickupStats = pickups, pStats
	s.opt.observe("pickups", s.in.Pickups)
	s.opt.observe("pickups.stats", s.in.PickupStats)
	s.balayerInventaire()
}

// balayerInventaire lit l'inventaire complet des images-cles, puis son suivi dans les paquets
// delta. MÊMES images-clés, MÊME horloge, même record de biped que les armes portées.
func (s *filmScan) balayerInventaire() {
	// Absence non fatale — un rejeu sans grenades reste un rejeu valide.
	inventory, invStats, err := ScanKeyframeInventory(s.film, loadoutFamilies(), 0)
	if err != nil {
		slog.Warn("inventaire illisible — rejeu sans grenades ni munitions", "err", err, "match_id", s.matchID)
		inventory = nil
	} else {
		slog.Info("inventaire : lectures de keyframe",
			"chunks", invStats.Chunks, "chunksIllisibles", invStats.ChunksUnread,
			"imagesCles", invStats.Keyframes, "records", invStats.Records,
			"grenadesParAncre", invStats.GrenadesByAnchor, "grenadesParPosition", invStats.GrenadesByPosition)
	}
	s.in.Inventory = inventory
	s.opt.observe("inventory", s.in.Inventory)
	s.opt.observe("inventory.stats", invStats)
	// Inventaire suivi dans les paquets DELTA : les compteurs de grenades (i22) et le jeu
	// selectionne (i47), transmis AU CHANGEMENT donc places la ou l'etat bouge. Absence non
	// fatale — l'axe des grenades retombe sur les seules images-cles.
	invDeltas, dStats, err := filmdec.ScanInventoryDeltas(s.fc)
	if err != nil {
		slog.Warn("inventaire delta illisible — grenades sans rafraichissement entre images-cles",
			"err", err, "match_id", s.matchID)
		invDeltas = nil
	} else {
		slog.Info("inventaire delta : lectures",
			"recordsDelta", dStats.Records, "masqueAvecI22", dStats.WithI22,
			"i22Lues", dStats.I22Read, "i22Implausibles", dStats.Implausible,
			"masqueAvecI47", dStats.WithI47, "i47Lues", dStats.I47Read,
			"accord", dStats.Accord, "accordVerifies", dStats.AccordChecked,
			"canalMunitionsRefuse", dStats.AmmoRefused)
	}
	s.in.InventoryDeltas = invDeltas
	s.in.InventoryDeltaAmmoRefused = dStats.AmmoRefused
	s.opt.observe("inventoryDeltas", s.in.InventoryDeltas)
	s.opt.observe("inventoryDeltas.stats", dStats)
}

// balayerCapacites lit les quatre canaux delta de l'equipement porte — identite, changements,
// camouflage, grappin — puis les impulsions et les charges.
func (s *filmScan) balayerCapacites() {
	// Identite de la capacite portee : lue dans les paquets DELTA, sur la MEME horloge. Rare
	// (une transmission par vie environ) mais elle porte le rang COMPLET, la ou les images-cles
	// ne voient que 16..23. Absence non fatale — le rejeu retombe sur cette seule fenetre.
	abilityRanks, aStats, err := filmdec.ScanAbilityRanks(s.fc)
	if err != nil {
		slog.Warn("identites de capacite illisibles — rejeu sans rang complet", "err", err, "match_id", s.matchID)
		abilityRanks = nil
	} else {
		slog.Info("capacites : lectures d i48",
			"recordsDelta", aStats.Records, "masqueAvecI48", aStats.WithI48,
			"lues", aStats.Read, "illisibles", aStats.Unread, "sansIdentite", aStats.Gated)
	}
	s.in.AbilityRanks = abilityRanks
	s.opt.observe("abilityRanks", s.in.AbilityRanks)
	s.opt.observe("abilityRanks.stats", aStats)
	// RAMASSAGES ET CONSOMMATIONS D'EQUIPEMENT : meme composant qu'au-dessus (i48), autre
	// question — non plus « que porte ce joueur » mais « que vient-il de ramasser ou d'user ».
	// Le temoin de NAISSANCE vient des positions BRUTES lues plus haut : sans lui, une
	// reapparition equipee serait comptee comme un ramassage, ce qui double le decompte sur
	// les modes ou les joueurs renaissent equipes. Absence non fatale.
	equipChanges, eStats, err := filmdec.ScanEquipmentChanges(s.fc, birthOfLives(s.in.Positions))
	if err != nil {
		slog.Warn("changements d equipement illisibles — rejeu sans ramassages d equipement",
			"err", err, "match_id", s.matchID)
		equipChanges, eStats = nil, filmdec.EquipmentChangeStats{}
	} else {
		slog.Info("equipement : changements lus",
			"emissions", eStats.Walk.Read, "vies", eStats.Lives,
			"ramassages", eStats.Taken, "consommations", eStats.Spent,
			"reapparitions", eStats.Spawned, "manqueesEstimees", eStats.MissedEstimate)
	}
	s.in.EquipmentChanges, s.in.EquipmentChangeStats = equipChanges, eStats
	s.opt.observe("equipmentChanges", s.in.EquipmentChanges)
	s.opt.observe("equipmentChanges.stats", s.in.EquipmentChangeStats)
	s.balayerEtatsActifs()
	s.balayerImpulsionsEtCharges()
}

// balayerEtatsActifs lit le camouflage et le grappin : deux voies des paquets DELTA, sur la MEME
// horloge que les positions.
func (s *filmScan) balayerEtatsActifs() {
	// Etat du camouflage : la voie i28 queue[1] (cf. filmdec/camo_state.go). Absence non fatale
	// — le rejeu sort sans episodes de camouflage, jamais avec des episodes devines.
	camoStates, cStats, err := filmdec.ScanCamoStates(s.fc)
	if err != nil {
		slog.Warn("etat de camouflage illisible — rejeu sans episodes de camo", "err", err, "match_id", s.matchID)
		camoStates = nil
	} else {
		slog.Info("camouflage : lectures d i28 queue[1]",
			"recordsDelta", cStats.Records, "masqueAvecI28", cStats.WithI28,
			"lues", cStats.Read, "illisibles", cStats.Unread, "sansVoie", cStats.NoChannel)
	}
	s.in.CamoStates = camoStates
	s.opt.observe("camoStates", s.in.CamoStates)
	s.opt.observe("camoStates.stats", cStats)
	// Evenements de grappin : le corps tag==3 d'i59 (cf. filmdec/grapple_state.go). Absence non
	// fatale — le rejeu sort sans tractions de grappin, jamais avec des tractions devinees.
	grappleReads, gStats, err := filmdec.ScanGrappleReads(s.fc)
	if err != nil {
		slog.Warn("evenements de grappin illisibles — rejeu sans tractions", "err", err, "match_id", s.matchID)
		grappleReads = nil
	} else {
		slog.Info("grappin : lectures d i59 tag==3",
			"recordsDelta", gStats.Records, "masqueAvecI59", gStats.WithI59,
			"lues", gStats.Read, "illisibles", gStats.Unread,
			"tag3", gStats.Tag3, "corpsCasses", gStats.BodyBroken)
	}
	s.in.GrappleReads = grappleReads
	s.opt.observe("grappleReads", s.in.GrappleReads)
	s.opt.observe("grappleReads.stats", gStats)
}

// balayerImpulsionsEtCharges lit l'USAGE des capacites : les impulsions et les charges
// restantes. L'IDENTITE, elle, vient d'i48 (deja balaye par balayerCapacites).
func (s *filmScan) balayerImpulsionsEtCharges() {
	// IMPULSIONS DE CAPACITE : le corps tag==1 des MEMES composants (i57 et son jumeau non
	// predit i59), lu dans les paquets DELTA sur la MEME horloge (cf.
	// filmdec/ability_impulses.go). C'est le canal d'usage du PROPULSEUR, mesure au lot R8.
	// Absence non fatale — le rejeu sort sans impulsions, jamais avec des impulsions devinees.
	impulses, iStats, err := filmdec.ScanAbilityImpulses(s.fc)
	if err != nil {
		slog.Warn("impulsions de capacite illisibles — rejeu sans impulsions", "err", err, "match_id", s.matchID)
		impulses, iStats = nil, filmdec.AbilityImpulseStats{}
	} else {
		slog.Info("capacites : lectures de tag d i57/i59",
			"recordsDelta", iStats.Records, "masqueAvecI57", iStats.WithI57,
			"masqueAvecI59", iStats.WithI59, "lues", iStats.Read, "illisibles", iStats.Unread,
			"tag1", iStats.Tag1, "composantAbsent", iStats.Absent)
	}
	s.in.AbilityImpulses, s.in.AbilityImpulseStats = impulses, iStats
	s.opt.observe("abilityImpulses", s.in.AbilityImpulses)
	// CHARGES D'EQUIPEMENT RESTANTES : les emplacements ARMES du composant i56, lus dans les
	// paquets DELTA sur la MEME horloge (cf. filmdec/ability_charges.go). C'est le canal des
	// charges mesure au lot R11. Absence non fatale — le rejeu sort sans releve de charges,
	// jamais avec des charges devinees.
	charges, chStats, err := filmdec.ScanAbilityCharges(s.fc)
	if err != nil {
		slog.Warn("charges d equipement illisibles — rejeu sans releve de charges", "err", err, "match_id", s.matchID)
		charges, chStats = nil, filmdec.AbilityChargeStats{}
	} else {
		slog.Info("capacites : lectures d i56",
			"recordsDelta", chStats.Records, "masqueAvecI56", chStats.WithI56,
			"lues", chStats.Read, "illisibles", chStats.Unread,
			"emplacementsArmes", chStats.Armed, "composantAbsent", chStats.Absent)
	}
	s.in.AbilityCharges, s.in.AbilityChargeStats = charges, chStats
	s.opt.observe("abilityCharges", s.in.AbilityCharges)
}

// balayerMonde lit la lunette, puis les objets du monde : poses d'equipement, socles, vehicules,
// et les trois calques gardes par le mode (marqueur de portage, zones, anneau de la bombe).
func (s *filmScan) balayerMonde() {
	// LA LUNETTE (schema 24) : les bascules vivent dans la liste d'evenements en tete de
	// paquet, pas dans les records — un balayage separe, sans verrou (il ne touche aucun etat
	// global de decodage). Le maintien est borne : au-dela, on cesse d'affirmer plutot que de
	// prolonger une entree dont la sortie n'a pas ete lue (cf. filmdec.ZoomStateAt).
	//
	// LA RECONSTRUCTION (`buildScopedLookup`) VIT DESORMAIS DANS `FilmInputs.applyTo` (lot 1.0) :
	// elle est PURE, et la porter la permet au fixture de figer une LISTE d'evenements plutot
	// qu'une fermeture. Son cout — un O(n) sur les evenements qu'on vient de balayer — quitte
	// donc la mesure de l'etape `zoomEvents` ; il ne lit aucun octet de film.
	s.in.ZoomEvents = filmdec.ScanZoomEvents(s.film)
	s.opt.observe("zoomEvents", s.in.ZoomEvents)
	// POSES d'equipement : records de CREATION de l'archetype 37, sur la MEME horloge
	// (cf. equipment_placements.go — decodage, journal et refus y vivent ensemble).
	s.in.Placements, s.in.PlacementStats = decodeFilmPlacements(s.fc, s.matchID, &s.world)
	s.opt.observe("placements", s.in.Placements)
	s.opt.observe("placements.stats", s.in.PlacementStats)
	// SOCLES : archetypes 42 (armes) et 37 (power-ups), sur la MEME horloge, AUX LARGEURS MPP que
	// la calibration des POSES vient de mesurer sur ce film (cf. build_ground_weapons.go).
	mpp := s.in.PlacementStats.Calibration.Widths
	s.in.Pads = decodeFilmPadScans(s.fc, s.matchID, &s.world, mpp)
	s.opt.observe("pads", s.in.Pads)
	// VEHICULES : archetype 40, sur la MEME horloge et AUX MEMES largeurs MPP que les socles —
	// le mot d'identite du chassis se lit derriere les memes deux champs de largeur variable
	// (cf. build_vehicles.go). CINQ lectures du film deja charge, aucune E/S.
	s.in.Vehicles = decodeFilmVehicleScan(s.fc, s.matchID, &s.world, mpp)
	s.opt.observe("vehicles", s.in.Vehicles)
	s.balayerCalquesGardes()
}

// balayerCalquesGardes lit les trois calques que l'APPELANT commande : le marqueur de portage du
// drapeau, l'etat des zones et l'anneau d'armement de la bombe. Sans la garde correspondante
// dans `Options`, aucun octet n'est lu — ce paquet ne devine aucun mode.
func (s *filmScan) balayerCalquesGardes() {
	// MARQUEUR DE PORTAGE : le controle independant du calque du drapeau, lu aux images-cles du
	// MEME film — sur les seuls films de CTF (cf. build_objectives_live.go).
	s.in.FlagMarks = decodeFilmCarrierMarks(s.film, s.matchID, s.opt.Flag)
	s.opt.observe("carrierMarks", s.in.FlagMarks)
	// PROPRIETES RESEAU ti=13 : l'etat des zones (jauge de capture, proprietaire), lu dans les
	// paquets delta du MEME film — sur les seuls matchs dont l'appelant a fourni le catalogue de
	// zones (cf. build_zones.go).
	s.in.ZoneReads = decodeFilmZoneReads(s.fc, s.matchID, len(s.opt.Zone.Zones))
	s.in.ZoneScanned = len(s.opt.Zone.Zones) > 0
	s.opt.observe("zoneReads", s.in.ZoneReads)
	// ANNEAU D'ARMEMENT ti=12 : la jauge d'armement de la bombe, lue dans les paquets delta du
	// MEME film — sur les seuls matchs que l'appelant reconnait Assaut armable (cf.
	// bomb_armings.go ; jamais One Bomb, ou le canal ne tient pas).
	s.in.BombReads = decodeFilmBombReads(s.fc, s.matchID, s.opt.Bomb)
	s.opt.observe("bombReads", s.in.BombReads)
}

// balayerPont lit les lancers, les projectiles, et LE PONT D'IDENTITE : le fil des morts, la
// table d'index de joueur et l'origine d'horloge.
func (s *filmScan) balayerPont() {
	// Lancers de grenade : décodés des paquets delta du MÊME film, sur la MÊME horloge.
	// Absence non fatale, comme les tirs et les armes portées.
	grenades, err := filmdec.ScanGrenadeThrows(s.film)
	if err != nil {
		slog.Warn("paquets delta illisibles — rejeu sans lancers de grenade", "err", err, "match_id", s.matchID)
		grenades = nil
	}
	s.in.Grenades = grenades
	s.opt.observe("grenades", s.in.Grenades)
	// Trajectoires de projectile : memes chunks, meme horloge. Absence non fatale.
	proj, err := filmdec.ScanProjectiles(s.film, &s.world)
	if err != nil {
		slog.Warn("projectiles illisibles — rejeu sans trajectoires", "err", err, "match_id", s.matchID)
		proj = nil
	}
	s.in.Projectiles = proj
	s.opt.observe("projectiles", s.in.Projectiles)
	// Le fil des morts NOMME les vies. Sans lui, le pont est vide et NI les tirs NI les lancers
	// ne sont publiés : ce n'est pas une dégradation cosmétique, d'où un warn explicite.
	deaths, err := ScanDeaths(s.film)
	if err != nil {
		slog.Warn("fil des morts illisible — aucun tir ni lancer ne sera publie",
			"err", err, "match_id", s.matchID)
		deaths = nil
	}
	s.in.Deaths = deaths
	s.opt.observe("deaths", s.in.Deaths)
	// L'index de joueur SE LIT dans le film (cf. player_index.go) : le roster vient du fil des
	// morts, et les 5 bits qui précèdent chaque xuid donnent son index. Sans cette table, aucun
	// tir ni lancer n'est publié — comme sans le fil des morts.
	if len(deaths) > 0 {
		idx, err := ScanPlayerIndices(s.film, rosterOf(deaths, s.opt.RosterXUIDs))
		if err != nil {
			slog.Warn("index de joueur illisible — aucun tir ni lancer ne sera publie",
				"err", err, "match_id", s.matchID)
		}
		table, collisions := injectiveOrEmpty(idx)
		if collisions > 0 {
			slog.Warn("index de joueur NON INJECTIF — table ecartee",
				"collisions", collisions, "match_id", s.matchID)
		}
		s.in.PlayerIndices = table
	}
	s.opt.observe("playerIndices", s.in.PlayerIndices)
	// L'origine d'horloge du film : deux en-têtes de paquet, aucune estimation (cf.
	// origin.go). Son absence n'est pas fatale — le document sort sans origine, et le
	// client retombe sur l'appariement.
	clockUS, err := ScanClockOrigin(s.film)
	if err != nil {
		slog.Warn("origine d'horloge illisible — rejeu sans origine publiee", "err", err, "match_id", s.matchID)
		clockUS = 0
	}
	s.in.FilmClockOriginUS = clockUS
	s.opt.observe("clockOrigin", s.in.FilmClockOriginUS)
}
