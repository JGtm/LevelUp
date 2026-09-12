// Package replay assemble l'artefact de rejeu 2D (vue du dessus) d'un match à partir
// des données décodées du film : trajectoires des joueurs (Étape A) et géométrie de
// carte (Étape B) ; le kill feed (Étape C) reste à faire. Assemblage pur — aucun accès
// DB ni HTTP ; le décodage lourd est délégué à internal/analysis/filmdec.
//
// Le document (ReplayDocument) est produit HORS LIGNE par cmd/replay-build à partir des
// SEULS chunks du film (zéro capture Cheat Engine) et servi tel quel par l'API. C'est
// délibérément un DTO d'artefact bespoke (pas un type canonical) : c'est une charge utile
// de rendu, versionnée par SchemaVersion pour la compat client.
//
// Repère : les positions sont en MÈTRES MONDE. Ce paragraphe disait le contraire
// (« l'échelle/offset absolus ne sont PAS garantis », handoff ALL_PLAYERS_TRAJECTORIES) et
// c'était vrai AVANT filmdec/map_bounds.go : le film ne porte que des indices de quantum, et
// tant que les bornes du BSP manquaient, la déquantification employait celles de Cliffhanger
// pour toutes les cartes — d'où un facteur d'échelle arbitraire. Depuis, cmd/replay-build
// EXIGE la carte (`-map`) et refuse de produire un artefact sans ses bornes.
//
// CE QUE ÇA AUTORISE, et pourquoi la correction n'est pas cosmétique : le fond de carte
// figé (`MapBackground`) est calé dans ce même repère, donc il se superpose au rejeu.
// Contrôle sur 000d5950 : les bornes du rejeu tombent dans le cadre de `ridgeline.png`
// (cf. TestMapBackground_DonneesReelles). Le client garde son auto-cadrage via Bounds.
package replay

// SchemaVersion est incrémenté quand la forme de ReplayDocument change d'une façon que le
// client web doit gérer. L'ajout de champs OPTIONNELS (omitempty) ne casse pas le client
// et n'incrémente pas la version ; seul un changement cassant le fait.
//
// EXCEPTION ASSUMÉE AU 2026-09-08 (v49, lot M1b, décision utilisateur ferme) : le champ ajouté
// (`coverage.bridge.deathOffsetMs`) est bien optionnel — le web sait déjà distinguer « connu »
// de « absent » sur le seul pointeur, sans regarder cette version. Le bump n'est donc pas
// exigé par la RÈGLE ci-dessus ; il est posé quand même pour que `backfill-replay` recuise
// tout le parc < 49 en une passe (plutôt que de laisser un artefact ancien répondre « calage
// inconnu » indéfiniment faute de recuisson déclenchée) — cf.
// `.ai/V7.5/v2/CHRONIQUE_49_2026-09-08.md`.
//
// v50 (2026-09-08, lot P2) : la section `identity` naît, `roster[].bid` est publié, et le
// nommage des vies change (élimination sur le roster). Le CONTENU CUIT change, donc le bump est
// exigé par la règle — pas seulement pour déclencher la recuisson.
//
// v51 (2026-09-10, lot 4.3) : deux changements de CONTENU CUIT — `identity.bipedSlots[].bid`
// (le tableau de l'API nomme les corps hors table) et `abilityLabels[].family` (le résumé
// d'usage y joint, d'où `UsageSummaryRev` us4 -> us5). Raison détaillée : `structure_test.go`.
//
// v52 (2026-09-11, lot B décodeur) : TROIS changements de CONTENU CUIT, chacun mesuré.
// `grenades[]` change de position ET porte désormais son `slot` sur les deux branches (le
// lancer revient à son lanceur) ; `projectiles[]` est coupé au premier pas impossible, `rest`
// tombant à false sur un vol coupé ; `geometry` devient les props de LA carte du match, donc
// vide sur toute carte non extraite. Le bump est exigé par la règle — le contenu change, et un
// artefact v51 doit se voir comme « à recuire », pas comme à jour. `coverage.projectiles`
// s'ajoute au passage, mais un champ optionnel ne l'aurait pas exigé à lui seul. Chronique
// détaillée et chiffres : `document_chronicle.go`.
// v53 (2026-09-12, lot B-bis) : UNE SEULE CAUSE, et elle ne touche qu'une carte du catalogue —
// mais plusieurs calques en dépendent. La porte du composant de position des objets du monde
// était écrite en dur à 3 bits ; l'index de région qu'elle porte fait DEUX bits sur Live Fire.
// Le décodeur y lisait les trois axes un bit trop tôt et rendait des positions fausses, que le
// garde-fou de v52 coupait ensuite au deuxième ou troisième point. Mesuré sur les quatre films
// Live Fire du parc : 614 vols de projectile tronqués -> 5, et 1 067 points publiés -> 8 378 ;
// les poses d'équipement passent de 49 et 27 à 227 et 114 sur les deux films mesurés, et les
// lancers de grenade retrouvent leur lien vers le projectile (2 -> 85, 0 -> 137). Aucune autre
// carte ne bouge d'un point (Cliffhanger, Banished Narrows, The Pit, Isolation : identiques).
// Chronique et chiffres : `document_chronicle.go`.
//
// v54 (2026-09-12, lot G.2bis) : UN SEUL changement, et il ne touche que les films de
// FilmMajorVersion 39-40 (mars a novembre 2025, 211 des 1 351 du cache) — mais sur ceux-la il
// change le ROSTER. `ScanDeaths` passait `filmMajorVersion = 0` au parseur d'events, donc le
// decoupage « gamertag en tete » ; sur ces films le gamertag vit douze octets plus loin et la
// lecture ramenait du rembourrage. La version est desormais LUE dans l'en-tete du registre du
// film. Mesure sur pieces (`gamertagsOf`, la table qui nomme `roster[]`) : `e5adf7b2` passe de
// 17 identites nommees / 2 noms distincts a 26 / 26, `111fa685` de 16 / 2 a 24 / 24. Temoins de
// version 41 (`000d5950`, `5676a9ba`) : identiques, 8 / 8 et 26 / 26 des deux cotes. Le bump est
// exige par la regle — le contenu cuit change — et il est ce qui rendra ces artefacts candidats
// a `backfill-replay --only-existing`. `coverage.filmMajorVersion` s'ajoute au passage — la
// version lue voyage desormais AVEC l'artefact ; champ optionnel, il ne l'aurait pas exige a
// lui seul. Chronique : `document_chronicle.go`.
const SchemaVersion = 54

