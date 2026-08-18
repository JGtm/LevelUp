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
// v2 (2026-08-02, lot 3.1/3.2) : les trois tables de libellés deviennent BILINGUES
// (`{en, fr}` au lieu d'une chaîne) et le type d'un lancer de grenade devient son RANG
// (`rank`) au lieu d'un nom. Motif : les catalogues étaient codés en Go, dont deux en
// français — ce qui interdisait l'anglais autant qu'un second titre — et les grenades
// étaient nommées deux fois, différemment, sur la même fiche.
//
// v3 (2026-08-13, plan parité lot 2) : le lancer de grenade publie son LIEN vers le
// projectile né de lui (`Grenade.proj`), et le document publie la table
// `killEffects` (weapon_key -> famille de rendu) qui donne leur famille aux effets de
// mort du kill feed. Les deux champs sont sérialisés en omitempty, mais la version
// monte quand même : les effets de repos de grenade côté client N'EXISTENT que si
// l'artefact porte le lien, et la reprise du backfill (lot 6) se fait par
// SchemaVersion — un artefact v2 doit se voir comme « à re-cuire », pas comme à jour.
//
// v4 (2026-08-14, plan parité lot 7.2) : le document publie son ORIGINE (`originMs`) —
// l'instant de sa frame 0 sur l'horloge du fil des éliminations (cf. origin.go). Le champ
// est optionnel, mais la version monte : sans lui le client ne peut PAS caler le fil
// autrement que par appariement statistique, et un artefact v3 doit donc se voir comme
// « à re-cuire », pas comme à jour.
// v5 (2026-08-14, plan parité lot 7.1) : le document publie `neutralDeaths` — le TYPE des
// morts que personne ne revendique (chute / hors-limites, ou sa propre source de dégât), lu
// dans le dead-state du film par le décodeur de source de dégât. Le champ est optionnel, mais
// la version monte : sans lui le fil ne peut poser sur ces lignes qu'un repère générique, et
// un artefact v4 doit se voir comme « à re-cuire », pas comme à jour.
//
// v6 (2026-08-14, plan PLAN_RANG_CAPACITE_I48 étape 1.2) : la capacité d'armure CHANGE DE
// GRANDEUR, et c'est pourquoi elle change aussi de nom. `Inventory.a` portait un INDEX
// TRONQUÉ — le motif d'ancrage du canal d'image-clé se termine par `010`, les bits de POIDS
// FORT du rang, si bien que ce canal ne voit que les rangs 16 à 23 et rendait `rang − 16`.
// Le document publie désormais `abilities` : le RANG de palette complet, sur une seule
// grandeur, alimenté par DEUX canaux (i48 pour toute la palette, l'image-clé pour sa fenêtre
// 16-23). `Inventory.a` est RETIRÉ plutôt que réinterprété : republier une autre grandeur
// sous la même clé aurait laissé tout client non mis à jour lire un nombre qui ne veut plus
// dire la même chose — c'est le défaut qui a coûté ce chantier. `abilityLabels` est donc
// keyé par RANG, et un artefact v5 doit se voir comme « à re-cuire », pas comme à jour.
//
// v7 (2026-08-16, plan PLAN_EQUIPEMENT_TI37 phase 1) : le document publie
// `equipmentEpisodes` — l'état ACTIF du camouflage et du surbouclier, en épisodes datés
// par vie (cf. equipment_episodes.go). Le champ est optionnel, mais la version monte :
// l'effet plein-fiche et les sons d'équipement côté client N'EXISTENT que si l'artefact
// porte les épisodes, et la reprise du backfill se fait par SchemaVersion — un artefact
// v6 doit se voir comme « à re-cuire », pas comme à jour (le re-build de masse pendant au
// registre portera ce champ avec le correctif de précision des objets du monde).
//
// v8 (2026-08-16, plan PLAN_GRAPPIN_LIGNE phase 1) : le document publie `grappleLines` —
// les TRACTIONS de grappin datées par vie avec leur point d'accroche en coordonnées monde
// (cf. grapple_lines.go ; source : le corps tag==3 d'i59, porté et prouvé au gate 0 du
// même plan). Le champ est optionnel, mais la version monte : la ligne joueur -> ancre
// côté client N'EXISTE que si l'artefact la porte, et la reprise du backfill se fait par
// SchemaVersion — un artefact v7 doit se voir comme « à re-cuire », pas comme à jour.
//
// v9 (2026-08-18, plan PLAN_POSES_EQUIPEMENT_PUBLICATION phase 2) : le document publie
// `equipmentPlacements` — les POSES d'objets d'équipement (mur de protection, capteur de
// menaces, et les objets du monde qui partagent l'archétype), avec leur position monde, leur
// fenêtre [t0, t1], l'identifiant `eqip` que le jeu leur donne, le poseur mesuré et son cap de
// visée (cf. equipment_placements.go ; identité prouvée au gate 1 de PLAN_IDENTITE_TI37, largeur
// du bloc MPP calibrée par oracle de position au gate 0 du présent plan). Le champ est
// optionnel, mais la version monte : les marqueurs d'équipement posé côté client N'EXISTENT que
// si l'artefact les porte, et la reprise du backfill se fait par SchemaVersion — un artefact v8
// doit se voir comme « à re-cuire », pas comme à jour.
//
// v10 (2026-08-18, plan PLAN_ORIGINE_POSES_ET_FAMILLES phase G) : chaque pose porte son
// ORIGINE MESURÉE (`origin` : `deployed` / `dropped` / `unknown`), et `coverage.placements`
// la croise avec la famille. La version monte alors qu'un champ s'ajoute à un sous-objet, et
// c'est justifié par ce que la mesure a trouvé : **`equipmentPlacements` n'était pas ce que
// son nom dit**. Sur les 11 films calibrés, 3 242 des 3 661 poses à poseur mesuré (88,6 %)
// naissent dans les 2 frames et les 1,5 m du DERNIER POINT de leur poseur — ce sont les objets
// qu'il PORTAIT, relâchés quand sa vie s'achève, pas des poses sur la carte. Un client v9
// dessine donc, aujourd'hui, un mur là où personne n'en a déployé ; sans montée de version il
// continuerait, puisque la reprise du backfill se fait par SchemaVersion.
//
// L'HYPOTHÈSE DU PLAN ÉTAIT L'AUTRE BOUT DE LA VIE, ET ELLE EST RÉFUTÉE : le critère écrit
// avant mesure cherchait une DOTATION AU SPAWN (création dans les 2 frames du DÉBUT de la vie
// du poseur). Elle compte 4 poses sur 3 661 (0,1 %), et les 4 sont des vies de 0,13 à 1,49 s
// où début et fin se confondent — le mode n'existe pas. Le témoin interne est dans la même
// mesure : distance médiane de la pose à la position de DÉBUT de vie 27,03 m, à celle de FIN
// 0,57 m, soit un facteur 47,5 sur les mêmes poses.
//
// v11 (2026-08-17, plan PLAN_ARMES_AU_SOL_2E_LECTURE phase 3) : `weaponPads` et `padPickups` —
// les SOCLES D'ARME du match. Chronique complète, et surtout ce que la mesure a REFUSÉ de
// publier (le ramasseur, les armes lâchées, le catalogue de carte) : document_ground_weapons.go.
//
// v12 (2026-08-18, plan PLAN_EXPLOITATION_REGISTRE_FILM lot A phase 1) : `scoreTimeline` — LE
// SCORE DANS LE TEMPS, et deux correctifs du calque `objectives` (vide en production, décalé de
// `originMs`). Chronique complète, oracle et limites : document_score.go.
//
// v13 (2026-08-18, plan PLAN_EXPLOITATION_REGISTRE_FILM lot E phase 1) : `Point.p` — LE
// DEUXIÈME AXE DE LA VISÉE (élévation en degrés, positif = vers le haut). Un champ optionnel
// sur un sous-objet, et pourtant la version monte : jusqu'ici le cône de visée était dessiné à
// sa longueur maximale sur chaque point porteur de cap, ce qui AFFIRMAIT une visée horizontale
// que le film contredit. Chronique complète, convention mesurée et réserve : document_aim.go.
//
// v14 (2026-08-18, plan PLAN_OBJECTIFS_VIVANTS_2E_LECTURE phase 1 item 1.3) : `flagCarries` — LA VIE DE
// CHAQUE DRAPEAU de CTF, et `coverage.flagCarries`. Chronique, sources et refus : document_objectives_live.go.
//
// v15 (2026-08-18, plan PLAN_EXPLOITATION_REGISTRE_FILM lot C-bis phase 2b) : `zoneStates` — L'ETAT
// DE CHAQUE ZONE (qui la tient, depuis quand, jusqu'a quel niveau de jauge), et `coverage.zones`.
// Chronique, sources et refus : document_zones.go.
const SchemaVersion = 15

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
	// ZoneStates est L'ÉTAT DE CHAQUE ZONE du mode, en intervalles de propriété (forme, sources et
	// refus : document_zones.go). Chaque entrée pointe une zone de `mapObjectives.zones` par son
	// index. Absente hors des modes à zones, et quand l'appelant n'a fourni aucun catalogue de
	// carte — `coverage.zones` distingue les deux silences.
	ZoneStates []ZoneState `json:"zoneStates,omitempty"`
	// Coverage dit, pour chaque calque, COMBIEN il a rattaché SUR COMBIEN existaient, et
	// pourquoi il a écarté le reste (cf. coverage.go).
	//
	// POURQUOI C'EST DANS LE DOCUMENT ET PAS SEULEMENT DANS LES JOURNAUX : publier 147 tirs
	// sans dire que 519 existent laisse croire à l'exhaustivité. L'écart doit être lisible
	// là où le résultat l'est. Absent des artefacts construits avant cette version.
	Coverage *Coverage `json:"coverage,omitempty"`
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
	XUID   string  `json:"xuid,omitempty"`
	Points []Point `json:"points"`
	// StartFrame / EndFrame (optionnels) bornent la vie de la track sur l'axe de temps :
	// le client peut masquer l'entité hors de cette fenêtre au lieu de la figer.
	StartFrame int `json:"startFrame,omitempty"`
	EndFrame   int `json:"endFrame,omitempty"`
}

