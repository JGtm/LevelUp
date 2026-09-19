package replay

// layers.go — LA TABLE DES CALQUES : QUELLE COUCHE A PRODUIT QUOI (lot 4.2.1-a du
// PLAN_DECODEUR_FILM_2026-09-13 ; ADR 0034 D-6 et D-7).
//
// # CE QUE CE FICHIER PORTE, ET OU IL ABOUTIT
//
// La TABLE, sa fermeture, et la passe qui pose [ReplayDocument.Layers] — publie a la racine
// DEPUIS LE SCHEMA 62 (lot 4.2.1-b). La table a ete ecrite et fermee au commit precedent SANS
// montee de schema, delibere : une montee marque tout le parc a recuire
// (`Digest.UpToDate`, `internal/replaybuild/artifact_digest.go`), et ce qui se discute dans une
// table de calques est son CONTENU ligne a ligne, pas la forme qui la transporte.
//
// LA POSE EST LA DERNIERE PASSE de `BuildFromPositions`, apres `clore` : les passes precedentes
// ecrivent encore (la seconde porte des tirs deplace des evenements, les replis se publient en
// dernier), et une table lue trop tot decrirait un document qui n existe pas encore.
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
// # LES QUATRE CALQUES A LA REQUETE N Y ENTRENT JAMAIS
//
// `mapObjectives`, `mapWeaponPads`, `weaponTiers` et `vehicleLabels` sont resolus PAR LE SERVICE,
// a la requete : la cuisson ne les ecrit pas (garde
// `TestDocumentShapeCalquesALaRequeteRestentHorsCuisson`, `document_shape_test.go`). Ils ne sont
// ni dans la table, ni dans les exemptions.
//
// `vehicleLabels` EST ENTRE DANS CETTE LISTE AU SCHEMA 62, et c est une correction : la mesure du
// 2026-09-17 (ecriture de cette table) a montre qu aucun chemin de `build*.go` ne pose
// `doc.VehicleLabels` — son seul ecrivain du depot est
// `internal/service/replay_vehicle_labels.go` (`resolveVehicleLabels`), a la requete, exactement
// comme ses trois voisins. Il manquait a `calquesALaRequete` depuis son ajout. Le reclasser
// change l empreinte de forme CUITE, donc il ne pouvait entrer que dans un commit qui monte
// `SchemaVersion` — celui-ci (decision de pilote du 2026-09-17 : « c est le seul moment ou ce
// reclassement est gratuit »).