// ReplayDocument est le rejeu 2D sérialisé d'un match.
type ReplayDocument struct {
	SchemaVersion int    `json:"schemaVersion"`
	MatchID       string `json:"matchId"`
	TitleSlug     string `json:"titleSlug"`
	// FrameCount est le nombre de pas de temps discrets. Les points de trajectoire
	// référencent cet axe via Point.T dans [0, FrameCount).
	FrameCount int     `json:"frameCount"`
	Bounds     Bounds  `json:"bounds"`
	Tracks     []Track `json:"tracks"`

	// --- Champs OPTIONNELS (n'incrémentent pas SchemaVersion) ---

	// FrameIntervalMS est la durée réelle d'un pas de temps, en millisecondes. Absent =
	// axe de temps sans échelle (anciens artefacts) : le client choisit sa cadence.
	// Présent, la vitesse « 1x » vaut 1000/FrameIntervalMS frames par seconde réelle.
	FrameIntervalMS int `json:"frameIntervalMs,omitempty"`
	// DurationMS est la durée réelle couverte par le rejeu, en millisecondes.
	DurationMS int `json:"durationMs,omitempty"`
	// OriginMs est l'instant de la FRAME 0 sur l'horloge du fil des éliminations, en
	// millisecondes — c'est-à-dire sur l'horloge que le client reconstruit par
	// `event_time_ms + t0_ms` (le T0 réel du match, déjà servi par la Match View).
	//
	// CE QU'IL FERME. La frame 0 est calée sur le premier paquet de POSITION du film, un
	// instant qui varie de 3,6 s à 39,8 s après le début du film selon le match (chargement
	// et mise en place). Sans ce champ, poser le fil sur l'axe du rejeu exigeait de mesurer
	// ce décalage par appariement statistique côté navigateur — ce qui suppose des victimes
	// nommées, et échoue quand `killer_victim_pairs` ne couvre pas le match. Avec lui, le
	// recalage est une soustraction : `replayMs = event_time_ms + t0_ms − originMs`.
	//
	// D'OÙ IL VIENT : la différence de deux en-têtes de paquet du MÊME film (premier paquet
	// de position − premier paquet du chunk 1, qui est le zéro du film). Aucune base, aucune
	// horloge murale, aucune estimation. Détail, mesures et témoin indépendant : origin.go.
	//
	// POINTEUR, PAS int : c'est le PIÈGE omitempty. Une origine de zéro (film dont le
	// premier paquet porte déjà une position) serait omise et relue comme « pas d'origine »,
	// donc traitée en repli alors qu'elle est mesurée. ABSENT veut dire, et seulement :
	// l'origine n'est pas établie — le client retombe alors sur son appariement.
	OriginMs *int64 `json:"originMs,omitempty"`
	// T0FilmMs est LE COUP D'ENVOI DU MATCH mesuré dans le film, sur la MÊME horloge
	// qu'`OriginMs` (celle du fil des éliminations) : l'instant où la grille se lève, daté par
	// le premier mouvement des pistes.
	//
	// CE QU'IL FERME. Le T0 servi jusqu'ici est estimé des `first_joined_time` de l'API ; sur
	// 10-15 % des matchs ces horodatages collent au `start_time` et le T0 tombe à ~0, ce qui
	// fait démarrer le rejeu sur des joueurs statufiés pendant tout le décompte. Ce champ le
	// mesure au lieu de l'estimer, et la mesure est plus stable que l'étalon (écart-type
	// 9 752 ms contre 12 764 ms sur les 49 matchs au T0-API sain).
	//
	// D'OÙ IL VIENT : `originMs` plus la frame du premier mouvement détecté, toutes pistes
	// confondues. Aucune base, aucune horloge murale, aucune constante de jeu en dur — le
	// détecteur mesure par match. Seuils, refus et mesures : t0_film.go.
	//
	// POINTEUR, PAS int64 : même PIÈGE omitempty que sur `OriginMs` ci-dessus. Un coup d'envoi
	// exactement à zéro (film dont la frame 0 est le coup d'envoi ET dont l'origine est nulle)
	// serait omis et relu comme « pas de mesure ». ABSENT veut dire, et seulement : le
	// détecteur a REFUSÉ — `coverage.t0Film.reason` dit lequel des trois refus, et le client
	// retombe sur le T0 de l'API.
	T0FilmMs *int64 `json:"t0FilmMs,omitempty"`
	// Geometry est le fond de carte : props Forge orientés (repères contextuels, pas les
	// sols). Absent si la géométrie n'a pas été fournie au build.
	Geometry []MapObject `json:"geometry,omitempty"`
	// GeometryBounds est l'étendue XY de Geometry, distincte de Bounds (les props
	// débordent de la zone parcourue). Le client peut cadrer sur l'union des deux.
	GeometryBounds *Bounds `json:"geometryBounds,omitempty"`
	// Structure est la géométrie STRUCTURELLE de la carte : l'emprise au sol de chaque
	// instance de géométrie instanciée du BSP (sols, plateformes, rampes, murs), avec
	// l'altitude de sa face supérieure. C'est le VRAI fond de carte, à distinguer de
	// Geometry (props Forge, 0,25 m² de médiane, 3,4 % de la carte couverts).
	// Absente si la carte n'a pas de fichier de structure figé (cf. cmd/mapstruct-build).
	Structure []Surface `json:"structure,omitempty"`
	// StructureBounds est l'étendue XY de Structure (elle déborde largement de Bounds :
	// la structure couvre toute la carte, les joueurs n'en parcourent qu'une partie).
	StructureBounds *Bounds `json:"structureBounds,omitempty"`
	// Shots est la liste des tirs décodés du film et RATTACHÉS à un slot (cf. shots.go).
	// Absent si le décodage n'a rien pu rattacher. Ce n'est PAS la liste exhaustive des
	// tirs du match : voir Shot pour ce que le champ garantit et ce qu'il ne garantit pas.
	Shots []Shot `json:"shots,omitempty"`
	// Loadouts est l'inventaire d'armes de chaque slot aux instants de keyframe (cf.
	// loadouts.go). Absent si le film n'a livré aucun loadout.
	Loadouts []Loadout `json:"loadouts,omitempty"`
	// Inventory est l'inventaire complet lu aux images-clés : grenades portées avec leur type,
	// capacité d'armure, munitions et emplacement dégainé (cf. inventory.go). Absent si le film
	// n'a livré aucun état.
	Inventory []Inventory `json:"inventory,omitempty"`
	// GrenadeLabels nomme les RANGS de type de grenade, dans l'ordre des compteurs
	// d'Inventory.G — et c'est LA SEULE table qui les nomme, y compris pour le type d'un
	// lancer (Grenade.Rank y est un index). Deux chaînes indépendantes établissent
	// l'ordre (35 lancers appariés aux décréments, et la table du binaire) : la question
	// est close. Source : replay_labels.toml du titre.
	GrenadeLabels []Label `json:"grenadeLabels,omitempty"`
	// Abilities est le RANG DE PALETTE de la capacité d'armure portée, lu au fil du film
	// (cf. abilities.go). Absent si aucun canal n'a rien rendu.
	Abilities []AbilityRead `json:"abilities,omitempty"`
	// GrenadeReads est l'axe des GRENADES PORTEES, alimente par DEUX canaux : le record de
	// biped des images-cles et les composants i22/i47 des paquets delta, chaque lecture disant
	// d'ou elle vient (cf. grenade_reads.go). Absent si aucun canal n'a rien rendu.
	//
	// IL NE REMPLACE PAS `Inventory` : celui-ci reste la source des munitions, de l'emplacement
	// degaine et du marqueur de lecture vide. Les deux axes coexistent parce qu'ils n'ont pas la
	// meme cadence, et les melanger ferait masquer une lecture pleine par une lecture partielle.
	GrenadeReads []GrenadeRead `json:"grenadeReads,omitempty"`
	// AbilityLabels nomme les RANGS de capacité que le document emploie.
	//
	// LA TABLE EST PARTIELLE, et un rang absent GARDE SON NUMÉRO à l'écran, marqué non
	// interprétable. Combler par le nom d'une capacité voisine se lirait comme une certitude.
	// La table est de surcroît PROPRE À LA PALETTE du match (cf. abilities.go) : deux films
	// peuvent donner deux noms différents au même rang, et un film dont la palette n'est pas
	// classée ne reçoit AUCUN nom.
	AbilityLabels map[string]Label `json:"abilityLabels,omitempty"`
	// EquipmentEpisodes est l'état ACTIF d'un équipement, en épisodes datés par vie
	// (cf. equipment_episodes.go) : camouflage (i28 queue[1], interrupteur mesuré) et
	// surbouclier (i5 non clampé, règle q > 64). Deux familles SEULEMENT, parce que deux
	// seulement sont mesurées — les autres équipements restent sans état plutôt que
	// devinés. Absent si aucune vie publiée ne porte d'épisode.
	EquipmentEpisodes []EquipmentEpisode `json:"equipmentEpisodes,omitempty"`
	// GrappleLines est la liste des TRACTIONS de grappin (cf. grapple_lines.go) : la
	// fenêtre datée [t0, t1] — du tir à l'ARRIVÉE mesurée sur la trajectoire — et le point
	// d'accroche en coordonnées monde. La position du joueur pendant la fenêtre est celle
	// de sa Track : la ligne se trace de la position courante vers l'ancre. Absent si
	// aucune traction n'a été lue (film sans grappin, ou vies non publiées).
	GrappleLines []GrappleLine `json:"grappleLines,omitempty"`
	// EquipmentPlacements est la liste des POSES d'objets d'équipement sur la carte
	// (cf. equipment_placements.go) : position monde, fenêtre [t0, t1], famille
	// (`wall` / `sensor` / `other`), identifiant `eqip` du jeu, poseur mesuré (-1 quand aucun)
	// et cap de visée du poseur quand il a été lu.
	//
	// `t1` EST UNE MISE AU REPOS, PAS UNE DISPARITION : le film ne date la disparition d'aucun
	// objet d'équipement (mesure du 2026-08-18). La fenêtre publiée est donc une BORNE
	// INFÉRIEURE de la présence de l'objet ; effacer la pose à `t1` affirmerait une disparition
	// que rien ne mesure.
	//
	// UNE POSE N'EST PAS RATTACHÉE À UNE Track : c'est un objet du monde, pas un joueur.
	// `owner` désigne la VIE qui l'a posé quand la proximité l'atteste, et rien d'autre.
	// Absent si le film n'a pas tranché la largeur de son bloc de réplication (la couverture
	// le dit alors : `calibrated: false`) ou s'il ne porte aucune pose.
	EquipmentPlacements []EquipmentPlacement `json:"equipmentPlacements,omitempty"`
	// WeaponChanges est la liste des PRISES ET DES LÂCHERS d'arme (cf.
	// document_weapon_changes.go) : qui, quand, quelle arme, et — sur un lâcher — jusqu'à
	// quelle frame le client peut montrer l'arme au sol. Les ré-annonces d'une arme déjà
	// portée au spawn en sont ÉCARTÉES : ce ne sont pas des ramassages. Absent si le film
	// n'en porte aucun.
	WeaponChanges []WeaponChange `json:"weaponChanges,omitempty"`
	// Pickups est la liste des RAMASSAGES NATIFS (cf. document_pickups.go) : l'événement
	// `biped_pickup` de la bobine, daté à la milliseconde, ATTRIBUÉ à son ramasseur et portant
	// l'identifiant de catalogue de l'objet.
	//
	// IL NE REMPLACE PAS `weaponChanges`, IL LE COMPLÈTE — et les deux sont publiés parce
	// qu'ils ne disent pas la même chose. `weaponChanges` sait qualifier (prise, lâcher,
	// échange) et connaît l'emplacement d'arme ; le canal natif ne qualifie rien mais il voit
	// des prises que l'autre rate et il nomme le ramasseur. Là où les deux voient la même
	// prise, ils s'accordent (21/21 et 11/12, arme nommée, à moins de 500 ms).
	Pickups []Pickup `json:"pickups,omitempty"`
	// EquipmentChanges est la liste des RAMASSAGES ET DES CONSOMMATIONS d'équipement (cf.
	// document_equipment_changes.go) : qui, quand, quelle capacité — et, sur une
	// consommation, laquelle vient d'être usée. Les annonces de RÉAPPARITION en sont
	// ÉCARTÉES : ce ne sont pas des ramassages, et ce que le joueur porte à sa naissance est
	// déjà dans `abilities`. Absent si le film n'en porte aucun.
	EquipmentChanges []EquipmentChange `json:"equipmentChanges,omitempty"`
	// Translocations est la liste des TÉLÉPORTATIONS du translocateur (cf.
	// document_translocations.go) : qui saute, à quelle frame, et d'où à où — daté ET situé
	// par l'ÉVÉNEMENT du film (type 117), jamais déduit d'un seuil spatial, ni du `spent`
	// (qui peut suivre l'usage de 16,5 s), ni d'une discontinuité de piste. Le va-et-vient
	// (`fx/fy/fz` -> `tx/ty/tz`) vient de la CHARGE de l'événement, déquantifiée aux bornes
	// de la carte ; il est ABSENT EN BLOC quand elle n'a pas pu être lue, et
	// `coverage.translocations.positioned` dit combien de sauts le portent. Absent si le
	// film n'en porte aucune.
	Translocations []Translocation `json:"translocations,omitempty"`
	// AbilityImpulses est la liste des IMPULSIONS DE CAPACITÉ (cf.
	// document_ability_impulses.go) : qui, à quelle frame, et de quel ÉQUIPEMENT — l'usage
	// MESURÉ du propulseur, daté par le corps `tag == 1` des composants i57/i59 (le même dont
	// le tag 3 porte le grappin) et attribué par le rang i48 lu dans la MÊME VIE et
	// ANTÉRIEUREMENT. Aucun seuil de vitesse, aucune heuristique : le film enregistre le
	// geste, le canal d'identité dit de quel équipement il vient.
	//
	// CE CALQUE NE COUVRE PAS TOUS LES ÉQUIPEMENTS, et c'est publié comme tel : seules les
	// familles que le titre déclare MESURÉES sur ce canal y entrent (aujourd'hui le
	// propulseur, et lui seul — le RÉPULSEUR n'y est pas, négatif MESURÉ des rapports R8/R9).
	// `coverage.abilityImpulses` porte l'entonnoir complet, `otherFamily` compris. Absent si
	// le film n'en porte aucune, ou si sa palette n'a pas été classée (sans nom de rang, pas
	// d'identité).
	AbilityImpulses []AbilityImpulse `json:"abilityImpulses,omitempty"`
	// AbilityCharges est la liste des CHARGES D'ÉQUIPEMENT RESTANTES (cf.
	// document_ability_charges.go) : qui, à quelle frame, quel ÉQUIPEMENT, et ce qu'il en
	// reste — le quartet haut d'i56, transmis AU CHANGEMENT et attribué par le rang i48 lu
	// dans la MÊME VIE et ANTÉRIEUREMENT. Ce sont les LECTURES du film, jamais un compte
	// d'usages dérivé (une baisse peut valoir plusieurs usages — R11 §2), et RIEN n'est
	// affirmé avant la première lecture : le film ne transmet rien au ramassage.
	//
	// CE CALQUE NE COUVRE PAS TOUS LES ÉQUIPEMENTS, et c'est publié comme tel : seules les
	// familles que le titre déclare MESURÉES sur ce canal y entrent (le grappin et le
	// propulseur, et eux seuls — le RÉPULSEUR n'arme jamais i56, négatif MESURÉ du rapport
	// R11). `coverage.abilityCharges` porte l'entonnoir complet. Absent si le film n'en
	// porte aucune, ou si sa palette n'a pas été classée (sans nom de rang, pas d'identité).
	AbilityCharges []AbilityCharge `json:"abilityCharges,omitempty"`
	// GroundWeapons est la liste des ARMES AU SOL individuelles (cf.
	// document_ground_weapon_items.go) : où chacune gît, de quand à quand l'afficher, qui l'a
	// lâchée et qui l'a prise quand le flux delta le dit. Les fins sont OBSERVÉES (ramassage
	// daté, ou recensement des images-clés) — jamais une durée de table. Les armes de socle
	// restent au calque `weaponPads`. Absent si le film n'en porte aucune.
	GroundWeapons []GroundWeapon `json:"groundWeapons,omitempty"`
	// Vehicles est LA VIE DE CHAQUE VÉHICULE du match (cf. document_vehicles.go) : où il naît,
	// sa trajectoire échantillonnée avec son cap, ses épisodes d'occupation (qui est à bord et
	// quand), et jusqu'à quelle frame l'afficher. La fin est une BORNE de recensement et sa
	// cause vaut `unknown` — la datation de la destruction a été mesurée et RÉFUTÉE. Absent
	// quand le film ne porte aucun véhicule ; `coverage.vehicles` dit lequel des silences.
	Vehicles []VehicleTrack `json:"vehicles,omitempty"`
	// VehicleLabels nomme les FAMILLES de châssis employées par `vehicles` et pointe leur
	// sprite — REMPLI À LA REQUÊTE par le service, jamais écrit dans l'artefact (même règle et
	// même raison que `mapObjectives` : ce qui se résout d'un catalogue du titre se résout au
	// service, sinon les artefacts déjà cuits resteraient muets). Absent quand aucune famille
	// n'est résolue.
	VehicleLabels map[string]VehicleLabel `json:"vehicleLabels,omitempty"`
	// WeaponPads (les SOCLES D'ARME du match) et PadPickups (leurs occupations ACHEVÉES) : une
	// donnée de MATCH et non de carte, publiée seulement là où la récurrence est mesurée.
	// Forme, chronique et refus de publication : document_ground_weapons.go.
	WeaponPads []WeaponPad `json:"weaponPads,omitempty"`
	PadPickups []PadPickup `json:"padPickups,omitempty"`
	// Grenades est la liste des LANCERS de grenade rattachés à un slot (cf. grenades.go).
	// Contrairement aux tirs, chaque lancer porte son auteur DANS le film — il n'est pas
	// deviné. Ce n'est pas l'inventaire de grenades (c'est i22, non résolu) : c'est
	// l'événement « ce joueur a lancé cette grenade à cet instant ».
	Grenades []Grenade `json:"grenades,omitempty"`
	// Projectiles est la liste des TRAJECTOIRES de projectile (cf. projectiles.go). Le dernier
	// point est la derniere position REPLIQUEE, pas un impact : le film ne porte aucun
	// evenement de detonation.
	Projectiles []Projectile `json:"projectiles,omitempty"`
	// WeaponLabels nomme les identifiants d'arme employés par le document : famille (8 chiffres
	// hexadécimaux, cf. Loadout.W) ou identifiant global (16 chiffres, cf. Shot.Weapon) -> nom
	// canonique.
	//
	// POURQUOI UNE TABLE ET PAS UN NOM SUR CHAQUE ÉVÉNEMENT : 475 tirs pour 22 armes distinctes.
	// Répéter le libellé alourdirait le document sans rien apprendre.
	//
	// LE TAG BRUT RESTE À CÔTÉ DU LIBELLÉ, jamais à sa place — règle du dépôt : on ne stocke
	// jamais une résolution qui peut s'améliorer. Un identifiant absent de cette table garde
	// donc son hexadécimal à l'écran, et n'emprunte pas le nom d'une arme voisine.
	//
	// Source : `weapon_names.toml` du titre (nom, bilingue) + `replay_labels.toml`
	// (effet de rendu), joints par le weapon_key du registre d'armes.
	WeaponLabels map[string]WeaponLabel `json:"weaponLabels,omitempty"`
	// KillEffects associe un weapon_key du titre à la famille de RENDU de ses effets
	// (mêmes valeurs que WeaponLabel.Fx). C'est la table qui donne leur famille aux
	// EFFETS DE MORT : les kills du feed portent un weapon_key (résolu côté base),
	// jamais un identifiant d'arme film — sans cette table, le client ne peut joindre
	// aucun effet à un kill. Une clé absente = effet neutre, jamais celui d'une voisine.
	// Source : replay_labels.toml du titre ([shot_effects]).
	KillEffects map[string]string `json:"killEffects,omitempty"`
	// NeutralDeaths dit, pour les morts que PERSONNE ne revendique, DE QUOI le joueur est
	// mort. Le fil du rejeu en fait des lignes grises sans tueur ni arme ; sans cette table
	// il n'a qu'un repère générique à y poser. Absente = aucune mort de ce type établie sur
	// ce film — le fil garde son repère neutre, jamais l'icône d'une autre mort.
	NeutralDeaths []NeutralDeath `json:"neutralDeaths,omitempty"`
	// Roster est la liste des joueurs du film : leur identité et leur index de film.
	//
	// CE QU'IL SERT : le client y trouve l'ensemble des joueurs du match, y compris ceux dont
	// aucune vie n'a pu être nommée. Il lui permet aussi de traduire l'index de film porté par
	// les événements (un lancer de grenade écrit son auteur par index) en identité.
	// Absent quand le film n'a livré ni fil des morts ni table d'index.
	Roster []RosterEntry `json:"roster,omitempty"`
	// MapObjectives est le calque STATIQUE des objectifs du MODE JOUÉ (zones de
	// Bastion/Extraction, apparitions et livraisons de drapeau, socles), joint par
	// map_id au catalogue versionné — REMPLI À LA REQUÊTE par le service, jamais écrit
	// dans l'artefact : l'artefact ne connaît ni sa carte ni son mode (cf.
	// map_objectives.go). Absent quand le mode n'a pas d'objectifs statiques (Slayer),
	// quand map_id est vide ou la carte hors catalogue — jamais une erreur.
	MapObjectives *MapObjectives `json:"mapObjectives,omitempty"`
	// MapWeaponPads est le calque des EMPLACEMENTS DE SOCLE de la carte, CROISÉ avec les
	// socles du match — REMPLI À LA REQUÊTE par le service, jamais écrit dans l'artefact
	// (même règle et même raison que MapObjectives, cf. map_weapon_pads.go).
	//
	// IL NE PORTE QUE LES EMPLACEMENTS ALLUMÉS, et c'est une décision produit du
	// 2026-08-19 : le fichier de carte POSE les socles, le mode les ALLUME. Un emplacement
	// que `weaponPads` ne confirme pas à moins d'un mètre ne part PAS au client —
	// Cliffhanger en porte dix-sept au fichier, dix en CTF et zéro en Super Fiesta.
	//
	// CE QU'IL CHANGE POUR LE CLIENT : la position dessinée devient celle du SPAWNER, connue
	// dès la première image et au centimètre, au lieu du centroïde des apparitions vues. La
	// PRÉSENCE ne change pas : elle reste celle du match, socle par socle. Absent quand
	// map_id est vide, la carte hors catalogue, ou qu'aucun emplacement n'est confirmé — le
	// client retombe alors sur les socles du film seuls.
	MapWeaponPads *MapWeaponPads `json:"mapWeaponPads,omitempty"`
	// Objectives est la liste des ACTIONS D'OBJECTIF nommées : ce que chaque joueur a
	// accompli (capture de drapeau, retour, prise de zone, porteur stoppé), daté à la
	// milliseconde et attribué à un xuid (cf. objectives.go).
	//
	// CE QU'ELLE APPORTE que les autres calques n'ont pas : les tirs et les positions
	// disent où les joueurs étaient ; celle-ci dit ce qu'ils ont FAIT. Absente quand le
	// mode n'est pas un mode à objectifs, ou quand l'appelant n'a pas fourni les lignes de
	// match nécessaires au pont d'identité.
	Objectives []ObjectiveAction `json:"objectives,omitempty"`
	// ScoreTimeline est LE SCORE DANS LE TEMPS des deux camps et de chaque joueur (forme, oracle
	// et limites : document_score.go). Absente quand l'appelant n'a rien fourni à lire.
	ScoreTimeline *ScoreTimeline `json:"scoreTimeline,omitempty"`
	// FlagCarries est LA VIE DE CHAQUE DRAPEAU de CTF, en intervalles d'état (forme, sources et refus :
	// document_objectives_live.go). Absente hors CTF — `coverage.flagCarries` dit lequel des deux silences.
	FlagCarries []FlagCarry `json:"flagCarries,omitempty"`
	// FlagReturnZone est LA RÈGLE DE RETOUR du mode, telle que le manifeste du titre la donne
	// (schéma 35) : le rayon de la zone autour d'un drapeau tombé, la minuterie qui le ramène
	// tout seul, et la durée quand UN défenseur s'y tient. Le client en tire le cercle et la
	// jauge ; l'occupation, elle, se compte chez lui — l'équipe d'un joueur n'est PAS dans le
	// film (cf. Track.Team), elle vit dans la base et le client la joint déjà.
	//
	// ABSENTE quand le titre ne la déclare pas, ou quand le film n'est pas une partie de CTF :
	// rien à dessiner, et surtout pas un cercle sur un mode qui n'en a pas.
	FlagReturnZone *FlagReturnZone `json:"flagReturnZone,omitempty"`
	// ObjectiveObjects est OÙ SE TROUVE L'OBJET D'OBJECTIF QUAND PERSONNE NE LE PORTE — les vies
	// LIBRES du crâne d'Oddball (forme, canal et refus : document_objective_objects.go). Un trou
	// entre deux vies est un portage, mais le document ne dit PAS par qui : l'oracle du porteur a
	// été mesuré et réfuté (phase D4). Absente quand le titre ne déclare aucun objet publiable ou
	// quand le film n'en porte pas — `coverage.objectiveObjects` dit lequel des silences.
	ObjectiveObjects []ObjectiveObjectLife `json:"objectiveObjects,omitempty"`
	// ZoneStates est L'ÉTAT DE CHAQUE ZONE du mode, en intervalles de propriété (forme, sources et
	// refus : document_zones.go). Chaque entrée pointe une zone de `mapObjectives.zones` par son
	// index. Absente hors des modes à zones, et quand l'appelant n'a fourni aucun catalogue de
	// carte — `coverage.zones` distingue les deux silences.
	ZoneStates []ZoneState `json:"zoneStates,omitempty"`
	// VipCrown est LES PERIODES DE PORT DE LA COURONNE VIP, en intervalles de frames nommes par le
	// xuid du VIP (forme, sources et garde de mode : document_vip_crown.go). La couronne est a la
	// position de son porteur — le client la pose sur sa piste. Absente hors VIP —
	// `coverage.vipCrown` dit lequel des silences (film non-VIP contre film VIP sans periode).
	VipCrown []VipPeriod `json:"vipCrown,omitempty"`
	// BombArmings est L'ARMEMENT DE LA BOMBE d'Assaut : le début du hold, l'instant armé et la
	// mèche — le compte à rebours [t, t+fuseMs] se dessine sans autre donnée (forme, provenance
	// et refus : document_bomb_armings.go). Absente hors des variantes d'Assaut couvertes
	// (jamais One Bomb) — `coverage.bombArmings` dit lequel des silences.
	BombArmings []BombArming `json:"bombArmings,omitempty"`
	// SkullCarries est LES PERIODES DE PORTAGE DU CRANE d'Oddball, en intervalles de frames nommes
	// par le xuid du porteur (forme, sources et garde de mode : document_skull_carries.go). Le
	// crane porte est a la position de son porteur — le client le pose sur sa piste, comme la
	// couronne VIP. Absente hors Oddball — `coverage.skullCarries` dit lequel des silences (film
	// non-Oddball contre film Oddball sans portage). Le crane LIBRE (`objectiveObjects`) reste la
	// couche POSITION ; celle-ci est la couche PORTEUR.
	SkullCarries []SkullCarry `json:"skullCarries,omitempty"`
	// BombCarries est LES PERIODES DE PORTAGE DE LA BOMBE d'Assaut, en intervalles de frames
	// nommes par le xuid du porteur (forme, sources et garde de mode : document_bomb_carries.go)
	// — le patron de `skullCarries`, sur le canal des armes tenues. La bombe portee est a la
	// position de son porteur ; entre un lacher et la prise suivante, le client la derive des
	// periodes et des pistes (dernier point du lacheur — la bombe au sol n'a pas de canal
	// mesure). Absente hors de la famille bomb — `coverage.bombCarries` dit lequel des silences.
	BombCarries []BombCarry `json:"bombCarries,omitempty"`
	// BombStats est LES CINQ STATISTIQUES D'OBJECTIF DE L'ASSAUT, par joueur, reconstruites du
	// film (forme, provenance et réserves : bomb_stats.go ; câblage : bomb_stats_document.go).
	// L'API 343 n'en publie AUCUNE pour ce mode — le bundle `Stats` de `GetMatchStats` ne porte
	// ni `bomb` ni la famille `BombStats` du moteur, qui est de la TÉLÉMÉTRIE Bond et n'est
	// jamais répliquée dans le film (mesure Ghidra du 2026-09-04).
	//
	// POURQUOI ELLES SONT DANS L'ARTEFACT ET PAS RECALCULÉES PAR LEUR CONSOMMATEUR : elles
	// naissent de quatre sources que SEULE la cuisson tient en pleine fidélité — la chronologie
	// de portage en MILLISECONDES (le document, lui, la publie en frames sous `bombCarries`),
	// le recalage film -> match, les armements datés et les actions d'objectif nommées. Les
	// re-dériver du document en ferait un SECOND décodeur du même fait, moins précis.
	//
	// ABSENT hors de la famille bomb — `bombStats.coverage` dit, DANS le bloc, quelles sources
	// ont été lues : un champ de joueur à `null` est une source non mesurée, jamais un zéro.
	BombStats *BombMatchStats `json:"bombStats,omitempty"`
	// BombEvents est LES FAITS DATÉS DE LA BOMBE — armements et explosions —, sur l'horloge du
	// FILM, la même que `objectives[].timeMs`. Chacun porte son acteur quand la jointure l'a
	// nommé, et la RÈGLE qui l'a nommé (`actorSource`) : `carry_drop` (la bombe a quitté les
	// mains, un geste observé) ou `carry_active` (personne d'autre ne la tenait, une présence
	// constatée) — deux forces de preuve, jamais confondues. Un fait SANS acteur est publié
	// quand même : l'instant est vrai, le nom manque, et l'inventer serait la seule vraie faute.
	BombEvents []BombEvent `json:"bombEvents,omitempty"`
	// Coverage dit, pour chaque calque, COMBIEN il a rattaché SUR COMBIEN existaient, et
	// pourquoi il a écarté le reste (cf. coverage.go).
	//
	// POURQUOI C'EST DANS LE DOCUMENT ET PAS SEULEMENT DANS LES JOURNAUX : publier 147 tirs
	// sans dire que 519 existent laisse croire à l'exhaustivité. L'écart doit être lisible
	// là où le résultat l'est. Absent des artefacts construits avant cette version.
	Coverage *Coverage `json:"coverage,omitempty"`
	// Identity est LE REGISTRE D'IDENTITÉ du film (schéma 50, lot P2) : chaque lien entre une
	// entité du film et un joueur, avec sa PROVENANCE (`direct` / `catalogue` / `externe` /
	// `deduit` / `non_resolu`), la voie exacte qui l'a produit, et les BORNES entre lesquelles
	// il vaut (cf. identity_registry_section.go).
	//
	// POURQUOI ELLE EST PUBLIÉE. Le document disait DÉJÀ qui porte une trace (`tracks[].xuid`)
	// sans jamais dire SUR QUOI ce nom repose : une lecture du film, un pont par morts, une
	// fermeture, une déduction par élimination — quatre forces de preuve que rien ne
	// distinguait. La section les sépare, et son récapitulatif (`identity.coverage`) est ce que
	// le gate corpus compare : une régression d'un lien `direct` vers un lien `deduit` y devient
	// visible, alors qu'elle était jusqu'ici indétectable.
	//
	// Absente des artefacts construits avant le schéma 50.
	Identity *IdentitySection `json:"identity,omitempty"`
}

