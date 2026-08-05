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
	// Objectives : les actions d'objectif NOMMÉES ET IDENTIFIÉES (cf. objectives.go).
	// Entrée de DONNÉES, comme Loadouts et Grenades.
	//
	// POURQUOI DÉJÀ IDENTIFIÉES, et pas décodées ici : le pont slot -> xuid a besoin des
	// lignes de match (`match_participants`), donc de la BASE. Ce paquet et le CLI hors
	// ligne n'en ouvrent aucune — l'appelant qui l'a résout le pont et fournit le
	// résultat, exactement comme `objectiveevents.Extract` reçoit son `Roster`.
	// Absente = rejeu sans calque d'objectifs.
	Objectives []objectiveevents.IdentifiedEvent
	// Scan : réglages du décodage offline ; zéro -> filmdec.DefaultScanFilmOptions().
	Scan *filmdec.ScanFilmOptions
	// Labels : le catalogue de libellés DU TITRE (armes, grenades, capacités), chargé
	// depuis config/titles/{slug}/mappings/ par l'appelant hors ligne (cf. catalog.go).
	// Absent = document sans table de libellés : le client affiche les identifiants
	// bruts, ce qui reste vrai — contrairement à un nom approché.
	Labels LabelCatalog
	// WorldRange : bornes de déquantification DE LA CARTE du match (AABB du BSP, cf.
	// filmdec.MapQuantCatalog). OBLIGATOIRE : sans elles le décodeur ne produit que des
	// quanta, et BuildFromFilm refuse d'émettre un document plutôt que des coordonnées
	// fausses (elles l'étaient jusqu'ici : les bornes de Cliffhanger étaient appliquées à
	// toutes les cartes, et le filtre de téléportation en m/s décalibré d'autant).
	WorldRange *filmdec.Vec3Range
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
	if opt.WorldRange == nil {
		return ReplayDocument{}, fmt.Errorf("%w (match %s) : le document de rejeu exige les bornes de la carte",
			filmdec.ErrUnknownMapBounds, matchID)
	}
	scan := filmdec.DefaultScanFilmOptions()
	if opt.Scan != nil {
		scan = *opt.Scan
	}
	scan.WorldRange = opt.WorldRange
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
	inventory, err := ScanFilmKeyframeInventory(filmDir, loadoutFamilies(), 0)
	if err != nil {
		slog.Warn("inventaire illisible — rejeu sans grenades ni munitions", "err", err, "filmDir", filmDir)
		inventory = nil
	}
	opt.Inventory = inventory
	// Lancers de grenade : décodés des paquets delta du MÊME film, sur la MÊME horloge.
	// Absence non fatale, comme les tirs et les armes portées.
	grenades, err := filmdec.ScanFilmGrenadeThrows(filmDir)
	if err != nil {
		slog.Warn("paquets delta illisibles — rejeu sans lancers de grenade", "err", err, "filmDir", filmDir)
		grenades = nil
	}
	opt.Grenades = grenades
	// Trajectoires de projectile : memes chunks, meme horloge. Absence non fatale.
	proj, err := filmdec.ScanFilmProjectiles(filmDir, opt.WorldRange)
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
	own := buildOwners(indexBySlot(sorted), opt.Deaths, opt.PlayerIndices)
	// L'IDENTITÉ se pose sur les traces dès que le pont existe : sans elle, un client ne peut
	// ni nommer un joueur, ni regrouper ses vies, ni colorer une équipe.
	nameTracks(doc.Tracks, own.SlotXUID)
	doc.Roster = buildRoster(opt.PlayerIndices, gamertagsOf(opt.Deaths))
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

	gren, grenCov := buildGrenades(sorted, opt.Grenades, origin, step, own.Owner, opt.Projectiles)
	doc.Grenades = keepGrenadesOfPublishedTracks(gren, doc.Tracks)
	grenCov.Unpublished = countUnpublished(len(gren), len(doc.Grenades))
	grenCov.Attached = len(doc.Grenades)
	grenCov.warnIfLossy("grenades")

	// Les actions d'objectif arrivent DÉJÀ identifiées par xuid : leur pont passe par les
	// lignes de match, donc par la base, que ce paquet n'ouvre pas (cf. Options.Objectives).
	objActions, objCov := buildObjectiveActions(opt.Objectives, interval, doc.FrameCount)
	doc.Objectives, objCov = dropUnpublishedActions(objActions, doc.Tracks, objCov)
	objCov.warnIfLossy("objectifs")

	doc.Coverage = buildCoverage(shotCov, grenCov, objCov, own)
	doc.Projectiles = buildProjectiles(opt.Projectiles, origin, step)
	doc.WeaponLabels = buildWeaponLabels(doc.Loadouts, doc.Shots, opt.Labels)
	doc.Inventory = keepInventoryOfPublishedTracks(
		buildInventory(opt.Inventory, origin, step), doc.Tracks)
	// Les rangs de grenade sont publiés dès qu'un calque les référence : l'inventaire
	// (compteurs portés) OU les lancers (Grenade.Rank). Les conditionner au seul
	// inventaire laissait des lancers pointer une table absente.
	if len(doc.Inventory) > 0 || len(doc.Grenades) > 0 {
		doc.GrenadeLabels = opt.Labels.Grenades
	}
	if len(doc.Inventory) > 0 {
		doc.AbilityLabels = abilityLabelsUsed(doc.Inventory, opt.Labels.Abilities)
	}
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