import (
	"strconv"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
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
	"vehicleCycles":       grammar.Rev, // vehicle_cycles.go <- `doc.Vehicles` SEUL : une couche d ANALYSE sur les vies deja publiees, donc la MEME revision que le calque dont elle derive
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
	"flagCarries":       facts.Rev, // build_objectives_live.go <- opt.Flag.Records (statborg) ; la jauge de retour vient de la grammaire (`FilmInputs.FlagGauge`), mais les LACHERS auxquels elle s apparie sortent du statborg — la revision la plus tardive gagne
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
	// N EST PAS UN CALQUE : `coverage` est la MESURE de la cuisson, toutes couches confondues
	// (47 balises, de `tracks` a `decoder`). Lui donner une couche serait faux dans les deux
	// sens — elle bouge avec n importe laquelle des quatre, et une montee de schema la change
	// aussi. C est d ailleurs `coverage` que `layers` complete : l une dit CE QUI a ete lu,
	// l autre SOUS QUELLE REVISION.
	"coverage": "2026-09-17 — mesure de la cuisson, tous calques confondus : aucune couche unique ne la produit",
	// NE SE DECRIT PAS LUI-MEME : `layers` est LA REPONSE sur les calques, comme `coverage` est
	// leur MESURE. Une entree `layers: <revision>` serait soit circulaire (la publication le pose,
	// mais ses valeurs bougent avec les quatre couches), soit fausse dans un sens ou dans l autre.
	// Un lecteur qui veut savoir si la table est la lit l OBJET, pas une entree dedans.
	"layers": "2026-09-17 — la reponse sur les calques, pas un calque : une entree sur elle-meme serait circulaire",
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
// `doc` nil rend nil : un appelant sans document n a aucun calque a declarer, et fabriquer une
// table de calques pour un document qui n existe pas serait une affirmation.
//
// UN DOCUMENT SANS AUCUNE PISTE NE DECLARE QUE LA PUBLICATION. `ouvrir` rend faux quand le film
// ne porte aucune position, et `BuildFromPositions` publie alors le document TEL QUEL : aucune
// des treize passes de calque n a tourne. Declarer produits les trente-et-un calques non gardes
// serait un mensonge, et precisement celui que cette table existe pour empecher. Le temoin est
// `FrameCount`, qui vaut zero dans ce cas exact et au moins un des qu une position est publiee
// (`frameSpan`, build.go).
func calquesProduits(doc *ReplayDocument, opt Options) map[string]string {
	if doc == nil {
		return nil
	}
	assemble := doc.FrameCount > 0
	out := make(map[string]string, len(couchesDesCalques))
	for nom, revision := range couchesDesCalques {
		if !assemble && revision != revisionDeLaPublication {
			continue
		}
		if garde, gardee := gardesDeProduction[nom]; gardee && !garde(doc, opt) {
			continue
		}
		out[nom] = revision
	}
	return out
}

// poserLesCalquesProduits publie [ReplayDocument.Layers] — LA DERNIERE PASSE de l assemblage.
//
// Elle ne pose RIEN quand la table ne rend rien : un document sans le moindre calque garderait
// alors `layers` absent, ce qui voudrait dire « artefact anterieur au schema 62 ». Le cas n est
// pas atteignable aujourd hui (les quatre calques de publication sont toujours produits) et la
// garde est la pour que le jour ou il le deviendrait, l ambiguite ne renaisse pas en silence.
func (a *assemblage) poserLesCalquesProduits() {
	if calques := calquesProduits(&a.doc, a.opt); len(calques) > 0 {
		a.doc.Layers = calques
	}
}

// RevisionsCourantesDesCouches rend, PAR FAMILLE, la revision que le binaire courant applique.
//
// # POURQUOI CE PAQUET L EXPORTE, ET POURQUOI C EST LE SEUL QUI PEUT
//
// La recuisson selective (lot 4.4.1) doit comparer les revisions LUES dans un artefact a celles
// du binaire courant. Les quatre revisions de couche vivent sous `film/internal/`, donc
// inaccessibles a `internal/replaybuild` : le compilateur l interdit, et c est la frontiere qui le
// veut (ADR 0034, D-1). `film/replay` est la couche de PUBLICATION — elle les importe deja toutes
// pour composer `coverage.decoder` — donc elle est le seul point ou cette table peut se lire sans
// ouvrir la frontiere.
//
// LA CLE EST LE NOM DE FAMILLE, et le prefixe d une revision est ce nom suivi d un tiret
// (`grammar-2026-09-15.42` -> famille `grammar`). Un appelant classe donc une valeur lue dans un
// artefact sans connaitre les cinq noms : il coupe au premier tiret. C est la forme qui evite une
// seconde table de prefixes chez le consommateur — la troisieme copie qu interdit la regle 6.
//
// LA FAMILLE DES FAITS S APPELLE `killsource`, et pas `facts` : c est la valeur de `facts.Rev`
// qui le decide, et la table dit ce que les artefacts PORTENT, pas ce que l architecture nomme.
func RevisionsCourantesDesCouches() map[string]string {
	return map[string]string{
		"source":      source.Rev,
		"profile":     profile.Rev,
		"grammar":     grammar.Rev,
		"killsource":  facts.Rev,
		"publication": revisionDeLaPublication,
	}
}

// FamilleDeRevision rend la famille d une revision de couche telle qu un artefact la porte, ou
// la chaine vide quand la valeur n a pas la forme attendue.
//
// UNE FONCTION ET PAS UN `strings.Cut` CHEZ L APPELANT : la forme d une revision est une
// propriete de CE paquet (c est lui qui les compose), et deux lectures du meme prefixe
// divergeraient au premier renommage.
func FamilleDeRevision(revision string) string {
	i := strings.IndexByte(revision, '-')
	if i <= 0 {
		return ""
	}
	return revision[:i]
}