// RosterEntry est un joueur du film : son identité, et l'index sous lequel le film le désigne.
//
// LES DEUX NE SONT PAS INTERCHANGEABLES. Le xuid IDENTIFIE ; l'index ORDONNE, et il n'a de
// sens qu'à l'intérieur de ce film. Les garder côte à côte est ce qui permet de traduire un
// événement (qui porte l'index) sans jamais confondre l'un avec l'autre.
type RosterEntry struct {
	// XUID en décimal, même forme que Track.XUID et que la base.
	XUID string `json:"xuid"`
	// FilmIndex est l'index du joueur DANS CE FILM, lu dans les cinq bits qui précèdent son
	// xuid (cf. player_index.go, 26 chunks concordants sur le film de référence).
	FilmIndex int `json:"filmIndex"`
	// Name est le gamertag TEL QUE LE FILM L'ÉCRIT, dans le même enregistrement que le xuid.
	//
	// CE N'EST PAS UNE RÉSOLUTION : rien n'est allé le chercher ailleurs, donc rien ne peut
	// l'avoir mal apparié. Il rend le rejeu lisible sans base de données. Ce qu'il ne donne
	// PAS, et que seule la base porte : l'équipe, et les compteurs du match. Vide si
	// l'enregistrement ne le portait pas.
	Name string `json:"name,omitempty"`
	// Bot est vrai pour une entrée déclarée par BOT_METADATA (schéma 36) : son XUID est VIDE
	// — un bot n'en a pas, et le normaliser en pseudo-identifiant fusionnerait des bots — et
	// son Name porte le suffixe « [bot] », comme la base l'écrit. FilmIndex est le slot du
	// roster de réplication que le paquet type 12 déclare.
	Bot bool `json:"bot,omitempty"`
	// Bid est l'identifiant STABLE d'un bot, forme `bid(N.0)` — la même que `match_participants`
	// écrit (schéma 50).
	//
	// IL ÉTAIT LU DEPUIS TOUJOURS ET N'ÉTAIT PAS PUBLIÉ (inventaire P1, E9). Faute de l'avoir,
	// la jointure web d'un bot se faisait sur le NOM NU — égalité de chaîne des deux côtés —,
	// et le commentaire de `rosterLogic.ts` le disait lui-même : deux bots homonymes fusionnent.
	// Vide pour un humain, et vide pour un bot dont la déclaration ne portait pas d'identifiant :
	// un `bid(0.0)` inventé joindrait deux bots distincts.
	Bid string `json:"bid,omitempty"`
}

