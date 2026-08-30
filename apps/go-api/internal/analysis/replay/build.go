package replay

import (
	"fmt"
	"log/slog"
	"math"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
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

// Options règle l'assemblage du document de rejeu.
type Options struct {
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
	Placements     []filmdec.EquipmentPlacement
	PlacementStats filmdec.EquipmentPlacementStats
	// Pads : ce que le film rend sur les SOCLES — armes au sol (`ti=42`) et power-ups (`ti=37`),
	// TROIS lectures chacun, `Scanned` disant qu'elles ont abouti (cf. build_ground_weapons.go).
	// Entree de DONNEES, comme Placements. Absente = rejeu sans socles — jamais des socles devines.
	Pads PadScans
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

// BuildFromFilm décode les positions bipeds des SEULS chunks du film de filmDir et en
// assemble le document de rejeu 2D. Aucune entrée Cheat Engine.
//
// HORS LIGNE par construction (I/O disque sur tout le film) — ne jamais appeler depuis un
// chemin de requête ; l'API sert l'artefact pré-construit.
func BuildFromFilm(matchID, titleSlug, filmDir string, opt Options) (ReplayDocument, error) {
	if opt.MapQuant == nil {
		return ReplayDocument{}, fmt.Errorf("%w (match %s) : le document de rejeu exige l'entrée de catalogue de la carte",
			filmdec.ErrUnknownMapBounds, matchID)
	}
	// UN SEUL decodage filmdec a la fois par process (verrou de paquet partage avec
	// killsource.Decode) : les balayages ci-dessous lisent et ecrivent les globaux de
	// filmdec (dont compWidthObs, sans verrou propre). Tenu jusqu'au retour : l'assemblage
	// pur qui suit est negligeable devant le decodage, et relacher plus tot inviterait un
	// entrelacement entre deux sous-balayages du MEME film.
	release := filmdec.LockProcessDecode()
	defer release()
	// Les largeurs d'axe du chemin WORLD-OBJECT sont un global de paquet : installées ici,
	// sous le verrou, pour TOUT le decodage du film, et restaurees au retour.
	defer installWorldObjectPrecision(*opt.MapQuant, filmDir)()
	worldRange := opt.MapQuant.Range()
	scan := filmdec.DefaultScanFilmOptions()
	if opt.Scan != nil {
		scan = *opt.Scan
	}
	scan.WorldRange = &worldRange
	// Le DÉCOUPAGE d'i0 vient du CATALOGUE, comme les bornes dont il est déduit — le
	// découpage lu dans le film (DetectI0Layout) est le contrôle, jamais l'entrée (doctrine
	// écrite sur WorldObjectPrecision, appliquée au bipède depuis le lot C catalogues
	// 2026-08-27 : sur une carte à plus de 2 régions, l'auto-détection lit l'index de
	// région comme un bit d'axe et le décodeur rejetterait tous les records — Live Fire).
	// Un opt.Scan qui force déjà son Layout (instruments) reste maître ; une entrée sans
	// largeurs (catalogue antérieur au champ) laisse l'auto-détection, comme le chemin
	// world-object laisse son défaut — jamais des largeurs nulles.
	if lay := opt.MapQuant.Layout(); scan.Layout == nil && lay.Valid() {
		scan.Layout = &lay
	}
	// Le cap de visée (Point.H) se lit dans le MÊME record que la position : la capture des
	// directions est donc toujours active pour l'artefact. Elle n'altère aucune position
	// (lecture seule après le vec3 d'i0).
	scan.CaptureDirs = true
	positions, err := filmdec.ScanFilmBipedPositions(filmDir, scan)
	if err != nil {
		return ReplayDocument{}, err
	}
	// Les tirs sont décodés du MÊME film et sur la MÊME horloge que les positions ; leur
	// absence n'est pas fatale (un film sans event de tir reste un rejeu valide).
	shots, err := filmdec.ScanFilmFireEvents(filmDir)
	if err != nil {
		slog.Warn("events de tir illisibles — rejeu sans tirs", "err", err, "filmDir", filmDir)
		shots = nil
	}
	// Armes portées : lues dans les keyframes du MÊME film, sur la MÊME horloge. Leur
	// absence n'est pas fatale (un rejeu sans armes reste un rejeu valide).
	loadouts, err := filmdec.ScanFilmKeyframeLoadouts(filmDir, loadoutFamilies())
	if err != nil {
		slog.Warn("keyframes illisibles — rejeu sans armes portées", "err", err, "filmDir", filmDir)
		loadouts = nil
	}
	opt.Loadouts = loadouts
	// Inventaire complet : MÊMES images-clés, MÊME horloge, même record de biped que les armes
	// portées. Absence non fatale — un rejeu sans grenades reste un rejeu valide.
	inventory, invStats, err := ScanFilmKeyframeInventory(filmDir, loadoutFamilies(), 0)
	if err != nil {
		slog.Warn("inventaire illisible — rejeu sans grenades ni munitions", "err", err, "filmDir", filmDir)
		inventory = nil
	} else {
		slog.Info("inventaire : lectures de keyframe",
			"chunks", invStats.Chunks, "chunksIllisibles", invStats.ChunksUnread,
			"imagesCles", invStats.Keyframes, "records", invStats.Records,
			"grenadesParAncre", invStats.GrenadesByAnchor, "grenadesParPosition", invStats.GrenadesByPosition)
	}
	opt.Inventory = inventory
	// Inventaire suivi dans les paquets DELTA : les compteurs de grenades (i22) et le jeu
	// selectionne (i47), transmis AU CHANGEMENT donc places la ou l'etat bouge. Absence non
	// fatale — l'axe des grenades retombe sur les seules images-cles.
	invDeltas, dStats, err := filmdec.ScanFilmInventoryDeltas(filmDir)
	if err != nil {
		slog.Warn("inventaire delta illisible — grenades sans rafraichissement entre images-cles",
			"err", err, "filmDir", filmDir)
		invDeltas = nil
	} else {
		slog.Info("inventaire delta : lectures",
			"recordsDelta", dStats.Records, "masqueAvecI22", dStats.WithI22,
			"i22Lues", dStats.I22Read, "i22Implausibles", dStats.Implausible,
			"masqueAvecI47", dStats.WithI47, "i47Lues", dStats.I47Read,
			"accord", dStats.Accord, "accordVerifies", dStats.AccordChecked,
			"canalMunitionsRefuse", dStats.AmmoRefused)
	}
	opt.InventoryDeltas = invDeltas
	opt.InventoryDeltaAmmoRefused = dStats.AmmoRefused
	// Identite de la capacite portee : lue dans les paquets DELTA, sur la MEME horloge. Rare
	// (une transmission par vie environ) mais elle porte le rang COMPLET, la ou les images-cles
	// ne voient que 16..23. Absence non fatale — le rejeu retombe sur cette seule fenetre.
	abilityRanks, aStats, err := filmdec.ScanFilmAbilityRanks(filmDir)
	if err != nil {
		slog.Warn("identites de capacite illisibles — rejeu sans rang complet", "err", err, "filmDir", filmDir)
		abilityRanks = nil
	} else {
		slog.Info("capacites : lectures d i48",
			"recordsDelta", aStats.Records, "masqueAvecI48", aStats.WithI48,
			"lues", aStats.Read, "illisibles", aStats.Unread, "sansIdentite", aStats.Gated)
	}
	opt.AbilityRanks = abilityRanks
	// Etat du camouflage : la voie i28 queue[1], lue dans les paquets DELTA, sur la MEME
	// horloge (cf. filmdec/camo_state.go). Absence non fatale — le rejeu sort sans episodes
	// de camouflage, jamais avec des episodes devines.
	camoStates, cStats, err := filmdec.ScanFilmCamoStates(filmDir)
	if err != nil {
		slog.Warn("etat de camouflage illisible — rejeu sans episodes de camo", "err", err, "filmDir", filmDir)
		camoStates = nil
	} else {
		slog.Info("camouflage : lectures d i28 queue[1]",
			"recordsDelta", cStats.Records, "masqueAvecI28", cStats.WithI28,
			"lues", cStats.Read, "illisibles", cStats.Unread, "sansVoie", cStats.NoChannel)
	}
	opt.CamoStates = camoStates
	// Evenements de grappin : le corps tag==3 d'i59, lu dans les paquets DELTA, sur la
	// MEME horloge (cf. filmdec/grapple_state.go). Absence non fatale — le rejeu sort sans
	// tractions de grappin, jamais avec des tractions devinees.
	grappleReads, gStats, err := filmdec.ScanFilmGrappleReads(filmDir)
	if err != nil {
		slog.Warn("evenements de grappin illisibles — rejeu sans tractions", "err", err, "filmDir", filmDir)
		grappleReads = nil
	} else {
		slog.Info("grappin : lectures d i59 tag==3",
			"recordsDelta", gStats.Records, "masqueAvecI59", gStats.WithI59,
			"lues", gStats.Read, "illisibles", gStats.Unread,
			"tag3", gStats.Tag3, "corpsCasses", gStats.BodyBroken)
	}
	opt.GrappleReads = grappleReads
	// POSES d'equipement : records de CREATION de l'archetype 37, sur la MEME horloge
	// (cf. equipment_placements.go — decodage, journal et refus y vivent ensemble).
	opt.Placements, opt.PlacementStats = decodeFilmPlacements(filmDir, &worldRange)
	// SOCLES : archetypes 42 (armes) et 37 (power-ups), sur la MEME horloge, AUX LARGEURS MPP que
	// la calibration des POSES vient de mesurer sur ce film (cf. build_ground_weapons.go).
	opt.Pads = decodeFilmPadScans(filmDir, &worldRange, opt.PlacementStats.Calibration.Widths)
	// MARQUEUR DE PORTAGE : le controle independant du calque du drapeau, lu aux images-cles du
	// MEME film — sur les seuls films de CTF (cf. build_objectives_live.go).
	opt.Flag.Marks = decodeFilmCarrierMarks(filmDir, opt.Flag)
	// PROPRIETES RESEAU ti=13 : l'etat des zones (jauge de capture, proprietaire), lu dans les
	// paquets delta du MEME film — sur les seuls matchs dont l'appelant a fourni le catalogue de
	// zones (cf. build_zones.go).
	opt.Zone.Reads = decodeFilmZoneReads(filmDir, matchID, len(opt.Zone.Zones))
	opt.Zone.Scanned = len(opt.Zone.Zones) > 0
	// Lancers de grenade : décodés des paquets delta du MÊME film, sur la MÊME horloge.
	// Absence non fatale, comme les tirs et les armes portées.
	grenades, err := filmdec.ScanFilmGrenadeThrows(filmDir)
	if err != nil {
		slog.Warn("paquets delta illisibles — rejeu sans lancers de grenade", "err", err, "filmDir", filmDir)
		grenades = nil
	}
	opt.Grenades = grenades
	// Trajectoires de projectile : memes chunks, meme horloge. Absence non fatale.
	proj, err := filmdec.ScanFilmProjectiles(filmDir, &worldRange)
	if err != nil {
		slog.Warn("projectiles illisibles — rejeu sans trajectoires", "err", err, "filmDir", filmDir)
		proj = nil
	}
	opt.Projectiles = proj
	// Le fil des morts NOMME les vies. Sans lui, le pont est vide et NI les tirs NI les lancers
	// ne sont publiés : ce n'est pas une dégradation cosmétique, d'où un warn explicite.
	deaths, err := ScanFilmDeaths(filmDir)
	if err != nil {
		slog.Warn("fil des morts illisible — aucun tir ni lancer ne sera publie",
			"err", err, "filmDir", filmDir)
		deaths = nil
	}
	opt.Deaths = deaths
	// L'index de joueur SE LIT dans le film (cf. player_index.go) : le roster vient du fil des
	// morts, et les 5 bits qui précèdent chaque xuid donnent son index. Sans cette table, aucun
	// tir ni lancer n'est publié — comme sans le fil des morts.
	if len(deaths) > 0 {
		idx, err := ScanFilmPlayerIndices(filmDir, rosterFromDeaths(deaths))
		if err != nil {
			slog.Warn("index de joueur illisible — aucun tir ni lancer ne sera publie",
				"err", err, "filmDir", filmDir)
		}
		table, collisions := injectiveOrEmpty(idx)
		if collisions > 0 {
			slog.Warn("index de joueur NON INJECTIF — table ecartee",
				"collisions", collisions, "filmDir", filmDir)
		}
		opt.PlayerIndices = table
	}
	// L'origine d'horloge du film : deux en-têtes de paquet, aucune estimation (cf.
	// origin.go). Son absence n'est pas fatale — le document sort sans origine, et le
	// client retombe sur l'appariement.
	clockUS, err := ScanFilmClockOrigin(filmDir)
	if err != nil {
		slog.Warn("origine d'horloge illisible — rejeu sans origine publiee", "err", err, "filmDir", filmDir)
		clockUS = 0
	}
	opt.FilmClockOriginUS = clockUS
	return BuildFromPositions(matchID, titleSlug, positions, shots, opt), nil
}

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
	doc.Tracks = decimateTracks(sorted, origin, step, opt.minPoints())
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
	// perdrait le record contemporain qui designe le poseur).
	doc.EquipmentPlacements, doc.Coverage.Placements = buildEquipmentPlacements(
		opt.Placements, opt.PlacementStats, sorted,
		replayClock{origin: origin, step: step, frames: doc.FrameCount,
			families: opt.Labels.EquipmentFamilies})
	logPlacementCoverage(doc.Coverage.Placements)
	// Les SOCLES — armes au sol ET power-ups —, sur le meme nuage NON decime (build_ground_weapons.go).
	attachWeaponPads(&doc, opt.Pads, sorted,
		replayClock{origin: origin, step: step, frames: doc.FrameCount}, opt.Labels)
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
	// LES OBJETS D'OBJECTIF LIBRES SONT POSÉS HORS DE LA GARDE DE MODE DU DRAPEAU, et c'est
	// délibéré : ce calque ne lit ni le statborg ni le fil des morts, donc rien de ce que cette
	// garde protège. La placer devant l'éteindrait sur Oddball — là où il sert.
	attachObjectiveObjects(&doc, opt, replayClock{origin: origin, step: step, frames: doc.FrameCount})
	// L'ETAT DES ZONES, sur la MEME horloge que les positions et sur les captures DEJA posees
	// (`doc.Objectives`) — cf. build_zones.go.
	attachZoneStates(&doc, opt, replayClock{origin: origin, step: step, frames: doc.FrameCount})
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
func decimateTracks(sorted []filmdec.BipedPosition, origin, step uint64, minPoints int) []Track {
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
