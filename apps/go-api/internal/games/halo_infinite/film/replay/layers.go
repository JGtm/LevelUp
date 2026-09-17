package replay

// layers.go — LA TABLE DES CALQUES : QUELLE COUCHE A PRODUIT QUOI (lot 4.2.1-a du
// PLAN_DECODEUR_FILM_2026-09-13 ; ADR 0034 D-6 et D-7).
//
// # CE QUE CE FICHIER EST, ET CE QU IL N EST PAS ENCORE
//
// Il porte la TABLE, et rien d autre : aucun champ n est ajoute au document par ce lot, et
// `SchemaVersion` ne bouge pas. La montee de schema qui publiera `layers` a la racine est un
// commit a part (lot 4.2.1-b), volontairement, parce qu une montee marque tout le parc a recuire
// (`Digest.UpToDate`, `internal/replaybuild/artifact_digest.go`) : ce qui se discute ici est le
// CONTENU de la table, ligne a ligne, pas une forme.
//
// # LE NOM D UN CALQUE EST LA CLE JSON DU DOCUMENT
//
// Trois nomenclatures existaient et il ne s agissait pas d en creer une quatrieme : les etapes
// de balayage (`observe_test.go`), les blocs de `Coverage` (47 balises) et les champs du
// document. C est LE LECTEUR qui decide, et le lecteur est `normalizeReplayDocument`
// (`apps/web/src/lib/replay/replayNormalize.ts`) : il comble des CHAMPS, jamais des etapes. Le
// nom d un calque est donc sa balise `json:` a la RACINE de [ReplayDocument].
//
// # LA VALEUR : L UNE DES CINQ REVISIONS CONNUES, JAMAIS UNE CHAINE LIBRE
//
// `source.Rev`, `profile.Rev`, `grammar.Rev`, `facts.Rev` — les quatre revisions de couche que
// `coverage.decoder` publie deja (`coverage_decoder.go`) — plus `publication-<SchemaVersion>`
// pour ce que produit `film/replay`, qui n a pas de revision de sources (le seul `...Rev` du
// paquet est `UsageSummaryRev`, `usage_summary.go`, et il concerne les usages, pas le document).
// La fermeture est tenue par `TestCalquesNePortentQueLesCinqRevisionsConnues`.
//
// # LA REGLE D ATTRIBUTION, ET POURQUOI LA COUCHE LA PLUS HAUTE SUFFIT
//
// On attribue a un calque LA COUCHE QUI DECODE CE QU IL PUBLIE — la substance de ses lignes —,
// et non les couches qui le filtrent, le datent ou le nomment.
//
// Citer la seule couche la plus haute suffit parce que LES REVISIONS SONT CHAINEES : chacune
// hache la VALEUR de celle du dessous (V15 (12) ; `profile/rev.go` : « VALEUR AMONT :
// `source.Rev`, et elle seule »). Une montee de `source` fait donc monter `profile`, qui fait
// monter `grammar`, qui fait monter `facts`. Un calque attribue a `grammar` est ainsi marque
// perime par toute montee de `source` ou de `profile` sans qu il faille les inscrire.
//
// C EST AUSSI POURQUOI NI `source` NI `profile` N APPARAISSENT DANS LA TABLE : aucun champ racine
// ne publie des octets de film bruts (ADR 0034 D-2 l interdit hors de `source`) ni une ligne de
// la table de profil. Les deux couches gouvernent tous les calques, par la chaine, sans en
// produire aucun. Elles restent des valeurs LEGALES (un calque futur peut en sortir) : la liste
// des cinq est fermee par le test, pas la liste des trois employees.
//
// # LES DEUX LIMITES, ECRITES PARCE QU ELLES SONT MESUREES
//
//  1. L ENRICHISSEMENT D IDENTITE TRAVERSE TOUS LES CALQUES ET N EST PAS ATTRIBUE. Le registre
//     d identite consomme `opt.Bots` (decode par `facts/killsource`) et `opt.StatborgIdentity`
//     (par `facts/objectives`), et il NOMME des vies dans des calques attribues a `grammar`
//     (`nameBotTracks`, `identity.go`). Une montee de `facts` SEULE peut donc changer le nom
//     d une vie dans un calque que cette table dit `grammar-...`. Attribuer `facts` a tout ce que
//     le registre touche etait l autre sortie : elle rend la table vraie et INUTILE (presque tous
//     les calques y passent). Le choix est la SELECTIVITE, et sa limite est cette ligne.
//  2. AUCUNE DES CINQ REVISIONS NE HACHE UN CATALOGUE DE DONNEES. `profile/rev.go` l ecrit comme
//     une limite assumee (« LE CATALOGUE DES CARTES N EST PAS HACHE »), et il en va de meme du
//     manifeste de libelles du titre. Les champs dont la substance EST un catalogue n ont donc
//     aucune couche : ils sont dans [calquesSansCouche], avec leur justification datee.
//
// # LE REGIME DE `layers`, IDENTIQUE A CELUI DE `coverage`
//
//	objet ABSENT            artefact anterieur au schema qui publie `layers`
//	entree ABSENTE          ce calque n a PAS ete produit — c est une REPONSE, pas un trou
//	entree PRESENTE         produit, sous la revision nommee
//
// Une entree manque quand la PASSE qui produit le calque n a pas tourne, et cela n arrive que
// derriere une garde : [gardesDeProduction] les porte, une par une, avec le site de production
// qui la ferme. Un calque dont la passe tourne a TOUJOURS son entree, fut-il vide — c est
// exactement la distinction que `coverage` tient aujourd hui a coups de blocs `omitempty`, et
// qu un tableau vide seul ne sait pas dire.
//
// # LES TROIS CALQUES A LA REQUETE N Y ENTRENT JAMAIS
//
// `mapObjectives`, `mapWeaponPads` et `weaponTiers` sont resolus PAR LE SERVICE, a la requete :
// la cuisson ne les ecrit pas (garde `TestDocumentShapeCalquesALaRequeteRestentHorsCuisson`,
// `document_shape_test.go`). Ils ne sont ni dans la table, ni dans les exemptions.