// Loadout est l'ensemble des armes PORTÉES par un slot à un instant de référence.
//
// CE QUE LE CHAMP GARANTIT : à l'instant T, ce slot AVAIT ces armes dans son inventaire.
// Témoin croisé sur une source indépendante (l'arme des events de tir) : 98,3 % d'accord
// contre 7,2 % pour le témoin qui ne casse QUE la jointure record->slot. Détail et limites
// en tête de loadouts.go.
//
// CE QU'IL NE GARANTIT PAS, et il faut le dire à l'écran :
//   - QUELLE arme est dégainée. Le loadout est l'inventaire, pas la main. Croiser avec le
//     dernier Shot du même slot désigne l'arme en main ; sans tir récent, on ne sait pas.
//   - la CONTINUITÉ. Un keyframe toutes les ~18-20 s : entre deux instants publiés, un
//     ramassage d'arme est invisible. Un client qui maintient la dernière valeur connue
//     affiche un état de référence, pas une mesure de l'instant.
//   - les GRENADES, la capacité d'armure, les munitions : NON décodées.
type Loadout struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot est le slot du biped porteur : il désigne la Track concernée.
	Slot uint32 `json:"slot"`
	// W liste les identifiants de FAMILLE d'arme (high-32 du weapon-id 64 bits) en
	// hexadécimal 8 chiffres, dans l'ordre de lecture du record. La famille est l'identité
	// de l'arme, le suffixe bas ne porte que la variante cosmétique (cf. weaponv3.CanonWeaponID) ;
	// les alias d'un même canon sont repliés — un canon = une entrée.
	W []string `json:"w"`
}