// boundsOf calcule l'étendue XY (et Z) de tous les points publiés.
func boundsOf(tracks []Track) Bounds {
	var b Bounds
	first := true
	for _, tr := range tracks {
		for _, p := range tr.Points {
			if first {
				b = Bounds{MinX: p.X, MinY: p.Y, MaxX: p.X, MaxY: p.Y, MinZ: p.Z, MaxZ: p.Z}
				first = false
				continue
			}
			b.MinX, b.MaxX = minf(b.MinX, p.X), maxf(b.MaxX, p.X)
			b.MinY, b.MaxY = minf(b.MinY, p.Y), maxf(b.MaxY, p.Y)
			b.MinZ, b.MaxZ = minf(b.MinZ, p.Z), maxf(b.MaxZ, p.Z)
		}
	}
	return b
}

// geometryBounds calcule l'étendue XY des props (nil si pas de géométrie).
func geometryBounds(objs []MapObject) *Bounds {
	if len(objs) == 0 {
		return nil
	}
	b := Bounds{MinX: objs[0].X, MinY: objs[0].Y, MaxX: objs[0].X, MaxY: objs[0].Y}
	for _, o := range objs[1:] {
		b.MinX, b.MaxX = minf(b.MinX, o.X), maxf(b.MaxX, o.X)
		b.MinY, b.MaxY = minf(b.MinY, o.Y), maxf(b.MaxY, o.Y)
	}
	return &b
}

// round2 arrondit au centième (cf. coordScale).
func round2(v float32) float32 {
	return float32(math.Round(float64(v)*coordScale) / coordScale)
}

// headingForJSON arrondit le cap au dixième de degré (la visée est quantifiée à
// 360/4096 ≈ 0,088°, une décimale ne perd donc rien) et évite le PIÈGE omitempty : un cap
// qui s'arrondit à 0 serait omis et relu comme « pas de visée ». On publie 360, qui est le
// même cap et reste sérialisé.
func headingForJSON(v float32) float32 {
	r := float32(math.Round(float64(v)*10) / 10)
	if r <= 0 {
		return 360
	}
	return r
}

// fractionForJSON arrondit une fraction [0,1] au millième et la rend par POINTEUR : c'est
// ce pointeur qui permet de publier un ZÉRO (bouclier brisé) sans qu'omitempty le confonde
// avec une absence de mesure. Cf. Point.Sh.
func fractionForJSON(v float32) *float32 {
	r := float32(math.Round(float64(v)*1000) / 1000)
	return &r
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