import (
	"strconv"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// revisionDeLaPublication est la revision de la couche de PUBLICATION : `film/replay` ne hache
// aucune source, et ce que sa forme change est exactement ce que `SchemaVersion` date.
//
// CONSEQUENCE VOULUE : toute montee de schema perime les calques attribues a la publication, et
// eux seuls — c est le cas que la recuisson selective du lot 4.4 doit ramener a une republication
// depuis les faits plutot qu a un decodage complet.
var revisionDeLaPublication = "publication-" + strconv.Itoa(SchemaVersion)

// couchesDesCalques — LA TABLE. Une entree par champ racine CUIT ; la valeur est la revision de
// la couche qui DECODE ce que le champ publie (cf. la regle d attribution en tete de fichier).
//
// Chaque ligne porte le SITE DE PRODUCTION qui la justifie. Les entrees `grammar` citent, quand
// l entree ne se lit pas dans le fichier de pose, le champ de [FilmInputs] d ou elle vient :
// `FilmInputs.applyTo` est la SEULE ecriture des entrees lues dans le film (`film_inputs.go`),
// donc tout ce qui en sort est decode par `grammar` — et tout ce qui n en sort pas vient de
// l appelant.
//
// Toute entree ajoutee ici est une DECISION : elle dit sous quelle revision un lecteur devra
// juger ce calque perime.
var couchesDesCalques = map[string]string{
	// --- LA PUBLICATION : ce que `film/replay` ecrit de lui-meme, sans decoder un octet.
	"schemaVersion":   revisionDeLaPublication, // build.go `ouvrir` <- la constante du paquet
	"matchId":         revisionDeLaPublication, // build.go `ouvrir` <- argument de l appelant
	"titleSlug":       revisionDeLaPublication, // build.go `ouvrir` <- argument de l appelant
	"frameIntervalMs": revisionDeLaPublication, // build.go `ouvrir` <- `Options.frameIntervalMS()`, un reglage

	// --- LA GRAMMAIRE : les calques dont les lignes sortent d une lecture de bits du film.
	// L HORLOGE ET LES BORNES en font partie : elles se calculent sur le nuage de positions, donc
	// elles bougent avec la grammaire qui le decode.
	"frameCount":          grammar.Rev, // build.go `ouvrir` <- `frameSpan` sur les positions
	"durationMs":          grammar.Rev, // build.go `ouvrir` <- frameCount x pas de temps
	"bounds":              grammar.Rev, // tracks_publication.go `poserLesTraces` <- positions
	"tracks":              grammar.Rev, // tracks_publication.go `poserLesTraces` <- FilmInputs.Positions
	"originMs":            grammar.Rev, // build_pistes.go `resolveOriginMs` <- FilmInputs.FilmClockOriginUS + fil des morts
	"t0FilmMs":            grammar.Rev, // build_pistes.go `DetectT0Film` <- pistes publiees
	"shots":               grammar.Rev, // build_pistes.go + vehicle_shots.go <- FilmInputs.Fire
	"loadouts":            grammar.Rev, // build_pistes.go <- FilmInputs.Loadouts
	"projectiles":         grammar.Rev, // build_pistes.go <- FilmInputs.Projectiles
	"grenades":            grammar.Rev, // build_pistes.go <- FilmInputs.Grenades
	"inventory":           grammar.Rev, // build_inventaire.go <- FilmInputs.Inventory
	"grenadeReads":        grammar.Rev, // build_inventaire.go <- FilmInputs.Inventory + InventoryDeltas
	"abilities":           grammar.Rev, // build_inventaire.go <- FilmInputs.AbilityRanks + Inventory
	"equipmentChanges":    grammar.Rev, // build_inventaire.go <- FilmInputs.EquipmentChanges
	"translocations":      grammar.Rev, // build_inventaire.go <- FilmInputs.Translocations
	"abilityImpulses":     grammar.Rev, // build_inventaire.go <- FilmInputs.AbilityImpulses
	"abilityCharges":      grammar.Rev, // build_inventaire.go <- FilmInputs.AbilityCharges
	"weaponLabels":        grammar.Rev, // build_inventaire.go `buildWeaponLabels` <- les identifiants d armes de loadouts / shots / weaponPads
	"abilityLabels":       grammar.Rev, // build_inventaire.go `abilityLabelsUsed` <- les rangs i48 de `abilities`
	"grappleLines":        grammar.Rev, // build_calques.go <- FilmInputs.GrappleReads
	"equipmentPlacements": grammar.Rev, // build_calques.go <- FilmInputs.Placements + SpawnEvents
	"weaponChanges":       grammar.Rev, // build_calques.go <- FilmInputs.WeaponChanges
	"pickups":             grammar.Rev, // build_calques.go <- FilmInputs.Pickups
	"weaponPads":          grammar.Rev, // build_ground_weapons.go <- FilmInputs.Pads (ti=42 et ti=37)
	"padPickups":          grammar.Rev, // build_ground_weapons.go <- FilmInputs.Pads
	"groundWeapons":       grammar.Rev, // build_calques.go `buildGroundWeaponItems` <- FilmInputs.Pads + WeaponChanges
	"objectiveObjects":    grammar.Rev, // build_objective_objects.go <- FilmInputs.Pads.Weapons SEUL : son en-tete mesure « aucune lecture de film ajoutee », ni statborg ni morts
	"vehicles":            grammar.Rev, // build_vehicles.go <- FilmInputs.Vehicles
	"zoneStates":          grammar.Rev, // build_zones.go <- FilmInputs.ZoneReads (le catalogue de zones vient de l appelant, il ne decode rien)

	// --- LES FAITS (`facts.Rev`, orthographiee `killsource-...`) : les calques dont les lignes
	// sortent du statborg ou du decodage killsource, tous deux faits PAR L APPELANT
	// (`internal/replaybuild`) et passes par des options que `FilmInputs.applyTo` n ecrit pas.
	"identity":          facts.Rev, // build_pistes.go <- `reg.Section`, qui publie `statborgSlots` (facts/objectives)
	"roster":            facts.Rev, // build_pistes.go `buildRoster` <- opt.Bots AJOUTE des lignes (identity.go), et les bots sont decodes par facts/killsource
	"neutralDeaths":     facts.Rev, // build_inventaire.go <- opt.NeutralDeaths, resolues par `replaybuild.neutralDeaths` sur `decfilm.Result`
	"equipmentEpisodes": facts.Rev, // build_pistes.go : les episodes sortent de la grammaire, mais `attachAllEquipmentKills` y ECRIT `k`/`a` depuis opt.Kills (killsource)
	"objectives":        facts.Rev, // objectives.go <- opt.Objectives (`objectives.IdentifiedEvent`)
	"scoreTimeline":     facts.Rev, // build_score.go <- opt.Score.Records (enregistrements d entite du statborg)
	"flagCarries":       facts.Rev, // build_objectives_live.go <- opt.Flag.Records (statborg)
	"flagReturnZone":    facts.Rev, // build_objectives_live.go, meme entree que `flagCarries`
	"vipCrown":          facts.Rev, // vip_crown.go <- opt.Vip.Records (statborg)
	"skullCarries":      facts.Rev, // skull_carries.go <- opt.Skull.Records (statborg)
	"bombArmings":       facts.Rev, // bomb_armings.go : l anneau vient de FilmInputs.BombReads, mais les armements sont DATES par les explosions de `doc.Objectives` (statborg)
	"bombCarries":       facts.Rev, // bomb_carries.go : les transitions viennent de opt.WeaponChanges, et la chronologie est pontee par le registre (statborg + bots)
	"bombStats":         facts.Rev, // bomb_stats_document.go <- armements, portages, actions d objectif et opt.MatchKills
	"bombEvents":        facts.Rev, // bomb_stats_document.go, meme entree que `bombStats`
}

// calquesSansCouche — LES CHAMPS RACINE CUITS QUI N ONT AUCUNE COUCHE PRODUCTRICE, avec la
// justification DATEE de chacun.
//
// UNE TABLE VIDE SERAIT LE RATCHET ; chaque entree est une DECISION, et le test exige qu elle
// porte sa date. Un champ racine cuit qui n est ni ici ni dans [couchesDesCalques] fait rougir
// `TestCalquesCouvrentTousLesChampsRacineCuits` — c est le sens de la faute qu on veut : classer
// un champ coute une ligne, l oublier ouvrirait un trou.
var calquesSansCouche = map[string]string{
	// N EST PAS CUIT DU TOUT, et c est mesure : aucun chemin de `build*.go` ne pose
	// `doc.VehicleLabels` ; le seul ecrivain du depot est
	// `internal/service/replay_vehicle_labels.go` (`resolveVehicleLabels`), a la REQUETE, comme
	// `mapObjectives` et ses deux voisins. Il n est pourtant PAS dans `calquesALaRequete`
	// (`document_shape_test.go`) : l y ajouter changerait l empreinte de forme cuite, ce que ce
	// lot n a pas le droit de faire. Consigne au §4 du plan, non traite.
	"vehicleLabels": "2026-09-17 — resolu a la requete par le service (replay_vehicle_labels.go), jamais par la cuisson",
	// N EST PAS UN CALQUE : `coverage` est la MESURE de la cuisson, toutes couches confondues
	// (47 balises, de `tracks` a `decoder`). Lui donner une couche serait faux dans les deux
	// sens — elle bouge avec n importe laquelle des quatre, et une montee de schema la change
	// aussi. C est d ailleurs `coverage` que `layers` complete : l une dit CE QUI a ete lu,
	// l autre SOUS QUELLE REVISION.
	"coverage": "2026-09-17 — mesure de la cuisson, tous calques confondus : aucune couche unique ne la produit",
	// SUBSTANCE = UN CATALOGUE QU AUCUNE DES CINQ REVISIONS NE HACHE (limite 2 en tete de
	// fichier). Les quatre suivants sortent du catalogue FIGE de la carte, recopie verbatim par
	// `ouvrir` (`opt.Geometry` / `opt.Structure`, poses par `replaybuild.buildReplayOptions`) ou
	// calcule sur lui. Les attribuer a la publication laisserait croire qu une montee de schema
	// est la seule chose qui les change, alors que c est le fichier de carte.
	"geometry":        "2026-09-17 — catalogue fige de la carte, recopie verbatim : aucune revision ne hache ses donnees",
	"geometryBounds":  "2026-09-17 — calcul de la publication SUR ce catalogue de carte : meme raison que `geometry`",
	"structure":       "2026-09-17 — catalogue fige de la carte, recopie verbatim : aucune revision ne hache ses donnees",
	"structureBounds": "2026-09-17 — calcul de la publication SUR ce catalogue de carte : meme raison que `structure`",
	// SUBSTANCE = LE MANIFESTE DE LIBELLES DU TITRE, recopie table entiere (`opt.Labels.Effects`,
	// `opt.Labels.Grenades`). Contrairement a `weaponLabels` et `abilityLabels`, dont la
	// POPULATION est decodee (les identifiants viennent des calques de grammaire), ces deux-ci ne
	// doivent leur contenu qu au manifeste `config/titles/{slug}/mappings/`.
	"killEffects":   "2026-09-17 — table du manifeste du titre recopiee verbatim : aucune couche ne la produit",
	"grenadeLabels": "2026-09-17 — table du manifeste du titre recopiee verbatim : aucune couche ne la produit",
}

// gardesDeProduction — LES CALQUES DONT LA PASSE NE TOURNE PAS QUAND UNE GARDE EST FERMEE.
//
// C est la SEULE raison pour laquelle une entree manque a un `layers` present, et c est ce qui
// fait de cette absence une REPONSE. Chaque garde est un refus ECRIT dans la production, cite
// avec son site : une garde de mode posee par l appelant (famille bomb, CTF, VIP, Oddball,
// catalogue de zones), un balayage qui n a pas abouti (`Scanned`), ou une entree que l appelant
// n a pas fournie.
//
// UN CALQUE ABSENT DE CETTE TABLE A TOUJOURS SON ENTREE, fut-il vide : sa passe a tourne, et
// « le film n en porte aucun » est une reponse que `layers` doit savoir donner.
var gardesDeProduction = map[string]func(doc *ReplayDocument, opt Options) bool{
	// `DetectT0Film` ne tourne pas sans origine : un instant cale sur zero serait faux de 3,6 s
	// a 50,8 s selon le match (build_pistes.go, `composerLaCouverture`).
	"t0FilmMs": func(doc *ReplayDocument, _ Options) bool { return doc.OriginMs != nil },
	// L ancre d une traction exige les bornes de la carte : sans `MapQuant`, aucune traction
	// n est publiee (build_calques.go, `poserGrappinEtPoses`).
	"grappleLines": func(_ *ReplayDocument, opt Options) bool { return opt.MapQuant != nil },
	// Inventaire ILLISIBLE : `BuildFromFilm` retombe sur `inventory = nil`, et la couverture
	// reste absente plutot que de publier quatre zeros (inventory.go, `attachInventoryCoverage`).
	// La tranche NULLE et la tranche VIDE ne disent pas la meme chose.
	"inventory": func(_ *ReplayDocument, opt Options) bool { return opt.Inventory != nil },
	// Le film ne declare NI i57 NI i59 / NI i56 : le balayage n a jamais commence
	// (build_inventaire.go, `poserImpulsionsEtCharges` — meme patron que l inventaire).
	"abilityImpulses": func(_ *ReplayDocument, opt Options) bool { return opt.AbilityImpulseStats.Scanned },
	"abilityCharges":  func(_ *ReplayDocument, opt Options) bool { return opt.AbilityChargeStats.Scanned },
	// Sans enregistrements d entite, ni courbe ni couverture de score (score_timeline.go,
	// `buildScoreTimeline` : `in == nil` -> `nil, nil`).
	"scoreTimeline": func(_ *ReplayDocument, opt Options) bool { return opt.Score != nil },
	// GARDES DE MODE, posees par l appelant sur `game_variant_name` : hors CTF / VIP / Oddball /
	// famille bomb, le calque n est pas balaye (flag_carries.go `buildFlagCarries`,
	// vip_crown.go `attachVipCrown`, skull_carries.go `attachSkullCarries`,
	// bomb_armings.go `attachBombArmings`, bomb_carries.go `attachBombCarries`,
	// bomb_stats_document.go `attachBombStats`).
	"flagCarries":    func(_ *ReplayDocument, opt Options) bool { return opt.Flag.Scanned },
	"flagReturnZone": func(_ *ReplayDocument, opt Options) bool { return opt.Flag.Scanned },
	"vipCrown":       func(_ *ReplayDocument, opt Options) bool { return opt.Vip.Scanned },
	"skullCarries":   func(_ *ReplayDocument, opt Options) bool { return opt.Skull.Scanned },
	"bombArmings":    func(_ *ReplayDocument, opt Options) bool { return opt.Bomb.Scanned },
	"bombCarries":    func(_ *ReplayDocument, opt Options) bool { return opt.Bomb.CarryScanned },
	"bombStats":      func(_ *ReplayDocument, opt Options) bool { return opt.Bomb.CarryScanned },
	"bombEvents":     func(_ *ReplayDocument, opt Options) bool { return opt.Bomb.CarryScanned },
	// Le catalogue de zones de l appelant COMMANDE le balayage de `ti=13` : sans zones, rien
	// n est lu (zone_states.go, `buildZoneStates`).
	"zoneStates": func(_ *ReplayDocument, opt Options) bool { return opt.Zone.Scanned },
	// Archetype ti=40 absent des images-cles, creations illisibles, ou assemblage sur positions
	// figees sans film (vehicle_tracks.go, `buildVehicleTracks`).
	"vehicles": func(_ *ReplayDocument, opt Options) bool { return opt.Vehicles.Scanned },
}

// calquesProduits rend, pour CETTE cuisson, les calques produits et la revision de la couche qui
// les a produits — la forme exacte que `layers` portera a la racine du document.
//
// ELLE N EST APPELEE PAR PERSONNE AU LOT 4.2.1-a, et c est voulu : le champ qui la consomme
// arrive avec la montee de schema (lot 4.2.1-b). Ce lot-ci fige la TABLE et sa fermeture.
//
// `doc` nil rend nil : un appelant sans document n a aucun calque a declarer, et fabriquer une
// table de calques pour un document qui n existe pas serait une affirmation.
func calquesProduits(doc *ReplayDocument, opt Options) map[string]string {
	if doc == nil {
		return nil
	}
	out := make(map[string]string, len(couchesDesCalques))
	for nom, revision := range couchesDesCalques {
		if garde, gardee := gardesDeProduction[nom]; gardee && !garde(doc, opt) {
			continue
		}
		out[nom] = revision
	}
	return out
}