// Shot est un tir décodé, placé à la position de son tireur.
//
// CE QUE LE CHAMP GARANTIT :
//   - le tir a INFLIGÉ DES DÉGÂTS. Le record du film (event type 105) n'existe que quand un
//     dégât est appliqué : il n'y a pas de record de tir manqué, donc pas de notion
//     « touché / raté » à afficher — tous les tirs publiés ici ont touché quelqu'un.
//   - l'origine (X, Y) est la position du biped tireur à l'instant du tir.
//
// CE QU'IL NE GARANTIT PAS :
//   - l'exhaustivité : seuls les tirs dont le tireur a pu être rattaché SANS AMBIGUÏTÉ sont
//     publiés (30 à 57 % des events selon le film) ;
//   - la VICTIME : elle n'est pas décodée du film (champ de largeur runtime). Le trait part
//     du tireur dans la direction visée, il ne relie pas deux joueurs.
type Shot struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot est le slot du biped tireur : il désigne la Track d'où part le tir.
	Slot uint32 `json:"slot"`
	// X, Y sont l'origine du tir (position du tireur).
	X float32 `json:"x"`
	Y float32 `json:"y"`
	// H (optionnel) est le CAP DE VISÉE du tir en degrés, même convention que Point.H.
	// Absent quand la visée n'était pas lisible hors ligne (le champ vit après des boucles
	// de longueur variable dans ~80 % des records) : le client dessine alors un simple
	// marqueur, sans direction. Même PIÈGE omitempty que Point.H, même parade (0 -> 360).
	H float32 `json:"h,omitempty"`
	// Weapon est l'identifiant global 64 bits de l'arme, en hexadécimal (un entier 64 bits
	// ne survit pas au `number` JavaScript). Clé de metadata.weapon_labels.weapon_id.
	Weapon string `json:"w,omitempty"`
	// Vehicle est le SLOT DU VÉHICULE d'où part le tir, quand le tireur était EMBARQUÉ.
	//
	// À QUOI IL SERT, ET POURQUOI IL FAUT UN MARQUEUR. Sur un tir à pied, `X`/`Y` sont la
	// position du BIPÈDE : le client peut retrouver le tireur dans ses pistes et y accrocher ce
	// qu'il veut. Sur un tir en véhicule, le bipède ne réplique plus (primitive V1a.4) et
	// `X`/`Y` sont la position INTERPOLÉE DU VÉHICULE : sans ce champ, le client ne saurait pas
	// que la piste du `Slot` est muette à cet instant, et chercherait un pion qui n'existe pas.
	// Il porte le slot plutôt qu'un booléen pour que l'effet puisse s'ancrer sur LE véhicule
	// concerné — c'est la même clé que `VehicleTrack.Slot`.
	//
	// POINTEUR, comme `VehicleRide.Seat` et pour la même raison : un slot vaut zéro en droit, et
	// `omitempty` sur un entier effacerait ce zéro exactement comme une absence de véhicule.
	// Nil = tir à pied, et c'est le cas nominal.
	Vehicle *uint32 `json:"v,omitempty"`
}