// Surface est l'emprise au sol d'un élément de structure : la projection sur (x, y) de
// l'AABB MONDE d'une instance de géométrie instanciée, plus les altitudes de ses faces
// haute et basse. Rectangle aligné sur les axes — PAS le maillage : le lien instance ->
// géométrie n'est pas résolu (layout du champ meshRef inconnu), on publie donc la boîte
// englobante et rien de plus. Une boîte de plateforme ou de mur suffit à une carte
// reconnaissable en vue de dessus ; elle ne rend pas les formes courbes.
//
// CE QUE LE CHAMP GARANTIT : Z est l'altitude de la face SUPÉRIEURE, celle sur laquelle un
// joueur se tient. Mesure : sur le film 000d5950, 80,6 % des positions à vitesse verticale
// quasi nulle sont à moins de 5 cm au-dessus de la surface la plus haute sous elles (11,9 %
// attendus par hasard, 37,5 % pour le témoin le plus sévère — altitudes permutées entre
// emprises), écart médian 8 mm.
type Surface struct {
	X0 float32 `json:"x0"`
	Y0 float32 `json:"y0"`
	X1 float32 `json:"x1"`
	Y1 float32 `json:"y1"`
	// Z est l'altitude de la face supérieure de la boîte (le « sol » de l'élément).
	Z float32 `json:"z"`
	// ZB est l'altitude de la face inférieure. PIÈGE : pas d'omitempty ici — une face à
	// exactement 0 serait omise et relue comme « au niveau de la mer », ce qui déplacerait
	// l'élément d'un étage.
	ZB float32 `json:"zb"`
	// Poly est l'emprise ORIENTÉE, quand elle est connue : 4 à 8 sommets XY dans le repère
	// monde. Absente, le client retombe sur le rectangle X0/Y0/X1/Y1.
	//
	// POURQUOI : X0..Y1 est une boîte alignée sur les axes du MONDE, alors que l'instance
	// porte sa propre base (forward, left, up). Pour une instance tournée, l'AABB déborde
	// largement de la pièce réelle. En prenant la boîte orientée, la surface totale de la
	// carte tombe de 47,4 %.
	//
	// CE QUE ÇA NE FAIT PAS, et il faut le savoir avant de s'en réjouir : l'emprise orientée
	// ne CREUSE rien. Sur la zone « Fer à cheval », le vide passe de 0,00 m² à 0,00 m² —
	// les neuf instances qui en bouchent le centre sont à yaw nul et échelle unité, donc leur
	// boîte orientée est identique à leur boîte alignée. Un anneau vit dans les TRIANGLES du
	// maillage ; aucune boîte, même exacte, ne le rendra.
	Poly [][2]float32 `json:"poly,omitempty"`
}

// Area renvoie l'aire au sol de l'emprise, en m².
func (s Surface) Area() float32 {
	w, h := s.X1-s.X0, s.Y1-s.Y0
	if w <= 0 || h <= 0 {
		return 0
	}
	return w * h
}

// MapObject est un prop Forge projeté en 2D : centre orienté + emprise de sa bounding box.
// Ce sont de PETITS objets (0,25 m² en moyenne) — décor et repères, pas les sols/murs.
type MapObject struct {
	// TypeID est l'identifiant global du tag Forge (permet un style par famille d'objet).
	TypeID int64 `json:"typeId"`
	// X, Y sont le centre au sol ; Z l'altitude (indication d'étage).
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z,omitempty"`
	// DX, DY sont l'emprise (largeur/profondeur) du modèle, avant rotation.
	DX float32 `json:"dx,omitempty"`
	DY float32 `json:"dy,omitempty"`
	// Yaw est la rotation autour de la verticale, en degrés.
	Yaw float32 `json:"yaw,omitempty"`
}
