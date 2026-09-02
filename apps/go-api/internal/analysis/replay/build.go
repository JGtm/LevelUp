package replay

import (
	"log/slog"
	"math"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// DefaultFrameIntervalMS est le pas de la grille de rééchantillonnage, en millisecondes.
// Les records du film arrivent à ~60 Hz par entité : à 100 ms (10 Hz) le rendu reste
// fluide (le client interpole) tout en divisant le volume de points par ~4.
const DefaultFrameIntervalMS = 100

// DefaultMinPoints est le nombre minimal de points pour qu'une vie soit publiée : une
// track d'un seul échantillon n'est pas une trajectoire.
const DefaultMinPoints = 2

// coordScale arrondit les coordonnées au centimètre : le quantum du décodeur est de
// ~1,4 cm, deux décimales ne perdent donc rien et allègent nettement le JSON.
const coordScale = 100

// BuildFromPositions assemble le document à partir de positions déjà décodées. PUR
// (aucune I/O) : c'est le cœur testable de l'assemblage.
//
// TIMELINE : les positions portent l'horodatage du paquet en MICROSECONDES ; elles sont
// rééchantillonnées sur une grille uniforme de FrameIntervalMS relative au premier paquet.
// Point.T est donc l'index de frame, et FrameIntervalMS donne son échelle réelle.
func BuildFromPositions(matchID, titleSlug string, pos []filmdec.BipedPosition,
	fire []filmdec.FireEvent, opt Options) ReplayDocument {
	interval := opt.frameIntervalMS()
	doc := ReplayDocument{
		SchemaVersion:   SchemaVersion,
		MatchID:         matchID,
		TitleSlug:       titleSlug,
		FrameIntervalMS: interval,
		Geometry:        opt.Geometry,
		GeometryBounds:  geometryBounds(opt.Geometry),
		Structure:       opt.Structure,
		StructureBounds: surfaceBounds(opt.Structure),
	}
	if len(pos) == 0 {
		return doc
	}
	sorted := append([]filmdec.BipedPosition(nil), pos...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TimestampUS < sorted[j].TimestampUS })

	origin := sorted[0].TimestampUS
	step := uint64(interval) * 1000
	doc.Tracks = decimateTracks(sorted, origin, step, opt.minPoints(), opt.Scoped)
	doc.FrameCount = frameSpan(sorted, origin, step)
	doc.DurationMS = doc.FrameCount * interval
	doc.Bounds = boundsOf(doc.Tracks)
	// Les tirs sont rattachés sur les positions NON décimées (le rattachement se joue à
	// ~120 ms, la grille du rejeu est à 100 ms : décimer d'abord perdrait des tireurs).
	// LE PONT slot -> joueur vient du seul fil des morts (cf. owners.go). Il conditionne les
	// tirs ET les lancers : le construire une seule fois, et le partager.
	// Les TIRS entrent dans la construction du pont — non pour désigner un tireur (l'événement
	// porte déjà son auteur), mais parce que la fermeture A a besoin de savoir QUAND un joueur
	// agit sans avoir de corps nommé. Cf. closures.go.
	own := buildOwners(indexBySlot(sorted), opt.Deaths, opt.PlayerIndices, fireRefs(fire))
	// L'IDENTITÉ se pose sur les traces dès que le pont existe : sans elle, un client ne peut
	// ni nommer un joueur, ni regrouper ses vies, ni colorer une équipe.
	nameTracks(doc.Tracks, own.SlotXUID)
	doc.Roster = buildRoster(opt.PlayerIndices, gamertagsOf(opt.Deaths))
	// L'ORIGINE se publie APRÈS le pont : son témoin (le calage du fil des morts) en sort.
	doc.OriginMs = resolveOriginMs(origin, opt.FilmClockOriginUS, own.DeathOffsetMS, own.DeathOffsetMatches)
	slog.Info("pont slot->joueur",
		"slots", len(own.Owner), "viesNommees", own.DeathsNamed, "viesTotal", own.LivesTotal,
		"lecturesIndex", own.IndexReadings, "desaccordsIndex", own.IndexDisagreements,
		"collisionsSlot", own.SlotCollisions)

	// Chaque calque rend sa COUVERTURE en même temps que son contenu. Le filtrage par
	// trajectoire publiée qui suit est lui aussi compté, sous une catégorie distincte.
	shots, shotCov := buildShots(sorted, fire, origin, step, own.Owner)
	doc.Shots = keepShotsOfPublishedTracks(shots, doc.Tracks)
	shotCov.Unpublished = countUnpublished(len(shots), len(doc.Shots))
	shotCov.Attached = len(doc.Shots)
	shotCov.warnIfLossy("tirs")

	doc.Loadouts = keepLoadoutsOfPublishedTracks(buildLoadouts(opt.Loadouts, origin, step), doc.Tracks)

	// Les projectiles se construisent AVANT les lancers : le lancer publie son lien vers le
	// projectile né de lui (Grenade.Proj), qui pointe un index de la tranche PUBLIÉE.
	var pubProjByRaw map[int]int
	doc.Projectiles, pubProjByRaw = buildProjectiles(opt.Projectiles, origin, step)

	gren, grenCov := buildGrenades(sorted, opt.Grenades, origin, step, own.Owner, opt.Projectiles, pubProjByRaw)
	doc.Grenades = keepGrenadesOfPublishedTracks(gren, doc.Tracks)
	grenCov.Unpublished = countUnpublished(len(gren), len(doc.Grenades))
	grenCov.Attached = len(doc.Grenades)
	grenCov.warnIfLossy("grenades")

	clock := replayScoreClock(&doc, interval, matchID)
	objCov := attachObjectiveActions(&doc, opt.Objectives, clock)
	scoreCov := attachScoreTimeline(&doc, opt.Score, opt.Deaths, clock, matchID)

	// L'ETAT ACTIF des deux familles mesurees (camo, surbouclier) : episodes dates par
	// vie, fermes a la mort quand rien n'a mesure la fin (cf. equipment_episodes.go).
	// Le surbouclier se lit dans les positions NON decimees : la decimation garde un
	// echantillon par frame et perdrait des transitions. Construit AVANT la couverture,
	// qui publie son compte.
	var camoNonBinary int
	doc.EquipmentEpisodes, camoNonBinary = buildEquipmentEpisodes(sorted, opt.CamoStates, origin, step, doc.Tracks)
	if camoNonBinary > 0 {
		slog.Warn("rejeu : lectures camo NON BINAIRES ignorees — l'interrupteur mesure ne connait que 0 et 4095",
			"lectures", camoNonBinary)
	}
	// Les FRAGS SOUS EFFET ACTIF : jointure des episodes avec les kills resolus par
	// l'appelant (cf. equipment_episode_kills.go). AVANT la couverture, qui publie
	// killsRead a cote des compteurs.
	killsRead := attachAllEquipmentKills(doc.EquipmentEpisodes, opt.Kills, own.SlotXUID, doc.OriginMs, interval)

	doc.Coverage = buildCoverage(shotCov, grenCov, objCov, own, doc.OriginMs != nil, scoreCov)
	// La couverture des episodes d'equipement se publie AVEC eux : « N episodes » sans
	// « sur M vies » se lirait comme une exhaustivite.
	doc.Coverage.Equipment = equipmentCoverage(doc.EquipmentEpisodes, doc.Tracks)
	doc.Coverage.Equipment.KillsRead = killsRead
	// Les TRACTIONS de grappin : fenetre mesuree par vie + ancre en coordonnees monde
	// (cf. grapple_lines.go). L'ancre exige les bornes de la carte : sans MapQuant,
	// aucune traction (regle map_bounds.go — pas de bornes, pas de coordonnee monde).
	switch {
	case opt.MapQuant != nil:
		var grapCov *GrappleCoverage
		doc.GrappleLines, grapCov = buildGrappleLines(opt.GrappleReads, *opt.MapQuant, origin, step, doc.Tracks)
		doc.Coverage.Grapple = grapCov
		slog.Info("rejeu : tractions de grappin",
			"tirs", grapCov.LightReads, "accroches", grapCov.HeavyReads,
			"tractions", grapCov.Pulls, "vies", grapCov.PullLives,
			"rates", grapCov.UnpairedFires, "corpsCasses", grapCov.BrokenBodies)
	case len(opt.GrappleReads) > 0:
		slog.Warn("rejeu : lectures de grappin sans bornes de carte — aucune traction publiee",
			"lectures", len(opt.GrappleReads))
	}
	// Les POSES d'equipement : famille par le manifeste du titre, poseur et cap MESURES sur le
	// nuage NON decime (une pose dure quelques dizaines de millisecondes ; la decimation
	// perdrait le record contemporain qui designe le poseur). La FIN OBSERVEE (schema 28)
	// vient du recensement ti=37 deja lu par la chaine des socles (opt.Pads.Powerups).
	doc.EquipmentPlacements, doc.Coverage.Placements = buildEquipmentPlacements(
		opt.Placements, opt.PlacementStats, sorted,
		replayClock{origin: origin, step: step, frames: doc.FrameCount,
			families: opt.Labels.EquipmentFamilies},
		opt.Pads.Powerups.Keyframes)
	logPlacementCoverage(doc.Coverage.Placements)
	// LES PRISES ET LES LACHERS d'arme, sur l'axe de frames du document. Les re-annonces d'une
	// arme deja portee au spawn sont ECARTEES ici : ce ne sont pas des ramassages.
	var wcCov WeaponChangeCoverage
	doc.WeaponChanges, wcCov = buildWeaponChanges(opt.WeaponChanges, origin, step)
	doc.Coverage.WeaponChanges = &wcCov
	slog.Info("rejeu : prises et lachers d arme",
		"decodes", wcCov.Decoded, "publies", wcCov.Published,
		"prises", wcCov.Taken, "lachers", wcCov.Dropped, "echanges", wcCov.Swapped,
		"reannonces", wcCov.Restated, "avantOrigine", wcCov.BeforeOrigin)
	// LES RAMASSAGES NATIFS, dates a la milliseconde et ATTRIBUES par le pont slot -> joueur
	// deja construit ci-dessus. Publies AVANT les socles : `attachWeaponPads` s'en sert pour
	// dater les occupations de socle qui restaient en intervalle de vingt secondes.
	var pkCov PickupCoverage
	judge := newPickupOriginJudge(opt, pos, doc.EquipmentPlacements)
	doc.Pickups, pkCov = buildPickups(opt.Pickups,
		replayClock{origin: origin, step: step, frames: doc.FrameCount,
			families: opt.Labels.EquipmentFamilies},
		pickupInputs{slotXUID: own.SlotXUID, st: opt.PickupStats,
			weaponKeys: opt.Labels.Keys, judge: judge})
	doc.Coverage.Pickups = &pkCov
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
	gwObjs := attachWeaponPads(&doc, opt.Pads, sorted,
		replayClock{origin: origin, step: step, frames: doc.FrameCount}, opt.Labels)
	// DATATION DES OCCUPATIONS DE SOCLE par l'evenement natif : l'intervalle de vingt secondes
	// devient un instant, et `xuid` cesse d'etre `null`, QUAND un ramassage natif de la meme
	// famille tombe dans la fenetre. Rien n'est efface : une occupation non couverte garde son
	// intervalle intact (pad_pickup_dating.go).
	padDating := datePadPickups(doc.WeaponPads, doc.PadPickups, doc.Pickups)
	doc.Coverage.PadDating = &padDating
	slog.Info("rejeu : datation des occupations de socle",
		"occupations", padDating.Occupations, "datees", padDating.Dated, "nommees", padDating.Named,
		"ambigues", padDating.Ambiguous, "nonCouvertes", padDating.Uncovered)
	// LES ARMES AU SOL INDIVIDUELLES (schema 27) : la meme chaine que les socles, publiee objet
	// par objet, LIEE aux lachers et aux prises du flux delta, et bornee par l observation —
	// jamais par une table de durees (document_ground_weapon_items.go).
	var gwiCov GroundWeaponItemsCoverage
	doc.GroundWeapons, gwiCov = buildGroundWeaponItems(gwObjs, opt.WeaponChanges, sorted,
		replayClock{origin: origin, step: step, frames: doc.FrameCount})
	doc.Coverage.GroundWeaponItems = &gwiCov
	logGroundWeaponItems(gwiCov)
	// La VIE DES DRAPEAUX, sur les pistes PUBLIEES (le drapeau porte est a la position de son
	// porteur, et c'est celle-la que le client dessine) — cf. build_objectives_live.go.
	attachFlagCarries(&doc, opt, own, replayClock{origin: origin, step: step, frames: doc.FrameCount})
	// LA COURONNE VIP, sur les pistes PUBLIEES (la couronne est a la position de son porteur) —
	// gardee de mode par l'appelant (opt.Vip.Scanned), cf. vip_crown.go.
	attachVipCrown(&doc, opt, own, replayClock{origin: origin, step: step, frames: doc.FrameCount})
	// LE PORTEUR DU CRANE d'Oddball, sur les pistes PUBLIEES (le crane est a la position de son
	// porteur) — garde de mode par l'appelant (opt.Skull.Scanned), cf. skull_carries.go. Le crane
	// LIBRE (attachObjectiveObjects, ci-dessous) reste la couche POSITION ; ce calque-ci est la
	// couche VIVANTE par-dessus.
	attachSkullCarries(&doc, opt, own, replayClock{origin: origin, step: step, frames: doc.FrameCount})
	// LE PORTEUR DE LA BOMBE d'Assaut, sur les pistes PUBLIEES (la bombe est a la position de
	// son porteur) — garde de mode par l'appelant (opt.Bomb.CarryScanned, TOUTES les variantes
	// de la famille bomb), source : le canal des armes tenues DEJA balaye (opt.WeaponChanges),
	// cf. bomb_carries.go.
	attachBombCarries(&doc, opt, own, replayClock{origin: origin, step: step, frames: doc.FrameCount})
	// LES OBJETS D'OBJECTIF LIBRES SONT POSÉS HORS DE LA GARDE DE MODE DU DRAPEAU, et c'est
	// délibéré : ce calque ne lit ni le statborg ni le fil des morts, donc rien de ce que cette
	// garde protège. La placer devant l'éteindrait sur Oddball — là où il sert.
	attachObjectiveObjects(&doc, opt, replayClock{origin: origin, step: step, frames: doc.FrameCount})
	// L'ETAT DES ZONES, sur la MEME horloge que les positions et sur les captures DEJA posees
	// (`doc.Objectives`) — cf. build_zones.go.
	attachZoneStates(&doc, opt, replayClock{origin: origin, step: step, frames: doc.FrameCount})
	// L'ARMEMENT DE LA BOMBE, sur la meme horloge que les actions d'objectif et confronte aux
	// explosions DEJA posees (`doc.Objectives`) — garde de mode par l'appelant
	// (opt.Bomb.Scanned), cf. bomb_armings.go.
	attachBombArmings(&doc, opt, clock)
	slog.Info("rejeu : episodes d'equipement actif",
		"viesPubliees", doc.Coverage.Equipment.TracksTotal,
		"viesCamo", doc.Coverage.Equipment.CamoLives,
		"episodesCamo", doc.Coverage.Equipment.CamoEpisodes,
		"viesSurbouclier", doc.Coverage.Equipment.OvershieldLives,
		"episodesSurbouclier", doc.Coverage.Equipment.OvershieldEpisodes)
	doc.WeaponLabels = buildWeaponLabels(doc.Loadouts, doc.Shots, doc.WeaponPads, opt.Labels)
	// La table weapon_key -> famille d'effet voyage telle quelle : les kills du feed sont
	// keyés par weapon_key (résolution base), pas par identifiant d'arme film — sans elle,
	// aucun effet de mort n'est joignable côté client.
	if len(opt.Labels.Effects) > 0 {
		doc.KillEffects = opt.Labels.Effects
	}
	// Les morts sans revendication ne sont publiées que pour les joueurs dont une trajectoire
	// l'est : le client déduit ces lignes DE SES PISTES, une entrée sans piste ne rencontrerait
	// jamais de ligne à décorer (même règle que les tirs, lancers et actions d'objectif).
	doc.NeutralDeaths = keepNeutralDeathsOfPublishedTracks(opt.NeutralDeaths, doc.Tracks)
	builtInv, invDroppedOrigin := buildInventory(opt.Inventory, origin, step)
	doc.Inventory = keepInventoryOfPublishedTracks(builtInv, doc.Tracks)
	// COUVERTURE DU CALQUE INVENTAIRE (audit AUDIT_AVAL_INVENTAIRE_2026-08-24.md, point 5),
	// journalisée comme les autres calques — construction dans inventory.go, avec le type
	// qu'elle publie.
	//
	// L'AFFECTATION EST GARDÉE, comme celles des calques frères (Grapple, FlagCarries, Zones) :
	// quand l'appelant n'a RIEN fourni à lire — inventaire illisible, cf. le repli `inventory = nil`
	// de BuildFromFilm —, la couverture reste ABSENTE. Publier {0,0,0,0} affirmerait « lecture
	// faite, zéro trouvé », qui est le contraire de ce qui s'est passé ; l'ABSENCE dit encore autre
	// chose, et la doctrine de coverage.go repose sur cette distinction.
	attachInventoryCoverage(&doc, opt.Inventory, builtInv, invDroppedOrigin)
	// POURQUOI UNE LECTURE D'INVENTAIRE EST VIDE : le croisement avec le fil des morts se fait
	// ICI, où les morts et leur decalage d'horloge existent — pas dans le projecteur
	// (cf. inventory_dead_readings.go).
	logInventoryEmptyCoverage(doc.Inventory, markInventoryDeadReadings(doc.Inventory, opt.Deaths, own,
		replayClock{origin: origin, step: step, frames: doc.FrameCount}))
	// LES GRENADES ONT LEUR PROPRE AXE, alimente par les deux canaux (cf. grenade_reads.go) :
	// ils n'ont pas la meme cadence, et les verser dans `Inventory` ferait masquer une lecture
	// pleine par une lecture partielle — la cellule de munitions se viderait.
	builtGren := buildGrenadeReads(opt.Inventory, opt.InventoryDeltas, origin, step)
	doc.GrenadeReads = keepGrenadeReadsOfPublishedTracks(builtGren, doc.Tracks)
	attachGrenadeReadCoverage(&doc, builtGren, opt.InventoryDeltaAmmoRefused)

	// Les rangs de grenade sont publiés dès qu'un calque les référence : l'inventaire
	// (compteurs portés) OU les lancers (Grenade.Rank). Les conditionner au seul
	// inventaire laissait des lancers pointer une table absente.
	if len(doc.Inventory) > 0 || len(doc.Grenades) > 0 || len(doc.GrenadeReads) > 0 {
		doc.GrenadeLabels = opt.Labels.Grenades
	}
	// La capacite portee a son PROPRE calque : ses deux canaux ne vivent pas sur la meme
	// horloge (i48 dans les deltas, l'ancre dans les images-cles), et ils publient la MEME
	// grandeur — le rang de palette.
	doc.Abilities = keepAbilitiesOfPublishedTracks(
		buildAbilityReads(opt.AbilityRanks, opt.Inventory, origin, step), doc.Tracks)
	// LA PALETTE SE CLASSE AVANT DE NOMMER, et un film ambigu ne recoit AUCUN nom : le
	// meme rang designe des capacites differentes d'une palette a l'autre.
	// LES RAMASSAGES ET LES CONSOMMATIONS d'equipement, sur le meme axe et avec les MEMES
	// rangs que doc.Abilities : c'est AbilityLabels qui les nomme, ou pas. Les annonces de
	// reapparition sont ECARTEES ici — ce ne sont pas des ramassages.
	ecChanges, ecCov := buildEquipmentChanges(
		opt.EquipmentChanges, opt.EquipmentChangeStats, origin, step)
	doc.EquipmentChanges = keepEquipmentChangesOfPublishedTracks(ecChanges, doc.Tracks)
	doc.Coverage.EquipmentChanges = &ecCov
	logEquipmentChangeCoverage(ecCov)
	palette := classifyAbilityPalette(doc.Abilities, opt.Labels.Abilities)
	doc.AbilityLabels = abilityLabelsUsed(doc.Abilities, palette)
	slog.Info("rejeu : palette de capacites",
		"palette", paletteIDOrNone(palette), "lectures", len(doc.Abilities),
		"rangsNommes", len(doc.AbilityLabels))
	slog.Info("rejeu : couverture par calque",
		"tirsRattaches", shotCov.Attached, "tirsDisponibles", shotCov.Available,
		"tirsSansSlot", shotCov.NoSlot, "tirsAmbigus", shotCov.Ambiguous,
		"tirsHorsFenetre", shotCov.OutOfWindow, "tirsNonPublies", shotCov.Unpublished,
		"grenadesRattachees", grenCov.Attached, "grenadesDisponibles", grenCov.Available,
		"verdictTirs", doc.Coverage.Verdict["shots"],
		"verdictGrenades", doc.Coverage.Verdict["grenades"],
		"verdictPont", doc.Coverage.Verdict["bridge"])
	return doc
}