// Bounds est l'étendue alignée sur les axes de tous les points de trajectoire, dans le
// repère monde partagé. Permet au client d'ajuster la scène au viewport (le range monde
// absolu est inutile au rendu — seule la disposition relative importe).
type Bounds struct {
	MinX float32 `json:"minX"`
	MinY float32 `json:"minY"`
	MaxX float32 `json:"maxX"`
	MaxY float32 `json:"maxY"`
	// MinZ / MaxZ (optionnels) donnent l'amplitude verticale, pour colorer les étages.
	// PIÈGE omitempty : une borne exactement nulle est omise — les valeurs sont issues
	// d'une déquantification à mi-bucket (min + step*(q+0.5)), un zéro exact est donc
	// hors d'atteinte en pratique ; le client lit une borne absente comme 0.
	MinZ float32 `json:"minZ,omitempty"`
	MaxZ float32 `json:"maxZ,omitempty"`
}

// Track est la trajectoire d'une entité (slot biped) sur la timeline du rejeu.
//
// ATTENTION : un slot est réattribué aux respawns — une Track = UNE VIE, pas un joueur.
// Le regroupement des vies par joueur se fait par XUID.
type Track struct {
	Slot uint32 `json:"slot"`
	// Team vaut -1 : L'ÉQUIPE N'EST PAS DANS LE FILM. Elle vit dans la base, avec le gamertag,
	// et le client la joint par XUID (cf. XUID ci-dessous). Le champ est conservé pour les
	// artefacts d'un titre qui la porterait ; le laisser à -1 n'est pas un oubli.
	Team int `json:"team"`
	// Name est TOUJOURS VIDE, et c'est délibéré : le film ne porte aucun gamertag. Le remplir
	// exigerait de lire la base depuis un outil hors ligne dont toute la valeur est de n'en
	// dépendre pas. Le client joint le nom par XUID.
	Name string `json:"name,omitempty"`
	// XUID est l'IDENTITÉ du porteur de cette vie, en décimal (un entier 64 bits ne survit pas
	// au `number` JavaScript ; le décimal est aussi la forme employée par la base).
	//
	// POURQUOI LE XUID ET PAS UN INDEX. Un index est un ORDRE, jamais une identité — la leçon
	// a coûté une fausse découverte à ce chantier (un tri alphabétique publié comme une
	// permutation du format). Le xuid est stable, global, et indépendant de tout tri : c'est
	// la seule clé sur laquelle un client peut joindre sans rien supposer.
	//
	// D'OÙ IL VIENT : le fil des morts du film nomme chaque vie par le xuid de sa victime
	// (cf. lives.go). Vide quand la vie n'a pas été nommée — 15 vies sur 105 sur le film de
	// référence, dont 4 antérieures au début réel du match et 6 survivants de fin de partie,
	// que le film ne clôt par aucun événement.
	XUID string `json:"xuid,omitempty"`
	// Bot est le NOM du bot qui porte cette vie (schéma 36), suffixe « [bot] » compris — un
	// bot n'a pas de xuid, et c'est le seul cas où une vie est nommée sans en avoir un.
	//
	// D'OÙ IL VIENT : BOT_METADATA (paquet type 12) déclare slot de roster + nom, et le pont
	// des fermetures attribue un slot de biped à cet index quand l'unicité le permet
	// (cf. nameBotTracks). Une vie de bot que le pont ne peut pas attribuer reste anonyme —
	// même règle que les humains : rien plutôt que faux.
	Bot    string  `json:"bot,omitempty"`
	Points []Point `json:"points"`
	// StartFrame / EndFrame (optionnels) bornent la vie de la track sur l'axe de temps :
	// le client peut masquer l'entité hors de cette fenêtre au lieu de la figer.
	StartFrame int `json:"startFrame,omitempty"`
	EndFrame   int `json:"endFrame,omitempty"`
}