// fireRefs réduit les événements de tir à ce que les fermetures ont le droit de connaître : QUI
// et QUAND. L'arme et la visée sont volontairement laissées dehors — elles n'ont aucun pouvoir
// de désignation, et les rendre visibles à la fermeture rouvrirait la porte au vote supprimé le
// 2026-07-28.
func fireRefs(fire []filmdec.FireEvent) []FireEventRef {
	out := make([]FireEventRef, len(fire))
	for i, e := range fire {
		out[i] = FireEventRef{FilmIndex: e.FilmIndex, TimestampUS: e.TimestampUS}
	}
	return out
}

// keepShotsOfPublishedTracks écarte les tirs dont le slot n'a pas de trajectoire publiée
// (track trop courte) : le client n'aurait rien à quoi les rattacher.
func keepShotsOfPublishedTracks(shots []Shot, tracks []Track) []Shot {
	return keepOfPublishedTracks(shots, tracks,
		func(s Shot, published map[uint32]bool) bool { return published[s.Slot] })
}

// decimateTracks projette les positions sur la grille de frames (un point par slot et par
// frame, le premier observé gagne) et produit une track par slot, dans l'ordre de première
// apparition.
func decimateTracks(sorted []filmdec.BipedPosition, origin, step uint64, minPoints int,
	scoped func(slot uint32, tsUS uint64) int) []Track {
	type acc struct {
		pts       []Point
		lastFrame int
	}
	accs := map[uint32]*acc{}
	var order []uint32
	for _, p := range sorted {
		if !p.HasWorld { // quantum sans bornes de carte : pas une coordonnée, on ne publie pas
			continue
		}
		frame := int((p.TimestampUS - origin) / step)
		a := accs[p.Slot]
		if a == nil {
			a = &acc{lastFrame: -1}
			accs[p.Slot] = a
			order = append(order, p.Slot)
		}
		if frame == a.lastFrame {
			continue
		}
		a.lastFrame = frame
		pt := Point{T: frame, X: round2(p.X), Y: round2(p.Y), Z: round2(p.Z)}
		if h, ok := p.AimHeadingDeg(); ok { // cap de visée du MÊME record (i21), si répliqué
			pt.H = headingForJSON(h)
		}
		// ÉLÉVATION du MÊME record et du MÊME composant que le cap (le R(11) qui suit le
		// R(12) d'i21) : les deux angles arrivent ensemble ou pas du tout, `AimPitchDeg`
		// partageant la validité `HasYaw` avec `AimHeadingDeg`. Publier l'un sans l'autre
		// n'a donc aucun sens — et l'absence de `p` sur un point qui porte `h` dit « à
		// plat », pas « inconnu » (cf. Point.P).
		if pitch, ok := p.AimPitchDeg(); ok {
			pt.P = pitchForJSON(pitch)
		}
		// LUNETTE : etat a bascule, d'une AUTRE source que les deux angles ci-dessus — on le
		// consulte a l'instant du point au lieu de le lire dedans (cf. Point.S, zoom_state.go).
		if scoped != nil {
			pt.S = scoped(p.Slot, p.TimestampUS)
		}
		// Vitalité du MÊME record que la position (i4 / i5). La décimation garde le PREMIER
		// échantillon de chaque frame : si deux records du même slot tombent dans la même
		// frame de 100 ms et que seul le second porte le bouclier, il est perdu. Cela
		// n'invente rien — c'est une perte, pas une erreur — et le témoin publié est mesuré
		// sur les positions NON décimées.
		// Témoin : P(bouclier nul | 500 ms avant une mort connue) = 50,49 % contre 38,18 %
		// chez un vivant à plus de 5 s d'une mort, soit un rapport de 1,32x — FAIBLE, et
		// c'est normal : le film ne réplique le bouclier que lorsqu'il CHANGE, donc une
		// mesure de bouclier est déjà une mesure de combat. Ce qui porte le rendu est le
		// témoin de FORME (27 404/27 404 quanta dans [0,64]), pas ce rapport.
		if sh, ok := p.ShieldAt(); ok {
			pt.Sh = fractionForJSON(sh)
		}
		if hp, ok := p.HealthAt(); ok {
			pt.Hp = fractionForJSON(hp)
		}
		a.pts = append(a.pts, pt)
	}
	tracks := make([]Track, 0, len(order))
	for _, slot := range order {
		pts := accs[slot].pts
		if len(pts) < minPoints {
			continue
		}
		tracks = append(tracks, Track{
			Slot:       slot,
			Team:       -1,
			Points:     pts,
			StartFrame: pts[0].T,
			EndFrame:   pts[len(pts)-1].T,
		})
	}
	return tracks
}

// frameSpan renvoie le nombre de frames couvrant tout le film (dernier index + 1).
func frameSpan(sorted []filmdec.BipedPosition, origin, step uint64) int {
	last := sorted[len(sorted)-1].TimestampUS
	return int((last-origin)/step) + 1
}

// round2 arrondit au centième (cf. coordScale).
func round2(v float32) float32 {
	return float32(math.Round(float64(v)*coordScale) / coordScale)
}

// fractionForJSON arrondit une fraction [0,1] au millième et la rend par POINTEUR : c'est
// ce pointeur qui permet de publier un ZÉRO (bouclier brisé) sans qu'omitempty le confonde
// avec une absence de mesure. Cf. Point.Sh.
func fractionForJSON(v float32) *float32 {
	r := float32(math.Round(float64(v)*1000) / 1000)
	return &r
}
